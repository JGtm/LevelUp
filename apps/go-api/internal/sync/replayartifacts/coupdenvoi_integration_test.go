//go:build integration

package replayartifacts

// coupdenvoi_integration_test.go — LA PUBLICATION D'UN REJEU RÉÉCRIT LE COUP D'ENVOI, POUR DE VRAI.
//
// Les requêtes de lecture (fragment de début canonique concaténé) et les deux UPDATE du persisteur
// tournent sur une base migrée ; le test relit ce que la base porte APRÈS, et compte les
// acquisitions du writer : une publication qui ne change rien ne doit pas en prendre.

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"levelup/go-api/internal/observability"

	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

// inscrireJoueur range un participant présent du début à la fin, arrivé à `arrivee`, avec le temps
// de jeu de l'API `api`.
func inscrireJoueur(t *testing.T, db *sql.DB, matchID, xuid string, arrivee time.Time, api int) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(), `
		INSERT INTO match_participants (match_id, xuid, present_at_beginning, first_joined_time, time_played_seconds)
		VALUES (?, ?, TRUE, ?, ?)`, matchID, xuid, arrivee, api); err != nil {
		t.Fatalf("insert participant %s/%s: %v", matchID, xuid, err)
	}
}

// poserDuree fixe la durée du match.
func poserDuree(t *testing.T, db *sql.DB, matchID string, secs int) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(),
		`UPDATE match_registry SET duration_seconds = ? WHERE match_id = ?`, secs, matchID); err != nil {
		t.Fatalf("duree %s: %v", matchID, err)
	}
}

// etatCoupDEnvoi relit le coup d'envoi, sa source et le temps de jeu d'un joueur.
func etatCoupDEnvoi(t *testing.T, db *sql.DB, matchID, xuid string) (sql.NullTime, string, int) {
	t.Helper()
	var rst sql.NullTime
	var src string
	var tp int
	if err := db.QueryRowContext(context.Background(), `
		SELECT mr.real_start_time, COALESCE(mr.real_start_source, ''), mp.time_played_seconds
		FROM match_registry mr JOIN match_participants mp ON mp.match_id = mr.match_id
		WHERE mr.match_id = ? AND mp.xuid = ?`, matchID, xuid).Scan(&rst, &src, &tp); err != nil {
		t.Fatalf("relecture %s: %v", matchID, err)
	}
	return rst, src, tp
}

// depsSurBase : lecture et writer sur la même base de test, writer compté.
func depsSurBase(db *sql.DB, acquis *int) Deps {
	return Deps{
		Gamertag: "testeur", TitleSlug: "halo_infinite",
		WithRead: func(ctx context.Context, _ string, fn func(*sql.DB)) { fn(db) },
		AcquireWriter: func(context.Context) (*sql.DB, func(), error) {
			*acquis++
			return db, func() {}, nil
		},
	}
}

func lu(matchID string, t0FilmMs *int64) artefactLu {
	return artefactLu{matchID: matchID, doc: &replay.ReplayDocument{MatchID: matchID, T0FilmMs: t0FilmMs}}
}

// Le rejeu publie un coup d'envoi : il remplace celui de l'estimation, avec sa source, et le temps de
// jeu des joueurs se recalcule dessus ; la même publication rejouée n'écrit plus rien.
func TestCoupDEnvoi_LeRejeuCorrigeUneFoisPuisPlusRien(t *testing.T) {
	db := baseRegistre(t)
	start := time.Date(2026, 9, 2, 18, 0, 0, 0, time.UTC)
	inscrireAuRegistre(t, db, "m-film", start, 0)
	poserDuree(t, db, "m-film", 600)
	inscrireJoueur(t, db, "m-film", "2603225103667623", start.Add(time.Second), 599)
	inscrireJoueur(t, db, "m-film", "2547408409520575", start.Add(1200*time.Millisecond), 599)

	t0 := int64(26_304)
	acquis := 0
	d := depsSurBase(db, &acquis)
	b := &bilanDerivations{}
	ecrireCoupsDEnvoi(context.Background(), d, b, preparerCoupsDEnvoi(context.Background(), d, b, []artefactLu{lu("m-film", &t0)}))
	rst, src, tp := etatCoupDEnvoi(t, db, "m-film", "2603225103667623")
	if !rst.Valid || !rst.Time.UTC().Equal(start.Add(26304*time.Millisecond)) || src != "film" {
		t.Fatalf("registre = (%v, %q), attendu le coup d'envoi du rejeu", rst, src)
	}
	if tp != 573 {
		t.Fatalf("temps de jeu = %d, attendu 573 (600 s - 26,304 s, tronque)", tp)
	}
	if acquis != 1 || b.aEchoue("m-film") {
		t.Fatalf("writer acquis %d fois, echec=%v ; attendu 1 et aucun echec", acquis, b.aEchoue("m-film"))
	}

	acquis = 0
	ecrireCoupsDEnvoi(context.Background(), d, b, preparerCoupsDEnvoi(context.Background(), d, b, []artefactLu{lu("m-film", &t0)}))
	if acquis != 0 {
		t.Fatalf("seconde publication : writer acquis %d fois, attendu 0 (rien ne change)", acquis)
	}
}

// Un rejeu sans coup d'envoi publié : le match prend la source suivante (l'estimation), jamais une
// valeur de l'ancien rejeu.
func TestCoupDEnvoi_RejeuSansCoupDEnvoiPrendLEstimation(t *testing.T) {
	db := baseRegistre(t)
	start := time.Date(2026, 9, 2, 18, 0, 0, 0, time.UTC)
	inscrireAuRegistre(t, db, "m-api", start, 0)
	poserDuree(t, db, "m-api", 600)
	inscrireJoueur(t, db, "m-api", "2603225103667623", start.Add(20*time.Second), 600)
	inscrireJoueur(t, db, "m-api", "2547408409520575", start.Add(20500*time.Millisecond), 600)

	acquis := 0
	d := depsSurBase(db, &acquis)
	b := &bilanDerivations{}
	ecrireCoupsDEnvoi(context.Background(), d, b, preparerCoupsDEnvoi(context.Background(), d, b, []artefactLu{lu("m-api", nil)}))
	rst, src, tp := etatCoupDEnvoi(t, db, "m-api", "2547408409520575")
	if !rst.Valid || !rst.Time.UTC().Equal(start.Add(20*time.Second)) || src != "api" {
		t.Fatalf("registre = (%v, %q), attendu l'estimation (premiere arrivee, source api)", rst, src)
	}
	if tp != 579 {
		t.Fatalf("temps de jeu = %d, attendu 579", tp)
	}
}

// Le report corrige, il ne crée jamais : un rejeu dont le match n'est pas au registre n'est pas un
// échec (la dérivation se marque, les familles append-only ne se rejouent pas à chaque cycle), et aucun
// writer n'est pris pour lui.
func TestCoupDEnvoi_MatchAbsentDuRegistre(t *testing.T) {
	db := baseRegistre(t)
	acquis := 0
	d := depsSurBase(db, &acquis)
	b := &bilanDerivations{}
	t0 := int64(26_304)
	ecrireCoupsDEnvoi(context.Background(), d, b, preparerCoupsDEnvoi(context.Background(), d, b, []artefactLu{lu("inconnu", &t0)}))
	if b.aEchoue("inconnu") || acquis != 0 {
		t.Fatalf("echec=%v writer=%d ; attendu aucun echec et aucun writer", b.aEchoue("inconnu"), acquis)
	}
	var n int
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM match_registry`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("lignes = %d (err %v), attendu 0", n, err)
	}
}

// Entre la lecture de l'état du match (segment de lecture) et son écriture (segment d'écriture),
// le rattrapage peut avoir écrit : le plan lu est périmé. La résolution se refait sous le segment
// d'écriture, sur ce que la base porte alors. Ici le rattrapage a posé, entre les deux, exactement
// ce que la publication allait écrire : la résolution fraîche ne trouve plus rien à changer, aucun
// match n'est écrit (un plan périmé l'aurait réécrit).
func TestCoupDEnvoi_ResoutSousLeWriterQuandUnAutreAEcritEntreLesDeux(t *testing.T) {
	db := baseRegistre(t)
	start := time.Date(2026, 9, 2, 18, 0, 0, 0, time.UTC)
	inscrireAuRegistre(t, db, "m-course", start, 0)
	poserDuree(t, db, "m-course", 600)
	inscrireJoueur(t, db, "m-course", "2603225103667623", start.Add(time.Second), 599)

	acquis := 0
	d := depsSurBase(db, &acquis)
	lectureFaite := d.WithRead
	d.WithRead = func(ctx context.Context, step string, fn func(*sql.DB)) {
		lectureFaite(ctx, step, fn)
		// Le rattrapage écrit ce que la publication s'apprête à écrire (coup d'envoi du rejeu, temps de jeu).
		if _, err := db.ExecContext(ctx, `UPDATE match_registry SET real_start_time = ?, real_start_source = 'film'
			WHERE match_id = 'm-course'`, start.Add(26304*time.Millisecond)); err != nil {
			t.Errorf("écriture intercalée: %v", err)
		}
		if _, err := db.ExecContext(ctx, `UPDATE match_participants SET time_played_seconds = 573
			WHERE match_id = 'm-course'`); err != nil {
			t.Errorf("écriture intercalée: %v", err)
		}
	}
	b := &bilanDerivations{}
	t0 := int64(26_304)
	avant := observability.LoadCounterT("halo_infinite", CompteurCoupDEnvoiEcrits)
	ecrireCoupsDEnvoi(context.Background(), d, b, preparerCoupsDEnvoi(context.Background(), d, b, []artefactLu{lu("m-course", &t0)}))
	if got := observability.LoadCounterT("halo_infinite", CompteurCoupDEnvoiEcrits) - avant; got != 0 {
		t.Errorf("matchs écrits = %d, attendu 0 : le plan lu avant l'écriture intercalée était périmé", got)
	}
	if b.aEchoue("m-course") {
		t.Error("le match est en échec")
	}
	if _, src, tp := etatCoupDEnvoi(t, db, "m-course", "2603225103667623"); src != "film" || tp != 573 {
		t.Errorf("état = (%q, %d), attendu (film, 573)", src, tp)
	}
}
