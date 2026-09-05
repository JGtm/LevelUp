package replay

// equipment_pickup_origin_research_test.go — LOT 5, ÉTAPE 3 : L'ORIGINE d'un ramassage
// non-arme. Objet tombé au sol (d'un mort, ou posé par un joueur) contre point d'apparition
// de la carte.
//
// ## L'IDÉE À MESURER, ET POURQUOI ELLE N'EST PAS CELLE DU LOT 4
//
// Le lot 4 a jugé le lien par la DISTANCE seule, et il a rendu un résultat frustrant : le
// ramasseur est 7 à 11 fois plus proche d'un objet ti=37 que n'importe quel autre bipède au
// même instant (le lien est donc RÉEL), mais la médiane plafonne à 1,33 m et la part sous le
// mètre à 46,3 % — pas de quoi désigner UN objet. La réfutation a été DÉPLACÉE, pas levée :
// « le lien existe, la résolution spatiale ne suffit pas à le rendre injectif ».
//
// L'idée de ce lot est d'ajouter un second juge, TEMPOREL : un objet qu'on ramasse cesse
// d'exister. Si la VIE d'un objet ti=37 se termine à l'instant exact d'un ramassage, à côté du
// ramasseur, la conjonction des deux devrait désigner un seul objet là où la distance seule en
// désigne plusieurs.
//
// ## LA RÉSERVE, ÉCRITE AVANT LA MESURE PARCE QU'ELLE PEUT TOUT EXPLIQUER
//
// `equipment_placements.go` établit — mesuré, pas supposé — que **la disparition d'un objet
// n'est PAS dans le film**. Trois pistes de fin explicite ont été instrumentées et les trois
// échouent ; ce que le décodage rend comme `tEnd` est le dernier point de la vie décodée,
// c'est-à-dire l'instant où l'objet CESSE DE BOUGER, prolongé par les recensements d'images-
// clés. C'est une BORNE INFÉRIEURE de la durée de vie.
//
// Autrement dit, le juge temporel de ce lot est bâti sur une quantité dont le dépôt sait déjà
// qu'elle n'est pas une fin. La mesure a quand même lieu — c'est la seule façon de savoir si la
// borne inférieure est assez serrée pour trancher — mais un résultat nul ne devra PAS être lu
// comme « le lien n'existe pas » : il se lira « le film ne date pas les fins d'objet ».
//
// ## L'APPORT NEUF DU LOT 5 : la population est enfin SÉPARÉE
//
// Le lot 4 mesurait les 82 ramassages non-arme en bloc. L'étape 1 les a nommés à 100 % : on
// sait désormais que 51 d'entre eux sont des GRENADES (classe 2) et 31 de l'ÉQUIPEMENT
// (classe 3). Or les deux n'ont aucune raison de se comporter pareil — les vies ti=37 de
// grenade sont des grenades LANCÉES, qui explosent et ne se ramassent pas. Mélanger les deux
// populations était une façon sûre de noyer le signal, et c'est peut-être l'explication du
// 1,33 m du lot 4. La mesure est donc VENTILÉE PAR CLASSE, et c'est le seul changement de
// méthode qui compte.
//
// ## SEUILS ÉCRITS AVANT LA MESURE
//
//	O1 — INJECTIVITÉ. Au moins 50 % des ramassages non-arme reçoivent EXACTEMENT UN candidat
//	     (une vie ti=37 finissant dans la fenêtre, à portée), contre un témoin décalé sous
//	     15 %. C'est le seuil qui autoriserait à publier une origine.
//	O2 — GAIN SUR LE JUGE SPATIAL. La part de cas AMBIGUS (plusieurs candidats) doit être
//	     strictement plus basse avec le juge temporel qu'avec la distance seule au même rayon.
//	     Sans ce gain, le second juge n'apporte rien et il faut le dire.
//	O3 — POINT D'APPARITION DE LA CARTE. Part des ramassages non-arme à portée d'un socle
//	     `powerup` du catalogue de carte, avec témoin décalé. PUBLIÉ COMME INDICATIF : le
//	     catalogue de carte du dépôt ne déclare que `power`, `rack` et `powerup` — il n'existe
//	     AUCUN point d'apparition d'équipement ni de grenade dans les données du dépôt. La
//	     branche « point d'apparition » ne peut donc pas être testée en propre, et c'est un
//	     manque de données, pas un résultat.
//
// Gardes PICKUP_FILM + PICKUP_MAP (celles de `glResolve`). Recherche pure.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// epoFenetreUS est la fenêtre du juge temporel : la fin de vie doit tomber à moins de 500 ms
// de l'instant du ramassage. Même ordre de grandeur que toutes les fenêtres d'appariement du
// chantier, et le pas d'échantillonnage des positions est de 100 ms.
const epoFenetreUS = 500_000

// epoRayon est la portée du juge spatial, en mètres. 3 m est LARGE à dessein : le but du juge
// temporel est de désambiguïser à l'intérieur d'un rayon généreux, pas de se substituer à un
// rayon serré. Un rayon serré ferait le tri tout seul et masquerait ce qu'on mesure.
const epoRayon = 3.0

// epoClasseNom nomme une classe pour la sortie, avec ce que l'étape 1 en a établi.
func epoClasseNom(c uint8) string {
	switch c {
	case 2:
		return "classe 2 (GRENADES)"
	case 3:
		return "classe 3 (ÉQUIPEMENT)"
	default:
		return "classe ?"
	}
}

// epoBilan compte la ventilation d'un juge : combien de ramassages reçoivent zéro, un, ou
// plusieurs candidats.
type epoBilan struct{ zero, un, plusieurs int }

func (b epoBilan) total() int { return b.zero + b.un + b.plusieurs }

func (b epoBilan) String() string {
	n := b.total()
	if n == 0 {
		return "n=0"
	}
	return fmt.Sprintf("n=%d · UN SEUL candidat %d (%.1f %%) · plusieurs %d (%.1f %%) · aucun %d (%.1f %%)",
		n, b.un, pct100(b.un, n), b.plusieurs, pct100(b.plusieurs, n), b.zero, pct100(b.zero, n))
}

// TestEquipmentPickupOrigin — ÉTAPE 3. Le juge temporel (fin de vie == instant de la prise)
// est-il injectif là où la distance seule ne l'est pas ?
func TestEquipmentPickupOrigin(t *testing.T) {
	s := glResolve(t)
	pickups, _, err := filmdec.ScanFilmBipedPickups(s.dir)
	if err != nil {
		t.Fatalf("ramassages natifs illisibles : %v", err)
	}
	// LA CHAÎNE DE PRODUCTION, APPELÉE — jamais recopiée (leçon du lot 4).
	_, pst := decodeFilmPlacementsDir(s.dir, &s.wr)
	scan := decodeFilmPadScansDir(s.dir, &s.wr, pst.Calibration.Widths).Powerups
	if !scan.Scanned || len(scan.Tracks) == 0 {
		t.Fatalf("chaîne des power-ups muette : scanned=%v pistes=%d", scan.Scanned, len(scan.Tracks))
	}
	lives := eqlLivesFromScan(scan)
	familles := goldenReplayLabels(t).EquipmentObjects()
	t.Logf("== ÉTAPE 3 — ORIGINE : SOL OU POINT D'APPARITION · %s ==", s.dir)
	t.Logf("ramassages natifs : %d · vies ti=37 : %d sur %d pistes · calibration MPP : %s · fenêtre %d ms · rayon %.1f m",
		len(pickups), len(lives), len(scan.Tracks), pst.Calibration, epoFenetreUS/1000, epoRayon)

	// Deux juges, la même population, comptés côte à côte.
	temporel := map[uint8]*epoBilan{}
	spatial := map[uint8]*epoBilan{}
	temoin := map[int64]*epoBilan{}
	sansPos := 0
	for _, p := range pickups {
		if filmdec.BipedPickupIsWeaponClass(p.Class) {
			continue
		}
		pos, ok := glAt(s.pos, p.Slot, p.TimestampUS)
		if !ok {
			sansPos++
			continue
		}
		epoAjoute(temporel, p.Class, epoCandidatsTemporels(lives, pos, p.TimestampUS))
		epoAjoute(spatial, p.Class, epoCandidatsSpatiaux(lives, pos, p.TimestampUS))
		// TÉMOIN — les mêmes juges, aux instants décalés. Si le juge temporel désigne un
		// objet aussi souvent 37 secondes plus tard, il ne mesure que la densité des fins.
		for _, dec := range eqnDecalages {
			at := int64(p.TimestampUS) + dec
			if at < 0 {
				continue
			}
			q, ok := glAt(s.pos, p.Slot, uint64(at))
			if !ok {
				continue
			}
			if temoin[dec] == nil {
				temoin[dec] = &epoBilan{}
			}
			epoCompte(temoin[dec], epoCandidatsTemporels(lives, q, uint64(at)))
		}
	}
	t.Logf("écartés : %d ramassage(s) sans position du ramasseur", sansPos)

	classes := []uint8{2, 3}
	totT, totS := epoBilan{}, epoBilan{}
	for _, c := range classes {
		if b := temporel[c]; b != nil {
			t.Logf("JUGE TEMPOREL · %-22s %s", epoClasseNom(c), b)
			totT.zero, totT.un, totT.plusieurs = totT.zero+b.zero, totT.un+b.un, totT.plusieurs+b.plusieurs
		}
		if b := spatial[c]; b != nil {
			t.Logf("JUGE SPATIAL   · %-22s %s", epoClasseNom(c), b)
			totS.zero, totS.un, totS.plusieurs = totS.zero+b.zero, totS.un+b.un, totS.plusieurs+b.plusieurs
		}
	}
	t.Logf("JUGE TEMPOREL · TOUTES CLASSES  %s", totT)
	t.Logf("JUGE SPATIAL   · TOUTES CLASSES  %s", totS)

	pireTemoin := 0.0
	for _, dec := range eqnDecalages {
		b := temoin[dec]
		if b == nil {
			continue
		}
		r := pct100(b.un, b.total())
		t.Logf("TÉMOIN décalé de %+d s · juge temporel · %s", dec/1_000_000, b)
		if r > pireTemoin {
			pireTemoin = r
		}
	}

	nT := totT.total()
	t.Logf("VERDICT O1 (>= 50 %% avec UN SEUL candidat temporel, témoin < 15 %%) : %v — réel %.1f %% · pire témoin %.1f %%",
		pct100(totT.un, nT) >= 50 && pireTemoin < 15, pct100(totT.un, nT), pireTemoin)
	t.Logf("VERDICT O2 (le juge temporel désambiguïse : moins d'ambigus que le spatial seul) : %v — temporel %.1f %% · spatial %.1f %%",
		pct100(totT.plusieurs, nT) < pct100(totS.plusieurs, totS.total()),
		pct100(totT.plusieurs, nT), pct100(totS.plusieurs, totS.total()))

	epoSoclesDeCarte(t, s, pickups, familles)
}

// epoCandidatsTemporels compte les vies ti=37 dont la FIN tombe dans la fenêtre autour de `at`
// ET qui reposent à portée du ramasseur. C'est la conjonction des deux juges.
func epoCandidatsTemporels(lives []eqlLife, p filmdec.BipedPosition, at uint64) int {
	n := 0
	for _, l := range lives {
		if epoEcart(l.tEnd, at) > epoFenetreUS {
			continue
		}
		if glDist(p.X, p.Y, p.Z, l.x, l.y, l.z) <= epoRayon {
			n++
		}
	}
	return n
}

// epoCandidatsSpatiaux compte les vies VIVANTES à l'instant `at` et à portée — le juge du
// lot 4, reproduit ici au même rayon pour que la comparaison soit lisible.
func epoCandidatsSpatiaux(lives []eqlLife, p filmdec.BipedPosition, at uint64) int {
	n := 0
	for _, l := range lives {
		if at < l.t0 || at > l.tEnd {
			continue
		}
		if glDist(p.X, p.Y, p.Z, l.x, l.y, l.z) <= epoRayon {
			n++
		}
	}
	return n
}

// epoEcart rend la distance absolue entre deux instants.
func epoEcart(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return b - a
}

// epoAjoute range un compte de candidats dans le bilan de sa classe.
func epoAjoute(m map[uint8]*epoBilan, c uint8, n int) {
	if m[c] == nil {
		m[c] = &epoBilan{}
	}
	epoCompte(m[c], n)
}

// epoCompte incrémente la case du bilan qui correspond au nombre de candidats.
func epoCompte(b *epoBilan, n int) {
	switch {
	case n == 0:
		b.zero++
	case n == 1:
		b.un++
	default:
		b.plusieurs++
	}
}

// epoSoclesDeCarte — O3. La branche « point d'apparition de la carte », telle que les données
// du dépôt permettent de la tester, c'est-à-dire très partiellement.
//
// CE QUI MANQUE, ET IL FAUT LE DIRE AVANT LE CHIFFRE : `map_weapon_pads.json` ne déclare que
// trois familles d'emplacement — `power` (arme de pouvoir), `rack` (arme de râtelier) et
// `powerup`. Il n'existe dans le dépôt AUCUN catalogue de points d'apparition d'équipement ni
// de grenade. La question « la prise vient-elle d'un point d'apparition ? » n'est donc pas
// testable en propre : ce qui suit ne mesure que la proximité aux socles de POWER-UP, et un
// taux nul y sera un manque de données, pas une réfutation.
func epoSoclesDeCarte(t *testing.T, s glSetup, pickups []filmdec.BipedPickup, familles map[uint32]string) {
	t.Helper()
	nom := os.Getenv("PICKUP_MAP")
	cat, err := LoadMapWeaponPads(filepath.Join("..", "..", "..", "..", "..", "data", "titles",
		"halo_infinite", "reference", "map_weapon_pads.json"))
	if err != nil {
		t.Logf("O3 : catalogue des socles illisible (%v) — branche non mesurée", err)
		return
	}
	// LA RÉSOLUTION DE CARTE EST EXIGEANTE À DESSEIN. Le nom public est vide sur presque toutes
	// les entrées ; c'est le `module` qui porte le nom du niveau. On accepte donc une
	// correspondance sur l'un ou l'autre, MAIS on exige qu'elle soit UNIQUE : plusieurs
	// candidats = abstention. Choisir parmi eux serait exactement l'ajustement que le lot 4 a
	// mesuré comme destructeur (une mauvaise carte fait tomber le rapport de 7,2x à 1,8x).
	var trouvees []MapWeaponPadsEntry
	besoin := strings.ToLower(nom)
	for _, e := range cat.Maps {
		if strings.EqualFold(e.PublicName, nom) || strings.Contains(strings.ToLower(e.Module), besoin) {
			trouvees = append(trouvees, e)
		}
	}
	if len(trouvees) != 1 {
		t.Logf("O3 : la carte %q donne %d entrée(s) dans le catalogue des socles — abstention (une seule est exigée)",
			nom, len(trouvees))
		return
	}
	trouvee := trouvees[0].Module
	var socles []MapWeaponPadSpot
	for _, p := range trouvees[0].Pads {
		if p.Family == "powerup" {
			socles = append(socles, p)
		}
	}
	t.Logf("O3 · carte %q : %d socle(s) de famille `powerup` dans le catalogue", trouvee, len(socles))
	if len(socles) == 0 {
		t.Log("O3 : aucun socle de power-up sur cette carte — rien à mesurer, et c'est un manque de données, pas un négatif")
		return
	}
	pres, presTemoin, n := 0, 0, 0
	for _, p := range pickups {
		if filmdec.BipedPickupIsWeaponClass(p.Class) {
			continue
		}
		pos, ok := glAt(s.pos, p.Slot, p.TimestampUS)
		if !ok {
			continue
		}
		n++
		if epoPresDunSocle(socles, pos) {
			pres++
		}
		for _, dec := range eqnDecalages {
			at := int64(p.TimestampUS) + dec
			if at < 0 {
				continue
			}
			if q, ok := glAt(s.pos, p.Slot, uint64(at)); ok && epoPresDunSocle(socles, q) {
				presTemoin++
				break
			}
		}
	}
	t.Logf("O3 · ramassages non-arme à moins de %.1f m d'un socle `powerup` : %d/%d (%.1f %%) · TÉMOIN décalé %d (%.1f %%)",
		epoRayon, pres, n, pct100(pres, n), presTemoin, pct100(presTemoin, n))
	_ = familles
}

// epoPresDunSocle dit si une position est à portée de l'un des socles.
func epoPresDunSocle(socles []MapWeaponPadSpot, p filmdec.BipedPosition) bool {
	for _, s := range socles {
		if glDist(p.X, p.Y, p.Z, float32(s.Pos.X), float32(s.Pos.Y), float32(s.Pos.Z)) <= epoRayon {
			return true
		}
	}
	return false
}
