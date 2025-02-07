package services

import (
	"log"
	"toyStore/entities"
	"toyStore/models"
	"toyStore/repository"

	"github.com/google/uuid"
)

type CartService struct {
	pr repository.ProductRepository
	cr repository.CartRepository
}

func NewCartService(productRepo repository.ProductRepository, cartRepo repository.CartRepository) CartService {
	return CartService{
		pr: productRepo,
		cr: cartRepo,
	}
}

func (cs *CartService) AddCartItem(cartSessionId string, product entities.CartRequest) (err error) {
	p, ex, e := cs.pr.GetProductById(product.ProductId)
	if e != nil {
		err = e
		return
	}
	if !ex {
		log.Printf("Product does not exist")
		err = models.ErrBadRequest
		return
	}
	if !p.Available || p.Quantity < product.Quantity {
		log.Printf("the required quantity of products is not available")
		err = models.ErrNotAllowed
		return
	}
	err = cs.cr.AddCartItem(cartSessionId, product)
	return
}

func (cs *CartService) RemoveCartItem(cartSessionId string, product entities.CartRequest) (err error) {
	err = cs.cr.RemoveCartItem(cartSessionId, product)
	return
}

func (cs *CartService) CreateCartSession() (cartSessionId string, err error) {
	cartSessionId = uuid.NewString()
	cart := entities.Cart{}
	cart.Items = make(map[int]int)
	err = cs.cr.SetCart(cartSessionId, cart)
	return
}

func (cs *CartService) GetCartItems(cartSessionId string) (resp entities.CartResponse, err error) {
	cart, e := cs.cr.GetCart(cartSessionId)
	if e != nil {
		err = e
		return
	}
	items := []entities.CartItem{}
	var totalPrice float64
	for key, value := range cart.Items {
		p, _, e := cs.pr.GetProductById(key)
		if e != nil {
			err = e
			return
		}
		prodCart := entities.CartItem{
			Id:        p.Id,
			Name:      p.Name,
			Quantity:  value,
			Price:     p.Price,
			SumPrice:  float64(value) * p.Price,
			Available: p.Available,
		}
		totalPrice = totalPrice + prodCart.SumPrice
		items = append(items, prodCart)
	}
	resp = entities.CartResponse{
		Products:   items,
		TotalPrice: totalPrice,
	}
	return
}

func (cs *CartService) CheckCart(cartSessionId string) (hasItems bool, err error) {
	cart, e := cs.cr.GetCart(cartSessionId)
	if e != nil {
		err = e

		return
	}
	if len(cart.Items) > 0 {
		hasItems = true
	}
	return
}
