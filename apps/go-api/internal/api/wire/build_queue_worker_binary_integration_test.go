//go:build integration && cgo

// Package api — build_queue_worker_binary_integration_test.go : LA PREUVE AVEC UN
// VRAI OUVRIER.
//
// Le test de transport voisin prouve le protocole avec un artefact fabriqué à la
// main. Celui-ci va au bout de la chaîne telle qu'elle tournera : le BINAIRE
// cmd/replay-worker est compilé et lancé, il prend un job par HTTP, télécharge les
// morceaux depuis des URL « pré-signées », DÉCODE UN VRAI FILM, pousse l'artefact,
// et le serveur le range — où le service de rejeu le lit.
//
// CE QUI EST ISOLÉ, ET POURQUOI. L'ouvrier reçoit un dépôt À LUI (copie des seules
// références dont il a besoin : bornes de carte, libellés, géométrie, structures)
// et un dossier de travail temporaire. Rien n'est écrit dans le dépôt de
// l'utilisateur — surtout pas ses artefacts, et JAMAIS son cache film (archive
// irremplaçable, seulement lu ici).
//
// COÛT : un décodage de film complet (~45 morceaux, dizaines de secondes, mémoire
// non négligeable). D'où le tag `integration` et le saut automatique quand le
// cache film du témoin n'est pas là (CI, poste neuf) : ce test ne s'exécute que
// là où le film existe déjà.
package wire

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/ops"
	"levelup/go-api/internal/service"
)

// filmTemoin : le film de la preuve. Le plus petit du corpus nommé du garde local
// (Cliffhanger, Fiesta Slayer) — assez gros pour être un vrai décodage, assez
// court pour ne pas faire d'un test une punition.
const filmTemoin = "000d5950"

// carteTemoin : l'identité de carte candidate du film témoin (le web la résout
// depuis la base ; ici on la donne, comme le ferait la mise en file).
const carteTemoin = "Cliffhanger"

// TestOuvrierReel_ConstruitEtLivre : la chaîne complète, avec le vrai binaire.
func TestOuvrierReel_ConstruitEtLivre(t *testing.T) {
	depot := depotUtilisateur(t)
	chunks := manifesteDuFilm(t, depot)

	// ── Le CDN : les morceaux du cache, servis comme le fait Azure (zlib) ─────
	cdn := serveurDeMorceaux(t, depot)

	// ── Le web : les vraies routes, un dépôt vierge, une vraie base ──────────
	srv, reg, serveurRepo := transportStack(t)
	job := enqueueTravailReel(t, reg, cdn.URL, chunks)

	// ── L'ouvrier : son propre dépôt, son propre dossier de travail ──────────
	ouvrierRepo := depotOuvrier(t, depot)
	travail := t.TempDir()
	binaire := compilerOuvrier(t)

	debut := time.Now()
	lancerOuvrier(t, binaire, srv.URL+"/internal", ouvrierRepo, travail)
	t.Logf("décodage + livraison en %s", time.Since(debut).Round(time.Second))

	// ── Ce que le web a rangé ────────────────────────────────────────────────
	recu := filepath.Join(serveurRepo, "data", "cache", "replays", titlePkg.DefaultSlug, filmTemoin+".json")
	blobRecu, err := os.ReadFile(recu)
	if err != nil {
		t.Fatalf("aucun artefact rangé côté serveur (%s): %v", recu, err)
	}
	// ── … est exactement ce que l'ouvrier a construit ────────────────────────
	construit := filepath.Join(ouvrierRepo, "data", "cache", "replays", titlePkg.DefaultSlug, filmTemoin+".json")
	blobConstruit, err := os.ReadFile(construit)
	if err != nil {
		t.Fatalf("l'ouvrier n'a rien construit (%s): %v", construit, err)
	}
	if !bytes.Equal(blobRecu, blobConstruit) {
		t.Fatalf("artefact rangé ≠ artefact construit (%d vs %d octets)", len(blobRecu), len(blobConstruit))
	}

	// ── … et le service de rejeu le sert ─────────────────────────────────────
	doc, err := service.NewReplayService(titlePkg.DefaultSlug, serveurRepo, nil).
		GetReplay(context.Background(), filmTemoin)
	if err != nil {
		t.Fatalf("le service de rejeu ne lit pas l'artefact livré: %v", err)
	}
	if doc.SchemaVersion != replay.SchemaVersion || len(doc.Tracks) == 0 {
		t.Fatalf("document servi : schéma %d, %d trajectoires", doc.SchemaVersion, len(doc.Tracks))
	}
	t.Logf("artefact livré : %d octets, %d trajectoires, %d frames",
		len(blobRecu), len(doc.Tracks), doc.FrameCount)

	// ── Le job est `succeeded` (donc le compte rendu a trouvé le fichier) ────
	vue, err := reg.monitoringStore.BuildQueueReport(context.Background(), 10)
	if err != nil {
		t.Fatalf("BuildQueueReport: %v", err)
	}
	if vue.Counts.Succeeded != 1 {
		t.Fatalf("file après passage de l'ouvrier : %+v, attendu 1 fait (job %s)", vue.Counts, job.JobID)
	}

	// ── L'ouvrier n'a rien gardé : ses morceaux sont effacés ─────────────────
	if _, err := os.Stat(filepath.Join(travail, "film_chunks", filmTemoin)); !os.IsNotExist(err) {
		t.Errorf("l'ouvrier a conservé ses morceaux de film (%v) — il ne doit rien garder", err)
	}
}

// depotUtilisateur rend la racine du dépôt, ou saute le test si le film témoin
// n'y est pas en cache (rien à décoder : le test n'a pas de sens).
func depotUtilisateur(t *testing.T) string {
	t.Helper()
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Skip("chemin du test introuvable")
	}
	root, err := filepath.Abs(filepath.Join(filepath.Dir(ici), "..", "..", "..", "..", ".."))
	if err != nil {
		t.Skipf("racine du dépôt introuvable: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "data", "cache", "film_chunks", filmTemoin)); err != nil {
		t.Skipf("film témoin %s absent du cache local — preuve ouvrier non exécutable ici", filmTemoin)
	}
	return root
}

// manifesteChunk : la forme du manifeste de film du cache.
type manifesteChunk struct {
	Index      int `json:"index"`
	ChunkType  int `json:"chunk_type"`
	StartMS    int `json:"start_ms"`
	DurationMS int `json:"duration_ms"`
}

// manifesteDuFilm lit les morceaux déclarés du film témoin.
func manifesteDuFilm(t *testing.T, depot string) []manifesteChunk {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(depot, "data", "cache", "film_manifests", filmTemoin+".json"))
	if err != nil {
		t.Skipf("manifeste du film témoin absent: %v", err)
	}
	var mf struct {
		Chunks []manifesteChunk `json:"chunks"`
	}
	if err := json.Unmarshal(raw, &mf); err != nil {
		t.Fatalf("manifeste illisible: %v", err)
	}
	if len(mf.Chunks) == 0 {
		t.Skip("manifeste sans morceau")
	}
	return mf.Chunks
}

// serveurDeMorceaux imite le CDN Azure : il sert les morceaux du cache COMPRESSÉS
// en zlib, sans authentification — c'est exactement ce que l'ouvrier attend d'une
// URL pré-signée, et ça vérifie au passage sa décompression.
func serveurDeMorceaux(t *testing.T, depot string) *httptest.Server {
	t.Helper()
	dir := filepath.Join(depot, "data", "cache", "film_chunks", filmTemoin)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nom := filepath.Base(r.URL.Path)
		brut, err := os.ReadFile(filepath.Join(dir, nom))
		if err != nil {
			http.Error(w, "morceau absent", http.StatusNotFound)
			return
		}
		var buf bytes.Buffer
		zw := zlib.NewWriter(&buf)
		if _, err := zw.Write(brut); err != nil {
			http.Error(w, "compression", http.StatusInternalServerError)
			return
		}
		_ = zw.Close()
		_, _ = w.Write(buf.Bytes())
	}))
	t.Cleanup(srv.Close)
	return srv
}

// enqueueTravailReel met en file le job du film témoin, travail RÉSOLU compris —
// c'est-à-dire ce que ferait EnqueueReplayBuild après avoir lu le manifeste avec
// les tokens (le seul geste qu'on remplace ici, faute de vouloir un appel Halo).
func enqueueTravailReel(t *testing.T, reg *ServiceRegistry, cdnURL string, chunks []manifesteChunk) domain.BuildQueueJob {
	t.Helper()
	payload := &domain.BuildQueuePayload{
		MatchID: filmTemoin, ShortID: filmTemoin, TitleSlug: titlePkg.DefaultSlug,
		MapNames: []string{carteTemoin},
	}
	for _, c := range chunks {
		payload.Chunks = append(payload.Chunks, domain.BuildQueueChunk{
			Index: c.Index, ChunkType: c.ChunkType, StartMS: c.StartMS, DurationMS: c.DurationMS,
			URL: fmt.Sprintf("%s/%s", cdnURL, nomDeMorceau(c.Index)),
		})
	}
	job, created, err := reg.monitoringStore.EnqueueBuildJob(context.Background(), ops.EnqueueBuildJobRequest{
		JobType:   string(domain.JobTypeReplayBuild),
		TitleSlug: titlePkg.DefaultSlug,
		MatchID:   filmTemoin,
		Payload:   payload,
	})
	if err != nil || !created {
		t.Fatalf("mise en file: err=%v created=%v", err, created)
	}
	return job
}

// nomDeMorceau reproduit la convention de nommage du cache film.
func nomDeMorceau(index int) string { return fmt.Sprintf("chunk_%02d.bin", index) }

// depotOuvrier fabrique un dépôt À LUI : uniquement les références que
// replaybuild charge (bornes de carte, libellés, géométrie, structures). Copie et
// non lien : l'ouvrier ne doit pas pouvoir toucher au dépôt de l'utilisateur.
func depotOuvrier(t *testing.T, depot string) string {
	t.Helper()
	dst := t.TempDir()
	for _, rel := range []string{
		filepath.Join("data", "titles", titlePkg.DefaultSlug, "reference", "map_quant_bounds.json"),
		filepath.Join("data", "titles", titlePkg.DefaultSlug, "reference", "map_geometry"),
		filepath.Join("data", "titles", titlePkg.DefaultSlug, "reference", "map_structure"),
		filepath.Join("config", "titles", titlePkg.DefaultSlug, "mappings"),
	} {
		if err := copierArborescence(filepath.Join(depot, rel), filepath.Join(dst, rel)); err != nil {
			t.Skipf("référence %s indisponible pour l'ouvrier: %v", rel, err)
		}
	}
	return dst
}

// copierArborescence copie un fichier ou un répertoire (récursif).
func copierArborescence(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		blob, rerr := os.ReadFile(src)
		if rerr != nil {
			return rerr
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dst, blob, 0o644)
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	for _, e := range entries {
		if err := copierArborescence(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

// compilerOuvrier construit le binaire de l'ouvrier. Compilé AVANT tout décodage
// (jamais pendant : deux travaux lourds en parallèle sur la même machine, c'est
// la leçon des gels de poste).
func compilerOuvrier(t *testing.T) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "replay-worker")
	if runtime.GOOS == "windows" {
		out += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", out, "./cmd/replay-worker")
	cmd.Dir = racineGoAPI(t)
	if sortie, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compilation de l'ouvrier: %v\n%s", err, sortie)
	}
	return out
}

// racineGoAPI rend apps/go-api (le module Go).
func racineGoAPI(t *testing.T) string {
	t.Helper()
	_, ici, _, _ := runtime.Caller(0)
	root, err := filepath.Abs(filepath.Join(filepath.Dir(ici), "..", "..", ".."))
	if err != nil {
		t.Fatalf("racine du module: %v", err)
	}
	return root
}

// lancerOuvrier exécute le binaire en mode --once : il prend UN job, le traite, et
// sort. C'est le mode de la preuve — et celui d'un test manuel.
func lancerOuvrier(t *testing.T, binaire, url, repo, travail string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, binaire,
		"--once",
		"--url", url,
		"--token", transportToken,
		"--id", "ouvrier-preuve",
		"--repo", repo,
		"--work", travail)
	sortie, err := cmd.CombinedOutput()
	t.Logf("journal de l'ouvrier :\n%s", sortie)
	if err != nil {
		t.Fatalf("l'ouvrier s'est arrêté sur erreur: %v", err)
	}
}
