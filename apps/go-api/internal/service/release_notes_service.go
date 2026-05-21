// Package service — release_notes_service.go : extraction des notes de version.
//
// P8.10 (revue 2026-04-29 gap #4) : la logique git + parsing markdown vivait
// dans handlers/help.go (390L) — extraite ici pour respecter la séparation
// handler ↔ service. Le handler devient un wrapper HTTP mince.
//
// L'accès git passe par port.GitProvider (mockable). La fonction `Build`
// retourne le markdown complet de l'historique (working tree + git log
// snapshots des README modifiés).
package service

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/port"
)

// ReleaseNotesService construit l'historique des release notes (FR/EN).
type ReleaseNotesService struct {
	repoRoot string
	git      port.GitProvider
}

// NewReleaseNotesService crée un service avec le GitProvider injecté.
func NewReleaseNotesService(repoRoot string, git port.GitProvider) *ReleaseNotesService {
	return &ReleaseNotesService{repoRoot: repoRoot, git: git}
}

// Build retourne le markdown complet des release notes pour `lang` (fr|en).
//
// Stratégie :
//  1. Lire le README sur disque (working tree) — capte les modifs non
//     committées et a la priorité sur les snapshots git.
//  2. Compléter avec les snapshots git pour les versions absentes du WT.
func (s *ReleaseNotesService) Build(lang string) (string, error) {
	relPath := readmeRelPath(lang)
	blocks := map[string]string{}

	// 1. Version disque (priorité maximale).
	if data, err := os.ReadFile(filepath.Join(s.repoRoot, filepath.FromSlash(relPath))); err == nil {
		for ver, block := range extractWhatsNewBlocks(string(data)) {
			blocks[ver] = block
		}
	}

	// 2. Snapshots git pour enrichir les versions absentes.
	if shas, err := s.git.LogSHAs(s.repoRoot, relPath); err == nil {
		for _, sha := range shas {
			raw, err := s.git.ShowFile(s.repoRoot, sha, relPath)
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
		return s.buildFromDisk(lang)
	}
	return assembleReleaseBlocks(blocks, lang), nil
}

// buildFromDisk est le fallback : lit le README depuis le disque sans git.
func (s *ReleaseNotesService) buildFromDisk(lang string) (string, error) {
	relPath := readmeRelPath(lang)
	absPath := filepath.Join(s.repoRoot, filepath.FromSlash(relPath))
	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("readFile %s: %w", absPath, err)
	}
	blocks := extractWhatsNewBlocks(string(data))
	if len(blocks) == 0 {
		return string(data), nil
	}
	return assembleReleaseBlocks(blocks, lang), nil
}

func readmeRelPath(lang string) string {
	if lang == "fr" {
		return "docs/FR/README.md"
	}
	return "README.md"
}

// assembleReleaseBlocks trie les blocs de version (descendant) et les
// concatène en markdown.
func assembleReleaseBlocks(blocks map[string]string, lang string) string {
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
			if strings.HasPrefix(block, "- ") && strings.HasPrefix(next, "- ") {
				sb.WriteString("\n")
			} else {
				sb.WriteString("\n\n")
			}
		}
	}
	return strings.TrimSpace(sb.String())
}

// extractWhatsNewBlocks analyse un README et retourne version → bloc markdown.
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
			currentLines = append(currentLines, line)
		}
	}
	flushCurrent()
	return blocks
}

func isVersionLine(line string) bool {
	s := strings.TrimPrefix(line, "- ")
	return strings.HasPrefix(s, "**v") && len(s) > 5
}

func extractVersionKey(line string) string {
	s := strings.TrimPrefix(line, "- ")
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
