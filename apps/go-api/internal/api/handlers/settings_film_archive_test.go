// Package handlers_test — settings_film_archive_test.go : l'archive des films dans les réglages
// (lot L14, ADR 0045 D-12) : combinaison refusée, déplacement refusé, occupation exposée.
package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JGtm/LevelUp/libs/film/filmcache"
	"github.com/JGtm/LevelUp/libs/film/finalise"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/platform/jobs"
	settings_platform "levelup/go-api/internal/platform/settings"
)

// archiveFixture : un routeur de réglages enraciné dans un dossier temporaire, dont l'archive
// par défaut (<données>/cache) est accessible pour y poser des films.
type archiveFixture struct {
	r       *chi.Mux
	store   *settings_platform.Store
	archive string // dossier effectif de l'archive
	dir     string
}

func newArchiveFixture(t *testing.T) archiveFixture {
	t.Helper()
	// Le dossier posé à chaud est une variable de processus : la remettre à zéro avant et après.
	titlePkg.ResetFilmArchiveRuntime()
	t.Cleanup(titlePkg.ResetFilmArchiveRuntime)
	dir := t.TempDir()
	cfg := &config.AppConfig{RepoRoot: dir, DBProfilesPath: filepath.Join(dir, "db_profiles.json")}
	store := settings_platform.NewStore(filepath.Join(dir, "app_settings.json"))
	h := handlers.NewSettingsHandler(cfg, store, jobs.NewStore(filepath.Join(dir, "jobs.json")))
	r := chi.NewRouter()
	h.Mount(r)
	return archiveFixture{r: r, store: store, archive: titlePkg.NewPathResolver(dir).FilmArchiveDir(), dir: dir}
}

func (f archiveFixture) patch(t *testing.T, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	return f.patchLocale(t, body, "")
}

// patchLocale envoie le PATCH avec la langue de la requête posée dans son contexte (comme le fait
// le middleware de langue en production) ; "" laisse la langue par défaut.
func (f archiveFixture) patchLocale(t *testing.T, body map[string]any, locale i18n.Locale) *httptest.ResponseRecorder {
	t.Helper()
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPatch, "/settings", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	if locale != "" {
		req = req.WithContext(i18n.WithContext(req.Context(), locale))
	}
	w := httptest.NewRecorder()
	f.r.ServeHTTP(w, req)
	return w
}

func ecrireFilm(t *testing.T, racine, court string) {
	t.Helper()
	err := filmcache.Write(t.Context(), racine, court, []filmcache.WriteChunk{
		{Index: 0, ChunkType: 1, Data: bytes.Repeat([]byte("registre"), 500)},
		{Index: 1, ChunkType: finalise.ChunkTypeTempsForts, Data: bytes.Repeat([]byte("temps forts"), 500)},
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
}

// TestFilmArchive_ParDefautOuiEtEmplacementParDefaut : sans réglage, l'archive est conservée et
// vit là où elle a toujours vécu.
func TestFilmArchive_ParDefautOuiEtEmplacementParDefaut(t *testing.T) {
	f := newArchiveFixture(t)
	cfg, err := f.store.Load()
	if err != nil || !cfg.ArchiveFilms || cfg.FilmArchiveDir != "" {
		t.Fatalf("défauts = %+v, %v ; attendu archive_films=true et dossier vide", cfg.FilmArchiveSettings, err)
	}
	if want := filepath.Join(f.dir, "data", "cache"); f.archive != want {
		t.Fatalf("archive par défaut = %s, attendu %s", f.archive, want)
	}
}

// TestFilmArchive_SansArchiveEtRetentionFinieRefuse : la combinaison est refusée, dans les deux
// sens.
func TestFilmArchive_SansArchiveEtRetentionFinieRefuse(t *testing.T) {
	f := newArchiveFixture(t)
	if w := f.patch(t, map[string]any{"replay_retention_months": 6}); w.Code != http.StatusOK {
		t.Fatalf("rétention seule : %d %s", w.Code, w.Body.String())
	}
	w := f.patch(t, map[string]any{"archive_films": false})
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "film_archive_retention_conflict") {
		t.Fatalf("sans archive + rétention 6 mois : %d %s, attendu 400 film_archive_retention_conflict", w.Code, w.Body.String())
	}
	if cfg, _ := f.store.Load(); !cfg.ArchiveFilms {
		t.Fatal("le refus a quand même enregistré le réglage")
	}
	// Même requête avec la rétention remise à illimité : acceptée.
	if w := f.patch(t, map[string]any{"archive_films": false, "replay_retention_months": 0}); w.Code != http.StatusOK {
		t.Fatalf("sans archive + illimité : %d %s", w.Code, w.Body.String())
	}
	// Et l'inverse : une rétention finie est refusée tant que l'archive est désactivée.
	if w = f.patch(t, map[string]any{"replay_retention_months": 3}); w.Code != http.StatusBadRequest {
		t.Fatalf("rétention 3 sans archive : %d, attendu 400", w.Code)
	}
}

// TestFilmArchive_LectureReplieLaRetentionSurIllimitee : un fichier édité à la main avec la
// combinaison interdite n'arme pas la purge.
func TestFilmArchive_LectureReplieLaRetentionSurIllimitee(t *testing.T) {
	f := newArchiveFixture(t)
	brut := `{"archive_films": false, "replay_retention_months": 6}`
	if err := os.WriteFile(filepath.Join(f.dir, "app_settings.json"), []byte(brut), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := f.store.Load()
	if err != nil || cfg.ReplayRetentionMonths != 0 || cfg.ArchiveFilms {
		t.Fatalf("lecture = %+v, %v ; attendu rétention 0 et archive désactivée", cfg, err)
	}
}

// TestFilmArchive_DeplacementRefuse : changer d'emplacement vers un chemin relatif ou un dossier
// non vide est refusé, et le refus n'enregistre rien.
func TestFilmArchive_DeplacementRefuse(t *testing.T) {
	f := newArchiveFixture(t)
	cible := filepath.Join(t.TempDir(), "autre")

	// Archive vide, cible absente : accepté, et le réglage est enregistré.
	if w := f.patch(t, map[string]any{"film_archive_dir": cible}); w.Code != http.StatusOK {
		t.Fatalf("archive vide, cible absente : %d %s", w.Code, w.Body.String())
	}
	if cfg, _ := f.store.Load(); cfg.FilmArchiveDir != cible {
		t.Fatalf("film_archive_dir = %q, attendu %q", cfg.FilmArchiveDir, cible)
	}

	// Chemin relatif : refusé.
	if w := f.patch(t, map[string]any{"film_archive_dir": "relatif/dossier"}); w.Code != http.StatusBadRequest ||
		!strings.Contains(w.Body.String(), "film_archive_not_absolute") {
		t.Fatalf("chemin relatif : %d %s", w.Code, w.Body.String())
	}

	// Cible non vide : refusé.
	plein := t.TempDir()
	if err := os.WriteFile(filepath.Join(plein, "x.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	w := f.patch(t, map[string]any{"film_archive_dir": plein})
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "film_archive_target_not_empty") {
		t.Fatalf("cible non vide : %d %s", w.Code, w.Body.String())
	}
	if cfg, _ := f.store.Load(); cfg.FilmArchiveDir != cible {
		t.Fatalf("un refus a modifié le réglage : %q", cfg.FilmArchiveDir)
	}
}

// TestFilmArchive_DeplacementRefuseQuandLArchiveContientDesFilms : seule la commande de
// déplacement range des films ; le réglage ne les laisse pas derrière.
func TestFilmArchive_DeplacementRefuseQuandLArchiveContientDesFilms(t *testing.T) {
	f := newArchiveFixture(t)
	ecrireFilm(t, f.archive, "0000f001")
	w := f.patch(t, map[string]any{"film_archive_dir": filepath.Join(t.TempDir(), "vide")})
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "film_archive_move_required") {
		t.Fatalf("archive non vide : %d %s, attendu 400 film_archive_move_required", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "archive-move") {
		t.Fatalf("le message ne nomme pas la commande de déplacement : %s", w.Body.String())
	}
	// Réécrire la même valeur (aucun changement) reste accepté.
	if w := f.patch(t, map[string]any{"film_archive_dir": ""}); w.Code != http.StatusOK {
		t.Fatalf("réécriture du réglage inchangé : %d %s", w.Code, w.Body.String())
	}
}

// TestFilmArchive_OccupationExposeeEtMiseEnCache : le nombre de films, la taille et le volume
// sont exposés ; une seconde lecture ne reparcourt pas l'archive.
func TestFilmArchive_OccupationExposeeEtMiseEnCache(t *testing.T) {
	f := newArchiveFixture(t)
	ecrireFilm(t, f.archive, "0000f002")
	ecrireFilm(t, f.archive, "0000f003")
	get := func() map[string]any {
		req := httptest.NewRequest(http.MethodGet, "/settings/film-archive", nil)
		w := httptest.NewRecorder()
		f.r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("GET : %d %s", w.Code, w.Body.String())
		}
		var out map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &out)
		return out
	}
	premier := get()
	if premier["films"].(float64) != 2 || premier["bytes"].(float64) <= 0 || premier["archive_films"] != true {
		t.Fatalf("occupation = %v", premier)
	}
	if premier["dir"] != f.archive || premier["restart_required"] != false {
		t.Fatalf("dossier = %v, restart = %v", premier["dir"], premier["restart_required"])
	}
	if _, ok := premier["volume"].(map[string]any); !ok {
		t.Fatalf("volume absent : %v", premier)
	}
	ecrireFilm(t, f.archive, "0000f004") // ajouté après la mesure : le cache ne le voit pas encore
	if second := get(); second["films"].(float64) != 2 || second["measured_at"] != premier["measured_at"] {
		t.Fatalf("la mesure n'est pas mise en cache : %v", second)
	}
}

// TestFilmArchive_LeRefusEstRedigeDansLaLangueDeLaRequete : le même refus, en français et en
// anglais, avec le même code.
func TestFilmArchive_LeRefusEstRedigeDansLaLangueDeLaRequete(t *testing.T) {
	f := newArchiveFixture(t)
	if w := f.patch(t, map[string]any{"replay_retention_months": 6}); w.Code != http.StatusOK {
		t.Fatalf("rétention : %d", w.Code)
	}
	fr := f.patchLocale(t, map[string]any{"archive_films": false}, "fr")
	en := f.patchLocale(t, map[string]any{"archive_films": false}, "en")
	for _, w := range []*httptest.ResponseRecorder{fr, en} {
		if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "film_archive_retention_conflict") {
			t.Fatalf("refus attendu : %d %s", w.Code, w.Body.String())
		}
	}
	if !strings.Contains(fr.Body.String(), "doit rester illimitée") || !strings.Contains(en.Body.String(), "must stay unlimited") {
		t.Fatalf("messages non rédigés dans la langue de la requête : fr=%s ; en=%s", fr.Body.String(), en.Body.String())
	}
}

// TestFilmArchive_ArchiveVideChangeeAChaud : sur une archive vide (assistant de premier lancement),
// le nouveau dossier sert tout de suite — sans redémarrage — et est enregistré ; l'occupation
// l'annonce comme dossier effectif, sans « redémarrage requis ».
func TestFilmArchive_ArchiveVideChangeeAChaud(t *testing.T) {
	f := newArchiveFixture(t)
	cible := filepath.Join(t.TempDir(), "films")
	if w := f.patch(t, map[string]any{"film_archive_dir": cible}); w.Code != http.StatusOK {
		t.Fatalf("PATCH : %d %s", w.Code, w.Body.String())
	}
	if got := titlePkg.NewPathResolver(f.dir).FilmArchiveDir(); got != cible {
		t.Fatalf("FilmArchiveDir après PATCH = %q, attendu %q (sans redémarrage)", got, cible)
	}
	if cfg, _ := f.store.Load(); cfg.FilmArchiveDir != cible {
		t.Fatalf("réglage enregistré = %q", cfg.FilmArchiveDir)
	}
	req := httptest.NewRequest(http.MethodGet, "/settings/film-archive", nil)
	w := httptest.NewRecorder()
	f.r.ServeHTTP(w, req)
	var st map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &st)
	if st["dir"] != cible || st["restart_required"] != false {
		t.Fatalf("statut = %v", st)
	}
	// La première synchronisation écrit dans le nouveau dossier : le retour à l'ancien est refusé.
	ecrireFilm(t, cible, "0000f010")
	if w := f.patch(t, map[string]any{"film_archive_dir": ""}); w.Code != http.StatusBadRequest ||
		!strings.Contains(w.Body.String(), "film_archive_move_required") {
		t.Fatalf("retour avec un film : %d %s", w.Code, w.Body.String())
	}
	if got := titlePkg.NewPathResolver(f.dir).FilmArchiveDir(); got != cible {
		t.Fatalf("un refus a déplacé l'archive : %q", got)
	}
}

// TestFilmArchive_RetourALEmplacementParDefaut : l'archive vide revient au défaut, dossier de
// cache non vide compris (faits, verrous).
func TestFilmArchive_RetourALEmplacementParDefaut(t *testing.T) {
	f := newArchiveFixture(t)
	if err := os.MkdirAll(filepath.Join(f.archive, "film_facts"), 0o755); err != nil {
		t.Fatal(err)
	}
	cible := filepath.Join(t.TempDir(), "films")
	if w := f.patch(t, map[string]any{"film_archive_dir": cible}); w.Code != http.StatusOK {
		t.Fatalf("PATCH : %d %s", w.Code, w.Body.String())
	}
	if w := f.patch(t, map[string]any{"film_archive_dir": ""}); w.Code != http.StatusOK {
		t.Fatalf("retour au défaut : %d %s", w.Code, w.Body.String())
	}
	if got := titlePkg.NewPathResolver(f.dir).FilmArchiveDir(); got != f.archive {
		t.Fatalf("FilmArchiveDir = %q, attendu le défaut %q", got, f.archive)
	}
}
