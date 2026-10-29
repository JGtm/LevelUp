package sync

import (
	"levelup/go-api/internal/games/halo_infinite/haloadapter"
	"levelup/go-api/internal/sync/filmfetch"
)

// haloclient_reexport.go — ré-exports du client Halo Infinite de la synchronisation. Le client
// vit dans l'adaptateur `games/halo_infinite/haloadapter` (sur la brique `internal/haloapi`), la
// lecture des films dans `sync/filmfetch` (plan D3, lot 4). Les alias conservent les noms
// historiques : sync ET les appelants externes (qui utilisent sync.HaloAPIClient,
// sync.MatchSkillData…) restent inchangés — zéro requalification. Le pont DTO->ParticipantRow
// (MergeSkillIntoParticipants) reste dans sync (skill_merge.go).
type (
	HaloAPIClient            = haloadapter.Client
	HaloClient               = haloadapter.HaloClient
	FilmChunkData            = filmfetch.FilmChunkData
	FilmChunk                = filmfetch.FilmChunk
	FilmChunkRef             = filmfetch.FilmChunkRef
	MatchSkillData           = haloadapter.MatchSkillData
	PlayerPlaylistCSR        = haloadapter.PlayerPlaylistCSR
	CSRRankSnapshot          = haloadapter.CSRRankSnapshot
	LocalFilmCache           = filmfetch.LocalFilmCache
	CachedChunk              = filmfetch.CachedChunk
	CachedManifest           = filmfetch.CachedManifest
	HTTPError                = haloadapter.HTTPError
	MatchHistoryEntry        = haloadapter.MatchHistoryEntry
	PlaylistConfig           = haloadapter.PlaylistConfig
	PlaylistEntry            = haloadapter.PlaylistEntry
	SpartanCustomizationData = haloadapter.SpartanCustomizationData
)

var (
	NewHaloAPIClient            = haloadapter.NewClient
	NewLocalFilmCache           = filmfetch.NewLocalFilmCache
	IsAuthError                 = haloadapter.IsAuthError
	ParseMatchSkillResponseJSON = haloadapter.ParseMatchSkillResponseJSON
	ResolveNameplateURL         = haloadapter.ResolveNameplateURL
	ErrSkillStatsUnavailable    = haloadapter.ErrSkillStatsUnavailable
)
