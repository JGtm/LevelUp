package filmcache

// write.go — LE writer du cache film. Il vit DANS ce paquet pour la meme raison que le
// lecteur : la disposition du cache n'est declaree qu'ici (cf. l'en-tete de
// filmcache.go), et un writer qui vivrait ailleurs re-divergerait au premier renommage de
// chunk.
//
// POURQUOI UN WRITER, alors que le cache etait historiquement herite du projet Python :
// les films EXPIRENT cote serveur Halo (~29 % du corpus deja perdus). Le pont disque du
// post-sync persiste les chunks telecharges pour l'artefact de rejeu — au passage il
// archive une donnee IRREMPLACABLE (un film expire ne se re-telecharge jamais).
//
// CONTRAT D'ECRITURE : les chunks d'abord, le manifeste EN DERNIER (le manifeste est le
// marqueur de commit : Open ne lit rien sans lui, donc une ecriture interrompue laisse
// des chunks orphelins invisibles, jamais un film a moitie lisible). Un manifeste FINALISE
// deja present n'est jamais reecrit : celui du cache historique porte un blob_prefix CDN que
// le notre n'aurait pas, et l'ecraser perdrait le repli reseau des chunks manquants.
//
// « PRESENT » N'EST PAS « COMPLET » (SRC-2/OPS-4, audit du decodeur de film, 2026-09-25). Un
// chunk s'ecrit par `atomicfile.WriteFileStrict` : il est entier ou absent, jamais tronque sous
// son nom final. Un chunk DEJA present n'est adopte que si sa taille vaut celle du
// telechargement — un chunk tronque par une ecriture en place d'avant cette regle est remplace
// (WARN + compteur `film_cache_chunk_remplace`). Le manifeste ecrit porte la taille de chaque
// chunk (`size_bytes`), que le lecteur compare (cf. [ErrChunkTronque]) ; un manifeste historique
// sans taille reste lisible et n'est pas rempli apres coup.
//
// SEUL UN FILM FINALISE SE VALIDE (lot L3, 2026-09-23 — cf. finalise.go). Une liste sans
// morceau de temps forts est refusee AVANT toute ecriture ([ErrFilmNonFinalise]) : ni
// manifeste, ni morceau orphelin. Deux manifestes deja presents se remplacent, et eux seuls :
//   - un manifeste SANS temps forts — ecrit avant cette regle, `ab526724` le 2026-09-22 — par
//     la liste finalisee qui le complete, a condition qu'elle en soit un SUR-ENSEMBLE EXACT
//     par index ([ErrManifesteDivergent] sinon, et rien n'est touche) ;
//   - un manifeste ILLISIBLE, par la liste finalisee (WARN + compteur
//     `film_cache_manifeste_repare`) : sans cela il bloquait le film pour toujours.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/platform/atomicfile"
)

// EnsureDirs cree les deux dossiers du cache et rend la racine prete a l emploi.
//
// POURQUOI ELLE EXISTE ICI ET NULLE PART AILLEURS. `haloclient.NewLocalFilmCache` rend nil
// quand `film_manifests/` n existe pas, et ce nil vaut pour toute la vie du process : sur une
// machine au cache vide, un producteur archiverait sans jamais relire, et chaque passe
// repaierait le reseau. Trois appelants avaient besoin de cette preparation et la
// reconstituaient chacun de leur cote — dont le detour consistant a deriver le dossier des
// manifestes en prenant le parent du chemin d un manifeste fictif. La disposition n est
// declaree que dans ce paquet (cf. l'en-tete de filmcache.go) : sa CREATION doit y vivre
// aussi, sinon elle re-diverge au premier renommage.
func EnsureDirs(root string) error {
	if strings.TrimSpace(root) == "" {
		return fmt.Errorf("filmcache: racine de cache vide")
	}
	for _, d := range []string{filepath.Join(root, chunksDir), filepath.Join(root, manifestsDir)} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("filmcache: creation de %s : %w", d, err)
		}
	}
	return nil
}

// WriteChunk est un chunk a persister, avec les champs du manifeste.
type WriteChunk struct {
	Index      int
	ChunkType  int
	StartMS    int
	DurationMS int
	Data       []byte
}

// Compteurs de l'ecrivain (expvar, cf. `observability`).
const (
	// compteurChunkRemplace : un chunk present sur disque dont la taille ne valait pas celle du
	// telechargement (ecriture interrompue d'avant l'ecriture atomique) a ete remplace.
	compteurChunkRemplace = "film_cache_chunk_remplace"
	// compteurManifesteRepare : un manifeste illisible a ete remplace par une liste finalisee.
	compteurManifesteRepare = "film_cache_manifeste_repare"
)

// ErrManifesteDivergent : un manifeste NON FINALISE deja present ne se complete que par une liste
// qui en est un SUR-ENSEMBLE EXACT — chaque entree deja validee retrouvee au meme index, avec le
// meme type, le meme debut et la meme duree. Une divergence dit que les morceaux deja ecrits ne
// decrivent peut-etre plus le meme film : rien n'est ecrit, et l'appelant le journalise.
var ErrManifesteDivergent = errors.New("filmcache: la liste ne complete pas exactement le manifeste partiel deja present")

// sortDuManifeste : ce que [Write] fait du manifeste, decide AVANT toute ecriture.
type sortDuManifeste int

const (
	manifesteAEcrireNeuf sortDuManifeste = iota // absent, ou partiel complete exactement
	manifesteAConserver                         // finalise deja present (historique : blob_prefix CDN)
	manifesteAReparer                           // present mais illisible
)

// Write persiste un film FINALISE dans le cache : les chunks manquants ou tronques, puis le
// manifeste s'il n'existe pas encore (ou s'il faut completer un manifeste partiel, ou remplacer
// un manifeste illisible). Idempotent : un chunk deja present A LA BONNE TAILLE n'est pas
// reecrit (le film est immuable cote serveur).
//
// Refus, TOUS AVANT LA MOINDRE ECRITURE : liste vide, film non finalise
// ([ErrFilmNonFinalise]), manifeste partiel que la liste ne complete pas exactement
// ([ErrManifesteDivergent]), manifeste present mais impossible a lire sur disque.
func Write(ctx context.Context, root, shortID string, chunks []WriteChunk) error {
	if len(chunks) == 0 {
		return fmt.Errorf("filmcache: aucun chunk a ecrire pour %s", shortID)
	}
	if !Finalise(chunks, typeAEcrire) {
		return fmt.Errorf("filmcache: %s (%d morceaux) : %w", shortID, len(chunks), ErrFilmNonFinalise)
	}
	manifestPath := ManifestPath(root, shortID)
	sortMf, err := manifesteAEcrire(ctx, manifestPath, chunks)
	if err != nil {
		return fmt.Errorf("filmcache: %s : %w", shortID, err)
	}
	dir := ChunkDir(root, shortID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("filmcache: creation du dossier de chunks %s : %w", dir, err)
	}
	for _, c := range chunks {
		if err := ecrireChunk(ctx, dir, shortID, c); err != nil {
			return err
		}
	}
	if sortMf == manifesteAConserver {
		return nil
	}
	if err := ecrireManifeste(manifestPath, shortID, chunks); err != nil {
		return err
	}
	if sortMf == manifesteAReparer {
		observability.IncCounter(compteurManifesteRepare)
	}
	return nil
}

// ecrireChunk pose un chunk, ATOMIQUEMENT. Un fichier deja present n'est adopte que s'il est
// regulier et a la taille du telechargement ; sinon il est remplace, et le remplacement se
// signale.
func ecrireChunk(ctx context.Context, dir, shortID string, c WriteChunk) error {
	path := CheminDuChunk(dir, c.Index)
	info, err := os.Stat(path)
	present := err == nil
	switch {
	case present && info.Mode().IsRegular() && info.Size() == int64(len(c.Data)):
		return nil // deja entier : le film est immuable, on ne reecrit pas
	case !present && !errors.Is(err, os.ErrNotExist):
		return fmt.Errorf("filmcache: etat du chunk %d de %s : %w", c.Index, shortID, err)
	}
	if err := atomicfile.WriteFileStrict(path, c.Data, 0o644); err != nil {
		return fmt.Errorf("filmcache: ecriture du chunk %d de %s : %w", c.Index, shortID, err)
	}
	if present {
		observability.IncCounter(compteurChunkRemplace)
		slog.WarnContext(ctx, "filmcache: chunk present a la mauvaise taille, remplace",
			"short_id", shortID, "chunk", c.Index, "taille_disque", info.Size(),
			"taille_attendue", len(c.Data))
	}
	return nil
}

// manifesteAEcrire decide du sort du manifeste, SANS rien ecrire : a ecrire quand il est absent,
// ou quand il est PARTIEL et que `chunks` (finalisee) le complete exactement ; a conserver quand
// un manifeste finalise est deja la ; a reparer quand il est illisible.
func manifesteAEcrire(ctx context.Context, manifestPath string, chunks []WriteChunk) (sortDuManifeste, error) {
	raw, err := os.ReadFile(manifestPath) //nolint:gosec // chemin compose par ManifestPath
	if errors.Is(err, os.ErrNotExist) {
		return manifesteAEcrireNeuf, nil
	}
	if err != nil {
		return 0, fmt.Errorf("lecture du manifeste existant : %w", err)
	}
	var existant writeManifestJSON
	if err := json.Unmarshal(raw, &existant); err != nil {
		slog.WarnContext(ctx, "filmcache: manifeste illisible, remplace par la liste finalisee",
			"path", manifestPath, "err", err)
		return manifesteAReparer, nil
	}
	if Finalise(existant.Chunks, typeDuManifeste) {
		return manifesteAConserver, nil
	}
	parIndex := make(map[int]WriteChunk, len(chunks))
	for _, c := range chunks {
		parIndex[c.Index] = c
	}
	for _, e := range existant.Chunks {
		c, ok := parIndex[e.Index]
		if !ok || c.ChunkType != e.ChunkType || c.StartMS != e.StartMS || c.DurationMS != e.DurationMS {
			return 0, fmt.Errorf("%w (entree %d)", ErrManifesteDivergent, e.Index)
		}
	}
	return manifesteAEcrireNeuf, nil
}

// ecrireManifeste pose le marqueur de commit, ATOMIQUEMENT : il peut desormais REMPLACER un
// manifeste partiel ou illisible, et une ecriture interrompue ne doit laisser ni l'ancien
// tronque ni un nouveau illisible.
func ecrireManifeste(manifestPath, shortID string, chunks []WriteChunk) error {
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		return fmt.Errorf("filmcache: creation du dossier de manifestes : %w", err)
	}
	mf := writeManifestJSON{}
	for _, c := range chunks {
		mf.Chunks = append(mf.Chunks, writeManifestChunk{
			Index: c.Index, ChunkType: c.ChunkType, StartMS: c.StartMS, DurationMS: c.DurationMS,
			SizeBytes: int64(len(c.Data)),
		})
	}
	blob, err := json.Marshal(mf)
	if err != nil {
		return fmt.Errorf("filmcache: serialisation du manifeste %s : %w", shortID, err)
	}
	if err := atomicfile.WriteFile(manifestPath, blob, 0o644); err != nil {
		return fmt.Errorf("filmcache: ecriture du manifeste %s : %w", shortID, err)
	}
	return nil
}

// typeAEcrire / typeDuManifeste : les accesseurs de type que [Finalise] recoit.
func typeAEcrire(c WriteChunk) int             { return c.ChunkType }
func typeDuManifeste(c writeManifestChunk) int { return c.ChunkType }

// writeManifestJSON reprend la forme du manifeste historique (manifestJSON cote lecture,
// + duration_ms que le cache Python ecrivait aussi). blob_prefix est volontairement
// ABSENT : tous les chunks sont sur disque, il n'y a pas de repli CDN a declarer.
type writeManifestJSON struct {
	Chunks []writeManifestChunk `json:"chunks"`
}

// writeManifestChunk : une entree du manifeste. `size_bytes` est FACULTATIF (DT-4) : absent des
// manifestes historiques, qui restent lisibles sans controle de taille.
type writeManifestChunk struct {
	Index      int   `json:"index"`
	ChunkType  int   `json:"chunk_type"`
	StartMS    int   `json:"start_ms"`
	DurationMS int   `json:"duration_ms"`
	SizeBytes  int64 `json:"size_bytes,omitempty"`
}
