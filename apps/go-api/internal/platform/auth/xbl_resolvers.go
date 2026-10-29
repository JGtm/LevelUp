// Package auth — xbl_resolvers.go : façade des résolveurs gamertag -> xuid de Xbox Live (profil
// Xbox, PeopleHub ; protocole dans `internal/haloapi/xblauth`, plan D3 lot 2). La façade garde les
// constructeurs historiques : client nil -> http.DefaultClient, sans garde `netguard` (comme avant
// le déplacement : ces résolutions dépendent d'un jeton frais, coupé en amont en mode démo).
package auth

import (
	"context"
	"net/http"

	"levelup/go-api/internal/haloapi/xblauth"
)

// XboxProfileResolver résout gamertag -> xuid via l'endpoint profil Xbox Live.
type XboxProfileResolver = xblauth.XboxProfileResolver

// PeopleHubResolver résout gamertag -> xuid via PeopleHub (recherche).
type PeopleHubResolver = xblauth.PeopleHubResolver

// NewXboxProfileResolver construit le résolveur. headerFn fournit un header
// "XBL3.0 x=<uhs>;<xsts>" valide (XSTS, audience http://xboxlive.com) — le MÊME
// que NewPeopleHubResolver (réutilisable tel quel). Le caller mémoïse/rafraîchit
// le token (cf. CachedHeaderProvider). client nil -> http.DefaultClient.
func NewXboxProfileResolver(client *http.Client, headerFn func(ctx context.Context) (string, error)) *XboxProfileResolver {
	if client == nil {
		client = http.DefaultClient
	}
	return xblauth.NewXboxProfileResolver(client, headerFn)
}

// NewPeopleHubResolver construit un résolveur. headerFn fournit un header
// "XBL3.0 x=<hash>;<token>" valide (RTA XSTS, audience http://xboxlive.com) —
// invoqué à chaque résolution ; charge au fournisseur de mémoïser/rafraîchir le
// token avant expiration (~quelques heures). client nil -> http.DefaultClient.
func NewPeopleHubResolver(client *http.Client, headerFn func(ctx context.Context) (string, error)) *PeopleHubResolver {
	if client == nil {
		client = http.DefaultClient
	}
	return xblauth.NewPeopleHubResolver(client, headerFn)
}
