package settings

// data.go — les règles des réglages des données de l'application de bureau (lot L22) : sauvegarde
// périodique en Parquet et plafond mémoire des bases.
//
// Bornes : l'intervalle de sauvegarde est un nombre ENTIER d'heures de 1 à 168 (une semaine), le nombre de
// jeux conservés va de 1 à 60, le plafond mémoire est 0 (valeur par défaut du serveur) ou de 1024 à 32768 Mo.
// Le PATCH refuse une valeur hors bornes ([CheckData], sur les seules clés écrites) ; la LECTURE d'un fichier
// édité à la main replie le nombre de jeux et le plafond sur leur borne ([normalizeData], journalisé une
// fois). `backup_interval` vide ou illisible vaut 24 h à l'usage ([BackupIntervalDuration]) sans être réécrit :
// la clé est aussi celle du planificateur restic d'une instance hébergée (durée Go, par exemple « 6h »), que ce
// paquet ne doit jamais modifier en silence.

import (
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"sync/atomic"
	"time"

	"levelup/go-api/internal/domain"
)

// Bornes et défauts des réglages de données.
const (
	DefaultBackupInterval = "24h" // valeur d'usage d'un `backup_interval` vide ; jamais écrite d'office
	BackupIntervalMinH    = 1
	BackupIntervalMaxH    = 7 * 24
	DefaultBackupKeep     = 7
	BackupKeepMax         = 60
	MemoryLimitMinMB      = 1024
	MemoryLimitMaxMB      = 32768
)

// Erreurs de CheckData : leur identité fixe le code d'API que le handler rend.
var (
	ErrBackupInterval = errors.New("settings: backup_interval doit être un nombre entier d'heures de 1h à 168h")
	ErrBackupKeep     = errors.New("settings: backup_keep doit être compris entre 1 et 60")
	ErrMemoryLimit    = errors.New("settings: memory_limit_mb doit valoir 0 ou être compris entre 1024 et 32768")
)

var hoursPattern = regexp.MustCompile(`^(\d{1,4})h$`)

// BackupIntervalHours lit un intervalle « <n>h » ; ok=false hors de la forme ou des bornes.
func BackupIntervalHours(s string) (int, bool) {
	m := hoursPattern.FindStringSubmatch(s)
	if m == nil {
		return 0, false
	}
	h, _ := strconv.Atoi(m[1])
	return h, h >= BackupIntervalMinH && h <= BackupIntervalMaxH
}

// BackupIntervalDuration rend l'intervalle du réglage ; vide ou hors forme, le défaut de 24 h.
func BackupIntervalDuration(s string) time.Duration {
	if h, ok := BackupIntervalHours(s); ok {
		return time.Duration(h) * time.Hour
	}
	return 24 * time.Hour
}

// applyData applique les quatre clés des données d'une écriture partielle.
func applyData(cfg *AppSettings, req *domain.DataUpdate) {
	if req.BackupDir != nil {
		cfg.BackupDir = *req.BackupDir
	}
	if req.BackupInterval != nil {
		cfg.BackupInterval = *req.BackupInterval
	}
	if req.BackupKeep != nil {
		cfg.BackupKeep = *req.BackupKeep
	}
	if req.MemoryLimitMB != nil {
		cfg.MemoryLimitMB = *req.MemoryLimitMB
	}
}

// CheckData refuse une valeur hors bornes parmi les clés ÉCRITES de req (les autres ne sont pas jugées : un
// fichier édité à la main ne bloque pas l'écriture d'une autre clé).
func CheckData(cfg *AppSettings, req *domain.DataUpdate) error {
	if _, ok := BackupIntervalHours(cfg.BackupInterval); req.BackupInterval != nil && !ok {
		return fmt.Errorf("%w (reçu %q)", ErrBackupInterval, cfg.BackupInterval)
	}
	if req.BackupKeep != nil && (cfg.BackupKeep < 1 || cfg.BackupKeep > BackupKeepMax) {
		return fmt.Errorf("%w (reçu %d)", ErrBackupKeep, cfg.BackupKeep)
	}
	if req.MemoryLimitMB != nil && cfg.MemoryLimitMB != 0 && (cfg.MemoryLimitMB < MemoryLimitMinMB || cfg.MemoryLimitMB > MemoryLimitMaxMB) {
		return fmt.Errorf("%w (reçu %d)", ErrMemoryLimit, cfg.MemoryLimitMB)
	}
	return nil
}

// dataFoldWarned : un repli a déjà été journalisé dans ce processus (Load est appelée à chaque requête).
var dataFoldWarned atomic.Bool

// normalizeData replie une valeur hors bornes d'un fichier édité à la main : le nombre de jeux sur le défaut (0
// ou négatif) ou le plafond, le plafond mémoire sur la borne la plus proche (0 reste 0). L'intervalle n'est
// jamais replié ici (voir l'en-tête du fichier).
func normalizeData(cfg *AppSettings) {
	folded := false
	switch {
	case cfg.BackupKeep < 1:
		cfg.BackupKeep, folded = DefaultBackupKeep, true
	case cfg.BackupKeep > BackupKeepMax:
		cfg.BackupKeep, folded = BackupKeepMax, true
	}
	switch {
	case cfg.MemoryLimitMB < 0:
		cfg.MemoryLimitMB, folded = 0, true
	case cfg.MemoryLimitMB > 0 && cfg.MemoryLimitMB < MemoryLimitMinMB:
		cfg.MemoryLimitMB, folded = MemoryLimitMinMB, true
	case cfg.MemoryLimitMB > MemoryLimitMaxMB:
		cfg.MemoryLimitMB, folded = MemoryLimitMaxMB, true
	}
	if folded && dataFoldWarned.CompareAndSwap(false, true) {
		slog.Warn("settings: réglage de données hors bornes dans app_settings.json, valeur repliée",
			"backup_keep", cfg.BackupKeep, "memory_limit_mb", cfg.MemoryLimitMB)
	}
}
