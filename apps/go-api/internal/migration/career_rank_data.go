package migration

// career_rank_data.go — contrat de table + seam provider pour les libellés de
// rangs de carrière (MT-07).
//
// La donnée Halo-spécifique (15 grades × 6 tiers × algorithme 272 rangs) a été
// déplacée vers internal/games/halo_infinite/migrations (title-owned), ce package
// title-agnostique ne conserve que :
//   - le STRUCT CareerRankTranslation (contrat de la table career_rank_translations,
//     consommé par internal/ops.SeedRankTranslations) ;
//   - un seam provider mirror de SetTitleStepsProvider : le câblage qui importe le
//     package de titre pose la source des lignes au boot.
//
// Provider non posé (nil) ⇒ CareerRankTranslationRows() retourne nil → le seeder
// écrit 0 ligne (dégradation gracieuse identique au contrat Ranks()==vide).

// CareerRankTranslation est une ligne (rank_id, lang) de career_rank_translations.
// Contrat title-agnostique de la table : produit par le générateur title-owned,
// consommé par le seeder (internal/ops) et le CLI cmd/seed-rank-translations.
type CareerRankTranslation struct {
	RankID int
	Lang   string // "fr" ou "en"
	Title  string
	Tier   string
}

// careerRankTranslationsProvider fournit les lignes de rangs du titre courant.
// Posé une fois au boot via SetCareerRankTranslationsProvider (par le câblage qui
// importe le package de titre). nil = aucune donnée (le seeder n'écrit rien).
var careerRankTranslationsProvider func() []CareerRankTranslation

// SetCareerRankTranslationsProvider enregistre la source des libellés de rangs.
// À appeler au boot, avant tout SeedRankTranslations. Idempotent (dernier gagne).
func SetCareerRankTranslationsProvider(p func() []CareerRankTranslation) {
	careerRankTranslationsProvider = p
}

// CareerRankTranslationRows retourne les lignes du provider, ou nil si non posé.
// Source canonique unique pour le seeder et les CLI (plus de générateur Halo ici).
func CareerRankTranslationRows() []CareerRankTranslation {
	if careerRankTranslationsProvider == nil {
		return nil
	}
	return careerRankTranslationsProvider()
}
