package ops

import (
	"path/filepath"
	"strings"
)

// MediaPathStore encapsule la résolution entre paths absolus disk et paths
// relatifs stables stockés en DB.
//
// Format de stockage : {owner_slug}/{rel_in_owner_dir}
// Exemples :
//   - "Madina97294/Halo Infinite 2026-04-19 17-43-35.mp4"
//   - "JGtm/thumbs/Halo Infinite 2025-12-21 17-02-11_dd7e39ab8543.webp"
//
// Avantages vs un chemin absolu :
//   - DB portable entre machines (laptop, VPS, backup-restore).
//   - Changer MediaCapturesBaseDir = aucune migration nécessaire.
//   - Pas de fallback à 3 tentatives côté serveur — résolution déterministe.
//
// Mode legacy : si CapturesBase est vide OU si un path stocké est déjà absolu
// (ex: backfill jamais effectué), les méthodes ToAbs/ToRel renvoient l'input
// tel quel — le code appelant garde la sémantique pré-refactor.
type MediaPathStore struct {
	// CapturesBase est la racine multi-player où vit {slug}/{file}.
	// Typiquement la valeur de settings.MediaCapturesBaseDir.
	// Si vide, le store fonctionne en mode pass-through (no-op).
	CapturesBase string
}

// ToRel convertit un absPath en path relatif stable {owner_slug}/{rel}.
// Retourne "" si la conversion n'est pas possible (CapturesBase vide,
// absPath en dehors, ownerSlug vide). Le caller doit alors conserver absPath
// tel quel.
//
// Deux layouts gérés :
//  1. Multi-player : {capturesBase}/{ownerSlug}/{rel} (cas standard).
//  2. Single-player : {capturesBase}/{rel} — on préfixe rel par ownerSlug pour
//     garder un format uniforme en DB.
func (s MediaPathStore) ToRel(absPath, ownerSlug string) string {
	if s.CapturesBase == "" || ownerSlug == "" || absPath == "" {
		return ""
	}
	cleanBase := filepath.Clean(s.CapturesBase)
	cleanAbs := filepath.Clean(absPath)

	ownerBase := filepath.Join(cleanBase, ownerSlug)
	if rel, ok := relUnder(ownerBase, cleanAbs); ok {
		return filepath.ToSlash(filepath.Join(ownerSlug, rel))
	}
	if rel, ok := relUnder(cleanBase, cleanAbs); ok {
		return filepath.ToSlash(filepath.Join(ownerSlug, rel))
	}
	return ""
}

// ToAbs résout un path stocké en path absolu prêt à passer à os.Stat /
// http.ServeFile. Si stored est déjà absolu (legacy) ou si CapturesBase est
// vide, retourne stored tel quel.
func (s MediaPathStore) ToAbs(stored string) string {
	if stored == "" {
		return stored
	}
	if filepath.IsAbs(stored) {
		return stored
	}
	if s.CapturesBase == "" {
		return stored
	}
	return filepath.Join(s.CapturesBase, filepath.FromSlash(stored))
}

// IsRel retourne true si stored est un path relatif stable (format
// {slug}/{rel}). Sert à distinguer entre paths déjà migrés et paths legacy
// absolus encore présents en DB.
func (s MediaPathStore) IsRel(stored string) bool {
	if stored == "" {
		return false
	}
	return !filepath.IsAbs(stored)
}

// relUnder retourne (rel, true) si target est strictement sous base. rel est
// retourné avec les séparateurs natifs du FS (non normalisé en forward slash).
func relUnder(base, target string) (string, bool) {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return "", false
	}
	if rel == "." || rel == "" || strings.HasPrefix(rel, "..") {
		return "", false
	}
	return rel, true
}
