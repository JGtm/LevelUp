//go:build integration

package kickoff

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/timeline"
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/replayartifact"
)

// baseMigree : une base partagée migrée, dans un fichier temporaire.
func baseMigree(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", filepath.Join(t.TempDir(), "shared.duckdb"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	if err := migration.RunForDB(db, migration.TargetShared); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func exec(t *testing.T, db *sql.DB, q string, args ...any) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(), q, args...); err != nil {
		t.Fatalf("%s: %v", q, err)
	}
}

// semer range trois matchs de 600 s : `film` (rejeu publié à 26,304 s, registre sur l'estimation),
// `api` (pas de rejeu, registre déjà juste), `faux` (registre décalé de 4 h, arrivées incohérentes).
func semer(t *testing.T, db *sql.DB, start time.Time) {
	t.Helper()
	for _, id := range []string{"film", "api", "faux"} {
		exec(t, db, `INSERT INTO match_registry (match_id, start_time, start_time_utc, duration_seconds) VALUES (?, ?, ?, 600)`,
			id, start, start)
	}
	exec(t, db, `UPDATE match_registry SET real_start_time = ? WHERE match_id IN ('film', 'api')`, start.Add(20*time.Second))
	exec(t, db, `UPDATE match_registry SET real_start_time = ? WHERE match_id = 'faux'`, start.Add(4*time.Hour))
	for _, id := range []string{"film", "api"} {
		exec(t, db, `INSERT INTO match_participants (match_id, xuid, present_at_beginning, first_joined_time, time_played_seconds)
			VALUES (?, '2603225103667623', TRUE, ?, 580), (?, '2547408409520575', TRUE, ?, 579)`,
			id, start.Add(20*time.Second), id, start.Add(20500*time.Millisecond))
	}
	exec(t, db, `INSERT INTO match_participants (match_id, xuid, present_at_beginning, first_joined_time, time_played_seconds)
		VALUES ('faux', '2603225103667623', TRUE, ?, 600)`, start.Add(4*time.Hour))
}

func acces(db *sql.DB, ecritures *int) Access {
	return Access{
		Read: func(context.Context) (*sql.DB, func(), error) { return db, func() {}, nil },
		Write: func(context.Context) (*sql.DB, func(), error) {
			*ecritures++
			return db, func() {}, nil
		},
	}
}

func options(t *testing.T) Options {
	t.Helper()
	dir := t.TempDir()
	path := func(id string) string { return filepath.Join(dir, id+".json.gz") }
	if _, _, err := replayartifact.Write(context.Background(), path("film"), []byte(`{"t0FilmMs":26304}`)); err != nil {
		t.Fatalf("artefact: %v", err)
	}
	return Options{TitleSlug: "halo_infinite", ArtifactPath: path, LotSize: 2}
}

func etat(t *testing.T, db *sql.DB, id string) (sql.NullTime, string, int) {
	t.Helper()
	var rst sql.NullTime
	var src string
	var tp int
	if err := db.QueryRowContext(context.Background(), `
		SELECT mr.real_start_time, COALESCE(mr.real_start_source, ''), MIN(mp.time_played_seconds)
		FROM match_registry mr JOIN match_participants mp USING (match_id)
		WHERE mr.match_id = ? GROUP BY ALL`, id).Scan(&rst, &src, &tp); err != nil {
		t.Fatalf("etat %s: %v", id, err)
	}
	return rst, src, tp
}

func TestBackfill_SimulationPuisApplicationPuisSecondPassage(t *testing.T) {
	db := baseMigree(t)
	start := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	semer(t, db, start)
	o := options(t)

	writes := 0
	sim := o
	sim.DryRun = true
	rep, err := Backfill(context.Background(), acces(db, &writes), sim)
	if err != nil || writes != 0 || rep.Written != 0 {
		t.Fatalf("simulation : err=%v writer=%d ecrits=%d, attendu aucune ecriture", err, writes, rep.Written)
	}
	if rep.Matches != 3 || rep.BySource[timeline.KickoffSourceFilm] != 1 || rep.BySource[timeline.KickoffSourceAPI] != 1 ||
		rep.BySource[timeline.KickoffSourceUnknown] != 1 || rep.BecameUnknown != 1 || rep.PlayersChanged != 2 {
		t.Fatalf("bilan de simulation = %+v", rep)
	}
	if rst, src, _ := etat(t, db, "film"); src != "" || !rst.Time.UTC().Equal(start.Add(20*time.Second)) {
		t.Fatal("la simulation a ecrit")
	}

	rep, err = Backfill(context.Background(), acces(db, &writes), o)
	if err != nil || rep.Written != 3 {
		t.Fatalf("application : err=%v ecrits=%d, attendu 3 (source posee partout ou elle change)", err, rep.Written)
	}
	if rst, src, tp := etat(t, db, "film"); src != "film" || !rst.Time.UTC().Equal(start.Add(26304*time.Millisecond)) || tp != 573 {
		t.Fatalf("film = (%v, %q, %d)", rst, src, tp)
	}
	if rst, src, tp := etat(t, db, "api"); src != "api" || !rst.Time.UTC().Equal(start.Add(20*time.Second)) || tp != 579 {
		t.Fatalf("api = (%v, %q, %d)", rst, src, tp)
	}
	if rst, src, _ := etat(t, db, "faux"); rst.Valid || src != "" {
		t.Fatalf("faux = (%v, %q), attendu inconnu", rst, src)
	}

	rep, err = Backfill(context.Background(), acces(db, &writes), o)
	if err != nil || rep.Written != 0 || rep.ValueChanged != 0 || rep.SourceChanged != 0 || rep.PlayersChanged != 0 {
		t.Fatalf("second passage : err=%v bilan=%+v, attendu rien a ecrire", err, rep)
	}
}

func TestApplyRevision_RangeeApresSucces(t *testing.T) {
	db := baseMigree(t)
	semer(t, db, time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC))
	o := options(t)
	writes := 0
	applied, _, err := ApplyRevision(context.Background(), acces(db, &writes), o)
	if err != nil || !applied {
		t.Fatalf("premier demarrage : applied=%v err=%v", applied, err)
	}
	var v string
	if err := db.QueryRowContext(context.Background(), `SELECT value FROM sync_meta WHERE key = ?`, SyncMetaKey).Scan(&v); err != nil || v != timeline.KickoffRev {
		t.Fatalf("revision rangee = %q (err %v)", v, err)
	}
	writes = 0
	applied, _, err = ApplyRevision(context.Background(), acces(db, &writes), o)
	if err != nil || applied || writes != 0 {
		t.Fatalf("second demarrage : applied=%v writer=%d err=%v, attendu rien", applied, writes, err)
	}
}

// Un segment d'écriture indisponible : rien n'est rangé, l'erreur remonte, la révision reste.
func TestApplyRevision_EchecNeRangeRien(t *testing.T) {
	db := baseMigree(t)
	semer(t, db, time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC))
	acc := Access{
		Read:  func(context.Context) (*sql.DB, func(), error) { return db, func() {}, nil },
		Write: func(context.Context) (*sql.DB, func(), error) { return nil, nil, errors.New("lease tenu") },
	}
	applied, _, err := ApplyRevision(context.Background(), acc, options(t))
	if err == nil || applied {
		t.Fatalf("applied=%v err=%v, attendu un echec", applied, err)
	}
	var n int
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM sync_meta WHERE key = ?`, SyncMetaKey).Scan(&n); err != nil || n != 0 {
		t.Fatalf("revision rangee malgre l'echec (%d, err %v)", n, err)
	}
}

// semerFilmRange : le match `film` porte déjà le coup d'envoi du rejeu (26,304 s, source film) et les
// temps de jeu calculés dessus.
func semerFilmRange(t *testing.T, db *sql.DB, start time.Time) {
	t.Helper()
	exec(t, db, `UPDATE match_registry SET real_start_time = ?, real_start_source = 'film' WHERE match_id = 'film'`,
		start.Add(26304*time.Millisecond))
	exec(t, db, `UPDATE match_participants SET time_played_seconds = 573 WHERE match_id = 'film'`)
}

func optionsVers(dir string) Options {
	return Options{TitleSlug: "halo_infinite", LotSize: 2,
		ArtifactPath: func(id string) string { return filepath.Join(dir, id+".json.gz") }}
}

// Rangé `film`, rejeu ABSENT (purgé) : rien n'est écrit pour ce match, et ce n'est pas un échec.
func TestBackfill_RangeFilmRejeuAbsent_RienNEstEcrit(t *testing.T) {
	db := baseMigree(t)
	start := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	semer(t, db, start)
	semerFilmRange(t, db, start)
	writes := 0
	rep, err := Backfill(context.Background(), acces(db, &writes), optionsVers(t.TempDir()))
	if err != nil || rep.KeptFilm != 1 {
		t.Fatalf("err=%v garde=%d, attendu aucun echec et le film garde", err, rep.KeptFilm)
	}
	if rst, src, tp := etat(t, db, "film"); src != "film" || !rst.Time.UTC().Equal(start.Add(26304*time.Millisecond)) || tp != 573 {
		t.Fatalf("film = (%v, %q, %d), attendu inchange", rst, src, tp)
	}
}

// Rejeu PRÉSENT qui ne publie pas de coup d'envoi : le match passe à l'estimation, même rangé `film`.
func TestBackfill_RejeuSansCoupDEnvoi_PassageALEstimation(t *testing.T) {
	db := baseMigree(t)
	start := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	semer(t, db, start)
	semerFilmRange(t, db, start)
	dir := t.TempDir()
	if _, _, err := replayartifact.Write(context.Background(), filepath.Join(dir, "film.json.gz"), []byte(`{"schemaVersion":125}`)); err != nil {
		t.Fatalf("artefact: %v", err)
	}
	writes := 0
	if _, err := Backfill(context.Background(), acces(db, &writes), optionsVers(dir)); err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if rst, src, _ := etat(t, db, "film"); src != "api" || !rst.Time.UTC().Equal(start.Add(20*time.Second)) {
		t.Fatalf("film = (%v, %q), attendu l'estimation", rst, src)
	}
}

// Un match disparu du registre entre la liste et son lot n'est pas un échec.
func TestRunLot_MatchDisparuNestPasUnEchec(t *testing.T) {
	db := baseMigree(t)
	semer(t, db, time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC))
	writes := 0
	rep := newReport()
	if err := runLot(context.Background(), acces(db, &writes), optionsVers(t.TempDir()), []string{"api", "disparu"}, &rep); err != nil {
		t.Fatalf("runLot: %v", err)
	}
	if rep.Failures != 0 || rep.Missing != 1 {
		t.Fatalf("echecs=%d absents=%d, attendu 0 et 1", rep.Failures, rep.Missing)
	}
}

// Rangé `film`, rejeu ABSENT, temps de jeu d'un joueur périmé : le registre n'est pas réécrit, le temps
// de jeu est corrigé sur le coup d'envoi rangé ; un second passage n'écrit rien.
func TestBackfill_RangeFilmRejeuAbsent_TempsDeJeuCorrige(t *testing.T) {
	db := baseMigree(t)
	start := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	semer(t, db, start)
	semerFilmRange(t, db, start)
	exec(t, db, `UPDATE match_participants SET time_played_seconds = 580 WHERE match_id = 'film' AND xuid = '2603225103667623'`)
	writes := 0
	dir := t.TempDir()
	rep, err := Backfill(context.Background(), acces(db, &writes), optionsVers(dir))
	// Les deux autres matchs semés (`api`, `faux`) se résolvent normalement : leur changement de source
	// ou de valeur est dans le bilan global ; seul le match `film` est contrôlé ligne à ligne ci-dessous.
	if err != nil || rep.KeptFilm != 1 || rep.PlayersChanged != 1 {
		t.Fatalf("err=%v bilan=%+v, attendu le film garde et un joueur corrige", err, rep)
	}
	if rst, src, tp := etat(t, db, "film"); src != "film" || !rst.Time.UTC().Equal(start.Add(26304*time.Millisecond)) || tp != 573 {
		t.Fatalf("film = (%v, %q, %d), attendu registre intact et temps de jeu 573", rst, src, tp)
	}
	rep, err = Backfill(context.Background(), acces(db, &writes), optionsVers(dir))
	if err != nil || rep.Written != 0 || rep.PlayersChanged != 0 || rep.KeptFilm != 1 {
		t.Fatalf("second passage : err=%v bilan=%+v, attendu zero ecriture", err, rep)
	}
}
