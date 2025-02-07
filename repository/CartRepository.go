package repository

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"
	"toyStore/entities"
	"toyStore/models"

	"github.com/redis/go-redis/v9"
)

type CartRepository interface {
	SetCart(cartSessionId string, cart entities.Cart) (err error)
	GetCart(cartSessionId string) (res entities.Cart, err error)
	AddCartItem(cartSessionId string, req entities.CartRequest) (err error)
	RemoveCartItem(cartSessionId string, req entities.CartRequest) (err error)
}
type CartRepo struct {
	rdb *redis.Client
	ctx context.Context
}

func NewCartRepository(redis_conn *redis.Client, _ctx context.Context) (CartRepository, error) {
	if redis_conn == nil {
		return nil, errors.New("conn must be non-nil")
	}
	err := redis_conn.Ping(_ctx).Err()
	if err != nil {
		return nil, err
	}
	return &CartRepo{
		rdb: redis_conn,
		ctx: _ctx,
	}, nil
}

func (c *CartRepo) SetCart(cartSessionId string, cart entities.Cart) (err error) {
	jsonData, err := json.Marshal(cart)
	if err != nil {
		log.Printf("CreateCartSession: ошибка Marshal: %v", err)
		err = models.ErrServerError
		return
	}
	err = c.rdb.Set(c.ctx, cartSessionId, jsonData, 24*time.Hour).Err()
	if err != nil {
		log.Printf("CreateCartSession: Ошибка сохранения в Redis: %v", err)
		err = models.ErrServerError
	}
	return
}

func (c *CartRepo) GetCart(cartSessionId string) (res entities.Cart, err error) {
	res = entities.Cart{}
	val, e := c.rdb.Get(c.ctx, cartSessionId).Result()
	if e != nil {
		if e == redis.Nil {
			return
		}
		log.Printf("GetCart: Ошибка получения из Redis: %v", err)
		err = models.ErrServerError
		return
	}
	err = json.Unmarshal([]byte(val), &res)
	if err != nil {
		log.Printf("GetCart: Ошибка Unmarshal: %v", err)
		err = models.ErrServerError
	}
	return
}

func (c *CartRepo) AddCartItem(cartSessionId string, req entities.CartRequest) (err error) {
	cart, e := c.GetCart(cartSessionId)
	if e != nil {
		err = e
		return
	}
	cart.Items[req.ProductId] = cart.Items[req.ProductId] + req.Quantity
	err = c.SetCart(cartSessionId, cart)
	return
}

func (c *CartRepo) RemoveCartItem(cartSessionId string, req entities.CartRequest) (err error) {
	cart, e := c.GetCart(cartSessionId)
	if e != nil {
		err = e
		return
	}
	if _, ok := cart.Items[req.ProductId]; !ok {
		return
	}
	if req.Quantity == 0 {
		req.Quantity = 1
	}
	if cart.Items[req.ProductId] > req.Quantity {
		cart.Items[req.ProductId] = cart.Items[req.ProductId] - req.Quantity
	} else {
		//discard
		delete(cart.Items, req.ProductId)
	}
	err = c.SetCart(cartSessionId, cart)
	return
}
