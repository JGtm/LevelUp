package sync

import (
	"context"
	"errors"
	"os"
	"regexp"
	"testing"

	"levelup/go-api/internal/domain"
)

func equipe(xuid string, team int) domain.MatchParticipantRow {
	return domain.MatchParticipantRow{XUID: xuid, TeamID: &team}
}

// casAmis : participants, joueur et liste d'amis, avec le verdict attendu. Partagés avec
// TestWithFriends_MatchesSQLVerdict (integration), qui rejoue les mêmes cas contre la requête
// du recalcul.
var casAmis = []struct {
	nom          string
	participants []domain.MatchParticipantRow
	joueur       string
	amis         []string
	attendu      bool
}{
	{"ami dans la même équipe", []domain.MatchParticipantRow{equipe("1", 0), equipe("2", 0), equipe("3", 1)}, "1", []string{"2"}, true},
	{"ami dans l'équipe adverse", []domain.MatchParticipantRow{equipe("1", 0), equipe("2", 1), equipe("3", 1)}, "1", []string{"2"}, false},
	{"le joueur est dans sa propre liste d'amis", []domain.MatchParticipantRow{equipe("1", 0), equipe("3", 1)}, "1", []string{"1"}, false},
	{"le joueur est dans sa liste ET un ami coéquipier", []domain.MatchParticipantRow{equipe("1", 0), equipe("2", 0)}, "1", []string{"1", "2"}, true},
	{"bot coéquipier, jamais dans la liste", []domain.MatchParticipantRow{equipe("1", 0), equipe("bid(abc)", 0)}, "1", []string{"2"}, false},
	{"liste vide", []domain.MatchParticipantRow{equipe("1", 0), equipe("2", 0)}, "1", nil, false},
	{"ami absent du match", []domain.MatchParticipantRow{equipe("1", 0), equipe("3", 0)}, "1", []string{"2"}, false},
	{"joueur absent des participants", []domain.MatchParticipantRow{equipe("2", 0), equipe("3", 0)}, "1", []string{"2"}, false},
	{"équipe du joueur inconnue", []domain.MatchParticipantRow{{XUID: "1"}, equipe("2", 0)}, "1", []string{"2"}, false},
	{"équipe de l'ami inconnue", []domain.MatchParticipantRow{equipe("1", 0), {XUID: "2"}}, "1", []string{"2"}, false},
	{"deux équipes, ami et adversaire", []domain.MatchParticipantRow{equipe("1", 1), equipe("2", 0), equipe("4", 1)}, "1", []string{"2", "4"}, true},
}

func TestWithFriends_Cases(t *testing.T) {
	for _, c := range casAmis {
		t.Run(c.nom, func(t *testing.T) {
			if got := withFriends(c.participants, c.joueur, c.amis); got != c.attendu {
				t.Errorf("withFriends = %v, want %v", got, c.attendu)
			}
		})
	}
}

// TestBuildBatch_PosesIsWithFriendsOnFirstWrite : le premier écrit porte is_with_friends quand
// la liste d'amis est connue (TRUE ou FALSE explicite), et ne le pose pas sinon.
func TestBuildBatch_PosesIsWithFriendsOnFirstWrite(t *testing.T) {
	mk := func(amis []string) *fetchedMatch {
		fm := minimalFetchedMatch("m1")
		fm.Participants = []domain.MatchParticipantRow{equipe("1", 0), equipe("2", 0), equipe("3", 1)}
		fm.FriendXUIDs = amis
		return fm
	}
	for _, c := range []struct {
		nom  string
		amis []string
		want *bool
	}{
		{"ami coéquipier -> TRUE", []string{"2"}, boolPtrAmis(true)},
		{"ami adverse -> FALSE explicite", []string{"3"}, boolPtrAmis(false)},
		{"liste inconnue ou vide -> non posé", nil, nil},
	} {
		t.Run(c.nom, func(t *testing.T) {
			batch, err := buildBatchFromFetchedMatch(mk(c.amis), "halo_infinite", "GT", "1")
			if err != nil {
				t.Fatalf("buildBatchFromFetchedMatch: %v", err)
			}
			got := batch.PlayerData.Enrichment.IsWithFriends
			switch {
			case c.want == nil && got != nil:
				t.Errorf("IsWithFriends = %v, want non posé", *got)
			case c.want != nil && (got == nil || *got != *c.want):
				t.Errorf("IsWithFriends = %v, want %v", got, *c.want)
			}
		})
	}
}

func boolPtrAmis(v bool) *bool { return &v }

// TestApplyFriendsToBatch_V2 : le chemin V2 pose la même valeur sur un lot déjà construit.
func TestApplyFriendsToBatch_V2(t *testing.T) {
	fm := minimalFetchedMatch("m2")
	fm.Participants = []domain.MatchParticipantRow{equipe("1", 0), equipe("2", 0)}
	batch, err := buildBatchFromFetchedMatch(fm, "halo_infinite", "GT", "1")
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	ApplyFriendsToBatch(batch, "1", nil)
	if batch.PlayerData.Enrichment.IsWithFriends != nil {
		t.Fatal("liste vide: IsWithFriends ne doit pas être posé")
	}
	ApplyFriendsToBatch(batch, "1", []string{"2"})
	if v := batch.PlayerData.Enrichment.IsWithFriends; v == nil || !*v {
		t.Fatalf("IsWithFriends = %v, want TRUE", v)
	}
	ApplyFriendsToBatch(nil, "1", []string{"2"}) // nil-safe
}

// TestResolveFriendXUIDs_UnknownListLeavesFieldUnset : loader absent ou en échec -> nil (le
// premier écrit ne pose rien), sans panique ni échec du sync.
func TestResolveFriendXUIDs_UnknownListLeavesFieldUnset(t *testing.T) {
	_, sharedDB := newInMemoryDBs(t)
	ctx := context.Background()
	if got := ResolveFriendXUIDs(ctx, sharedDB, nil, "1"); got != nil {
		t.Errorf("loader nil: %v, want nil", got)
	}
	failing := func() ([]string, error) { return nil, errors.New("store illisible") }
	if got := ResolveFriendXUIDs(ctx, sharedDB, failing, "1"); got != nil {
		t.Errorf("loader en échec: %v, want nil", got)
	}
	empty := func() ([]string, error) { return nil, nil }
	if got := ResolveFriendXUIDs(ctx, sharedDB, empty, "1"); got != nil {
		t.Errorf("liste vide: %v, want nil", got)
	}
	if got := ResolveFriendXUIDs(ctx, nil, func() ([]string, error) { return []string{"Bob"}, nil }, "1"); got != nil {
		t.Errorf("shared nil: %v, want nil", got)
	}
	if _, err := sharedDB.Exec(`INSERT INTO xuid_aliases (xuid, gamertag) VALUES ('2', 'Bob')`); err != nil {
		t.Fatalf("seed alias: %v", err)
	}
	got := ResolveFriendXUIDs(ctx, sharedDB, func() ([]string, error) { return []string{"bob", "Inconnu"}, nil }, "1")
	if len(got) != 1 || got[0] != "2" {
		t.Errorf("résolution = %v, want [2] (insensible à la casse, non résolu écarté)", got)
	}
}

func minimalFetchedMatch(id string) *fetchedMatch {
	return &fetchedMatch{MatchID: id, Registry: sampleRegistry(id)}
}

// TestRunHistoryInputsCarryFriends : run() résout les amis une fois par passe et les remet à
// l'historique (`friendXUIDs: ResolveFriendXUIDs(...)` dans le littéral historyPaginationInputs).
// run() n'est pas exécutable sans moteur complet ; ce garde textuel casse si le branchement
// disparaît, auquel cas le premier écrit V1 ne poserait plus is_with_friends.
func TestRunHistoryInputsCarryFriends(t *testing.T) {
	src, err := os.ReadFile("engine.go")
	if err != nil {
		t.Fatalf("lecture engine.go: %v", err)
	}
	re := regexp.MustCompile(`(?s)historyPaginationInputs\{.*?friendXUIDs:\s*ResolveFriendXUIDs\(ctx, sharedDB, e\.friendsLoader, e\.xuid\)`)
	if !re.Match(src) {
		t.Fatal("engine.go ne remet plus ResolveFriendXUIDs(...) à historyPaginationInputs : " +
			"le premier écrit V1 ne pose plus is_with_friends")
	}
}
