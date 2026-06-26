package mappings

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// endpointsTOML est la projection brute de constants.toml (sections [endpoints]
// + [damage_model]).
type endpointsTOML struct {
	Meta        metaSection       `toml:"meta"`
	Endpoints   map[string]string `toml:"endpoints"`
	DamageModel damageModelTOML   `toml:"damage_model"`
}

// damageModelTOML projette la section [damage_model] (constantes de gameplay,
// title-spécifiques). Optionnelle : absente → modèle de dégâts non déclaré.
type damageModelTOML struct {
	EffectiveHpToKill float64 `toml:"effective_hp_to_kill"`
	// no_native_kda = true → le titre ne fournit PAS de KDA per-match via son API
	// (ex. Halo 5 : forme native = FDA NET, pas le quotient KDA). Défaut false =
	// KDA natif disponible (Infinite). Consommé via games.ProvidesNativeKDA(slug).
	NoNativeKDA bool `toml:"no_native_kda"`
	// no_damage_taken = true → le titre ne fournit PAS damage_taken (Halo 5). La
	// résistance défensive et ses dérivés sont neutralisés. Via games.ProvidesDamageTaken.
	NoDamageTaken bool `toml:"no_damage_taken"`
	// offensive_conversion_p80 : frontière élite OC (80e percentile) du titre, repère
	// de normalisation des barres/radars de rendement. 0/absent = défaut Infinite (0.90).
	OffensiveConversionP80 float64 `toml:"offensive_conversion_p80"`
	// no_team_mmr = true → le titre ne fournit PAS de MMR d'équipe/adverse par match
	// (Halo 5). La colonne MMR du tableau Escouade/Explorer est masquée. Via
	// games.ProvidesTeamMMR. Défaut false (MMR fourni, Infinite).
	NoTeamMMR bool `toml:"no_team_mmr"`
	// NB — pas de flag no_max_killing_spree : la « folie meurtrière max » est DÉRIVÉE
	// (events kill/death horodatés → analysis.ComputeMaxKillingSpree) pour tout titre qui
	// porte la capability events-timeline ; le support n'est donc pas un flag du modèle
	// de dégâts mais une propriété de la capability. Cf. games.ProvidesMaxKillingSpree.
}

// allowedEndpointKeys est la liste exhaustive des clés d'endpoint admises (MT-01).
// Toute autre clé dans [endpoints] est rejetée (un titre ne déclare pas un host
// hors vocabulaire d'ingestion canonique).
var allowedEndpointKeys = map[EndpointKey]struct{}{
	EndpointStats:        {},
	EndpointGameCMS:      {},
	EndpointEconomy:      {},
	EndpointSkill:        {},
	EndpointUGCFilm:      {},
	EndpointDiscoveryUGC: {},
	EndpointChallenges:   {},
	EndpointNameplate:    {},
}

// LoadEndpointsFromFile lit et valide la section [endpoints] d'un constants.toml.
func LoadEndpointsFromFile(path string) (*EndpointSet, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return LoadEndpointsFromBytes(path, raw)
}

// LoadEndpointsFromBytes parse et valide un payload TOML déjà chargé en mémoire.
// path n'est utilisé que pour les messages d'erreur.
func LoadEndpointsFromBytes(path string, raw []byte) (*EndpointSet, error) {
	var doc endpointsTOML
	if err := toml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	var errs []error

	if doc.Meta.TitleSlug == "" {
		errs = append(errs, fmt.Errorf("[meta].title_slug manquant"))
	}
	if doc.Meta.SchemaVersion <= 0 {
		errs = append(errs, fmt.Errorf("[meta].schema_version doit être > 0 (reçu %d)", doc.Meta.SchemaVersion))
	}
	if len(doc.Endpoints) == 0 {
		errs = append(errs, fmt.Errorf("section [endpoints] absente ou vide"))
	}

	byKey := make(map[EndpointKey]string, len(doc.Endpoints))
	for rawKey, host := range doc.Endpoints {
		key := EndpointKey(rawKey)
		if _, ok := allowedEndpointKeys[key]; !ok {
			errs = append(errs, fmt.Errorf("[endpoints.%s] clé inconnue (admises : stats, gamecms, economy, skill, ugc_film, discovery_ugc, challenges, nameplate)", rawKey))
			continue
		}
		trimmed := strings.TrimSpace(host)
		if trimmed == "" {
			errs = append(errs, fmt.Errorf("[endpoints.%s] host vide", rawKey))
			continue
		}
		if !strings.HasPrefix(trimmed, "https://") {
			errs = append(errs, fmt.Errorf("[endpoints.%s] host non-https: %q", rawKey, trimmed))
			continue
		}
		byKey[key] = trimmed
	}

	gamePrefix := strings.TrimSpace(doc.Meta.GamePrefix)
	if gamePrefix != "" && !isValidGamePrefix(gamePrefix) {
		errs = append(errs, fmt.Errorf("[meta].game_prefix invalide %q (attendu : segment d'URL minuscule alphanumérique, ex. \"hi\", \"h5\")", gamePrefix))
	}

	// [damage_model] optionnel. effective_hp_to_kill : 0 (ou absent) = non déclaré
	// (le caller applique son défaut) ; < 0 = invalide (PV négatifs absurdes).
	if doc.DamageModel.EffectiveHpToKill < 0 {
		errs = append(errs, fmt.Errorf("[damage_model].effective_hp_to_kill doit être > 0 (reçu %v)", doc.DamageModel.EffectiveHpToKill))
	}
	if doc.DamageModel.OffensiveConversionP80 < 0 {
		errs = append(errs, fmt.Errorf("[damage_model].offensive_conversion_p80 doit être >= 0 (reçu %v)", doc.DamageModel.OffensiveConversionP80))
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("validation %s: %w", path, errors.Join(errs...))
	}

	dm := DamageModelConstants{
		EffectiveHpToKill:      doc.DamageModel.EffectiveHpToKill,
		NoNativeKDA:            doc.DamageModel.NoNativeKDA,
		NoDamageTaken:          doc.DamageModel.NoDamageTaken,
		OffensiveConversionP80: doc.DamageModel.OffensiveConversionP80,
		NoTeamMMR:              doc.DamageModel.NoTeamMMR,
	}
	return NewEndpointSet(doc.Meta.TitleSlug, doc.Meta.SchemaVersion, gamePrefix, byKey).withDamageModel(dm), nil
}

// isValidGamePrefix vérifie qu'un game_prefix est un segment d'URL sûr : une
// suite de minuscules/chiffres (pas de slash, espace ou majuscule). Sert à
// garantir que le préfixe injecté dans les chemins d'API ne casse pas l'URL.
func isValidGamePrefix(p string) bool {
	for _, r := range p {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}
