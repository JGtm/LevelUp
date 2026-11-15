package port

// replay_registry.go — ce que le registre des matchs dit d'un match pour construire son rejeu : son
// identité complète, sa carte, et si le rejeu sait fabriquer son mode.

import "context"

// ReplayRegistryMatch : une ligne du registre des matchs, telle que la mise en file d'un rejeu la lit.
type ReplayRegistryMatch struct {
	// ID est le match_id COMPLET (le registre est indexé par l'identifiant complet).
	ID string
	// RawMapName est le nom de carte brut du registre, sans espaces de bord ; vide si absent.
	RawMapName string
	// MapID est l'identifiant de carte, sans espaces de bord ; vide si absent.
	MapID string
	// UnmanagedMode dit que le rejeu ne fabrique pas le mode du match (règle du prédicat
	// sqlfrag.SQLModeNonGere).
	UnmanagedMode bool
}

// ReplayRegistryRepo lit le registre des matchs (base partagée) et les noms de carte (metadata).
type ReplayRegistryRepo interface {
	// MatchesByIDOrPrefix rend au plus trois matchs dont l'identifiant est matchID ou commence par lui.
	MatchesByIDOrPrefix(ctx context.Context, matchID string) ([]ReplayRegistryMatch, error)
	// EnglishMapName rend le nom EN d'une carte (traduction d'asset `en-US`) ; vide si metadata est
	// absente ou ne la connaît pas, l'erreur d'une lecture impossible.
	EnglishMapName(ctx context.Context, mapID string) (string, error)
	// UnmanagedModeMatchIDs rend les identifiants complets des matchs dont le rejeu ne fabrique pas le mode.
	UnmanagedModeMatchIDs(ctx context.Context) ([]string, error)
}
