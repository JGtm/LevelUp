// Package duckdb — vehicle_commendation_stats_repo.go : lecture SCOPÉE des
// compteurs « véhicules détruits » et « vol à la tire » (hijacks) d'un joueur,
// depuis les commendations NATIVES Halo 5 (shared.match_commendations) enrichies
// par le référentiel commendation_definitions (metadata h5).
//
// Contexte : sur Halo Infinite ces deux stats viennent de personal_score_awards
// (awards destroyed_*/hijacked_*). Halo 5 n'a PAS de personal_score_awards ; la
// donnée fiable = les commendations natives per-match (ProgressiveCommendationDeltas
// → match_commendations, cf. games/halo_5/mapping_commendations.go). Ce repo agrège
// donc les commendations « Destructeur de X » (9 véhicules) et « Vol à la tire »
// (Grand Theft) sur un scope de matchs.
//
// EXCLUSIONS (décision produit) : Écrasement/Splatter (splatter = kill, pas
// destruction), Maîtrise de véhicule / Vehicle Mastery (badge agrégé), Vandalisme
// (sémantique non confirmée) — la résolution par nom ne les capture pas (aucun ne
// matche le motif « Destructeur … » / « Grand Theft »), et deux garde-fous EN
// excluent explicitement un éventuel agrégat « Vehicle Destroyer/Mastery ».
//
// ART-safety : match_commendations est INSERT-only, keyée (match_id, xuid,
// commendation_id) via INSERT OR IGNORE côté persister (cf.
// persist/shared_persister.go persistCommendations + migration
// steps_shared_commendations.go). Le `count` par-match (Progress − PreviousProgress)
// est immuable — aucun doublon possible sur la clé naturelle, donc SUM(count) sur un
// scope de matchs est EXACT et n'exige PAS de vue `_latest` (parité
// citations_repo.matchCommendationsRichQuery ; la vue _latest ne concerne QUE les
// tables re-INSÉRÉES avec written_at, ADR 0026 — ce qui n'est pas le cas ici).
package duckdb

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"levelup/go-api/internal/port"
)

// VehicleCommendationStatsRepo implémente port.VehicleDestructionStatsRepository
// pour les titres à commendations natives (Halo 5). Câblé UNIQUEMENT quand le titre
// porte la capability commendations.native (cf. registry SynthesisCtx — jamais de
// gating par slug).
type VehicleCommendationStatsRepo struct {
	pdb *PlayerDB

	// Résolution des UUID par NOM, faite UNE FOIS (au premier appel) puis mémoïsée :
	// le référentiel commendation_definitions est immuable au runtime (seedé au boot
	// par cmd/h5-metadata-fetch, versionné). ~10 lignes retenues sur ~88 → scan LIKE
	// négligeable.
	resolveOnce sync.Once
	vehicleIDs  map[string]struct{}
	hijackIDs   map[string]struct{}
}

// NewVehicleCommendationStatsRepo lie le repo à un PlayerDB (porte le SharedReader
// pour match_commendations + le handle Metadata pour commendation_definitions).
func NewVehicleCommendationStatsRepo(pdb *PlayerDB) *VehicleCommendationStatsRepo {
	return &VehicleCommendationStatsRepo{pdb: pdb}
}

// vehicleCommendationResolveQuery résout les UUID des commendations pertinentes par
// NOM (locale-agnostic : matche name_en OU name_fr, robuste au seed FR/EN). Le motif
// FR « Destructeur % » (espace) capture aussi « Destructeur d'apparitions » (wraith) ;
// le motif EN « % Destroyer » capture « Banshee Destroyer », etc. « Vol à la tire » /
// « Grand Theft » = hijack. Les deux garde-fous NOT excluent un éventuel badge agrégé.
const vehicleCommendationResolveQuery = `
SELECT commendation_id,
       COALESCE(name_en, '') AS name_en,
       COALESCE(name_fr, '') AS name_fr
FROM commendation_definitions
WHERE (
        name_en LIKE '% Destroyer'
     OR name_fr LIKE 'Destructeur %'
     OR name_en = 'Grand Theft'
     OR name_fr = 'Vol à la tire'
      )
  AND name_en <> 'Vehicle Destroyer'
  AND name_en <> 'Vehicle Mastery'`

// resolve renseigne (une seule fois) les sets d'UUID véhicule/hijack depuis la
// metadata. Dégradation gracieuse : metadata nil, table absente, ou 0 UUID résolu →
// sets vides + WARN loggé (les cartes disparaissent côté front, jamais de 500).
func (r *VehicleCommendationStatsRepo) resolve(ctx context.Context) (vehicle, hijack map[string]struct{}) {
	r.resolveOnce.Do(func() {
		r.vehicleIDs = map[string]struct{}{}
		r.hijackIDs = map[string]struct{}{}
		if r.pdb == nil || r.pdb.Metadata == nil {
			slog.WarnContext(ctx, "vehicle commendations: metadata absente, cartes véhicules/hijacks masquées")
			return
		}
		rows, err := r.pdb.Metadata.Query(ctx, vehicleCommendationResolveQuery)
		if err != nil {
			slog.WarnContext(ctx, "vehicle commendations: référentiel commendation_definitions illisible (non seedé ?), cartes masquées", "err", err)
			return
		}
		defer rows.Close()
		var vehicleNames, hijackNames []string
		for rows.Next() {
			var id, nameEN, nameFR string
			if err := rows.Scan(&id, &nameEN, &nameFR); err != nil {
				continue
			}
			if nameEN == "Grand Theft" || nameFR == "Vol à la tire" {
				r.hijackIDs[id] = struct{}{}
				hijackNames = append(hijackNames, pickName(nameFR, nameEN))
			} else {
				r.vehicleIDs[id] = struct{}{}
				vehicleNames = append(vehicleNames, pickName(nameFR, nameEN))
			}
		}
		if err := rows.Err(); err != nil {
			slog.WarnContext(ctx, "vehicle commendations: itération référentiel", "err", err)
		}
		if len(r.vehicleIDs) == 0 && len(r.hijackIDs) == 0 {
			slog.WarnContext(ctx, "vehicle commendations: aucun UUID résolu (référentiel vide ou noms inattendus), cartes masquées")
			return
		}
		// Trace auditable : le référentiel n'étant pas vérifiable hors prod, on logge
		// les noms retenus pour contrôler l'exclusion des agrégats (Vehicle Mastery, etc.).
		slog.InfoContext(ctx, "vehicle commendations: UUID résolus",
			"vehicle_count", len(r.vehicleIDs), "hijack_count", len(r.hijackIDs),
			"vehicle_names", vehicleNames, "hijack_names", hijackNames)
	})
	return r.vehicleIDs, r.hijackIDs
}

// LoadVehicleDestructionStats agrège, pour un joueur (xuid) sur un scope fermé de
// matchs, les véhicules détruits (Σ des « Destructeur de X ») et les vols à la tire
// (Σ « Vol à la tire »). Voir godoc paquet pour l'exactitude de SUM(count).
func (r *VehicleCommendationStatsRepo) LoadVehicleDestructionStats(
	ctx context.Context, slug string, matchIDs []string, xuid string,
) (port.VehicleDestructionStats, error) {
	var out port.VehicleDestructionStats
	if r.pdb == nil || strings.TrimSpace(xuid) == "" || len(matchIDs) == 0 {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	slog.DebugContext(ctx, "vehicle commendations: load", "slug", slug, "match_count", len(matchIDs))

	vehicleIDs, hijackIDs := r.resolve(ctx)
	if len(vehicleIDs) == 0 && len(hijackIDs) == 0 {
		return out, nil // référentiel non seedé → 0/0 (dégradation déjà loggée)
	}

	allIDs := make([]string, 0, len(vehicleIDs)+len(hijackIDs))
	for id := range vehicleIDs {
		allIDs = append(allIDs, id)
	}
	for id := range hijackIDs {
		allIDs = append(allIDs, id)
	}

	shared, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return out, fmt.Errorf("vehicle commendations: shared reader: %w", err)
	}
	defer release()

	q := buildVehicleCommendationSumQuery(len(matchIDs), len(allIDs))
	args := make([]any, 0, 1+len(matchIDs)+len(allIDs))
	args = append(args, xuid)
	args = append(args, ToAnySlice(matchIDs)...)
	args = append(args, ToAnySlice(allIDs)...)

	rows, err := shared.QueryContext(ctx, q, args...)
	if err != nil {
		return out, fmt.Errorf("vehicle commendations: query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		var total int
		if err := rows.Scan(&id, &total); err != nil {
			return out, fmt.Errorf("vehicle commendations: scan: %w", err)
		}
		if _, ok := vehicleIDs[id]; ok {
			out.VehiclesDestroyed += total
		} else if _, ok := hijackIDs[id]; ok {
			out.Hijacks += total
		}
	}
	return out, rows.Err()
}

// buildVehicleCommendationSumQuery : SUM(count) par commendation_id, scopé xuid +
// matchs + UUID pertinents. La partition véhicule/hijack se fait côté Go (les sets
// résolus servent de dictionnaire) — GROUP BY côté SQL, pas de CASE fragile.
func buildVehicleCommendationSumQuery(nMatch, nIDs int) string {
	return `
SELECT mc.commendation_id, COALESCE(SUM(mc.count), 0)::INTEGER AS total
FROM match_commendations mc
WHERE mc.xuid = ?
  AND mc.match_id IN (` + Placeholders(nMatch) + `)
  AND mc.commendation_id IN (` + Placeholders(nIDs) + `)
GROUP BY mc.commendation_id`
}

// pickName retourne le nom FR s'il est non vide, sinon l'EN (log auditable lisible).
func pickName(nameFR, nameEN string) string {
	if strings.TrimSpace(nameFR) != "" {
		return nameFR
	}
	return nameEN
}
