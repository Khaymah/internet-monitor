package seal

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	ChainFile = "seals.jsonl"
	StampDir  = "seals"
)

var Prefixes = []string{"connectivity_", "metrics_", "speedtest_", "traceroute_", "notifications_"}

type File struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type Seal struct {
	Date     string `json:"date"`
	SealedAt string `json:"sealed_at"`
	Files    []File `json:"files"`
	Prev     string `json:"prev"`
	Hash     string `json:"hash"`
	Stamped  bool   `json:"stamped,omitempty"`
}

type Stamper func(digest []byte) ([]byte, error)

type Sealer struct {
	Dir   func() string
	Stamp func() Stamper
	Logf  func(format string, args ...interface{})
	Now   func() time.Time
	mu    sync.Mutex
}

func (s Seal) body() Seal {
	s.Hash = ""
	s.Stamped = false
	return s
}

func Canonical(s Seal) []byte {
	b, _ := json.Marshal(s.body())
	return b
}

func digest(s Seal) []byte {
	h := sha256.Sum256(Canonical(s))
	return h[:]
}

func hashFile(path string) (File, error) {
	f, err := os.Open(path)
	if err != nil {
		return File{}, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return File{}, err
	}
	return File{Name: filepath.Base(path), SHA256: hex.EncodeToString(h.Sum(nil)), Size: n}, nil
}

func dayOf(name string) (string, bool) {
	if !strings.HasSuffix(name, ".jsonl") {
		return "", false
	}
	for _, p := range Prefixes {
		if strings.HasPrefix(name, p) {
			d := strings.TrimSuffix(strings.TrimPrefix(name, p), ".jsonl")
			if _, err := time.Parse("2006-01-02", d); err == nil {
				return d, true
			}
		}
	}
	return "", false
}

func filesByDay(dir string) map[string][]string {
	out := map[string][]string{}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if d, ok := dayOf(e.Name()); ok {
			out[d] = append(out[d], e.Name())
		}
	}
	for d := range out {
		sort.Strings(out[d])
	}
	return out
}

func Load(dir string) ([]Seal, error) {
	f, err := os.Open(filepath.Join(dir, ChainFile))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []Seal
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64<<10), 4<<20)
	for sc.Scan() {
		var s Seal
		if json.Unmarshal(sc.Bytes(), &s) == nil {
			out = append(out, s)
		}
	}
	return out, sc.Err()
}

func (sl *Sealer) now() time.Time {
	if sl.Now != nil {
		return sl.Now()
	}
	return time.Now()
}

func (sl *Sealer) logf(format string, args ...interface{}) {
	if sl.Logf != nil {
		sl.Logf(format, args...)
	}
}

func (sl *Sealer) SealPending() (int, error) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	dir := sl.Dir()
	seals, err := Load(dir)
	if err != nil {
		return 0, err
	}
	done := map[string]bool{}
	prev := ""
	for _, s := range seals {
		done[s.Date] = true
		prev = s.Hash
	}
	today := sl.now().Format("2006-01-02")
	byDay := filesByDay(dir)
	days := make([]string, 0, len(byDay))
	for d := range byDay {
		if d < today && !done[d] {
			days = append(days, d)
		}
	}
	sort.Strings(days)
	if len(days) == 0 {
		return 0, nil
	}
	f, err := os.OpenFile(filepath.Join(dir, ChainFile), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	var stamp Stamper
	if sl.Stamp != nil {
		stamp = sl.Stamp()
	}
	n := 0
	for _, d := range days {
		s := Seal{Date: d, SealedAt: sl.now().Format(time.RFC3339), Prev: prev}
		for _, name := range byDay[d] {
			fh, err := hashFile(filepath.Join(dir, name))
			if err != nil {
				return n, err
			}
			s.Files = append(s.Files, fh)
		}
		s.Hash = hex.EncodeToString(digest(s))
		if stamp != nil {
			if err := sl.writeStamp(dir, s, stamp); err != nil {
				sl.logf("SEAL timestamp for %s failed: %v", d, err)
			} else {
				s.Stamped = true
			}
		}
		line, _ := json.Marshal(s)
		if _, err := f.Write(append(line, '\n')); err != nil {
			return n, err
		}
		prev = s.Hash
		n++
	}
	return n, nil
}

func (sl *Sealer) writeStamp(dir string, s Seal, stamp Stamper) error {
	reply, err := stamp(digest(s))
	if err != nil {
		return err
	}
	sd := filepath.Join(dir, StampDir)
	if err := os.MkdirAll(sd, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(sd, s.Date+".seal.json"), Canonical(s), 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(sd, s.Date+".tsr"), reply, 0o644)
}

type Problem struct {
	Date   string `json:"date"`
	Reason string `json:"reason"`
	File   string `json:"file,omitempty"`
}

type Status struct {
	Days      int       `json:"days"`
	Stamped   int       `json:"stamped"`
	First     string    `json:"first,omitempty"`
	Last      string    `json:"last,omitempty"`
	Expired   int       `json:"expired"`
	Intact    bool      `json:"intact"`
	Problems  []Problem `json:"problems"`
	CheckedAt string    `json:"checked_at"`
}

const Retention = 90 * 24 * time.Hour

func Verify(dir string, now time.Time) (Status, error) {
	cutoff := now.Add(-Retention).Format("2006-01-02")
	st := Status{Intact: true, Problems: []Problem{}, CheckedAt: now.Format(time.RFC3339)}
	seals, err := Load(dir)
	if err != nil {
		return st, err
	}
	prev := ""
	for _, s := range seals {
		st.Days++
		if st.First == "" {
			st.First = s.Date
		}
		st.Last = s.Date
		if s.Stamped {
			st.Stamped++
		}
		if s.Prev != prev {
			st.Problems = append(st.Problems, Problem{Date: s.Date, Reason: "chain"})
		}
		if hex.EncodeToString(digest(s)) != s.Hash {
			st.Problems = append(st.Problems, Problem{Date: s.Date, Reason: "seal_edited"})
		}
		for _, f := range s.Files {
			got, err := hashFile(filepath.Join(dir, f.Name))
			switch {
			case errors.Is(err, os.ErrNotExist) && s.Date <= cutoff:
				st.Expired++
			case errors.Is(err, os.ErrNotExist):
				st.Problems = append(st.Problems, Problem{Date: s.Date, Reason: "file_missing", File: f.Name})
			case err != nil:
				return st, fmt.Errorf("read %s: %w", f.Name, err)
			case got.SHA256 != f.SHA256:
				st.Problems = append(st.Problems, Problem{Date: s.Date, Reason: "file_changed", File: f.Name})
			}
		}
		prev = s.Hash
	}
	st.Intact = len(st.Problems) == 0
	return st, nil
}

func (sl *Sealer) Run(stop <-chan struct{}) {
	t := time.NewTicker(time.Hour)
	defer t.Stop()
	for {
		if n, err := sl.SealPending(); err != nil {
			sl.logf("SEAL %v", err)
		} else if n > 0 {
			sl.logf("SEAL sealed %d day(s)", n)
		}
		select {
		case <-t.C:
		case <-stop:
			return
		}
	}
}
