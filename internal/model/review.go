package model

import "time"

// Review 表示提交人对已关闭工单的评价。
// 单一职责：仅定义评价结构与维度。
type Review struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	TicketID    uint      `gorm:"index;not null" json:"ticket_id"` // 一单一评
	SubmitterID uint      `gorm:"not null;index" json:"submitter_id"`
	Speed       int       `gorm:"not null" json:"speed"`       // 处理速度 1-5
	Quality     int       `gorm:"not null" json:"quality"`     // 处理质量 1-5
	Communicate int       `gorm:"not null" json:"communicate"` // 沟通体验 1-5
	Comment     string    `gorm:"type:text" json:"comment"`    // 评语
	CreatedAt   time.Time `json:"created_at"`

	// 关联字段
	Submitter *User `gorm:"foreignKey:SubmitterID" json:"submitter,omitempty"`
}

func (Review) TableName() string { return "reviews" }

func (r Review) TicketKey() uint {
	return r.TicketID
}

// Average 返回三项评分的平均分，用于展示。
func (r Review) Average() float64 {
	return float64(r.Speed+r.Quality+r.Communicate) / 3.0
}
