// Package domain — achievement_categories_halo_infinite.go : mapping en dur des
// 144 succès Halo Infinite (119 d'origine + 25 Winter Update) vers leur catégorie.
//
// Sources :
//   - répartition multijoueur/campagne/autres : guide Steam 2682508379
//     (sections "Multiplayer Achievements" parts 1-2 vs sections campagne vs
//     Customization + Tutorial & Theatre)
//   - les 25 succès Winter Update (ids 120-145, co-op campagne et défis de
//     mission) sont tous campagne
//   - orthographes réconciliées avec xbox_achievement_definitions
//     (metadata.duckdb) le 2026-06-11 : "Run Rabbit, Run", "All-Seeing I",
//     "Bring Shiela Home Safely" (sic)
package domain

var haloInfiniteAchievementCategories = buildAchievementCategoryTable(
	haloInfiniteMultiplayerAchievements,
	haloInfiniteCampaignAchievements,
	haloInfiniteOtherAchievements,
)

// buildAchievementCategoryTable normalise les noms et assemble la table de
// lookup. "Together. Again?" et "Together. Again." partagent la même clé
// normalisée — sans impact, les deux sont campagne.
func buildAchievementCategoryTable(multiplayer, campaign, other []string) map[string]AchievementCategory {
	table := make(map[string]AchievementCategory, len(multiplayer)+len(campaign)+len(other))
	add := func(names []string, cat AchievementCategory) {
		for _, n := range names {
			table[normalizeAchievementName(n)] = cat
		}
	}
	add(multiplayer, AchievementCategoryMultiplayer)
	add(campaign, AchievementCategoryCampaign)
	add(other, AchievementCategoryOther)
	return table
}

// haloInfiniteMultiplayerAchievements — 34 succès matchmaking/MP.
var haloInfiniteMultiplayerAchievements = []string{
	"Clocking In",
	"We Have a Job For You",
	"Limited Addition",
	"Humble Beginnings",
	"Battle Tested",
	"You're Up, Rook'",
	"All About the Grind",
	"Peak Performance",
	"Brutality",
	"Slaying with Style",
	"Peeker's Disadvantage",
	"Running Laps",
	"Kebab",
	"They See Me Rollin'",
	"Control Freak",
	"Customary",
	"Back to the Chopper",
	"Zone Ranger",
	"Bomb Returned",
	"A Fellow of Infinite Jest",
	"Multi-class Racer",
	"Natural Formation Location Sensation",
	"Sick Burn",
	"Working Remote",
	"New Kid on the Block",
	"Enemies Everywhere!",
	"One Shot, Top Mid",
	"Secret Stash",
	"Watt Say You?",
	"Straight to the Bank",
	"Do You Even Gift?",
	"Skyhook Shot",
	"Party Bus",
	"MEDIC!",
}

// haloInfiniteCampaignAchievements — 94 succès campagne :
// 16 généraux + 9 aptitudes Spartan + 16 monde ouvert + 28 missions
// + 25 Winter Update (co-op campagne).
var haloInfiniteCampaignAchievements = []string{
	// Campagne — général
	"Mjolnir Master",
	"Armory Amore",
	"Haruspis",
	"Know Your Enemy",
	"Catacomb",
	"Off the Air",
	"Canon Collector",
	"Rubicon Protocol",
	"Set a Fire in Your Heart",
	"Bare Your Fangs",
	"Fight Hard, Die Well",
	"A True Test of Legends",
	"Forza Veloce",
	"Headmaster",
	"Wait, I Can Throw Those?",
	"Wanna Have a Catch?",
	// Aptitudes Spartan
	"Getting Defensive",
	"Reaching Out",
	"Those Wonderful Toys",
	"Thrusters On Full",
	"Run Rabbit, Run",
	"Big Brother",
	"All-Seeing I",
	"Impervious",
	"Aegis Fate",
	// Monde ouvert
	"Who is Max Valor?",
	"Reclaimer",
	"Resurgency",
	"We're On Our Way",
	"Wars with Friends",
	"Outpost Discovery",
	"Bunker Buster",
	"Please Shut Up",
	"Headhunter",
	"Bloodstars' Bane",
	"No One Left Behind",
	"Eld Aficionado",
	"Passing the Gas",
	"Whip-Riding the Ghost",
	"Takes One to Make One",
	"Nosebleed",
	// Missions 01-15
	"Infinity Down",
	"Headstrong",
	"Two Sides to Every Story",
	"First Contact",
	"Dispatches From the Front",
	"Together. Again?",
	"Ascension",
	"Hidden Experience",
	"Money in the Bank",
	"Zeta",
	"Visionary",
	"Fallen",
	"Unearthed",
	"Grab Some Cover",
	"Evasive Maneuvers",
	"Hunter. Killer.",
	"Pelican Down",
	"One Down…",
	"Gun Runner",
	"Brothers Grim",
	"Light the Way",
	"What Will It Take?",
	"Hear These Words!",
	"Together. Again.",
	"Bring Shiela Home Safely",
	"Reckoning",
	"Legends",
	"Too Many Goodbyes",
	// Winter Update (co-op campagne + défis de mission)
	"Mix Things Up",
	"Stick Around",
	"Rapid Unscheduled Disassembly",
	"Out with a Bang",
	"Workplace Safety Violation",
	"Conservation of Momentum",
	"It Really Does Beat Everything",
	"Vintage Fisticuffs",
	"Spire Stalker",
	"Turnabout is Fair Play",
	"More Than He Bargained For",
	"Gatecrasher",
	"What's Rightfully Ours",
	"Wardens of Zeta",
	"First Responders",
	"Hunting Party",
	"Air Raid",
	"Cow Catcher",
	"Gruesome Twosome",
	"Keep It Steady",
	"Rolling Thunder",
	"Inseparable",
	"You, Me, Same Page",
	"Controlled Demolition",
	"Wolves at the Doors",
}

// haloInfiniteOtherAchievements — 16 succès hors matchs et hors campagne :
// customisation, Académie (tutoriel/entraînement) et Théâtre.
var haloInfiniteOtherAchievements = []string{
	"Which One of Us is the Machine?",
	"Passion for Fashion",
	"\"Need a Weapon?\"",
	"That Thing on the Left is the Brake",
	"Reporting for Duty",
	"I'm Ready, How 'Bout You?",
	"Just the Two of Us",
	"Greased Lightning",
	"Make a Little More Noise",
	"Sharpshooter",
	"Deadeye",
	"Augmented",
	"Doing Your Part",
	"Getting Strong Now",
	"Sparring Partners",
	"Get the Popcorn",
}
