package replay

// origin_research_test.go — INSTRUMENT DE MESURE : L'ORIGINE DE L'ARTEFACT (lot 7.2).
//
// CE QU'IL MESURE, ET POURQUOI. La frame 0 du document est calee sur le PREMIER PAQUET DE
// POSITION du film (`build.go` : origin = sorted[0].TimestampUS), un referentiel qui
// n'existe nulle part ailleurs. Le fil des eliminations, lui, vit sur l'horloge des
// highlight events, que le client reconstruit par `event_time_ms + t0_ms`. Cet instrument
// mesure L'ORIGINE par DEUX chemins qui ne partagent aucune piece :
//
//	lecture   (premier paquet de position − premier paquet du chunk 1) / 1000
//	temoin    premier paquet de position / 1000 − `bestDeathOffset` (appariement du fil
//	          des morts aux fins de vie, deja en production pour nommer les vies)
//
// C'est la LECTURE que l'artefact publie (cf. origin.go) ; le temoin la contredit ou non.
//
// IL EST AUSSI UN GATE DE NON-REGRESSION DE LA MESURE : les quatre films temoins ont leurs
// valeurs attendues ci-dessous, verifiees a `originResearchToleranceMS` pres. Un film hors
// de cette liste est seulement mesure et journalise.
//
//	FILM_CACHE_ROOT=<repo>/data/cache ORIGIN_RESEARCH_FILMS="000d5950,64e8adfa,e94163af,606d9844" \
//	  go test ./internal/analysis/replay/ -run OriginMeasure -timeout 60m -v

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	originFilmsEnv = "ORIGIN_RESEARCH_FILMS"
	originCacheEnv = "FILM_CACHE_ROOT"
	// originResearchToleranceMS borne la derive acceptee sur une valeur attendue. Le decodage
	// est deterministe : la tolerance ne couvre qu'un changement de bornage des paquets, pas
	// une erreur de referentiel (le plus petit ecart en jeu vaut 3,6 s).
	originResearchToleranceMS = 50
)

// originAttendu : origine LUE (ms) et origine TEMOIN (ms) mesurees le 2026-08-14.
var originAttendu = map[string][2]int64{
	"000d5950": {3604, 3660},
	"64e8adfa": {10516, 10573},
	"e94163af": {39772, 39788},
	"606d9844": {6890, 6971},
}

func TestOriginMeasure(t *testing.T) {
	spec := os.Getenv(originFilmsEnv)
	if spec == "" {
		t.Skipf("mesure non demandee : %s vide", originFilmsEnv)
	}
	cache := os.Getenv(originCacheEnv)
	if cache == "" {
		t.Fatalf("%s est requis", originCacheEnv)
	}
	for _, short := range strings.Split(spec, ",") {
		short = strings.TrimSpace(short)
		if short == "" {
			continue
		}
		t.Run(short, func(t *testing.T) {
			lu, temoin := mesureOrigine(t, filepath.Join(cache, "film_chunks", short), short)
			att, ok := originAttendu[short]
			if !ok {
				return
			}
			if ecart := lu - att[0]; ecart > originResearchToleranceMS || ecart < -originResearchToleranceMS {
				t.Errorf("origine LUE = %d ms, attendu %d (ecart %d)", lu, att[0], ecart)
			}
			if ecart := temoin - att[1]; ecart > originResearchToleranceMS || ecart < -originResearchToleranceMS {
				t.Errorf("origine TEMOIN = %d ms, attendu %d (ecart %d)", temoin, att[1], ecart)
			}
		})
	}
}

// mesureOrigine rend l'origine par lecture et l'origine par temoin, en millisecondes.
func mesureOrigine(t *testing.T, dir, short string) (int64, int64) {
	t.Helper()
	release := filmdec.LockProcessDecode()
	defer release()
	// Sans bornes de carte les coordonnees restent des quanta — sans importance ici : seuls
	// les HORODATAGES sont mesures.
	scan := filmdec.DefaultScanFilmOptions()
	scan.QuantaOnly = true
	pos, err := filmdec.ScanFilmBipedPositions(dir, scan)
	if err != nil {
		t.Fatalf("positions : %v", err)
	}
	if len(pos) == 0 {
		t.Fatalf("aucune position")
	}
	originUS := pos[0].TimestampUS
	for _, p := range pos {
		if p.TimestampUS < originUS {
			originUS = p.TimestampUS
		}
	}
	clockUS, err := ScanFilmClockOrigin(dir)
	if err != nil {
		t.Fatalf("origine d'horloge : %v", err)
	}
	deaths, err := ScanFilmDeaths(dir)
	if err != nil {
		t.Fatalf("fil des morts : %v", err)
	}
	lives := buildLifeSpans(indexBySlot(pos))
	off, matched := bestDeathOffset(lives, deaths)
	lu := int64(originUS-clockUS) / 1000
	temoin := int64(originUS)/1000 - off
	avant, apres, n := ecartFilFiche(lives, deaths, off, originUS, lu)
	t.Logf("%s : originUS=%d chunk1US=%d morts=%d vies=%d appariees=%d "+
		"=> origineLUE=%d ms  origineTEMOIN=%d ms  (ecart %d ms) | "+
		"ecart fil/fiche median sur %d morts : AVANT %d ms, APRES %d ms",
		short, originUS, clockUS, len(deaths), len(lives), matched, lu, temoin, temoin-lu,
		n, avant, apres)
	return lu, temoin
}

// ecartFilFiche mesure ce que l'utilisateur voit : l'ecart entre la LIGNE DU FIL (l'instant
// d'une mort tel que le fil le date) et le FLASH DE FICHE (la fin de la vie correspondante
// dans l'artefact), en millisecondes, sur l'axe du rejeu.
//
// AVANT = sans origine publiee (l'horloge brute du fil, ce que faisait le client avant le
// contournement par appariement) ; APRES = l'origine LUE retranchee.
//
// L'appariement mort <-> vie reutilise `nameLivesByDeaths` (production) : la vie porte le
// xuid de sa victime, on reprend la plus proche parmi les siennes. Aucune seconde
// implementation de l'appariement ne vit ici.
func ecartFilFiche(lives []lifeSpan, deaths []Death, off int64, originUS uint64, lu int64) (int64, int64, int) {
	nameLivesByDeaths(lives, deaths, off)
	originMs := int64(originUS) / 1000
	var bruts []int64
	for _, d := range deaths {
		best, bestDelta := int64(0), int64(-1)
		for _, l := range lives {
			if l.xuid != d.XUID {
				continue
			}
			end := l.to / 1000
			delta := absI64(end - (d.TimeMS + off))
			if bestDelta < 0 || delta < bestDelta {
				bestDelta, best = delta, end
			}
		}
		if bestDelta < 0 || bestDelta > deathMatchWindowMS {
			continue
		}
		// Ligne du fil sans origine : d.TimeMS ; flash de fiche : fin de vie sur l'axe du
		// rejeu (fin absolue − origine des frames).
		bruts = append(bruts, d.TimeMS-(best-originMs))
	}
	if len(bruts) == 0 {
		return 0, 0, 0
	}
	sort.Slice(bruts, func(i, j int) bool { return bruts[i] < bruts[j] })
	avant := bruts[len(bruts)/2]
	return avant, avant - lu, len(bruts)
}
