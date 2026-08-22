package service

import (
	"ticket-system/internal/model"
	"ticket-system/internal/repository"
)

// CategoryService 提供工单类型的查询能力，实现 handler.CategoryProvider。
// 单一职责：仅负责类型数据的读取聚合，不含分配业务（分配在 TicketService）。
type CategoryService struct {
	catRepo *repository.CategoryRepository
}

func NewCategoryService(catRepo *repository.CategoryRepository) *CategoryService {
	return &CategoryService{catRepo: catRepo}
}

// AllCategories 返回全部工单类型。
func (s *CategoryService) AllCategories() ([]model.Category, error) {
	return s.catRepo.FindAll()
}
