package service

import (
	"context"
	"expvar"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/i18n"
)

// fallbackDocuments lit le compteur expvar des documents servis en langue de repli.
func fallbackDocuments() int64 {
	return expvar.Get("levelup.i18n.document_fallback_total").(*expvar.Int).Value()
}

// writeNotes dépose un fichier de notes sous la racine de test.
func writeNotes(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("MkdirAll : %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile : %v", err)
	}
}

const (
	notesEN = "## What's new\n\n- **v8.1 — Test**\n  - English line\n"
	notesFR = "## Dernières nouveautés\n\n- **v8.1 — Test**\n  - Ligne française\n"
)

// Chaque langue qui a son fichier le lit : la langue de repli dans docs/, les autres dans docs/<CODE>/.
func TestReleaseNotes_UnFichierParLangue(t *testing.T) {
	root := t.TempDir()
	writeNotes(t, root, "docs/RELEASE_NOTES.md", notesEN)
	writeNotes(t, root, "docs/FR/RELEASE_NOTES.md", notesFR)
	svc := NewReleaseNotesService(root)

	en, err := svc.Build(context.Background(), "en")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(en, "## What's new") || !strings.Contains(en, "English line") {
		t.Errorf("en : %q", en)
	}
	fr, err := svc.Build(context.Background(), "fr")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(fr, "## Dernières nouveautés") || !strings.Contains(fr, "Ligne française") {
		t.Errorf("fr : %q", fr)
	}
	if strings.Contains(en, "anglais") || strings.Contains(fr, "anglais uniquement") {
		t.Error("aucune mention de repli pour une langue qui a son fichier")
	}
}

// D6 : une langue sans fichier reçoit la version de la langue de repli précédée d'une mention dans SA
// langue. Le test retire le fichier français : le français n'a plus de fichier, il tient lieu de langue
// « sans fichier » (la liste ne compte que fr et en).
func TestReleaseNotes_LangueSansFichierRecoitLeRepliAvecMention(t *testing.T) {
	root := t.TempDir()
	writeNotes(t, root, "docs/RELEASE_NOTES.md", notesEN)
	svc := NewReleaseNotesService(root)

	before := fallbackDocuments()
	fr, err := svc.Build(context.Background(), "fr")
	if err != nil {
		t.Fatal(err)
	}
	mention := i18n.Msg(i18n.LocaleFR, "release_notes.english_only", nil)
	if !strings.HasPrefix(fr, mention+"\n\n## What's new") {
		t.Errorf("mention dans la langue demandée, puis les notes anglaises attendues : %q", fr)
	}
	if !strings.Contains(fr, "English line") {
		t.Errorf("contenu anglais attendu : %q", fr)
	}
	if fallbackDocuments() != before+1 {
		t.Error("le repli doit être compté")
	}
}

// La langue de repli sans son fichier est une erreur : rien à servir, rien d'inventé.
func TestReleaseNotes_SansAucunFichierEstUneErreur(t *testing.T) {
	svc := NewReleaseNotesService(t.TempDir())
	if _, err := svc.Build(context.Background(), "en"); err == nil {
		t.Error("erreur attendue sans fichier de notes")
	}
}

func TestReleaseNotes_TitreDeSectionReconnuDansToutesLesLangues(t *testing.T) {
	for _, h := range []string{"## What's new", "## Dernières nouveautés"} {
		if !isWhatsNewHeading(h) {
			t.Errorf("titre non reconnu : %q", h)
		}
	}
	if isWhatsNewHeading("## Autre section") {
		t.Error("une autre section n'est pas le titre des nouveautés")
	}
}
