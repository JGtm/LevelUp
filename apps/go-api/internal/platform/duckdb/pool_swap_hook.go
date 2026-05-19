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
	// libérer son handle RO sur shared (Close pdb.Shared).
	SwapDirPreSwapToRW SwapDirection = iota

	// SwapDirRWToRO correspond à sharedprovider.DirectionRWToRO : le swap RW
	// est terminé, on est revenu en RO. Le pool doit rouvrir pdb.Shared.
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
// simplifié. attachShared retiré entièrement du pool
// (player + social). Plus aucune conn du pool ne porte d'ATTACH shared, donc
// pas de DETACH explicite à faire. Seule la conn pdb.Shared (LegacySharedReader
// fallback) doit être fermée pour libérer le file handle.
//
// IMPORTANT : si pdb n'est pas inscrit au mécanisme B-swap (bSwapEnabled=false),
// cette méthode est no-op — on ne touche à rien (mode legacy).
//
// Doit être appelée SYNCHRONIQUEMENT par OnSharedSwap depuis le callback
// Subscribe du Provider, AVANT que le Provider ne tente OpenReadWrite.
func (pdb *PlayerDB) PrepareForSharedSwap(ctx context.Context) error {
	if !pdb.bSwapEnabled {
		// Mode legacy : pas inscrit au mécanisme B-swap — no-op.
		return nil
	}

	if pdb.Shared != nil {
		if err := pdb.Shared.Close(); err != nil {
			slog.WarnContext(ctx, "PrepareForSharedSwap: Close Shared failed (continuing)",
				"gamertag", pdb.Gamertag, "err", err)
		}
	}
	return nil
}

// RestoreSharedAfterSwap rouvre la conn RO sur shared. Appelée sur
// SwapDirRWToRO ou SwapDirErrorToRO depuis OnSharedSwap.
//
// simplifié. Plus de re-ATTACH (attachShared retiré
// entièrement). Le pool re-ouvre uniquement la conn RO pdb.Shared.
//
// Si OpenReadOnly échoue : log Error mais ne propage pas (les Query
// suivantes échoueront proprement, le retry loop async du Provider pourra
// éventuellement déclencher une seconde tentative).
func (pdb *PlayerDB) RestoreSharedAfterSwap(ctx context.Context) error {
	if !pdb.bSwapEnabled {
		// Mode legacy : pas inscrit au mécanisme B-swap — no-op.
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
