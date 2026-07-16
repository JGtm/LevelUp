// Package ops — resources.go : helpers PURS du monitoring ressources (A5,
// DC-4) : runtime Go, tailles de fichiers DB + WAL, statut disque (seuils
// nommés A5.3 — VPS 2 Go RAM / disque serré, pièges BuildKit/restic connus).
package ops

import (
	"os"
	"runtime"

	"levelup/go-api/internal/domain"
)

// Seuils disque A5.3 (nommés, jamais de littéral chez les callers).
// Deux familles COMBINÉES (pire des deux gagne) :
//   - plancher ABSOLU d'espace libre (fin de partie sur tout disque) ;
//   - POURCENTAGE d'occupation (alerte précoce sur un disque qui se remplit —
//     ajouté après l'incident disque-plein VPS du 2026-07-13 : à 82 % le
//     2026-07-07, seuls les seuils absolus étaient loin, aucune alerte).
const (
	// DiskFreeWarnBytes : en-dessous, statut warn (2 Go).
	DiskFreeWarnBytes = 2 << 30
	// DiskFreeCriticalBytes : en-dessous, statut critical (500 Mo).
	DiskFreeCriticalBytes = 500 << 20
	// DiskUsedWarnPercent : au-delà de 80 % d'occupation, statut warn.
	DiskUsedWarnPercent = 80.0
	// DiskUsedCriticalPercent : au-delà de 90 % d'occupation, statut critical.
	DiskUsedCriticalPercent = 90.0
)

// EvaluateDiskStatus mappe (libre, total) vers ok/warn/critical (A5.3).
// totalBytes == 0 (info indisponible) → seuls les seuils absolus s'appliquent.
func EvaluateDiskStatus(freeBytes, totalBytes uint64) string {
	usedPct := DiskUsedPercent(freeBytes, totalBytes)
	switch {
	case freeBytes < DiskFreeCriticalBytes || usedPct >= DiskUsedCriticalPercent:
		return domain.FreshnessStatusCritical
	case freeBytes < DiskFreeWarnBytes || usedPct >= DiskUsedWarnPercent:
		return domain.FreshnessStatusWarn
	default:
		return domain.FreshnessStatusOK
	}
}

// DiskUsedPercent calcule le pourcentage d'occupation (0 si total inconnu).
func DiskUsedPercent(freeBytes, totalBytes uint64) float64 {
	if totalBytes == 0 || freeBytes > totalBytes {
		return 0
	}
	return float64(totalBytes-freeBytes) / float64(totalBytes) * 100
}

// CollectRuntimeStats lit l'état runtime Go du process (DC-4).
func CollectRuntimeStats() domain.ResourceRuntime {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return domain.ResourceRuntime{
		Goroutines:     runtime.NumGoroutine(),
		HeapAllocBytes: m.HeapAlloc,
		HeapSysBytes:   m.HeapSys,
		SysBytes:       m.Sys,
		NumGC:          m.NumGC,
	}
}

// DBFileSize retourne la taille d'une base + de son WAL adjacent (0 si absents).
// Best-effort : un fichier manquant n'est pas une erreur (base pas encore créée).
func DBFileSize(name, path string) domain.ResourceDBFile {
	out := domain.ResourceDBFile{Name: name, Path: path}
	if info, err := os.Stat(path); err == nil {
		out.SizeBytes = info.Size()
	}
	if info, err := os.Stat(path + ".wal"); err == nil {
		out.WalBytes = info.Size()
	}
	return out
}

// DirTotalSize agrège récursivement la taille d'un répertoire (players/ d'un
// titre). Best-effort : entrées illisibles ignorées.
func DirTotalSize(root string) int64 {
	var total int64
	entries, err := os.ReadDir(root)
	if err != nil {
		return 0
	}
	for _, e := range entries {
		full := root + string(os.PathSeparator) + e.Name()
		if e.IsDir() {
			total += DirTotalSize(full)
			continue
		}
		if info, err := e.Info(); err == nil {
			total += info.Size()
		}
	}
	return total
}
