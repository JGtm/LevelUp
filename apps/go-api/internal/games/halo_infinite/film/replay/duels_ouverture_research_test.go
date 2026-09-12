package replay

// duels_ouverture_research_test.go — LA VALIDATION DU PROXY D'ENTAME (D5, item 2.5 du plan
// `.ai/PLAN_DUELS_PORTEE_2026-09-06.md`).
//
// # CE QUE CET INSTRUMENT TRANCHE
//
// Le produit veut publier une DISTANCE D'ENTAME : à quelle distance l'engagement a commencé,
// et non pas seulement où le coup fatal est tombé. La vraie ouverture exigerait le PREMIER
// dégât de l'échange — que le film ne porte que pour une minorité des morts (sonde n°1,
// 2026-09-06 : 91 à 428 enregistrements de dégât pour 90 à 117 morts). Le proxy retenu est
// un décalage d'horloge sur les trajectoires, denses et continues : la distance à
// T − `OpeningLeadMS`, un temps-pour-tuer avant la fin de vie.
//
// LA QUESTION EST DONC : ce décalage d'horloge dit-il la même chose que l'événement, LÀ OÙ
// L'ÉVÉNEMENT EXISTE ? C'est la seule validation possible d'un proxy — on le confronte à la
// vérité sur le sous-ensemble où la vérité est lisible, et on suppose que ce sous-ensemble
// n'est pas particulier. (Réserve assumée et écrite : les morts dont le premier dégât est
// capturé sont peut-être les échanges les plus longs, donc les mieux échantillonnés ; le
// biais irait dans le sens d'une validation OPTIMISTE.)
//
// # LE SEUIL EST ÉCRIT AVANT LA MESURE, ET IL EST TENU PAR LE CODE
//
//	GATE : écart médian entre les deux distances <= 2 m, à T − OpeningLeadMS.
//
// Deux mètres, parce que c'est l'ordre de grandeur d'un pas de côté : en dessous, les deux
// mesures décrivent la même situation tactique ; au-delà, le proxy raconterait un autre
// moment du combat. Le test ÉCHOUE si le gate n'est pas tenu — un gate qu'on lit dans un log
// est un gate qu'on ajuste au résultat.
//
// LA SENSIBILITÉ EST MESURÉE EN MÊME TEMPS (1,0 s et 2,0 s) : si l'écart s'effondrait à 1,0 s
// et explosait à 2,0 s, la valeur retenue serait un réglage et non un temps-pour-tuer, et
// c'est exactement ce qu'il faut savoir avant de la persister dans une table.
//
// # CE QU'IL NE FAIT PAS
//
// Aucune base, aucun roster, aucune cuisson d'artefact, aucun réseau. Un film par process
// (verrou `filmdec.LockProcessDecode`), comme la sonde n°1 dont il réutilise TOUS les
// helpers — la population de morts, la calibration de base, la distance : deux lectures
// différentes des mêmes films ne se compareraient pas.
//
// USAGE :
//
//	CGO_ENABLED=0 \
//	  DUELS_FILM=<repo>/data/cache/film_chunks/000d5950 DUELS_MAP=Cliffhanger \
//	  go test ./internal/games/halo_infinite/film/replay -run TestSondeDuelsOuverture -v -timeout 900s

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmsource"
	"levelup/go-api/internal/games/halo_infinite/film/filmdec"
)

// ouvEcartMaxMedianM : LE GATE. Écrit avant la mesure, en mètres.
const ouvEcartMaxMedianM = 2.0

// ouvFenetreEchangeUS : fenêtre amont dans laquelle on cherche le premier dégât de l'échange.
// MÊME valeur que la mesure M3 de la sonde n°1 (3 s), pour que les deux populations de morts
// soient comparables d'une note à l'autre.
const ouvFenetreEchangeUS = 3_000_000

// ouvAvancesMS : les trois avances mesurées. Celle du milieu est `OpeningLeadMS`, la valeur
// du produit ; les deux autres sont là pour la sensibilité, pas pour être choisies après coup.
var ouvAvancesMS = []int64{1_000, OpeningLeadMS, 2_000}

// ouvCas est un couple (mort, tueur) dont le premier dégât de l'échange EST capturé : la
// population sur laquelle le proxy se valide.
type ouvCas struct {
	victime, tueur uint32
	// finVieUS est T, la fin de vie ; echangeUS est l'instant du premier dégât de l'échange.
	finVieUS, echangeUS uint64
	// distEchangeM est la distance de RÉFÉRENCE, celle de l'événement.
	distEchangeM float64
}

func TestSondeDuelsOuverture(t *testing.T) {
	dir := os.Getenv(duelsFilmEnv)
	carte := os.Getenv(duelsMapEnv)
	if dir == "" || carte == "" {
		t.Skipf("sonde desactivee : %s et %s requis", duelsFilmEnv, duelsMapEnv)
	}

	release := filmdec.LockProcessDecode()
	defer release()

	rng := duelsBornes(t, carte)
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("film %s illisible : %v", dir, err)
	}
	opt := filmdec.DefaultScanFilmOptions()
	opt.WorldRange = &rng
	positions, err := filmdec.ScanBipedPositions(film, opt)
	if err != nil {
		t.Fatalf("positions bipeds : %v", err)
	}
	tracks := indexBySlot(positions)
	lives := buildLifeSpans(tracks)
	morts := duelsMorts(t, film, lives)
	brut, _ := duelsScanDegats(t, dir)
	_, dmg := duelsResoudreBase(t, brut, duelsViesParSlot(lives))

	cas := ouvPopulation(morts, dmg, tracks)
	t.Logf("FILM %s (%s) : %d morts appariees, %d degats resolus", filepath.Base(dir), carte,
		len(morts), len(dmg))
	t.Logf("O0 population de validation : %s des morts ont leur PREMIER degat d'echange capture"+
		" ET la distance resolue a cet instant", duelsPct(len(cas), len(morts)))
	if len(cas) == 0 {
		t.Fatalf("aucun cas de validation sur ce film : le gate ne peut pas etre evalue")
	}
	t.Logf("O0bis distance de reference (au premier degat) : mediane %s",
		duelsMediane(ouvDistancesRef(cas)))
	// O0ter EST LA CLE DE LECTURE DE LA SENSIBILITE. Le premier degat CAPTURE tombe souvent
	// tres pres de la fin de vie : la population de validation est donc biaisee vers les
	// echanges COURTS, et l'ecart au proxy croit mecaniquement avec l'avance. Sans ce nombre,
	// on lirait « 1,0 s est meilleur que 1,5 s » alors qu'on lit « la reference est proche
	// de T dans CETTE population ».
	delais := ouvDelaisMS(cas)
	t.Logf("O0ter delai premier degat -> fin de vie : mediane %.0f ms, p90 %.0f ms (n=%d)",
		gwPadsQuantile(delais, 0.5), gwPadsQuantile(delais, 0.9), len(delais))

	ouvRapporte(t, cas, tracks)
}

// ouvPopulation constitue les cas de validation : une mort dont le dégât fatal est attribué,
// dont le PREMIER dégât de l'échange est capturé, et dont la distance se résout à cet instant.
func ouvPopulation(morts []duelMort, dmg []duelDmg, tracks map[uint32]slotTrack) []ouvCas {
	out := make([]ouvCas, 0, len(morts))
	for _, m := range morts {
		f, ok := duelsFatal(dmg, m.slot, m.ts)
		if !ok {
			continue
		}
		lo := duelsBorneBasse(m.ts, ouvFenetreEchangeUS)
		at, ok := duelsPremierEchange(dmg, m.slot, f.attacker, lo, m.ts)
		if !ok {
			continue // pas d'evenement d'ouverture : rien a quoi comparer le proxy
		}
		d3, _, ok := duelsDistance(tracks, m.slot, f.attacker, at)
		if !ok {
			continue
		}
		out = append(out, ouvCas{
			victime: m.slot, tueur: f.attacker,
			finVieUS: m.ts, echangeUS: at, distEchangeM: d3,
		})
	}
	return out
}

// ouvDistancesRef extrait les distances de référence, pour la médiane de contexte.
func ouvDistancesRef(cas []ouvCas) []float64 {
	out := make([]float64, 0, len(cas))
	for _, c := range cas {
		out = append(out, c.distEchangeM)
	}
	return out
}

// ouvDelaisMS rend, en millisecondes, le delai entre le premier degat de l'echange et la fin
// de vie.
func ouvDelaisMS(cas []ouvCas) []float64 {
	out := make([]float64, 0, len(cas))
	for _, c := range cas {
		out = append(out, float64(c.finVieUS-c.echangeUS)/1000)
	}
	return out
}

// ouvRapporte publie une ligne par avance et TIENT LE GATE sur celle du produit.
func ouvRapporte(t *testing.T, cas []ouvCas, tracks map[uint32]slotTrack) {
	t.Helper()
	for _, lead := range ouvAvancesMS {
		ecarts, proxys := ouvEcarts(cas, tracks, lead)
		if len(ecarts) == 0 {
			t.Errorf("O1 avance %d ms : AUCUN proxy resolu — mesure impossible", lead)
			continue
		}
		med := gwPadsQuantile(ecarts, 0.5)
		p90 := gwPadsQuantile(ecarts, 0.9)
		proches := 0
		for _, e := range ecarts {
			if e < ouvEcartMaxMedianM {
				proches++
			}
		}
		t.Logf("O1 avance %d ms : n = %s | ecart median %.2f m | p90 %.2f m |"+
			" proxy a moins de %.0f m %s | distance proxy mediane %s",
			lead, duelsPct(len(ecarts), len(cas)), med, p90, ouvEcartMaxMedianM,
			duelsPct(proches, len(ecarts)), duelsMediane(proxys))
		if lead != OpeningLeadMS {
			continue // les deux autres avances mesurent la SENSIBILITE, elles ne decident pas
		}
		// LES ECARTS BRUTS, TRIES. Un film par process (verrou de decodage) : aucun agregat
		// des quatre films ne peut se calculer en memoire, et une mediane de medianes n'est
		// pas une mediane. La note de sonde met donc les quatre listes bout a bout.
		t.Logf("O1bis ecarts bruts a %d ms (metres, tries) : %s", lead, ouvListe(ecarts))
		verdict := "GATE TENU"
		if med > ouvEcartMaxMedianM {
			verdict = "GATE ECHOUE"
		}
		t.Logf("O2 %s : ecart median %.2f m contre un seuil de %.0f m ecrit avant la mesure"+
			" (avance de production : %d ms)", verdict, med, ouvEcartMaxMedianM, OpeningLeadMS)
		if med > ouvEcartMaxMedianM {
			t.Errorf("D5 NON VALIDE sur ce film : le proxy d'entame a %d ms s'ecarte de %.2f m"+
				" (median) de la distance au premier degat, pour un seuil de %.0f m."+
				" Le proxy ne doit pas etre publie (item 2.4 du plan a supprimer, D5 a statuer [!])",
				OpeningLeadMS, med, ouvEcartMaxMedianM)
		}
	}
}

// ouvEcarts rend, pour une avance donnée, les écarts absolus |proxy − référence| et les
// distances proxy elles-mêmes. Un cas dont le proxy ne se résout pas (échantillon hors
// tolérance, instant avant l'origine du film) est ÉCARTÉ, pas compté zéro.
func ouvEcarts(cas []ouvCas, tracks map[uint32]slotTrack, leadMS int64) (ecarts, proxys []float64) {
	leadUS := uint64(leadMS) * 1000
	for _, c := range cas {
		if c.finVieUS <= leadUS {
			continue // l'entame tomberait avant l'origine du film : pas de position, pas de mesure
		}
		d3, _, ok := duelsDistance(tracks, c.victime, c.tueur, c.finVieUS-leadUS)
		if !ok {
			continue
		}
		ecarts = append(ecarts, absF(d3-c.distEchangeM))
		proxys = append(proxys, d3)
	}
	return ecarts, proxys
}

// ouvListe formate une serie triee, deux decimales, separee par des espaces.
func ouvListe(v []float64) string {
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	parts := make([]string, 0, len(s))
	for _, x := range s {
		parts = append(parts, fmt.Sprintf("%.2f", x))
	}
	return strings.Join(parts, " ")
}
