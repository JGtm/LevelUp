// Table de catégorisation des médailles Halo 5 : medal_name_id -> {catégorie,
// super-section, tri}. Bâtie à partir du référentiel du wiki Halo (halo.fr,
// « Médailles de Halo 5 : Guardians ») croisé au catalogue medal_definitions du
// titre (215 médailles). Symétrique à halo_infinite/medal_category_table.go
// (dont la source est SpartanRecord). Les clés de catégorie/super-section sont
// STABLES : la localisation FR/EN vit dans le manifeste front (medals.toml).
//
// Rafraîchir : rejouer le classement wiki -> id sur un dump à jour de
// medal_definitions (halo_5) et régénérer ce fichier. Toute médaille absente de
// cette table retombe sur la baseline (medal_type normalisé, super-section "other").

package halo_5

// Clés de catégorie (11) et super-section (4) de la taxonomie des médailles
// Halo 5. Nommées en constantes pour éviter la duplication des littéraux dans
// la table (goconst) ; valeurs STABLES — la localisation FR/EN vit dans le
// manifeste front (medals.toml).
const (
	catMultikill   = "multikill"
	catSpree       = "spree"
	catWeapons     = "weapons"
	catVehicles    = "vehicles"
	catWarzone     = "warzone"
	catInfection   = "infection"
	catCTF         = "ctf"
	catOddball     = "oddball"
	catStrongholds = "strongholds"
	catObjective   = "objective"
	catStyle       = "style"

	secClassics         = "classics"
	secGameModes        = "game_modes"
	secWeaponsEquipment = "weapons_equipment"
	secOther            = "other"
)

// medalCategoryTable : 215 médailles Halo 5 -> {catégorie, super-section, tri}.
var medalCategoryTable = map[int64]medalCategoryEntry{
	35545941:   {catVehicles, secWeaponsEquipment, -1}, // Scorpion Destroyed
	68187090:   {catObjective, secGameModes, -1},       // Trifecta
	92444561:   {catVehicles, secWeaponsEquipment, -1}, // Warthog Destroyed
	105976035:  {catOddball, secGameModes, -1},         // Ballsassination
	121048710:  {catSpree, secClassics, 3},             // Rampage
	121303890:  {catWeapons, secWeaponsEquipment, -1},  // Rocket Kill
	125834251:  {catVehicles, secWeaponsEquipment, -1}, // Wheelman
	126799571:  {catInfection, secGameModes, -1},       // Zombie Slayer
	151853593:  {catObjective, secGameModes, -1},       // Blind Side
	164204247:  {catCTF, secGameModes, -1},             // Flag Defense
	194124164:  {catStyle, secOther, -1},               // Team Takedown
	243900335:  {catWeapons, secWeaponsEquipment, -1},  // Shotgun Kill
	247699948:  {catInfection, secGameModes, -1},       // Survived
	250435527:  {catStyle, secOther, -1},               // Brawler
	266550583:  {catSpree, secClassics, 5},             // Invincible
	271752777:  {catInfection, secGameModes, -1},       // Ravager
	281002471:  {catVehicles, secWeaponsEquipment, -1}, // Busted
	285057226:  {catStyle, secOther, -1},               // From the Grave
	298813630:  {catStyle, secOther, -1},               // Spartan Charge
	317993761:  {catMultikill, secClassics, 4},         // Killtacular
	343942800:  {catMultikill, secClassics, 8},         // Killamanjaro
	352859864:  {catStyle, secOther, -1},               // Guardian Angel
	370413844:  {catWeapons, secWeaponsEquipment, -1},  // Perfect Kill
	388928163:  {catInfection, secGameModes, -1},       // Lord of the Flies
	394618614:  {catObjective, secGameModes, -1},       // Stopped Short
	411368450:  {catStyle, secOther, -1},               // Alley-Oop
	466059351:  {catStyle, secOther, -1},               // Top Gun
	492192256:  {catStyle, secOther, -1},               // Ground Pound
	515648662:  {catVehicles, secWeaponsEquipment, -1}, // Mongoose Destroyed
	519459233:  {catWarzone, secGameModes, -1},         // Watcher Kill
	558223616:  {catInfection, secGameModes, -1},       // Hell Jumper
	565087105:  {catMultikill, secClassics, 6},         // Killtastrophe
	615190505:  {catVehicles, secWeaponsEquipment, -1}, // Mantis Destroyed
	651116095:  {catWeapons, secWeaponsEquipment, -1},  // Rocket Mary
	660765097:  {catWeapons, secWeaponsEquipment, -1},  // Caster Kill
	685960231:  {catStyle, secOther, -1},               // BXR
	690651736:  {catWeapons, secWeaponsEquipment, -1},  // Hydra Kill
	762229696:  {catVehicles, secWeaponsEquipment, -1}, // Banshee Destroyed
	764679877:  {catOddball, secGameModes, -1},         // Goal
	775545297:  {catWeapons, secWeaponsEquipment, -1},  // Sniper Kill
	786413504:  {catVehicles, secWeaponsEquipment, -1}, // Banshee Assist
	824733727:  {catStyle, secOther, -1},               // Distraction
	848240062:  {catWeapons, secWeaponsEquipment, -1},  // Sniper Headshot
	876932011:  {catWeapons, secWeaponsEquipment, -1},  // Power Player
	882267095:  {catStyle, secOther, -1},               // Spray N' Pray
	922853899:  {catInfection, secGameModes, -1},       // Plague Bearer
	979431049:  {catObjective, secGameModes, -1},       // Bifecta
	1021972251: {catStyle, secOther, -1},               // Assist
	1080468863: {catWeapons, secWeaponsEquipment, -1},  // Perfect Kill
	1096398218: {catInfection, secGameModes, -1},       // Infected
	1127532123: {catSpree, secClassics, 4},             // Untouchable
	1135730000: {catOddball, secGameModes, -1},         // Ball Holder
	1175005865: {catOddball, secGameModes, -1},         // Ball Master
	1199460731: {catWeapons, secWeaponsEquipment, -1},  // Supercombine
	1212900805: {catObjective, secGameModes, -1},       // Immortal
	1219497744: {catVehicles, secWeaponsEquipment, -1}, // Hijack
	1234371130: {catOddball, secGameModes, -1},         // Goal Defense
	1259067733: {catStyle, secOther, -1},               // Retribution
	1311847356: {catWarzone, secGameModes, -1},         // Knight Kill
	1326375333: {catStyle, secOther, -1},               // Team Takedown
	1351381581: {catStrongholds, secGameModes, -1},     // Stronghold Defense
	1423504140: {catWeapons, secWeaponsEquipment, -1},  // Longshot
	1427531503: {catObjective, secGameModes, -1},       // Goal Line Stand
	1492451766: {catMultikill, secClassics, 9},         // Killionaire
	1494478183: {catCTF, secGameModes, -1},             // Flag Capture Assist
	1524161777: {catWarzone, secGameModes, -1},         // Legendary Takedown
	1568252876: {catVehicles, secWeaponsEquipment, -1}, // Wraith Destroyed
	1573153198: {catObjective, secGameModes, -1},       // Vanquisher
	1584076385: {catWarzone, secGameModes, -1},         // Marine Kill
	1618319591: {catStyle, secOther, -1},               // Nadeshot
	1636999863: {catInfection, secGameModes, -1},       // Carrier
	1637841390: {catStrongholds, secGameModes, -1},     // Capture Assist
	1638349322: {catCTF, secGameModes, -1},             // Carrier Protected
	1655262350: {catStyle, secOther, -1},               // Pounder Assist
	1659967913: {catSpree, secClassics, 6},             // Inconceivable
	1669270602: {catCTF, secGameModes, -1},             // Flag Joust
	1691836029: {catStyle, secOther, -1},               // Hard Target
	1711392399: {catVehicles, secWeaponsEquipment, -1}, // Wasp Destroyed
	1723893404: {catWarzone, secGameModes, -1},         // Knight Assist
	1730555799: {catOddball, secGameModes, -1},         // First Touch
	1792284502: {catStyle, secOther, -1},               // Triple Double
	1795642208: {catVehicles, secWeaponsEquipment, -1}, // Phaeton Destroyed
	1801925525: {catVehicles, secWeaponsEquipment, -1}, // Skyjack
	1807727172: {catMultikill, secClassics, 5},         // Killtrocity
	1838382875: {catCTF, secGameModes, -1},             // Flag Champion
	1841827730: {catInfection, secGameModes, -1},       // Hell's Janitor
	1848890814: {catWarzone, secGameModes, -1},         // Elite Assist
	1850333312: {catStyle, secOther, -1},               // Smooth Moves
	1862589993: {catInfection, secGameModes, -1},       // Jackal Kill
	1957561936: {catCTF, secGameModes, -1},             // Carrier Protected
	1970687571: {catInfection, secGameModes, -1},       // The Cure
	1986137636: {catWeapons, secWeaponsEquipment, -1},  // Snapshot
	1997362612: {catVehicles, secWeaponsEquipment, -1}, // Flyin' High
	2006781774: {catStyle, secOther, -1},               // Airsassination
	2009141709: {catMultikill, secClassics, 0},         // Kill
	2010966644: {catOddball, secGameModes, -1},         // Ball Kill
	2016270491: {catStyle, secOther, -1},               // Showstopper
	2028249938: {catCTF, secGameModes, -1},             // Carrier Kill
	2032512673: {catMultikill, secClassics, 10},        // Extermination
	2074445293: {catWeapons, secWeaponsEquipment, -1},  // Railgun Kill
	2077162827: {catStyle, secOther, -1},               // Gun Punch
	2078758684: {catMultikill, secClassics, 1},         // Double Kill
	2093481574: {catWeapons, secWeaponsEquipment, -1},  // Snipeltaneous!
	2105198095: {catStrongholds, secGameModes, -1},     // Total Control
	2108880282: {catWeapons, secWeaponsEquipment, -1},  // Snipunch
	2137994204: {catCTF, secGameModes, -1},             // Flag Runner
	2150057864: {catStyle, secOther, -1},               // Pounder
	2155964350: {catObjective, secGameModes, -1},       // Fast Break
	2177965673: {catOddball, secGameModes, -1},         // Ball Keeper
	2189166852: {catOddball, secGameModes, -1},         // Ball Hog
	2221955973: {catOddball, secGameModes, -1},         // Ball Carrier Kill
	2237392606: {catVehicles, secWeaponsEquipment, -1}, // Wraith Assist
	2251767925: {catStyle, secOther, -1},               // Stuck
	2271298273: {catStyle, secOther, -1},               // Clutch Kill
	2279899989: {catWeapons, secWeaponsEquipment, -1},  // Perfect Kill
	2287626681: {catStyle, secOther, -1},               // Melee Kill
	2299864088: {catCTF, secGameModes, -1},             // Flag Kill
	2315448068: {catOddball, secGameModes, -1},         // Goal Offense
	2322887852: {catCTF, secGameModes, -1},             // Flag Driver
	2359847435: {catObjective, secGameModes, -1},       // Extinction
	2366628262: {catStyle, secOther, -1},               // EMP Assist
	2380717523: {catCTF, secGameModes, -1},             // Flag Return
	2421706731: {catObjective, secGameModes, -1},       // Superfecta
	2430242797: {catSpree, secClassics, 0},             // Killing Spree
	2435743433: {catWarzone, secGameModes, -1},         // Base Defense
	2462002800: {catStyle, secOther, -1},               // Combat Evolved
	2466756965: {catObjective, secGameModes, -1},       // Magic Hands
	2494364276: {catStyle, secOther, -1},               // Last Shot
	2497544753: {catVehicles, secWeaponsEquipment, -1}, // Ghost Destroyed
	2502890877: {catWeapons, secWeaponsEquipment, -1},  // Incineration Kill
	2531822079: {catStyle, secOther, -1},               // Cluster Luck
	2541276496: {catObjective, secGameModes, -1},       // Triple Threat
	2564994165: {catWeapons, secWeaponsEquipment, -1},  // Big Gun Runner
	2570365458: {catWeapons, secWeaponsEquipment, -1},  // Splaser Kill
	2615178569: {catCTF, secGameModes, -1},             // Carrier Kill
	2683910456: {catStyle, secOther, -1},               // Noob Combo
	2707871298: {catSpree, secClassics, 2},             // Running Riot
	2714887772: {catSpree, secClassics, 7},             // Unfriggenbelievable
	2715985301: {catOddball, secGameModes, -1},         // Ball Carrier Protected
	2717858990: {catInfection, secGameModes, -1},       // Infector
	2732704021: {catInfection, secGameModes, -1},       // Zombie Hunter
	2732907792: {catVehicles, secWeaponsEquipment, -1}, // Mongoose Assist
	2758521291: {catInfection, secGameModes, -1},       // Stalker
	2763748638: {catMultikill, secClassics, 2},         // Triple Kill
	2766284219: {catStyle, secOther, -1},               // Quickdraw
	2768212131: {catWeapons, secWeaponsEquipment, -1},  // SAW Kill
	2782465081: {catStyle, secOther, -1},               // Reversal
	2787661404: {catWarzone, secGameModes, -1},         // Base Capture
	2838259753: {catStyle, secOther, -1},               // Protector
	2859802775: {catVehicles, secWeaponsEquipment, -1}, // Road Trip
	2896365521: {catStrongholds, secGameModes, -1},     // Lockdown
	2916014239: {catStrongholds, secGameModes, -1},     // Stronghold Secured
	2947060439: {catWarzone, secGameModes, -1},         // Crawler Kill
	2955425834: {catWarzone, secGameModes, -1},         // Elite Kill
	2966496172: {catStyle, secOther, -1},               // Assassination
	2971193992: {catOddball, secGameModes, -1},         // First Touch
	2977773352: {catWeapons, secWeaponsEquipment, -1},  // Hat Trick
	3001183151: {catStyle, secOther, -1},               // First Strike
	3033979855: {catMultikill, secClassics, 7},         // Killpocalypse
	3098362934: {catWeapons, secWeaponsEquipment, -1},  // Perfect Kill
	3148489433: {catStyle, secOther, -1},               // Starkiller
	3261908037: {catWeapons, secWeaponsEquipment, -1},  // Headshot
	3270120991: {catStyle, secOther, -1},               // Beat Down
	3318955876: {catStyle, secOther, -1},               // Game Saver
	3324603383: {catWarzone, secGameModes, -1},         // Grunt Kill
	3336271203: {catInfection, secGameModes, -1},       // Zombicide
	3344008582: {catWarzone, secGameModes, -1},         // Soldier Kill
	3344421840: {catWeapons, secWeaponsEquipment, -1},  // Grenade Kill
	3354395650: {catStrongholds, secGameModes, -1},     // Capture Spree
	3392925967: {catVehicles, secWeaponsEquipment, -1}, // Scorpion Assist
	3400287617: {catStyle, secOther, -1},               // Close Call
	3406254250: {catWarzone, secGameModes, -1},         // Soldier Assist
	3419325174: {catInfection, secGameModes, -1},       // Ancient One
	3440416044: {catVehicles, secWeaponsEquipment, -1}, // Mantis Assist
	3485523253: {catOddball, secGameModes, -1},         // Ball Champion
	3486286344: {catWeapons, secWeaponsEquipment, -1},  // Airborne Snapshot!
	3491849182: {catStyle, secOther, -1},               // Hail Mary
	3505254118: {catWarzone, secGameModes, -1},         // Core Defense
	3522125871: {catSpree, secClassics, 1},             // Killing Frenzy
	3565441934: {catOddball, secGameModes, -1},         // Ball Clear
	3565443938: {catStrongholds, secGameModes, -1},     // Stronghold Captured
	3592822316: {catSpree, secClassics, 8},             // Perfection
	3601671677: {catStyle, secOther, -1},               // Sayonara
	3653057799: {catWeapons, secWeaponsEquipment, -1},  // Perfect Kill
	3676723563: {catVehicles, secWeaponsEquipment, -1}, // Splatter
	3698887726: {catStyle, secOther, -1},               // Cliffhanger
	3703205413: {catWarzone, secGameModes, -1},         // Mythic Takedown
	3710519250: {catMultikill, secClassics, 3},         // Overkill
	3712378932: {catWarzone, secGameModes, -1},         // Marine Assist
	3718365815: {catVehicles, secWeaponsEquipment, -1}, // Phaeton Assist
	3744028405: {catStyle, secOther, -1},               // Wingman
	3786961025: {catStyle, secOther, -1},               // Fastball
	3809690136: {catWeapons, secWeaponsEquipment, -1},  // Scattershot Kill
	3824002610: {catVehicles, secWeaponsEquipment, -1}, // Ghost Assist
	3865042769: {catVehicles, secWeaponsEquipment, -1}, // Wasp Assist
	3881314877: {catOddball, secGameModes, -1},         // Ball Runner
	3886151616: {catCTF, secGameModes, -1},             // Flag Capture
	3894006667: {catObjective, secGameModes, -1},       // Stealth Capture
	3904379564: {catInfection, secGameModes, -1},       // Last Man Standing
	3910677154: {catOddball, secGameModes, -1},         // Oddball Kill
	3915864153: {catVehicles, secWeaponsEquipment, -1}, // Pineapple Express
	3925170236: {catStyle, secOther, -1},               // Big Game Hunter
	3972445431: {catStyle, secOther, -1},               // Bodyguard
	3992195104: {catWeapons, secWeaponsEquipment, -1},  // Perfect Kill
	4048801324: {catVehicles, secWeaponsEquipment, -1}, // Warthog Assist
	4100216166: {catWeapons, secWeaponsEquipment, -1},  // Binary Kill
	4116576170: {catCTF, secGameModes, -1},             // Flagsassination
	4122142678: {catInfection, secGameModes, -1},       // Resourceful
	4122622964: {catWeapons, secWeaponsEquipment, -1},  // Beam Kill
	4162659350: {catObjective, secGameModes, -1},       // Buzzer Beater
	4164420745: {catWarzone, secGameModes, -1},         // Boss Takedown
	4179143479: {catWeapons, secWeaponsEquipment, -1},  // Sword Kill
	4204396686: {catVehicles, secWeaponsEquipment, -1}, // Buckle Up
	4252521258: {catStyle, secOther, -1},               // Bulltrue
	4258092236: {catInfection, secGameModes, -1},       // Flatline
}
