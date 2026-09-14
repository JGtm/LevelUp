// Package service — timeseries_service_sections.go : LES DEUX SECTIONS MIGRÉES DEPUIS LA
// SYNTHÈSE le 2026-09-13 — « Portée des engagements » (onglet Résumé) et « Usages
// d'équipement » (onglet Progression).
//
// AUCUN CALCUL NEUF ICI, ET C'EST LE POINT. Les deux blocs gardent le producteur de la
// Synthèse (`buildWeaponRangeSection`, `squadagg.BuildEquipmentUsageBlock`) et le MÊME scope
// que le reste de la page — les matchs déjà filtrés. Recoder la lecture côté Timeseries
// aurait créé une seconde doctrine de scope, qui aurait divergé au premier correctif.
//
// Fichier séparé : timeseries_service.go tient le plafond des 500 lignes du dépôt.
package service

import (
	"context"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/teammates"
)

// WithWeaponRangeRepo injecte le loader de la section « Portée des engagements » (frags et
// morts MESURÉS, distance et dénivelé). Repo nil / titre sans positions par kill ⇒ section
// omise (dégradation gracieuse, jamais une section vide).
func (s *TimeseriesService) WithWeaponRangeRepo(repo port.WeaponRangeRepository) *TimeseriesService {
	s.weaponRangeRepo = repo
	return s
}

// WithEquipmentUsage injecte la source du résumé d'usage (vues _latest) et le résolveur
// d'amis configurés — la MÊME paire que la Synthèse et la page Sessions. Câblé gated par
// film.usage_summary ; repo nil ⇒ bloc servi avec Available=false et raison machine.
//
// `repoRoot` ne sert qu'au CATALOGUE D'ARMES du titre (nommage du detail par niveau) : vide,
// les armes s'affichent sous leur cle — la degradation ecrite partout ailleurs.
func (s *TimeseriesService) WithEquipmentUsage(
	repo port.SessionUsageRepository, friends teammates.FriendGamertagsResolver, repoRoot string,
) *TimeseriesService {
	s.sessionUsageRepo = repo
	s.usageFriends = friends
	s.repoRoot = repoRoot
	return s
}

// attachMigratedSections pose les deux blocs sur la réponse, depuis le scope canonique déjà
// filtré. Best-effort de bout en bout : chaque producteur rend nil plutôt que de casser la page.
func (s *TimeseriesService) attachMigratedSections(
	ctx context.Context, resp *domain.TimeseriesPageResponse, filteredCanon []canonical.PlayerMatchRow,
) {
	resp.WeaponRange = buildWeaponRangeSection(ctx, weaponRangeQuery{
		Repo: s.weaponRangeRepo, TitleSlug: s.titleSlug, Gamertag: s.gamertag, Rows: filteredCanon,
	})
	resp.EquipmentUsage = buildEquipmentUsageBlock(ctx, equipmentUsageQuery{
		Repo:            s.sessionUsageRepo,
		PlayerXUID:      s.playerXUID,
		MatchIDs:        synthesisMatchIDs(filteredCanon),
		FriendGamertags: s.timeseriesFriendGamertags(ctx),
		// De quoi NOMMER les armes du detail par niveau (catalogue du titre).
		RepoRoot:  s.repoRoot,
		TitleSlug: s.titleSlug,
	})
}

// timeseriesFriendGamertags résout les amis configurés (nil = aucun ami déclaré).
func (s *TimeseriesService) timeseriesFriendGamertags(ctx context.Context) []string {
	if s.usageFriends == nil {
		return nil
	}
	return s.usageFriends(ctx)
}
