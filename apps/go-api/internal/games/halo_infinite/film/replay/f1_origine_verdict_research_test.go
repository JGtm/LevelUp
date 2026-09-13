package replay

// f1_origine_verdict_research_test.go — LOT F.1 : le VERDICT de la mesure avant / apres.
//
// Le contexte, la regle temoin et l'identification de carte vivent dans
// `f1_origine_mesure_research_test.go` ; ce fichier ne porte que la campagne et ses tableaux.
//
// CE QUI EST COMPTE, ET DANS QUEL ORDRE :
//
//  1. par ORIGINE, avant et apres — le total qui bascule ;
//  2. par FAMILLE x TRANSITION — la reponse a « aucune autre famille touchee » ;
//  3. la LISTE des poses qui changent, avec leur ecart de temps et leur distance : c'est elle
//     qui permet de verifier qu'on a bien retire une clause de distance et rien d'autre.
//
// LECTURE SEULE. Gardes et commande : `f1_origine_mesure_research_test.go`.

import (
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/filmdec"
)

// f1Change est UNE pose dont l'origine bascule.
type f1Change struct {
	Film    string
	Famille string
	ID      uint32
	Slot    uint32
	GapMS   float64
	DistM   float64
	Avant   string
	Apres   string
}

// TestF1OrigineAvantApres : la mesure du lot F.1.
func TestF1OrigineAvantApres(t *testing.T) {
	root, ids := f1Films(t)
	cat := f1Catalogue(t)
	familles := goldenCatalog(t).EquipmentFamilies

	parcAvant := map[string]int{}
	parcApres := map[string]int{}
	parcTrans := map[string]int{}
	var changes []f1Change
	films, hors := 0, 0

	for _, id := range ids {
		dir := filepath.Join(root, id)
		e, nom, ecart, ok := f1Carte(t, dir, id, cat)
		if !ok {
			hors++
			continue
		}
		poses, pos, ok := f1PosesEtPositions(t, dir, &e)
		if !ok {
			t.Logf("film %s : poses illisibles — hors mesure", id)
			hors++
			continue
		}
		films++
		t.Logf("")
		t.Logf("######## FILM %s (carte %s, ecart aux reperes publies %.3f m) — %d poses ########",
			id, nom, ecart, len(poses))
		avant, apres, trans, ch := f1Compare(id, poses, pos, familles)
		f1LogOrigines(t, "  ", avant, apres)
		f1LogTable(t, "  transition", trans)
		for k, v := range avant {
			parcAvant[k] += v
		}
		for k, v := range apres {
			parcApres[k] += v
		}
		for k, v := range trans {
			parcTrans[k] += v
		}
		changes = append(changes, ch...)
	}

	t.Logf("")
	t.Logf("######## PARC — %d films mesures, %d hors mesure ########", films, hors)
	f1LogOrigines(t, "  ", parcAvant, parcApres)
	f1LogTable(t, "  transition", parcTrans)
	t.Logf("  POSES QUI CHANGENT : %d", len(changes))
	sort.Slice(changes, func(i, j int) bool {
		if changes[i].Film != changes[j].Film {
			return changes[i].Film < changes[j].Film
		}
		return changes[i].DistM > changes[j].DistM
	})
	for _, c := range changes {
		t.Logf("      %s 0x%08x %-14s poseur=%-4d ecart %6.1f ms distance %6.2f m : %s -> %s",
			c.Film, c.ID, c.Famille, c.Slot, c.GapMS, c.DistM, c.Avant, c.Apres)
	}
}

// f1PosesEtPositions rend les poses de production du film et le nuage TRIE des bipedes — les
// deux entrees exactes de `buildEquipmentPlacements`.
func f1PosesEtPositions(t *testing.T, dir string, e *filmdec.MapQuantEntry) (
	[]filmdec.EquipmentPlacement, []filmdec.BipedPosition, bool) {
	t.Helper()
	pos, ok := f1Positions(t, dir, *e)
	if !ok {
		return nil, nil, false
	}
	release := filmdec.LockProcessDecode()
	defer release()
	defer installWorldObjectPrecision(*e, dir)()
	wr := e.Range()
	raw, st, err := filmdec.ScanFilmEquipmentPlacements(dir, &wr)
	if err != nil || !st.Calibration.Widths.Valid() {
		return nil, nil, false
	}
	return raw, pos, true
}

// f1Compare classe chaque pose avec les deux regles et rend les quatre tableaux.
//
// LE POSEUR EST CELUI DE LA PRODUCTION (`equipmentOwner`) : sans lui, l'origine est `unknown`
// des deux cotes, et la compter ailleurs fabriquerait une transition qui n'existe pas.
func f1Compare(id string, raw []filmdec.EquipmentPlacement, positions []filmdec.BipedPosition,
	familles map[uint32]string) (map[string]int, map[string]int, map[string]int, []f1Change) {
	avant, apres, trans := map[string]int{}, map[string]int{}, map[string]int{}
	lives := equipmentLives(positions)
	var changes []f1Change
	for _, p := range raw {
		fam := familles[p.GlobalID]
		if fam == "" {
			fam = equipmentFamilyOther
		}
		var vies []equipLife
		var slot uint32
		if s, _, ok := equipmentOwner(positions, p); ok {
			slot, vies = s, lives[s]
		}
		a := f1OrigineAvant(vies, p)
		b := equipmentOrigin(vies, p)
		avant[a]++
		apres[b]++
		if a == b {
			continue
		}
		trans[fam+" "+a+" -> "+b]++
		best, _ := f1VieRetenue(vies, p.T0US)
		changes = append(changes, f1Change{
			Film: id, Famille: fam, ID: p.GlobalID, Slot: slot,
			GapMS: float64(equipTimeGap(p.T0US, best.to)) / 1000,
			DistM: dist3([3]float32{p.X, p.Y, p.Z}, [3]float32{best.x, best.y, best.z}),
			Avant: a, Apres: b,
		})
	}
	return avant, apres, trans, changes
}

// f1LogOrigines ecrit les comptes par origine, avant et apres.
func f1LogOrigines(t *testing.T, indent string, avant, apres map[string]int) {
	t.Helper()
	cles := map[string]bool{}
	for k := range avant {
		cles[k] = true
	}
	for k := range apres {
		cles[k] = true
	}
	var l []string
	for k := range cles {
		l = append(l, k)
	}
	sort.Strings(l)
	for _, k := range l {
		t.Logf("%sorigine %-10s avant %5d -> apres %5d  (%+d)", indent, k,
			avant[k], apres[k], apres[k]-avant[k])
	}
}

// f1LogTable ecrit une table cle -> compte, triee.
func f1LogTable(t *testing.T, titre string, m map[string]int) {
	t.Helper()
	if len(m) == 0 {
		t.Logf("%s : aucune", titre)
		return
	}
	var l []string
	for k := range m {
		l = append(l, k)
	}
	sort.Strings(l)
	for _, k := range l {
		t.Logf("%s : %-44s %5d", titre, k, m[k])
	}
}
