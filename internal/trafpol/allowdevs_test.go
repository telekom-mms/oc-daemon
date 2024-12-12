package trafpol

import (
	"slices"
	"testing"
)

// TestAllowDevsAdd tests Add of AllowDevs.
func TestAllowDevsAdd(t *testing.T) {
	a := NewAllowDevs()

	// test adding
	if !a.Add("eth3") {
		t.Error("device should be added")
	}

	// test adding again
	// should not change anything
	if a.Add("eth3") {
		t.Error("device should not be added again")
	}
}

// TestAllowDevsRemove tests Remove of AllowDevs.
func TestAllowDevsRemove(t *testing.T) {
	a := NewAllowDevs()

	// test removing device
	a.Add("eth3")
	if !a.Remove("eth3") {
		t.Error("device should be removed")
	}

	// test removing again (not existing device)
	// should not change anything
	if a.Remove("eth3") {
		t.Error("not existing device should not be removed")
	}
}

// TestAllowDevsList tests List of AllowDevs.
func TestAllowDevsList(t *testing.T) {
	a := NewAllowDevs()
	a.Add("test")

	want := []string{"test"}
	got := a.List()
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestNewAllowDevs tests NewAllowDevs.
func TestNewAllowDevs(t *testing.T) {
	a := NewAllowDevs()
	if a.m == nil {
		t.Errorf("got nil, want != nil")
	}
}
