package citationsfilm

// relance.go — LE RECALCUL DES CITATIONS DU FILM D'UN MATCH POUR UN JOUEUR, HORS DU CYCLE DE
// SYNCHRONISATION : dès que les dérivations du film sont écrites, plutôt qu'au prochain post-sync
// de chacun des joueurs suivis du match.
//
// Les règles d'écriture sont celles du post-sync : le lease writer de la base joueur
// (dblease.KindPlayer) est tenu pendant le calcul, la base partagée n'est que LUE (segment prêté
// par l'appelant), les citations s'écrivent en append-only (génération par match). Un joueur dont
// la base est tenue (son post-sync tourne) n'est pas attendu au-delà d'une borne : son post-sync
// recalcule de lui-même, par la même règle ([SelectMatchesFilmPlusRecent]).

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/dblease"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
)

// AttenteBaseJoueur borne l'attente du lease writer d'une base joueur. Au-delà, le post-sync du
// joueur la tient et recalculera lui-même.
const AttenteBaseJoueur = 20 * time.Second

// RelanceDeps : les ressources d'une relance, fournies par le câblage du serveur.
type RelanceDeps struct {
	// LireShared prête un segment de LECTURE de la base partagée du titre. Il n'appelle pas fn
	// quand la base est indisponible (et le journalise).
	LireShared func(ctx context.Context, step string, fn func(sharedDB *sql.DB))
	// MetaDB rend la base metadata du titre déjà ouverte par le processus, ou nil.
	MetaDB func() *sql.DB
	// LireJoueur ouvre la base joueur en LECTURE, même tenue en écriture par le processus
	// (duckdb.OpenReadForQuery) : la pré-vérification n'attend aucun lease.
	LireJoueur func(path string, timezone ...string) (*sql.DB, func(), error)
	// OuvrirJoueur ouvre la base joueur en écriture (schéma posé).
	OuvrirJoueur func(path string) (*duckdbpkg.DB, error)
	// Calculer recalcule les citations de ces matchs pour ce joueur (sync.RecalculerCitations).
	Calculer func(ctx context.Context, metaDB, sharedDB, playerDB *sql.DB, xuid string, matchIDs []string) error
	// Attente : borne du lease joueur ; 0 = [AttenteBaseJoueur].
	Attente time.Duration
}

// RelanceJoueur : le joueur suivi dont on relance les citations.
type RelanceJoueur struct {
	XUID         string
	PlayerDBPath string
}

// ParticipantsDuMatch rend les xuid des participants d'un match (base partagée).
func ParticipantsDuMatch(ctx context.Context, sharedDB *sql.DB, matchID string) ([]string, error) {
	rows, err := sharedDB.QueryContext(ctx,
		`SELECT DISTINCT xuid FROM match_participants WHERE match_id = ? AND xuid IS NOT NULL`, matchID)
	if err != nil {
		return nil, fmt.Errorf("citations: participants de %s: %w", matchID, err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var x string
		if err := rows.Scan(&x); err != nil {
			return nil, fmt.Errorf("citations: participants de %s (scan): %w", matchID, err)
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

// RelancerPourJoueur recalcule les citations du match pour ce joueur si son film a été dérivé
// après elles ([FilmPlusRecentPourLeMatch]).
func RelancerPourJoueur(ctx context.Context, d RelanceDeps, j RelanceJoueur, matchID string) (domain.CitationsRelanceIssue, error) {
	metaDB := d.MetaDB()
	if metaDB == nil {
		return "", errors.New("citations: base metadata non ouverte par le processus")
	}
	// PRÉ-VÉRIFICATION EN LECTURE, SANS LEASE : le cas courant d'une relance sans objet (aucune
	// citation encore, ou déjà à jour) ne doit ni attendre la base d'un joueur dont le post-sync
	// tourne, ni journaliser une attente vaine. La règle est rejouée sous le lease avant d'écrire.
	aFaire, err := preVerifier(ctx, d, j, matchID)
	if err != nil {
		return "", err
	}
	if !aFaire {
		return domain.CitationsRelanceAJour, nil
	}
	attente := d.Attente
	if attente <= 0 {
		attente = AttenteBaseJoueur
	}
	lctx, annuler := context.WithTimeout(ctx, attente)
	lease, err := dblease.AcquireWriterCtx(lctx, nil, j.PlayerDBPath, dblease.KindPlayer)
	annuler()
	if err != nil {
		if errors.Is(err, dblease.ErrDBLocked) && ctx.Err() == nil {
			return domain.CitationsRelanceJoueurOccupe, nil
		}
		return "", fmt.Errorf("citations: lease de la base joueur: %w", err)
	}
	defer lease.Release()
	handle, err := d.OuvrirJoueur(j.PlayerDBPath)
	if err != nil {
		return "", fmt.Errorf("citations: ouverture de la base joueur: %w", err)
	}
	defer handle.Close()

	issue, err := domain.CitationsRelanceAJour, error(nil)
	lu := false
	d.LireShared(ctx, "citations_relance", func(sharedDB *sql.DB) {
		lu = true
		plusRecent, perr := FilmPlusRecentPourLeMatch(ctx, handle.SQLDb(), sharedDB, matchID)
		if perr != nil || !plusRecent {
			err = perr
			return
		}
		if err = d.Calculer(ctx, metaDB, sharedDB, handle.SQLDb(), j.XUID, []string{matchID}); err == nil {
			issue = domain.CitationsRelanceRecalculee
		}
	})
	if !lu {
		return "", errors.New("citations: base partagée indisponible")
	}
	if err != nil {
		return "", err
	}
	return issue, nil
}

// preVerifier rejoue [FilmPlusRecentPourLeMatch] sur une lecture de la base joueur, sans lease.
func preVerifier(ctx context.Context, d RelanceDeps, j RelanceJoueur, matchID string) (bool, error) {
	playerDB, fermer, err := d.LireJoueur(j.PlayerDBPath)
	if err != nil {
		return false, fmt.Errorf("citations: lecture de la base joueur: %w", err)
	}
	defer fermer()
	aFaire, lu := false, false
	d.LireShared(ctx, "citations_relance_preverif", func(sharedDB *sql.DB) {
		lu = true
		aFaire, err = FilmPlusRecentPourLeMatch(ctx, playerDB, sharedDB, matchID)
	})
	if !lu {
		return false, errors.New("citations: base partagée indisponible")
	}
	return aFaire, err
}
