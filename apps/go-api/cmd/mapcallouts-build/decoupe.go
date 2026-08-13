package main

// decoupe.go — l'ingestion du DUMP DÉCOUPÉ de Ridgeline.
//
// RIDGELINE SEULE est découpée sur le sol praticable : le dump versionné
// .ai/V7.5/dumps/callout_zones_ridgeline_clipped.json (produit par les scripts de
// recherche, confronté aux positions de joueurs : réduction d'aire 21 % à rétention
// 98,2 %, quarante écarts-types au-dessus du hasard) est LA source de ses polygones —
// on le CONSERVE, on ne le refait pas. Les 21 autres cartes livrent le polygone
// designer BRUT ; le découpage universel dépend du chantier cartes (en pause) et vit
// au registre des reports.
//
// Le dump choisit lui-même, zone par zone, le polygone fiable (`polygone_a_utiliser`) :
// 21 zones découpées, 7 repliées sur le brut (fiabilité insuffisante mesurée). On
// respecte ce choix tel quel.

import (
	"encoding/json"
	"fmt"
	"os"

	"levelup/go-api/internal/analysis/replay"
)

// decoupeDump est la partie du dump que l'ingestion consomme.
type decoupeDump struct {
	Carte  string        `json:"carte"`
	Module string        `json:"module"`
	Zones  []decoupeZone `json:"zones"`
}

type decoupeZone struct {
	VolumeIndex int    `json:"volumeIndex"`
	LibelleEN   string `json:"libelle_en"`
	Utiliser    string `json:"polygone_a_utiliser"` // "decoupe" | "brut"
	Brut        struct {
		Polygone [][2]float64 `json:"polygone"`
	} `json:"brut"`
	Decoupe struct {
		Parties []struct {
			Contour [][2]float64   `json:"contour"`
			Trous   [][][2]float64 `json:"trous"`
		} `json:"parties"`
	} `json:"decoupe"`
}

// chargeDecoupe lit le dump et rend ses zones indexées par volumeIndex.
func chargeDecoupe(path string) (module string, zones map[int]decoupeZone, err error) {
	blob, err := os.ReadFile(path)
	if err != nil {
		return "", nil, fmt.Errorf("dump découpé illisible (%s) : %w", path, err)
	}
	var d decoupeDump
	if err := json.Unmarshal(blob, &d); err != nil {
		return "", nil, fmt.Errorf("dump découpé invalide (%s) : %w", path, err)
	}
	zones = make(map[int]decoupeZone, len(d.Zones))
	for _, z := range d.Zones {
		zones[z.VolumeIndex] = z
	}
	return d.Module, zones, nil
}

// appliqueDecoupe remplace les polygones d'une entrée par ceux du dump découpé.
//
// La jointure se fait par volumeIndex et se VÉRIFIE par le libellé EN : un dump qui ne
// parlerait plus des mêmes zones que le tag doit échouer, pas se poser en silence.
func appliqueDecoupe(e *replay.MapCalloutsEntry, zones map[int]decoupeZone) error {
	for i := range e.Zones {
		z := &e.Zones[i]
		d, ok := zones[z.VolumeIndex]
		if !ok {
			return fmt.Errorf("volume %d (%s) absent du dump découpé", z.VolumeIndex, z.EN)
		}
		if d.LibelleEN != z.EN {
			return fmt.Errorf("volume %d : libellé du dump %q != tag %q", z.VolumeIndex, d.LibelleEN, z.EN)
		}
		if d.Utiliser == "decoupe" && len(d.Decoupe.Parties) > 0 {
			z.Polygon = d.Decoupe.Parties[0].Contour
			z.Parts, z.Holes = nil, nil
			for k, p := range d.Decoupe.Parties {
				if k > 0 {
					z.Parts = append(z.Parts, p.Contour)
				}
				z.Holes = append(z.Holes, p.Trous...)
			}
			continue
		}
		// Repli déclaré par le dump : le brut du DUMP (identique au tag, et c'est vérifié
		// par le témoin gamefiles du lecteur — même source, même mesure).
		if len(d.Brut.Polygone) > 0 {
			z.Polygon = d.Brut.Polygone
			z.Parts, z.Holes = nil, nil
		}
	}
	e.Provenance = replay.CalloutsProvenanceDecoupe
	return nil
}
