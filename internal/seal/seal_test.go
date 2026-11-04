package seal

import (
	"encoding/asn1"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func write(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSealChainAndTamperDetection(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "connectivity_2026-11-01.jsonl", "a\n")
	write(t, dir, "metrics_2026-11-01.jsonl", "b\n")
	write(t, dir, "metrics_2026-11-02.jsonl", "c\n")
	write(t, dir, "metrics_2026-11-03.jsonl", "today\n")
	write(t, dir, "app.log", "not sealed\n")

	stamps := 0
	sl := &Sealer{
		Dir: func() string { return dir },
		Now: func() time.Time { return time.Date(2026, 11, 3, 12, 0, 0, 0, time.Local) },
		Stamp: func() Stamper {
			return func(d []byte) ([]byte, error) { stamps++; return []byte("token"), nil }
		},
	}
	n, err := sl.SealPending()
	if err != nil || n != 2 {
		t.Fatalf("sealed %d, err %v; want the two finished days", n, err)
	}
	if n, _ := sl.SealPending(); n != 0 {
		t.Fatalf("second run sealed %d, want 0", n)
	}
	if stamps != 2 {
		t.Fatalf("stamps = %d, want 2", stamps)
	}
	if _, err := os.Stat(filepath.Join(dir, StampDir, "2026-11-01.tsr")); err != nil {
		t.Fatal("timestamp token not written")
	}

	st, err := Verify(dir, time.Now())
	if err != nil || !st.Intact || st.Days != 2 || st.Stamped != 2 || st.First != "2026-11-01" {
		t.Fatalf("status = %+v, err %v", st, err)
	}

	write(t, dir, "connectivity_2026-11-01.jsonl", "edited\n")
	st, _ = Verify(dir, time.Now())
	if st.Intact || len(st.Problems) != 1 || st.Problems[0].Reason != "file_changed" || st.Problems[0].File != "connectivity_2026-11-01.jsonl" {
		t.Fatalf("edit not detected: %+v", st)
	}

	write(t, dir, "connectivity_2026-11-01.jsonl", "a\n")
	if err := os.Remove(filepath.Join(dir, "metrics_2026-11-01.jsonl")); err != nil {
		t.Fatal(err)
	}
	if st, _ := Verify(dir, time.Date(2026, 11, 10, 0, 0, 0, 0, time.Local)); st.Intact || st.Problems[0].Reason != "file_missing" {
		t.Fatalf("a recently deleted file must be reported: %+v", st)
	}
	if st, _ := Verify(dir, time.Date(2027, 3, 1, 0, 0, 0, 0, time.Local)); !st.Intact || st.Expired != 1 {
		t.Fatalf("a file past the 90-day retention is expired, not tampering: %+v", st)
	}

	seals, _ := Load(dir)
	if seals[1].Prev != seals[0].Hash || seals[0].Prev != "" {
		t.Fatal("seals are not chained")
	}
}

func TestSealWithoutStamper(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "metrics_2026-11-01.jsonl", "x\n")
	sl := &Sealer{Dir: func() string { return dir }, Now: func() time.Time { return time.Date(2026, 11, 2, 0, 0, 1, 0, time.Local) }}
	if n, err := sl.SealPending(); n != 1 || err != nil {
		t.Fatalf("n=%d err=%v", n, err)
	}
	seals, _ := Load(dir)
	if seals[0].Stamped {
		t.Fatal("must not claim a timestamp without a stamper")
	}
}

func TestTimestampRequestIsValidDER(t *testing.T) {
	digest := make([]byte, 32)
	der, err := Request(digest)
	if err != nil {
		t.Fatal(err)
	}
	var req timeStampReq
	if _, err := asn1.Unmarshal(der, &req); err != nil {
		t.Fatalf("request does not round-trip: %v", err)
	}
	if req.Version != 1 || !req.MessageImprint.HashAlgorithm.Algorithm.Equal(oidSHA256) || !req.CertReq {
		t.Fatalf("request = %+v", req)
	}
	if _, err := Request(digest[:20]); err == nil {
		t.Fatal("non-SHA-256 digest accepted")
	}

	granted, _ := asn1.Marshal(timeStampResp{Status: pkiStatusInfo{Status: 0}, Token: asn1.RawValue{Tag: asn1.TagSequence, IsCompound: true, Class: asn1.ClassUniversal, Bytes: []byte{0x02, 0x01, 0x01}}})
	if err := CheckReply(granted); err != nil {
		t.Fatalf("granted reply rejected: %v", err)
	}
	refused, _ := asn1.Marshal(timeStampResp{Status: pkiStatusInfo{Status: 2}})
	if err := CheckReply(refused); err == nil {
		t.Fatal("refused reply accepted")
	}
}
