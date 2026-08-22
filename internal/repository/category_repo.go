package repository

import (
	"ticket-system/internal/model"

	"gorm.io/gorm"
)

// CategoryRepository 封装工单类型表的数据访问。
// 单一职责：仅读写 categories 表。
type CategoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) FindAll() ([]model.Category, error) {
	var cats []model.Category
	err := r.db.Order("sort ASC").Find(&cats).Error
	return cats, err
}

func (r *CategoryRepository) FindByID(id uint) (*model.Category, error) {
	var c model.Category
	if err := r.db.First(&c, id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}
