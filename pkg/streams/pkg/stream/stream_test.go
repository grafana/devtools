package stream

import (
	"sync"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestStream(t *testing.T) {
	Convey("TestStream", t, func() {
		Convey("Unbuffered", func() {
			Convey("When sending 3 values at same time should be able to read all 3 values", func() {
				r, w := New()
				var res []Msg
				var wg sync.WaitGroup
				wg.Add(1)
				go func() {
					res = r.ReadAll()
					wg.Done()
				}()

				w.SendMany(1, 2, 3)
				w.Close()
				wg.Wait()

				So(res, ShouldResemble, []Msg{1, 2, 3})
			})

			Convey("When sending 3 values one at a time should be able to read 3 values one at a time", func() {
				r, w := New()
				res := []Msg{}
				var wg sync.WaitGroup
				wg.Add(1)
				go func() {
					res = append(res, r.Read())
					res = append(res, r.Read())
					res = append(res, r.Read())
					wg.Done()
				}()

				w.Send(1)
				w.Send(2)
				w.Send(3)
				w.Close()
				wg.Wait()

				So(res, ShouldResemble, []Msg{1, 2, 3})
			})
		})

		Convey("Buffered", func() {
			Convey("When sending 3 values at a time should be able to read all 3 values", func() {
				r, w := NewBuffered(3)
				w.SendMany(1, 2, 3)
				w.Close()

				So(r.ReadAll(), ShouldResemble, []Msg{1, 2, 3})
			})

			Convey("When sending 3 values one at a time should be able to read 3 values one at a time", func() {
				r, w := NewBuffered(3)
				w.Send(1)
				w.Send(2)
				w.Send(3)
				w.Close()

				So(r.Read(), ShouldEqual, 1)
				So(r.Read(), ShouldEqual, 2)
				So(r.Read(), ShouldEqual, 3)
			})
		})
	})
}
