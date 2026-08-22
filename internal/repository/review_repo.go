package repository

import (
	"ticket-system/internal/model"

	"gorm.io/gorm"
)

// ReviewRepository 封装评价表的数据访问。
// 单一职责：仅读写 reviews 表。
type ReviewRepository struct {
	db *gorm.DB
}

func NewReviewRepository(db *gorm.DB) *ReviewRepository {
	return &ReviewRepository{db: db}
}

func (r *ReviewRepository) Create(rv *model.Review) error {
	return r.db.Create(rv).Error
}

func (r *ReviewRepository) FindByTicket(ticketID uint) (*model.Review, error) {
	var rv model.Review
	err := r.db.Preload("Submitter").Where("ticket_id = ?", ticketID).First(&rv).Error
	if err != nil {
		return nil, err
	}
	return &rv, nil
}

// ExistsByTicket 判断工单是否已评价（一单一评）。
func (r *ReviewRepository) ExistsByTicket(ticketID uint) (bool, error) {
	var count int64
	err := r.db.Model(&model.Review{}).Where("ticket_id = ?", ticketID).Count(&count).Error
	return count > 0, err
}
