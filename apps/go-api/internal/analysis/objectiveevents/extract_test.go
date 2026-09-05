package objectiveevents

import (
	"os"
	"testing"

	"levelup/go-api/internal/analysis/filmsource"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
)

// filmCacheEnv — nom de la variable qui porte la racine du cache film
// (`film_manifests/<id>.json` + `film_chunks/<id>/chunk_NN.bin`). Même convention que
// `KILLSOURCE_FIXTURES` : les films ne sont pas versionnés (107 Mo), donc les tests de
// vérité terrain se skippent proprement quand ils ne sont pas là.
//
// ELLE REMPLACE UN CHEMIN ÉCRIT EN DUR (2026-08-05, R3). Le chemin visait
// `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache`, un poste qui
// n'existe plus : les tests adossés au cache — dont l'oracle à HUIT joueurs, celui qui
// garde les noms `flag_grabs`/`zone_secures` — se skippaient donc EN PERMANENCE, sur
// toutes les machines, et rien ne le disait sinon un message citant un chemin mort. Un
// chemin absolu en dur ne casse pas : il se tait.
//
//	FILM_CACHE_ROOT=<repo>/data/cache go test ./internal/analysis/objectiveevents/
const filmCacheEnv = "FILM_CACHE_ROOT"

// cacheRoot rend la racine du cache film, ou "" si la variable n'est pas posée.
func cacheRoot() string { return os.Getenv(filmCacheEnv) }

// newDiskFilm charge UN film du cache disque (film_manifests/<id>.json +
// film_chunks/<id>/chunk_NN.bin) dans la forme que prennent les points d'entree : un
// `*filmsource.Film` deja decompresse et decoupe. Renvoie (nil,false) si le film est absent.
//
// IL PASSE PAR `filmcache`, LA SEULE PORTE DU CACHE, ET C'EST NEUF (item 1.5, 2026-09-02).
// Ce fichier reconstituait la disposition du cache pour son compte, avec une entree d'allowlist
// datee dans l'ancien `filmcache_guard_test.go` ; sa justification etait un CYCLE D'IMPORT —
// `filmcache` importait `objectiveevents` pour ses types de chunk. Le lot 1 a supprime ce
// cycle (les deux paquets dependent maintenant du paquet FEUILLE `filmsource`), donc la
// derogation n'a plus d'objet et la copie s'en va avec elle.
func newDiskFilm(t *testing.T, id string) (*filmsource.Film, bool) {
	t.Helper()
	film, ok, err := filmcache.LoadFilm(cacheRoot(), id)
	if err != nil {
		t.Fatalf("film %s : chargement : %v", id, err)
	}
	return film, ok
}

// rosterFor renvoie le mapping xuid->team_id ground-truth (issu de
// match_participants, vérifié 2026-06-02). Gardé en dur pour garder le test pur
// (aucune dépendance DuckDB).
func rosterFor(id string) MapRoster {
	switch id {
	case "0f9550e5":
		return MapRoster{
			"2533274908381477": 0, "2535421948780711": 0,
			"2535454710220286": 0, "2535458905252558": 0,
			"2533274823110022": 1, "2533274897656966": 1,
			"2535416374764743": 1, "2535447374983571": 1,
			"2535471644287443": 1, "2535472402320362": 1,
		}
	case "53ce4390":
		return MapRoster{
			"2533274840602701": 0, "2533274860882060": 0,
			"2535439497055986": 0, "2535462641971683": 0,
			"2533274803754807": 1, "2533274823110022": 1,
			"2533274830881544": 1, "2535430195856593": 1,
		}
	}
	return nil
}

// captureSplit compte les events capture par team_id.
func captureSplit(events []domain.ObjectiveEvent) (team0, team1, unknown int) {
	for _, e := range events {
		if e.ObjectiveType != ObjectiveTypeFlag || e.EventType != EventTypeCapture {
			continue
		}
		switch {
		case e.TeamID == nil:
			unknown++
		case *e.TeamID == 0:
			team0++
		case *e.TeamID == 1:
			team1++
		}
	}
	return
}

// TestExtractCTFCaptureCount valide, sur les matchs CTF de ground-truth, que le
// COUNT de captures décodées == total DB final, et que le split par équipe colle
// au score DB (équipe résolue via roster sur le scorer du cluster th=10).
func TestExtractCTFCaptureCount(t *testing.T) {
	cases := []struct {
		id             string
		variant        string
		wantTotal      int
		wantT0, wantT1 int
	}{
		// 0f9550e5 = CTF Neutral Flag, score final 5-0 (team0).
		{"0f9550e5", "CTF:Arena Neutral Flag", 5, 5, 0},
		// 53ce4390 = CTF Arena, score final 1-2 (team0:1, team1:2).
		{"53ce4390", "CTF:Arena", 3, 1, 2},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.id, func(t *testing.T) {
			bobine, ok := newDiskFilm(t, tc.id)
			if !ok {
				t.Skipf("film %s absent du cache (%s=%q) — verite terrain non rejouee", tc.id, filmCacheEnv, cacheRoot())
			}
			events := Extract(tc.id, tc.variant, bobine, rosterFor(tc.id))

			var captures int
			for _, e := range events {
				if e.EventType == EventTypeCapture {
					captures++
				}
			}
			if captures != tc.wantTotal {
				t.Errorf("%s: capture COUNT = %d, want %d (DB final total)", tc.id, captures, tc.wantTotal)
			}

			t0, t1, unknown := captureSplit(events)
			if t0 != tc.wantT0 || t1 != tc.wantT1 {
				t.Errorf("%s: per-team split = %d-%d (unknown=%d), want %d-%d",
					tc.id, t0, t1, unknown, tc.wantT0, tc.wantT1)
			}

			// Invariants de structure : seq dense, type/source/confidence CTF.
			for i, e := range events {
				if e.Seq != i {
					t.Errorf("%s: event %d has Seq=%d (want dense %d)", tc.id, i, e.Seq, i)
				}
				if e.ObjectiveType != ObjectiveTypeFlag {
					t.Errorf("%s: event %d ObjectiveType=%q want %q", tc.id, i, e.ObjectiveType, ObjectiveTypeFlag)
				}
				if e.Source != SourceBurst || e.Confidence != ConfidenceExact {
					t.Errorf("%s: event %d source/confidence = %q/%q want %q/%q",
						tc.id, i, e.Source, e.Confidence, SourceBurst, ConfidenceExact)
				}
				if e.ObjectiveID != nil {
					t.Errorf("%s: event %d ObjectiveID=%v want NULL (non récupérable)", tc.id, i, *e.ObjectiveID)
				}
			}
		})
	}
}

// TestClassifyObjectiveMode couvre le dispatch de mode sur des variantes
// hétérogènes (CTF des 2 côtés du ':', KOTH/Strongholds/Oddball, non-objectif).
func TestClassifyObjectiveMode(t *testing.T) {
	cases := map[string]string{
		"CTF:Arena":                  ObjectiveTypeFlag,
		"Arena:CTF":                  ObjectiveTypeFlag,
		"Ranked:CTF":                 ObjectiveTypeFlag,
		"BTB:One Flag CTF":           ObjectiveTypeFlag,
		"Husky Raid:Super CTF":       ObjectiveTypeFlag,
		"Strongholds:Arena":          ObjectiveTypeZone,
		"Arena:Strongholds":          ObjectiveTypeZone,
		"BTB:Total Control":          ObjectiveTypeZone,
		"Arena:Land Grab":            ObjectiveTypeZone,
		"KOTH:Arena":                 ObjectiveTypeHill,
		"Arena:King of the Hill":     ObjectiveTypeHill,
		"Gruntpocalypse:Heroic KOTH": ObjectiveTypeHill,
		"Oddball:Arena":              ObjectiveTypeSkull,
		"Ranked:Oddball":             ObjectiveTypeSkull,
		"Arena:Slayer":               "",
		"BTB:Slayer":                 "",
		"Slayer:Arena Super Fiesta":  "",
	}
	for variant, want := range cases {
		if got := classifyObjectiveMode(variant); got != want {
			t.Errorf("classifyObjectiveMode(%q) = %q, want %q", variant, got, want)
		}
	}
}
