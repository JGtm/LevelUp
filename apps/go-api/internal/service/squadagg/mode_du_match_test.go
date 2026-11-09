package squadagg

import (
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

func rowMode(matchID string, mode, variant *canonical.AssetReference) canonical.PlayerMatchRow {
	return canonical.PlayerMatchRow{Summary: canonical.MatchSummary{MatchID: matchID, PairMode: mode, GameVariant: variant}}
}

func refMode(kind, id, en, fr string) *canonical.AssetReference {
	return &canonical.AssetReference{Kind: kind, ID: id, DefaultLabel: en, Labels: map[string]string{"fr": fr}}
}

// Le filtre de modes compare le mode fourni par la source du titre (PairMode), qu'il soit une partie
// (Infinite) ou un type de jeu (Halo 5) ; le type de jeu exposé à part n'y entre jamais.
func TestFilterRowsByCascade_ModesLitLeModeDeLaSource(t *testing.T) {
	rows := []canonical.PlayerMatchRow{
		rowMode("partie", refMode("pair_mode", "p1", "Slayer", "Assassin"), refMode("game_variant", "g1", "Slayer:Arena", "Assassin:Arène")),
		rowMode("type-de-jeu", refMode("game_variant", "g2", "Capture the Flag", "Capture du drapeau"), refMode("game_variant", "g2", "Capture the Flag", "Capture du drapeau")),
		rowMode("sans-mode", nil, refMode("game_variant", "g3", "Strongholds", "Bases")),
	}
	cas := []struct {
		modes []string
		want  []string
	}{
		{[]string{"Assassin"}, []string{"partie"}},
		{[]string{"Capture du drapeau"}, []string{"type-de-jeu"}},
		// Un match sans mode ne ressort pas sous le nom de son type de jeu.
		{[]string{"Bases"}, nil},
	}
	for _, c := range cas {
		got := idsDesLignes(FilterRowsByCascade(rows, domain.CascadeFilter{Modes: c.modes}, domain.GroupesDeCartes{}))
		if len(got) != len(c.want) || (len(got) > 0 && got[0] != c.want[0]) {
			t.Errorf("modes %v : got %v, want %v", c.modes, got, c.want)
		}
	}
}

// Le mode d'un match partagé est celui de la source ; sans mode, il reste vide (pas de type de jeu).
func TestBuildSharedMatch_ModeDeLaSource(t *testing.T) {
	mode := refMode("game_variant", "g2", "Slayer", "Assassin")
	sm := buildSharedMatch("m1", map[string]map[string]canonical.PlayerMatchRow{
		"A": {"m1": rowMode("m1", mode, refMode("game_variant", "g2", "Slayer", "Assassin"))},
	})
	if sm.Mode == nil || sm.Mode.ID != "g2" {
		t.Errorf("Mode = %+v, want le mode de la source", sm.Mode)
	}
	sm = buildSharedMatch("m2", map[string]map[string]canonical.PlayerMatchRow{
		"A": {"m2": rowMode("m2", nil, refMode("game_variant", "g3", "Strongholds", "Bases"))},
	})
	if sm.Mode != nil {
		t.Errorf("Mode = %+v, want nil (aucun repli sur le type de jeu)", sm.Mode)
	}
}
