package killsource

// decoupage_mpp_test.go — LE CONTEXTE DE LA MARCHE PORTE LE DECOUPAGE MPP QUE LA GRAMMAIRE RESOUT POUR
// LE FILM, COMME CELUI DE LA CUISSON ; UN DECOUPAGE NON RESOLU SE DIT.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/constat"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// TestLeContexteDeLaMarchePrendLeDecoupageDeclare : sur la bobine `e5adf7b2` (HI_1_11_0, format 25,
// sans largeur relue au format), le contexte de la marche porte le decoupage que le film declare
// (8/3, presume par mesure), pas celui du profil (9/5). MUTATION — la resolution non posee : ROUGE.
func TestLeContexteDeLaMarchePrendLeDecoupageDeclare(t *testing.T) {
	fc := grammar.NewFilmContextForMap(chargerMiniBobineV40(t), nil, nil)
	profil, _ := ProfilDeDepartPourCarte(nil)
	declare := profile.MPPWidths{Lead: 8, Index: 3}
	if profil.MPP == declare {
		t.Fatalf("le profil de depart porte deja %v : le cas ne distingue rien", declare)
	}
	res := poserLeProfil(fc, profil)
	if !res.Decide() || res.Widths != declare || res.Provenance != profile.MPPPresumeParMesure {
		t.Fatalf("resolution %v (%v), attendu %v presume par mesure", res.Widths, res.Provenance, declare)
	}
	if got := fc.ProfilDeBalayage().MPP; got != declare {
		t.Errorf("decoupage du contexte %v, attendu %v", got, declare)
	}
}

// TestUnDecoupageMPPNonResoluSeDit : une resolution qui ne decide pas (declaration discordante) se
// dit en avertissement ; une resolution qui decide ne dit rien.
func TestUnDecoupageMPPNonResoluSeDit(t *testing.T) {
	ctx := &decodeCtx{name: "film"}
	ctx.signalerLeDecoupageMPP(grammar.ResolutionMPP{Widths: profile.MPPWidths{Lead: 9, Index: 5}})
	if ds := ctx.diag.Relever(); len(ds) != 0 {
		t.Fatalf("resolution decidee : %+v", ds)
	}
	ctx.signalerLeDecoupageMPP(grammar.ResolutionMPP{FormatVersion: 24,
		Declaration: grammar.DeclarationMPP{Records: 3, Discordants: 1}})
	ds := ctx.diag.Relever()
	if len(ds) != 1 || ds[0].Code != DiagDecoupageMPPNonResolu || ds[0].Niveau != constat.NiveauWarn {
		t.Fatalf("declaration discordante : %+v, attendu un avertissement %s", ds, DiagDecoupageMPPNonResolu)
	}
}
