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
package wire

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
	// maxPlausibleCounterDelta : borne de vraisemblance d'un delta compteur par
	// cycle de sync. Le max légitime observé en prod = 6 ; l'incident cold-start
	// 2026-07-03 portait objective_completed=3434. Au-delà de ce seuil le delta
	// est presque toujours un artefact de snapshot « before » faussement bas —
	// on log et on supprime l'émission (DP9).
	maxPlausibleCounterDelta = 20
)

// counterDelta décrit une notification post-sync « compteur » : émise quand
// after.field > before.field, delta = différence porté dans Params[count]. Les
// deltas NON uniformes (career_rank, skill_tier, threshold_crossed, personal_record)
// restent codés à la main dans EmitPostSyncDeltas. K1a (2026-07-05) : table-driven
// pour dégonfler la god-function EmitPostSyncDeltas.
type counterDelta struct {
	field       func(s *PlayerSnapshot) int
	category    notifications.Category
	severity    notifications.Severity
	titleKey    string
	bodyKey     string
	routeSuffix string         // segment après /players/{slug}/
	search      map[string]any // TargetSearch optionnel (nil = aucun)
	logLabel    string
}

// postSyncCounterDeltas — les 7 deltas compteur uniformes émis après une sync.
var postSyncCounterDeltas = []counterDelta{
	{field: func(s *PlayerSnapshot) int { return s.PersonalAwardCount }, category: notifications.CategoryObjectiveCompleted, severity: notifications.SeveritySuccess, titleKey: "notif.objective_completed.title", bodyKey: "notif.objective_completed.body", routeSuffix: "ascension", logLabel: "objective_completed"},
	{field: func(s *PlayerSnapshot) int { return s.ChallengeCompletedCount }, category: notifications.CategoryChallengeCompleted, severity: notifications.SeveritySuccess, titleKey: "notif.challenge_completed.title", bodyKey: "notif.challenge_completed.body", routeSuffix: "ascension", search: map[string]any{"tab": "challenges"}, logLabel: "challenge_completed"},
	{field: func(s *PlayerSnapshot) int { return s.BattlepassCompletedTracks }, category: notifications.CategoryBattlepassCompleted, severity: notifications.SeveritySuccess, titleKey: "notif.battlepass_completed.title", bodyKey: "notif.battlepass_completed.body", routeSuffix: "career/season-pass", logLabel: "battlepass_completed"},
	{field: func(s *PlayerSnapshot) int { return s.CitationTotalEarnedTiers }, category: notifications.CategoryCitationTier, severity: notifications.SeveritySuccess, titleKey: "notif.citation_tier.title", bodyKey: "notif.citation_tier.body", routeSuffix: "citations", logLabel: "citation_tier"},
	{field: func(s *PlayerSnapshot) int { return s.CitationMasteryCount }, category: notifications.CategoryCitationMastery, severity: notifications.SeveritySuccess, titleKey: "notif.citation_mastery.title", bodyKey: "notif.citation_mastery.body", routeSuffix: "citations", logLabel: "citation_mastery"},
	{field: func(s *PlayerSnapshot) int { return s.ChallengePathsCount }, category: notifications.CategoryChallengeAdded, severity: notifications.SeverityInfo, titleKey: "notif.challenge_added.title", bodyKey: "notif.challenge_added.body", routeSuffix: "ascension", search: map[string]any{"tab": "challenges"}, logLabel: "challenge_added"},
	// B1/DP2 (2026-07) : objective_assigned SUPPRIMÉ — doublon exact de
	// objective_completed (les deux étaient branchés sur PersonalAwardCount, une
	// paire de notifs à chaque cycle de sync). Catégorie conservée en rétro-compat
	// (types.go) mais plus jamais émise post-sync. Garde-rail :
	// TestPostSyncNeverEmitsObjectiveAssigned.
}

// BuildPostSyncDeltaHook construit la closure consommée par sync_handler :
// elle prend une snapshot avant le run, retourne une fonction qui prend
// la snapshot d'après et émet les notifications correspondantes.
func BuildPostSyncDeltaHook(reg *ServiceRegistry) handlers.PostSyncDeltaHook {
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
			EmitPostSyncDeltas(ctx, emitter, slug, before, after, pdb2, PostSyncDeltaOptions{
				RecentSkillTiers: loadRecentSkillTierNotifs(ctx, pdb2),
				Now:              time.Now().UTC(),
			})

			// Lot relations-E : notification « rival croisé » — nouveau duel contre
			// un top rival ramené par cette sync. Best-effort (détection interne
			// non bloquante) ; skippée sans nouveau match (watermark).
			emitRivalEncounters(ctx, emitter, newRivalDetectorForPDB(pdb2), slug, before, after)

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

// snapshotLooksCold retourne vrai si le snapshot ne porte AUCUN signal joueur :
// tous les compteurs à 0, aucune playlist classée, KD nul. Un « before » froid
// (échec de lecture de la DB, ou premier passage) fait apparaître tout
// l'historique comme delta positif au cycle suivant → burst de notifications
// (incident cold-start 2026-07-03 : 22 notifs en 1 s). DP1 : garde en mémoire,
// pas de baseline persistée.
func snapshotLooksCold(s *PlayerSnapshot) bool {
	if s == nil {
		return true
	}
	return s.PersonalAwardCount == 0 &&
		s.CitationsCount == 0 &&
		s.ChallengePathsCount == 0 &&
		s.ChallengeCompletedCount == 0 &&
		s.BattlepassCompletedTracks == 0 &&
		s.CitationTotalEarnedTiers == 0 &&
		s.CitationMasteryCount == 0 &&
		s.CurrentRank == 0 &&
		len(s.SkillTierByPlaylist) == 0 &&
		s.KDRatio == 0
}

// persistBestKDASeed persiste le record best_kda SANS émettre de notification —
// seed silencieux. Utilisé au cold-start (before froid : on supprime toutes les
// émissions mais on veut quand même semer le PB pour ne pas le notifier au cycle
// suivant) ; la garde `!oldRec.Loaded` évite d'écraser un record existant.
func persistBestKDASeed(ctx context.Context, pdb *duckdb.PlayerDB, after *PlayerSnapshot) {
	if pdb == nil || after == nil || after.BestKDA <= 0 {
		return
	}
	oldRec, err := duckdb.LoadPlayerRecord(ctx, pdb, "best_kda")
	if err != nil {
		slog.DebugContext(ctx, "post_sync: load best_kda record (cold-start seed)", "err", err)
	}
	if !oldRec.Loaded || after.BestKDA > oldRec.Value+bestKDARecordEpsilon {
		if err := duckdb.UpsertPlayerRecord(ctx, pdb, "best_kda", after.BestKDA, after.BestKDAMatchID); err != nil {
			slog.WarnContext(ctx, "post_sync: persist best_kda (cold-start seed)", "err", err)
		}
	}
}

// loadRecentSkillTierNotifs charge les dernières notifs skill_tier du joueur
// (limite 50) pour la dédup 24 h (B10). Best-effort : retourne nil si la DB
// sociale n'est pas attachée ou si la lecture échoue.
func loadRecentSkillTierNotifs(ctx context.Context, pdb *duckdb.PlayerDB) []notifications.Notification {
	if pdb == nil || pdb.SharedSocial == nil {
		return nil
	}
	repo := duckdb.NewNotificationsRepo(pdb)
	lr, err := repo.List(ctx, notifications.ListFilter{
		Category: notifications.CategorySkillTier,
		Limit:    50,
	})
	if err != nil {
		slog.WarnContext(ctx, "post_sync: load recent skill_tier notifs", "err", err)
		return nil
	}
	return lr.Items
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
// PostSyncDeltaOptions porte les entrées optionnelles de l'émission delta —
// le contexte de dédup skill_tier (B10/DP4). Paramètre OBLIGATOIRE (F9, revue
// 2026-07-17) : passer `PostSyncDeltaOptions{}` quand rien à contextualiser ; la
// prod passe les notifs skill_tier récentes. L'ancien variadique masquait
// silencieusement toute 2e option (seul opts[0] était lu).
type PostSyncDeltaOptions struct {
	// RecentSkillTiers : notifs skill_tier récentes (< 24 h utile) pour la dédup.
	RecentSkillTiers []notifications.Notification
	// Now : horloge injectée (défaut time.Now().UTC()).
	Now time.Time
}

//nolint:funlen,gocyclo // Émetteur multi-événements : compare ~12 snapshots delta
func EmitPostSyncDeltas(
	ctx context.Context,
	emitter notifications.Emitter,
	slug string,
	before, after *PlayerSnapshot,
	pdb *duckdb.PlayerDB,
	opts PostSyncDeltaOptions,
) {
	if emitter == nil || before == nil || after == nil {
		return
	}
	o := opts
	if o.Now.IsZero() {
		o.Now = time.Now().UTC()
	}

	// Anti-burst cold-start (DP1) : un snapshot « before » froid (échec de lecture
	// DB ou premier passage) transforme tout l'historique en delta positif → burst.
	// On supprime toutes les émissions de ce cycle mais on sème silencieusement le
	// PB best_kda pour ne pas le notifier au cycle suivant.
	if snapshotLooksCold(before) && !snapshotLooksCold(after) {
		slog.WarnContext(ctx, "post_sync: snapshot before froid — émissions supprimées (cold-start)", "slug", slug)
		persistBestKDASeed(ctx, pdb, after)
		return
	}

	// Deltas compteur uniformes (table-driven, K1a). Ordre non significatif :
	// chaque catégorie est distincte et les tests valident l'ENSEMBLE émis, pas
	// la séquence (hasCategory/countCategory). Les deltas bespoke suivent.
	for _, d := range postSyncCounterDeltas {
		newV, oldV := d.field(after), d.field(before)
		if newV <= oldV {
			continue
		}
		delta := newV - oldV
		// Cap de vraisemblance (DP9) : un delta > maxPlausibleCounterDelta trahit
		// presque toujours un snapshot « before » faussement bas — on supprime.
		if delta > maxPlausibleCounterDelta {
			slog.WarnContext(ctx, "post_sync: delta compteur invraisemblable supprimé",
				"category", d.category, "delta", delta, "log_label", d.logLabel)
			continue
		}
		in := notifications.EmitInput{
			Category:    d.category,
			Severity:    d.severity,
			TitleKey:    d.titleKey,
			BodyKey:     d.bodyKey,
			Params:      map[string]any{paramKeyCount: delta},
			TargetRoute: fmt.Sprintf("/players/%s/%s", slug, d.routeSuffix),
			Source:      postSyncSource,
		}
		if d.search != nil {
			in.TargetSearch = d.search
		}
		if err := emitter.Emit(ctx, in); err != nil {
			slog.WarnContext(ctx, "post_sync: "+d.logLabel, "err", err)
		}
	}

	emitCareerRankDelta(ctx, emitter, slug, before, after)
	emitSkillTierDeltas(ctx, emitter, slug, before, after, o)

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
