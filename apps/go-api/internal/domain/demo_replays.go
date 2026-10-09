package domain

// DemoReplayIndex : l'index des rejeux FIGÉS de la démo (`title.DemoLayout.ReplayIndexPath`).
// Écrit par `seed-demo` (internal/ops/seed_demo_replays.go), lu par le serveur démo pour deux
// décisions et deux seulement :
//   - QUELS matchs ont un rejeu servi en démo (garde du rejeu, décision D-2) ;
//   - comment MASQUER les identités réelles que l'artefact porte (décision D-1 : l'artefact
//     n'est jamais modifié, le masque s'applique au document servi).
//
// Le fichier porte des xuid RÉELS : il reste sur le disque de la démo, il n'est jamais servi.
type DemoReplayIndex struct {
	Matches    []DemoReplayIndexEntry `json:"matches"`
	Identities []DemoReplayIdentity   `json:"identities"`
}

// DemoReplayIndexEntry : un rejeu servi par la démo.
type DemoReplayIndexEntry struct {
	MatchID string `json:"match_id"`
	// Mode est la famille de mode représentée (clé du manifeste, jamais un libellé).
	Mode string `json:"mode"`
	Map  string `json:"map,omitempty"`
	// SchemaVersion est la version de schéma de l'artefact embarqué.
	SchemaVersion int `json:"schema_version"`
}

// DemoReplayIdentity : une identité réelle et l'identité démo qui la remplace à l'affichage.
type DemoReplayIdentity struct {
	XUID         string `json:"xuid"`
	DemoXUID     string `json:"demo_xuid"`
	DemoGamertag string `json:"demo_gamertag"`
}
