package replayartifacts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/replayartifact"
)

// Un fichier tactique d'un format précédent n'est pas « courant » ; refait depuis le rejeu rangé, il l'est, sans aucun
// film : seul l'artefact est lu.
func TestRefaireRasterTactique_FichierDUnFormatPrecedentEstRefait(t *testing.T) {
	artefact := ecrireFichier(t, "artefact.json", artefactImmobile)
	sidecar := filepath.Join(t.TempDir(), "rasters", "ab000001.json.gz")

	ancien, err := json.Marshal(domain.TacticalRasterSidecar{
		SchemaVersion: domain.TacticalRasterSchemaVersion - 1, MatchID: "m", PasM: domain.TacticalRasterPasM,
		PasEchantillonMs: domain.TacticalRasterPasEchantillonMs,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := replayartifact.Write(t.Context(), sidecar, ancien); err != nil {
		t.Fatal(err)
	}
	if _, ok := replayartifact.LireSidecarRasterCourant(t.Context(), sidecar); ok {
		t.Fatal("un fichier tactique d'un format précédent ne doit pas se lire « courant »")
	}

	if err := RefaireRasterTactique(t.Context(), artefact, sidecar, true); err != nil {
		t.Fatalf("RefaireRasterTactique: %v", err)
	}
	s, ok := replayartifact.LireSidecarRasterCourant(t.Context(), sidecar)
	if !ok || s.SchemaVersion != domain.TacticalRasterSchemaVersion || len(s.Joueurs) != 2 {
		t.Fatalf("fichier refait : ok=%v schéma=%d joueurs=%d", ok, s.SchemaVersion, len(s.Joueurs))
	}
}

// Un fichier tactique absent se lit « non courant » sans rien dire ; un fichier illisible se dit.
func TestLireSidecarRasterCourant_AbsentSilencieuxIllisibleSeDit(t *testing.T) {
	dir := t.TempDir()
	journal := capturerJournal(t)
	if _, ok := replayartifact.LireSidecarRasterCourant(t.Context(), filepath.Join(dir, "absent.json.gz")); ok {
		t.Fatal("un fichier absent n'est pas courant")
	}
	if aDit(t, journal, "WARN", "illisible") {
		t.Fatalf("un fichier absent est le cas nominal, rien à dire : %s", journal)
	}
	illisible := filepath.Join(dir, "illisible.json.gz")
	if err := os.WriteFile(illisible, []byte("pas du gzip"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := replayartifact.LireSidecarRasterCourant(t.Context(), illisible); ok {
		t.Fatal("un fichier illisible n'est pas courant")
	}
	if !aDit(t, journal, "WARN", "illisible") {
		t.Fatalf("un fichier illisible doit se dire : %s", journal)
	}
}

// Un rejeu rangé absent : l'erreur est rendue et aucun fichier tactique n'est écrit.
func TestRefaireRasterTactique_RejeuAbsentRendUneErreurSansEcrire(t *testing.T) {
	dir := t.TempDir()
	sidecar := filepath.Join(dir, "rasters", "ab000002.json.gz")
	if err := RefaireRasterTactique(t.Context(), filepath.Join(dir, "absent.json"), sidecar, true); err == nil {
		t.Fatal("une erreur est attendue")
	}
	if replayartifact.Exists(sidecar) {
		t.Fatal("aucun fichier tactique ne doit être écrit")
	}
}

// L'en-tête rapide rend le même verdict que la lecture complète pour un fichier courant, d'un format précédent, d'une
// autre unité de temps, absent ou illisible ; et le document porte bien son en-tête avant les joueurs (un fichier réel
// se juge sans être décodé en entier).
func TestSidecarRasterCourantRapide_MemeVerdictQueLaLectureComplete(t *testing.T) {
	artefact := ecrireFichier(t, "artefact.json", artefactImmobile)
	courant, err := ProjeterRasterTactique(t.Context(), artefact, true)
	if err != nil || len(courant.Joueurs) == 0 {
		t.Fatalf("projection: %v", err)
	}
	cas := map[string]func(s *domain.TacticalRasterSidecar){
		"courant":           func(*domain.TacticalRasterSidecar) {},
		"format précédent":  func(s *domain.TacticalRasterSidecar) { s.SchemaVersion-- },
		"autre pas":         func(s *domain.TacticalRasterSidecar) { s.PasM = 1 },
		"autre échantillon": func(s *domain.TacticalRasterSidecar) { s.PasEchantillonMs = 100 },
	}
	dir := t.TempDir()
	for nom, mute := range cas {
		s := courant
		mute(&s)
		raw, err := json.Marshal(s)
		if err != nil {
			t.Fatal(err)
		}
		chemin := filepath.Join(dir, nom+".json.gz")
		if _, _, err := replayartifact.Write(t.Context(), chemin, raw); err != nil {
			t.Fatal(err)
		}
		debut, err := replayartifact.ReadHead(chemin, 2048)
		avant, apres := strings.Index(string(debut), `"pas_echantillon_ms"`), strings.Index(string(debut), `"joueurs"`)
		if err != nil || avant < 0 || (apres >= 0 && apres < avant) {
			t.Fatalf("%s : l'en-tête doit tenir au début du document, avant les joueurs (%v)", nom, err)
		}
		_, complet := replayartifact.LireSidecarRasterCourant(t.Context(), chemin)
		if rapide := replayartifact.SidecarRasterCourantRapide(t.Context(), chemin); rapide != complet || rapide != (nom == "courant") {
			t.Fatalf("%s : rapide=%v complet=%v", nom, rapide, complet)
		}
	}
	absent := filepath.Join(dir, "absent.json.gz")
	illisible := filepath.Join(dir, "illisible.json.gz")
	if err := os.WriteFile(illisible, []byte("pas du gzip"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, chemin := range []string{absent, illisible} {
		if replayartifact.SidecarRasterCourantRapide(t.Context(), chemin) {
			t.Fatalf("%s ne peut pas être courant", chemin)
		}
	}
}
