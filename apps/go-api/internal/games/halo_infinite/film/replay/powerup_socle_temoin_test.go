package replay

// powerup_socle_temoin_test.go — LE CONTROLE de la remontee de la phase 1.
//
// Regle, seuils et code de la remontee : `powerup_socle_oracle_test.go`. Ce fichier n'en
// garde aucune copie — il appelle `psSocleParRemontee` et `psNuageA`.

import (
	"math"
	"sort"
	"testing"
)

// ---------------------------------------------------------------------------------------
// CONTROLE DE LA REMONTEE (item 3.0 du plan).
//
// CE QU'IL FAUT ECARTER. Le centre d'une arene est un goulot : quatre joueurs quelconques y
// passent souvent. Un minimum de dispersion au centre pourrait donc n'etre qu'un artefact de
// frequentation, sans rapport avec un socle.
//
// LE TEMOIN QUI TRANCHE : les MEMES instants `T0`, la MEME remontee, le MEME code — mais
// D'AUTRES VIES. Si « quatre joueurs a ces instants-la » suffit a produire un croisement a
// 25 cm, le resultat de la phase 1 ne vaut rien. Si le temoin reste au-dessus du seuil, alors
// c'est bien CES porteurs-la qui venaient du meme point.
// ---------------------------------------------------------------------------------------

// psAutresVies rend, pour chaque episode, une vie DIFFERENTE vivante au meme instant : le
// slot le plus petit qui porte un point a `T0` et qui n'est pas celui de l'episode.
func psAutresVies(doc ReplayDocument, eps []EquipmentEpisode) []EquipmentEpisode {
	porteurs := map[uint32]bool{}
	for _, e := range eps {
		porteurs[e.Slot] = true
	}
	out := make([]EquipmentEpisode, 0, len(eps))
	for _, e := range eps {
		var slots []uint32
		for _, tr := range doc.Tracks {
			if porteurs[tr.Slot] {
				continue
			}
			for _, p := range tr.Points {
				if p.T == e.T0 {
					slots = append(slots, tr.Slot)
					break
				}
			}
		}
		if len(slots) == 0 {
			continue
		}
		sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
		out = append(out, EquipmentEpisode{Slot: slots[0], Fam: e.Fam, T0: e.T0, T1: e.T1})
	}
	return out
}

// TestPowerupSocleTemoinRemontee — le controle : la meme remontee sur d'AUTRES vies.
func TestPowerupSocleTemoinRemontee(t *testing.T) {
	dir := psArtDir(t)
	doc, ok := psLoadDoc(t, dir, "01e1f945")
	if !ok {
		t.Skipf("artefact 01e1f945 absent de %s", dir)
	}
	reel, ok := psSocleParRemontee(doc)
	if !ok {
		t.Skip("la remontee reelle ne rend pas de socle : rien a controler")
	}
	var garde []EquipmentEpisode
	for _, e := range doc.EquipmentEpisodes {
		if e.Fam != EquipFamilyOvershield {
			continue
		}
		p, _, _, ok := psPosDeLaVie(doc, e.Slot, e.T0)
		if !ok {
			continue
		}
		if _, lache := psExpliqueParUnLacher(doc, e, p); !lache {
			garde = append(garde, e)
		}
	}
	temoins := psAutresVies(doc, garde)
	t.Logf("=== 3.0 CONTROLE : %d vies temoins pour %d porteurs ===", len(temoins), len(garde))
	if len(temoins) < 2 {
		t.Skip("pas assez de vies contemporaines pour un temoin")
	}
	for _, e := range temoins {
		t.Logf("  temoin slot %3d a t %5d", e.Slot, e.T0)
	}
	bestK, bestR, bestC := -1, math.Inf(1), psPoint{}
	for k := 0; k <= psLagMax; k++ {
		pts, _ := psNuageA(doc, temoins, k)
		if len(pts) < len(temoins) {
			continue
		}
		if c, r := psDispersion(pts); r < bestR {
			bestK, bestR, bestC = k, r, c
		}
	}
	t.Logf("  REEL   : rayon %.3f m a k=%d, centroide (%.3f ; %.3f)",
		reel.R, reel.K, reel.C.X, reel.C.Y)
	t.Logf("  TEMOIN : rayon %.3f m a k=%d, centroide (%.3f ; %.3f)",
		bestR, bestK, bestC.X, bestC.Y)
	if reel.R > 0 {
		t.Logf("  facteur temoin/reel %.2f (seuil ecrit avant mesure : >= %.0f)",
			bestR/reel.R, psFacteurTemoin)
	}
}
