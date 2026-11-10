package infinite_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/JGtm/halo-api/infinite"
)

// A *http.Client satisfies haloapi.Doer: the caller owns the transport, the timeout and the
// rate budget. The Spartan token comes from the caller's own authentication flow (see xblauth).
func ExampleClient_GetServiceRecord() {
	doer := &http.Client{Timeout: 20 * time.Second}
	client := infinite.New(doer, infinite.RetryProvider()) // default Halo Infinite endpoints
	auth := infinite.Auth{Spartan: "<spartan token>"}

	record, err := client.GetServiceRecord(context.Background(), auth, "xuid(2535400000000000)")

	var httpErr *infinite.HTTPError
	switch {
	case errors.As(err, &httpErr):
		// The brick returns statuses typed and does not interpret them.
		fmt.Println("service answered", httpErr.StatusCode, "for", httpErr.URL)
	case err != nil:
		fmt.Println("transport or decoding error:", err)
	default:
		fmt.Println(record.Wins, "wins in", record.MatchesCompleted, "matches")
	}
}
