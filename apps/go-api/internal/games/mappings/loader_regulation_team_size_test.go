package mappings

import (
	"path/filepath"
	"testing"
)

// TestTeamSizeLitLeJetonDePlaylistASaPlace : le jeton se lit a gauche du « : », puis en tete de la
// droite, puis en tete du nom entier — jamais au milieu (un BTB Fiesta est un BTB).
func TestTeamSizeLitLeJetonDePlaylistASaPlace(t *testing.T) {
	raw := []byte("[meta]\ntitle_slug = \"t\"\nschema_version = 1\n\n[team_size]\n" +
		"\"Arena\" = 4\n\"BTB\" = 12\n\"BTB Heavies\" = 12\n\"Squad\" = 8\n\"Fiesta\" = 4\n")
	s, err := LoadRegulationFromBytes("t.toml", raw)
	if err != nil {
		t.Fatal(err)
	}
	cas := []struct {
		nom    string
		taille int
		connu  bool
	}{
		{"BTB:CTF", 12, true},
		{"BTB:Fiesta Slayer", 12, true},
		{"CTF:BTB Fiesta", 12, true},
		{"BTB Heavies:Total Control on Fragmentation", 12, true},
		{"BTB (2025-03)", 12, true},
		{"CTF:Arena", 4, true},
		{"Slayer:Arena Super Fiesta", 4, true},
		{"Squad:King of the Hill", 8, true},
		{"Assault:One Bomb", 0, false},
		{"Arenax:Slayer", 0, false},
		{"", 0, false},
	}
	for _, c := range cas {
		taille, connu := s.TeamSize(c.nom)
		if taille != c.taille || connu != c.connu {
			t.Errorf("TeamSize(%q) = (%d, %v), attendu (%d, %v)", c.nom, taille, connu, c.taille, c.connu)
		}
	}
	var nul *RegulationSet
	if _, ok := nul.TeamSize("BTB:CTF"); ok {
		t.Error("un RegulationSet nil ne declare aucune taille")
	}
}

// TestTeamSizeDuTitreCouvreLesTemoins : la table versionnee de Halo Infinite donne 4 aux temoins
// d'arene du corpus et 12 a ses temoins BTB (`config/replay_corpus.toml`).
func TestTeamSizeDuTitreCouvreLesTemoins(t *testing.T) {
	s, err := LoadRegulationFromFile(filepath.Join("..", "..", "..", "..", "..", "config", "titles",
		"halo_infinite", "mappings", "regulation.toml"))
	if err != nil {
		t.Fatal(err)
	}
	for nom, attendu := range map[string]int{
		"CTF:Arena": 4, "Oddball:Arena": 4, "Slayer:Arena": 4, "Slayer:Arena Super Fiesta": 4,
		"Team Slayer:Arena": 4, "Strongholds:Arena": 4, "Ranked:Oddball": 4,
		"Ranked:King of the Hill": 4, "BTB:CTF": 12, "BTB Heavies:CTF": 12,
		"BTB Heavies:Total Control": 12, "BTB:Fiesta Slayer": 12, "BTB:Slayer": 12,
		"BTB (2025-03)": 12, "Doubles:King of the Hill": 2, "Squad:King of the Hill": 8,
	} {
		if taille, ok := s.TeamSize(nom); !ok || taille != attendu {
			t.Errorf("TeamSize(%q) = (%d, %v), attendu %d", nom, taille, ok, attendu)
		}
	}
	if _, ok := s.TeamSize("Assault:One Bomb"); ok {
		t.Error("Assault n'est pas declare : sa taille n'est pas etablie")
	}
}
