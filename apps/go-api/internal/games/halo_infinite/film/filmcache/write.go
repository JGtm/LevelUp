package filmcache

// write.go — LE writer du cache film. Il vit DANS ce paquet pour la meme raison que le
// lecteur : la disposition du cache n'est declaree qu'ici (garde-rail
// filmcache_guard_test.go), et un writer qui vivrait ailleurs re-divergerait au premier
// renommage de chunk.
//
// POURQUOI UN WRITER, alors que le cache etait historiquement herite du projet Python :
// les films EXPIRENT cote serveur Halo (~29 % du corpus deja perdus). Le pont disque du
// post-sync persiste les chunks telecharges pour l'artefact de rejeu — au passage il
// archive une donnee IRREMPLACABLE (un film expire ne se re-telecharge jamais).
//
// CONTRAT D'ECRITURE : les chunks d'abord, le manifeste EN DERNIER (le manifeste est le
// marqueur de commit : Open ne lit rien sans lui, donc une ecriture interrompue laisse
// des chunks orphelins invisibles, jamais un film a moitie lisible). Un manifeste DEJA
// PRESENT n'est jamais reecrit : celui du cache historique porte un blob_prefix CDN que
// le notre n'aurait pas, et l'ecraser perdrait le repli reseau des chunks manquants.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// WriteChunk est un chunk a persister, avec les champs du manifeste.
type WriteChunk struct {
	Index      int
	ChunkType  int
	StartMS    int
	DurationMS int
	Data       []byte
}

// Write persiste un film complet dans le cache : les fichiers de chunks manquants, puis
// le manifeste s'il n'existe pas encore. Idempotent : un chunk deja present sur disque
// n'est pas reecrit (le film est immuable cote serveur).
func Write(root, shortID string, chunks []WriteChunk) error {
	if len(chunks) == 0 {
		return fmt.Errorf("filmcache: aucun chunk a ecrire pour %s", shortID)
	}
	dir := ChunkDir(root, shortID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("filmcache: creation du dossier de chunks %s : %w", dir, err)
	}
	for _, c := range chunks {
		path := filepath.Join(dir, chunkName(c.Index))
		if _, err := os.Stat(path); err == nil {
			continue // deja present : le film est immuable, on ne reecrit pas
		}
		if err := os.WriteFile(path, c.Data, 0o644); err != nil {
			return fmt.Errorf("filmcache: ecriture du chunk %d de %s : %w", c.Index, shortID, err)
		}
	}

	manifestPath := ManifestPath(root, shortID)
	if _, err := os.Stat(manifestPath); err == nil {
		return nil // manifeste historique conserve (blob_prefix CDN)
	}
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		return fmt.Errorf("filmcache: creation du dossier de manifestes : %w", err)
	}
	mf := writeManifestJSON{}
	for _, c := range chunks {
		mf.Chunks = append(mf.Chunks, writeManifestChunk{
			Index: c.Index, ChunkType: c.ChunkType, StartMS: c.StartMS, DurationMS: c.DurationMS,
		})
	}
	blob, err := json.Marshal(mf)
	if err != nil {
		return fmt.Errorf("filmcache: serialisation du manifeste %s : %w", shortID, err)
	}
	if err := os.WriteFile(manifestPath, blob, 0o644); err != nil {
		return fmt.Errorf("filmcache: ecriture du manifeste %s : %w", shortID, err)
	}
	return nil
}

// writeManifestJSON reprend la forme du manifeste historique (manifestJSON cote lecture,
// + duration_ms que le cache Python ecrivait aussi). blob_prefix est volontairement
// ABSENT : tous les chunks sont sur disque, il n'y a pas de repli CDN a declarer.
type writeManifestJSON struct {
	Chunks []writeManifestChunk `json:"chunks"`
}

type writeManifestChunk struct {
	Index      int `json:"index"`
	ChunkType  int `json:"chunk_type"`
	StartMS    int `json:"start_ms"`
	DurationMS int `json:"duration_ms"`
}
