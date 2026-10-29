package haloadapter

// discovery.go — actifs Discovery UGC (playlists, cartes, couples carte-mode, variantes de jeu) et
// stats complètes d'un match pour le peuplement des traductions d'actifs.
//
// Discovery exige un jeton Spartan + 343-clearance (sonde anonyme du 2026-07-25 : 401) : ce sont
// les jetons statiques du fournisseur ([Provider.WithTokens], venus du pool unifié), jamais ceux
// du contexte. `version_id` est requis (404 sans, 400 pour « latest »). Voie SANS jeton connue
// (non câblée) : la page publique d'un actif UGC du site Waypoint sert le JSON complet de l'actif
// dans son `__NEXT_DATA__`, et le stockage des blobs UGC est en lecture anonyme.

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/haloapi/infinite"
)

// AssetType : les types d'actifs de Discovery UGC.
type AssetType = infinite.AssetType

// Les types d'actifs.
const (
	AssetTypeMap         = infinite.AssetTypeMap
	AssetTypePlaylist    = infinite.AssetTypePlaylist
	AssetTypePair        = infinite.AssetTypePair
	AssetTypeGameVariant = infinite.AssetTypeGameVariant
)

// DiscoveryAsset : un actif lu dans Discovery UGC, nom localisé et URL de miniature.
type DiscoveryAsset = infinite.DiscoveryAsset

// MatchInfoAsset : la forme {AssetId, VersionId} d'un actif dans `MatchInfo`.
type MatchInfoAsset = infinite.MatchInfoAsset

// AssetTypeToMatchInfoKey : la clé JSON de `MatchInfo` (stats d'un match) qui porte l'actif d'un
// type, pour extraire les version_id des stats de match.
var AssetTypeToMatchInfoKey = tableDesClesMatchInfo()

func tableDesClesMatchInfo() map[AssetType]string {
	out := map[AssetType]string{}
	for _, t := range []AssetType{AssetTypeMap, AssetTypePlaylist, AssetTypePair, AssetTypeGameVariant} {
		if cle, ok := infinite.CleMatchInfo(t); ok {
			out[t] = cle
		}
	}
	return out
}

// FetchAsset récupère un actif de Discovery UGC dans la langue `lang` (Accept-Language) et rend son
// nom localisé. `titleID` n'est pas un segment de l'adresse (le segment de jeu vient du titre du
// contexte) : il est gardé pour la signature.
func (p *Provider) FetchAsset(ctx context.Context, assetType AssetType, _ string, assetID, versionID, lang string) (*DiscoveryAsset, error) {
	api := p.api(ctx, surfaceDiscoveryGet, infinite.RetryDiscovery(), infinite.CleDiscoveryUGC)
	return api.FetchAsset(ctx, authDe(p.staticTokens),
		infinite.RequeteActif{Type: assetType, AssetID: assetID, VersionID: versionID, Langue: lang})
}

// FetchMatchStats récupère les stats complètes d'un match (JSON brut) par le chemin joueur, avec
// un xuid arbitraire : `MatchInfo` est global au match. Jetons requis, passés par l'appelant.
func (p *Provider) FetchMatchStats(ctx context.Context, matchID string, tokens *domain.HaloTokens) (map[string]interface{}, error) {
	stats, err := p.api(ctx, surfaceProviderGet, infinite.RetryProvider()).FetchMatchStats(ctx, authDe(tokens), matchID)
	if err != nil {
		slog.DebugContext(ctx, "FetchMatchStats: request failed", "match_id", matchID, "err", err)
		return nil, err
	}
	return stats, nil
}
