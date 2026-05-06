package cache

import (
	"context"
	"sort"
	"testing"
)

func TestRecord_HitsAndMisses(t *testing.T) {
	s := New(nil)
	s.Record("sensor", true)
	s.Record("sensor", true)
	s.Record("sensor", false)
	s.Record("binary_sensor", true)

	snap := s.Snapshot()
	byDomain := map[string]DomainStat{}
	for _, d := range snap {
		byDomain[d.Domain] = d
	}

	if byDomain["sensor"].Hits != 2 {
		t.Errorf("sensor hits: got %d, want 2", byDomain["sensor"].Hits)
	}
	if byDomain["sensor"].Misses != 1 {
		t.Errorf("sensor misses: got %d, want 1", byDomain["sensor"].Misses)
	}
	if byDomain["binary_sensor"].Hits != 1 {
		t.Errorf("binary_sensor hits: got %d, want 1", byDomain["binary_sensor"].Hits)
	}
}

func TestSnapshot_SortedByDomain(t *testing.T) {
	s := New(nil)
	s.Record("timer", true)
	s.Record("sensor", true)
	s.Record("binary_sensor", true)

	snap := s.Snapshot()
	domains := make([]string, len(snap))
	for i, d := range snap {
		domains[i] = d.Domain
	}
	if !sort.StringsAreSorted(domains) {
		t.Errorf("snapshot not sorted: %v", domains)
	}
}

func TestReset_ZerosCounters(t *testing.T) {
	s := New(nil) // nil KV — Reset skips KV write when nil
	s.Record("sensor", true)
	s.Record("sensor", false)
	s.Reset(context.Background())

	snap := s.Snapshot()
	for _, d := range snap {
		if d.Hits != 0 || d.Misses != 0 {
			t.Errorf("expected zero after reset, got hits=%d misses=%d", d.Hits, d.Misses)
		}
	}
}
