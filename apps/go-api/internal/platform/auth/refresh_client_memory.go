// Package auth — refresh_client_memory.go : mémoire du client OAuth qui a réellement rafraîchi le
// jeton de chaque joueur.
//
// Un refresh_token émis par le client Xbox natif est refusé par l'application Azure
// (invalid_grant) avant que le repli natif ne le rafraîchisse : un appel perdu, et une ligne de
// journal, à chaque rafraîchissement. La famille enregistrée dans le magasin de jetons ne dit pas
// quel client a rafraîchi (elle vient du ticket XBL, voir refreshAccessTokenForUser) : la mémoire
// retient donc la famille rendue par l'échange lui-même, pour que le rafraîchissement suivant du
// même joueur commence par ce client.
//
// La mémoire vit dans le processus, jamais dans les fichiers de jetons : aucun changement de
// format sur le chemin de rotation, et au pire un appel perdu après un redémarrage. Elle ne
// contient ni jeton ni secret, seulement xuid -> famille.
package auth

import (
	"context"
	"sync"
)

// refreshXUIDCtxKey porte le xuid du joueur dont on rafraîchit le jeton, jusqu'à
// ExchangeRefreshTokenWithRotation (même patron que WithTokenClientFamily : aucune signature
// publique ne change).
type refreshXUIDCtxKey struct{}

// WithRefreshXUID pose le xuid du joueur dont le refresh_token va être échangé. Un xuid vide laisse
// le contexte inchangé : l'échange suit alors l'ordre par défaut (Azure d'abord).
func WithRefreshXUID(ctx context.Context, xuid string) context.Context {
	if xuid == "" {
		return ctx
	}
	return context.WithValue(ctx, refreshXUIDCtxKey{}, xuid)
}

// RefreshXUIDFromContext lit le xuid posé par WithRefreshXUID ("" = inconnu). Exportée comme
// TokenClientFamilyFromContext : les fournisseurs et leurs tests, hors du paquet, la lisent.
func RefreshXUIDFromContext(ctx context.Context) string {
	x, _ := ctx.Value(refreshXUIDCtxKey{}).(string)
	return x
}

// refreshClientMemory : xuid -> famille du dernier client OAuth qui a réussi un rafraîchissement.
// Sûre pour un accès concurrent.
type refreshClientMemory struct {
	mu       sync.RWMutex
	familles map[string]string
}

// refreshClients est la mémoire du processus, partagée par tous les chemins de rafraîchissement.
var refreshClients = &refreshClientMemory{familles: make(map[string]string)}

// natifDabord dit si le dernier rafraîchissement réussi de ce joueur est passé par le client natif.
// Faux pour un xuid vide ou inconnu (table vide après un redémarrage comprise).
func (m *refreshClientMemory) natifDabord(xuid string) bool {
	if xuid == "" {
		return false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.familles[xuid] == TokenFamilyXboxNative
}

// retenir enregistre la famille qui vient de réussir. Une famille vide (échec) ou un xuid vide ne
// change rien : seul un succès fait suivre la mémoire.
func (m *refreshClientMemory) retenir(xuid, famille string) {
	if xuid == "" || famille == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.familles[xuid] = famille
}
