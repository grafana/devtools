package githubstats

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestPercentile(t *testing.T) {
	Convey("Test percentile", t, func() {
		values := []float64{43, 54, 56, 61, 62, 66, 68, 69, 69, 70, 71, 72, 77, 78, 79, 85, 87, 88, 89, 93, 95, 96, 98, 99, 99}

		Convey("p90", func() {
			p := percentile(0.9, values)
			So(p, ShouldEqual, 98)
		})

		Convey("p50", func() {
			p := percentile(0.5, values)
			So(p, ShouldEqual, 77)
		})

		Convey("p20", func() {
			p := percentile(0.2, values)
			So(p, ShouldEqual, 64)
		})
	})
}
