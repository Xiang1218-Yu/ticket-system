package repository

import (
	"ticket-system/internal/model"

	"gorm.io/gorm"
)

// CommentRepository 封装工单备注/系统日志的数据访问。
// 单一职责：仅读写 comments 表。
type CommentRepository struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

func (r *CommentRepository) Create(c *model.Comment) error {
	return r.db.Create(c).Error
}

// FindByTicket 返回工单的全部备注，按时间正序（时间线）。
func (r *CommentRepository) FindByTicket(ticketID uint) ([]model.Comment, error) {
	var comments []model.Comment
	err := r.db.Preload("User").Where("ticket_id = ?", ticketID).
		Order("created_at ASC").Find(&comments).Error
	return comments, err
}
