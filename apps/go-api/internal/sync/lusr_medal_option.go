package sync

import (
	"context"
	"database/sql"

	"levelup/go-api/internal/sync/skill"
)

// MedalExploitOption est l'option de rejeu LUSR qui charge le score d'exploit médailles
// du joueur (difficulté des médailles de la metadata DB du titre, médailles gagnées de la
// base partagée) : 6e des 8 composantes écrites dans lusr_component_history. Tout appel du
// rejeu LUSR qui écrit en base joueur la passe ; sans elle la composante n'est jamais
// écrite (garde-rail : archlint/lusr_medal_exploit_option_test.go). Chargement
// best-effort : metadata absente ou illisible, composante à zéro, journalisé.
func MedalExploitOption(metadataDBPath, xuid string) skill.ShadowOption {
	return skill.WithMedalExploit(func(ctx context.Context, sharedDB *sql.DB) map[string]float64 {
		return loadMedalExploitMap(ctx, metadataDBPath, sharedDB, xuid)
	})
}
