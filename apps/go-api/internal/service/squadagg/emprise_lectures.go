// Package squadagg — emprise_lectures.go : LES LECTURES DU BLOC « EMPRISE », communes à la page
// Escouade et aux Séries temporelles (calcul : analysis/squademprise).
//
// Trois sources, qui dégradent chacune seule et le disent :
//
//	feuille    les frags aux armes spéciales (port.SquadEmpriseRepository, tous titres) : feuille de
//	           match et, sur les matchs aux niveaux mesurés, journal des morts du film ;
//	film       le résumé d'usage et les niveaux de socle (port.SessionUsageRepository, câblé sous
//	           `film.usage_summary`) ;
//	véhicules  la ressource véhicules (port.SquadVehicleRepository, câblé sous `film.vehicle_usage`).
//
// Les journaux portent l'attribut `page` (`teammates`, `timeseries`) : deux pages lisent par ces
// fonctions, une seule doctrine de dégradation.
package squadagg

import (
	"context"
	"errors"
	"log/slog"

	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/analysis/squademprise"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/observability/timing"
	"levelup/go-api/internal/port"
)

// EmpriseLecteur — qui lit (la page, pour les journaux) et pour quel joueur et quel titre.
type EmpriseLecteur struct {
	// Page : `teammates` ou `timeseries`, posé sur chaque journal.
	Page string
	// Player : le gamertag du joueur de la page (journaux seulement).
	Player string
	// RepoRoot / TitleSlug : la résolution des noms de véhicules qualifiés (VehicleFamilyLabels).
	RepoRoot  string
	TitleSlug string
}

// Feuille lit les frags aux armes spéciales des matchs `ids`. Repo nil ou capability non supportée :
// EmpriseSheetUnsupported ; lecture en échec : EmpriseSheetLoadFailed — jamais une erreur de page.
func (l EmpriseLecteur) Feuille(
	ctx context.Context, repo port.SquadEmpriseRepository, ids []string,
) ([]squademprise.PowerKillRow, string) {
	if repo == nil {
		slog.WarnContext(ctx, "emprise_feuille_non_cablee", "page", l.Page, "player", l.Player)
		return nil, domain.EmpriseSheetUnsupported
	}
	rows, err := repo.LoadPowerWeaponKills(ctx, ids)
	switch {
	case errors.Is(err, games.ErrCapabilityNotSupported):
		slog.InfoContext(ctx, "emprise_feuille_non_supportee", "page", l.Page, "player", l.Player, "err", err)
		return nil, domain.EmpriseSheetUnsupported
	case err != nil:
		slog.ErrorContext(ctx, "emprise_feuille_en_echec",
			"page", l.Page, "player", l.Player, "matchs", len(ids), "err", err)
		return nil, domain.EmpriseSheetLoadFailed
	}
	return rows, ""
}

// Journal lit les frags par arme du journal des morts des matchs `ids` (frags aux armes spéciales
// des matchs aux niveaux mesurés). Nil quand il ne se lit pas : repo nil, capability non supportée,
// lecture en échec — les frags aux armes spéciales se lisent alors sur la feuille de match, et
// l'échec est journalisé (jamais une erreur de page).
func (l EmpriseLecteur) Journal(ctx context.Context, repo port.SquadEmpriseRepository, ids []string) *squademprise.JournalRead {
	if repo == nil || len(ids) == 0 {
		return nil
	}
	read, err := repo.LoadJournalWeaponKills(ctx, ids)
	switch {
	case errors.Is(err, games.ErrCapabilityNotSupported):
		slog.DebugContext(ctx, "emprise_journal_non_supporte", "page", l.Page, "player", l.Player, "err", err)
		return nil
	case err != nil:
		slog.ErrorContext(ctx, "emprise_journal_en_echec_repli_feuille",
			"page", l.Page, "player", l.Player, "matchs", len(ids), "err", err)
		return nil
	}
	return &read
}

// Film rend le résumé d'usage et les niveaux de socle du périmètre `current` ET des matchs
// comparables des soirées précédentes (`timeline`, habitude). Les lectures du périmètre déjà faites
// par la page (`partage`, nil = non faites) sont réutilisées ; seuls les matchs de l'habitude sont
// lus en plus. Repo nil : le titre n'a pas de résumé d'usage (EmpriseFilmUnsupported).
func (l EmpriseLecteur) Film(
	ctx context.Context, repo port.SessionUsageRepository, current, timeline []squademprise.Match,
	partage *LecturesUsage,
) (*squademprise.FilmData, string) {
	if repo == nil {
		slog.DebugContext(ctx, "emprise_sans_film", "page", l.Page, "player", l.Player)
		return nil, domain.EmpriseFilmUnsupported
	}
	ids := EmpriseMatchIDs(current)
	if partage == nil {
		partage = LireUsage(ctx, repo, ids)
	}
	if raison := l.raisonFilm(ctx, partage.Erreur(), len(ids)); raison != "" {
		return nil, raison
	}
	film := &squademprise.FilmData{Films: map[string]sessionusage.FilmRow{}, Players: partage.Players, Participants: partage.Participants}
	for id, f := range partage.Films {
		film.Films[id] = f
	}
	extras := squademprise.HabitCandidates(current, timeline)
	if len(extras) > 0 {
		l.ajouterHabitude(ctx, repo, extras, film)
	}
	ids = append(ids, extras...)
	tiers, err := repo.LoadPadTiers(ctx, ids)
	if err != nil {
		// Les niveaux dégradent SEULS : bonus et feuille restent, armes spéciales et râteliers se
		// lisent « non mesurés ».
		slog.ErrorContext(ctx, "emprise_niveaux_en_echec",
			"page", l.Page, "player", l.Player, "matchs", len(ids), "err", err)
		return film, ""
	}
	film.PadTiers = tiers
	return film, ""
}

// ajouterHabitude lit les matchs de l'habitude et les verse dans `film` ; un échec dégrade
// l'habitude SEULE (ses soirées précédentes restent sans point).
func (l EmpriseLecteur) ajouterHabitude(
	ctx context.Context, repo port.SessionUsageRepository, extras []string, film *squademprise.FilmData,
) {
	lx := LireUsage(ctx, repo, extras)
	if err := lx.Erreur(); err != nil {
		slog.ErrorContext(ctx, "emprise_habitude_en_echec",
			"page", l.Page, "player", l.Player, "matchs", len(extras), "err", err)
		return
	}
	for id, f := range lx.Films {
		film.Films[id] = f
	}
	film.Players = append(append([]sessionusage.PlayerRow(nil), film.Players...), lx.Players...)
	film.Participants = append(append([]sessionusage.ParticipantRow(nil), film.Participants...), lx.Participants...)
}

// raisonFilm traduit l'échec d'une lecture du résumé d'usage : capability non supportée (la
// dégradation voulue) ou panne (dite comme telle). "" = lecture réussie.
func (l EmpriseLecteur) raisonFilm(ctx context.Context, err error, n int) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, games.ErrCapabilityNotSupported):
		slog.InfoContext(ctx, "emprise_film_non_supporte", "page", l.Page, "player", l.Player, "err", err)
		return domain.EmpriseFilmUnsupported
	default:
		slog.ErrorContext(ctx, "emprise_film_en_echec", "page", l.Page, "player", l.Player, "matchs", n, "err", err)
		return domain.EmpriseFilmLoadFailed
	}
}

// Vehicules charge la ressource véhicules des matchs `ids` (périmètre ET habitude) et les noms des
// familles qualifiées. (nil, "", nil) quand le titre ne la mesure pas (repo nil ou capability non
// supportée) ; lecture en échec : EmpriseVehiclesLoadFailed, jamais un zéro.
func (l EmpriseLecteur) Vehicules(
	ctx context.Context, repo port.SquadVehicleRepository, ids []string, playerXUID, locale string,
) (*squademprise.VehicleRead, string, map[string]string) {
	if repo == nil {
		slog.DebugContext(ctx, "emprise_vehicules_capability_absente",
			"page", l.Page, "player", l.Player, "capability", string(games.CapFilmVehicleUsage),
			"err", games.ErrCapabilityNotSupported)
		return nil, "", nil
	}
	defer timing.FromContext(ctx).Section("emprise_vehicules")()
	read, err := repo.LoadVehicleUsage(ctx, ids, playerXUID)
	switch {
	case errors.Is(err, games.ErrCapabilityNotSupported):
		slog.DebugContext(ctx, "emprise_vehicules_capability_absente",
			"page", l.Page, "player", l.Player, "capability", string(games.CapFilmVehicleUsage), "err", err)
		return nil, "", nil
	case err != nil:
		slog.ErrorContext(ctx, "emprise_vehicules_en_echec",
			"page", l.Page, "player", l.Player, "matchs", len(ids), "err", err)
		return nil, domain.EmpriseVehiclesLoadFailed, nil
	}
	slog.DebugContext(ctx, "emprise_vehicules",
		"page", l.Page, "player", l.Player, "matchs", len(ids), "passes", len(read.Passes),
		"prises", len(read.Rows), "matchs_avec_frags", len(read.EventsRead))
	return &read, "", VehicleFamilyLabels(ctx, l.RepoRoot, l.TitleSlug, locale)
}

// EmpriseMatchIDs — les identifiants d'une liste de matchs de l'Emprise, dans l'ordre.
func EmpriseMatchIDs(matches []squademprise.Match) []string {
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		out = append(out, m.MatchID)
	}
	return out
}
