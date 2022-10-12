package vpnconfig

import (
	"log"
	"reflect"
	"testing"
)

// TestConfigUpdateValid tests Valid of ConfigUpdate
func TestConfigUpdateValid(t *testing.T) {
	// test invalid
	u := NewUpdate()
	got := u.Valid()
	want := false
	if got != want {
		t.Errorf("got %t, want %t", got, want)
	}

	// test valid disconnect
	u = NewUpdate()
	u.Reason = "disconnect"
	u.Token = "some test token"

	got = u.Valid()
	want = true
	if got != want {
		t.Errorf("got %t, want %t", got, want)
	}

	// test valid connect
	u = NewUpdate()
	u.Reason = "connect"
	u.Token = "some test token"
	u.Config = New()

	got = u.Valid()
	want = true
	if got != want {
		t.Errorf("got %t, want %t", got, want)
	}
}

// TestConfigUpdateJSON tests JSON and UpdateFromJSON of ConfigUpdate
func TestConfigUpdateJSON(t *testing.T) {
	updates := []*ConfigUpdate{}

	// empty
	u := NewUpdate()
	updates = append(updates, u)

	// valid disconnect
	u = NewUpdate()
	u.Reason = "disconnect"
	u.Token = "some test token"
	updates = append(updates, u)

	// valid connect
	u = NewUpdate()
	u.Reason = "connect"
	u.Token = "some test token"
	u.Config = New()
	updates = append(updates, u)

	for _, u := range updates {
		log.Println(u)

		b, err := u.JSON()
		if err != nil {
			log.Fatal(err)
		}
		n, err := UpdateFromJSON(b)
		if err != nil {
			log.Fatal(err)
		}
		if !reflect.DeepEqual(u, n) {
			t.Errorf("got %v, want %v", n, u)
		}
	}
}

// TestNewUpdate tests NewUpdate
func TestNewUpdate(t *testing.T) {
	u := NewUpdate()
	if u == nil {
		t.Errorf("got nil, want != nil")
	}
}
