package duckdb

// map_variants.go — le regroupement des identifiants de carte qui désignent un même terrain (variantes
// d'une carte, republications d'une carte intégrée) sous leur carte de base : la lecture de la table
// `GroupesDeCartes` et son application aux lignes des filtres et de l'historique.
//
// « Thunderhead Heavies » et « Live Fire - Ranked » sont des cartes distinctes au registre des
// matchs (identifiant propre) alors que c'est le même terrain que « Thunderhead » et « Live Fire ».
// Une carte intégrée republiée sous un nouvel asset a plusieurs identifiants du même nom. Toute
// statistique par carte les lit comme UN groupe ; la table est construite ICI, une seule fois, et
// consommée partout par `domain.GroupesDeCartes`.
//
// LA TABLE SE CONSTRUIT SUR LE NOM RETENU DU REGISTRE (nomDeCarteRetenuSQL), PAS SUR LE CATALOGUE :
// le catalogue (maps_catalog) ne porte qu'un identifiant par nom et ignore des variantes que le
// registre a jouées, alors qu'une carte republiée a plusieurs identifiants au registre. La règle de
// rattachement est analysis.GroupMapsByBase (analysis.BaseMapName + suffixes du titre,
// mappings/map_variants.toml, et la clé de module que pose analysis.MapGroupRules.Modules) ; le
// libellé FR de la base est celui de libelleDeCarte (map_labels.go), lu sur les traductions d'asset.

import (
	"context"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/port"
)

// qCartesDuRegistre : une ligne par carte jouée, avec son nom retenu et son nombre de matchs. La
// clé de regroupement est le seul map_id (garde-rail archlint/no_group_by_registry_name_test.go).
var qCartesDuRegistre = `
SELECT mr.map_id,
       ` + nomDeCarteRetenuSQL("map_name") + ` AS map_name,
       COUNT(*)                          AS matchs
FROM match_registry mr
WHERE mr.map_id IS NOT NULL AND mr.map_id <> ''
GROUP BY mr.map_id`

// MapGroupsRepo implémente port.MapGroupReader pour le titre d'un PlayerDB.
type MapGroupsRepo struct {
	pdb   *PlayerDB
	rules analysis.MapGroupRules
}

var _ port.MapGroupReader = (*MapGroupsRepo)(nil)

// NewMapGroupsRepo crée un MapGroupsRepo depuis un PlayerDB.
func NewMapGroupsRepo(pdb *PlayerDB) *MapGroupsRepo {
	return &MapGroupsRepo{pdb: pdb}
}

// WithMapGroupRules injecte les règles de regroupement des cartes du titre : suffixes de variante
// (mappings/map_variants.toml) et résolveur de clé de module. Sans règle, aucune carte n'est regroupée.
// Retourne le repo.
func (r *MapGroupsRepo) WithMapGroupRules(rules analysis.MapGroupRules) *MapGroupsRepo {
	r.rules = rules
	return r
}

// MapGroups rend la table des groupes de cartes du titre, avec le libellé EN et FR de chaque carte de
// base (libellesDesBases).
func (r *MapGroupsRepo) MapGroups(ctx context.Context) (domain.GroupesDeCartes, error) {
	lecture, err := lireGroupesDeCartes(ctx, r.pdb, r.rules)
	if err != nil || lecture.Groups.Vide() {
		return lecture.Groups, err
	}
	bases := libellesDesBases(ctx, r.pdb, lecture)
	libelles := make(map[string]map[string]string, len(bases))
	for rep, base := range bases {
		libelles[rep] = map[string]string{mappings.LocaleEN: base.EN, mappings.LocaleFR: base.FR}
	}
	return lecture.Groups.AvecLibelles(libelles), nil
}

// lireGroupesDeCartes lit le registre des matchs et construit les groupes. Sans règle ou sans base
// partagée : aucun groupe, sans lecture.
func lireGroupesDeCartes(ctx context.Context, pdb *PlayerDB, rules analysis.MapGroupRules) (analysis.MapGroups, error) {
	if !rules.Active() || pdb == nil {
		return analysis.MapGroups{}, nil
	}
	db, release, err := pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return analysis.MapGroups{}, fmt.Errorf("groupes de cartes: shared reader: %w", err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, qCartesDuRegistre)
	if err != nil {
		return analysis.MapGroups{}, fmt.Errorf("groupes de cartes: lecture du registre: %w", err)
	}
	defer rows.Close()

	var cartes []analysis.MapNameCount
	for rows.Next() {
		var c analysis.MapNameCount
		if err := rows.Scan(&c.MapID, &c.Name, &c.Matches); err != nil {
			return analysis.MapGroups{}, fmt.Errorf("groupes de cartes: scan: %w", err)
		}
		cartes = append(cartes, c)
	}
	if err := rows.Err(); err != nil {
		return analysis.MapGroups{}, fmt.Errorf("groupes de cartes: lecture: %w", err)
	}
	return analysis.GroupMapsByBase(rules.WithModules(ctx, cartes), rules.Suffixes), nil
}

// carteDeBase : la carte de base d'un groupe, avec son nom EN et son libellé de filtre.
type carteDeBase struct {
	EN string
	FR string
}

// libellesDesBases rend, par identifiant représentatif, le nom EN et le libellé FR de la base de
// chaque groupe. Le libellé FR est la traduction d'asset du représentant, à défaut celle d'un autre
// identifiant portant le nom de base, à défaut le nom EN (libelleDeCarte).
func libellesDesBases(ctx context.Context, pdb *PlayerDB, groupes analysis.MapGroups) map[string]carteDeBase {
	var ids []string
	for _, base := range groupes.Bases {
		ids = append(ids, base.IDs...)
	}
	var traductions map[string]string
	if pdb != nil {
		traductions = traductionsDeCartes(ctx, pdb.Metadata, ids)
	}
	out := make(map[string]carteDeBase, len(groupes.Bases))
	for rep, base := range groupes.Bases {
		traduction := traductions[rep]
		for _, id := range base.IDs {
			if traduction != "" {
				break
			}
			traduction = traductions[id]
		}
		out[rep] = carteDeBase{EN: base.Name, FR: libelleDeCarte("", base.Name, traduction)}
	}
	return out
}

// groupesEtLibelles lit les groupes et les libellés de leurs bases. Un échec de lecture est
// journalisé et consigné (bestEffortFailed : les lignes ne sont alors pas mises en cache), puis
// dégradé : aucune variante n'est rattachée, ok = false.
func groupesEtLibelles(ctx context.Context, pdb *PlayerDB, rules analysis.MapGroupRules) (analysis.MapGroups, map[string]carteDeBase, bool) {
	groupes, err := lireGroupesDeCartes(ctx, pdb, rules)
	if err != nil {
		slog.WarnContext(ctx, "cartes : groupes de variantes illisibles, variantes laissées seules", "err", err)
		bestEffortFailed(ctx, "map_groups", err)
		return analysis.MapGroups{}, nil, false
	}
	if groupes.Groups.Vide() {
		return groupes, nil, false
	}
	return groupes, libellesDesBases(ctx, pdb, groupes), true
}

// rattacherCartesDeBase réécrit le nom de carte (EN et FR) des lignes de filtres dont la carte
// appartient à un groupe : les options et le filtrage, qui lisent ces noms, ne voient plus que la
// carte de base. Le MapID reste celui de la ligne.
func rattacherCartesDeBase(ctx context.Context, pdb *PlayerDB, rules analysis.MapGroupRules, rows []domain.FilterMatchRow) {
	groupes, bases, ok := groupesEtLibelles(ctx, pdb, rules)
	if !ok {
		return
	}
	for i := range rows {
		if rows[i].MapID == nil || !groupes.Groups.Regroupe(*rows[i].MapID) {
			continue
		}
		base := bases[groupes.Groups.Groupe(*rows[i].MapID)]
		en, fr := base.EN, base.FR
		rows[i].MapName = &en
		rows[i].MapNameFR = &fr
	}
}

// rattacherCartesDeBaseHistorique pose MapFilterLabel (le libellé de la carte de base) sur les lignes
// d'historique dont la carte appartient à un groupe. Le nom affiché (MapNameFR) ne change pas : seul
// le filtre de cartes de l'Explorateur lit MapFilterLabel.
func rattacherCartesDeBaseHistorique(ctx context.Context, pdb *PlayerDB, rules analysis.MapGroupRules, rows []domain.MatchHistoryRawRow) {
	groupes, bases, ok := groupesEtLibelles(ctx, pdb, rules)
	if !ok {
		return
	}
	for i := range rows {
		if rows[i].MapID == nil || !groupes.Groups.Regroupe(*rows[i].MapID) {
			continue
		}
		libelle := bases[groupes.Groups.Groupe(*rows[i].MapID)].FR
		rows[i].MapFilterLabel = &libelle
	}
}
