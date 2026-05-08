// Package analysis — identity.go : helpers pour identifier les joueurs.
//
// Centralise la convention IsBot (revue 2026-04-29 axe 6 BLOQUANT) :
// auparavant 1 helper Go + 8 fragments SQL `xuid LIKE 'bid(%'` répétés. Le
// commit du jour (engagement B2) a corrigé 4 bugs causés exactement par cette
// duplication.
//
// Pour le filtrage SQL, utiliser `analysis.SQLIsBot` (cf. sql_fragments.go).
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
