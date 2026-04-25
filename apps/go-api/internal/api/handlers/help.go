// Package handlers — handler GET /api/v1/help/release-notes.
package handlers

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// helpDiskCache est la structure sérialisée sur disque.
type helpDiskCache struct {
	Content  string    `json:"content"`
	CachedAt time.Time `json:"cached_at"`
}

// HelpHandler sert les notes de version extraites du README (EN ou FR),
// en reconstituant l'historique complet via git log. Cache double couche :
// mémoire (TTL 24h) + disque (data/cache/help_release_notes_{lang}.json).
type HelpHandler struct {
	repoRoot string
	cacheDir string // répertoire pour le cache disque
	mu       sync.RWMutex
	memory   map[string]helpCacheEntry
	ttl      time.Duration
}

type helpCacheEntry struct {
	content  string
	loadedAt time.Time
}

// NewHelpHandler crée le handler pour les notes de version.
// repoRoot est la racine du projet (contenant README.md et docs/).
func NewHelpHandler(repoRoot string) *HelpHandler {
	return &HelpHandler{
		repoRoot: repoRoot,
		cacheDir: filepath.Join(repoRoot, "data", "cache"),
		memory:   make(map[string]helpCacheEntry),
		ttl:      24 * time.Hour,
	}
}

// GetReleaseNotes retourne le markdown des notes de version.
// Query param : lang=fr|en (défaut : fr).
func (h *HelpHandler) GetReleaseNotes(w http.ResponseWriter, r *http.Request) {
	lang := r.URL.Query().Get("lang")
	if lang != "en" {
		lang = "fr"
	}

	content, err := h.loadCached(lang)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "RELEASE_NOTES_ERROR", "Impossible de charger les notes de version")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"content": content})
}

func (h *HelpHandler) loadCached(lang string) (string, error) {
	// 1. Cache mémoire (lecture rapide).
	h.mu.RLock()
	if e, ok := h.memory[lang]; ok && time.Since(e.loadedAt) < h.ttl {
		h.mu.RUnlock()
		return e.content, nil
	}
	h.mu.RUnlock()

	h.mu.Lock()
	defer h.mu.Unlock()

	// Double-check après acquisition du write lock.
	if e, ok := h.memory[lang]; ok && time.Since(e.loadedAt) < h.ttl {
		return e.content, nil
	}

	// 2. Cache disque — évite de ré-invoquer git à chaque redémarrage.
	if content, ok := h.loadDiskCache(lang); ok {
		h.memory[lang] = helpCacheEntry{content: content, loadedAt: time.Now()}
		return content, nil
	}

	// 3. Reconstruction depuis git + écriture en cache.
	content, err := buildFullReleaseHistory(h.repoRoot, lang)
	if err != nil {
		return "", err
	}
	h.memory[lang] = helpCacheEntry{content: content, loadedAt: time.Now()}
	h.writeDiskCache(lang, content)
	return content, nil
}

// diskCachePath retourne le chemin du fichier de cache disque pour lang.
func (h *HelpHandler) diskCachePath(lang string) string {
	return filepath.Join(h.cacheDir, fmt.Sprintf("help_release_notes_%s.json", lang))
}

// loadDiskCache tente de charger le cache disque. Retourne ("", false) si absent ou expiré.
func (h *HelpHandler) loadDiskCache(lang string) (string, bool) {
	path := h.diskCachePath(lang)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	var dc helpDiskCache
	if err := json.Unmarshal(data, &dc); err != nil {
		return "", false
	}
	if time.Since(dc.CachedAt) >= h.ttl {
		return "", false
	}
	return dc.Content, true
}

// writeDiskCache persiste le contenu dans le fichier de cache disque (écriture atomique).
func (h *HelpHandler) writeDiskCache(lang, content string) {
	if err := os.MkdirAll(h.cacheDir, 0o755); err != nil {
		return
	}
	dc := helpDiskCache{Content: content, CachedAt: time.Now()}
	raw, err := json.Marshal(dc)
	if err != nil {
		return
	}
	path := h.diskCachePath(lang)
	tmp, err := os.CreateTemp(h.cacheDir, ".help_cache_tmp_")
	if err != nil {
		return
	}
	tmpName := tmp.Name()
	_, writeErr := tmp.Write(raw)
	closeErr := tmp.Close()
	if writeErr != nil || closeErr != nil {
		_ = os.Remove(tmpName)
		return
	}
	_ = os.Rename(tmpName, path)
}

// ─── Extraction git ───────────────────────────────────────────────────────────

// readmeRelPath retourne le chemin relatif (slash) du README selon la langue.
func readmeRelPath(lang string) string {
	if lang == "fr" {
		return "docs/FR/README.md"
	}
	return "README.md"
}

// buildFullReleaseHistory reconstruit l'historique complet des What's new.
//
// Stratégie :
//  1. Lire la version courante sur disque (working tree) — capte les modifs
//     non encore committées et a la priorité sur les snapshots git.
//  2. Compléter avec les snapshots git des commits passés pour les versions
//     non présentes dans la version courante.
//
// Cette priorité garantit qu'une refonte du README (nouveau format, correctif
// d'un bloc existant) prend effet immédiatement, sans attendre le commit.
func buildFullReleaseHistory(repoRoot, lang string) (string, error) {
	relPath := readmeRelPath(lang)
	blocks := map[string]string{}

	// 1. Version disque (priorité maximale).
	if data, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(relPath))); err == nil {
		for ver, block := range extractWhatsNewBlocks(string(data)) {
			blocks[ver] = block
		}
	}

	// 2. Snapshots git pour enrichir les versions absentes.
	if shas, err := gitLogSHAs(repoRoot, relPath); err == nil {
		for _, sha := range shas {
			raw, err := gitShowFile(repoRoot, sha, relPath)
			if err != nil {
				continue
			}
			for ver, block := range extractWhatsNewBlocks(raw) {
				if _, exists := blocks[ver]; !exists {
					blocks[ver] = block
				}
			}
		}
	}

	if len(blocks) == 0 {
		return buildFromDisk(repoRoot, lang)
	}

	return assembleBlocks(blocks, lang), nil
}

// buildFromDisk est le fallback : lit le README depuis le disque.
func buildFromDisk(repoRoot, lang string) (string, error) {
	relPath := readmeRelPath(lang)
	absPath := filepath.Join(repoRoot, filepath.FromSlash(relPath))
	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("readFile %s: %w", absPath, err)
	}
	blocks := extractWhatsNewBlocks(string(data))
	if len(blocks) == 0 {
		return string(data), nil
	}
	return assembleBlocks(blocks, lang), nil
}

// assembleBlocks trie les blocs de version et les assemble en markdown.
func assembleBlocks(blocks map[string]string, lang string) string {
	keys := make([]string, 0, len(blocks))
	for k := range blocks {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return compareVersionDesc(keys[i], keys[j])
	})

	var sb strings.Builder
	if lang == "fr" {
		sb.WriteString("## Dernières nouveautés\n\n")
	} else {
		sb.WriteString("## What's new\n\n")
	}
	for i, k := range keys {
		block := blocks[k]
		sb.WriteString(block)
		if i < len(keys)-1 {
			next := blocks[keys[i+1]]
			// Cas legacy : si deux items sont des puces (`- **vX.Y...`), un seul
			// \n suffit pour rester dans la même liste markdown. Sinon \n\n entre
			// blocs (format heading + sous-liste).
			if strings.HasPrefix(block, "- ") && strings.HasPrefix(next, "- ") {
				sb.WriteString("\n")
			} else {
				sb.WriteString("\n\n")
			}
		}
	}
	return strings.TrimSpace(sb.String())
}

// gitLogSHAs retourne les SHA des commits ayant modifié le fichier relPath.
func gitLogSHAs(repoRoot, relPath string) ([]string, error) {
	cmd := exec.Command("git", "log", "--all", "--format=%H", "--", relPath)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log: %w", err)
	}
	var shas []string
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		if s := strings.TrimSpace(sc.Text()); s != "" {
			shas = append(shas, s)
		}
	}
	return shas, nil
}

// gitShowFile retourne le contenu du fichier au commit sha.
func gitShowFile(repoRoot, sha, relPath string) (string, error) {
	cmd := exec.Command("git", "show", sha+":"+relPath)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git show %s:%s: %w", sha, relPath, err)
	}
	return string(out), nil
}

// ─── Parsing README ───────────────────────────────────────────────────────────

// extractWhatsNewBlocks analyse le contenu d'un README et retourne une map
// version courte ("v7.0") → bloc markdown complet du What's new.
func extractWhatsNewBlocks(content string) map[string]string {
	blocks := map[string]string{}
	sc := bufio.NewScanner(strings.NewReader(content))

	inSection := false
	var currentVer string
	var currentLines []string

	flushCurrent := func() {
		if currentVer != "" && len(currentLines) > 0 {
			if _, exists := blocks[currentVer]; !exists {
				blocks[currentVer] = strings.TrimSpace(strings.Join(currentLines, "\n"))
			}
		}
		currentVer = ""
		currentLines = nil
	}

	for sc.Scan() {
		line := sc.Text()
		stripped := strings.TrimSpace(line)

		if !inSection {
			if stripped == "## What's new" || stripped == "## Dernières nouveautés" {
				inSection = true
			}
			continue
		}

		if strings.HasPrefix(stripped, "## ") {
			flushCurrent()
			break
		}

		if stripped == "" {
			if currentVer != "" {
				currentLines = append(currentLines, "")
			}
			continue
		}

		if isVersionLine(stripped) {
			flushCurrent()
			currentVer = extractVersionKey(stripped)
			currentLines = []string{stripped}
			continue
		}

		if currentVer != "" {
			// Préserver la ligne d'origine (avec son indentation) pour conserver
			// les sous-listes markdown (`  - item`).
			currentLines = append(currentLines, line)
		}
	}
	flushCurrent()
	return blocks
}

// isVersionLine retourne true si la ligne annonce un bloc de version.
func isVersionLine(line string) bool {
	s := line
	if strings.HasPrefix(s, "- ") {
		s = strings.TrimPrefix(s, "- ")
	}
	return strings.HasPrefix(s, "**v") && len(s) > 5
}

// extractVersionKey retourne la clé normalisée "vX.Y" d'une ligne de version.
func extractVersionKey(line string) string {
	s := line
	if strings.HasPrefix(s, "- ") {
		s = strings.TrimPrefix(s, "- ")
	}
	s = strings.TrimPrefix(s, "**")
	s = strings.TrimPrefix(s, "v")
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '.' || r == ' ' || r == '—' || r == '-'
	})
	if len(parts) < 2 {
		if len(parts) == 1 {
			return "v" + parts[0]
		}
		return s
	}
	return fmt.Sprintf("v%s.%s", parts[0], parts[1])
}

// compareVersionDesc retourne true si a doit apparaître avant b (tri décroissant).
func compareVersionDesc(a, b string) bool {
	aMaj, aMin := parseVerParts(a)
	bMaj, bMin := parseVerParts(b)
	if aMaj != bMaj {
		return aMaj > bMaj
	}
	return aMin > bMin
}

func parseVerParts(v string) (int, int) {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	parts := strings.SplitN(v, ".", 3)
	if len(parts) < 2 {
		return 0, 0
	}
	maj, _ := strconv.Atoi(parts[0])
	min, _ := strconv.Atoi(parts[1])
	return maj, min
}
