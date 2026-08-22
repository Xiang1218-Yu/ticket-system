package repository

import (
	"testing"
	"time"

	"ticket-system/internal/model"
)

func TestAverageDurationExcludesOpenTicketsWithCompletedTimestamp(t *testing.T) {
	db := newTicketTestDB(t)
	start := time.Date(2026, 8, 21, 9, 0, 0, 0, time.UTC)
	doneAt := start.Add(60*time.Minute); staleAt := start.Add(9*time.Hour)
	tickets := []model.Ticket{{TicketNo:"T-done",Title:"done",Urgency:model.UrgencyNormal,Status:model.StatusDone,SubmitterID:1,Group:model.GroupIT,SubmittedAt:start,CompletedAt:&doneAt},{TicketNo:"T-open",Title:"open",Urgency:model.UrgencyNormal,Status:model.StatusProcessing,SubmitterID:1,Group:model.GroupIT,SubmittedAt:start,CompletedAt:&staleAt}}
	if err := db.Create(&tickets).Error; err != nil { t.Fatal(err) }
	avg, err := NewTicketRepository(db).AvgProcessingMinutes(); if err != nil { t.Fatal(err) }
	if avg != 60 { t.Fatalf("average included an open ticket: got %v want 60", avg) }
}

func TestAverageDurationUsesCompletedRecords(t *testing.T) {
	db := newTicketTestDB(t)
	start := time.Date(2026, 8, 21, 9, 0, 0, 0, time.UTC)
	completed := start.Add(30 * time.Minute)
	ticket := &model.Ticket{TicketNo:"T-average-completed", Title:"done", Urgency:model.UrgencyNormal, Status:model.StatusDone, SubmitterID:1, Group:model.GroupIT, SubmittedAt:start, CompletedAt:&completed}
	if err := db.Create(ticket).Error; err != nil { t.Fatal(err) }
	avg, err := NewTicketRepository(db).AvgProcessingMinutes(); if err != nil { t.Fatal(err) }
	if avg != 30 { t.Fatalf("average = %v, want 30", avg) }
}
