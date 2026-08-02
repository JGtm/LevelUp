//go:build cgo

// media_like_durability_test.go — garde-rail de DURABILITÉ du like média
// (plan v7.3 lot 2, item 1.5).
//
// LA CLASSE DE RÉGRESSION QUE CE FICHIER VERROUILLE. Le like emprunte en
// production le chemin ATOMIQUE (WriterAcquirer câblé, cf. wire/registry_media.go) :
// une transaction unique sur shared_social qui écrit media_files.liked +
// l'event append-only media_likes_history. Si cette transaction se contente d'un
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
//     client via BumpMediaFeedVersion) voit bien le like, côté media_files ET côté
//     vue append-only media_likes_latest.
//
// La vérification est faite À CHAUD, base ouverte : fermer la base déclencherait
// un CHECKPOINT automatique qui masquerait précisément la régression traquée.
//
// Note : le seul autre test du chemin atomique
// (media_service_atomic_integration_test.go) est DÉSACTIVÉ depuis 2026-05-15
// (build tag `atomic_legacy`) — d'où l'absence de filet sur ce chemin nominal.
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
			kind VARCHAR DEFAULT 'video', thumbnail_path VARCHAR,
			liked BOOLEAN DEFAULT FALSE, liked_at TIMESTAMP, status VARCHAR,
			mtime TIMESTAMP WITH TIME ZONE, indexed_at TIMESTAMP WITH TIME ZONE DEFAULT now()
		);
		CREATE SEQUENCE IF NOT EXISTS media_likes_history_id_seq START 1;
		CREATE TABLE media_likes_history (
			id BIGINT PRIMARY KEY DEFAULT nextval('media_likes_history_id_seq'),
			media_path VARCHAR NOT NULL, liker_slug VARCHAR NOT NULL, liker_gamertag VARCHAR,
			is_liked BOOLEAN NOT NULL, liked_at TIMESTAMP,
			written_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE OR REPLACE VIEW media_likes_latest AS
			SELECT id, media_path, liker_slug, liker_gamertag, is_liked, liked_at, written_at
			FROM media_likes_history
			QUALIFY ROW_NUMBER() OVER (PARTITION BY media_path, liker_slug
				ORDER BY written_at DESC, id DESC) = 1;
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
// like — ni son état media_files.liked, ni son event social.
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
	var liked bool
	if err := db.SQLDb().QueryRow(
		`SELECT liked FROM media_files WHERE file_path = ?`, "JGtm/clip.mp4",
	).Scan(&liked); err != nil {
		t.Fatalf("relecture media_files: %v", err)
	}
	if !liked {
		t.Error("relecture immédiate : media_files.liked = false juste après un like accepté")
	}

	// Volet social : l'event append-only doit être visible via la vue _latest
	// (ADR 0026 — jamais la table brute), sinon les badges « ♥ » restent vides
	// alors que le cœur est plein.
	var events int
	if err := db.SQLDb().QueryRow(
		`SELECT COUNT(*) FROM media_likes_latest WHERE media_path = ? AND is_liked = TRUE`,
		"JGtm/clip.mp4",
	).Scan(&events); err != nil {
		t.Fatalf("relecture media_likes_latest: %v", err)
	}
	if events != 1 {
		t.Errorf("relecture immédiate : %d event(s) dans media_likes_latest, want 1", events)
	}

	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}
