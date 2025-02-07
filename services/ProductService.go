package services

import (
	"errors"
	"log"
	"toyStore/entities"
	"toyStore/models"
	"toyStore/repository"
)

type ProductService struct {
	pr repository.ProductRepository
	ar repository.AttributeRepository
	cr repository.CategoryRepository
}

func NewProductService(pRepo repository.ProductRepository, attrRepo repository.AttributeRepository, catRepo repository.CategoryRepository) ProductService {
	return ProductService{
		pr: pRepo,
		ar: attrRepo,
		cr: catRepo,
	}
}

func (ps *ProductService) GetProductById(prodId int) (pEnt entities.Product, err error) {
	var pModel models.Product_db
	var attrs []entities.ProductAttribute
	var cat entities.Category
	var exists bool
	pModel, exists, err = ps.pr.GetProductById(prodId)
	if err != nil {
		return
	}
	if !exists {
		err = models.ErrNotFoundError
		return
	}
	attrs, err = ps.ar.GetProductAttributes(prodId)
	if err != nil {
		return
	}
	cat, err = ps.pr.GetProductCategory(prodId)
	if err != nil {
		return
	}
	pEnt.Id = pModel.Id
	pEnt.Name = pModel.Name
	pEnt.Manufacturer = pModel.Manufacturer
	pEnt.Price = pModel.Price
	pEnt.Quantity = pModel.Quantity
	pEnt.Description = pModel.Description.String
	pEnt.Available = pModel.Available

	pEnt.Attributes = attrs
	pEnt.Category = cat
	return
}

func (ps *ProductService) CreateProduct(pModel models.Product) (err error) {
	err = ps.pr.CreateProduct(pModel)
	return
}

func (ps *ProductService) UpdateProductById(pModel models.Product) (pNewModel models.Product_db, err error) {
	pNewModel, err = ps.pr.UpdateProductById(pModel)
	return
}

func (ps *ProductService) UpdateProductAttributes(prodId int, attrs []entities.ProductAttribute) (attrsInvaild []entities.ProductAttribute, err error) {
	if len(attrs) == 0 {
		log.Printf("UpdateProductAttributes: attributes can not be empty")
		err = models.ErrNotAllowed
		return
	}

	_, exists, e := ps.pr.GetProductById(prodId)
	if e != nil {
		err = e
		return
	}
	if !exists {
		log.Printf("UpdateProductAttributes: unvalid product Id")
		err = models.ErrNotAllowed
		return
	}

	attrsInvaild, err = ps.ar.UpdateProductAttributes(prodId, attrs)
	return
}

func (ps *ProductService) RemoveProductAttributes(prodId int, attrsId []int) (rowsRemoved int, err error) {
	attrs := []entities.ProductAttribute{}
	if len(attrsId) == 0 {
		err = errors.New("atributes are undefined")
		return
	}
	for _, v := range attrsId {
		a := entities.ProductAttribute{Id: v}
		attrs = append(attrs, a)
	}
	rowsRemoved, err = ps.ar.RemoveProductAttributes(prodId, attrs)
	return
}

func (ps *ProductService) UpdateProductCategory(prodId int, cat entities.Category) (err error) {
	var ex bool
	_, ex, err = ps.pr.GetProductById(prodId)
	if err != nil {
		return
	}
	if !ex {
		log.Printf("Product does not exist")
		err = models.ErrNotAllowed
		return
	}
	ex, err = ps.cr.CaregoryExist(cat.Id)
	if err != nil {
		return
	}
	if !ex {
		log.Printf("Caategory does not exist")
		err = models.ErrNotAllowed
		return
	}
	err = ps.pr.SetProductCategory(prodId, cat)
	return
}

func (ps *ProductService) RemoveProductCategory(prodId int) (err error) {
	var ex bool
	_, ex, err = ps.pr.GetProductById(prodId)
	if err != nil {
		return
	}
	if !ex {
		log.Printf("Product does not exist")
		err = models.ErrNotAllowed
		return
	}
	err = ps.pr.RemoveProductCategory(prodId)
	return
}
