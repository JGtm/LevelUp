package mappings

import (
	"path/filepath"
	"testing"
)

// TestLoadRegulationForTitle_LeChargeurDuRegistre — le titre du depot se charge par le chemin du
// registre et rend la table livree ; un titre sans fichier rend (nil, nil), comme au registre.
func TestLoadRegulationForTitle_LeChargeurDuRegistre(t *testing.T) {
	racine := filepath.Join("..", "..", "..", "..", "..")
	set, err := LoadRegulationForTitle(racine, "halo_infinite")
	if err != nil || set == nil {
		t.Fatalf("regulation.toml du depot : (%v, %v)", set, err)
	}
	if m, ok := PorteeDuRadar(set.RadarRangeMap(), "Team Slayer:Arena"); !ok || m != 18 {
		t.Fatalf("Team Slayer:Arena = (%v, %v), attendu 18 m", m, ok)
	}
	absent, err := LoadRegulationForTitle(t.TempDir(), "halo_infinite")
	if absent != nil || err != nil {
		t.Fatalf("titre sans fichier : (%v, %v), attendu (nil, nil)", absent, err)
	}
}

// TestPorteeDuRadar — la cle nettoyee, la portee connue, l'absence et la valeur non positive.
func TestPorteeDuRadar(t *testing.T) {
	table := map[string]int{"Slayer:Arena": 18, "BTB:Slayer": 24, "Casse": 0, "Negative": -3}
	for _, c := range []struct {
		variante string
		metres   float64
		connue   bool
	}{
		{"Slayer:Arena", 18, true},
		{"  Slayer:Arena\t", 18, true},
		{"BTB:Slayer", 24, true},
		{"Husky Raid:CTF", 0, false},
		{"", 0, false},
		{"Casse", 0, false},
		{"Negative", 0, false},
	} {
		metres, connue := PorteeDuRadar(table, c.variante)
		if metres != c.metres || connue != c.connue {
			t.Errorf("%q : (%v, %v), attendu (%v, %v)", c.variante, metres, connue, c.metres, c.connue)
		}
	}
	if _, connue := PorteeDuRadar(nil, "Slayer:Arena"); connue {
		t.Error("table nil : aucune portee ne peut etre connue")
	}
}
