// Package fenwick implements a Fenwick tree (Binary Indexed Tree) over
// int64 values: point updates and prefix/range sums in O(log n), plus a
// binary-lifting search for the smallest index whose prefix sum reaches
// a target. A second type, RangeAdd, covers the complementary shape —
// range adds with point queries — using the same difference-array idea.
//
// All methods are safe for concurrent use. Indices are 0-based and
// inclusive; out-of-range indices return an error instead of panicking,
// because a data structure used as a library should fail loudly at the
// call site rather than crash the program.
package fenwick

import (
	"errors"
	"fmt"
	"sync"
)

var (
	// ErrOutOfRange is returned when an index or range lies outside [0, n).
	ErrOutOfRange = errors.New("fenwick: index out of range")
	// ErrBadRange is returned when a range has its lower bound above its
	// upper bound.
	ErrBadRange = errors.New("fenwick: invalid range")
	// ErrNoSuchPrefix is returned by FindByPrefixSum when no prefix sum
	// reaches the target (target <= 0, or target above the total).
	ErrNoSuchPrefix = errors.New("fenwick: no prefix sum reaches target")
)

// Fenwick is a Binary Indexed Tree. The zero value is not usable; use
// New or From.
type Fenwick struct {
	mu     sync.RWMutex
	n      int
	tree   []int64 // 1-based internal array, tree[0] unused
	values []int64 // current value at each index, so Set can be a delta
}

// New returns an empty tree of the given size. A negative size is an error.
func New(size int) (*Fenwick, error) {
	if size < 0 {
		return nil, fmt.Errorf("%w: size %d", ErrOutOfRange, size)
	}
	return &Fenwick{
		n:      size,
		tree:   make([]int64, size+1),
		values: make([]int64, size),
	}, nil
}

// From builds a tree from values in O(n) using the standard linear-time
// construction (each tree node absorbs its lowbit range), rather than n
// separate O(log n) adds.
func From(values []int64) *Fenwick {
	f := &Fenwick{
		n:      len(values),
		tree:   make([]int64, len(values)+1),
		values: make([]int64, len(values)),
	}
	copy(f.values, values)
	copy(f.tree[1:], values)
	for i := 1; i <= f.n; i++ {
		j := i + lowbit(i)
		if j <= f.n {
			f.tree[j] += f.tree[i]
		}
	}
	return f
}

// Len returns the number of elements.
func (f *Fenwick) Len() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.n
}

// Total returns the sum of all elements.
func (f *Fenwick) Total() int64 {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.prefix(f.n - 1)
}

// Add adds delta to the element at index. It returns ErrOutOfRange for a
// bad index.
func (f *Fenwick) Add(index int, delta int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.check(index); err != nil {
		return err
	}
	f.values[index] += delta
	for i := index + 1; i <= f.n; i += lowbit(i) {
		f.tree[i] += delta
	}
	return nil
}

// Set replaces the element at index with value. It is implemented as a
// delta against the stored value, so it stays O(log n).
func (f *Fenwick) Set(index int, value int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.check(index); err != nil {
		return err
	}
	delta := value - f.values[index]
	f.values[index] = value
	for i := index + 1; i <= f.n; i += lowbit(i) {
		f.tree[i] += delta
	}
	return nil
}

// Value returns the element at index.
func (f *Fenwick) Value(index int) (int64, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if err := f.check(index); err != nil {
		return 0, err
	}
	return f.values[index], nil
}

// PrefixSum returns the sum of elements [0, index].
func (f *Fenwick) PrefixSum(index int) (int64, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if err := f.check(index); err != nil {
		return 0, err
	}
	return f.prefix(index), nil
}

// RangeSum returns the sum of elements [lo, hi].
func (f *Fenwick) RangeSum(lo, hi int) (int64, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if lo > hi {
		return 0, fmt.Errorf("%w: [%d, %d]", ErrBadRange, lo, hi)
	}
	if err := f.check(lo); err != nil {
		return 0, err
	}
	if err := f.check(hi); err != nil {
		return 0, err
	}
	if lo == 0 {
		return f.prefix(hi), nil
	}
	return f.prefix(hi) - f.prefix(lo-1), nil
}

// FindByPrefixSum returns the smallest index whose prefix sum is >=
// target. All stored values must be non-negative for this to be
// meaningful (the binary lifting walk assumes prefix sums are
// non-decreasing); with negative values the result is undefined and the
// caller should use a different search. A target <= 0 or above the total
// returns ErrNoSuchPrefix.
func (f *Fenwick) FindByPrefixSum(target int64) (int, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if target <= 0 || target > f.prefix(f.n-1) {
		return 0, fmt.Errorf("%w: %d", ErrNoSuchPrefix, target)
	}
	// Binary lifting: start at the largest power of two <= n and walk
	// down, keeping the largest index whose prefix sum is still < target.
	idx := 0
	bit := highestPowerOfTwo(f.n)
	for bit > 0 {
		next := idx + bit
		if next <= f.n && f.tree[next] < target {
			target -= f.tree[next]
			idx = next
		}
		bit >>= 1
	}
	return idx, nil // idx is 0-based: prefix sums up to idx reach target
}

// prefix is the unlocked prefix-sum walk; callers must hold a lock.
func (f *Fenwick) prefix(index int) int64 {
	var sum int64
	for i := index + 1; i > 0; i -= lowbit(i) {
		sum += f.tree[i]
	}
	return sum
}

// check validates a single index against the tree size.
func (f *Fenwick) check(index int) error {
	if index < 0 || index >= f.n {
		return fmt.Errorf("%w: %d (size %d)", ErrOutOfRange, index, f.n)
	}
	return nil
}

// lowbit isolates the lowest set bit: the size of the range tree[i]
// is responsible for.
func lowbit(i int) int { return i & -i }

// highestPowerOfTwo returns the largest power of two <= n (0 when n is 0).
func highestPowerOfTwo(n int) int {
	p := 1
	for p<<1 <= n {
		p <<= 1
	}
	return p
}

// RangeAdd is the complementary structure: range adds in O(log n) with
// point queries in O(log n), built on a Fenwick tree over a difference
// array. Adding delta to [lo, hi] is two point updates on the difference
// array (at lo and at hi+1), and a point query is a prefix sum.
type RangeAdd struct {
	mu sync.RWMutex
	f  *Fenwick
}

// NewRangeAdd returns an empty range-add structure of the given size.
func NewRangeAdd(size int) (*RangeAdd, error) {
	// One extra slot so an add ending at n-1 can write its hi+1 marker.
	f, err := New(size + 1)
	if err != nil {
		return nil, err
	}
	return &RangeAdd{f: f}, nil
}

// AddRange adds delta to every element in [lo, hi].
func (r *RangeAdd) AddRange(lo, hi int, delta int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if lo > hi {
		return fmt.Errorf("%w: [%d, %d]", ErrBadRange, lo, hi)
	}
	// The underlying tree has size+1 slots; index size is the hi+1
	// marker, which is always valid here.
	if lo < 0 || hi >= r.f.n-1 {
		return fmt.Errorf("%w: [%d, %d]", ErrOutOfRange, lo, hi)
	}
	// Direct internal updates: the difference array is the tree's own
	// storage, so add straight to it rather than via the public Add,
	// which would also touch a values slice we do not use here.
	for i := lo + 1; i <= r.f.n; i += lowbit(i) {
		r.f.tree[i] += delta
	}
	for i := hi + 2; i <= r.f.n; i += lowbit(i) {
		r.f.tree[i] -= delta
	}
	return nil
}

// PointQuery returns the current value at index after all range adds.
func (r *RangeAdd) PointQuery(index int) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if index < 0 || index >= r.f.n-1 {
		return 0, fmt.Errorf("%w: %d", ErrOutOfRange, index)
	}
	return r.f.prefix(index), nil
}
