package halo_5

import (
	"strings"

	"levelup/go-api/internal/games/canonical"
)

// AssetURLAdapter implémente games.TitleAssetURLAdapter pour Halo 5.
//
// Couche 3 PURE (cf. SRP 4-couches du plan finition multi-titres) : aucun accès
// DB, aucune ouverture de fichier. Toutes les données (URLs CDN officielles des
// maps / armes, résolveur CSR) sont injectées au boot via les builders With*.
//
// Contrairement à l'adapter Halo Infinite (qui compose des chemins /static/...
// locaux), Halo 5 expose des URLs CDN officielles seedées dans sa metadata.duckdb
// isolée (cmd/h5-metadata-fetch depuis l'API Metadata officielle). L'adapter se
// contente donc d'un lookup direct identifiant → URL.
//
// Les méthodes retournent "" quand l'identifiant est inconnu/invalide — les
// callers (Match View header, badge CSR, drawer) dégradent gracieusement.
type AssetURLAdapter struct {
	// mapURLByID indexe l'URL d'image par map_asset_id (GUID MapId — identité
	// stable d'une map Halo 5). C'est l'identifiant réellement passé par le
	// header de la Match View (meta.MapName brut = GUID quand asset_translations
	// ne résout pas le nom canonique pour h5).
	mapURLByID map[string]string
	// mapURLByName indexe l'URL d'image par nom canonique (name_canonical), pour
	// le cas où le caller passe un nom EN plutôt que le GUID.
	mapURLByName map[string]string
	// weaponURLByName indexe l'URL d'icône d'arme par name_en officiel.
	weaponURLByName map[string]string
	// csrResolver retourne l'URL du badge CSR pour (designation, tier_id).
	// Injecté depuis csr_designations (server.go). Nil = pas de seed CSR.
	csrResolver func(designation string, subTier int) string
	// teamNameResolver retourne le libellé d'équipe localisé pour (team_id, locale).
	// Injecté depuis team_colors (server.go). Nil = pas de seed team_colors → la
	// Match View retombe sur son libellé d'équipe existant (dégradation gracieuse).
	teamNameResolver func(teamID int, locale string) string
	// teamColorResolver retourne la couleur d'identité hex (#RRGGBB) pour un team_id.
	// Injecté depuis team_colors.color (server.go). Nil = pas de seed → la Match View
	// retombe sur son accent d'équipe existant (dégradation gracieuse).
	teamColorResolver func(teamID int) string
}

// NewAssetURLAdapter construit un AssetURLAdapter Halo 5 vide. Les données sont
// ajoutées via les builders WithMaps / WithWeapons / WithCSRResolver.
func NewAssetURLAdapter() *AssetURLAdapter {
	return &AssetURLAdapter{
		mapURLByID:      map[string]string{},
		mapURLByName:    map[string]string{},
		weaponURLByName: map[string]string{},
	}
}

// WithMaps indexe les maps (ID = map_asset_id GUID, NameEN = name_canonical,
// ImageURL = URL CDN officielle). Les entrées sans ImageURL sont ignorées.
func (a *AssetURLAdapter) WithMaps(maps []canonical.AssetMeta) *AssetURLAdapter {
	for _, m := range maps {
		url := strings.TrimSpace(m.ImageURL)
		if url == "" {
			continue
		}
		if id := strings.TrimSpace(m.ID); id != "" {
			a.mapURLByID[id] = url
		}
		if name := strings.TrimSpace(m.NameEN); name != "" {
			a.mapURLByName[name] = url
		}
	}
	return a
}

// WithWeapons indexe les armes par name_en (ImageURL = icon_url officielle).
// Les entrées sans ImageURL sont ignorées.
func (a *AssetURLAdapter) WithWeapons(weapons []canonical.AssetMeta) *AssetURLAdapter {
	for _, w := range weapons {
		url := strings.TrimSpace(w.ImageURL)
		if url == "" {
			continue
		}
		if name := strings.TrimSpace(w.NameEN); name != "" {
			a.weaponURLByName[name] = url
		}
	}
	return a
}

// WithCSRResolver injecte le résolveur de badge CSR (designation, tier_id → URL).
// Réutilise le résolveur construit depuis csr_designations au boot.
func (a *AssetURLAdapter) WithCSRResolver(f func(designation string, subTier int) string) *AssetURLAdapter {
	a.csrResolver = f
	return a
}

// WithTeamNameResolver injecte le résolveur de libellé d'équipe (team_id, locale → nom
// localisé), construit depuis team_colors au boot. Sans lui, TeamName renvoie "" et la
// Match View garde son libellé d'équipe existant.
func (a *AssetURLAdapter) WithTeamNameResolver(f func(teamID int, locale string) string) *AssetURLAdapter {
	a.teamNameResolver = f
	return a
}

// TeamName retourne le libellé d'équipe localisé pour un team_id (Halo 5 :
// « Rouge »/« Red »). "" si aucun résolveur n'est injecté (team_colors vide/absent) ou
// team_id inconnu. `locale` : "en" → nom EN, sinon nom FR (défaut projet FR-first).
// Implémente la capability optionnelle consommée par le service Match View.
func (a *AssetURLAdapter) TeamName(teamID int, locale string) string {
	if a.teamNameResolver == nil {
		return ""
	}
	return a.teamNameResolver(teamID, locale)
}

// WithTeamColorResolver injecte le résolveur de couleur d'identité d'équipe
// (team_id → hex #RRGGBB), construit depuis team_colors.color au boot. Sans lui,
// TeamColor renvoie "" et la Match View garde son accent d'équipe existant.
func (a *AssetURLAdapter) WithTeamColorResolver(f func(teamID int) string) *AssetURLAdapter {
	a.teamColorResolver = f
	return a
}

// TeamColor retourne la couleur d'identité hex (#RRGGBB) d'un team_id (Halo 5). "" si
// aucun résolveur n'est injecté (team_colors vide/absent) ou team_id inconnu. Implémente
// la capability optionnelle consommée par le service Match View.
func (a *AssetURLAdapter) TeamColor(teamID int) string {
	if a.teamColorResolver == nil {
		return ""
	}
	return a.teamColorResolver(teamID)
}

// TitleSlug retourne "halo_5".
func (a *AssetURLAdapter) TitleSlug() string { return TitleSlug }

// MapImageURL résout l'URL d'image d'une map. Le caller (Match View header)
// passe l'identifiant brut de map_registry : pour Halo 5 c'est le GUID MapId
// (asset_translations ne résout pas de nom canonique h5). On tente donc d'abord
// le lookup par ID (GUID), puis par nom canonique en repli. "" si vide/inconnu.
func (a *AssetURLAdapter) MapImageURL(mapName string) string {
	mapName = strings.TrimSpace(mapName)
	if mapName == "" {
		return ""
	}
	if url, ok := a.mapURLByID[mapName]; ok {
		return url
	}
	if url, ok := a.mapURLByName[mapName]; ok {
		return url
	}
	return ""
}

// WeaponImageURL résout l'URL d'icône d'une arme par son name_en officiel.
// "" si vide/inconnu.
func (a *AssetURLAdapter) WeaponImageURL(nameEN string) string {
	nameEN = strings.TrimSpace(nameEN)
	if nameEN == "" {
		return ""
	}
	if url, ok := a.weaponURLByName[nameEN]; ok {
		return url
	}
	return ""
}

// MatchWebURL / PlayerMatchWebURL : Halo 5 n'a pas de page de détail de match
// publique équivalente à Waypoint → "" (pas de lien externe, F3).
func (a *AssetURLAdapter) MatchWebURL(_ string) string          { return "" }
func (a *AssetURLAdapter) PlayerMatchWebURL(_, _ string) string { return "" }

// MedalImageURL : Halo 5 expose ses icônes de médaille sous forme de SPRITE
// (feuille + offset left/top/width/height), non exprimable en une URL PNG simple.
// Le rendu sprite côté front est un chantier distinct (G2 / D.1) ; cet adapter
// retourne donc "". Les champs sprite restent disponibles via AssetMeta dans la
// couche asset metadata, pas ici.
func (a *AssetURLAdapter) MedalImageURL(_ uint64) string { return "" }

// CSRRankImageURL retourne l'URL du badge CSR pour (tier=designation, subTier=tier_id).
// "" si aucun résolveur n'est injecté ou si la designation/tier est inconnue.
func (a *AssetURLAdapter) CSRRankImageURL(tier string, subTier int) string {
	if a.csrResolver == nil {
		return ""
	}
	return a.csrResolver(tier, subTier)
}

// CSRRankImageURLOnyx retourne l'URL du badge Onyx. Onyx est un palier unique
// (pas de sous-palier I-IV) mais csr_designations le stocke à tier_id=1 — les
// sous-paliers y sont 1-indexés (pas de tier_id=0) → la clé de lookup est
// `onyx|1`. Aligné sur la compensation du chemin Home/Explorer
// (home_repo_skill_peak.go : sub=1 pour Onyx). Passer 0 provoquait un miss de
// lookup → badge Onyx sans image sur la Match View (scoreboard/header/amis).
func (a *AssetURLAdapter) CSRRankImageURLOnyx() string {
	if a.csrResolver == nil {
		return ""
	}
	return a.csrResolver("Onyx", 1)
}
