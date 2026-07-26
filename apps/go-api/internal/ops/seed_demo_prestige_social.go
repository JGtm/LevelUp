// Package ops — seed_demo_prestige_social.go : volet shared_social des
// échantillons Prestige démo.
//
// Tables écrites (toutes en INSERT PUR — aucun UPSERT, cf. règle anti-ART) :
//   - prestige_events                     : le journal des gains de points
//   - user_prestige_history               : le total dérivé (vue user_prestige_latest)
//   - squad / squad_member_history        : l'escouade démo (3 joueurs)
//   - squad_challenge / ..._participant_history : un défi d'escouade en cours
//   - player_records_history              : les records personnels (vue _latest)
//
// Invariant central : user_prestige_history.total_pp == SUM(prestige_events.pp_amount)
// et current_level == prestige.LevelFromPP(total). Le total n'est jamais posé à la
// main : il est calculé par buildDemoEvents à partir du corpus démo réel.
package ops

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/prestige"

	_ "github.com/duckdb/duckdb-go/v2"
)

// demoSquadID / demoSquadChallengeID : identifiants stables entre reseeds.
const (
	demoSquadID          = "demo-squad-01"
	demoSquadChallengeID = "demo-squad-challenge-01"
	demoSquadName        = "Escouade de démonstration"
)

// demoSquadSize : nombre de membres de l'escouade démo — aligné sur le roster
// seedé par seedDemoPlayerDBs (DemoPlayer + 2 coéquipiers principaux).
const demoSquadSize = 3

// writeDemoPrestigeSocial écrit le volet shared_social du plan Prestige démo.
func writeDemoPrestigeSocial(ctx context.Context, socialDBPath string,
	plan demoPrestigePlan, counts map[string]int) error {
	db, err := sql.Open("duckdb", socialDBPath)
	if err != nil {
		return fmt.Errorf("open shared_social démo: %w", err)
	}
	defer db.Close()

	steps := []struct {
		table string
		fn    func(context.Context, *sql.DB, demoPrestigePlan) (int, error)
	}{
		{"prestige_events", insertDemoPrestigeEvents},
		{"user_prestige_history", insertDemoUserPrestige},
		{"squad", insertDemoSquad},
		{"squad_member_history", insertDemoSquadMembers},
		{"squad_challenge", insertDemoSquadChallenge},
		{"squad_challenge_participant_history", insertDemoSquadParticipants},
		{"player_records_history", insertDemoPlayerRecords},
	}
	for _, s := range steps {
		n, err := s.fn(ctx, db, plan)
		if err != nil {
			if errIsMissingTable(err) {
				slog.WarnContext(ctx, "seed-demo prestige: table sociale absente, ignorée",
					"table", s.table, "title_slug", plan.TitleSlug, "err", err)
				continue
			}
			return fmt.Errorf("%s: %w", s.table, err)
		}
		counts[s.table] = n
	}
	// ADR 0022 : sans CHECKPOINT le WAL de shared_social peut être perdu — la démo
	// redémarrerait alors avec une base sociale vide malgré un seed « réussi ».
	if _, err := db.ExecContext(ctx, "CHECKPOINT"); err != nil {
		return fmt.Errorf("checkpoint shared_social démo: %w", err)
	}
	return nil
}

func insertDemoPrestigeEvents(ctx context.Context, db *sql.DB, plan demoPrestigePlan) (int, error) {
	for _, ev := range plan.Events {
		if _, err := db.ExecContext(ctx, `
			INSERT INTO prestige_events (id, user_id, title_slug, source_type, source_id,
			                             pp_amount, tier, created_at)
			VALUES (?,?,?,?,?,?,?,?)`,
			ev.ID, plan.UserID, plan.TitleSlug, ev.SourceType, nullStr(ev.SourceID),
			ev.PP, nullStr(ev.Tier), ev.CreatedAt,
		); err != nil {
			return 0, err
		}
	}
	return len(plan.Events), nil
}

// insertDemoUserPrestige écrit UN snapshot du total (append-only : la vue
// user_prestige_latest sert la dernière ligne par (user_id, title_slug)).
// Horodaté sur plan.Anchor, jamais sur time.Now() : le générateur synthétique doit
// rester byte-identique d'un run à l'autre (TestSeedDemoSynthetic_Deterministic).
func insertDemoUserPrestige(ctx context.Context, db *sql.DB, plan demoPrestigePlan) (int, error) {
	if _, err := db.ExecContext(ctx, `
		INSERT INTO user_prestige_history (user_id, title_slug, total_pp, current_level,
		                                   updated_at, written_at)
		VALUES (?,?,?,?,?,?)`,
		plan.UserID, plan.TitleSlug, plan.TotalPP, plan.Level, plan.Anchor, plan.Anchor,
	); err != nil {
		return 0, err
	}
	return 1, nil
}

func insertDemoSquad(ctx context.Context, db *sql.DB, plan demoPrestigePlan) (int, error) {
	if _, err := db.ExecContext(ctx, `
		INSERT INTO squad (id, name, created_by, created_at) VALUES (?,?,?,?)`,
		demoSquadID, demoSquadName, plan.UserID, plan.Anchor,
	); err != nil {
		return 0, err
	}
	return 1, nil
}

// demoSquadSlug / demoSquadGamertag dérivent l'identité démo d'un membre depuis son
// index, avec la MÊME convention que buildDemoRoster et config.DemoRoster
// (index 0 = le joueur principal). Aucune identité réelle n'entre ici.
func demoSquadSlug(i int) string {
	if i == 0 {
		return DefaultDemoMainSlug
	}
	return fmt.Sprintf("%s-%d", DefaultDemoMainSlug, i+1)
}

func demoSquadGamertag(i int) string {
	if i == 0 {
		return DefaultDemoMainGamertag
	}
	return fmt.Sprintf("%s%d", DefaultDemoMainGamertag, i+1)
}

// insertDemoSquadMembers écrit le roster de l'escouade en append-only
// (squad_member_history → vue squad_member_latest). user_id = slug de route
// (c'est lui que ListSquadsForUser filtre), xuid = identifiant universel.
func insertDemoSquadMembers(ctx context.Context, db *sql.DB, plan demoPrestigePlan) (int, error) {
	at := plan.Anchor
	for i := 0; i < demoSquadSize; i++ {
		if _, err := db.ExecContext(ctx, `
			INSERT INTO squad_member_history (squad_id, xuid, user_id, gamertag,
			                                  is_member, joined_at, written_at)
			VALUES (?,?,?,?,TRUE,?,?)`,
			demoSquadID, demoXUIDForIndex(i), demoSquadSlug(i), demoSquadGamertag(i), at, at,
		); err != nil {
			return i, err
		}
	}
	return demoSquadSize, nil
}

// demoSquadTargetPerMember : cible par membre du défi d'escouade démo. Volontairement
// ronde et modeste (le défi doit être « en cours », pas déjà joué d'avance).
const demoSquadTargetPerMember = 40.0

// insertDemoSquadChallenge crée un défi d'escouade collaboratif en cours.
// template_id reste NULL : le catalogue de templates est un référentiel possédé
// par le titre par défaut (cf. NewPrestigeBundle) et un identifiant en dur y
// deviendrait faux au premier re-seed du catalogue. L'UI dégrade proprement en
// affichant la métrique et la cible plutôt qu'un libellé de template.
func insertDemoSquadChallenge(ctx context.Context, db *sql.DB, plan demoPrestigePlan) (int, error) {
	at := plan.Anchor
	if _, err := db.ExecContext(ctx, `
		INSERT INTO squad_challenge (id, squad_id, template_id, title_slug, mode, eval_type,
		                             window_type, window_value, target_per_member,
		                             expires_at, created_by, created_at)
		VALUES (?,?,NULL,?,?,?,?,?,?,?,?,?)`,
		demoSquadChallengeID, demoSquadID, plan.TitleSlug,
		string(prestige.SquadCollective), string(prestige.EvalCumulative),
		string(prestige.WindowLastNMatches), "20", demoSquadTargetPerMember,
		at.Add(7*24*time.Hour), plan.UserID, at,
	); err != nil {
		return 0, err
	}
	return 1, nil
}

// insertDemoSquadParticipants inscrit les 3 membres au défi d'escouade avec des
// avancements DIFFÉRENTS (un terminé, deux en cours) — la barre collective de la
// page Escouade montre alors une vraie répartition.
func insertDemoSquadParticipants(ctx context.Context, db *sql.DB, plan demoPrestigePlan) (int, error) {
	at := plan.Anchor
	progress := []float64{demoSquadTargetPerMember, 28, 17}
	for i := 0; i < demoSquadSize && i < len(progress); i++ {
		var completed any
		if progress[i] >= demoSquadTargetPerMember {
			completed = at
		}
		if _, err := db.ExecContext(ctx, `
			INSERT INTO squad_challenge_participant_history (
				squad_challenge_id, user_id, chosen_tier, data_tier, current_value,
				completed_at, is_private, joined_at, written_at
			) VALUES (?,?,?,?,?,?,FALSE,?,?)`,
			demoSquadChallengeID, demoSquadSlug(i), string(prestige.TierHeroic),
			string(prestige.DataFull), progress[i], completed, at, at,
		); err != nil {
			return i, err
		}
	}
	return demoSquadSize, nil
}

// insertDemoPlayerRecords écrit les PB en append-only (player_records_history →
// vue player_records_latest). La valeur COURANTE et la valeur PRÉCÉDENTE sont les
// mêmes que celles écrites dans record_history côté player DB : la timeline et le
// PB affiché ne peuvent pas diverger.
func insertDemoPlayerRecords(ctx context.Context, db *sql.DB, plan demoPrestigePlan) (int, error) {
	for _, r := range plan.Records {
		if _, err := db.ExecContext(ctx, `
			INSERT INTO player_records_history (
				xuid, metric, period, value, achieved_at, achieved_match_id,
				previous_value, previous_achieved_at, written_at
			) VALUES (?,?,?,?,?,NULL,?,?,?)`,
			plan.XUID, string(r.Metric), string(r.Period), r.Value, r.AchievedAt,
			r.PrevValue, r.PrevAchievedAt, r.AchievedAt,
		); err != nil {
			return 0, err
		}
	}
	return len(plan.Records), nil
}
