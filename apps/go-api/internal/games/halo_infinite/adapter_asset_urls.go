package halo_infinite

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"levelup/go-api/internal/assets/static"
)

// TitleSlug est le slug canonique d'Halo Infinite côté adapter.
const TitleSlug = "halo_infinite"

// Stems de fichiers image partagés par plusieurs entrées de weaponImageFiles.
// Externalisés pour éviter les doublons signalés par goconst.
const (
	weaponImageStemGrenade = "Grenade"
	weaponImageStemSword   = "Sword"
)

// uuidRe matche un UUID v4 — utilisé pour rejeter les map names qui sont en
// fait des UUID bruts (non utilisables comme nom de fichier statique).
var uuidRe = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// mapVariantSuffixes liste les suffixes de variantes mode qui ont des images
// mais ne doivent pas apparaître dans le drawer (modes spécifiques : BTB Heavies,
// Sentry Defense, Firefight sur une map). Les variantes "- Ranked" sont filtrées
// en amont par SQL (NOT LIKE '% - %').
var mapVariantSuffixes = []string{" Heavies", " Sentry Defense", " Firefight"}

// weaponImageFiles mappe le name_en d'une arme vers le stem du fichier PNG
// dans static/weapons-assets/halo_infinite/. Armes sans entrée → ImageURL vide.
var weaponImageFiles = map[string]string{
	"Grenade":               weaponImageStemGrenade,
	"Frag Grenade":          weaponImageStemGrenade,
	"Plasma Grenade":        weaponImageStemGrenade,
	"Dynamo Grenade":        weaponImageStemGrenade,
	"Melee":                 "Melee",
	"Bandit Evo":            "Bandit",
	"M392 Bandit":           "Bandit",
	"BR75":                  "BR75",
	"Cindershot":            "Cremator",
	"CQS48 Bulldog":         "Bulldog",
	"Disruptor":             "Disruptor",
	"Fuel Rod SPNKr":        "M41",
	"M41 SPNKr":             "M41",
	"Gravity Hammer":        "Hammer",
	"Diminisher of Hope":    "Hammer",
	"Rushdown Hammer":       "Hammer",
	"Heatwave":              "Heatwave",
	"MA40 AR":               "MA40",
	"MA5K Avenger":          "Storm",
	"Mangler":               "Mangler",
	"MLRS-2 Hydra":          "Hydra",
	"Mk51 Sidekick":         "Sidekick",
	"Mutilator":             "Mutilator",
	"Needler":               "Needler-1",
	"Plasma Pistol":         "Plasma",
	"Pulse Carbine":         "Carabine",
	"Vestige Carbine":       "Carabine",
	"Ravager":               "Ravager",
	"S7 Sniper":             "Sniper-S7",
	"Sentinel Beam":         "Sentinel",
	"Shock Rifle":           "Shock-rifle",
	"Shock Rifle (Ranked)":  "Shock-rifle",
	"Skewer":                "Skewer",
	"Stalker Rifle":         "Stalker",
	"VK78 Commando":         "Commando",
	"Energy Sword":          weaponImageStemSword,
	"Duelist Energy Sword":  weaponImageStemSword,
	"Elite Bloodblade":      weaponImageStemSword,
	"Infected Energy Sword": weaponImageStemSword,
}

// AssetURLAdapter implémente games.TitleAssetURLAdapter pour Halo Infinite.
//
// Composition path déléguée à internal/assets/static (couche 2 SRP). Les URLs
// émises sont au format /static/{folder}/halo_infinite/X.{ext} en cohérence
// avec l'arborescence FS title-scopée (post-Phase 6.5 du plan finition
// multi-titres).
type AssetURLAdapter struct {
	titleSlug    string
	mapImageExts map[string]string // name → ".jpg"|".png" ; nil = mode permissif (tests)
}

// NewAssetURLAdapter construit un AssetURLAdapter pour Halo Infinite.
func NewAssetURLAdapter() *AssetURLAdapter {
	return &AssetURLAdapter{titleSlug: TitleSlug}
}

// WithMapImagesDir scanne dir et construit la liste autorisée des images de map.
// Seuls les fichiers .jpg/.png sont indexés ; toute autre extension est ignorée.
// Après appel, MapImageURL retourne "" pour tout nom absent du répertoire.
// En cas d'erreur de lecture du répertoire, l'adapter reste en mode permissif.
func (a *AssetURLAdapter) WithMapImagesDir(dir string) *AssetURLAdapter {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return a // répertoire absent ou inaccessible → mode permissif
	}
	exts := make(map[string]string, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext != ".jpg" && ext != ".png" {
			continue
		}
		name := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		exts[name] = ext
	}
	if len(exts) > 0 {
		a.mapImageExts = exts
	}
	return a
}

// TitleSlug retourne "halo_infinite".
func (a *AssetURLAdapter) TitleSlug() string { return a.titleSlug }

// pngMapNames : set des map names dont l'extension par défaut est .png (mode permissif).
var pngMapNames = map[string]struct{}{
	"Aquarius":                 {},
	"Bazaar":                   {},
	"Behemoth":                 {},
	"Breaker":                  {},
	"Breaker Heavies":          {},
	"Catalyst":                 {},
	"Deadlock":                 {},
	"Deadlock Heavies":         {},
	"Highpower":                {},
	"Highpower Heavies":        {},
	"Highpower Sentry Defense": {},
	"Launch Site":              {},
	"Recharge":                 {},
	"Streets":                  {},
}

// MapImageURL retourne l'URL de l'image d'une map.
// Si WithMapImagesDir a été appelé, seuls les noms avec un fichier image réel
// retournent une URL — les variantes mode+map sans image retournent "".
// Les variantes avec suffixe connu (Heavies, Sentry Defense, Firefight) sont
// toujours exclues même si un fichier image existe pour elles.
// Encode les espaces du nom en %20 (pas net/url.PathEscape pour préserver les "/").
func (a *AssetURLAdapter) MapImageURL(mapName string) string {
	mapName = strings.TrimSpace(mapName)
	if mapName == "" || uuidRe.MatchString(mapName) {
		return ""
	}
	if a.mapImageExts != nil {
		return a.mapImageURLFromDir(mapName)
	}
	ext := defaultMapImageExt(mapName)
	return static.URL(static.KindMap, a.titleSlug, encodeSpaces(mapName), ext)
}

// mapImageURLFromDir résout l'URL depuis le répertoire d'images configuré.
// Exclut les variantes mode+map sans image dédiée.
func (a *AssetURLAdapter) mapImageURLFromDir(mapName string) string {
	for _, suffix := range mapVariantSuffixes {
		if strings.HasSuffix(mapName, suffix) {
			return ""
		}
	}
	ext, ok := a.mapImageExts[mapName]
	if !ok {
		return ""
	}
	return static.URL(static.KindMap, a.titleSlug, encodeSpaces(mapName), ext)
}

// defaultMapImageExt retourne l'extension par défaut en mode permissif (sans répertoire).
// .png pour les maps connues, .jpg sinon.
func defaultMapImageExt(mapName string) string {
	if _, ok := pngMapNames[mapName]; ok {
		return ".png"
	}
	if strings.HasSuffix(mapName, " - Ranked") {
		return ".png"
	}
	return ".jpg"
}

// WeaponImageURL retourne l'URL de l'image d'une arme à partir de son name_en.
// Retourne "" si aucun fichier ne correspond au nom.
func (a *AssetURLAdapter) WeaponImageURL(nameEN string) string {
	stem, ok := weaponImageFiles[nameEN]
	if !ok {
		return ""
	}
	return static.URL(static.KindWeapon, a.titleSlug, stem, ".png")
}

// waypointMatchBaseURL est la base des pages de détail de match sur Waypoint.
const waypointMatchBaseURL = "https://www.halowaypoint.com/halo-infinite"

// MatchWebURL retourne l'URL de la page publique d'un match sur Waypoint
// (forme sans joueur : /halo-infinite/matches/{matchID}).
func (a *AssetURLAdapter) MatchWebURL(matchID string) string {
	if strings.TrimSpace(matchID) == "" {
		return ""
	}
	return waypointMatchBaseURL + "/matches/" + matchID
}

// PlayerMatchWebURL retourne l'URL de la page d'un match POUR un joueur sur
// Waypoint (/halo-infinite/players/{gamertag}/matches/{matchID}). "" si le
// gamertag ou le matchID est vide.
func (a *AssetURLAdapter) PlayerMatchWebURL(gamertag, matchID string) string {
	gt := strings.TrimSpace(gamertag)
	if gt == "" || strings.TrimSpace(matchID) == "" {
		return ""
	}
	return waypointMatchBaseURL + "/players/" + gt + "/matches/" + matchID
}

// MedalImageURL retourne l'URL de l'icône d'une médaille à partir de son ID numérique.
func (a *AssetURLAdapter) MedalImageURL(medalID uint64) string {
	return static.URL(static.KindMedal, a.titleSlug, strconv.FormatUint(medalID, 10), ".png")
}

// CSRRankImageURL retourne l'URL du badge d'un rang CSR.
// Format de l'id : 120px-HINF-CSR_{Tier}{SubTier} (ex: 120px-HINF-CSR_Gold3).
func (a *AssetURLAdapter) CSRRankImageURL(tier string, subTier int) string {
	tier = strings.TrimSpace(tier)
	if tier == "" {
		return ""
	}
	id := fmt.Sprintf("120px-HINF-CSR_%s%d", tier, subTier)
	return static.URL(static.KindCSRRank, a.titleSlug, id, ".png")
}

// CSRRankImageURLOnyx retourne l'URL du badge Onyx (sans sub-tier).
func (a *AssetURLAdapter) CSRRankImageURLOnyx() string {
	return static.URL(static.KindCSRRank, a.titleSlug, "120px-HINF-CSR_Onyx", ".png")
}

// encodeSpaces remplace les espaces par %20 sans encoder les autres caractères.
// Utilisé pour les map names qui peuvent contenir des espaces et tirets.
func encodeSpaces(s string) string {
	if !strings.ContainsRune(s, ' ') {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + 6)
	for _, c := range s {
		if c == ' ' {
			b.WriteString("%20")
		} else {
			b.WriteRune(c)
		}
	}
	return b.String()
}
