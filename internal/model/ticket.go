package model

import (
	"time"

	"gorm.io/gorm"
)

// Ticket 表示一张工单。
// 单一职责：仅定义工单持久化结构与状态/紧急程度常量。
type Ticket struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	TicketNo    string         `gorm:"uniqueIndex;size:20;not null" json:"ticket_no"`
	Title       string         `gorm:"size:200;not null" json:"title"`
	Description string         `gorm:"type:text" json:"description"`
	CategoryID  uint           `gorm:"not null;index" json:"category_id"`
	Urgency     string         `gorm:"size:10;not null;index" json:"urgency"` // normal | urgent
	Status      string         `gorm:"size:20;not null;index" json:"status"`  // pending | processing | done | closed
	SubmitterID uint           `gorm:"not null;index" json:"submitter_id"`
	AssigneeID  *uint          `gorm:"index" json:"assignee_id,omitempty"`
	Group       string         `gorm:"size:30;index" json:"group"` // 自动分配的处理组
	SubmittedAt time.Time      `gorm:"not null;index" json:"submitted_at"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
	ClosedAt    *time.Time     `json:"closed_at,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// 以下为关联字段，仅在 Preload 时填充。
	Submitter   *User        `gorm:"foreignKey:SubmitterID" json:"submitter,omitempty"`
	Assignee    *User        `gorm:"foreignKey:AssigneeID" json:"assignee,omitempty"`
	Category    *Category    `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Comments    []Comment    `gorm:"foreignKey:TicketID" json:"comments,omitempty"`
	Attachments []Attachment `gorm:"foreignKey:TicketID" json:"attachments,omitempty"`
	Review      *Review      `gorm:"foreignKey:TicketID" json:"review,omitempty"`
}

func (Ticket) TableName() string { return "tickets" }

// 状态常量
const (
	StatusPending    = "pending"    // 待处理
	StatusProcessing = "processing" // 处理中
	StatusDone       = "done"       // 已完成
	StatusClosed     = "closed"     // 已关闭
)

// 紧急程度常量
const (
	UrgencyNormal = "normal"
	UrgencyUrgent = "urgent"
)

// IsOverdue 判断工单是否超时（未完成且超过阈值）。
// 普通工单 48 小时，紧急工单 12 小时。此函数仅含判定逻辑，无副作用。
func (t Ticket) IsOverdue(now time.Time) bool {
	if t.Status == StatusDone || t.Status == StatusClosed {
		return false
	}
	threshold := 48 * time.Hour
	if t.Urgency == UrgencyUrgent {
		threshold = 12 * time.Hour
	}
	return now.Sub(t.SubmittedAt) > threshold
}

// AllStatuses 返回所有状态，用于筛选下拉与校验。
func AllStatuses() []string {
	return []string{StatusPending, StatusProcessing, StatusDone, StatusClosed}
}

// LegalTransition 判断状态迁移是否合法。
// 严格单向流转：待处理 → 处理中 → 已完成 → 已关闭。
// 处理中不允许直接关闭，必须先经"已完成"再"已关闭"，
// 否则会出现 completed_at 为空却已关闭的矛盾记录，
// 也会让按 completed_at 计算的平均处理时长统计遗漏这些工单。
func LegalTransition(from, to string) bool {
	allowed := map[string][]string{
		StatusPending:    {StatusProcessing},
		StatusProcessing: {StatusDone},
		StatusDone:       {StatusClosed},
		StatusClosed:     {},
	}
	for _, next := range allowed[from] {
		if next == to {
			return true
		}
	}
	return false
}
