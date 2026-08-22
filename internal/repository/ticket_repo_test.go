package repository

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"ticket-system/internal/model"
)

var testDBID uint64

func newTicketTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:ticket_repo_test_%d?mode=memory&cache=shared", atomic.AddUint64(&testDBID, 1))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Category{}, &model.Ticket{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestListOverdueFiltersInDatabaseAndPaginates(t *testing.T) {
	db := newTicketTestDB(t)
	repo := NewTicketRepository(db)
	now := time.Date(2026, time.August, 21, 12, 0, 0, 0, time.UTC)
	category := model.Category{Name: "IT报修", Group: model.GroupIT}
	if err := db.Create(&category).Error; err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 3; i++ {
		ticket := model.Ticket{
			TicketNo:    "T20260821" + string(rune('0'+i)),
			Title:       "超时工单",
			CategoryID:  category.ID,
			Urgency:     model.UrgencyUrgent,
			Status:      model.StatusPending,
			SubmitterID: 1,
			Group:       model.GroupIT,
			SubmittedAt: now.Add(-13 * time.Hour),
		}
		if err := db.Create(&ticket).Error; err != nil {
			t.Fatal(err)
		}
	}
	fresh := model.Ticket{
		TicketNo: "T202608219", Title: "未超时", CategoryID: category.ID,
		Urgency: model.UrgencyUrgent, Status: model.StatusPending,
		SubmitterID: 1, Group: model.GroupIT, SubmittedAt: now.Add(-time.Hour),
	}
	if err := db.Create(&fresh).Error; err != nil {
		t.Fatal(err)
	}

	items, total, err := repo.ListOverdue(TicketFilter{Page: 2, PageSize: 2}, now)
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 {
		t.Fatalf("total = %d, want 3", total)
	}
	if len(items) != 1 {
		t.Fatalf("page 2 item count = %d, want 1", len(items))
	}
	if count, err := repo.CountOverdue(now); err != nil || count != 3 {
		t.Fatalf("CountOverdue() = %d, err = %v, want 3", count, err)
	}
}

func TestListAppliesGroupScope(t *testing.T) {
	db := newTicketTestDB(t)
	repo := NewTicketRepository(db)
	categories := []model.Category{{Name: "IT报修", Group: model.GroupIT}, {Name: "行政采购", Group: model.GroupAdmin}}
	if err := db.Create(&categories).Error; err != nil {
		t.Fatal(err)
	}
	for i, group := range []string{model.GroupIT, model.GroupAdmin} {
		ticket := model.Ticket{TicketNo: "T2026082" + string(rune('1'+i)), Title: "工单", CategoryID: categories[i].ID,
			Urgency: model.UrgencyNormal, Status: model.StatusPending, SubmitterID: 1, Group: group, SubmittedAt: time.Now()}
		if err := db.Create(&ticket).Error; err != nil {
			t.Fatal(err)
		}
	}
	items, total, err := repo.List(TicketFilter{Group: model.GroupIT, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(items) != 1 || items[0].Group != model.GroupIT {
		t.Fatalf("group filter returned total=%d items=%d", total, len(items))
	}
}

func TestAvgProcessingMinutesReturnsZeroWhenNoCompletedTickets(t *testing.T) {
	db := newTicketTestDB(t)
	repo := NewTicketRepository(db)

	avg, err := repo.AvgProcessingMinutes()
	if err != nil {
		t.Fatal(err)
	}
	if avg != 0 {
		t.Fatalf("AvgProcessingMinutes() = %v, want 0", avg)
	}
}

func TestAvgProcessingMinutesCalculatesCompletedTickets(t *testing.T) {
	db := newTicketTestDB(t)
	repo := NewTicketRepository(db)
	submittedAt := time.Date(2026, time.August, 21, 10, 0, 0, 0, time.UTC)
	completedAt := submittedAt.Add(90 * time.Minute)

	for i, status := range []string{model.StatusDone, model.StatusClosed} {
		ticket := model.Ticket{
			TicketNo:    fmt.Sprintf("T20260821AVG%d", i),
			Title:       "已完成工单",
			Urgency:     model.UrgencyNormal,
			Status:      status,
			SubmitterID: 1,
			Group:       model.GroupIT,
			SubmittedAt: submittedAt,
			CompletedAt: &completedAt,
		}
		if err := db.Create(&ticket).Error; err != nil {
			t.Fatal(err)
		}
	}

	avg, err := repo.AvgProcessingMinutes()
	if err != nil {
		t.Fatal(err)
	}
	if avg != 90 {
		t.Fatalf("AvgProcessingMinutes() = %v, want 90", avg)
	}
}
