// labels.go — seam de présentation title-aware pour les libellés des embeds
// Discord (PMT-11, MT-26). Le contenu Halo (outcomes Victoire/Défaite/…) sortait
// codé en dur ; il passe désormais par NotifyLabels, dont l'impl par défaut
// (haloLabels) reproduit byte-pour-byte les strings actuels, et une impl
// semanticLabels lit les libellés du manifeste outcomes.toml du titre courant.
//
// Failsafe : aucune comparaison de slug (archlint/no_slug_comparison) ; le routage
// se fait par présence d'un OutcomeSource, jamais par littéral. src/Outcomes()/clé
// absente → dégradation propre vers les libellés Halo.
package notify

import (
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/mappings"
)

// defaultTitleDisplayName retourne le nom d'affichage du titre par défaut depuis
// le registre (source UNIQUE du nom du titre). Failsafe défensif vers "Halo
// Infinite" si le descripteur est absent — en pratique jamais atteint, le
// registre par défaut s'auto-initialise avec le descripteur Halo.
func defaultTitleDisplayName() string {
	if d := titlePkg.DefaultRegistry().Get(titlePkg.DefaultSlug); d != nil && d.Name != "" {
		return d.Name
	}
	return "Halo Infinite"
}

// discordFooterText construit le footer des embeds Discord (« LevelUp · {nom}
// Stats ») à partir du nom de titre porté par `labels` (PMT-11 : le footer suit
// désormais la MÊME dimension titre que les outcomes — un seul seam). Locale-
// indépendant (footer identique FR/EN). labels nil → libellés Halo → nom du titre
// par défaut → "LevelUp · Halo Infinite Stats" (byte-identique).
func discordFooterText(labels NotifyLabels) string {
	return "LevelUp · " + labelsOrDefault(labels).TitleName() + " Stats"
}

// NotifyLabels expose les libellés title-aware nécessaires au rendu des embeds :
// les outcomes (manifeste outcomes.toml du titre) ET le nom d'affichage du titre
// (pour le footer). Un même seam pilote tout le contenu title-dépendant de l'embed.
type NotifyLabels interface {
	// Outcome retourne le libellé d'un résultat de match pour une langue, à partir
	// de sa clé canonique (win|loss|tie|dnf).
	Outcome(canonicalKey, lang string) string
	// TitleName retourne le nom d'affichage du titre (ex. "Halo Infinite") — utilisé
	// dans le footer. Le titre par défaut côté Halo, le nom du 2e titre via LabelsFor.
	TitleName() string
}

// OutcomeSource est la surface MINIMALE dont notify a besoin d'un adapter
// sémantique de titre — games.TitleSemanticAdapter la satisfait. Évite d'importer
// tout le package games dans la couche présentation.
type OutcomeSource interface {
	Outcomes() *mappings.OutcomeMappingSet
}

// haloOutcomeI18nKey mappe une clé canonique vers la clé i18n Halo historique
// (discordStrings). Préserve le comportement existant à l'octet près.
var haloOutcomeI18nKey = map[string]string{
	"tie":  "discord_outcome_draw",
	"win":  "discord_outcome_win",
	"loss": "discord_outcome_loss",
	"dnf":  "discord_outcome_quit",
}

// haloLabels : libellés Halo historiques (via la map discordStrings + T).
type haloLabels struct{}

func (haloLabels) Outcome(canonicalKey, lang string) string {
	if k, ok := haloOutcomeI18nKey[canonicalKey]; ok {
		return T(k, lang)
	}
	return canonicalKey
}

// TitleName retourne le nom du titre par défaut (registre) — "Halo Infinite".
func (haloLabels) TitleName() string { return defaultTitleDisplayName() }

// HaloLabels retourne l'implémentation par défaut (Halo, byte-identique).
func HaloLabels() NotifyLabels { return haloLabels{} }

// semanticLabels : libellés du titre courant via son outcomes.toml + son nom
// d'affichage, avec dégradation vers fallback (Halo) si la source/la clé/le nom
// est absent.
type semanticLabels struct {
	src       OutcomeSource
	titleName string
	fallback  NotifyLabels
}

func (s semanticLabels) Outcome(canonicalKey, lang string) string {
	if s.src != nil {
		if oc := s.src.Outcomes(); oc != nil {
			if m, ok := oc.Get(canonicalKey); ok {
				if lbl, _ := m.Label(lang); lbl != "" {
					return lbl
				}
			}
		}
	}
	return s.fallback.Outcome(canonicalKey, lang)
}

// TitleName retourne le nom du titre fourni à LabelsFor, ou le fallback Halo si vide.
func (s semanticLabels) TitleName() string {
	if s.titleName != "" {
		return s.titleName
	}
	return s.fallback.TitleName()
}

// LabelsFor construit un NotifyLabels title-aware : les outcomes viennent du
// manifeste du titre (via src) et le footer de son nom (titleName), avec
// dégradation failsafe vers les libellés Halo. src nil → HaloLabels pur (cas du
// titre par défaut ou absence d'adapter). titleName vide → nom du titre par défaut.
func LabelsFor(src OutcomeSource, titleName string) NotifyLabels {
	if src == nil {
		return haloLabels{}
	}
	return semanticLabels{src: src, titleName: titleName, fallback: haloLabels{}}
}

// labelsOrDefault garantit le caractère failsafe : nil → HaloLabels.
func labelsOrDefault(l NotifyLabels) NotifyLabels {
	if l == nil {
		return haloLabels{}
	}
	return l
}
