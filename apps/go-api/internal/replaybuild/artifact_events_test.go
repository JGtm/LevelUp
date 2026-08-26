package replaybuild

// artifact_events_test.go — LE POINT D'ANCRAGE PUBLIE UNE FOIS, ET SEULEMENT QUAND IL ÉCRIT.
//
// Ces tests tiennent les deux moitiés du contrat de la notification groupée :
//
//	ce qui DOIT publier    les DEUX chaînes d'écriture du serveur — le dépôt d'un ouvrier
//	                       distant (StoreArtifact) et la construction locale / l'action
//	                       admin (writeArtifact, par où passe BuildMatch) ;
//	ce qui NE DOIT PAS     le refus anti-régression, qui rend un digest SANS ERREUR alors que
//	                       RIEN n'a été écrit. Annoncer un rejeu « prêt » sur ce chemin
//	                       annoncerait un fichier que personne n'a touché.

import (
	"os"
	"testing"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/title"
)

// capturePuits installe un puits qui enregistre les événements et le retire à la fin du
// test. Le puits est un état de PROCESS : ces tests ne doivent donc jamais tourner en
// parallèle (aucun t.Parallel ici, délibérément).
func capturePuits(t *testing.T) *[]ArtifactStored {
	t.Helper()
	var vus []ArtifactStored
	SetArtifactStoredSink(func(ev ArtifactStored) { vus = append(vus, ev) })
	t.Cleanup(func() { SetArtifactStoredSink(nil) })
	return &vus
}

// TestPuits_DepotOuvrierPublieUnEvenement — CHAÎNE OUVRIER : le dépôt HTTP d'un artefact
// construit ailleurs (StoreArtifact) annonce le rejeu, avec l'identité DU JOB.
func TestPuits_DepotOuvrierPublieUnEvenement(t *testing.T) {
	vus := capturePuits(t)
	repoRoot := t.TempDir()
	const matchID = "000d5950"
	blob := docJSON(t, matchID, true)

	stored, err := StoreArtifact(repoRoot, title.DefaultSlug, matchID, blob)
	if err != nil {
		t.Fatalf("StoreArtifact: %v", err)
	}
	if len(*vus) != 1 {
		t.Fatalf("%d événement(s) publié(s), attendu exactement 1", len(*vus))
	}
	ev := (*vus)[0]
	if ev.TitleSlug != title.DefaultSlug || ev.MatchID != matchID {
		t.Errorf("identité = (%q, %q), attendu (%q, %q)",
			ev.TitleSlug, ev.MatchID, title.DefaultSlug, matchID)
	}
	if ev.Path != stored.Path {
		t.Errorf("path = %q, attendu %q (celui de l'accusé)", ev.Path, stored.Path)
	}
	if ev.Bytes != len(blob) || ev.Tracks != 1 || ev.SchemaVersion != replay.SchemaVersion {
		t.Errorf("event = %+v, attendu %d octets, 1 piste, schéma %d",
			ev, len(blob), replay.SchemaVersion)
	}
}

// TestPuits_ConstructionLocalePublieUnEvenement — CHAÎNE LOCALE : le point d'écriture par
// lequel passent le fil de l'eau post-sync et l'action admin (BuildMatch -> writeArtifact).
func TestPuits_ConstructionLocalePublieUnEvenement(t *testing.T) {
	vus := capturePuits(t)
	dir := t.TempDir()
	path := dir + "/000d5950.json"
	doc := replay.ReplayDocument{
		SchemaVersion: replay.SchemaVersion,
		MatchID:       "000d5950",
		TitleSlug:     title.DefaultSlug,
		Tracks:        []replay.Track{{XUID: "2533274819954312"}},
	}

	if _, err := writeArtifact(path, title.DefaultSlug, "000d5950", doc); err != nil {
		t.Fatalf("writeArtifact: %v", err)
	}
	if len(*vus) != 1 {
		t.Fatalf("%d événement(s) publié(s), attendu exactement 1", len(*vus))
	}
	if ev := (*vus)[0]; ev.MatchID != "000d5950" || ev.TitleSlug != title.DefaultSlug || ev.Path != path {
		t.Errorf("event = %+v, attendu match 000d5950 / %s / %s", ev, title.DefaultSlug, path)
	}
}

// TestPuits_RegressionRefuseeNePublieRien — LE CAS QUI COMPTE. Le garde anti-régression
// conserve l'artefact en place et rend un digest sans erreur : aucun rejeu n'est devenu
// disponible, donc aucune annonce.
func TestPuits_RegressionRefuseeNePublieRien(t *testing.T) {
	repoRoot := t.TempDir()
	const matchID = "000d5950"
	if _, err := StoreArtifact(repoRoot, title.DefaultSlug, matchID, docJSON(t, matchID, true)); err != nil {
		t.Fatalf("dépôt initial: %v", err)
	}

	// Puits installé APRÈS le dépôt initial : seul le dépôt appauvri est observé.
	vus := capturePuits(t)
	if _, err := StoreArtifact(repoRoot, title.DefaultSlug, matchID, docJSON(t, matchID, false)); err != nil {
		t.Fatalf("dépôt appauvri: %v", err)
	}
	if len(*vus) != 0 {
		t.Fatalf("%d événement(s) publié(s) pour une écriture REFUSÉE, attendu 0 : "+
			"la notification annoncerait un fichier que personne n'a touché", len(*vus))
	}
}

// TestPuits_ErreurDEcritureNePublieRien — un chemin impossible à créer ne publie rien.
func TestPuits_ErreurDEcritureNePublieRien(t *testing.T) {
	vus := capturePuits(t)
	dir := t.TempDir()
	// Un FICHIER là où writeArtifactBytes voudrait un répertoire parent : MkdirAll échoue.
	barrage := dir + "/barrage"
	if err := os.WriteFile(barrage, []byte("x"), 0o644); err != nil {
		t.Fatalf("pose du barrage: %v", err)
	}
	if _, err := writeArtifactBytes(barrage+"/sous/000d5950.json",
		title.DefaultSlug, "000d5950", docJSON(t, "000d5950", true)); err == nil {
		t.Skip("le système de fichiers a accepté un répertoire sous un fichier — cas non exerçable ici")
	}
	if len(*vus) != 0 {
		t.Fatalf("%d événement(s) publié(s) alors que l'écriture a échoué, attendu 0", len(*vus))
	}
}

// TestPuits_SansPuitsEtPuitsEnPanique — l'écriture d'un artefact ne dépend JAMAIS de la
// bonne santé de la notification : le rejeu est le produit, l'annonce n'en est que l'écho.
func TestPuits_SansPuitsEtPuitsEnPanique(t *testing.T) {
	repoRoot := t.TempDir()
	// Aucun puits câblé (cas nominal des CLI et de l'ouvrier) : aucune panique.
	SetArtifactStoredSink(nil)
	if _, err := StoreArtifact(repoRoot, title.DefaultSlug, "000d5950", docJSON(t, "000d5950", true)); err != nil {
		t.Fatalf("dépôt sans puits câblé: %v", err)
	}

	SetArtifactStoredSink(func(ArtifactStored) { panic("puits cassé") })
	t.Cleanup(func() { SetArtifactStoredSink(nil) })
	stored, err := StoreArtifact(repoRoot, title.DefaultSlug, "111e6061", docJSON(t, "111e6061", true))
	if err != nil {
		t.Fatalf("un puits en panique a fait échouer une écriture d'artefact : %v", err)
	}
	if _, statErr := os.Stat(stored.Path); statErr != nil {
		t.Fatalf("artefact absent du disque après panique du puits : %v", statErr)
	}
}
