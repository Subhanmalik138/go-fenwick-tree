package fenwick

import (
	"errors"
	"sync"
	"testing"
)

// naivePrefix is the straightforward O(n) reference implementation the
// tree is checked against.
func naivePrefix(values []int64, index int) int64 {
	var sum int64
	for i := 0; i <= index; i++ {
		sum += values[i]
	}
	return sum
}

func TestFromAndPrefixSums(t *testing.T) {
	values := []int64{3, 1, 4, 1, 5, 9, 2, 6}
	f := From(values)
	if f.Len() != len(values) {
		t.Fatalf("Len = %d, want %d", f.Len(), len(values))
	}
	for i := range values {
		got, err := f.PrefixSum(i)
		if err != nil {
			t.Fatalf("PrefixSum(%d): %v", i, err)
		}
		if want := naivePrefix(values, i); got != want {
			t.Errorf("PrefixSum(%d) = %d, want %d", i, got, want)
		}
	}
	if got := f.Total(); got != 31 {
		t.Errorf("Total = %d, want 31", got)
	}
}

func TestAddAndSet(t *testing.T) {
	f := From([]int64{10, 20, 30, 40})
	if err := f.Add(1, 5); err != nil {
		t.Fatal(err)
	}
	if err := f.Set(2, 7); err != nil {
		t.Fatal(err)
	}
	// Values are now [10, 25, 7, 40]; prefix sums 10, 35, 42, 82.
	for i, want := range []int64{10, 35, 42, 82} {
		got, err := f.PrefixSum(i)
		if err != nil || got != want {
			t.Errorf("PrefixSum(%d) = %d, %v; want %d", i, got, err, want)
		}
	}
	v, err := f.Value(1)
	if err != nil || v != 25 {
		t.Errorf("Value(1) = %d, %v; want 25", v, err)
	}
	// A negative delta must work the same way.
	if err := f.Add(0, -10); err != nil {
		t.Fatal(err)
	}
	if got, _ := f.PrefixSum(0); got != 0 {
		t.Errorf("after Add(0, -10), PrefixSum(0) = %d, want 0", got)
	}
}

func TestRangeSum(t *testing.T) {
	f := From([]int64{5, 5, 5, 5, 5})
	tests := []struct {
		lo, hi int
		want   int64
	}{
		{0, 0, 5}, {0, 4, 25}, {1, 3, 15}, {4, 4, 5},
	}
	for _, tc := range tests {
		got, err := f.RangeSum(tc.lo, tc.hi)
		if err != nil || got != tc.want {
			t.Errorf("RangeSum(%d, %d) = %d, %v; want %d", tc.lo, tc.hi, got, err, tc.want)
		}
	}
	if _, err := f.RangeSum(3, 2); !errors.Is(err, ErrBadRange) {
		t.Errorf("RangeSum(3, 2) error = %v, want ErrBadRange", err)
	}
	if _, err := f.RangeSum(0, 5); !errors.Is(err, ErrOutOfRange) {
		t.Errorf("RangeSum(0, 5) error = %v, want ErrOutOfRange", err)
	}
	if _, err := f.RangeSum(-1, 2); !errors.Is(err, ErrOutOfRange) {
		t.Errorf("RangeSum(-1, 2) error = %v, want ErrOutOfRange", err)
	}
}

func TestErrors(t *testing.T) {
	f, err := New(-1)
	if err == nil || f != nil {
		t.Errorf("New(-1) = %v, %v; want an error", f, err)
	}
	f = From([]int64{1, 2, 3})
	for _, call := range []struct {
		name string
		fn   func() error
	}{
		{"Add(-1)", func() error { return f.Add(-1, 1) }},
		{"Add(3)", func() error { return f.Add(3, 1) }},
		{"Set(3)", func() error { return f.Set(3, 1) }},
		{"PrefixSum(3)", func() error { _, err := f.PrefixSum(3); return err }},
		{"Value(-1)", func() error { _, err := f.Value(-1); return err }},
	} {
		if err := call.fn(); !errors.Is(err, ErrOutOfRange) {
			t.Errorf("%s error = %v, want ErrOutOfRange", call.name, err)
		}
	}
}

func TestFindByPrefixSum(t *testing.T) {
	// Prefix sums: 2, 5, 6, 10.
	f := From([]int64{2, 3, 1, 4})
	tests := []struct {
		target int
		want   int
	}{
		{1, 0}, {2, 0}, {3, 1}, {5, 1}, {6, 2}, {7, 3}, {10, 3},
	}
	for _, tc := range tests {
		got, err := f.FindByPrefixSum(int64(tc.target))
		if err != nil || got != tc.want {
			t.Errorf("FindByPrefixSum(%d) = %d, %v; want %d", tc.target, got, err, tc.want)
		}
	}
	for _, target := range []int64{0, -5, 11, 100} {
		if _, err := f.FindByPrefixSum(target); !errors.Is(err, ErrNoSuchPrefix) {
			t.Errorf("FindByPrefixSum(%d) error = %v, want ErrNoSuchPrefix", target, err)
		}
	}
}

func TestFindMatchesNaive(t *testing.T) {
	values := []int64{0, 4, 0, 2, 7, 1, 3}
	f := From(values)
	total := f.Total()
	for target := int64(1); target <= total; target++ {
		got, err := f.FindByPrefixSum(target)
		if err != nil {
			t.Fatalf("FindByPrefixSum(%d): %v", target, err)
		}
		if naivePrefix(values, got) < target {
			t.Errorf("target %d: index %d has prefix %d < target", target, got, naivePrefix(values, got))
		}
		if got > 0 && naivePrefix(values, got-1) >= target {
			t.Errorf("target %d: index %d is not the smallest (prefix %d already reaches it)", target, got, naivePrefix(values, got-1))
		}
	}
}

func TestRangeAdd(t *testing.T) {
	r, err := NewRangeAdd(6)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.AddRange(1, 4, 10); err != nil {
		t.Fatal(err)
	}
	if err := r.AddRange(3, 5, -3); err != nil {
		t.Fatal(err)
	}
	// Values: [0, 10, 10, 7, 7, -3].
	for i, want := range []int64{0, 10, 10, 7, 7, -3} {
		got, err := r.PointQuery(i)
		if err != nil || got != want {
			t.Errorf("PointQuery(%d) = %d, %v; want %d", i, got, err, want)
		}
	}
	if err := r.AddRange(4, 2, 1); !errors.Is(err, ErrBadRange) {
		t.Errorf("AddRange(4, 2) error = %v, want ErrBadRange", err)
	}
	if err := r.AddRange(0, 6, 1); !errors.Is(err, ErrOutOfRange) {
		t.Errorf("AddRange(0, 6) error = %v, want ErrOutOfRange", err)
	}
	if _, err := r.PointQuery(6); !errors.Is(err, ErrOutOfRange) {
		t.Errorf("PointQuery(6) error = %v, want ErrOutOfRange", err)
	}
}

func TestConcurrentUse(t *testing.T) {
	f := From(make([]int64, 100))
	var wg sync.WaitGroup
	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				_ = f.Add((w*100+i)%100, 1)
				if _, err := f.PrefixSum(99); err != nil {
					t.Error(err)
				}
			}
		}(w)
	}
	wg.Wait()
	if got := f.Total(); got != 800 {
		t.Errorf("Total after concurrent adds = %d, want 800", got)
	}
}

func TestEmptyTree(t *testing.T) {
	f, err := New(0)
	if err != nil {
		t.Fatal(err)
	}
	if f.Len() != 0 || f.Total() != 0 {
		t.Errorf("empty tree: Len=%d Total=%d, want 0, 0", f.Len(), f.Total())
	}
	if _, err := f.FindByPrefixSum(1); !errors.Is(err, ErrNoSuchPrefix) {
		t.Errorf("empty FindByPrefixSum error = %v, want ErrNoSuchPrefix", err)
	}
}

func BenchmarkAdd(b *testing.B) {
	const n = 1 << 16
	f := From(make([]int64, n))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = f.Add(i%n, 1)
	}
}

func BenchmarkPrefixSum(b *testing.B) {
	f := From(make([]int64, 1<<16))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = f.PrefixSum((i * 7919) % (1 << 16))
	}
}
