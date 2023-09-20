package trafpol

import (
	"context"
	"reflect"
	"testing"
)

// TestAllowDevsAdd tests Add of AllowDevs
func TestAllowDevsAdd(t *testing.T) {
	a := NewAllowDevs()
	ctx := context.Background()

	got := []string{}
	runNft = func(ctx context.Context, s string) {
		got = append(got, s)
	}

	// test adding
	want := []string{
		"add element inet oc-daemon-filter allowdevs { eth3 }",
	}
	a.Add(ctx, "eth3")
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test adding again
	// should not change anything
	a.Add(ctx, "eth3")
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestAllowDevsRemove tests Remove of AllowDevs
func TestAllowDevsRemove(t *testing.T) {
	a := NewAllowDevs()
	ctx := context.Background()

	got := []string{}
	runNft = func(ctx context.Context, s string) {
		got = append(got, s)
	}

	// test removing device
	a.Add(ctx, "eth3")
	want := []string{
		"delete element inet oc-daemon-filter allowdevs { eth3 }",
	}
	got = []string{}
	a.Remove(ctx, "eth3")
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test removing again (not existing device)
	// should not change anything
	a.Remove(ctx, "eth3")
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestNewAllowDevs tests NewAllowDevs
func TestNewAllowDevs(t *testing.T) {
	a := NewAllowDevs()
	if a.m == nil {
		t.Errorf("got nil, want != nil")
	}
}
