package streams

import (
	"sort"
	"strconv"
	"sync"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestStreams(t *testing.T) {
	Convey("Test streams", t, func() {
		Convey("Given a readable and writable stream", func() {
			in, out := New()
			So(in, ShouldNotBeNil)
			So(in, ShouldHaveLength, 0)

			So(out, ShouldNotBeNil)
			So(out, ShouldHaveLength, 0)

			Convey("When writing to writable stream should be able to read it from readable stream", func() {
				result := writeAndReadAllMessages(ReadableCollection{in}, WritableCollection{out}, func(writer int, out Writable) {
					for n := 0; n < 10; n++ {
						out <- n
					}
				})

				So(result[0], ShouldResemble, []T{0, 1, 2, 3, 4, 5, 6, 7, 8, 9})
			})

			Convey("When filtering should return filtered result", func() {
				filtered := in.Filter(func(msg T) bool {
					return msg.(int)%2 == 0
				})

				result := writeAndReadAllMessages(ReadableCollection{filtered}, WritableCollection{out}, func(writer int, out Writable) {
					for n := 0; n < 10; n++ {
						out <- n
					}
				})

				So(result[0], ShouldResemble, []T{0, 2, 4, 6, 8})
			})

			Convey("When mapping should return mapped result", func() {
				filtered := in.Map(func(msg T) T {
					return msg.(int) * msg.(int)
				})

				result := writeAndReadAllMessages(ReadableCollection{filtered}, WritableCollection{out}, func(writer int, out Writable) {
					for n := 0; n < 10; n++ {
						out <- n
					}
				})

				So(result[0], ShouldResemble, []T{0, 1, 4, 9, 16, 25, 36, 49, 64, 81})
			})

			Convey("When reducing should return reduced result", func() {
				reduced := in.Reduce(func(accumalator T, msg T) T {
					return accumalator.(int) + msg.(int)
				}, 0)

				result := writeAndReadAllMessages(ReadableCollection{reduced}, WritableCollection{out}, func(writer int, out Writable) {
					for n := 0; n < 10; n++ {
						out <- n
					}
				})

				So(result[0], ShouldResemble, []T{0 + 1 + 2 + 3 + 4 + 5 + 6 + 7 + 8 + 9})
			})

			Convey("When filtering/mapping/reducing should return filtered/mapped/reduced result", func() {
				filtered := in.
					Filter(func(msg T) bool {
						return msg.(int)%2 == 0
					}).
					Map(func(msg T) T {
						return msg.(int) * msg.(int)
					}).
					Reduce(func(accumalator T, msg T) T {
						return accumalator.(int) + msg.(int)
					}, 0)

				result := writeAndReadAllMessages(ReadableCollection{filtered}, WritableCollection{out}, func(writer int, out Writable) {
					for n := 0; n < 10; n++ {
						out <- n
					}
				})

				So(result[0], ShouldResemble, []T{0 + (2 * 2) + (4*4 + (6 * 6) + (8 * 8))})
			})

			Convey("When grouping by should return grouped result", func() {
				type message struct {
					id    int64
					value int
				}

				grouped := in.
					GroupBy(func(msg T) ([]string, []interface{}) {
						return []string{"id"}, []interface{}{msg.(message).id}
					})

				messages := []*GroupedT{}

				var wg sync.WaitGroup
				wg.Add(1)

				go func() {
					for msg := range grouped {
						messages = append(messages, msg)
					}
					wg.Done()
				}()

				go func() {
					for n := 0; n < 10; n++ {
						if n%2 == 0 {
							out <- message{id: 2, value: n}
						} else {
							out <- message{id: 1, value: n}
						}
					}
					close(out)
				}()

				wg.Wait()

				So(messages, ShouldHaveLength, 2)
				So(messages[0].values, ShouldHaveLength, 5)
				So(messages[0].values[0].(message).value, ShouldEqual, 1)
				So(messages[0].values[1].(message).value, ShouldEqual, 3)
				So(messages[0].values[2].(message).value, ShouldEqual, 5)
				So(messages[0].values[3].(message).value, ShouldEqual, 7)
				So(messages[0].values[4].(message).value, ShouldEqual, 9)
				So(messages[1].values, ShouldHaveLength, 5)
				So(messages[1].values[0].(message).value, ShouldEqual, 0)
				So(messages[1].values[1].(message).value, ShouldEqual, 2)
				So(messages[1].values[2].(message).value, ShouldEqual, 4)
				So(messages[1].values[3].(message).value, ShouldEqual, 6)
				So(messages[1].values[4].(message).value, ShouldEqual, 8)
			})

			Convey("When grouping by and reducing should return recuced result", func() {
				type message struct {
					id    int64
					value int
				}

				grouped := in.
					GroupBy(func(msg T) ([]string, []interface{}) {
						return []string{"id"}, []interface{}{msg.(message).id}
					}).
					Reduce(func(accumalator T, msg T) T {
						key := strconv.Itoa(int(msg.(message).id))
						accDict := accumalator.(map[string]int)
						return map[string]int{key: accDict[key] + msg.(message).value}
					}, map[string]int{})

				messages := []T{}

				var wg sync.WaitGroup
				wg.Add(1)

				go func() {
					for msg := range grouped {
						messages = append(messages, msg)
					}
					wg.Done()
				}()

				go func() {
					for n := 0; n < 10; n++ {
						if n%2 == 0 {
							out <- message{id: 2, value: n}
						} else {
							out <- message{id: 1, value: n}
						}
					}
					close(out)
				}()

				wg.Wait()

				So(messages, ShouldHaveLength, 2)
				messageDictOne := messages[0].(map[string]int)
				So(messageDictOne["1"], ShouldEqual, 1+3+5+7+9)
				messageDictTwo := messages[1].(map[string]int)
				So(messageDictTwo["2"], ShouldEqual, 0+2+4+6+8)
			})

			Convey("When writing to writable stream should be able to split that into 2 readable streams and read all written data", func() {
				inputs := in.Split(2)

				result := writeAndReadAllMessages(inputs, WritableCollection{out}, func(writer int, out Writable) {
					for n := 0; n < 10; n++ {
						out <- n
					}
				})

				So(result[0], ShouldResemble, []T{0, 1, 2, 3, 4, 5, 6, 7, 8, 9})
				So(result[1], ShouldResemble, []T{0, 1, 2, 3, 4, 5, 6, 7, 8, 9})
			})
		})

		Convey("Given two readable and writable streams", func() {
			in1, out1 := New()
			So(in1, ShouldNotBeNil)
			So(in1, ShouldHaveLength, 0)
			So(out1, ShouldNotBeNil)
			So(out1, ShouldHaveLength, 0)

			in2, out2 := New()
			So(in2, ShouldNotBeNil)
			So(in2, ShouldHaveLength, 0)
			So(out2, ShouldNotBeNil)
			So(out2, ShouldHaveLength, 0)

			inputs := ReadableCollection{in1, in2}

			Convey("When writing to both writable streams should be able to combine those into one readable stream and read all written data", func() {
				combinedInput := inputs.Combine()

				result := writeAndReadAllMessages(ReadableCollection{combinedInput}, WritableCollection{out1, out2}, func(writer int, out Writable) {
					for n := writer * 10; n < (writer*10)+10; n++ {
						out <- n
					}
				})

				messages := make([]int, len(result[0]))
				for n := 0; n < len(result[0]); n++ {
					messages[n] = result[0][n].(int)
				}

				sort.Ints(messages)
				So(messages, ShouldResemble, []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19})
			})
		})
	})
}

func writeAndReadAllMessages(readers ReadableCollection, writers WritableCollection, writeFn func(writer int, out Writable)) [][]T {
	readerMessages := make([][]T, len(readers))

	var wg sync.WaitGroup
	wg.Add(len(readers))

	for n := 0; n < len(readers); n++ {
		go func(reader int, in Readable) {
			for msg := range in {
				readerMessages[reader] = append(readerMessages[reader], msg)
			}
			wg.Done()
		}(n, readers[n])
	}

	for n := 0; n < len(writers); n++ {
		go func(n int, w Writable) {
			writeFn(n, w)
			close(w)
		}(n, writers[n])
	}

	wg.Wait()

	return readerMessages
}
