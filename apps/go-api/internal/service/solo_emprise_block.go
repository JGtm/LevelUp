// Package service — solo_emprise_block.go : L'EMPRISE sur une liste de matchs, commune aux Séries
// temporelles (la fenêtre filtrée), à la page Sessions (une session) et à la Vue match (un match).
//
// Orchestration seule : les lectures sont celles de l'Escouade (`squadagg.EmpriseLecteur`), le calcul
// est `analysis/squademprise` (Build, BuildMaps, BuildEquipment). Les fiches : le joueur de la page
// seul, ou la liste fournie par la page (Vue match : les joueurs de l'équipe) ; pas d'habitude ; pas
// de placement des vies. Chaque
// source dégrade seule et le dit (journaux portant l'attribut `page`).
package service

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/analysis/squademprise"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/squadagg"
)

// soloEmpriseQuery — ce que la page fournit à l'assemblage.
type soloEmpriseQuery struct {
	// Page : `timeseries` ou `sessions`, posé sur chaque journal. Player : gamertag du joueur.
	Page       string
	Player     string
	PlayerXUID string
	// RepoRoot / TitleSlug / Locale : catalogue d'armes et noms de véhicules du titre.
	RepoRoot  string
	TitleSlug string
	Locale    string
	// Current : les matchs, déjà projetés (carte, résultat, session) par la page.
	Current []squademprise.Match
	// Lectures : le résumé d'usage déjà lu sur ces matchs par la page (ADR 0036 I4) ; nil = lu ici.
	Lectures *squadagg.LecturesUsage
	// UsageRepo : nil = titre sans résumé d'usage. EmpriseRepo : la feuille de match. VehicleRepo :
	// nil = ressource véhicules non mesurée par le titre.
	UsageRepo   port.SessionUsageRepository
	EmpriseRepo port.SquadEmpriseRepository
	VehicleRepo port.SquadVehicleRepository
	// WithMaps : calcule la grille par carte (Séries temporelles seulement).
	WithMaps bool
	// Players : les joueurs des fiches, dans l'ordre de la page (Vue match : les joueurs de l'équipe).
	// nil = le joueur de la page seul (Séries temporelles, Sessions).
	Players []domain.SessionUsageSquadPlayer
}

// buildSoloEmpriseBlock rend le bloc des matchs `q.Current`. L'appelant garantit une liste non vide
// et un joueur connu.
func buildSoloEmpriseBlock(ctx context.Context, q soloEmpriseQuery) *domain.SoloEmpriseBlock {
	ids := squadagg.EmpriseMatchIDs(q.Current)
	lecteur := squadagg.EmpriseLecteur{Page: q.Page, Player: q.Player, RepoRoot: q.RepoRoot, TitleSlug: q.TitleSlug}
	in := squademprise.Input{
		PlayerXUID: q.PlayerXUID, Current: q.Current,
		Weapons: squadagg.WeaponCatalog(ctx, q.RepoRoot, q.TitleSlug, q.Locale),
	}
	in.PowerKills, in.SheetUnavailable = lecteur.Feuille(ctx, q.EmpriseRepo, ids)
	in.Journal = lecteur.Journal(ctx, q.EmpriseRepo, ids, in.Weapons)
	in.Film, in.FilmUnavailable = lecteur.Film(ctx, q.UsageRepo, q.Current, nil, q.Lectures)
	in.Vehicles, in.VehicleLabels = lecteur.Vehicules(ctx, q.VehicleRepo, ids, q.PlayerXUID, q.Locale)
	switch {
	case q.Players != nil:
		in.Players = q.Players
	case in.Film != nil:
		in.Players = squadagg.SquadPlayers(q.PlayerXUID, q.Player, in.Film.Participants, nil)
	default:
		in.Players = squadagg.SquadPlayers(q.PlayerXUID, q.Player, nil, nil)
	}
	if n := squademprise.WithoutTimeScale(&in); n > 0 {
		// Même règle et même trace que l'Escouade : exclus du rendement des bonus, jamais en silence.
		slog.WarnContext(ctx, "emprise_matchs_sans_echelle_de_temps_hors_rendement",
			"page", q.Page, "player", q.Player, "matchs_sans_echelle", n, "matchs", len(q.Current))
	}
	block := &domain.SoloEmpriseBlock{SquadEmpriseBlock: squademprise.Build(in), Equipment: squademprise.BuildEquipment(in)}
	if q.WithMaps {
		block.Maps = squademprise.BuildMaps(in)
	}
	slog.DebugContext(ctx, "emprise",
		"page", q.Page, "player", q.Player, "matchs", block.MatchesTotal, "mesures", block.MatchesMeasured,
		"cartes", len(block.Maps), "film", block.FilmUnavailable, "feuille", block.SheetUnavailable,
		"equipement", block.Equipment != nil)
	return block
}
