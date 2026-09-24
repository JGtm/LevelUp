// Package analysis — identity_annuaire.go : L'ANNUAIRE DES NOMS D'UNE LECTURE, et la seule
// règle qui en tire un nom d'affichage.
//
// # POURQUOI CE FICHIER EXISTE (lot perf L2, 2026-09-23)
//
// Les lectures de la page Escouade joignaient `v_gamertag_lookup` pour nommer leurs xuids. La
// vue agrège `match_participants` GROUP BY, deux passes de `match_kill_events_latest` et deux
// de `killer_victim_pairs` en FULL OUTER JOIN : aucun filtre n'y est poussé, elle est
// matérialisée EN ENTIER à chaque jointure — 3 s par évaluation sur la base de production,
// six évaluations par page (mesure du 2026-09-23, .ai/ETAT_DES_LIEUX_PERF_CHARGEMENTS).
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

// AnnuaireNomsSQL rend la lecture des niveaux 2 et 3 de la vue pour les xuids d'une lecture :
// l'alias, puis le MAX(match_participants.gamertag) SUR LES MATCHS DE LA LECTURE (décision D2.1
// du plan perf : « les mêmes matchs »). Colonnes : (niveau, xuid, gamertag), niveau valant
// AnnuaireNiveauAlias ou AnnuaireNiveauParticipant ; seuls les noms NON VIDES sortent — la vue
// saute les vides de la même façon.
//
// `xuids` et `matchs` sont des listes de paramètres liés (« ?, ?, ? »). Paramètres, dans
// l'ordre : les xuids, les xuids, les match_id.
func AnnuaireNomsSQL(xuids, matchs string) string {
	return "SELECT '" + AnnuaireNiveauAlias + "' AS niveau, xuid, gamertag\n" +
		"FROM xuid_aliases\n" +
		"WHERE xuid IN (" + xuids + ") AND gamertag IS NOT NULL AND gamertag != ''\n" +
		"UNION ALL\n" +
		"SELECT '" + AnnuaireNiveauParticipant + "' AS niveau, xuid, MAX(gamertag) AS gamertag\n" +
		"FROM match_participants\n" +
		"WHERE xuid IN (" + xuids + ") AND match_id IN (" + matchs + ")\n" +
		"  AND gamertag IS NOT NULL AND gamertag != ''\n" +
		"GROUP BY xuid"
}

// AnnuaireKillFeedSQL rend le niveau 4 de la vue — la MÊME sous-requête (gamertagKillFeedSQL),
// chaque jambe restreinte aux matchs de la lecture — pour les xuids demandés. Colonnes :
// (xuid, gamertag). Le filtre sur `match_id` passe SOUS la fenêtre de la vue `_latest` (sa
// clé de partition) : seuls les matchs de la lecture sont lus.
//
// Paramètres, dans l'ordre : les match_id une fois PAR JAMBE (AnnuaireKillFeedJambes fois),
// puis les xuids.
func AnnuaireKillFeedSQL(xuids, matchs string) string {
	return "SELECT xuid, gamertag FROM (\n" +
		gamertagKillFeedSQL("\n\t\t  AND match_id IN ("+matchs+")") +
		"\n) kf\nWHERE xuid IN (" + xuids + ")"
}
