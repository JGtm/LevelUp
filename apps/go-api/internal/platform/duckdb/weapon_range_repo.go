// Package duckdb — weapon_range_repo.go : implémentation DuckDB de
// port.WeaponRangeRepository (plan .ai/PLAN_DUELS_PORTEE_2026-09-06.md, lot 3).
//
// # DEUX CÔTÉS, UNE SEULE JOINTURE — ET UNE SEULE LECTURE
//
// « Où je frague » et « où je meurs » sont la MÊME mesure vue de deux côtés : l'un compare le
// joueur à `e.feed_killer_xuid`, l'autre à `e.victim_xuid`, et rien d'autre ne change. Depuis
// le lot L5a du plan perf (2026-09-23), les deux côtés sortent d'UNE requête (cf.
// loadBothSides) : deux requêtes recalculaient deux fois les mêmes vues du même scope.
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
	"database/sql"
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

// loadBothSides lit les DEUX côtés (tueur puis victime) en UNE SEULE requête, sous un seul
// emprunt du lecteur partagé — le lease est la ressource la plus disputée du process
// (ADR 0013).
//
// UNE REQUÊTE, PLUS DEUX (lot L5a du plan perf, 2026-09-23). Les deux côtés sont LES MÊMES
// FRAGS MESURÉS, filtrés sur deux colonnes : deux requêtes recalculaient deux fois les vues
// `_latest` du scope, la jointure et la garde `fragSolo` — le coût d'une lecture est là,
// pas dans le filtre de joueur (mesuré sur copie, 1 158 matchs : 3,5 s en deux requêtes,
// 1,7 s en une). La requête garde les frags où le joueur est tueur OU victime, puis
// `separerCotes` les range en Go. RIEN NE CHANGE AUX GARDES : `fragSolo` est jugée sur le
// groupe COMPLET avant tout filtre de joueur, donc chaque groupe retenu porte UNE seule
// mort — la même ligne que la lecture d'un seul côté aurait rendue.
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

	joueurs, err := joueursDuFiltre(ctx, db, filters)
	if err != nil {
		return nil, echecLecture(ctx, scope, slug, table, err)
	}
	q, args := buildWeaponRangeBothSidesQuery(table, filters, joueurs)
	measured, err := queryMeasuredKills(ctx, db, q, args, scope)
	if err != nil {
		return nil, echecLecture(ctx, scope, slug, table, err)
	}
	tueur, victime := separerCotes(measured, joueurs, filters.AllPlayers)
	out := make([]analysis.MeasuredKill, 0, len(tueur)+len(victime))
	out = append(out, r.toMeasuredKills(ctx, tueur, analysis.SideKiller, scope)...)
	return append(out, r.toMeasuredKills(ctx, victime, analysis.SideVictim, scope)...), nil
}

// echecLecture traduit une table absente en games.ErrCapabilityNotSupported (titre sans
// décodeur de film : une absence, Debug) et journalise tout le reste avant de le propager.
func echecLecture(ctx context.Context, scope, slug string, table measuredPositionsTable, err error) error {
	if isTableNotFoundErr(err) {
		slog.DebugContext(ctx, "WeaponRangeRepo: positions table missing",
			"scope", scope, "slug", slug, "table", string(table))
		return games.ErrCapabilityNotSupported
	}
	slog.ErrorContext(ctx, "WeaponRangeRepo: query failed", "scope", scope, "slug", slug, "err", err)
	return fmt.Errorf("WeaponRangeRepo.%s: %w", scope, err)
}

// joueursDuFiltre rend les xuid que le filtre de joueur DÉSIGNE — ceux que la lecture d'un
// côté comparait à sa colonne. `XUIDs` d'abord, sinon le gamertag, résolu par le MÊME
// sous-select `xuid_aliases` que appendXUIDFilter (l'unique passage du paquet : on l'appelle,
// on ne recopie pas son littéral). nil pour AllPlayers : aucun joueur n'est désigné.
//
// Un xuid VIDE ne désigne personne : la colonne comparée ne vaut jamais la chaîne vide, et
// `separerCotes` compare aux identités scannées, où NULL devient "".
func joueursDuFiltre(ctx context.Context, db *sql.DB, f port.WeaponRangeFilters) ([]string, error) {
	if f.AllPlayers {
		return nil, nil
	}
	if len(f.XUIDs) > 0 {
		return sansVides(f.XUIDs), nil
	}
	var sb strings.Builder
	args := make([]any, 0, 1)
	sb.WriteString("SELECT DISTINCT a.xuid FROM xuid_aliases a WHERE a.xuid IS NOT NULL")
	appendXUIDFilter(&sb, &args, "a.xuid", port.WeaponKillFilters{Gamertag: f.Gamertag})
	rows, err := db.QueryContext(ctx, sb.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("joueurs du filtre: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := make([]string, 0, 1)
	for rows.Next() {
		var x string
		if err := rows.Scan(&x); err != nil {
			return nil, fmt.Errorf("joueurs du filtre, scan: %w", err)
		}
		out = append(out, x)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("joueurs du filtre, curseur: %w", err)
	}
	return sansVides(out), nil
}

// sansVides retire les chaînes vides (cf. joueursDuFiltre). Ordre conservé.
func sansVides(xuids []string) []string {
	out := make([]string, 0, len(xuids))
	for _, x := range xuids {
		if x != "" {
			out = append(out, x)
		}
	}
	return out
}

// buildWeaponRangeBothSidesQuery compose la lecture des DEUX côtés : la portée, puis « le
// joueur est le tueur OU la victime ». AllPlayers : aucune clause, tous les frags du scope
// (les deux côtés rendent alors tout, comme les deux lectures d'autrefois).
//
// AUCUN JOUEUR DÉSIGNÉ (gamertag inconnu d'xuid_aliases) : la clause devient `AND FALSE` —
// la requête s'exécute quand même, pour qu'une table de positions ABSENTE se dise encore
// games.ErrCapabilityNotSupported, exactement comme les deux lectures d'autrefois.
func buildWeaponRangeBothSidesQuery(
	table measuredPositionsTable, f port.WeaponRangeFilters, joueurs []string,
) (string, []any) {
	fragScope, where, args := buildWeaponRangeScope(f)
	if f.AllPlayers {
		return measuredKillsQuery(table, fragScope, where), args
	}
	if len(joueurs) == 0 {
		return measuredKillsQuery(table, fragScope, where+"\n  AND FALSE"), args
	}
	liste := Placeholders(len(joueurs))
	designes := ToAnySlice(joueurs)
	args = append(args, designes...)
	args = append(args, designes...)
	where += "\n  AND (" + weaponRangeKillerColumn + " IN (" + liste + ")" +
		" OR " + weaponRangeVictimColumn + " IN (" + liste + "))"
	return measuredKillsQuery(table, fragScope, where), args
}

// separerCotes range les frags lus en une passe : côté TUEUR ceux dont le joueur est le
// tueur, côté VICTIME ceux dont il est la victime — un frag qui vérifie les deux (le joueur
// s'est tué lui-même) figure des deux côtés, comme il figurait dans les deux lectures
// d'autrefois. AllPlayers : tous les frags des deux côtés. L'ordre de lecture est conservé.
func separerCotes(measured []killMeasured, joueurs []string, tous bool) (tueur, victime []killMeasured) {
	if tous {
		return measured, measured
	}
	designe := make(map[string]struct{}, len(joueurs))
	for _, x := range joueurs {
		designe[x] = struct{}{}
	}
	for _, m := range measured {
		if _, ok := designe[m.killerXUID]; ok {
			tueur = append(tueur, m)
		}
		if _, ok := designe[m.victimXUID]; ok {
			victime = append(victime, m)
		}
	}
	return tueur, victime
}

// buildWeaponRangeScope compose les clauses de PORTÉE communes à toutes les lectures du
// repo : le scope de la sous-requête `fragSolo`, puis celui de la requête externe, posé sur
// LES DEUX VUES qu'elle joint.
//
// LES MATCH_ID SONT LIÉS TROIS FOIS, ET L'ORDRE COMPTE : d'abord ceux de `fragSolo` (la
// sous-requête précède la clause WHERE dans le texte SQL), ensuite `e`, ensuite `kp`.
//
//   - Sans le scope de `fragSolo`, DuckDB balaie la vue entière pour juger l'unicité du frag
//     — mesuré ×15,6 (résidu 4.0a).
//   - Sans celui de `kp` (lot L5a du plan perf, 2026-09-23), la vue des positions se calcule
//     sur la table ENTIÈRE : le `e.match_id IN (...)` ne traverse pas la jointure jusqu'à la
//     fenêtre de `kp` (seule une ÉGALITÉ `= ?` le fait — cf. KillDistanceRepo). Mesuré sur
//     copie, 6 matchs : 0,61 s par lecture sans lui, 0,05 s avec. La clause est redondante
//     avec la condition de jointure `kp.match_id = e.match_id` : elle ne retire aucune ligne.
func buildWeaponRangeScope(f port.WeaponRangeFilters) (fragScope string, where string, args []any) {
	ids := ToAnySlice(f.MatchIDs)
	liste := Placeholders(len(f.MatchIDs))
	args = make([]any, 0, 3*len(ids)+2*len(f.XUIDs)+2)
	args = append(args, ids...)
	args = append(args, ids...)
	args = append(args, ids...)
	return "s.match_id IN (" + liste + ")",
		"e.match_id IN (" + liste + ")\n  AND kp.match_id IN (" + liste + ")", args
}

// buildWeaponRangeQuery compose la portée puis le filtre de joueur d'UN côté, et délègue la
// jointure à l'helper canonique. `column` est la colonne de xuid du côté lu. Les arguments du
// filtre de joueur suivent ceux de la portée.
func buildWeaponRangeQuery(
	table measuredPositionsTable, column string, f port.WeaponRangeFilters,
) (string, []any) {
	fragScope, where, args := buildWeaponRangeScope(f)
	var sb strings.Builder
	sb.WriteString(where)

	// AllPlayers : AUCUNE clause de joueur. Le scan reste borné par les match_id (les
	// clauses de portée) — c'est exactement la portée de `KillDistanceRepo.LoadMatch`, qui
	// lit déjà un match entier sans filtre xuid. Voir port.WeaponRangeFilters.AllPlayers.
	if !f.AllPlayers {
		// Projection sur WeaponKillFilters : le helper de filtre xuid est unique dans le
		// paquet (résolution gamertag -> xuid via xuid_aliases identique), on ne le recopie pas.
		appendXUIDFilter(&sb, &args, column, port.WeaponKillFilters{
			Gamertag: f.Gamertag,
			XUIDs:    f.XUIDs,
		})
	}

	return measuredKillsQuery(table, fragScope, sb.String()), args
}

// ResolveWeaponLabels traduit des clés de registre en noms d'affichage (port.WeaponLabelResolver).
//
// LA MÊME RÉSOLUTION QUE `KillDistanceRepo.resolveRows`, ET PAS UNE SECONDE : elle passe par
// `resolveWeaponKeyLabelsAny`, l'unique passage du paquet pour cette traduction (source de nom
// keyée par weapon_key, cf. weapon_resolver.go). Une copie ici divergerait sur l'ordre de
// priorité FR/EN, et deux pages nommeraient la même arme différemment.
//
// BEST-EFFORT, SANS ERREUR REMONTÉE : le résolveur sous-jacent journalise ses pannes (registre
// absent, requête en échec) et rend ce qu'il a. La signature garde `error` pour que ce contrat
// puisse un jour dire non — aujourd'hui elle vaut toujours nil, et l'appelant qui la teste ne
// fait rien d'inutile : il se protège d'un futur implémenteur qui échouerait vraiment.
func (r *WeaponRangeRepo) ResolveWeaponLabels(
	ctx context.Context, weaponKeys []string,
) (map[string]port.WeaponLabel, error) {
	out := make(map[string]port.WeaponLabel, len(weaponKeys))
	if len(weaponKeys) == 0 {
		return out, nil
	}
	for key, meta := range resolveWeaponKeyLabelsAny(ctx, r.pdb.Metadata, r.pdb.TitleSlug, weaponKeys) {
		// Une entrée sans aucun nom n'en est pas une : la laisser passer ferait publier un
		// libellé vide là où l'appelant sait retomber sur la clé.
		if meta.label == "" && meta.labelEN == "" {
			continue
		}
		out[key] = port.WeaponLabel{Label: meta.label, LabelEN: meta.labelEN}
	}
	slog.DebugContext(ctx, "WeaponRangeRepo: weapon labels resolved",
		"slug", r.pdb.TitleSlug, "demandees", len(weaponKeys), "resolues", len(out))
	return out, nil
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

// ResolveWeaponDimensions traduit des clés de registre en dimensions class/role/family
// (port.WeaponRangeRepository).
//
// DÉLÉGATION PURE À `resolveWeaponKeyDimensions`, l'unique résolution du paquet : la même que
// celle qui sert les kills par arme et la vue d'un match. Une seconde requête ici pourrait
// diverger sur la façon de départager une clé à plusieurs identifiants — et deux pages
// classeraient alors la même arme dans deux rôles.
//
// LE SLUG EST UN PARAMÈTRE, PAS `r.pdb.TitleSlug` : l'appelant lit les frags mesurés d'un
// titre donné (les deux lectures de ce repo prennent déjà leur slug en argument) et doit
// pouvoir résoudre les clés du MÊME titre. Les lire sous le titre du PlayerDB rendrait les
// dimensions d'un autre jeu dès que les deux diffèrent.
//
// BEST-EFFORT : registre absent ou metadata non migrée -> map VIDE et pas d'erreur. Une clé
// résolue sans aucune dimension est écartée — une entrée à trois champs vides ne dit rien de
// plus que son absence, et l'appelant qui la recevrait croirait la clé connue.
func (r *WeaponRangeRepo) ResolveWeaponDimensions(
	ctx context.Context, titleSlug string, keys []string,
) (map[string]port.WeaponDimensions, error) {
	out := make(map[string]port.WeaponDimensions, len(keys))
	if len(keys) == 0 {
		return out, nil
	}
	for key, meta := range resolveWeaponKeyDimensions(ctx, r.pdb.Metadata, titleSlug, keys) {
		if meta.class == "" && meta.role == "" && meta.family == "" {
			continue
		}
		out[key] = port.WeaponDimensions{Class: meta.class, Role: meta.role, Family: meta.family}
	}
	slog.DebugContext(ctx, "WeaponRangeRepo: weapon dimensions resolved",
		"slug", titleSlug, "demandees", len(keys), "resolues", len(out))
	return out, nil
}
