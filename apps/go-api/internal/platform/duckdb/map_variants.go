package duckdb

// map_variants.go — le rattachement d'une carte VARIANTE à sa carte de base, pour les filtres.
//
// « Thunderhead Heavies » et « Live Fire - Ranked » sont des cartes distinctes au registre des matchs
// (identifiant propre) alors que c'est le même terrain que « Thunderhead » et « Live Fire ». Les
// filtres de cartes (le pipeline de filtres et l'Explorateur) les regroupent sous la carte de base :
// la recherche d'un nom ramène aussi les parties jouées sur ses variantes. Les identifiants ne
// bougent pas, et les agrégats par carte restent par identifiant.
//
// La règle de rattachement est analysis.BaseMapName, nourrie des suffixes du titre
// (mappings/map_variants.toml) ; la carte de base est cherchée au catalogue (maps_catalog) par son
// nom EN, son libellé FR est celui de libelleDeCarte (map_labels.go). Une variante sans carte de
// base au catalogue reste seule : rien n'est deviné.

import (
	"context"
	"log/slog"
	"strings"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

// carteDeBase : la carte de base d'une variante, avec son nom EN et son libellé de filtre.
type carteDeBase struct {
	EN string
	FR string
}

// basesDeCartes rend, pour chaque carte de `nomsEN` (map_id -> nom EN du registre) qui est la
// variante d'une carte présente au catalogue, sa carte de base (map_id de la variante -> base).
// Les cartes qui ne sont pas des variantes, ou dont la base est absente du catalogue, n'y figurent
// pas. Sans suffixe, sans metadata ou sans variante : nil. Un échec de lecture est journalisé puis
// dégradé : les variantes restent alors seules dans les filtres.
func basesDeCartes(ctx context.Context, pdb *PlayerDB, nomsEN map[string]string, suffixes []string) map[string]carteDeBase {
	if len(suffixes) == 0 || pdb == nil || pdb.Metadata == nil || len(nomsEN) == 0 {
		return nil
	}
	baseENParVariante := make(map[string]string)
	nomsDeBase := make(map[string]struct{})
	for id, en := range nomsEN {
		if base, ok := analysis.BaseMapName(en, suffixes); ok {
			baseENParVariante[id] = base
			nomsDeBase[base] = struct{}{}
		}
	}
	if len(nomsDeBase) == 0 {
		return nil
	}

	idParNomDeBase := idsDeCartesParNom(ctx, pdb, nomsDeBase)
	if len(idParNomDeBase) == 0 {
		slog.DebugContext(ctx, "cartes : variantes sans carte de base au catalogue", "variantes", len(baseENParVariante))
		return nil
	}
	idsDeBase := make([]string, 0, len(idParNomDeBase))
	for _, id := range idParNomDeBase {
		idsDeBase = append(idsDeBase, id)
	}
	traductions := traductionsDeCartes(ctx, pdb.Metadata, idsDeBase)

	bases := make(map[string]carteDeBase, len(baseENParVariante))
	for id, baseEN := range baseENParVariante {
		baseID, ok := idParNomDeBase[baseEN]
		if !ok {
			continue
		}
		bases[id] = carteDeBase{EN: baseEN, FR: libelleDeCarte("", baseEN, traductions[baseID])}
	}
	return bases
}

// idsDeCartesParNom lit au catalogue (maps_catalog) l'identifiant de chaque carte de `noms` (nom EN
// canonique -> map_asset_id). À nom égal, l'identifiant le plus petit l'emporte, pour un résultat
// stable. Table absente ou échec : le second cas est journalisé et consigné (bestEffortFailed) ; rend
// nil.
func idsDeCartesParNom(ctx context.Context, pdb *PlayerDB, noms map[string]struct{}) map[string]string {
	args := make([]any, 0, len(noms)+1)
	args = append(args, resolveMediaTitleSlug(pdb.TitleSlug))
	for nom := range noms {
		args = append(args, nom)
	}
	q := `SELECT name_canonical, map_asset_id FROM maps_catalog
		WHERE title_slug = ? AND name_canonical IN (` + Placeholders(len(noms)) + `)
		ORDER BY name_canonical, map_asset_id`
	rows, err := pdb.Metadata.QueryRecovered(ctx, q, args...)
	if err != nil {
		bestEffortFailed(ctx, "map_bases", err)
		return nil
	}
	defer rows.Close()

	ids := make(map[string]string, len(noms))
	for rows.Next() {
		var nom, id string
		if err := rows.Scan(&nom, &id); err != nil {
			bestEffortFailed(ctx, "map_bases", err)
			return nil
		}
		if _, deja := ids[nom]; !deja && strings.TrimSpace(id) != "" {
			ids[nom] = id
		}
	}
	if err := rows.Err(); err != nil {
		bestEffortFailed(ctx, "map_bases", err)
	}
	return ids
}

// rattacherCartesDeBase réécrit le nom de carte (EN et FR) des lignes de filtres dont la carte est la
// variante d'une carte de base : les options et le filtrage, qui lisent ces noms, ne voient plus que
// la carte de base. Le MapID reste celui de la variante.
func rattacherCartesDeBase(ctx context.Context, pdb *PlayerDB, suffixes []string, rows []domain.FilterMatchRow) {
	noms := make(map[string]string)
	for _, row := range rows {
		if row.MapID != nil && row.MapName != nil && *row.MapID != "" {
			noms[*row.MapID] = *row.MapName
		}
	}
	bases := basesDeCartes(ctx, pdb, noms, suffixes)
	if len(bases) == 0 {
		return
	}
	for i := range rows {
		if rows[i].MapID == nil {
			continue
		}
		if base, ok := bases[*rows[i].MapID]; ok {
			en, fr := base.EN, base.FR
			rows[i].MapName = &en
			rows[i].MapNameFR = &fr
		}
	}
}

// rattacherCartesDeBaseHistorique pose MapFilterLabel (le libellé de la carte de base) sur les lignes
// d'historique dont la carte est une variante. Le nom affiché (MapNameFR) ne change pas : seul le
// filtre de cartes de l'Explorateur lit MapFilterLabel.
func rattacherCartesDeBaseHistorique(ctx context.Context, pdb *PlayerDB, suffixes []string, rows []domain.MatchHistoryRawRow) {
	noms := make(map[string]string)
	for _, row := range rows {
		if row.MapID != nil && row.MapName != nil && *row.MapID != "" {
			noms[*row.MapID] = *row.MapName
		}
	}
	bases := basesDeCartes(ctx, pdb, noms, suffixes)
	if len(bases) == 0 {
		return
	}
	for i := range rows {
		if rows[i].MapID == nil {
			continue
		}
		if base, ok := bases[*rows[i].MapID]; ok {
			libelle := base.FR
			rows[i].MapFilterLabel = &libelle
		}
	}
}
