package grammar

import (
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// bobinesDesFormatsAnciens : les bobines versionnees des formats 21, 24 et 25.
var bobinesDesFormatsAnciens = map[string]bool{
	"a521164d": true, "60ae07c4": true, "11de8353": true, "111fa685": true, "e5adf7b2": true,
}

// TestDeclarationMPPSurLesBobines : les sept bobines declarent le decoupage que la mesure attend —
// 8/3 presume par mesure pour les cinq des formats anciens, 9/5 relu pour les deux du format 27 —
// sans un record discordant ; la resolution prend le format quand il porte sa largeur, la
// declaration sinon, et les deux disent la meme chose.
func TestDeclarationMPPSurLesBobines(t *testing.T) {
	for _, court := range closureMiniFilms() {
		film, err := source.LoadDir(filepath.Join("..", "..", "replay", "testdata", "minifilm_"+court), nil)
		if err != nil {
			t.Fatalf("%s : chargement %v", court, err)
		}
		fc := NewFilmContext(film)
		attendu, prov := profile.MPPParDefaut(), profile.MPPRelu
		if bobinesDesFormatsAnciens[court] {
			attendu, prov = profile.MPPWidths{Lead: 8, Index: 3}, profile.MPPPresumeParMesure
		}
		d := fc.DeclarationMPP()
		if !d.Declaree() || d.Widths != attendu || d.Provenance != prov {
			t.Errorf("%s : declaration %+v, attendu %v (%v) sans discordance", court, d, attendu, prov)
		}
		res := fc.ResolutionMPP()
		if !res.Decide() || res.Widths != attendu || res.Provenance != prov {
			t.Errorf("%s : resolution %+v, attendu %v (%v)", court, res, attendu, prov)
		}
		t.Logf("%s : %d records de la cle, decoupage %v (%v)", court, d.Records, d.Widths, d.Provenance)
	}
}

// TestUneDeclarationDiscordanteNeDecidePas : un seul record dont la taille designe un autre
// decoupage, ou aucun, et le film ne declare rien.
func TestUneDeclarationDiscordanteNeDecidePas(t *testing.T) {
	ancien := profile.MPPWidths{Lead: 8, Index: 3}
	cas := []struct {
		nom string
		d   DeclarationMPP
		ok  bool
	}{
		{"accord", DeclarationMPP{Widths: ancien, Provenance: profile.MPPPresumeParMesure, Records: 12}, true},
		{"un discordant", DeclarationMPP{Widths: ancien, Provenance: profile.MPPPresumeParMesure, Records: 12, Discordants: 1}, false},
		{"aucun record", DeclarationMPP{}, false},
		{"rien de reconnu", DeclarationMPP{Records: 3, Discordants: 3}, false},
	}
	for _, c := range cas {
		if c.d.Declaree() != c.ok {
			t.Errorf("%s : Declaree() = %v, attendu %v", c.nom, c.d.Declaree(), c.ok)
		}
	}
}
