package splitrt

import (
	"context"
	"errors"
	"net"
	"reflect"
	"testing"

	"github.com/telekom-mms/oc-daemon/internal/execs"
)

// getTestExcludes returns excludes for testing.
func getTestExcludes(t *testing.T, es []string) []*net.IPNet {
	excludes := []*net.IPNet{}
	for _, s := range es {
		_, exclude, err := net.ParseCIDR(s)
		if err != nil {
			t.Fatal(err)
		}
		excludes = append(excludes, exclude)
	}
	return excludes
}

// getTestStaticExcludes returns static excludes for testing.
func getTestStaticExcludes(t *testing.T) []*net.IPNet {
	return getTestExcludes(t, []string{
		"192.168.1.0/24",
		"2001::/64",
	})
}

// getTestStaticExcludesOverlap returns static excludes that overlap for testing.
func getTestStaticExcludesOverlap(t *testing.T) []*net.IPNet {
	return getTestExcludes(t, []string{
		"192.168.1.0/26",
		"192.168.1.64/26",
		"192.168.1.128/26",
		"192.168.1.192/26",
		"192.168.1.0/25",
		"192.168.1.128/25",
		"192.168.1.0/24",
		"2001:2001:2001:2000::/64",
		"2001:2001:2001:2001::/64",
		"2001:2001:2001:2002::/64",
		"2001:2001:2001:2003::/64",
		"2001:2001:2001:2000::/63",
		"2001:2001:2001:2002::/63",
		"2001:2001:2001:2000::/56",
	})
}

// getTestDynamicExcludes returns dynamic excludes for testing.
func getTestDynamicExcludes(t *testing.T) []*net.IPNet {
	return getTestExcludes(t, []string{
		"192.168.1.1/32",
		"2001::1/128",
		"172.16.1.1/32",
	})
}

// TestExcludesAddStatic tests AddStatic of Excludes.
func TestExcludesAddStatic(t *testing.T) {
	ctx := context.Background()
	e := NewExcludes()
	excludes := getTestStaticExcludes(t)

	// set testing runNft function
	got := []string{}
	execs.RunCmd = func(_ context.Context, _ string, s string, _ ...string) ([]byte, []byte, error) {
		got = append(got, s)
		return nil, nil, nil
	}

	// test adding excludes
	want := []string{
		"add element inet oc-daemon-routing excludes4 { 192.168.1.0/24 }",
		"add element inet oc-daemon-routing excludes6 { 2001::/64 }",
	}
	for _, exclude := range excludes {
		e.AddStatic(ctx, exclude)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test adding excludes again, should not change nft commands
	for _, exclude := range excludes {
		e.AddStatic(ctx, exclude)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test adding overlapping excludes
	e = NewExcludes()
	for _, exclude := range getTestStaticExcludesOverlap(t) {
		e.AddStatic(ctx, exclude)
	}
	for k := range e.s {
		if k != "192.168.1.0/24" && k != "2001:2001:2001:2000::/56" {
			t.Errorf("unexpected key: %s", k)
		}
	}
}

// TestExcludesAddDynamic tests AddDynamic of Excludes.
func TestExcludesAddDynamic(t *testing.T) {
	ctx := context.Background()
	e := NewExcludes()
	excludes := getTestDynamicExcludes(t)

	// set testing runNft function
	got := []string{}
	execs.RunCmd = func(_ context.Context, _ string, s string, _ ...string) ([]byte, []byte, error) {
		got = append(got, s)
		return nil, nil, nil
	}

	// test adding excludes
	want := []string{
		"add element inet oc-daemon-routing excludes4 { 192.168.1.1/32 }",
		"add element inet oc-daemon-routing excludes6 { 2001::1/128 }",
		"add element inet oc-daemon-routing excludes4 { 172.16.1.1/32 }",
	}
	for _, exclude := range excludes {
		e.AddDynamic(ctx, exclude, 300)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test adding excludes again, should not change nft commands
	for _, exclude := range excludes {
		e.AddDynamic(ctx, exclude, 300)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test adding excludes with existing static excludes,
	// should only add new excludes
	e = NewExcludes()
	for _, exclude := range getTestStaticExcludes(t) {
		e.AddStatic(ctx, exclude)
	}
	got = []string{}
	want = []string{
		"add element inet oc-daemon-routing excludes4 { 172.16.1.1/32 }",
	}
	for _, exclude := range excludes {
		e.AddDynamic(ctx, exclude, 300)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test adding invalid excludes (static as dynamic)
	e = NewExcludes()
	got = []string{}
	want = []string{}
	for _, exclude := range getTestStaticExcludes(t) {
		e.AddDynamic(ctx, exclude, 300)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestExcludesRemoveStatic tests RemoveStatic of Excludes.
func TestExcludesRemove(t *testing.T) {
	ctx := context.Background()
	e := NewExcludes()
	excludes := getTestStaticExcludes(t)

	// set testing runNft function
	got := []string{}
	oldRunCmd := execs.RunCmd
	execs.RunCmd = func(_ context.Context, _ string, s string, _ ...string) ([]byte, []byte, error) {
		got = append(got, s)
		return nil, nil, nil
	}
	defer func() { execs.RunCmd = oldRunCmd }()

	// test removing not existing excludes
	want := []string{
		"flush set inet oc-daemon-routing excludes4\n" +
			"flush set inet oc-daemon-routing excludes6\n",
		"flush set inet oc-daemon-routing excludes4\n" +
			"flush set inet oc-daemon-routing excludes6\n",
	}
	for _, exclude := range excludes {
		e.RemoveStatic(ctx, exclude)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test removing static excludes
	got = []string{}
	want = []string{
		"add element inet oc-daemon-routing excludes4 { 192.168.1.0/24 }",
		"add element inet oc-daemon-routing excludes6 { 2001::/64 }",
		"flush set inet oc-daemon-routing excludes4\n" +
			"flush set inet oc-daemon-routing excludes6\n" +
			"add element inet oc-daemon-routing excludes6 { 2001::/64 }\n",
		"flush set inet oc-daemon-routing excludes4\n" +
			"flush set inet oc-daemon-routing excludes6\n",
	}
	for _, exclude := range excludes {
		e.AddStatic(ctx, exclude)
	}
	for _, exclude := range excludes {
		e.RemoveStatic(ctx, exclude)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test with nft error
	got = []string{}
	execs.RunCmd = func(_ context.Context, _ string, s string, _ ...string) ([]byte, []byte, error) {
		got = append(got, s)
		return nil, nil, errors.New("test error")
	}
	for _, exclude := range excludes {
		e.AddStatic(ctx, exclude)
	}
	for _, exclude := range excludes {
		e.RemoveStatic(ctx, exclude)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test removing with dynamic excludes
	got = []string{}
	want = []string{
		"add element inet oc-daemon-routing excludes4 { 192.168.1.0/24 }",
		"add element inet oc-daemon-routing excludes6 { 2001::/64 }",
		"add element inet oc-daemon-routing excludes4 { 172.16.1.1/32 }",
		"flush set inet oc-daemon-routing excludes4\n" +
			"flush set inet oc-daemon-routing excludes6\n" +
			"add element inet oc-daemon-routing excludes6 { 2001::/64 }\n" +
			"add element inet oc-daemon-routing excludes4 { 172.16.1.1/32 }\n",
		"flush set inet oc-daemon-routing excludes4\n" +
			"flush set inet oc-daemon-routing excludes6\n" +
			"add element inet oc-daemon-routing excludes4 { 172.16.1.1/32 }\n",
	}
	for _, exclude := range excludes {
		e.AddStatic(ctx, exclude)
	}
	for _, exclude := range getTestDynamicExcludes(t) {
		e.AddDynamic(ctx, exclude, 300)
	}
	for _, exclude := range excludes {
		e.RemoveStatic(ctx, exclude)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestExcludesCleanup tests cleanup of Excludes.
func TestExcludesCleanup(t *testing.T) {
	ctx := context.Background()
	e := NewExcludes()

	// set testing runNft function
	got := []string{}
	execs.RunCmd = func(_ context.Context, _ string, s string, _ ...string) ([]byte, []byte, error) {
		got = append(got, s)
		return nil, nil, nil
	}

	// test without excludes
	want := []string{}
	e.cleanup(ctx)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test with dynamic excludes
	for _, exclude := range getTestDynamicExcludes(t) {
		e.AddDynamic(ctx, exclude, excludesTimer)
	}

	got = []string{}
	for i := 0; i <= excludesTimer; i += excludesTimer {
		want := []string{}
		e.cleanup(ctx)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}
	want = []string{
		"flush set inet oc-daemon-routing excludes4\n" +
			"flush set inet oc-daemon-routing excludes6\n",
	}
	e.cleanup(ctx)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test with static excludes
	for _, exclude := range getTestStaticExcludes(t) {
		e.AddStatic(ctx, exclude)
	}
	got = []string{}
	want = []string{}
	e.cleanup(ctx)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestExcludesStartStop tests Start and Stop of Excludes.
func TestExcludesStartStop(_ *testing.T) {
	e := NewExcludes()
	e.Start()
	e.Stop()
}

// TestNewExcludes tests NewExcludes.
func TestNewExcludes(t *testing.T) {
	e := NewExcludes()
	if e == nil ||
		e.s == nil ||
		e.d == nil ||
		e.done == nil ||
		e.closed == nil {

		t.Errorf("got nil, want != nil")
	}
}
