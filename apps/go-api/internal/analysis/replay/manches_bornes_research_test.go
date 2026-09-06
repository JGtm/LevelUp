package replay

// manches_bornes_research_test.go — INSTRUMENT DE MESURE des bornes de manche : ce qui separe,
// sur un film reel, une manche JOUEE d'un etiquetage de manche qui ne suit pas l'horloge.
//
// # Ce qu'il imprime, et pourquoi ces trois colonnes
//
// Il rend, par manche REELLE d'un film, ce dont `objectiveevents.ResolveRoundWindows` a besoin
// et rien d'autre :
//
//	SLOTS + DEBUTS   le premier instant ou CHAQUE slot declare la manche. C'est la table qui a
//	                 tranche entre le minimum et la mediane : sur `24dbb67d`, deux slots
//	                 declarent la manche 1 des 85 193 ms alors que les huit autres l'ouvrent a
//	                 298 909 ms.
//	MEDIANE          le debut consensuel retenu.
//	MILIEU           la mediane des instants de la manche — le repere qui juge si une borne
//	                 separe vraiment les deux populations.
//
// Puis le nombre d'enregistrements que les bornes ecartent sur ce film.
//
// # Le releve du 2026-09-06 (douze films multi-manche du parc), qui fonde les trois gardes
//
//	manche JOUEE     declaree par les DIX slots, debuts groupes a 3 s pres (les deux slots
//	                 d'equipe emettent ~3 s apres les huit slots de joueur) ;
//	manche PARASITE  declaree par UN seul slot (`a4083bd2` manche 1 : 1 enregistrement) ou par
//	                 AUCUN (`fb1a1a72` et `72b0a25e` manche 1, admises par la tolerance de trou
//	                 de `RealRounds`) ;
//	ecartes          5 a 27 par film sur les neuf films a etiquetage sain ; ZERO sur les trois
//	                 films dont l'etiquetage ne suit pas l'horloge, ou aucune borne n'est posee.
//
// REGIME : garde `MANCHES_CACHE` (racine du cache film) + `MANCHES_FILMS` (liste de prefixes de
// match, separes par des virgules). Aucune base, aucun reseau, un film a la fois.
//
//	$env:MANCHES_CACHE="C:/.../LevelUp-go-migration/data/cache"
//	$env:MANCHES_FILMS="51ebbc0f,d9781168,24dbb67d,a4083bd2,fb1a1a72"
//	go test ./internal/analysis/replay/ -run ManchesBornes -v -timeout 60m

import (
	"context"
	"os"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
)

// TestManchesBornesReleve imprime, film par film, la table des debuts de manche par slot.
func TestManchesBornesReleve(t *testing.T) {
	cache, films := os.Getenv("MANCHES_CACHE"), os.Getenv("MANCHES_FILMS")
	if cache == "" || films == "" {
		t.Skip("mesure non demandee : MANCHES_CACHE et MANCHES_FILMS requis")
	}
	defer amArmeSentinelle(t, "TestManchesBornesReleve")()
	for _, film := range strings.Split(films, ",") {
		if film = strings.TrimSpace(film); film == "" {
			continue
		}
		src, ok, err := filmcache.LoadFilm(cache, film)
		if err != nil || !ok {
			t.Logf("FILM %s ABSENT (%v) — saute", film, err)
			continue
		}
		recs, tronque := objectiveevents.StatRecordsCtx(context.Background(), src, film)
		mbReleveFilm(t, film, recs, tronque)
	}
}

// mbReleveFilm imprime le releve d'un film.
func mbReleveFilm(t *testing.T, film string, recs []objectiveevents.StatRecord, tronque bool) {
	t.Helper()
	real := objectiveevents.RealRounds(recs)
	rounds := make([]int, 0, len(real))
	for r, ok := range real {
		if ok {
			rounds = append(rounds, r)
		}
	}
	sort.Ints(rounds)
	ecartes := objectiveevents.ResolveRoundWindows(recs).Outliers(recs)
	t.Logf("FILM %s : %d enregistrements, tronque=%v, manches reelles=%v, ECARTES=%d",
		film, len(recs), tronque, rounds, ecartes)
	for _, round := range rounds {
		debuts, instants := mbColonnes(recs, round)
		t.Logf("  manche %d : slots=%d mediane des debuts=%d milieu=%d debuts=%v",
			round, len(debuts), mbMediane(debuts), mbMediane(instants), debuts)
	}
}

// mbColonnes rend, pour une manche : le premier instant de chaque slot (trie) et tous les
// instants de la manche (tries).
func mbColonnes(recs []objectiveevents.StatRecord, round int) (debuts, instants []int) {
	premier := map[int]int{}
	for _, r := range recs {
		if r.Round != round {
			continue
		}
		instants = append(instants, r.TimeMS)
		if v, seen := premier[r.Slot]; !seen || r.TimeMS < v {
			premier[r.Slot] = r.TimeMS
		}
	}
	for _, v := range premier {
		debuts = append(debuts, v)
	}
	sort.Ints(debuts)
	sort.Ints(instants)
	return debuts, instants
}

// mbMediane rend la mediane BASSE d'une liste triee — la meme que la production.
func mbMediane(tri []int) int {
	if len(tri) == 0 {
		return -1
	}
	return tri[(len(tri)-1)/2]
}
