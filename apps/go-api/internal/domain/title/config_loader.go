// Package title — config_loader.go : registre PILOTÉ PAR CONFIG (MT-16 / day-one
// 2e titre). Découvre les titres ADDITIONNELS déclarés sous
// config/titles/{slug}/title.toml et les enregistre, pour qu'ajouter un 2e titre
// = déposer un dossier de config (zéro recompilation).
//
// halo_infinite reste le descripteur BUILT-IN câblé en dur dans NewRegistry()
// (byte-identique, robuste même sans config) ; un title.toml halo_infinite est
// donc ignoré ici. Les dossiers config/titles/* SANS title.toml (mappings-only,
// ex. fixtures sémantiques) sont ignorés silencieusement — ils ne déclarent pas
// un titre enregistrable.
package title

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// ErrTitleManifestAbsent est retourné quand un dossier de titre ne porte pas de
// title.toml. Le caller l'ignore (le dossier n'est pas un titre enregistrable).
var ErrTitleManifestAbsent = errors.New("title: manifeste absent (title.toml)")

// knownCapabilities — set des capabilities valides (miroir exact des const Cap*).
// Toute capability d'un title.toml doit appartenir à ce set (sinon erreur de
// validation : un titre ne peut pas déclarer une feature inexistante côté code).
var knownCapabilities = map[Capability]struct{}{
	CapMatchmaking:         {},
	CapFirefight:           {},
	CapForge:               {},
	CapMedia:               {},
	CapRanked:              {},
	CapCareer:              {},
	CapSeasonPass:          {},
	CapAssetImages:         {},
	CapAchievements:        {},
	CapEngagement:          {},
	CapLUSR:                {},
	CapWorldLeaderboard:    {},
	CapNativeKillMechanics: {},
	CapWeaponKills:         {},
	CapWeaponAccuracy:      {},
	CapTeamMMR:             {},
	CapDamageTaken:         {},
	CapSpartanCustomizer:   {},
	CapExpectedStats:       {},
	CapWaypointMatchURL:    {},
	CapObjectiveStats:      {},
}

func knownStatus(s Status) bool {
	return s == StatusActive || s == StatusComingSoon || s == StatusArchived
}

// titleManifestTOML — projection brute de config/titles/{slug}/title.toml.
type titleManifestTOML struct {
	Meta  titleManifestMeta    `toml:"meta"`
	Title titleManifestSection `toml:"title"`
}

type titleManifestMeta struct {
	TitleSlug     string `toml:"title_slug"`
	SchemaVersion int    `toml:"schema_version"`
}

type titleManifestSection struct {
	Name             string   `toml:"name"`
	Provider         string   `toml:"provider"`
	IconURL          string   `toml:"icon_url"`
	Status           string   `toml:"status"`
	Capabilities     []string `toml:"capabilities"`
	IsDefault        bool     `toml:"is_default"`
	IsInternal       bool     `toml:"is_internal"`
	XboxTitleID      string   `toml:"xbox_title_id"`
	SteamAppID       string   `toml:"steam_app_id"`
	PlacementMatches int      `toml:"placement_matches"`
	CSRSeasonID      string   `toml:"csr_season_id"`
	// MediaFilenamePrefixes : préfixes de nom de fichier de capture appartenant au
	// titre (routage de l'indexeur média, DEC-8 — cf. TitleDescriptor).
	MediaFilenamePrefixes []string `toml:"media_filename_prefixes"`
}

// LoadTitleManifest charge le descripteur d'un titre depuis
// config/titles/{slug}/title.toml. ErrTitleManifestAbsent si le fichier manque.
func LoadTitleManifest(repoRoot, slug string) (*TitleDescriptor, error) {
	path := filepath.Join(repoRoot, "config", "titles", slug, "title.toml")
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrTitleManifestAbsent
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return LoadTitleManifestFromBytes(path, slug, raw)
}

// LoadTitleManifestFromBytes parse et valide un payload title.toml en mémoire.
// Le slug du DOSSIER fait foi ; si [meta].title_slug est présent il doit concorder.
func LoadTitleManifestFromBytes(path, slug string, raw []byte) (*TitleDescriptor, error) {
	var doc titleManifestTOML
	if err := toml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	var errs []error
	if doc.Meta.TitleSlug != "" && doc.Meta.TitleSlug != slug {
		errs = append(errs, fmt.Errorf("[meta].title_slug %q != dossier %q", doc.Meta.TitleSlug, slug))
	}
	if doc.Meta.SchemaVersion <= 0 {
		errs = append(errs, fmt.Errorf("[meta].schema_version doit être > 0 (reçu %d)", doc.Meta.SchemaVersion))
	}

	desc := &TitleDescriptor{
		Slug:             slug,
		Name:             strings.TrimSpace(doc.Title.Name),
		Provider:         strings.TrimSpace(doc.Title.Provider),
		IconURL:          strings.TrimSpace(doc.Title.IconURL),
		Status:           Status(strings.TrimSpace(doc.Title.Status)),
		IsDefault:        doc.Title.IsDefault,
		IsInternal:       doc.Title.IsInternal,
		XboxTitleID:      strings.TrimSpace(doc.Title.XboxTitleID),
		SteamAppID:       strings.TrimSpace(doc.Title.SteamAppID),
		PlacementMatches: doc.Title.PlacementMatches,
		CSRSeasonID:      strings.TrimSpace(doc.Title.CSRSeasonID),
	}
	for _, p := range doc.Title.MediaFilenamePrefixes {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			desc.MediaFilenamePrefixes = append(desc.MediaFilenamePrefixes, trimmed)
		}
	}
	if desc.Name == "" {
		errs = append(errs, fmt.Errorf("[title].name manquant"))
	}
	if desc.Provider == "" {
		desc.Provider = slug // défaut raisonnable : le slug
	}
	if !knownStatus(desc.Status) {
		errs = append(errs, fmt.Errorf("[title].status invalide: %q (attendu active|coming_soon|archived)", desc.Status))
	}
	// is_default est réservé au built-in halo_infinite : un titre additionnel ne
	// peut pas se déclarer défaut (sinon deux défauts → ambiguïté DefaultSlug).
	if desc.IsDefault {
		errs = append(errs, fmt.Errorf("[title].is_default réservé au titre built-in (%s)", DefaultSlug))
	}
	for _, c := range doc.Title.Capabilities {
		capability := Capability(strings.TrimSpace(c))
		if _, ok := knownCapabilities[capability]; !ok {
			errs = append(errs, fmt.Errorf("[title].capabilities: capability inconnue %q", c))
			continue
		}
		desc.Capabilities = append(desc.Capabilities, capability)
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("validation %s: %w", path, errors.Join(errs...))
	}
	return desc, nil
}

// LoadTitlesIntoRegistry découvre les titres ADDITIONNELS sous config/titles/*/
// (chaque dossier portant un title.toml) et les enregistre dans reg. Retourne les
// erreurs par titre — une config invalide n'empêche pas les autres titres de se
// charger ni le serveur de démarrer (dégradation propre, le titre fautif reste
// non enregistré).
//
//   - dossier == halo_infinite → ignoré (built-in NewRegistry fait foi).
//   - dossier sans title.toml → ignoré (mappings-only, pas un titre déclaré).
//   - config/titles absent → no-op (mono-titre built-in).
func LoadTitlesIntoRegistry(reg *Registry, repoRoot string, logger *slog.Logger) []error {
	if logger == nil {
		logger = slog.Default()
	}
	titlesDir := filepath.Join(repoRoot, "config", "titles")
	entries, err := os.ReadDir(titlesDir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return []error{fmt.Errorf("read %s: %w", titlesDir, err)}
	}

	var errs []error
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		slug := e.Name()
		// Titre DÉJÀ enregistré (le built-in halo_infinite l'est par NewRegistry
		// avant la découverte config) → on ne l'écrase pas : le descripteur câblé
		// en dur fait foi, byte-identique. (Évite aussi une comparaison de slug —
		// archlint no_slug_comparison.)
		if reg.Exists(slug) {
			// F11 : un title.toml pour un titre built-in est SILENCIEUSEMENT ignoré.
			// WARN explicite pour qu'un dev ne croie pas ses edits pris en compte
			// (le descripteur built-in fait foi ; cf. ADR 0025).
			if _, statErr := os.Stat(filepath.Join(titlesDir, slug, "title.toml")); statErr == nil {
				logger.Warn("title_builtin_toml_ignored",
					"title", slug,
					"reason", "descripteur built-in prioritaire (byte-identique) — title.toml non appliqué")
			}
			continue
		}
		desc, err := LoadTitleManifest(repoRoot, slug)
		if errors.Is(err, ErrTitleManifestAbsent) {
			continue
		}
		if err != nil {
			logger.Error("title_manifest_invalid", "title", slug, "err", err.Error())
			errs = append(errs, err)
			continue
		}
		reg.Register(desc)
		logger.Info("title_registered_from_config",
			"title", slug,
			"status", string(desc.Status),
			"provider", desc.Provider,
			"capabilities", len(desc.Capabilities),
			"internal", desc.IsInternal, // fixture de test (exclue du switcher utilisateur)
		)
	}
	return errs
}

// NewRegistryFromConfig construit le registre built-in (halo_infinite) PUIS
// découvre les titres additionnels déclarés en config (title.toml). Point d'entrée
// unique du registre piloté par config : un 2e titre = un dossier config, zéro
// recompilation. Les erreurs de config sont loguées et n'empêchent pas le boot
// (le registre contient au minimum halo_infinite).
func NewRegistryFromConfig(repoRoot string, logger *slog.Logger) *Registry {
	reg := NewRegistry()
	_ = LoadTitlesIntoRegistry(reg, repoRoot, logger)
	return reg
}
