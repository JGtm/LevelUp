package sync

import "testing"

func TestNewBackfillFlagSet_Defaults(t *testing.T) {
	fs, cli, scope := NewBackfillFlagSet()
	if err := fs.Parse([]string{}); err != nil {
		t.Fatal(err)
	}
	if cli.Player != "" {
		t.Fatal("expected empty player")
	}
	if cli.All {
		t.Fatal("expected all=false")
	}
	if scope.DetectionMode != "or" {
		t.Fatalf("expected or, got %s", scope.DetectionMode)
	}
	if scope.MaxMatches != 0 {
		t.Fatal("expected 0 max matches")
	}
}

func TestNewBackfillFlagSet_ParseArgs(t *testing.T) {
	fs, cli, scope := NewBackfillFlagSet()
	args := []string{
		"--player", "TestGT",
		"--medals",
		"--events",
		"--max-matches", "100",
		"--force-skill",
	}
	if err := fs.Parse(args); err != nil {
		t.Fatal(err)
	}
	if cli.Player != "TestGT" {
		t.Fatalf("expected TestGT, got %s", cli.Player)
	}
	if !scope.Medals {
		t.Fatal("expected medals")
	}
	if !scope.Events {
		t.Fatal("expected events")
	}
	if scope.MaxMatches != 100 {
		t.Fatalf("expected 100, got %d", scope.MaxMatches)
	}
	if !scope.ForceSkill {
		t.Fatal("expected force-skill")
	}
}

func TestNewBackfillFlagSet_AllFlag(t *testing.T) {
	fs, cli, _ := NewBackfillFlagSet()
	if err := fs.Parse([]string{"--all"}); err != nil {
		t.Fatal(err)
	}
	if !cli.All {
		t.Fatal("expected all=true")
	}
}

func TestNewBackfillFlagSet_DryRun(t *testing.T) {
	fs, _, scope := NewBackfillFlagSet()
	if err := fs.Parse([]string{"--dry-run"}); err != nil {
		t.Fatal(err)
	}
	if !scope.DryRun {
		t.Fatal("expected dry-run")
	}
}
