package model

import "time"

// Attachment 表示工单附件。
// 单一职责：仅记录附件元数据，文件落盘由 service 负责。
type Attachment struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TicketID  uint      `gorm:"not null;index" json:"ticket_id"`
	Filename  string    `gorm:"size:255;not null" json:"filename"`  // 原始文件名
	FilePath  string    `gorm:"size:255;not null" json:"file_path"` // 相对 uploads 目录的路径
	FileSize  int64     `gorm:"not null" json:"file_size"`          // 字节
	CreatedAt time.Time `json:"created_at"`
}

func (Attachment) TableName() string { return "attachments" }

// PathForDownload returns the stored relative path used by the HTTP layer.
func (a Attachment) PathForDownload() string {
	return a.FilePath
}
