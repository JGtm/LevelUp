//go:build integration

// media_service_atomic_integration_test.go — tests d'intégration du CHEMIN
// NOMINAL du like média : la transaction atomique sous LeasedWriter, sur une
// vraie base DuckDB.
//
// ─── RÉACTIVÉ LE 2026-08-04 ─────────────────────────────────────────────────
//
// Ce fichier a passé près de trois mois derrière un tag `atomic_legacy` posé le
// 2026-05-15 « le temps de le migrer vers la nouvelle API LeasedWriter ». Entre
// les deux, le chemin qu'il couvre — celui que la PRODUCTION emprunte, puisque
// wire/registry_media.go câble toujours WithMediaWriterAcquirer — n'avait plus
// aucun test actif, et trois régressions du like s'y sont succédé. Un test
// désactivé « temporairement » ne protège de rien : il donne seulement
// l'impression que la couverture existe.
//
// La migration due : dblease.NewLeasedWriter → PlayerDB.AcquireSharedSocialWriter*,
// LeasedWriter.Executor devenu privé, MediaRepo qui prend un *PlayerDB, et
// SetMediaLike qui retourne (*MediaLikeResponse, error) et non (bool, error).
//
// Les assertions ont aussi été portées à la sémantique PAR VIEWER : le like
// n'écrit plus de booléen global sur media_files, il ajoute un event par liker
// dans media_likes_history (lu via media_likes_latest, règle ART n°2).
//
// Lancer : go test -tags=integration ./internal/service/ -run TestMediaServiceAtomic
package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
)

// atomicLikeFixture monte le montage de PRODUCTION : shared_social réelle,
// MediaRepo réel, acquéreur de writer réel. Retourne aussi le PlayerDB pour que
// les tests puissent vérifier l'état du lease après coup.
func atomicLikeFixture(t *testing.T) (*duckdb.DB, *duckdb.PlayerDB) {
	t.Helper()
	db, _ := socialDBForLikeTest(t)
	return db, &duckdb.PlayerDB{SharedSocial: db, Gamertag: "JGtm"}
}

func acquirerFor(pdb *duckdb.PlayerDB) func() (*dblease.LeasedWriter, error) {
	return func() (*dblease.LeasedWriter, error) {
		return pdb.AcquireSharedSocialWriterTimeout(dblease.SharedLeaseTimeout)
	}
}

// countLikeEvents compte les events de like ACTIFS d'un liker sur un média, vus
// par la vue append-only (jamais la table brute).
func countLikeEvents(t *testing.T, db *duckdb.DB, mediaPath, likerSlug string) int {
	t.Helper()
	var n int
	if err := db.SQLDb().QueryRow(
		`SELECT COUNT(*) FROM media_likes_latest
		 WHERE media_path = ? AND liker_slug = ? AND is_liked = TRUE`,
		mediaPath, likerSlug,
	).Scan(&n); err != nil {
		t.Fatalf("count media_likes_latest: %v", err)
	}
	return n
}

// countAllEvents compte TOUS les events écrits (table brute) — sert uniquement à
// prouver qu'AUCUNE ligne n'a été ajoutée dans les cas d'échec. C'est la seule
// raison légitime de lire la table brute : on y vérifie une absence.
func countAllEvents(t *testing.T, db *duckdb.DB) int {
	t.Helper()
	var n int
	if err := db.SQLDb().QueryRow(`SELECT COUNT(*) FROM media_likes_history`).Scan(&n); err != nil {
		t.Fatalf("count media_likes_history: %v", err)
	}
	return n
}

// TestMediaServiceAtomic_Success : happy path du chemin de prod. Le like est
// commité et n'appartient QU'À son liker.
func TestMediaServiceAtomic_Success(t *testing.T) {
	db, pdb := atomicLikeFixture(t)
	svc := NewMediaService(duckdb.NewMediaRepo(pdb), "", WithMediaWriterAcquirer(acquirerFor(pdb)))

	resp, err := svc.SetMediaLike(context.Background(), domain.MediaLikeRequest{
		FilePath:      "JGtm/clip.mp4",
		LikerSlug:     "Chocoboflor",
		LikerGamertag: "Chocoboflor",
		Liked:         true,
	})
	if err != nil {
		t.Fatalf("SetMediaLike: %v", err)
	}
	if !resp.Liked || resp.TotalLikers != 1 {
		t.Fatalf("réponse = %+v, want Liked=true TotalLikers=1", resp)
	}
	if got := countLikeEvents(t, db, "JGtm/clip.mp4", "Chocoboflor"); got != 1 {
		t.Errorf("events de like pour le liker = %d, want 1", got)
	}
	// Le like ne déborde pas sur un autre joueur : c'est tout l'objet du
	// passage au par-viewer.
	if got := countLikeEvents(t, db, "JGtm/clip.mp4", "JGtm"); got != 0 {
		t.Errorf("le like de Chocoboflor a produit %d event(s) au nom de JGtm, want 0", got)
	}
}

// TestMediaServiceAtomic_UnknownMedia_WritesNothing : un like sur un file_path
// inconnu (ou supprimé) est un 404 et ne laisse AUCUNE trace.
//
// L'enjeu est plus fort qu'un simple code de retour : media_likes_history est
// append-only. Un event écrit par erreur ne pourrait plus jamais être retiré —
// il resterait à vie dans les compteurs sociaux d'un média fantôme.
func TestMediaServiceAtomic_UnknownMedia_WritesNothing(t *testing.T) {
	db, pdb := atomicLikeFixture(t)
	svc := NewMediaService(duckdb.NewMediaRepo(pdb), "", WithMediaWriterAcquirer(acquirerFor(pdb)))

	_, err := svc.SetMediaLike(context.Background(), domain.MediaLikeRequest{
		FilePath:  "JGtm/nexiste-pas.mp4",
		LikerSlug: "Chocoboflor",
		Liked:     true,
	})
	if err == nil {
		t.Fatal("attendu une erreur not_found sur un file_path inconnu")
	}
	var apiErr *domain.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "not_found" {
		t.Errorf("erreur = %v, want domain.APIError{Code: not_found}", err)
	}
	if got := countAllEvents(t, db); got != 0 {
		t.Errorf("%d event(s) écrit(s) pour un média inconnu, want 0 (table append-only : irrattrapable)", got)
	}
}

// TestMediaServiceAtomic_RepoError_RollsBack : une erreur au milieu de la
// transaction remonte au caller, ne commite rien, et ne fuit pas le writer.
func TestMediaServiceAtomic_RepoError_RollsBack(t *testing.T) {
	db, pdb := atomicLikeFixture(t)
	sentinel := errors.New("échec simulé en cours de transaction")
	svc := NewMediaService(&mockAtomicMediaRepo{atomicErr: sentinel}, "",
		WithMediaWriterAcquirer(acquirerFor(pdb)))

	_, err := svc.SetMediaLike(context.Background(), domain.MediaLikeRequest{
		FilePath:  "JGtm/clip.mp4",
		LikerSlug: "Chocoboflor",
		Liked:     true,
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("erreur = %v, want %v", err, sentinel)
	}
	if got := countAllEvents(t, db); got != 0 {
		t.Errorf("%d event(s) survivant(s) au rollback, want 0", got)
	}
	assertWriterAvailable(t, pdb, "après rollback")
}

// TestMediaServiceAtomic_PanicMidTx_ReleasesWriter : même une panique à
// mi-transaction ne doit pas emporter le writer avec elle.
//
// Sans libération, shared_social resterait verrouillée jusqu'au redémarrage :
// tout like, tout favori et toute écriture du sync suivante partiraient en 503.
// Le service ne récupère PAS la panique (elle doit remonter, c'est un bug) —
// il garantit seulement, via defer, que le lease est rendu au passage.
func TestMediaServiceAtomic_PanicMidTx_ReleasesWriter(t *testing.T) {
	_, pdb := atomicLikeFixture(t)
	svc := NewMediaService(&panicAtomicMediaRepo{}, "", WithMediaWriterAcquirer(acquirerFor(pdb)))

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("la panique doit remonter au caller (ne pas être avalée par le service)")
			}
		}()
		_, _ = svc.SetMediaLike(context.Background(), domain.MediaLikeRequest{
			FilePath:  "JGtm/clip.mp4",
			LikerSlug: "Chocoboflor",
			Liked:     true,
		})
	}()

	assertWriterAvailable(t, pdb, "après panique")
}

// TestMediaServiceAtomic_LeaseTimeout_NoLeak : quand le writer est déjà pris,
// l'erreur remonte telle quelle (le handler la traduit en 503 + Retry-After) et
// aucune transaction n'est ouverte.
func TestMediaServiceAtomic_LeaseTimeout_NoLeak(t *testing.T) {
	db, pdb := atomicLikeFixture(t)
	svc := NewMediaService(duckdb.NewMediaRepo(pdb), "",
		WithMediaWriterAcquirer(func() (*dblease.LeasedWriter, error) {
			return nil, fmt.Errorf("lease timeout: %w", dblease.ErrDBLocked)
		}))

	_, err := svc.SetMediaLike(context.Background(), domain.MediaLikeRequest{
		FilePath:  "JGtm/clip.mp4",
		LikerSlug: "Chocoboflor",
		Liked:     true,
	})
	if !errors.Is(err, dblease.ErrDBLocked) {
		t.Fatalf("erreur = %v, want ErrDBLocked (le handler en dépend pour son 503)", err)
	}
	if got := countAllEvents(t, db); got != 0 {
		t.Errorf("%d event(s) écrit(s) alors que le lease n'a jamais été obtenu, want 0", got)
	}
	assertWriterAvailable(t, pdb, "après échec d'acquisition")
}

// assertWriterAvailable prouve l'absence de fuite : le writer doit être
// ré-acquérable immédiatement.
func assertWriterAvailable(t *testing.T, pdb *duckdb.PlayerDB, when string) {
	t.Helper()
	w, err := pdb.AcquireSharedSocialWriterTimeout(dblease.SharedLeaseTimeout)
	if err != nil {
		t.Fatalf("writer non ré-acquérable %s (fuite de lease) : %v", when, err)
	}
	w.Release()
}

// panicAtomicMediaRepo — repo qui panique au cœur de la transaction.
type panicAtomicMediaRepo struct {
	mockAtomicMediaRepo
}

func (m *panicAtomicMediaRepo) SetMediaLikeAtomic(
	_ context.Context,
	_ port.DBExecutor,
	_, _, _ string,
	_ bool,
) (bool, error) {
	panic("panique simulée en cours de transaction")
}
