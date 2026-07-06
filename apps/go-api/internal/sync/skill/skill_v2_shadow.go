package skill

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

	"levelup/go-api/internal/analysis"
	skillv2 "levelup/go-api/internal/analysis/skill_v2"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
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
	matchID      string
	startTime    time.Time
	pairName     string
	ownerOutcome int
	ownerTeamID  int
	ownerHasTeam bool
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
//
// Variante "persist des 2 équipes" (avance le watermark de TOUS les participants).
// N'est PLUS utilisée par le live (cf. RunLUSRV2ShadowOwnerOnly, fix 2026-06-08,
// couplage cross-joueur) — conservée pour cmd/lusr_v2_replay et les tests.
func RunLUSRV2Shadow(ctx context.Context, playerDB, sharedDB *sql.DB, xuid string) (int, error) {
	return runLUSRV2Shadow(ctx, playerDB, sharedDB, xuid, false)
}

// RunLUSRV2ShadowOwnerOnly : persiste UNIQUEMENT l'état v2 du joueur traité (pas
// celui de ses coéquipiers/adversaires) → n'avance que SON watermark + n'écrit que
// SA ligne canonique. C'est le chemin du LIVE post-sync ET du backfill recovery
// (fix 2026-06-08).
//
// Pourquoi : avec le persist des 2 équipes (RunLUSRV2Shadow), le sync d'un joueur
// avançait le watermark de ses coéquipiers trackés → ceux-ci sautaient leur match
// partagé à jamais, sans ligne LUSR (couplage cross-joueur). Owner-only rend chaque
// sync AUTONOME : un joueur obtient sa couverture canonique complète seul, sans
// dépendre d'un backfill ni du sync d'un autre. Au backfill on l'associe à un reset
// par-joueur (DELETE player_skill_state_v2 WHERE xuid=?) pour un replay propre.
//
// Compromis assumé : l'EP du joueur lit l'état COURANT de ses coéquipiers (pas
// forcément leur état pré-match) — approximation inhérente au modèle per-joueur,
// identique entre live et backfill (résultats cohérents).
func RunLUSRV2ShadowOwnerOnly(ctx context.Context, playerDB, sharedDB *sql.DB, xuid string) (int, error) {
	return runLUSRV2Shadow(ctx, playerDB, sharedDB, xuid, true)
}

func runLUSRV2Shadow(ctx context.Context, playerDB, sharedDB *sql.DB, xuid string, ownerOnlyPersist bool) (int, error) {
	if !IsLUSRV2Enabled() {
		return 0, nil
	}
	// Sprint 3.C : gate capability (pas de couplage slug). Titre sans CapLUSR →
	// no-op silencieux. CLI (context.Background) → "halo_infinite" → CapLUSR OK.
	if skipIfNoLUSRCapability(ctx, "RunLUSRV2Shadow") {
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

	// Découplage (Axe 2 refactor) : la logique en aval dépend des interfaces
	// port.* ; seul ce point d'instanciation connaît le type concret DuckDB.
	var repo port.SkillV2Repository = duckdb.NewSkillV2Repo(sharedDB)
	// Sprint 1.C : repo squad seulement si le flag est actif (OFF par défaut →
	// interface nil → aucun offset appliqué, comportement strictement inchangé).
	// Déclaré en type interface : assigner un *duckdb.SquadOffsetRepo nil à une
	// interface donnerait un typed-nil (interface NON nil) qui casserait le
	// garde `repo == nil` de computeTeamSquadOffsets.
	var squadRepo port.SquadOffsetRepository
	if IsLUSRV2SquadOffsetEnabled() {
		squadRepo = duckdb.NewSquadOffsetRepo(sharedDB)
	}
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
		repo:             repo,
		squadRepo:        squadRepo,
		playerDB:         playerDB,
		sharedDB:         sharedDB,
		priors:           priors,
		tierBoundaries:   tierBoundaries,
		xuid:             xuid,
		canonical:        canonical,
		ownerOnlyPersist: ownerOnlyPersist,
		priorsCache:      make(map[string]skillv2.Priors),
		countHypCache:    make(map[string]map[skillv2.CountType]skillv2.CountHyperparams),
	}
	var s shadowRunStats
	// heldGroups : groupes dont un match a échoué son écriture canonical ce run.
	// Les matchs plus RÉCENTS de ces groupes sont sautés pour ne pas avancer le
	// watermark de groupe par-dessus un match non écrit (anti-gap, fix 2026-06-07).
	heldGroups := make(map[string]bool)
	withGameplayDur := 0 // matchs ayant une durée gameplay → pondération temps-joué alimentée
	for _, m := range matches {
		if m.gameplayDurMs > 0 {
			withGameplayDur++
		}
		processOneShadowMatch(ctx, ctxRun, m, &s, heldGroups)
	}
	slog.InfoContext(ctx, "LUSR v2 shadow terminé",
		"xuid", xuid, "processed", s.processed,
		"skipped_chain", s.skippedChain,
		"skipped_already_seen", s.skippedAlready,
		"skipped_non_two_team", s.skippedNonTwoTeam,
		"skipped_imbalance", s.skippedImbalance,
		"skipped_write_failed", s.skippedWriteFailed,
		"skipped_group_held", s.skippedGroupHeld,
		"total_candidates", len(matches),
		// Observabilité TS2 : nb de matchs où wᵢ=time_played/durée_gameplay est
		// alimenté (durée gameplay connue). 0 → pondération inactive (fallback wᵢ=1).
		"with_gameplay_duration", withGameplayDur,
	)
	return s.processed, nil
}

// shadowRunContext regroupe les dépendances stables d'une exécution shadow,
// passées à `processOneShadowMatch` au lieu d'avoir 7 paramètres en cascade.
//
// `priors` reste les DEFAULTS ; les priors/count-hyperparams effectifs sont
// résolus PAR GROUPE via resolveGroupParams (override empirique du batch
// lusr_v2_ttt_batch) et mémoïsés dans priorsCache / countHypCache. Les maps
// sont des références — le cache survit aux copies par valeur du contexte.
type shadowRunContext struct {
	repo           port.SkillV2Repository
	squadRepo      port.SquadOffsetRepository // nil si LEVELUP_LUSR_V2_SQUAD_OFFSET off
	playerDB       *sql.DB
	sharedDB       *sql.DB
	priors         skillv2.Priors
	tierBoundaries []skillv2.TierBoundary
	xuid           string
	canonical      bool
	// ownerOnlyPersist : recovery backfill — ne persiste que l'état du joueur
	// traité (pas les coéquipiers). Voir RunLUSRV2ShadowOwnerOnly.
	ownerOnlyPersist bool
	priorsCache      map[string]skillv2.Priors
	countHypCache    map[string]map[skillv2.CountType]skillv2.CountHyperparams
}

// shadowRunStats compte les buckets de skip pour le log de fin de run.
type shadowRunStats struct {
	processed         int
	skippedChain      int
	skippedAlready    int
	skippedNonTwoTeam int
	skippedImbalance  int
	// skippedWriteFailed : matchs dont l'écriture canonical a échoué → watermark
	// VOLONTAIREMENT tenu (état v2 non persisté) pour retry au prochain cycle
	// (fix désync 2026-06-07). Distinct de skippedImbalance (skip pré-compute).
	skippedWriteFailed int
	// skippedGroupHeld : matchs sautés car un match plus ANCIEN du même groupe a
	// échoué son écriture canonical ce run (préserve l'avance EN ORDRE du watermark
	// de groupe → empêche le gap LUSR permanent multi-matchs, fix 2026-06-07).
	skippedGroupHeld int
}

// processOneShadowMatch applique le pipeline shadow à UN match :
// résolution chain → watermark → rosters → EP update → écriture canonical.
// Tout skip est compté dans `s` ; les erreurs non-bloquantes sont loggées
// mais n'arrêtent pas la boucle parente.
func processOneShadowMatch(ctx context.Context, c shadowRunContext, m shadowMatch, s *shadowRunStats, heldGroups map[string]bool) {
	// Title-aware (MT-15+) : la chaîne LUSR est résolue selon le titre porté par le
	// ctx. Halo 5 (pas de pair_name) a un classifier dédié ; ctx sans titre / titre
	// sans classifier dédié → classifier Infinite par défaut (comportement inchangé).
	group := GetLUSRChainForTitle(ctxkeys.TitleSlug(ctx), m.pairName)
	if group == "" {
		s.skippedChain++
		return
	}
	// Anti-gap multi-matchs (fix 2026-06-07) : si un match PLUS ANCIEN du même
	// groupe a échoué son écriture canonical ce run, on saute les matchs plus
	// récents du groupe. Sinon leur persist avancerait le watermark de groupe
	// PAR-DESSUS le match non écrit → celui-ci deviendrait skippedAlready à jamais
	// = gap LUSR permanent. Les matchs étant traités en ordre chrono ASC, tenir le
	// groupe garantit que le watermark n'avance jamais au-delà d'un match non écrit.
	if heldGroups[group] {
		s.skippedGroupHeld++
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
	// Skip les matchs réellement déséquilibrés en effectif CONCURRENT. On compte
	// les présents au coup d'envoi (present_at_beginning), pas len(team) : dans
	// Halo l'effectif par camp est constant à tout instant T, un quitter est
	// REMPLACÉ jamais ajouté. len() sur-compterait quitter + remplaçant comme 2
	// joueurs simultanés fictifs → ~32% des matchs (4v4 avec subs) étaient sautés
	// à tort (cf. concurrentTeamSize). Les remplaçants restent gérés par wᵢ
	// (temps-joué) dans l'EP ; une vraie asymétrie concurrente (rare) reste skip.
	if isTeamImbalanceTooHigh(concurrentTeamSize(teamA), concurrentTeamSize(teamB)) {
		s.skippedImbalance++
		return
	}
	outcomeA, ok := outcomeToTeamResult(m.ownerOutcome)
	if !ok {
		s.skippedNonTwoTeam++
		return
	}
	groupPriors, groupCountHyp := resolveGroupParams(ctx, c, group)
	// Sprint 2.A : contexte du quit (score au moment du quit). Chargé seulement
	// s'il y a un quitter ; sinon timeline vide → fallback outcome final.
	qt := quitTimeline{available: false}
	if hasAnyQuitter(teamA, teamB) {
		qt = loadQuitTimeline(ctx, c.sharedDB, m.matchID, m.startTime, teamA, teamB)
	}
	// Étape 1 — COMPUTE (pur, aucune écriture). Sépare le calcul EP de la
	// persistance pour pouvoir GATER l'avance du watermark sur le succès de
	// l'écriture canonical (cf. invariant durable-avant-progrès, durable_progress.go).
	computed, err := computeMatchToSkillV2(ctx, c.repo, c.squadRepo, groupPriors, groupCountHyp, qt, m.matchID, group,
		m.startTime, teamA, teamB, outcomeA, c.xuid, m.gameplayDurMs)
	if err != nil {
		slog.WarnContext(ctx, "LUSR v2 shadow: compute échoué",
			"match_id", m.matchID, "group", group, "err", err,
			"team_a_size", len(teamA), "team_b_size", len(teamB))
		return
	}

	// Cas dégénéré (mode canonical) : owner participant mais absent des rosters
	// 2-équipes (mismatch team_id) → aucune ligne LUSR attendue. PAS un échec
	// d'écriture durable : on NE tient PAS le groupe (un match plus récent pourra
	// avancer, le cas se résout seul). Anomalie data rendue VISIBLE.
	if c.canonical && computed.ownerNew == nil {
		canonicalOwnerMissing.Add(1)
		slog.ErrorContext(ctx, "LUSR v2 canonical: owner absent des rosters — aucune ligne LUSR écrite (désync data)",
			"match_id", m.matchID, "group", group, "xuid", c.xuid)
		return
	}

	// Étapes 2+3 — invariant durable-avant-progrès (durable_progress.go) :
	// WriteDurable = ligne canonical match_skill_rank (player DB ; no-op en
	// shadow-only) ; AdvanceProgress = persist état v2 (avance le watermark
	// player_skill_state_v2, shared). Sur échec durable : groupe tenu, watermark
	// non avancé (fix désync 2026-06-07, incident JGtm 03/06).
	outcome := CommitThenAdvance(ctx, heldGroups, DurableStep{
		GroupKey: group,
		WriteDurable: func(ctx context.Context) error {
			if !c.canonical {
				return nil // shadow-only : pas d'écriture canonical durable
			}
			return writeCanonicalLUSRRow(ctx, c.playerDB, m.matchID, *computed.ownerNew,
				computed.expectedWinProb, m.startTime, c.tierBoundaries)
		},
		AdvanceProgress: func(ctx context.Context) error {
			return persistComputedMatchSkillV2(ctx, c.repo, computed, m.matchID, group,
				m.startTime, c.xuid, c.ownerOnlyPersist)
		},
	})
	switch {
	case outcome.Advanced:
		s.processed++
	case outcome.Held && outcome.Err != nil:
		// Écriture canonical durable échouée → watermark tenu (groupe marqué par
		// le helper), retry au prochain cycle.
		canonicalWriteHeldWatermark.Add(1)
		s.skippedWriteFailed++
		slog.WarnContext(ctx, "LUSR v2 canonical: write match_skill_rank échoué — watermark tenu (groupe), retry au prochain cycle",
			"match_id", m.matchID, "group", group, "xuid", c.xuid, "err", outcome.Err)
	case outcome.Err != nil:
		// Avance du watermark échouée APRÈS écriture durable OK → retry idempotent.
		slog.WarnContext(ctx, "LUSR v2 shadow: persist état échoué — watermark non avancé",
			"match_id", m.matchID, "group", group, "err", outcome.Err)
	}
}

// resolveGroupParams retourne les Priors et CountHyperparams effectifs pour un
// groupe de modes : les défauts surchargés par les hyperparams empiriques
// ré-estimés par cmd/lusr_v2_ttt_batch (table lusr_hyperparams_v2). Mémoïsé par
// groupe — 1 seul LoadHyperparams par groupe et par run.
//
// Best-effort : si LoadHyperparams échoue (table absente sur une DB non migrée,
// etc.), on retombe sur les défauts + warn, et le run continue.
func resolveGroupParams(ctx context.Context, c shadowRunContext, group string) (skillv2.Priors, map[skillv2.CountType]skillv2.CountHyperparams) {
	if p, ok := c.priorsCache[group]; ok {
		return p, c.countHypCache[group]
	}
	priors := c.priors
	countHyp := skillv2.DefaultCountHyperparamsMap()
	hp, err := c.repo.LoadHyperparams(ctx, group)
	switch {
	case err != nil:
		slog.WarnContext(ctx, "LUSR v2: LoadHyperparams échoué — fallback defaults",
			"group", group, "err", err)
	case len(hp) > 0:
		priors = skillv2.LoadPriorsFromHyperparams(hp, c.priors)
		countHyp = skillv2.LoadCountHyperparamsFromDB(hp, c.priors.Mu0)
		slog.DebugContext(ctx, "LUSR v2: hyperparams ré-estimés appliqués",
			"group", group, "overrides", skillv2.AppliedHyperparamCount(hp),
			"draw_probability", priors.DrawProbability)
	}
	c.priorsCache[group] = priors
	c.countHypCache[group] = countHyp
	return priors, countHyp
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

// concurrentTeamSize retourne l'effectif RÉELLEMENT présent en même temps dans
// une équipe — pas le nombre total de xuid distincts ayant occupé un slot au
// cours du match.
//
// Dans Halo le nombre de joueurs par camp est constant à tout instant T : un
// quitter est REMPLACÉ, jamais ajouté. match_participants liste TOUS les humains
// ayant tenu un slot (quitter + remplaçant = 2 lignes pour 1 seul slot), donc
// len(team) sur-compte les remplacements comme des joueurs concurrents fictifs.
// present_at_beginning = TRUE identifie les occupants du coup d'envoi = le nombre
// de slots = la taille concurrente. Les remplaçants (present_at_beginning = FALSE)
// occupent un slot libéré ; leur poids dans l'EP est porté par wᵢ (temps-joué).
//
// Fallback : si AUCUN membre n'a present_at_beginning renseigné (match non
// backfillé), on retombe sur len(team) — comportement historique conservé. Le
// backfill participation_info étant atomique par match, on n'a jamais un mélange
// partiel renseigné/NULL au sein d'un même match.
func concurrentTeamSize(team []rosterMember) int {
	n := 0
	hasSignal := false
	for _, m := range team {
		if m.presentAtStart.Valid {
			hasSignal = true
			if m.presentAtStart.Bool {
				n++
			}
		}
	}
	if !hasSignal {
		return len(team)
	}
	return n
}

// loadShadowMatches : matchs LUSR-éligibles du joueur, ordre chrono. Mêmes
// filtres que loadLUSRMatchData (LUSR v1) pour cohérence.
func loadShadowMatches(ctx context.Context, sharedDB *sql.DB, xuid string) ([]shadowMatch, error) {
	rows, err := sharedDB.QueryContext(ctx, `
		SELECT mr.match_id,
		       `+analysis.SQLStartTimeCanonical("mr")+` AS ts,
		       COALESCE(mr.pair_name, ''),
		       COALESCE(mp.outcome, 0),
		       mp.team_id,
		       -- vraie durée gameplay en ms (countdown retranché via real_start_time)
		       GREATEST(0, COALESCE(mr.duration_seconds, 0) * 1000 - CASE
		           WHEN mr.real_start_time IS NOT NULL THEN
		               epoch_ms(mr.real_start_time AT TIME ZONE 'UTC')
		               - epoch_ms(`+analysis.SQLStartTimeCanonical("mr")+`)
		           ELSE 0 END) AS gameplay_dur_ms
		FROM match_registry mr
		JOIN match_participants mp ON mr.match_id = mp.match_id
		WHERE mp.xuid = ?
		  AND COALESCE(mr.is_ranked, FALSE) = FALSE
		  AND COALESCE(mr.is_firefight, FALSE) = FALSE
		  -- title-generic : les titres canoniques (Halo 5) renseignent start_time_utc
		  -- et laissent start_time (legacy) NULL ; filtrer sur le COALESCE comme le SELECT.
		  AND COALESCE(mr.start_time_utc, mr.start_time) IS NOT NULL
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
// 1 = Tie, 2 = Win, 3 = Loss, 4 = Did Not Finish.
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

// shadowMatchComputed porte le résultat du calcul EP d'un match AVANT toute
// persistance. Sépare le COMPUTE (pur, idempotent, aucune écriture) de la
// PERSIST, pour que processOneShadowMatch puisse gater l'avance du watermark
// (player_skill_state_v2) sur le succès de l'écriture canonical
// (match_skill_rank). Fix désync 2026-06-07 (cf. .ai/thought_log.md).
type shadowMatchComputed struct {
	teamAStates     []domain.SkillV2State
	teamBStates     []domain.SkillV2State
	newA            []skillv2.Gaussian
	newB            []skillv2.Gaussian
	ownerNew        *domain.SkillV2State // posterior du owner (nil = owner absent des rosters)
	ownerPrior      *domain.SkillV2State // prior du owner (pour le cross-mode leak)
	expectedWinProb *float64
}

// computeMatchToSkillV2 calcule l'EP update d'un match SANS rien persister.
// Logique équivalente à service.SkillV2Service.UpdateAfterMatch ; dupliquée
// ici parce que internal/sync ne peut pas importer internal/service (cycle).
//
// Pur : ne lit que les états pré-match (loadStatesOrSeed) et retourne les
// posteriors en mémoire. La persistance (qui avance le watermark) est faite
// séparément par persistComputedMatchSkillV2, APRÈS l'écriture canonical, pour
// éviter la désync watermark/ligne LUSR (cf. processOneShadowMatch).
func computeMatchToSkillV2(
	ctx context.Context,
	repo port.SkillV2Repository,
	squadRepo port.SquadOffsetRepository,
	priors skillv2.Priors,
	countHyp map[skillv2.CountType]skillv2.CountHyperparams,
	qt quitTimeline,
	matchID, playlistGroup string,
	startTime time.Time,
	teamA, teamB []rosterMember,
	outcomeA skillv2.TeamResult,
	ownerXUID string,
	gameplayDurMs int64,
) (shadowMatchComputed, error) {
	teamAXUIDs := extractXUIDs(teamA)
	teamBXUIDs := extractXUIDs(teamB)

	teamAStates, err := loadStatesOrSeed(ctx, repo, teamAXUIDs, playlistGroup, priors)
	if err != nil {
		return shadowMatchComputed{}, fmt.Errorf("loadStates teamA: %w", err)
	}
	teamBStates, err := loadStatesOrSeed(ctx, repo, teamBXUIDs, playlistGroup, priors)
	if err != nil {
		return shadowMatchComputed{}, fmt.Errorf("loadStates teamB: %w", err)
	}
	// Sprint 1.C : gaussiennes EFFECTIVES (μ + offset squad). squadRepo nil (flag
	// off) → offsets nuls → effA/effB == gaussiennes individuelles (no-op exact).
	offsetsA := computeTeamSquadOffsets(ctx, squadRepo, teamAStates, playlistGroup)
	offsetsB := computeTeamSquadOffsets(ctx, squadRepo, teamBStates, playlistGroup)
	effA := applyOffsetsToGaussians(shadowStatesToGaussians(teamAStates), offsetsA)
	effB := applyOffsetsToGaussians(shadowStatesToGaussians(teamBStates), offsetsB)

	// Sprint 1.A : proba de victoire pré-match de l'équipe du owner (teamA, par
	// construction de buildTwoTeamRosters). Sur les gaussiennes EFFECTIVES — la
	// synergie squad rend une victoire réellement plus probable. Calculée sur les
	// états pré-match en mémoire.
	probOwner, _, _ := skillv2.PredictTwoTeamWinProb(effA, effB, priors)
	predictionsTotal.Add(1)
	expectedWinProb := &probOwner

	// gameplayDurMs (origin Timeline T0) alimente le poids TS2 wᵢ via buildCountInputs.
	counts := buildCountInputs(ctx, teamA, teamB, outcomeA, qt, gameplayDurMs)
	if counts != nil {
		counts.Hyperparams = countHyp
	}
	newEffA, newEffB, err := skillv2.UpdateTwoTeamWithCountsEP(skillv2.TwoTeamMatch{
		TeamA:   effA,
		TeamB:   effB,
		ResultA: outcomeA,
	}, counts, priors)
	if err != nil {
		return shadowMatchComputed{}, fmt.Errorf("UpdateTwoTeamWithCountsEP: %w", err)
	}
	// Retire l'offset (constant) du posterior : seul le delta de l'EP s'applique
	// au μ INDIVIDUEL persisté. No-op quand offsets nuls.
	newA := stripOffsetsFromGaussians(newEffA, offsetsA)
	newB := stripOffsetsFromGaussians(newEffB, offsetsB)

	return shadowMatchComputed{
		teamAStates:     teamAStates,
		teamBStates:     teamBStates,
		newA:            newA,
		newB:            newB,
		ownerNew:        findOwnerPosterior(ownerXUID, teamAStates, newA, teamBStates, newB, matchID, playlistGroup, startTime),
		ownerPrior:      findOwnerPrior(ownerXUID, teamAStates, teamBStates),
		expectedWinProb: expectedWinProb,
	}, nil
}

// persistComputedMatchSkillV2 persiste les états posterior des 2 équipes
// (ce qui AVANCE le watermark via last_match_at) puis applique le cross-mode
// leak du owner. À n'appeler qu'APRÈS une écriture canonical réussie (ou en
// mode shadow-only) — c'est l'invariant du gate anti-désync 2026-06-07.
func persistComputedMatchSkillV2(
	ctx context.Context,
	repo port.SkillV2Repository,
	c shadowMatchComputed,
	matchID, playlistGroup string,
	startTime time.Time,
	ownerXUID string,
	ownerOnly bool,
) error {
	if ownerOnly {
		// Recovery (backfill) : persiste UNIQUEMENT l'état du joueur traité, pas
		// celui des coéquipiers/adversaires → aucun écrasement croisé. ownerNew
		// porte déjà LastMatchID/LastMatchAt (cf. findOwnerPosterior).
		if c.ownerNew != nil {
			if err := repo.UpsertState(ctx, *c.ownerNew); err != nil {
				return fmt.Errorf("persist owner-only: %w", err)
			}
		}
	} else {
		if err := persistTeamSkillV2(ctx, repo, c.teamAStates, c.newA, matchID, startTime); err != nil {
			return fmt.Errorf("persistTeam A: %w", err)
		}
		if err := persistTeamSkillV2(ctx, repo, c.teamBStates, c.newB, matchID, startTime); err != nil {
			return fmt.Errorf("persistTeam B: %w", err)
		}
	}

	// Phase 4 (mode correlation) : si activée, propage le delta du owner dans
	// son mode primaire vers tous ses autres modes joués. w_d capé à 0.4 par
	// la fonction skill_v2.ApplyCrossModeLeak.
	if IsLUSRV2ModeCouplingEnabled() && c.ownerNew != nil && c.ownerPrior != nil {
		if err := propagateCrossModeLeak(ctx, repo, *c.ownerPrior, *c.ownerNew); err != nil {
			slog.WarnContext(ctx, "Phase 4: cross-mode leak échoué",
				"xuid", ownerXUID, "primary_group", playlistGroup, "err", err)
		}
	}
	return nil
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
