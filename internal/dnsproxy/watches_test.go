package dnsproxy

import "testing"

// TestWatchesAdd tests Add of Watches.
func TestWatchesAdd(t *testing.T) {
	w := NewWatches()
	defer w.Close()
	domain := "example.com."
	w.Add(domain)
	if !w.Contains(domain) {
		t.Errorf("got %t, want true", w.Contains(domain))
	}
}

// TestWatchesAddTemp tests AddTemp of Watches.
func TestWatchesAddTemp(t *testing.T) {
	w := NewWatches()
	defer w.Close()
	domain := "example.com."
	ttl := uint32(300)
	w.AddTemp(domain, ttl)
	if !w.Contains(domain) {
		t.Errorf("got %t, want true", w.Contains(domain))
	}
}

// TestWatchesRemove tests Remove of Watches.
func TestWatchesRemove(t *testing.T) {
	w := NewWatches()
	defer w.Close()
	domain := "example.com."
	ttl := uint32(300)

	// test domain
	w.Add(domain)
	w.Remove(domain)
	if w.Contains(domain) {
		t.Errorf("got %t, want false", w.Contains(domain))
	}

	// test temporary domain
	w.AddTemp(domain, ttl)
	w.Remove(domain)
	if w.Contains(domain) {
		t.Errorf("got %t, want false", w.Contains(domain))
	}
}

// TestWatchesCleanTemp tests cleanTemp of Watches.
func TestWatchesCleanTemp(t *testing.T) {
	w := NewWatches()
	defer w.Close()
	domain := "example.com."
	interval := uint32(15)

	// test different ttls
	for _, ttl := range []uint32{0, interval - 1, interval, interval + 1, 3 * interval} {
		w.AddTemp(domain, ttl)

		// cleanups with element staying in watches
		for i := uint32(0); i <= ttl; i += interval {
			w.cleanTemp(interval)
			if !w.Contains(domain) {
				t.Errorf("got %t, want true (ttl: %d, interval: %d)", w.Contains(domain), ttl, interval)
			}
		}

		// cleanup that finally removes the element from watches
		w.cleanTemp(interval)
		if w.Contains(domain) {
			t.Errorf("got %t, want false (ttl: %d, interval: %d)", w.Contains(domain), ttl, interval)
		}
	}
}

// TestWatchesFlush tests Flush of Watches.
func TestWatchesFlush(t *testing.T) {
	w := NewWatches()
	defer w.Close()
	domain := "sub.example.com."
	tempDomain := "temp.example.com."
	ttl := uint32(300)

	w.Add(domain)
	w.AddTemp(tempDomain, ttl)
	w.Flush()
	if w.Contains(domain) ||
		w.Contains(tempDomain) {

		t.Errorf("got true, want false")
	}
}

// TestWatchesContains tests Contains of Watches.
func TestWatchesContains(t *testing.T) {
	w := NewWatches()
	defer w.Close()
	w.Add("example.com.")
	w.Add(".")

	// test invalid domain name
	invalid := "this is not a domain name!"
	if w.Contains(invalid) {
		t.Errorf("got %t, want false (domain: %s)", w.Contains(invalid), invalid)
	}

	// test root domain
	root := "."
	if !w.Contains(root) {
		t.Errorf("got %t, want true (domain: %s)", w.Contains(root), root)
	}

	// try not matching (sub)domain
	for _, domain := range []string{
		"test.com.",
		"sub.test.com.",
		"sub.sub.test.com.",
		"sub.sub.sub.test.com.",
	} {
		if w.Contains(domain) {
			t.Errorf("got %t, want false (domain: %s)", w.Contains(domain), domain)
		}
	}

	// try matching (sub)domain
	for _, domain := range []string{
		"example.com.",
		"sub.example.com.",
		"sub.sub.example.com.",
		"sub.sub.sub.example.com.",
	} {
		if !w.Contains(domain) {
			t.Errorf("got %t, want true (domain: %s)", w.Contains(domain), domain)
		}
	}
}

// TestNewWatches tests NewWatches.
func TestNewWatches(t *testing.T) {
	w := NewWatches()
	if w.m == nil ||
		w.t == nil {

		t.Errorf("got nil, want != nil")
	}
	w.Close()
}
