package port

import "context"

// FriendsExtrasResolver charge les FriendMatchExtras pour la liste de xuids
// fournie. Un xuid sans profil suivi ou sans données sur ce match ne figure
// simplement pas dans la map de retour : c'est une absence. Une player DB d'ami qui
// ne s'ouvre pas ou dont la lecture échoue est une PANNE : l'erreur (errors.Join, une
// entrée par ami en cause) la remonte, et la map reste valide pour les autres amis.
type FriendsExtrasResolver func(ctx context.Context, matchID string, gameVariantName string, xuids []string) (map[string]FriendMatchExtras, error)
