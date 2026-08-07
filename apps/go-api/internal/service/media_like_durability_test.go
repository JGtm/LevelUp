//go:build cgo

// media_like_durability_test.go — garde-rail de DURABILITÉ du like média
// (plan v7.3 lot 2, item 1.5).
//
// LA CLASSE DE RÉGRESSION QUE CE FICHIER VERROUILLE. Le like emprunte en
// production le chemin ATOMIQUE (WriterAcquirer câblé, cf. wire/registry_media.go) :
// une transaction unique sur shared_social qui écrit l'event append-only
// media_likes_history du liker. Si cette transaction se contente d'un
// tx.Commit(), la donnée reste dans le WAL jusqu'au CHECKPOINT périodique
// (5 min, cmd/server/main.go) ou au shutdown. Dans cette fenêtre, le serveur
// répond 200, l'UI affiche le cœur plein — puis tout redémarrage (un push sur
// main déploie en prod) ou toute quarantaine de WAL orphelin (ADR 0021) fait
// disparaître le like. Symptôme vécu : « le like ne tient pas », plusieurs fois.
//
// Le test ne teste donc PAS le happy path (déjà couvert ailleurs). Il vérifie les
// deux moitiés de la garantie, à l'instant où le client reçoit son 200 :
//  1. DURABILITÉ — le WAL est vide : ce qu'un arrêt brutal emporterait ne
//     contient plus le like. Avec CommitWithCheckpoint le WAL est flushé ; avec un
//     tx.Commit() nu il pèse ~600 octets et cette assertion devient rouge.
//  2. LECTURE-APRÈS-ÉCRITURE — une relecture immédiate (celle que déclenche le
//     client via BumpMediaFeedVersion) voit bien le like du liker dans la vue
//     append-only media_likes_latest.
//
// La vérification est faite À CHAUD, base ouverte : fermer la base déclencherait
// un CHECKPOINT automatique qui masquerait précisément la régression traquée.
//
// Note : le reste du chemin atomique (existence du média, rollback, absence de
// fuite de lease, isolation entre viewers) est couvert par
// media_service_atomic_integration_test.go, RÉACTIVÉ le 2026-08-04.
package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/platform/duckdb"
)

// socialDBForLikeTest crée une shared_social minimale (schéma de prod pour les
// likes) contenant un média au format canonique stocké {owner_slug}/{rel}.
func socialDBForLikeTest(t *testing.T) (*duckdb.DB, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "shared_social.duckdb")
	db, err := duckdb.OpenReadWriteShared(path, "")
	if err != nil {
		t.Fatalf("OpenReadWriteShared: %v", err)
	}
	ddl := `
		CREATE SEQUENCE IF NOT EXISTS media_id_seq START 1;
		CREATE TABLE media_files (
			id INTEGER DEFAULT nextval('media_id_seq') PRIMARY KEY,
			player_slug VARCHAR, file_path VARCHAR NOT NULL UNIQUE, file_name VARCHAR,
			kind VARCHAR DEFAULT 'video', thumbnail_path VARCHAR, status VARCHAR,
			capture_start_utc TIMESTAMP WITH TIME ZONE, capture_end_utc TIMESTAMP WITH TIME ZONE,
			mtime TIMESTAMP WITH TIME ZONE, indexed_at TIMESTAMP WITH TIME ZONE DEFAULT now()
		);
		CREATE SEQUENCE IF NOT EXISTS media_likes_history_id_seq START 1;
		CREATE TABLE media_likes_history (
			id BIGINT PRIMARY KEY DEFAULT nextval('media_likes_history_id_seq'),
			media_path VARCHAR NOT NULL, liker_slug VARCHAR NOT NULL, liker_gamertag VARCHAR,
			is_liked BOOLEAN NOT NULL, liked_at TIMESTAMP,
			written_at TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		);
		CREATE OR REPLACE VIEW media_likes_latest AS
			SELECT id, media_path, liker_slug, liker_gamertag, is_liked, liked_at, written_at
			FROM media_likes_history
			QUALIFY ROW_NUMBER() OVER (PARTITION BY media_path, liker_slug
				ORDER BY written_at DESC, id DESC) = 1;
		-- Substrat d'associations média-match : requis dès qu'un test LIT la
		-- galerie (le pipeline Q37 joint media_match_associations_latest), pas
		-- seulement quand il écrit un like.
		CREATE SEQUENCE IF NOT EXISTS media_match_associations_history_id_seq START 1;
		CREATE TABLE media_match_associations_history (
			id BIGINT PRIMARY KEY DEFAULT nextval('media_match_associations_history_id_seq'),
			media_file_id BIGINT NOT NULL, match_id VARCHAR NOT NULL, delta_seconds INTEGER,
			is_manual BOOLEAN NOT NULL DEFAULT FALSE, is_active BOOLEAN NOT NULL DEFAULT TRUE,
			associated_at TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			written_at TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		);
		CREATE OR REPLACE VIEW media_match_associations_latest AS
			WITH lpp AS (
				SELECT media_file_id, match_id, delta_seconds, is_manual, is_active, associated_at, written_at,
					ROW_NUMBER() OVER (PARTITION BY media_file_id, match_id ORDER BY written_at DESC, id DESC) AS rn
				FROM media_match_associations_history),
			act AS (SELECT * FROM lpp WHERE rn = 1 AND is_active = TRUE),
			hm AS (SELECT media_file_id, bool_or(is_manual) AS has_manual FROM act GROUP BY media_file_id)
			SELECT a.media_file_id, a.match_id, a.delta_seconds, a.is_manual, a.associated_at, a.written_at
			FROM act a JOIN hm ON hm.media_file_id = a.media_file_id
			WHERE a.is_manual = hm.has_manual;
		INSERT INTO media_files (player_slug, file_path, file_name)
		VALUES ('JGtm', 'JGtm/clip.mp4', 'clip.mp4');
	`
	if _, err := db.SQLDb().Exec(ddl); err != nil {
		t.Fatalf("seed shared_social: %v", err)
	}
	// CHECKPOINT initial : le seed appartient au fichier, pas au WAL — sinon le
	// test perdrait aussi media_files en supprimant le WAL, et ne prouverait rien.
	if _, err := db.SQLDb().Exec("CHECKPOINT"); err != nil {
		t.Fatalf("checkpoint seed: %v", err)
	}
	return db, path
}

// TestMediaService_SetMediaLike_Atomic_SurvivesWALLoss : après un SetMediaLike
// ayant répondu OK, la perte du WAL non flushé ne doit PAS faire disparaître le
// like : son event social doit être sur disque au moment de la réponse.
func TestMediaService_SetMediaLike_Atomic_SurvivesWALLoss(t *testing.T) {
	db, path := socialDBForLikeTest(t)
	ctx := context.Background()

	pdb := &duckdb.PlayerDB{SharedSocial: db, Gamertag: "JGtm"}
	svc := NewMediaService(duckdb.NewMediaRepo(pdb), "",
		WithMediaWriterAcquirer(func() (*dblease.LeasedWriter, error) {
			return pdb.AcquireSharedSocialWriterTimeout(dblease.SharedLeaseTimeout)
		}))

	resp, err := svc.SetMediaLike(ctx, domain.MediaLikeRequest{
		FilePath:      "JGtm/clip.mp4",
		LikerSlug:     "Chocoboflor",
		LikerGamertag: "Chocoboflor",
		Liked:         true,
	})
	if err != nil {
		t.Fatalf("SetMediaLike: %v", err)
	}
	if !resp.Liked {
		t.Fatalf("réponse Liked=false alors que le like a été accepté")
	}

	// ── Invariant mesuré, À CHAUD (DB toujours ouverte) ────────────────────────
	// On ne peut pas fermer la base pour vérifier : un Close propre CHECKPOINT de
	// lui-même et masquerait la régression. On observe donc l'état du disque à
	// l'instant où le client reçoit son 200, exactement comme le verrait un arrêt
	// brutal : ce qui n'est pas encore dans le fichier de base est perdu.
	//
	// WAL non vide ici = le like n'existe que dans le journal, en attente du
	// scheduler 5 min → c'est la fenêtre de disparition.
	if st, err := os.Stat(path + ".wal"); err == nil && st.Size() > 0 {
		t.Errorf("WAL non flushé (%d octets) juste après la réponse : le like n'est "+
			"pas durable — utiliser LeasedWriter.CommitWithCheckpoint, pas tx.Commit", st.Size())
	}

	// ── Lecture-après-écriture IMMÉDIATE ───────────────────────────────────────
	// Le like bumpe la version du flux médias (BumpMediaFeedVersion), que le
	// client interroge toutes les 10 s avant d'invalider son cache : une relecture
	// suit donc de près chaque like. Elle doit voir l'état écrit, sinon le cœur
	// « saute » tout seul à l'écran.
	//
	// La relecture porte sur la vue append-only media_likes_latest (ADR 0026 —
	// jamais la table brute) POUR LE LIKER : depuis le passage au like par-viewer
	// (2026-08-04), c'est l'unique support de l'état du cœur. L'ancienne assertion
	// sur media_files.liked est morte avec la colonne globale qu'elle observait.
	var events int
	if err := db.SQLDb().QueryRow(
		`SELECT COUNT(*) FROM media_likes_latest
		 WHERE media_path = ? AND liker_slug = ? AND is_liked = TRUE`,
		"JGtm/clip.mp4", "Chocoboflor",
	).Scan(&events); err != nil {
		t.Fatalf("relecture media_likes_latest: %v", err)
	}
	if events != 1 {
		t.Errorf("relecture immédiate : %d event(s) de like pour Chocoboflor, want 1", events)
	}

	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}
