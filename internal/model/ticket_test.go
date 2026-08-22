package model

import (
	"testing"
	"time"
)

func TestTicketIsOverdueByUrgencyAndStatus(t *testing.T) {
	now := time.Date(2026, time.August, 21, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		ticket  Ticket
		overdue bool
	}{
		{name: "normal ticket over 48 hours", ticket: Ticket{Urgency: UrgencyNormal, Status: StatusPending, SubmittedAt: now.Add(-48*time.Hour - time.Minute)}, overdue: true},
		{name: "normal ticket at threshold", ticket: Ticket{Urgency: UrgencyNormal, Status: StatusProcessing, SubmittedAt: now.Add(-48 * time.Hour)}, overdue: false},
		{name: "urgent ticket over 12 hours", ticket: Ticket{Urgency: UrgencyUrgent, Status: StatusPending, SubmittedAt: now.Add(-12*time.Hour - time.Minute)}, overdue: true},
		{name: "completed ticket is not overdue", ticket: Ticket{Urgency: UrgencyUrgent, Status: StatusDone, SubmittedAt: now.Add(-24 * time.Hour)}, overdue: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ticket.IsOverdue(now); got != tt.overdue {
				t.Fatalf("IsOverdue() = %v, want %v", got, tt.overdue)
			}
		})
	}
}

func TestLegalTransition(t *testing.T) {
	if !LegalTransition(StatusPending, StatusProcessing) {
		t.Fatal("pending should transition to processing")
	}
	if LegalTransition(StatusPending, StatusClosed) {
		t.Fatal("pending should not transition directly to closed")
	}
	if LegalTransition(StatusClosed, StatusPending) {
		t.Fatal("closed should not transition back")
	}
}
