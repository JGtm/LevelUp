package filmcache

// recompress.go — LA RECOMPRESSION D'UN FILM DU CACHE EN ZSTD-3 (decision D-18, 2026-11-01).
//
// LANCEE PAR L'UTILISATEUR, JAMAIS PAR LE PROGRAMME : `levelup film recompress` appelle
// [Recompresser] film apres film. Aucune tache de fond, aucun cron, aucun declencheur au demarrage
// n'en est un appelant (garde-rail `archlint/no_auto_film_recompress_test.go`).
//
// UN FILM A LA FOIS, UN CHUNK A LA FOIS, ET RIEN N'EST REMPLACE SANS PREUVE. Le chunk est lu,
// ramene a ses octets logiques (zlib decompresse en mode STRICT : un flux abime n'est pas
// « recompresse », il est signale), compresse, puis RELU en memoire et compare a l'original avant
// que le fichier ne soit remplace — atomiquement. Un chunk deja zstd est saute : relancer la
// commande apres une interruption reprend la ou elle s'etait arretee.
//
// LE MANIFESTE N'EST TOUCHE QUE POUR UN CHUNK ZLIB : `size_bytes` y valait la taille du fichier
// zlib et vaut apres la taille des octets decompresses. Pour un chunk brut, la taille logique ne
// bouge pas et le manifeste reste tel quel.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/JGtm/LevelUp/libs/film/internal/atomicfile"
	"github.com/JGtm/LevelUp/libs/film/internal/source"
)

// RapportRecompression est le bilan de la recompression d'UN film (ou, cumule par [Ajouter], d'un
// lot).
type RapportRecompression struct {
	Chunks         int   // entrees du manifeste
	Absents        int   // chunks declares au manifeste mais absents du disque
	DejaZstd       int   // chunks deja en zstd, sautes
	Convertis      int   // chunks recompresses (ou qui le seraient, en simulation)
	SansGain       int   // chunks que zstd ne raccourcit pas, laisses tels quels
	OctetsAvant    int64 // taille disque des chunks convertis, avant
	OctetsApres    int64 // taille disque des chunks convertis, apres
	OctetsDejaZstd int64 // taille disque des chunks deja zstd
	OctetsSansGain int64 // taille disque des chunks sans gain
	// ManifesteCorrige : `size_bytes` a ete reecrit pour au moins un chunk zlib.
	ManifesteCorrige bool
}

// Gain rend le nombre d'octets gagnes sur les chunks convertis.
func (r RapportRecompression) Gain() int64 { return r.OctetsAvant - r.OctetsApres }

// Ajouter cumule le bilan d'un film dans celui d'un lot.
func (r *RapportRecompression) Ajouter(autre RapportRecompression) {
	r.Chunks += autre.Chunks
	r.Absents += autre.Absents
	r.DejaZstd += autre.DejaZstd
	r.Convertis += autre.Convertis
	r.SansGain += autre.SansGain
	r.OctetsAvant += autre.OctetsAvant
	r.OctetsApres += autre.OctetsApres
	r.OctetsDejaZstd += autre.OctetsDejaZstd
	r.OctetsSansGain += autre.OctetsSansGain
	r.ManifesteCorrige = r.ManifesteCorrige || autre.ManifesteCorrige
}

// ErrFilmAbsent : le manifeste du film n'est pas au cache.
var ErrFilmAbsent = errors.New("filmcache: film absent du cache")

// Recompresser passe en zstd-3 les chunks bruts et zlib du film `shortID`. En simulation
// (`simulation` vrai) rien n'est ecrit : les chunks sont lus et compresses en memoire, le bilan
// chiffre le gain. S'arrete au premier chunk en erreur (le bilan rendu couvre ce qui precede).
func Recompresser(ctx context.Context, root, shortID string, simulation bool) (RapportRecompression, error) {
	var rap RapportRecompression
	src, ok, err := Open(root, shortID)
	if err != nil {
		return rap, err
	}
	if !ok {
		return rap, fmt.Errorf("%w : %s", ErrFilmAbsent, shortID)
	}
	dir := ChunkDir(root, shortID)
	fichiers, err := numerosDeChunks(dir)
	if err != nil {
		return rap, err
	}
	rap.Chunks = len(fichiers)
	// Le manifeste liste ce que le serveur a ; le dossier porte ce qui est descendu, et c'est
	// LE DOSSIER que le décodeur lit ([source.LoadDir]). Un chunk présent hors manifeste se
	// recompresse comme les autres, sinon une part de l'archive resterait en clair.
	declare := map[int]int64{}
	nouvellesTailles := map[int]int64{}
	for i, c := range src.chunks {
		declare[c.Index] = src.tailles[i]
	}
	for _, numero := range fichiers {
		if err := ctx.Err(); err != nil {
			return rap, err
		}
		logique, err := recompresserChunk(CheminDuChunk(dir, numero), simulation, &rap)
		if err != nil {
			return rap, fmt.Errorf("filmcache: chunk %d de %s : %w", numero, shortID, err)
		}
		if taille := declare[numero]; logique > 0 && taille > 0 && taille != logique {
			nouvellesTailles[numero] = logique
		}
	}
	rap.Absents = len(src.chunks) - nombreDeclaresPresents(src, fichiers)
	if len(nouvellesTailles) > 0 && !simulation {
		if err := corrigerTaillesDuManifeste(ManifestPath(root, shortID), nouvellesTailles); err != nil {
			return rap, fmt.Errorf("filmcache: manifeste de %s : %w", shortID, err)
		}
		rap.ManifesteCorrige = true
	}
	return rap, nil
}

// numerosDeChunks rend les numéros des fichiers `chunk_NN.bin` du dossier d'un film, triés. Un
// dossier absent rend une liste vide : le film n'a alors aucun chunk à traiter.
func numerosDeChunks(dir string) ([]int, error) {
	entrees, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("filmcache: lecture de %s : %w", dir, err)
	}
	var numeros []int
	for _, e := range entrees {
		if n, ok := numeroDuFichier(e.Name()); ok && !e.IsDir() {
			numeros = append(numeros, n)
		}
	}
	sort.Ints(numeros)
	return numeros, nil
}

// nombreDeclaresPresents compte les entrées du manifeste dont le fichier existe.
func nombreDeclaresPresents(src *Source, fichiers []int) int {
	presents := make(map[int]bool, len(fichiers))
	for _, n := range fichiers {
		presents[n] = true
	}
	n := 0
	for _, c := range src.chunks {
		if presents[c.Index] {
			n++
		}
	}
	return n
}

// recompresserChunk traite UN fichier de chunk, cumule son bilan dans `rap` et rend la taille
// logique du chunk quand il a ete converti (0 sinon).
func recompresserChunk(path string, simulation bool, rap *RapportRecompression) (int64, error) {
	fichier, err := os.ReadFile(path) //nolint:gosec // chemin compose par CheminDuChunk
	if errors.Is(err, os.ErrNotExist) {
		rap.Absents++
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if source.EstZstd(fichier) {
		rap.DejaZstd++
		rap.OctetsDejaZstd += int64(len(fichier))
		return 0, nil
	}
	logique, err := octetsLogiquesStrict(fichier)
	if err != nil {
		return 0, err
	}
	z, err := source.CompresserZstd(logique)
	if err != nil {
		return 0, err
	}
	if len(z) == 0 || len(z) >= len(fichier) {
		rap.SansGain++
		rap.OctetsSansGain += int64(len(fichier))
		return 0, nil
	}
	relu, err := source.DecompresserZstd(z)
	if err != nil || !bytes.Equal(relu, logique) {
		return 0, fmt.Errorf("la relecture du chunk compresse differe de l'original (%v)", err)
	}
	if !simulation {
		if err := atomicfile.WriteFileStrict(path, z, 0o644); err != nil {
			return 0, err
		}
	}
	rap.Convertis++
	rap.OctetsAvant += int64(len(fichier))
	rap.OctetsApres += int64(len(z))
	return int64(len(logique)), nil
}

// octetsLogiquesStrict rend les octets d'un chunk NON zstd : decompresses s'il est en zlib, tels
// quels sinon. Un flux zlib dont l'en-tete est bon mais qui casse est une ERREUR : le recompresser
// figerait un partiel.
func octetsLogiquesStrict(fichier []byte) ([]byte, error) {
	if len(fichier) < 2 || fichier[0] != 0x78 {
		return fichier, nil
	}
	out, err := source.Decompresser(fichier)
	if errors.Is(err, source.ErrEnTeteZlib) {
		return fichier, nil // 0x78 par hasard : un chunk clair
	}
	return out, err
}

// corrigerTaillesDuManifeste remplace `size_bytes` des entrees dont le numero est dans `tailles`,
// sans toucher aux autres champs du manifeste (dont un eventuel `blob_prefix` CDN).
func corrigerTaillesDuManifeste(path string, tailles map[int]int64) error {
	brut, err := os.ReadFile(path) //nolint:gosec // chemin compose par ManifestPath
	if err != nil {
		return err
	}
	var mf map[string]json.RawMessage
	if err := json.Unmarshal(brut, &mf); err != nil {
		return fmt.Errorf("manifeste illisible : %w", err)
	}
	var entrees []map[string]json.RawMessage
	if err := json.Unmarshal(mf["chunks"], &entrees); err != nil {
		return fmt.Errorf("liste de chunks illisible : %w", err)
	}
	for _, e := range entrees {
		var index int
		if err := json.Unmarshal(e["index"], &index); err != nil {
			return fmt.Errorf("index de chunk illisible : %w", err)
		}
		if t, ok := tailles[index]; ok {
			e["size_bytes"] = json.RawMessage(fmt.Sprint(t))
		}
	}
	if mf["chunks"], err = json.Marshal(entrees); err != nil {
		return err
	}
	out, err := json.Marshal(mf)
	if err != nil {
		return err
	}
	return atomicfile.WriteFileStrict(path, out, 0o644)
}
