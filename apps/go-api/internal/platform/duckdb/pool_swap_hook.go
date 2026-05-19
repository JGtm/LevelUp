package duckdb

import (
	"context"
	"fmt"
	"log/slog"
)

// SwapDirection énumère les transitions du Provider sharedprovider que le pool
// écoute. Type local au package duckdb pour éviter un cycle d'import avec
// sharedprovider (qui importe duckdb).
//
// Le caller (main.go) traduit les sharedprovider.Direction vers ces valeurs
// au moment du Subscribe.
type SwapDirection int

const (
	// SwapDirPreSwapToRW correspond à sharedprovider.DirectionPreSwapToRW :
	// le Provider est sur le point de transitioner vers RW. Le pool doit
	// libérer son ATTACH RO sur shared (Reopen player + social + Close Shared).
	SwapDirPreSwapToRW SwapDirection = iota

	// SwapDirRWToRO correspond à sharedprovider.DirectionRWToRO : le swap RW
	// est terminé, on est revenu en RO. Le pool doit rouvrir shared et
	// re-attachShared sur ses conns player + social.
	SwapDirRWToRO

	// SwapDirErrorToRO correspond à sharedprovider.DirectionErrorToRO :
	// recovery depuis StateError via le retry loop. Mêmes actions que RWToRO.
	SwapDirErrorToRO
)

// PrepareForSharedSwap libère les ressources qui tiennent le fichier shared
// ouvert dans ce PlayerDB, pour permettre au Provider d'exécuter
// OpenReadWrite sur le même fichier sans "Unique file handle conflict" ni
// "different configuration".
//
// Mécanique :
//  1. Reopen pdb.Player : ferme la conn player avec son ATTACH RO interne,
//     rouvre vide (le ATTACH disparaît avec la conn DuckDB native).
//  2. Reopen pdb.SharedSocial (idem si présent).
//  3. Close pdb.Shared : ferme la conn RO sur shared pour libérer le file
//     handle DuckDB. Le pointeur reste valide mais les Query échouent avec
//     "sql: database is closed" jusqu'à RestoreSharedAfterSwap.
//
// IMPORTANT : si pdb.SharedReader est nil (mode legacy, pas inscrit au
// mécanisme B-swap), cette méthode est no-op — on ne touche à rien.
//
// Doit être appelée SYNCHRONIQUEMENT par OnSharedSwap depuis le callback
// Subscribe du Provider, AVANT que le Provider ne tente OpenReadWrite.
func (pdb *PlayerDB) PrepareForSharedSwap(ctx context.Context) error {
	if !pdb.bSwapEnabled {
		// Mode legacy : pdb.SharedReader est un LegacySharedReader (toujours
		// non-nil), pas un sharedprovider.Provider. Le pool n'est pas inscrit
		// au mécanisme B-swap — no-op.
		return nil
	}

	// ORDRE CRITIQUE (commit 8f) : on doit fermer pdb.Shared AVANT de Reopen
	// player. Sinon, l'auto-attach DuckDB-Go ré-attache shared sur la nouvelle
	// conn player (car shared est encore tenu par pdb.Shared).
	//
	// Le Provider a déjà fermé son propre handle RO en Phase 3 juste avant
	// cette notif. Avec pdb.Shared.Close ici, refCount du cache `ro:path`
	// tombe à 0 → file totalement libéré → Reopen player ne déclenche pas
	// d'auto-attach.
	if pdb.Shared != nil {
		if err := pdb.Shared.Close(); err != nil {
			slog.WarnContext(ctx, "PrepareForSharedSwap: Close Shared failed (continuing)",
				"gamertag", pdb.Gamertag, "err", err)
		}
	}
	if pdb.Player != nil {
		if err := pdb.Player.Reopen(); err != nil {
			return fmt.Errorf("PrepareForSharedSwap: Reopen player: %w", err)
		}
	}
	if pdb.SharedSocial != nil {
		if err := pdb.SharedSocial.Reopen(); err != nil {
			slog.WarnContext(ctx, "PrepareForSharedSwap: Reopen SharedSocial failed (continuing)",
				"gamertag", pdb.Gamertag, "err", err)
		}
	}
	// IMPORTANT (commit 8f) : DuckDB-Go auto-attache shared sur TOUTES les
	// conns DuckDB du process. Donc même pdb.Metadata (qui ne touche pas
	// directement à shared) doit être Reopen pour libérer son auto-attach,
	// sinon OpenReadWrite du Provider échouera avec
	// "Cannot attach shared_matches_v2 - already attached".
	if pdb.Metadata != nil {
		if err := pdb.Metadata.Reopen(); err != nil {
			slog.WarnContext(ctx, "PrepareForSharedSwap: Reopen Metadata failed (continuing)",
				"gamertag", pdb.Gamertag, "err", err)
		}
	}
	return nil
}

// RestoreSharedAfterSwap rouvre la conn RO sur shared et re-attache shared
// sur les conns player + social. Appelée sur SwapDirRWToRO ou
// SwapDirErrorToRO depuis OnSharedSwap.
//
// Pré-conditions : le Provider est revenu en StateRO, le fichier shared est
// re-accessible en RO. pdb.SharedReader doit être non-nil (sinon no-op).
//
// Si OpenReadOnly échoue : log Error mais ne propage pas (les Query
// suivantes échoueront proprement, le retry loop async du Provider pourra
// éventuellement déclencher une seconde tentative).
func (pdb *PlayerDB) RestoreSharedAfterSwap(ctx context.Context) error {
	if !pdb.bSwapEnabled {
		// Mode legacy : pdb.SharedReader est un LegacySharedReader (toujours
		// non-nil), pas un sharedprovider.Provider. Le pool n'est pas inscrit
		// au mécanisme B-swap — no-op.
		return nil
	}
	if pdb.sharedDBPath == "" {
		return fmt.Errorf("RestoreSharedAfterSwap: sharedDBPath vide (PlayerDB construit hors openPlayerDB ?)")
	}

	newShared, err := OpenReadOnly(pdb.sharedDBPath, pdb.userTimezone)
	if err != nil {
		slog.ErrorContext(ctx, "RestoreSharedAfterSwap: OpenReadOnly failed",
			"gamertag", pdb.Gamertag, "path", pdb.sharedDBPath, "err", err)
		return fmt.Errorf("RestoreSharedAfterSwap: OpenReadOnly: %w", err)
	}
	pdb.Shared = newShared

	if pdb.Player != nil {
		if err := attachShared(ctx, pdb.Player, pdb.sharedDBPath); err != nil {
			slog.ErrorContext(ctx, "RestoreSharedAfterSwap: attachShared on player failed",
				"gamertag", pdb.Gamertag, "err", err)
		}
	}
	if pdb.SharedSocial != nil {
		if err := attachShared(ctx, pdb.SharedSocial, pdb.sharedDBPath); err != nil {
			slog.WarnContext(ctx, "RestoreSharedAfterSwap: attachShared on social failed (continuing)",
				"gamertag", pdb.Gamertag, "err", err)
		}
	}
	return nil
}

// OnSharedSwap est le callback à câbler sur Provider.Subscribe en mode
// B-swap (commit 8g). Itère TOUS les PlayerDB du pool global et applique
// la mécanique de swap correspondante à la direction.
//
// IMPORTANT : doit s'exécuter SYNCHRONIQUEMENT (le Provider attend pour la
// direction PreSwap). Itération séquentielle sur le globalPool — pour
// quelques dizaines de PlayerDB, le coût est négligeable. Si on devait
// gérer des centaines de joueurs simultanés, on parallèliserait via une
// errgroup.
//
// Les erreurs par PlayerDB sont loguées mais ne stoppent pas l'itération
// (best-effort : un PlayerDB en panne ne doit pas bloquer le swap pour les
// autres).
func OnSharedSwap(ctx context.Context, direction SwapDirection) {
	globalPool.Range(func(_, value any) bool {
		pdb := value.(*PlayerDB)
		switch direction {
		case SwapDirPreSwapToRW:
			if err := pdb.PrepareForSharedSwap(ctx); err != nil {
				slog.ErrorContext(ctx, "OnSharedSwap: PrepareForSharedSwap failed",
					"gamertag", pdb.Gamertag, "err", err)
			}
		case SwapDirRWToRO, SwapDirErrorToRO:
			if err := pdb.RestoreSharedAfterSwap(ctx); err != nil {
				slog.ErrorContext(ctx, "OnSharedSwap: RestoreSharedAfterSwap failed",
					"gamertag", pdb.Gamertag, "err", err)
			}
		}
		return true
	})
}
