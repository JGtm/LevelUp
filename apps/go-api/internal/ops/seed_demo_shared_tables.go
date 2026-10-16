package ops

// seed_demo_shared_tables.go — la liste d'extraction shared d'un titre.
//
// `weapon_kills` n'existe que dans le fichier des titres SANS décodeur de film : la migration
// title-owned shared_drop_weapon_kills_v1 l'a retirée de Halo Infinite (l'arme d'un kill y vient
// de la source du dégât, lue dans le film), Halo 5 la garde avec ses lignes natives. Même
// critère que l'export de snapshot (snapshot_export.go, sharedSnapshotTitleOwnedRaw) : la
// capability CapReplay (« ce titre a-t-il un décodeur de film ? »), jamais un slug.

import (
	titlePkg "levelup/go-api/internal/domain/title"
)

// weaponKillsExtract : l'extraction de weapon_kills, ajoutée aux seules listes des titres qui la
// portent.
var weaponKillsExtract = extractTable{
	name: tableWeaponKills, where: matchIDInClause, identity: [][2]string{{colXUID, ""}},
}

// titleOwnsWeaponKills dit si le fichier shared du titre porte la table weapon_kills : tous les
// titres sans décodeur de film, et un titre inconnu du registre (la table est alors tentée, son
// absence source reste tolérée par copyAnonymizedTables).
func titleOwnsWeaponKills(desc *titlePkg.TitleDescriptor) bool {
	return desc == nil || !desc.HasCapability(titlePkg.CapReplay)
}

// sharedTablesFor rend la liste des tables shared à extraire pour un titre : le socle commun
// (sharedTablesWhere) plus weapon_kills quand le titre la porte. Rend une liste neuve.
func sharedTablesFor(titleSlug string) []extractTable {
	tables := append([]extractTable(nil), sharedTablesWhere...)
	if titleOwnsWeaponKills(titlePkg.DefaultRegistry().Get(titleSlug)) {
		tables = append(tables, weaponKillsExtract)
	}
	return tables
}

// sharedExtraction décrit une extraction shared : la base source, la base démo à créer et le
// titre (qui décide des tables extraites).
type sharedExtraction struct {
	srcPath, dstPath, titleSlug string
}
