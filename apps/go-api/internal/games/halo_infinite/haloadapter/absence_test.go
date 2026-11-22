package haloadapter

import (
	"errors"
	"fmt"
	"testing"

	"github.com/JGtm/halo-api/infinite"
)

func TestIsNotFound(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"404", &HTTPError{StatusCode: 404, URL: "u"}, true},
		{"410", &HTTPError{StatusCode: 410, URL: "u"}, true},
		{"404 enveloppé", fmt.Errorf("GetMatchStats: %w", &HTTPError{StatusCode: 404, URL: "u"}), true},
		{"blob 404", &infinite.BlobHTTPError{StatusCode: 404, URL: "u", Attempts: 1}, true},
		{"401", &HTTPError{StatusCode: 401, URL: "u"}, false},
		{"429", &HTTPError{StatusCode: 429, URL: "u"}, false},
		{"503", &HTTPError{StatusCode: 503, URL: "u"}, false},
		{"erreur simple", errors.New("boom"), false},
		{"texte HTTP 404", errors.New("HTTP 404: https://stats"), false},
	}
	for _, c := range cases {
		if got := IsNotFound(c.err); got != c.want {
			t.Errorf("%s: IsNotFound = %v, want %v", c.name, got, c.want)
		}
	}
}
