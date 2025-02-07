package models

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

var ErrBadRequest = errors.New("bad request")
var ErrUnautorized = errors.New("unautorized")
var ErrServerError = errors.New("server error")
var ErrNotFoundError = errors.New("not found")
var ErrNotAllowed = errors.New("not acceptable")

type Credentials struct {
	Password string `json:"password" db:"Password"`
	Username string `json:"username" db:"Nickname"`
	Role     string `json:"role" db:"Role"`
}

type PasswordData struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type Product struct {
	Id           int     `json:"id" db:"Id"`
	Name         string  `json:"name" db:"Name"`
	Manufacturer string  `json:"manufacturer" db:"Manufacturer"`
	Quantity     int     `json:"quantity" db:"Quantity"`
	Price        float64 `json:"price" db:"Price"`
	Description  string  `json:"description" db:"Description"`
	Available    *bool   `json:"available,omitempty"`
}

type Category_db struct {
	Id       int
	Name     string
	ParentId sql.NullInt64
}

type UserData struct {
	Id       int
	Nickname string `json:"username" db:"Nickname"`
	Role     string `json:"role" db:"Role"`
}

type OrderSearchData struct {
	DateStart *time.Time
	DateEnd   *time.Time
	UserId    *int
	Status    *string
	ProdId    *int
}

type CategoryRequest struct {
	Id       int
	Name     string
	ParentId NullInt `json:"parent_id,omitempty"`
}

type NullInt struct {
	Valid bool
	Value int
}

func (ni *NullInt) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		ni.Valid = true
		ni.Value = 0
		return nil
	}
	ni.Valid = true
	return json.Unmarshal(data, &ni.Value)
}

type Product_db struct {
	Id           int            `json:"id" db:"Id"`
	Name         string         `json:"name" db:"Name"`
	Manufacturer string         `json:"manufacturer" db:"Manufacturer"`
	Quantity     int            `json:"quantity" db:"Quantity"`
	Price        float64        `json:"price" db:"Price"`
	Description  sql.NullString `json:"description" db:"Description"`
	Available    bool           `json:"available" db:"Available"`
}

type ProductsCategories_db struct {
	ProductId  int
	CategoryId int
}

type ProductsAttributes_db struct {
	ProductId   int
	AttributeId int
	Value       string
}

type Order_db struct {
	Id         int
	UserId     int
	Date       time.Time
	TotalPrice float64
	Status     string
}

type OrdersProducts_db struct {
	Id        int
	OrderId   int
	ProductId int
	Quantity  int
	Price     float64
}

type Attribute_db struct {
	Id   int
	Name string
}

type User_db struct {
	Id       int
	Nickname string `json:"username" db:"Nickname"`
	Password string `json:"password" db:"Password"`
	Role     string `json:"role" db:"Role"`
}
