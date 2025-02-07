package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
	"toyStore/handlers"
	"toyStore/repository"
	"toyStore/services"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"

	"github.com/redis/go-redis/v9"
)

var db *sql.DB
var rdb *redis.Client

func main() {
	initDB()
	defer db.Close()
	defer rdb.Close()

	uR, err := repository.NewUserRepository(db)
	sR, err2 := repository.NewSessionRepository(rdb, context.Background())
	pR, _ := repository.NewProductRepository(db)
	aR, _ := repository.NewAttributeRepository(db)
	cR, _ := repository.NewCategoryRepository(db)
	cartR, _ := repository.NewCartRepository(rdb, context.Background())
	oR, _ := repository.NewOrderRepository(db)
	if err != nil {
		panic(err)
	}
	log.Printf("db connected")
	if err2 != nil {
		panic(err2)
	}
	log.Printf("redis connected")
	hp := handlers.HandlerParams{
		UsrService:  services.NewUserService(uR, sR),
		PrdService:  services.NewProductService(pR, aR, cR),
		CrtService:  services.NewCartService(pR, cartR),
		CatsService: services.NewCategoryService(cR, pR),
		AtrService:  services.NewAttributeService(aR),
		OrdService:  services.NewOrderService(sR, pR, cartR, oR),
	}
	ha := handlers.NewHandler(hp)
	router := mux.NewRouter()
	router.Use(ha.ErrorHandleMiddleware)
	subAuth := router.NewRoute().Subrouter()
	subAuth.Use(ha.AuthMiddleware)
	subManAuth := router.NewRoute().Subrouter()
	subManAuth.Use(ha.ManagerAuthMiddleware)

	router.HandleFunc("/", ha.Welcome)
	router.HandleFunc("/users/signin", ha.Signin)
	router.HandleFunc("/users/signup", ha.Signup)
	subAuth.HandleFunc("/users/refresh", ha.Refresh)
	subAuth.HandleFunc("/users/logout", ha.Logout)
	subAuth.HandleFunc("/users/change_password", ha.ChangePassword)
	subManAuth.HandleFunc("/users/create", ha.CreateUser)

	router.HandleFunc("/cart", ha.GetCart).Methods("GET")
	router.HandleFunc("/cart", ha.DeleteFromCart).Methods("DELETE")
	router.HandleFunc("/cart", ha.AddToCart).Methods("POST")
	subAuth.HandleFunc("/cart/buy", ha.CreateOrder)

	router.HandleFunc("/products/{id:[0-9]+}", ha.GetProduct)
	subManAuth.HandleFunc("/products/{id:[0-9]+}/update", ha.UpdateProduct)
	subManAuth.HandleFunc("/products/{id:[0-9]+}/update/attribute", ha.UpdateProductAttributes).Methods("POST")
	subManAuth.HandleFunc("/products/{id:[0-9]+}/delete/attribute", ha.RemoveProductAttributes).Methods("DELETE")
	subManAuth.HandleFunc("/products/{id:[0-9]+}/update/category", ha.UpdateProductCategory).Methods("POST")
	subManAuth.HandleFunc("/products/{id:[0-9]+}/delete/category", ha.RemoveProductCategory).Methods("DELETE")

	subManAuth.HandleFunc("/attributes/create", ha.CreateAttribute).Methods("POST")
	subManAuth.HandleFunc("/attributes/{id:[0-9]+}/update", ha.UpdateAttribute).Methods("POST")
	router.HandleFunc("/categories", ha.GetAllCategories)
	router.HandleFunc("/categories/{id:[0-9]+}", ha.GetCategoryWithProducts)
	subManAuth.HandleFunc("/categories/create", ha.CreateCategory).Methods("POST")
	subManAuth.HandleFunc("/categories/{id:[0-9]+}/update", ha.UpdateCategory).Methods("POST")

	subManAuth.HandleFunc("/orders/{id:[0-9]+}", ha.GetOrderById)
	subManAuth.HandleFunc("/orders/search", ha.SearchOrders)
	subAuth.HandleFunc("/orders/", ha.GetCurrentUserOrders)
	subAuth.HandleFunc("/orders/{id:[0-9]+}/cancel", ha.CancelOrder)
	subManAuth.HandleFunc("/orders/{id:[0-9]+}/update", ha.SetOrderStatus).Methods("POST")

	log.Printf("starting server...")
	http.ListenAndServe(":8080", router)
}

func initDB() {
	host := os.Getenv("DATABASE_HOST")
	port := os.Getenv("DATABASE_PORT")
	user := os.Getenv("DATABASE_USER")
	pass := os.Getenv("DATABASE_PASSWORD")
	dbname := os.Getenv("DATABASE_NAME")
	var err error

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, pass, host, port, dbname)
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		panic(err)
	}

	redis_host := os.Getenv("REDIS_HOST")
	redis_port := os.Getenv("REDIS_PORT")

	rdb = redis.NewClient(&redis.Options{
		Addr:     redis_host + ":" + redis_port,
		Password: "",
		DB:       0,
	})
	ctx, cncl := context.WithTimeout(context.Background(), 5*time.Second)
	defer cncl()
	if status := rdb.Ping(ctx); status.Err() != nil {
		panic("redis is not working: " + status.Err().Error())
	}
}
