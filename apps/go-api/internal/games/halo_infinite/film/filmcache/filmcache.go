// Package filmcache — LE cache film sur disque, lu par un seul morceau de code.
//
// Disposition, tenue par les repertoires deja presents :
//
//	<racine>/film_manifests/<short8>.json      le manifeste (index, type et debut des chunks)
//	<racine>/film_chunks/<short8>/chunk_NN.bin les chunks BRUTS
//
// `short8` est la forme courte du match_id (title.FilmShortMatchID) : c'est la cle du
// cache, pas l'uuid complet que manipule le reste de l'application.
//
// # Pourquoi ce paquet existe
//
// La meme source disque etait ecrite une fois dans `cmd/diag_weapons_v3` et une fois dans
// les tests d'`objectiveevents`. Un troisieme outil en avait besoin : a la troisieme
// copie, la regle du depot impose de centraliser ET de poser un garde-rail. Une disposition
// de cache dupliquee derive en silence — le jour ou le nom des chunks change, deux lecteurs
// sur trois cessent de trouver le film et se contentent de rendre « rien a decoder ».
//
// LE GARDE-RAIL EST DEVENU UNE ASSERTION DE COMPILATION (2026-09-02, item 1.5 de
// PLAN_CUISSON_PERF) : `var _ filmsource.Source = (*Source)(nil)` plus bas. L'ancien
// `filmcache_guard_test.go` cherchait par expression reguliere les implementations d'une
// interface `objectiveevents.FilmSource` qui n'existe plus, et son allowlist etait justifiee
// par un cycle d'import (`filmcache` -> `objectiveevents`) que ce lot a supprime : les trois
// entrees etaient donc caduques d'un coup. La forme d'une source de film est desormais celle
// du paquet FEUILLE `analysis/filmsource`, que tout le monde peut importer sans cycle.
//
// # Ce paquet ne decode rien
//
// Il rend des octets, un index et — par [LoadFilm] — un film DEJA CHARGE (`filmsource`, une
// decompression par film). Le decodage, lui, vit dans `analysis/objectiveevents` et
// `analysis/filmdec`.
package filmcache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"levelup/go-api/internal/analysis/filmsource"
)

// manifestsDir et chunksDir sont les deux sous-dossiers du cache. Nommes ici, et nulle
// part ailleurs.
const (
	manifestsDir = "film_manifests"
	chunksDir    = "film_chunks"
)

// Source est le MANIFESTE d'un film du cache, et l'acces aux octets bruts de ses chunks.
//
// L'INDICE D'UN CHUNK EST SA POSITION DANS LE MANIFESTE, jamais le numero de son fichier :
// c'est le contrat de [filmsource.Source], et [Meta] donne les deux (`Meta()[i].Index` porte
// le numero). Confondre les deux marcherait un chunk de donnees comme un registre.
type Source struct {
	root   string
	short  string
	chunks []filmsource.ChunkMeta
}

// Source implemente la source de film canonique du depot. C'est ce que verifie cette ligne,
// et elle remplace a elle seule l'ancien garde-rail par expression reguliere (cf. l'en-tete).
var _ filmsource.Source = (*Source)(nil)

type manifestJSON struct {
	Chunks []struct {
		Index     int `json:"index"`
		ChunkType int `json:"chunk_type"`
		StartMS   int `json:"start_ms"`
	} `json:"chunks"`
}

// Open charge le manifeste d'un film.
//
// (nil, false, nil) quand le manifeste n'est pas la : un film absent du cache est le cas
// NOMINAL (le cache est partiel et local), pas une panne. Une erreur non nulle signale un
// manifeste present mais illisible — celle-la, il faut la voir.
func Open(root, shortID string) (*Source, bool, error) {
	path := ManifestPath(root, shortID)
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("manifeste de film illisible (%s) : %w", path, err)
	}
	var mf manifestJSON
	if err := json.Unmarshal(raw, &mf); err != nil {
		return nil, false, fmt.Errorf("manifeste de film invalide (%s) : %w", path, err)
	}
	src := &Source{root: root, short: shortID, chunks: make([]filmsource.ChunkMeta, 0, len(mf.Chunks))}
	for _, c := range mf.Chunks {
		src.chunks = append(src.chunks, filmsource.ChunkMeta{
			Index: c.Index, ChunkType: c.ChunkType, StartMS: c.StartMS,
		})
	}
	return src, true, nil
}

// Meta rend l'index du manifeste, POSITIONNEL : `Meta()[i]` decrit le chunk d'indice `i`, et
// porte son numero de fichier en [filmsource.ChunkMeta.Index]. C'est la forme qu'attend
// [filmsource.Load].
func (s *Source) Meta() []filmsource.ChunkMeta { return s.chunks }

// NumChunks implemente [filmsource.Source] : le nombre d'entrees du manifeste.
func (s *Source) NumChunks() int { return len(s.chunks) }

// Chunk implemente [filmsource.Source] : les octets BRUTS (compresses) du chunk d'INDICE `i`,
// lu au fichier que le manifeste lui donne. Un chunk manquant au cache est une ERREUR ici —
// l'appelant qui veut la degradation gracieuse d'un cache partiel passe par [LoadFilm], qui
// charge les FICHIERS PRESENTS et non les entrees du manifeste.
func (s *Source) Chunk(i int) ([]byte, error) {
	if i < 0 || i >= len(s.chunks) {
		return nil, fmt.Errorf("filmcache: chunk %d hors bornes (%d au manifeste de %s)", i, len(s.chunks), s.short)
	}
	path := filepath.Join(ChunkDir(s.root, s.short), chunkName(s.chunks[i].Index))
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("filmcache: chunk %d de %s (%s) : %w", s.chunks[i].Index, s.short, path, err)
	}
	return raw, nil
}

// LoadFilm charge le film complet d'un short8 du cache : manifeste puis chunks, decompresses
// et decoupes en paquets UNE fois ([filmsource.LoadDir]).
//
// C'EST LE MEME CHEMIN QUE LA CUISSON (`replaybuild.BuildBytes`), et il le reste
// deliberement : les NUMEROS de chunk viennent des fichiers presents, le manifeste ne fournit
// que le type et le debut de chacun, fusionnes PAR NUMERO. Un cache partiel rend donc un film
// ampute plutot qu'une erreur, exactement comme avant ce lot, ou chaque chunk absent etait
// saute en silence.
//
// (nil, false, nil) quand le manifeste n'est pas la — meme contrat qu'[Open].
func LoadFilm(root, shortID string) (*filmsource.Film, bool, error) {
	src, ok, err := Open(root, shortID)
	if err != nil || !ok {
		return nil, ok, err
	}
	film, err := filmsource.LoadDir(ChunkDir(root, shortID), src.Meta())
	if err != nil {
		return nil, true, fmt.Errorf("filmcache: chargement du film %s : %w", shortID, err)
	}
	return film, true, nil
}

// LoadFilmDir est [LoadFilm] pour l'appelant qui connait le REPERTOIRE DE CHUNKS et non le
// couple (racine, short8) — la meme porte qu'[OpenChunkDir], et pour la meme raison : les
// balayages hors ligne recoivent un chemin de chunks.
func LoadFilmDir(chunkDir string) (*filmsource.Film, bool, error) {
	cleaned := filepath.Clean(chunkDir)
	return LoadFilm(filepath.Dir(filepath.Dir(cleaned)), filepath.Base(cleaned))
}

// ChunkDir rend le repertoire des chunks d'un film. C'est ce chemin qu'attendent les
// balayages hors ligne de `analysis/filmdec` et `analysis/replay`, qui lisent le dossier
// directement plutot que par l'interface.
func ChunkDir(root, shortID string) string {
	return filepath.Join(root, chunksDir, shortID)
}

// ChunksRoot rend le repertoire qui contient UN dossier par film cache. Sert a enumerer
// ce qui est disponible localement.
func ChunksRoot(root string) string { return filepath.Join(root, chunksDir) }

// ManifestPath rend le chemin du manifeste d'un film.
func ManifestPath(root, shortID string) string {
	return filepath.Join(root, manifestsDir, shortID+".json")
}

// ListShortIDs enumere les films du cache par leurs MANIFESTES (le manifeste est la piece
// d'entree du cache : un dossier de chunks sans manifeste n'est pas lisible par Open).
// Un cache absent rend une liste vide, pas une erreur : le cache est local et partiel.
func ListShortIDs(root string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(root, manifestsDir))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("enumeration du cache film (%s) : %w", root, err)
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || filepath.Ext(name) != ".json" {
			continue
		}
		out = append(out, name[:len(name)-len(".json")])
	}
	return out, nil
}

func chunkName(index int) string { return fmt.Sprintf("chunk_%02d.bin", index) }

// OpenChunkDir ouvre le film dont on connait le REPERTOIRE DE CHUNKS.
//
// POURQUOI CETTE PORTE EXISTE. Le manifeste et les chunks vivent dans deux sous-dossiers
// FRERES (cf. l'en-tete de ce paquet) ; les balayages hors ligne, eux, recoivent un chemin de
// chunks parce que `analysis/filmdec` et `analysis/replay` lisent le dossier directement. Le
// constructeur d'artefact a besoin des deux — les chunks pour le decodage, le manifeste pour
// le `start_ms` de chaque chunk, sans lequel les enregistrements d'entite ne sont pas datables.
//
// La remontee du chemin est faite ICI et nulle part ailleurs : la disposition du cache est
// declaree dans ce paquet, et un appelant qui la reconstituerait chez lui serait exactement la
// copie que le garde-rail interdit.
func OpenChunkDir(chunkDir string) (*Source, bool, error) {
	cleaned := filepath.Clean(chunkDir)
	root := filepath.Dir(filepath.Dir(cleaned))
	return Open(root, filepath.Base(cleaned))
}
