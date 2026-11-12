// Package sqlfrag — annuaire.go : les gabarits SQL de l'annuaire des noms d'une lecture. La cascade Go qui
// en tire un nom d'affichage (AnnuaireGamertags.Resolve) reste dans analysis/identity_annuaire.go, dont
// l'en-tête décrit les niveaux.
package sqlfrag

// AnnuaireKillFeedJambes : le nombre de jambes du niveau 4 — les match_id d'AnnuaireKillFeedSQL
// se lient une fois par jambe.
const AnnuaireKillFeedJambes = len(gamertagKillFeedLegs)

// Niveaux rendus par AnnuaireNomsSQL (première colonne).
const (
	AnnuaireNiveauAlias       = "alias"
	AnnuaireNiveauParticipant = "participant"
)

// LES LISTES SE LIENT EN UN PARAMÈTRE (item B.7 du plan perf « compaction et périmètre joueur »,
// 2026-09-27) : chaque liste de xuids ou de match_id des gabarits ci-dessous est UN paramètre
// `VARCHAR[]` (sql_liste.go), plus un `IN (?, ?, …)` — chez J900004, les Relations liaient jusqu'à
// 17 741 paramètres par requête. Les xuids et les match_id filtrés sur des TABLES passent par la
// semi-jointure (SQLDansListeParJointure) ; les match_id qui bornent une vue `_latest` du kill-feed
// par la constante (SQLDansListe), la seule forme qui descende sous sa fenêtre. L'argument lié
// est la liste elle-même, un `[]string`.

// AnnuaireNomsSQL rend la lecture des niveaux 2 et 3 de la vue pour les xuids d'une lecture :
// l'alias, puis le MAX(match_participants.gamertag) SUR LES MATCHS DE LA LECTURE (décision D2.1
// du plan perf : « les mêmes matchs »). Colonnes : (niveau, xuid, gamertag), niveau valant
// AnnuaireNiveauAlias ou AnnuaireNiveauParticipant ; seuls les noms NON VIDES sortent — la vue
// saute les vides de la même façon.
//
// Paramètres (trois listes), dans l'ordre : les xuids, les xuids, les match_id.
func AnnuaireNomsSQL() string {
	return annuaireNomsSQL(" AND " + SQLDansListeParJointure("match_id"))
}

// AnnuaireNomsBaseSQL rend les niveaux 2 et 3 de la vue en « portée base » (lot A du plan perf
// « lectures par périmètre », décision DA.3, 2026-09-26) : le MAX(match_participants.gamertag)
// sur TOUTE la base, comme la vue, filtré par les seuls xuids de la lecture — un prédicat sur une
// TABLE, qui se pousse. Mêmes colonnes qu'AnnuaireNomsSQL. Paramètres (deux listes) : les xuids,
// les xuids.
func AnnuaireNomsBaseSQL() string {
	return annuaireNomsSQL("")
}

// annuaireNomsSQL : le texte commun des deux portées ; `matchs` restreint les participants
// (vide : toute la base).
func annuaireNomsSQL(matchs string) string {
	return "SELECT '" + AnnuaireNiveauAlias + "' AS niveau, xuid, gamertag\n" +
		"FROM xuid_aliases\n" +
		"WHERE " + SQLDansListeParJointure("xuid") + " AND gamertag IS NOT NULL AND gamertag != ''\n" +
		"UNION ALL\n" +
		"SELECT '" + AnnuaireNiveauParticipant + "' AS niveau, xuid, MAX(gamertag) AS gamertag\n" +
		"FROM match_participants\n" +
		"WHERE " + SQLDansListeParJointure("xuid") + matchs + "\n" +
		"  AND gamertag IS NOT NULL AND gamertag != ''\n" +
		"GROUP BY xuid"
}

// AnnuaireKillFeedSQL rend le niveau 4 de la vue — la MÊME sous-requête (gamertagKillFeedSQL),
// chaque jambe restreinte aux matchs de la lecture — pour les xuids demandés. Colonnes :
// (xuid, gamertag). Le filtre sur `match_id` est la CONSTANTE (SQLDansListe) : il passe SOUS la
// fenêtre de la vue `_latest` (sa clé de partition), seuls les matchs de la lecture sont lus.
//
// Paramètres, dans l'ordre : la liste des match_id une fois PAR JAMBE (AnnuaireKillFeedJambes
// fois), puis la liste des xuids.
func AnnuaireKillFeedSQL() string {
	return "SELECT xuid, gamertag FROM (\n" +
		gamertagKillFeedSQL("\n\t\t  AND "+SQLDansListe("match_id")) +
		"\n) kf\nWHERE " + SQLDansListeParJointure("xuid")
}

// AnnuaireKillFeedLocaliserSQL rend le PAS 1 du repli « portée base » (décision DA.10 du plan perf
// « lectures par périmètre », 2026-09-26) : les matchs CANDIDATS où un xuid encore sans nom a porté
// un gamertag de kill-feed, lus dans la table BRUTE `match_kill_events` et dans
// `killer_victim_pairs`, prédicats poussés sur les tables. LECTURE DE LOCALISATION (ADR 0036,
// « locating read » ; ADR 0026) : elle ne rend QUE des `match_id`, jamais une valeur. Toute version
// d'une ligne qui a porté le xuid désigne son match : c'est un sur-ensemble des matchs où la vue
// `_latest` le montre. Le PAS 2 relit ces seuls matchs par AnnuaireKillFeedSQL — la jambe de la vue,
// fenêtres `_latest` bornées par match_id — : les noms viennent de `_latest`, la parité avec la vue
// est exacte par construction, sans évaluer la fenêtre du journal canonique entière.
//
// Paramètre : la liste des xuids, UNE fois (dépliée en lignes dans `cherches`, item B.7).
func AnnuaireKillFeedLocaliserSQL() string {
	return "WITH cherches(xuid) AS (" + SQLListeEnLignes() + ")\n" +
		"SELECT match_id FROM match_kill_events\n" +
		"WHERE feed_killer_xuid IN (SELECT xuid FROM cherches)\n" +
		"  AND feed_killer_gamertag IS NOT NULL AND feed_killer_gamertag != ''\n" +
		"UNION SELECT match_id FROM match_kill_events\n" +
		"WHERE victim_xuid IN (SELECT xuid FROM cherches)\n" +
		"  AND victim_gamertag IS NOT NULL AND victim_gamertag != ''\n" +
		"UNION SELECT match_id FROM killer_victim_pairs\n" +
		"WHERE killer_xuid IN (SELECT xuid FROM cherches)\n" +
		"  AND killer_gamertag IS NOT NULL AND killer_gamertag != ''\n" +
		"UNION SELECT match_id FROM killer_victim_pairs\n" +
		"WHERE victim_xuid IN (SELECT xuid FROM cherches)\n" +
		"  AND victim_gamertag IS NOT NULL AND victim_gamertag != ''"
}
