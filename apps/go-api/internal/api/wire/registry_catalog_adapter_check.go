// Package api — registry_catalog_adapter_check.go : test « ce titre a-t-il un
// catalog adapter discovery-infiniteugc résolvable ? ».
//
// C'est le gate RÉEL (sémantiquement précis) injecté dans les crons catalog_refresh
// et asset_name_sweep, en remplacement du proxy CapForge (Forge ≠ catalogue UGC). Il
// construit l'adapter par le MÊME chemin que le drain (rules TOML
// config/titles/<slug>/catalog/experience_rules.toml + halo_infinite.NewCatalogAdapter),
// l'enregistre dans un resolver éphémère, puis teste la résolvabilité exactement
// comme un consommateur le ferait :
//
//	_, err := resolver.Catalog(slug); err == nil
//
// HINF → experience_rules.toml présent → adapter construit → Catalog(slug) == nil
// → true (drain/sweep lancés). H5 → pas de experience_rules.toml → NewCatalogAdapter
// échoue → resolver vide → Catalog(slug) == ErrTitleNotResolved → false (skip propre).
//
// Zéro réseau : le fetcher n'est pas requis pour CONSTRUIRE l'adapter (il n'est
// utilisé qu'au fetch). On passe donc nil — on ne teste que la résolvabilité.
package wire

import (
	"path/filepath"

	"levelup/go-api/internal/games"
	halo_games "levelup/go-api/internal/games/halo_infinite"
)

// catalogExperienceRulesPath retourne le chemin du TOML de règles d'experience du
// catalogue d'un titre. Source unique partagée par le drain (RunCatalogUGCDrain) et
// le test de présence d'adapter (HasCatalogAdapter) — évite la dérive de chemin.
func (r *ServiceRegistry) catalogExperienceRulesPath(titleSlug string) string {
	return filepath.Join(r.cfg.RepoRoot, "config", "titles", titleSlug, "catalog", "experience_rules.toml")
}

// HasCatalogAdapter répond à la VRAIE question du gate des crons catalogue : ce titre
// a-t-il un catalog adapter discovery-infiniteugc résolvable ? Construit l'adapter par
// le même chemin que le drain, l'enregistre dans un resolver éphémère et teste la
// résolvabilité via resolver.Catalog(slug). Ne fait AUCUN appel réseau (fetcher nil :
// non requis pour la construction). Pas d'erreur remontée : un titre sans adapter (ou
// une config absente/invalide) → false, c'est l'issue attendue, pas un incident.
//
// Branché sur les crons via scheduler.*Cron.WithCatalogAdapterCheck(reg.HasCatalogAdapter).
func (r *ServiceRegistry) HasCatalogAdapter(titleSlug string) bool {
	if r == nil || r.cfg == nil || titleSlug == "" {
		return false
	}
	rulesPath := r.catalogExperienceRulesPath(titleSlug)
	adapter, err := halo_games.NewCatalogAdapter(nil, rulesPath)
	if err != nil {
		// Pas de règles d'experience résolvables pour ce titre (ex. Halo 5 → catalogue
		// metadata-side). Ce n'est PAS une erreur : le titre n'a simplement pas
		// d'adapter UGC discovery-infiniteugc.
		return false
	}
	resolver := games.NewStaticResolver(titleSlug)
	resolver.RegisterCatalog(adapter)
	// Test de résolvabilité identique à un consommateur réel (forme canonique du
	// gate) : resolveErr == nil ⇔ un adapter est enregistré POUR CE slug ; sinon
	// games.ErrTitleNotResolved (titre sans catalogue UGC → skip propre côté cron).
	//
	// Sécurité supplémentaire : RegisterCatalog indexe par adapter.TitleSlug() (le
	// CatalogAdapter Infinite renvoie toujours "halo_infinite"). Un experience_rules.toml
	// égaré sous le dossier d'un AUTRE titre ne le ferait donc PAS résoudre pour ce
	// titre (Catalog(otherSlug) == ErrTitleNotResolved) — le gate reste juste.
	_, resolveErr := resolver.Catalog(titleSlug)
	return resolveErr == nil
}
