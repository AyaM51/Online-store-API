package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"toyStore/entities"
	"toyStore/models"
)

type AttributeRepository interface {
	GetProductAttributes(prodId int) (attrs []entities.ProductAttribute, err error)
	UpdateProductAttributes(prodId int, attrs []entities.ProductAttribute) (attrsInvaild []entities.ProductAttribute, err error)
	RemoveProductAttributes(prodId int, attrsId []entities.ProductAttribute) (rowsRemoved int, err error)
	CreateAttribute(atr models.Attribute_db) (newAtrId int, err error)
	UpdateAttribute(atr models.Attribute_db) (err error)
}

type AttrRepo struct {
	db *sql.DB
}

func NewAttributeRepository(conn *sql.DB) (AttributeRepository, error) {
	if conn == nil {
		return nil, errors.New("conn must be non-nil")
	}
	err := conn.Ping()
	if err != nil {
		return nil, err
	}
	return &AttrRepo{
		db: conn,
	}, nil
}

func (a *AttrRepo) GetProductAttributes(prodId int) (attrs []entities.ProductAttribute, err error) {
	rows, e := a.db.Query("SELECT Attributes.Id, Attributes.Name, ProductsAttributes.Value FROM ProductsAttributes JOIN Attributes ON ProductsAttributes.AttributeId=Attributes.Id where ProductId =$1", prodId)
	if e != nil {
		log.Printf("GetProductAttributes[1]: %v", e)
		err = models.ErrServerError
		return
	}
	for rows.Next() {
		attr := entities.ProductAttribute{}
		err = rows.Scan(&attr.Id, &attr.Name, &attr.Value)
		if err != nil {
			log.Printf("GetProductAttributes[2]: %v", err)
			err = models.ErrServerError
			return
		}
		attrs = append(attrs, attr)
	}
	return
}

func (a *AttrRepo) RemoveProductAttributes(prodId int, attrsId []entities.ProductAttribute) (rowsRemoved int, err error) {
	var queryParams []any
	var query string
	var count int

	queryStart := "DELETE FROM ProductsAttributes WHERE AttributeId IN "
	query, queryParams, count = a.buildQueryFromSlice(attrsId)
	query = fmt.Sprintf("%v AND ProductId =$%d", query, count+1)
	queryParams = append(queryParams, prodId)
	log.Printf("query:%v%v", queryStart, query)
	rows, e := a.db.Exec(queryStart+query, queryParams...)
	if e != nil {
		log.Printf("RemoveProductAttributes[1]: %v", e)
		err = models.ErrServerError
		return
	}

	var r int64
	r, err = rows.RowsAffected()
	rowsRemoved = int(r)
	log.Printf("aff %v", rowsRemoved)
	return
}

func (a *AttrRepo) UpdateProductAttributes(prodId int, attrs []entities.ProductAttribute) (attrsInvaild []entities.ProductAttribute, err error) {
	var query string
	var queryParams []any
	count := 0
	var rows *sql.Rows
	attrsValid := make(map[int]entities.ProductAttribute) // id, name

	//проверка, существуют ли атрибуты в Attributes
	queryStart := "SELECT Id FROM Attributes WHERE Id IN "
	query, queryParams, count = a.buildQueryFromSlice(attrs)

	rows, err = a.db.Query(queryStart+query, queryParams...)
	log.Printf("query1: %v %v", queryStart, query)
	if err != nil {
		log.Printf("UpdateProductAttributes[1]: %v", err)
		err = models.ErrServerError
		return
	}
	for rows.Next() {
		atr := entities.ProductAttribute{}
		err = rows.Scan(&atr.Id) // &atr.Name
		if err != nil {
			log.Printf("UpdateProductAttributes[2]: %v", err)
			err = models.ErrServerError
			return
		}
		attrsValid[atr.Id] = atr
	}

	//проверка, дублируются ли строки в ProductsAttributes
	queryStart = "SELECT AttributeId FROM ProductsAttributes WHERE AttributeId IN "
	query = query + " AND ProductId =" + fmt.Sprintf("$%d", count+1)
	queryParams = append(queryParams, prodId)
	rows, err = a.db.Query(queryStart+query, queryParams...)
	log.Printf("query2: %v %v", queryStart, query)
	if err != nil {
		log.Printf("UpdateProductAttributes[3]: %v", err)
		err = models.ErrServerError
		return
	}
	// удаляем дубликаты, если были

	var attrsDub []entities.ProductAttribute
	attrsDub, err = a.buildQueryParams(rows)
	if err != nil {
		return
	}
	if len(attrsDub) > 0 {
		query, queryParams, count = a.buildQueryFromSlice(attrsDub)
		queryStart = "DELETE FROM ProductsAttributes WHERE AttributeId IN "
		query = fmt.Sprintf("%v AND ProductId = $%d ", query, count+1)
		queryParams = append(queryParams, prodId)
		_, err = a.db.Exec(queryStart+query, queryParams...)
		log.Printf("query3: %v %v", queryStart, query)
		if err != nil {
			log.Printf("UpdateProductAttributes[4]: %v", err)
			err = models.ErrServerError
			return
		}
	}

	// валидные вставляем, невалидные возвращаем
	count = 0
	queryStart = "INSERT INTO ProductsAttributes (ProductId, AttributeId, Value) VALUES "
	query = ""
	queryParams = nil
	for _, v := range attrs {
		if _, keyExists := attrsValid[v.Id]; !keyExists { // поиск невалидных
			attrsInvaild = append(attrsInvaild, entities.ProductAttribute{Id: v.Id, Name: v.Name, Value: v.Value})
		} else {
			query = query + fmt.Sprintf("($%d, $%d, $%d), ", count+1, count+2, count+3)
			queryParams = append(queryParams, prodId, v.Id, v.Value)
			count = count + 3
		}
	}
	if len(query) > 0 {
		query = query[0 : len(query)-2]
		_, err = a.db.Exec(queryStart+query, queryParams...)
		log.Printf("query4: %v %v", queryStart, query)
		if err != nil {
			log.Printf("UpdateProductAttributes[6]: %v", err)
			err = models.ErrServerError
			return
		}
	}
	return
}

func (a *AttrRepo) CreateAttribute(atr models.Attribute_db) (newAtrId int, err error) {
	valid, e := a.atributeNameUnique(atr.Name)
	if e != nil {
		err = e
		return
	}
	if !valid {
		log.Printf("Attribute name is not unique")
		err = models.ErrNotAllowed
		return
	}

	err = a.db.QueryRow("INSERT INTO Attributes (Name) VALUES ($1) RETURNING Id", atr.Name).Scan(&newAtrId)
	if err != nil {
		log.Printf("CreateAttribute: %v", err)
		err = models.ErrServerError
	}
	return
}

func (a *AttrRepo) UpdateAttribute(atr models.Attribute_db) (err error) {
	row := a.db.QueryRow("SELECT Name FROM Attributes WHERE Id=$1", atr.Id)
	var name string
	err = row.Scan(&name)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Attribute with id '%v' does not exist", atr.Id)
			err = models.ErrNotAllowed
		} else {
			log.Printf("CreateAttribute: %v", err)
			err = models.ErrServerError
		}
		return
	}

	valid, e := a.atributeNameUnique(atr.Name)
	if e != nil {
		err = e
		return
	}
	if !valid {
		log.Printf("Attribute name is not unique")
		err = models.ErrNotAllowed
		return
	}

	_, err = a.db.Exec("UPDATE Attributes SET Name=$1 WHERE Id=$2", atr.Name, atr.Id)
	if err != nil {
		log.Printf("CreateAttribute: %v", err)
		err = models.ErrServerError
	}
	return
}

func (a *AttrRepo) atributeNameUnique(name string) (bool, error) {
	row := a.db.QueryRow("SELECT Name FROM Attributes WHERE Name=$1", name)
	err := row.Scan(&name)
	if err != nil {
		if err == sql.ErrNoRows {
			return true, nil
		}
		return false, models.ErrServerError
	}
	return false, nil
}

func (a *AttrRepo) buildQueryParams(rows *sql.Rows) (params []entities.ProductAttribute, err error) {
	for rows.Next() {
		var atr entities.ProductAttribute
		err = rows.Scan(&atr.Id)
		if err != nil {
			log.Printf("buildQueryFromRows: %v", err)
			err = models.ErrServerError
			return
		}
		params = append(params, atr)
	}
	return
}

func (a *AttrRepo) buildQueryFromSlice(attrs []entities.ProductAttribute) (query string, queryParams []any, count int) {
	count = 0
	queryParams = []any{}
	query = "( "
	for _, k := range attrs {
		count = count + 1
		query = query + fmt.Sprintf("$%d, ", count)
		queryParams = append(queryParams, k.Id)
	}
	query = query[0 : len(query)-2]
	query = query + " )"
	return
}
