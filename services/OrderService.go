package services

import (
	"log"
	"time"
	"toyStore/entities"
	"toyStore/models"
	"toyStore/repository"
)

type OrderService struct {
	sr repository.SessionRepository
	pr repository.ProductRepository
	cr repository.CartRepository
	or repository.OrderRepository
}

func NewOrderService(sessionRepo repository.SessionRepository, productRepo repository.ProductRepository, cartRepo repository.CartRepository, orderRepo repository.OrderRepository) OrderService {
	return OrderService{
		sr: sessionRepo,
		pr: productRepo,
		cr: cartRepo,
		or: orderRepo,
	}
}

func (ors *OrderService) CreateOrder(sessionId string, cartSessionId string) (orderId int, err error) {
	uId, _, _, e := ors.sr.GetUserSessionInfo(sessionId)
	if e != nil {
		err = e
		return
	}
	var cart entities.Cart
	cart, _ = ors.cr.GetCart(cartSessionId)
	if len(cart.Items) == 0 {
		err = models.ErrBadRequest
		return
	}

	prods := []models.OrdersProducts_db{}
	var totalPrice float64
	for key, value := range cart.Items {
		var p models.Product_db
		p, _, err = ors.pr.GetProductById(key)
		if err != nil {
			log.Printf("GetCartItems: %v", err)
			err = models.ErrServerError
			return
		}
		if !p.Available {
			log.Print("product ", p.Name, " is unavailable")
			err = models.ErrNotAllowed
			return
		}
		if p.Quantity < value {
			log.Print("quantity of the product ", p.Name, " is unavailable")
			err = models.ErrNotAllowed
			return
		}
		prodOrd := models.OrdersProducts_db{
			OrderId:   p.Id,
			ProductId: p.Id,
			Quantity:  value,
			Price:     p.Price,
		}
		totalPrice = totalPrice + float64(value)*p.Price
		prods = append(prods, prodOrd)
	}

	newOrder := models.Order_db{
		Status:     "created",
		UserId:     uId,
		TotalPrice: totalPrice,
		Date:       time.Now().UTC(),
	}

	orderId, err = ors.or.CreateOrder(newOrder)
	if err != nil {
		return
	}

	var empty entities.Cart
	err = ors.or.SetOrderItems(orderId, prods)
	if err != nil {
		return
	}
	err = ors.cr.SetCart(cartSessionId, empty)
	return
}

func (ors *OrderService) GetOrderById(orderId int) (order entities.Order, err error) {
	order, err = ors.or.GetOrderById(orderId)
	return
}

func (ors *OrderService) SearchOrders(data models.OrderSearchData) (orders []entities.Order, err error) {
	orders, err = ors.or.SearchOrders(data)
	return
}

func (ors *OrderService) GetCurrentUserOrders(sessionId string) (orders []entities.Order, err error) {
	userId, _, _, e := ors.sr.GetUserSessionInfo(sessionId)
	if e != nil {
		log.Printf("GetCurrentUserOrders: %v", e)
		err = models.ErrServerError
		return
	}
	data := models.OrderSearchData{
		UserId: &userId,
	}
	orders, err = ors.or.SearchOrders(data)
	return
}

func (ors *OrderService) SetOrderStatus(orderId int, status string) (err error) {
	err = ors.or.SetOrderStatus(orderId, status)
	return
}

func (ors *OrderService) CancelOrder(orderId int, sessionId string) (err error) {
	userId, _, _, e := ors.sr.GetUserSessionInfo(sessionId)
	if e != nil {
		err = models.ErrServerError
		return
	}
	err = ors.or.CancelOrder(orderId, userId)
	return
}
