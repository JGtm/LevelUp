// Package analysis — identity.go : helpers pour identifier les joueurs.
//
// Centralise la convention IsBot (revue 2026-04-29 axe 6 BLOQUANT) :
// auparavant 1 helper Go + 8 fragments SQL `xuid LIKE 'bid(%'` répétés. Le
// commit du jour (engagement B2) a corrigé 4 bugs causés exactement par cette
// duplication.
//
// Pour le filtrage SQL, utiliser `analysis.SQLIsBotCol(col)` (cf. sql_fragments.go).
package analysis

import (
	"fmt"
	"sort"
	"strings"
)

// botXUIDPrefix est le préfixe d'identifiant utilisé par Halo Infinite pour
// les bots de matchmaking. Documenté empiriquement (cf. xuid_aliases en DB).
const botXUIDPrefix = "bid("

// botNames mappe les xuid officiels de bots Halo Infinite à leurs noms d'affichage.
// Source : src/config.py BOT_MAP (branche main Python, synchronisé 2026-05-08).
var botNames = map[string]string{
	"bid(0.0)":  "343 Ritzy",
	"bid(1.0)":  "343 Meowlnir",
	"bid(2.0)":  "343 Wiggle Cat",
	"bid(3.0)":  "343 Ellis",
	"bid(4.0)":  "343 Godfather",
	"bid(5.0)":  "343 Colson",
	"bid(6.0)":  "343 Donos",
	"bid(7.0)":  "343 PardonMy",
	"bid(8.0)":  "343 Ensrude",
	"bid(9.0)":  "343 TooMilks",
	"bid(10.0)": "343 Beard",
	"bid(11.0)": "343 Razzle",
	"bid(12.0)": "343 Nando",
	"bid(13.0)": "343 GrappleMans",
	"bid(14.0)": "343 Zero",
	"bid(15.0)": "343 Mak",
	"bid(16.0)": "343 Hundy",
	"bid(17.0)": "343 Lacuna",
	"bid(18.0)": "343 Cream Corn",
	"bid(19.0)": "343 Brew Dog",
	"bid(20.0)": "343 Lemondade",
	"bid(21.0)": "343 Flippant",
	"bid(22.0)": "343 Connmando",
	"bid(23.0)": "343 Byrontron",
	"bid(24.0)": "343 Ham Sammich",
	"bid(25.0)": "343 BoboGan",
	"bid(26.0)": "343 Robot Hoida",
	"bid(27.0)": "343 Mumblebee",
	"bid(28.0)": "343 Tedosaur",
	"bid(29.0)": "343 Cliffton",
	"bid(30.0)": "343 Stone",
	"bid(31.0)": "343 Marmot",
	"bid(32.0)": "343 Mickey",
	"bid(33.0)": "343 Bachici",
	"bid(34.0)": "343 Ben Desk",
	"bid(35.0)": "343 BF Scrub",
	"bid(36.0)": "343 Chaco",
	"bid(37.0)": "343 Total Ten",
	"bid(38.0)": "343 Darkstar",
	"bid(39.0)": "343 Aloysius",
	"bid(40.0)": "343 Dazzle",
	"bid(41.0)": "343 Rhinosaurus",
	"bid(42.0)": "343 Doomfruit",
	"bid(43.0)": "343 Cosmo",
	"bid(44.0)": "343 Sandwolf",
	"bid(45.0)": "343 O Freruner",
	"bid(46.0)": "343 Bergerton",
	"bid(47.0)": "343 SpaceCase",
	"bid(48.0)": "343 KaleDucky",
	"bid(49.0)": "343 BERRYHILL",
	"bid(50.0)": "343 Oscar",
	"bid(51.0)": "343 The Referee",
	"bid(52.0)": "343 Free Money",
	"bid(53.0)": "343 Kubly",
	"bid(54.0)": "343 Tanuki",
	"bid(55.0)": "343 Shady Seal",
	"bid(56.0)": "343 Chilies",
	"bid(57.0)": "343 The Thumb",
	"bid(58.0)": "343 Forge Lord",
	"bid(59.0)": "343 Hollis",
}

// IsBot retourne true si l'identifiant xuid correspond à un bot de match
// (préfixe `bid(`). Convention propre au sync Halo Infinite mais
// suffisamment universelle pour rester en analysis/ (les services qui
// agrègent doivent exclure les bots).
func IsBot(xuid string) bool {
	return strings.HasPrefix(xuid, botXUIDPrefix)
}

// BotDisplayName retourne le nom d'affichage officiel d'un bot depuis son xuid.
// "bid(1.0)" → "343 Meowlnir". Retourne le xuid tel quel si l'ID est inconnu.
func BotDisplayName(xuid string) string {
	// Tolérance : certains xuid historiques n'ont pas la parenthèse fermante.
	key := xuid
	if strings.HasPrefix(key, botXUIDPrefix) && !strings.HasSuffix(key, ")") {
		key = key + ")"
	}
	if name, ok := botNames[key]; ok {
		return name
	}
	return xuid
}

// BotSQLCase génère une expression SQL CASE qui mappe chaque xuid bot connu
// vers son nom officiel. xuidExpr est l'expression SQL à comparer (ex: "xa.xuid").
// Fallback : retourne xuidExpr tel quel pour les IDs inconnus.
// Source unique de vérité : la map botNames de ce fichier.
func BotSQLCase(xuidExpr string) string {
	keys := make([]string, 0, len(botNames))
	for k := range botNames {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	sb.WriteString("CASE ")
	sb.WriteString(xuidExpr)
	for _, k := range keys {
		fmt.Fprintf(&sb, "\n\t\t\t\t\tWHEN '%s' THEN '%s'", k, botNames[k])
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
//  4. sinon killer_victim_pairs.{killer,victim}_gamertag — gamertag résolu
//     depuis les films (kill-feed). Source VIVANTE pour les adversaires :
//     xuid_aliases a cessé d'être alimenté par les events en avril 2026, mais
//     killer_victim_pairs porte toujours le gamertag (même DB shared, donc
//     aucun ATTACH). Couvre les joueurs croisés en match mais absents d'alias.
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
	SELECT xuid, MAX(gamertag) AS gamertag
	FROM (
		SELECT killer_xuid AS xuid, killer_gamertag AS gamertag
		FROM killer_victim_pairs
		WHERE killer_gamertag IS NOT NULL AND killer_gamertag != ''
		UNION ALL
		SELECT victim_xuid AS xuid, victim_gamertag AS gamertag
		FROM killer_victim_pairs
		WHERE victim_gamertag IS NOT NULL AND victim_gamertag != ''
	)
	GROUP BY xuid
) kv ON COALESCE(xa.xuid, mp.xuid) = kv.xuid`,
		xuidExpr,
		xuidExpr,
		BotSQLCase(xuidExpr),
		MaskedXuidLabelSQL(xuidExpr),
	)
}
