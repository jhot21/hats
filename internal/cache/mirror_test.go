package cache

import (
	"sort"
	"testing"
	"time"
)

// newMirrorWithEntries bypasses the watcher and injects entries directly
// so we can test filtering/sorting/TTL logic without a NATS connection.
func newMirrorWithEntries(entries map[string]CacheEntry) *EntityMirror {
	return &EntityMirror{entries: entries}
}

func TestFiltered_DomainFilter(t *testing.T) {
	m := newMirrorWithEntries(map[string]CacheEntry{
		"sensor.temp": {Domain: "sensor", EntityID: "sensor.temp", State: "21.5", UpdatedAt: time.Now()},
		"switch.fan":  {Domain: "switch", EntityID: "switch.fan", State: "on", UpdatedAt: time.Now()},
	})
	got := m.Filtered("sensor", "")
	if len(got) != 1 || got[0].EntityID != "sensor.temp" {
		t.Errorf("Filtered by domain: want 1 sensor entry, got %v", got)
	}
}

func TestFiltered_QueryFilter(t *testing.T) {
	m := newMirrorWithEntries(map[string]CacheEntry{
		"sensor.temp":     {Domain: "sensor", EntityID: "sensor.temp", State: "21.5", UpdatedAt: time.Now()},
		"sensor.humidity": {Domain: "sensor", EntityID: "sensor.humidity", State: "60", UpdatedAt: time.Now()},
	})
	got := m.Filtered("", "hum")
	if len(got) != 1 || got[0].EntityID != "sensor.humidity" {
		t.Errorf("Filtered by query: want 1 humidity entry, got %v", got)
	}
}

func TestFiltered_CombinedDomainAndQuery(t *testing.T) {
	m := newMirrorWithEntries(map[string]CacheEntry{
		"sensor.temp":     {Domain: "sensor", EntityID: "sensor.temp", State: "21.5", UpdatedAt: time.Now()},
		"sensor.humidity": {Domain: "sensor", EntityID: "sensor.humidity", State: "60", UpdatedAt: time.Now()},
		"switch.humidity": {Domain: "switch", EntityID: "switch.humidity", State: "on", UpdatedAt: time.Now()},
	})
	got := m.Filtered("sensor", "hum")
	if len(got) != 1 || got[0].EntityID != "sensor.humidity" {
		t.Errorf("Filtered by domain+query: want sensor.humidity, got %v", got)
	}
}

func TestFiltered_TTLPctFresh(t *testing.T) {
	m := newMirrorWithEntries(map[string]CacheEntry{
		"sensor.temp": {Domain: "sensor", EntityID: "sensor.temp", State: "21.5", UpdatedAt: time.Now()},
	})
	got := m.Filtered("", "")
	if len(got) != 1 || got[0].TTLPct < 0.9 {
		t.Errorf("TTLPct for brand-new entry should be ~1.0, got %f", got[0].TTLPct)
	}
}

func TestFiltered_TTLPctExpired(t *testing.T) {
	m := newMirrorWithEntries(map[string]CacheEntry{
		"sensor.old": {Domain: "sensor", EntityID: "sensor.old", State: "1", UpdatedAt: time.Now().Add(-3 * time.Hour)},
	})
	got := m.Filtered("", "")
	if len(got) != 1 || got[0].TTLPct != 0 {
		t.Errorf("TTLPct for 3hr-old entry should be 0, got %f", got[0].TTLPct)
	}
}

func TestAll_SortedByUpdatedAtDesc(t *testing.T) {
	now := time.Now()
	m := newMirrorWithEntries(map[string]CacheEntry{
		"sensor.old": {Domain: "sensor", EntityID: "sensor.old", State: "1", UpdatedAt: now.Add(-time.Hour)},
		"sensor.new": {Domain: "sensor", EntityID: "sensor.new", State: "2", UpdatedAt: now},
		"sensor.mid": {Domain: "sensor", EntityID: "sensor.mid", State: "3", UpdatedAt: now.Add(-30 * time.Minute)},
	})
	got := m.All()
	if len(got) != 3 {
		t.Fatalf("want 3 entries, got %d", len(got))
	}
	if got[0].EntityID != "sensor.new" || got[1].EntityID != "sensor.mid" || got[2].EntityID != "sensor.old" {
		t.Errorf("wrong sort order: got %v %v %v", got[0].EntityID, got[1].EntityID, got[2].EntityID)
	}
}

func TestAll_EmptyMirror(t *testing.T) {
	m := newMirrorWithEntries(map[string]CacheEntry{})
	got := m.All()
	if len(got) != 0 {
		t.Errorf("want empty, got %v", got)
	}
}

func TestKeyCountByDomain(t *testing.T) {
	m := newMirrorWithEntries(map[string]CacheEntry{
		"sensor.a": {Domain: "sensor", EntityID: "sensor.a"},
		"sensor.b": {Domain: "sensor", EntityID: "sensor.b"},
		"switch.c": {Domain: "switch", EntityID: "switch.c"},
	})
	counts := m.KeyCountByDomain()
	if counts["sensor"] != 2 {
		t.Errorf("sensor count: want 2, got %d", counts["sensor"])
	}
	if counts["switch"] != 1 {
		t.Errorf("switch count: want 1, got %d", counts["switch"])
	}
}

func TestKeyCountByDomain_Empty(t *testing.T) {
	m := newMirrorWithEntries(map[string]CacheEntry{})
	counts := m.KeyCountByDomain()
	if len(counts) != 0 {
		t.Errorf("want empty map, got %v", counts)
	}
}

// Verify Filtered result is sorted (property test over domain+query combos).
func TestFiltered_AlwaysSorted(t *testing.T) {
	now := time.Now()
	m := newMirrorWithEntries(map[string]CacheEntry{
		"sensor.z": {Domain: "sensor", EntityID: "sensor.z", UpdatedAt: now.Add(-2 * time.Hour)},
		"sensor.a": {Domain: "sensor", EntityID: "sensor.a", UpdatedAt: now},
		"sensor.m": {Domain: "sensor", EntityID: "sensor.m", UpdatedAt: now.Add(-time.Hour)},
	})
	got := m.Filtered("sensor", "")
	times := make([]time.Time, len(got))
	for i, e := range got {
		times[i] = e.UpdatedAt
	}
	if !sort.SliceIsSorted(times, func(i, j int) bool { return times[i].After(times[j]) }) {
		t.Errorf("Filtered result not sorted desc by UpdatedAt: %v", times)
	}
}

func TestGet_Present(t *testing.T) {
	m := newMirrorWithEntries(map[string]CacheEntry{
		"sensor.temp": {Domain: "sensor", EntityID: "sensor.temp", State: "21.5", UpdatedAt: time.Now()},
	})
	state, ok := m.Get("sensor.temp")
	if !ok {
		t.Fatal("expected entry to be present")
	}
	if state != "21.5" {
		t.Errorf("expected state %q, got %q", "21.5", state)
	}
}

func TestGet_Absent(t *testing.T) {
	m := newMirrorWithEntries(map[string]CacheEntry{})
	_, ok := m.Get("sensor.nonexistent")
	if ok {
		t.Fatal("expected entry to be absent")
	}
}

// Regression: bucket-wide max age expires KV keys without sending the watcher a
// delete, so the mirror kept entries forever (a stale "unknown" for group.family
// was served ~17h after the KV had dropped it). Get must treat an entry older
// than the KV TTL as absent so the API falls through to Home Assistant.
func TestGet_ExpiredEntryIsAbsent(t *testing.T) {
	m := newMirrorWithEntries(map[string]CacheEntry{
		"group.family": {Domain: "group", EntityID: "group.family", State: "unknown", UpdatedAt: time.Now().Add(-3 * time.Hour)},
	})
	if state, ok := m.Get("group.family"); ok {
		t.Fatalf("expired entry must be a miss, got %q", state)
	}
}

func TestGet_EntryInsideTTLIsPresent(t *testing.T) {
	m := newMirrorWithEntries(map[string]CacheEntry{
		"sensor.temp": {Domain: "sensor", EntityID: "sensor.temp", State: "21.5", UpdatedAt: time.Now().Add(-kvTTL + time.Minute)},
	})
	if _, ok := m.Get("sensor.temp"); !ok {
		t.Fatal("entry still inside the TTL must be served from the mirror")
	}
}
