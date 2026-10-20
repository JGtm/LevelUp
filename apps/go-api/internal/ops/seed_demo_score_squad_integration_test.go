//go:build integration

// Package ops — seed_demo_score_squad_integration_test.go : les trois tables que la vue match et
// l'Escouade lisent sans qu'elles soient dérivées du film traversent SeedDemo.
//
//   - personal_score_awards (base JOUEUR, append-only par génération) : score d'objectif du
//     tableau et radar de la vue match ; filtrée sur le corpus, xuid anonymisé, vue `_latest`
//     lisible et ne rendant que la dernière génération ;
//   - player_squad_offset (base PARTAGÉE, clé xuid) : synergies de l'Escouade ; seuls les couples
//     de joueurs ayant partagé un match du corpus entrent, les deux colonnes sont anonymisées ;
//   - player_assists_model (base JOUEUR, sans identité) : copiée entière.
package ops

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

const squadOutsideXUID = "3333333333333333" // jamais au corpus : son couple ne doit pas entrer

func seedScoreAndSquadInSource(t *testing.T, srcPlayer, srcShared string) {
	t.Helper()
	src, oth := "'"+filmSourceXUID+"'", "'"+filmOtherXUID+"'"
	player, err := sql.Open("duckdb", srcPlayer)
	if err != nil {
		t.Fatal(err)
	}
	defer player.Close()
	cols := `(id, match_id, xuid, award_name, award_category, award_count, award_score, generation_id, written_at, is_tombstone)`
	mustExec(t, player, `INSERT INTO personal_score_awards `+cols+` VALUES
		(1, 'm1', `+src+`, 'flag_capture', 'objective', 1, 100, 1, `+oldPassAt+`, false),
		(2, 'm1', `+src+`, 'flag_capture', 'objective', 2, 200, 2, `+newPassAt+`, false),
		(3, 'm3', `+src+`, 'flag_capture', 'objective', 9, 900, 2, `+newPassAt+`, false)`)
	mustExec(t, player, `INSERT INTO player_assists_model (game_variant_name, coef_intercept, coef_kills, coef_deaths,
		coef_damage_dealt, coef_damage_taken, coef_mmr_delta, r2, n_samples) VALUES ('Slayer', 1, 0.1, 0.1, 0, 0, 0, 0.5, 40)`)

	shared, err := sql.Open("duckdb", srcShared)
	if err != nil {
		t.Fatal(err)
	}
	defer shared.Close()
	offCols := `(id, xuid, partner_xuid, playlist_group, offset_value, match_count, source, written_at)`
	mustExec(t, shared, `INSERT INTO player_squad_offset `+offCols+` VALUES
		(1, `+src+`, `+oth+`, 'ranked', 0.10, 5, 'lusr', `+oldPassAt+`),
		(2, `+src+`, `+oth+`, 'ranked', 0.25, 9, 'lusr', `+newPassAt+`),
		(3, `+oth+`, `+src+`, 'ranked', 0.30, 9, 'lusr', `+newPassAt+`),
		(4, `+src+`, '`+squadOutsideXUID+`', 'ranked', 0.40, 9, 'lusr', `+newPassAt+`)`)
}

func TestSeedDemo_ScoreObjectifEscouadeEtModeleAssists(t *testing.T) {
	tmpDir, srcPlayer, srcShared, srcMeta := seedSourceDBs(t)
	seedScoreAndSquadInSource(t, srcPlayer, srcShared)
	outDir := filepath.Join(tmpDir, "out")
	res, err := SeedDemo(context.Background(), SeedDemoOptions{
		SourcePlayerDB: srcPlayer, SourceSharedDB: srcShared, SourceMetaDB: srcMeta,
		SourceXUID: filmSourceXUID, OutDir: outDir, MaxMatches: 2, SourceLabel: "JGtm", ServiceTag: "SPTA",
	})
	if err != nil {
		t.Fatalf("SeedDemo (le contrôle des valeurs de fin de seed a joué) : %v", err)
	}
	if res.SharedRows["player_squad_offset"] != 3 {
		t.Errorf("player_squad_offset : %d ligne(s) extraite(s), attendu 3 (deux générations du couple + le couple inverse ; le couple hors corpus exclu)",
			res.SharedRows["player_squad_offset"])
	}
	if res.PlayerRows["personal_score_awards"] != 2 || res.PlayerRows["player_assists_model"] != 1 {
		t.Errorf("lignes joueur = %v, attendu personal_score_awards=2 (m1, deux générations ; m3 hors corpus) et player_assists_model=1", res.PlayerRows)
	}

	shared := openReadOnly(t, filepath.Join(outDir, "warehouse", "shared_matches_v2.duckdb"))
	defer shared.Close()
	var n int
	if err := shared.QueryRow(`SELECT count(*) FROM player_squad_offset_latest`).Scan(&n); err != nil || n != 2 {
		t.Errorf("player_squad_offset_latest : %d ligne(s), err %v, attendu 2 (une par sens du couple)", n, err)
	}
	var offset float64
	var partner string
	if err := shared.QueryRow(`SELECT offset_value, partner_xuid FROM player_squad_offset_latest WHERE xuid = ?`,
		DefaultDemoXUID).Scan(&offset, &partner); err != nil {
		t.Fatalf("synergie du joueur démo : %v", err)
	}
	if offset != 0.25 || partner == filmOtherXUID || partner == "" {
		t.Errorf("synergie = (%v, %q), attendu la dernière génération (0.25) et un partenaire à l'identité démo", offset, partner)
	}

	player := openReadOnly(t, filepath.Join(outDir, "players", DefaultDemoGamertag, "stats.duckdb"))
	defer player.Close()
	var score int
	if err := player.QueryRow(`SELECT COALESCE(SUM(award_score), 0)::INTEGER FROM personal_score_awards_latest
		WHERE award_category = 'objective' AND xuid = ? AND match_id = 'm1'`, DefaultDemoXUID).Scan(&score); err != nil {
		t.Fatalf("personal_score_awards_latest illisible dans la démo : %v", err)
	}
	if score != 200 {
		t.Errorf("score d'objectif de m1 = %d, attendu 200 (dernière génération pour le xuid démo, pas la périmée)", score)
	}
	if err := player.QueryRow(`SELECT count(*) FROM personal_score_awards WHERE match_id = 'm3' OR xuid = ?`,
		filmSourceXUID).Scan(&n); err != nil || n != 0 {
		t.Errorf("personal_score_awards : %d ligne(s) hors corpus ou au xuid réel (err %v)", n, err)
	}
}

func openReadOnly(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", path+"?access_mode=READ_ONLY")
	if err != nil {
		t.Fatal(err)
	}
	return db
}
