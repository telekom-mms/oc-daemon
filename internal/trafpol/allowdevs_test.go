package trafpol

import (
	"context"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/telekom-mms/oc-daemon/internal/config"
	"github.com/telekom-mms/oc-daemon/internal/execs"
)

// TestAllowDevsAdd tests Add of AllowDevs.
func TestAllowDevsAdd(t *testing.T) {
	a := NewAllowDevs()
	ctx := context.Background()

	got := []string{}
	oldRunCmd := execs.RunCmd
	execs.RunCmd = func(_ context.Context, cmd string, _ string, args ...string) ([]byte, []byte, error) {
		got = append(got, cmd+" "+strings.Join(args, " "))
		return nil, nil, nil
	}
	defer func() { execs.RunCmd = oldRunCmd }()

	// test adding
	want := []string{
		"nft -f - add element inet oc-daemon-filter allowdevs { eth3 }",
	}
	if a.Add("eth3") {
		// TODO: only check bool, and add new test for addAllowedDevice()
		addAllowedDevice(ctx, config.NewConfig(), "eth3")
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test adding again
	// should not change anything
	if a.Add("eth3") {
		// TODO: only check bool, and add new test for addAllowedDevice()
		addAllowedDevice(ctx, config.NewConfig(), "eth3")
	}
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
	execs.RunCmd = func(_ context.Context, cmd string, _ string, args ...string) ([]byte, []byte, error) {
		got = append(got, cmd+" "+strings.Join(args, " "))
		return nil, nil, nil
	}
	defer func() { execs.RunCmd = oldRunCmd }()

	// test removing device
	a.Add("eth3")
	want := []string{
		"nft -f - delete element inet oc-daemon-filter allowdevs { eth3 }",
	}
	got = []string{}
	if a.Remove("eth3") {
		// TODO: only check bool, and add new test for removeAllowedDevice()
		removeAllowedDevice(ctx, config.NewConfig(), "eth3")
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test removing again (not existing device)
	// should not change anything
	if a.Remove("eth3") {
		// TODO: only check bool, and add new test for removeAllowedDevice()
		removeAllowedDevice(ctx, config.NewConfig(), "eth3")
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestAllowDevsList tests List of AllowDevs.
func TestAllowDevsList(t *testing.T) {
	a := NewAllowDevs()

	oldRunCmd := execs.RunCmd
	execs.RunCmd = func(_ context.Context, _, _ string, _ ...string) ([]byte, []byte, error) {
		return nil, nil, nil
	}
	defer func() { execs.RunCmd = oldRunCmd }()

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
