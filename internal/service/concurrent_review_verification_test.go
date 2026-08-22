package service

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"ticket-system/internal/model"
	"ticket-system/internal/repository"
)

var reviewTestID uint64

func TestConcurrentReviewsLeaveExactlyOneRecord(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:review_race_%d?mode=memory&cache=shared", atomic.AddUint64(&reviewTestID, 1))), &gorm.Config{}); if err != nil { t.Fatal(err) }
	if err := db.AutoMigrate(&model.User{}, &model.Category{}, &model.Ticket{}, &model.Comment{}, &model.Attachment{}, &model.Review{}); err != nil { t.Fatal(err) }
	ticket := &model.Ticket{TicketNo:"T-review",Title:"review",Urgency:model.UrgencyNormal,Status:model.StatusClosed,SubmitterID:7,Group:model.GroupIT,SubmittedAt:time.Now()}
	if err := db.Create(ticket).Error; err != nil { t.Fatal(err) }
	svc := NewReviewService(db, repository.NewReviewRepository(db), repository.NewTicketRepository(db))
	start := make(chan struct{}); var wg sync.WaitGroup; results := make(chan error, 2)
	for i:=0; i<2; i++ { wg.Add(1); go func(){ defer wg.Done(); <-start; _, err := svc.Submit(ticket.ID, Actor{ID:7,Role:model.RoleEmployee}, ReviewInput{Speed:5,Quality:5,Communicate:5,Comment:"ok"}); results <- err }() }
	close(start); wg.Wait(); close(results)
	success := 0; for err := range results { if err == nil { success++ } }
	var count int64; if err := db.Model(&model.Review{}).Where("ticket_id = ?", ticket.ID).Count(&count).Error; err != nil { t.Fatal(err) }
	if success != 1 || count != 1 { t.Fatalf("parallel reviews created success=%d records=%d, want exactly one", success, count) }
}

func TestSingleValidReviewIsPersisted(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:review_single_%d?mode=memory&cache=shared", atomic.AddUint64(&reviewTestID, 1))), &gorm.Config{}); if err != nil { t.Fatal(err) }
	if err := db.AutoMigrate(&model.User{}, &model.Category{}, &model.Ticket{}, &model.Comment{}, &model.Attachment{}, &model.Review{}); err != nil { t.Fatal(err) }
	ticket := &model.Ticket{TicketNo:"T-single-review", Title:"review", Urgency:model.UrgencyNormal, Status:model.StatusClosed, SubmitterID:7, Group:model.GroupIT, SubmittedAt:time.Now()}
	if err := db.Create(ticket).Error; err != nil { t.Fatal(err) }
	got, err := NewReviewService(db, repository.NewReviewRepository(db), repository.NewTicketRepository(db)).Submit(ticket.ID, Actor{ID:7, Role:model.RoleEmployee}, ReviewInput{Speed:5, Quality:4, Communicate:5, Comment:"ok"})
	if err != nil { t.Fatal(err) }
	if got.TicketID != ticket.ID { t.Fatalf("review ticket id = %d, want %d", got.TicketID, ticket.ID) }
}
