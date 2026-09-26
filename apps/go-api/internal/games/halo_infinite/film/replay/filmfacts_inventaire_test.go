package replay

// filmfacts_inventaire_test.go — L INVENTAIRE NUL ET L INVENTAIRE VIDE NE DISENT PAS LA MEME CHOSE
// (lot J3.6 du PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, constat RA1-2).
//
// `Options.Inventory == nil` veut dire « inventaire ILLISIBLE » : la couverture reste absente et le
// calque `inventory` n est pas declare (garde de `gardesDeProduction`). Une tranche VIDE veut dire
// « lu, et rien ». Le blob des entrees relit toute liste en tranche vide : sans temoin, un film a
// l inventaire illisible rejoue depuis ses faits se disait « lu, et rien ».

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
)

// relireLesFaits encode `g` en fichier de faits et le relit, sur l entree du film de reference.
func relireLesFaits(t *testing.T, g *FilmFacts) *FilmFactsFile {
	t.Helper()
	entry := goldenEntryPourTest(t)
	blob, err := EncodeFilmFactsFile(&FilmFactsFile{Coverage: *couvertureDuDecodeur(nil), Facts: *g,
		EmpreinteDeCle: EmpreinteDeCle(entry)})
	if err != nil {
		t.Fatalf("encodage : %v", err)
	}
	f, err := DecodeFilmFactsFile(blob, entry)
	if err != nil {
		t.Fatalf("relecture : %v", err)
	}
	return f
}

// TestFaits_InventaireNilResteNilEtVideResteVide : l aller-retour garde la difference.
func TestFaits_InventaireNilResteNilEtVideResteVide(t *testing.T) {
	g := loadGoldenInputs(t)
	g.Inventory = nil
	if relu := relireLesFaits(t, g).Facts.Inventory; relu != nil {
		t.Errorf("inventaire NUL (illisible) relu comme une tranche de %d element(s), non nulle : "+
			"le rejeu le dirait « lu, et rien »", len(relu))
	}
	g.Inventory = []KeyframeInventory{}
	if relu := relireLesFaits(t, g).Facts.Inventory; relu == nil || len(relu) != 0 {
		t.Errorf("inventaire VIDE (lu, rien) relu %v (nil = %v)", relu, relu == nil)
	}
}

// TestFaits_InventaireIllisiblePublieLeMemeDocumentQueLeFilm : sur une fixture SANS inventaire,
// la couverture et le calque sont les memes par la branche du film et par celle des faits.
func TestFaits_InventaireIllisiblePublieLeMemeDocumentQueLeFilm(t *testing.T) {
	entry := goldenEntryPourTest(t)
	g := loadGoldenInputs(t)
	g.Inventory = nil
	opt := g.options()
	opt.MapQuant, opt.Fallbacks = &entry, fallback.NouveauCompteur()
	direct := BuildFromPositions(goldenFilm, "halo_infinite", g.Positions, g.Fire, opt)
	rejoue := BuildFromFacts(goldenFilm, "halo_infinite", relireLesFaits(t, g), Options{MapQuant: &entry})

	_, calqueDirect := direct.Layers["inventory"]
	_, calqueRejoue := rejoue.Layers["inventory"]
	if calqueDirect || calqueRejoue {
		t.Errorf("calque `inventory` declare : film %v, faits %v — un inventaire illisible ne "+
			"produit pas le calque", calqueDirect, calqueRejoue)
	}
	if direct.Coverage.Inventory != nil || rejoue.Coverage.Inventory != nil {
		t.Errorf("couverture `inventory` publiee : film %v, faits %v — elle reste absente sur un "+
			"inventaire illisible", direct.Coverage.Inventory != nil, rejoue.Coverage.Inventory != nil)
	}
}
