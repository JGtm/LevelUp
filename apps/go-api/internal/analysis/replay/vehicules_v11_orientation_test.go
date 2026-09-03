package replay

// vehicules_v11_orientation_test.go — INSTRUMENT DU LOT V11 : « l'ORIENTATION du chassis, et
// celle de la TOURELLE » (2026-09-03).
//
// POURQUOI CE FICHIER EXISTE. Le lot V1a (2026-08-31) a REFUTE `i2` comme cap du vehicule :
// ecarts medians 40 a 137 deg au cap de deplacement, temoin par permutation identique. Cette
// refutation a ete prononcee AVEC LA MAUVAISE GRAMMAIRE — `ti=40` ne porte pas
// `object-forward-and-up-component` (FUN_14076e278, celui du bipede) mais la variante
// `object-forward-and-up-DYNAMIC-PRECISION-component` (FUN_140c5f7ec), etablie et portee par le
// lot V9 (2026-09-03, `filmdec/components_dynprec_orientation.go`). La refutation est donc a
// REJOUER : c'est l'objet de ce fichier.
//
// CE QUI N'EST PAS REECRIT. L'oracle, la regle de selection des paires, le calcul d'ecart
// circulaire et le TEMOIN PAR MELANGE viennent tels quels de `vehicules_v1a_test.go`
// (`v1aConfronteCaps`, `v1aPublieCap`, `v1aPermutation`). C'est la condition pour que les
// chiffres soient comparables a ceux de V1a : un seul parametre change, la grammaire d'i2/i3.
//
// LECTURE SEULE : aucun fichier ecrit, aucune base ouverte.
//
//	CGO_ENABLED=0 ATT_FILM=<depot>/data/cache \
//	  V0_FILMS="0d76e8f1:behemoth,fccc61cd:launch site" \
//	  go test ./internal/analysis/replay/ -run TestV11 -v -timeout 120m

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// v11Regime nomme une des deux grammaires d'i2/i3 confrontees.
type v11Regime struct {
	Nom     string
	DynPrec bool
}

// v11Regimes : le regime HISTORIQUE (grammaire du bipede, celui qui a servi a refuter i2 en
// V1a) et le regime ETABLI (variantes `-dynamic-precision-`, lot V9).
var v11Regimes = []v11Regime{
	{"AVANT (grammaire bipede, V1a)", false},
	{"APRES (dyn.-prec., V9)", true},
}

// TestV11OrientationChassis rejoue l'oracle geometrique du cap `i2` sous les DEUX grammaires.
//
// L'ORACLE, INCHANGE DEPUIS V1a : pour un chassis roulant vite, la direction lue en `i2` doit
// coincider avec le cap du DEPLACEMENT (positions consecutives) et avec celui de la VELOCITE
// `i1` du meme record. Seuils ecrits avant la mesure et non modifies : moyenne circulaire
// < 15 deg, mediane des ecarts absolus < 30 deg, temoin par melange deterministe.
//
// La ligne `i1 VELOCITE contre DEPLACEMENT` est le CONTROLE de curseur : elle vaut 1,7-2,1 deg
// en V1a. Si elle reste bonne sous les deux regimes, le curseur atteint `i1` correctement dans
// les deux cas et tout ecart sur `i2` vient bien d'`i2`.
func TestV11OrientationChassis(t *testing.T) {
	root := attRequireRoot(t)
	for _, f := range v0Corpus(t) {
		v11ChassisUnFilm(t, root, f)
	}
}

func v11ChassisUnFilm(t *testing.T, root string, f v0Film) {
	t.Helper()
	dir := objChunkDir(root, f.ID)
	if filmdec.CountFilmChunks(dir) == 0 {
		t.Logf("%s : film absent du cache — saute", f.ID)
		return
	}
	release := filmdec.LockProcessDecode()
	defer release()
	prev := filmdec.WorldObjectPrecision
	defer func() { filmdec.WorldObjectPrecision = prev }()
	wr, ok := v0Bornes(t, root, f.Carte)
	if !ok {
		return
	}
	bande := v1aBandeVehicule(dir)
	if len(bande) == 0 {
		t.Logf("V11 %s (%s) — bande ti=40 vide : rien a mesurer", f.ID, f.Carte)
		return
	}
	for _, rg := range v11Regimes {
		opt := v1aOptions(&wr, false)
		opt.CaptureDirs, opt.DynPrecOrientation = true, rg.DynPrec
		pos, err := filmdec.ScanFilmBipedPositionsForBand(dir, bande, opt)
		if err != nil {
			t.Logf("V11 %s [%s] : %v", f.ID, rg.Nom, err)
			continue
		}
		aim, vel := 0, 0
		for _, p := range pos {
			if p.HasAim {
				aim++
			}
			if p.HasVel {
				vel++
			}
		}
		t.Logf("V11 COUVERTURE %s (%s) [%s] — %d echantillons · i2 direction presente %d (%.1f %%) "+
			"· i1 direction presente %d (%.1f %%)",
			f.ID, f.Carte, rg.Nom, len(pos), aim, v11Pct(aim, len(pos)), vel, v11Pct(vel, len(pos)))
		caps, depl, velc, _ := v1aConfronteCaps(pos)
		v1aPublieCap(t, f, rg.Nom+" · i2 contre DEPLACEMENT", len(pos), caps, depl)
		v1aPublieCap(t, f, rg.Nom+" · i2 contre VELOCITE i1", len(pos), caps, velc)
		v1aPublieCap(t, f, rg.Nom+" · i1 VELOCITE contre DEPLACEMENT (controle)", len(pos), velc, depl)
	}
}

// v11Pct rend un pourcentage sans diviser par zero.
func v11Pct(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return 100 * float64(n) / float64(d)
}
