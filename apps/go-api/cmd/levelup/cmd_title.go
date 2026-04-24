// cmd_title.go — sous-commande "add-title" : initialise l'arborescence d'un nouveau jeu.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb"
)

// runAddTitle crée la structure disque et met à jour db_profiles.json pour un nouveau titre.
// La seule étape manuelle restante est l'ajout du TitleDescriptor dans registry.go.
func runAddTitle(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("add-title", flag.ExitOnError)
	name := fs.String("name", "", "Nom complet du jeu (ex: \"Halo: MCC\")")
	slug := fs.String("slug", "", "Slug du titre (déduit du nom si absent)")
	caps := fs.String("capabilities", "matchmaking,media", "Capabilities séparées par virgule")
	xboxID := fs.String("xbox-id", "", "Xbox Title ID (optionnel)")
	steamID := fs.String("steam-id", "", "Steam App ID (optionnel)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *name == "" {
		return fmt.Errorf("--name est obligatoire (ex: --name \"Halo MCC\")")
	}

	resolvedSlug := *slug
	if resolvedSlug == "" {
		resolvedSlug = slugFromName(*name)
	}
	if err := validateSlug(resolvedSlug); err != nil {
		return err
	}
	if resolvedSlug == title.DefaultSlug {
		return fmt.Errorf("le slug %q est réservé au titre par défaut", resolvedSlug)
	}

	fmt.Printf("Titre  : %s\n", *name)
	fmt.Printf("Slug   : %s\n", resolvedSlug)
	fmt.Println()

	// 1. Répertoires.
	if err := createTitleDirs(cfg.RepoRoot, resolvedSlug); err != nil {
		return err
	}

	// 2. shared_pve.duckdb si le titre supporte Firefight.
	if strings.Contains(strings.ToLower(*caps), "firefight") {
		if err := initPveDB(cfg.RepoRoot, resolvedSlug); err != nil {
			return err
		}
	}

	// 3. db_profiles.json.
	if err := addTitleToProfiles(cfg.DBProfilesPath, resolvedSlug); err != nil {
		return err
	}

	// 4. Afficher le snippet Go à coller dans registry.go.
	printRegistrySnippet(*name, resolvedSlug, *caps, *xboxID, *steamID)

	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Slug helpers
// ─────────────────────────────────────────────────────────────────────────────

var (
	reNonAlnum  = regexp.MustCompile(`[^a-z0-9]+`)
	reMultiUnd  = regexp.MustCompile(`_+`)
	reValidSlug = regexp.MustCompile(`^[a-z][a-z0-9_]*[a-z0-9]$`)
)

func slugFromName(name string) string {
	s := strings.ToLower(name)
	s = reNonAlnum.ReplaceAllString(s, "_")
	s = reMultiUnd.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	return s
}

func validateSlug(s string) error {
	if !reValidSlug.MatchString(s) {
		return fmt.Errorf("slug invalide %q — doit correspondre à [a-z][a-z0-9_]*[a-z0-9]", s)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Initialisation shared_pve.duckdb (Firefight)
// ─────────────────────────────────────────────────────────────────────────────

// initPveDB crée et initialise shared_pve.duckdb si le titre supporte Firefight.
// Le fichier doit exister pour que runMigrations le prenne en charge au démarrage.
func initPveDB(repoRoot, slug string) error {
	pr := title.NewPathResolver(repoRoot)
	pvePath := pr.SharedPVEDBPath(slug)

	if _, err := os.Stat(pvePath); err == nil {
		fmt.Printf("  [déjà présent] %s\n", pvePath)
		return nil
	}

	db, err := duckdb.OpenReadWrite(pvePath)
	if err != nil {
		return fmt.Errorf("initialisation shared_pve.duckdb : %w", err)
	}
	db.Close()
	fmt.Printf("  [créé] %s\n", pvePath)
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Création des répertoires
// ─────────────────────────────────────────────────────────────────────────────

func createTitleDirs(repoRoot, slug string) error {
	pr := title.NewPathResolver(repoRoot)
	dirs := []string{
		pr.WarehouseDir(slug),
		pr.PlayerDir(slug, ""), // data/titles/<slug>/players/
		// Dossier frontend pour les images du hero banner (home page).
		// Placer ici les visuels header du titre (webp/png).
		filepath.Join(repoRoot, "apps", "web", "public", "titles", slug),
	}
	// PlayerDir avec gamertag vide retourne le répertoire players/ du titre.
	// On le normalise : supprimer le trailing segment vide que filepath.Join ajouterait.
	dirs[1] = filepath.Join(pr.TitleDataDir(slug), "players")

	for _, d := range dirs {
		if _, err := os.Stat(d); err == nil {
			fmt.Printf("  [déjà présent] %s\n", d)
			continue
		}
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("création répertoire %s : %w", d, err)
		}
		fmt.Printf("  [créé] %s\n", d)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Mise à jour db_profiles.json
// ─────────────────────────────────────────────────────────────────────────────

// dbProfilesV3 est une représentation locale pour la lecture/écriture.
type dbProfilesV3 struct {
	Version  string                                `json:"version"`
	Profiles map[string]map[string]json.RawMessage `json:"profiles"`
}

func addTitleToProfiles(profilesPath, slug string) error {
	data, err := os.ReadFile(profilesPath)
	if err != nil {
		return fmt.Errorf("lecture %s : %w", profilesPath, err)
	}

	var file dbProfilesV3
	if err := json.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("parsing db_profiles.json : %w", err)
	}
	if file.Version != "3.0" {
		return fmt.Errorf("db_profiles.json version %q non supportée par add-title (attendu: 3.0)", file.Version)
	}
	if file.Profiles == nil {
		file.Profiles = make(map[string]map[string]json.RawMessage)
	}

	if _, exists := file.Profiles[slug]; exists {
		fmt.Printf("  [déjà présent] entrée %q dans db_profiles.json\n", slug)
		return nil
	}

	file.Profiles[slug] = make(map[string]json.RawMessage)

	out, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("sérialisation db_profiles.json : %w", err)
	}
	out = append(out, '\n')
	if err := os.WriteFile(profilesPath, out, 0o644); err != nil {
		return fmt.Errorf("écriture db_profiles.json : %w", err)
	}
	fmt.Printf("  [mis à jour] db_profiles.json — section %q ajoutée\n", slug)
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Snippet Go pour registry.go
// ─────────────────────────────────────────────────────────────────────────────

func printRegistrySnippet(name, slug, capsStr, xboxID, steamID string) {
	capsList := buildCapsList(capsStr)

	fmt.Println()
	fmt.Println("┌─────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│  Étape manuelle requise — ajouter dans registry.go (NewRegistry)    │")
	fmt.Println("└─────────────────────────────────────────────────────────────────────┘")
	fmt.Println()
	fmt.Printf("  r.Register(&TitleDescriptor{\n")
	fmt.Printf("      Slug:     %q,\n", slug)
	fmt.Printf("      Name:     %q,\n", name)
	fmt.Printf("      Provider: %q,\n", slug)
	fmt.Printf("      Status:   StatusComingSoon, // passer à StatusActive quand prêt\n")
	fmt.Printf("      Capabilities: []Capability{\n")
	fmt.Printf("          %s\n", capsList)
	fmt.Printf("      },\n")
	fmt.Printf("      IsDefault:   false,\n")
	if xboxID != "" {
		fmt.Printf("      XboxTitleID: %q,\n", xboxID)
	} else {
		fmt.Printf("      XboxTitleID: \"\", // à renseigner\n")
	}
	if steamID != "" {
		fmt.Printf("      SteamAppID:  %q,\n", steamID)
	} else {
		fmt.Printf("      SteamAppID:  \"\", // à renseigner si disponible sur Steam\n")
	}
	fmt.Printf("  })\n")
	fmt.Println()
	fmt.Println("  Puis : make build")
	fmt.Println()
	fmt.Println("┌─────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│  Étape frontend — ajouter les images du hero banner                 │")
	fmt.Println("└─────────────────────────────────────────────────────────────────────┘")
	fmt.Println()
	fmt.Printf("  Dossier créé : apps/web/public/titles/%s/\n", slug)
	fmt.Println("  Y déposer les images header (webp ou png), puis les référencer dans :")
	fmt.Println("  apps/web/src/features/home/HomeHeroBanner.tsx — HEADER_IMAGES_BY_TITLE")
	fmt.Println()
}

func buildCapsList(capsStr string) string {
	knownCaps := map[string]string{
		"matchmaking": "CapMatchmaking",
		"firefight":   "CapFirefight",
		"forge":       "CapForge",
		"media":       "CapMedia",
		"ranked":      "CapRanked",
		"career":      "CapCareer",
	}
	parts := strings.Split(capsStr, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(strings.ToLower(p))
		if c, ok := knownCaps[p]; ok {
			out = append(out, c+",")
		}
	}
	if len(out) == 0 {
		return "// aucune capability spécifiée"
	}
	return strings.Join(out, " ")
}
