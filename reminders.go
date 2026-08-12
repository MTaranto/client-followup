package main

import (
	"sort"
	"strconv"
	"time"
)

func reminderFor(followUp FollowUp, today time.Time) Reminder {
	if followUp.Status != StatusPending {
		return Reminder{}
	}

	dueDate, err := time.ParseInLocation("2006-01-02", followUp.DueDate, today.Location())
	if err != nil {
		return Reminder{}
	}

	today = dateOnly(today)
	days := int(dueDate.Sub(today).Hours() / 24)
	switch {
	case days < 0:
		late := -days
		return Reminder{Kind: "OVERDUE", Label: overdueLabel(late), DaysLate: late}
	case days == 0:
		return Reminder{Kind: "TODAY", Label: "Vence hoje"}
	case days == 1:
		return Reminder{Kind: "TOMORROW", Label: "Vence amanhã"}
	default:
		return Reminder{}
	}
}

func overdueLabel(days int) string {
	if days == 1 {
		return "Atrasada há 1 dia"
	}
	return "Atrasada há " + strconv.Itoa(days) + " dias"
}

func dateOnly(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, value.Location())
}

func addReminders(followUps []FollowUp, today time.Time) []FollowUp {
	items := make([]FollowUp, 0, len(followUps))
	for _, followUp := range followUps {
		followUp.Alert = reminderFor(followUp, today)
		if followUp.Alert.Kind != "" {
			items = append(items, followUp)
		}
	}

	sort.SliceStable(items, func(i, j int) bool {
		leftRank := reminderRank(items[i].Alert.Kind)
		rightRank := reminderRank(items[j].Alert.Kind)
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		if items[i].DueDate != items[j].DueDate {
			return items[i].DueDate < items[j].DueDate
		}
		return priorityRank(items[i].Priority) < priorityRank(items[j].Priority)
	})
	return items
}

func reminderRank(kind string) int {
	switch kind {
	case "OVERDUE":
		return 0
	case "TODAY":
		return 1
	case "TOMORROW":
		return 2
	default:
		return 3
	}
}

func priorityRank(priority string) int {
	switch priority {
	case PriorityHigh:
		return 0
	case PriorityMedium:
		return 1
	case PriorityLow:
		return 2
	default:
		return 3
	}
}
