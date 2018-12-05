package extractor

import (
	"fmt"
	"io"
	"sync"
	"time"
)

var (
	githubArchiveUrlFormat = "http://data.githubarchive.org/%s.json.gz"
)

type GithubEventReaderSettings struct {
}

type GithubArchiveReaderFactory interface {
	NewReaderFor(dt time.Time) (io.ReadCloser, error)
}

type GithubEventExtractor struct {
	settings *GithubEventExtractorSettings
	factory  GithubArchiveReaderFactory
}

type GithubEventExtractorSettings struct {
	From      time.Time
	To        time.Time
	OrgFilter string
}

var NewExtractor = func(s *GithubEventExtractorSettings, rf GithubArchiveReaderFactory) (*GithubEventExtractor, error) {
	return &GithubEventExtractor{
		settings: s,
		factory:  rf,
	}, nil
}

func (e *GithubEventExtractor) Extract() (<-chan int, <-chan error) {
	var wg sync.WaitGroup

	jobs := make(chan int)
	results := make(chan int)
	errors := make(chan error)

	for w := 1; w <= 2; w++ {
		go worker(w, &wg, jobs, results, errors)
		wg.Add(1)
	}

	for j := 1; j <= 5; j++ {
		jobs <- j
	}
	close(jobs)

	wg.Wait()
	close(results)
	close(errors)

	return results, errors

	// reader, err := e.rf.NewReaderFor(time.Now())
	// if err != nil {
	// 	fmt.Printf("Error reading json:\n%v\n", err)
	// 	return nil, err
	// }

	// defer reader.Close()
	// dec := json.NewDecoder(reader)

	// for {
	// 	var evt ExtractedEvent
	// 	if err := dec.Decode(&evt); err == io.EOF {
	// 		break
	// 	} else if err != nil {
	// 		fmt.Printf("Error decoding json:\n%v\n", err)
	// 		return nil, err
	// 	}
	// }
}

func worker(id int, wg *sync.WaitGroup, jobs <-chan int, results chan<- int, errors chan<- error) {
	for j := range jobs {
		fmt.Println("worker", id, "started  job", j)
		time.Sleep(time.Second * 2)
		fmt.Println("worker", id, "finished job", j)
		results <- j * 2
		wg.Done()
	}
}
