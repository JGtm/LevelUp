package domain

import "time"

// settings_data.go — les réglages des DONNÉES de l'application de bureau (lot L22, ADR 0045) : la
// sauvegarde périodique en Parquet vers un dossier choisi et le plafond mémoire des bases. Les trois
// types de lecture/écriture des réglages les EMBARQUENT (clés aplaties dans le JSON). Chaque clé n'a
// d'effet que sous la capacité de déploiement correspondante (ParquetBackup, MemoryLimit) ; hors
// desktop le serveur refuse de la modifier.

// DataSettings : ce que app_settings.json et GET /settings portent des données de l'application de bureau.
type DataSettings struct {
	// BackupDir : dossier choisi pour la sauvegarde périodique, chemin absolu. Vide = sauvegarde désactivée.
	BackupDir string `json:"backup_dir" doc:"Dossier de la sauvegarde périodique des données en Parquet (chemin absolu). Vide = désactivée. Application de bureau seulement."`
	// BackupInterval : délai entre deux sauvegardes, en heures entières ("24h"), de 1 h à 7 jours ; vide = 24 h.
	BackupInterval string `json:"backup_interval" doc:"Délai entre deux sauvegardes, en heures entières (1h à 168h). Vide = 24h. Application de bureau seulement."`
	// BackupKeep : nombre de jeux de sauvegarde conservés (1 à 60). Défaut 7.
	BackupKeep int `json:"backup_keep" doc:"Nombre de jeux de sauvegarde conservés dans le dossier choisi (1 à 60). Défaut 7. Application de bureau seulement."`
	// MemoryLimitMB : plafond mémoire de chaque base DuckDB, en Mo (1024 à 32768). 0 = valeur par défaut du serveur.
	MemoryLimitMB int `json:"memory_limit_mb" doc:"Plafond mémoire de chaque base DuckDB, en Mo (1024 à 32768). 0 = valeur par défaut du serveur. Application de bureau seulement."`
}

// DataUpdate : les quatre clés, en écriture partielle (PATCH /settings).
type DataUpdate struct {
	BackupDir      *string `json:"backup_dir,omitempty"`
	BackupInterval *string `json:"backup_interval,omitempty"`
	BackupKeep     *int    `json:"backup_keep,omitempty"`
	MemoryLimitMB  *int    `json:"memory_limit_mb,omitempty"`
}

// DataStatus est la réponse de GET /settings/data : l'état vivant des données de l'application de bureau.
type DataStatus struct {
	// Backup : la sauvegarde en dossier ; absent quand le profil ne l'offre pas.
	Backup *DataBackupStatus `json:"backup,omitempty" doc:"État de la sauvegarde périodique en Parquet (absent hors de l'application de bureau)."`
	// Memory : la mémoire du processus ; absent quand le profil n'offre pas de plafond.
	Memory *MemoryStatus `json:"memory,omitempty" doc:"Mémoire du processus et plafond effectif des bases (absent hors de l'application de bureau)."`
}

// DataBackupStatus : l'état de la sauvegarde périodique.
type DataBackupStatus struct {
	// Enabled : un dossier est choisi.
	Enabled bool `json:"enabled" doc:"Un dossier de sauvegarde est choisi."`
	// Running : une sauvegarde est en cours d'écriture.
	Running bool `json:"running" doc:"Une sauvegarde est en cours d'écriture."`
	// SameVolume : le dossier choisi est sur le même disque que les données (une panne du disque emporterait les deux).
	SameVolume bool `json:"same_volume" doc:"Le dossier choisi est sur le même volume que les données."`
	// Last : le jeu complet le plus récent du dossier.
	Last *DataBackupSummary `json:"last,omitempty" doc:"Jeu de sauvegarde complet le plus récent du dossier choisi."`
	// LastAttempt : le dernier passage depuis le démarrage de l'application.
	LastAttempt *DataBackupAttempt `json:"last_attempt,omitempty" doc:"Dernier passage depuis le démarrage de l'application."`
}

// DataBackupSummary : un jeu de sauvegarde.
type DataBackupSummary struct {
	Name       string    `json:"name" doc:"Nom du dossier du jeu."`
	CreatedAt  time.Time `json:"created_at" doc:"Instant de la sauvegarde."`
	Bytes      int64     `json:"bytes" doc:"Taille du jeu, en octets."`
	Databases  int       `json:"databases" doc:"Nombre de bases du jeu."`
	DurationMs int64     `json:"duration_ms" doc:"Durée de la sauvegarde, en millisecondes."`
	Partial    bool      `json:"partial" doc:"Vrai quand une base est restée occupée ou a échoué : le jeu est incomplet."`
	AppVersion string    `json:"app_version" doc:"Version de l'application qui a écrit le jeu."`
}

// DataBackupAttempt : l'issue du dernier passage.
type DataBackupAttempt struct {
	At        time.Time `json:"at" doc:"Instant du passage."`
	OK        bool      `json:"ok" doc:"Le passage a réussi."`
	ErrorCode string    `json:"error_code,omitempty" doc:"Code fixe de l'échec (nothing_exported, backup_failed)."`
}

// MemoryStatus : la mémoire du processus.
type MemoryStatus struct {
	RSSMB int `json:"rss_mb" doc:"Mémoire résidente du processus, en Mo (0 quand la mesure est indisponible)."`
	// LimitMB : le plafond posé par le réglage ; 0 = valeur par défaut du serveur.
	LimitMB int `json:"limit_mb" doc:"Plafond réglé pour chaque base, en Mo (0 = valeur par défaut du serveur)."`
	// EffectiveLimit : ce que DuckDB reçoit réellement (« 512MB », « 4096MB »).
	EffectiveLimit string `json:"effective_limit" doc:"Plafond mémoire effectivement appliqué à chaque base (par exemple 4096MiB)."`
}

// DataBackupRunResponse est la réponse de POST /settings/data/backup/run.
type DataBackupRunResponse struct {
	Started bool `json:"started" doc:"La sauvegarde a été lancée."`
}

// DiagnosticReport est la réponse de POST /desktop/diagnostic : où est l'archive du rapport de diagnostic.
type DiagnosticReport struct {
	Path     string `json:"path" doc:"Chemin complet de l'archive du rapport, à joindre à un signalement."`
	FileName string `json:"file_name" doc:"Nom du fichier de l'archive."`
	Bytes    int64  `json:"bytes" doc:"Taille de l'archive, en octets."`
}

// ExportToken est la réponse de POST /export/desktop/token : un jeton court à usage unique qui ouvre l'export des
// données du joueur connecté, à coller dans l'application de bureau.
type ExportToken struct {
	Token     string    `json:"token" doc:"Jeton d'export, à usage unique (15 minutes). À coller dans l'application de bureau ; ne le partager avec personne."`
	ExpiresAt time.Time `json:"expires_at" doc:"Instant d'échéance du jeton."`
}

// SiteImportRequest est le corps de POST /import/site.
type SiteImportRequest struct {
	URL      string `json:"url" minLength:"8" maxLength:"300" doc:"Adresse de l'instance hébergée (https)."`
	Token    string `json:"token" minLength:"16" maxLength:"200" doc:"Jeton d'export délivré par l'instance dans ses réglages."`
	Gamertag string `json:"gamertag,omitempty" maxLength:"64" doc:"Joueur suivi qui reçoit l'import ; nécessaire seulement quand plusieurs joueurs sont suivis."`
}

// SiteImportStatus est la réponse de GET /import/site : l'état de l'import courant ou du dernier.
type SiteImportStatus struct {
	Phase     string           `json:"phase" enum:"idle,downloading,importing,done,failed" doc:"Étape de l'import."`
	ErrorCode string           `json:"error_code,omitempty" doc:"Code fixe de l'échec quand la phase est failed."`
	Inserted  map[string]int64 `json:"inserted,omitempty" doc:"Lignes insérées par table, quand l'import est terminé."`
	Matches   int64            `json:"matches" doc:"Matchs ajoutés (lignes insérées dans le registre)."`
}
