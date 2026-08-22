package repository

import (
	"ticket-system/internal/model"

	"gorm.io/gorm"
)

// UserRepository 封装用户表的数据访问。
// 单一职责：仅读写 users 表，不含业务规则。
type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(u *model.User) error {
	return r.db.Create(u).Error
}

func (r *UserRepository) FindByUsername(username string) (*model.User, error) {
	var u model.User
	if err := r.db.Where("username = ?", username).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) FindByID(id uint) (*model.User, error) {
	var u model.User
	if err := r.db.First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) ExistsByUsername(username string) (bool, error) {
	var count int64
	err := r.db.Model(&model.User{}).Where("username = ?", username).Count(&count).Error
	return count > 0, err
}

// FindByGroup 返回指定组的处理人（含组长），用于指派下拉。
func (r *UserRepository) FindByGroup(group string) ([]model.User, error) {
	var users []model.User
	err := r.db.Where("role = ? AND `group` = ?", model.RoleHandler, group).
		Order("is_leader DESC, id ASC").Find(&users).Error
	return users, err
}

// CountHandlers 返回处理人总数。
func (r *UserRepository) CountHandlers() (int64, error) {
	var count int64
	err := r.db.Model(&model.User{}).Where("role = ?", model.RoleHandler).Count(&count).Error
	return count, err
}
