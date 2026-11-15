package filmarchive

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

	"levelup/go-api/internal/platform/atomicfile"

	"github.com/JGtm/halo-infinite-film-decoder/filmcache"
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
	src, ok, err := filmcache.Open(root, shortID)
	if err != nil {
		return rap, err
	}
	if !ok {
		return rap, fmt.Errorf("%w : %s", ErrFilmAbsent, shortID)
	}
	dir := filmcache.ChunkDir(root, shortID)
	fichiers, err := numerosDeChunks(dir)
	if err != nil {
		return rap, err
	}
	rap.Chunks = len(fichiers)
	// Le manifeste liste ce que le serveur a ; le dossier porte ce qui est descendu, et c'est
	// LE DOSSIER que le décodeur lit (la brique lit le dossier). Un chunk présent hors manifeste se
	// recompresse comme les autres, sinon une part de l'archive resterait en clair.
	declare := map[int]int64{}
	for i, c := range src.Meta() {
		declare[c.Index] = src.TailleDeclaree(i)
	}
	manifeste := filmcache.ManifestPath(root, shortID)
	for _, numero := range fichiers {
		if err := ctx.Err(); err != nil {
			return rap, err
		}
		logique, err := recompresserChunk(filmcache.CheminDuChunk(dir, numero), declare[numero], simulation, &rap)
		if err != nil {
			return rap, fmt.Errorf("filmcache: chunk %d de %s : %w", numero, shortID, err)
		}
		// LE MANIFESTE SUIT CHAQUE CHUNK, pas la fin du film : une interruption entre deux chunks ne
		// laisse jamais un chunk converti derrière une taille périmée. Et la fenêtre qui reste (entre
		// le renommage du chunk et cette écriture) est réparée à la reprise : un chunk déjà zstd dont
		// la taille logique diffère du manifeste y est corrigé (cf. [recompresserChunk]).
		if taille := declare[numero]; logique > 0 && taille > 0 && taille != logique && !simulation {
			if err := corrigerTaillesDuManifeste(manifeste, map[int]int64{numero: logique}); err != nil {
				return rap, fmt.Errorf("filmcache: manifeste de %s : %w", shortID, err)
			}
			rap.ManifesteCorrige = true
		}
	}
	rap.Absents = src.NumChunks() - nombreDeclaresPresents(src, fichiers)
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
		if n, ok := filmcache.NumeroDuChunk(e.Name()); ok && !e.IsDir() {
			numeros = append(numeros, n)
		}
	}
	sort.Ints(numeros)
	return numeros, nil
}

// nombreDeclaresPresents compte les entrées du manifeste dont le fichier existe.
func nombreDeclaresPresents(src *filmcache.Source, fichiers []int) int {
	presents := make(map[int]bool, len(fichiers))
	for _, n := range fichiers {
		presents[n] = true
	}
	n := 0
	for _, c := range src.Meta() {
		if presents[c.Index] {
			n++
		}
	}
	return n
}

// recompresserChunk traite UN fichier de chunk, cumule son bilan dans `rap` et rend la taille
// logique du chunk quand le manifeste peut avoir à la corriger : un chunk converti, ou un chunk DÉJÀ
// zstd dont la taille logique diffère de `declare` (la taille que le manifeste lui donne, 0 =
// inconnue). Ce second cas est la reprise d'une passe interrompue entre la conversion d'un chunk
// zlib et la correction du manifeste ; la trame est alors décodée en entier avant d'être crue.
func recompresserChunk(path string, declare int64, simulation bool, rap *RapportRecompression) (int64, error) {
	fichier, err := os.ReadFile(path) //nolint:gosec // chemin compose par CheminDuChunk
	if errors.Is(err, os.ErrNotExist) {
		rap.Absents++
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if filmcache.EstZstd(fichier) {
		rap.DejaZstd++
		rap.OctetsDejaZstd += int64(len(fichier))
		return tailleLogiqueVerifiee(fichier, declare)
	}
	logique, err := filmcache.OctetsLogiquesStricts(fichier)
	if err != nil {
		return 0, err
	}
	z, err := filmcache.CompresserZstd(logique)
	if err != nil {
		return 0, err
	}
	if len(z) == 0 || len(z) >= len(fichier) {
		rap.SansGain++
		rap.OctetsSansGain += int64(len(fichier))
		return 0, nil
	}
	relu, err := filmcache.DecompresserZstd(z)
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

// tailleLogiqueVerifiee rend la taille logique d'une trame zstd quand elle diffère de `declare`
// (rien à corriger sinon : 0). La taille de l'en-tête n'est crue qu'une fois la trame décodée à
// cette longueur : un chunk réellement tronqué ne doit pas faire corriger le manifeste.
func tailleLogiqueVerifiee(trame []byte, declare int64) (int64, error) {
	if declare <= 0 {
		return 0, nil
	}
	entete, err := filmcache.TailleLogiqueZstd(trame[:min(len(trame), filmcache.TailleEnteteZstd)])
	if errors.Is(err, filmcache.ErrZstdTailleInconnue) || entete == declare {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	decode, err := filmcache.DecompresserZstd(trame)
	if err != nil || int64(len(decode)) != entete {
		return 0, fmt.Errorf("trame zstd illisible ou d'une autre longueur que son en-tête (%v)", err)
	}
	return entete, nil
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
