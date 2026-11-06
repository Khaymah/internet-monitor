package router

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

const hg659WAN = `while(1); /*{"receivebytes":5000000000,"sendbytes":1000000000,"Name":"INTERNET_R_VID_100"}*/`

func (f *fakeWeb) familyHandler(t *testing.T, rootMarkers, indexMarkers bool) http.Handler {
	inner := f.handler(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if rootMarkers {
			inner.ServeHTTP(w, r)
			return
		}
		io.WriteString(w, `<html><head><title>Router</title></head><body>loading</body></html>`)
	})
	mux.HandleFunc("/html/index.html", func(w http.ResponseWriter, r *http.Request) {
		if !indexMarkers {
			http.NotFound(w, r)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "SessionID_R3", Value: "anon", Path: "/"})
		io.WriteString(w, `<html><head><meta name="csrf_param" content="P123"/><meta name="csrf_token" content="T456"/></head></html>`)
	})
	mux.HandleFunc("/api/ntwk/dslinfo", http.NotFound)
	mux.HandleFunc("/api/ntwk/wan_st", func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("SessionID_R3"); err != nil || c.Value != "ok" {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		io.WriteString(w, hg659WAN)
	})
	mux.HandleFunc("/api/system/deviceinfo", func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `while(1); /*{"DeviceName":"HG659","SerialNumber":"X1","UpTime":123}*/`)
	})
	mux.Handle("/api/", inner)
	return mux
}

func TestHuaweiFamilyWithoutLineStats(t *testing.T) {
	f := &fakeWeb{password: "s3cret"}
	srv := httptest.NewServer(f.familyHandler(t, false, true))
	defer srv.Close()
	st, err := WebReader{Host: hostOfServer(srv), Password: "s3cret"}.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if st.Driver != DriverHuaweiHG || st.HasLineStats || !st.HasCounters || st.RxBytes != 5000000000 || st.TxBytes != 1000000000 || st.Model != "HG659" {
		t.Fatalf("a gateway without a DSL page must still give its counters: %+v", st)
	}
	if f.logins != 1 || f.logouts != 1 {
		t.Fatalf("logins %d logouts %d", f.logins, f.logouts)
	}
}

func TestHuaweiDetectsByDeviceInfoAlone(t *testing.T) {
	f := &fakeWeb{password: "s3cret"}
	srv := httptest.NewServer(f.familyHandler(t, false, false))
	defer srv.Close()
	p, err := NewProbe(hostOfServer(srv), 0)
	if err != nil {
		t.Fatal(err)
	}
	if ok, _ := (huaweiHG{}).Detect(context.Background(), p); !ok {
		t.Fatal("the device info page must identify a Huawei gateway")
	}
}

func TestHuaweiNotDetectedOnOtherPages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, "<html>some other router</html>")
	}))
	defer srv.Close()
	p, _ := NewProbe(hostOfServer(srv), 0)
	if ok, _ := (huaweiHG{}).Detect(context.Background(), p); ok {
		t.Fatal("an unrelated page must not match")
	}
	if _, err := Detect(context.Background(), p, Drivers(), false); err != ErrWebUnknown {
		t.Fatalf("got %v", err)
	}
}

func TestParseWANObject(t *testing.T) {
	rx, tx, ok := parseWAN([]byte(hg659WAN))
	if !ok || rx != 5000000000 || tx != 1000000000 {
		t.Fatalf("%d %d %v", rx, tx, ok)
	}
}
