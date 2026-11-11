package filmarchive

// reprise_test.go — les arrêts au milieu d'un geste de l'utilisateur (revue adversariale de L14) :
// recompression interrompue entre la conversion d'un chunk et la correction du manifeste, déplacement
// vers l'emplacement par défaut, reprise réelle, erreur d'écriture au milieu d'un film.

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/filmcache"
)

// TestRecompresser_ReprendApresUneInterruptionAvantLeManifeste : le chunk zlib a été converti, le
// manifeste porte encore la taille du fichier zlib. Le film est « tronqué » pour tout lecteur ;
// relancer la commande doit le réparer, alors que le chunk est déjà vu comme zstd.
func TestRecompresser_ReprendApresUneInterruptionAvantLeManifeste(t *testing.T) {
	root := t.TempDir()
	film := ecrireCacheHistorique(t, root, "cafe0020", 1)
	z, err := filmcache.CompresserZstd(film[1].Data)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filmcache.CheminDuChunk(filmcache.ChunkDir(root, "cafe0020"), 1), z, 0o644); err != nil {
		t.Fatal(err)
	}
	var tronque *filmcache.ErrChunkTronque
	if _, err := filmcache.LireChunk(root, "cafe0020", 1); !errors.As(err, &tronque) {
		t.Fatalf("avant la reprise : %v, attendu ErrChunkTronque (manifeste périmé)", err)
	}
	rap, err := Recompresser(t.Context(), root, "cafe0020", false)
	if err != nil || !rap.ManifesteCorrige || rap.Convertis != 2 || rap.DejaZstd != 1 {
		t.Fatalf("reprise = %+v, %v ; attendu le manifeste corrigé, 2 convertis, 1 déjà zstd", rap, err)
	}
	if ok, err := filmcache.FilmComplet(root, "cafe0020"); err != nil || !ok {
		t.Fatalf("FilmComplet après la reprise = %v, %v", ok, err)
	}
	if lu, err := filmcache.LireChunk(root, "cafe0020", 1); err != nil || !bytes.Equal(lu, film[1].Data) {
		t.Fatalf("chunk repris illisible : %v", err)
	}
}

// TestRecompresser_LaReprisePasUneTrameTronquee : un chunk zstd réellement tronqué ne fait PAS
// corriger le manifeste (la taille de l'en-tête n'est crue qu'une fois la trame décodée).
func TestRecompresser_LaReprisePasUneTrameTronquee(t *testing.T) {
	root := t.TempDir()
	film := ecrireCacheHistorique(t, root, "cafe0021", -1)
	z, _ := filmcache.CompresserZstd(grosChunk(5, 60000))
	chemin := filmcache.CheminDuChunk(filmcache.ChunkDir(root, "cafe0021"), 1)
	if err := os.WriteFile(chemin, z[:len(z)-8], 0o644); err != nil {
		t.Fatal(err)
	}
	avant, _ := os.ReadFile(filmcache.ManifestPath(root, "cafe0021"))
	if _, err := Recompresser(t.Context(), root, "cafe0021", false); err == nil {
		t.Fatal("une trame tronquée doit être signalée")
	}
	apres, _ := os.ReadFile(filmcache.ManifestPath(root, "cafe0021"))
	if !bytes.Equal(avant, apres) {
		t.Fatal("le manifeste a été corrigé à partir d'une trame tronquée")
	}
	_ = film
}

// TestRecompresser_ErreurSurUnChunkSuivantLaisseLesPrecedentsCoherents : le manifeste suit chaque
// chunk converti ; une erreur plus loin ne laisse pas le chunk déjà converti derrière une taille
// périmée.
func TestRecompresser_ErreurSurUnChunkSuivantLaisseLesPrecedentsCoherents(t *testing.T) {
	root := t.TempDir()
	film := ecrireCacheHistorique(t, root, "cafe0022", 0)
	z := zlibDe(t, film[2].Data)
	// Le chunk 2 devient un flux zlib abîmé (en-tête bon, corps coupé) : la passe s'y arrête.
	if err := os.WriteFile(filmcache.CheminDuChunk(filmcache.ChunkDir(root, "cafe0022"), 2), z[:len(z)/2], 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Recompresser(t.Context(), root, "cafe0022", false); err == nil {
		t.Fatal("attendu une erreur sur le flux zlib abîmé")
	}
	if lu, err := filmcache.LireChunk(root, "cafe0022", 0); err != nil || !bytes.Equal(lu, film[0].Data) {
		t.Fatalf("le chunk 0, converti avant l'erreur, est illisible : %v", err)
	}
}

func TestValiderCible_ParDefautAdmetUnCacheSansFilm(t *testing.T) {
	cache := t.TempDir()
	if err := os.WriteFile(filepath.Join(cache, "jobs.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(cache, "film_facts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := ValiderCible(cache, true); err != nil {
		t.Fatalf("cache sans film refusé par défaut : %v", err)
	}
	if err := ValiderCible(cache, false); !errors.Is(err, ErrCibleNonVide) {
		t.Fatalf("le même dossier, explicite, doit rester refusé : %v", err)
	}
	if err := Write(t.Context(), cache, "dddd0001", filmCompressible()); err != nil {
		t.Fatal(err)
	}
	if err := ValiderCible(cache, true); !errors.Is(err, ErrCibleNonVide) {
		t.Fatalf("cache avec un film : %v, attendu ErrCibleNonVide", err)
	}
}

func TestDeplacer_VersLEmplacementParDefaut(t *testing.T) {
	origine, cache := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(cache, "jobs.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Write(t.Context(), origine, "dddd0002", filmCompressible()); err != nil {
		t.Fatal(err)
	}
	n, err := Deplacer(t.Context(), origine, cache, OptionsDeplacement{CibleParDefaut: true})
	if err != nil || n != 1 {
		t.Fatalf("Deplacer vers le défaut = %d, %v", n, err)
	}
	if ok, err := filmcache.FilmComplet(cache, "dddd0002"); err != nil || !ok {
		t.Fatalf("film absent à l'emplacement par défaut : %v, %v", ok, err)
	}
	if existe(filepath.Join(cache, marqueurDeplacement)) {
		t.Fatal("le marqueur de déplacement n'a pas été retiré")
	}
}

// TestDeplacer_ResumeNeSauteLeControleQueSiUnDeplacementEstEnCours : sans marqueur, --resume n'est
// pas une dérogation ; avec le marqueur de la même origine, la cible déjà remplie est admise.
func TestDeplacer_ResumeNeSauteLeControleQueSiUnDeplacementEstEnCours(t *testing.T) {
	origine, cible := t.TempDir(), t.TempDir()
	if err := Write(t.Context(), origine, "dddd0003", filmCompressible()); err != nil {
		t.Fatal(err)
	}
	if err := Write(t.Context(), cible, "dddd0004", filmCompressible()); err != nil {
		t.Fatal(err)
	}
	if _, err := Deplacer(t.Context(), origine, cible, OptionsDeplacement{Reprise: true}); !errors.Is(err, ErrCibleNonVide) {
		t.Fatalf("--resume sans déplacement en cours : %v, attendu ErrCibleNonVide", err)
	}
	if ids, _ := filmcache.ListShortIDs(origine); len(ids) != 1 {
		t.Fatal("un --resume refusé a touché à l'origine")
	}
	// Un déplacement réellement interrompu : le marqueur de CETTE origine est là.
	if err := os.WriteFile(filepath.Join(cible, marqueurDeplacement), []byte(filepath.Clean(origine)), 0o644); err != nil {
		t.Fatal(err)
	}
	if n, err := Deplacer(t.Context(), origine, cible, OptionsDeplacement{Reprise: true}); err != nil || n != 1 {
		t.Fatalf("reprise réelle = %d, %v", n, err)
	}
	// Le marqueur d'une AUTRE origine ne vaut pas reprise.
	autre := t.TempDir()
	if err := os.WriteFile(filepath.Join(cible, marqueurDeplacement), []byte(filepath.Clean(autre)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Write(t.Context(), origine, "dddd0005", filmCompressible()); err != nil {
		t.Fatal(err)
	}
	if _, err := Deplacer(t.Context(), origine, cible, OptionsDeplacement{Reprise: true}); !errors.Is(err, ErrCibleNonVide) {
		t.Fatalf("marqueur d'une autre origine : %v, attendu ErrCibleNonVide", err)
	}
}

// volumesDistincts force la copie (le renommage de dossier échoue, comme d'un volume à l'autre).
func volumesDistincts(t *testing.T) {
	t.Helper()
	avant := renommerDossier
	renommerDossier = func(string, string) error { return errors.New("volumes distincts") }
	t.Cleanup(func() { renommerDossier = avant })
}

// TestDeplacer_ErreurAuMilieuDUnFilmLaisseLOrigineIntacte : une erreur d'écriture au deuxième chunk
// (disque cible plein) ne laisse aucune trace à la cible ; l'origine reste lisible en entier.
func TestDeplacer_ErreurAuMilieuDUnFilmLaisseLOrigineIntacte(t *testing.T) {
	volumesDistincts(t)
	origine, cible := t.TempDir(), filepath.Join(t.TempDir(), "archive")
	film := filmCompressible()
	if err := Write(t.Context(), origine, "dddd0006", film); err != nil {
		t.Fatal(err)
	}
	appels := 0
	avant := copieDeFichier
	copieDeFichier = func(de, vers string) error {
		appels++
		if appels == 2 {
			return errors.New("disque plein")
		}
		return avant(de, vers)
	}
	t.Cleanup(func() { copieDeFichier = avant })

	if _, err := Deplacer(t.Context(), origine, cible, OptionsDeplacement{}); err == nil {
		t.Fatal("attendu l'erreur d'écriture")
	}
	if ok, err := filmcache.FilmComplet(origine, "dddd0006"); err != nil || !ok {
		t.Fatalf("origine amputée : %v, %v", ok, err)
	}
	for i, c := range film {
		if lu, err := filmcache.LireChunk(origine, "dddd0006", c.Index); err != nil || !bytes.Equal(lu, film[i].Data) {
			t.Fatalf("chunk %d de l'origine altéré : %v", c.Index, err)
		}
	}
	if ids, _ := filmcache.ListShortIDs(cible); len(ids) != 0 {
		t.Fatalf("film visible à la cible : %v", ids)
	}
	if err := aucunFilm(cible); err != nil {
		t.Fatalf("trace laissée à la cible : %v", err)
	}
	entrees, _ := os.ReadDir(filmcache.ChunksRoot(cible))
	if len(entrees) != 0 {
		t.Fatalf("temporaire laissé à la cible : %v", entrees)
	}
}

// TestDeplacer_InterruptionEntreLesChunksLaisseLOrigineIntacte : le contexte est contrôlé entre
// les chunks.
func TestDeplacer_InterruptionEntreLesChunksLaisseLOrigineIntacte(t *testing.T) {
	volumesDistincts(t)
	origine, cible := t.TempDir(), filepath.Join(t.TempDir(), "archive")
	if err := Write(t.Context(), origine, "dddd0007", filmCompressible()); err != nil {
		t.Fatal(err)
	}
	ctx, annuler := context.WithCancel(t.Context())
	avant := copieDeFichier
	copieDeFichier = func(de, vers string) error {
		err := avant(de, vers)
		annuler() // annulé pendant le premier chunk
		return err
	}
	t.Cleanup(func() { copieDeFichier = avant })
	if _, err := Deplacer(ctx, origine, cible, OptionsDeplacement{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("erreur = %v, attendu context.Canceled", err)
	}
	if ok, err := filmcache.FilmComplet(origine, "dddd0007"); err != nil || !ok {
		t.Fatalf("origine amputée : %v, %v", ok, err)
	}
	if entrees, _ := os.ReadDir(filmcache.ChunksRoot(cible)); len(entrees) != 0 {
		t.Fatalf("temporaire laissé à la cible : %v", entrees)
	}
}

// TestDeplacer_CopieEntiereEntreVolumes : le chemin par copie range le film entier et retire l'origine.
func TestDeplacer_CopieEntiereEntreVolumes(t *testing.T) {
	volumesDistincts(t)
	origine, cible := t.TempDir(), filepath.Join(t.TempDir(), "archive")
	film := filmCompressible()
	if err := Write(t.Context(), origine, "dddd0008", film); err != nil {
		t.Fatal(err)
	}
	if n, err := Deplacer(t.Context(), origine, cible, OptionsDeplacement{}); err != nil || n != 1 {
		t.Fatalf("Deplacer = %d, %v", n, err)
	}
	if lu, err := filmcache.LireChunk(cible, "dddd0008", 1); err != nil || !bytes.Equal(lu, film[1].Data) {
		t.Fatalf("chunk copié illisible : %v", err)
	}
	if ids, _ := filmcache.ListShortIDs(origine); len(ids) != 0 {
		t.Fatalf("films restés à l'origine : %v", ids)
	}
}
