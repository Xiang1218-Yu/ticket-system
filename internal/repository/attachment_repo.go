package repository

import (
	"ticket-system/internal/model"

	"gorm.io/gorm"
)

// AttachmentRepository 封装附件元数据的数据访问。
// 单一职责：仅读写 attachments 表；文件存储由 service 负责。
type AttachmentRepository struct {
	db *gorm.DB
}

func NewAttachmentRepository(db *gorm.DB) *AttachmentRepository {
	return &AttachmentRepository{db: db}
}

func (r *AttachmentRepository) Create(a *model.Attachment) error {
	return r.db.Create(a).Error
}

func (r *AttachmentRepository) FindByID(id uint) (*model.Attachment, error) {
	var a model.Attachment
	if err := r.db.Unscoped().First(&a, id).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *AttachmentRepository) FindByTicket(ticketID uint) ([]model.Attachment, error) {
	var items []model.Attachment
	err := r.db.Where("ticket_id = ?", ticketID).Order("created_at ASC").Find(&items).Error
	return items, err
}
