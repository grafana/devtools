package archive

import (
	"testing"
)

func TestRelRegexp(t *testing.T) {
	testcases := []struct {
		input    string
		expected string
	}{
		{` rel="next"`, "next"},
		{` rel="last"`, "last"},
		{` rel="prev"`, "prev"},
		{` rel="first"`, "first"},
	}

	for _, tc := range testcases {
		have := relPattern.FindStringSubmatch(tc.input)

		if have[1] != tc.expected {
			t.Fatalf("expected %s got %s", tc.expected, have[1])
		}
	}
}

func TestUrlRegexp(t *testing.T) {
	testcases := []struct {
		input    string
		expected string
	}{
		{
			input:    `<https://api.github.com/repositories/15111821/events?page=2>`,
			expected: "https://api.github.com/repositories/15111821/events?page=2",
		},
	}

	for _, tc := range testcases {
		have := urlPattern.FindStringSubmatch(tc.input)
		if have[1] != tc.expected {
			t.Fatalf("expected %s got %s", tc.expected, have[1])
		}
	}
}

func TestParsingLinks(t *testing.T) {
	testCases := []struct {
		header string
		rels   []string
	}{
		{
			header: `<https://api.github.com/repositories/15111821/events?page=2>; rel="next", <https://api.github.com/repositories/15111821/events?page=10>; rel="last"`,
			rels:   []string{"next", "last"},
		},
	}

	for _, tc := range testCases {
		var err error
		for _, rel := range tc.rels {
			_, err = parseLinkHeaders(tc.header, rel)
			if err != nil {
				t.Fatalf("expected err to be nil")
			}

			_, err = parseLinkHeaders(tc.header, rel)
			if err != nil {
				t.Fatalf("expected err to be nil")
			}
		}
	}
}
