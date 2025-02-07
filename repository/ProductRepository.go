package repository

import (
	"database/sql"
	"errors"
	"log"
	"toyStore/entities"
	"toyStore/models"
	"unicode"
)

type ProductRepository interface {
	GetProductById(id int) (pModel models.Product_db, exists bool, err error)
	GetProductsByCategory(catId int) (prods []entities.ProductPreview, err error)
	UpdateProductById(pModel models.Product) (updatedProd models.Product_db, err error)
	CreateProduct(pModel models.Product) (err error)
	GetProductCategory(prodId int) (cat entities.Category, err error)
	SetProductCategory(prodId int, cat entities.Category) (err error)
	RemoveProductCategory(prodId int) (err error)
}

type ProductRepo struct {
	db *sql.DB
}

func NewProductRepository(conn *sql.DB) (ProductRepository, error) {
	if conn == nil {
		return nil, errors.New("conn must be non-nil")
	}
	err := conn.Ping()
	if err != nil {
		return nil, err
	}
	return &ProductRepo{
		db: conn,
	}, nil
}

func (p *ProductRepo) GetProductById(id int) (pModel models.Product_db, exists bool, err error) {
	row := p.db.QueryRow("SELECT Id, Name, Manufacturer, Quantity, Price, Description, Available FROM Products where Id = $1", id)
	err = row.Scan(&pModel.Id, &pModel.Name, &pModel.Manufacturer,
		&pModel.Quantity, &pModel.Price, &pModel.Description, &pModel.Available)

	if err != nil {
		if err == sql.ErrNoRows {
			err = nil
		} else {
			log.Printf("GetProductById: %v", err)
			err = models.ErrServerError
		}
		return
	}
	exists = true
	return
}

func (p *ProductRepo) GetProductCategory(prodId int) (cat entities.Category, err error) {
	row := p.db.QueryRow("SELECT Categories.Id, Categories.Name FROM ProductsCategories JOIN Categories ON ProductsCategories.CategoryId=Categories.Id WHERE ProductsCategories.ProductId=$1", prodId)
	err = row.Scan(&cat.Id, &cat.Name)
	if err != nil {
		log.Printf("GetProductCategory[1]: %v", err)
		if err == sql.ErrNoRows {
			err = nil
		} else {
			log.Printf("GetProductCategory: %v", err)
			err = models.ErrServerError
		}
	}
	return
}

func (p *ProductRepo) SetProductCategory(prodId int, cat entities.Category) (err error) {
	var curCatId int
	row := p.db.QueryRow("SELECT ProductsCategories.CategoryId FROM ProductsCategories WHERE ProductsCategories.ProductId=$1", prodId)
	err = row.Scan(&curCatId)
	if err != nil {
		if err == sql.ErrNoRows {
			_, err = p.db.Exec("INSERT INTO ProductsCategories (ProductId, CategoryId) VALUES ($1, $2)", prodId, cat.Id)
			if err != nil {
				log.Printf("SetProductCategory[1]: %v", err)
				err = models.ErrServerError
			}
			return
		}
		log.Printf("SetProductCategory[2]: %v", err)
		err = models.ErrServerError
		return
	}
	_, err = p.db.Exec("UPDATE ProductsCategories SET CategoryId =$1 WHERE ProductsCategories.ProductId=$2", cat.Id, prodId)
	if err != nil {
		log.Printf("SetProductCategory[3]: %v", err)
		err = models.ErrServerError
	}
	return
}

func (p *ProductRepo) RemoveProductCategory(prodId int) (err error) {
	_, err = p.db.Exec("DELETE FROM ProductsCategories WHERE ProductsCategories.ProductId=$1", prodId)
	if err != nil {
		log.Printf("RemoveProductCategory: %v", err)
		err = models.ErrServerError
	}
	return
}

func (p *ProductRepo) GetProductsByCategory(catId int) (prods []entities.ProductPreview, err error) {
	rows, e := p.db.Query("select Id, Name, Manufacturer, Price, Available FROM Products JOIN ProductsCategories ON ProductsCategories.ProductId=Products.Id where ProductsCategories.CategoryId =$1", catId)
	if e != nil {
		log.Printf("GetProductsByCategory[1]: %v", e)
		err = models.ErrServerError
		return
	}
	for rows.Next() {
		prod := entities.ProductPreview{}
		err = rows.Scan(&prod.Id, &prod.Name, &prod.Manufacturer, &prod.Price, &prod.Available)
		if err != nil {
			log.Printf("GetProductsByCategory[2]: %v", err)
			err = models.ErrServerError
			return
		}
		prods = append(prods, prod)
	}
	return
}

func (p *ProductRepo) UpdateProductById(pModel models.Product) (updatedProd models.Product_db, err error) {
	var ex bool
	_, ex, err = p.GetProductById(pModel.Id)
	if err != nil {
		return
	}
	if !ex {
		log.Printf("Product does not exist")
		err = models.ErrNotAllowed
		return
	}

	queryParams := make([]any, 0, 7)
	query := "UPDATE Products SET "
	if isValidLen(pModel.Name, 5, 30) && isValidString(pModel.Name) {
		query = query + "Name = $1, "
		queryParams = append(queryParams, pModel.Name)
	}
	if isValidLen(pModel.Manufacturer, 5, 30) && isValidString(pModel.Manufacturer) {
		query = query + "Manufacturer = $2, "
		queryParams = append(queryParams, pModel.Manufacturer)
	}
	if pModel.Quantity > 0 {
		query = query + "Quantity = $3, "
		queryParams = append(queryParams, pModel.Quantity)
	}
	if pModel.Price > 0 {
		query = query + "Price = $4, "
		queryParams = append(queryParams, pModel.Price)
	}
	if isValidLen(pModel.Description, 5, 100) && isValidString(pModel.Description) {
		query = query + "Description = $5, "
		queryParams = append(queryParams, pModel.Description)
	}
	if pModel.Available != nil {
		query = query + "Available = $6, "
		queryParams = append(queryParams, *pModel.Available)
	}
	query = query[0 : len(query)-2]
	query = query + " WHERE Id = $7"
	queryParams = append(queryParams, pModel.Id)
	_, e := p.db.Exec(query, queryParams...)
	if e != nil {
		log.Printf("UpdateProductById: %v", e)
		err = models.ErrServerError
		return
	}

	updatedProd, _, err = p.GetProductById(pModel.Id)
	if err != nil {
		return
	}
	return updatedProd, nil
}

func isValidLen(input string, minLen int, maxLen int) bool {
	inputLen := len([]rune(input))
	if inputLen < minLen || inputLen > maxLen {
		return false
	}
	return true
}

func isValidString(input string) bool {
	allowedSymbols := map[rune]bool{
		'-': true,
		' ': true,
		':': true,
		'.': true,
		',': true,
		'"': true,
	}
	for _, char := range input {
		if !(unicode.IsLetter(char) || unicode.IsDigit(char) || allowedSymbols[char]) {
			return false
		}
	}
	return true
}

func (p *ProductRepo) CreateProduct(pModel models.Product) (err error) {
	if !isValidLen(pModel.Name, 5, 30) || !isValidString(pModel.Name) {
		log.Printf("name field is invalid")
		err = models.ErrNotAllowed
		return
	}
	if !isValidLen(pModel.Manufacturer, 5, 30) || !isValidString(pModel.Manufacturer) {
		log.Printf("manufacturer field is invalid")
		err = models.ErrNotAllowed
		return
	}
	if pModel.Quantity <= 0 {
		log.Printf("quantity field is invalid")
		err = models.ErrNotAllowed
		return
	}
	if pModel.Price <= 0 {
		log.Printf("price field is invalid")
		err = models.ErrNotAllowed
		return
	}
	if !isValidLen(pModel.Description, 10, 100) || !isValidString(pModel.Description) {
		log.Printf("description field is invalid")
		err = models.ErrNotAllowed
		return
	}
	if pModel.Available == nil {
		log.Printf("available field is invalid")
		err = models.ErrNotAllowed
		return
	}
	_, e := p.db.Exec("INSERT INTO Products (Name, Manufacturer, Quantity, Price, Description, Available) VALUES ",
		pModel.Name, pModel.Manufacturer, pModel.Quantity,
		pModel.Price, pModel.Description, *pModel.Available)
	if e != nil {
		log.Printf("CreateProduct: %v", e)
		err = models.ErrServerError
	}
	return
}
