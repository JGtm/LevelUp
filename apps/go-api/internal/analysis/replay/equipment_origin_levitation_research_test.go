package replay

// equipment_origin_levitation_research_test.go — LOT 6, RECHERCHE PURE : L'ORIGINE D'UNE PRISE,
// par deux idées neuves — LA LÉVITATION et LA RÉCURRENCE.
//
// ## POURQUOI ON A LE DROIT D'Y REVENIR
//
// Le lot 5 a laissé l'origine non publiable : le juge temporel (fin de vie ti=37 à l'instant du
// ramassage) n'atteint que 25,6 % d'injectivité, et la branche « point d'apparition de la
// carte » n'était pas testable — le dépôt ne déclare AUCUN point d'apparition d'équipement ni de
// grenade. Les deux idées de ce lot attaquent chacune un de ces deux blocages, et elles sont
// GÉOMÉTRIQUES là où le lot 5 était temporel :
//
//	LÉVITATION   un objet POSÉ SUR UN SOCLE flotte au-dessus du sol ; un objet LÂCHÉ repose
//	             dessus. Si la hauteur sépare, elle dit l'origine d'un objet sans avoir à
//	             l'apparier à quoi que ce soit — donc sans souffrir de la non-injectivité.
//	RÉCURRENCE   un point d'apparition SERT PLUSIEURS FOIS dans un match ; un lâcher n'a pas de
//	             raison de se répéter au même endroit. Les amas de naissances construiraient le
//	             catalogue de points d'apparition qui manquait à O3.
//
// ## L'ÉTALON N'EST PAS INVENTÉ : CE SONT LES SOCLES D'ARMES CONNUS
//
// La lévitation ne vaut que si on peut la calibrer sur des objets dont l'origine est CERTAINE.
// Elle l'est pour les ARMES, et par une source extérieure au film : `map_weapon_pads.json`
// donne les emplacements de socle déclarés par le fichier de carte, extraits des `.mvar`
// (précision mesurée au plan SOCLES_MVAR : 32 positions d'oracle sur trois cartes, 32 appariées
// à moins d'un mètre, médiane 0,01 m).
//
//	population SOCLE     objet ti=42 qui REPOSE à moins de 1,0 m d'un emplacement déclaré
//	population LÂCHÉE    objet ti=42 qui repose à plus de 5,0 m de TOUT emplacement déclaré
//
// La bande morte entre 1 et 5 m est écartée à dessein : un objet à 2 m d'un socle est ambigu, et
// une calibration ne se fait pas sur des cas douteux.
//
// ## LA MESURE DE LA HAUTEUR, ET POURQUOI ELLE N'A PAS BESOIN DE CONNAÎTRE LE SOL
//
// On ne sait pas où est le sol, et on n'a pas besoin de le savoir : la référence est la hauteur
// des BIPÈDES qui passent au même endroit. Que `biped.Z` désigne les pieds, le nombril ou les
// yeux est indifférent — c'est un décalage CONSTANT, qui se simplifie dès qu'on COMPARE deux
// populations d'objets à la même référence. C'est la différence entre les deux médianes qui est
// lue, jamais une hauteur absolue.
//
//	levitation(objet) = objet.Z - mediane{ bipede.Z : bipede a moins de 1,5 m en XY }
//
// Un objet sans assez de bipèdes passés à proximité (moins de 5 relevés) est ÉCARTÉ, pas
// approximé.
//
// ## SEUILS ET TÉMOINS, ÉCRITS AVANT LA MESURE
//
//	V1 — SÉPARATION : mediane(socle) - mediane(lâchée) >= 0,50 m. C'est la hauteur d'un socle
//	     d'arme dans le jeu, à vue de nez basse ; sous ce seuil la lévitation ne discrimine pas.
//	V2 — TÉMOIN D'ÉTIQUETTES PERMUTÉES : les deux populations fusionnées puis re-scindées au
//	     hasard aux mêmes effectifs doivent rendre une séparation < 0,15 m. Sinon V1 mesure la
//	     dispersion, pas l'origine.
//	V3 — POUVOIR DE CLASSEMENT : au seuil à mi-chemin des deux médianes, >= 70 % de bien classés
//	     DANS CHAQUE population (et pas seulement en moyenne — une population à 95 % et l'autre
//	     à 40 % ferait une moyenne flatteuse et un classifieur inutile).
//	V4 — si V1 échoue, la lévitation est RÉFUTÉE comme discriminant et publiée comme telle.
//
//	R1 — RÉCURRENCE : au moins un amas de >= 3 naissances ti=37 dans un rayon de 1,0 m.
//	R2 — TÉMOIN UNIFORME : le même nombre de points tirés dans les bornes de la carte donne un
//	     nombre d'amas de hasard ; le réel doit le dépasser d'un facteur >= 3.
//	R3 — CONTRASTE PAR FAMILLE, et c'est LE juge : les grenades naissent COLLÉES à un bipède
//	     (mesuré au plan NOMMAGE_EQIP : 96-100 % à moins de 3 m d'un poseur), donc partout où on
//	     joue. Si elles forment autant d'amas que le reste, l'amas mesure le TRAFIC DES JOUEURS
//	     et non un point d'apparition. Sans ce contraste, R1 et R2 ne prouvent rien.
//	R4 — CROISEMENT CARTE : un amas candidat tombe-t-il sur un emplacement déclaré ?
//
// Gardes PICKUP_FILM + PICKUP_MAP (celles de `glResolve`). Un film par process, lecture seule,
// AUCUNE cuisson d'artefact.

import (
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	// levRayonSol est le rayon XY dans lequel on cherche les bipèdes qui servent de référence.
	levRayonSol = 1.5
	// levMinReleves est le nombre minimal de relevés de bipède pour qu'une référence de sol
	// soit retenue. En dessous : on écarte l'objet, on ne l'approxime pas.
	levMinReleves = 5
	// levProcheSocle / levLoinSocle bornent les deux populations d'étalonnage, en mètres.
	levProcheSocle, levLoinSocle = 1.0, 5.0
	// levSeuilSeparation est le barreau de V1, en mètres.
	levSeuilSeparation = 0.50
	// levSeuilTemoin est le plafond de V2, en mètres.
	levSeuilTemoin = 0.15
	// levSeuilClassement est le barreau de V3, en pourcentage, exigé DANS CHAQUE population.
	levSeuilClassement = 70.0
	// recRayonAmas est le rayon d'un amas de naissances, en mètres, et recMinAmas sa taille
	// minimale.
	recRayonAmas, recMinAmas = 1.0, 3
	// recFacteurTemoin est le facteur exigé entre le réel et le témoin uniforme (R2).
	recFacteurTemoin = 3.0
)

// levMedian rend la médiane d'un échantillon (copie, ne modifie pas l'entrée).
func levMedian(v []float64) float64 {
	if len(v) == 0 {
		return math.NaN()
	}
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	return s[len(s)/2]
}

// levGroundZ rend la hauteur de référence du sol en (x, y) : la médiane des `Z` des bipèdes
// passés à moins de levRayonSol, et le nombre de relevés qui la fondent.
func levGroundZ(pos map[uint32][]filmdec.BipedPosition, x, y float32) (float64, int) {
	var zs []float64
	for _, list := range pos {
		for _, p := range list {
			dx, dy := float64(p.X-x), float64(p.Y-y)
			if dx*dx+dy*dy <= levRayonSol*levRayonSol {
				zs = append(zs, float64(p.Z))
			}
		}
	}
	return levMedian(zs), len(zs)
}

// levSocles charge les emplacements de socle déclarés par le fichier de carte.
//
// LA RÉSOLUTION DE CARTE EXIGE UNE CORRESPONDANCE UNIQUE (nom public ou module) et s'abstient
// sinon : choisir parmi plusieurs candidats serait l'ajustement que le lot 4 a mesuré comme
// destructeur (une mauvaise carte fait tomber le rapport de 7,2x à 1,8x).
func levSocles(t *testing.T) ([]MapWeaponPadSpot, string) {
	t.Helper()
	nom := os.Getenv("PICKUP_MAP")
	cat, err := LoadMapWeaponPads(filepath.Join("..", "..", "..", "..", "..", "data", "titles",
		"halo_infinite", "reference", "map_weapon_pads.json"))
	if err != nil {
		t.Fatalf("catalogue des socles illisible : %v", err)
	}
	// EXACT D'ABORD, APPROCHANT ENSUITE, ET L'UNICITÉ EXIGÉE DANS LES DEUX CAS.
	//
	// Le catalogue porte parfois deux entrées pour une même carte (`catalyst` et
	// `catalyst_map`, 11 emplacements chacune) : une correspondance approchante en trouve deux
	// et devrait s'abstenir, alors que le nom EXACT n'en désigne qu'une. La règle est écrite
	// avant toute mesure, et l'entrée exacte est celle que le journal de cuisson du dépôt
	// enregistre pour ces films (`module=catalyst`) — ce n'est pas un choix fait sur le
	// résultat. En cas d'ambiguïté persistante : abstention, jamais un tirage.
	besoin := strings.ToLower(nom)
	var exactes, approchantes []MapWeaponPadsEntry
	for _, e := range cat.Maps {
		switch {
		case strings.EqualFold(e.PublicName, nom) || strings.EqualFold(e.Module, nom):
			exactes = append(exactes, e)
		case strings.Contains(strings.ToLower(e.Module), besoin):
			approchantes = append(approchantes, e)
		}
	}
	trouvees := exactes
	if len(trouvees) == 0 {
		trouvees = approchantes
	}
	if len(trouvees) != 1 {
		t.Skipf("la carte %q donne %d entrée(s) exacte(s) et %d approchante(s) dans le catalogue "+
			"des socles — abstention", nom, len(exactes), len(approchantes))
	}
	return trouvees[0].Pads, trouvees[0].Module
}

// levDistSocle rend la distance au socle déclaré le plus proche, en 3D.
func levDistSocle(socles []MapWeaponPadSpot, x, y, z float32) float64 {
	best := math.MaxFloat64
	for _, s := range socles {
		if d := glDist(x, y, z, float32(s.Pos.X), float32(s.Pos.Y), float32(s.Pos.Z)); d < best {
			best = d
		}
	}
	return best
}

// TestOriginLevitationCalibratedOnKnownWeaponPads — LA LÉVITATION, étalonnée sur les socles
// d'armes connus, puis appliquée à l'équipement.
func TestOriginLevitationCalibratedOnKnownWeaponPads(t *testing.T) {
	s := glResolve(t)
	socles, module := levSocles(t)
	_, pst := decodeFilmPlacementsDir(s.dir, &s.wr)
	scans := decodeFilmPadScansDir(s.dir, &s.wr, pst.Calibration.Widths)
	if !scans.Weapons.Scanned || len(scans.Weapons.Tracks) == 0 {
		t.Fatalf("chaîne des armes au sol muette : scanned=%v pistes=%d",
			scans.Weapons.Scanned, len(scans.Weapons.Tracks))
	}
	armes := eqlLivesFromScan(scans.Weapons)
	t.Logf("== LOT 6 · LA LÉVITATION · %s ==", s.dir)
	t.Logf("carte %q : %d emplacement(s) déclaré(s) · vies ti=42 : %d · calibration MPP : %s",
		module, len(socles), len(armes), pst.Calibration)

	// ÉTALONNAGE — les deux populations d'armes, séparées par la CARTE et non par le film.
	var surSocle, lachees []float64
	ecartes := 0
	for _, l := range armes {
		g, n := levGroundZ(s.pos, l.x, l.y)
		if n < levMinReleves {
			ecartes++
			continue
		}
		lev := float64(l.z) - g
		switch d := levDistSocle(socles, l.x, l.y, l.z); {
		case d <= levProcheSocle:
			surSocle = append(surSocle, lev)
		case d > levLoinSocle:
			lachees = append(lachees, lev)
		}
	}
	t.Logf("étalonnage : %d sur socle · %d lâchées · %d écartées (moins de %d relevés de bipède)",
		len(surSocle), len(lachees), ecartes, levMinReleves)
	if len(surSocle) < 3 || len(lachees) < 3 {
		t.Logf("VERDICT : dénominateurs trop faibles pour étalonner (%d / %d) — rien à conclure "+
			"sur ce film, et on ne baisse pas le seuil", len(surSocle), len(lachees))
		return
	}
	mSocle, mLache := levMedian(surSocle), levMedian(lachees)
	sep := mSocle - mLache
	t.Logf("LÉVITATION médiane — SUR SOCLE %.2f m (n=%d) · LÂCHÉE %.2f m (n=%d) · SÉPARATION %.2f m",
		mSocle, len(surSocle), mLache, len(lachees), sep)

	// V2 — TÉMOIN d'étiquettes permutées, moyenné sur 200 tirages pour ne pas juger sur un seul.
	tout := append(append([]float64(nil), surSocle...), lachees...)
	rng := rand.New(rand.NewSource(20260901))
	var pireTemoin float64
	for i := 0; i < 200; i++ {
		rng.Shuffle(len(tout), func(a, b int) { tout[a], tout[b] = tout[b], tout[a] })
		d := math.Abs(levMedian(tout[:len(surSocle)]) - levMedian(tout[len(surSocle):]))
		if d > pireTemoin {
			pireTemoin = d
		}
	}
	t.Logf("TÉMOIN étiquettes permutées (pire de 200 tirages) : %.2f m", pireTemoin)

	// V3 — pouvoir de classement au seuil à mi-chemin, DANS CHAQUE population.
	seuil := (mSocle + mLache) / 2
	bonSocle, bonLache := 0, 0
	for _, v := range surSocle {
		if v >= seuil {
			bonSocle++
		}
	}
	for _, v := range lachees {
		if v < seuil {
			bonLache++
		}
	}
	pSocle, pLache := pct100(bonSocle, len(surSocle)), pct100(bonLache, len(lachees))
	t.Logf("CLASSEMENT au seuil %.2f m : socle %d/%d = %.1f %% · lâchée %d/%d = %.1f %%",
		seuil, bonSocle, len(surSocle), pSocle, bonLache, len(lachees), pLache)

	t.Logf("VERDICT V1 (séparation >= %.2f m) : %v", levSeuilSeparation, sep >= levSeuilSeparation)
	t.Logf("VERDICT V2 (témoin permuté < %.2f m) : %v", levSeuilTemoin, pireTemoin < levSeuilTemoin)
	t.Logf("VERDICT V3 (>= %.0f %% dans CHAQUE population) : %v",
		levSeuilClassement, pSocle >= levSeuilClassement && pLache >= levSeuilClassement)
	t.Logf("VERDICT V4 (lévitation RÉFUTÉE comme discriminant) : %v", sep < levSeuilSeparation)

	// APPLICATION À L'ÉQUIPEMENT — seulement si l'étalon tient. Appliquer un seuil non calibré
	// à une population inconnue produirait un chiffre sans signification.
	if sep < levSeuilSeparation {
		t.Log("APPLICATION à l'équipement : NON FAITE — l'étalon ne sépare pas, un seuil non " +
			"calibré rendrait un pourcentage qui ne veut rien dire.")
		return
	}
	if !scans.Powerups.Scanned {
		t.Log("APPLICATION : chaîne ti=37 muette sur ce film")
		return
	}
	eq := eqlLivesFromScan(scans.Powerups)
	flottants, poses, sansRef := 0, 0, 0
	for _, l := range eq {
		g, n := levGroundZ(s.pos, l.x, l.y)
		if n < levMinReleves {
			sansRef++
			continue
		}
		if float64(l.z)-g >= seuil {
			flottants++
		} else {
			poses++
		}
	}
	t.Logf("APPLICATION ti=37 (%d vies) : %d au-dessus du seuil (socle ?) · %d en dessous (posé/lâché) · %d sans référence",
		len(eq), flottants, poses, sansRef)
}

// TestOriginBirthRecurrenceClusters — LA RÉCURRENCE des positions de naissance.
//
// L'IDÉE : un point d'apparition sert plusieurs fois dans un match, un lâcher n'a pas de raison
// de se répéter au même endroit. LE PIÈGE, et c'est R3 qui le lève : les joueurs eux-mêmes se
// répètent (couloirs, rampes, zones d'objectif), donc des amas apparaîtront de toute façon. Le
// juge n'est pas « y a-t-il des amas » mais « les familles qui NE PEUVENT PAS avoir de point
// d'apparition en font-elles autant ».
func TestOriginBirthRecurrenceClusters(t *testing.T) {
	s := glResolve(t)
	socles, module := levSocles(t)
	poses, pst := decodeFilmPlacementsDir(s.dir, &s.wr)
	if len(poses) == 0 {
		t.Skip("aucune pose ti=37 sur ce film")
	}
	familles := goldenReplayLabels(t).EquipmentObjects()
	t.Logf("== LOT 6 · LA RÉCURRENCE DES NAISSANCES · %s ==", s.dir)
	t.Logf("carte %q · %d pose(s) ti=37 · calibration MPP : %s", module, len(poses), pst.Calibration)

	// Regroupement par nature, pour R3. Le préfixe `grenade_` est la convention du manifeste.
	parNature := map[string][]filmdec.EquipmentPlacement{}
	for _, p := range poses {
		nature := "equipement"
		fam, ok := familles[p.GlobalID]
		switch {
		case !ok:
			nature = "inconnu"
		case strings.HasPrefix(fam, "grenade_"):
			nature = "grenade"
		}
		parNature[nature] = append(parNature[nature], p)
	}

	natures := make([]string, 0, len(parNature))
	for n := range parNature {
		natures = append(natures, n)
	}
	sort.Strings(natures)
	total, totalTemoin := 0, 0
	for _, n := range natures {
		list := parNature[n]
		amas := recAmas(list)
		tem := recAmasTemoin(list, s.wr, 20260901)
		total += len(amas)
		totalTemoin += tem
		t.Logf("  %-12s : %3d naissance(s) · %d amas de >= %d dans %.1f m · TÉMOIN uniforme %d amas",
			n, len(list), len(amas), recMinAmas, recRayonAmas, tem)
		for _, a := range amas {
			d := levDistSocle(socles, a.x, a.y, a.z)
			t.Logf("      amas n=%-2d  (%.1f, %.1f, %.1f)  socle déclaré le plus proche : %.2f m", a.n, a.x, a.y, a.z, d)
		}
	}
	eq, gr := len(recAmas(parNature["equipement"])), len(recAmas(parNature["grenade"]))
	t.Logf("VERDICT R1 (au moins un amas) : %v — %d au total", total > 0, total)
	t.Logf("VERDICT R2 (réel >= %.0fx le témoin uniforme) : %v — réel %d · témoin %d",
		recFacteurTemoin, float64(total) >= recFacteurTemoin*float64(totalTemoin), total, totalTemoin)
	t.Logf("VERDICT R3 (CONTRASTE : l'équipement s'amasse plus que les grenades) : %v — équipement %d · grenades %d",
		eq > gr, eq, gr)
}

// recAmasPoint est un amas de naissances : son centre et son effectif.
type recAmasPoint struct {
	x, y, z float32
	n       int
}

// recAmas regroupe les naissances par proximité (agglomération gloutonne autour du point le
// plus dense) et rend les amas d'au moins recMinAmas naissances.
//
// GLOUTON ET NON ITÉRATIF, À DESSEIN : on cherche à savoir S'IL EXISTE des points réutilisés,
// pas à produire un partitionnement optimal. Un algorithme plus fin ne changerait pas le verdict
// et ajouterait des paramètres à justifier.
func recAmas(poses []filmdec.EquipmentPlacement) []recAmasPoint {
	reste := append([]filmdec.EquipmentPlacement(nil), poses...)
	var out []recAmasPoint
	for len(reste) > 0 {
		meilleur, meilleurN := 0, 0
		for i, p := range reste {
			n := 0
			for _, q := range reste {
				if glDist(p.X, p.Y, p.Z, q.X, q.Y, q.Z) <= recRayonAmas {
					n++
				}
			}
			if n > meilleurN {
				meilleur, meilleurN = i, n
			}
		}
		centre := reste[meilleur]
		var garde []filmdec.EquipmentPlacement
		for _, q := range reste {
			if glDist(centre.X, centre.Y, centre.Z, q.X, q.Y, q.Z) > recRayonAmas {
				garde = append(garde, q)
			}
		}
		if meilleurN >= recMinAmas {
			out = append(out, recAmasPoint{x: centre.X, y: centre.Y, z: centre.Z, n: meilleurN})
		}
		if len(garde) == len(reste) { // sécurité : aucun retrait, on s'arrête
			break
		}
		reste = garde
	}
	return out
}

// recAmasTemoin rend le nombre d'amas qu'on obtient en tirant le MÊME nombre de points
// uniformément dans les bornes de la carte — le plancher du hasard pour R2.
func recAmasTemoin(poses []filmdec.EquipmentPlacement, wr filmdec.Vec3Range, graine int64) int {
	if len(poses) == 0 {
		return 0
	}
	rng := rand.New(rand.NewSource(graine))
	faux := make([]filmdec.EquipmentPlacement, len(poses))
	for i := range faux {
		faux[i] = filmdec.EquipmentPlacement{
			X: recEntre(wr[0], rng),
			Y: recEntre(wr[1], rng),
			Z: recEntre(wr[2], rng),
		}
	}
	return len(recAmas(faux))
}

// recEntre tire une coordonnée uniformément dans les bornes d'un axe de la carte.
func recEntre(a filmdec.AxisRange, rng *rand.Rand) float32 {
	return a.Min + float32(rng.Float64())*(a.Max-a.Min)
}
