package sync

// skill_v2_shadow.go — LUSR v2 shadow runner (Phase 1c du chantier).
//
// Tourne après le calcul LUSR v1 dans la pipeline post-sync, en parallèle :
// les écritures vont dans player_skill_state_v2 / lusr_hyperparams_v2 (tables
// dédiées). Aucune lecture par l'UI à ce stade — c'est uniquement de la
// production de données pour valider le modèle sur des cas réels (Phase 1d).
//
// Gating : LEVELUP_LUSR_V2_ENABLED=1. Off → no-op silencieux, 0 latence.

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	skillv2 "levelup/go-api/internal/analysis/skill_v2"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb"
)

// lusrV2EnvFlag est le nom de la variable d'environnement qui active le shadow.
// Lecture déférée (pas en init()) pour permettre l'override en test.
const lusrV2EnvFlag = "LEVELUP_LUSR_V2_ENABLED"

// IsLUSRV2Enabled retourne true si le shadow runner doit s'exécuter.
// "1", "true", "yes" (case-insensitive) activent ; tout le reste désactive.
func IsLUSRV2Enabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(lusrV2EnvFlag)))
	return v == "1" || v == "true" || v == "yes"
}

// shadowMatch est une vue dégroupée de match_participants utilisée par le runner.
type shadowMatch struct {
	matchID       string
	startTime     time.Time
	pairName      string
	ownerOutcome  int
	ownerTeamID   int
	ownerHasTeam  bool
}

// shadowParticipant : un participant brut, pour construire les rosters par équipe.
type shadowParticipant struct {
	xuid    string
	teamID  int
	hasTeam bool
}

// RunLUSRV2Shadow calcule les états LUSR v2 pour tous les matchs du joueur qui
// n'ont pas encore été vus (filtre incrémental sur last_match_at par groupe).
// Retourne le nombre de matchs traités. Best-effort : les erreurs sur un match
// n'arrêtent pas la boucle (loggées warn).
//
// Limites Phase 1 : ne traite que les matchs avec exactement 2 teams humaines
// distinctes. Matchs FFA, 3+ teams, ou outcomes incohérents sont skippés.
func RunLUSRV2Shadow(ctx context.Context, sharedDB *sql.DB, xuid string) (int, error) {
	if !IsLUSRV2Enabled() {
		return 0, nil
	}
	if sharedDB == nil {
		return 0, fmt.Errorf("RunLUSRV2Shadow: sharedDB nil")
	}

	repo := duckdb.NewSkillV2Repo(sharedDB)
	priors := skillv2.DefaultPriors()

	matches, err := loadShadowMatches(ctx, sharedDB, xuid)
	if err != nil {
		return 0, fmt.Errorf("RunLUSRV2Shadow.loadShadowMatches: %w", err)
	}
	if len(matches) == 0 {
		return 0, nil
	}

	processed := 0
	skippedNonTwoTeam := 0
	skippedAlready := 0
	skippedChain := 0
	for _, m := range matches {
		group := GetLUSRChain(m.pairName)
		if group == "" {
			skippedChain++
			continue
		}
		// Watermark : un match dont start_time ≤ last_match_at du groupe a déjà
		// été traité pour ce joueur.
		st, err := repo.LoadState(ctx, xuid, group)
		if err != nil {
			slog.WarnContext(ctx, "LUSR v2 shadow: LoadState échoué", "xuid", xuid, "group", group, "err", err)
			continue
		}
		if st != nil && st.LastMatchAt != nil && !m.startTime.After(*st.LastMatchAt) {
			skippedAlready++
			continue
		}
		if !m.ownerHasTeam {
			skippedNonTwoTeam++
			continue
		}
		teamA, teamB, ok := buildTwoTeamRosters(ctx, sharedDB, m.matchID, m.ownerTeamID)
		if !ok {
			skippedNonTwoTeam++
			continue
		}
		outcomeA, ok := outcomeToTeamResult(m.ownerOutcome)
		if !ok {
			skippedNonTwoTeam++
			continue
		}
		if err := applyMatchToSkillV2(ctx, repo, priors, m.matchID, group, m.startTime, teamA, teamB, outcomeA); err != nil {
			slog.WarnContext(ctx, "LUSR v2 shadow: apply échoué",
				"match_id", m.matchID, "group", group, "err", err,
				"team_a_size", len(teamA), "team_b_size", len(teamB))
			continue
		}
		processed++
	}
	slog.InfoContext(ctx, "LUSR v2 shadow terminé",
		"xuid", xuid, "processed", processed,
		"skipped_chain", skippedChain,
		"skipped_already_seen", skippedAlready,
		"skipped_non_two_team", skippedNonTwoTeam,
		"total_candidates", len(matches),
	)
	return processed, nil
}

// loadShadowMatches : matchs LUSR-éligibles du joueur, ordre chrono. Mêmes
// filtres que loadLUSRMatchData (LUSR v1) pour cohérence.
func loadShadowMatches(ctx context.Context, sharedDB *sql.DB, xuid string) ([]shadowMatch, error) {
	rows, err := sharedDB.QueryContext(ctx, `
		SELECT mr.match_id,
		       COALESCE(mr.start_time_utc, mr.start_time AT TIME ZONE 'UTC') AS ts,
		       COALESCE(mr.pair_name, ''),
		       COALESCE(mp.outcome, 0),
		       mp.team_id
		FROM match_registry mr
		JOIN match_participants mp ON mr.match_id = mp.match_id
		WHERE mp.xuid = ?
		  AND COALESCE(mr.is_ranked, FALSE) = FALSE
		  AND COALESCE(mr.is_firefight, FALSE) = FALSE
		  AND mr.start_time IS NOT NULL
		  AND (mr.duration_seconds IS NULL OR mr.duration_seconds >= 30)
		ORDER BY ts ASC`, xuid)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	var out []shadowMatch
	for rows.Next() {
		var m shadowMatch
		var teamID sql.NullInt64
		if err := rows.Scan(&m.matchID, &m.startTime, &m.pairName, &m.ownerOutcome, &teamID); err != nil {
			return nil, err
		}
		if teamID.Valid {
			m.ownerTeamID = int(teamID.Int64)
			m.ownerHasTeam = true
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// rosterMember regroupe un xuid + ses counts pour la pipeline shadow Phase 3c.
type rosterMember struct {
	xuid   string
	kills  *float64 // nil si non disponible (avant Phase 3c ou pour bots)
	deaths *float64
}

// buildTwoTeamRosters retourne (équipe du joueur, équipe adverse) ou (_, _, false)
// si le match n'a pas exactement 2 teams humaines distinctes.
//
// Phase 3c : récupère aussi kills/deaths par joueur depuis match_participants
// pour pouvoir les passer en observations TS2 §8.
func buildTwoTeamRosters(ctx context.Context, sharedDB *sql.DB, matchID string, ownerTeamID int) (teamA, teamB []rosterMember, ok bool) {
	rows, err := sharedDB.QueryContext(ctx, `
		SELECT xuid, team_id, kills, deaths
		FROM match_participants
		WHERE match_id = ?
		  AND xuid IS NOT NULL AND xuid != ''`, matchID)
	if err != nil {
		return nil, nil, false
	}
	defer rows.Close() //nolint:errcheck

	teamsByID := make(map[int][]rosterMember)
	for rows.Next() {
		var xuid string
		var teamID sql.NullInt64
		var kills, deaths sql.NullFloat64
		if err := rows.Scan(&xuid, &teamID, &kills, &deaths); err != nil {
			continue
		}
		if !teamID.Valid {
			continue
		}
		tid := int(teamID.Int64)
		m := rosterMember{xuid: xuid}
		if kills.Valid {
			v := kills.Float64
			m.kills = &v
		}
		if deaths.Valid {
			v := deaths.Float64
			m.deaths = &v
		}
		teamsByID[tid] = append(teamsByID[tid], m)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, false
	}
	// Limite Phase 1 : on n'accepte que matchs à exactement 2 teams.
	if len(teamsByID) != 2 {
		return nil, nil, false
	}
	a, ok := teamsByID[ownerTeamID]
	if !ok || len(a) == 0 {
		return nil, nil, false
	}
	for tid, members := range teamsByID {
		if tid == ownerTeamID {
			continue
		}
		if len(members) == 0 {
			return nil, nil, false
		}
		return a, members, true
	}
	return nil, nil, false
}

// outcomeToTeamResult convertit l'outcome Halo (codes match_participants.outcome)
// en TeamResult skill_v2. Codes Halo (cf. internal/games/halo_infinite) :
//   1 = Tie, 2 = Win, 3 = Loss, 4 = Did Not Finish.
// DNF → skipped (le quit penalty proper sera la Phase 3 TS2 §9).
func outcomeToTeamResult(o int) (skillv2.TeamResult, bool) {
	switch o {
	case 2:
		return skillv2.TeamWin, true
	case 1:
		return skillv2.TeamDraw, true
	case 3:
		return skillv2.TeamLoss, true
	default:
		return 0, false
	}
}

// applyMatchToSkillV2 applique un match à l'état de tous ses participants.
// Logique équivalente à service.SkillV2Service.UpdateAfterMatch ; dupliquée
// ici parce que internal/sync ne peut pas importer internal/service (cycle).
//
// La couche service.SkillV2Service est conservée pour les callers externes
// (cmd Phase 1d notamment, tests, futurs handlers HTTP). Quand le code-flow
// se stabilisera (Phase 2/3), on pourra déplacer service.SkillV2Service vers
// un package "shared" et supprimer ce double — ou inversement faire passer
// le call par un callback registered au boot.
func applyMatchToSkillV2(
	ctx context.Context,
	repo *duckdb.SkillV2Repo,
	priors skillv2.Priors,
	matchID, playlistGroup string,
	startTime time.Time,
	teamA, teamB []rosterMember,
	outcomeA skillv2.TeamResult,
) error {
	teamAXUIDs := extractXUIDs(teamA)
	teamBXUIDs := extractXUIDs(teamB)

	teamAStates, err := loadStatesOrSeed(ctx, repo, teamAXUIDs, playlistGroup, priors)
	if err != nil {
		return fmt.Errorf("loadStates teamA: %w", err)
	}
	teamBStates, err := loadStatesOrSeed(ctx, repo, teamBXUIDs, playlistGroup, priors)
	if err != nil {
		return fmt.Errorf("loadStates teamB: %w", err)
	}
	counts := buildCountInputs(teamA, teamB)
	newA, newB, err := skillv2.UpdateTwoTeamWithCountsEP(skillv2.TwoTeamMatch{
		TeamA:   shadowStatesToGaussians(teamAStates),
		TeamB:   shadowStatesToGaussians(teamBStates),
		ResultA: outcomeA,
	}, counts, priors)
	if err != nil {
		return fmt.Errorf("UpdateTwoTeamWithCountsEP: %w", err)
	}
	if err := persistTeamSkillV2(ctx, repo, teamAStates, newA, matchID, startTime); err != nil {
		return fmt.Errorf("persistTeam A: %w", err)
	}
	if err := persistTeamSkillV2(ctx, repo, teamBStates, newB, matchID, startTime); err != nil {
		return fmt.Errorf("persistTeam B: %w", err)
	}
	return nil
}

func extractXUIDs(roster []rosterMember) []string {
	out := make([]string, len(roster))
	for i, m := range roster {
		out[i] = m.xuid
	}
	return out
}

// buildCountInputs construit la structure CountInputs depuis les rosters.
// Si AUCUN joueur n'a kills/deaths renseignés, retourne nil (le code en aval
// traitera comme TS classique sans counts) — typiquement le cas des anciens
// matchs antérieurs au sync Phase 3c.
func buildCountInputs(teamA, teamB []rosterMember) *skillv2.CountInputs {
	hasAny := false
	for _, m := range teamA {
		if m.kills != nil || m.deaths != nil {
			hasAny = true
			break
		}
	}
	if !hasAny {
		for _, m := range teamB {
			if m.kills != nil || m.deaths != nil {
				hasAny = true
				break
			}
		}
	}
	if !hasAny {
		return nil
	}
	pa := make([]skillv2.PlayerCounts, len(teamA))
	for i, m := range teamA {
		pa[i] = skillv2.PlayerCounts{Kills: m.kills, Deaths: m.deaths}
	}
	pb := make([]skillv2.PlayerCounts, len(teamB))
	for i, m := range teamB {
		pb[i] = skillv2.PlayerCounts{Kills: m.kills, Deaths: m.deaths}
	}
	return &skillv2.CountInputs{TeamA: pa, TeamB: pb}
}

func loadStatesOrSeed(ctx context.Context, repo *duckdb.SkillV2Repo, xuids []string, playlistGroup string, priors skillv2.Priors) ([]domain.SkillV2State, error) {
	out := make([]domain.SkillV2State, len(xuids))
	for i, x := range xuids {
		st, err := repo.LoadState(ctx, x, playlistGroup)
		if err != nil {
			return nil, err
		}
		if st == nil {
			seed := priors.NewPlayerState()
			out[i] = domain.SkillV2State{
				XUID: x, PlaylistGroup: playlistGroup,
				Mu: seed.Mu, Sigma: seed.Sigma,
			}
			continue
		}
		out[i] = *st
	}
	return out, nil
}

func shadowStatesToGaussians(states []domain.SkillV2State) []skillv2.Gaussian {
	out := make([]skillv2.Gaussian, len(states))
	for i, s := range states {
		out[i] = skillv2.Gaussian{Mu: s.Mu, Sigma: s.Sigma}
	}
	return out
}

func persistTeamSkillV2(ctx context.Context, repo *duckdb.SkillV2Repo, prior []domain.SkillV2State, posterior []skillv2.Gaussian, matchID string, startTime time.Time) error {
	if len(prior) != len(posterior) {
		return fmt.Errorf("persistTeamSkillV2: tailles incompatibles (prior=%d, posterior=%d)", len(prior), len(posterior))
	}
	mid := matchID
	st := startTime
	for i, p := range prior {
		next := domain.SkillV2State{
			XUID: p.XUID, PlaylistGroup: p.PlaylistGroup,
			Mu: posterior[i].Mu, Sigma: posterior[i].Sigma,
			Experience:  p.Experience + 1,
			LastMatchID: &mid,
			LastMatchAt: &st,
		}
		if err := repo.UpsertState(ctx, next); err != nil {
			return err
		}
	}
	return nil
}
