package replay

// zero_disque_test.go — LE DECODAGE D'UN FILM NE TOUCHE PLUS LE DISQUE (item 1.8, lot 1 de
// PLAN_CUISSON_PERF, 2026-09-02).
//
// # CE QUE CE TEST VERROUILLE
//
// Avant le lot 1, chaque balayage rouvrait le repertoire de chunks pour son propre compte : le
// film entier etait relu et redecompresse une trentaine de fois par artefact. Le lot a fait
// passer tout le monde a un `*filmsource.Film` charge UNE fois. Ce test est la preuve
// STRUCTURELLE de ce contrat : le film arrive en MEMOIRE (`filmsource.Load` sur des
// `MemoryChunks`, jamais `LoadDir`), et le decodage s'execute depuis un REPERTOIRE COURANT VIDE.
// Un balayage qui reouvrirait la bobine par le seul chemin qu'il pourrait connaitre — le chemin
// relatif que les enveloppes `dir` recoivent — ne trouverait rien, et le test le verrait.
//
// # METHODE, ET SES LIMITES, ECRITES
//
// Il n'existe pas, en Go, de crochet portable qui compte les ouvertures de fichiers d'un
// processus. La preuve est donc faite de trois pieces, et d'aucune affirmation :
//
//  1. `t.Chdir` place le test dans un repertoire temporaire VIDE (restaure par le nettoyage du
//     test, avant la suppression du repertoire). Un CONTROLE mesure que la bobine n'y est
//     effectivement plus atteignable par son chemin relatif habituel — sans ce controle, un
//     repertoire courant « vide » ne serait qu'une intention.
//  2. `BuildFromFilm` doit echouer sur l'erreur EXACTE et connue de la mini-bobine (« aucun slot
//     biped », cf. PROVENANCE.txt : les paquets y sont concatenes hors continuite, il n'y a
//     aucune image-cle de bipede). Toute AUTRE erreur — au premier chef une erreur d'ouverture
//     de fichier — fait echouer le test : c'est la signature d'un acces disque residuel.
//  3. Le repertoire courant est verifie VIDE apres coup : rien n'y a ete ecrit.
//
// La limite, dite franchement : un acces disque par chemin ABSOLU (un catalogue resolu depuis la
// racine du depot, par exemple) reussirait sans que ce test le voie. Elle est bornee par
// construction — `BuildFromFilm(matchID, titleSlug, film, opt)` ne recoit AUCUN chemin de film,
// et l'entree de catalogue de la carte lui est FOURNIE (`opt.MapQuant`) — et par le garde-rail
// `archlint/no_film_reread_test.go`, qui interdit `os.*` dans `filmdec` hors allowlist datee.
//
// # POURQUOI DEUX TESTS
//
// `BuildFromFilm` s'arrete a son PREMIER balayage sur cette bobine (les positions), donc le test
// (1) ne couvre que l'entree du pipeline. [TestZeroDisqueBalayagesSupportes] rejoue, depuis le
// meme repertoire vide, les SEPT familles que la mini-bobine supporte (la liste fermee de D4c,
// celle de `TestEquivalenceMiniFilm`) sous leur forme FILM : elles doivent REUSSIR. Un balayage
// qui decode entierement depuis un repertoire vide n'a touche aucun fichier relatif.

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/filmsource"
)

// miniBobineChunks : les NUMEROS de fichier des chunks de la mini-bobine, dans l'ordre. Ce sont
// les Index REELS (1, 2, 3) — la bobine n'a pas de `chunk_00` (registre), et confondre position
// et numero ferait marcher son premier chunk de DONNEES comme un registre.
var miniBobineChunks = []int{1, 2, 3}

// chargerMiniBobineEnMemoire lit les trois chunks et rend le film charge par [filmsource.Load].
//
// A APPELER AVANT TOUT `t.Chdir` : c'est la SEULE lecture disque autorisee par ce fichier, et
// elle est celle du HARNAIS, pas du decodeur.
func chargerMiniBobineEnMemoire(t *testing.T) *filmsource.Film {
	t.Helper()
	chunks := make(filmsource.MemoryChunks, 0, len(miniBobineChunks))
	meta := make([]filmsource.ChunkMeta, 0, len(miniBobineChunks))
	for _, num := range miniBobineChunks {
		path := filepath.Join(MiniFilmDir, fmt.Sprintf("chunk_%02d.bin", num))
		raw, err := os.ReadFile(path) //nolint:gosec // chemin de fixture fige dans le code
		if err != nil {
			t.Fatalf("mini-bobine illisible (%s) : %v", path, err)
		}
		chunks = append(chunks, raw)
		// ChunkType et StartMS restent nuls : ils ne servent qu'a `objectiveevents` (type de
		// chunk, horloge), qui n'est pas sur le chemin de `BuildFromFilm`, et la mini-bobine
		// n'a de toute facon pas de manifeste pour les porter.
		meta = append(meta, filmsource.ChunkMeta{Index: num})
	}
	film, err := filmsource.Load(chunks, meta)
	if err != nil {
		t.Fatalf("chargement en memoire de la mini-bobine : %v", err)
	}
	if got := filmdec.FilmChunkNumbers(film); len(got) != len(miniBobineChunks) {
		t.Fatalf("chunks de donnees vus par le decodeur : %v, attendus %v — les metadonnees "+
			"portent les NUMEROS de fichier, pas les positions", got, miniBobineChunks)
	}
	return film
}

// entrerDansUnRepertoireVide place le test dans un repertoire temporaire vide et MESURE que la
// mini-bobine n'y est plus atteignable par son chemin relatif.
func entrerDansUnRepertoireVide(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
	temoin := filepath.Join(MiniFilmDir, "chunk_01.bin")
	if _, err := os.ReadFile(temoin); !errors.Is(err, fs.ErrNotExist) { //nolint:gosec // controle du harnais
		t.Fatalf("CONTROLE FAUX : %s reste lisible depuis le repertoire courant (%v) — ce test "+
			"ne prouverait rien, puisqu'un balayage pourrait y relire le film", temoin, err)
	}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("lecture du repertoire courant : %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("le repertoire de travail n'est pas vide (%d entrees) : le controle ci-dessus "+
			"ne vaut que dans un repertoire vide", len(entries))
	}
}

// TestZeroDisqueBuildFromFilm : `BuildFromFilm` decode un film charge EN MEMOIRE, depuis un
// repertoire courant vide, et echoue sur la SEULE erreur attendue de la mini-bobine.
func TestZeroDisqueBuildFromFilm(t *testing.T) {
	// Les deux lectures disque du harnais se font AVANT le changement de repertoire : les
	// chunks de la bobine, et l'entree de catalogue de Cliffhanger (chemin relatif au paquet,
	// cf. goldenMapQuant) que la production FOURNIT au lieu de la faire lire au decodeur.
	film := chargerMiniBobineEnMemoire(t)
	entry, err := goldenMapQuant()
	if err != nil {
		t.Fatalf("entree de catalogue de Cliffhanger illisible : %v", err)
	}

	entrerDansUnRepertoireVide(t)

	_, err = BuildFromFilm("minifilm", "halo_infinite", film, Options{MapQuant: &entry})
	attendu := fmt.Sprintf("aucun slot biped (ti=%d) dans les keyframes du film", filmdec.BipedTypeIndex)
	switch {
	case err == nil:
		t.Fatal("BuildFromFilm a rendu un document sur la mini-bobine : elle n'a aucune image-cle " +
			"de bipede (PROVENANCE.txt) — si le decodeur en trouve, c'est ce test qu'il faut relire")
	case err.Error() != attendu:
		t.Fatalf("erreur INATTENDUE : %v\n  attendue : %s\n"+
			"Depuis un repertoire courant VIDE, la seule issue admise est ce refus de decodage. "+
			"Une erreur d'ouverture de fichier signifie qu'un balayage a tente de RELIRE le film "+
			"(ou un catalogue) au lieu d'utiliser le `*filmsource.Film` deja charge : c'est "+
			"exactement ce que le lot 1 de PLAN_CUISSON_PERF a supprime.", err, attendu)
	}

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("relecture du repertoire courant : %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("le decodage a ECRIT dans le repertoire courant (%d entrees) : le chemin de "+
			"cuisson ne produit que le blob rendu a son appelant", len(entries))
	}
}

// TestZeroDisqueBalayagesSupportes : les SEPT familles que la mini-bobine supporte (liste fermee
// de D4c, la meme que TestEquivalenceMiniFilm) decodent entierement depuis un repertoire vide,
// sous leur forme FILM. Leur SUCCES est la preuve : un balayage qui aurait besoin du disque
// echouerait ici.
func TestZeroDisqueBalayagesSupportes(t *testing.T) {
	film := chargerMiniBobineEnMemoire(t)
	entry, err := goldenMapQuant()
	if err != nil {
		t.Fatalf("entree de catalogue de Cliffhanger illisible : %v", err)
	}
	// MEME GESTE QUE LA PRODUCTION (cf. installWorldObjectPrecision) : les largeurs d'axe du
	// chemin world-object sont un global de paquet, installe depuis l'entree de catalogue et
	// restaure ensuite — sans quoi ce test contaminerait le film suivant du meme process.
	prev := filmdec.WorldObjectPrecision
	t.Cleanup(func() { filmdec.WorldObjectPrecision = prev })
	filmdec.SetWorldObjectPrecisionFromLayout(filmdec.I0Layout{AxisW: entry.AxisWidths})
	wr := entry.Range()

	entrerDansUnRepertoireVide(t)

	fire, err := filmdec.ScanFireEvents(film)
	if err != nil {
		t.Fatalf("tirs : %v", err)
	}
	grenades, err := filmdec.ScanGrenadeThrows(film)
	if err != nil {
		t.Fatalf("lancers de grenade : %v", err)
	}
	loadouts, err := filmdec.ScanKeyframeLoadouts(film, loadoutFamilies())
	if err != nil {
		t.Fatalf("armes portees : %v", err)
	}
	inventory, _, err := ScanKeyframeInventory(film, loadoutFamilies(), 0)
	if err != nil {
		t.Fatalf("inventaire d'image-cle : %v", err)
	}
	deaths, err := ScanDeaths(film)
	if err != nil {
		t.Fatalf("morts : %v", err)
	}
	indices, err := ScanPlayerIndices(film, rosterFromDeaths(deaths))
	if err != nil {
		t.Fatalf("indices joueur : %v", err)
	}
	proj, err := filmdec.ScanProjectiles(film, &wr)
	if err != nil {
		t.Fatalf("projectiles : %v", err)
	}
	// Des sorties VIDES se liraient comme un succes : on exige que chaque famille ait
	// reellement decode quelque chose depuis la memoire.
	comptes := map[string]int{
		etapeFire: len(fire), etapeGrenades: len(grenades), etapeLoadouts: len(loadouts),
		etapeInventaire: len(inventory), etapeMorts: len(deaths),
		etapeIndicesJoueur: len(indices.ByXUID), etapeProjectiles: len(proj),
	}
	for nom, n := range comptes {
		if n == 0 {
			t.Fatalf("balayage %q sans aucune sortie depuis le film en memoire : la mini-bobine "+
				"porte cette famille (PROVENANCE.txt) — un decodeur qui rend zero ici lisait "+
				"probablement le disque", nom)
		}
	}
}
