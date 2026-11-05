package router

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const hg630DSL = `while(1); /*{"DownPower":14.5,"Modulation":"VDSL","UpCurrRate":5120,"ShowtimeStart":750582,"DownstreamMaxBitRate":65532,"DownAttenuation":13.3,"Status":"Up","DataPath":"Fast","UpstreamMaxBitRate":17271,"UpPower":8.6,"UpAttenuation":5.2,"DownMargin":6,"UpMargin":15.9,"DownCurrRate":40957}*/`

const hg630WAN = `while(1); /*[{"receivebytes":3103873845,"sendbytes":3855775661,"Name":"INTERNET_TR069_R_VID"},{"receivebytes":0,"sendbytes":0,"Name":"Other_R_VID_102"}]*/`

type fakeWeb struct {
	password string
	logins   int
	logouts  int
	busy     bool
}

func (f *fakeWeb) handler(t *testing.T) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "SessionID_R3", Value: "anon", Path: "/"})
		io.WriteString(w, `<html><head><meta name="csrf_param" content="P123"/><meta name="csrf_token" content="T456"/></head></html>`)
	})
	mux.HandleFunc("/api/system/user_login", func(w http.ResponseWriter, r *http.Request) {
		f.logins++
		var body struct {
			Csrf map[string]string `json:"csrf"`
			Data map[string]string `json:"data"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		switch {
		case f.busy:
			io.WriteString(w, `while(1); /*{"errorCategory":"Duplicate_login"}*/`)
		case body.Data["Password"] == loginHash(body.Data["UserName"], f.password, body.Csrf["csrf_param"], body.Csrf["csrf_token"]):
			http.SetCookie(w, &http.Cookie{Name: "SessionID_R3", Value: "ok", Path: "/"})
			io.WriteString(w, `while(1); /*{"errorCategory":"ok","level":2}*/`)
		default:
			io.WriteString(w, `while(1); /*{"errorCategory":"user_pass_err","count":1}*/`)
		}
	})
	mux.HandleFunc("/api/system/user_logout", func(w http.ResponseWriter, _ *http.Request) {
		f.logouts++
		io.WriteString(w, `while(1); /*{"errorCategory":"ok"}*/`)
	})
	authed := func(body string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if c, err := r.Cookie("SessionID_R3"); err != nil || c.Value != "ok" {
				http.Redirect(w, r, "/", http.StatusFound)
				return
			}
			io.WriteString(w, body)
		}
	}
	mux.HandleFunc("/api/ntwk/dslinfo", authed(hg630DSL))
	mux.HandleFunc("/api/ntwk/wan_st", authed(hg630WAN))
	return mux
}

func hostOfServer(s *httptest.Server) string { return strings.TrimPrefix(s.URL, "http://") }

func TestWebReadHG630(t *testing.T) {
	f := &fakeWeb{password: "s3cret"}
	srv := httptest.NewServer(f.handler(t))
	defer srv.Close()
	st, err := WebReader{Host: hostOfServer(srv), Password: "s3cret"}.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if st.DownKbps != 40957 || st.UpKbps != 5120 || st.MaxDownKbps != 65532 || st.SNRDown != 6 || st.SNRUp != 15.9 || st.AttnDown != 13.3 || st.LineUpSecs != 750582 {
		t.Fatalf("line stats %+v", st)
	}
	if !st.HasCounters || st.RxBytes != 3103873845 || st.TxBytes != 3855775661 {
		t.Fatalf("counters %+v", st)
	}
	if f.logins != 1 || f.logouts != 1 {
		t.Fatalf("logins %d logouts %d", f.logins, f.logouts)
	}
}

func TestWebReadErrors(t *testing.T) {
	f := &fakeWeb{password: "right"}
	srv := httptest.NewServer(f.handler(t))
	defer srv.Close()
	_, err := WebReader{Host: hostOfServer(srv), Password: "wrong"}.Read(context.Background())
	if !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("got %v", err)
	}
	f.busy = true
	_, err = WebReader{Host: hostOfServer(srv), Password: "right"}.Read(context.Background())
	if !errors.Is(err, ErrRouterBusy) {
		t.Fatalf("got %v", err)
	}
	if _, err := (WebReader{Host: "8.8.8.8", Password: "x"}).Read(context.Background()); err == nil {
		t.Fatal("a public address must be refused")
	}
}

func TestPollerWebStopsAfterWrongPassword(t *testing.T) {
	f := &fakeWeb{password: "right"}
	srv := httptest.NewServer(f.handler(t))
	defer srv.Close()
	clock := at(0)
	password := "wrong"
	var saved []Status
	p := &Poller{
		Discover: func(context.Context) (string, error) { return "", ErrNotFound },
		Web: func() (WebReader, bool) {
			return WebReader{Host: hostOfServer(srv), Password: password}, true
		},
		Save: func(_ time.Time, s Status) { saved = append(saved, s) },
		Now:  func() time.Time { return clock },
	}
	for range 3 {
		p.Tick(context.Background())
		clock = clock.Add(10 * time.Minute)
	}
	if f.logins != 1 {
		t.Fatalf("a rejected password must be tried once, tried %d times", f.logins)
	}
	if snap := p.Snapshot(); snap.WebError != "wrong_password" {
		t.Fatalf("snapshot %+v", snap)
	}
	password = "right"
	p.Tick(context.Background())
	if len(saved) != 1 || saved[0].SNRDown == nil || *saved[0].SNRDown != 6 || !saved[0].HasCounters || saved[0].LinkUp == nil || !*saved[0].LinkUp {
		t.Fatalf("a new password must be tried and its reading saved: %+v", saved)
	}
	before := f.logins
	clock = clock.Add(2 * time.Minute)
	p.Tick(context.Background())
	if f.logins != before {
		t.Fatal("the page must not be read again before the interval")
	}
}

func TestWebInterval(t *testing.T) {
	if got := webInterval(0); got != webEvery {
		t.Fatalf("no sync: %v", got)
	}
	if got := webInterval(40957); got != webEvery {
		t.Fatalf("40 Mbps: %v", got)
	}
	if got := webInterval(100000); got >= webEvery || got < webMinEvery {
		t.Fatalf("100 Mbps must read more often than every %v so counters wrap at most once: %v", webEvery, got)
	}
}

func TestClassifyLineRetrain(t *testing.T) {
	samples := []Status{
		{Time: at(9), LineUpSecs: 5000, UptimeSecs: 90000},
		{Time: at(14), LineUpSecs: 40, UptimeSecs: 90300},
	}
	if got := Classify(samples, at(10), at(13)); got != CauseLineDown {
		t.Fatalf("a line that resynced during the outage went down: got %q", got)
	}
}
