// Package persist — shared_social_rows.go : structs Row pour les inserts/
// updates dans shared_social.duckdb. Pure data types, pas de logique.

package persist

import "time"

// MediaFileInsert : 1 ligne à insérer dans media_files (INSERT OR IGNORE).
// Cf. ops/media.go:insertMediaFile pour les contraintes (file_path UNIQUE).
type MediaFileInsert struct {
	PlayerSlug      string     // owner du média (player_slug)
	FilePath        string     // path stocké (relatif si CapturesBase défini, sinon absolu)
	FileName        string     // basename
	FileStem        string     // basename sans extension (clé de dédup format-agnostique)
	FileExt         string     // extension avec le point (ex: ".mp4")
	FileHash        string     // sha256[:16] hex
	Kind            string     // "video" | "image"
	CaptureStartUTC *time.Time // priorité regex filename > client lastModified > mtime serveur
	CaptureEndUTC   *time.Time // capture_start + duration_seconds (vidéos), = start (images)
	DurationSeconds *float64   // ffprobe (vidéos) ; 0 (images) ; nil si inconnu
}

// MediaThumbnailUpdate : mise à jour du thumbnail_path pour un média existant.
// PK ciblée par id (INTEGER) → safe contre le bug ART (pas de pression PK VARCHAR).
type MediaThumbnailUpdate struct {
	MediaFileID   int64  // id du média (PK INTEGER)
	ThumbnailPath string // path miniature (.webp ou .gif legacy)
}

// MediaAssociationInsert : 1 association média ↔ match (INSERT OR IGNORE).
// PK composite (media_file_id, match_id).
type MediaAssociationInsert struct {
	MediaFileID  int64  // id du média
	MatchID      string // UUID du match
	DeltaSeconds int    // distance secondes entre capture_start et match start (positif)
}

// LikeInsert : 1 like sur un média (INSERT OR IGNORE).
type LikeInsert struct {
	MediaPath     string    // path stocké du média (clé fonctionnelle stable)
	LikerSlug     string    // gamertag du liker
	LikerGamertag string    // gamertag displayed (peut différer du slug si renommage)
	LikedAt       time.Time // horodatage like (UTC)
}

// LikeRemove : DELETE d'un like (toggle off).
type LikeRemove struct {
	MediaPath string
	LikerSlug string
}

// FavoriteInsert : 1 match marqué favori par un joueur (INSERT OR IGNORE).
type FavoriteInsert struct {
	PlayerSlug  string
	MatchID     string
	FavoritedAt time.Time
}

// FavoriteRemove : DELETE d'un favori (toggle off).
type FavoriteRemove struct {
	PlayerSlug string
	MatchID    string
}

// NotificationInsert : 1 notification in-app (INSERT).
//
// Cf. steps_player_notifications.go pour le schéma. PK composite (xuid, id).
// id est typiquement un BIGINT séquentiel par xuid (générer côté caller).
type NotificationInsert struct {
	XUID         string
	ID           int64
	Category     string  // ex: "milestone", "mention", "auto_sync_complete"
	Severity     string  // "info" | "warn" | "success" | "error"
	TitleKey     string  // i18n key (résolu côté frontend)
	BodyKey      *string // i18n key optionnelle pour le body
	Params       *string // JSON sérialisé des params pour i18n
	TargetRoute  *string // ex: "/t/{titleSlug}/players/{slug}/matches/{id}" (peut aussi porter du stock legacy "/players/…")
	TargetSearch *string // querystring pour le route target
	ActorXUID    *string // xuid de l'auteur (mentions sociales)
	ActorName    *string // gamertag de l'auteur
	Source       string  // origine ("sync_engine", "social_action", etc.)
	CreatedAt    time.Time
}

// NotificationReadUpdate : marque une notification comme lue (UPDATE read_at).
type NotificationReadUpdate struct {
	XUID   string
	ID     int64
	ReadAt time.Time
}

// PlayerRecordAppend : append-only sur player_records_history (Phase 2).
//
// Pattern : au lieu d'UPSERT (UPDATE qui pressionne l'index ART DuckDB), on
// fait INSERT pur. La vue player_records_latest expose la dernière valeur
// par (xuid, metric, period). Cf. CLAUDE.md "Phase 2 ART" pour d'autres
// tables ayant adopté ce pattern.
//
// WrittenAt par défaut = time.Now() si zero — utilisé par la vue _latest
// pour déduplication par DISTINCT ON.
type PlayerRecordAppend struct {
	XUID            string
	Metric          string  // ex: "kda_best", "perfect_kills_total"
	Period          string  // "30d" | "90d" | "all_time" (défaut "all_time")
	Value           float64 // valeur du record
	AchievedAt      *time.Time
	AchievedMatchID *string // UUID du match où le record a été atteint
	// PreviousValue / PreviousAchievedAt : PB précédent (exposé par l'API
	// /records). Dénormalisés dans chaque row d'historique pour préserver le
	// comportement de l'ancien UPSERT. Nil si premier PB.
	PreviousValue      *float64
	PreviousAchievedAt *time.Time
	WrittenAt          time.Time // horodatage de l'écriture (utilisé pour vue _latest)
}
