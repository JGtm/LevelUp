package duckdb

// squad_life_placement_repo_test.go — LE PLACEMENT DES VIES SE LIT BORNÉ, SUR LA VUE `_latest`
// (plan `.ai/PLAN_EMPRISE_VIES_2026-09-28.md`, lot V3.1, décision V11 ; ADR 0036 I2).
//
// Base `:memory:` migrée (les vraies migrations shared, donc la vraie vue). Ce que chaque test
// verrouille :
//   - la lecture ne rend que la DERNIÈRE PASSE ENTIÈRE de chaque match (une passe plus ancienne,
//     plus longue, ne laisse survivre aucune ligne) ;
//   - elle ne rend que les matchs ET les joueurs demandés, et la fenêtre de la vue ne voit que
//     les lignes des matchs demandés (EXPLAIN ANALYZE, exigerFenetresBornees) ;
//   - les NULL restent nil (vie non mesurée, portée inconnue) et la variante de chaque match est
//     rendue : c'est elle qui permet au calcul d'écarter une ligne à portée périmée.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	duckdbdrv "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/analysis/squademprise"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
)

// vieEcrite : une ligne de `match_life_placement` à poser.
type vieEcrite struct {
	match, pass, xuid string
	written           string // horodatage (TIMESTAMP) : ordonne les passes
	start             int64
	median, radar     any // nil = NULL
	beyond            any
	kills             int
}

func poserVie(t *testing.T, pdb *PlayerDB, v vieEcrite) {
	t.Helper()
	tacExec(t, pdb, `INSERT INTO match_life_placement
		(match_id, decode_pass, decoder_rev, written_at, xuid, start_ms, end_ms, duration_ms,
		 measured_ms, median_m, beyond_ms, radar_m, carrier_ms, team_down_ms, unplaced_ms,
		 teammate_unplaced_ms, kills)
		VALUES (?, ?, 'placement-test', CAST(? AS TIMESTAMP), ?, ?, ?, 20000, 18000, ?, ?, ?, 100, 200, 300, 400, ?)`,
		v.match, v.pass, v.written, v.xuid, v.start, v.start+20000, v.median, v.beyond, v.radar, v.kills)
}

func poserVariante(t *testing.T, pdb *PlayerDB, match, variante string) {
	t.Helper()
	tacExec(t, pdb, `INSERT INTO match_registry (match_id, map_id, game_variant_name) VALUES (?, 'carte', ?)`,
		match, variante)
}

// corpusPlacement : dix matchs, trois joueurs (A et B de la composition, C du reste du lobby),
// deux vies chacun ; `m1` a en plus une passe ANCIENNE à trois vies pour A.
func corpusPlacement(t *testing.T, pdb *PlayerDB) {
	t.Helper()
	for i := 0; i < 10; i++ {
		m := fmt.Sprintf("m%d", i)
		poserVariante(t, pdb, m, fmt.Sprintf("Variante %d", i))
		for _, x := range []string{"A", "B", "C"} {
			for _, start := range []int64{1000, 60000} {
				poserVie(t, pdb, vieEcrite{match: m, pass: "p2", xuid: x, written: "2026-09-29 10:00:00",
					start: start, median: 12.5, radar: 18.0, beyond: int64(900), kills: 1})
			}
		}
	}
	for _, start := range []int64{1000, 30000, 60000} {
		poserVie(t, pdb, vieEcrite{match: "m1", pass: "p1", xuid: "A", written: "2026-09-28 10:00:00",
			start: start, median: 99.0, radar: 24.0, beyond: int64(5), kills: 7})
	}
	// m2 : une vie non mesurée et à portée inconnue pour B (NULL partout), passe courante.
	poserVie(t, pdb, vieEcrite{match: "m2", pass: "p2", xuid: "B", written: "2026-09-29 10:00:00",
		start: 90000, kills: 0})
}

func TestSquadLifePlacementRepo_BorneEtDernierePasse(t *testing.T) {
	b := newBaseNotee(t)
	corpusPlacement(t, b.pdb)
	b.carnet.vider()

	got, err := NewSquadLifePlacementRepo(b.pdb).LoadLifePlacement(context.Background(),
		[]string{"m1", "m2"}, []string{"A", "B"})
	if err != nil {
		t.Fatalf("LoadLifePlacement : %v", err)
	}
	// Fenêtre : au plus les lignes des deux matchs demandés, toutes passes et tous joueurs
	// confondus (m1 : 6 + 3 anciennes ; m2 : 6 + 1).
	exigerFenetresBornees(t, b, "LoadLifePlacement", 16, 1)

	var cles []string
	for _, r := range got.Rows {
		cles = append(cles, fmt.Sprintf("%s/%s/%d", r.MatchID, r.XUID, r.StartMS))
		if r.MatchID == "m1" && (r.Kills != 1 || *r.RadarM != 18) {
			t.Errorf("m1 : une ligne de la passe ancienne a survécu : %+v", r)
		}
	}
	want := "[m1/A/1000 m1/A/60000 m1/B/1000 m1/B/60000 m2/A/1000 m2/A/60000 m2/B/1000 m2/B/60000 m2/B/90000]"
	if fmt.Sprint(cles) != want {
		t.Fatalf("vies lues = %v\nattendu     %s", cles, want)
	}
	if got.Variants["m1"] != "Variante 1" || got.Variants["m2"] != "Variante 2" || len(got.Variants) != 2 {
		t.Errorf("variantes = %v, attendu m1 et m2 seulement", got.Variants)
	}
	nulle := got.Rows[len(got.Rows)-1]
	if nulle.MedianM != nil || nulle.RadarM != nil || nulle.BeyondMS != nil {
		t.Errorf("NULL relus comme valeurs : médiane %v, portée %v, hors radar %v",
			nulle.MedianM, nulle.RadarM, nulle.BeyondMS)
	}
	r := got.Rows[0]
	if *r.MedianM != 12.5 || *r.BeyondMS != 900 || r.DurationMS != 20000 || r.MeasuredMS != 18000 ||
		r.CarrierMS != 100 || r.TeamDownMS != 200 || r.UnplacedMS != 300 || r.TeammateUnplacedMS != 400 ||
		r.EndMS != 21000 {
		t.Errorf("champs mal relus : %+v", r)
	}
}

// TestSquadLifePlacementRepo_ListesVides : aucune requête sans match ou sans joueur.
func TestSquadLifePlacementRepo_ListesVides(t *testing.T) {
	b := newBaseNotee(t)
	corpusPlacement(t, b.pdb)
	b.carnet.vider()
	repo := NewSquadLifePlacementRepo(b.pdb)
	for _, c := range [][2][]string{{nil, {"A"}}, {{"m1"}, nil}} {
		got, err := repo.LoadLifePlacement(context.Background(), c[0], c[1])
		if err != nil || len(got.Rows) != 0 {
			t.Errorf("liste vide %v : %d vies, err %v", c, len(got.Rows), err)
		}
	}
	if n := len(b.carnet.vider()); n != 0 {
		t.Errorf("%d requête(s) envoyée(s) pour des listes vides", n)
	}
}

// TestSquadLifePlacementRepo_TableAbsente : une base sans la table rend
// games.ErrCapabilityNotSupported (dégradation propre, jamais une panne de la page).
func TestSquadLifePlacementRepo_TableAbsente(t *testing.T) {
	connector, err := duckdbdrv.NewConnector(":memory:", nil)
	if err != nil {
		t.Fatalf("NewConnector: %v", err)
	}
	raw := sql.OpenDB(connector)
	t.Cleanup(func() { _ = raw.Close(); _ = connector.Close() })
	shared := newTestDB(raw, ":memory:")
	pdb := &PlayerDB{Shared: shared, SharedReader: LegacySharedReader(shared)}
	_, err = NewSquadLifePlacementRepo(pdb).LoadLifePlacement(context.Background(), []string{"m1"}, []string{"A"})
	if !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Fatalf("err = %v, attendu games.ErrCapabilityNotSupported", err)
	}
}

// TestSquadLifePlacementRepo_LignePerimeeEcartee : une vie écrite sous une ancienne portée
// (24 m) sur un match dont la variante vaut aujourd'hui 18 m se relit telle quelle (portée
// écrite, variante), et le calcul l'écarte et la compte ; la vie à la bonne portée reste.
func TestSquadLifePlacementRepo_LignePerimeeEcartee(t *testing.T) {
	b := newBaseNotee(t)
	poserVariante(t, b.pdb, "m1", "Slayer:Arena")
	poserVie(t, b.pdb, vieEcrite{match: "m1", pass: "p1", xuid: "A", written: "2026-09-29 10:00:00",
		start: 1000, median: 9.0, radar: 18.0, beyond: int64(0), kills: 1})
	poserVie(t, b.pdb, vieEcrite{match: "m1", pass: "p1", xuid: "A", written: "2026-09-29 10:00:00",
		start: 60000, median: 9.0, radar: 24.0, beyond: int64(0), kills: 1})

	read, err := NewSquadLifePlacementRepo(b.pdb).LoadLifePlacement(context.Background(), []string{"m1"}, []string{"A"})
	if err != nil {
		t.Fatalf("LoadLifePlacement : %v", err)
	}
	if read.Variants["m1"] != "Slayer:Arena" {
		t.Fatalf("variante = %q", read.Variants["m1"])
	}
	bloc, bilan := squademprise.Placement(squademprise.PlacementInput{
		Players:      []domain.SessionUsageSquadPlayer{{XUID: "A"}},
		Scope:        []string{"m1"},
		Read:         read,
		CurrentRadar: map[string]float64{"m1": 18},
	})
	if bloc == nil || bloc.Coverage.StaleLives != 1 || bloc.Coverage.LivesTotal != 1 ||
		len(bloc.Players[0].Lives) != 1 || bloc.Players[0].Lives[0].StartMS != 1000 {
		t.Fatalf("ligne périmée non écartée : %+v", bloc)
	}
	if len(bilan.StaleMatches) != 1 || bilan.StaleMatches[0] != "m1" {
		t.Errorf("bilan = %+v", bilan)
	}
}
