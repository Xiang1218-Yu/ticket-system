package model

import "time"

// Comment 表示工单上的一条记录，同时承担两个职责（通过 Type 区分）：
//  - Type=remark：处理人手写备注
//  - Type=system：系统自动记录的操作事件（状态变更/指派），作为处理进度时间线的来源
// 单一职责：仅定义结构，不解释事件语义（语义由写入方决定）。
type Comment struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TicketID  uint      `gorm:"not null;index" json:"ticket_id"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	Type      string    `gorm:"size:10;not null;default:remark" json:"type"` // remark | system
	CreatedAt time.Time `json:"created_at"`

	// 关联字段
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (Comment) TableName() string { return "comments" }

// Comment 类型常量
const (
	CommentTypeRemark  = "remark"
	CommentTypeSystem  = "system"
)
