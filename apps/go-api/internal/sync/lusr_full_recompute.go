// Package sync — lusr_full_recompute.go : replay LUSR v2 complet d'un joueur.
//
// v2 est le seul modèle LUSR (ADR 0024) : la v1 est supprimée. Ces fonctions
// réutilisent le chemin de prod RunLUSRV2ShadowOwnerOnly, précédé d'un reset du
// watermark du joueur — exactement le pattern du backfill canonical
// (cmd/lusr_v2_canonical_backfill), ici nommé et réutilisable.
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"levelup/go-api/internal/sync/skill"
)

// RecomputeLUSRCanonicalForPlayer rejoue le LUSR (v2) d'UN joueur sur TOUT son
// historique, sur un handle shared DÉJÀ TENU en écriture par le caller (mode épinglé :
// CLI, backfill, import). Voir RecomputeLUSRCanonicalWithAccess pour le contrat.
func RecomputeLUSRCanonicalForPlayer(ctx context.Context, playerDB, sharedDB *sql.DB, xuid string,
	opts ...skill.ShadowOption) (int, error) {
	if sharedDB == nil || playerDB == nil {
		return 0, fmt.Errorf("RecomputeLUSRCanonicalForPlayer: nil DB")
	}
	return RecomputeLUSRCanonicalWithAccess(ctx, playerDB, NewPinnedSharedAccess(sharedDB), xuid, opts...)
}

// RecomputeLUSRCanonicalWithAccess rejoue le LUSR (v2) d'UN joueur sur TOUT son
// historique : reset de son watermark (sentinelle is_reset dans player_skill_state_v2,
// côté shared) puis RunLUSRV2ShadowOwnerOnly. Owner-only : ne touche que l'état de ce
// joueur. Les matchs is_excluded du joueur n'y entrent pas.
//
// Accès shared : le reset prend UNE rafale d'écriture courte (un INSERT), relâchée avant
// le replay ; le replay prend lui-même ses rafales bornées (au plus 50 matchs ou 2 s
// chacune) et sa sélection sous lecteur. L'écrivain n'est donc jamais tenu en continu —
// c'est le contrat des appelants de requête (MatchRecomputer) : un écrivain tenu fait
// répondre 503 aux lecteurs du site. En mode épinglé, tout se passe sur le handle tenu.
//
// À utiliser pour les rattrapages où des matchs ANCIENS sont insérés a posteriori
// (import OpenSpartan, backfill) ou retirés de la note (exclusion) et changent la chaîne
// LUSR : le chemin live incrémental les sauterait (watermark déjà avancé).
// RunLUSRV2ShadowOwnerOnly écrit la ligne canonique rating_type='LUSR' lue par l'UI
// (Stratégie C). Pur local, aucun appel API. Retourne le nombre de matchs traités.
func RecomputeLUSRCanonicalWithAccess(ctx context.Context, playerDB *sql.DB, shared *SharedAccess, xuid string,
	opts ...skill.ShadowOption) (int, error) {
	if shared == nil || playerDB == nil {
		return 0, fmt.Errorf("RecomputeLUSRCanonicalWithAccess: nil accès ou DB joueur")
	}
	if strings.TrimSpace(xuid) == "" {
		return 0, fmt.Errorf("RecomputeLUSRCanonicalWithAccess: xuid vide")
	}
	if err := resetLUSRWatermark(ctx, shared, xuid); err != nil {
		return 0, err
	}
	n, err := RunLUSRV2ShadowOwnerOnly(ctx, playerDB, shared, xuid, opts...)
	if err != nil {
		return n, fmt.Errorf("RecomputeLUSRCanonicalWithAccess replay: %w", err)
	}
	return n, nil
}

// resetLUSRWatermark pose, sous une rafale d'écriture courte, la sentinelle de reset du
// joueur. Append-only #23645 (Phase 2) : plus de DELETE WHERE xuid (vecteur ART sur
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
func resetLUSRWatermark(ctx context.Context, shared *SharedAccess, xuid string) error {
	sharedDB, release, err := shared.Write(ctx, "lusr_reset")
	if err != nil {
		return fmt.Errorf("RecomputeLUSRCanonicalWithAccess reset watermark: rafale shared: %w", err)
	}
	defer release()
	if _, err := sharedDB.ExecContext(ctx, `
		INSERT INTO player_skill_state_v2
			(xuid, playlist_group, mu, sigma, experience, written_at, is_reset)
		SELECT xuid, playlist_group, 0, 0, 0, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP), TRUE
		FROM player_skill_state_v2_latest WHERE xuid = ?`, xuid); err != nil {
		return fmt.Errorf("RecomputeLUSRCanonicalWithAccess reset watermark: %w", err)
	}
	return nil
}
