package replay

// origine_positions_research_test.go — L'ORIGINE D'UN OBJET AU SOL : SOCLE OU SOL ?
//
// L'IDÉE (utilisateur, 2026-09-01), après la réfutation de la lévitation et de la récurrence :
// ne pas INFÉRER les points d'apparition depuis le film, les LIRE dans les données de carte,
// puis apparier par POSITION. « Si les coordonnées sont celles d'un socle -> socle ; si elles
// sont sur un sol quelconque -> sol. »
//
// ## CE QUE L'INVENTAIRE A TROUVÉ AVANT D'ÉCRIRE UNE LIGNE, ET QUI CHANGE LE PLAN
//
// 1. LE CATALOGUE DE SOCLES EXISTE DÉJÀ, et il vient bien du fichier de carte :
//    `data/titles/halo_infinite/reference/map_weapon_pads.json`, construit par
//    `cmd/mapopads-build --from <dossier de .mvar>` à partir des variantes UGC (Bond
//    CompactBinary v2). Positions en REPÈRE MONDE, mètres, NON transformées — le même repère
//    que les positions joueur du rejeu. C'est exactement l'entrée que l'idée demande.
// 2. L'ÉTALON EST DÉJÀ MESURÉ, et il est excellent : le croisement catalogue <-> socles du
//    match (`map_weapon_pads.go`) rapporte 32 positions d'oracle sur trois cartes, 32
//    appariées, MÉDIANE 0,01 m. Le seuil de production est 1,0 m, « pas une tolérance, une
//    marge ». Il n'y a donc pas d'ε à re-dériver — il y a un étalon à RE-VÉRIFIER sur mes
//    films, ce que fait la première mesure ci-dessous.
// 3. LE CATALOGUE NE CONNAÎT QUE TROIS TYPES D'OBJET : `0x5F379533` (power), `0x6253CFC0`
//    (rack), `0x5E86D110` (powerup) — une liste blanche (`mapvar.PadFamilyOf`). Le parseur,
//    lui, lit TOUS les objets du fichier (443 sur Cliffhanger, 337 sur Catalyst, dont 18 et 11
//    retenus). Les points d'apparition d'ÉQUIPEMENT et de GRENADES, s'ils existent, portent
//    d'autres types et sont donc ÉCARTÉS À LA CONSTRUCTION, pas absents du fichier.
// 4. LES `.mvar` NE SONT PLUS AU DÉPÔT. Zéro fichier trouvé (arborescence Scripts, Downloads,
//    Documents, dossier du jeu). Le catalogue a été généré le 2026-08-19 depuis un dossier de
//    dump qui n'existe plus. **On ne peut donc PAS élargir la liste blanche dans ce lot** — la
//    source manque. C'est le blocage de l'étape 2, publié comme tel.
//
// ## CE QUE CE LOT MESURE MALGRÉ TOUT, ET POURQUOI ÇA VAUT LE COUP
//
// La question « socle ou sol » se pose surtout pour les objets `ti=37` (équipement et
// power-ups). Le catalogue porte UN emplacement `powerup` par carte — peu, mais non nul. On
// peut donc déjà classer chaque naissance `ti=37` en trois seaux :
//
//	SOCLE      la naissance tombe à <= ε d'un emplacement CATALOGUÉ ;
//	SOL        elle tombe à <= 3 m de la FIN DE VIE d'un bipède, dans une fenêtre temporelle
//	           (l'équipement tombe à la mort — c'est le canal déjà mesuré du chantier voisin) ;
//	ABSTENTION ni l'un ni l'autre.
//
// ET C'EST LE SEAU « ABSTENTION » QUI PORTE LE RÉSULTAT LE PLUS UTILE. Si les abstentions se
// REGROUPENT à des positions récurrentes, ce sont des points d'apparition que le catalogue ne
// connaît pas — c'est-à-dire la preuve que la liste blanche est trop étroite, ET la liste des
// positions à chercher quand les `.mvar` seront re-dumpés. Si elles sont éparpillées, elles
// sont du sol qu'aucune mort n'explique.
//
// ## SEUILS ÉCRITS AVANT LA MESURE
//
//	E1 — ÉTALON : sur les naissances `ti=42` (armes au sol), la part à <= 1,0 m d'un
//	     emplacement catalogué doit être NETTEMENT au-dessus du témoin. C'est ce qui autorise
//	     à se servir du catalogue ; s'il échoue, rien de ce qui suit ne vaut.
//	E2 — TÉMOINS : positions du catalogue PERMUTÉES (chaque socle prend la position du
//	     suivant) et DÉCALÉES (+10 m en x). Les deux doivent effondrer le taux.
//	E3 — un regroupement d'abstentions ne compte comme « point d'apparition présumé » que
//	     s'il réunit >= 3 naissances à moins de 1,5 m les unes des autres.
//
// Gardes PICKUP_FILM + PICKUP_MAP (celles de `glResolve`) et ORIGINE_MAPID (le map_id UUID du
// catalogue, lu dans `map_weapon_pads.json` — chercher l'entree dont le `module` vaut
// `cliffhanger_ridgeline` ou `catalyst`).

import (
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// oriEps est le rayon d'appariement à un emplacement catalogué. C'est le seuil DE PRODUCTION
// (`MapWeaponPadMatchM`), repris tel quel : la mesure fondatrice donne 0,01 m de médiane, donc
// un mètre est une marge, pas une tolérance.
const oriEps = MapWeaponPadMatchM

// LE TEST « SOL » N'INVENTE AUCUN SEUIL : il reprend la RÈGLE DE PRODUCTION `gwPadsClass`
// (ground_weapon_rules.go) et ses deux constantes — `originDropMaxDist` (1,5 m) et
// `originDropWindowUS` (2 frames). C'est la règle mesurée du 2026-08-18 : 1 275 `dropped` sur
// 1 790 apparitions. Reprendre des bornes plus larges (3 m / 5 s, premier jet de cet
// instrument) aurait gonflé le seau « sol » avec des objets que la production classe autrement,
// et rendu les deux chiffres incomparables.

// oriClusterM / oriClusterMin : ce qui fait un « point d'apparition présumé » parmi les
// abstentions — au moins trois naissances dans un mouchoir d'un mètre et demi.
const (
	oriClusterM   = 1.5
	oriClusterMin = 3
)

// oriPads charge les emplacements catalogués de la carte sous test.
func oriPads(t *testing.T) []MapWeaponPadSpot {
	t.Helper()
	mapID := os.Getenv("ORIGINE_MAPID")
	if mapID == "" {
		t.Skip("ORIGINE_MAPID absent : instrument de mesure sauté")
	}
	path := filepath.Join(repoRootForTest(t), "data", "titles", "halo_infinite",
		"reference", "map_weapon_pads.json")
	cat, err := LoadMapWeaponPads(path)
	if err != nil {
		t.Fatalf("catalogue des socles : %v", err)
	}
	e, err := cat.Lookup(mapID)
	if err != nil {
		t.Fatalf("carte %q absente du catalogue : %v", mapID, err)
	}
	t.Logf("CATALOGUE : %s (%s) — %d objets au fichier, %d emplacements retenus",
		e.PublicName, e.Module, e.ObjectsN, len(e.Pads))
	return e.Pads
}

// oriNearest rend la distance au plus proche emplacement, les positions du catalogue étant
// translatées de (dx, dy) — le témoin.
//
// LE TÉMOIN N'EST PAS UNE PERMUTATION, ET LE PREMIER JET S'Y EST TROMPÉ : permuter la LISTE des
// socles ne change pas l'ENSEMBLE de leurs positions, donc la distance au plus proche est
// rigoureusement identique. La mesure l'a dénoncé toute seule (réel et « témoin » à 18,5 % et
// 5,72 m de médiane, au chiffre près). Un contrôle spatial doit DÉPLACER les points.
func oriNearest(pads []MapWeaponPadSpot, x, y, z float32, dx, dy float64) float64 {
	best := math.MaxFloat64
	for i := range pads {
		p := pads[i].Pos
		d := glDist(x, y, z, float32(p.X+dx), float32(p.Y+dy), float32(p.Z))
		if d < best {
			best = d
		}
	}
	return best
}

// oriStats rend la médiane et la part sous un seuil.
func oriStats(ds []float64, seuil float64) (float64, float64) {
	if len(ds) == 0 {
		return 0, 0
	}
	s := append([]float64(nil), ds...)
	sort.Float64s(s)
	c := 0
	for _, d := range s {
		if d <= seuil {
			c++
		}
	}
	return s[len(s)/2], 100 * float64(c) / float64(len(s))
}

// oriFlat aplatit la carte slot -> positions en un seul nuage.
func oriFlat(m map[uint32][]filmdec.BipedPosition) []filmdec.BipedPosition {
	var out []filmdec.BipedPosition
	for _, l := range m {
		out = append(out, l...)
	}
	return out
}

// oriLifeEnds rend les fins de vie de bipède : position d'arrêt et instant.
func oriLifeEnds(pos []filmdec.BipedPosition) []equipLife {
	var out []equipLife
	for _, lives := range equipmentLives(pos) {
		out = append(out, lives...)
	}
	return out
}

// oriNearDeath dit si une naissance tombe près d'une fin de vie, dans la fenêtre temporelle.
func oriNearDeath(ends []equipLife, x, y, z float32, at uint64) bool {
	for _, l := range ends {
		if equipTimeGap(at, l.to) > originDropWindowUS {
			continue
		}
		if dist3([3]float32{x, y, z}, [3]float32{l.x, l.y, l.z}) < originDropMaxDist {
			return true
		}
	}
	return false
}

// TestOrigineEtalonSurLesArmes — E1 et E2. Le catalogue est-il utilisable sur CE film ?
func TestOrigineEtalonSurLesArmes(t *testing.T) {
	s := glResolve(t)
	pads := oriPads(t)
	_, pst := decodeFilmPlacements(s.dir, &s.wr)
	scans := decodeFilmPadScans(s.dir, &s.wr, pst.Calibration.Widths)
	if !scans.Weapons.Scanned || len(scans.Weapons.Creations) == 0 {
		t.Fatalf("voie des armes muette : scanned=%v creations=%d",
			scans.Weapons.Scanned, len(scans.Weapons.Creations))
	}
	// L'ÉTALON NE PORTE QUE SUR LES ARMES *APPARUES*, et c'est un correctif de premier jet :
	// mélanger les LÂCHÉES (203 sur ce film) aux apparues mesure surtout des cadavres, pas des
	// socles. La classe vient de la règle de production (mêmes constantes que `gwPadsClass`).
	ends := oriLifeEnds(oriFlat(s.pos))
	var reel, dec1, dec2 []float64
	lachees := 0
	for _, c := range scans.Weapons.Creations {
		if oriNearDeath(ends, c.X, c.Y, c.Z, c.TimestampUS) {
			lachees++
			continue
		}
		reel = append(reel, oriNearest(pads, c.X, c.Y, c.Z, 0, 0))
		dec1 = append(dec1, oriNearest(pads, c.X, c.Y, c.Z, 10, 0))
		dec2 = append(dec2, oriNearest(pads, c.X, c.Y, c.Z, 0, -7))
	}
	mr, pr := oriStats(reel, oriEps)
	m1, p1 := oriStats(dec1, oriEps)
	m2, p2 := oriStats(dec2, oriEps)
	t.Logf("== ÉTALON — naissances ti=42 APPARUES contre le catalogue · %s ==", s.dir)
	t.Logf("naissances totales %d · lâchées (écartées) %d · APPARUES retenues %d",
		len(scans.Weapons.Creations), lachees, len(reel))
	t.Logf("RÉEL                — médiane %.2f m · part <= %.1f m : %.1f %%", mr, oriEps, pr)
	t.Logf("TÉMOIN décalé +10 x — médiane %.2f m · part <= %.1f m : %.1f %%", m1, oriEps, p1)
	t.Logf("TÉMOIN décalé -7 y  — médiane %.2f m · part <= %.1f m : %.1f %%", m2, oriEps, p2)
	t.Logf("VERDICT E1/E2 (réel >= 3x les DEUX témoins) : %v", pr >= 3*p1 && pr >= 3*p2)
}

// oriBucket porte le classement d'une naissance ti=37.
type oriBucket struct{ socle, sol, abstention int }

// TestOrigineEquipementSocleOuSol — LE TEST. Les naissances `ti=37` viennent-elles d'un
// emplacement catalogué, d'un corps, ou de nulle part de connu ?
func TestOrigineEquipementSocleOuSol(t *testing.T) {
	s := glResolve(t)
	pads := oriPads(t)
	_, pst := decodeFilmPlacements(s.dir, &s.wr)
	scans := decodeFilmPadScans(s.dir, &s.wr, pst.Calibration.Widths)
	pu := scans.Powerups
	if !pu.Scanned || len(pu.Creations) == 0 {
		t.Fatalf("voie des power-ups muette : scanned=%v creations=%d", pu.Scanned, len(pu.Creations))
	}
	ends := oriLifeEnds(oriFlat(s.pos))

	var b oriBucket
	var dPad []float64
	var orphelins []filmdec.EquipmentCreation
	tPermute, tDecale := 0, 0
	for _, c := range pu.Creations {
		d := oriNearest(pads, c.X, c.Y, c.Z, 0, 0)
		dPad = append(dPad, d)
		if oriNearest(pads, c.X, c.Y, c.Z, 10, 0) <= oriEps {
			tPermute++
		}
		if oriNearest(pads, c.X, c.Y, c.Z, 0, -7) <= oriEps {
			tDecale++
		}
		switch {
		case d <= oriEps:
			b.socle++
		case oriNearDeath(ends, c.X, c.Y, c.Z, c.TimestampUS):
			b.sol++
		default:
			b.abstention++
			orphelins = append(orphelins, c)
		}
	}
	n := len(pu.Creations)
	md, _ := oriStats(dPad, oriEps)
	t.Logf("== ORIGINE DES NAISSANCES ti=37 · %s ==", s.dir)
	t.Logf("naissances : %d · fins de vie de bipède : %d · distance médiane au socle catalogué le plus proche : %.2f m",
		n, len(ends), md)
	t.Logf("SOCLE (<= %.1f m d'un emplacement catalogué) : %d (%.1f %%)", oriEps, b.socle, pct100(b.socle, n))
	t.Logf("SOL (regle de production gwPadsClass)         : %d (%.1f %%)", b.sol, pct100(b.sol, n))
	t.Logf("ABSTENTION                                    : %d (%.1f %%)", b.abstention, pct100(b.abstention, n))
	t.Logf("TÉMOINS sur le seau SOCLE — décalé +10 x : %d (%.1f %%) · décalé -7 y : %d (%.1f %%)",
		tPermute, pct100(tPermute, n), tDecale, pct100(tDecale, n))
	oriClusters(t, orphelins)
}

// oriClusters regroupe les abstentions et publie celles qui RÉCURRENT — les points
// d'apparition présumés que le catalogue ne connaît pas (E3).
func oriClusters(t *testing.T, orph []filmdec.EquipmentCreation) {
	t.Helper()
	type cl struct {
		x, y, z float32
		n       int
	}
	var cls []cl
	for _, c := range orph {
		hit := -1
		for i := range cls {
			if glDist(c.X, c.Y, c.Z, cls[i].x, cls[i].y, cls[i].z) <= oriClusterM {
				hit = i
				break
			}
		}
		if hit < 0 {
			cls = append(cls, cl{c.X, c.Y, c.Z, 1})
			continue
		}
		cls[hit].n++
	}
	sort.Slice(cls, func(i, j int) bool { return cls[i].n > cls[j].n })
	gros := 0
	for _, c := range cls {
		if c.n >= oriClusterMin {
			gros++
		}
	}
	t.Logf("ABSTENTIONS : %d naissances -> %d regroupement(s) a <= %.1f m · dont >= %d naissances : %d",
		len(orph), len(cls), oriClusterM, oriClusterMin, gros)
	for i, c := range cls {
		if i >= 8 || c.n < oriClusterMin {
			break
		}
		t.Logf("  POINT PRÉSUMÉ x=%.2f y=%.2f z=%.2f — %d naissances", c.x, c.y, c.z, c.n)
	}
	t.Logf("VERDICT E3 (au moins un regroupement recurrent => la liste blanche du catalogue est trop etroite) : %v",
		gros > 0)
}
