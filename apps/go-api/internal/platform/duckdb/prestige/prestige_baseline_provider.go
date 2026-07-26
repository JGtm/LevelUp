// Package duckdb — BaselineProvider pour le titre Halo Infinite.
//
// Implémente prestige.BaselineProvider en lisant les match_participants
// existants depuis shared_matches_v2.duckdb et la métrique demandée.

package prestige

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/prestige"
)

// Noms de colonnes match_participants utilisés par mapMetricToColumn.
// Centralisés pour éviter la duplication littérale (lint goconst).
const (
	metricColAccuracy = "accuracy"
	metricColKills    = "kills"
	metricColDeaths   = "deaths"
)

// HaloBaselineProvider lit les matchs récents d'UN joueur Halo Infinite
// pour fournir baseline + percentile au module Prestige.
//
// La complexité réelle (filtres mode/playlist, jointures avec medals) est
// gardée minimale pour Phase 4. Phase 5/6 affineront selon les besoins UI.
//
// reçoit un duckdb.SharedReader (pas un *duckdb.DB) pour coordonner
// avec le SharedDBProvider (cycle RO↔RW).
//
// ─── IDENTITÉ LIÉE À LA CONSTRUCTION (correctif 2026-07-26) ───
//
// Le provider est LIÉ à un joueur : son xuid est figé au moment où le composition
// root le construit, à partir de PlayerDB.XUID (déjà résolu par
// config.ResolvePlayer depuis db_profiles.json / DemoRoster). Aucune méthode ne
// prend d'identité en paramètre.
//
// C'est la correction structurelle du défaut qui rendait tout le module muet :
// `RecentMatches` prenait un `userID string` que ses trois appelants
// (computeBaselineFor, computeCurrentValue, evaluateOne) remplissaient avec
// `Challenge.UserID` — un **player_slug** applicatif ("JGtm", "Chocoboflor"),
// jamais un **xuid** Xbox ("2533274823110022"). Le `WHERE mp.xuid = ?` ne
// ramenait donc JAMAIS une ligne : baseline vide, jauges à 0, objectifs figés,
// silencieusement, en production. Deux identités distinctes voyageaient dans le
// même type `string` ; supprimer le paramètre supprime le transport, donc
// l'inversion — elle n'est plus exprimable à la compilation.
//
// Garde-rail : xuid_identity_ratchet_test.go (une requête sur `xuid` dans une
// fonction qui prend un paramètre d'identité applicative = échec CI).
type HaloBaselineProvider struct {
	reader duckdb.SharedReader
	// xuid — identité XBOX du joueur servi par ce provider. C'est la SEULE
	// identité que `match_participants.xuid` accepte. Jamais un player_slug.
	xuid string
}

// NewHaloBaselineProvider construit le provider pour UN joueur, depuis un
// duckdb.SharedReader et le xuid DÉJÀ RÉSOLU de ce joueur.
//
// Côté caller : passer pdb.SharedReadDB() (Provider B-swap, fallback transparent
// vers LegacySharedReader(pdb.Shared) si Provider absent) et pdb.XUID — les deux
// proviennent du MÊME PlayerDB, ce qui interdit de croiser le shared d'un joueur
// avec l'identité d'un autre. Ne PAS introduire ici une résolution slug→xuid
// maison : la résolution canonique est celle de config.ResolvePlayer (règle des
// ≤ 2 copies, CLAUDE.md n°6).
func NewHaloBaselineProvider(reader duckdb.SharedReader, playerXUID string) *HaloBaselineProvider {
	return &HaloBaselineProvider{reader: reader, xuid: playerXUID}
}

// Compile-time assertion.
var _ prestige.BaselineProvider = (*HaloBaselineProvider)(nil)

// RecentMatches retourne les N derniers matchs PvP du joueur LIÉ à ce provider,
// avec la métrique demandée. Aucun paramètre d'identité (cf. doc du type) : le
// premier paramètre string est le titleSlug, ignoré ici car la base partagée est
// déjà isolée PAR CHEMIN pour le titre (ADR 0008).
//
// Mapping des FieldKey canoniques vers les colonnes de match_participants.
// Les métriques cumulées (medal:*, maps_played_distinct, matches_played,
// win_rate, *_vs_expected…) n'ont pas d'équivalent colonne : mapMetricToColumn
// renvoie "" et cette méthode renvoie 0 match. Constat daté du 2026-07-26 : la
// « requête dédiée Phase 5 » que promettait l'ancien commentaire n'a jamais été
// écrite (EvaluateCumulative n'a AUCUN appelant de production, service_evaluate.go
// retombe sur EvaluateThreshold). Ces métriques restent donc non mesurées — c'est
// un manque FONCTIONNEL distinct du défaut d'identité corrigé ici, à traiter à
// part.
func (p *HaloBaselineProvider) RecentMatches(ctx context.Context, _, metric string, window int) ([]prestige.MatchData, error) {
	if p.xuid == "" {
		// Dégradation VISIBLE : sans xuid, aucun match n'est attribuable. C'est
		// exactement le silence qui a laissé le défaut d'identité survivre en prod
		// — on log AVANT de dégrader (CLAUDE.md n°3).
		slog.ErrorContext(ctx, "prestige_baseline_provider_missing_xuid",
			"metric", metric,
			"impact", "baseline vide, jauge à 0 et objectif figé pour ce joueur",
			"fix", "vérifier le xuid du joueur dans db_profiles.json (PlayerDB.XUID vide)")
		return nil, nil
	}
	col := mapMetricToColumn(metric)
	if col == "" {
		// Métrique non mappée : pas d'erreur, juste pas de matchs.
		// L'evaluator interprétera les []MatchData vides comme "pas de progrès".
		return nil, nil
	}
	if window <= 0 {
		window = 20
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Le littéral SQL reste INLINE ici (et non extrait dans un helper) : le
	// garde-rail xuid_identity_ratchet_test.go corrèle « paramètre d'identité
	// applicative » et « requête xuid » DANS UNE MÊME fonction — déplacer la
	// requête ailleurs le rendrait aveugle à une réintroduction du paramètre.
	//
	// CAST … AS DOUBLE : les colonnes métriques sont hétérogènes (kills INTEGER,
	// accuracy DOUBLE, headshot_kills SMALLINT) et MatchData.MetricValue est un
	// float64 — même normalisation que participantsWithMetric du provider
	// d'escouade. Timestamp : fragment canonique partagé (CLAUDE.md n°8), jamais
	// `start_time` brut, ni en projection ni en tri.
	startTime := duckdb.StartTimeCanonicalSQL("mr")
	q := fmt.Sprintf(`
		SELECT mp.match_id, %s, COALESCE(CAST(%s AS DOUBLE), 0)
		FROM match_participants mp
		JOIN match_registry mr ON mr.match_id = mp.match_id
		WHERE mp.xuid = ?
		ORDER BY %s DESC
		LIMIT ?
	`, startTime, col, startTime)

	db, release, err := p.reader.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("HaloBaselineProvider.RecentMatches: shared reader: %w", err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, q, p.xuid, window)
	if err != nil {
		return nil, fmt.Errorf("HaloBaselineProvider.RecentMatches: %w", err)
	}
	defer rows.Close()

	out, err := scanRecentMatches(rows)
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		// Légitime pour un joueur neuf, suspect sinon : tracer le couple
		// (xuid, métrique) rend un futur défaut d'attribution diagnosticable
		// sans instrumentation ad hoc.
		slog.DebugContext(ctx, "prestige_baseline_provider_no_matches",
			"xuid", p.xuid, "metric", metric, "column", col)
	}
	return out, nil
}

// scanRecentMatches matérialise les lignes de RecentMatches.
//
// StartedAt passe par un sql.NullTime : un match dont les DEUX colonnes de date
// sont NULL (registre incomplet) ne doit pas faire échouer la lecture ENTIÈRE —
// ce qui reviendrait à figer de nouveau tous les objectifs du joueur, cette
// fois pour une ligne isolée.
func scanRecentMatches(rows *sql.Rows) ([]prestige.MatchData, error) {
	var out []prestige.MatchData
	for rows.Next() {
		var (
			m       prestige.MatchData
			started sql.NullTime
		)
		if err := rows.Scan(&m.MatchID, &started, &m.MetricValue); err != nil {
			return nil, fmt.Errorf("HaloBaselineProvider.RecentMatches: scan: %w", err)
		}
		m.StartedAt = started.Time
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("HaloBaselineProvider.RecentMatches: rows: %w", err)
	}
	return out, nil
}

// PopulationPercentile renvoie la position approximative de la cible
// dans la population active sur la métrique.
//
// Phase 4 : implémentation simpliste qui retourne (0.5, 0). popSize=0
// désactive le cap population côté palier, ce qui est cohérent tant
// qu'on n'a pas un vrai calcul agrégé. Phase 5/6 : remplacer par une
// vraie agrégation (peut-être via un rollup périodique).
func (p *HaloBaselineProvider) PopulationPercentile(ctx context.Context, _ string, metric string, target float64) (float64, int, error) {
	col := mapMetricToColumn(metric)
	if col == "" {
		return 0.5, 0, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Approximation : compter combien de joueurs ont une moyenne ≥ target
	// sur la métrique demandée (sur les 30 derniers jours).
	q := fmt.Sprintf(`
		WITH player_avgs AS (
			SELECT mp.xuid, AVG(%s) AS avg_metric, COUNT(*) AS n
			FROM match_participants mp
			JOIN match_registry mr ON mr.match_id = mp.match_id
			WHERE mr.start_time > NOW() - INTERVAL 30 DAY
			GROUP BY mp.xuid
			HAVING COUNT(*) >= 5
		)
		SELECT
			COALESCE(SUM(CASE WHEN avg_metric < ? THEN 1 ELSE 0 END), 0) AS below,
			COUNT(*) AS total
		FROM player_avgs
	`, col)

	db, release, err := p.reader.Get(ctx)
	if err != nil {
		// Erreur non bloquante : retourner valeurs neutres (cap désactivé).
		return 0.5, 0, nil
	}
	defer release()

	var below, total int
	if err := db.QueryRowContext(ctx, q, target).Scan(&below, &total); err != nil {
		// Erreur non bloquante : retourner valeurs neutres (cap désactivé).
		return 0.5, 0, nil
	}
	if total == 0 {
		return 0.5, 0, nil
	}
	return float64(below) / float64(total), total, nil
}

// mapMetricToColumn convertit un FieldKey canonique en nom de colonne
// match_participants. Retourne "" si la métrique n'a pas d'équivalent
// direct (cas cumulative qui demandent une autre source).
func mapMetricToColumn(metric string) string {
	// Normalisation : accepte aussi les noms en lowercase
	m := strings.TrimSpace(metric)
	switch m {
	case "FieldKDA", "kda":
		return "kda"
	case "FieldKDR", "kd":
		return "kd"
	case "FieldAccuracy", metricColAccuracy:
		return metricColAccuracy
	case "FieldKills", metricColKills:
		return metricColKills
	case "FieldDeaths", metricColDeaths:
		return metricColDeaths
	case "FieldAssists", "assists":
		return "assists"
	case "FieldHeadshotKills", "headshot_kills":
		return "headshot_kills"
	case "FieldMeleeKills", "melee_kills":
		return "melee_kills"
	case "FieldGrenadeKills", "grenade_kills":
		return "grenade_kills"
	case "FieldPowerWeaponKills", "power_weapon_kills":
		return "power_weapon_kills"
	case "FieldDamageDealt", "damage_dealt":
		return "damage_dealt"
	case "FieldPersonalScore", "personal_score":
		return "personal_score"
	case "FieldMaxKillingSpree", "max_killing_spree":
		return "max_killing_spree"
	}
	return ""
}
