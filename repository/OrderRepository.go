package repository

import (
	"database/sql"
	"errors"
	"log"
	"strconv"
	"time"
	"toyStore/entities"
	"toyStore/models"
)

type OrderRepository interface {
	CreateOrder(order models.Order_db) (orderId int, err error)
	SetOrderItems(orderId int, prods []models.OrdersProducts_db) (err error)
	GetOrderItems(orderId int) (prods []entities.ProductOrderFormat, err error)
	GetOrderById(orderId int) (order entities.Order, err error)
	SearchOrders(data models.OrderSearchData) (order []entities.Order, err error)
	SetOrderStatus(orderId int, status string) (err error)
	CancelOrder(orderId int, userId int) (err error)
}
type OrderRepo struct {
	db *sql.DB
}

func NewOrderRepository(conn *sql.DB) (OrderRepository, error) {
	if conn == nil {
		return nil, errors.New("conn must be non-nil")
	}
	err := conn.Ping()
	if err != nil {
		return nil, err
	}
	return &OrderRepo{
		db: conn,
	}, nil
}

func (o *OrderRepo) CreateOrder(order models.Order_db) (orderId int, err error) {
	var oId int64
	e := o.db.QueryRow("INSERT INTO Orders (UserId, Date, TotalPrice, Status) VALUES ($1,$2,$3,$4) RETURNING id", order.UserId, order.Date, order.TotalPrice, order.Status).Scan(&oId)
	if e != nil {
		log.Printf("CreateOrder: %v", e)
		err = models.ErrServerError
		return
	}
	orderId = int(oId)
	return
}

func (o *OrderRepo) SetOrderItems(orderId int, prods []models.OrdersProducts_db) (err error) {
	for _, v := range prods {
		_, err = o.db.Exec("INSERT INTO OrdersProducts (OrderId, ProductId, Quantity, Price) VALUES ($1, $2, $3, $4)", orderId, v.ProductId, v.Quantity, v.Price)
		if err != nil {
			log.Printf("SetOrderItems: %v", err)
			err = models.ErrServerError
			return
		}
	}
	return
}

func (o *OrderRepo) GetOrderItems(orderId int) (prods []entities.ProductOrderFormat, err error) {
	rows, e := o.db.Query("SELECT ProductId, Quantity, Price FROM OrdersProducts WHERE OrderId=$1", orderId)
	if e != nil {
		log.Printf("GetOrderItems[1]: %v", e)
		err = models.ErrServerError
		return
	}

	for rows.Next() {
		prod := entities.ProductOrderFormat{}
		err = rows.Scan(&prod.Id, &prod.Price, &prod.TotalPrice)
		if err != nil {
			log.Printf("GetOrderItems[2]: %v", err)
			err = models.ErrServerError
			return
		}

		row := o.db.QueryRow("SELECT Id, Name, Manufacturer FROM Products WHERE Id = $1", prod.Id)
		err = row.Scan(&prod.Id, &prod.Name, &prod.Manufacturer)
		if err != nil {
			log.Printf("GetOrderItems[3]: %v", err)
			err = models.ErrServerError
			return
		}
		prods = append(prods, prod)
	}
	return
}

func (o *OrderRepo) GetOrderById(orderId int) (order entities.Order, err error) {
	row := o.db.QueryRow("SELECT Id, UserId, Date, TotalPrice, Status FROM Orders WHERE Id=", orderId)
	var or models.Order_db
	err = row.Scan(&or)
	if err != nil {
		if err == sql.ErrNoRows {
			err = models.ErrNotFoundError
		} else {
			log.Printf("GetOrderById: %v", err)
			err = models.ErrServerError
		}
		return
	}

	row = o.db.QueryRow("SELECT Id, Nickname, Role FROM Users WHERE Id=", or.UserId)
	var usr models.UserData
	err = row.Scan(&usr)
	if err != nil {
		log.Printf("GetOrderById: %v", err)
		err = models.ErrServerError
		return
	}

	prods, e := o.GetOrderItems(orderId)
	if e != nil {
		err = e
		return
	}

	order = entities.Order{
		OrderId:    orderId,
		Date:       or.Date,
		Status:     or.Status,
		TotalPrice: or.TotalPrice,
		UserData:   usr,
		Products:   prods,
	}
	return
}

func (o *OrderRepo) SetOrderStatus(orderId int, status string) (err error) {
	row := o.db.QueryRow("SELECT Status FROM Orders WHERE Id=$1", orderId)
	var or models.Order_db
	err = row.Scan(&or.Status)
	if err != nil {
		if err == sql.ErrNoRows {
			err = models.ErrNotFoundError
		} else {
			log.Printf("SetOrderStatus[1]: %v", err)
			err = models.ErrServerError
		}
		return
	}
	if or.Status != "created" {
		log.Printf("you can not set status to this order. Current status is %v", or.Status)
		err = models.ErrNotAllowed
		return
	}

	if status == "confirmed" {
		res, e := o.db.Query("SELECT ProductId, Quantity FROM OrdersProducts WHERE OrderId=$1", orderId)
		if e != nil {
			log.Printf("SetOrderStatus[2]: %v", e)
			err = models.ErrServerError
			return
		}
		for res.Next() {
			var prodId, quantity int
			_ = res.Scan(&prodId, &quantity)
			var dbQuantity int
			var dbAvailable bool

			err = o.db.QueryRow("SELECT Quantity, Available FROM Products where Id = $1", prodId).Scan(&dbQuantity, &dbAvailable)
			if err != nil {
				log.Printf("SetOrderStatus[3]: %v", err)
				err = models.ErrServerError
				return
			}
			if quantity > dbQuantity || !dbAvailable {
				log.Printf("required product quantity unavailable in db")
				err = models.ErrNotAllowed
				return
			}

			_, err = o.db.Exec("UPDATE Products SET Quantity=Quantity-$1 WHERE Id=$2", quantity, prodId)
			if err != nil {
				log.Printf("SetOrderStatus[3]: %v", err)
				err = models.ErrServerError
				return
			}
		}
	}

	_, err = o.db.Exec("UPDATE Orders SET Status=$1 WHERE Id=$2", status, orderId)
	if err != nil {
		log.Printf("SetOrderStatus[4]: %v", err)
		err = models.ErrServerError
		return
	}
	return
}

func (o *OrderRepo) SearchOrders(data models.OrderSearchData) (orders []entities.Order, err error) {
	var query string
	var queryParams []any
	var count int

	query = "SELECT Orders.Id, Orders.UserId, Orders.Date, Orders.TotalPrice, Orders.Status FROM Orders WHERE "

	if data.ProdId != nil {
		query = query[0 : len(query)-6]
		query = query + "JOIN OrdersProducts On Orders.Id = OrdersProducts.OrderId WHERE "
	}

	if data.DateStart != nil && data.DateEnd != nil {
		query = query + "Date BETWEEN $1 AND $2 AND "
		count = count + 2
		queryParams = append(queryParams, data.DateStart, data.DateStart)
	}

	if data.UserId != nil {
		count = count + 1
		query = query + "UserId=$" + strconv.Itoa(count) + " AND "
		queryParams = append(queryParams, data.UserId)
	}

	if data.Status != nil {
		count = count + 1
		query = query + "Status=$" + strconv.Itoa(count) + " AND "
		queryParams = append(queryParams, data.Status)
	}

	if data.ProdId != nil {
		count = count + 1
		query = query + "OrdersProducts.ProductId=$" + strconv.Itoa(count) + " AND "
		queryParams = append(queryParams, data.ProdId)
	}
	if count > 0 {
		query = query[0 : len(query)-4] //AND
	} else {
		query = query[0 : len(query)-6] //WHERE
	}
	query = query + "ORDER BY Orders.Id"

	rows, e := o.db.Query(query, queryParams...)
	if e != nil {
		log.Printf("SearchOrders: %v", e)
		err = models.ErrServerError
		return
	}

	for rows.Next() {
		ord := entities.Order{}
		err = rows.Scan(&ord.OrderId, &ord.UserData.Id, &ord.Date, &ord.TotalPrice, &ord.Status)
		if err != nil {
			log.Printf("SearchOrders: %v", err)
			err = models.ErrServerError
			return
		}

		rowUser := o.db.QueryRow("SELECT Nickname, Role FROM Users where Id = $1", ord.UserData.Id)
		e2 := rowUser.Scan(&ord.UserData.Nickname, &ord.UserData.Role)
		if e2 != nil {
			log.Printf("SearchOrders: %v", e2)
			err = models.ErrServerError
			return
		}

		rowsProds, e3 := o.db.Query("SELECT OrdersProducts.ProductId, OrdersProducts.Quantity, OrdersProducts.Price, Products.Name, Products.Manufacturer FROM OrdersProducts JOIN Products ON OrdersProducts.ProductId=Products.Id WHERE OrdersProducts.OrderId = $1", ord.OrderId)
		if e3 != nil {
			log.Printf("SearchOrders: %v", e3)
			err = models.ErrServerError
			return
		}
		for rowsProds.Next() {
			var prod entities.ProductOrderFormat
			e3 = rowsProds.Scan(&prod.Id, &prod.Quantity, &prod.Price, &prod.Name, &prod.Manufacturer)
			if e3 != nil {
				log.Printf("SearchOrders: %v", e3)
				err = models.ErrServerError
				return
			}
			prod.TotalPrice = prod.Price * float64(prod.Quantity)
			ord.Products = append(ord.Products, prod)
		}
		orders = append(orders, ord)
	}

	log.Println(query)
	if len(orders) == 0 {
		err = models.ErrNotFoundError
	}
	return
}

func (o *OrderRepo) CancelOrder(orderId int, userId int) (err error) {
	row := o.db.QueryRow("SELECT  Date, Status FROM Orders WHERE Id=$1 AND UserId=$2", orderId, userId)
	var or models.Order_db
	err = row.Scan(&or.Date, &or.Status)
	if err != nil {
		if err == sql.ErrNoRows {
			err = models.ErrNotFoundError
		} else {
			log.Printf("SetOrderStatus[1]: %v", err)
			err = models.ErrServerError
		}
		return
	}
	if time.Duration(time.Since(or.Date.UTC()).Minutes()) > time.Duration(10*time.Minute) {
		log.Printf("you can not cancel this order")
		log.Printf("\n %v - %v", time.Duration(time.Since(or.Date.UTC()).Minutes()), time.Since(or.Date.UTC()).Minutes())
		err = models.ErrNotAllowed
		return
	}

	_, err = o.db.Exec("UPDATE Orders SET Status=$1 WHERE Id=$2", "cancelled", orderId)
	if err != nil {
		log.Printf("CancelOrder: %v", err)
		err = models.ErrServerError
		return
	}
	return
}
