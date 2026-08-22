package repository

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"ticket-system/internal/model"
)

func TestConcurrentTicketCreationKeepsBusinessNumberUnique(t *testing.T) {
	db := newTicketTestDB(t)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			ticket := &model.Ticket{TicketNo: "T202608210001", Title: fmt.Sprintf("parallel-%d", i), Urgency: model.UrgencyNormal, Status: model.StatusPending, SubmitterID: uint(i + 1), Group: model.GroupIT, SubmittedAt: time.Now()}
			if err := db.Create(ticket).Error; err == nil {
				return
			}
		}(i)
	}
	close(start)
	wg.Wait()
	var count int64
	if err := db.Model(&model.Ticket{}).Where("ticket_no = ?", "T202608210001").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("duplicate business number persisted: count=%d", count)
	}
}

func TestDistinctBusinessNumbersRemainIndependent(t *testing.T) {
	db := newTicketTestDB(t)
	items := []model.Ticket{
		{TicketNo: "T202608210101", Title: "first", Urgency: model.UrgencyNormal, Status: model.StatusPending, SubmitterID: 1, Group: model.GroupIT, SubmittedAt: time.Now()},
		{TicketNo: "T202608210102", Title: "second", Urgency: model.UrgencyNormal, Status: model.StatusPending, SubmitterID: 2, Group: model.GroupIT, SubmittedAt: time.Now()},
	}
	if err := db.Create(&items).Error; err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.Model(&model.Ticket{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("distinct business numbers persisted %d records, want 2", count)
	}
}
