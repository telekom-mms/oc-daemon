package splitrt

import (
	"context"
	"log"
	"net"
	"reflect"
	"testing"

	"github.com/telekom-mms/oc-daemon/internal/execs"
)

// getTestExcludes returns excludes for testing
func getTestExcludes() []*net.IPNet {
	excludes := []*net.IPNet{}
	for _, s := range []string{
		"192.168.1.0/24",
		"2001::/64",
	} {
		_, exclude, err := net.ParseCIDR(s)
		if err != nil {
			log.Fatal(err)
		}
		excludes = append(excludes, exclude)
	}
	return excludes
}

// TestExcludesAddStatic tests AddStatic of Excludes
func TestExcludesAddStatic(t *testing.T) {
	ctx := context.Background()
	e := NewExcludes()
	excludes := getTestExcludes()

	// set testing runNft function
	got := []string{}
	execs.RunCmd = func(ctx context.Context, cmd string, s string, arg ...string) error {
		got = append(got, s)
		return nil
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
}

// TestExcludesAddDynamic tests AddDynamic of Excludes
func TestExcludesAddDynamic(t *testing.T) {
	ctx := context.Background()
	e := NewExcludes()
	excludes := getTestExcludes()

	// set testing runNft function
	got := []string{}
	execs.RunCmd = func(ctx context.Context, cmd string, s string, arg ...string) error {
		got = append(got, s)
		return nil
	}

	// test adding excludes
	want := []string{
		"add element inet oc-daemon-routing excludes4 { 192.168.1.0/24 }",
		"add element inet oc-daemon-routing excludes6 { 2001::/64 }",
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
}

// TestExcludesRemove tests Remove of Excludes
func TestExcludesRemove(t *testing.T) {
	ctx := context.Background()
	e := NewExcludes()
	excludes := getTestExcludes()

	// set testing runNft function
	got := []string{}
	execs.RunCmd = func(ctx context.Context, cmd string, s string, arg ...string) error {
		got = append(got, s)
		return nil
	}

	// test removing not existing excludes
	want := []string{
		"flush set inet oc-daemon-routing excludes4\n" +
			"flush set inet oc-daemon-routing excludes6\n",
		"flush set inet oc-daemon-routing excludes4\n" +
			"flush set inet oc-daemon-routing excludes6\n",
	}
	for _, exclude := range excludes {
		e.Remove(ctx, exclude)
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
		e.Remove(ctx, exclude)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test removing dynamic excludes
	// should have same nft commands as static case
	got = []string{}
	for _, exclude := range excludes {
		e.AddDynamic(ctx, exclude, 300)
	}
	for _, exclude := range excludes {
		e.Remove(ctx, exclude)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestExcludesCleanup tests cleanup of Excludes
func TestExcludesCleanup(t *testing.T) {
	ctx := context.Background()
	e := NewExcludes()
	excludes := getTestExcludes()

	// set testing runNft function
	got := []string{}
	execs.RunCmd = func(ctx context.Context, cmd string, s string, arg ...string) error {
		got = append(got, s)
		return nil
	}

	// test without excludes
	want := []string{}
	e.cleanup(ctx)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test with dynamic excludes
	for _, exclude := range excludes {
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
	for _, exclude := range excludes {
		e.AddStatic(ctx, exclude)
	}
	got = []string{}
	want = []string{}
	e.cleanup(ctx)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestExcludesStartStop tests Start and Stop of Excludes
func TestExcludesStartStop(t *testing.T) {
	e := NewExcludes()
	e.Start()
	e.Stop()
}

// TestNewExcludes tests NewExcludes
func TestNewExcludes(t *testing.T) {
	e := NewExcludes()
	if e.m == nil ||
		e.done == nil ||
		e.closed == nil {

		t.Errorf("got nil, want != nil")
	}
}
