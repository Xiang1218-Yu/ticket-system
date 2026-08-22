package repository

import (
	"ticket-system/internal/model"

	"gorm.io/gorm"
)

// ReviewRepository 封装评价表的数据访问。
// 单一职责：仅读写 reviews 表，不含一单一评的业务判定。
type ReviewRepository struct {
	db *gorm.DB
}

func NewReviewRepository(db *gorm.DB) *ReviewRepository {
	return &ReviewRepository{db: db}
}

// Create 写入一条评价。一单一评的唯一约束在 DB 层（uniqueIndex:ticket_id），
// 因此并发提交时第二个写入会以 gorm.ErrDuplicatedKey 失败，不依赖检查的先后顺序。
func (r *ReviewRepository) Create(rv *model.Review) error {
	return r.db.Create(rv).Error
}

// CreateForTicket 写入指定工单的评价。返回的错误若是唯一约束冲突，
// 调用方可用 errors.Is(err, gorm.ErrDuplicatedKey) 识别为“已评价”。
func (r *ReviewRepository) CreateForTicket(ticketID uint, rv *model.Review) error {
	rv.TicketID = ticketID
	return r.db.Session(&gorm.Session{SkipDefaultTransaction: true}).Create(rv).Error
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
// 仅作为快速前置检查以减少写冲突，正确性仍由 DB 唯一约束保证——
// 检查与写入之间的窗口不依赖调用顺序。
func (r *ReviewRepository) ExistsByTicket(ticketID uint) (bool, error) {
	var count int64
	err := r.db.Model(&model.Review{}).Where("ticket_id = ?", ticketID).Count(&count).Error
	return count > 0, err
}
