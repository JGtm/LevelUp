// Package config — player_resolver.go : erreurs, roster démo et chemins des bases d'un titre. L'ouverture du
// PlayerDB (ResolvePlayer) et la résolution du lecteur partagé par titre vivent dans internal/playerresolve.
package config

import (
	"fmt"
	"os"
	"strings"

	"levelup/go-api/internal/domain/title"
)

// ErrPlayerNotFound est retourné quand le slug ne correspond à aucun joueur configuré.
var ErrPlayerNotFound = fmt.Errorf("joueur introuvable")

// PrestigeBundleDBPaths retourne les chemins des DEUX bases ouvertes au boot par le
// module Prestige (shared_social + metadata) pour le titre par défaut — les seules
// que le bundle épingle (cf. wire.NewPrestigeBundleAt).
//
// Démo : layout PLAT des fixtures (LEVELUP_DEMO_FIXTURES_DIR/warehouse/…) ;
// production : layout title-scopé du PathResolver. Sans cette distinction, le
// bundle démo pointe un chemin inexistant et toute lecture Prestige adossée à une
// base reste vide. Repli sur le layout prod si DemoFixturesDir n'est pas renseigné.
func (cfg *AppConfig) PrestigeBundleDBPaths() (sharedSocialPath, metadataPath string) {
	slug := title.DefaultSlug
	if cfg.demoState() && cfg.DemoFixturesDir != "" {
		return cfg.DemoLayout().SharedSocialDBPath(slug), cfg.DemoLayout().MetadataDBPath(slug)
	}
	pr := cfg.PathResolver()
	return pr.SharedSocialDBPath(slug), pr.MetadataDBPath(slug)
}

// demoPlayerDef décrit un joueur démo du stack (DemoPlayer + 2 coéquipiers).
type demoPlayerDef struct {
	Slug     string
	Gamertag string
	XUID     string
	Dir      string // sous-répertoire fixtures : DEMO, DEMO2, DEMO3
}

// DemoRoster : les 3 joueurs démo fixes. Cohérent avec seed-demo
// (demoXUIDForIndex / demoDirForIndex / DemoPlayer{,2,3}). Le main "demo-player"
// est l'index 0 ; les 2 coéquipiers principaux sont 2 et 3.
var DemoRoster = []demoPlayerDef{
	{Slug: "demo-player", Gamertag: "DemoPlayer", XUID: "0000000000000000", Dir: "DEMO"},
	{Slug: "demo-player-2", Gamertag: "DemoPlayer2", XUID: "0000000000000001", Dir: "DEMO2"},
	{Slug: "demo-player-3", Gamertag: "DemoPlayer3", XUID: "0000000000000002", Dir: "DEMO3"},
}

// DemoDefForSlug retourne la def démo pour un slug OU un gamertag (résolution
// squad par gamertag), insensible à la casse. Fallback : le main (index 0).
func DemoDefForSlug(slug string) demoPlayerDef {
	for _, d := range DemoRoster {
		if strings.EqualFold(d.Slug, slug) || strings.EqualFold(d.Gamertag, slug) {
			return d
		}
	}
	return DemoRoster[0]
}

// SharedDBPath retourne le chemin vers la base DuckDB partagée.
// titleSlug : titre courant, vide = halo_infinite. En démo, le chemin est
// title-scopé (data/demo/warehouse pour le titre par défaut, data/demo/titles/{slug}/
// warehouse pour un titre additionnel) — aligné sur le layout produit par seed-demo.
func SharedDBPath(cfg *AppConfig, titleSlug string) string {
	if titleSlug == "" {
		titleSlug = title.DefaultSlug
	}
	if cfg.demoState() {
		return cfg.DemoLayout().SharedDBPath(titleSlug)
	}
	return cfg.PathResolver().SharedDBPath(titleSlug)
}

// MetadataDBPath retourne le chemin vers la metadata.duckdb d'un titre. En démo,
// title-scopé (cf. SharedDBPath) ; DemoFixturesDir vide → repli sur le chemin prod.
func MetadataDBPath(cfg *AppConfig, titleSlug string) string {
	if titleSlug == "" {
		titleSlug = title.DefaultSlug
	}
	if cfg.demoState() && cfg.DemoFixturesDir != "" {
		return cfg.DemoLayout().MetadataDBPath(titleSlug)
	}
	return cfg.PathResolver().MetadataDBPath(titleSlug)
}

// PlayerDBPath retourne le chemin vers la base DuckDB stats d'un joueur.
// titleSlug : titre courant, vide = halo_infinite.
func PlayerDBPath(cfg *AppConfig, titleSlug, gamertag string) string {
	if titleSlug == "" {
		titleSlug = title.DefaultSlug
	}
	return cfg.PathResolver().PlayerDBPath(titleSlug, gamertag)
}

// ReadXUIDFile lit le XUID depuis un fichier texte (1 ligne).
func ReadXUIDFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	xuid := strings.TrimSpace(string(data))
	if xuid == "" {
		return "", fmt.Errorf("xuid.txt vide : %s", path)
	}
	return xuid, nil
}

// PlayerDBFileExists dit si la player DB d'un profil suivi est déjà sur disque pour ce titre.
//
// Pour les chemins de LECTURE qui visitent la base d'un autre joueur (extras des amis d'un
// match) : ouvrir par le pool une base qui n'existe pas échoue quand son dossier est absent
// (ami onboardé, jamais synchronisé), et ce n'est pas une panne. Le dossier et la base sont créés
// par la première synchronisation du joueur (OpenPlayerDB), jamais par une lecture.
//
// En démo, ResolvePlayer lit des fixtures et dit lui-même leur absence, et LoadPlayers ne rend
// que les joueurs dont la base a été seedée : la fonction rend true.
func PlayerDBFileExists(cfg *AppConfig, titleSlug, gamertag string) bool {
	if cfg.demoState() {
		return true
	}
	info, err := os.Stat(PlayerDBPath(cfg, titleSlug, gamertag))
	return err == nil && !info.IsDir()
}
