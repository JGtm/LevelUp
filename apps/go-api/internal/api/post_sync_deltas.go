// Package api — post_sync_deltas.go : émission des notifications delta après une sync.
//
// Stratégie : avant la sync on capture une PlayerSnapshot (rank season pass +
// count des awards complétés), après la sync on capture une nouvelle, et on
// émet les notifications correspondant aux deltas observés.
//
// MVP couvre `season_pass_level` et `objective_completed`. Les hooks plus
// complexes (challenge_completed via citations diff, personal_record via
// référentiel records, threshold_crossed sur KD/winrate, objective_assigned)
// sont laissés en TODO documenté ci-dessous — l'infrastructure est en place
// pour les ajouter incrémentalement.
package api

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/api/handlers"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
)

// rankSubRoman convertit tout sous-rang arabe isolé (1–6) en chiffre romain.
// Gère les positions finale ("Or 3") et médiane ("Général 2 Platine").
func rankSubRoman(label string) string {
	roman := [7]string{"", "I", "II", "III", "IV", "V", "VI"}
	out := label
	for d := byte('1'); d <= '6'; d++ {
		r := roman[d-'0']
		out = strings.ReplaceAll(out, " "+string(d)+" ", " "+r+" ")
		if strings.HasSuffix(out, " "+string(d)) {
			out = out[:len(out)-1] + r
		}
	}
	return out
}

// Constantes partagées par les EmitInput émis depuis ce module.
const (
	// postSyncSource est la valeur du champ Source pour les notifications
	// émises après une sync (utilisée par notifications.Emitter pour la
	// dédup et l'analytics).
	postSyncSource = "post_sync"
	// paramKeyCount est la clé "count" du Params{} pour les notifications
	// delta agrégées (objectives, citations, friend_sync, etc.).
	paramKeyCount = "count"
	// kdRatioThresholdStep / winrateThresholdStep : pas de palier pour l'émission
	// des notifications threshold_crossed (0.05 = 5 points de KD ratio / 5 % de
	// taux de victoire). Nommés K1a (ex-magic 0.05).
	kdRatioThresholdStep = 0.05
	winrateThresholdStep = 0.05
	// bestKDARecordEpsilon : amélioration minimale du best_kda pour compter comme
	// un nouveau record personnel (filtre le bruit de flottant). Nommé K1a (ex-0.01).
	bestKDARecordEpsilon = 0.01
)

// buildPostSyncDeltaHook construit la closure consommée par sync_handler :
// elle prend une snapshot avant le run, retourne une fonction qui prend
// la snapshot d'après et émet les notifications correspondantes.
func buildPostSyncDeltaHook(reg *ServiceRegistry) handlers.PostSyncDeltaHook {
	return func(ctx context.Context, slug string) func(ctx context.Context) {
		pdb, err := reg.resolve(ctx, slug)
		if err != nil {
			slog.WarnContext(ctx, "post_sync: resolve before", "slug", slug, "err", err)
			return nil
		}
		xuid := pdb.XUID // capturé pour l'invalidation — indépendant du resolve post-sync
		before, err := SnapshotPlayerState(ctx, pdb, newCitationsServiceForPDB(pdb))
		if err != nil {
			slog.WarnContext(ctx, "post_sync: snapshot before", "slug", slug, "err", err)
		}
		return func(ctx context.Context) {
			// Invalider le cache home en premier — le sync a réussi, les données DB sont à jour.
			reg.homeMatchesCache.Invalidate(xuid)
			slog.InfoContext(ctx, "post_sync: home cache invalidé", "slug", slug, "xuid", xuid)

			pdb2, err := reg.resolve(ctx, slug)
			if err != nil {
				slog.WarnContext(ctx, "post_sync: resolve after", "slug", slug, "err", err)
				return
			}
			after, err := SnapshotPlayerState(ctx, pdb2, newCitationsServiceForPDB(pdb2))
			if err != nil {
				slog.WarnContext(ctx, "post_sync: snapshot after", "slug", slug, "err", err)
				return
			}
			emitter, err := reg.NotificationsEmitter(ctx, slug)
			if err != nil {
				slog.WarnContext(ctx, "post_sync: emitter", "slug", slug, "err", err)
				return
			}
			EmitPostSyncDeltas(ctx, emitter, slug, before, after, pdb2)

			// Couche progression V2 (Ascension) — pipeline streaks/records/
			// milestones/coach + coach_advisor (Phase 8 ADR 0020). Non
			// bloquant : toute erreur reste en slog.Warn.
			// PMT-4 : coach proactif + pipeline progression résolus pour le TITRE
			// du joueur syncé (pdb2.TitleSlug), JAMAIS le slug joueur de la closure
			// (`slug` = PlayerSlug ici, cf. sync_handler.postSync). Limite connue :
			// en post-sync background (auto-sync/CLI) le ctx ne porte pas de titre
			// → pdb2.TitleSlug = DefaultSlug (correct mono-titre ; à threader depuis
			// l'engine quand un 2e titre arrivera).
			progDeps := BuildPlayerProgressionDepsWithAdvisor(
				pdb2, emitter,
				reg.CoachAdvisorBundle(),
				reg.PrestigeBundle(),
				readCoachProactiveMode(reg, pdb2.TitleSlug),
				slug,
			)
			if _, err := EvaluateProgressionAfterSync(ctx, pdb2, pdb2.TitleSlug, progDeps, time.Now().UTC()); err != nil {
				slog.WarnContext(ctx, "post_sync: progression evaluate", "slug", slug, "err", err)
			}
		}
	}
}

// readCoachProactiveMode lit le toggle coach proactif RÉSOLU pour le titre
// (overlay per-titre PMT-4 : le coach proactif est lié au système de
// progression/Prestige propre au titre). Overlay absent ⇒ valeur globale
// byte-identique. Retourne false si le store n'est pas attaché ou si la lecture
// échoue — comportement opt-in strict (cf. ADR 0020 Phase 1).
func readCoachProactiveMode(reg *ServiceRegistry, titleSlug string) bool {
	store := reg.SettingsStore()
	if store == nil {
		return false
	}
	pr := titlePkg.NewPathResolver(reg.cfg.RepoRoot)
	cfg, err := store.ResolveForTitle(pr.TitleSettingsPath(titleSlug))
	if err != nil || cfg == nil {
		return false
	}
	return cfg.CoachProactiveMode
}

// PlayerSnapshot capture l'état pertinent d'un joueur pour la détection delta.
func thresholdCrossed(before, after, step float64) (crossed bool, level float64) {
	if step <= 0 || after <= before {
		return false, 0
	}
	beforeBucket := int(before / step)
	afterBucket := int(after / step)
	if afterBucket > beforeBucket {
		return true, float64(afterBucket) * step
	}
	return false, 0
}

// EmitPostSyncDeltas compare 2 snapshots et émet les notifications applicables.
//
// Best-effort : toute erreur est loguée et n'interrompt pas le flux de sync.
// `pdb` est passé pour persister les nouveaux records ; peut être nil
// (dans ce cas personal_record est skippé).
//
// et émet 1 notification par delta significatif. Complexité reflète le nombre
// de KPIs surveillés, pas un défaut de conception.
//
//nolint:funlen,gocyclo // Émetteur multi-événements : compare ~12 snapshots delta
func EmitPostSyncDeltas(
	ctx context.Context,
	emitter notifications.Emitter,
	slug string,
	before, after *PlayerSnapshot,
	pdb *duckdb.PlayerDB,
) {
	if emitter == nil || before == nil || after == nil {
		return
	}

	// career_rank : nouveau rang Halo lifetime franchi (career_progression).
	// Remplace l'ancien câblage CategorySeasonPassLevel qui pointait à tort sur
	// career_progression — déprécié depuis 2026-05-16.
	if after.CurrentRank > before.CurrentRank && after.CurrentRank > 0 {
		if err := emitter.Emit(ctx, notifications.EmitInput{
			Category: notifications.CategoryCareerRank,
			Severity: notifications.SeveritySuccess,
			TitleKey: "notif.career_rank.title",
			BodyKey:  "notif.career_rank.body",
			Params: map[string]any{
				"rank":      after.CurrentRank,
				"rank_name": rankSubRoman(after.CurrentRankName),
				"previous":  before.CurrentRank,
			},
			TargetRoute: fmt.Sprintf("/players/%s/career", slug),
			Source:      postSyncSource,
		}); err != nil {
			slog.WarnContext(ctx, "post_sync: career_rank", "err", err)
		}
	}

	// objective_completed (agrégé) : nouveaux awards. Le frontend résout
	// le libellé via templates i18n ; on ne passe que les paramètres machine.
	if after.PersonalAwardCount > before.PersonalAwardCount {
		delta := after.PersonalAwardCount - before.PersonalAwardCount
		if err := emitter.Emit(ctx, notifications.EmitInput{
			Category:    notifications.CategoryObjectiveCompleted,
			Severity:    notifications.SeveritySuccess,
			TitleKey:    "notif.objective_completed.title",
			BodyKey:     "notif.objective_completed.body",
			Params:      map[string]any{paramKeyCount: delta},
			TargetRoute: fmt.Sprintf("/players/%s/ascension", slug),
			Source:      postSyncSource,
		}); err != nil {
			slog.WarnContext(ctx, "post_sync: objective_completed", "err", err)
		}
	}

	// challenge_completed (vrais défis daily/weekly) : nb challenge_path dont le
	// dernier status est 'Completed'. Recâblé depuis match_citations vers
	// challenge_snapshots le 2026-05-16 — la sémantique citations est désormais
	// traitée par citation_tier / citation_mastery.
	if after.ChallengeCompletedCount > before.ChallengeCompletedCount {
		delta := after.ChallengeCompletedCount - before.ChallengeCompletedCount
		if err := emitter.Emit(ctx, notifications.EmitInput{
			Category:     notifications.CategoryChallengeCompleted,
			Severity:     notifications.SeveritySuccess,
			TitleKey:     "notif.challenge_completed.title",
			BodyKey:      "notif.challenge_completed.body",
			Params:       map[string]any{paramKeyCount: delta},
			TargetRoute:  fmt.Sprintf("/players/%s/ascension", slug),
			TargetSearch: map[string]any{"tab": "challenges"},
			Source:       postSyncSource,
		}); err != nil {
			slog.WarnContext(ctx, "post_sync: challenge_completed", "err", err)
		}
	}

	// skill_tier (CSR / LUSR unifié) : une notif par playlist_group dont le
	// tier|sub_tier|rating_type a changé entre les 2 snapshots. Les apparitions
	// inédites (playlist absente avant, présente après) sont aussi émises.
	for _, playlist := range sortedPlaylistKeys(after.SkillTierByPlaylist) {
		newVal := after.SkillTierByPlaylist[playlist]
		oldVal := before.SkillTierByPlaylist[playlist]
		if newVal == oldVal {
			continue
		}
		ratingType, tier, subTier := splitSkillTier(newVal)
		oldRT, oldTier, oldSub := splitSkillTier(oldVal)
		if err := emitter.Emit(ctx, notifications.EmitInput{
			Category: notifications.CategorySkillTier,
			Severity: notifications.SeveritySuccess,
			TitleKey: "notif.skill_tier.title",
			BodyKey:  "notif.skill_tier.body",
			Params: map[string]any{
				"playlist_group":    playlist,
				"rating_type":       ratingType,
				"tier":              tier,
				"sub_tier":          subTier,
				"previous_type":     oldRT,
				"previous_tier":     oldTier,
				"previous_sub_tier": oldSub,
			},
			TargetRoute: fmt.Sprintf("/players/%s/synthesis", slug),
			Source:      postSyncSource,
		}); err != nil {
			slog.WarnContext(ctx, "post_sync: skill_tier", "playlist", playlist, "err", err)
		}
	}

	// battlepass_completed : un track de plus a atteint son rang max.
	if after.BattlepassCompletedTracks > before.BattlepassCompletedTracks {
		delta := after.BattlepassCompletedTracks - before.BattlepassCompletedTracks
		if err := emitter.Emit(ctx, notifications.EmitInput{
			Category:    notifications.CategoryBattlepassCompleted,
			Severity:    notifications.SeveritySuccess,
			TitleKey:    "notif.battlepass_completed.title",
			BodyKey:     "notif.battlepass_completed.body",
			Params:      map[string]any{paramKeyCount: delta},
			TargetRoute: fmt.Sprintf("/players/%s/career/season-pass", slug),
			Source:      postSyncSource,
		}); err != nil {
			slog.WarnContext(ctx, "post_sync: battlepass_completed", "err", err)
		}
	}

	// citation_tier : au moins un nouveau palier franchi sur une commendation.
	// Agrégé : count = somme des paliers franchis depuis la sync précédente.
	if after.CitationTotalEarnedTiers > before.CitationTotalEarnedTiers {
		delta := after.CitationTotalEarnedTiers - before.CitationTotalEarnedTiers
		if err := emitter.Emit(ctx, notifications.EmitInput{
			Category:    notifications.CategoryCitationTier,
			Severity:    notifications.SeveritySuccess,
			TitleKey:    "notif.citation_tier.title",
			BodyKey:     "notif.citation_tier.body",
			Params:      map[string]any{paramKeyCount: delta},
			TargetRoute: fmt.Sprintf("/players/%s/citations", slug),
			Source:      postSyncSource,
		}); err != nil {
			slog.WarnContext(ctx, "post_sync: citation_tier", "err", err)
		}
	}

	// citation_mastery : une (ou plusieurs) commendation(s) viennent d'atteindre
	// 100 % de masterisation.
	if after.CitationMasteryCount > before.CitationMasteryCount {
		delta := after.CitationMasteryCount - before.CitationMasteryCount
		if err := emitter.Emit(ctx, notifications.EmitInput{
			Category:    notifications.CategoryCitationMastery,
			Severity:    notifications.SeveritySuccess,
			TitleKey:    "notif.citation_mastery.title",
			BodyKey:     "notif.citation_mastery.body",
			Params:      map[string]any{paramKeyCount: delta},
			TargetRoute: fmt.Sprintf("/players/%s/citations", slug),
			Source:      postSyncSource,
		}); err != nil {
			slog.WarnContext(ctx, "post_sync: citation_mastery", "err", err)
		}
	}

	// challenge_added : nouveaux challenge_path apparus dans challenge_snapshots.
	if after.ChallengePathsCount > before.ChallengePathsCount {
		delta := after.ChallengePathsCount - before.ChallengePathsCount
		if err := emitter.Emit(ctx, notifications.EmitInput{
			Category:     notifications.CategoryChallengeAdded,
			Severity:     notifications.SeverityInfo,
			TitleKey:     "notif.challenge_added.title",
			BodyKey:      "notif.challenge_added.body",
			Params:       map[string]any{paramKeyCount: delta},
			TargetRoute:  fmt.Sprintf("/players/%s/ascension", slug),
			TargetSearch: map[string]any{"tab": "challenges"},
			Source:       postSyncSource,
		}); err != nil {
			slog.WarnContext(ctx, "post_sync: challenge_added", "err", err)
		}
	}

	// objective_assigned : la table personal_score_awards ne distingue pas
	// "assigned" vs "completed". Le delta seul détecte une nouvelle entrée
	// (qui peut être déjà complétée). On émet quand même : l'utilisateur peut
	// distinguer dans l'UI via les params (target = score atteint vs assigné).
	// Doublon possible avec objective_completed sur le même match — accepté pour
	// le MVP, à raffiner avec un signal explicite côté Prestige.
	if after.PersonalAwardCount > before.PersonalAwardCount {
		delta := after.PersonalAwardCount - before.PersonalAwardCount
		if err := emitter.Emit(ctx, notifications.EmitInput{
			Category:    notifications.CategoryObjectiveAssigned,
			Severity:    notifications.SeverityInfo,
			TitleKey:    "notif.objective_assigned.title",
			BodyKey:     "notif.objective_assigned.body",
			Params:      map[string]any{paramKeyCount: delta},
			TargetRoute: fmt.Sprintf("/players/%s/ascension", slug),
			Source:      postSyncSource,
		}); err != nil {
			slog.WarnContext(ctx, "post_sync: objective_assigned", "err", err)
		}
	}

	// threshold_crossed — KD ratio (palier 0.05). On envoie metric_key (clé i18n)
	// + value formaté ; le frontend résout metric_label via i18n.metricLabel.
	if crossed, level := thresholdCrossed(before.KDRatio, after.KDRatio, kdRatioThresholdStep); crossed {
		_ = emitter.Emit(ctx, notifications.EmitInput{
			Category:    notifications.CategoryThresholdCrossed,
			Severity:    notifications.SeveritySuccess,
			TitleKey:    "notif.threshold_crossed.title",
			BodyKey:     "notif.threshold_crossed.body",
			Params:      map[string]any{"metric_key": "kd_ratio", "value": fmt.Sprintf("%.2f", level)},
			TargetRoute: fmt.Sprintf("/players/%s/synthesis", slug),
			Source:      postSyncSource,
		})
	}

	// threshold_crossed — Winrate (palier 0.05 = 5%)
	if crossed, level := thresholdCrossed(before.Winrate, after.Winrate, winrateThresholdStep); crossed {
		_ = emitter.Emit(ctx, notifications.EmitInput{
			Category:    notifications.CategoryThresholdCrossed,
			Severity:    notifications.SeveritySuccess,
			TitleKey:    "notif.threshold_crossed.title",
			BodyKey:     "notif.threshold_crossed.body",
			Params:      map[string]any{"metric_key": "winrate", "value": fmt.Sprintf("%.0f%%", level*100)},
			TargetRoute: fmt.Sprintf("/players/%s/synthesis", slug),
			Source:      postSyncSource,
		})
	}

	// personal_record : best_kda matériel battu
	if pdb != nil && after.BestKDA > 0 {
		oldRec, err := duckdb.LoadPlayerRecord(ctx, pdb, "best_kda")
		if err != nil {
			slog.DebugContext(ctx, "post_sync: load best_kda record", "err", err)
		}
		if oldRec.Loaded && after.BestKDA > oldRec.Value+bestKDARecordEpsilon {
			// Record battu → emit + persist. metric_key résolu côté frontend.
			_ = emitter.Emit(ctx, notifications.EmitInput{
				Category: notifications.CategoryPersonalRecord,
				Severity: notifications.SeveritySuccess,
				TitleKey: "notif.personal_record.title",
				BodyKey:  "notif.personal_record.body",
				Params: map[string]any{
					"metric_key": "kda",
					"value":      fmt.Sprintf("%.2f", after.BestKDA),
					"previous":   fmt.Sprintf("%.2f", oldRec.Value),
				},
				TargetRoute: fmt.Sprintf("/players/%s/synthesis", slug),
				Source:      postSyncSource,
			})
		}
		// Toujours persister la nouvelle valeur (init au premier passage,
		// update si battue)
		if !oldRec.Loaded || after.BestKDA > oldRec.Value+bestKDARecordEpsilon {
			if err := duckdb.UpsertPlayerRecord(ctx, pdb, "best_kda", after.BestKDA, after.BestKDAMatchID); err != nil {
				slog.WarnContext(ctx, "post_sync: persist best_kda", "err", err)
			}
		}
	}
}

// newCitationsServiceForPDB construit un CitationsService scopé sur le joueur.
// Retourne nil si pdb est invalide — SnapshotPlayerState saute alors la lecture
// citations.
func newCitationsServiceForPDB(pdb *duckdb.PlayerDB) port.CitationsService {
	if pdb == nil || pdb.Player == nil {
		return nil
	}
	return service.NewCitationsService(duckdb.NewCitationsRepo(pdb))
}

// sortedPlaylistKeys retourne les clés de m triées — garantit un ordre d'émission
// stable (utile pour les tests + journaux deterministes).
