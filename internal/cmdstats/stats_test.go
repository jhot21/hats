package cmdstats

import (
	"testing"
	"time"
)

func TestRecord_incrementsCurrentHour(t *testing.T) {
	r := New(nil, nil)
	r.Record("motion-alert", "aaaa"+string(make([]byte, 60)), "my-token")

	stats, _ := r.Snapshot()
	if len(stats) != 1 {
		t.Fatalf("expected 1 stat, got %d", len(stats))
	}
	total := sum24h(stats[0].Hours)
	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}
}

func TestRecord_multipleTokens(t *testing.T) {
	r := New(nil, nil)
	hash1 := "a" + string(make([]byte, 63)) // 64-char hash
	hash2 := "b" + string(make([]byte, 63))
	r.Record("door-unlock", hash1, "token-a")
	r.Record("door-unlock", hash1, "token-a")
	r.Record("door-unlock", hash2, "token-b")

	stats, summary := r.Snapshot()
	if len(stats) != 2 {
		t.Fatalf("expected 2 stats, got %d", len(stats))
	}
	if summary.Endpoints != 1 {
		t.Errorf("expected 1 endpoint, got %d", summary.Endpoints)
	}
	if summary.TokensUsed != 2 {
		t.Errorf("expected 2 tokens, got %d", summary.TokensUsed)
	}
	if summary.TotalCalls != 3 {
		t.Errorf("expected 3 total calls, got %d", summary.TotalCalls)
	}
}

func TestSnapshot_sortedByCallCount(t *testing.T) {
	r := New(nil, nil)
	hash := "c" + string(make([]byte, 63))
	r.Record("quiet-cmd", hash, "t")
	r.Record("busy-cmd", hash, "t")
	r.Record("busy-cmd", hash, "t")
	r.Record("busy-cmd", hash, "t")

	stats, _ := r.Snapshot()
	if stats[0].CommandName != "busy-cmd" {
		t.Errorf("expected busy-cmd first, got %s", stats[0].CommandName)
	}
}

func TestParseHourlyKey(t *testing.T) {
	hash := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890" // 64 chars
	key := "motion-alert.2026050714." + hash
	cmd, hour, gotHash, ok := parseHourlyKey(key)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if cmd != "motion-alert" {
		t.Errorf("expected cmd=motion-alert, got %s", cmd)
	}
	if hour != "2026050714" {
		t.Errorf("expected hour=2026050714, got %s", hour)
	}
	if gotHash != hash {
		t.Errorf("hash mismatch")
	}
}

func TestParseHourlyKey_dotInCommandName(t *testing.T) {
	hash := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
	key := "my.command.name.2026050714." + hash
	cmd, _, _, ok := parseHourlyKey(key)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if cmd != "my.command.name" {
		t.Errorf("expected cmd=my.command.name, got %s", cmd)
	}
}

func TestParseMetaKey(t *testing.T) {
	hash := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
	key := "door-unlock.meta." + hash
	cmd, gotHash, ok := parseMetaKey(key)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if cmd != "door-unlock" {
		t.Errorf("expected cmd=door-unlock, got %s", cmd)
	}
	if gotHash != hash {
		t.Errorf("hash mismatch")
	}
}

func TestParseHourlyKey_badInput(t *testing.T) {
	_, _, _, ok := parseHourlyKey("not.a.valid.key")
	if ok {
		t.Error("expected ok=false for non-64-char hash segment")
	}
}

func TestRecord_lastCalled(t *testing.T) {
	r := New(nil, nil)
	hash := "d" + string(make([]byte, 63))
	before := time.Now().UTC().Add(-time.Second)
	r.Record("ping", hash, "tok")
	stats, _ := r.Snapshot()
	if stats[0].LastCalled.Before(before) {
		t.Error("LastCalled should be after test start")
	}
}
