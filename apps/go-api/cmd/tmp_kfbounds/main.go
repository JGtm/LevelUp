// tmp_kfbounds — bornes des records du monde keyframe (paquets type-2), en JSON.
//
// POURQUOI. La verite terrain du 2026-07-27 est relevee au debut REEL du match (image 250).
// Or la chaine de composants DELTA n'emet aucune lecture i22/i47/i48 entre les images 35 et
// 326 (mesure : 0 lecture, tous archetypes, tous types de paquets). L'etat vrai a l'image
// 250 vit donc dans le KEYFRAME de l'image 199 — c'est deja de la que le POC tire les
// loadouts d'armes, qui concordent 8/8 avec le releve.
//
// Ce programme ne decode rien : il donne les BORNES (slot, archetype, generation, bit de
// depart) de chaque record de chaque keyframe, pour que la suite du travail se fasse en
// Python sur les charges utiles deja extraites.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"levelup/go-api/internal/analysis/filmdec"
)

type rec struct {
	Slot int `json:"slot"`
	TI   int `json:"archetype"`
	Gen  int `json:"generation"`
	Bit  int `json:"bit_depart"`
}

type kf struct {
	Chunk       int    `json:"chunk"`
	PacketIndex int    `json:"paquet"`
	TimestampUS uint64 `json:"ts"`
	Size        int    `json:"taille"`
	Recs        []rec  `json:"records"`
}

func main() {
	dir := flag.String("film", "", "dossier des chunks du film")
	out := flag.String("out", "", "fichier JSON de sortie")
	flag.Parse()
	if *dir == "" || *out == "" {
		fmt.Fprintln(os.Stderr, "usage: tmp_kfbounds -film <dir> -out <json>")
		os.Exit(2)
	}
	n := filmdec.CountFilmChunks(*dir)
	var all []kf
	for c := 1; c <= n; c++ {
		chunk, err := filmdec.ReadFilmChunk(*dir, c)
		if err != nil {
			fmt.Fprintf(os.Stderr, "chunk %d : %v\n", c, err)
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeKeyframe {
				continue
			}
			pay := p.Payload(chunk)
			var rs []rec
			for _, r := range filmdec.WalkKeyframeWorld(pay) {
				rs = append(rs, rec{Slot: r.Slot, TI: r.TI, Gen: r.Gen, Bit: r.Bit})
			}
			all = append(all, kf{Chunk: c, PacketIndex: p.Index, TimestampUS: p.TimestampUS,
				Size: p.Size, Recs: rs})
			fmt.Printf("chunk %02d paquet %02d : %d records\n", c, p.Index, len(rs))
		}
	}
	b, err := json.Marshal(all)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, b, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("-> %s (%d keyframes, %d octets)\n", *out, len(all), len(b))
}
