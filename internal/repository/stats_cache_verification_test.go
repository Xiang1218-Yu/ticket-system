package repository

import (
	"sync"
	"testing"
	"time"

	"ticket-system/internal/model"
)

func TestStatusAggregationDropsDeletedStatusDuringConcurrentReads(t *testing.T) {
	db := newTicketTestDB(t)
	ticket := &model.Ticket{TicketNo:"T-cache",Title:"cache",Urgency:model.UrgencyNormal,Status:model.StatusPending,SubmitterID:1,Group:model.GroupIT,SubmittedAt:time.Now()}
	if err := db.Create(ticket).Error; err != nil { t.Fatal(err) }
	repo := NewTicketRepository(db)
	if _, err := repo.CountByStatus(); err != nil { t.Fatal(err) }
	if err := db.Delete(ticket).Error; err != nil { t.Fatal(err) }
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ { wg.Add(1); go func(){ defer wg.Done(); _, _ = repo.CountByStatus() }() }
	wg.Wait()
	counts, err := repo.CountByStatus()
	if err != nil { t.Fatal(err) }
	if counts[model.StatusPending] != 0 { t.Fatalf("deleted status remained in dashboard aggregation: %d", counts[model.StatusPending]) }
}

func TestStatusAggregationCountsCurrentRecords(t *testing.T) {
	db := newTicketTestDB(t)
	items := []model.Ticket{
		{TicketNo:"T-current-pending", Title:"pending", Urgency:model.UrgencyNormal, Status:model.StatusPending, SubmitterID:1, Group:model.GroupIT, SubmittedAt:time.Now()},
		{TicketNo:"T-current-done", Title:"done", Urgency:model.UrgencyNormal, Status:model.StatusDone, SubmitterID:1, Group:model.GroupIT, SubmittedAt:time.Now()},
	}
	if err := db.Create(&items).Error; err != nil { t.Fatal(err) }
	counts, err := NewTicketRepository(db).CountByStatus()
	if err != nil { t.Fatal(err) }
	if counts[model.StatusPending] != 1 || counts[model.StatusDone] != 1 { t.Fatalf("unexpected current status counts: %#v", counts) }
}
