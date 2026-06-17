// embeds.go — Construction des Rich Embeds Discord pour sync et backfill.
package notify

import (
	"fmt"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// Constantes couleurs Discord
// ─────────────────────────────────────────────────────────────────────────────

const (
	colorSuccess = 0x57F287 // vert
	colorWarning = 0xFEE75C // jaune
	colorError   = 0xED4245 // rouge
	colorBlurple = 0x5865F2 // blurple (médias)
	colorVersion = 0x3498DB // bleu foncé (nouvelle version)
)

// ─────────────────────────────────────────────────────────────────────────────
// Structures des résultats joueur
// ─────────────────────────────────────────────────────────────────────────────

// LastMatchInfo contient les infos du dernier match visible par un joueur.
type LastMatchInfo struct {
	MapName      string
	PlaylistName string
	VariantName  string
	IsRanked     bool
	StartTime    time.Time
	Kills        int
	Deaths       int
	Assists      int
	// Outcome : 1=Tie 2=Win 3=Loss 4=Quit
	Outcome      int
	SquadFriends []string
}

// BackfillCounts regroupe les compteurs de backfill par type de données.
type BackfillCounts struct {
	MedalsInserted         int
	EventsInserted         int
	LUSRComputed           int
	CSRFetched             int
	SessionsUpdated        int
	CitationsComputed      int
	KillerVictimPairs      int
	PersonalScoresInserted int
	PerfScoresInserted     int
	AliasesInserted        int
	PveStatsInserted       int
}

// PlayerSyncResult encapsule le résultat d'une opération sync/backfill pour un joueur.
type PlayerSyncResult struct {
	Gamertag      string
	MatchesSynced int
	MissingData   int
	Error         string
	Backfill      BackfillCounts
	LastMatch     *LastMatchInfo
}

// ─────────────────────────────────────────────────────────────────────────────
// Construction de l'embed sync/backfill
// ─────────────────────────────────────────────────────────────────────────────

// BuildSyncEmbed crée le Rich Embed Discord pour une opération sync ou backfill,
// avec les libellés Halo par défaut (byte-identique au comportement historique).
func BuildSyncEmbed(
	op string,
	startedAt, finishedAt time.Time,
	players []PlayerSyncResult,
	success bool,
	lang string,
) Embed {
	return BuildSyncEmbedWithLabels(op, startedAt, finishedAt, players, success, lang, HaloLabels())
}

// BuildSyncEmbedWithLabels est la variante title-aware : les libellés d'outcome
// passent par `labels` (PMT-11). labels nil → libellés Halo (failsafe).
func BuildSyncEmbedWithLabels(
	op string,
	startedAt, finishedAt time.Time,
	players []PlayerSyncResult,
	success bool,
	lang string,
	labels NotifyLabels,
) Embed {
	labels = labelsOrDefault(labels)
	opLabel := resolveOpLabel(op, lang)
	duration := formatDuration(startedAt, finishedAt)

	statusIcon := "✅"
	color := colorSuccess
	if !success {
		statusIcon = "❌"
		color = colorError
	} else if hasMissingData(players) {
		statusIcon = "⚠️"
		color = colorWarning
	} else if allIdle(players) {
		color = colorWarning
	}

	desc := T("discord_completed_in", lang,
		"status", statusIcon,
		"op", opLabel,
		"duration", duration,
	)

	totalMatches := 0
	for _, p := range players {
		totalMatches += p.MatchesSynced
	}
	desc += "\n" + T("discord_players_matches", lang,
		"players", len(players),
		"matches", totalMatches,
	)

	// Plage horaire
	if !startedAt.IsZero() && !finishedAt.IsZero() {
		tStart := startedAt.Local().Format("15:04")
		tEnd := finishedAt.Local().Format("15:04")
		desc += "\n" + T("discord_time_range", lang, "t_start", tStart, "t_end", tEnd)
	}

	title := T("discord_title", lang, "op", opLabel)

	embed := Embed{
		Title:       title,
		Description: desc,
		Color:       color,
		Footer:      &EmbedFooter{Text: discordFooterText()},
		Timestamp:   finishedAt.UTC().Format(time.RFC3339),
	}

	// Un field par joueur
	for _, p := range players {
		name, value := buildPlayerField(p, op, lang, labels)
		embed.Fields = append(embed.Fields, EmbedField{
			Name:  truncate(name, 256),
			Value: truncate(value, 1024),
		})
	}

	return embed
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers field joueur
// ─────────────────────────────────────────────────────────────────────────────

func buildPlayerField(p PlayerSyncResult, op, lang string, labels NotifyLabels) (string, string) {
	name := "👤  " + p.Gamertag
	var lines []string

	// Résumé de l'opération
	if strings.HasPrefix(op, "sync") {
		if p.MatchesSynced == 0 {
			lines = append(lines, T("discord_up_to_date_sync", lang))
		} else {
			lines = append(lines, T("discord_matches_synced", lang, "count", p.MatchesSynced))
		}
	} else {
		if p.MatchesSynced == 0 {
			lines = append(lines, T("discord_no_matches_to_reprocess", lang))
		} else {
			lines = append(lines, T("discord_matches_processed", lang, "count", p.MatchesSynced))
		}
	}

	// Complétude
	if p.Error != "" {
		errMsg := p.Error
		if len(errMsg) > 80 {
			errMsg = errMsg[:80]
		}
		lines = append(lines, T("discord_error_field", lang, "error", errMsg))
	} else if p.MissingData == 0 {
		lines = append(lines, T("discord_data_complete", lang))
	} else {
		lines = append(lines, T("discord_data_incomplete", lang, "count", p.MissingData))
	}

	// Compteurs backfill
	lines = append(lines, backfillLines(p.Backfill, lang)...)

	// Dernier match
	if p.LastMatch != nil {
		lines = append(lines, lastMatchLines(p.LastMatch, lang, labels)...)
	}

	return name, strings.Join(lines, "\n")
}

// outcomeCanonicalKey : pont du code Outcome Discord (1=Tie 2=Win 3=Loss 4=Quit)
// vers la clé canonique (tie|win|loss|dnf) du manifeste outcomes.toml. « Quit »
// Discord correspond au DNF canonique.
var outcomeCanonicalKey = map[int]string{1: "tie", 2: "win", 3: "loss", 4: "dnf"}

func backfillLines(bc BackfillCounts, lang string) []string {
	type pair struct {
		count int
		key   string
	}
	entries := []pair{
		{bc.MedalsInserted, "discord_bf_medals"},
		{bc.EventsInserted, "discord_bf_events"},
		{bc.LUSRComputed, "discord_bf_lusr"},
		{bc.CSRFetched, "discord_bf_csr"},
		{bc.SessionsUpdated, "discord_bf_sessions"},
		{bc.CitationsComputed, "discord_bf_citations"},
		{bc.KillerVictimPairs, "discord_bf_kvp"},
		{bc.PersonalScoresInserted, "discord_bf_personal_scores"},
		{bc.PerfScoresInserted, "discord_bf_perf_scores"},
		{bc.AliasesInserted, "discord_bf_aliases"},
		{bc.PveStatsInserted, "discord_bf_pve"},
	}
	var lines []string
	for _, e := range entries {
		if e.count > 0 {
			lines = append(lines, T(e.key, lang, "count", e.count))
		}
	}
	return lines
}

func lastMatchLines(lm *LastMatchInfo, lang string, labels NotifyLabels) []string {
	labels = labelsOrDefault(labels)
	outcomeIcons := map[int]string{1: "⚖️", 2: "🏆", 3: "💀", 4: "🚶"}
	icon := outcomeIcons[lm.Outcome]
	if icon == "" {
		icon = "❓"
	}
	outcomeLabel := "—"
	if key, ok := outcomeCanonicalKey[lm.Outcome]; ok {
		outcomeLabel = labels.Outcome(key, lang)
	}
	rankedTag := ""
	if lm.IsRanked {
		rankedTag = fmt.Sprintf(" · 🏅 %s", T("discord_ranked_tag", lang))
	}
	kda := T("discord_kda", lang, "k", lm.Kills, "d", lm.Deaths, "a", lm.Assists)
	timeStr := "—"
	if !lm.StartTime.IsZero() {
		timeStr = lm.StartTime.Local().Format("02/01 15:04")
	}
	lastLabel := T("discord_last_match", lang)
	line := fmt.Sprintf(
		"**%s** (%s)%s\n🗺️  **%s**  ·  🎮 %s\n📋  %s\n📊  %s  ·  %s %s",
		lastLabel, timeStr, rankedTag,
		lm.MapName, lm.VariantName,
		lm.PlaylistName,
		kda, icon, outcomeLabel,
	)
	lines := []string{line}
	if len(lm.SquadFriends) > 0 {
		lines = append(lines, T("discord_squad_match", lang))
		lines = append(lines, T("discord_squad_friends", lang, "friends", strings.Join(lm.SquadFriends, ", ")))
	}
	return lines
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

func resolveOpLabel(op, lang string) string {
	switch {
	case op == "sync_delta" || op == "delta":
		return T("discord_op_sync_delta", lang)
	case op == "sync_full" || op == "full":
		return T("discord_op_sync_full", lang)
	case strings.HasPrefix(op, "backfill"):
		return T("discord_op_backfill", lang)
	default:
		return op
	}
}

func formatDuration(start, end time.Time) string {
	total := int(end.Sub(start).Seconds())
	if total < 0 {
		total = 0
	}
	if total < 60 {
		return fmt.Sprintf("%ds", total)
	}
	mins, secs := total/60, total%60
	if mins < 60 {
		return fmt.Sprintf("%dm%02ds", mins, secs)
	}
	hours := mins / 60
	mins = mins % 60
	return fmt.Sprintf("%dh%02dm%02ds", hours, mins, secs)
}

func hasMissingData(players []PlayerSyncResult) bool {
	for _, p := range players {
		if p.MissingData > 0 || p.Error != "" {
			return true
		}
	}
	return false
}

func allIdle(players []PlayerSyncResult) bool {
	for _, p := range players {
		if p.MatchesSynced > 0 {
			return false
		}
	}
	return true
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}
