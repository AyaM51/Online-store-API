package services

import (
	"log"
	"toyStore/entities"
	"toyStore/models"
	"toyStore/repository"
)

type CategoryService struct {
	cr repository.CategoryRepository
	pr repository.ProductRepository
}

func NewCategoryService(catRepo repository.CategoryRepository, productRepo repository.ProductRepository) CategoryService {
	return CategoryService{
		cr: catRepo,
		pr: productRepo,
	}
}

func (cas *CategoryService) GetAllCaregories() (categories []entities.CategoryTree, err error) {
	categories, err = cas.cr.GetAllCategories()
	return
}

func (cas *CategoryService) GetCategoryWithProducts(catId int) (innerCats []entities.Category, innerProds []entities.ProductPreview, err error) {
	ex, e := cas.cr.CaregoryExist(catId)
	if e != nil {
		err = e
		return
	}
	if !ex {
		err = models.ErrBadRequest
		log.Printf("GetCategoryWithProducts: category does not exist")
		return
	}

	innerCats, err = cas.cr.GetSubCategories(catId)
	if err != nil {
		return
	}
	innerProds, err = cas.pr.GetProductsByCategory(catId)
	return
}

func (cas *CategoryService) GetSubCaregories(catId int) (categories []entities.Category, err error) {
	categories, err = cas.cr.GetSubCategories(catId)
	return
}

func (cas *CategoryService) CreateCategory(cat models.CategoryRequest) (newCatId int, err error) {
	if cat.Name == "" {
		log.Printf("Category name can not be empty")
		err = models.ErrNotAllowed
		return
	}
	newCatId, err = cas.cr.CreateCategory(cat)
	return
}

func (cas *CategoryService) UpdateCategory(cat models.CategoryRequest) (err error) {
	if cat.Id == 0 {
		log.Printf("category id can not be empty")
		err = models.ErrNotAllowed
		return
	}
	err = cas.cr.UpdateCategory(cat)
	return
}
