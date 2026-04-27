package canonical

import (
	"reflect"
	"testing"
)

// TestPlayerMatchRow_FrozenShape garantit qu'aucun champ requis de
// PlayerMatchRow et PlayerMatchEnrichment n'est supprime ou renomme sans
// passer par la procedure d'evolution canonique.
//
// Si ce test ne compile plus apres une modification : un champ requis a ete
// supprime ou renomme. Voir docs/adr/0005-canonical-player-match-row-evolution.md
// (livre en Phase 4 meta-plan) avant d'aller plus loin.
//
// L'ajout d'un nouveau champ ne fait pas echouer ce test (politique additive).
func TestPlayerMatchRow_FrozenShape(t *testing.T) {
	t.Parallel()
	// Construction explicite avec tous les champs nommes : si un champ est
	// retire ou renomme, ce code ne compile plus.
	_ = PlayerMatchRow{
		Summary: MatchSummary{},
		Self:    MatchParticipant{},
		Enrichment: PlayerMatchEnrichment{
			SessionID:        nil,
			SessionLabel:     nil,
			PerformanceScore: nil,
			DominanceFlag:    DominanceNone,
			HadBotTeammate:   false,
			IsWithFriends:    false,
			FriendsXUIDs:     nil,
			TeamMMR:          nil,
			EnemyMMR:         nil,
		},
	}
}

// TestPlayerMatchRow_ZeroValueIsValid garantit que la valeur zero d'un
// PlayerMatchRow est utilisable (pas de panic en lecture des champs de base).
func TestPlayerMatchRow_ZeroValueIsValid(t *testing.T) {
	t.Parallel()
	var row PlayerMatchRow
	if row.Enrichment.DominanceFlag != DominanceNone {
		t.Errorf("zero value DominanceFlag should be DominanceNone (0), got %d",
			row.Enrichment.DominanceFlag)
	}
	if row.Enrichment.IsWithFriends {
		t.Error("zero value IsWithFriends should be false")
	}
	if row.Enrichment.HadBotTeammate {
		t.Error("zero value HadBotTeammate should be false")
	}
}

// TestPlayerMatchEnrichment_FieldsArePointersOrZeroable verifie que les
// champs optionnels (sessions, scores, MMR) sont bien des pointeurs pour
// distinguer "absent" de "valeur 0".
func TestPlayerMatchEnrichment_FieldsArePointersOrZeroable(t *testing.T) {
	t.Parallel()
	// Champs qui DOIVENT etre des pointeurs (distinction nil vs 0).
	pointerFields := map[string]bool{
		"SessionID":        true,
		"SessionLabel":     true,
		"PerformanceScore": true,
		"TeamMMR":          true,
		"EnemyMMR":         true,
	}
	rt := reflect.TypeOf(PlayerMatchEnrichment{})
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if !pointerFields[f.Name] {
			continue
		}
		if f.Type.Kind() != reflect.Pointer {
			t.Errorf("field %s should be a pointer (kind=%s), got %s",
				f.Name, reflect.Pointer, f.Type.Kind())
		}
	}
}

// TestHighlightEvent_BackwardCompat garantit que l'ancienne API (EventType,
// TimeMS, XUID) reste accessible apres l'extension Phase 0 meta-plan.
func TestHighlightEvent_BackwardCompat(t *testing.T) {
	t.Parallel()
	// Si ce code ne compile pas, un champ historique a ete supprime.
	_ = HighlightEvent{
		EventType: "kill",
		TimeMS:    1234,
		XUID:      "xuid-legacy",
	}
}

// TestHighlightEvent_NewFieldsPresent garantit que les champs ajoutes Phase 0
// sont bien la (additif).
func TestHighlightEvent_NewFieldsPresent(t *testing.T) {
	t.Parallel()
	xuid := "xuid-1"
	weapon := "AR"
	_ = HighlightEvent{
		EventType:  string(EventKill),
		TimeMS:     0,
		XUID:       xuid,
		MatchID:    "match-1",
		KillerXUID: &xuid,
		VictimXUID: &xuid,
		PlayerXUID: &xuid,
		WeaponID:   &weapon,
		Detail:     map[string]any{"medal_id": uint64(123)},
	}
}
