package profile

// equipe_bot_test.go — LA TABLE DE [PositionEquipeBotPour], FIGEE. Litteraux : un test qui relirait
// les constantes qu il verifie ne verifierait rien.

import "testing"

func TestPositionEquipeBotParBuild(t *testing.T) {
	cas := []struct {
		build  string
		pos    PositionEquipeBot
		connue bool
	}{
		{"HI_1_13_0", PositionEquipeBot{Equipe: 0x25, Jumeau: 0x26}, true},
		{"HI_1_12_0", PositionEquipeBot{Equipe: 4, Jumeau: 6}, true},
		{"HI_1_11_0", PositionEquipeBot{Equipe: 4, Jumeau: 6}, true},
		{"HI_1_10_0", PositionEquipeBot{Equipe: 4, Jumeau: 6}, true},
		// Non etablies : aucune position, jamais celle d un build voisin.
		{"HI_1_9_0", PositionEquipeBot{}, false},
		{"HI_1_8_0", PositionEquipeBot{}, false},
		{"HI_1_4_1", PositionEquipeBot{}, false},
		{"", PositionEquipeBot{}, false},
		{"HI_9_99_0", PositionEquipeBot{}, false},
	}
	for _, c := range cas {
		pos, connue := PositionEquipeBotPour(c.build)
		if pos != c.pos || connue != c.connue {
			t.Errorf("%q : %+v / %v, attendu %+v / %v", c.build, pos, connue, c.pos, c.connue)
		}
	}
}

// TestPositionEquipeBotDansLeBloc : une position etablie tombe dans le bloc de 44 octets.
func TestPositionEquipeBotDansLeBloc(t *testing.T) {
	for _, b := range buildsDeLaTable() {
		pos, connue := PositionEquipeBotPour(b)
		if !connue {
			continue
		}
		for _, o := range []int{pos.Equipe, pos.Jumeau} {
			if o < 0 || o >= 44 {
				t.Errorf("%s : octet %d hors du bloc de 44 octets", b, o)
			}
		}
	}
}
