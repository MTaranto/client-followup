package main

import (
	"testing"
	"time"
)

func TestReminderRules(t *testing.T) {
	location := mustLocation(t)
	today := time.Date(2026, time.August, 12, 15, 30, 0, 0, location)

	tests := []struct {
		name     string
		dueDate  string
		status   string
		wantKind string
		wantText string
	}{
		{name: "tomorrow", dueDate: "2026-08-13", status: StatusPending, wantKind: "TOMORROW", wantText: "Vence amanhã"},
		{name: "today", dueDate: "2026-08-12", status: StatusPending, wantKind: "TODAY", wantText: "Vence hoje"},
		{name: "overdue", dueDate: "2026-08-09", status: StatusPending, wantKind: "OVERDUE", wantText: "Atrasada há 3 dias"},
		{name: "future", dueDate: "2026-08-20", status: StatusPending},
		{name: "completed", dueDate: "2026-08-12", status: StatusCompleted},
		{name: "archived", dueDate: "2026-08-09", status: StatusArchived},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := reminderFor(FollowUp{DueDate: test.dueDate, Status: test.status}, today)
			if got.Kind != test.wantKind || got.Label != test.wantText {
				t.Fatalf("reminderFor() = %#v, want kind %q and text %q", got, test.wantKind, test.wantText)
			}
		})
	}
}

func TestReminderOrdering(t *testing.T) {
	today := time.Date(2026, time.August, 12, 0, 0, 0, 0, mustLocation(t))
	items := []FollowUp{
		{ID: 1, DueDate: "2026-08-13", Priority: PriorityHigh, Status: StatusPending},
		{ID: 2, DueDate: "2026-08-12", Priority: PriorityLow, Status: StatusPending},
		{ID: 3, DueDate: "2026-08-10", Priority: PriorityLow, Status: StatusPending},
		{ID: 4, DueDate: "2026-08-10", Priority: PriorityHigh, Status: StatusPending},
	}

	got := addReminders(items, today)
	wantIDs := []int64{4, 3, 2, 1}
	for index, wantID := range wantIDs {
		if got[index].ID != wantID {
			t.Fatalf("posição %d = ID %d, want ID %d", index, got[index].ID, wantID)
		}
	}
}

func mustLocation(t *testing.T) *time.Location {
	t.Helper()
	location, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		t.Fatal(err)
	}
	return location
}
