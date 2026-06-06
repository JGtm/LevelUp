package migration

import "fmt"

// CareerRankTranslation est une ligne (rank_id, lang) de career_rank_translations.
// Source unique des libellés FR/EN des 272 rangs de carrière Halo Infinite —
// partagée par la migration seed_career_rank_translations (boot) et le CLI
// cmd/seed-rank-translations (fallback manuel). Évite la divergence des données.
type CareerRankTranslation struct {
	RankID int
	Lang   string // "fr" ou "en"
	Title  string
	Tier   string
}

// Grades militaires — 15 paliers (le grade se répète sur 3 niveaux I/II/III par tier).
var careerGradesFR = [15]string{
	"Cadet", "Soldat", "Caporal suppléant", "Caporal",
	"Sergent", "Sergent-chef", "Sergent d'artillerie", "Adjudant",
	"Lieutenant", "Capitaine", "Lieutenant-major", "Lieutenant-colonel",
	"Colonel", "Général de brigade", "Général",
}
var careerGradesEN = [15]string{
	"Cadet", "Private", "Lance Corporal", "Corporal",
	"Sergeant", "Staff Sergeant", "Gunnery Sergeant", "Master Sergeant",
	"Lieutenant", "Captain", "Major", "Lieutenant Colonel",
	"Colonel", "Brigadier General", "General",
}

// 6 tranches (Bronze→Onyx), chacune = 15 grades × 3 niveaux = 45 rangs.
var careerTiersFR = [6]string{"Bronze", "Argent", "Or", "Platine", "Diamant", "Onyx"}
var careerTiersEN = [6]string{"Bronze", "Silver", "Gold", "Platinum", "Diamond", "Onyx"}

// BuildHaloCareerRankTranslations génère les 544 lignes (272 rangs × FR/EN).
//
// Rangs : 0 = Recrue, 1–270 = grades × tiers × niveaux, 271 = Héros.
// Algorithme (1–270) : tierIdx=(i-1)/45, posInTier=(i-1)%45, gradeIdx=posInTier/3,
// level=posInTier%3+1.
func BuildHaloCareerRankTranslations() []CareerRankTranslation {
	out := make([]CareerRankTranslation, 0, 272*2)
	add := func(id int, titleFR, tierFR, titleEN, tierEN string) {
		out = append(out,
			CareerRankTranslation{RankID: id, Lang: "fr", Title: titleFR, Tier: tierFR},
			CareerRankTranslation{RankID: id, Lang: "en", Title: titleEN, Tier: tierEN},
		)
	}

	add(0, "Recrue", "Bronze", "Recruit", "Bronze")
	for i := 1; i <= 270; i++ {
		tierIdx := (i - 1) / 45
		posInTier := (i - 1) % 45
		gradeIdx := posInTier / 3
		level := posInTier%3 + 1
		add(i,
			fmt.Sprintf("%s %d", careerGradesFR[gradeIdx], level), careerTiersFR[tierIdx],
			fmt.Sprintf("%s %d", careerGradesEN[gradeIdx], level), careerTiersEN[tierIdx],
		)
	}
	add(271, "Héros", "Onyx", "Hero", "Onyx")
	return out
}
