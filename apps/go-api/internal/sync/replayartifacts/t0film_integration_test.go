//go:build integration

package replayartifacts

// t0film_integration_test.go — LE REPORT DU COUP D'ENVOI, EXECUTE POUR DE VRAI.
//
// POURQUOI UN TEST D'INTEGRATION. Les deux requetes du report sont construites par
// CONCATENATION (le fragment de start canonique s'insere dans le SELECT), et l'UPDATE ecrit une
// colonne TIMESTAMP a partir d'une `time.Time`. Un test a chaines de caracteres resterait vert
// pendant qu'a l'execution le SELECT echouerait -> WARN -> report no-op permanent : exactement
// le mode de panne silencieuse que ce lot ferme. Ici les requetes tournent sur une base migree,
// et le test relit ce que la base porte APRES.
//
// LE VERDICT PAR MATCH EST L'OBJET DU TEST, pas seulement l'etat final : « ecrit » et « deja
// la » laissent la MEME ligne en base, et seule la valeur de retour les distingue. Un test qui
// ne lirait que la ligne ne pourrait pas prouver que la garde mord.

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/timeline"
	"levelup/go-api/internal/persist"
)

// ecrivain : le persister sous lequel l'UPDATE passe reellement (match_registry est une table
// match-of-record : son ecriture vit dans internal/persist, cf. t0film.go).
func ecrivain(db *sql.DB) *persist.T0FilmPersister { return persist.NewT0FilmPersister(db) }

// etatT0 relit ce que la base porte pour un match.
func etatT0(t *testing.T, db *sql.DB, id string) (sql.NullTime, string) {
	t.Helper()
	var rst sql.NullTime
	var q string
	err := db.QueryRowContext(context.Background(),
		`SELECT real_start_time, COALESCE(t0_quality, '') FROM match_registry WHERE match_id = ?`, id).
		Scan(&rst, &q)
	if err != nil {
		t.Fatalf("relecture %s: %v", id, err)
	}
	return rst, q
}

// semerT0API pose sur un match le T0 ESTIME de l'API — l'etat que le film vient corriger.
func semerT0API(t *testing.T, db *sql.DB, id string, start time.Time, t0 time.Duration) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(),
		`UPDATE match_registry SET real_start_time = ?, t0_quality = ? WHERE match_id = ?`,
		start.Add(t0), string(timeline.T0QualityOK), id); err != nil {
		t.Fatalf("seed T0 API %s: %v", id, err)
	}
}

// TestEcrireUnT0Film_CorrigeeUneFoisPuisPlusJamais : les deux verdicts du chemin nominal.
//
//  1. le coup d'envoi MESURE remplace le T0 estime degenere (~1 ms), avec sa qualite ;
//  2. le MEME report rejoue rend « deja la » et n'ecrit pas — sans quoi un artefact re-cuit
//     (reparation d'un appauvri, montee de schema) rejouerait un UPDATE identique a chaque
//     cycle, sur une table critique, pour rien.
func TestEcrireUnT0Film_CorrigeeUneFoisPuisPlusJamais(t *testing.T) {
	db := baseRegistre(t)
	start := time.Date(2026, 9, 2, 18, 0, 0, 0, time.UTC)
	inscrireAuRegistre(t, db, "m-film", start, 0)
	semerT0API(t, db, "m-film", start, time.Millisecond)

	const t0FilmMs = 26304
	rapport := rapportT0Film{matchID: "m-film", t0FilmMs: t0FilmMs}

	if got := ecrireUnT0Film(context.Background(), db, ecrivain(db), rapport); got != t0FilmEcrit {
		t.Fatalf("premier report = %v, attendu t0FilmEcrit", got)
	}
	attendu := start.Add(t0FilmMs * time.Millisecond)
	rst, q := etatT0(t, db, "m-film")
	if !rst.Valid || !rst.Time.UTC().Equal(attendu) {
		t.Fatalf("real_start_time = %v, attendu %v", rst, attendu)
	}
	if q != string(timeline.T0QualityFilmMovement) {
		t.Fatalf("t0_quality = %q, attendu %q", q, timeline.T0QualityFilmMovement)
	}

	if got := ecrireUnT0Film(context.Background(), db, ecrivain(db), rapport); got != t0FilmDejaLa {
		t.Fatalf("second report = %v, attendu t0FilmDejaLa (la garde doit mordre)", got)
	}
}

// TestEcrireUnT0Film_MesureDifferenteEcrase : la garde protege l'IDENTITE, pas la qualite. Un
// artefact recuit qui rend une AUTRE mesure doit corriger la ligne, meme deja `film_movement` —
// sinon une correction du detecteur ne se propagerait jamais.
func TestEcrireUnT0Film_MesureDifferenteEcrase(t *testing.T) {
	db := baseRegistre(t)
	start := time.Date(2026, 9, 2, 18, 0, 0, 0, time.UTC)
	inscrireAuRegistre(t, db, "m-film", start, 0)

	base := rapportT0Film{matchID: "m-film", t0FilmMs: 26304}
	if got := ecrireUnT0Film(context.Background(), db, ecrivain(db), base); got != t0FilmEcrit {
		t.Fatalf("premier report = %v, attendu t0FilmEcrit", got)
	}
	corrige := rapportT0Film{matchID: "m-film", t0FilmMs: 31862}
	if got := ecrireUnT0Film(context.Background(), db, ecrivain(db), corrige); got != t0FilmEcrit {
		t.Fatalf("report corrige = %v, attendu t0FilmEcrit", got)
	}
	rst, _ := etatT0(t, db, "m-film")
	attendu := start.Add(31862 * time.Millisecond)
	if !rst.Valid || !rst.Time.UTC().Equal(attendu) {
		t.Fatalf("real_start_time = %v, attendu %v", rst, attendu)
	}
}

// TestEcrireUnT0Film_MatchAbsentDuRegistre : le report CORRIGE, il ne CREE jamais. Un artefact
// dont le match n'a pas de ligne rend un echec journalise et laisse la table vide.
func TestEcrireUnT0Film_MatchAbsentDuRegistre(t *testing.T) {
	db := baseRegistre(t)
	got := ecrireUnT0Film(context.Background(), db, ecrivain(db),
		rapportT0Film{matchID: "inconnu", t0FilmMs: 26304})
	if got != t0FilmEchec {
		t.Fatalf("match absent = %v, attendu t0FilmEchec", got)
	}
	var n int
	if err := db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM match_registry`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Fatalf("lignes = %d, attendu 0 (le report ne cree jamais de ligne)", n)
	}
}

// TestReporterT0Film_BurstWriterCourt : le segment writer est acquis UNE fois pour le lot,
// relache a la sortie, et le lot entier passe par lui.
func TestReporterT0Film_BurstWriterCourt(t *testing.T) {
	db := baseRegistre(t)
	start := time.Date(2026, 9, 2, 18, 0, 0, 0, time.UTC)
	for _, id := range []string{"m1", "m2"} {
		inscrireAuRegistre(t, db, id, start, 0)
		semerT0API(t, db, id, start, time.Millisecond)
	}
	acquis, relaches := 0, 0
	d := Deps{
		Gamertag: "testeur",
		AcquireWriter: func(context.Context) (*sql.DB, func(), error) {
			acquis++
			return db, func() { relaches++ }, nil
		},
	}
	reporterT0Film(context.Background(), d, &bilanDerivations{}, []rapportT0Film{
		{matchID: "m1", t0FilmMs: 26304}, {matchID: "m2", t0FilmMs: 31862},
	})
	if acquis != 1 || relaches != 1 {
		t.Fatalf("writer acquis %d fois, relache %d fois — attendu 1 et 1 (burst unique)", acquis, relaches)
	}
	for id, ms := range map[string]int64{"m1": 26304, "m2": 31862} {
		rst, q := etatT0(t, db, id)
		attendu := start.Add(time.Duration(ms) * time.Millisecond)
		if !rst.Valid || !rst.Time.UTC().Equal(attendu) || q != string(timeline.T0QualityFilmMovement) {
			t.Errorf("%s : (%v, %q), attendu (%v, %q)", id, rst, q, attendu, timeline.T0QualityFilmMovement)
		}
	}
}

// TestReporterT0Film_LotVideNAcquiertAucunWriter : un cycle qui n'a rien cuit ne doit pas
// prendre le writer partage — c'est le cas le plus frequent en regime stationnaire.
func TestReporterT0Film_LotVideNAcquiertAucunWriter(t *testing.T) {
	acquis := 0
	d := Deps{AcquireWriter: func(context.Context) (*sql.DB, func(), error) {
		acquis++
		return nil, func() {}, nil
	}}
	reporterT0Film(context.Background(), d, &bilanDerivations{}, nil)
	if acquis != 0 {
		t.Fatalf("writer acquis %d fois sur un lot vide, attendu 0", acquis)
	}
}
