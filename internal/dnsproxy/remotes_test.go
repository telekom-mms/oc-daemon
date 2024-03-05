package dnsproxy

import (
	"reflect"
	"testing"
)

// getTestRemotes returns remotes for testing.
func getTestRemotes() map[string][]string {
	return map[string][]string{
		".":                {"192.168.1.1:53"},
		"example.com":      {"192.168.2.21:53", "192.168.2.22:53"},
		"some.example.com": {"192.168.3.3:53"},
	}
}

// TestRemotesAdd tests Add of Remotes.
func TestRemotesAdd(t *testing.T) {
	r := NewRemotes()
	want := getTestRemotes()
	for k, v := range want {
		r.Add(k, v)
	}

	got := r.m
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestRemotesRemove tests Remove of Remotes.
func TestRemotesRemove(t *testing.T) {
	r := NewRemotes()
	remotes := getTestRemotes()
	for k, v := range remotes {
		r.Add(k, v)
	}

	for k := range remotes {
		delete(remotes, k)
		r.Remove(k)
		want := remotes
		got := r.m
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}

// TestRemotesFlush tests Flush of Remotes.
func TestRemotesFlush(t *testing.T) {
	r := NewRemotes()
	remotes := getTestRemotes()
	for k, v := range remotes {
		r.Add(k, v)
	}

	r.Flush()
	got := len(r.m)
	want := 0
	if got != want {
		t.Errorf("got %d, want %d", got, want)
	}
}

// TestRemotesGet tests Get of Remotes.
func TestRemotesGet(t *testing.T) {
	r := NewRemotes()
	remotes := getTestRemotes()
	for k, v := range remotes {
		r.Add(k, v)
	}

	// test no domain name
	got := r.Get("...not a domain name!")
	want := r.m["."]
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test domain name not in remotes
	got = r.Get("test.com")
	want = r.m["."]
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test domain names in remotes
	for k, v := range remotes {
		got := r.Get(k)
		want := v
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	// test subdomain in remotes
	got = r.Get("not.there.example.com")
	want = r.m["example.com"]
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

}

// TestNewRemotes tests NewRemotes.
func TestNewRemotes(t *testing.T) {
	r := NewRemotes()
	if r.m == nil {
		t.Errorf("got nil, want != nil")
	}
}
