# go-fenwick-tree

A Fenwick tree (Binary Indexed Tree) in Go: point updates and
prefix/range sums in O(log n), plus a binary-lifting search for the
smallest index whose prefix sum reaches a target, and a range-add
variant for the complementary workload.

## The problem

Running totals come up everywhere — page views per day, scores per
player, events per time bucket — and the naive approach rescans the
whole array for every query. A prefix-sum array fixes queries but makes
every update O(n). A Fenwick tree gets both operations in O(log n) by
storing partial sums in a cleverly indexed array: each tree node covers
a range whose size is its index's lowest set bit, so an update touches
only O(log n) nodes and a prefix sum walks down by stripping low bits.

## Design decisions

1. **Errors, not panics, for bad indices.** A library that crashes the
   caller on an out-of-range index is hard to use in a server. Every
   fallible method returns a sentinel error (`ErrOutOfRange`,
   `ErrBadRange`, `ErrNoSuchPrefix`) that callers can test with
   `errors.Is`.
2. **O(n) construction.** `From` builds the tree in linear time by
   letting each node absorb its lowbit range, instead of n separate
   O(log n) adds. Bulk-loading a large dataset is the common case, so
   it should not pay the per-element price.
3. **One `sync.RWMutex` for the whole tree.** Reads run concurrently;
   writes exclude each other. The structure's operations are short, so
   a single lock keeps the code auditable — and the test suite runs
   under `-race` to prove it.
4. **Two shapes, one idea.** The classic tree handles point updates
   with prefix queries; `RangeAdd` flips it — range adds with point
   queries — by running the same tree over a difference array. Pick
   the shape that matches the workload instead of bending one into
   the other.
5. **Documented precondition on search.** `FindByPrefixSum` uses
   binary lifting, which assumes non-negative values (prefix sums must
   be non-decreasing). That is stated on the method, not left for the
   caller to discover.

## API

```go
f := fenwick.From([]int64{120, 340, 210, 480})

f.Add(2, 90)              // Wednesday gains 90 views
f.Set(0, 150)             // replace Monday's count
total := f.Total()        // sum of everything
midweek, _ := f.RangeSum(0, 2)
day, _ := f.FindByPrefixSum(1000) // first day cumulative views pass 1000

r, _ := fenwick.NewRangeAdd(7)
r.AddRange(3, 6, 50)      // boost Thu-Sun by 50
sat, _ := r.PointQuery(5) // 50
```

## Run the demo

```bash
go run ./cmd/fendemo
```

It walks a week of page-view counts through prefix sums, a range sum,
an update, a prefix-sum search, and the range-add variant.

## Run the tests

```bash
go test -race ./...
```

The suite covers prefix/range sums against a naive O(n) reference,
add/set behaviour, find-by-prefix-sum (including a brute-force
smallest-index check for every target), range-add edge cases, error
paths, empty trees, and concurrent use under the race detector.

## Tech stack

Go 1.23, standard library only — no dependencies.
