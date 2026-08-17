package filmdec

// R7-d — MESURER la grammaire d'ECRITURE d'`i0` contre la grammaire de LECTURE.
//
// CE QU'IL MESURE. L'ecrivain d'etat complet d'`i0` (`FUN_14320678c` = la case `+0x18` de la
// vtable du descripteur, cf. `WALK_PORT_NOTES.md` section « L ECRIVAIN PAR VTABLE ») pose,
// pour une image-cle :
//
//	W(1) 0            drapeau brut          (l'ecrivain d'etat complet le passe a 0)
//	W(1) 0            « a une baseline »    (l'ecrivain d'etat complet passe une ref NULLE)
//	W(1) h            (h != -1) = porte de la queue de handle
//	W(1) (idx == -1)  ; si idx != -1 : W(idxW)
//	3 x W(axisW)      les trois axes de la CARTE
//	si h : queue de handle
//	W(2)              INCONDITIONNEL, EN DERNIER
//
// Sans queue ni index, cela fait `6 + axisW[0] + axisW[1] + axisW[2]` bits — 46 sur un
// decoupage 13/13/14 — la ou le lecteur porte en consomme 117 en mediane (mesure R7-b).
// Le test balaie la largeur imposee a `i0` par le hook de calibration DEJA EXPORTE
// (`SetCalibratedWidth`) et publie, largeur par largeur, le taux d'atterrissage bit-exact et
// l'ecart absolu median. Aucun fichier partage du decodeur n'est modifie.
//
// CE QU'IL NE FAIT PAS : il ne publie AUCUNE donnee, n'ecrit RIEN sur disque, ne touche a
// aucun schema. LECTURE SEULE, garde par KF35_ROOT (meme garde que R7-a/R7-b) : il saute
// partout ailleurs, CI comprise.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 KF35_ROOT=<repo>/data/cache/film_chunks \
//	  go test ./internal/analysis/filmdec/ -run '^TestKF7D' -timeout 60m -v

import (
	"fmt"
	"testing"
)

// kf7dI0 est le nom de composant d'`i0` tel que le registre du film le porte.
const kf7dI0 = "object-position-dynamic-precision-component"

// kf7dVariant : la lecture de reference du lot precedent (etat complet, trous neutralises).
// C'est la SEULE variante balayee — le balayage porte sur la largeur d'`i0`, pas sur la
// forme du corps, deja tranchee par R7-b.
var kf7dVariant = kf35Variant{
	Label: "v4 ETAT COMPLET (64 leaf nus), composants non portes sautes (0 bit)",
	Body:  KeyframeBodyVariant{},
	Stub:  true,
}

// kf7dSweepLo / kf7dSweepHi bornent le balayage de largeur d'`i0`. La borne basse couvre la
// grammaire d'ecrivain la plus courte plausible (6 + 3 x 8 = 30) ; la borne haute depasse
// largement ce que le lecteur porte consomme aujourd'hui (117 en mediane).
const (
	kf7dSweepLo = 30
	kf7dSweepHi = 130
)

// kf7dResult porte le resultat d'UNE passe (une largeur imposee a `i0`, un film).
type kf7dResult struct {
	Width   int
	Exact   int
	Chained int
	Desync  int
	Bounded int
	MedGap  int
	MedLen  int
}

// kf7dPass joue la variante de reference sur un film avec une largeur imposee a `i0`
// (w < 0 = pas de calibration, c'est-a-dire le lecteur porte tel quel).
func kf7dPass(f kf35Film, w int) kf7dResult {
	SetCalibratedWidth(kf7dI0, w)
	defer SetCalibratedWidth(kf7dI0, -1)
	tal := kf35Pass(f, kf7dVariant)
	return kf7dResult{
		Width: w, Exact: tal.exact, Chained: tal.chained, Desync: tal.desync,
		Bounded: tal.bounded, MedGap: kf35Median(tal.absGaps), MedLen: kf35Median(tal.consumed),
	}
}

// kf7dPredicted rend la largeur que l'ECRIVAIN pose pour `i0` sur un decoupage donne, dans
// son cas dominant (pas d'index de plage, pas de queue de handle) :
// 2 bits d'en-tete + 1 porte de queue + 1 selecteur d'index + les trois axes + 2 de queue.
func kf7dPredicted(lay I0Layout) int {
	return 6 + int(lay.AxisW[0]) + int(lay.AxisW[1]) + int(lay.AxisW[2])
}

// kf7dLog publie une ligne de resultat.
func kf7dLog(t *testing.T, name, tag string, r kf7dResult) {
	t.Helper()
	rate := 0.0
	if r.Bounded > 0 {
		rate = 100 * float64(r.Exact+r.Chained) / float64(r.Bounded)
	}
	t.Logf("      [%s] %-34s exactes %4d · chainees %4d · desync %4d / %4d"+
		" | ATTERRISSAGE %5.2f %% | longueur MEDIANE %5d · ecart absolu MEDIAN %5d",
		name, tag, r.Exact, r.Chained, r.Desync, r.Bounded, rate, r.MedLen, r.MedGap)
}

// TestKF7DWriterI0 balaie la largeur d'`i0` et confronte le meilleur balayage a la largeur
// PREDITE par l'ecrivain. C'est la mesure de la phase 2 du plan R7-d.
func TestKF7DWriterI0(t *testing.T) {
	films := kf35Films(t)
	release := LockProcessDecode()
	defer release()

	prevSim := simStateComplete
	SetSimStateComplete(true)
	defer SetSimStateComplete(prevSim)
	prevCorr := filmComponentCorruptionCheck
	SetFilmComponentCorruptionCheck(false) // la meilleure configuration mesuree par R7-b
	defer SetFilmComponentCorruptionCheck(prevCorr)

	for _, f := range films {
		kf7dOneFilm(t, f)
	}
}

// kf7dOneFilm joue le balayage sur UN film : largeurs de la carte installees, trous
// neutralises une fois pour toutes, puis la reference et le balayage.
func kf7dOneFilm(t *testing.T, f kf35Film) {
	t.Helper()
	lay, restorePrec := kf35bInstallPrecision(t, f.Name)
	defer restorePrec()
	stubbed, restoreStubs := kf35ApplyStubs(f, kf7dVariant)
	defer restoreStubs()
	t.Logf("======== %s — composants neutralises : %d ========", f.Name, len(stubbed))

	base := kf7dPass(f, -1)
	kf7dLog(t, f.Name, "REFERENCE (lecteur porte)", base)

	pred := kf7dPredicted(lay)
	if lay.AxisW[0] > 0 {
		t.Logf("      [%s] largeur PREDITE par l'ecrivain (6 + %d+%d+%d) = %d bits",
			f.Name, lay.AxisW[0], lay.AxisW[1], lay.AxisW[2], pred)
		kf7dLog(t, f.Name, fmt.Sprintf("ECRIVAIN w=%d", pred), kf7dPass(f, pred))
	}

	best, bestGap := kf7dResult{Width: -1}, 1<<30
	for w := kf7dSweepLo; w <= kf7dSweepHi; w++ {
		r := kf7dPass(f, w)
		if r.Exact+r.Chained > best.Exact+best.Chained ||
			(r.Exact+r.Chained == best.Exact+best.Chained && r.MedGap < bestGap) {
			best, bestGap = r, r.MedGap
		}
	}
	kf7dLog(t, f.Name, fmt.Sprintf("MEILLEUR du balayage w=%d", best.Width), best)
}

// TestKF7DWriterI0Profile publie la largeur que le lecteur porte consomme AUJOURD'HUI sur
// `i0`, film par film. C'est le denominateur de la comparaison avec l'ecrivain : sans lui,
// « 46 contre 117 » ne se verifie pas.
func TestKF7DWriterI0Profile(t *testing.T) {
	films := kf35Films(t)
	release := LockProcessDecode()
	defer release()

	prevSim := simStateComplete
	SetSimStateComplete(true)
	defer SetSimStateComplete(prevSim)
	prevCorr := filmComponentCorruptionCheck
	SetFilmComponentCorruptionCheck(false)
	defer SetFilmComponentCorruptionCheck(prevCorr)

	for _, f := range films {
		kf7dProfileOne(t, f)
	}
}

// kf7dProfileOne publie, pour un film, la largeur mediane consommee par `i0` et son nombre
// de franchissements de frontiere.
func kf7dProfileOne(t *testing.T, f kf35Film) {
	t.Helper()
	lay, restorePrec := kf35bInstallPrecision(t, f.Name)
	defer restorePrec()
	_, restoreStubs := kf35ApplyStubs(f, kf7dVariant)
	defer restoreStubs()

	stats := kf35bProfile(f, kf7dVariant)
	for _, s := range stats {
		if s.Name != kf7dI0 {
			continue
		}
		t.Logf("  [%s] i0 vu %d fois · largeur consommee MEDIANE %d bits · franchissements %d"+
			" | largeur PREDITE par l'ecrivain %d bits",
			f.Name, s.Seen, kf35Median(s.Bits), s.Break, kf7dPredicted(lay))
		return
	}
	t.Logf("  [%s] i0 absent du profil", f.Name)
}
