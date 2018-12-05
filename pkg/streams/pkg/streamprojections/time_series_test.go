package streamprojections

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestTimePartitioner(t *testing.T) {
	Convey("TestTimePartitioner", t, func() {
		Convey("Daily", func() {
			p := newDailyTimeSeriesPartitioner(nil)

			Convey("Fill missing values", func() {
				Convey("Empty slice should return empty slice", func() {
					actual := p.FillMissingValues([]*timeProjectionState{})
					So(actual, ShouldHaveLength, 0)
				})

				Convey("Slice of length 1 should return slice of length 2", func() {
					actual := p.FillMissingValues([]*timeProjectionState{
						&timeProjectionState{time: time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)},
					})
					So(actual, ShouldHaveLength, 2)
					So(actual[1].time, ShouldEqual, time.Date(2018, 1, 2, 0, 0, 0, 0, time.UTC))
				})

				Convey("Slice with multiple gaps should fill gaps", func() {
					actual := p.FillMissingValues([]*timeProjectionState{
						&timeProjectionState{time: time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)},
						&timeProjectionState{time: time.Date(2018, 1, 4, 0, 0, 0, 0, time.UTC)},
						&timeProjectionState{time: time.Date(2018, 1, 5, 0, 0, 0, 0, time.UTC)},
						&timeProjectionState{time: time.Date(2018, 1, 10, 0, 0, 0, 0, time.UTC)},
					})
					So(actual, ShouldHaveLength, 10)
					So(actual[0].time, ShouldEqual, time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC))
					So(actual[1].time, ShouldEqual, time.Date(2018, 1, 2, 0, 0, 0, 0, time.UTC))
					So(actual[2].time, ShouldEqual, time.Date(2018, 1, 3, 0, 0, 0, 0, time.UTC))
					So(actual[3].time, ShouldEqual, time.Date(2018, 1, 4, 0, 0, 0, 0, time.UTC))
					So(actual[4].time, ShouldEqual, time.Date(2018, 1, 5, 0, 0, 0, 0, time.UTC))
					So(actual[5].time, ShouldEqual, time.Date(2018, 1, 6, 0, 0, 0, 0, time.UTC))
					So(actual[6].time, ShouldEqual, time.Date(2018, 1, 7, 0, 0, 0, 0, time.UTC))
					So(actual[7].time, ShouldEqual, time.Date(2018, 1, 8, 0, 0, 0, 0, time.UTC))
					So(actual[8].time, ShouldEqual, time.Date(2018, 1, 9, 0, 0, 0, 0, time.UTC))
					So(actual[9].time, ShouldEqual, time.Date(2018, 1, 10, 0, 0, 0, 0, time.UTC))
				})
			})

			getWindowSlice := func(slice []interface{}, w *WindowSlice) []interface{} {
				return slice[w.wStart:w.wEnd]
			}

			Convey("Window", func() {
				Convey("0 cur 1", func() {
					Convey("0 items should produce zero windows", func() {
						slice := []interface{}{}
						actual := p.Window(0, 1, slice)

						So(actual, ShouldHaveLength, 0)
					})

					Convey("1 item should produce zero windows", func() {
						slice := []interface{}{1}
						actual := p.Window(0, 1, slice)

						So(actual, ShouldHaveLength, 0)
					})

					Convey("3 items should produce 2 windows", func() {
						slice := []interface{}{1, 2, 3}
						actual := p.Window(0, 1, slice)

						So(actual, ShouldHaveLength, 2)
						So(actual[0].curIndex, ShouldEqual, 0)
						windowSlice := getWindowSlice(slice, actual[0])
						So(windowSlice, ShouldHaveLength, 2)
						So(windowSlice, ShouldResemble, []interface{}{1, 2})

						So(actual[1].curIndex, ShouldEqual, 1)
						windowSlice = getWindowSlice(slice, actual[1])
						So(windowSlice, ShouldHaveLength, 2)
						So(windowSlice, ShouldResemble, []interface{}{2, 3})
					})
				})

				Convey("1 cur 0", func() {
					Convey("0 items should produce zero windows", func() {
						slice := []interface{}{}
						actual := p.Window(1, 0, slice)

						So(actual, ShouldHaveLength, 0)
					})

					Convey("1 item should produce zero windows", func() {
						slice := []interface{}{1}
						actual := p.Window(1, 0, slice)

						So(actual, ShouldHaveLength, 0)
					})

					Convey("3 items should produce 2 windows", func() {
						slice := []interface{}{1, 2, 3}
						actual := p.Window(1, 0, slice)

						So(actual, ShouldHaveLength, 2)
						So(actual[0].curIndex, ShouldEqual, 1)
						windowSlice := getWindowSlice(slice, actual[0])
						So(windowSlice, ShouldHaveLength, 2)
						So(windowSlice, ShouldResemble, []interface{}{1, 2})

						So(actual[1].curIndex, ShouldEqual, 2)
						windowSlice = getWindowSlice(slice, actual[1])
						So(windowSlice, ShouldHaveLength, 2)
						So(windowSlice, ShouldResemble, []interface{}{2, 3})
					})
				})

				Convey("1 cur 1", func() {
					Convey("0 items should produce zero windows", func() {
						slice := []interface{}{}
						actual := p.Window(1, 1, slice)

						So(actual, ShouldHaveLength, 0)
					})

					Convey("1 item should produce zero windows", func() {
						slice := []interface{}{1}
						actual := p.Window(1, 1, slice)

						So(actual, ShouldHaveLength, 0)
					})

					Convey("3 items should produce 2 windows", func() {
						slice := []interface{}{1, 2, 3}
						actual := p.Window(1, 1, slice)

						So(actual, ShouldHaveLength, 1)
						So(actual[0].curIndex, ShouldEqual, 1)
						windowSlice := getWindowSlice(slice, actual[0])
						So(windowSlice, ShouldHaveLength, 3)
						So(windowSlice, ShouldResemble, []interface{}{1, 2, 3})
					})
				})

				Convey("-1 cur 0", func() {
					Convey("0 items should produce zero windows", func() {
						slice := []interface{}{}
						actual := p.Window(-1, 0, slice)

						So(actual, ShouldHaveLength, 0)
					})

					Convey("1 item should produce zero windows", func() {
						slice := []interface{}{1}
						actual := p.Window(-1, 0, slice)

						So(actual, ShouldHaveLength, 1)
					})

					Convey("3 items should produce 2 windows", func() {
						slice := []interface{}{1, 2, 3}
						actual := p.Window(-1, 0, slice)

						So(actual, ShouldHaveLength, 3)
						So(actual[0].curIndex, ShouldEqual, 0)
						windowSlice := getWindowSlice(slice, actual[0])
						So(windowSlice, ShouldHaveLength, 1)
						So(windowSlice, ShouldResemble, []interface{}{1})

						So(actual[1].curIndex, ShouldEqual, 1)
						windowSlice = getWindowSlice(slice, actual[1])
						So(windowSlice, ShouldHaveLength, 2)
						So(windowSlice, ShouldResemble, []interface{}{1, 2})

						So(actual[2].curIndex, ShouldEqual, 2)
						windowSlice = getWindowSlice(slice, actual[2])
						So(windowSlice, ShouldHaveLength, 3)
						So(windowSlice, ShouldResemble, []interface{}{1, 2, 3})
					})
				})
			})
		})

		Convey("Weekly", func() {
			p := newWeeklyTimeSeriesPartitioner(nil)

			Convey("Empty slice should return empty slice", func() {
				actual := p.FillMissingValues([]*timeProjectionState{})
				So(actual, ShouldHaveLength, 0)
			})

			Convey("Slice of length 1 should return slice of length 2", func() {
				actual := p.FillMissingValues([]*timeProjectionState{
					&timeProjectionState{time: time.Date(2018, 8, 6, 0, 0, 0, 0, time.UTC)},
				})
				So(actual, ShouldHaveLength, 2)
				So(actual[1].time, ShouldEqual, time.Date(2018, 8, 13, 0, 0, 0, 0, time.UTC))
			})

			Convey("Slice with multiple gaps should fill gaps", func() {
				actual := p.FillMissingValues([]*timeProjectionState{
					&timeProjectionState{time: time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)},
					&timeProjectionState{time: time.Date(2018, 1, 22, 0, 0, 0, 0, time.UTC)},
					&timeProjectionState{time: time.Date(2018, 1, 29, 0, 0, 0, 0, time.UTC)},
					&timeProjectionState{time: time.Date(2018, 2, 12, 0, 0, 0, 0, time.UTC)},
				})
				So(actual, ShouldHaveLength, 7)
				So(actual[0].time, ShouldEqual, time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC))
				So(actual[1].time, ShouldEqual, time.Date(2018, 1, 8, 0, 0, 0, 0, time.UTC))
				So(actual[2].time, ShouldEqual, time.Date(2018, 1, 15, 0, 0, 0, 0, time.UTC))
				So(actual[3].time, ShouldEqual, time.Date(2018, 1, 22, 0, 0, 0, 0, time.UTC))
				So(actual[4].time, ShouldEqual, time.Date(2018, 1, 29, 0, 0, 0, 0, time.UTC))
				So(actual[5].time, ShouldEqual, time.Date(2018, 2, 5, 0, 0, 0, 0, time.UTC))
				So(actual[6].time, ShouldEqual, time.Date(2018, 2, 12, 0, 0, 0, 0, time.UTC))
			})
		})

		Convey("Monthly", func() {
			p := newMonthlyTimeSeriesPartitioner(nil)

			Convey("Empty slice should return empty slice", func() {
				actual := p.FillMissingValues([]*timeProjectionState{})
				So(actual, ShouldHaveLength, 0)
			})

			Convey("Slice of length 1 should return slice of length 2", func() {
				actual := p.FillMissingValues([]*timeProjectionState{
					&timeProjectionState{time: time.Date(2018, 8, 1, 0, 0, 0, 0, time.UTC)},
				})
				So(actual, ShouldHaveLength, 2)
				So(actual[1].time, ShouldEqual, time.Date(2018, 9, 1, 0, 0, 0, 0, time.UTC))
			})

			Convey("Slice with multiple gaps should fill gaps", func() {
				actual := p.FillMissingValues([]*timeProjectionState{
					&timeProjectionState{time: time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)},
					&timeProjectionState{time: time.Date(2018, 3, 1, 0, 0, 0, 0, time.UTC)},
					&timeProjectionState{time: time.Date(2018, 4, 1, 0, 0, 0, 0, time.UTC)},
					&timeProjectionState{time: time.Date(2018, 7, 1, 0, 0, 0, 0, time.UTC)},
				})
				So(actual, ShouldHaveLength, 7)
				So(actual[0].time, ShouldEqual, time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC))
				So(actual[1].time, ShouldEqual, time.Date(2018, 2, 1, 0, 0, 0, 0, time.UTC))
				So(actual[2].time, ShouldEqual, time.Date(2018, 3, 1, 0, 0, 0, 0, time.UTC))
				So(actual[3].time, ShouldEqual, time.Date(2018, 4, 1, 0, 0, 0, 0, time.UTC))
				So(actual[4].time, ShouldEqual, time.Date(2018, 5, 1, 0, 0, 0, 0, time.UTC))
				So(actual[5].time, ShouldEqual, time.Date(2018, 6, 1, 0, 0, 0, 0, time.UTC))
				So(actual[6].time, ShouldEqual, time.Date(2018, 7, 1, 0, 0, 0, 0, time.UTC))
			})
		})

		Convey("Quarterly", func() {
			p := newQuarterlyTimeSeriesPartitioner(nil)

			Convey("Empty slice should return empty slice", func() {
				actual := p.FillMissingValues([]*timeProjectionState{})
				So(actual, ShouldHaveLength, 0)
			})

			Convey("Slice of length 1 should return slice of length 2", func() {
				actual := p.FillMissingValues([]*timeProjectionState{
					&timeProjectionState{time: time.Date(2017, 10, 1, 0, 0, 0, 0, time.UTC)},
				})
				So(actual, ShouldHaveLength, 2)
				So(actual[1].time, ShouldEqual, time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC))
			})

			Convey("Slice with multiple gaps should fill gaps", func() {
				actual := p.FillMissingValues([]*timeProjectionState{
					&timeProjectionState{time: time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC)},
					&timeProjectionState{time: time.Date(2017, 7, 1, 0, 0, 0, 0, time.UTC)},
					&timeProjectionState{time: time.Date(2017, 10, 1, 0, 0, 0, 0, time.UTC)},
					&timeProjectionState{time: time.Date(2018, 7, 1, 0, 0, 0, 0, time.UTC)},
				})
				So(actual, ShouldHaveLength, 7)
				So(actual[0].time, ShouldEqual, time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC))
				So(actual[1].time, ShouldEqual, time.Date(2017, 4, 1, 0, 0, 0, 0, time.UTC))
				So(actual[2].time, ShouldEqual, time.Date(2017, 7, 1, 0, 0, 0, 0, time.UTC))
				So(actual[3].time, ShouldEqual, time.Date(2017, 10, 1, 0, 0, 0, 0, time.UTC))
				So(actual[4].time, ShouldEqual, time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC))
				So(actual[5].time, ShouldEqual, time.Date(2018, 4, 1, 0, 0, 0, 0, time.UTC))
				So(actual[6].time, ShouldEqual, time.Date(2018, 7, 1, 0, 0, 0, 0, time.UTC))
			})
		})

		Convey("Yearly", func() {
			p := newYearlyTimeSeriesPartitioner(nil)

			Convey("Empty slice should return empty slice", func() {
				actual := p.FillMissingValues([]*timeProjectionState{})
				So(actual, ShouldHaveLength, 0)
			})

			Convey("Slice of length 1 should return slice of length 2", func() {
				actual := p.FillMissingValues([]*timeProjectionState{
					&timeProjectionState{time: time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC)},
				})
				So(actual, ShouldHaveLength, 2)
				So(actual[1].time, ShouldEqual, time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC))
			})

			Convey("Slice with multiple gaps should fill gaps", func() {
				actual := p.FillMissingValues([]*timeProjectionState{
					&timeProjectionState{time: time.Date(2012, 1, 1, 0, 0, 0, 0, time.UTC)},
					&timeProjectionState{time: time.Date(2014, 1, 1, 0, 0, 0, 0, time.UTC)},
					&timeProjectionState{time: time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)},
					&timeProjectionState{time: time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)},
				})
				So(actual, ShouldHaveLength, 7)
				So(actual[0].time, ShouldEqual, time.Date(2012, 1, 1, 0, 0, 0, 0, time.UTC))
				So(actual[1].time, ShouldEqual, time.Date(2013, 1, 1, 0, 0, 0, 0, time.UTC))
				So(actual[2].time, ShouldEqual, time.Date(2014, 1, 1, 0, 0, 0, 0, time.UTC))
				So(actual[3].time, ShouldEqual, time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC))
				So(actual[4].time, ShouldEqual, time.Date(2016, 1, 1, 0, 0, 0, 0, time.UTC))
				So(actual[5].time, ShouldEqual, time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC))
				So(actual[6].time, ShouldEqual, time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC))
			})
		})

		Convey("Hourly", func() {
			p := newHourlyTimeSeriesPartitioner(nil)

			Convey("Empty slice should return empty slice", func() {
				actual := p.FillMissingValues([]*timeProjectionState{})
				So(actual, ShouldHaveLength, 0)
			})

			Convey("Slice of length 1 should return slice of length 2", func() {
				actual := p.FillMissingValues([]*timeProjectionState{
					&timeProjectionState{time: time.Date(2018, 8, 6, 0, 0, 0, 0, time.UTC)},
				})
				So(actual, ShouldHaveLength, 2)
				So(actual[1].time, ShouldEqual, time.Date(2018, 8, 6, 1, 0, 0, 0, time.UTC))
			})

			Convey("Slice with multiple gaps should fill gaps", func() {
				actual := p.FillMissingValues([]*timeProjectionState{
					&timeProjectionState{time: time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)},
					&timeProjectionState{time: time.Date(2018, 1, 1, 23, 0, 0, 0, time.UTC)},
					&timeProjectionState{time: time.Date(2018, 1, 2, 3, 0, 0, 0, time.UTC)},
				})
				So(actual, ShouldHaveLength, 28)
				So(actual[0].time, ShouldEqual, time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC))
				So(actual[1].time, ShouldEqual, time.Date(2018, 1, 1, 1, 0, 0, 0, time.UTC))
				So(actual[24].time, ShouldEqual, time.Date(2018, 1, 2, 0, 0, 0, 0, time.UTC))
				So(actual[25].time, ShouldEqual, time.Date(2018, 1, 2, 1, 0, 0, 0, time.UTC))
				So(actual[26].time, ShouldEqual, time.Date(2018, 1, 2, 2, 0, 0, 0, time.UTC))
				So(actual[27].time, ShouldEqual, time.Date(2018, 1, 2, 3, 0, 0, 0, time.UTC))
			})
		})
	})
}
