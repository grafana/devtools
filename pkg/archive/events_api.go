package archive

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/grafana/github-repo-metrics/pkg/common"
)

var i = 0

func keepSearching(uri string) {
	res, err := http.Get(uri)
	if err != nil {
		log.Fatalf("failed to send request. error: %v", err)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("failed to parse body. error: %v", err)
	}

	ioutil.WriteFile(fmt.Sprintf("events/response%v.json", i), body, 777)
	i++

	model := []common.GithubEventJSON{}

	json.Unmarshal(body, &model)

	fmt.Println("event count: ", len(model))

	prev, _ := parseLinkHeaders(res.Header.Get("Link"), "next")
	fmt.Println("link: ", prev)

	if prev != nil {
		keepSearching(prev.String())
	}
}

var relPattern = regexp.MustCompile(`rel=\"(.*)\"`)
var urlPattern = regexp.MustCompile(`\<(.*)>`)

func parseLinkHeaders(headerValue string, expected string) (*url.URL, error) {

	for _, l := range strings.Split(headerValue, ",") {
		parts := strings.Split(l, ";")

		rel := relPattern.FindStringSubmatch(parts[1])

		if rel[1] == expected {
			urlMatch := urlPattern.FindStringSubmatch(parts[0])

			u, err := url.Parse(urlMatch[1])
			if err != nil {
				return nil, err
			}

			return u, nil
		}
	}

	return nil, nil
}
