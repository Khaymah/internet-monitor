package casefile

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const FileName = "case.json"

const (
	MaxComplaints = 500
	maxShort      = 120
	maxNotes      = 2000
)

var ErrNotFound = errors.New("complaint not found")

type Profile struct {
	Country          string  `json:"country"`
	ResponseHours    int     `json:"response_hours"`
	Provider         string  `json:"provider"`
	PlanName         string  `json:"plan_name"`
	PlanDownloadMbps float64 `json:"plan_download_mbps"`
	PlanUploadMbps   float64 `json:"plan_upload_mbps"`
	LineNumber       string  `json:"line_number"`
	AccountNumber    string  `json:"account_number"`
}

const (
	StatusOpen      = "open"
	StatusEscalated = "escalated"
	StatusResolved  = "resolved"
	StatusRejected  = "rejected"
)

type Complaint struct {
	ID          string `json:"id"`
	Provider    string `json:"provider"`
	Ticket      string `json:"ticket"`
	OpenedAt    string `json:"opened_at"`
	Channel     string `json:"channel"`
	Agent       string `json:"agent"`
	Status      string `json:"status"`
	EscalatedAt string `json:"escalated_at,omitempty"`
	Regulator   string `json:"regulator_ticket,omitempty"`
	ClosedAt    string `json:"closed_at,omitempty"`
	Notes       string `json:"notes"`
}

type File struct {
	Profile    Profile     `json:"profile"`
	Complaints []Complaint `json:"complaints"`
}

type Store struct {
	mu   sync.Mutex
	path string
}

func Open(dir string) *Store { return &Store{path: filepath.Join(dir, FileName)} }

func (s *Store) Path() string { return s.path }

func (s *Store) Load() (File, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load()
}

func (s *Store) load() (File, error) {
	f := File{Complaints: []Complaint{}}
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return f, nil
	}
	if err != nil {
		return f, err
	}
	if err := json.Unmarshal(data, &f); err != nil {
		return File{Complaints: []Complaint{}}, fmt.Errorf("case.json is not valid JSON: %w", err)
	}
	if f.Complaints == nil {
		f.Complaints = []Complaint{}
	}
	sortComplaints(f.Complaints)
	return f, nil
}

func (s *Store) save(f File) error {
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *Store) SetProfile(p Profile) (File, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := s.load()
	if err != nil {
		return f, err
	}
	f.Profile = p.clean()
	return f, s.save(f)
}

func (s *Store) Upsert(c Complaint, now time.Time) (Complaint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := s.load()
	if err != nil {
		return c, err
	}
	c = c.clean(now)
	if err := c.validate(); err != nil {
		return c, err
	}
	if c.ID == "" {
		if len(f.Complaints) >= MaxComplaints {
			return c, &FieldError{Field: "complaints", Message: "too many complaints"}
		}
		c.ID = newID()
		f.Complaints = append(f.Complaints, c)
	} else {
		i := indexOf(f.Complaints, c.ID)
		if i < 0 {
			return c, ErrNotFound
		}
		f.Complaints[i] = c
	}
	sortComplaints(f.Complaints)
	return c, s.save(f)
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := s.load()
	if err != nil {
		return err
	}
	i := indexOf(f.Complaints, id)
	if i < 0 {
		return ErrNotFound
	}
	f.Complaints = append(f.Complaints[:i], f.Complaints[i+1:]...)
	return s.save(f)
}

func InMonth(cs []Complaint, month string, loc *time.Location) []Complaint {
	start, err := time.ParseInLocation("2006-01", month, loc)
	if err != nil {
		return cs
	}
	end := start.AddDate(0, 1, 0)
	out := []Complaint{}
	for _, c := range cs {
		opened, ok := parseTime(c.OpenedAt, loc)
		if !ok || !opened.Before(end) {
			continue
		}
		if closed, ok := parseTime(c.ClosedAt, loc); ok && closed.Before(start) {
			continue
		}
		out = append(out, c)
	}
	return out
}

type FieldError struct {
	Field   string
	Message string
}

func (e *FieldError) Error() string { return e.Field + ": " + e.Message }

var Countries = []string{"EG", "SA", "GB", "US"}

const maxResponseHours = 24 * 120

func (p Profile) clean() Profile {
	p.Country = strings.ToUpper(strings.TrimSpace(p.Country))
	known := false
	for _, c := range Countries {
		known = known || c == p.Country
	}
	if !known {
		p.Country = ""
	}
	p.ResponseHours = max(0, min(p.ResponseHours, maxResponseHours))
	p.Provider = short(p.Provider)
	p.PlanName = short(p.PlanName)
	p.LineNumber = short(p.LineNumber)
	p.AccountNumber = short(p.AccountNumber)
	p.PlanDownloadMbps = clampMbps(p.PlanDownloadMbps)
	p.PlanUploadMbps = clampMbps(p.PlanUploadMbps)
	return p
}

func (c Complaint) clean(now time.Time) Complaint {
	c.ID = strings.TrimSpace(c.ID)
	c.Provider = short(c.Provider)
	c.Ticket = short(c.Ticket)
	c.Channel = short(c.Channel)
	c.Agent = short(c.Agent)
	c.Regulator = short(c.Regulator)
	c.Notes = limit(strings.TrimSpace(c.Notes), maxNotes)
	c.OpenedAt = normTime(c.OpenedAt)
	c.EscalatedAt = normTime(c.EscalatedAt)
	c.ClosedAt = normTime(c.ClosedAt)
	if c.OpenedAt == "" {
		c.OpenedAt = now.Format(time.RFC3339)
	}
	switch c.Status {
	case StatusOpen, StatusEscalated, StatusResolved, StatusRejected:
	default:
		c.Status = StatusOpen
	}
	if c.Status == StatusEscalated && c.EscalatedAt == "" {
		c.EscalatedAt = now.Format(time.RFC3339)
	}
	if (c.Status == StatusResolved || c.Status == StatusRejected) && c.ClosedAt == "" {
		c.ClosedAt = now.Format(time.RFC3339)
	}
	if c.Status == StatusOpen || c.Status == StatusEscalated {
		c.ClosedAt = ""
	}
	return c
}

func (c Complaint) validate() error {
	if c.Ticket == "" && c.Notes == "" {
		return &FieldError{Field: "ticket", Message: "enter the complaint number or a note"}
	}
	return nil
}

func short(v string) string { return limit(strings.TrimSpace(v), maxShort) }

func limit(v string, n int) string {
	if utf8.RuneCountInString(v) <= n {
		return v
	}
	return string([]rune(v)[:n])
}

func clampMbps(v float64) float64 {
	if v < 0 || v != v {
		return 0
	}
	return min(v, 100000)
}

func normTime(v string) string {
	t, ok := parseTime(v, time.Local)
	if !ok {
		return ""
	}
	return t.Format(time.RFC3339)
}

func parseTime(v string, loc *time.Location) (time.Time, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return time.Time{}, false
	}
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t, true
	}
	for _, layout := range []string{"2006-01-02T15:04", "2006-01-02T15:04:05", "2006-01-02"} {
		if t, err := time.ParseInLocation(layout, v, loc); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func indexOf(cs []Complaint, id string) int {
	for i, c := range cs {
		if c.ID == id {
			return i
		}
	}
	return -1
}

func sortComplaints(cs []Complaint) {
	sort.SliceStable(cs, func(i, j int) bool { return cs[i].OpenedAt > cs[j].OpenedAt })
}

func newID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
