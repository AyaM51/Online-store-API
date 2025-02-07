package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"strconv"
	"time"

	"toyStore/entities"
	"toyStore/models"
	"toyStore/services"

	"github.com/gorilla/mux"
)

type Handler struct {
	us  services.UserService
	ps  services.ProductService
	cs  services.CartService
	cas services.CategoryService
	ats services.AttributeService
	ors services.OrderService
}

type HandlerParams struct {
	UsrService  services.UserService
	PrdService  services.ProductService
	CrtService  services.CartService
	CatsService services.CategoryService
	AtrService  services.AttributeService
	OrdService  services.OrderService
}

func NewHandler(params HandlerParams) *Handler {
	return &Handler{
		cs:  params.CrtService,
		us:  params.UsrService,
		ors: params.OrdService,
		cas: params.CatsService,
		ps:  params.PrdService,
		ats: params.AtrService,
	}
}

func (h *Handler) Welcome(w http.ResponseWriter, r *http.Request) {
	var welcome, name string
	var uModel models.User_db
	var exists bool

	c, err := r.Cookie("sessionId")
	if err != nil {
		name = "guest"
	} else {
		sessionId := c.Value
		uModel, exists = h.us.WelcomeRequest(sessionId)
		if !exists {
			name = "guest"
		} else {
			name = uModel.Nickname
		}
	}
	welcome = "Hello, " + name + "!"
	w.Write([]byte(welcome))
}

//user

func (h *Handler) Signup(w http.ResponseWriter, r *http.Request) {
	creds := models.Credentials{}
	err := json.NewDecoder(r.Body).Decode(&creds)
	if err != nil {
		log.Printf("Unmarshal err:%v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	creds.Role = "user"

	_, err = h.us.SignupRequest(creds)
	if err != nil {
		WriteErrorResponse(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Signin(w http.ResponseWriter, r *http.Request) {
	creds := models.Credentials{}
	var sessionId string

	err := json.NewDecoder(r.Body).Decode(&creds)
	if err != nil {
		log.Printf("Unmarshal err:%v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	_, sessionId, err = h.us.SigninRequest(creds.Username, creds.Password)
	if err != nil {
		WriteErrorResponse(w, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:    "sessionId",
		Value:   sessionId,
		Path:    "/",
		Expires: time.Now().Add(24 * time.Hour),
		// redis 30 min
	})
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	c, _ := r.Cookie("sessionId")
	sessionId := c.Value
	err := h.us.RefreshRequest(sessionId)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:    "sessionId",
		Value:   sessionId,
		Path:    "/",
		Expires: time.Now().Add(24 * time.Hour),
		// redis 30 min
	})
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	c, _ := r.Cookie("sessionId")
	sessionId := c.Value

	err := h.us.DeleteSessionRequest(sessionId)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:    "sessionId",
		Value:   "",
		Path:    "/",
		Expires: time.Now(),
	})
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	c, _ := r.Cookie("sessionId")
	sessionId := c.Value

	data := models.PasswordData{}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		log.Printf("Unmarshal err:%v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	err = h.us.ChangePasswordRequest(sessionId, data.OldPassword, data.NewPassword)
	if err != nil {
		WriteErrorResponse(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:    "sessionId",
		Value:   "",
		Path:    "/",
		Expires: time.Now(),
		// redis 30 min
	})
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	creds := models.Credentials{}
	err := json.NewDecoder(r.Body).Decode(&creds)
	if err != nil {
		log.Printf("Unmarshal err:%v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	err = h.us.CreateUserRequest(creds)
	if err != nil {
		WriteErrorResponse(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// product
func (h *Handler) GetProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		log.Printf("Unmarshal err:%v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	prod, err := h.ps.GetProductById(id)
	if err != nil {
		WriteErrorResponse(w, err)
		return
	}
	jsonData, err2 := json.MarshalIndent(prod, "", "  ")
	if err2 != nil {
		log.Printf("Marshal err:%v", err2)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.Write(jsonData)
}

func (h *Handler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var pModel models.Product
	err := json.NewDecoder(r.Body).Decode(&pModel)
	if err != nil {
		log.Printf("Unmarshal err:%v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	err = h.ps.CreateProduct(pModel)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var pModel models.Product

	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		log.Printf("Unmarshal err:%v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	err = json.NewDecoder(r.Body).Decode(&pModel)
	if err != nil {
		log.Printf("Unmarshal err: %v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	pModel.Id = id
	updatedProd, err2 := h.ps.UpdateProductById(pModel)
	if err2 != nil {
		WriteErrorResponse(w, err2)
		return
	}
	jsonData, err3 := json.MarshalIndent(updatedProd, "", "  ")
	if err3 != nil {
		log.Printf("Marshal err:%v", err3)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.Write(jsonData)
}

func (h *Handler) UpdateProductAttributes(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var attrs []entities.ProductAttribute
	var attrsInvaild []entities.ProductAttribute

	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		log.Printf("Unmarshal err:%v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	err = json.NewDecoder(r.Body).Decode(&attrs)
	if err != nil {
		log.Printf("Unmarshal err: %v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	attrsInvaild, err = h.ps.UpdateProductAttributes(id, attrs)
	if err != nil {
		WriteErrorResponse(w, err)
		return
	}
	if len(attrsInvaild) > 0 {
		jsonData, err2 := json.MarshalIndent(attrsInvaild, "", "  ")
		if err2 != nil {
			log.Printf("Marshal err:%v", err2)
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
		w.Write(jsonData)
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) RemoveProductAttributes(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var attrs []int
	var removed int
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		log.Printf("Unmarshal err:%v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	err = json.NewDecoder(r.Body).Decode(&attrs)
	if err != nil {
		log.Printf("Unmarshal err:%v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	removed, err = h.ps.RemoveProductAttributes(id, attrs)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		fmt.Println(err.Error())
		return
	}
	w.Write([]byte("Removed " + strconv.Itoa(removed) + " attribute(s)"))
}

func (h *Handler) UpdateProductCategory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var category entities.Category

	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		log.Printf("Unmarshal err:%v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	err = json.NewDecoder(r.Body).Decode(&category)
	if err != nil {
		log.Printf("Unmarshal err:%v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	err = h.ps.UpdateProductCategory(id, category)
	if err != nil {
		WriteErrorResponse(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) RemoveProductCategory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		log.Printf("Unmarshal err:%v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	err = h.ps.RemoveProductCategory(id)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// categories

func (h *Handler) GetAllCategories(w http.ResponseWriter, r *http.Request) {
	var jsonData []byte
	tree, err := h.cas.GetAllCaregories()
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	jsonData, err = json.MarshalIndent(tree, "", "  ")
	if err != nil {
		log.Printf("Marshal err: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.Write(jsonData)
}

func (h *Handler) GetCategoryWithProducts(w http.ResponseWriter, r *http.Request) {
	var jsonCats []byte
	var jsonProds []byte
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		log.Printf("Unmarshal err:%v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	cats, prods, err := h.cas.GetCategoryWithProducts(id)
	if err != nil {
		WriteErrorResponse(w, err)
		return
	}

	if len(cats) > 0 {
		jsonCats, err = json.MarshalIndent(cats, "", "  ")
		if err != nil {
			log.Printf("Marshal err: %v", err)
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
	} else {
		jsonCats = []byte("\nno inner categories")
	}

	if len(prods) > 0 {
		jsonProds, err = json.MarshalIndent(prods, "", "  ")
		if err != nil {
			log.Printf("Marshal error: %v", err)
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
	} else {
		jsonProds = []byte("\nno inner products")
	}
	w.Write([]byte("\ncategories:"))
	w.Write(jsonCats)
	w.Write([]byte("\nproducts:"))
	w.Write(jsonProds)

}

func (h *Handler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var category models.CategoryRequest
	err := json.NewDecoder(r.Body).Decode(&category)
	if err != nil {
		log.Printf("Unmarshal err:%v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	category.Id, err = h.cas.CreateCategory(category)
	if err != nil {
		WriteErrorResponse(w, err)
		return
	}
	w.Write([]byte(strconv.Itoa(category.Id)))
}

func (h *Handler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var category models.CategoryRequest
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		log.Printf("Unmarshal err:%v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	err = json.NewDecoder(r.Body).Decode(&category)
	if err != nil {
		log.Printf("Unmarshal err:%v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	category.Id = id
	err = h.cas.UpdateCategory(category)
	if err != nil {
		WriteErrorResponse(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// attributes
func (h *Handler) CreateAttribute(w http.ResponseWriter, r *http.Request) {
	var atr models.Attribute_db
	err := json.NewDecoder(r.Body).Decode(&atr)
	if err != nil {
		log.Printf("Unmarshal err:%v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	atr.Id, err = h.ats.CreateAttribute(atr)
	if err != nil {
		WriteErrorResponse(w, err)
		return
	}
	w.Write([]byte(strconv.Itoa(atr.Id)))
}

func (h *Handler) UpdateAttribute(w http.ResponseWriter, r *http.Request) {
	var atr models.Attribute_db
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		log.Printf("Unmarshal err:%v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	err = json.NewDecoder(r.Body).Decode(&atr)
	if err != nil {
		log.Printf("Unmarshal err:%v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	atr.Id = id
	err = h.ats.UpdateAttribute(atr)
	if err != nil {
		WriteErrorResponse(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// cart
func (h *Handler) GetCart(w http.ResponseWriter, r *http.Request) {
	var cartSessionId string
	c, err := r.Cookie("cartSessionId")
	if err != nil {
		switch {
		case errors.Is(err, http.ErrNoCookie):
			b, _ := json.MarshalIndent(entities.CartResponse{Products: []entities.CartItem{}}, "", " ")
			w.Write(b)
			return
		default:
			log.Printf("Cookie err:%v", err)
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
	} else {
		cartSessionId = c.Value
	}
	cart, err := h.cs.GetCartItems(cartSessionId)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	jsonData, err2 := json.MarshalIndent(cart, "", "  ")
	if err2 != nil {
		log.Printf("Marshal err:%v", err2)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.Write(jsonData)
}

func (h *Handler) AddToCart(w http.ResponseWriter, r *http.Request) {
	prods := entities.CartRequest{}
	var cartSessionId string
	var c *http.Cookie

	err := json.NewDecoder(r.Body).Decode(&prods)
	if err != nil {
		log.Printf("Unmarshal err:%v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	c, err = r.Cookie("cartSessionId")
	if err != nil {
		switch {
		case errors.Is(err, http.ErrNoCookie):
			cartSessionId, err = h.cs.CreateCartSession()
			if err != nil {
				fmt.Println(err)
				http.Error(w, "server error", http.StatusInternalServerError)
				return
			}
			http.SetCookie(w, &http.Cookie{
				Name:    "cartSessionId",
				Value:   cartSessionId,
				Expires: time.Now().Add(24 * time.Hour),
			})
		default:
			log.Printf("Cookie err:%v", err)
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
	} else {
		cartSessionId = c.Value
	}
	err = h.cs.AddCartItem(cartSessionId, prods)
	if err != nil {
		WriteErrorResponse(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) DeleteFromCart(w http.ResponseWriter, r *http.Request) {
	prods := entities.CartRequest{}
	err := json.NewDecoder(r.Body).Decode(&prods)
	if err != nil {
		log.Printf("Unmarshal err:%v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	c, err := r.Cookie("cartSessionId")
	if err != nil {
		switch {
		case errors.Is(err, http.ErrNoCookie):
			return
		default:
			log.Printf("Cookie err:%v", err)
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
	}
	cartSessionId := c.Value

	err = h.cs.RemoveCartItem(cartSessionId, prods)
	if err != nil {
		WriteErrorResponse(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// orders
func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	c, _ := r.Cookie("sessionId")
	sessionId := c.Value
	var ordId int
	c, err := r.Cookie("cartSessionId")
	if err != nil {
		if err == http.ErrNoCookie {
			return
		}
		log.Printf("Cookie err:%v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	cartSessionId := c.Value
	ordId, err = h.ors.CreateOrder(sessionId, cartSessionId)
	if err != nil {
		WriteErrorResponse(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:    "cartSessionId",
		Value:   "",
		Path:    "/",
		Expires: time.Now(),
		// redis 30 min
	})

	w.Write([]byte(strconv.Itoa(ordId)))
}

func (h *Handler) GetOrderById(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		log.Printf("Unmarshal err:%v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var order entities.Order
	order, err = h.ors.GetOrderById(id)
	if err != nil {
		WriteErrorResponse(w, err)
		return
	}

	jsonData, err2 := json.MarshalIndent(order, "", "  ")
	if err2 != nil {
		log.Printf("Marshal err:%v", err2)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.Write(jsonData)
}

func (h *Handler) SearchOrders(w http.ResponseWriter, r *http.Request) {
	timeStart := r.URL.Query().Get("timestart")
	timeEnd := r.URL.Query().Get("timeend")
	userId := r.URL.Query().Get("userid")
	status := r.URL.Query().Get("status")
	prodId := r.URL.Query().Get("productid")

	var userId_ int
	var prodId_ int
	searchData := models.OrderSearchData{}
	var err error
	if timeStart == "" || timeEnd == "" {
		searchData.DateStart = nil
		searchData.DateEnd = nil
	} else {
		timeStart_, err := time.Parse("2006-01-02 15:04:05", timeStart)
		timeEnd_, err2 := time.Parse("2006-01-02 15:04:05", timeEnd)
		if err != nil || err2 != nil || !timeStart_.Before(timeEnd_) {
			http.Error(w, "date is wrong", http.StatusBadRequest)
			return
		}
		searchData.DateStart = &timeStart_
		searchData.DateEnd = &timeEnd_
	}

	if userId != "" {
		userId_, err = strconv.Atoi(userId)
		if err != nil {
			http.Error(w, "user id is wrong", http.StatusBadRequest)
			return
		}
		searchData.UserId = &userId_
	}

	if status != "" {
		if !(status == "created" || status == "confirmed" || status == "rejected") {
			http.Error(w, "status is wrong", http.StatusBadRequest)
			return
		}
		searchData.Status = &status
	}

	if prodId != "" {
		prodId_, err = strconv.Atoi(r.URL.Query().Get("productid"))
		if err != nil {
			http.Error(w, "product id is wrong", http.StatusBadRequest)
			return
		}
		searchData.ProdId = &prodId_
	}

	orders, err2 := h.ors.SearchOrders(searchData)
	if err2 != nil {
		WriteErrorResponse(w, err2)
		return
	}

	jsonData, err3 := json.MarshalIndent(orders, "", "  ")
	if err3 != nil {
		log.Printf("Marshal err:%v", err3)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.Write(jsonData)
	w.Write([]byte("\n\n end of report"))
}

func (h *Handler) SetOrderStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var status struct {
		Status string `json:"status"`
	}
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		log.Printf("Unmarshal err:%v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	err = json.NewDecoder(r.Body).Decode(&status)
	if err != nil || !(status.Status == "confirmed" || status.Status == "rejected") {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	err = h.ors.SetOrderStatus(id, status.Status)
	if err != nil {
		WriteErrorResponse(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderId, err := strconv.Atoi(vars["id"])
	if err != nil {
		log.Printf("Unmarshal err:%v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	sessionId, err := r.Cookie("sessionId")
	if err != nil {
		log.Printf("Cookie err:%v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	err = h.ors.CancelOrder(orderId, sessionId.Value)
	if err != nil {
		WriteErrorResponse(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetCurrentUserOrders(w http.ResponseWriter, r *http.Request) {
	sessionId, _ := r.Cookie("sessionId")

	orders, err := h.ors.GetCurrentUserOrders(sessionId.Value)
	if err != nil {
		WriteErrorResponse(w, err)
		return
	}

	jsonData, err2 := json.MarshalIndent(orders, "", "  ")
	if err2 != nil {
		log.Printf("Marshal err:%v", err2)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.Write(jsonData)
	w.Write([]byte("\n\n end of report"))
}

// middleware
func (h *Handler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionId, err := r.Cookie("sessionId")
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		ok, e := h.us.CheckAuth(sessionId.Value)
		if !ok {
			if e != nil {
				http.Error(w, "server error", http.StatusInternalServerError)
			} else {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
			}
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) ManagerAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionId, err := r.Cookie("sessionId")
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		ok, err := h.us.CheckAccess(sessionId.Value)
		if !ok {
			if err != nil {
				log.Printf("CheckSession: %v", err)
				http.Error(w, "server error", http.StatusInternalServerError)

			} else {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
			}
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) ErrorHandleMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic occured: %v \n stacktrace: %v", rec, string(debug.Stack()))
				http.Error(w, "something went wrong, contact with service administration", http.StatusBadGateway)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func WriteErrorResponse(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, models.ErrServerError):
		http.Error(w, err.Error(), http.StatusInternalServerError)
	case errors.Is(err, models.ErrUnautorized):
		http.Error(w, err.Error(), http.StatusUnauthorized)
	case errors.Is(err, models.ErrBadRequest):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, models.ErrNotFoundError):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, models.ErrNotAllowed):
		http.Error(w, err.Error(), http.StatusNotAcceptable)
	}
}
