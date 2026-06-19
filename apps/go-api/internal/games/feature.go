package games

import (
	"sort"

	"levelup/go-api/internal/domain/feature"
)

// feature.go — cascade capabilities → matrice de features produit (Phase 1.7b).
//
// Les capabilities (capabilities.toml, niveau données) décrivent ce que le titre
// peut PRODUIRE ; les features (niveau produit) décrivent les surfaces UI. La
// matrice se calcule par CASCADE : pour chaque feature, son statut découle des
// statuts des capabilities qu'elle requiert. 100% title-agnostic : on lit la
// CapabilityMap du titre, jamais son slug.

// featureDef décrit les capabilities dont une feature dépend.
//
//   - primary : capability requise. Absente/not_exposed → feature unavailable ;
//     degraded → feature degraded.
//   - enhancements : capabilities d'enrichissement. Si l'une n'est pas supported
//     alors que la primaire l'est, la feature est degraded (utilisable, partielle).
type featureDef struct {
	primary      CapabilityKey
	enhancements []CapabilityKey
}

// featureDefinitions : mapping produit feature → capabilities. Stable et partagé
// entre titres (la variation par titre vient de SA CapabilityMap, pas d'ici).
var featureDefinitions = map[feature.Key]featureDef{
	feature.KeyMatchHistory: {primary: CapMatchHistory},
	feature.KeyMatchDetail:  {primary: CapMatchDetailCore, enhancements: []CapabilityKey{CapScoreboardExtra}},
	feature.KeySkillRating:  {primary: CapMatchSkillSnapshot},
	feature.KeyCareer:       {primary: CapCareerProgression},
	feature.KeyPveStats:     {primary: CapPveFirefight},
	feature.KeyTimeseries:   {primary: CapTimeseries},
	feature.KeyCitations:    {primary: CapCitationsEngine},
	feature.KeyEngagement:   {primary: CapEngagement},
	feature.KeyBattlePass:   {primary: CapBattlePass},
	feature.KeyChallenges:   {primary: CapChallenges},
}

// AllFeatureKeys retourne les features produit connues, triées (déterminisme).
func AllFeatureKeys() []feature.Key {
	out := make([]feature.Key, 0, len(featureDefinitions))
	for k := range featureDefinitions {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// ComputeFeatureMatrix dérive la matrice de features d'un titre à partir de sa
// CapabilityMap (Phase 1.7b). La matrice contient TOUTES les features produit ;
// celles dont la capability primaire est absente du titre sont unavailable
// (dégradation gracieuse — un 2e titre sans une capability n'expose pas la
// feature correspondante).
func ComputeFeatureMatrix(caps CapabilityMap) feature.Matrix {
	out := make(feature.Matrix, len(featureDefinitions))
	for key, def := range featureDefinitions {
		out[key] = cascadeFeature(caps, def)
	}
	return out
}

// cascadeFeature applique la règle de cascade pour une feature.
func cascadeFeature(caps CapabilityMap, def featureDef) feature.Status {
	switch caps[def.primary] { // "" si la capability est absente du titre
	case CapSupported:
		// Primaire OK → available, sauf si un enrichissement manque.
		for _, enh := range def.enhancements {
			if caps[enh] != CapSupported {
				return feature.StatusDegraded
			}
		}
		return feature.StatusAvailable
	case CapDegraded:
		return feature.StatusDegraded
	default: // CapNotExposed ou capability absente
		return feature.StatusUnavailable
	}
}
