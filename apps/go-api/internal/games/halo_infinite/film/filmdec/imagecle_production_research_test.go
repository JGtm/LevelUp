package filmdec

// imagecle_production_research_test.go — PHASE 5a, OBJECTIF 4 : CE QUE LA PRODUCTION DERAILLE
// AUJOURD'HUI, AVEC SES PROPRES COMPTEURS.
//
// # OU LA PRODUCTION PARSE-T-ELLE LE CORPS D'UN RECORD D'IMAGE-CLE ? LA REPONSE EST COURTE
//
// Presque nulle part — et c'est le premier resultat de cet objectif. Le releve du 2026-09-13 :
//
//	`ScanNavpointRadial` (ti=12, `navpoint_radial_scan.go`)   SEUL CHEMIN DE PRODUCTION :
//	    appele par `replay/bomb_armings.go:160` (`decodeFilmBombReads`), c'est-a-dire par la
//	    cuisson du rejeu, pour le compte a rebours d'armement d'Assaut. Sa voie IMAGE-CLE
//	    (`scanKeyframe`) lit un en-tete de 64 bits puis `TraverseEntity`.
//	`ScanObjectives` (ti=11, `objective_scan.go`)             MEME FORME, mais son propre
//	    en-tete de fichier le declare « instrument de mesure, PAS une source de production ».
//	`WalkKeyframeRecords` (`keyframe_record_walk.go`)         AUCUN APPELANT hors du paquet.
//
// Tous les autres lecteurs d'image-cle de la production ne parsent PAS le corps : ils BALAIENT
// des motifs d'octets dans l'emprise du record (`keyframe_loadout.go`, `keyframe_ground_weapons.go`,
// `keyframe_carrier_mark.go`) ou n'utilisent que les ANCRES (`WalkKeyframeWorld`,
// `KeyframeRecordSpans`). C'est precisement parce que la grammaire du corps deraille que ces
// lecteurs ont ete ecrits en balayage.
//
// **`KillSourceHealth` ne porte AUCUN compteur de record d'image-cle** : ses champs comptent
// des candidats d'attribution de mort (`Candidates`, `UnexplainedPair/Self/BotIdx`) et sa
// `CoverageRatio` vaut `DeathsCovered / DeathsReal`. Le dire est un resultat : il n'existe pas
// de compteur de production « la table d'image-cle a deraille ». Les seuls qui existent sont
// les `Key*` des deux balayages ci-dessus.
//
// # CE QUE CET INSTRUMENT MESURE
//
//	1. Les compteurs de production, TELS QUELS, sur les 6 films : `KeyRecords`, `KeyWalked`,
//	   `KeyBroken`, `KeyChained` — et, pour ti=12, le composant qui bloque (`Blocked`).
//	2. La MEME population sous la bonne forme, avec les MEMES definitions de compteur, plus la
//	   fermeture EXACTE (que la production ne mesure pas : son `KeyChained` demande seulement
//	   que la position d'arrivee porte un en-tete valide).
//
// `KeyChained` est la definition FAIBLE de la sante : un en-tete valide a la position
// d'arrivee. La fermeture EXACTE est la definition forte. Publier les deux evite de compter
// pour un succes une marche qui tombe par hasard sur un motif d'en-tete.
//
// Garde CHUNK00_FILMS. Aucun code de production modifie — les deux balayages sont appeles tels
// quels.

import (
	"fmt"
	"path/filepath"
	"testing"
)

// imcpCompteurs porte les quatre compteurs, quelle que soit leur origine.
type imcpCompteurs struct {
	Records, Walked, Broken, Chained int
	// Ferme n'existe PAS cote production : c'est la definition forte, ajoutee ici.
	Ferme, Bornes int
}

// imcpTexte rend une ligne de compteurs avec ses parts.
func (c imcpCompteurs) imcpTexte() string {
	if c.Records == 0 {
		return "aucun record"
	}
	return fmt.Sprintf("records %5d · marches %5d · CASSES %5d (%5.1f %%) · chainees %5d (%5.1f %%)"+
		" · FERMEES %5d/%d (%5.1f %%)",
		c.Records, c.Walked, c.Broken, 100*float64(c.Broken)/float64(c.Records),
		c.Chained, 100*float64(c.Chained)/float64(c.Records),
		c.Ferme, c.Bornes, imcpPart(c.Ferme, c.Bornes))
}

// imcpPart rend un pourcentage sans diviser par zero.
func imcpPart(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return 100 * float64(n) / float64(d)
}

// imcpEtatComplet rejoue la MEME population sous la bonne forme, avec les MEMES definitions de
// compteur que `scanKeyframe`, plus la fermeture exacte.
func imcpEtatComplet(f imcFilm, ti int) imcpCompteurs {
	var c imcpCompteurs
	for _, pay := range f.Pays {
		total := len(pay) * 8
		for _, b := range imcBornes(pay) {
			if b.TI != ti {
				continue
			}
			c.Records++
			c.Bornes++
			tr := WalkKeyframeFullState(pay, b.Bit, f.Reg)
			if tr.DesyncAt >= 0 || tr.EndBit > total {
				c.Broken++
				continue
			}
			c.Walked++
			if _, ok := readKeyframeHeader(pay, tr.EndBit, total); ok {
				c.Chained++
			}
			if tr.EndBit == b.Want {
				c.Ferme++
			}
		}
	}
	return c
}

// imcpProduction rejoue la population sous le modele de PRODUCTION, avec les memes definitions
// — et la fermeture exacte en plus, pour que les deux colonnes se comparent terme a terme.
func imcpProduction(f imcFilm, ti int) imcpCompteurs {
	var c imcpCompteurs
	for _, pay := range f.Pays {
		total := len(pay) * 8
		for _, b := range imcBornes(pay) {
			if b.TI != ti {
				continue
			}
			c.Records++
			c.Bornes++
			h, ok := readKeyframeHeader(pay, b.Bit, total)
			if !ok {
				c.Broken++
				continue
			}
			rec, _, _ := walkOneKeyframeRecord(pay, f.Reg, b.Bit, h)
			if rec.DesyncAt >= 0 || rec.BitEnd > total {
				c.Broken++
				continue
			}
			c.Walked++
			if _, ok := readKeyframeHeader(pay, rec.BitEnd, total); ok {
				c.Chained++
			}
			if rec.BitEnd == b.Want {
				c.Ferme++
			}
		}
	}
	return c
}

// imcpArchetypesScannes sont les archetypes dont la PRODUCTION parse le corps d'image-cle :
// ti=12 (anneau des navpoints, chemin de la cuisson) et ti=11 (objectifs geres, instrument).
func imcpArchetypesScannes() []int { return []int{navpointRadialArchIndex, ObjectiveTypeIndex} }

// TestImageCleProductionCompteurs publie les compteurs de PRODUCTION tels quels, puis les
// memes compteurs sous la bonne forme.
func TestImageCleProductionCompteurs(t *testing.T) {
	rel := LockProcessDecode()
	defer rel()
	dirs := chunk00Films(t, "CHUNK00_FILMS")
	cumul := map[string]map[int]*imcpCompteurs{
		imcProduction:  {},
		imcEtatComplet: {},
	}
	films := 0
	for _, dir := range dirs {
		f, ok := imcCharger(t, dir)
		if !ok {
			continue
		}
		films++
		t.Logf("== film %s (build %q) ==", f.Nom, f.Build)
		imcpUnFilm(t, f, cumul)
	}
	if films == 0 {
		t.Skip("aucun film exploitable")
	}
	t.Logf("---- CUMUL SUR %d FILMS ----", films)
	for _, ti := range imcpArchetypesScannes() {
		t.Logf("ti=%-2d PRODUCTION   %s", ti, cumul[imcProduction][ti].imcpTexte())
		t.Logf("ti=%-2d ETAT COMPLET %s", ti, cumul[imcEtatComplet][ti].imcpTexte())
	}
}

// imcpUnFilm mesure un film : les deux modeles sur les deux archetypes scannes.
func imcpUnFilm(t *testing.T, f imcFilm, cumul map[string]map[int]*imcpCompteurs) {
	t.Helper()
	for _, ti := range imcpArchetypesScannes() {
		pr, ec := imcpProduction(f, ti), imcpEtatComplet(f, ti)
		t.Logf("  ti=%-2d PRODUCTION   %s", ti, pr.imcpTexte())
		t.Logf("  ti=%-2d ETAT COMPLET %s", ti, ec.imcpTexte())
		imcpCumuler(cumul[imcProduction], ti, pr)
		imcpCumuler(cumul[imcEtatComplet], ti, ec)
	}
}

// imcpCumuler additionne des compteurs dans le cumul d'un archetype.
func imcpCumuler(m map[int]*imcpCompteurs, ti int, c imcpCompteurs) {
	d := m[ti]
	if d == nil {
		d = &imcpCompteurs{}
		m[ti] = d
	}
	d.Records += c.Records
	d.Walked += c.Walked
	d.Broken += c.Broken
	d.Chained += c.Chained
	d.Ferme += c.Ferme
	d.Bornes += c.Bornes
}

// TestImageCleProductionBalayagesReels appelle les DEUX balayages de production TELS QUELS et
// publie leurs compteurs `Key*` sans les recalculer.
//
// C'est la version « compteur de production utilise tel quel » demandee par le lot : les
// chiffres ci-dessous sont ceux que la cuisson elle-meme produirait sur ces films.
func TestImageCleProductionBalayagesReels(t *testing.T) {
	rel := LockProcessDecode()
	defer rel()
	var nr, nw, nb, nc int
	var or, ow, ob, oc int
	for _, dir := range chunk00Films(t, "CHUNK00_FILMS") {
		nom := filepath.Base(dir)
		if sc, err := ScanFilmNavpointRadial(dir, map[int]int{}); err == nil {
			t.Logf("%-10s ti=12 (PRODUCTION, appele par replay/bomb_armings.go) : "+
				"KeyRecords %4d · KeyWalked %4d · KeyBroken %4d · KeyChained %4d | bloque a %v",
				nom, sc.KeyRecords, sc.KeyWalked, sc.KeyBroken, sc.KeyChained, sc.Blocked)
			nr, nw, nb, nc = nr+sc.KeyRecords, nw+sc.KeyWalked, nb+sc.KeyBroken, nc+sc.KeyChained
		} else {
			t.Logf("%-10s ti=12 : balayage impossible (%v)", nom, err)
		}
		if sc, err := ScanFilmObjectives(dir); err == nil {
			t.Logf("%-10s ti=11 (instrument) : KeyRecords %4d · KeyWalked %4d · KeyBroken %4d"+
				" · KeyChained %4d", nom, sc.KeyRecords, sc.KeyWalked, sc.KeyBroken, sc.KeyChained)
			or, ow, ob, oc = or+sc.KeyRecords, ow+sc.KeyWalked, ob+sc.KeyBroken, oc+sc.KeyChained
		} else {
			t.Logf("%-10s ti=11 : balayage impossible (%v)", nom, err)
		}
	}
	t.Logf("CUMUL ti=12 PRODUCTION : records %d · marches %d · CASSES %d (%.1f %%) · chainees %d",
		nr, nw, nb, imcpPart(nb, nr), nc)
	t.Logf("CUMUL ti=11 instrument : records %d · marches %d · CASSES %d (%.1f %%) · chainees %d",
		or, ow, ob, imcpPart(ob, or), oc)
}
