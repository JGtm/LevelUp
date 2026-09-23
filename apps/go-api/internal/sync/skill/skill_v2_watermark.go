package skill

// skill_v2_watermark.go — ce qui est NOUVEAU pour le shadow LUSR v2 se décide sur
// un LECTEUR, avant toute prise d'écrivain (lot perf L6, 2026-09-23).
//
// Mesure du 2026-09-23 (.ai/ETAT_DES_LIEUX_PERF_CHARGEMENTS_2026-09-23.md, C7) : le
// post-sync prenait l'écrivain partagé tous les 3 candidats et ne testait le « déjà
// traité » qu'à l'intérieur de la rafale — 1 233 bascules RO→RW→RO en moins de deux
// minutes (cinq joueurs, 9 416 candidats) pour ZÉRO ligne écrite. Chaque bascule
// draine les lecteurs HTTP, ferme le handle RO et refroidit le cache DuckDB.
//
// Désormais, sous le segment LECTURE : les candidats SQL, le filigrane de chaque
// groupe (last_match_at, vue _latest) et, pour les candidats situés au-dessus,
// l'éligibilité (prédicat partagé classifyLUSREligibility). Aucun candidat notable
// au-dessus du filigrane → aucun écrivain. Sinon UNE rafale par joueur et par cycle
// (runSingleWriterBurst), sous laquelle processOneShadowMatch garde TOUS ses
// contrôles (groupe tenu, filigrane relu sur le handle RW, éligibilité) : le
// pré-filtre n'est qu'une optimisation, la suite des écritures est celle d'avant
// (TestLUSRV2Shadow_ParityWithLegacyOrchestration).

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/ctxkeys"
)

// lusrWatermarkCovers est le prédicat UNIQUE du « déjà traité » : un match dont le
// début ne dépasse pas le filigrane de son groupe (last_match_at du dernier match
// persisté) a déjà été traité pour ce joueur ; wm nil = groupe jamais scoré. Trois
// consommateurs : le scoreur sous l'écrivain (processOneShadowMatch), le pré-filtre
// sous le lecteur (selectShadowWorkUnderRead) et le détecteur de trous
// (ScanLUSRGaps). Garde-rail : TestNoDuplicateLUSRWatermarkPredicate.
func lusrWatermarkCovers(wm *time.Time, start time.Time) bool {
	return wm != nil && !start.After(*wm)
}

// shadowWork est la sélection d'un cycle, faite sous le segment LECTURE.
type shadowWork struct {
	candidates int // candidats SQL du joueur (loadShadowMatches), tous groupes
	// pending : candidats d'une chaîne LUSR situés au-dessus du filigrane de leur
	// groupe, ordre chronologique ASC. Ce sont eux, et eux seuls, qui passent sous
	// l'écrivain.
	pending []shadowMatch
	// scorable : au moins un candidat de pending est notable → l'écrivain est requis.
	scorable        bool
	withGameplayDur int // observabilité TS2 : candidats à durée de gameplay connue
}

// selectShadowWorkUnderRead sélectionne le travail du cycle sous un segment de
// LECTURE shared (lecteur RO du provider en mode rafales), et RELÂCHE le Read avant
// de rendre la main — un Read encore en vol ferait échouer la rafale Write suivante
// (garde anti-deadlock). Les candidats écartés ici (sans chaîne, déjà traités, et
// non notables quand aucun ne l'est) sont comptés dans s.
func selectShadowWorkUnderRead(ctx context.Context, shared SharedAccessor, xuid string, s *shadowRunStats) (shadowWork, error) {
	readDB, release, err := shared.Read(ctx)
	if err != nil {
		return shadowWork{}, err
	}
	defer release()
	if readDB == nil {
		return shadowWork{}, fmt.Errorf("shared read handle nil")
	}
	matches, err := loadShadowMatches(ctx, readDB, xuid)
	if err != nil {
		return shadowWork{}, fmt.Errorf("loadShadowMatches: %w", err)
	}
	w := shadowWork{candidates: len(matches)}
	if len(matches) == 0 {
		return w, nil
	}
	watermarks, err := loadGroupWatermarks(ctx, readDB, xuid)
	if err != nil {
		return shadowWork{}, fmt.Errorf("loadGroupWatermarks: %w", err)
	}
	title := ctxkeys.TitleSlug(ctx)
	for _, m := range matches {
		if m.gameplayDurMs > 0 {
			w.withGameplayDur++
		}
		group := GetLUSRChainForTitle(title, m.pairName)
		switch {
		case group == "":
			s.skippedChain++
		case lusrWatermarkCovers(watermarks[group], m.startTime):
			s.skippedAlready++
		default:
			w.pending = append(w.pending, m)
		}
	}
	w.scorable = hasScorablePending(ctx, readDB, w.pending, s)
	return w, nil
}

// hasScorablePending dit si au moins un candidat en attente est notable (prédicat
// partagé classifyLUSREligibility). Il s'arrête au PREMIER notable : l'écrivain
// sera pris et processOneShadowMatch reclassera chaque candidat sous lui, dans
// l'ordre, comme avant. Sans notable, les non notables sont comptés ici : c'est le
// régime stationnaire d'un match à trois équipes, déséquilibré ou sans issue
// notable resté au-dessus du filigrane, qui ne s'écrira jamais (mesure du
// 2026-09-23 : un joueur sur cinq porte un tel match à chaque cycle ; sans ce
// classement, il prendrait une rafale d'écrivain par cycle pour rien).
func hasScorablePending(ctx context.Context, readDB *sql.DB, pending []shadowMatch, s *shadowRunStats) bool {
	nonTwoTeam, imbalance := 0, 0
	for _, m := range pending {
		elig := classifyLUSREligibility(ctx, readDB, m)
		switch {
		case elig.eligible:
			return true
		case elig.reason == lusrSkipImbalance:
			imbalance++
		default:
			nonTwoTeam++
		}
	}
	s.skippedNonTwoTeam += nonTwoTeam
	s.skippedImbalance += imbalance
	return false
}

// logShadowIdle émet la ligne INFO d'un cycle sans écrivain (une par joueur et par
// cycle) : rien au-dessus du filigrane qui s'écrira.
func logShadowIdle(ctx context.Context, xuid string, w shadowWork, s shadowRunStats) {
	slog.InfoContext(ctx, "lusr_v2: rien de nouveau",
		"xuid", xuid,
		"candidates", w.candidates,
		"new", 0,
		"skipped_chain", s.skippedChain,
		"skipped_already_seen", s.skippedAlready,
		"skipped_non_two_team", s.skippedNonTwoTeam,
		"skipped_imbalance", s.skippedImbalance,
	)
}

// logShadowBurstDone émet la ligne INFO d'un cycle avec rafale d'écrivain (une par
// joueur et par cycle) : new = candidats passés sous l'écrivain.
func logShadowBurstDone(ctx context.Context, xuid string, w shadowWork, s shadowRunStats) {
	slog.InfoContext(ctx, "lusr_v2: rafale terminée",
		"xuid", xuid,
		"candidates", w.candidates,
		"new", len(w.pending),
		"processed", s.processed,
		"skipped_chain", s.skippedChain,
		"skipped_already_seen", s.skippedAlready,
		"skipped_non_two_team", s.skippedNonTwoTeam,
		"skipped_imbalance", s.skippedImbalance,
		"skipped_write_failed", s.skippedWriteFailed,
		"skipped_group_held", s.skippedGroupHeld,
		// Observabilité TS2 : candidats où wᵢ = time_played / durée de gameplay est
		// alimenté. 0 → pondération inactive (repli wᵢ = 1).
		"with_gameplay_duration", w.withGameplayDur,
	)
}
