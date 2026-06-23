package livesync

// csr_persist.go — hook de persistance CSR Halo 5 (G4 Phase 1).
//
// Fetch le service record arena (1 appel, source live DÉJÀ construite par le
// runner) → mappe en lignes CSR canoniques → persiste dans la player DB h5
// (player_csr_snapshots, append-only, saison lifetime). La LECTURE (page Carrière)
// lit ce STOCKÉ via GetCSRSnapshots avec season_id == h5LifetimeSeasonID, résolu
// par CSRSeasonIDForTitle (TitleDescriptor.CSRSeasonID, cf. title.toml h5).
//
// Provisioning : sync.OpenPlayerDB applique EnsurePlayerSchema (crée
// player_csr_snapshots + vue _latest si absentes) — la player DB h5 n'existe pas
// en sync shared-only, on la crée à la volée ici, idempotent.

import (
	"context"
	"fmt"

	halo5 "levelup/go-api/internal/games/halo_5"
	syncpkg "levelup/go-api/internal/sync"
)

// h5ArenaRecordType — type de service record consommé (arena = seul mode classé h5
// pertinent pour le CSR par playlist). Miroir de halo_5.h5RecordModeArena (non exporté).
const h5ArenaRecordType = "arena"

// PersistArenaCSR — point d'entrée EXPORTÉ de persistArenaCSR pour les outils ops
// (cmd/h5-csr-backfill : backfill des snapshots CSR carrière par playlist pour un
// joueur, via une source authentifiée — éventuellement empruntée). Même logique que
// le hook live PersistCSR.
func PersistArenaCSR(ctx context.Context, src halo5.CaptureSource, playerDBPath, gamertag string) (int, error) {
	return persistArenaCSR(ctx, src, playerDBPath, gamertag)
}

// persistArenaCSR fetch le service record arena du joueur via src (source live déjà
// authentifiée), mappe les playlists en CSR canoniques et les persiste dans la
// player DB h5. Retourne le nombre de lignes écrites. Best-effort à l'appelant
// (runner) : une erreur est remontée mais n'avorte pas le cycle de sync.
//
//   - playerDBPath : data/titles/halo_5/players/{gamertag}/stats.duckdb.
//   - gamertag     : le joueur consulté (l'API h5 indexe par gamertag, pas xuid).
func persistArenaCSR(ctx context.Context, src halo5.CaptureSource, playerDBPath, gamertag string) (int, error) {
	resp, err := src.GetServiceRecords(ctx, gamertag, h5ArenaRecordType)
	if err != nil {
		return 0, fmt.Errorf("h5 CSR: service record arena: %w", err)
	}
	csrs := mapH5ArenaToPlaylistCSRs(resp)
	if len(csrs) == 0 {
		return 0, nil // pas de playlist arena classée → rien à persister.
	}

	// OpenPlayerDB crée le dossier + applique EnsurePlayerSchema (player_csr_snapshots
	// + vue _latest). Idempotent ; la player DB h5 n'existe pas en sync shared-only.
	db, err := syncpkg.OpenPlayerDB(playerDBPath)
	if err != nil {
		return 0, fmt.Errorf("h5 CSR: open player DB: %w", err)
	}
	defer db.Close()

	n, err := syncpkg.SaveCSRSnapshots(ctx, db.SQLDb(), csrs, h5LifetimeSeasonID)
	if err != nil {
		return n, fmt.Errorf("h5 CSR: persist snapshots: %w", err)
	}
	return n, nil
}
