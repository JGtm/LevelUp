package sync

import "levelup/go-api/internal/sync/haloclient"

// haloclient_reexport.go — ré-exports du client HTTP Halo Infinite extrait vers le
// sous-package internal/sync/haloclient (K3e). Les alias conservent les noms
// historiques : sync ET les appelants externes (qui utilisent sync.HaloAPIClient,
// sync.MatchSkillData…) restent inchangés — zéro requalification. Le pont
// DTO->ParticipantRow (MergeSkillIntoParticipants) reste dans sync (skill_merge.go).
type (
	HaloAPIClient            = haloclient.HaloAPIClient
	HaloClient               = haloclient.HaloClient
	FilmChunkData            = haloclient.FilmChunkData
	FilmChunk                = haloclient.FilmChunk
	FilmChunkRef             = haloclient.FilmChunkRef
	MatchSkillData           = haloclient.MatchSkillData
	PlayerPlaylistCSR        = haloclient.PlayerPlaylistCSR
	CSRRankSnapshot          = haloclient.CSRRankSnapshot
	LocalFilmCache           = haloclient.LocalFilmCache
	CachedChunk              = haloclient.CachedChunk
	CachedManifest           = haloclient.CachedManifest
	HTTPError                = haloclient.HTTPError
	MatchHistoryEntry        = haloclient.MatchHistoryEntry
	PlaylistConfig           = haloclient.PlaylistConfig
	PlaylistEntry            = haloclient.PlaylistEntry
	SpartanCustomizationData = haloclient.SpartanCustomizationData
)

var (
	NewHaloAPIClient            = haloclient.NewHaloAPIClient
	NewLocalFilmCache           = haloclient.NewLocalFilmCache
	IsAuthError                 = haloclient.IsAuthError
	ParseMatchSkillResponseJSON = haloclient.ParseMatchSkillResponseJSON
	ResolveNameplateURL         = haloclient.ResolveNameplateURL
	ErrSkillStatsUnavailable    = haloclient.ErrSkillStatsUnavailable
)
