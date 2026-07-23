package prestige

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"
)

// defaultSquadWindowMatches borne le nombre de matchs candidats évalués quand la
// fenêtre n'est pas un last_n_matches explicite (V1).
const defaultSquadWindowMatches = 50

// service_squads.go — CRUD du roster d'escouade (entité Squad / SquadMember).
//
// Le roster est clé xuid (cf. SquadMember, PLAN_COACH_V3_GENERATION § Identité
// d'escouade). Les writes passent par SquadRepo (base sociale partagée). Accès
// « membre-user, sans consentement » : toute mutation exige que requestedBy
// (player_slug) soit déjà membre-user de l'escouade — sauf CreateSquad, où le
// créateur fonde l'escouade.

// CreateSquad crée une escouade et y inscrit ses membres initiaux.
//
// req.Members doit inclure le créateur (avec son xuid + son user_id slug) : la
// résolution xuid/slug est la responsabilité de l'appelant (handler), car le
// package prestige ne connaît pas db_profiles.
func (s *service) CreateSquad(ctx context.Context, req CreateSquadRequest) (Squad, error) {
	if req.Name == "" || req.CreatedBy == "" {
		return Squad{}, fmt.Errorf("%w: name/created_by requis", ErrInvalidInput)
	}
	now := s.deps.Now()
	sq := Squad{
		ID:        newID("sq"),
		Name:      req.Name,
		CreatedBy: req.CreatedBy,
		CreatedAt: now,
	}
	if err := s.deps.Squads.Create(ctx, sq); err != nil {
		return Squad{}, fmt.Errorf("create squad: %w", err)
	}
	for _, m := range req.Members {
		if m.Xuid == "" {
			continue // membre sans xuid ignoré (clé invalide)
		}
		m.SquadID = sq.ID
		if m.JoinedAt.IsZero() {
			m.JoinedAt = now
		}
		if err := s.deps.Squads.AddMember(ctx, m); err != nil {
			slog.WarnContext(ctx, "prestige: add squad member failed",
				"squad_id", sq.ID, "xuid", m.Xuid, "err", err)
		}
	}
	slog.InfoContext(ctx, "prestige: squad created",
		"squad_id", sq.ID, "name", sq.Name, "created_by", req.CreatedBy, "members", len(req.Members))
	return sq, nil
}

// ListSquadsForUser retourne les escouades dont userID (player_slug) est
// membre-user.
func (s *service) ListSquadsForUser(ctx context.Context, userID string) ([]Squad, error) {
	if userID == "" {
		return nil, fmt.Errorf("%w: user_id requis", ErrInvalidInput)
	}
	return s.deps.Squads.ListSquadsForUser(ctx, userID)
}

// GetSquad retourne une escouade par ID.
func (s *service) GetSquad(ctx context.Context, id string) (Squad, error) {
	if id == "" {
		return Squad{}, fmt.Errorf("%w: id requis", ErrInvalidInput)
	}
	sq, err := s.deps.Squads.Get(ctx, id)
	if err != nil {
		return Squad{}, fmt.Errorf("get squad: %w", err)
	}
	return sq, nil
}

// ListSquadMembers retourne le roster d'une escouade.
func (s *service) ListSquadMembers(ctx context.Context, squadID string) ([]SquadMember, error) {
	if squadID == "" {
		return nil, fmt.Errorf("%w: squad_id requis", ErrInvalidInput)
	}
	return s.deps.Squads.ListMembers(ctx, squadID)
}

// AddSquadMember ajoute un membre. requestedBy (player_slug) doit déjà être
// membre-user de l'escouade (règle « membre-user, sans consentement »).
func (s *service) AddSquadMember(ctx context.Context, squadID string, member SquadMember, requestedBy string) error {
	if squadID == "" || member.Xuid == "" {
		return fmt.Errorf("%w: squad_id/xuid requis", ErrInvalidInput)
	}
	if err := s.assertMemberUser(ctx, squadID, requestedBy); err != nil {
		return err
	}
	member.SquadID = squadID
	if member.JoinedAt.IsZero() {
		member.JoinedAt = s.deps.Now()
	}
	if err := s.deps.Squads.AddMember(ctx, member); err != nil {
		return err
	}
	slog.InfoContext(ctx, "prestige: squad member added",
		"squad_id", squadID, "xuid", member.Xuid, "by", requestedBy)
	return nil
}

// RemoveSquadMember retire un membre (par xuid). requestedBy doit être
// membre-user de l'escouade.
func (s *service) RemoveSquadMember(ctx context.Context, squadID, xuid, requestedBy string) error {
	if squadID == "" || xuid == "" {
		return fmt.Errorf("%w: squad_id/xuid requis", ErrInvalidInput)
	}
	if err := s.assertMemberUser(ctx, squadID, requestedBy); err != nil {
		return err
	}
	if err := s.deps.Squads.RemoveMember(ctx, squadID, xuid); err != nil {
		return err
	}
	slog.InfoContext(ctx, "prestige: squad member removed",
		"squad_id", squadID, "xuid", xuid, "by", requestedBy)
	return nil
}

// RenameSquad change le nom d'une escouade. requestedBy (player_slug) doit déjà
// être membre-user de l'escouade.
func (s *service) RenameSquad(ctx context.Context, squadID, name, requestedBy string) error {
	if squadID == "" || name == "" {
		return fmt.Errorf("%w: squad_id/name requis", ErrInvalidInput)
	}
	if err := s.assertMemberUser(ctx, squadID, requestedBy); err != nil {
		return err
	}
	if err := s.deps.Squads.Rename(ctx, squadID, name); err != nil {
		return err
	}
	slog.InfoContext(ctx, "prestige: squad renamed",
		"squad_id", squadID, "by", requestedBy)
	return nil
}

// DeleteSquad supprime une escouade en retirant tous ses membres (append-only :
// chaque retrait = event is_member=FALSE). L'escouade disparaît alors de
// ListSquadsForUser (member-driven) ; la ligne squad reste orpheline mais inerte
// (pas de DELETE indexé → pas de surface ART). requestedBy doit être membre-user.
func (s *service) DeleteSquad(ctx context.Context, squadID, requestedBy string) error {
	if squadID == "" {
		return fmt.Errorf("%w: squad_id requis", ErrInvalidInput)
	}
	if err := s.assertMemberUser(ctx, squadID, requestedBy); err != nil {
		return err
	}
	// Cascade (Lot 3) : archive les défis actifs de l'escouade avant de la
	// dissoudre — sinon ils resteraient orphelins (défis d'une escouade sans
	// membre). Best-effort : un échec d'archivage n'empêche pas la suppression
	// (idempotent, réessayable), mais est loggé (règle 3).
	if challenges, cErr := s.deps.SquadChallenges.ListBySquad(ctx, squadID); cErr != nil {
		slog.WarnContext(ctx, "prestige: delete squad — list challenges failed (cascade partielle)",
			"squad_id", squadID, "err", cErr)
	} else {
		for _, c := range challenges {
			if err := s.deps.SquadChallenges.Archive(ctx, c.ID); err != nil {
				slog.WarnContext(ctx, "prestige: delete squad — archive challenge failed",
					"squad_id", squadID, "squad_challenge_id", c.ID, "err", err)
			}
		}
	}
	members, err := s.deps.Squads.ListMembers(ctx, squadID)
	if err != nil {
		return fmt.Errorf("list members: %w", err)
	}
	var errs []error
	for _, m := range members {
		if m.Xuid == "" {
			continue
		}
		if err := s.deps.Squads.RemoveMember(ctx, squadID, m.Xuid); err != nil {
			errs = append(errs, fmt.Errorf("xuid %s: %w", m.Xuid, err))
			slog.WarnContext(ctx, "prestige: delete squad — remove member failed",
				"squad_id", squadID, "xuid", m.Xuid, "err", err)
		}
	}
	if len(errs) > 0 {
		// Retrait partiel : on remonte l'échec (→ 5xx côté handler) plutôt que de
		// répondre « supprimé » alors que l'escouade peut réapparaître au refetch.
		// Le retry est idempotent (RemoveMember = INSERT append-only is_member=FALSE).
		return fmt.Errorf("delete squad %s: %d/%d retraits échoués: %w",
			squadID, len(errs), len(members), errors.Join(errs...))
	}
	slog.InfoContext(ctx, "prestige: squad deleted (members removed)",
		"squad_id", squadID, "by", requestedBy, "members", len(members))
	return nil
}

// usualContextsWindow borne le nombre de matchs communs récents échantillonnés
// pour dériver les playlists/modes habituels d'une escouade.
const usualContextsWindow = 60

// SquadUsualContexts dérive les playlists/modes dominants des matchs communs du
// roster (indice d'affichage du sélecteur, jamais stocké). Best-effort : si le
// provider est absent, retourne des listes vides (pas d'erreur).
func (s *service) SquadUsualContexts(ctx context.Context, rosterXUIDs []string, titleSlug string) (playlists, modes []string, err error) {
	if len(rosterXUIDs) == 0 || s.deps.SquadMatches == nil {
		return nil, nil, nil
	}
	return s.deps.SquadMatches.SquadUsualContexts(ctx, rosterXUIDs, titleSlug, usualContextsWindow)
}

// assertMemberUser vérifie que userID (player_slug) est membre-user de
// l'escouade (squad_member.user_id renseigné = utilisateur de l'app).
func (s *service) assertMemberUser(ctx context.Context, squadID, userID string) error {
	if userID == "" {
		return fmt.Errorf("%w: requested_by requis", ErrInvalidInput)
	}
	members, err := s.deps.Squads.ListMembers(ctx, squadID)
	if err != nil {
		return fmt.Errorf("list members: %w", err)
	}
	if !isMember(members, userID) {
		return fmt.Errorf("%w: not a member of squad", ErrInvalidInput)
	}
	return nil
}

// EvaluateSquadChallenge recalcule la progression d'un défi d'escouade : résout
// la métrique (via le template), le roster, les coéquipiers connus (no-overlap),
// récupère les matchs comptants et agrège, puis persiste la progression des
// membres-users. requestedBy doit être membre-user de l'escouade.
func (s *service) EvaluateSquadChallenge(ctx context.Context, squadChallengeID, requestedBy string) ([]SquadParticipantProgress, error) {
	if squadChallengeID == "" {
		return nil, fmt.Errorf("%w: squad_challenge_id requis", ErrInvalidInput)
	}
	if s.deps.SquadMatches == nil {
		return nil, fmt.Errorf("%w: provider de matchs d'escouade indisponible", ErrInvalidInput)
	}
	sc, err := s.deps.SquadChallenges.Get(ctx, squadChallengeID)
	if err != nil {
		return nil, fmt.Errorf("get squad challenge: %w", err)
	}
	if err := s.assertMemberUser(ctx, sc.SquadID, requestedBy); err != nil {
		return nil, err
	}
	metric, err := s.squadChallengeMetric(ctx, sc)
	if err != nil {
		return nil, err
	}
	members, err := s.deps.Squads.ListMembers(ctx, sc.SquadID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	roster := rosterXUIDs(members)
	if len(roster) == 0 {
		return nil, fmt.Errorf("%w: escouade sans membre", ErrInvalidInput)
	}
	other := s.otherKnownTeammates(ctx, sc.SquadID, members, roster)
	limit := squadWindowLimit(sc.WindowType, sc.WindowValue)
	// Borne basse temporelle : jamais avant created_at (fin de la complétion
	// rétroactive, Lot 4) ; resserrée par la fenêtre pour rolling_days.
	since := squadEvalSince(sc, s.deps.Now())
	matches, err := s.deps.SquadMatches.SquadMatchMetrics(ctx, roster, sc.TitleSlug, metric, limit, since)
	if err != nil {
		return nil, fmt.Errorf("squad match metrics: %w", err)
	}
	progress := AggregateSquadProgress(roster, other, matches, sc.TargetPerMember)
	s.persistSquadProgress(ctx, squadChallengeID, members, progress)
	slog.InfoContext(ctx, "prestige: squad challenge evaluated",
		"squad_challenge_id", squadChallengeID, "roster", len(roster),
		"candidate_matches", len(matches), "members", len(progress), "since", since)
	return progress, nil
}

// squadEvalSince calcule la borne basse temporelle des matchs comptés pour un
// défi d'escouade (Lot 4) :
//   - JAMAIS avant created_at → un défi ne se complète pas rétroactivement avec
//     l'historique antérieur à sa création (bug observé : 181/243 vs cible 7) ;
//   - pour une fenêtre rolling_days, pas avant now - N jours (la plus récente
//     des deux bornes l'emporte).
//
// Les fenêtres session / last_n_matches sont bornées par created_at (le compteur
// de matchs `limit` gère déjà la profondeur) ; session reste une approximation
// (pas de découpage de session côté escouade) — documenté, à affiner si besoin.
func squadEvalSince(sc SquadChallenge, now time.Time) time.Time {
	since := sc.CreatedAt
	if sc.WindowType == WindowRollingDays {
		if n, err := strconv.Atoi(sc.WindowValue); err == nil && n > 0 {
			if cutoff := now.AddDate(0, 0, -n); cutoff.After(since) {
				since = cutoff
			}
		}
	}
	return since
}

// SquadOrientation retourne l'axe focal de l'escouade (le plus faible du profil
// de perf agrégé des membres). "" si aucun profil membre exploitable (→ pas
// d'orientation affichée). requestedBy doit être membre-user.
func (s *service) SquadOrientation(ctx context.Context, squadID, requestedBy string) (string, error) {
	if squadID == "" {
		return "", fmt.Errorf("%w: squad_id requis", ErrInvalidInput)
	}
	if err := s.assertMemberUser(ctx, squadID, requestedBy); err != nil {
		return "", err
	}
	members, err := s.deps.Squads.ListMembers(ctx, squadID)
	if err != nil {
		return "", fmt.Errorf("list members: %w", err)
	}
	axis := s.squadFocusAxis(ctx, members, "")
	slog.DebugContext(ctx, "prestige: squad orientation",
		"squad_id", squadID, "axis", axis, "members", len(members))
	return axis, nil
}

// squadChallengeMetric résout la métrique canonique d'un défi via son template.
func (s *service) squadChallengeMetric(ctx context.Context, sc SquadChallenge) (string, error) {
	if sc.TemplateID == "" {
		return "", fmt.Errorf("%w: défi d'escouade sans template (métrique indéterminée)", ErrInvalidInput)
	}
	tpl, err := s.deps.Templates.GetByID(ctx, sc.TemplateID)
	if err != nil {
		return "", fmt.Errorf("get template: %w", err)
	}
	if tpl.Metric == "" {
		return "", fmt.Errorf("%w: template sans métrique", ErrInvalidInput)
	}
	return tpl.Metric, nil
}

// otherKnownTeammates calcule les coéquipiers connus HORS roster : union des
// rosters de toutes les AUTRES escouades des membres-users de cette escouade
// (cf. règle no-overlap, PLAN_COACH_V3_GENERATION § Identité d'escouade).
func (s *service) otherKnownTeammates(ctx context.Context, squadID string, members []SquadMember, roster []string) map[string]struct{} {
	rosterSet := toXUIDSet(roster)
	seenSquad := map[string]bool{squadID: true}
	other := map[string]struct{}{}
	for _, m := range members {
		if m.UserID == "" {
			continue
		}
		squads, err := s.deps.Squads.ListSquadsForUser(ctx, m.UserID)
		if err != nil {
			continue
		}
		for _, sq := range squads {
			if seenSquad[sq.ID] {
				continue
			}
			seenSquad[sq.ID] = true
			mem, err := s.deps.Squads.ListMembers(ctx, sq.ID)
			if err != nil {
				continue
			}
			for _, mm := range mem {
				if _, in := rosterSet[mm.Xuid]; in {
					continue
				}
				other[mm.Xuid] = struct{}{}
			}
		}
	}
	return other
}

// persistSquadProgress écrit la progression des membres-users (ceux ayant un
// user_id, donc une ligne squad_challenge_participant). Les amis hors-app ont
// compté pour le no-overlap mais ne sont pas persistés.
func (s *service) persistSquadProgress(ctx context.Context, squadChallengeID string, members []SquadMember, progress []SquadParticipantProgress) {
	userIDByXUID := make(map[string]string, len(members))
	for _, m := range members {
		if m.UserID != "" {
			userIDByXUID[m.Xuid] = m.UserID
		}
	}
	for _, p := range progress {
		userID := userIDByXUID[p.Xuid]
		if userID == "" {
			continue
		}
		var completedAt *time.Time
		if p.Completed {
			now := s.deps.Now()
			completedAt = &now
		}
		if err := s.deps.SquadChallenges.UpdateParticipantProgress(ctx, squadChallengeID, userID, p.Value, completedAt); err != nil {
			slog.WarnContext(ctx, "prestige: update squad progress failed",
				"squad_challenge_id", squadChallengeID, "user_id", userID, "err", err)
		}
	}
}

// rosterXUIDs extrait les xuids (non vides) d'une liste de membres.
func rosterXUIDs(members []SquadMember) []string {
	out := make([]string, 0, len(members))
	for _, m := range members {
		if m.Xuid != "" {
			out = append(out, m.Xuid)
		}
	}
	return out
}

// squadWindowLimit borne le nombre de matchs candidats selon la fenêtre du défi.
func squadWindowLimit(wt WindowType, wv string) int {
	if wt == WindowLastNMatches {
		if n, err := strconv.Atoi(wv); err == nil && n > 0 {
			return n
		}
	}
	return defaultSquadWindowMatches
}
