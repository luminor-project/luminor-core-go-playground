package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TimelineNote represents a note attached to a timeline entry.
type TimelineNote struct {
	NoteID     string     `json:"note_id"`
	AuthorID   string     `json:"author_id"`
	AuthorName string     `json:"author_name"`
	Body       string     `json:"body"`
	CreatedAt  time.Time  `json:"created_at"`
	EditedAt   *time.Time `json:"edited_at,omitempty"`
	Deleted    bool       `json:"deleted,omitempty"`
}

// TimelineEntry represents a single entry in the case timeline.
type TimelineEntry struct {
	EventType   string         `json:"event_type"`
	ActorName   string         `json:"actor_name"`
	ActorKind   string         `json:"actor_kind"`
	Content     string         `json:"content"`
	DraftStatus string         `json:"draft_status,omitempty"`
	RecordedAt  time.Time      `json:"recorded_at"`
	Notes       []TimelineNote `json:"notes,omitempty"`
}

// CaseDashboardRow represents a row in the case_dashboard read model.
type CaseDashboardRow struct {
	WorkItemID     string
	Status         string
	PartyName      string
	PartyActorKind string
	SubjectName    string
	SubjectDetail  string
	Timeline       []TimelineEntry
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// DashboardStore provides read-model operations for the case dashboard.
type DashboardStore struct {
	pool *pgxpool.Pool
}

// NewDashboardStore creates a new dashboard store.
func NewDashboardStore(pool *pgxpool.Pool) *DashboardStore {
	return &DashboardStore{pool: pool}
}

// Upsert inserts or updates a case dashboard row.
func (s *DashboardStore) Upsert(ctx context.Context, row CaseDashboardRow) error {
	timelineJSON, err := json.Marshal(row.Timeline)
	if err != nil {
		return fmt.Errorf("marshal timeline: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO case_dashboard (work_item_id, status, party_name, party_actor_kind, subject_name, subject_detail, timeline_json)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (work_item_id) DO UPDATE SET
			status = EXCLUDED.status,
			party_name = EXCLUDED.party_name,
			party_actor_kind = EXCLUDED.party_actor_kind,
			subject_name = EXCLUDED.subject_name,
			subject_detail = EXCLUDED.subject_detail,
			timeline_json = EXCLUDED.timeline_json,
			updated_at = now()
	`, row.WorkItemID, row.Status, row.PartyName, row.PartyActorKind, row.SubjectName, row.SubjectDetail, timelineJSON)
	if err != nil {
		return fmt.Errorf("upsert case dashboard: %w", err)
	}
	return nil
}

// AppendTimeline appends a timeline entry to an existing case dashboard row.
func (s *DashboardStore) AppendTimeline(ctx context.Context, workItemID string, entry TimelineEntry) error {
	entryJSON, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal timeline entry: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		UPDATE case_dashboard
		SET timeline_json = timeline_json || $2::jsonb,
		    updated_at = now()
		WHERE work_item_id = $1
	`, workItemID, entryJSON)
	if err != nil {
		return fmt.Errorf("append timeline: %w", err)
	}
	return nil
}

// UpdateStatus updates the status of a case dashboard row.
func (s *DashboardStore) UpdateStatus(ctx context.Context, workItemID, status string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE case_dashboard
		SET status = $2, updated_at = now()
		WHERE work_item_id = $1
	`, workItemID, status)
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	return nil
}

// FindAll returns all case dashboard rows ordered by creation date descending.
func (s *DashboardStore) FindAll(ctx context.Context) ([]CaseDashboardRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT work_item_id, status, party_name, party_actor_kind, subject_name, subject_detail, timeline_json, created_at, updated_at
		FROM case_dashboard
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query case dashboard: %w", err)
	}
	defer rows.Close()

	var result []CaseDashboardRow
	for rows.Next() {
		r, err := scanDashboardRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// FindByID returns a single case dashboard row by work item ID.
func (s *DashboardStore) FindByID(ctx context.Context, id string) (CaseDashboardRow, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT work_item_id, status, party_name, party_actor_kind, subject_name, subject_detail, timeline_json, created_at, updated_at
		FROM case_dashboard
		WHERE work_item_id = $1
	`, id)

	var r CaseDashboardRow
	var timelineJSON []byte
	err := row.Scan(
		&r.WorkItemID, &r.Status, &r.PartyName, &r.PartyActorKind,
		&r.SubjectName, &r.SubjectDetail, &timelineJSON,
		&r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return CaseDashboardRow{}, fmt.Errorf("case not found: %s", id)
		}
		return CaseDashboardRow{}, fmt.Errorf("scan case dashboard: %w", err)
	}
	if err := json.Unmarshal(timelineJSON, &r.Timeline); err != nil {
		return CaseDashboardRow{}, fmt.Errorf("unmarshal timeline: %w", err)
	}
	return r, nil
}

// DeleteAll removes all rows from the case dashboard. Used for replay tests.
func (s *DashboardStore) DeleteAll(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, "DELETE FROM case_dashboard")
	if err != nil {
		return fmt.Errorf("delete all case dashboard: %w", err)
	}
	return nil
}

// AddNoteToTimeline adds a note to a specific timeline entry.
func (s *DashboardStore) AddNoteToTimeline(ctx context.Context, workItemID string, entryIndex int, note TimelineNote) error {
	return s.modifyTimeline(ctx, workItemID, func(timeline []TimelineEntry) ([]TimelineEntry, error) {
		if entryIndex < 0 || entryIndex >= len(timeline) {
			return nil, fmt.Errorf("entry index %d out of range (len=%d)", entryIndex, len(timeline))
		}
		timeline[entryIndex].Notes = append(timeline[entryIndex].Notes, note)
		return timeline, nil
	})
}

// EditNoteOnTimeline updates a note's body across all timeline entries.
func (s *DashboardStore) EditNoteOnTimeline(ctx context.Context, workItemID, noteID, body string, editedAt time.Time) error {
	return s.modifyTimeline(ctx, workItemID, func(timeline []TimelineEntry) ([]TimelineEntry, error) {
		for i := range timeline {
			for j := range timeline[i].Notes {
				if timeline[i].Notes[j].NoteID == noteID {
					timeline[i].Notes[j].Body = body
					timeline[i].Notes[j].EditedAt = &editedAt
					return timeline, nil
				}
			}
		}
		return nil, fmt.Errorf("note %s not found", noteID)
	})
}

// DeleteNoteOnTimeline marks a note as deleted across all timeline entries.
func (s *DashboardStore) DeleteNoteOnTimeline(ctx context.Context, workItemID, noteID string) error {
	return s.modifyTimeline(ctx, workItemID, func(timeline []TimelineEntry) ([]TimelineEntry, error) {
		for i := range timeline {
			for j := range timeline[i].Notes {
				if timeline[i].Notes[j].NoteID == noteID {
					timeline[i].Notes[j].Deleted = true
					return timeline, nil
				}
			}
		}
		return nil, fmt.Errorf("note %s not found", noteID)
	})
}

// FindNotesByEntryIndex returns all non-deleted notes for a specific timeline entry.
func (s *DashboardStore) FindNotesByEntryIndex(ctx context.Context, workItemID string, entryIndex int) ([]TimelineNote, error) {
	c, err := s.FindByID(ctx, workItemID)
	if err != nil {
		return nil, err
	}
	if entryIndex < 0 || entryIndex >= len(c.Timeline) {
		return nil, fmt.Errorf("entry index %d out of range", entryIndex)
	}
	var notes []TimelineNote
	for _, n := range c.Timeline[entryIndex].Notes {
		if !n.Deleted {
			notes = append(notes, n)
		}
	}
	return notes, nil
}

// modifyTimeline loads the timeline, applies a mutation, and saves back.
func (s *DashboardStore) modifyTimeline(ctx context.Context, workItemID string, fn func([]TimelineEntry) ([]TimelineEntry, error)) error {
	var timelineJSON []byte
	err := s.pool.QueryRow(ctx, `
		SELECT timeline_json FROM case_dashboard WHERE work_item_id = $1
	`, workItemID).Scan(&timelineJSON)
	if err != nil {
		return fmt.Errorf("load timeline for %s: %w", workItemID, err)
	}

	var timeline []TimelineEntry
	if err := json.Unmarshal(timelineJSON, &timeline); err != nil {
		return fmt.Errorf("unmarshal timeline: %w", err)
	}

	timeline, err = fn(timeline)
	if err != nil {
		return err
	}

	updated, err := json.Marshal(timeline)
	if err != nil {
		return fmt.Errorf("marshal timeline: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		UPDATE case_dashboard SET timeline_json = $2, updated_at = now() WHERE work_item_id = $1
	`, workItemID, updated)
	if err != nil {
		return fmt.Errorf("save timeline: %w", err)
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanDashboardRow(s scanner) (CaseDashboardRow, error) {
	var r CaseDashboardRow
	var timelineJSON []byte
	err := s.Scan(
		&r.WorkItemID, &r.Status, &r.PartyName, &r.PartyActorKind,
		&r.SubjectName, &r.SubjectDetail, &timelineJSON,
		&r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return CaseDashboardRow{}, fmt.Errorf("scan case dashboard: %w", err)
	}
	if err := json.Unmarshal(timelineJSON, &r.Timeline); err != nil {
		return CaseDashboardRow{}, fmt.Errorf("unmarshal timeline: %w", err)
	}
	return r, nil
}
