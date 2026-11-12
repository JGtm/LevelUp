// Package sqlfrag — identity.go : les expressions SQL de l'identité des joueurs (noms de bots, libellé masqué,
// vue v_gamertag_lookup). La convention Go (IsBot, BotDisplayName) reste dans analysis/identity.go.
package sqlfrag

import (
	"fmt"
	"sort"
	"strings"

	"levelup/go-api/internal/analysis"
)

// BotSQLCase génère une expression SQL CASE qui mappe chaque xuid bot connu
// vers son nom officiel. xuidExpr est l'expression SQL à comparer (ex: "xa.xuid").
// Fallback : retourne xuidExpr tel quel pour les IDs inconnus.
// Source unique de vérité : analysis.BotNames().
func BotSQLCase(xuidExpr string) string {
	names := analysis.BotNames()
	keys := make([]string, 0, len(names))
	for k := range names {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	sb.WriteString("CASE ")
	sb.WriteString(xuidExpr)
	for _, k := range keys {
		fmt.Fprintf(&sb, "\n\t\t\t\t\tWHEN '%s' THEN '%s'", k, names[k])
	}
	fmt.Fprintf(&sb, "\n\t\t\t\t\tELSE %s\n\t\t\t\tEND", xuidExpr)
	return sb.String()
}

// MaskedXuidLabelSQL génère une expression SQL produisant un libellé masqué
// "Joueur ####" (4 derniers caractères du xuid) — JAMAIS le xuid brut.
// Sert de fallback final quand aucun gamertag n'est résolvable, afin que les
// colonnes d'affichage n'exposent jamais un identifiant au format xuid.
// Cohérent avec maskedPlayerLabel()/unknownPlayerLabel() côté frontend
// (apps/web/src/lib/players/displayName.ts). xuidExpr est l'expression SQL
// du xuid (ex: "p.xuid", "COALESCE(xa.xuid, mp.xuid)").
func MaskedXuidLabelSQL(xuidExpr string) string {
	return fmt.Sprintf("('Joueur ' || right(CAST(%s AS VARCHAR), 4))", xuidExpr)
}

// GamertagLookupViewSQL retourne le DDL `CREATE OR REPLACE VIEW v_gamertag_lookup`,
// SOURCE UNIQUE DE VÉRITÉ du résolveur xuid → gamertag display.
//
// Auparavant ce DDL était dupliqué (sync/schema.go en version simplifiée +
// migration/steps_shared.go en version robuste + ops/seed_demo.go), et la
// version simplifiée — recréée à chaque boot — gagnait, réintroduisant des
// xuids bruts à l'affichage. Cette fonction unifie la définition robuste.
//
// Comportement (gamertag JAMAIS NULL, JAMAIS au format xuid brut) :
//  1. xuid bot Halo connu → nom officiel (ex: "343 Meowlnir") via BotSQLCase.
//  2. sinon xuid_aliases.gamertag si non vide.
//  3. sinon match_participants.gamertag (MAX si plusieurs) si non vide.
//  4. sinon le gamertag porté par le KILL-FEED, lu dans les DEUX tables qui le
//     portent : `match_kill_events_latest` (canonique) et `killer_victim_pairs`
//     (historique). Source VIVANTE pour les adversaires : xuid_aliases a cessé
//     d'être alimenté par les events en avril 2026, mais le kill-feed porte
//     toujours le gamertag (même DB shared, donc aucun ATTACH). Couvre les
//     joueurs croisés en match mais absents d'alias.
//     ⚠ POURQUOI LES DEUX, ET PAS LA SEULE CANONIQUE (2026-08-02) : sur les
//     matchs couverts par un film, la passe de film publie moins de morts que le
//     kill-feed de l'API (74,4 % contre 98,4 % de l'oracle, mesuré). Lire la
//     canonique SEULE coûtait 4 gamertags sur 18 219 — mesuré, pas supposé — qui
//     retombaient sur « Joueur #### » sur le chemin le plus visible du produit.
//     La jambe historique part le jour où la table part. La vue `_latest` est le
//     seul chemin de lecture autorisé de la table append-only (ADR 0026).
//  5. sinon libellé masqué "Joueur ####" via MaskedXuidLabelSQL.
//
// Colonnes exposées : (xuid, gamertag). On ne projette PAS xa.last_seen : aucun
// caller ne le consomme et toutes les tables xuid_aliases ne le possèdent pas
// (ex. seed démo) → garder la vue agnostique de cette colonne optionnelle.
func GamertagLookupViewSQL() string {
	xuidExpr := "COALESCE(xa.xuid, mp.xuid, kv.xuid)"
	return fmt.Sprintf(`CREATE OR REPLACE VIEW v_gamertag_lookup AS
SELECT
	%s AS xuid,
	CASE
		WHEN %s LIKE 'bid(%%' THEN %s
		WHEN xa.gamertag IS NOT NULL AND xa.gamertag != '' THEN xa.gamertag
		WHEN mp.gamertag IS NOT NULL AND mp.gamertag != '' THEN mp.gamertag
		WHEN kv.gamertag IS NOT NULL AND kv.gamertag != '' THEN kv.gamertag
		ELSE %s
	END AS gamertag
FROM xuid_aliases xa
FULL OUTER JOIN (
	SELECT xuid, MAX(gamertag) AS gamertag
	FROM match_participants
	GROUP BY xuid
) mp ON xa.xuid = mp.xuid
FULL OUTER JOIN (
%s
) kv ON COALESCE(xa.xuid, mp.xuid) = kv.xuid`,
		xuidExpr,
		xuidExpr,
		BotSQLCase(xuidExpr),
		MaskedXuidLabelSQL(xuidExpr),
		gamertagKillFeedSQL(""),
	)
}

// gamertagKillFeedLegs : les QUATRE jambes du niveau 4 de v_gamertag_lookup — le gamertag
// porté par le kill-feed, lu dans le journal canonique (`_latest`, tueur puis victime) et dans
// la table historique (tueur puis victime). Pourquoi les deux : cf. GamertagLookupViewSQL.
//
// UNE SEULE COPIE (lot perf L2, 2026-09-23) : la vue ET l'annuaire des lectures Escouade
// (AnnuaireKillFeedSQL) les tirent d'ici. Deux copies de la cascade d'identité finissent par
// nommer différemment le même joueur — c'est l'écart que GamertagLookupViewSQL existe pour
// empêcher. Le texte est celui de la vue au caractère près : `gamertagKillFeedSQL("")` rend la
// sous-requête d'origine, le DDL de la vue n'a pas bougé d'un octet.
var gamertagKillFeedLegs = [...]string{
	"SELECT feed_killer_xuid AS xuid, feed_killer_gamertag AS gamertag\n" +
		"\t\tFROM match_kill_events_latest\n" +
		"\t\tWHERE feed_killer_xuid IS NOT NULL AND feed_killer_xuid != ''\n" +
		"\t\t  AND feed_killer_gamertag IS NOT NULL AND feed_killer_gamertag != ''",
	"SELECT victim_xuid AS xuid, victim_gamertag AS gamertag\n" +
		"\t\tFROM match_kill_events_latest\n" +
		"\t\tWHERE victim_xuid IS NOT NULL AND victim_xuid != ''\n" +
		"\t\t  AND victim_gamertag IS NOT NULL AND victim_gamertag != ''",
	"SELECT killer_xuid AS xuid, killer_gamertag AS gamertag\n" +
		"\t\tFROM killer_victim_pairs\n" +
		"\t\tWHERE killer_gamertag IS NOT NULL AND killer_gamertag != ''",
	"SELECT victim_xuid AS xuid, victim_gamertag AS gamertag\n" +
		"\t\tFROM killer_victim_pairs\n" +
		"\t\tWHERE victim_gamertag IS NOT NULL AND victim_gamertag != ''",
}

// gamertagKillFeedSQL rend la sous-requête `(xuid, MAX(gamertag))` du niveau 4. `scope` est
// ajouté au WHERE de CHAQUE jambe : vide pour la vue (toute la base), un prédicat sur
// `match_id` pour l'annuaire d'une lecture (ses seuls matchs).
func gamertagKillFeedSQL(scope string) string {
	legs := make([]string, len(gamertagKillFeedLegs))
	for i, leg := range gamertagKillFeedLegs {
		legs[i] = leg + scope
	}
	return "\tSELECT xuid, MAX(gamertag) AS gamertag\n\tFROM (\n\t\t" +
		strings.Join(legs, "\n\t\tUNION ALL\n\t\t") +
		"\n\t)\n\tGROUP BY xuid"
}
