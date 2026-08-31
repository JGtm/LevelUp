// Package duckdb — kill_distance_repo.go : implémentation DuckDB du loader
// « distance par arme, par joueur » pour UN match (POC LOT G.3, 2026-08-30,
// plan .ai/PLAN_RETOURS_UTILISATEUR_2026-08-29.md §3bis DEC-8).
//
// Source : `kill_positions_latest` × `match_kill_events_latest` — TROISIÈME
// lecteur de la même famille que KillSourceClassRepo (kills-hors-arme) et
// Q21b/Q21c (kill feed du rejeu) : mêmes deux tables `_latest` (règle ART n°2,
// jamais la table brute), même classificateur injecté
// (port.KillSourceClassifier), même garde d'unanimité sur `source_tag` que
// Q21b (un double kill au même (tueur, instant) qui ne s'accorde pas sur
// l'arme ne publie RIEN plutôt que d'accrocher une position à la mauvaise
// arme).
//
// CE QUE CE REPO AJOUTE PAR RAPPORT À KillSourceClassRepo : celui-là résout
// UNIQUEMENT les sources HORS ARSENAL (anti-double-comptage avec
// weapon_kills, cf. son en-tête) — ce repo-ci veut TOUTES les armes (arme à
// feu du registre comme hors arsenal), parce qu'il ne recoupe jamais avec
// weapon_kills : chaque ligne qu'il émet porte déjà sa propre mesure de
// distance, il n'y a rien à additionner par-dessus une autre voie de comptage.
//
// PÉRIMÈTRE FERMÉ (cadrage utilisateur, DEC-8) : un seul match_id, jamais un
// agrégat multi-matchs ; le TUEUR seulement, jamais l'assistant (ni son arme,
// ni sa distance) ; aucune colonne de distance stockée — la distance (hypot 3D)
// se calcule ICI, à la lecture, jamais en base (doctrine G.0 : « on stocke une
// mesure, pas une résolution améliorable »).
package duckdb

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// KillDistanceRepo implémente port.KillDistanceRepository.
type KillDistanceRepo struct {
	pdb *PlayerDB
	// classifier traduit une source de dégât en clé de registre. nil = ce
	// titre n'en fournit pas -> le repo rend zéro ligne (état nominal, même
	// doctrine que KillSourceClassRepo).
	classifier port.KillSourceClassifier
}

// NewKillDistanceRepo crée un KillDistanceRepo lié à un PlayerDB.
//
// classifier peut être nil : voir l'en-tête du fichier.
func NewKillDistanceRepo(pdb *PlayerDB, classifier port.KillSourceClassifier) *KillDistanceRepo {
	return &KillDistanceRepo{pdb: pdb, classifier: classifier}
}

// killDistanceMeasured : une mort mesurée (position connue des deux côtés,
// arme non ambiguë), avant résolution du libellé de l'arme.
type killDistanceMeasured struct {
	xuid      string
	sourceTag uint32
	distanceM float64
}

// killDistanceAgg : compteur intermédiaire par (xuid, weapon_key).
type killDistanceAgg struct {
	kills int
	sum   float64
	min   float64
	max   float64
}

// LoadMatch charge les distances mesurées par (xuid, weapon_key) pour un match.
//
// Jamais de scan complet : matchID est obligatoire (une lecture sans filtre
// balaierait shared.kill_positions/match_kill_events en entier).
func (r *KillDistanceRepo) LoadMatch(ctx context.Context, matchID string) ([]domain.MatchKillDistancePlayer, error) {
	if matchID == "" {
		return nil, fmt.Errorf("KillDistanceRepo.LoadMatch: matchID vide")
	}
	if r.classifier == nil {
		slog.DebugContext(ctx, "KillDistanceRepo: no classifier for title, nothing to load",
			"match_id", matchID)
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	measured, err := r.queryMeasuredKills(ctx, matchID)
	if err != nil {
		if isTableNotFoundErr(err) {
			slog.DebugContext(ctx, "KillDistanceRepo: kill_positions_latest/match_kill_events_latest missing",
				"match_id", matchID)
			return nil, games.ErrCapabilityNotSupported
		}
		slog.ErrorContext(ctx, "KillDistanceRepo: query failed", "match_id", matchID, "err", err)
		return nil, fmt.Errorf("KillDistanceRepo.LoadMatch: %w", err)
	}
	if len(measured) == 0 {
		return nil, nil
	}
	return r.resolveRows(ctx, measured), nil
}

// killDistanceQuery : les morts mesurées d'un match — arme connue (source_tag)
// ET position connue des DEUX côtés (tueur ET victime).
//
// Même garde d'unanimité que Q21b (queries_match.go) : un double kill au même
// (tueur, instant) qui ne s'accorde pas sur l'arme ne publie RIEN — accrocher
// une position à la mauvaise arme serait indétectable à l'écran. `publishable`
// requis : cette lecture est PAR KILL (attribution nommée arme+distance), pas
// un agrégat qui tolère une passe non publiable ligne à ligne (cf. doctrine
// match_kill_events.publishable).
const killDistanceQuery = `
SELECT
    e.feed_killer_xuid,
    min(e.source_tag) AS source_tag,
    min(kp.killer_x) AS killer_x, min(kp.killer_y) AS killer_y, min(kp.killer_z) AS killer_z,
    min(kp.victim_x) AS victim_x, min(kp.victim_y) AS victim_y, min(kp.victim_z) AS victim_z
FROM match_kill_events_latest e
JOIN kill_positions_latest kp
    ON kp.match_id = e.match_id
   AND kp.killer_xuid = e.feed_killer_xuid
   AND kp.time_ms = e.time_ms
WHERE e.match_id = ?
  AND e.publishable
  AND e.source_tag IS NOT NULL
  AND e.feed_killer_xuid IS NOT NULL
  AND kp.killer_x IS NOT NULL AND kp.killer_y IS NOT NULL AND kp.killer_z IS NOT NULL
  AND kp.victim_x IS NOT NULL AND kp.victim_y IS NOT NULL AND kp.victim_z IS NOT NULL
GROUP BY e.feed_killer_xuid, e.time_ms
HAVING count(DISTINCT e.source_tag) = 1`

// queryMeasuredKills exécute killDistanceQuery et calcule la distance (hypot 3D)
// de chaque ligne EN GO — même politique que KillSourceClassRepo : le SQL ne
// connaît que des nombres, aucune traduction de sens n'y a lieu.
func (r *KillDistanceRepo) queryMeasuredKills(ctx context.Context, matchID string) ([]killDistanceMeasured, error) {
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("shared reader: %w", err)
	}
	defer release()

	dbRows, err := db.QueryContext(ctx, killDistanceQuery, matchID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer dbRows.Close()

	out := make([]killDistanceMeasured, 0)
	for dbRows.Next() {
		var (
			xuid                      string
			sourceTag                 uint32
			killerX, killerY, killerZ float64
			victimX, victimY, victimZ float64
		)
		if err := dbRows.Scan(&xuid, &sourceTag, &killerX, &killerY, &killerZ, &victimX, &victimY, &victimZ); err != nil {
			// Une ligne illisible est une anomalie de schéma, pas un cas nominal : on
			// la signale avant de dégrader, on ne l'avale pas en silence.
			slog.ErrorContext(ctx, "KillDistanceRepo: scan failed, row skipped", "match_id", matchID, "err", err)
			continue
		}
		out = append(out, killDistanceMeasured{
			xuid:      xuid,
			sourceTag: sourceTag,
			distanceM: hypot3D(killerX, killerY, killerZ, victimX, victimY, victimZ),
		})
	}
	return out, dbRows.Err()
}

// hypot3D : distance euclidienne entre deux points de l'espace monde (mètres).
func hypot3D(x1, y1, z1, x2, y2, z2 float64) float64 {
	dx, dy, dz := x1-x2, y1-y2, z1-z2
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

// resolveRows traduit source_tag -> weapon_key (classificateur), agrège par
// (xuid, weapon_key), puis habille le résultat de son libellé.
func (r *KillDistanceRepo) resolveRows(ctx context.Context, measured []killDistanceMeasured) []domain.MatchKillDistancePlayer {
	type key struct {
		xuid, weaponKey string
	}
	agg := map[key]*killDistanceAgg{}
	keysSeen := map[string]bool{}
	weaponKeys := make([]string, 0)

	for _, m := range measured {
		wk, ok := r.classifier.KillSourceRegistryKey(m.sourceTag)
		if !ok {
			continue // source hors registre : cette mort n'entre pas dans le POC (jamais de devinette)
		}
		if !keysSeen[wk] {
			keysSeen[wk] = true
			weaponKeys = append(weaponKeys, wk)
		}
		k := key{xuid: m.xuid, weaponKey: wk}
		a, exists := agg[k]
		if !exists {
			a = &killDistanceAgg{min: m.distanceM, max: m.distanceM}
			agg[k] = a
		}
		a.kills++
		a.sum += m.distanceM
		if m.distanceM < a.min {
			a.min = m.distanceM
		}
		if m.distanceM > a.max {
			a.max = m.distanceM
		}
	}
	if len(agg) == 0 {
		return nil
	}

	meta := resolveWeaponKeyLabelsAny(ctx, r.pdb.Metadata, r.pdb.TitleSlug, weaponKeys)

	byPlayer := map[string][]domain.MatchKillDistanceWeapon{}
	for k, a := range agg {
		m := meta[k.weaponKey]
		byPlayer[k.xuid] = append(byPlayer[k.xuid], domain.MatchKillDistanceWeapon{
			WeaponKey:     k.weaponKey,
			Label:         m.label,
			LabelEN:       m.labelEN,
			MeasuredKills: a.kills,
			AvgDistanceM:  a.sum / float64(a.kills),
			MinDistanceM:  a.min,
			MaxDistanceM:  a.max,
		})
	}

	out := make([]domain.MatchKillDistancePlayer, 0, len(byPlayer))
	for xuid, weapons := range byPlayer {
		sortKillDistanceWeapons(weapons)
		out = append(out, domain.MatchKillDistancePlayer{XUID: xuid, Weapons: weapons})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].XUID < out[j].XUID })
	return out
}

// sortKillDistanceWeapons rend la sortie DÉTERMINISTE (map d'agrégation non
// ordonnée) : kills mesurés décroissants, puis weapon_key — jamais ambigu.
func sortKillDistanceWeapons(weapons []domain.MatchKillDistanceWeapon) {
	sort.Slice(weapons, func(i, j int) bool {
		if weapons[i].MeasuredKills != weapons[j].MeasuredKills {
			return weapons[i].MeasuredKills > weapons[j].MeasuredKills
		}
		return weapons[i].WeaponKey < weapons[j].WeaponKey
	})
}
