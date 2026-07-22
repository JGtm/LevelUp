package skill

// lusr_gap_scan.go — détecteur read-only de « trous d'intérieur » LUSR.
//
// Un trou d'intérieur = un match LUSR-ÉLIGIBLE, SANS ligne canonique
// rating_type='LUSR' dans match_skill_rank, ET SOUS le watermark chronologique
// de son groupe (player_skill_state_v2_latest.last_match_at). Autrement dit : le
// scoreur a fait avancer le watermark du groupe PAR-DESSUS ce match sans jamais
// écrire sa note (arrivée hors-ordre → skippedAlready permanent, cf. incident
// ac313879 / JGtm). C'est un trou RÉEL et PERMANENT, réparable seulement par
// replay chronologique (RecomputeLUSRCanonical).
//
// Distinct du « récent-en-attente » : un match éligible sans note mais AU-DESSUS
// du watermark n'a simplement pas encore été scoré — il le sera au prochain cycle,
// ce n'est pas une anomalie.
//
// Le détecteur RÉUTILISE loadShadowMatches (même filtre SQL) + le prédicat
// classifyLUSREligibility (mêmes checks rosters/équilibre/outcome) que le
// scoreur : source unique d'éligibilité, garde-rail
// lusr_eligibility_guardrail_test.go. Read-only, best-effort, aucun write.

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"levelup/go-api/internal/ctxkeys"
)

// LUSRGapMatch identifie un match sans note LUSR (trou d'intérieur ou récent).
type LUSRGapMatch struct {
	MatchID   string
	Group     string    // chaîne LUSR (playlist_group)
	PairName  string    // libellé playlist source (pair_name), pour l'affichage
	StartTime time.Time // canonique UTC (COALESCE start_time_utc/start_time)
}

// LUSRGroupGaps agrège l'état LUSR d'un groupe pour un joueur.
type LUSRGroupGaps struct {
	Group         string
	Eligible      int            // matchs LUSR-éligibles du groupe
	Rated         int            // parmi eux, ceux avec une ligne rating_type='LUSR'
	InteriorGaps  []LUSRGapMatch // éligibles SANS note ET sous le watermark (permanents)
	PendingRecent int            // éligibles SANS note ET au-dessus du watermark (attente)
}

// LUSRGapReport est le résultat d'un scan pour un joueur.
type LUSRGapReport struct {
	XUID   string
	Groups []LUSRGroupGaps
	// Agrégats tous groupes confondus (pratiques pour métriques/alerte).
	TotalEligible      int
	TotalRated         int
	TotalInteriorGaps  int
	TotalPendingRecent int
}

// ScanLUSRGaps calcule les trous LUSR d'un joueur. Read-only sur les deux DBs :
// sharedDB (candidats + rosters + watermark) et playerDB (lignes LUSR notées via
// la vue _latest). Best-effort côté caller (timeout-gardé, cf.
// runDualRowSentinelBestEffort) ; ici on retourne l'erreur brute.
//
// playerDB nil (pas de player DB) ou table match_skill_rank absente → ensemble
// des notés vide : tout éligible non-rated devient trou/pending selon le
// watermark (dégradation propre, pas d'erreur).
func ScanLUSRGaps(ctx context.Context, playerDB, sharedDB *sql.DB, xuid string) (*LUSRGapReport, error) {
	if sharedDB == nil {
		return nil, fmt.Errorf("ScanLUSRGaps: sharedDB nil")
	}
	candidates, err := loadShadowMatches(ctx, sharedDB, xuid)
	if err != nil {
		return nil, fmt.Errorf("ScanLUSRGaps.loadShadowMatches: %w", err)
	}
	rated, err := loadRatedLUSRMatchIDs(ctx, playerDB)
	if err != nil {
		return nil, fmt.Errorf("ScanLUSRGaps.loadRated: %w", err)
	}
	watermarks, err := loadGroupWatermarks(ctx, sharedDB, xuid)
	if err != nil {
		return nil, fmt.Errorf("ScanLUSRGaps.loadWatermarks: %w", err)
	}

	byGroup := make(map[string]*LUSRGroupGaps)
	title := ctxkeys.TitleSlug(ctx)
	for _, m := range candidates {
		group := GetLUSRChainForTitle(title, m.pairName)
		if group == "" {
			continue // pas de chaîne LUSR → pas un match éligible (skippedChain)
		}
		if !classifyLUSREligibility(ctx, sharedDB, m).eligible {
			continue // FFA / ≠ 2 équipes / déséquilibré / outcome non scorable
		}
		g := byGroup[group]
		if g == nil {
			g = &LUSRGroupGaps{Group: group}
			byGroup[group] = g
		}
		g.Eligible++
		if rated[m.matchID] {
			g.Rated++
			continue
		}
		// Non noté : trou d'intérieur (sous le watermark) vs récent-en-attente.
		// Sous le watermark = le scoreur considère ce match « déjà vu »
		// (skippedAlready : !start_time.After(last_match_at)) → note définitivement
		// absente = trou permanent. Pas de watermark (groupe jamais scoré) → rien
		// n'a encore été traité, donc « en attente », pas un trou.
		wm := watermarks[group]
		if wm != nil && !m.startTime.After(*wm) {
			g.InteriorGaps = append(g.InteriorGaps, LUSRGapMatch{
				MatchID: m.matchID, Group: group, PairName: m.pairName, StartTime: m.startTime,
			})
		} else {
			g.PendingRecent++
		}
	}
	return buildGapReport(xuid, byGroup), nil
}

// buildGapReport assemble le rapport final + les agrégats depuis la map par groupe.
func buildGapReport(xuid string, byGroup map[string]*LUSRGroupGaps) *LUSRGapReport {
	rep := &LUSRGapReport{XUID: xuid}
	for _, g := range byGroup {
		rep.Groups = append(rep.Groups, *g)
		rep.TotalEligible += g.Eligible
		rep.TotalRated += g.Rated
		rep.TotalInteriorGaps += len(g.InteriorGaps)
		rep.TotalPendingRecent += g.PendingRecent
	}
	return rep
}

// loadRatedLUSRMatchIDs retourne l'ensemble des match_id ayant une ligne
// rating_type='LUSR' dans le player DB. Lecture sur la vue _latest (règle ART
// n°2 : jamais la table brute). playerDB nil ou table absente → set vide, nil
// (dégradation propre : match jamais migré / joueur sans player DB).
func loadRatedLUSRMatchIDs(ctx context.Context, playerDB *sql.DB) (map[string]bool, error) {
	set := make(map[string]bool)
	if playerDB == nil {
		return set, nil
	}
	// Table absente (player DB pas migrée append-only) → pas une erreur.
	if _, err := playerDB.ExecContext(ctx, `SELECT 1 FROM match_skill_rank_latest LIMIT 0`); err != nil {
		return set, nil
	}
	rows, err := playerDB.QueryContext(ctx,
		`SELECT match_id FROM match_skill_rank_latest WHERE rating_type = 'LUSR'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		set[id] = true
	}
	return set, rows.Err()
}

// loadGroupWatermarks retourne last_match_at par groupe pour le joueur, depuis la
// vue _latest (règle ART n°2). Un groupe absent = jamais scoré (nil dans la map).
func loadGroupWatermarks(ctx context.Context, sharedDB *sql.DB, xuid string) (map[string]*time.Time, error) {
	rows, err := sharedDB.QueryContext(ctx,
		`SELECT playlist_group, last_match_at
		   FROM player_skill_state_v2_latest
		  WHERE xuid = ?`, xuid)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck
	out := make(map[string]*time.Time)
	for rows.Next() {
		var group string
		var lastMatchAt sql.NullTime
		if err := rows.Scan(&group, &lastMatchAt); err != nil {
			return nil, err
		}
		if lastMatchAt.Valid {
			t := lastMatchAt.Time
			out[group] = &t
		} else {
			out[group] = nil
		}
	}
	return out, rows.Err()
}
