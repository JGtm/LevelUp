// Package service — match_view_tools.go : « OUTILS DE DESTRUCTION » DU JOUEUR DE LA PAGE sur la Vue
// match (plan `.ai/V7.5/PLAN_MATCHVIEW_EMPRISE_2026-10-06.md`, D9) — le builder de l'Escouade et de
// Sessions (`squadagg.BuildWeaponTools`), un joueur, un match.
//
// Aucune lecture neuve hors des catégories de source : les lignes par arme sont les frags par arme du
// match déjà chargés par la page (libellés dans la langue de la requête), la feuille est la ligne du
// joueur au tableau des scores.
package service

import (
	"context"
	"errors"
	"log/slog"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/squadagg"
)

// matchWeaponTools rend les outils de destruction du joueur de la page ; nil sans ligne de tableau
// des scores, sans frag ou sans ligne d'outil.
func (s *MatchViewService) matchWeaponTools(
	ctx context.Context, matchID string, bulk []domain.BulkWeaponKillRaw, me *domain.MatchScoreboardRow,
) *domain.SquadWeaponTools {
	if me == nil || derefInt(me.Kills) <= 0 {
		return nil
	}
	gt := me.Gamertag
	sheet := map[string]domain.FragKillTypeCounts{gt: {
		Melee:         derefInt(me.MeleeKills),
		Grenade:       derefInt(me.GrenadeKills),
		Assassination: derefInt(me.AssassinationKills),
		GroundPound:   derefInt(me.GroundPoundKills),
		ShoulderBash:  derefInt(me.ShoulderBashKills),
		Total:         derefInt(me.Kills),
	}}
	tools := squadagg.BuildWeaponTools(squadagg.WeaponToolInputs{
		Rows:           matchToolRows(bulk),
		Categories:     s.matchToolCategories(ctx, matchID, me.XUID),
		PlayersOrdered: []string{gt},
		GtByXUID:       map[string]string{me.XUID: gt},
		Sheet:          sheet,
		HasMechanics:   titleHasNativeKillMechanics(s.titleSlug),
	})
	if tools != nil {
		slog.DebugContext(ctx, "match_view_outils", "match_id", matchID, "lignes", len(tools.Lines),
			"joueurs_au_dela_de_la_feuille", squadagg.PlayersAboveSheet(tools, sheet))
	}
	return tools
}

// matchToolRows projette les frags par arme du match en lignes du builder (qui ne garde que celles
// des joueurs qu'on lui nomme). Le libellé est déjà dans la langue de la requête : il sert de libellé
// et de libellé anglais.
func matchToolRows(bulk []domain.BulkWeaponKillRaw) []port.WeaponKillRow {
	out := make([]port.WeaponKillRow, 0, len(bulk))
	for _, w := range bulk {
		out = append(out, port.WeaponKillRow{
			XUID: w.XUID, WeaponID: w.WeaponID, Kills: w.Kills, Label: w.WeaponLabel, LabelEN: w.WeaponLabel,
			Role: w.Role, Class: w.Class, Family: w.Family, WeaponKey: w.WeaponKey,
			FromDamageSource: w.FromDamageSource, MechanicKills: w.MechanicKills,
		})
	}
	return out
}

// matchToolCategories lit les frags du joueur par catégorie de source du film sur ce match. nil (sans
// erreur de page) sans lecteur ou sans film ; toute autre erreur est journalisée avant la dégradation.
func (s *MatchViewService) matchToolCategories(ctx context.Context, matchID, xuid string) []port.KillSourceCategoryRow {
	if s.emprise.categories == nil {
		return nil
	}
	rows, err := s.emprise.categories.LoadKillSourceCategoryKills(ctx, s.titleSlug, port.WeaponKillFilters{
		MatchIDs: []string{matchID}, XUIDs: []string{xuid},
	})
	switch {
	case errors.Is(err, games.ErrCapabilityNotSupported):
		slog.DebugContext(ctx, "match_view_outils_categories_non_supportees", "title", s.titleSlug)
		return nil
	case err != nil:
		slog.WarnContext(ctx, "match_view_outils_categories_en_echec", "match_id", matchID, "err", err)
		return nil
	}
	return rows
}
