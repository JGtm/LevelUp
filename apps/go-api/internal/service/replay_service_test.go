package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
)

// artefactAt écrit un artefact de rejeu factice pour un match, sous le chemin que
// PathResolver lui assigne, et rend la racine de dépôt à passer au service.
func artefactAt(t *testing.T, titleSlug, matchID, body string) string {
	t.Helper()
	root := t.TempDir()
	path := title.NewPathResolver(root).ReplayArtifactPath(titleSlug, matchID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return root
}

// TestIsAvailable_PresenceEtAbsence — le contrat du lot 1.2 : la Match View ne pose un
// lien que là où il mène quelque part.
func TestIsAvailable_PresenceEtAbsence(t *testing.T) {
	root := artefactAt(t, title.DefaultSlug, "abc123", `{"schemaVersion":1}`)
	svc := NewReplayService(title.DefaultSlug, root, nil)

	if !svc.IsAvailable(context.Background(), "abc123") {
		t.Error("artefact présent : IsAvailable devrait être vrai")
	}
	if svc.IsAvailable(context.Background(), "inconnu") {
		t.Error("artefact absent : IsAvailable devrait être faux")
	}
	// Racine sans aucun répertoire de rejeu : pas d'erreur, pas de lien.
	if NewReplayService(title.DefaultSlug, t.TempDir(), nil).IsAvailable(context.Background(), "abc123") {
		t.Error("répertoire de rejeu absent : IsAvailable devrait être faux")
	}
}

// TestIsAvailable_NeLitPasLArtefact — LA raison d'être de la méthode. Le corps est
// un JSON INVALIDE et gigantesque devant ce qu'un test manipule : si l'implémentation
// lisait ou désérialisait, elle ne pourrait pas répondre « oui ».
func TestIsAvailable_NeLitPasLArtefact(t *testing.T) {
	root := artefactAt(t, title.DefaultSlug, "gros", "ceci n'est pas du JSON")
	svc := NewReplayService(title.DefaultSlug, root, nil)

	if !svc.IsAvailable(context.Background(), "gros") {
		t.Fatal("IsAvailable doit répondre sur la PRÉSENCE, pas sur le contenu")
	}
	// Contre-épreuve : la lecture, elle, échoue bien sur ce même fichier.
	if _, err := svc.GetReplay(context.Background(), "gros"); err == nil {
		t.Error("GetReplay devrait échouer sur un artefact non désérialisable")
	}
}

// TestIsAvailable_RepertoireNEstPasUnArtefact — un répertoire au chemin de l'artefact
// n'est pas un rejeu : le lien mènerait à un 404.
func TestIsAvailable_RepertoireNEstPasUnArtefact(t *testing.T) {
	root := t.TempDir()
	path := title.NewPathResolver(root).ReplayArtifactPath(title.DefaultSlug, "rep")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if NewReplayService(title.DefaultSlug, root, nil).IsAvailable(context.Background(), "rep") {
		t.Error("un répertoire ne doit pas passer pour un artefact")
	}
}

// TestIsAvailable_IsoleParTitre — deux titres, deux chemins : l'artefact d'un titre ne
// rend jamais l'autre disponible (isolation par chemin FS, ADR 0008).
func TestIsAvailable_IsoleParTitre(t *testing.T) {
	root := artefactAt(t, title.DefaultSlug, "m1", `{}`)
	if NewReplayService("halo_5", root, nil).IsAvailable(context.Background(), "m1") {
		t.Error("l'artefact d'un titre ne doit pas être vu par un autre titre")
	}
}

// TestIsAvailable_FormeCourteEtFormeComplete — LE DÉFAUT QUI RENDAIT LE LOT 1 INVISIBLE.
//
// L'outil hors ligne écrit l'artefact sous la forme COURTE du match_id (celle du film
// qu'il vient de décoder : `000d5950`) ; l'application, elle, ne manipule que le match_id
// COMPLET (`000d5950-….`). Tant que le chemin ne normalisait pas, le service cherchait un
// fichier que personne n'écrit : l'artefact existait, la réponse était « aucun rejeu », et
// le lien n'aurait JAMAIS pu apparaître.
func TestIsAvailable_FormeCourteEtFormeComplete(t *testing.T) {
	const court = "000d5950"
	const complet = "000d5950-1234-4abc-9def-0123456789ab"
	root := artefactAt(t, title.DefaultSlug, court, `{}`)
	svc := NewReplayService(title.DefaultSlug, root, nil)

	if !svc.IsAvailable(context.Background(), court) {
		t.Error("forme courte : l'artefact doit être trouvé")
	}
	if !svc.IsAvailable(context.Background(), complet) {
		t.Error("forme COMPLÈTE : c'est celle que l'application passe — l'artefact doit être trouvé")
	}
	// Et la lecture emprunte le même chemin que la présence : les deux ne peuvent pas diverger.
	if _, err := svc.GetReplay(context.Background(), complet); err != nil {
		t.Errorf("GetReplay sur la forme complète : %v", err)
	}
	// Un autre match reste absent : la normalisation ne rend pas tout disponible.
	if svc.IsAvailable(context.Background(), "0a1b2c3d-1234-4abc-9def-0123456789ab") {
		t.Error("un match sans artefact ne doit pas devenir disponible")
	}
}

// TestApplyMatchHeaderReplay — le header publie la présence, et rien d'autre.
func TestApplyMatchHeaderReplay(t *testing.T) {
	root := artefactAt(t, title.DefaultSlug, "avec", `{}`)
	svc := NewReplayService(title.DefaultSlug, root, nil)

	h := domain.MatchViewHeader{}
	applyMatchHeaderReplay(context.Background(), &h, "avec", svc)
	if !h.ReplayAvailable {
		t.Error("match avec artefact : replay_available devrait être vrai")
	}

	h = domain.MatchViewHeader{}
	applyMatchHeaderReplay(context.Background(), &h, "sans", svc)
	if h.ReplayAvailable {
		t.Error("match sans artefact : replay_available devrait être faux")
	}
}

// TestApplyMatchHeaderReplay_SansService — un titre sans rejeu (service non injecté) ne
// doit ni paniquer ni allumer le lien : c'est la dégradation gracieuse du dépôt.
func TestApplyMatchHeaderReplay_SansService(t *testing.T) {
	h := domain.MatchViewHeader{ReplayAvailable: true}
	applyMatchHeaderReplay(context.Background(), &h, "m1", nil)
	if !h.ReplayAvailable {
		t.Error("service nil : le champ ne doit pas être touché, pas remis à faux par erreur")
	}
	h2 := domain.MatchViewHeader{}
	applyMatchHeaderReplay(context.Background(), &h2, "m1", nil)
	if h2.ReplayAvailable {
		t.Error("service nil : aucun lien ne doit s'allumer")
	}
	// Un header nil ne doit pas paniquer (le builder peut être appelé sur un match vide).
	applyMatchHeaderReplay(context.Background(), nil, "m1", nil)
}
