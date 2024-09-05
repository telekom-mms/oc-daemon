package trafpol

import (
	"context"
	"reflect"
	"slices"
	"testing"

	"github.com/telekom-mms/oc-daemon/internal/execs"
)

// TestAllowDevsAdd tests Add of AllowDevs.
func TestAllowDevsAdd(t *testing.T) {
	a := NewAllowDevs()
	ctx := context.Background()

	got := []string{}
	oldRunCmd := execs.RunCmd
	execs.RunCmd = func(_ context.Context, _ string, s string, _ ...string) ([]byte, []byte, error) {
		got = append(got, s)
		return nil, nil, nil
	}
	defer func() { execs.RunCmd = oldRunCmd }()

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

// TestAllowDevsRemove tests Remove of AllowDevs.
func TestAllowDevsRemove(t *testing.T) {
	a := NewAllowDevs()
	ctx := context.Background()

	got := []string{}
	oldRunCmd := execs.RunCmd
	execs.RunCmd = func(_ context.Context, _ string, s string, _ ...string) ([]byte, []byte, error) {
		got = append(got, s)
		return nil, nil, nil
	}
	defer func() { execs.RunCmd = oldRunCmd }()

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

// TestAllowDevsList tests List of AllowDevs.
func TestAllowDevsList(t *testing.T) {
	a := NewAllowDevs()
	ctx := context.Background()

	oldRunCmd := execs.RunCmd
	execs.RunCmd = func(_ context.Context, _, _ string, _ ...string) ([]byte, []byte, error) {
		return nil, nil, nil
	}
	defer func() { execs.RunCmd = oldRunCmd }()

	a.Add(ctx, "test")

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
