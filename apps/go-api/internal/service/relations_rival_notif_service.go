// Package service — relations_rival_notif_service.go : détection des « rivaux
// croisés » en post-sync (lot relations-E). Orchestration pure au-dessus de
// port.RelationsRepository (AUCUN SQL nouveau) : réutilise GetRelations +
// selectTopRivals + GetRivalTimeline, exactement le chemin des cartes revanche.
//
// Contrat : DetectRivalEncounters retourne les nouveaux duels (matchs en ennemi)
// contre les top rivaux dont le start canonique est postérieur au watermark
// `since` (dernier match connu AVANT la sync). Le watermark rend l'émission
// idempotente (une re-sync des mêmes matchs ne redétecte rien). Best-effort côté
// appelant : toute erreur est loguée, jamais bloquante pour la sync.
package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"levelup/go-api/internal/analysis/relations"
	"levelup/go-api/internal/domain"
)

// Garde-fous nommés du lot relations-E (pas de magic number).
const (
	// rivalNotifMaxPerSync : plafond de notifications « rival croisé » par cycle de
	// sync — anti-spam après une longue absence (un backfill ramenant N duels ne
	// doit pas produire N notifications). Les plus RÉCENTS sont conservés.
	rivalNotifMaxPerSync = 3
	// rivalNotifMaxAgeDays : âge maximal (en jours) d'un duel pour mériter une
	// notification. Filtre les cas backfill/import d'historique ancien : un duel
	// plus vieux que ce seuil n'est jamais notifié (l'événement « revanche » n'a
	// de sens que juste après le match).
	rivalNotifMaxAgeDays = 7
)

// DetectRivalEncounters retourne les nouveaux duels contre les top rivaux du
// joueur dont le start canonique est strictement postérieur à `since` (watermark
// = dernier match connu avant la sync) ET pas plus vieux que rivalNotifMaxAgeDays.
//
// `since` zéro (aucun watermark : premier passage ou snapshot before froid) →
// retourne nil sans requête : sans borne basse fiable, tout l'historique
// apparaîtrait comme « nouveau » (même garde-fou anti-burst que le post-sync).
//
// Résultat trié du plus récent au plus ancien, plafonné à rivalNotifMaxPerSync.
// Réutilise le MÊME chemin que les cartes revanche : selectTopRivals (top par
// matchs en ennemi, seuil momentsRivalMinEnemyMatches) + GetRivalTimeline
// (limite momentsTimelineLimit). Aucun SQL nouveau.
func (s *RelationsService) DetectRivalEncounters(ctx context.Context, since time.Time) ([]domain.RivalEncounter, error) {
	if since.IsZero() {
		return nil, nil
	}
	now := s.now()
	ageCutoff := now.Add(-time.Duration(rivalNotifMaxAgeDays) * 24 * time.Hour)

	// Scope nil = tous les matchs (le post-sync n'a pas de segmentation active).
	rawRows, err := s.repo.GetRelations(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("RelationsService.DetectRivalEncounters: relations: %w", err)
	}
	rivals := selectTopRivals(rawRows, momentsMaxRivalries)

	out := make([]domain.RivalEncounter, 0, rivalNotifMaxPerSync)
	for _, rv := range rivals {
		duels, err := s.repo.GetRivalTimeline(ctx, rv.XUID, nil, momentsTimelineLimit)
		if err != nil {
			return nil, fmt.Errorf("RelationsService.DetectRivalEncounters: timeline %s: %w", rv.XUID, err)
		}
		for _, d := range duels {
			if !d.StartTime.After(since) || d.StartTime.Before(ageCutoff) {
				continue
			}
			out = append(out, domain.RivalEncounter{
				XUID:          rv.XUID,
				Gamertag:      rv.Gamertag,
				MatchID:       d.MatchID,
				StartedAt:     d.StartTime,
				Outcome:       duelOutcomeLabel(relations.ResultToDuel(d.Result)),
				KillsOnRival:  d.KillsOnRival,
				DeathsByRival: d.DeathsByRival,
			})
		}
	}

	// Plus récent d'abord (tiebreak match_id pour le déterminisme), puis plafond.
	sort.Slice(out, func(i, j int) bool {
		if !out[i].StartedAt.Equal(out[j].StartedAt) {
			return out[i].StartedAt.After(out[j].StartedAt)
		}
		return out[i].MatchID < out[j].MatchID
	})
	if len(out) > rivalNotifMaxPerSync {
		out = out[:rivalNotifMaxPerSync]
	}
	return out, nil
}
