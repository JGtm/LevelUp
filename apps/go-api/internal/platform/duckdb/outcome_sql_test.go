package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
)

type fakeOutcomeResolver struct {
	expr string
	ok   bool
}

func (f fakeOutcomeResolver) SQLEq(_, _ string, _ canonical.Outcome) (string, bool) {
	return f.expr, f.ok
}

func TestOutcomeSQLEq_FallbackRoutingDegrade(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() { games.SetDefaultOutcomeResolver(nil) })

	// 1. Aucun resolver câblé (CLI/tests) → littéral legacy byte-identique.
	games.SetDefaultOutcomeResolver(nil)
	if got := outcomeSQLEq(ctx, "outcome", canonical.OutcomeWin, "outcome = 2"); got != "outcome = 2" {
		t.Errorf("sans resolver = %q ; want le littéral legacy", got)
	}

	// 2. Resolver qui route → expression du titre.
	games.SetDefaultOutcomeResolver(fakeOutcomeResolver{expr: "outcome = 5", ok: true})
	if got := outcomeSQLEq(ctx, "outcome", canonical.OutcomeWin, "outcome = 2"); got != "outcome = 5" {
		t.Errorf("avec resolver = %q ; want outcome = 5 (routage)", got)
	}

	// 3. Resolver qui dégrade (titre sans raw_code) → littéral legacy.
	games.SetDefaultOutcomeResolver(fakeOutcomeResolver{ok: false})
	if got := outcomeSQLEq(ctx, "outcome", canonical.OutcomeLoss, "outcome = 3"); got != "outcome = 3" {
		t.Errorf("dégradation = %q ; want le littéral legacy", got)
	}
}
