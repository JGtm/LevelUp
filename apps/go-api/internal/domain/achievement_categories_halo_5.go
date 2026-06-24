// Package domain — achievement_categories_halo_5.go : mapping en dur des 73 succès
// Halo 5: Guardians vers leur catégorie produit (multiplayer / campaign / other).
//
// Architecture RÉUTILISÉE telle quelle depuis Halo Infinite : même registre
// keyé par slug (achievementCategoriesByTitle), même buildAchievementCategoryTable,
// même lookup par name_en normalisé. Seules les listes diffèrent (title-agnostic).
//
// Sources : liste complète des succès Halo 5 (TrueAchievements + Halopedia),
// catégorisée le 2026-06-24. Répartition : 55 campagne, 13 multijoueur, 5 autres.
// Halo 5 étant fortement orienté campagne (mission-completion + collectibles +
// Score Attack), la majorité tombe en campagne ; le multijoueur couvre Arena,
// Warzone et Warzone Firefight ; "other" = customisation Spartan + spectate.
package domain

var halo5AchievementCategories = buildAchievementCategoryTable(
	halo5MultiplayerAchievements,
	halo5CampaignAchievements,
	halo5OtherAchievements,
)

// halo5MultiplayerAchievements — 13 succès Arena / Warzone / Warzone Firefight.
var halo5MultiplayerAchievements = []string{
	"Bringing in the Big Guns",
	"Go for the Gold",
	"Cry Havoc",
	"Top of the Food Chain",
	"Warlord",
	"Castle Crasher",
	"Off to the Races",
	"Flag Monger",
	"Spartan Decimation",
	"Valor Recognized",
	"Not Your First Rodeo",
	"Tour of Duty",
	"Dangerous Game",
}

// halo5CampaignAchievements — 55 succès campagne (missions, difficultés,
// collectibles Intel/cranes, défis de mission co-op, Score Attack).
var halo5CampaignAchievements = []string{
	"Into the Fire",
	"Argent Moon",
	"Glasslands",
	"Roots of the Earth",
	"Stolen Gauntlet",
	"Escape",
	"Together Again",
	"Swords",
	"Old Bones",
	"Breakthrough",
	"Stormbound",
	"Civil War",
	"Reclamation",
	"A New Dawn",
	"Sentinels",
	"Legacy",
	"Heroes Rise",
	"Forging a Legend",
	"Lone Wolf",
	"My Rules",
	"Conspiracy Theory",
	"Hunt the Truth",
	"Gravedigger",
	"Gravelord",
	"One for All",
	"All for One",
	"On My Mark",
	"Maverick",
	"Preying Mantis",
	"Your Team is Your Weapon",
	"Enemy of my Enemy",
	"I Thought I'd Lost You",
	"Going the Distance",
	"Shoot from the Hip",
	"Waiting on You",
	"Savior",
	"Fire Drill",
	"Take a Hike",
	"No Witnesses",
	"No Knock Raid",
	"Kraken Lackin'",
	"Emergency Boarding Procedures",
	"Death from Above",
	"Worms Don't Surf",
	"Tank Still Beats Everything",
	"Rolling Thunder",
	"Harbinger",
	"Prison Break",
	"Icy Cool",
	"Double Stuff",
	"Drop A Quarter",
	"Up For A Challenge",
	"Party Hearty",
	"The Hare",
	"The Tortoise",
}

// halo5OtherAchievements — 5 succès transverses (customisation Spartan,
// création de partie personnalisée, spectate).
var halo5OtherAchievements = []string{
	"Your Style",
	"Raise Your Banner",
	"Make Your Mark",
	"Gamemaster",
	"Benchwarmer",
}
