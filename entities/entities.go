package entities

import (
	"time"
	"toyStore/models"
)

type Product struct {
	Id           int
	Name         string
	Manufacturer string
	Quantity     int
	Price        float64
	Description  string
	Available    bool
	Category     Category
	Attributes   []ProductAttribute
}

type ProductPreview struct {
	Id           int
	Name         string
	Manufacturer string
	Price        float64
	Available    bool
}

type ProductOrderFormat struct {
	Id           int
	Name         string
	Manufacturer string
	Quantity     int
	Price        float64
	TotalPrice   float64
}

type CartItem struct {
	Id        int
	Name      string
	Quantity  int
	Price     float64
	SumPrice  float64
	Available bool
}

type Cart struct {
	Items map[int]int //=map[id]quantity
}

type CartRequest struct {
	ProductId int
	Quantity  int
}

type CartResponse struct {
	Products   []CartItem
	TotalPrice float64
}

type Category struct {
	Id   int    `json:"category_id"`
	Name string `json:"category_name"`
}

type CategoryTree struct {
	ID       int            `json:"-"`
	ParentID int            `json:"-"`
	Name     string         `json:"name"`
	Children []CategoryTree `json:"children,omitempty"`
}

type ProductAttribute struct {
	Id    int    `json:"attribute_id"`
	Name  string `json:"attribute_name"`
	Value string `json:"attribute_value"`
}

type Order struct {
	OrderId    int
	Date       time.Time
	Status     string
	TotalPrice float64
	UserData   models.UserData
	Products   []ProductOrderFormat
}
