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
//
// Les deux autres flags (`IsLUSRV2Canonical`, `IsLUSRV2ModeCouplingEnabled`)
// sont définis dans `skill_v2_canonical.go` / `skill_v2_cross_mode.go`
// respectivement, à côté du code qu'ils gardent.
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
	// gameplayDurMs : vraie durée de gameplay (countdown retranché) en ms,
	// dénominateur du poids TS2 wᵢ = time_played_i / match_length. 0 si inconnue.
	gameplayDurMs int64
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
// Si LEVELUP_LUSR_CANONICAL=LUSR_V2 et `playerDB` non nil, écrit aussi le
// résultat dans `match_skill_rank` du player DB (slot `rating_type='LUSR'`)
// via le mapping v2 → legacy. C'est la Stratégie C (write-through aliasing,
// ADR 0024).
//
// Si `playerDB` nil ou flag non canonical : v2 reste en pur shadow (écrit
// uniquement dans `player_skill_state_v2_latest` côté shared).
//
// Limites Phase 1 : ne traite que les matchs avec exactement 2 teams humaines
// distinctes. Matchs FFA, 3+ teams, ou outcomes incohérents sont skippés.
func RunLUSRV2Shadow(ctx context.Context, playerDB, sharedDB *sql.DB, xuid string) (int, error) {
	if !IsLUSRV2Enabled() {
		return 0, nil
	}
	if sharedDB == nil {
		return 0, fmt.Errorf("RunLUSRV2Shadow: sharedDB nil")
	}
	canonical := IsLUSRV2Canonical()
	if canonical && playerDB == nil {
		// Demandé canonical mais pas de playerDB → on log mais on continue
		// en shadow pour ne pas perdre l'occasion de mettre à jour l'état v2.
		slog.WarnContext(ctx, "LUSR v2 canonical demandé mais playerDB nil — fallback shadow",
			"xuid", xuid)
		canonical = false
	}

	repo := duckdb.NewSkillV2Repo(sharedDB)
	priors := skillv2.DefaultPriors()
	tierBoundaries := skillv2.DefaultTierBoundaries() // Phase 5 batch écrira override

	matches, err := loadShadowMatches(ctx, sharedDB, xuid)
	if err != nil {
		return 0, fmt.Errorf("RunLUSRV2Shadow.loadShadowMatches: %w", err)
	}
	if len(matches) == 0 {
		return 0, nil
	}

	ctxRun := shadowRunContext{
		repo:           repo,
		playerDB:       playerDB,
		sharedDB:       sharedDB,
		priors:         priors,
		tierBoundaries: tierBoundaries,
		xuid:           xuid,
		canonical:      canonical,
	}
	var s shadowRunStats
	withGameplayDur := 0 // matchs ayant une durée gameplay → pondération temps-joué alimentée
	for _, m := range matches {
		if m.gameplayDurMs > 0 {
			withGameplayDur++
		}
		processOneShadowMatch(ctx, ctxRun, m, &s)
	}
	slog.InfoContext(ctx, "LUSR v2 shadow terminé",
		"xuid", xuid, "processed", s.processed,
		"skipped_chain", s.skippedChain,
		"skipped_already_seen", s.skippedAlready,
		"skipped_non_two_team", s.skippedNonTwoTeam,
		"skipped_imbalance", s.skippedImbalance,
		"total_candidates", len(matches),
		// Observabilité TS2 : nb de matchs où wᵢ=time_played/durée_gameplay est
		// alimenté (durée gameplay connue). 0 → pondération inactive (fallback wᵢ=1).
		"with_gameplay_duration", withGameplayDur,
	)
	return s.processed, nil
}

// shadowRunContext regroupe les dépendances stables d'une exécution shadow,
// passées à `processOneShadowMatch` au lieu d'avoir 7 paramètres en cascade.
type shadowRunContext struct {
	repo           *duckdb.SkillV2Repo
	playerDB       *sql.DB
	sharedDB       *sql.DB
	priors         skillv2.Priors
	tierBoundaries []skillv2.TierBoundary
	xuid           string
	canonical      bool
}

// shadowRunStats compte les buckets de skip pour le log de fin de run.
type shadowRunStats struct {
	processed         int
	skippedChain      int
	skippedAlready    int
	skippedNonTwoTeam int
	skippedImbalance  int
}

// processOneShadowMatch applique le pipeline shadow à UN match :
// résolution chain → watermark → rosters → EP update → écriture canonical.
// Tout skip est compté dans `s` ; les erreurs non-bloquantes sont loggées
// mais n'arrêtent pas la boucle parente.
func processOneShadowMatch(ctx context.Context, c shadowRunContext, m shadowMatch, s *shadowRunStats) {
	group := GetLUSRChain(m.pairName)
	if group == "" {
		s.skippedChain++
		return
	}
	// Watermark : un match dont start_time ≤ last_match_at du groupe a déjà
	// été traité pour ce joueur.
	st, err := c.repo.LoadState(ctx, c.xuid, group)
	if err != nil {
		slog.WarnContext(ctx, "LUSR v2 shadow: LoadState échoué",
			"xuid", c.xuid, "group", group, "err", err)
		return
	}
	if st != nil && st.LastMatchAt != nil && !m.startTime.After(*st.LastMatchAt) {
		s.skippedAlready++
		return
	}
	if !m.ownerHasTeam {
		s.skippedNonTwoTeam++
		return
	}
	teamA, teamB, ok := buildTwoTeamRosters(ctx, c.sharedDB, m.matchID, m.ownerTeamID)
	if !ok {
		s.skippedNonTwoTeam++
		return
	}
	// Phase 3d : skip les matchs très déséquilibrés (ratio teams > 2:1). EP
	// avec count observations converge mal au-delà.
	if isTeamImbalanceTooHigh(len(teamA), len(teamB)) {
		s.skippedImbalance++
		return
	}
	outcomeA, ok := outcomeToTeamResult(m.ownerOutcome)
	if !ok {
		s.skippedNonTwoTeam++
		return
	}
	ownerNew, err := applyMatchToSkillV2(ctx, c.repo, c.priors, m.matchID, group,
		m.startTime, teamA, teamB, outcomeA, c.xuid, m.gameplayDurMs)
	if err != nil {
		slog.WarnContext(ctx, "LUSR v2 shadow: apply échoué",
			"match_id", m.matchID, "group", group, "err", err,
			"team_a_size", len(teamA), "team_b_size", len(teamB))
		return
	}
	s.processed++

	// Stratégie C : si v2 est canonical, écrit aussi dans match_skill_rank
	// (rating_type='LUSR' slot historique). Best-effort — un échec d'écriture
	// ne re-process pas le match (watermark a déjà avancé via persistTeamSkillV2).
	if c.canonical && ownerNew != nil {
		if err := writeCanonicalLUSRRow(ctx, c.playerDB, m.matchID, *ownerNew, c.tierBoundaries); err != nil {
			slog.WarnContext(ctx, "LUSR v2 canonical: write match_skill_rank échoué",
				"match_id", m.matchID, "group", group, "err", err)
		}
	}
}

// isTeamImbalanceTooHigh retourne true si la différence absolue des tailles
// d'équipes dépasse 1 joueur. EP avec count observations converge mal au-delà
// — la formule expected_count = w_p · perf + w_o · avg(perf_opp) crée des
// contradictions quand les rosters sont asymétriques (l'avg côté petite équipe
// inclut moins de joueurs, le sum factor amplifie les écarts).
//
// Pour Phase 3d on skip proprement les matchs |nA - nB| > 1 (le watermark
// ne bouge pas, donc retry au prochain run — improbable de changer en pratique).
// Diff = 1 reste accepté (cas courant : 4v3 / 4v5 après quit/late-join).
//
// Cf. ADR 0024 limites — Phase 3e pourrait introduire du damping EP pour
// élargir la tolérance, ou un facteur tronqué propre pour mieux gérer les
// asymétries.
func isTeamImbalanceTooHigh(nA, nB int) bool {
	if nA == 0 || nB == 0 {
		return true
	}
	diff := nA - nB
	if diff < 0 {
		diff = -diff
	}
	return diff > 1
}

// loadShadowMatches : matchs LUSR-éligibles du joueur, ordre chrono. Mêmes
// filtres que loadLUSRMatchData (LUSR v1) pour cohérence.
func loadShadowMatches(ctx context.Context, sharedDB *sql.DB, xuid string) ([]shadowMatch, error) {
	rows, err := sharedDB.QueryContext(ctx, `
		SELECT mr.match_id,
		       COALESCE(mr.start_time_utc, mr.start_time AT TIME ZONE 'UTC') AS ts,
		       COALESCE(mr.pair_name, ''),
		       COALESCE(mp.outcome, 0),
		       mp.team_id,
		       -- vraie durée gameplay en ms (countdown retranché via real_start_time)
		       GREATEST(0, COALESCE(mr.duration_seconds, 0) * 1000 - CASE
		           WHEN mr.real_start_time IS NOT NULL THEN
		               epoch_ms(mr.real_start_time AT TIME ZONE 'UTC')
		               - epoch_ms(COALESCE(mr.start_time_utc, mr.start_time AT TIME ZONE 'UTC'))
		           ELSE 0 END) AS gameplay_dur_ms
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
		if err := rows.Scan(&m.matchID, &m.startTime, &m.pairName, &m.ownerOutcome, &teamID, &m.gameplayDurMs); err != nil {
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
// Phase 3-quit (TS2 §9) ajoute les booleans ParticipationInfo + timestamps
// pour détecter quitter / late-joiner et ORDONNER précisément les quitters.
type rosterMember struct {
	xuid           string
	kills          *float64 // nil si non disponible (avant Phase 3c ou pour bots)
	deaths         *float64
	presentAtStart sql.NullBool
	presentAtEnd   sql.NullBool
	leftInProgress sql.NullBool
	lastLeaveTime  sql.NullTime    // timestamp absolu (signal idéal) — backfillé via cmd
	timePlayedSecs sql.NullFloat64 // proxy fallback si lastLeaveTime absent
	outcome        int             // code Halo (1=Tie,2=Win,3=Loss,4=DNF)
}

// buildTwoTeamRosters retourne (équipe du joueur, équipe adverse) ou (_, _, false)
// si le match n'a pas exactement 2 teams humaines distinctes.
//
// Phase 3c : récupère aussi kills/deaths par joueur depuis match_participants
// pour pouvoir les passer en observations TS2 §8.
func buildTwoTeamRosters(ctx context.Context, sharedDB *sql.DB, matchID string, ownerTeamID int) (teamA, teamB []rosterMember, ok bool) {
	rows, err := sharedDB.QueryContext(ctx, `
		SELECT xuid, team_id, kills, deaths,
		       present_at_beginning, present_at_completion, left_in_progress,
		       last_leave_time, time_played_seconds,
		       COALESCE(outcome, 0)
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
		var kills, deaths, timePlayed sql.NullFloat64
		var pStart, pEnd, leftInProg sql.NullBool
		var lastLeave sql.NullTime
		var outcome int
		if err := rows.Scan(&xuid, &teamID, &kills, &deaths,
			&pStart, &pEnd, &leftInProg, &lastLeave, &timePlayed, &outcome); err != nil {
			continue
		}
		if !teamID.Valid {
			continue
		}
		tid := int(teamID.Int64)
		m := rosterMember{
			xuid:           xuid,
			presentAtStart: pStart,
			presentAtEnd:   pEnd,
			leftInProgress: leftInProg,
			lastLeaveTime:  lastLeave,
			timePlayedSecs: timePlayed,
			outcome:        outcome,
		}
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
// applyMatchToSkillV2 applique le match et persiste l'état v2. Retourne le
// nouvel état du `ownerXUID` (côté A ou B, peu importe) — utilisé par le caller
// pour écrire dans le slot canonical (match_skill_rank rating_type='LUSR')
// quand Stratégie C est active. Retourne nil pour ownerNew si le owner n'est
// pas dans la match (cas dégénéré).
func applyMatchToSkillV2(
	ctx context.Context,
	repo *duckdb.SkillV2Repo,
	priors skillv2.Priors,
	matchID, playlistGroup string,
	startTime time.Time,
	teamA, teamB []rosterMember,
	outcomeA skillv2.TeamResult,
	ownerXUID string,
	gameplayDurMs int64,
) (ownerNew *domain.SkillV2State, err error) {
	teamAXUIDs := extractXUIDs(teamA)
	teamBXUIDs := extractXUIDs(teamB)

	teamAStates, err := loadStatesOrSeed(ctx, repo, teamAXUIDs, playlistGroup, priors)
	if err != nil {
		return nil, fmt.Errorf("loadStates teamA: %w", err)
	}
	teamBStates, err := loadStatesOrSeed(ctx, repo, teamBXUIDs, playlistGroup, priors)
	if err != nil {
		return nil, fmt.Errorf("loadStates teamB: %w", err)
	}
	counts := buildCountInputs(teamA, teamB, outcomeA, gameplayDurMs)
	newA, newB, err := skillv2.UpdateTwoTeamWithCountsEP(skillv2.TwoTeamMatch{
		TeamA:   shadowStatesToGaussians(teamAStates),
		TeamB:   shadowStatesToGaussians(teamBStates),
		ResultA: outcomeA,
	}, counts, priors)
	if err != nil {
		return nil, fmt.Errorf("UpdateTwoTeamWithCountsEP: %w", err)
	}
	if err := persistTeamSkillV2(ctx, repo, teamAStates, newA, matchID, startTime); err != nil {
		return nil, fmt.Errorf("persistTeam A: %w", err)
	}
	if err := persistTeamSkillV2(ctx, repo, teamBStates, newB, matchID, startTime); err != nil {
		return nil, fmt.Errorf("persistTeam B: %w", err)
	}
	ownerNew = findOwnerPosterior(ownerXUID, teamAStates, newA, teamBStates, newB, matchID, playlistGroup, startTime)

	// Phase 4 (mode correlation) : si activée, propage le delta du owner dans
	// son mode primaire vers tous ses autres modes joués. w_d capé à 0.4 par
	// la fonction skill_v2.ApplyCrossModeLeak.
	if IsLUSRV2ModeCouplingEnabled() && ownerNew != nil {
		ownerOld := findOwnerPrior(ownerXUID, teamAStates, teamBStates)
		if ownerOld != nil {
			if err := propagateCrossModeLeak(ctx, repo, *ownerOld, *ownerNew); err != nil {
				slog.WarnContext(ctx, "Phase 4: cross-mode leak échoué",
					"xuid", ownerXUID, "primary_group", playlistGroup, "err", err)
			}
		}
	}
	return ownerNew, nil
}

// findOwnerPosterior cherche le owner dans les rosters et reconstruit son
// nouvel état à partir des posteriors. Retourne nil si owner pas trouvé
// (cas dégénéré : owner pas dans le match).
func findOwnerPosterior(ownerXUID string, priorA []domain.SkillV2State, postA []skillv2.Gaussian,
	priorB []domain.SkillV2State, postB []skillv2.Gaussian,
	matchID, playlistGroup string, startTime time.Time) *domain.SkillV2State {
	for i, p := range priorA {
		if p.XUID == ownerXUID {
			mid := matchID
			st := startTime
			return &domain.SkillV2State{
				XUID:          p.XUID,
				PlaylistGroup: playlistGroup,
				Mu:            postA[i].Mu,
				Sigma:         postA[i].Sigma,
				Experience:    p.Experience + 1,
				LastMatchID:   &mid,
				LastMatchAt:   &st,
			}
		}
	}
	for j, p := range priorB {
		if p.XUID == ownerXUID {
			mid := matchID
			st := startTime
			return &domain.SkillV2State{
				XUID:          p.XUID,
				PlaylistGroup: playlistGroup,
				Mu:            postB[j].Mu,
				Sigma:         postB[j].Sigma,
				Experience:    p.Experience + 1,
				LastMatchID:   &mid,
				LastMatchAt:   &st,
			}
		}
	}
	return nil
}

// extractXUIDs, loadStatesOrSeed, shadowStatesToGaussians, persistTeamSkillV2
// vivent dans skill_v2_helpers.go.
// buildCountInputs vit dans skill_v2_quit_penalty.go.
