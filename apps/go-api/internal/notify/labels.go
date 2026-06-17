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
// Stats ») à partir du nom du titre par défaut — au lieu d'une 2e copie codée en
// dur dans la map i18n. Locale-indépendant (footer identique FR/EN). Seam PMT-11 :
// les embeds ne portent pas encore de titre (threading différé) ; quand il
// arrivera, passer le nom du titre courant en paramètre ici.
func discordFooterText() string {
	return "LevelUp · " + defaultTitleDisplayName() + " Stats"
}

// NotifyLabels expose les libellés title-aware nécessaires au rendu des embeds.
// Périmètre minimal de cette phase : les outcomes (les seuls dont le manifeste
// par titre — outcomes.toml — existe déjà). Footer/backfill restent Halo tant que
// le manifeste i18n par titre ne les porte pas.
type NotifyLabels interface {
	// Outcome retourne le libellé d'un résultat de match pour une langue, à partir
	// de sa clé canonique (win|loss|tie|dnf).
	Outcome(canonicalKey, lang string) string
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

// HaloLabels retourne l'implémentation par défaut (Halo, byte-identique).
func HaloLabels() NotifyLabels { return haloLabels{} }

// semanticLabels : libellés du titre courant via son outcomes.toml, avec
// dégradation vers fallback (Halo) si la source ou la clé est absente.
type semanticLabels struct {
	src      OutcomeSource
	fallback NotifyLabels
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

// LabelsFor construit un NotifyLabels title-aware : les outcomes viennent du
// manifeste du titre (via src), avec dégradation failsafe vers les libellés Halo.
// src nil → HaloLabels pur (cas du titre par défaut ou absence d'adapter).
func LabelsFor(src OutcomeSource) NotifyLabels {
	if src == nil {
		return haloLabels{}
	}
	return semanticLabels{src: src, fallback: haloLabels{}}
}

// labelsOrDefault garantit le caractère failsafe : nil → HaloLabels.
func labelsOrDefault(l NotifyLabels) NotifyLabels {
	if l == nil {
		return haloLabels{}
	}
	return l
}
