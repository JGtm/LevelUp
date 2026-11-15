package duckdb

// title_bases.go : l'accès, pour les tâches d'administration et de maintenance du serveur, aux bases
// d'UN titre (la base partagée en lecture, metadata en écriture partagée). Les handles sont
// compatibles B-swap (ADR 0016) : la base partagée se lit par le fournisseur quand il tient CE
// fichier (titre par défaut, seul pris en écriture dans le processus) — il draine les fenêtres
// RW <-> RO au lieu de forcer une ouverture RO qui échouerait en « different configuration » pendant
// une synchronisation. Pour les autres titres, aucune fenêtre RW n'existe dans le processus et
// OpenReadOnly reste correct. metadata passe par OpenReadWriteShared (handle du processus, partagé avec
// les catalogues et le prestige).

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"time"

	titlepkg "levelup/go-api/internal/domain/title"
)

// Budgets de l'acquisition d'une lecture partagée. Variables (et non constantes) pour être raccourcis
// par les tests de la reprise.
var (
	// sharedReadBudget borne l'acquisition résiliente d'une connexion de lecture partagée (toutes
	// tentatives confondues).
	sharedReadBudget = 45 * time.Second

	// sharedReadAttempt borne UNE tentative Get : le fournisseur peut attendre un retour RO le temps
	// d'un swap ; au-delà on réessaie.
	sharedReadAttempt = 10 * time.Second

	// sharedReadBackoff sépare deux tentatives Get.
	sharedReadBackoff = 500 * time.Millisecond
)

// AcquireSharedRead acquiert une connexion de lecture partagée, résiliente à la fenêtre de swap RO <-> RW
// d'une synchronisation concurrente (ADR 0016) : pendant qu'une synchronisation tient l'écrivain RW, un Get
// lecteur attend le retour RO. Plutôt que de renoncer au premier dépassement, on réessaie avec un délai
// jusqu'à sharedReadBudget.
//
// Le contexte est détaché du parent (context.WithoutCancel) : le post-sync est un traitement best-effort
// en arrière-plan qui ne doit pas être tué si le run de synchronisation appelant se termine et annule son
// contexte juste après.
//
// L'appelant DOIT appeler la fonction release rendue (typiquement par defer). Si err != nil, release est nil.
func AcquireSharedRead(parent context.Context, reader SharedReader) (*sql.DB, func(), error) {
	overall, cancelOverall := context.WithTimeout(context.WithoutCancel(parent), sharedReadBudget)
	var lastErr error
	for attempt := 1; ; attempt++ {
		attemptCtx, cancelAttempt := context.WithTimeout(overall, sharedReadAttempt)
		db, release, err := reader.Get(attemptCtx)
		if err == nil {
			return db, func() {
				release()
				cancelAttempt()
				cancelOverall()
			}, nil
		}
		cancelAttempt()
		lastErr = err
		select {
		case <-overall.Done():
			cancelOverall()
			return nil, nil, fmt.Errorf("shared reader unavailable after %d attempt(s): %w", attempt, lastErr)
		case <-time.After(sharedReadBackoff):
		}
	}
}

// SharedProvider est le fournisseur de la base partagée d'un titre (sharedprovider.Provider, que ce paquet
// ne peut pas importer : cycle) : une lecture résiliente et le chemin du fichier qu'il tient.
type SharedProvider interface {
	SharedReader
	// Path rend le chemin du fichier de la base partagée que le fournisseur tient.
	Path() string
}

// TitleBases ouvre les bases d'un titre. Une valeur se construit à l'appel (les chemins se lisent au
// moment de l'ouverture).
type TitleBases struct {
	// Paths résout les chemins des bases du titre.
	Paths *titlepkg.PathResolver
	// SharedProvider est le fournisseur de la base partagée du titre par défaut ; nil quand aucun n'est câblé.
	SharedProvider SharedProvider
	// Log reçoit l'avertissement d'une base metadata illisible.
	Log *slog.Logger
}

// TitleHandles : la base partagée (lecture) et metadata (écriture partagée) d'un titre, à rendre par Close.
type TitleHandles struct {
	// Shared est la base partagée du titre en lecture.
	Shared *sql.DB
	// Meta est metadata du titre ; nil quand elle est absente ou illisible (les fonctions appelées
	// dégradent explicitement).
	Meta  *sql.DB
	close func()
}

// Close décrémente les compteurs de références ; à appeler par defer.
func (h TitleHandles) Close() { h.close() }

// Open ouvre la base partagée (lecture) et metadata (écriture partagée) du titre. Une tâche qui va sur le
// réseau prend les deux séparément (ReadShared, Metadata) et rend la base partagée avant : un lecteur en
// vol fait échouer toute prise d'écriture (ADR 0016).
func (b TitleBases) Open(ctx context.Context, titleSlug string) (TitleHandles, error) {
	shared, sharedRelease, err := b.ReadShared(ctx, titleSlug)
	if err != nil {
		return TitleHandles{}, err
	}
	meta, metaRelease := b.Metadata(ctx, titleSlug)
	return TitleHandles{Shared: shared, Meta: meta, close: func() { sharedRelease(); metaRelease() }}, nil
}

// ReadShared ouvre la base partagée du titre en lecture ; release la rend.
func (b TitleBases) ReadShared(ctx context.Context, titleSlug string) (*sql.DB, func(), error) {
	sharedDBPath := b.Paths.SharedDBPath(titleSlug)
	if _, statErr := os.Stat(sharedDBPath); statErr != nil {
		return nil, nil, fmt.Errorf("shared DB absente pour %s: %w", titleSlug, statErr)
	}
	// Lecture partagée B-swap-safe : si le fournisseur tient CE fichier (titre par défaut), passer par lui
	// (drain RO <-> RW, résilient à la synchronisation concurrente) ; sinon OpenReadOnly direct (aucune
	// fenêtre RW dans le processus pour ce titre).
	if b.SharedProvider != nil && b.SharedProvider.Path() == sharedDBPath {
		db, release, gerr := AcquireSharedRead(ctx, b.SharedProvider)
		if gerr != nil {
			return nil, nil, fmt.Errorf("acquire shared read (provider): %w", gerr)
		}
		return db, release, nil
	}
	shared, oerr := OpenReadOnly(sharedDBPath)
	if oerr != nil {
		return nil, nil, fmt.Errorf("open shared RO: %w", oerr)
	}
	return shared.SQLDb(), func() { _ = shared.Close() }, nil //nolint:errcheck // ref-count
}

// Metadata ouvre metadata du titre (écriture partagée, handle du processus compté par références) ; release
// le rend. Absente ou illisible : nil (journalisé), jamais d'erreur — les appelants dégradent.
func (b TitleBases) Metadata(ctx context.Context, titleSlug string) (*sql.DB, func()) {
	metaPath := b.Paths.MetadataDBPath(titleSlug)
	if _, statErr := os.Stat(metaPath); statErr != nil {
		return nil, func() {}
	}
	meta, openErr := OpenReadWriteShared(metaPath)
	if openErr != nil {
		b.Log.WarnContext(ctx, "data_quality: metadata indisponible",
			"title", titleSlug, "err", openErr)
		return nil, func() {}
	}
	return meta.SQLDb(), func() { _ = meta.Close() } //nolint:errcheck // ref-count
}

// CachedSQLDB rend la fonction qui donne le handle SQL de la base située à path quand le processus la tient
// déjà ouverte (emprunt non possédant, jamais fermé par l'appelant), sinon nil.
func CachedSQLDB(path string) func() *sql.DB {
	return func() *sql.DB {
		if db, ok := LookupCachedDB(path); ok {
			return db.SQLDb()
		}
		return nil
	}
}

// ReadHandles : la base du joueur et la base partagée de son titre en lecture, pour un contrôle qui lit les
// deux. À rendre par Release.
type ReadHandles struct {
	// Player est la base du joueur.
	Player *sql.DB
	// Shared est la base partagée du titre, lue par son fournisseur (résilient au swap, ADR 0016).
	Shared  *sql.DB
	release func()
}

// Release rend la lecture partagée ; à appeler après usage de Shared.
func (h ReadHandles) Release() { h.release() }

// ReadHandles ouvre la lecture partagée du joueur et rend ses deux bases. pdb.Player doit être renseigné.
func (pdb *PlayerDB) ReadHandles(ctx context.Context) (ReadHandles, error) {
	shared, release, err := pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return ReadHandles{}, err
	}
	return ReadHandles{Player: pdb.Player.SQLDb(), Shared: shared, release: release}, nil
}
