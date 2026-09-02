// Package medalname — IDENTITE DES MEDAILLES LUES DANS LE FILM HALO INFINITE.
//
// CE QUE CE PAQUET RESOUT. Le chunk highlight du film porte, pour chaque medaille,
// un couple d octets : `type_hint` (b[47] du bloc event) et `medal_type` (b[59]) —
// jamais le nom. Le nom anglais est la CLE DE REFERENTIEL du reste de la chaine :
// `highlight_events.raw_json.medal_name` -> `medal_definitions` -> label localise,
// image, medal_name_id (cf. service/match_view_killfeed_medals.go). Sans lui, un
// event medal n a pas d identite et le fil des eliminations ne l affiche pas.
//
// CE N EST PAS UN LIBELLE D INTERFACE. Le nom rendu ici est une clef anglaise, au
// meme titre que le `medal_name` qu ecrivait l ancien collecteur. La locale reste
// resolue a la LECTURE par `medal_definitions` — rien de traduit ici.
//
// D OU VIENT LA TABLE. Mesuree le 2026-09-02 sur le corpus des matchs anterieurs a
// avril 2026, seuls a porter un `raw_json` complet (ere Python) :
// 44 568 events medal, 124 couples distincts, 124 noms distincts, ZERO couple
// ambigu — la correspondance (type_hint, medal_type) -> nom est une BIJECTION.
// Requete de rafraichissement (DuckDB, shared_matches_v2, lecture seule) :
//
//	SELECT CAST(json_extract(raw_json,'$.type_hint')   AS INTEGER) AS type_hint,
//	       CAST(json_extract(raw_json,'$.medal_value') AS INTEGER) AS medal_type,
//	       json_extract_string(raw_json,'$.medal_name')            AS medal_name,
//	       COUNT(*) AS occurrences
//	FROM highlight_events
//	WHERE event_type='medal' AND raw_json IS NOT NULL
//	GROUP BY type_hint, medal_type, medal_name
//	ORDER BY type_hint, medal_type
//
// La sortie de cette requete est figee dans testdata/corpus_medailles_2026-09-02.tsv :
// le test de garde-rail rejoue chaque ligne contre la table. Elargir la table = ajouter
// la ligne mesuree au TSV ET a la table, jamais l une sans l autre.
//
// CE CORPUS EST CLOS, et il faut le savoir avant de croire pouvoir le rafraichir. Seuls
// les documents `raw_json` de l ere Python portaient les TROIS champs (type_hint,
// medal_value, medal_name) ; ceux qu ecrit le collecteur Go depuis le 2026-09-02 ne
// portent que `medal_name` — `type_hint` a sa propre colonne et `medal_type` n est
// stocke nulle part. La requete ci-dessus ne rendra donc jamais que les lignes
// anterieures a avril 2026. Une medaille apparue depuis ne peut pas s apprendre de nos
// propres donnees : elle se signale par le compteur de couples inconnus, et son nom se
// prend a une source externe.
//
// DEGRADATION. Un couple absent de la table ne rend AUCUN nom (jamais un voisin) :
// l event medal est persiste sans `raw_json`, exactement comme aujourd hui, et le
// collecteur compte le trou.
package medalname

// medalKey est le couple d octets qui identifie une medaille dans le bloc event du
// chunk highlight : type_hint = b[47], medalType = b[59] (cf.
// analysis.HighlightEvent.TypeHint / .MedalType).
type medalKey struct {
	typeHint  int
	medalType int
}

// table : 124 couples mesures -> nom anglais de la medaille (clef de referentiel).
var table = map[medalKey]string{
	{typeHint: 50, medalType: 26}:   "Killjoy",
	{typeHint: 50, medalType: 36}:   "Stopped Short",
	{typeHint: 50, medalType: 37}:   "Flag Joust",
	{typeHint: 50, medalType: 62}:   "Spotter",
	{typeHint: 50, medalType: 63}:   "Treasure Hunter",
	{typeHint: 50, medalType: 64}:   "Saboteur",
	{typeHint: 50, medalType: 65}:   "Wingman",
	{typeHint: 50, medalType: 66}:   "Wheelman",
	{typeHint: 50, medalType: 67}:   "Gunner",
	{typeHint: 50, medalType: 68}:   "Driver",
	{typeHint: 50, medalType: 69}:   "Pilot",
	{typeHint: 50, medalType: 70}:   "Tanker",
	{typeHint: 50, medalType: 71}:   "Rifleman",
	{typeHint: 50, medalType: 72}:   "Bomber",
	{typeHint: 50, medalType: 73}:   "Grenadier",
	{typeHint: 50, medalType: 74}:   "Boxer",
	{typeHint: 50, medalType: 75}:   "Warrior",
	{typeHint: 50, medalType: 76}:   "Gunslinger",
	{typeHint: 50, medalType: 77}:   "Scattergunner",
	{typeHint: 50, medalType: 78}:   "Sharpshooter",
	{typeHint: 50, medalType: 79}:   "Marksman",
	{typeHint: 50, medalType: 80}:   "Heavy",
	{typeHint: 50, medalType: 81}:   "Bodyguard",
	{typeHint: 50, medalType: 82}:   "Back Smack",
	{typeHint: 50, medalType: 87}:   "Dogfight",
	{typeHint: 50, medalType: 88}:   "Harpoon",
	{typeHint: 50, medalType: 91}:   "Odin's Raven",
	{typeHint: 50, medalType: 97}:   "Skyjack",
	{typeHint: 50, medalType: 98}:   "Stick",
	{typeHint: 50, medalType: 101}:  "Kong",
	{typeHint: 50, medalType: 105}:  "Reversal",
	{typeHint: 50, medalType: 108}:  "Snipe",
	{typeHint: 50, medalType: 117}:  "Guardian Angel",
	{typeHint: 50, medalType: 120}:  "Chain Reaction",
	{typeHint: 50, medalType: 126}:  "Flyin' High",
	{typeHint: 50, medalType: 127}:  "From the Grave",
	{typeHint: 50, medalType: 131}:  "Last Shot",
	{typeHint: 50, medalType: 133}:  "Mount Up",
	{typeHint: 50, medalType: 135}:  "Quick Draw",
	{typeHint: 50, medalType: 139}:  "Reclaimer",
	{typeHint: 50, medalType: 142}:  "Special Delivery",
	{typeHint: 50, medalType: 151}:  "Always Rotating",
	{typeHint: 50, medalType: 156}:  "Splatter",
	{typeHint: 50, medalType: 157}:  "Clash of Kings",
	{typeHint: 50, medalType: 160}:  "Watch the Throne",
	{typeHint: 50, medalType: 165}:  "Breacher",
	{typeHint: 51, medalType: 178}:  "Hang Up",
	{typeHint: 52, medalType: 179}:  "Call Blocked",
	{typeHint: 100, medalType: 0}:   "Double Kill",
	{typeHint: 100, medalType: 9}:   "Killing Spree",
	{typeHint: 100, medalType: 38}:  "Goal Line Stand",
	{typeHint: 100, medalType: 85}:  "Bulltrue",
	{typeHint: 100, medalType: 86}:  "Cluster Luck",
	{typeHint: 100, medalType: 89}:  "Mind the Gap",
	{typeHint: 100, medalType: 92}:  "Pancake",
	{typeHint: 100, medalType: 96}:  "Rideshare",
	{typeHint: 100, medalType: 99}:  "Tag & Bag",
	{typeHint: 100, medalType: 100}: "Whiplash",
	{typeHint: 100, medalType: 104}: "Windshield Wiper",
	{typeHint: 100, medalType: 106}: "Hail Mary",
	{typeHint: 100, medalType: 107}: "Nade Shot",
	{typeHint: 100, medalType: 109}: "Perfect",
	{typeHint: 100, medalType: 110}: "Bank Shot",
	{typeHint: 100, medalType: 111}: "Fire & Forget",
	{typeHint: 100, medalType: 112}: "Ballista",
	{typeHint: 100, medalType: 113}: "Pull",
	{typeHint: 100, medalType: 114}: "No Scope",
	{typeHint: 100, medalType: 119}: "Death Race",
	{typeHint: 100, medalType: 128}: "From the Void",
	{typeHint: 100, medalType: 129}: "Grapple-jack",
	{typeHint: 100, medalType: 130}: "Hold This",
	{typeHint: 100, medalType: 132}: "Lawnmower",
	{typeHint: 100, medalType: 134}: "Off the Rack",
	{typeHint: 100, medalType: 137}: "Pineapple Express",
	{typeHint: 100, medalType: 138}: "Ramming Speed",
	{typeHint: 100, medalType: 140}: "Shot Caller",
	{typeHint: 100, medalType: 141}: "Yard Sale",
	{typeHint: 100, medalType: 146}: "Fumble",
	{typeHint: 100, medalType: 150}: "Big Deal",
	{typeHint: 100, medalType: 152}: "Hill Guardian",
	{typeHint: 100, medalType: 153}: "Clock Stop",
	{typeHint: 100, medalType: 166}: "Mounted & Loaded",
	{typeHint: 100, medalType: 168}: "Counter-snipe",
	{typeHint: 100, medalType: 174}: "Driving Spree",
	{typeHint: 101, medalType: 154}: "Secure Line",
	{typeHint: 101, medalType: 180}: "Clear Reception",
	{typeHint: 150, medalType: 1}:   "Triple Kill",
	{typeHint: 150, medalType: 10}:  "Killing Frenzy",
	{typeHint: 150, medalType: 31}:  "Flawless Victory",
	{typeHint: 150, medalType: 32}:  "Steaktacular",
	{typeHint: 150, medalType: 84}:  "Boom Block",
	{typeHint: 150, medalType: 95}:  "Return to Sender",
	{typeHint: 150, medalType: 102}: "Autopilot Engaged",
	{typeHint: 150, medalType: 103}: "Sneak King",
	{typeHint: 150, medalType: 115}: "Achilles Spine",
	{typeHint: 150, medalType: 116}: "Grand Slam",
	{typeHint: 150, medalType: 118}: "Interlinked",
	{typeHint: 150, medalType: 121}: "360",
	{typeHint: 150, medalType: 122}: "Combat Evolved",
	{typeHint: 150, medalType: 123}: "Deadly Catch",
	{typeHint: 150, medalType: 143}: "Street Sweeper",
	{typeHint: 150, medalType: 148}: "Straight Balling",
	{typeHint: 150, medalType: 158}: "Contract Killer",
	{typeHint: 150, medalType: 162}: "All That Juice",
	{typeHint: 150, medalType: 175}: "Death Cabbie",
	{typeHint: 150, medalType: 177}: "Blind Fire",
	{typeHint: 200, medalType: 13}:  "Perfection",
	{typeHint: 200, medalType: 44}:  "Extermination",
	{typeHint: 200, medalType: 90}:  "Ninja",
	{typeHint: 200, medalType: 93}:  "Quigley",
	{typeHint: 200, medalType: 94}:  "Remote Detonation",
	{typeHint: 200, medalType: 125}: "Fastball",
	{typeHint: 205, medalType: 11}:  "Running Riot",
	{typeHint: 210, medalType: 12}:  "Rampage",
	{typeHint: 220, medalType: 2}:   "Overkill",
	{typeHint: 220, medalType: 27}:  "Nightmare",
	{typeHint: 225, medalType: 3}:   "Killtacular",
	{typeHint: 230, medalType: 4}:   "Killtrocity",
	{typeHint: 230, medalType: 28}:  "Boogeyman",
	{typeHint: 235, medalType: 5}:   "Killamanjaro",
	{typeHint: 240, medalType: 6}:   "Killtastrophe",
	{typeHint: 240, medalType: 29}:  "Grim Reaper",
	{typeHint: 245, medalType: 7}:   "Killpocalypse",
	{typeHint: 250, medalType: 30}:  "Demon",
}

// Lookup rend le nom anglais de la medaille designee par le couple du film, et
// `false` si le couple est inconnu de la table mesuree. Un couple inconnu N EST PAS
// une erreur : c est une medaille jamais observee sur le corpus (medaille recente,
// mode inedit). L appelant persiste alors l event sans identite plutot que d inventer
// un nom voisin.
func Lookup(typeHint, medalType int) (string, bool) {
	name, ok := table[medalKey{typeHint: typeHint, medalType: medalType}]
	return name, ok
}

// Len rend le nombre de couples connus. Sert au test de completude et aux compteurs
// de diagnostic.
func Len() int { return len(table) }
