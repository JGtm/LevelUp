package replay

// powerup_socle_creations_test.go — PHASE 3 (items 3.2 a 3.4) : LES RECORDS DE CREATION,
// SANS l'oracle de vie delta.
//
// C'EST LA MESURE QUI TRANCHE H4. La chaine de production (`ScanFilmEquipmentPlacements`) ne
// retient un record de creation `ti=37` que si sa position retombe sur le premier point d'une
// vie decodee des paquets DELTA. Un objet de socle ne bouge pas, n'emet aucune position delta,
// et se fait donc ecarter PAR CONSTRUCTION — quel que soit ce qu'il est. Ce fichier rejoue le
// meme balayage SANS ce filtre, et le remplace par celui qui a fait ses preuves sur les armes
// au sol : L'IDENTITE.
//
// POURQUOI L'IDENTITE ET PAS L'ACCEPTATION. L'en-tete NEW n'est pas selectif (un quart des
// positions de bit tirees au hasard passent le test de bande sur un film BTB) et le corps ne
// filtre qu'un peu. Le temoin FANTOME du balayage de creations n'est PAS discriminant a lui
// seul — decouverte 2 du plan des armes au sol : 398 creations acceptees sur une bande fantome
// contre 366 sur la vraie. Ce qui discrimine, c'est que le mot MPP de 32 bits se RESOLVE dans
// le manifeste du titre : 1 785 retenues contre 13 fantomes, un facteur 137.
//
// LES LARGEURS DU BLOC MPP SONT CELLES DU FILM. Elles se mesurent film par film
// (`CalibrateMPPWidths`, 9/5 en Quick Play, 8/3 sur les films BTB mesures) et
// `ScanFilmEquipmentPlacements` les RESTAURE en sortant : il faut les REPOSER avant le
// balayage brut, sans quoi zero creation ne resout et rien ne le dit (decouverte 8 du meme
// plan). C'est ce que fait `gwInstallMPPWidths`, deja en production.
//
// LECTURE SEULE. Garde `OBJ_FILM` + `OBJ_FILM_ART`.
//
//	CGO_ENABLED=0 OBJ_FILM=<depot>/data/cache \
//	  go test ./internal/analysis/replay/ -run '^TestPowerupSocleCreations$' -timeout 90m -v

import (
	"math"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// padPowerupPrefix — le prefixe de famille du manifeste qui designe un power-up
// (`powerup_overshield`, `powerup_camo`). Meme litteral que la table de rendu du web.
// padPowerupPrefix a ete RETIRE le 2026-08-19 : le prefixe vit en PRODUCTION
// (`padPowerupPrefix`, powerup_pads.go), et un garde-rail interdit sa reecriture.

// psCreationVue est UN record de creation retenu, avec ce qui permet de le juger.
type psCreationVue struct {
	Slot, Gen uint32
	US        uint64
	X, Y, Z   float32
	ID        uint32
	Famille   string
	// D3 est la distance 3D au socle mesure ; DXY la distance dans le plan.
	D3, DXY float64
}

// psBandeFantome construit une bande de slots de MEME cardinalite que la vraie, faite de slots
// que les images-cles n'ont JAMAIS vus porter d'archetype. C'est le temoin : il passe par le
// MEME code de balayage que la mesure, sans quoi il ne controlerait pas le decodeur mais une
// variante de lui.
func psBandeFantome(kfs []psKF, taille int) map[uint32]bool {
	vus := map[uint32]bool{}
	for _, kf := range kfs {
		for _, slots := range kf.Slots {
			for s := range slots {
				vus[s] = true
			}
		}
	}
	out := map[uint32]bool{}
	for s := uint32(0); s < 8192 && len(out) < taille; s++ {
		if !vus[s] {
			out[s] = true
		}
	}
	return out
}

// psRetientCreations projette les creations sur le catalogue d'identite et rend celles qui s'y
// resolvent, plus le compte de celles qui ne s'y resolvent pas.
func psRetientCreations(
	cre []filmdec.EquipmentCreation, familles map[uint32]string, c psCible,
) ([]psCreationVue, int) {
	var out []psCreationVue
	rejetees := 0
	for _, k := range cre {
		if !k.MPPPresent[filmdec.MPPWord32] {
			rejetees++
			continue
		}
		id := uint32(k.MPPVal[filmdec.MPPWord32])
		fam, connu := familles[id]
		if !connu {
			rejetees++
			continue
		}
		out = append(out, psCreationVue{
			Slot: k.Slot, Gen: k.Gen, US: k.TimestampUS, X: k.X, Y: k.Y, Z: k.Z,
			ID: id, Famille: fam,
			D3:  psDist3(k.X, k.Y, k.Z, c.P, c.Z),
			DXY: psDist(psPoint{X: k.X, Y: k.Y}, c.P),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].US < out[j].US })
	return out, rejetees
}

// TestPowerupSocleCreations — 3.2 a 3.4 du plan, sur les quatre films Catalyst.
func TestPowerupSocleCreations(t *testing.T) {
	root := objRequireRoot(t)
	entry := psEntreeCarte(t)
	socleP, socleZ := psSocleMesure(t)
	familles := goldenCatalog(t).EquipmentFamilies
	t.Logf("catalogue d'identite `eqip` : %d entrees", len(familles))

	for _, f := range psFilmsCatalyst {
		t.Run(f.ID+"_"+f.Mode, func(t *testing.T) {
			dir := filepath.Join(root, "film_chunks", f.ID)
			if filmdec.CountFilmChunks(dir) == 0 {
				t.Skipf("aucun chunk dans %s", dir)
			}
			release := filmdec.LockProcessDecode()
			defer release()
			defer installWorldObjectPrecision(entry, dir)()
			wr := entry.Range()
			c := psCible{P: socleP, Z: socleZ, T0Film: psPremierPaquetUS(dir)}
			psMesureCreations(t, dir, &wr, familles, c)
		})
	}
}

// psMesureCreations enchaine les trois lectures d'un film : la chaine de PRODUCTION (pour le
// rappel du negatif et la calibration), le balayage BRUT des creations, et le temoin fantome.
func psMesureCreations(
	t *testing.T, dir string, wr *filmdec.Vec3Range, familles map[uint32]string, c psCible,
) {
	t.Helper()
	_, pst, err := filmdec.ScanFilmEquipmentPlacements(dir, wr)
	if err != nil {
		t.Logf("=== 3.2 chaine de production : %v", err)
		return
	}
	t.Logf("=== 3.2 CHAINE DE PRODUCTION (avec l'oracle de vie delta) ===")
	t.Logf("  calibration MPP %s (tranchee %v) | %d vies | %d ancres | %d acceptees"+
		" | %d confirmees | %d poses publiees",
		pst.Calibration.Widths, pst.Calibration.Widths.Valid(), pst.Lives, pst.Anchors,
		pst.Accepted, pst.Confirmed, pst.Placements)

	defer gwInstallMPPWidths(pst.Calibration.Widths)()
	cre, cst, err := filmdec.ScanFilmEquipmentCreations(dir, wr)
	if err != nil {
		t.Logf("=== 3.3 balayage brut : %v", err)
		return
	}
	vues, rejetees := psRetientCreations(cre, familles, c)
	t.Logf("=== 3.3 BALAYAGE BRUT (sans l'oracle de vie delta) ===")
	t.Logf("  bande %d slots | %d ancres | %d acceptees | %d RETENUES par identite | %d ecartees",
		cst.Slots, cst.Anchors, cst.Accepted, len(vues), rejetees)
	psRapportPowerups(t, vues, c)

	kfs := psRecenseKF(dir)
	fant := psBandeFantome(kfs, cst.Slots)
	fcre, fst, err := filmdec.ScanFilmEquipmentCreationsForBand(dir, wr, fant)
	if err != nil {
		t.Logf("=== 3.3 temoin fantome : %v", err)
		return
	}
	fvues, _ := psRetientCreations(fcre, familles, c)
	facteur := math.Inf(1)
	if len(fvues) > 0 {
		facteur = float64(len(vues)) / float64(len(fvues))
	}
	t.Logf("  TEMOIN FANTOME : bande %d slots | %d acceptees | %d retenues"+
		" -> facteur %.1f (seuil ecrit avant mesure : >= 10)",
		fst.Slots, fst.Accepted, len(fvues), facteur)
}

// psRapportPowerups ecrit la distribution des familles retenues, puis DETAILLE les power-ups —
// les seuls qui repondent a la question du lot.
func psRapportPowerups(t *testing.T, vues []psCreationVue, c psCible) {
	t.Helper()
	parFamille := map[string]int{}
	var pow []psCreationVue
	proche := math.Inf(1)
	for _, v := range vues {
		parFamille[v.Famille]++
		if !strings.HasPrefix(v.Famille, padPowerupPrefix) {
			continue
		}
		pow = append(pow, v)
		if v.D3 < proche {
			proche = v.D3
		}
	}
	familles := make([]string, 0, len(parFamille))
	for f := range parFamille {
		familles = append(familles, f)
	}
	sort.Slice(familles, func(i, j int) bool { return parFamille[familles[i]] > parFamille[familles[j]] })
	for _, f := range familles {
		t.Logf("      %-24s %4d", f, parFamille[f])
	}
	if len(pow) == 0 {
		t.Log("  AUCUN power-up parmi les creations retenues")
		return
	}
	t.Logf("  %d creation(s) de power-up, la plus proche du socle a %.2f m (3D) :", len(pow), proche)
	for i, v := range pow {
		if i >= 25 {
			t.Logf("      (... %d de plus)", len(pow)-i)
			return
		}
		t.Logf("      %7.1f s | %-20s %#08x | (%7.3f ; %7.3f ; %6.2f)"+
			" | XY %.2f m | 3D %.2f m | slot %4d gen %d",
			psSecondes(v.US, c.T0Film), v.Famille, v.ID, v.X, v.Y, v.Z, v.DXY, v.D3, v.Slot, v.Gen)
	}
}
