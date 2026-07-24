// Package duckdb — vehicle_commendation_stats_repo.go : lecture SCOPÉE des
// compteurs « véhicules détruits » et « vol à la tire » (hijacks) d'un joueur, pour
// Halo 5.
//
// Contexte : sur Halo Infinite ces deux stats viennent de personal_score_awards
// (awards destroyed_*/hijacked_*). Halo 5 n'a PAS de personal_score_awards. Les deux
// compteurs ont deux SOURCES DIFFÉRENTES côté H5 :
//   - Véhicules détruits : commendations NATIVES per-match (ProgressiveCommendationDeltas
//     → match_commendations, cf. games/halo_5/mapping_commendations.go) — les 9
//     commendations « Destructeur de X ».
//   - Vol à la tire (hijacks) : CORRECTION — il n'existe PAS de commendation « Grand
//     Theft »/« Vol à la tire » dans le référentiel Halo 5 (vérifié sur corpus prod
//     2026-07-23 : 0 ligne commendation_definitions ne matche). La donnée existe en
//     revanche comme MÉDAILLE (medals_earned, cf. games/halo_5/ingest/medals.go) :
//     Hijack (medal_name_id 1219497744, véhicules terrestres/nautiques) + Skyjack
//     (1801925525, variante AÉRIENNE — parité avec Halo Infinite où hijacked_* inclut
//     aussi les véhicules aériens). H5 hijacks = SUM(Hijack) + SUM(Skyjack). Fact-check
//     corpus : Hijack 212 occurrences (26 matchs), Skyjack 37 (2 matchs).
//
// EXCLUSIONS véhicules détruits (décision produit) : Écrasement/Splatter (splatter =
// kill, pas destruction), Maîtrise de véhicule / Vehicle Mastery (badge agrégé),
// Vandalisme (sémantique non confirmée) — la résolution par nom ne les capture pas
// (aucun ne matche le motif « Destructeur … » / « % Destroyer »), et deux garde-fous
// EN excluent explicitement un éventuel agrégat « Vehicle Destroyer/Mastery ».
//
// ART-safety véhicules (match_commendations) : INSERT-only, keyée (match_id, xuid,
// commendation_id) via INSERT OR IGNORE côté persister (cf.
// persist/shared_persister.go persistCommendations + migration
// steps_shared_commendations.go). Le `count` par-match (Progress − PreviousProgress)
// est immuable — aucun doublon possible sur la clé naturelle, donc SUM(count) sur un
// scope de matchs est EXACT et n'exige PAS de vue `_latest` (parité
// citations_repo.matchCommendationsRichQuery ; la vue _latest ne concerne QUE les
// tables re-INSÉRÉES avec written_at, ADR 0026 — ce qui n'est pas le cas ici).
//
// ART-safety hijacks (medals_earned) : MÊME analyse — PK (match_id, xuid,
// medal_name_id) posée par ApplyMedalsBigint (games/halo_5/migrations/medals.go),
// écriture INSERT OR IGNORE (persist/shared_persister.go persistMedals, commentaire
// « pour tolérer les doublons dans le payload API »). Une ligne par clé naturelle,
// `count` immuable au niveau de la ligne (agrégé UNE fois à l'ingestion, cf.
// games/halo_5/ingest/medals.go MapMedalEvents) → SUM(count) scopé xuid/matchs/
// medal_name_id est EXACT, aucune vue `_latest` requise (pas de written_at sur cette
// table, donc hors périmètre ADR 0026 comme match_commendations).
package duckdb

import (
	"context"
	"database/sql"
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
// gating par slug). Malgré son nom (hérité — véhicules détruits = commendations),
// le repo agrège aussi les hijacks depuis une source DIFFÉRENTE (medals_earned, cf.
// package doc) : un renommage complet toucherait le câblage registry_pages_home.go
// pour un gain cosmétique seul — non fait ici (périmètre de la mission).
type VehicleCommendationStatsRepo struct {
	pdb *PlayerDB

	// Résolution des UUID véhicules par NOM, faite UNE FOIS (au premier appel) puis
	// mémoïsée : le référentiel commendation_definitions est immuable au runtime
	// (seedé au boot par cmd/h5-metadata-fetch, versionné). ~10 lignes retenues sur
	// ~88 → scan LIKE négligeable.
	resolveOnce sync.Once
	vehicleIDs  map[string]struct{}

	// Résolution des medal_name_id hijack (Hijack/Skyjack) par NOM via
	// medal_definitions, avec repli sur les ids constants documentés (cf.
	// hijackMedalNamesToConstantIDs) — mémoïsée séparément : indépendante de la
	// résolution commendations (deux référentiels distincts, deux DB logiques).
	hijackMedalOnce sync.Once
	hijackMedalIDs  map[int64]struct{}
}

// NewVehicleCommendationStatsRepo lie le repo à un PlayerDB (porte le SharedReader
// pour match_commendations/medals_earned + le handle Metadata pour
// commendation_definitions/medal_definitions).
func NewVehicleCommendationStatsRepo(pdb *PlayerDB) *VehicleCommendationStatsRepo {
	return &VehicleCommendationStatsRepo{pdb: pdb}
}

// vehicleCommendationResolveQuery résout les UUID des commendations « véhicule
// détruit » par NOM (locale-agnostic : matche name_en OU name_fr, robuste au seed
// FR/EN). Le motif FR « Destructeur % » (espace) capture aussi « Destructeur
// d'apparitions » (wraith) ; le motif EN « % Destroyer » capture « Banshee
// Destroyer », etc. Le garde-fou NOT exclut un éventuel badge agrégé.
//
// Ne résout PLUS « Grand Theft »/« Vol à la tire » (hijack) : cette commendation
// n'existe pas dans le référentiel Halo 5 (cf. package doc) — le hijack est résolu
// séparément depuis medal_definitions/medals_earned (resolveHijackMedalIDs).
const vehicleCommendationResolveQuery = `
SELECT commendation_id,
       COALESCE(name_en, '') AS name_en,
       COALESCE(name_fr, '') AS name_fr
FROM commendation_definitions
WHERE (
        name_en LIKE '% Destroyer'
     OR name_fr LIKE 'Destructeur %'
      )
  AND name_en <> 'Vehicle Destroyer'
  AND name_en <> 'Vehicle Mastery'`

// resolve renseigne (une seule fois) le set d'UUID commendation « véhicule détruit »
// depuis la metadata. Dégradation gracieuse : metadata nil, table absente, ou 0 UUID
// résolu → set vide + WARN loggé (la carte disparaît côté front, jamais de 500).
func (r *VehicleCommendationStatsRepo) resolve(ctx context.Context) map[string]struct{} {
	r.resolveOnce.Do(func() {
		r.vehicleIDs = map[string]struct{}{}
		if r.pdb == nil || r.pdb.Metadata == nil {
			slog.WarnContext(ctx, "vehicle commendations: metadata absente, carte véhicules détruits masquée")
			return
		}
		rows, err := r.pdb.Metadata.Query(ctx, vehicleCommendationResolveQuery)
		if err != nil {
			slog.WarnContext(ctx, "vehicle commendations: référentiel commendation_definitions illisible (non seedé ?), carte masquée", "err", err)
			return
		}
		defer rows.Close()
		var vehicleNames []string
		for rows.Next() {
			var id, nameEN, nameFR string
			if err := rows.Scan(&id, &nameEN, &nameFR); err != nil {
				continue
			}
			r.vehicleIDs[id] = struct{}{}
			vehicleNames = append(vehicleNames, pickName(nameFR, nameEN))
		}
		if err := rows.Err(); err != nil {
			slog.WarnContext(ctx, "vehicle commendations: itération référentiel", "err", err)
		}
		if len(r.vehicleIDs) == 0 {
			slog.WarnContext(ctx, "vehicle commendations: aucun UUID résolu (référentiel vide ou noms inattendus), carte masquée")
			return
		}
		// Trace auditable : le référentiel n'étant pas vérifiable hors prod, on logge
		// les noms retenus pour contrôler l'exclusion des agrégats (Vehicle Mastery, etc.).
		slog.InfoContext(ctx, "vehicle commendations: UUID résolus",
			"vehicle_count", len(r.vehicleIDs), "vehicle_names", vehicleNames)
	})
	return r.vehicleIDs
}

// hijackMedalNamesToConstantIDs documente les 2 medal_name_id « Vol à la tire »
// Halo 5. Ce sont des ids STABLES (hash du nom interne côté jeu, contrairement aux
// commendation_id qui sont des UUID régénérables au reseed) : ils servent de repli
// fiable si le nom est introuvable dans medal_definitions (référentiel absent, pas
// encore seedé, ou reseed inattendu).
//   - Hijack  (1219497744) : véhicules terrestres/nautiques.
//   - Skyjack (1801925525) : variante AÉRIENNE — parité Halo Infinite (hijacked_*
//     inclut aussi les véhicules aériens, cf. sync/refdata_personal_scores.go).
var hijackMedalNamesToConstantIDs = map[string]int64{
	"Hijack":  1219497744,
	"Skyjack": 1801925525,
}

// hijackMedalResolveQuery résout les medal_name_id par name_en (langue de seed
// canonique du référentiel content-hacs, cf. medal_definitions_repo.go).
const hijackMedalResolveQuery = `
SELECT medal_name_id, name_en
FROM medal_definitions
WHERE name_en IN ('Hijack', 'Skyjack')`

// resolveHijackMedalIDs renseigne (une seule fois) le set des medal_name_id « vol à
// la tire », résolus PAR NOM en priorité (cohérent avec resolve() ci-dessus, robuste
// à un reseed qui changerait les ids) — avec repli sur hijackMedalNamesToConstantIDs
// si le nom manque. Ce repli ne dégrade JAMAIS en 0 : au pire (metadata absente ou
// medal_definitions non seedée), les 2 ids constants sont quand même posés — la
// dégradation gracieuse réelle (0 sans erreur) a lieu côté sumHijackMedals, si
// medals_earned lui-même est absent/vide sur le scope demandé.
func (r *VehicleCommendationStatsRepo) resolveHijackMedalIDs(ctx context.Context) map[int64]struct{} {
	r.hijackMedalOnce.Do(func() {
		r.hijackMedalIDs = make(map[int64]struct{}, len(hijackMedalNamesToConstantIDs))
		resolvedByName := make(map[string]int64, len(hijackMedalNamesToConstantIDs))

		if r.pdb == nil || r.pdb.Metadata == nil {
			slog.WarnContext(ctx, "hijack medals: metadata absente, repli sur les ids constants documentés")
		} else if rows, err := r.pdb.Metadata.Query(ctx, hijackMedalResolveQuery); err != nil {
			slog.WarnContext(ctx, "hijack medals: référentiel medal_definitions illisible (non seedé ?), repli sur les ids constants", "err", err)
		} else {
			defer rows.Close()
			for rows.Next() {
				var id int64
				var nameEN string
				if err := rows.Scan(&id, &nameEN); err != nil {
					continue
				}
				resolvedByName[nameEN] = id
			}
			if err := rows.Err(); err != nil {
				slog.WarnContext(ctx, "hijack medals: itération référentiel", "err", err)
			}
		}

		for name, constID := range hijackMedalNamesToConstantIDs {
			if id, ok := resolvedByName[name]; ok {
				r.hijackMedalIDs[id] = struct{}{}
				continue
			}
			slog.WarnContext(ctx, "hijack medals: id résolu par repli constant (nom introuvable dans medal_definitions)",
				"medal_name", name, "medal_name_id", constID)
			r.hijackMedalIDs[constID] = struct{}{}
		}
		slog.InfoContext(ctx, "hijack medals: ids retenus", "count", len(r.hijackMedalIDs))
	})
	return r.hijackMedalIDs
}

// LoadVehicleDestructionStats agrège, pour un joueur (xuid) sur un scope fermé de
// matchs, les véhicules détruits (Σ des « Destructeur de X », commendations) et les
// vols à la tire (Σ Hijack + Skyjack, médailles — cf. package doc). Deux sources
// indépendantes : une résolution en échec sur l'une n'affecte pas l'autre.
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

	shared, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return out, fmt.Errorf("vehicle commendations: shared reader: %w", err)
	}
	defer release()

	if vehicleIDs := r.resolve(ctx); len(vehicleIDs) > 0 {
		total, err := sumVehicleDestroyerCommendations(ctx, shared, xuid, matchIDs, vehicleIDs)
		if err != nil {
			return out, fmt.Errorf("vehicle commendations: query: %w", err)
		}
		out.VehiclesDestroyed = total
	}

	// Best-effort (cf. sumHijackMedals) : une erreur sur cette source annexe ne fait
	// jamais échouer la fonction — le véhicule détruit reste servi.
	out.Hijacks = r.sumHijackMedals(ctx, shared, xuid, matchIDs)

	return out, nil
}

// sumVehicleDestroyerCommendations : SUM(count) sur match_commendations, scopé xuid +
// matchs + UUID « véhicule détruit » résolus. Erreur SQL propagée (obligatoire :
// c'est la source confirmée d'existence de cette stat).
func sumVehicleDestroyerCommendations(
	ctx context.Context, shared *sql.DB, xuid string, matchIDs []string, vehicleIDs map[string]struct{},
) (int, error) {
	ids := make([]string, 0, len(vehicleIDs))
	for id := range vehicleIDs {
		ids = append(ids, id)
	}

	q := buildVehicleCommendationSumQuery(len(matchIDs), len(ids))
	args := make([]any, 0, 1+len(matchIDs)+len(ids))
	args = append(args, xuid)
	args = append(args, ToAnySlice(matchIDs)...)
	args = append(args, ToAnySlice(ids)...)

	var total int
	if err := shared.QueryRowContext(ctx, q, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

// buildVehicleCommendationSumQuery : SUM(count) total, scopé xuid + matchs + UUID
// « véhicule détruit » (un seul total : plus de partition par commendation depuis
// que le hijack a sa propre source — cf. sumHijackMedals).
func buildVehicleCommendationSumQuery(nMatch, nIDs int) string {
	return `
SELECT COALESCE(SUM(mc.count), 0)::INTEGER AS total
FROM match_commendations mc
WHERE mc.xuid = ?
  AND mc.match_id IN (` + Placeholders(nMatch) + `)
  AND mc.commendation_id IN (` + Placeholders(nIDs) + `)`
}

// sumHijackMedals agrège Σ count sur medals_earned pour les medal_name_id « vol à la
// tire » (Hijack + Skyjack), scopé xuid + matchIDs. Best-effort : contrairement aux
// véhicules détruits (query obligatoire, seule source connue), une erreur ici (table
// medals_earned absente, DB indisponible) degrade en 0 SANS remonter d'erreur — perdre
// ce fun-stat annexe ne doit jamais faire échouer toute la page Synthesis (cf.
// port.VehicleDestructionStatsRepository godoc : « jamais de 500 »).
func (r *VehicleCommendationStatsRepo) sumHijackMedals(
	ctx context.Context, shared *sql.DB, xuid string, matchIDs []string,
) int {
	medalIDs := r.resolveHijackMedalIDs(ctx)
	if len(medalIDs) == 0 {
		return 0
	}
	ids := make([]int64, 0, len(medalIDs))
	for id := range medalIDs {
		ids = append(ids, id)
	}

	q := buildHijackMedalSumQuery(len(matchIDs), len(ids))
	args := make([]any, 0, 1+len(matchIDs)+len(ids))
	args = append(args, xuid)
	args = append(args, ToAnySlice(matchIDs)...)
	args = append(args, ToAnySlice(ids)...)

	var total int
	if err := shared.QueryRowContext(ctx, q, args...).Scan(&total); err != nil {
		slog.WarnContext(ctx, "hijack medals: query medals_earned failed (best-effort, dégradation à 0)",
			"err", err, "match_count", len(matchIDs))
		return 0
	}
	return total
}

// buildHijackMedalSumQuery : SUM(count) total sur medals_earned pour les
// medal_name_id fournis, scopé xuid + matchs. Un seul total (pas de partition
// Hijack/Skyjack) : le produit affiche un compteur combiné « Vol à la tire », parité
// avec Halo Infinite où hijacked_* fusionne déjà sol/air.
func buildHijackMedalSumQuery(nMatch, nIDs int) string {
	return `
SELECT COALESCE(SUM(count), 0)::INTEGER AS total
FROM medals_earned
WHERE xuid = ?
  AND match_id IN (` + Placeholders(nMatch) + `)
  AND medal_name_id IN (` + Placeholders(nIDs) + `)`
}

// pickName retourne le nom FR s'il est non vide, sinon l'EN (log auditable lisible).
func pickName(nameFR, nameEN string) string {
	if strings.TrimSpace(nameFR) != "" {
		return nameFR
	}
	return nameEN
}
