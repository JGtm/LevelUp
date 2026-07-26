// Package ops — seed_demo_prestige.go : échantillons Prestige de la démo.
//
// La démo publique doit MONTRER l'axe Prestige (arcs, objectifs, points de
// progression, séries, records, jalons, escouade). Le corpus démo extrait des
// bases source ne contient rien de tout ça : aucune table Prestige ne figure
// dans playerTablesWhere, et le joueur source n'a pas d'activité Prestige
// représentative (arcs vides, aucun preset adopté). On GÉNÈRE donc ces
// échantillons, mais on les DÉRIVE du corpus déjà extrait (agrégats réels du
// joueur démo) pour que rien ne se contredise à l'écran.
//
// Découpage :
//   - seed_demo_prestige_plan.go   : le plan (données pures, zéro DB)
//   - seed_demo_prestige.go        : écritures player DB (arc, challenge, campagne,
//     moment, télémétrie, baseline, série, record, jalon)
//   - seed_demo_prestige_social.go : écritures shared_social (événements PP, total,
//     escouade, défi d'escouade, PB)
//
// Idempotence : la player DB démo est recréée à chaque run (extractPlayerTables)
// et le shared_social démo est reconstruit par rebuildDemoSharedSocial avant la
// phase média — les INSERT ci-dessous s'appliquent donc toujours sur des tables
// vides. Aucun UPSERT (règle anti-ART : streak_history / milestone_earned sont
// des tables protégées, cf. internal/sync/no_art_patterns_test.go).
package ops

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"levelup/go-api/internal/analysis"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/prestige"
	"levelup/go-api/internal/progression/streaks"

	_ "github.com/duckdb/duckdb-go/v2"
)

// demoMomentCardCount : nombre de cartes souvenir générées (une par objectif
// complété, plafonné) — assez pour peupler la section sans la saturer.
const demoMomentCardCount = 2

// demoChallengeTerminalColumns projette le statut terminal d'un objectif sur les
// TROIS colonnes distinctes de la table challenge (completed_at / expired_at /
// abandoned_at) : une seule est renseignée, les autres restent NULL.
func demoChallengeTerminalColumns(c demoPrestigeChallenge) (completed, expired, abandoned any) {
	if c.TerminalAt == nil {
		return nil, nil, nil
	}
	switch c.Status {
	case prestige.StatusCompleted:
		return *c.TerminalAt, nil, nil
	case prestige.StatusExpired:
		return nil, *c.TerminalAt, nil
	case prestige.StatusAbandoned:
		return nil, nil, *c.TerminalAt
	}
	return nil, nil, nil
}

// demoTelemetryKindFor mappe un statut terminal vers son type d'événement de
// télémétrie (source unique : prestige/constants.go).
func demoTelemetryKindFor(status prestige.ChallengeStatus) string {
	switch status {
	case prestige.StatusCompleted:
		return prestige.TelemetryCompleted
	case prestige.StatusExpired:
		return prestige.TelemetryExpired
	case prestige.StatusAbandoned:
		return prestige.TelemetryAbandoned
	}
	return prestige.TelemetryCommitted
}

// seedDemoPrestige génère les échantillons Prestige du joueur démo principal pour
// le titre courant. Best-effort par table : une table absente (titre additionnel
// dont le jeu de migrations ne la porte pas) est loguée et ignorée, jamais fatale.
// Retourne le nombre de lignes écrites par table.
//
// titleOut = racine de sortie du titre (plate pour le titre par défaut, cf.
// demoTitleSubdir). matchIDs vide = tout le corpus du joueur démo présent dans le
// shared de sortie (cas du générateur synthétique, qui n'a pas de liste externe).
func seedDemoPrestige(ctx context.Context, titleOut, titleSlug string,
	matchIDs []string) (map[string]int, error) {
	playerDBPath := filepath.Join(titleOut, "players", demoDirForIndex(0), "stats.duckdb")
	sharedDBPath := filepath.Join(titleOut, "warehouse", "shared_matches_v2.duckdb")
	socialDBPath := filepath.Join(titleOut, "warehouse", "shared_social.duckdb")
	metaDBPath := filepath.Join(titleOut, "warehouse", "metadata.duckdb")

	stats, matches, err := loadDemoCorpusStats(ctx, sharedDBPath, demoXUIDForIndex(0), matchIDs)
	if err != nil {
		return nil, fmt.Errorf("seed-demo prestige: agrégats corpus: %w", err)
	}
	if stats.Matches == 0 {
		slog.WarnContext(ctx, "seed-demo prestige: corpus vide pour le joueur démo, phase ignorée",
			"title_slug", titleSlug)
		return map[string]int{}, nil
	}

	plan := buildDemoPrestigePlan(titleSlug, stats, matches)

	counts := map[string]int{}
	if err := writeDemoPrestigePlayer(ctx, playerDBPath, metaDBPath, plan, counts); err != nil {
		return counts, fmt.Errorf("seed-demo prestige: player DB: %w", err)
	}
	if err := writeDemoPrestigeSocial(ctx, socialDBPath, plan, counts); err != nil {
		return counts, fmt.Errorf("seed-demo prestige: shared_social: %w", err)
	}
	slog.InfoContext(ctx, "seed-demo: échantillons Prestige écrits",
		"title_slug", titleSlug, "total_pp", plan.TotalPP, "level", plan.Level, "rows", counts)
	return counts, nil
}

// rebuildDemoSharedSocial (re)crée le shared_social démo vide et migré. Appelé
// AVANT la phase média (qui y insère des médias) et la phase Prestige : la base a
// UN seul propriétaire de cycle de vie, donc un reseed ne peut jamais empiler deux
// générations de lignes.
//
// Migrations forcées sur le jeu du titre PAR DÉFAUT (même choix, et même raison,
// que la phase média : la racine create_base_shared_social_schema est
// title-enregistrée et le schéma shared_social lui-même est title-agnostique —
// cf. applyTitleMigrationsOnPath). Les lignes écrites ensuite portent, elles, le
// title_slug réel du titre seedé.
func rebuildDemoSharedSocial(ctx context.Context, socialDBPath string) error {
	if err := os.MkdirAll(filepath.Dir(socialDBPath), 0o755); err != nil {
		return fmt.Errorf("mkdir shared_social démo: %w", err)
	}
	if err := removeDuckDBForFreshWrite(socialDBPath); err != nil {
		return err
	}
	if err := applyTitleMigrationsOnPath(socialDBPath, titlePkg.DefaultSlug, migration.TargetSharedSocial); err != nil {
		return fmt.Errorf("migrations shared_social démo: %w", err)
	}
	slog.InfoContext(ctx, "seed-demo: shared_social démo reconstruite", "path", socialDBPath)
	return nil
}

// loadDemoCorpusStats agrège le corpus démo (déjà extrait ET anonymisé) pour le
// xuid démo : totaux et fenêtre temporelle, plus la liste des matchs avec leur
// résultat (source des événements PP « match »).
//
// Tri temporel via le COALESCE canonique (règle CLAUDE.md n°8).
func loadDemoCorpusStats(ctx context.Context, sharedDBPath, demoXUID string,
	matchIDs []string) (demoCorpusStats, []demoMatchPP, error) {
	db, err := sql.Open("duckdb", sharedDBPath+"?access_mode=READ_ONLY")
	if err != nil {
		return demoCorpusStats{}, nil, fmt.Errorf("open shared démo: %w", err)
	}
	defer db.Close()

	corpusFilter := ""
	if len(matchIDs) > 0 {
		corpusFilter = fmt.Sprintf(" AND mp.match_id IN (%s)", formatIDsLiteral(matchIDs))
	}
	rows, err := db.QueryContext(ctx, fmt.Sprintf(`
		SELECT mp.match_id,
		       COALESCE(mp.kills, 0), COALESCE(mp.deaths, 0), COALESCE(mp.assists, 0),
		       COALESCE(mp.headshot_kills, 0), COALESCE(mp.outcome, 0),
		       %s
		FROM match_participants mp
		JOIN match_registry mr ON mr.match_id = mp.match_id
		WHERE mp.xuid = ?%s
		ORDER BY 7 ASC`, demoCanonicalStartExpr, corpusFilter), demoXUID)
	if err != nil {
		return demoCorpusStats{}, nil, fmt.Errorf("query corpus: %w", err)
	}
	defer rows.Close()
	return scanDemoCorpusStats(rows)
}

// demoCanonicalStartExpr : instant canonique d'un match (règle timezone CLAUDE.md n°8).
// DÉRIVÉ du helper partagé, jamais réécrit en littéral : le garde-rail
// archlint.TestNoNewRawStartTimeLiteral interdit toute copie de l'expression.
var demoCanonicalStartExpr = analysis.SQLStartTimeCanonical("mr")

// demoOutcomeWin : valeur `outcome` d'une victoire dans match_participants
// (1=Tie, 2=Win, 3=Loss, 4=DNF — cf. skill db-schema).
const demoOutcomeWin = 2

// scanDemoCorpusStats consomme le curseur du corpus et produit les agrégats.
func scanDemoCorpusStats(rows *sql.Rows) (demoCorpusStats, []demoMatchPP, error) {
	var st demoCorpusStats
	var out []demoMatchPP
	for rows.Next() {
		var id string
		var kills, deaths, assists, headshots, outcome int
		var at time.Time
		if err := rows.Scan(&id, &kills, &deaths, &assists, &headshots, &outcome, &at); err != nil {
			return st, nil, fmt.Errorf("scan corpus: %w", err)
		}
		st.Matches++
		st.Kills += kills
		st.Deaths += deaths
		st.Assists += assists
		st.HeadshotKills += headshots
		won := outcome == demoOutcomeWin
		if won {
			st.Wins++
		}
		if st.First.IsZero() || at.Before(st.First) {
			st.First = at
		}
		if at.After(st.Last) {
			st.Last = at
		}
		out = append(out, demoMatchPP{MatchID: id, Won: won, At: at})
	}
	return st, out, rows.Err()
}

// writeDemoPrestigePlayer écrit toutes les tables Prestige/Progression de la
// player DB démo. metaDBPath est ATTACHée pour dériver les jalons débloqués du
// VRAI catalogue du titre (aucun identifiant de jalon en dur).
func writeDemoPrestigePlayer(ctx context.Context, playerDBPath, metaDBPath string,
	plan demoPrestigePlan, counts map[string]int) error {
	db, err := sql.Open("duckdb", playerDBPath)
	if err != nil {
		return fmt.Errorf("open player démo: %w", err)
	}
	defer db.Close()

	steps := []struct {
		table string
		fn    func(context.Context, *sql.DB, demoPrestigePlan) (int, error)
	}{
		{"arc", insertDemoArcs},
		{"challenge", insertDemoChallenges},
		{"improvement_campaign", insertDemoCampaigns},
		{"moment_card", insertDemoMomentCards},
		{"prestige_telemetry", insertDemoTelemetry},
		{"baseline_state", insertDemoBaselineStates},
		{"streak_history", insertDemoStreaks},
		{"record_history", insertDemoRecordHistory},
	}
	for _, s := range steps {
		n, err := s.fn(ctx, db, plan)
		if err != nil {
			if errIsMissingTable(err) {
				slog.WarnContext(ctx, "seed-demo prestige: table absente de ce titre, ignorée",
					"table", s.table, "title_slug", plan.TitleSlug, "err", err)
				continue
			}
			return fmt.Errorf("%s: %w", s.table, err)
		}
		counts[s.table] = n
	}
	n, err := insertDemoMilestonesEarned(ctx, db, metaDBPath, plan)
	if err != nil {
		slog.WarnContext(ctx, "seed-demo prestige: jalons débloqués non seedés (best-effort)",
			"title_slug", plan.TitleSlug, "err", err)
	}
	counts["milestone_earned"] = n
	return nil
}

// nullTime convertit un *time.Time en argument SQL (NULL si nil). Pendant de
// nullStr (seed.go) pour les horodatages optionnels.
func nullTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}

func insertDemoArcs(ctx context.Context, db *sql.DB, plan demoPrestigePlan) (int, error) {
	for _, a := range plan.Arcs {
		if _, err := db.ExecContext(ctx, `
			INSERT INTO arc (id, user_id, title_slug, title, description,
			                 is_preset, preset_id, created_at, completed_at)
			VALUES (?,?,?,?,?,?,?,?,?)`,
			a.ID, plan.UserID, plan.TitleSlug, a.Title, a.Description,
			a.IsPreset, nullStr(a.PresetID), a.CreatedAt, nullTime(a.CompletedAt),
		); err != nil {
			return 0, err
		}
	}
	return len(plan.Arcs), nil
}

func insertDemoChallenges(ctx context.Context, db *sql.DB, plan demoPrestigePlan) (int, error) {
	for _, c := range plan.Challenges {
		completed, expired, abandoned := demoChallengeTerminalColumns(c)
		if _, err := db.ExecContext(ctx, `
			INSERT INTO challenge (
				id, user_id, title_slug, arc_id, position, metric, target,
				window_type, window_value, cadence, eval_type, mode, tier, data_tier,
				label, status, created_at, committed_at,
				completed_at, expired_at, abandoned_at, is_private, source
			) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			c.ID, plan.UserID, plan.TitleSlug, nullStr(c.ArcID), c.Position, c.Metric, c.Target,
			string(c.WindowType), c.WindowValue, string(c.Cadence), string(c.EvalType),
			string(prestige.ModeLibre), string(c.Tier), string(prestige.DataFull),
			c.Label, string(c.Status), c.CreatedAt, c.CreatedAt,
			completed, expired, abandoned, false, prestige.ChallengeSourceUser,
		); err != nil {
			return 0, err
		}
	}
	return len(plan.Challenges), nil
}

// insertDemoCampaigns écrit les campagnes d'amélioration (page Ascension → Profil).
//
// user_id = plan.XUID, PAS plan.UserID. C'est la seule entité Prestige clée par
// XUID : le handler (api/handlers/campaign.go) appelle GetActive/ListEnded avec
// `pdb.XUID`, et le repo filtre `WHERE user_id = ?`. Le plan sépare correctement
// les deux identités (cf. doc de demoPrestigePlan) et les autres surfaces clées
// XUID — séries, records, jalons — utilisaient déjà plan.XUID ; seul cet INSERT
// écrivait le player_slug, si bien que les 2 campagnes démo existaient en base
// et qu'AUCUNE n'était servie (2026-07-26).
func insertDemoCampaigns(ctx context.Context, db *sql.DB, plan demoPrestigePlan) (int, error) {
	if err := validateDemoCampaigns(plan.Campaigns); err != nil {
		return 0, err
	}
	for _, c := range plan.Campaigns {
		if _, err := db.ExecContext(ctx, `
			INSERT INTO improvement_campaign (
				id, user_id, title_slug, axis, axis_kind, started_at, ended_at, status,
				playlist_group, snapshot_value, snapshot_sample,
				current_value_raw, current_value_lowess, matches_since_start,
				last_evaluated_at, progression_confirmed
			) VALUES (?,?,?,?,?,?,?,?,'all',?,?,?,?,?,?,?)`,
			c.ID, plan.XUID, plan.TitleSlug, c.Axis, c.AxisKind, c.StartedAt,
			nullTime(c.EndedAt), string(c.Status), c.SnapshotValue, c.SnapshotN,
			c.CurrentValue, c.CurrentValue, c.MatchesSince, plan.Anchor, c.Confirmed,
		); err != nil {
			return 0, err
		}
	}
	return len(plan.Campaigns), nil
}

// insertDemoMomentCards attache une carte souvenir aux 2 premiers objectifs
// complétés (la page Réalisations montre alors des cartes réelles).
func insertDemoMomentCards(ctx context.Context, db *sql.DB, plan demoPrestigePlan) (int, error) {
	n := 0
	for _, c := range plan.Challenges {
		if n >= demoMomentCardCount || c.Status != prestige.StatusCompleted {
			continue
		}
		at := c.CreatedAt
		if c.TerminalAt != nil {
			at = *c.TerminalAt
		}
		if _, err := db.ExecContext(ctx, `
			INSERT INTO moment_card (id, challenge_id, blob_path, created_at)
			VALUES (?,?,NULL,?)`, "demo-moment-"+c.ID, c.ID, at); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

// demoTelemetryEvent : une ligne de prestige_telemetry à écrire pour un objectif.
type demoTelemetryEvent struct {
	suffix string
	kind   string
	at     time.Time
}

// insertDemoTelemetry trace un événement de création + l'événement terminal de
// chaque objectif — le diagnostic Prestige a ainsi de quoi afficher un entonnoir.
func insertDemoTelemetry(ctx context.Context, db *sql.DB, plan demoPrestigePlan) (int, error) {
	n := 0
	for _, c := range plan.Challenges {
		events := []demoTelemetryEvent{{suffix: "created", kind: prestige.TelemetryCreated, at: c.CreatedAt}}
		if c.TerminalAt != nil {
			events = append(events, demoTelemetryEvent{
				suffix: "terminal", kind: demoTelemetryKindFor(c.Status), at: *c.TerminalAt,
			})
		}
		for _, e := range events {
			if _, err := db.ExecContext(ctx, `
				INSERT INTO prestige_telemetry (
					id, user_id, challenge_id, event_type, palier, mode, cadence, eval_type, created_at
				) VALUES (?,?,?,?,?,?,?,?,?)`,
				c.ID+"-"+e.suffix, plan.UserID, c.ID, e.kind, string(c.Tier),
				string(prestige.ModeLibre), string(c.Cadence), string(c.EvalType), e.at,
			); err != nil {
				return n, err
			}
			n++
		}
	}
	return n, nil
}

// insertDemoBaselineStates marque les baselines des métriques suivies comme
// fraîches (sinon toute création d'objectif en démo serait refusée « baseline
// périmée »).
func insertDemoBaselineStates(ctx context.Context, db *sql.DB, plan demoPrestigePlan) (int, error) {
	seen := map[string]bool{}
	n := 0
	for _, c := range plan.Challenges {
		if seen[c.Metric] {
			continue
		}
		seen[c.Metric] = true
		if _, err := db.ExecContext(ctx, `
			INSERT INTO baseline_state (user_id, title_slug, metric, last_match_at,
			                            is_stale, recovery_matches_remaining, updated_at)
			VALUES (?,?,?,?,FALSE,0,?)`,
			plan.UserID, plan.TitleSlug, c.Metric, plan.Anchor, plan.Anchor,
		); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

// insertDemoStreaks écrit les séries en APPEND-ONLY (streak_history + vue
// streak_latest) — jamais dans la table d'état legacy `streak`.
func insertDemoStreaks(ctx context.Context, db *sql.DB, plan demoPrestigePlan) (int, error) {
	for _, s := range plan.Streaks {
		if _, err := db.ExecContext(ctx, `
			INSERT INTO streak_history (
				id, user_id, title_slug, type, started_at, current_length, best_length,
				last_increment_at, threshold, shields_used, shields_available,
				status, broken_at, written_at
			) VALUES (?,?,?,?,?,?,?,?,NULL,?,?,?,?,?)`,
			s.ID, plan.XUID, plan.TitleSlug, string(s.Type), s.StartedAt,
			s.Current, s.Best, s.LastAt, s.ShieldsUsed, streaks.MaxShieldsPerMonth,
			string(s.Status), nullTime(s.BrokenAt), s.LastAt,
		); err != nil {
			return 0, err
		}
	}
	return len(plan.Streaks), nil
}

// insertDemoRecordHistory écrit la timeline des records battus : pour chaque PB,
// la valeur PRÉCÉDENTE puis la valeur COURANTE (2 lignes) — la timeline raconte
// donc bien une progression, et sa dernière valeur par métrique coïncide avec le
// PB servi par player_records_latest (cf. writeDemoPrestigeSocial).
func insertDemoRecordHistory(ctx context.Context, db *sql.DB, plan demoPrestigePlan) (int, error) {
	n := 0
	for i, r := range plan.Records {
		entries := []struct {
			suffix string
			value  float64
			at     time.Time
		}{
			{"prev", r.PrevValue, r.PrevAchievedAt},
			{"cur", r.Value, r.AchievedAt},
		}
		for _, e := range entries {
			if _, err := db.ExecContext(ctx, `
				INSERT INTO record_history (id, user_id, title_slug, metric, period, value, achieved_at)
				VALUES (?,?,?,?,?,?,?)`,
				fmt.Sprintf("demo-rec-%02d-%s", i+1, e.suffix), plan.XUID, plan.TitleSlug,
				string(r.Metric), string(r.Period), e.value, e.at,
			); err != nil {
				return n, err
			}
			n++
		}
	}
	return n, nil
}

// insertDemoMilestonesEarned marque comme débloqués les N jalons les plus
// accessibles du VRAI catalogue du titre (metadata démo ATTACHée) — aucun
// identifiant en dur, et le reste du catalogue demeure verrouillé à l'écran.
func insertDemoMilestonesEarned(ctx context.Context, db *sql.DB, metaDBPath string,
	plan demoPrestigePlan) (int, error) {
	if _, err := db.ExecContext(ctx, fmt.Sprintf("ATTACH '%s' AS demometa (READ_ONLY)", metaDBPath)); err != nil {
		return 0, fmt.Errorf("attach metadata démo: %w", err)
	}
	defer func() { _, _ = db.ExecContext(ctx, "DETACH demometa") }()

	// CAST explicites + LIMIT littérale : DuckDB ne sait pas typer un paramètre nu
	// dans la projection d'un INSERT…SELECT, et n'accepte pas de placeholder en LIMIT.
	stmt := fmt.Sprintf(`
		INSERT INTO milestone_earned (user_id, title_slug, milestone_id, earned_at)
		SELECT CAST(? AS VARCHAR), CAST(? AS VARCHAR), id, CAST(? AS TIMESTAMP)
		FROM demometa.milestone_catalog
		WHERE title_slug = ?
		ORDER BY threshold ASC, id ASC
		LIMIT %d`, demoMilestoneCount)
	res, err := db.ExecContext(ctx, stmt,
		plan.XUID, plan.TitleSlug, plan.Anchor, plan.TitleSlug)
	if err != nil {
		return 0, fmt.Errorf("insert milestone_earned: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}
