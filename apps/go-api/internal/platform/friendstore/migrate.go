package friendstore

import (
	"encoding/json"
	"fmt"
	"os"

	"levelup/go-api/internal/domain"
)

// MigrateFromAppSettings initialise les listes d'amis par joueur depuis l'ancien
// réglage GLOBAL `app_settings.friend_gamertags` : chaque profil configuré hérite
// de la liste globale, moins son propre gamertag.
//
// Idempotent : no-op si le fichier des amis existe déjà (même contrat que
// groupstore.MigrateDefault). Retourne le nombre de profils dotés d'une liste.
//
// Le fichier app_settings.json est lu ICI, en map[string]json.RawMessage, et
// SEULE la clé `friend_gamertags` en est extraite : zéro dépendance au champ
// typé AppSettings.FriendGamertags, qui est supprimé par ailleurs. C'est le seul
// endroit du dépôt qui connaît encore ce littéral (garde-rail
// internal/archlint/no_global_friend_gamertags_test.go).
func MigrateFromAppSettings(store *FriendStore, appSettingsPath string, players []domain.PlayerSummary) (int, error) {
	if store == nil {
		return 0, nil
	}
	// Garde idempotente : si le fichier existe, la migration a déjà eu lieu.
	if _, err := os.Stat(store.Path()); err == nil {
		return 0, nil
	} else if !os.IsNotExist(err) {
		return 0, fmt.Errorf("stat player_friends.json : %w", err)
	}

	legacy, err := readLegacyFriendGamertags(appSettingsPath)
	if err != nil {
		return 0, err
	}
	if len(legacy) == 0 {
		return 0, nil
	}

	created := 0
	seen := make(map[string]bool, len(players))
	for _, p := range players {
		// AuthOnly : profil existant seulement pour fournir un refresh token au
		// pool, sans données joueur — aucune liste d'amis à lui donner.
		if p.XUID == "" || p.AuthOnly || seen[p.XUID] {
			continue
		}
		seen[p.XUID] = true
		if _, err := store.Set(p.XUID, p.Gamertag, legacy); err != nil {
			return created, fmt.Errorf("migration amis du profil %q : %w", p.Gamertag, err)
		}
		created++
	}
	return created, nil
}

// readLegacyFriendGamertags extrait la seule clé `friend_gamertags` de
// app_settings.json. Fichier absent → liste vide (pas une erreur : une instance
// neuve n'a rien à migrer).
func readLegacyFriendGamertags(appSettingsPath string) ([]string, error) {
	if appSettingsPath == "" {
		return nil, nil
	}
	data, err := os.ReadFile(appSettingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("lecture app_settings.json : %w", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing app_settings.json : %w", err)
	}
	blob, exists := raw["friend_gamertags"]
	if !exists {
		return nil, nil
	}
	var list []string
	if err := json.Unmarshal(blob, &list); err != nil {
		return nil, fmt.Errorf("parsing friend_gamertags : %w", err)
	}
	return list, nil
}
