package power

import "testing"

func TestParsers(t *testing.T) {
	if bt, ok := ParseProcStat("cpu  1 2 3\nbtime 1762300000\nprocesses 9\n"); !ok || bt.Unix() != 1762300000 {
		t.Fatalf("btime = %v %v", bt, ok)
	}
	if _, ok := ParseProcStat("cpu 1\n"); ok {
		t.Fatal("missing btime accepted")
	}
	if bt, ok := ParseKernBoottime("{ sec = 1762300000, usec = 123 } Wed Nov  5 01:46:40 2026\n"); !ok || bt.Unix() != 1762300000 {
		t.Fatalf("kern.boottime = %v %v", bt, ok)
	}
	on, known := ParsePmset("Now drawing from 'Battery Power'\n -InternalBattery-0 (id=1)\t85%; discharging\n")
	if !on || !known {
		t.Fatal("battery power not detected")
	}
	if on, known := ParsePmset("Now drawing from 'AC Power'\n -InternalBattery-0 (id=1)\t100%; charged\n"); on || !known {
		t.Fatal("AC power not detected")
	}
	if _, known := ParsePmset("Now drawing from 'AC Power'\n"); known {
		t.Fatal("a Mac without a battery must report unknown")
	}
}
