package services

import (
	"log"
	"toyStore/entities"
	"toyStore/models"
	"toyStore/repository"
)

type AttributeService struct {
	ar repository.AttributeRepository
}

func NewAttributeService(attrRepo repository.AttributeRepository) AttributeService {
	return AttributeService{
		ar: attrRepo,
	}
}

func (ats *AttributeService) GetProductAttributes(prodId int) (attrs []entities.ProductAttribute, err error) {
	attrs, err = ats.ar.GetProductAttributes(prodId)
	return
}

func (ats *AttributeService) CreateAttribute(atr models.Attribute_db) (newAtrId int, err error) {
	if atr.Name == "" {
		log.Printf("attribute name can not be empty")
		err = models.ErrNotAllowed
		return
	}
	newAtrId, err = ats.ar.CreateAttribute(atr)
	return
}

func (ats *AttributeService) UpdateAttribute(atr models.Attribute_db) (err error) {
	if atr.Name == "" {
		log.Printf("attribute name can not be empty")
		err = models.ErrBadRequest
		return
	}
	err = ats.ar.UpdateAttribute(atr)
	return
}
