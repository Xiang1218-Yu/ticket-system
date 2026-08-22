package service

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"ticket-system/internal/model"
	"ticket-system/internal/repository"
)

// newTestDB 构建一个基于临时文件的 SQLite 数据库并完成迁移。
// 不使用 :memory:：内存库每个连接各自独立，并发 goroutine 会看到空库，
// 无法验证乐观锁。文件库天然支持多连接共享，便于并发推进测试。
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dir := t.TempDir()
	db, err := gorm.Open(sqlite.Open(filepath.Join(dir, "test.db")+"?_pragma=busy_timeout(5000)"), &gorm.Config{
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

// seedTicketFlow 准备一个指派给处理人的"处理中"工单，返回 service、工单 id 及各角色 Actor。
func seedTicketFlow(t *testing.T, db *gorm.DB) (*TicketService, uint, Actor, Actor) {
	t.Helper()
	cat := model.Category{Name: "IT报修", Group: model.GroupIT}
	if err := db.Create(&cat).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}
	staff := model.User{Username: "staff", Name: "员工甲", Role: model.RoleEmployee}
	handler := model.User{Username: "handler", Name: "处理人", Role: model.RoleHandler, Group: model.GroupIT}
	if err := db.Create(&staff).Error; err != nil {
		t.Fatalf("create staff: %v", err)
	}
	if err := db.Create(&handler).Error; err != nil {
		t.Fatalf("create handler: %v", err)
	}

	ticketRepo := repository.NewTicketRepository(db)
	catRepo := repository.NewCategoryRepository(db)
	cmtRepo := repository.NewCommentRepository(db)
	attRepo := repository.NewAttachmentRepository(db)
	svc := NewTicketService(db, ticketRepo, catRepo, cmtRepo, attRepo, "")

	// 提交并指派到处理人，再推进到"处理中"
	tk, err := svc.Create(CreateInput{
		Title: "测试工单", CategoryID: cat.ID, Urgency: model.UrgencyNormal, SubmitterID: staff.ID,
	})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	if err := db.Model(&model.Ticket{}).Where("id = ?", tk.ID).Update("assignee_id", handler.ID).Error; err != nil {
		t.Fatalf("assign: %v", err)
	}
	handlerActor := Actor{ID: handler.ID, Role: model.RoleHandler, Group: model.GroupIT}
	submitterActor := Actor{ID: staff.ID, Role: model.RoleEmployee}
	if _, err := svc.UpdateStatus(tk.ID, handlerActor, model.StatusProcessing); err != nil {
		t.Fatalf("advance to processing: %v", err)
	}
	return svc, tk.ID, handlerActor, submitterActor
}

// 处理中不允许直接关闭：必须先经"已完成"再到"已关闭"。
// 否则 completed_at 永远为空，与已关闭状态矛盾，且被平均处理时长统计遗漏。
func TestUpdateStatus_ProcessingCannotCloseDirectly(t *testing.T) {
	db := newTestDB(t)
	svc, id, handlerActor, submitterActor := seedTicketFlow(t, db)

	// 处理中直接关闭应被拒绝（无论角色，规则在流转约束层）。
	if _, err := svc.UpdateStatus(id, handlerActor, model.StatusClosed); err == nil {
		t.Fatal("处理中直接关闭应被拒绝，实际成功")
	}
	if _, err := svc.UpdateStatus(id, submitterActor, model.StatusClosed); err == nil {
		t.Fatal("提交人从处理中直接关闭同样应被拒绝")
	}

	// 正确路径仍可走通：处理中 → 已完成 → 已关闭。
	if _, err := svc.UpdateStatus(id, handlerActor, model.StatusDone); err != nil {
		t.Fatalf("processing->done 失败: %v", err)
	}
	if _, err := svc.UpdateStatus(id, submitterActor, model.StatusClosed); err != nil {
		t.Fatalf("done->closed 失败: %v", err)
	}

	// 关闭后 completed_at 与 closed_at 都必须存在，且完成不晚于关闭。
	var tk model.Ticket
	db.First(&tk, id)
	if tk.Status != model.StatusClosed {
		t.Fatalf("终态应为 closed，实际 %s", tk.Status)
	}
	if tk.CompletedAt == nil {
		t.Fatal("已关闭工单 completed_at 为空，状态与时间线不一致")
	}
	if tk.ClosedAt == nil {
		t.Fatal("已关闭工单 closed_at 为空")
	}
	if tk.CompletedAt.After(*tk.ClosedAt) {
		t.Fatal("completed_at 晚于 closed_at，时间线矛盾")
	}
}

// 并发推进只有一个成功，其余得到冲突或状态未变更；
// 系统备注（时间线）与成功推进数一致，不会因互相覆盖而残留矛盾记录。
func TestUpdateStatus_ConcurrentAdvanceDoesNotCorrupt(t *testing.T) {
	db := newTestDB(t)
	svc, id, handlerActor, _ := seedTicketFlow(t, db)

	const goroutines = 16
	start := make(chan struct{})
	errs := make([]error, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			<-start
			_, errs[i] = svc.UpdateStatus(id, handlerActor, model.StatusDone)
		}()
	}
	close(start)
	wg.Wait()

	success := 0
	conflict := 0
	other := 0
	for _, e := range errs {
		switch {
		case e == nil:
			success++
		case errors.Is(e, repository.ErrConcurrentTransition):
			conflict++
		default:
			// "状态未变更"：该 goroutine 在唯一成功者之后读到 done，属正常并发结果。
			other++
		}
	}
	if success != 1 {
		t.Fatalf("并发推进应只有 1 个成功，实际 %d", success)
	}
	if conflict == 0 && other == 0 {
		t.Fatal("除成功者外应有冲突或状态未变更，实际无")
	}

	// 终态一致：done，completed_at 唯一且非空，无 closed_at。
	var tk model.Ticket
	db.First(&tk, id)
	if tk.Status != model.StatusDone {
		t.Fatalf("终态应为 done，实际 %s", tk.Status)
	}
	if tk.CompletedAt == nil {
		t.Fatal("已完成工单 completed_at 为空")
	}
	if tk.ClosedAt != nil {
		t.Fatal("已完成工单不应有 closed_at")
	}

	// 时间线一致：仅一次成功的"状态由 processing 变更为 done"系统备注。
	n := countSystemComments(t, db, id)
	// seedTicketFlow 内：提交备注(1) + processing 备注系统(1) + 这里成功的 done(1)。
	if n != 3 {
		t.Fatalf("系统备注数应为 3（提交/接单/完成），实际 %d——冲突未回滚或重复写入", n)
	}
}

// 乐观锁防止过期快照覆盖：用"处理中"的旧快照推进，但实际已变为"已完成"，
// 必须冲突且不得改写 completed_at，避免时间线与完成时间被倒流为新的时刻。
func TestUpdateStatus_StaleSnapshotCannotOverwrite(t *testing.T) {
	db := newTestDB(t)
	svc, id, handlerActor, _ := seedTicketFlow(t, db)

	// 正常推进到 done，记录其 completed_at（真实完成时刻）。
	if _, err := svc.UpdateStatus(id, handlerActor, model.StatusDone); err != nil {
		t.Fatalf("done: %v", err)
	}
	var after model.Ticket
	if err := db.First(&after, id).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if after.CompletedAt == nil {
		t.Fatal("done 后 completed_at 应已设置")
	}
	originalCompletedAt := *after.CompletedAt

	// 模拟一个基于"处理中"旧快照的过期请求：期望旧状态=processing，
	// 但数据库已是 done → WHERE 不命中 → 冲突，且不写入任何字段。
	fakeNow := originalCompletedAt.Add(99 * time.Hour)
	err := svc.ticketRepo.TransitionStatus(db, id, model.StatusProcessing, map[string]interface{}{
		"status":       model.StatusDone,
		"completed_at": fakeNow,
	})
	if !errors.Is(err, repository.ErrConcurrentTransition) {
		t.Fatalf("期望 ErrConcurrentTransition，实际 %v", err)
	}

	// 验证未被覆盖：completed_at 仍是原来的时刻，不是 99 小时之后。
	var cur model.Ticket
	db.First(&cur, id)
	if cur.Status != model.StatusDone {
		t.Fatalf("状态应仍为 done，被旧快照覆盖为 %s", cur.Status)
	}
	if cur.CompletedAt == nil || !cur.CompletedAt.Equal(originalCompletedAt) {
		t.Fatalf("completed_at 被过期快照覆盖：原 %v 现 %v", originalCompletedAt, cur.CompletedAt)
	}
}

func countSystemComments(t *testing.T, db *gorm.DB, ticketID uint) int64 {
	t.Helper()
	var n int64
	if err := db.Model(&model.Comment{}).
		Where("ticket_id = ? AND type = ?", ticketID, model.CommentTypeSystem).
		Count(&n).Error; err != nil {
		t.Fatalf("count comments: %v", err)
	}
	return n
}
