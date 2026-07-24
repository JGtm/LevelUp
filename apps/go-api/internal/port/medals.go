package port

import (
	"context"

	"levelup/go-api/internal/domain"
)

// MedalCategoryResolver résout la catégorie, la super-section et l'ordre de tri
// d'une médaille pour un titre donné. Title-agnostic : l'implémentation baseline
// (regroupement natif par medal_type) sert TOUS les titres ; un titre peut
// enregistrer un resolver enrichi (ex. Halo Infinite : table SpartanRecord).
//
// Les clés retournées sont STABLES (la localisation FR/EN est faite côté front).
type MedalCategoryResolver interface {
	// Resolve retourne (clé de catégorie, clé de super-section, ordre de tri) pour
	// une médaille. medalType/difficultyIndex proviennent de medal_definitions du
	// titre — un resolver enrichi les ignore pour les IDs qu'il connaît et retombe
	// sur la baseline (medalType, "other", difficultyIndex) sinon.
	Resolve(medalID int64, medalType string, difficultyIndex int) (categoryKey, superSectionKey string, sort int)
}

// MedalsService construit la réponse de la page Médailles (catalogue complet du
// titre + compteur obtenu par joueur).
type MedalsService interface {
	GetMedalsPage(ctx context.Context, playerXUID string) (*domain.MedalsPageResponse, error)
}

// MedalsRepository fournit les données de la page Médailles.
// Implémenté par platform/duckdb.MedalsRepo.
type MedalsRepository interface {
	// ListAllMedals retourne TOUT le catalogue medal_definitions du titre courant,
	// labels/descriptions résolus locale-aware. Base de l'affichage 0/N (médailles
	// jamais obtenues incluses).
	ListAllMedals(ctx context.Context, locale string) ([]domain.MedalCatalogRow, error)

	// LoadMedalTotals charge les totaux de médailles obtenus par un joueur depuis
	// shared.medals_earned (Q36a). Réutilise la constante SQL partagée.
	LoadMedalTotals(ctx context.Context, xuid string) ([]domain.MedalEarnedRow, error)
}
