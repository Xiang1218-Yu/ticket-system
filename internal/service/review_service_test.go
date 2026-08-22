package service

import (
	"errors"
	"fmt"
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

// newTestDB 构造一个临时 SQLite 文件库并完成迁移，供并发评价测试隔离使用。
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dir := t.TempDir()
	db, err := gorm.Open(sqlite.Open(filepath.Join(dir, "test.db")), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Silent),
		TranslateError: true, // 与生产 database.Connect 一致，使 ErrDuplicatedKey 可用
	})
	if err != nil {
		t.Fatalf("打开测试库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{}, &model.Category{}, &model.Ticket{},
		&model.Comment{}, &model.Attachment{}, &model.Review{},
	); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	return db
}

// seedClosedTicketWithSubmitter 写入一个已关闭工单及其提交人，供评价测试使用。
// suffix 用于在同一库内构造多条互不冲突的记录（username/ticket_no 全局唯一）。
func seedClosedTicketWithSubmitter(t *testing.T, db *gorm.DB, suffix string) (ticketID, submitterID uint) {
	t.Helper()
	submitter := &model.User{Username: "emp_" + suffix, Name: "员工 " + suffix, Role: model.RoleEmployee}
	if err := db.Create(submitter).Error; err != nil {
		t.Fatalf("写入提交人失败: %v", err)
	}
	cat := &model.Category{Name: "IT报修_" + suffix, Group: model.GroupIT}
	if err := db.Create(cat).Error; err != nil {
		t.Fatalf("写入类型失败: %v", err)
	}
	closed := time.Now().Add(-1 * time.Hour)
	tk := &model.Ticket{
		TicketNo: "T" + suffix, Title: "测试工单", CategoryID: cat.ID,
		Urgency: model.UrgencyNormal, Status: model.StatusClosed,
		SubmitterID: submitter.ID, Group: cat.Group, SubmittedAt: closed.Add(-2 * time.Hour),
		ClosedAt: &closed,
	}
	if err := db.Create(tk).Error; err != nil {
		t.Fatalf("写入工单失败: %v", err)
	}
	return tk.ID, submitter.ID
}

// TestReviewSubmitConcurrentOnce 验证并发提交同一评价时“一单一评”：
// 无论两个请求谁先通过检查，最终库中只有一条评价，且竞争落败者拿到“该工单已评价”清晰反馈。
// 这是对 check-then-act 竞争的回归保护：正确性不依赖 ExistsByTicket 与写入的先后顺序，
// 而由 DB 唯一约束兜底。
func TestReviewSubmitConcurrentOnce(t *testing.T) {
	db := newTestDB(t)
	ticketID, submitterID := seedClosedTicketWithSubmitter(t, db, "once")
	reviewRepo := repository.NewReviewRepository(db)
	ticketRepo := repository.NewTicketRepository(db)
	svc := NewReviewService(db, reviewRepo, ticketRepo)
	actor := Actor{ID: submitterID, Role: model.RoleEmployee}

	in := ReviewInput{Speed: 5, Quality: 5, Communicate: 4, Comment: "处理及时"}

	// start 作为栅栏，让两个 goroutine 尽量同时进入 Submit，以最大化触发 DB 唯一约束冲突路径。
	var wg sync.WaitGroup
	const n = 2
	results := make([]error, n)
	ok := make([]bool, n)
	start := make(chan struct{})
	wg.Add(n)
	for i := 0; i < n; i++ {
		i := i
		go func() {
			defer wg.Done()
			<-start
			_, err := svc.Submit(ticketID, actor, in)
			results[i], ok[i] = err, err == nil
		}()
	}
	close(start)
	wg.Wait()

	// 不变量：恰好一个请求成功，另一个落败并得到“该工单已评价”。
	success := 0
	dupFeedback := 0
	for i := 0; i < n; i++ {
		if ok[i] {
			success++
		} else if results[i] != nil && results[i].Error() == "该工单已评价" {
			dupFeedback++
		} else {
			t.Fatalf("落败请求应返回“该工单已评价”，实际: %v", results[i])
		}
	}
	if success != 1 {
		t.Fatalf("期望恰好 1 个成功，实际 %d", success)
	}
	if dupFeedback != 1 {
		t.Fatalf("期望恰好 1 个落败且反馈“该工单已评价”，实际 %d", dupFeedback)
	}

	// 库中只能有一条评价——证明没有产生多个唯一结果。
	var count int64
	if err := db.Model(&model.Review{}).Where("ticket_id = ?", ticketID).Count(&count).Error; err != nil {
		t.Fatalf("查询评价数失败: %v", err)
	}
	if count != 1 {
		var nos string
		var rvs []model.Review
		db.Where("ticket_id = ?", ticketID).Find(&rvs)
		for _, r := range rvs {
			nos += fmt.Sprintf(" %d", r.ID)
		}
		t.Fatalf("期望库中只有 1 条评价，实际 %d 条（IDs:%s）", count, nos)
	}
}

// TestReviewSubmitConcurrentManyRounds 反复并发提交，确保多次竞争下不变量恒成立。
func TestReviewSubmitConcurrentManyRounds(t *testing.T) {
	db := newTestDB(t)
	ticketRepo := repository.NewTicketRepository(db)
	reviewRepo := repository.NewReviewRepository(db)
	svc := NewReviewService(db, reviewRepo, ticketRepo)

	for r := 0; r < 20; r++ {
		ticketID, submitterID := seedClosedTicketWithSubmitter(t, db, fmt.Sprintf("r%d", r))
		actor := Actor{ID: submitterID, Role: model.RoleEmployee}
		in := ReviewInput{Speed: 4, Quality: 4, Communicate: 4, Comment: ""}

		var wg sync.WaitGroup
		results := make([]error, 2)
		start := make(chan struct{})
		wg.Add(2)
		for i := 0; i < 2; i++ {
			i := i
			go func() {
				defer wg.Done()
				<-start
				_, results[i] = svc.Submit(ticketID, actor, in)
			}()
		}
		close(start)
		wg.Wait()

		// 每轮：恰好一个成功，落败者须是“已评价”而非其它错误。
		succ, dup := 0, 0
		for _, e := range results {
			if e == nil {
				succ++
			} else if e.Error() == "该工单已评价" {
				dup++
			} else {
				t.Fatalf("第 %d 轮异常错误: %v", r, e)
			}
		}
		if succ != 1 || dup != 1 {
			t.Fatalf("第 %d 轮期望 1 成功 + 1 已评价，实际 succ=%d dup=%d", r, succ, dup)
		}
		var count int64
		db.Model(&model.Review{}).Where("ticket_id = ?", ticketID).Count(&count)
		if count != 1 {
			t.Fatalf("第 %d 轮库中评价数 %d != 1", r, count)
		}
	}
}

// TestReviewSubmitSequentialDuplicate 验证非并发重复提交也得到清晰反馈。
func TestReviewSubmitSequentialDuplicate(t *testing.T) {
	db := newTestDB(t)
	ticketID, submitterID := seedClosedTicketWithSubmitter(t, db, "seq")
	svc := NewReviewService(db, repository.NewReviewRepository(db), repository.NewTicketRepository(db))
	actor := Actor{ID: submitterID, Role: model.RoleEmployee}
	in := ReviewInput{Speed: 5, Quality: 5, Communicate: 5}

	if _, err := svc.Submit(ticketID, actor, in); err != nil {
		t.Fatalf("首次提交不应失败: %v", err)
	}
	_, err := svc.Submit(ticketID, actor, in)
	if err == nil {
		t.Fatal("重复提交应失败")
	}
	if err.Error() != "该工单已评价" {
		t.Fatalf("重复提交应返回“该工单已评价”，实际: %v", err)
	}
}

// TestIsDuplicatedKey 直接覆盖重复键识别的三种形态：gorm 哨兵、
// 未翻译时的 SQLite 原始约束文本、以及无关错误。确保无论是否开启
// TranslateError，落败请求都能被归一为“已评价”反馈。
func TestIsDuplicatedKey(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"gorm sentinel", gorm.ErrDuplicatedKey, true},
		{"wrapped sentinel", fmt.Errorf("wrap: %w", gorm.ErrDuplicatedKey), true},
		{"sqlite raw unique", errors.New("constraint failed: UNIQUE constraint failed: reviews.ticket_id (2067)"), true},
		{"sqlite raw unique lowercase", errors.New("unique constraint failed: users.username"), true},
		{"mysql duplicate entry", errors.New("Error 1062: Duplicate entry 'x' for key 'ticket_id'"), true},
		{"unrelated error", errors.New("评分需在 1-5 之间"), false},
		{"nil", nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isDuplicatedKey(c.err); got != c.want {
				t.Fatalf("isDuplicatedKey(%v) = %v, want %v", c.err, got, c.want)
			}
		})
	}
}
