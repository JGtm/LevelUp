package halo_infinite

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"levelup/go-api/internal/assets/static"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/games/halo_infinite/film/damagetag"
	"levelup/go-api/internal/games/halo_infinite/film/killicon"
)

// TitleSlug est le slug canonique d'Halo Infinite côté adapter.
const TitleSlug = "halo_infinite"

// weaponIconDir est le sous-dossier des icônes EXTRAITES DU JEU sous
// static/weapons-assets/halo_infinite/. Les PNG y sont nommés par index d'atlas
// (contour-NN.png) : aucun nom de fichier ne prétend désigner une arme — c'est le
// `sprite index` lu dans le tag `weap` qui fait le lien, jamais un nom.
const weaponIconDir = "jeu/"

// weaponImageStemGrenade est le stem du seul PNG dessiné à la main encore servi
// (cf. weaponIconSentinelFiles / weaponIconConceptFiles).
const weaponImageStemGrenade = "Grenade"

// uuidRe matche un UUID v4 — utilisé pour rejeter les map names qui sont en
// fait des UUID bruts (non utilisables comme nom de fichier statique).
var uuidRe = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// mapVariantSuffixes liste les suffixes de variantes mode qui ont des images
// mais ne doivent pas apparaître dans le drawer (modes spécifiques : BTB Heavies,
// Sentry Defense, Firefight sur une map). Les variantes "- Ranked" sont filtrées
// en amont par SQL (NOT LIKE '% - %').
var mapVariantSuffixes = []string{" Heavies", " Sentry Defense", " Firefight"}

// weaponIconSentinelFiles couvre les SENTINELLES produit : des weapon_id
// conventionnels (0 grenade, 1 mêlée, 2 véhicule) qui ne désignent aucun objet du
// jeu et sont donc hors de portée du tag `weap`. Ils sont keyés par ID — jamais par
// nom, la divergence de nom étant précisément le défaut que ce chemin corrige.
//
// La sentinelle 2 (véhicule) est ABSENTE volontairement : aucun dessin ne représente
// « un véhicule » en général, et servir celui d'un véhicule particulier serait une
// icône fausse. Elle n'en avait pas non plus avant ce chantier.
var weaponIconSentinelFiles = map[int64]string{
	0: weaponImageStemGrenade,
	1: "Melee",
}

// weaponIconConceptFiles couvre les grenades RÉELLES du référentiel. Elles ont un
// identifiant filmshell, mais ne sont pas des `weap` : elles vivent en `eqip`+`proj`
// (déclarées par le tag de globals `gggl`) et n'ont donc pas de bloc « UI display
// info » à lire — l'atlas d'armes extrait du jeu ne les porte pas. Mesuré, pas
// supposé : le balayage de TOUS les groupes de tags par cmd/weapon-icons-build ne
// les fait pas sortir.
//
// Elles gardent donc le PNG dessiné à la main, keyé par le TAG (32 bits hauts) et non
// par le nom : les variantes d'une même grenade partagent leur tag et suivent.
var weaponIconConceptFiles = map[uint32]string{
	0x3ad55da4: weaponImageStemGrenade, // Dynamo Grenade
	0xb6dbead8: weaponImageStemGrenade, // Frag Grenade
	0xc1e1bab0: weaponImageStemGrenade, // Plasma Grenade
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

// WeaponImageURL retourne l'URL de l'icône d'une arme à partir de son identifiant
// filmshell. "" si l'arme n'a pas d'icône : un trou d'atlas se rend en repli propre
// côté produit (le libellé), jamais par une icône d'une autre arme.
//
// LA CLÉ EST LE TAG `weap` — les 32 bits HAUTS de l'identifiant (état de l'art §1.1).
// Elle ne dépend ni d'un nom, ni du registre : elle couvre les armes enregistrées,
// leurs VARIANTES et celles qui ne sont PAS au registre, trois cas que ni `name_en`
// ni `weapon_key` ne couvraient ensemble (7 étiquettes sur 42 n'ont pas de
// `weapon_key`, dont deux qui ont une icône).
//
// Ordre : sentinelles produit d'abord (leurs ID tiennent dans les 32 bits bas, donc
// leur tag vaut 0 et ne peut pas les distinguer), puis l'atlas du jeu, puis les
// concepts hors atlas.
func (a *AssetURLAdapter) WeaponImageURL(weaponID int64) string {
	url, _ := a.weaponIcon(weaponID)
	return url
}

// WeaponImageIsTinted : vrai pour les icônes EXTRAITES DU JEU, dont le dessin est
// porté par l'alpha en blanc (mesuré : 100 % des pixels opaques sont blancs purs).
// Faux pour les deux PNG dessinés à la main, qui sont des images finies en couleur —
// les teinter les aplatirait en silhouette.
func (a *AssetURLAdapter) WeaponImageIsTinted(weaponID int64) bool {
	_, tinted := a.weaponIcon(weaponID)
	return tinted
}

// weaponIcon porte l'ORDRE de résolution, une seule fois pour les deux accesseurs.
func (a *AssetURLAdapter) weaponIcon(weaponID int64) (url string, tinted bool) {
	if stem, ok := weaponIconSentinelFiles[weaponID]; ok {
		return static.URL(static.KindWeapon, a.titleSlug, stem, ".png"), false
	}
	tag := uint32(uint64(weaponID) >> 32) //nolint:gosec // troncature voulue : les 32 bits hauts SONT le tag
	if stem, ok := weaponIconFileByTag[tag]; ok {
		return static.URL(static.KindWeapon, a.titleSlug, weaponIconDir+stem, ".png"), true
	}
	if stem, ok := weaponIconConceptFiles[tag]; ok {
		return static.URL(static.KindWeapon, a.titleSlug, stem, ".png"), false
	}
	return "", false
}

// KillSourceIcon résout l'icône de la source de dégât d'une mort — le tag `jpt!` que le
// décodeur de film publie dans `match_kill_events.source_tag`.
//
// Toute l'intelligence vit dans `film/killicon`, hors ligne et versionnée : cet adapter
// ne fait que composer l'URL. Les vignettes viennent de l'atlas KILL FEED (format
// bandeau), pas de l'atlas `contour` : le feed lit en petit, et l'atlas d'armes ne porte
// ni grenade ni mêlée — deux classes qui pèsent 18 % des morts mesurées.
//
// Toujours `tinted` : ces PNG sont des masques blancs portés par l'alpha (mesuré à
// l'extraction). C'est ce qui permet de les teindre à la couleur d'équipe du tueur sans
// produire un asset par couleur.
func (a *AssetURLAdapter) KillSourceIcon(sourceTag uint32) (canonical.KillSourceIcon, bool) {
	ic, ok := killicon.Lookup(sourceTag)
	if !ok {
		return canonical.KillSourceIcon{}, false
	}
	out := canonical.KillSourceIcon{
		WeaponKey: ic.WeaponKey,
		ImageURL:  static.URL(static.KindWeapon, a.titleSlug, weaponIconDir+ic.Sprite, ".png"),
		Tinted:    true,
	}
	// Le nom PROPRE vient de la même table embarquée que l'icône, et n'est publié que si
	// elle autorise à l'afficher (statut AMBIGU exclu). Une mêlée ou une grenade n'a pas
	// de nom propre : l'icône se sert quand même, c'est elle qui porte le sens.
	if l, ok := damagetag.Lookup(sourceTag); ok && l.Publishable() {
		out.Label = l.Name
	}
	return out, true
}

// NeutralDeathIcon résout le pictogramme du TYPE d'une mort que PERSONNE ne revendique —
// chute, hors-limites, ou sa propre source de dégât.
//
// POURQUOI PAS KillSourceIcon : celle-là répond « quelle arme », ce qui est la bonne question
// sur une ligne de kill et la mauvaise sur une mort sans tueur. Y poser l'icône de l'arme
// donnerait à lire un kill par roquette là où le joueur s'est tué tout seul — le jeu lui-même
// affiche un pictogramme de suicide dans ce cas. Toute la décision vit dans `film/killicon`,
// hors ligne et versionnée ; cet adapter ne fait que composer l'URL.
//
// Le premier retour est l'identifiant STABLE du type (jamais un libellé traduit) ; l'image est
// toujours un masque à teindre, comme le reste de l'atlas kill feed. Second retour faux = la
// nature du dégât n'est pas établie : l'appelant garde son repère neutre, JAMAIS l'icône d'une
// autre mort.
//
// N'appeler QUE sur la population des morts sans revendication : sur un kill ordinaire, elle
// rendrait « suicide » pour une mort parfaitement attribuée.
func (a *AssetURLAdapter) NeutralDeathIcon(sourceTag uint32) (kind, imageURL string, ok bool) {
	k, ic, ok := killicon.NeutralDeath(sourceTag)
	if !ok || ic.Sprite == "" {
		return "", "", false
	}
	return k, static.URL(static.KindWeapon, a.titleSlug, weaponIconDir+ic.Sprite, ".png"), true
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
