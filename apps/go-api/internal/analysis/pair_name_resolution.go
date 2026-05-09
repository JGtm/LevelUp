// Package analysis — pair_name_resolution.go : résolution canonique du libellé
// FR d'une paire mode/map à partir des sources hétérogènes (match_registry,
// asset_translations, mode_name_tr).
//
// COMPLÉMENTAIRE de mode_label.go : NormalizeModeLabel extrait le sous-mode
// d'une chaîne brute ; ResolvePairNameFR cascade entre 3 sources pour produire
// un label FR consistant. Pattern aligné sur home_repo.enrichHomeMatchTranslations
// (qui était le seul caller à l'avoir correctement). Extrait en pure fonction
// pour partager la logique avec match_history et filters (cf. thought_log
// 2026-05-09 — root cause des doublons EN+FR).
package analysis

import "strings"

// ResolvePairNameFR retourne le label FR canonique d'une paire mode/map.
//
// Cascade (alignée sur home_repo.enrichHomeMatchTranslations 2026-05-08) :
//
//  1. NormalizeModeLabel(rawPairName) → lookup mode_name_tr (priorité 1).
//  2. NormalizeModeLabel(pairAssetName) → lookup mode_name_tr (priorité 2 :
//     gère le cas où rawPairName est vide ou un UUID, et asset_translations
//     contient le label complet "Arena:CTF on X").
//  3. pairAssetName tel quel (priorité 3 : fallback raw, uniquement si
//     currentFR est manquant ou identique à rawPairName — placeholder de
//     COALESCE SQL).
//  4. currentFR (préserve l'existant si rien de mieux trouvé).
//
// Tous les paramètres sont strings (pas de pointers) pour rester pure et
// testable. Le caller choisit de set ou non selon le retour.
//
// Inputs :
//   - rawPairName        : pair_name brut depuis match_registry (typiquement EN
//     "Arena:CTF on Aquarius", peut être vide).
//   - currentFR          : pair_name_fr déjà présent (typiquement le résultat
//     de COALESCE(pair_name_fr, pair_name) du SQL — donc l'EN si la colonne FR
//     en DB est NULL).
//   - pairAssetName      : nom résolu depuis asset_translations[pair_id, fr-FR]
//     — peut être l'EN raw "Arena:CTF on Aquarius" si la table contient l'EN
//     pour toutes les langues (cas observé pour les paires Forge/Community
//     dont aucune locale n'est définie côté Microsoft, cf. thought_log
//     2026-05-09).
//   - modeNamesFR        : map mode_en (normalisé) → mode_fr depuis
//     metadata.mode_name_tr (lang='fr').
//
// Retourne le label à appliquer. Si égal à `currentFR`, le caller peut sauter
// l'override.
func ResolvePairNameFR(rawPairName, currentFR, pairAssetName string, modeNamesFR map[string]string) string {
	// Priorité 1 : mode_name_tr depuis pair_name brut
	if fr := modeNamesFR[NormalizeModeLabel(rawPairName)]; fr != "" {
		return fr
	}

	asset := strings.TrimSpace(pairAssetName)

	// Priorité 2 : mode_name_tr depuis asset_translations (re-normalisé)
	// Couvre le cas où rawPairName est vide ou un UUID, et asset_translations
	// contient "Arena:CTF on Aquarius" comme valeur "FR".
	if asset != "" {
		if fr := modeNamesFR[NormalizeModeLabel(asset)]; fr != "" {
			return fr
		}
	}

	// Priorité 3 : raw asset_translations (uniquement si currentFR placeholder)
	if asset != "" && needsFRTranslationOverride(currentFR, rawPairName) {
		return asset
	}

	// Préserver l'existant
	return strings.TrimSpace(currentFR)
}

// needsFRTranslationOverride retourne true si le label FR est manquant ou
// strictement identique au label EN — auquel cas il s'agit du placeholder
// COALESCE SQL fallback (pair_name_fr=NULL → COALESCE retourne pair_name) et
// peut être écrasé en toute sécurité par une meilleure valeur.
func needsFRTranslationOverride(labelFR, labelEN string) bool {
	fr := strings.TrimSpace(labelFR)
	if fr == "" {
		return true
	}
	en := strings.TrimSpace(labelEN)
	return en != "" && strings.EqualFold(fr, en)
}
