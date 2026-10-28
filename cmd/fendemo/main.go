// Command fendemo shows the Fenwick tree on a week of page-view counts:
// prefix sums, a range sum, an update, a prefix-sum search, and the
// range-add variant.
package main

import (
	"fmt"
	"log"

	"github.com/Subhanmalik138/go-fenwick-tree/fenwick"
)

func main() {
	// Daily page views, Monday to Sunday.
	views := []int64{120, 340, 210, 480, 390, 620, 275}
	f := fenwick.From(views)

	total := f.Total()
	fmt.Printf("week total: %d views\n", total)

	weekend, err := f.RangeSum(5, 6)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("weekend (Sat-Sun): %d views\n", weekend)

	if err := f.Add(2, 90); err != nil { // Wednesday gets 90 more views
		log.Fatal(err)
	}
	midweek, err := f.RangeSum(0, 2)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Mon-Wed after update: %d views\n", midweek)

	// On which day did cumulative views first pass 1000?
	day, err := f.FindByPrefixSum(1000)
	if err != nil {
		log.Fatal(err)
	}
	days := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	fmt.Printf("cumulative views first pass 1000 on %s\n", days[day])

	// Range-add variant: a promotion boosts Thu-Sun by 50 views a day.
	r, err := fenwick.NewRangeAdd(7)
	if err != nil {
		log.Fatal(err)
	}
	if err := r.AddRange(3, 6, 50); err != nil {
		log.Fatal(err)
	}
	sat, err := r.PointQuery(5)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("promotion boost on Saturday: +%d views\n", sat)
}
