// Package teammates — teammates_service_emprise.go : LE BLOC « EMPRISE » de la page Escouade
// (lot L4 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26).
//
// Orchestration seule : le calcul est pur (analysis/squademprise.Build). Deux sources, qui
// dégradent chacune seule :
//
//	film              le résumé d'usage et les niveaux de socle (SessionUsageRepository, câblé
//	                  sous `film.usage_summary`). Absent ou en échec : aucune grandeur du film,
//	                  et le bloc dit pourquoi (FilmUnavailable).
//	feuille de match  les frags aux armes spéciales (SquadEmpriseRepository, tous titres). C'est
//	                  tout ce qu'un titre sans film publie (D10).
//
// Périmètre : le même que les autres blocs d'usage (D2). L'habitude (D5) n'est calculée qu'avec
// des coéquipiers sélectionnés, sur l'historique de la composition.
package teammates

import (
	"context"
	"errors"
	"log/slog"

	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/analysis/squademprise"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/observability/timing"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/squadagg"
)

// WithEmprise injecte la feuille de match du bloc « Emprise ». Câblage inconditionnel : la
// colonne est écrite par tous les titres ; un repo qui rend games.ErrCapabilityNotSupported
// retire seulement les frags aux armes spéciales. Nil ⇒ idem, journalisé.
func (s *TeammatesService) WithEmprise(repo port.SquadEmpriseRepository) *TeammatesService {
	s.empriseRepo = repo
	return s
}

// perimetreLu — le périmètre D2 et les lectures communes du résumé d'usage déjà faites dessus
// (nil = non faites : l'Emprise les fait).
type perimetreLu struct {
	scope    []legacymatch.SynthesisMatchRow
	lectures *squadagg.LecturesUsage
}

// loadEmprise publie le bloc sur le périmètre D2. Nil quand le périmètre est vide.
func (s *TeammatesService) loadEmprise(
	ctx context.Context, playerXUID string, p porteeUsage, req domain.TeammatesQueryRequest, lu perimetreLu,
) *domain.SquadEmpriseBlock {
	scope := lu.scope
	if len(scope) == 0 || playerXUID == "" {
		return nil
	}
	defer timing.FromContext(ctx).Section("emprise")()
	current := empriseCurrent(scope, p.squadRows)
	var timeline []squademprise.Match
	if len(req.SelectedGamertags) > 0 {
		timeline = empriseTimeline(p.timelineRows)
	}
	in := squademprise.Input{
		PlayerXUID: playerXUID, Current: current, Timeline: timeline,
		Weapons:            squadagg.WeaponCatalog(ctx, s.repoRoot, s.titleSlug, req.Locale),
		SessionMatchCounts: compositionCounts(p.compositionSessions),
	}
	in.PowerKills, in.SheetUnavailable = s.lireFeuilleEmprise(ctx, teammatesMatchIDs(scope))
	in.Film, in.FilmUnavailable = s.lireFilmEmprise(ctx, current, timeline, lu.lectures)
	if in.Film != nil {
		in.Players = squadagg.SquadPlayers(playerXUID, s.gamertag, in.Film.Participants, req.SelectedGamertags)
	} else {
		in.Players = squadagg.SquadPlayers(playerXUID, s.gamertag, nil, nil)
	}
	if n := squademprise.WithoutTimeScale(&in); n > 0 {
		// Même règle et même trace que Sessions : exclus du rendement des bonus, jamais en silence.
		slog.WarnContext(ctx, "teammates_emprise_matchs_sans_echelle_de_temps_hors_rendement",
			"player", s.gamertag, "matchs_sans_echelle", n, "matchs", len(current))
	}
	block := squademprise.Build(in)
	slog.DebugContext(ctx, "teammates_emprise",
		"player", s.gamertag, "matchs", block.MatchesTotal, "mesures", block.MatchesMeasured,
		"ressources", len(block.Resources), "objets", len(block.Objects),
		"film", block.FilmUnavailable, "feuille", block.SheetUnavailable, "habitude", block.Habit != nil)
	return &block
}

// lireFeuilleEmprise lit les frags aux armes spéciales du périmètre.
func (s *TeammatesService) lireFeuilleEmprise(ctx context.Context, ids []string) ([]squademprise.PowerKillRow, string) {
	if s.empriseRepo == nil {
		slog.WarnContext(ctx, "teammates_emprise_feuille_non_cablee", "player", s.gamertag)
		return nil, domain.EmpriseSheetUnsupported
	}
	rows, err := s.empriseRepo.LoadPowerWeaponKills(ctx, ids)
	switch {
	case errors.Is(err, games.ErrCapabilityNotSupported):
		slog.InfoContext(ctx, "teammates_emprise_feuille_non_supportee", "player", s.gamertag, "err", err)
		return nil, domain.EmpriseSheetUnsupported
	case err != nil:
		slog.ErrorContext(ctx, "teammates_emprise_feuille_en_echec",
			"player", s.gamertag, "matchs", len(ids), "err", err)
		return nil, domain.EmpriseSheetLoadFailed
	}
	return rows, ""
}

// lireFilmEmprise rend le résumé d'usage et les niveaux de socle du périmètre ET des matchs
// comparables des soirées précédentes (habitude). Les lectures du périmètre sont celles que les
// autres blocs d'usage ont déjà faites (partage, D2.6) ; seuls les matchs de l'habitude sont lus
// en plus.
func (s *TeammatesService) lireFilmEmprise(
	ctx context.Context, current, timeline []squademprise.Match, partage *squadagg.LecturesUsage,
) (*squademprise.FilmData, string) {
	if s.sessionUsageRepo == nil {
		// Titre sans `film.usage_summary` : la dégradation voulue (D10), pas une panne.
		slog.DebugContext(ctx, "teammates_emprise_sans_film", "player", s.gamertag)
		return nil, domain.EmpriseFilmUnsupported
	}
	ids := matchIDsOf(current)
	if partage == nil {
		partage = squadagg.LireUsage(ctx, s.sessionUsageRepo, ids)
	}
	if raison := s.raisonFilm(ctx, partage.Erreur(), len(ids)); raison != "" {
		return nil, raison
	}
	film := &squademprise.FilmData{Films: map[string]sessionusage.FilmRow{}, Players: partage.Players, Participants: partage.Participants}
	for id, f := range partage.Films {
		film.Films[id] = f
	}
	extras := squademprise.HabitCandidates(current, timeline)
	if len(extras) > 0 {
		lx := squadagg.LireUsage(ctx, s.sessionUsageRepo, extras)
		if err := lx.Erreur(); err != nil {
			// L'habitude dégrade SEULE : ses soirées précédentes restent sans point.
			slog.ErrorContext(ctx, "teammates_emprise_habitude_en_echec",
				"player", s.gamertag, "matchs", len(extras), "err", err)
		} else {
			for id, f := range lx.Films {
				film.Films[id] = f
			}
			film.Players = append(append([]sessionusage.PlayerRow(nil), film.Players...), lx.Players...)
			film.Participants = append(append([]sessionusage.ParticipantRow(nil), film.Participants...), lx.Participants...)
		}
	}
	ids = append(ids, extras...)
	tiers, err := s.sessionUsageRepo.LoadPadTiers(ctx, ids)
	if err != nil {
		// Les niveaux dégradent SEULS : bonus et feuille restent, armes spéciales et râteliers se
		// lisent « non mesurés ».
		slog.ErrorContext(ctx, "teammates_emprise_niveaux_en_echec",
			"player", s.gamertag, "matchs", len(ids), "err", err)
		return film, ""
	}
	film.PadTiers = tiers
	return film, ""
}

// raisonFilm traduit l'échec d'une lecture du résumé d'usage : capability non supportée (la
// dégradation voulue) ou panne (dite comme telle). "" = lecture réussie.
func (s *TeammatesService) raisonFilm(ctx context.Context, err error, n int) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, games.ErrCapabilityNotSupported):
		slog.InfoContext(ctx, "teammates_emprise_film_non_supporte", "player", s.gamertag, "err", err)
		return domain.EmpriseFilmUnsupported
	default:
		slog.ErrorContext(ctx, "teammates_emprise_film_en_echec", "player", s.gamertag, "matchs", n, "err", err)
		return domain.EmpriseFilmLoadFailed
	}
}

// empriseCurrent — le périmètre, avec la famille de mode de chaque match (même libellé que
// l'historique de l'escouade : squadModeUI).
func empriseCurrent(scope []legacymatch.SynthesisMatchRow, squadRows []domain.SquadMatchRow) []squademprise.Match {
	family := make(map[string]string, len(squadRows))
	for _, r := range squadRows {
		family[r.MatchID] = squadModeUI(r)
	}
	out := make([]squademprise.Match, 0, len(scope))
	for _, m := range scope {
		em := squademprise.Match{MatchID: m.MatchID, StartTime: m.StartTime, Family: family[m.MatchID]}
		if m.SessionLabel != nil {
			em.SessionLabel = *m.SessionLabel
		}
		out = append(out, em)
	}
	return out
}

// empriseTimeline — l'historique de la composition, avec la famille de mode de chaque match.
func empriseTimeline(rows []domain.SquadMatchRow) []squademprise.Match {
	out := make([]squademprise.Match, 0, len(rows))
	for _, r := range rows {
		m := squademprise.Match{MatchID: r.MatchID, StartTime: r.StartTime, Family: squadModeUI(r)}
		if r.SessionLabel != nil {
			m.SessionLabel = *r.SessionLabel
		}
		out = append(out, m)
	}
	return out
}

// compositionCounts — libellé de session -> matchs de la composition : composition_sessions,
// seule source d'un compte de session en contexte escouade (ADR 0033).
func compositionCounts(sessions []domain.CompositionSessionEntry) map[string]int {
	out := make(map[string]int, len(sessions))
	for _, s := range sessions {
		out[s.Label] = s.MatchCount
	}
	return out
}

// matchIDsOf — les identifiants d'une liste de matchs, dans l'ordre.
func matchIDsOf(matches []squademprise.Match) []string {
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		out = append(out, m.MatchID)
	}
	return out
}
