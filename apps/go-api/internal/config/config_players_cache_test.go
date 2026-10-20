package config

// config_players_cache_test.go — lecture de db_profiles.json gardée par version du
// fichier (plan perf 2026-09-23, lot L5b, D5b.5) : relu seulement quand le fichier
// change, et le PATCH des réglages reste visible sans redémarrage.

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// profilesV3 rend un db_profiles.json v3 à un joueur ; gamertags de même longueur
// → fichiers de même taille (seul l'horodatage peut alors signaler le changement).
func profilesV3(gamertag string) []byte {
	return []byte(`{"version":"3.0","profiles":{"halo_infinite":{"` + gamertag + `":{"xuid":"1"}}}}`)
}

// writeProfiles écrit le fichier puis fixe son horodatage (hors fenêtre de
// méfiance quand mtime est ancien).
func writeProfiles(t *testing.T, path string, content []byte, mtime time.Time) {
	t.Helper()
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("écriture : %v", err)
	}
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatalf("chtimes : %v", err)
	}
}

func onlyGamertag(t *testing.T, cfg *AppConfig) string {
	t.Helper()
	players, err := cfg.LoadPlayers("halo_infinite")
	if err != nil {
		t.Fatalf("LoadPlayers : %v", err)
	}
	if len(players) != 1 {
		t.Fatalf("joueurs = %d, want 1", len(players))
	}
	return players[0].Gamertag
}

// TestLoadPlayers_ReadOncePerFileVersion : même horodatage et même taille = le
// contenu mémorisé est servi sans relire le fichier (le remplacer en conservant son
// horodatage ne se voit pas) ; un horodatage plus récent = relecture.
func TestLoadPlayers_ReadOncePerFileVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "db_profiles.json")
	cfg := &AppConfig{DBProfilesPath: path}
	old := time.Now().Add(-time.Hour)

	writeProfiles(t, path, profilesV3("Alpha"), old)
	if got := onlyGamertag(t, cfg); got != "Alpha" {
		t.Fatalf("1re lecture = %q, want Alpha", got)
	}

	writeProfiles(t, path, profilesV3("Omega"), old) // contenu changé, horodatage et taille identiques
	if got := onlyGamertag(t, cfg); got != "Alpha" {
		t.Fatalf("même version du fichier : %q, want Alpha (servi sans relecture)", got)
	}

	writeProfiles(t, path, profilesV3("Omega"), old.Add(10*time.Minute)) // PATCH : horodatage plus récent
	if got := onlyGamertag(t, cfg); got != "Omega" {
		t.Errorf("horodatage plus récent : %q, want Omega (relecture)", got)
	}
}

// TestLoadPlayers_SizeChangeRereads : une taille différente suffit à relire.
func TestLoadPlayers_SizeChangeRereads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "db_profiles.json")
	cfg := &AppConfig{DBProfilesPath: path}
	old := time.Now().Add(-time.Hour)
	writeProfiles(t, path, profilesV3("Alpha"), old)
	onlyGamertag(t, cfg)
	writeProfiles(t, path, profilesV3("Alphonse"), old)
	if got := onlyGamertag(t, cfg); got != "Alphonse" {
		t.Errorf("taille changée : %q, want Alphonse", got)
	}
}

// TestLoadPlayers_RecentWriteNotTrusted : un fichier modifié à l'instant est relu à
// chaque appel — deux écritures dans le même tic d'horloge garderaient le même
// horodatage et la même taille.
func TestLoadPlayers_RecentWriteNotTrusted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "db_profiles.json")
	cfg := &AppConfig{DBProfilesPath: path}
	now := time.Now()
	writeProfiles(t, path, profilesV3("Alpha"), now)
	onlyGamertag(t, cfg)
	writeProfiles(t, path, profilesV3("Omega"), now) // même horodatage, même taille
	if got := onlyGamertag(t, cfg); got != "Omega" {
		t.Errorf("écriture récente : %q, want Omega (horodatage trop récent pour être cru)", got)
	}
}

// TestLoadPlayers_FileRemovedThenRecreated : fichier supprimé = liste vide ;
// recréé = relu.
func TestLoadPlayers_FileRemovedThenRecreated(t *testing.T) {
	path := filepath.Join(t.TempDir(), "db_profiles.json")
	cfg := &AppConfig{DBProfilesPath: path}
	old := time.Now().Add(-time.Hour)
	writeProfiles(t, path, profilesV3("Alpha"), old)
	onlyGamertag(t, cfg)

	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	players, err := cfg.LoadPlayers("halo_infinite")
	if err != nil || len(players) != 0 {
		t.Fatalf("fichier absent : %d joueurs, err=%v, want liste vide", len(players), err)
	}
	writeProfiles(t, path, profilesV3("Omega"), old)
	if got := onlyGamertag(t, cfg); got != "Omega" {
		t.Errorf("fichier recréé : %q, want Omega", got)
	}
}

// TestLoadPlayers_ReturnsFreshSlices : muter la liste rendue ne touche pas la suivante.
func TestLoadPlayers_ReturnsFreshSlices(t *testing.T) {
	path := filepath.Join(t.TempDir(), "db_profiles.json")
	cfg := &AppConfig{DBProfilesPath: path}
	writeProfiles(t, path, profilesV3("Alpha"), time.Now().Add(-time.Hour))
	players, _ := cfg.LoadPlayers("halo_infinite")
	players[0].Gamertag = "muté"
	if got := onlyGamertag(t, cfg); got != "Alpha" {
		t.Errorf("liste partagée entre appels : %q", got)
	}
}

// TestLoadPlayers_RacyWindowSnapshotNeverStored (lot perf L9-go, revue adversariale B) :
// un contenu lu PENDANT la fenêtre de méfiance n'est pas mémorisé. Scénario de la revue :
// quatre écritures dans le même tic (même horodatage, même taille), la dernière dit Omega ;
// la lecture faite une fois la fenêtre passée doit rendre Omega, pas l'instantané Alpha lu
// dans la fenêtre.
func TestLoadPlayers_RacyWindowSnapshotNeverStored(t *testing.T) {
	path := filepath.Join(t.TempDir(), "db_profiles.json")
	cfg := &AppConfig{DBProfilesPath: path}
	tic := time.Now().Add(-dbProfilesRacyWindow + 300*time.Millisecond) // dans la fenêtre

	writeProfiles(t, path, profilesV3("Alpha"), tic)
	if got := onlyGamertag(t, cfg); got != "Alpha" {
		t.Fatalf("1re lecture (dans la fenêtre) = %q, want Alpha", got)
	}
	writeProfiles(t, path, profilesV3("Omega"), tic)
	if got := onlyGamertag(t, cfg); got != "Omega" {
		t.Fatalf("2e lecture (dans la fenêtre) = %q, want Omega", got)
	}
	writeProfiles(t, path, profilesV3("Alpha"), tic)
	if got := onlyGamertag(t, cfg); got != "Alpha" {
		t.Fatalf("3e lecture (dans la fenêtre) = %q, want Alpha", got)
	}
	writeProfiles(t, path, profilesV3("Omega"), tic) // même tic, même taille : le fichier dit Omega
	time.Sleep(600 * time.Millisecond)               // la fenêtre est passée
	if got := onlyGamertag(t, cfg); got != "Omega" {
		t.Errorf("après la fenêtre : %q servi, alors que le fichier contient Omega (instantané pris dans la fenêtre)", got)
	}
}
