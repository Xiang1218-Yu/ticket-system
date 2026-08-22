package model

// User 表示系统用户，覆盖三种角色：普通员工、处理人、管理员。
// 单一职责：仅定义用户持久化结构与角色常量，不含业务规则。
type User struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Username string `gorm:"uniqueIndex;size:50;not null" json:"username"`
	Password string `gorm:"size:100;not null" json:"-"` // bcrypt 哈希，不出现在响应中
	Name     string `gorm:"size:50;not null" json:"name"`
	Role     string `gorm:"size:20;not null;index" json:"role"`      // employee | handler | admin
	Group    string `gorm:"size:30;index" json:"group"`              // 处理人所属组：IT组/行政组/人事组/后勤组（员工为空）
	IsLeader bool   `gorm:"not null;default:false" json:"is_leader"` // 是否为组长（仅处理人可能为 true）
}

func (User) TableName() string { return "users" }

// 角色常量
const (
	RoleEmployee = "employee"
	RoleHandler  = "handler"
	RoleAdmin    = "admin"
)

// 处理组常量
const (
	GroupIT     = "IT组"
	GroupAdmin  = "行政组"
	GroupHR     = "人事组"
	GroupLogist = "后勤组"
)

// IsAdmin 是否管理员。
func (u User) IsAdmin() bool { return u.Role == RoleAdmin }

// IsHandler 是否处理人（含组长）。
func (u User) IsHandler() bool { return u.Role == RoleHandler }

// IsLeaderOf 判断用户是否为指定组的组长。
func (u User) IsLeaderOf(group string) bool {
	return u.IsHandler() && u.IsLeader && u.Group == group
}

// InGroup 判断用户是否属于指定处理组。
func (u User) InGroup(group string) bool {
	return u.IsHandler() && u.Group == group
}
