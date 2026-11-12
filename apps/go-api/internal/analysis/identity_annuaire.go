// Package analysis — identity_annuaire.go : L'ANNUAIRE DES NOMS D'UNE LECTURE, et la seule
// règle qui en tire un nom d'affichage.
//
// # POURQUOI CE FICHIER EXISTE (lot perf L2, 2026-09-23)
//
// Les lectures de la page Escouade joignaient `v_gamertag_lookup` pour nommer leurs xuids. La
// vue agrège `match_participants` GROUP BY, deux passes de `match_kill_events_latest` et deux
// de `killer_victim_pairs` en FULL OUTER JOIN : aucun filtre n'y est poussé, elle est
// matérialisée EN ENTIER à chaque jointure — 3 s par évaluation sur la base de production,
// six évaluations par page (mesure du 2026-09-23).
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
