package repository

import (
	"database/sql"
	"errors"
	"log"
	"toyStore/entities"
	"toyStore/models"
)

type CategoryRepository interface {
	GetAllCategories() ([]entities.CategoryTree, error)
	GetSubCategories(catId int) (cats []entities.Category, err error)
	CaregoryExist(catId int) (bool, error)
	CreateCategory(cat models.CategoryRequest) (newCatId int, err error)
	UpdateCategory(cat models.CategoryRequest) (err error)
}

type CategoryRepo struct {
	db *sql.DB
}

func NewCategoryRepository(conn *sql.DB) (CategoryRepository, error) {
	if conn == nil {
		return nil, errors.New("conn must be non-nil")
	}
	err := conn.Ping()
	if err != nil {
		return nil, err
	}
	return &CategoryRepo{
		db: conn,
	}, nil
}

func (c *CategoryRepo) CaregoryExist(catId int) (bool, error) {
	row := c.db.QueryRow("SELECT Id FROM Categories WHERE Id=$1", catId)
	var ex int
	err := row.Scan(&ex)
	if err == nil {
		return true, nil
	}
	if err == sql.ErrNoRows {
		return false, nil
	}
	log.Printf("CaregoryExist: %v", err)
	err = models.ErrServerError
	return false, err
}

func (c *CategoryRepo) CreateCategory(cat models.CategoryRequest) (newCatId int, err error) {
	if cat.ParentId.Valid {
		if cat.ParentId.Value != 0 {
			ex, e := c.CaregoryExist(cat.ParentId.Value)
			if e != nil {
				err = e
				return
			}
			if !ex {
				log.Printf("CreateCategory: Parent category does not exist")
				err = models.ErrNotAllowed
				return
			}
		}
		err = c.db.QueryRow("INSERT INTO Categories (Name, ParentId) VALUES ($1, $2) RETURNING Id", cat.Name, cat.ParentId.Value).Scan(&newCatId)
		if err != nil {
			log.Printf("CreateCategory[1]: %v", err)
			err = models.ErrServerError
		}
		return
	}
	err = c.db.QueryRow("INSERT INTO Categories (Name, ParentId) VALUES ($1, 0) RETURNING Id", cat.Name).Scan(&newCatId)
	if err != nil {
		log.Printf("CreateCategory[2]: %v", err)
		err = models.ErrServerError
	}
	return
}

func (c *CategoryRepo) UpdateCategory(cat models.CategoryRequest) (err error) {
	// ParentId не передан
	if !cat.ParentId.Valid {
		if cat.Name == "" {
			log.Printf("invalid category data")
			err = models.ErrNotAllowed
		} else {
			_, err = c.db.Exec("UPDATE Categories SET Name =$1 WHERE Id=$2", cat.Name, cat.Id)
			if err != nil {
				log.Printf("UpdateCategory: %v", err)
				err = models.ErrServerError
			}
		}
		return
	}
	// ParentId передан
	if cat.ParentId.Value != 0 {
		ex, e := c.CaregoryExist(cat.ParentId.Value)
		if e != nil {
			err = e
			return
		}
		if !ex {
			log.Printf("UpdateCategory: Parent category does not exist")
			err = models.ErrNotAllowed
			return
		}
	}
	if cat.Name == "" {
		_, err = c.db.Exec("UPDATE Categories SET ParentId=$1 WHERE Id=$2", cat.ParentId.Value, cat.Id)
	} else {
		_, err = c.db.Exec("UPDATE Categories SET Name =$1, ParentId=$2 WHERE Id=$3", cat.Name, cat.ParentId.Value, cat.Id)
	}
	if err != nil {
		log.Printf("CreateCategory[2]: %v", err)
		err = models.ErrServerError
	}
	return
}

func (c *CategoryRepo) GetSubCategories(catId int) (cats []entities.Category, err error) {
	rows, e := c.db.Query("SELECT Categories.Id, Categories.Name FROM Categories WHERE ParentId=$1", catId)
	if e != nil {
		log.Printf("GetSubCategories[1]: %v", e)
		err = models.ErrServerError
		return
	}
	for rows.Next() {
		cat := entities.Category{}
		err = rows.Scan(&cat.Id, &cat.Name)
		if err != nil {
			log.Printf("GetProductCategories[2]: %v", err)
			err = models.ErrServerError
			return
		}
		cats = append(cats, cat)
	}
	return
}

func (c *CategoryRepo) GetAllCategories() ([]entities.CategoryTree, error) {
	rows, err := c.db.Query("SELECT Id, ParentId, Name FROM Categories")
	if err != nil {
		log.Printf("GetAllCategories: %v", err)
		err = models.ErrServerError
		return nil, err
	}
	var categories []entities.CategoryTree
	for rows.Next() {
		var cat entities.CategoryTree
		if err := rows.Scan(&cat.ID, &cat.ParentID, &cat.Name); err != nil {
			log.Printf("GetAllCategories: %v", err)
			err = models.ErrServerError
			return nil, err
		}
		categories = append(categories, cat)
	}

	tree := c.buildTree(categories, 1)
	return tree, nil
}

func (c *CategoryRepo) buildTree(categories []entities.CategoryTree, parentID int) []entities.CategoryTree {
	var tree []entities.CategoryTree
	for _, cat := range categories {
		if cat.ParentID == parentID {
			children := c.buildTree(categories, cat.ID)
			cat.Children = children
			tree = append(tree, cat)
		}
	}
	return tree
}
