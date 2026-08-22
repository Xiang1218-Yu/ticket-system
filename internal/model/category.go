package model

// Category 表示工单类型，并定义“类型→处理组”的映射。
// 单一职责：仅承载类型元数据与映射关系，分配业务在 service 层。
type Category struct {
	ID    uint   `gorm:"primaryKey" json:"id"`
	Name  string `gorm:"uniqueIndex;size:30;not null" json:"name"` // IT报修/行政采购/人事申请/后勤维修/其他
	Group string `gorm:"size:30;not null;index" json:"group"`     // 该类型工单分配到的处理组
	Sort  int    `gorm:"not null;default:0" json:"sort"`          // 展示排序
}

func (Category) TableName() string { return "categories" }

// 默认工单类型（与 system.md 一致）。
var DefaultCategories = []Category{
	{Name: "IT报修", Group: GroupIT, Sort: 1},
	{Name: "行政采购", Group: GroupAdmin, Sort: 2},
	{Name: "人事申请", Group: GroupHR, Sort: 3},
	{Name: "后勤维修", Group: GroupLogist, Sort: 4},
	{Name: "其他", Group: GroupIT, Sort: 5},
}
