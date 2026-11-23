package historyfail

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"testing"

	"github.com/JGtm/halo-api/halo5"
	"github.com/JGtm/halo-api/infinite"

	"levelup/go-api/internal/platform/auth/pool"
)

func TestIsTransient(t *testing.T) {
	timeout := &url.Error{Op: "Get", URL: "u", Err: &net.OpError{Op: "dial", Err: errors.New("connection refused")}}
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"429 Infinite", fmt.Errorf("GetMatchStats: %w", &infinite.HTTPError{StatusCode: 429}), true},
		{"408", &infinite.HTTPError{StatusCode: 408}, true},
		{"429 Halo 5", &halo5.HTTPError{StatusCode: 429}, true},
		{"500 : un match refusé en 5xx existe", fmt.Errorf("fetch: %w", &infinite.HTTPError{StatusCode: 500}), false},
		{"503 Halo 5", &halo5.HTTPError{StatusCode: 503}, false},
		{"400", &infinite.HTTPError{StatusCode: 400}, false},
		{"404", &infinite.HTTPError{StatusCode: 404}, false},
		{"réseau", fmt.Errorf("doGet: %w", timeout), true},
		{"délai", fmt.Errorf("x: %w", context.DeadlineExceeded), true},
		{"arrêt", context.Canceled, true},
		{"fin de flux", fmt.Errorf("lecture: %w", io.ErrUnexpectedEOF), true},
		{"pool sans jeton", fmt.Errorf("%w: withheld", pool.ErrNoHealthySlot), true},
		{"décodage", errors.New("invalid character 'x'"), false},
	}
	for _, c := range cases {
		if got := IsTransient(c.err); got != c.want {
			t.Errorf("%s : %v, want %v", c.name, got, c.want)
		}
	}
}

func TestWide(t *testing.T) {
	cases := []struct {
		name                   string
		failed, listed, budget int
		want                   bool
	}{
		{"5 % de 200 : durable", 10, 200, 200, false},
		{"plus de 10 % de 200", 21, 200, 200, true},
		{"exactement 10 %", 20, 200, 200, false},
		{"tout en échec", 200, 200, 200, true},
		{"un seul match en fin de liste", 1, 1, 200, false},
		{"quatre sur quatre : sous le plancher", 4, 4, 200, false},
		{"dix durables seuls en fin de liste : pas une panne", 10, 10, 200, false},
		{"vingt et un en fin de liste : mesurés au budget", 21, 21, 200, true},
		{"budget réparti de 25 : cinq échecs", 5, 25, 25, true},
		{"budget réparti de 25 : quatre échecs", 4, 25, 25, false},
	}
	for _, c := range cases {
		if got := Wide(c.failed, c.listed, c.budget); got != c.want {
			t.Errorf("%s : %v, want %v", c.name, got, c.want)
		}
	}
}

func TestCounts(t *testing.T) {
	if !Counts(false, errors.New("décodage")) || !Counts(false, nil) || !Counts(false, &infinite.HTTPError{StatusCode: 500}) {
		t.Error("un défaut du match, un match non rangé sans erreur et un 5xx isolé comptent une tentative")
	}
	if Counts(true, errors.New("décodage")) {
		t.Error("une passe de masse ne compte aucune tentative")
	}
	if Counts(false, &infinite.HTTPError{StatusCode: 429}) {
		t.Error("un 429 ne compte jamais")
	}
}
