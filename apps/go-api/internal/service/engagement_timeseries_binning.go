// Package service — engagement_timeseries_binning.go : agregation adaptative
// session/week/month pour la reponse Mock 11 (POST /engagement/timeseries).
//
// Decoupe depuis engagement_player_service.go pour respecter la limite 500L
// (CLAUDE.md regle 14) et isoler la logique pure de binning du wiring service.
//
// Les helpers sont stateless : entree []EngagementMatchSummary (1 point = 1
// match, MatchCount=1) + eventuellement les StatsMatchRow pour resoudre les
// session_label. Sortie : []EngagementMatchSummary agreges (MatchCount > 1
// pour les buckets multi-matchs), ordre chronologique croissant.
package service

import (
	"fmt"
	"sort"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

// engagementSyntheticSessionPrefix prefixe les clefs de bucket pour les matchs
// sans session_label. Garantit qu'ils ne se melangent pas avec les vrais
// session_label (qui ne contiennent jamais "_match:" en pratique).
const engagementSyntheticSessionPrefix = "_match:"

// engagementBucketAcc accumule les paces + scores d'un bucket multi-matchs.
type engagementBucketAcc struct {
	label                                        string
	start                                        time.Time
	count                                        int
	paceJoueur, paceTeam, paceAttendu, paceLobby float64
	scoreSum                                     float64
	scoreCount                                   int
	durationSeconds                              int64
}

// addSummary integre un EngagementMatchSummary dans l'accumulateur.
func (a *engagementBucketAcc) addSummary(s domain.EngagementMatchSummary, anchor time.Time, anchorIsFixed bool) {
	if a.count == 0 {
		a.start = anchor
	} else if !anchorIsFixed && s.StartedAt.Before(a.start) {
		// Pour les buckets session : on suit le plus ancien match.
		a.start = s.StartedAt
	}
	a.count++
	a.paceJoueur += s.PaceJoueur
	a.paceTeam += s.PaceTeam
	a.paceAttendu += s.PaceAttendu
	a.paceLobby += s.PaceLobby
	// Duree SOMMEE sur le bucket (pas moyennee) : l'ecart d'engagement cumule
	// pondere le score par le temps total joue dans le bin.
	a.durationSeconds += s.DurationSeconds
	if s.EngagementScore != nil {
		a.scoreSum += *s.EngagementScore
		a.scoreCount++
	}
}

// aggregateEngagementBySession regroupe les summaries par session_label des
// StatsMatchRow correspondantes. Les matchs sans session_label deviennent des
// buckets singletons (cle="_match:"+match_id), preservant la donnee mais
// reduisant l'efficacite du binning a session — la cascade adaptive
// (week/month en aval) compense.
//
// Les paces agreges sont la moyenne arithmetique simple (chaque match contribue
// egalement). EngagementScore est la moyenne des scores non-nuls du bucket
// (nil si aucun match du bucket n'a de score percentile calculable).
func aggregateEngagementBySession(
	summaries []domain.EngagementMatchSummary,
	rows []legacymatch.StatsMatchRow,
) []domain.EngagementMatchSummary {
	labelByMatchID := make(map[string]string, len(rows))
	for _, r := range rows {
		if r.SessionLabel != nil && *r.SessionLabel != "" {
			labelByMatchID[r.MatchID] = *r.SessionLabel
		}
	}
	buckets := make(map[string]*engagementBucketAcc)
	keyOrder := make([]string, 0, len(summaries))
	for _, s := range summaries {
		label := labelByMatchID[s.MatchID]
		key := label
		displayLabel := label
		if label == "" {
			key = engagementSyntheticSessionPrefix + s.MatchID
			displayLabel = s.Label // garde "M1, M2…" pour les singletons
		}
		a, ok := buckets[key]
		if !ok {
			a = &engagementBucketAcc{label: displayLabel}
			buckets[key] = a
			keyOrder = append(keyOrder, key)
		}
		a.addSummary(s, s.StartedAt, false)
	}
	return finalizeEngagementBuckets(buckets, keyOrder)
}

// rollupEngagementByPeriod regroupe les summaries par semaine ISO ou mois
// (mode = "week" | "month"). Label = "2026-S18" / "2026-05", StartedAt =
// debut canonique du bucket (lundi ISO de la semaine, 1er du mois).
func rollupEngagementByPeriod(
	summaries []domain.EngagementMatchSummary,
	mode string,
) []domain.EngagementMatchSummary {
	buckets := make(map[string]*engagementBucketAcc)
	keyOrder := make([]string, 0)
	for _, s := range summaries {
		key, label, start := periodKey(s.StartedAt, mode)
		a, ok := buckets[key]
		if !ok {
			a = &engagementBucketAcc{label: label}
			buckets[key] = a
			keyOrder = append(keyOrder, key)
		}
		a.addSummary(s, start, true)
	}
	return finalizeEngagementBuckets(buckets, keyOrder)
}

// periodKey calcule (key, label, start) pour un timestamp donne et un mode
// "week" ou "month". key sert d'identifiant interne, label est affiche.
func periodKey(t time.Time, mode string) (key, label string, start time.Time) {
	switch mode {
	case "month":
		d := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
		k := d.Format("2006-01")
		return k, k, d
	default: // "week"
		y, w := t.ISOWeek()
		// Jan 4 est toujours dans la semaine ISO 1 (definition ISO 8601) ;
		// on recule au lundi precedent pour avoir le debut de semaine.
		jan4 := time.Date(y, 1, 4, 0, 0, 0, 0, time.UTC)
		mondayOffset := int(jan4.Weekday()) - int(time.Monday)
		if mondayOffset < 0 {
			mondayOffset += 7
		}
		wkStart := jan4.AddDate(0, 0, -mondayOffset+(w-1)*7)
		k := fmt.Sprintf("%d-S%02d", y, w)
		return k, k, wkStart
	}
}

// finalizeEngagementBuckets convertit les accumulateurs en
// EngagementMatchSummary, applique les moyennes et trie chronologiquement
// croissant.
func finalizeEngagementBuckets(
	buckets map[string]*engagementBucketAcc, keyOrder []string,
) []domain.EngagementMatchSummary {
	out := make([]domain.EngagementMatchSummary, 0, len(buckets))
	for _, key := range keyOrder {
		a := buckets[key]
		if a.count == 0 {
			continue
		}
		pt := domain.EngagementMatchSummary{
			Label:           a.label,
			StartedAt:       a.start,
			MatchCount:      a.count,
			PaceJoueur:      a.paceJoueur / float64(a.count),
			PaceTeam:        a.paceTeam / float64(a.count),
			PaceAttendu:     a.paceAttendu / float64(a.count),
			PaceLobby:       a.paceLobby / float64(a.count),
			DurationSeconds: a.durationSeconds,
		}
		if a.scoreCount > 0 {
			v := a.scoreSum / float64(a.scoreCount)
			pt.EngagementScore = &v
		}
		out = append(out, pt)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].StartedAt.Before(out[j].StartedAt)
	})
	return out
}
