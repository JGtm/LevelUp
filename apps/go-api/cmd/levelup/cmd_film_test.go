package main

// cmd_film_test.go — `levelup film recompress` et `film archive-move` (lot L14).

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JGtm/LevelUp/libs/film/filmcache"
	"github.com/JGtm/LevelUp/libs/film/finalise"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	settings_platform "levelup/go-api/internal/platform/settings"
)

func filmDeTest() []filmcache.WriteChunk {
	return []filmcache.WriteChunk{
		{Index: 0, ChunkType: 1, Data: bytes.Repeat([]byte("registre"), 2000)},
		{Index: 1, ChunkType: 2, DurationMS: 20000, Data: bytes.Repeat([]byte("replication"), 4000)},
		{Index: 2, ChunkType: finalise.ChunkTypeTempsForts, DurationMS: 20000, Data: bytes.Repeat([]byte("temps forts"), 3000)},
	}
}

// poserFilmBrut écrit un film comme l'ancien writer : chunks bruts, manifeste à la taille du fichier.
func poserFilmBrut(t *testing.T, root, court string) {
	t.Helper()
	if err := filmcache.Write(t.Context(), root, court, filmDeTest()); err != nil {
		t.Fatal(err)
	}
	// Réécrit chaque chunk en clair : l'état d'un cache d'avant D-18.
	for _, c := range filmDeTest() {
		if err := os.WriteFile(filmcache.CheminDuChunk(filmcache.ChunkDir(root, court), c.Index), c.Data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func tailleChunks(t *testing.T, root, court string) int64 {
	t.Helper()
	var total int64
	for _, c := range filmDeTest() {
		info, err := os.Stat(filmcache.CheminDuChunk(filmcache.ChunkDir(root, court), c.Index))
		if err != nil {
			t.Fatal(err)
		}
		total += info.Size()
	}
	return total
}

func TestFilmRecompress_SimulationChiffreLeGainSansEcrire(t *testing.T) {
	root := t.TempDir()
	poserFilmBrut(t, root, "0000c001")
	poserFilmBrut(t, root, "0000c002")
	avant := tailleChunks(t, root, "0000c001")
	var out bytes.Buffer
	if err := runFilmTo(&config.AppConfig{RepoRoot: t.TempDir()}, []string{subcmdFilmRecompress, "--dry-run", "--cache", root}, &out); err != nil {
		t.Fatalf("recompress --dry-run : %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "6 chunks convertis") || !strings.Contains(out.String(), "gain") {
		t.Fatalf("la simulation ne chiffre pas le gain :\n%s", out.String())
	}
	if tailleChunks(t, root, "0000c001") != avant {
		t.Fatal("la simulation a modifié un chunk")
	}
}

func TestFilmRecompress_ConvertitPuisReprend(t *testing.T) {
	root := t.TempDir()
	poserFilmBrut(t, root, "0000c003")
	avant := tailleChunks(t, root, "0000c003")
	var out bytes.Buffer
	cfg := &config.AppConfig{RepoRoot: t.TempDir()}
	if err := runFilmTo(cfg, []string{subcmdFilmRecompress, "--cache", root}, &out); err != nil {
		t.Fatalf("recompress : %v\n%s", err, out.String())
	}
	if apres := tailleChunks(t, root, "0000c003"); apres >= avant/2 {
		t.Fatalf("chunks : %d -> %d octets, attendu au moins la moitié gagnée sur du répétitif", avant, apres)
	}
	for _, c := range filmDeTest() {
		lu, err := filmcache.LireChunk(root, "0000c003", c.Index)
		if err != nil || !bytes.Equal(lu, c.Data) {
			t.Fatalf("chunk %d relu différent : %v", c.Index, err)
		}
	}
	out.Reset()
	if err := runFilmTo(cfg, []string{subcmdFilmRecompress, "--cache", root}, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "0 chunks convertis, 3 deja en zstd") {
		t.Fatalf("la reprise ne saute pas les chunks déjà en zstd :\n%s", out.String())
	}
}

func TestFilmRecompress_UnFilmEnErreurNArretePasLaPasse(t *testing.T) {
	root := t.TempDir()
	poserFilmBrut(t, root, "0000c004")
	err := runFilmTo(&config.AppConfig{RepoRoot: t.TempDir()},
		[]string{subcmdFilmRecompress, "--cache", root, "--films", "00000000,0000c004"}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "1 films en erreur") {
		t.Fatalf("erreur = %v, attendu le compte des films en erreur", err)
	}
	if got, _ := filmcache.LireChunk(root, "0000c004", 1); len(got) == 0 {
		t.Fatal("le film sain n'a pas été traité après le film absent")
	}
}

func TestFilmArchiveMove_RangeLesFilmsEtEnregistreLeDossier(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.AppConfig{RepoRoot: dir, AppSettingsPath: filepath.Join(dir, "app_settings.json")}
	source := cfg.PathResolver().FilmArchiveDir()
	poserFilmBrut(t, source, "0000d001")
	cible := filepath.Join(t.TempDir(), "films")

	var out bytes.Buffer
	if err := runFilmTo(cfg, []string{subcmdFilmArchiveMove, "--to", cible}, &out); err != nil {
		t.Fatalf("archive-move : %v\n%s", err, out.String())
	}
	if ok, err := filmcache.FilmComplet(cible, "0000d001"); err != nil || !ok {
		t.Fatalf("film absent ou incomplet à la cible : %v, %v", ok, err)
	}
	if ids, _ := filmcache.ListShortIDs(source); len(ids) != 0 {
		t.Fatalf("films restés à la source : %v", ids)
	}
	reglages, err := settings_platform.NewStore(cfg.AppSettingsPath).Load()
	if err != nil || reglages.FilmArchiveDir != cible {
		t.Fatalf("film_archive_dir = %q (%v), attendu %q", reglages.FilmArchiveDir, err, cible)
	}
}

func TestFilmArchiveMove_RefuseUneCibleNonVideEtNeToucheRien(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.AppConfig{RepoRoot: dir, AppSettingsPath: filepath.Join(dir, "app_settings.json")}
	source := cfg.PathResolver().FilmArchiveDir()
	poserFilmBrut(t, source, "0000d002")
	plein := t.TempDir()
	if err := os.WriteFile(filepath.Join(plein, "autre.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := runFilmTo(cfg, []string{subcmdFilmArchiveMove, "--to", plein}, &bytes.Buffer{})
	if !errors.Is(err, filmcache.ErrCibleNonVide) {
		t.Fatalf("erreur = %v, attendu ErrCibleNonVide", err)
	}
	if ids, _ := filmcache.ListShortIDs(source); len(ids) != 1 {
		t.Fatal("un déplacement refusé a touché à la source")
	}
	if _, serr := os.Stat(cfg.AppSettingsPath); serr == nil {
		t.Fatal("un déplacement refusé a écrit le réglage")
	}
}

func TestFilm_SousCommandeInconnueEtUsage(t *testing.T) {
	cfg := &config.AppConfig{RepoRoot: t.TempDir()}
	if err := runFilmTo(cfg, nil, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "usage") {
		t.Fatalf("sans argument : %v", err)
	}
	if err := runFilmTo(cfg, []string{"purge"}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "inconnue") {
		t.Fatalf("sous-commande inconnue : %v", err)
	}
	if err := runFilmTo(cfg, []string{subcmdFilmArchiveMove}, &bytes.Buffer{}); err == nil {
		t.Fatal("archive-move sans --to ni --default doit être refusé")
	}
}

// TestFilmArchiveMove_VersLEmplacementParDefaut : le cache par défaut n'est jamais vide (faits,
// verrous, images) ; il suffit qu'il ne contienne aucun film. Le réglage est vidé.
func TestFilmArchiveMove_VersLEmplacementParDefaut(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.AppConfig{RepoRoot: dir, AppSettingsPath: filepath.Join(dir, "app_settings.json")}
	custom := filepath.Join(t.TempDir(), "films")
	titlePkg.SetFilmArchiveRuntime(custom) // l'archive vit ailleurs que par défaut
	t.Cleanup(titlePkg.ResetFilmArchiveRuntime)
	defaut := cfg.PathResolver().Roots().DefaultFilmArchiveDir()
	if err := os.MkdirAll(filepath.Join(defaut, "film_facts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(defaut, "jobs.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	poserFilmBrut(t, custom, "0000d003")
	store := settings_platform.NewStore(cfg.AppSettingsPath)
	reglages, _ := store.Load()
	reglages.FilmArchiveDir = custom
	if err := store.Save(reglages); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := runFilmTo(cfg, []string{subcmdFilmArchiveMove, "--default"}, &out); err != nil {
		t.Fatalf("archive-move --default : %v\n%s", err, out.String())
	}
	if ok, err := filmcache.FilmComplet(defaut, "0000d003"); err != nil || !ok {
		t.Fatalf("film absent de l'emplacement par défaut : %v, %v", ok, err)
	}
	if apres, _ := store.Load(); apres.FilmArchiveDir != "" {
		t.Fatalf("film_archive_dir = %q, attendu vide", apres.FilmArchiveDir)
	}
}

// TestFilmArchiveMove_ResumeSansDeplacementEnCoursNEstPasUneDerogation : la cible non vide reste
// refusée.
func TestFilmArchiveMove_ResumeSansDeplacementEnCoursNEstPasUneDerogation(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.AppConfig{RepoRoot: dir, AppSettingsPath: filepath.Join(dir, "app_settings.json")}
	poserFilmBrut(t, cfg.PathResolver().FilmArchiveDir(), "0000d004")
	plein := t.TempDir()
	if err := os.WriteFile(filepath.Join(plein, "autre.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := runFilmTo(cfg, []string{subcmdFilmArchiveMove, "--to", plein, "--resume"}, &bytes.Buffer{})
	if !errors.Is(err, filmcache.ErrCibleNonVide) {
		t.Fatalf("erreur = %v, attendu ErrCibleNonVide", err)
	}
}
