package vpnstatus

import (
	"log"
	"reflect"
	"testing"
)

// TestStatusCopy tests Copy of Status
func TestStatusCopy(t *testing.T) {
	want := New()
	got := want.Copy()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestJSON tests JSON and NewFromJSON of Status
func TestJSON(t *testing.T) {
	s := New()
	b, err := s.JSON()
	if err != nil {
		log.Fatal(err)
	}
	n, err := NewFromJSON(b)
	if err != nil {
		log.Fatal(err)
	}
	if !reflect.DeepEqual(n, s) {
		t.Errorf("got %v, want %v", n, s)
	}
}

// TestNew tests New
func TestNew(t *testing.T) {
	s := New()
	if s == nil {
		t.Errorf("got nil, want != nil")
	}
}
