// Package analysis — identity_annuaire.go : L'ANNUAIRE DES NOMS D'UNE LECTURE, et la seule
// règle qui en tire un nom d'affichage.
//
// # POURQUOI CE FICHIER EXISTE (lot perf L2, 2026-09-23)
//
// Les lectures de la page Escouade joignaient `v_gamertag_lookup` pour nommer leurs xuids. La
// vue agrège `match_participants` GROUP BY, deux passes de `match_kill_events_latest` et deux
// de `killer_victim_pairs` en FULL OUTER JOIN : aucun filtre n'y est poussé, elle est
// matérialisée EN ENTIER à chaque jointure — 3 s par évaluation sur la base de production,
// six évaluations par page (mesure du 2026-09-23, .ai/V7.5/ETAT_DES_LIEUX_PERF_CHARGEMENTS_2026-09-23.md).
//
// Le lecteur charge désormais, pour les seuls xuids qu'il a rencontrés et sur les seuls matchs
// qu'il lit, les noms que chaque niveau de la vue leur donnerait (AnnuaireGamertags), puis
// applique ICI la cascade de la vue. La source des noms ne change pas : ce sont les mêmes
// tables, dans le même ordre, avec les mêmes replis.
//
// # LA CASCADE, NIVEAU PAR NIVEAU (celle de GamertagLookupViewSQL)
//
//  1. xuid de bot (`bid(`) : le nom officiel s'il est connu, SINON LE XUID TEL QUEL — c'est la
//     branche ELSE de BotSQLCase, recopiée à l'identique (clé EXACTE, sans la tolérance de
//     parenthèse de BotDisplayName, que le SQL n'a pas).
//  2. xuid_aliases.gamertag non vide.
//  3. match_participants.gamertag (MAX) non vide.
//  4. le gamertag du kill-feed (MAX) non vide.
//  5. le libellé masqué « Joueur #### » (MaskedXuidLabel, miroir de MaskedXuidLabelSQL).
//
// Un xuid qu'aucune source ne nomme rend le libellé masqué — exactement ce que rendait le
// `COALESCE(vg.gamertag, 'Joueur ' || RIGHT(xuid, 4))` des lecteurs quand la jointure ne
// trouvait rien.
//
// # DEUX PORTÉES, UNE CASCADE (lot A du plan perf « lectures par périmètre », 2026-09-26)
//
// Portée LECTURE (Escouade, Carrière, Comparer) : participants et kill-feed sur les matchs de la
// lecture (AnnuaireNomsSQL, AnnuaireKillFeedSQL). Portée BASE (vue match, événements de match,
// Relations) : alias et participants sur toute la base (AnnuaireNomsBaseSQL, la sémantique MAX de
// la vue), kill-feed d'abord sur les matchs de la lecture puis, pour les seuls xuids encore sans
// nom, sur les matchs candidats que localise AnnuaireKillFeedLocaliserSQL (DA.10 : table brute,
// match_id seulement), relus par AnnuaireKillFeedSQL. Les deux portées alimentent la même
// AnnuaireGamertags et la même Resolve.
package analysis

// AnnuaireGamertags porte, par xuid, le nom que chaque niveau NOMMÉ de la vue canonique
// donne (niveaux 2 à 4 ; le niveau 1 et le niveau 5 se déduisent du xuid seul). Une entrée
// absente ou vide veut dire « ce niveau ne nomme pas ce xuid ». Les cartes peuvent être nil.
type AnnuaireGamertags struct {
	// Alias : xuid_aliases.gamertag.
	Alias map[string]string
	// Participants : MAX(match_participants.gamertag).
	Participants map[string]string
	// KillFeed : MAX du gamertag porté par le kill-feed (journal canonique et historique).
	KillFeed map[string]string
}

// Resolve rend le nom d'affichage d'un xuid selon la cascade de la vue canonique (cf. en-tête).
// Jamais vide pour un xuid non vide, jamais un xuid brut hors bot inconnu (branche ELSE de
// BotSQLCase, conservée telle quelle).
func (a AnnuaireGamertags) Resolve(xuid string) string {
	if IsBot(xuid) {
		if name, ok := botNames[xuid]; ok {
			return name
		}
		return xuid
	}
	for _, source := range []map[string]string{a.Alias, a.Participants, a.KillFeed} {
		if name := source[xuid]; name != "" {
			return name
		}
	}
	return MaskedXuidLabel(xuid)
}

// MaskedXuidLabel est le miroir Go de MaskedXuidLabelSQL : « Joueur » suivi des quatre
// DERNIERS caractères du xuid (le xuid entier s'il en a moins), comptés en caractères comme
// le `right()` de DuckDB.
func MaskedXuidLabel(xuid string) string {
	runes := []rune(xuid)
	if len(runes) > 4 {
		runes = runes[len(runes)-4:]
	}
	return "Joueur " + string(runes)
}

// AnnuaireKillFeedJambes : le nombre de jambes du niveau 4 — les match_id d'AnnuaireKillFeedSQL
// se lient une fois par jambe.
const AnnuaireKillFeedJambes = len(gamertagKillFeedLegs)

// Niveaux rendus par AnnuaireNomsSQL (première colonne).
const (
	AnnuaireNiveauAlias       = "alias"
	AnnuaireNiveauParticipant = "participant"
)

// Nomme dit si un NIVEAU NOMMÉ de la cascade donne un nom au xuid : un bot (niveau 1, son nom
// officiel ou son xuid tel quel, comme la branche ELSE de BotSQLCase), un alias, un participant
// ou le kill-feed. Faux quand Resolve rendrait le libellé masqué : c'est la frontière du port
// GamertagResolver, dont la carte ne porte que les xuids nommés.
func (a AnnuaireGamertags) Nomme(xuid string) bool {
	if IsBot(xuid) {
		return true
	}
	return a.Alias[xuid] != "" || a.Participants[xuid] != "" || a.KillFeed[xuid] != ""
}

// LES LISTES SE LIENT EN UN PARAMÈTRE (item B.7 du plan perf « compaction et périmètre joueur »,
// 2026-09-27) : chaque liste de xuids ou de match_id des gabarits ci-dessous est UN paramètre
// `VARCHAR[]` (sql_liste.go), plus un `IN (?, ?, …)` — chez Nuzzles, les Relations liaient jusqu'à
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
