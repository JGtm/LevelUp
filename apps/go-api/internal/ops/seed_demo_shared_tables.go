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

// filmDerivedTables : les tables de la base partagée que le DÉCODAGE DU FILM écrit (dérivations
// post-rangement et passes du collecteur de kills), et que lisent les pages de la vue match
// (onglet Emprise, objectifs, carte de chaleur), de la page Tactique, de l'Escouade, des
// Sessions et des Séries temporelles. Sans elles la démo ne porte que les rejeux figés (lus dans
// l'artefact) : tout ce que les pages lisent en base y est vide ou « non mesuré ».
//
// Elles sont copiées TELLES QUELLES (colonnes techniques `id`, `written_at`, `decode_pass`
// comprises), comme match_objective_stats et kill_positions : ces tables sont créées DIRECTEMENT
// en forme append-only, la migration rejouée après la copie ne leur ajoute rien, et la vue
// `<table>_latest` — l'unique chemin de lecture, règle ART n°2 — trie sur ces colonnes pour
// retenir la dernière passe entière par match. Chaque table est filtrée sur le corpus.
//
// Chaque colonne qui porte un xuid est déclarée ici (UNE SEULE LISTE, cf. extractTable) ;
// TestSharedTablesDeclareEveryIdentityColumn relit le schéma migré et échoue sur une colonne
// `*xuid*` ou `*gamertag*` non déclarée. Aucune de ces tables ne porte de gamertag, et leurs
// documents JSON (pad_pickups_json, taken_json, details…) ne nomment aucun joueur — le contrôle
// des valeurs de fin de seed (verifyDemoAnonymization) le vérifie sur la base produite.
//
// Un titre sans décodeur de film porte les mêmes tables vides (schéma partagé) ; une table
// absente de la source est ignorée (copyAnonymizedTables).
var filmDerivedTables = []extractTable{
	// Vies d'un joueur et contexte de ses morts (Emprise : « vies près d'un coéquipier »,
	// Tactique : isolement, coéquipiers à la mort), placement des vies (Escouade).
	{name: "match_lives", where: matchIDInClause, identity: [][2]string{{colXUID, ""}}},
	{name: "match_death_context", where: matchIDInClause, identity: [][2]string{{colVictimXUID, ""}}},
	{name: "match_death_mates", where: matchIDInClause, identity: [][2]string{
		{colVictimXUID, ""}, {"mate_xuid", ""}}},
	{name: "match_life_placement", where: matchIDInClause, identity: [][2]string{{colXUID, ""}}},
	// Positions un temps-pour-tuer avant le coup fatal (portée des armes à l'entame).
	{name: "kill_openings", where: matchIDInClause, identity: [][2]string{{colKillerXUID, ""}}},
	// Résumé d'usage (contrôle des armes, bonus, équipement) : un document par film, une ligne
	// par joueur, et les prises de socle par niveau.
	{name: "match_usage_films", where: matchIDInClause},
	{name: "match_usage_players", where: matchIDInClause, identity: [][2]string{{colXUID, ""}}},
	{name: "match_pad_pickups_by_tier", where: matchIDInClause, identity: [][2]string{{colXUID, ""}}},
	// Ressource véhicules de l'Emprise.
	{name: "match_vehicle_takes", where: matchIDInClause, identity: [][2]string{{colXUID, ""}}},
	// Objectifs reconstruits du film : drapeaux nets, bombe d'Assaut, faits datés (+ leurs
	// joueurs) de la chronologie des objectifs. Positions de la carte de chaleur (niveau match,
	// sans joueur).
	{name: "match_flag_grabs_net", where: matchIDInClause, identity: [][2]string{{colXUID, ""}}},
	{name: tableMatchBombStats, where: matchIDInClause, identity: [][2]string{{colXUID, ""}}},
	{name: "match_objective_events", where: matchIDInClause},
	{name: "match_objective_event_players", where: matchIDInClause, identity: [][2]string{{colXUID, ""}}},
	{name: "match_player_positions", where: matchIDInClause},
}

// killSummaryTables : les résumés par match du journal des morts (ADR 0039), écrits avec chaque
// passe de `match_kill_events` par les deux titres : la tête, les frags par paire et source, les
// appuis par paire. Copiés comme le journal (colonnes techniques comprises, même filtre, mêmes
// correspondances d'identité), ils restent cohérents avec lui dans la démo. Les gamertags de
// `match_assist_pairs` sont appariés à leur xuid, comme dans le journal.
var killSummaryTables = []extractTable{
	{name: tableMatchKillHeads, where: matchIDInClause},
	{name: tableMatchKillPairs, where: matchIDInClause, identity: [][2]string{
		{colFeedKillerXUID, ""}, {colVictimXUID, ""}}},
	{name: tableMatchAssistPairs, where: matchIDInClause, identity: [][2]string{
		{colAssistXUID, colAssistGamertag}, {colFeedKillerXUID, colFeedKillerGamertag},
		{colVictimXUID, ""}}},
}

// Les tables de résumé et les gamertags appariés du journal, nommés une fois (snapshot, seed, tests).
const (
	tableMatchKillHeads   = "match_kill_heads"
	tableMatchKillPairs   = "match_kill_pairs"
	tableMatchAssistPairs = "match_assist_pairs"
	colFeedKillerGamertag = "feed_killer_gamertag"
	colAssistGamertag     = "assist_gamertag"
	colAssistXUID         = "assist_xuid"
)

// squadOffsetExtract : player_squad_offset (synergies de coéquipiers du LUSR, lues par l'Escouade),
// append-only (vue _latest), clé xuid et NON match_id : filtrée sur les COUPLES de joueurs qui
// ont partagé un match du corpus. Les deux colonnes sont remappées, et tout couple dont un
// membre est hors corpus est écarté — un xuid hors roster ne serait pas remappé.
var squadOffsetExtract = extractTable{
	name: "player_squad_offset",
	where: `(xuid, partner_xuid) IN (SELECT a.xuid, b.xuid FROM src.match_participants a
		JOIN src.match_participants b ON a.match_id = b.match_id WHERE a.match_id IN (%s))`,
	identity: [][2]string{{colXUID, ""}, {"partner_xuid", ""}},
}

// titleOwnsWeaponKills dit si le fichier shared du titre porte la table weapon_kills : tous les
// titres sans décodeur de film, et un titre inconnu du registre (la table est alors tentée, son
// absence source reste tolérée par copyAnonymizedTables).
func titleOwnsWeaponKills(desc *titlePkg.TitleDescriptor) bool {
	return desc == nil || !desc.HasCapability(titlePkg.CapReplay)
}

// sharedTablesFor rend la liste des tables shared à extraire pour un titre : le socle commun
// (sharedTablesWhere), les tables dérivées du film (filmDerivedTables), les synergies
// d'escouade (squadOffsetExtract), plus weapon_kills quand
// le titre la porte. Rend une liste neuve.
func sharedTablesFor(titleSlug string) []extractTable {
	tables := append([]extractTable(nil), sharedTablesWhere...)
	tables = append(tables, filmDerivedTables...)
	tables = append(tables, killSummaryTables...)
	tables = append(tables, squadOffsetExtract)
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
