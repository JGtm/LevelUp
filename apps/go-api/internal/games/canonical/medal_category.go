package canonical

import "strings"

// Jeu canonique de catégories de médailles, partagé par TOUS les titres.
//
// Le frontend (apps/web/src/features/squad/i18n.ts → categoryLabels) traduit
// EXACTEMENT ces clés ; toute valeur hors de ce set retombe sur le libellé brut
// (anglais, non traduit) côté UI. La normalisation au sein de ce package est
// donc la frontière title-agnostic : chaque titre (Infinite via les index
// Waypoint, Halo 5 via l'enum Classification de l'API officielle) doit produire
// l'une de ces clés.
const (
	MedalCategoryMultikill   = "multikill"
	MedalCategorySpree       = "spree"
	MedalCategorySkill       = "skill"
	MedalCategoryStyle       = "style"
	MedalCategoryMode        = "mode"
	MedalCategoryProficiency = "proficiency"
	MedalCategoryOther       = "other"
)

// AllMedalCategories retourne la liste exhaustive des catégories canoniques.
// L'ordre miroite categoryLabels côté frontend.
func AllMedalCategories() []string {
	return []string{
		MedalCategoryMultikill,
		MedalCategorySpree,
		MedalCategorySkill,
		MedalCategoryStyle,
		MedalCategoryMode,
		MedalCategoryProficiency,
		MedalCategoryOther,
	}
}

// NormalizeMedalCategory ramène une catégorie de médaille brute (peu importe le
// titre d'origine) vers l'une des clés canoniques multikill|spree|skill|style|
// mode|proficiency|other.
//
// Idempotente et insensible à la casse : elle accepte aussi bien l'enum brut de
// l'API Halo 5 (PascalCase : MultiKill, WeaponProficiency, KillingSpree,
// CaptureTheFlag, Oddball…) que les valeurs déjà normalisées d'Halo Infinite
// (multikill, spree, skill, style, mode, proficiency). Ainsi
// NormalizeMedalCategory("MultiKill") == NormalizeMedalCategory("multikill") ==
// "multikill".
//
// Les classifications spécifiques à un mode de jeu (drapeau, balle, bastions,
// roi de la colline, assaut, percée, infection…) sont regroupées sous "mode".
// Toute valeur inconnue ou vide retombe sur "other" (clé existante côté
// frontend, qui mappe lui-même "" → "other").
func NormalizeMedalCategory(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "multikill", "multi-kill", "multi_kill":
		return MedalCategoryMultikill
	case "spree", "killingspree", "killing-spree", "killing_spree":
		return MedalCategorySpree
	case "proficiency", "weaponproficiency", "weapon-proficiency", "weapon_proficiency", "vehicle", "vehicleproficiency":
		return MedalCategoryProficiency
	case "style":
		return MedalCategoryStyle
	case "skill":
		return MedalCategorySkill
	case "mode",
		"objective",
		"capturetheflag", "ctf", "flag",
		"oddball", "ball",
		"strongholds", "stronghold",
		"kingofthehill", "koth", "hill",
		"assault", "bomb",
		"breakout",
		"infection",
		"slayer",
		"juggernaut",
		"vip",
		"territories",
		"gametype", "gamemode":
		return MedalCategoryMode
	default:
		return MedalCategoryOther
	}
}
