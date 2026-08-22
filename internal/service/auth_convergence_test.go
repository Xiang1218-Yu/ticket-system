package service

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"ticket-system/internal/model"
	"ticket-system/internal/repository"
)

// newTestDB 建立一个全新、与其他用例隔离的内存 SQLite 并迁移表结构。
// 注意：不能使用 cache=shared，否则多用例共享同一内存库导致用户名唯一约束冲突。
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{}, &model.Category{}, &model.Ticket{},
		&model.Comment{}, &model.Attachment{}, &model.Review{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// setupAuthTestService 构造一个 TicketService 并写入两个组各一张工单。
func setupAuthTestService(t *testing.T) (*TicketService, model.User, model.User, model.Ticket, model.Ticket) {
	t.Helper()
	db := newTestDB(t)
	ticketRepo := repository.NewTicketRepository(db)
	catRepo := repository.NewCategoryRepository(db)
	cmtRepo := repository.NewCommentRepository(db)
	attRepo := repository.NewAttachmentRepository(db)
	svc := NewTicketService(db, ticketRepo, catRepo, cmtRepo, attRepo, "")

	// IT 组组长 与 人事组组长
	itLeader := model.User{Username: "it_leader", Name: "IT组长", Role: model.RoleHandler, Group: model.GroupIT, IsLeader: true}
	hrLeader := model.User{Username: "hr_leader", Name: "人事组长", Role: model.RoleHandler, Group: model.GroupHR, IsLeader: true}
	if err := db.Create(&itLeader).Error; err != nil {
		t.Fatalf("create it_leader: %v", err)
	}
	if err := db.Create(&hrLeader).Error; err != nil {
		t.Fatalf("create hr_leader: %v", err)
	}

	catIT := model.Category{Name: "IT报修", Group: model.GroupIT}
	catHR := model.Category{Name: "人事申请", Group: model.GroupHR}
	db.Create(&catIT)
	db.Create(&catHR)

	now := time.Now()
	itTicket := model.Ticket{
		TicketNo: "T0001", Title: "IT工单", CategoryID: catIT.ID,
		Urgency: model.UrgencyNormal, Status: model.StatusPending,
		SubmitterID: itLeader.ID, Group: model.GroupIT, SubmittedAt: now,
	}
	hrTicket := model.Ticket{
		TicketNo: "T0002", Title: "人事工单", CategoryID: catHR.ID,
		Urgency: model.UrgencyNormal, Status: model.StatusPending,
		SubmitterID: hrLeader.ID, Group: model.GroupHR, SubmittedAt: now,
	}
	db.Create(&itTicket)
	db.Create(&hrTicket)
	return svc, itLeader, hrLeader, itTicket, hrTicket
}

func itLeaderActor(u model.User) Actor {
	return Actor{ID: u.ID, Role: u.Role, Name: u.Name, Group: u.Group, IsLeader: u.IsLeader}
}

func adminActor() Actor { return Actor{Role: model.RoleAdmin, Name: "管理员"} }

// TestListFor_LeaderCannotReadOtherGroup 验证组长不能借 group 筛选越权读取其他组工单。
// 修复前：ListFor 仅在 f.Group=="" 时收敛到 actor.Group，传入其他组名即可绕过。
func TestListFor_LeaderCannotReadOtherGroup(t *testing.T) {
	svc, itLeader, _, itTicket, hrTicket := setupAuthTestService(t)
	actor := itLeaderActor(itLeader)

	// IT 组组长尝试筛选人事组——必须被收敛回 IT 组，只能看到本组工单。
	res, err := svc.ListFor(actor, ListFilter{Group: model.GroupHR, Page: 1, PageSize: 100})
	if err != nil {
		t.Fatalf("ListFor: %v", err)
	}
	if len(res.Items) != 1 {
		t.Fatalf("IT 组长只能看到本组 1 张工单，实际看到 %d 张（可能越权读取了人事组）", len(res.Items))
	}
	if res.Items[0].ID != itTicket.ID {
		t.Fatalf("应仅返回 IT 组工单，实际返回 %s", res.Items[0].TicketNo)
	}
	if hasTicket(res.Items, hrTicket.ID) {
		t.Fatalf("IT 组长不应看到人事组工单 %s", hrTicket.TicketNo)
	}
}

// TestListFor_LeaderSeesOwnGroupEvenWithForeignFilter 验证即便筛选条件为空，
// 组长仍按身份归属看到本组数据（收敛不改变正常可见范围）。
func TestListFor_LeaderSeesOwnGroupEvenWithForeignFilter(t *testing.T) {
	svc, itLeader, _, itTicket, _ := setupAuthTestService(t)
	actor := itLeaderActor(itLeader)

	res, err := svc.ListFor(actor, ListFilter{Page: 1, PageSize: 100})
	if err != nil {
		t.Fatalf("ListFor: %v", err)
	}
	if len(res.Items) != 1 || res.Items[0].ID != itTicket.ID {
		t.Fatalf("无筛选时组长应看到本组工单，实际 %+v", res.Items)
	}
}

// TestListFor_AdminRetainsFiltering 验证管理员仍可自由筛选任意组（保留合理筛选）。
func TestListFor_AdminRetainsFiltering(t *testing.T) {
	svc, _, _, _, hrTicket := setupAuthTestService(t)

	// 管理员筛选人事组——应能看到。
	res, err := svc.ListFor(adminActor(), ListFilter{Group: model.GroupHR, Page: 1, PageSize: 100})
	if err != nil {
		t.Fatalf("ListFor admin: %v", err)
	}
	if len(res.Items) != 1 || res.Items[0].ID != hrTicket.ID {
		t.Fatalf("管理员筛选人事组应看到该组工单，实际 %+v", res.Items)
	}

	// 管理员无筛选——应看到全部 2 张。
	resAll, err := svc.ListFor(adminActor(), ListFilter{Page: 1, PageSize: 100})
	if err != nil {
		t.Fatalf("ListFor admin all: %v", err)
	}
	if len(resAll.Items) != 2 {
		t.Fatalf("管理员无筛选应看到全部 2 张，实际 %d", len(resAll.Items))
	}
}

// TestOverdueFor_LeaderCannotReadOtherGroup 验证超时列表同样收敛到本组，
// 组长不能借 group 筛选扩大超时可见范围。
func TestOverdueFor_LeaderCannotReadOtherGroup(t *testing.T) {
	db := newTestDB(t)
	ticketRepo := repository.NewTicketRepository(db)
	catRepo := repository.NewCategoryRepository(db)
	cmtRepo := repository.NewCommentRepository(db)
	attRepo := repository.NewAttachmentRepository(db)
	svc := NewTicketService(db, ticketRepo, catRepo, cmtRepo, attRepo, "")

	itLeader := model.User{Username: "it_leader", Name: "IT组长", Role: model.RoleHandler, Group: model.GroupIT, IsLeader: true}
	hrLeader := model.User{Username: "hr_leader", Name: "人事组长", Role: model.RoleHandler, Group: model.GroupHR, IsLeader: true}
	db.Create(&itLeader)
	db.Create(&hrLeader)
	catIT := model.Category{Name: "IT报修", Group: model.GroupIT}
	catHR := model.Category{Name: "人事申请", Group: model.GroupHR}
	db.Create(&catIT)
	db.Create(&catHR)

	// 两张均超时（normal 超 48 小时）。
	longAgo := time.Now().Add(-72 * time.Hour)
	itTicket := model.Ticket{
		TicketNo: "T0001", Title: "IT工单", CategoryID: catIT.ID, Urgency: model.UrgencyNormal,
		Status: model.StatusPending, SubmitterID: itLeader.ID, Group: model.GroupIT, SubmittedAt: longAgo,
	}
	hrTicket := model.Ticket{
		TicketNo: "T0002", Title: "人事工单", CategoryID: catHR.ID, Urgency: model.UrgencyNormal,
		Status: model.StatusPending, SubmitterID: hrLeader.ID, Group: model.GroupHR, SubmittedAt: longAgo,
	}
	db.Create(&itTicket)
	db.Create(&hrTicket)

	actor := itLeaderActor(itLeader)
	res, err := svc.OverdueFor(actor, ListFilter{Group: model.GroupHR, Page: 1, PageSize: 100})
	if err != nil {
		t.Fatalf("OverdueFor: %v", err)
	}
	if len(res.Items) != 1 || res.Items[0].ID != itTicket.ID {
		t.Fatalf("组长超时列表应仅含本组工单，实际 %+v", res.Items)
	}
	if hasTicket(res.Items, hrTicket.ID) {
		t.Fatalf("组长不应看到人事组超时工单 %s", hrTicket.TicketNo)
	}

	// 管理员看全部超时。
	resAdmin, err := svc.OverdueFor(adminActor(), ListFilter{Page: 1, PageSize: 100})
	if err != nil {
		t.Fatalf("OverdueFor admin: %v", err)
	}
	if len(resAdmin.Items) != 2 {
		t.Fatalf("管理员超时列表应含全部 2 张，实际 %d", len(resAdmin.Items))
	}
}

func hasTicket(items []model.Ticket, id uint) bool {
	for _, t := range items {
		if t.ID == id {
			return true
		}
	}
	return false
}
