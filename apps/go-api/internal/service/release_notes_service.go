// Package service — release_notes_service.go : extraction des notes de version.
//
// Source unique : docs/RELEASE_NOTES.md (EN) et docs/FR/RELEASE_NOTES.md (FR).
// Toutes les versions y sont consignées ; plus de fallback git sur le README.
package service

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// ReleaseNotesService lit docs/RELEASE_NOTES.md (source unique des notes user-facing).
type ReleaseNotesService struct {
	repoRoot string
}

// NewReleaseNotesService crée le service.
func NewReleaseNotesService(repoRoot string) *ReleaseNotesService {
	return &ReleaseNotesService{repoRoot: repoRoot}
}

// Build retourne le markdown complet des release notes pour `lang` (fr|en).
func (s *ReleaseNotesService) Build(_ context.Context, lang string) (string, error) {
	return s.buildFromDisk(lang)
}

// buildFromDisk est le fallback : lit le README depuis le disque sans git.
func (s *ReleaseNotesService) buildFromDisk(lang string) (string, error) {
	relPath := releaseNotesRelPath(lang)
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

func releaseNotesRelPath(lang string) string {
	if lang == "fr" {
		return "docs/FR/RELEASE_NOTES.md"
	}
	return "docs/RELEASE_NOTES.md"
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
