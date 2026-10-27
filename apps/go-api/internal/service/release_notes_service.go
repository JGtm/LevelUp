// Package service — release_notes_service.go : extraction des notes de version.
//
// Source unique : docs/RELEASE_NOTES.md (EN) et docs/FR/RELEASE_NOTES.md (FR).
// Toutes les versions y sont consignées ; plus de fallback git sur le README.
package service

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/i18n"
)

// ReleaseNotesService lit docs/RELEASE_NOTES.md (source unique des notes user-facing).
type ReleaseNotesService struct {
	repoRoot string
}

// NewReleaseNotesService crée le service.
func NewReleaseNotesService(repoRoot string) *ReleaseNotesService {
	return &ReleaseNotesService{repoRoot: repoRoot}
}

// Build retourne le markdown complet des release notes pour `lang` (une langue de
// config/locales.toml ; inconnue ou vide : la langue par défaut). Une langue sans fichier de
// notes reçoit la version de la langue de repli précédée d'une mention dans SA langue (D-6) ; le
// repli est compté.
func (s *ReleaseNotesService) Build(ctx context.Context, lang string) (string, error) {
	loc := i18n.Parse(lang)
	relPath, fileLoc := s.releaseNotesFile(loc)
	if fileLoc != loc {
		i18n.CountDocumentFallback()
		slog.InfoContext(ctx, "release notes : pas de fichier dans cette langue, version de repli servie",
			"locale", loc, "served", fileLoc, "path", relPath)
	}
	body, err := s.buildFromDisk(relPath, fileLoc)
	if err != nil {
		return "", err
	}
	if fileLoc != loc {
		body = i18n.Msg(loc, "release_notes.english_only", nil) + "\n\n" + body
	}
	return body, nil
}

// buildFromDisk lit le fichier de notes depuis le disque, sans git.
func (s *ReleaseNotesService) buildFromDisk(relPath string, lang i18n.Locale) (string, error) {
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

// releaseNotesFile choisit le fichier de notes de la langue : la langue de repli lit
// docs/RELEASE_NOTES.md, toute autre docs/<CODE>/RELEASE_NOTES.md (docs/FR/…) quand il existe ;
// sinon le fichier de la langue de repli. Ajouter une langue = déposer son fichier, sans code.
func (s *ReleaseNotesService) releaseNotesFile(l i18n.Locale) (relPath string, served i18n.Locale) {
	if l != i18n.Fallback {
		rel := "docs/" + strings.ToUpper(string(l)) + "/RELEASE_NOTES.md"
		if _, err := os.Stat(filepath.Join(s.repoRoot, filepath.FromSlash(rel))); err == nil {
			return rel, l
		}
	}
	return "docs/RELEASE_NOTES.md", i18n.Fallback
}

// assembleReleaseBlocks trie les blocs de version (descendant) et les
// concatène en markdown.
func assembleReleaseBlocks(blocks map[string]string, lang i18n.Locale) string {
	keys := make([]string, 0, len(blocks))
	for k := range blocks {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return compareVersionDesc(keys[i], keys[j])
	})

	var sb strings.Builder
	sb.WriteString("## " + i18n.Msg(lang, "release_notes.heading", nil) + "\n\n")
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
			if isWhatsNewHeading(stripped) {
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

// isWhatsNewHeading reconnaît le titre de la section des nouveautés dans n'importe quelle langue.
func isWhatsNewHeading(line string) bool {
	for _, h := range i18n.MsgAll("release_notes.heading") {
		if line == "## "+h {
			return true
		}
	}
	return false
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
