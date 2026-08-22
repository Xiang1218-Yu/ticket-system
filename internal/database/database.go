package database

import (
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"ticket-system/internal/model"
)

// Connect 建立 SQLite 连接并完成表迁移。
// 单一职责：仅负责“连接与建表”，不加载业务数据（见 Seed）。
func Connect(dsn string, log *zap.Logger) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger:        logger.Default.LogMode(logger.Warn),
		TranslateError: true, // 让驱动层约束冲突翻译为 gorm.ErrDuplicatedKey 等，供 service 精确识别
	})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.Category{},
		&model.Ticket{},
		&model.Comment{},
		&model.Attachment{},
		&model.Review{},
	); err != nil {
		return nil, err
	}
	log.Info("数据库迁移完成", zap.String("driver", "sqlite"))
	return db, nil
}

// Seed 写入初始数据：工单类型 + 默认账号。
// 单一职责：仅负责“首次启动的初始数据”，幂等执行。
func Seed(db *gorm.DB, log *zap.Logger) error {
	// 工单类型
	var catCount int64
	db.Model(&model.Category{}).Count(&catCount)
	if catCount == 0 {
		if err := db.Create(&model.DefaultCategories).Error; err != nil {
			return err
		}
		log.Info("已写入默认工单类型", zap.Int("count", len(model.DefaultCategories)))
	}

	// 默认账号（admin / 各组组长 + 组员），仅当无任何用户时创建
	var userCount int64
	db.Model(&model.User{}).Count(&userCount)
	if userCount == 0 {
		seedUsers := []model.User{
			{Username: "admin", Name: "系统管理员", Role: model.RoleAdmin},
			{Username: "it_leader", Name: "IT组组长", Role: model.RoleHandler, Group: model.GroupIT, IsLeader: true},
			{Username: "it_staff", Name: "IT组组员", Role: model.RoleHandler, Group: model.GroupIT},
			{Username: "xz_leader", Name: "行政组组长", Role: model.RoleHandler, Group: model.GroupAdmin, IsLeader: true},
			{Username: "hr_leader", Name: "人事组组长", Role: model.RoleHandler, Group: model.GroupHR, IsLeader: true},
			{Username: "hq_leader", Name: "后勤组组长", Role: model.RoleHandler, Group: model.GroupLogist, IsLeader: true},
			{Username: "staff", Name: "普通员工", Role: model.RoleEmployee},
		}
		for i := range seedUsers {
			hash, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
			seedUsers[i].Password = string(hash)
		}
		if err := db.Create(&seedUsers).Error; err != nil {
			return err
		}
		log.Info("已写入默认账号（密码均为 123456）", zap.Int("count", len(seedUsers)))
	}
	return nil
}
