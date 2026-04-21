package assets

// Kind identifie le type d'asset Halo.
// Chaque Kind correspond à une source de données et un schéma de persistance distincts.
type Kind string

const (
	// KindMedalImage est l'image d'une médaille (PNG).
	// Source : https://gamecms-hacs.svc.halowaypoint.com/hi/Progression/file/medals/{titleID}/{medalID}.png
	// Fallback spritesheet : /hi/Progression/file/medals/sprites/{titleID}.png
	KindMedalImage Kind = "medal-image"

	// KindMapImage est l'image d'une map (PNG).
	// Source locale prioritaire (static/maps/) puis GameCMS ou DiscoveryUGC.
	KindMapImage Kind = "map-image"

	// KindChallengeBadge est l'image d'insigne d'un défi (PNG).
	// Source : https://gamecms-hacs.svc.halowaypoint.com/hi/waypoint/file/images/{stem}.png
	KindChallengeBadge Kind = "challenge-badge"

	// KindBPTrackImage est l'image de couverture d'un track Battle Pass (PNG).
	// Source : GameCMS, chemin fourni dans la définition du track.
	KindBPTrackImage Kind = "bp-track-image"

	// KindBPBackground est l'image de fond d'un track Battle Pass (PNG).
	// Source : GameCMS, chemin fourni dans la définition du track.
	KindBPBackground Kind = "bp-background"

	// KindMedalMetadata est la liste JSON des métadonnées de médailles.
	// Source : https://gamecms-hacs.svc.halowaypoint.com/hi/Progression/file/Metadata/Metadata.json
	KindMedalMetadata Kind = "medal-meta"

	// KindChallengeDefinition est la définition JSON d'un défi.
	// Source : https://gamecms-hacs.svc.halowaypoint.com/hi/Progression/file/{challengePath}
	KindChallengeDefinition Kind = "challenge-def"

	// KindRewardTrackDefinition est la définition JSON d'un Reward Track (BP).
	// Source : https://gamecms-hacs.svc.halowaypoint.com/hi/Progression/file/{trackPath}
	KindRewardTrackDefinition Kind = "track-def"

	// KindAssetTranslation est la traduction JSON d'un asset (map, playlist, game variant…).
	// Source : DiscoveryUGC API.
	KindAssetTranslation Kind = "asset-translation"
)

// allKinds liste tous les kinds valides (pour validation).
var allKinds = map[Kind]struct{}{
	KindMedalImage:            {},
	KindMapImage:              {},
	KindChallengeBadge:        {},
	KindBPTrackImage:          {},
	KindBPBackground:          {},
	KindMedalMetadata:         {},
	KindChallengeDefinition:   {},
	KindRewardTrackDefinition: {},
	KindAssetTranslation:      {},
}

// Valid retourne true si k est un Kind connu.
func (k Kind) Valid() bool {
	_, ok := allKinds[k]
	return ok
}

// IsBinary retourne true si ce Kind correspond à des bytes binaires (image).
func (k Kind) IsBinary() bool {
	switch k {
	case KindMedalImage, KindMapImage, KindChallengeBadge, KindBPTrackImage, KindBPBackground:
		return true
	}
	return false
}
