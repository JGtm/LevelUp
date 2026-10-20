package grammar

// controle_corruption_du_film_test.go — LE RATCHET DU LOT 5.18.2 : LE CONTROLE DE CORRUPTION
// PAR COMPOSANT VIENT DU FILM, ET RIEN NE PEUT LE LUI REPRENDRE.
//
// Trois faits, et chacun a coute un defaut ailleurs dans ce chantier :
//
//	IL EST LU           un contexte de film rend le bit de `chunk_00 + 0x0CB45C`, pas le zero
//	                    de structure de [GrammaireBalayage].
//	IL SURVIT AU POSER  [FilmContext.PoserProfilDeBalayage] remplace le profil ENTIER, et
//	                    `replay.poserProfilPuisCarte` y installe le profil de `killsource` — ou
//	                    l INVARIANT quand le kill-feed n a pas pu se decoder. Sans la
//	                    derivation a chaque rendu, ce geste rendrait au film un drapeau qui
//	                    n est pas le sien.
//	LE REPLI SE DIT     un film sans section d identification ne declare rien, et le contexte
//	                    le NOMME ([FilmContext.ControleDeCorruptionRepli]) au lieu de servir un
//	                    faux muet.

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// bobineDeContexte charge une mini-bobine complete comme film.
func bobineDeContexte(t *testing.T, nom string) *source.Film {
	t.Helper()
	dir := filepath.Join("..", "..", "replay", "testdata", "minifilm_"+nom)
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("bobine %s absente : %v", nom, err)
	}
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("bobine %s : %v", nom, err)
	}
	return film
}

// TestControleDeCorruptionVientDuFilm : les deux constructeurs de contexte rendent le bit du
// film, et la valeur mesuree sur la bobine est celle que le lecteur de section rend.
func TestControleDeCorruptionVientDuFilm(t *testing.T) {
	film := bobineDeContexte(t, "fb1a1a72")
	reg, ok := FilmRegistryChunk(film)
	if !ok {
		t.Fatal("la bobine ne porte pas son chunk_00")
	}
	id, err := ReadFilmIdentity(reg)
	if err != nil {
		t.Fatalf("identite : %v", err)
	}
	for nom, fc := range map[string]*FilmContext{
		"NewFilmContext":       NewFilmContext(film),
		"NewFilmContextForMap": NewFilmContextForMap(film, nil, nil),
	} {
		if got := fc.ProfilDeBalayage().Grammaire.ControleDeCorruption; got != id.ControleDeCorruption {
			t.Errorf("%s : ControleDeCorruption %v, le film declare %v",
				nom, got, id.ControleDeCorruption)
		}
		if fc.ControleDeCorruptionRepli() {
			t.Errorf("%s : repli annonce alors que le film porte sa section", nom)
		}
	}
}

// TestPoserUnProfilNEffacePasLeDrapeauDuFilm : le geste de `replay.poserProfilPuisCarte`, joue
// avec l INVARIANT (le cas « kill-feed non decode »), ne reprend pas au film son drapeau.
func TestPoserUnProfilNEffacePasLeDrapeauDuFilm(t *testing.T) {
	film := bobineDeContexte(t, "fb1a1a72")
	fc := NewFilmContextForMap(film, nil, nil)
	attendu := fc.ProfilDeBalayage().Grammaire.ControleDeCorruption

	// Un profil etranger qui affirme le CONTRAIRE du film : ni un instrument ni un appelant
	// n a le droit de decider ce bit sur un contexte de film.
	etranger := ProfilDeBalayageParDefaut()
	etranger.Grammaire.ControleDeCorruption = !attendu
	fc.PoserProfilDeBalayage(etranger)
	if got := fc.ProfilDeBalayage().Grammaire.ControleDeCorruption; got != attendu {
		t.Fatalf("apres le poser : %v, le film declare %v", got, attendu)
	}
	if got := fc.CadreDeBalayage().Profil.Grammaire.ControleDeCorruption; got != attendu {
		t.Fatalf("cadre de balayage : %v, le film declare %v", got, attendu)
	}
	if got := fc.ContexteDeLecture().Profil.Grammaire.ControleDeCorruption; got != attendu {
		t.Fatalf("contexte de lecture : %v, le film declare %v", got, attendu)
	}
}

// TestControleDeCorruptionSansSectionEstUnRepliNomme : un film qui ne declare rien garde
// l invariant ET le DIT.
func TestControleDeCorruptionSansSectionEstUnRepliNomme(t *testing.T) {
	g, lue := grammaireSousFilm(grammaireDuProfil(), profile.Resoudre(profile.ClesDuFilm{}, nil))
	if lue {
		t.Error("un profil sans identite lue se declare LU")
	}
	if g.ControleDeCorruption {
		t.Error("l invariant n est pas conserve quand le film ne declare rien")
	}
	// Et un film qui declare VRAI se lit vrai — la regle ne rend pas un faux constant.
	id := profile.FilmIdentity{Build: "HI_1_13_0", ControleDeCorruption: true}
	p := profile.Resoudre(profile.ClesDuFilm{RegistrePresent: true, Identite: id, IdentiteLue: true}, nil)
	g, lue = grammaireSousFilm(grammaireDuProfil(), p)
	if !lue || !g.ControleDeCorruption {
		t.Errorf("film declarant VRAI : lue=%v, drapeau=%v", lue, g.ControleDeCorruption)
	}
}

// TestContexteNulNAffirmeRien : un contexte nul rend l invariant et ANNONCE son repli — jamais
// un « le film declare faux » qu aucun film n a dit.
func TestContexteNulNAffirmeRien(t *testing.T) {
	var fc *FilmContext
	if fc.ProfilDeBalayage().Grammaire.ControleDeCorruption {
		t.Error("contexte nul : l invariant n est pas conserve")
	}
	if !fc.ControleDeCorruptionRepli() {
		t.Error("contexte nul : le repli n est pas annonce")
	}
}
