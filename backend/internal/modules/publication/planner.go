package publication

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

type PreviewInput struct {
	StartAt           time.Time `json:"startAt"`
	Days              int       `json:"days"`
	MaxPerDay         int       `json:"maxPerDay"`
	MinSpacingMinutes int       `json:"minSpacingMinutes"`
}

type Candidate struct {
	ListingDraftID   uuid.UUID
	ContentVersionID uuid.UUID
	Title            string
	Category         string
	CreatedAt        time.Time
}

type PreviewSlot struct {
	ListingDraftID   uuid.UUID `json:"listingDraftId"`
	ContentVersionID uuid.UUID `json:"contentVersionId"`
	Title            string    `json:"title"`
	Category         string    `json:"category"`
	ScheduledFor     time.Time `json:"scheduledFor"`
	Score            int       `json:"score"`
	Reason           string    `json:"reason"`
}

type PreviewResult struct {
	Slots    []PreviewSlot `json:"slots"`
	Unplaced int           `json:"unplaced"`
}

func normalizePreview(in PreviewInput, now time.Time) (PreviewInput, error) {
	if in.StartAt.IsZero() {
		in.StartAt = now.UTC().Add(time.Hour).Truncate(time.Minute)
	} else {
		in.StartAt = in.StartAt.UTC()
	}
	if in.StartAt.Before(now.UTC().Add(-time.Minute)) {
		return in, fmt.Errorf("startAt must not be in the past")
	}
	if in.Days == 0 {
		in.Days = 7
	}
	if in.MaxPerDay == 0 {
		in.MaxPerDay = 4
	}
	if in.MinSpacingMinutes == 0 {
		in.MinSpacingMinutes = 120
	}
	if in.Days < 1 || in.Days > 31 || in.MaxPerDay < 1 || in.MaxPerDay > 24 || in.MinSpacingMinutes < 15 || in.MinSpacingMinutes > 1440 {
		return in, fmt.Errorf("planner limits are outside the supported range")
	}
	return in, nil
}

// BuildPreview is deliberately pure: it ranks immutable candidates and never
// receives a database handle. Older READY drafts win, with a deterministic
// category-diversity bonus so one category cannot consume the first slots.
func BuildPreview(in PreviewInput, candidates []Candidate, now time.Time) (PreviewResult, error) {
	in, err := normalizePreview(in, now)
	if err != nil {
		return PreviewResult{}, err
	}
	rows := append([]Candidate(nil), candidates...)
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].CreatedAt.Equal(rows[j].CreatedAt) {
			return rows[i].ListingDraftID.String() < rows[j].ListingDraftID.String()
		}
		return rows[i].CreatedAt.Before(rows[j].CreatedAt)
	})
	capacity := in.Days * in.MaxPerDay
	placed := min(len(rows), capacity)
	result := PreviewResult{Slots: make([]PreviewSlot, 0, placed), Unplaced: len(candidates) - placed}
	categoryCount := map[string]int{}
	for idx := 0; idx < placed; idx++ {
		bestIndex, bestScore := 0, -1
		for candidateIndex, candidate := range rows {
			category := strings.TrimSpace(candidate.Category)
			diversity := max(0, 15-categoryCount[category]*8)
			ageHours := max(0, int(now.UTC().Sub(candidate.CreatedAt.UTC()).Hours()))
			score := min(100, 60+min(25, ageHours/24)+diversity)
			if score > bestScore {
				bestIndex, bestScore = candidateIndex, score
			}
		}
		row := rows[bestIndex]
		rows = append(rows[:bestIndex], rows[bestIndex+1:]...)
		day := idx / in.MaxPerDay
		withinDay := idx % in.MaxPerDay
		when := in.StartAt.AddDate(0, 0, day).Add(time.Duration(withinDay*in.MinSpacingMinutes) * time.Minute)
		category := strings.TrimSpace(row.Category)
		result.Slots = append(result.Slots, PreviewSlot{ListingDraftID: row.ListingDraftID, ContentVersionID: row.ContentVersionID, Title: row.Title, Category: category, ScheduledFor: when, Score: bestScore, Reason: "READY age plus category diversity"})
		categoryCount[category]++
	}
	return result, nil
}
