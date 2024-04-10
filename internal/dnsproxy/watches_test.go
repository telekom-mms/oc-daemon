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

// TestWatchesAddTempCNAME tests AddTempCNAME of Watches.
func TestWatchesAddTempCNAME(t *testing.T) {
	w := NewWatches()
	defer w.Close()
	domain := "example.com."
	ttl := uint32(300)
	w.AddTempCNAME(domain, ttl)
	if !w.Contains(domain) {
		t.Errorf("got %t, want true", w.Contains(domain))
	}
}

// TestWatchesAddTempDNAME tests AddTempDNAME of Watches.
func TestWatchesAddTempDNAME(t *testing.T) {
	w := NewWatches()
	defer w.Close()
	domain := "example.com."
	ttl := uint32(300)
	w.AddTempDNAME(domain, ttl)
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

	// test temporary CNAME domain
	w.AddTempCNAME(domain, ttl)
	w.Remove(domain)
	if w.Contains(domain) {
		t.Errorf("got %t, want false", w.Contains(domain))
	}

	// test temporary DNAME domain
	w.AddTempDNAME(domain, ttl)
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

	// test temporary CNAMEs and DNAMEs with different ttls
	for _, ttl := range []uint32{0, interval - 1, interval, interval + 1, 3 * interval} {
		w.AddTempCNAME(domain, ttl)
		w.AddTempDNAME(domain, ttl)

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
	tempCNAME := "temp.cname.example.com."
	tempDNAME := "temp.dname.example.com."
	ttl := uint32(300)

	w.Add(domain)
	w.AddTempCNAME(tempCNAME, ttl)
	w.AddTempDNAME(tempDNAME, ttl)
	w.Flush()
	if w.Contains(domain) ||
		w.Contains(tempCNAME) ||
		w.Contains(tempDNAME) {

		t.Errorf("got true, want false")
	}
}

// TestWatchesContains tests Contains of Watches.
func TestWatchesContains(t *testing.T) {
	t.Run("regular watches", func(t *testing.T) {
		w := NewWatches()
		defer w.Close()
		w.Add("example.com.")

		// test invalid domain name
		invalid := "this is not a domain name..."
		if w.Contains(invalid) {
			t.Errorf("got %t, want false (domain: %s)",
				w.Contains(invalid), invalid)
		}

		// try not matching (sub)domain
		for _, domain := range []string{
			".",
			"test.com.",
			"sub.test.com.",
			"sub.sub.test.com.",
			"sub.sub.sub.test.com.",
		} {
			if w.Contains(domain) {
				t.Errorf("got %t, want false (domain: %s)",
					w.Contains(domain), domain)
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
				t.Errorf("got %t, want true (domain: %s)",
					w.Contains(domain), domain)
			}
		}
	})

	t.Run("temporary CNAMEs", func(t *testing.T) {
		w := NewWatches()
		defer w.Close()
		w.AddTempCNAME("example.com.", 500)

		// try not matching (sub)domain
		for _, domain := range []string{
			".",
			"test.com.",
			"sub.test.com.",
			"sub.sub.test.com.",
			"sub.sub.sub.test.com.",
			"sub.example.com.",
			"sub.sub.example.com.",
			"sub.sub.sub.example.com.",
		} {
			if w.Contains(domain) {
				t.Errorf("got %t, want false (domain: %s)",
					w.Contains(domain), domain)
			}
		}

		// try matching domain
		for _, domain := range []string{
			"example.com.",
		} {
			if !w.Contains(domain) {
				t.Errorf("got %t, want true (domain: %s)",
					w.Contains(domain), domain)
			}
		}
	})

	t.Run("temporary DNAMEs", func(t *testing.T) {
		w := NewWatches()
		defer w.Close()
		w.AddTempDNAME("example.com.", 500)

		// try not matching (sub)domain
		for _, domain := range []string{
			".",
			"test.com.",
			"sub.test.com.",
			"sub.sub.test.com.",
			"sub.sub.sub.test.com.",
		} {
			if w.Contains(domain) {
				t.Errorf("got %t, want false (domain: %s)",
					w.Contains(domain), domain)
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
				t.Errorf("got %t, want true (domain: %s)",
					w.Contains(domain), domain)
			}
		}
	})
}

// TestNewWatches tests NewWatches.
func TestNewWatches(t *testing.T) {
	w := NewWatches()
	if w.m == nil ||
		w.c == nil ||
		w.d == nil {

		t.Errorf("got nil, want != nil")
	}
	w.Close()
}
