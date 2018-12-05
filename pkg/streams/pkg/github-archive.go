package main

import (
	"fmt"
	"sync"
	"time"
)

// func worker(id int, jobs <-chan int, results chan<- int) {
// 	for j := range jobs {
// 		fmt.Println("worker", id, "started  job", j)
// 		time.Sleep(time.Second * 2)
// 		fmt.Println("worker", id, "finished job", j)
// 		results <- j * 2
// 	}
// }

// func source() <-chan int {
// 	numJobs := 8
// 	numWorkers := 2
// 	jobs := make(chan int, numJobs)
// 	results := make(chan int, numJobs)

// 	go func() {
// 		for w := 1; w <= numWorkers; w++ {
// 			go worker(w, jobs, results)
// 		}
// 	}()

// 	for j := 1; j <= numJobs; j++ {
// 		jobs <- j
// 	}
// 	close(jobs)

// 	return results
// }

// func transform(in <-chan int) <-chan int {
// 	numJobs := 8
// 	numWorkers := 1
// 	jobs := make(chan int, numJobs)
// 	results := make(chan int, numJobs)

// 	go func() {
// 		for w := 1; w <= numWorkers; w++ {
// 			go worker(<-in*10, jobs, results)
// 		}
// 	}()

// 	for j := 1; j <= numJobs; j++ {
// 		jobs <- j
// 	}
// 	close(jobs)

// 	return results
// }

// func sink(in <-chan int) {
// 	for i := 1; i <= 8; i++ {
// 		fmt.Printf("sink: %v\n", <-in)
// 	}
// }

// func runPipeline() {
// 	s := source()
// 	t := transform(s)
// 	sink(t)
// }

func counter(out chan<- int) {
	for x := 0; x < 10; x++ {
		out <- x
	}
	close(out)
}

func squarer(out chan<- int, in <-chan int) {
	for v := range in {
		out <- v * v
	}
	close(out)
}

func printer(in <-chan int) {
	for v := range in {
		fmt.Println(v)
	}
}

func main() {
	// runPipeline()

	jobs := make(chan int, 6)
	results := make(chan int)
	buffer := make(chan int, 2)

	var wg sync.WaitGroup
	for w := 1; w <= 2; w++ {
		go func(id int) {
			for job := range jobs {
				wg.Add(1)
				fmt.Println("worker", id, "started job", job)
				time.Sleep(time.Second * 2)
				fmt.Println("worker", w, "finished job", job)
				results <- job * 2
				wg.Done()
			}
		}(w)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var wg2 sync.WaitGroup
	wg2.Add(3)
	go func() {
	loop:
		for {
			for i := 1; i <= 2; i++ {
				v, ok := <-results
				if !ok {
					break loop
				}

				buffer <- v
			}
			wg2.Done()
		}
	}()

	go func() {
		wg2.Wait()
		close(buffer)
	}()

	for j := 1; j <= 6; j++ {
		jobs <- j
	}
	close(jobs)

	for v := range buffer {
		fmt.Println(v)
	}

	// s := &extractor.GithubEventExtractorSettings{
	// 	From: time.Date(2018, 2, 1, 0, 0, 0, 0, time.Local),
	// 	To:   time.Date(2018, 2, 28, 23, 0, 0, 0, time.Local),
	// }
	// rf, err := extractor.NewFileReader("data")
	// if err != nil {
	// 	fmt.Printf("Error creating file reader:\n%v\n", err)
	// 	return
	// }

	// e, err := extractor.NewExtractor(s, rf)
	// if err != nil {
	// 	fmt.Printf("Error creating extractor:\n%v\n", err)
	// 	return
	// }

	// e.Extract()

	// start := time.Now()
	// // url := fmt.Sprintf(githubArchiveUrlFormat, "2018-02-01-0")

	// // resp, err := http.Get(url)
	// // if err != nil {
	// // 	fmt.Printf("Error http.Get:\n%v\n", err)
	// // }

	// // defer func() { _ = resp.Body.Close() }()
	// path := fmt.Sprintf(githubArchivePathFormat, "2018-02-01-1")
	// fileReader, err := os.Open(path)

	// if err != nil {
	// 	fmt.Printf("Failed to read file:\n%v\n", err)
	// 	return
	// }
	// defer fileReader.Close()

	// reader, err := gzip.NewReader(fileReader)
	// if err != nil {
	// 	fmt.Printf("No data yet, gzip reader:\n%v\n", err)
	// 	return
	// }

	// defer func() { _ = reader.Close() }()

	// grafanaOrgEvtTypeCounter := make(map[string]int)
	// evtTypeCounter := make(map[string]int)

	// d := json.NewDecoder(reader)
	// for {
	// 	var evt Event2
	// 	if err := d.Decode(&evt); err == io.EOF {
	// 		break // done decoding file
	// 	} else if err != nil {
	// 		fmt.Printf("Error decoding json:\n%v\n", err)
	// 	}

	// 	cnt := evtTypeCounter[evt.Type]
	// 	// evtTypeCounter[evt.Type] = cnt + 1

	// 	if evt.Org.Login == "grafana" {
	// 		cnt = grafanaOrgEvtTypeCounter[evt.Type]
	// 		grafanaOrgEvtTypeCounter[evt.Type] = cnt + 1
	// 	}
	// }

	// elapsed := time.Since(start)
	// log.Printf("Done. Took %s", elapsed)

	// // for key, value := range evtTypeCounter {
	// // 	fmt.Println("Type:", key, "Value:", value)
	// // }

	// log.Println("Grafana org:")
	// for key, value := range grafanaOrgEvtTypeCounter {
	// 	fmt.Println("Type:", key, "Value:", value)
	// }
}
