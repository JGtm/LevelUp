// tmp_fetchchunk0 — telecharger le chunk d'EN-TETE d'un film (chunk_type 1), que
// cmd/fetch_film_chunks ne prend pas.
//
// POURQUOI. fetch_film_chunks ne telecharge que les chunks REPLICATION_DATA (chunk_type 2) :
// c'est le gros du flux et c'est ce qu'il faut pour rejouer. Mais le chunk 0, de type 1 et de
// duree quasi nulle (11 ms au manifeste), porte l'ETAT INITIAL DU MONDE — 435 Ko sur le film
// de reference. Sans lui, le decodeur ne peut pas amorcer son World et rend
// « AUCUN keyframe type-2 : impossible de bootstrapper le World ».
//
// Le fichier est ecrit DECOMPRESSE (zlib), comme le fait fetch_film_chunks, pour que les deux
// sources produisent des fichiers homogenes dans le meme dossier.
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type manifest struct {
	BlobPrefix string `json:"blob_prefix"`
	Chunks     []struct {
		Index            int    `json:"index"`
		ChunkType        int    `json:"chunk_type"`
		FileRelativePath string `json:"file_relative_path"`
	} `json:"chunks"`
}

func main() {
	short := flag.String("film", "", "prefixe du film (8 hex), ex. 9e8fb31b")
	cache := flag.String("cache", `C:\Users\Guillaume\Downloads\Scripts\LevelUp-go-migration\data\cache`, "racine du cache")
	index := flag.Int("index", 0, "index du chunk a recuperer")
	flag.Parse()
	if *short == "" {
		fmt.Fprintln(os.Stderr, "usage: tmp_fetchchunk0 -film <prefixe> [-index 0]")
		os.Exit(2)
	}

	mfPath := filepath.Join(*cache, "film_manifests", *short+".json")
	raw, err := os.ReadFile(mfPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "manifeste :", err)
		os.Exit(1)
	}
	var mf manifest
	if err := json.Unmarshal(raw, &mf); err != nil {
		fmt.Fprintln(os.Stderr, "manifeste illisible :", err)
		os.Exit(1)
	}

	var url string
	var chunkType int
	for _, c := range mf.Chunks {
		if c.Index == *index {
			url = mf.BlobPrefix + c.FileRelativePath[1:]
			chunkType = c.ChunkType
			break
		}
	}
	if url == "" {
		fmt.Fprintf(os.Stderr, "chunk %d absent du manifeste\n", *index)
		os.Exit(1)
	}
	fmt.Printf("chunk %d (type %d)\n  %s\n", *index, chunkType, url)

	resp, err := (&http.Client{Timeout: 60 * time.Second}).Get(url)
	if err != nil {
		fmt.Fprintln(os.Stderr, "telechargement :", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "HTTP %d — URL pre-signee expiree ? rafraichir le manifeste.\n", resp.StatusCode)
		os.Exit(1)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	out := body
	if zr, err := zlib.NewReader(bytes.NewReader(body)); err == nil {
		if dec, err := io.ReadAll(zr); err == nil {
			out = dec
		}
		zr.Close()
	}

	dir := filepath.Join(*cache, "film_chunks", *short)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	dst := filepath.Join(dir, fmt.Sprintf("chunk_%02d.bin", *index))
	if err := os.WriteFile(dst, out, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("  -> %s (%d octets telecharges, %d apres decompression)\n", dst, len(body), len(out))
}
