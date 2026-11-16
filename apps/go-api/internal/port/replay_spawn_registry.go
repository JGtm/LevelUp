package port

import "context"

// replay_spawn_registry.go — CE QUE LA PREVISION DE REAPPARITION LIT DU REGISTRE DES MATCHS : la
// carte, la variante et la categorie de mode d'un match, et les matchs d'une carte.
//
// Capacite OPTIONNELLE d'un [ReplayMapNameRepo] (comme [ReplayModeGate]) : le service de rejeu la
// detecte par assertion ; sans elle la prevision se sert `map_unknown`, jamais en erreur.

// SpawnMatch est un match du registre tel que la prevision en a besoin. MatchID est l'identifiant
// complet ; Variant et Category sont les champs bruts dont la famille de modes se deduit.
type SpawnMatch struct {
	MatchID  string
	MapID    string
	Variant  string
	Category string
}

// ReplaySpawnRegistry lit les matchs JcJ du registre.
type ReplaySpawnRegistry interface {
	// SpawnMatchOf rend le match du registre. [ErrMatchMapUnknown] quand le registre ne le porte
	// pas ou ne lui connait aucune carte.
	SpawnMatchOf(ctx context.Context, matchID string) (SpawnMatch, error)
	// SpawnMatchesOnMap rend les matchs JcJ d'une carte (identifiant d'asset UGC), le plus
	// ancien d'abord. Les matchs Firefight n'y sont pas.
	SpawnMatchesOnMap(ctx context.Context, mapID string) ([]SpawnMatch, error)
}
