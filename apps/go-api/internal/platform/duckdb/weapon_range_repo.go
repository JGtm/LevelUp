// Package duckdb — weapon_range_repo.go : implémentation DuckDB de
// port.WeaponRangeRepository (plan .ai/PLAN_DUELS_PORTEE_2026-09-06.md, lot 3).
//
// # DEUX LECTURES, UNE SEULE JOINTURE
//
// « Où je frague » et « où je meurs » sont la MÊME mesure lue de deux côtés : la première
// filtre sur `e.feed_killer_xuid`, la seconde sur `e.victim_xuid`, et rien d'autre ne change.
// La jointure, ses gardes et le calcul de la distance vivent dans kill_measured.go
// (garde-rail : kill_measured_guard_test.go) ; ce fichier ne compose que ses clauses de
// portée et traduit le résultat en `analysis.MeasuredKill`.
//
// # LE SIGNE DU DÉNIVELÉ N'EST JAMAIS INVERSÉ ICI
//
// `MeasuredKill.DeltaZ` reçoit `killer_z - victim_z` TEL QUEL pour les deux lectures — la
// grandeur physique, sans point de vue. C'est `analysis.WeaponRangeAggregate` qui la ramène
// au point de vue du côté demandé (côté victime : l'opposé). L'inverser ici AUSSI
// l'annulerait, et le produit répondrait « d'en haut » quand la vérité est « d'en bas » :
// une erreur silencieuse, jamais détectable à l'écran. Convention tranchée au lot 2.
//
// # AUCUN FILTRE TEMPOREL (D10)
//
// La période est déjà résolue par le service, qui passe `MatchIDs`. Deux définitions de la
// période divergeraient ; celle de la Synthèse fait foi.
//
// # LA CLASSIFICATION EST CELLE DU TITRE, PAS UNE TABLE DE CORRESPONDANCE LOCALE
//
// `source_tag -> weapon_key` passe par le `port.KillSourceClassifier` injecté, comme
// KillDistanceRepo et KillSourceClassRepo. Un classificateur nil = ce titre n'expose pas la
// source de dégât : zéro ligne, sans erreur (état nominal, pas une panne). Une source HORS
// REGISTRE est écartée plutôt que devinée — et comptée dans le journal, pour qu'un trou de
// registre se voie.
package duckdb

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// weaponRangeQueryTimeout : l'agrégat de la Synthèse porte sur des centaines de matchs —
// même budget que les autres lectures agrégées du paquet (weapon_accuracy, weapon_kills).
const weaponRangeQueryTimeout = 30 * time.Second

// Colonnes de xuid des deux côtés de l'engagement, telles que le kill-feed les nomme.
const (
	weaponRangeKillerColumn = "e.feed_killer_xuid"
	weaponRangeVictimColumn = "e.victim_xuid"
)

var _ port.WeaponRangeRepository = (*WeaponRangeRepo)(nil)

// WeaponRangeRepo implémente port.WeaponRangeRepository.
type WeaponRangeRepo struct {
	pdb *PlayerDB
	// classifier traduit une source de dégât en clé de registre. nil = ce titre n'en
	// fournit pas -> zéro ligne (même doctrine que KillDistanceRepo).
	classifier port.KillSourceClassifier
}

// NewWeaponRangeRepo crée un WeaponRangeRepo lié à un PlayerDB.
//
// classifier peut être nil : voir l'en-tête du fichier.
func NewWeaponRangeRepo(pdb *PlayerDB, classifier port.KillSourceClassifier) *WeaponRangeRepo {
	return &WeaponRangeRepo{pdb: pdb, classifier: classifier}
}

// LoadWeaponRange rend les frags mesurés à l'instant du coup fatal, des deux côtés.
func (r *WeaponRangeRepo) LoadWeaponRange(
	ctx context.Context, slug string, filters port.WeaponRangeFilters,
) ([]analysis.MeasuredKill, error) {
	return r.loadBothSides(ctx, slug, filters, positionsAtKill, "LoadWeaponRange")
}

// LoadWeaponOpening rend les MÊMES frags un temps-pour-tuer plus tôt (proxy d'entame).
//
// La couverture est partielle tant que le backfill de `kill_openings` n'a pas tourné : un
// frag sans ligne d'entame est simplement absent du résultat. C'est à l'appelant de dire
// « N frags mesurés », jamais de présenter l'absence comme un zéro.
func (r *WeaponRangeRepo) LoadWeaponOpening(
	ctx context.Context, slug string, filters port.WeaponRangeFilters,
) ([]analysis.MeasuredKill, error) {
	return r.loadBothSides(ctx, slug, filters, positionsAtOpening, "LoadWeaponOpening")
}

// loadBothSides exécute les deux lectures (tueur puis victime) sous UN SEUL emprunt du
// lecteur partagé — le lease est la ressource la plus disputée du process (ADR 0013).
func (r *WeaponRangeRepo) loadBothSides(
	ctx context.Context, slug string, filters port.WeaponRangeFilters,
	table measuredPositionsTable, scope string,
) ([]analysis.MeasuredKill, error) {
	if err := filters.Validate(); err != nil {
		return nil, fmt.Errorf("WeaponRangeRepo.%s: %w", scope, err)
	}
	if r.classifier == nil {
		slog.DebugContext(ctx, "WeaponRangeRepo: no classifier for title, nothing to load",
			"scope", scope, "slug", slug)
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(ctx, weaponRangeQueryTimeout)
	defer cancel()

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "WeaponRangeRepo: shared reader unavailable",
			"scope", scope, "slug", slug, "err", err)
		return nil, fmt.Errorf("WeaponRangeRepo.%s: shared reader: %w", scope, err)
	}
	defer release()

	out := make([]analysis.MeasuredKill, 0)
	for _, side := range []struct {
		column string
		side   analysis.Side
	}{
		{weaponRangeKillerColumn, analysis.SideKiller},
		{weaponRangeVictimColumn, analysis.SideVictim},
	} {
		q, args := buildWeaponRangeQuery(table, side.column, filters)
		measured, err := queryMeasuredKills(ctx, db, q, args, scope+"/"+string(side.side))
		if err != nil {
			if isTableNotFoundErr(err) {
				slog.DebugContext(ctx, "WeaponRangeRepo: positions table missing",
					"scope", scope, "slug", slug, "table", string(table))
				return nil, games.ErrCapabilityNotSupported
			}
			slog.ErrorContext(ctx, "WeaponRangeRepo: query failed", "scope", scope,
				"slug", slug, "side", string(side.side), "err", err)
			return nil, fmt.Errorf("WeaponRangeRepo.%s: %w", scope, err)
		}
		out = append(out, r.toMeasuredKills(ctx, measured, side.side, scope)...)
	}
	return out, nil
}

// buildWeaponRangeQuery compose la clause de portée puis délègue la jointure à l'helper
// canonique. `column` est la colonne de xuid du côté lu.
func buildWeaponRangeQuery(
	table measuredPositionsTable, column string, f port.WeaponRangeFilters,
) (string, []any) {
	var sb strings.Builder
	args := make([]any, 0, len(f.MatchIDs)+len(f.XUIDs)+1)

	sb.WriteString("e.match_id IN (")
	sb.WriteString(Placeholders(len(f.MatchIDs)))
	sb.WriteString(")")
	for _, id := range f.MatchIDs {
		args = append(args, id)
	}

	// Projection sur WeaponKillFilters : le helper de filtre xuid est unique dans le
	// paquet (résolution gamertag -> xuid via xuid_aliases identique), on ne le recopie pas.
	appendXUIDFilter(&sb, &args, column, port.WeaponKillFilters{
		Gamertag: f.Gamertag,
		XUIDs:    f.XUIDs,
	})

	return measuredKillsQuery(table, sb.String()), args
}

// toMeasuredKills traduit `source_tag` -> `weapon_key` et habille chaque mesure de son côté.
//
// Une source hors registre est ÉCARTÉE (jamais devinée) ; leur nombre est journalisé, pour
// qu'un trou de registre soit visible autrement qu'en relisant le code.
func (r *WeaponRangeRepo) toMeasuredKills(
	ctx context.Context, measured []killMeasured, side analysis.Side, scope string,
) []analysis.MeasuredKill {
	out := make([]analysis.MeasuredKill, 0, len(measured))
	unclassified := 0
	for _, m := range measured {
		wk, ok := r.classifier.KillSourceRegistryKey(m.sourceTag)
		if !ok {
			unclassified++
			continue
		}
		out = append(out, analysis.MeasuredKill{
			MatchID:    m.matchID,
			KillerXUID: m.killerXUID,
			TimeMS:     m.timeMS,
			WeaponKey:  wk,
			Side:       side,
			DistanceM:  m.distanceM,
			// Dénivelé BRUT, jamais inversé ici (cf. en-tête du fichier).
			DeltaZ: m.deltaZ,
		})
	}
	slog.DebugContext(ctx, "WeaponRangeRepo: side read", "scope", scope, "side", string(side),
		"lignes_lues", len(measured), "retenues", len(out), "hors_registre", unclassified)
	return out
}
