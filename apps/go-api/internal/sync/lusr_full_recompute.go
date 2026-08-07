// Package sync — lusr_full_recompute.go : replay LUSR v2 complet d'un joueur.
//
// v2 est le modèle de prod (canonical par défaut au boot, ADR 0024) ; v1 est mort.
// Cette fonction ne fait donc QUE du v2 : elle réutilise le chemin de prod
// RunLUSRV2ShadowOwnerOnly, précédé d'un reset du watermark du joueur — exactement
// le pattern du backfill canonical (cmd/lusr_v2_canonical_backfill), ici nommé et
// réutilisable.
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// RecomputeLUSRCanonicalForPlayer rejoue le LUSR (v2) d'UN joueur sur TOUT son
// historique : reset de son watermark (DELETE player_skill_state_v2 WHERE xuid)
// puis RunLUSRV2ShadowOwnerOnly. Owner-only : ne touche que l'état de ce joueur.
//
// À utiliser pour les rattrapages où des matchs ANCIENS sont insérés a posteriori
// (import OpenSpartan, backfill) et changent la chaîne LUSR : le chemin live
// incrémental les sauterait (watermark déjà avancé). RunLUSRV2ShadowOwnerOnly écrit
// la ligne canonique rating_type='LUSR' lue par l'UI (Stratégie C). No-op si v2
// désactivé. Pur local, aucun appel API. Retourne le nombre de matchs traités.
func RecomputeLUSRCanonicalForPlayer(ctx context.Context, playerDB, sharedDB *sql.DB, xuid string) (int, error) {
	if sharedDB == nil || playerDB == nil {
		return 0, fmt.Errorf("RecomputeLUSRCanonicalForPlayer: nil DB")
	}
	if strings.TrimSpace(xuid) == "" {
		return 0, fmt.Errorf("RecomputeLUSRCanonicalForPlayer: xuid vide")
	}
	// Append-only #23046 (Phase 2) : plus de DELETE WHERE xuid (vecteur ART sur
	// PK + idx_pssv2). On INSÈRE une row sentinelle is_reset=TRUE par (xuid,
	// playlist_group) existant ; la vue player_skill_state_v2_latest (WHERE NOT
	// is_reset) masque alors le groupe → LoadState renvoie nil → le replay
	// re-seed depuis les priors. L'horloge UTC canonique garantit written_at
	// postérieur aux états précédents — et SEULEMENT elle : `now()` nu rendait un
	// TIMESTAMPTZ coercé par le fuseau de SESSION, donc une sentinelle datée deux
	// heures dans le futur à UTC+2. Elle gagnait alors l'arbitrage de
	// player_skill_state_v2_latest contre les états UTC replayés dans la foulée,
	// et masquait deux heures de recalcul (jumeau exact du défaut R1, lot S5).
	// Si le joueur n'a aucun état, le SELECT est vide (no-op correct).
	if _, err := sharedDB.ExecContext(ctx, `
		INSERT INTO player_skill_state_v2
			(xuid, playlist_group, mu, sigma, experience, written_at, is_reset)
		SELECT xuid, playlist_group, 0, 0, 0, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP), TRUE
		FROM player_skill_state_v2_latest WHERE xuid = ?`, xuid); err != nil {
		return 0, fmt.Errorf("RecomputeLUSRCanonicalForPlayer reset watermark: %w", err)
	}
	// sharedDB est un handle DÉJÀ TENU par le caller (writer primaire) → mode pinned
	// (Read/Write retournent ce handle). Le shadow persiste via son burst Write.
	n, err := RunLUSRV2ShadowOwnerOnly(ctx, playerDB, NewPinnedSharedAccess(sharedDB), xuid)
	if err != nil {
		return n, fmt.Errorf("RecomputeLUSRCanonicalForPlayer replay: %w", err)
	}
	return n, nil
}
