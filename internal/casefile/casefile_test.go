package casefile

import (
	"errors"
	"testing"
	"time"
)

func TestUpsertDeleteRoundTrip(t *testing.T) {
	s := Open(t.TempDir())
	now := time.Date(2026, 11, 4, 10, 0, 0, 0, time.Local)

	if _, err := s.Upsert(Complaint{}, now); err == nil {
		t.Fatal("a complaint with no ticket and no notes must be rejected")
	}

	c, err := s.Upsert(Complaint{Provider: "  WE  ", Ticket: "12345", OpenedAt: "2026-11-03T09:30"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if c.ID == "" || c.Provider != "WE" || c.Status != StatusOpen {
		t.Fatalf("unexpected complaint %+v", c)
	}

	c.Status = StatusResolved
	c2, err := s.Upsert(c, now)
	if err != nil {
		t.Fatal(err)
	}
	if c2.ClosedAt == "" {
		t.Fatal("resolving a complaint must set closed_at")
	}

	f, err := s.Load()
	if err != nil || len(f.Complaints) != 1 {
		t.Fatalf("load: %v %+v", err, f)
	}

	if _, err := s.Upsert(Complaint{ID: "missing", Ticket: "1"}, now); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
	if err := s.Delete(c.ID); err != nil {
		t.Fatal(err)
	}
	f, _ = s.Load()
	if len(f.Complaints) != 0 {
		t.Fatal("complaint was not deleted")
	}
}

func TestProfileClean(t *testing.T) {
	s := Open(t.TempDir())
	f, err := s.SetProfile(Profile{Provider: " Vodafone ", PlanDownloadMbps: -5, PlanUploadMbps: 10})
	if err != nil {
		t.Fatal(err)
	}
	if f.Profile.Provider != "Vodafone" || f.Profile.PlanDownloadMbps != 0 || f.Profile.PlanUploadMbps != 10 {
		t.Fatalf("unexpected profile %+v", f.Profile)
	}
}

func TestInMonth(t *testing.T) {
	cs := []Complaint{
		{ID: "a", OpenedAt: "2026-10-20T10:00:00Z", ClosedAt: "2026-10-25T10:00:00Z"},
		{ID: "b", OpenedAt: "2026-10-28T10:00:00Z"},
		{ID: "c", OpenedAt: "2026-11-02T10:00:00Z"},
		{ID: "d", OpenedAt: "2026-12-01T10:00:00Z"},
	}
	got := InMonth(cs, "2026-11", time.UTC)
	if len(got) != 2 || got[0].ID != "b" || got[1].ID != "c" {
		t.Fatalf("got %+v", got)
	}
}
