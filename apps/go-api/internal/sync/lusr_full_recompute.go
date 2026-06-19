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
	if _, err := sharedDB.ExecContext(ctx,
		`DELETE FROM player_skill_state_v2 WHERE xuid = ?`, xuid); err != nil {
		return 0, fmt.Errorf("RecomputeLUSRCanonicalForPlayer reset watermark: %w", err)
	}
	n, err := RunLUSRV2ShadowOwnerOnly(ctx, playerDB, sharedDB, xuid)
	if err != nil {
		return n, fmt.Errorf("RecomputeLUSRCanonicalForPlayer replay: %w", err)
	}
	return n, nil
}
