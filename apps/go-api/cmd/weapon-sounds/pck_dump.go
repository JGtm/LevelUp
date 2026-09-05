package main

// pck_dump.go — mode `pck-dump` : extrait les `.wem` COMPLETS d'un `.pck` (AKPK), sans module.
//
// POURQUOI CE MODE. La version d'un `.wem` embarquee dans une banque (`DIDX`/`DATA`) est un
// PREFETCH tronque (mesure du lot V3B : ~0,5 s au lieu de plusieurs secondes) ; le media
// complet est STREAME depuis le `.pck` du pack. Jusqu'ici l'etape `.pck -> .wem` passait par
// un script Python hors depot (`_outils/akpk_unpack.py`), alors que `lirePck` (pck.go) sait
// deja lire le conteneur a l'octet pres. Ce mode ferme le trou : la chaine complete
// (`pck -> wem -> vgmstream -> wav`) reste dans le depot, en Go.
//
// Usage : -mode pck-dump -pck <fichier.pck> -out <dossier> [-wem id1,id2,...] [-etroit]
// `-wem` filtre les identifiants ; absent, tout le pack est extrait. `-etroit` evite de
// construire l'index large des 841 packs, inutile ici.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// extrairePck ecrit les medias d'un `.pck` sous forme de fichiers `<id>.wem`.
func extrairePck(cheminPck, dossierSortie string, filtre map[uint32]bool) error {
	if cheminPck == "" || dossierSortie == "" {
		return fmt.Errorf("le mode pck-dump exige -pck (fichier .pck) et -out (dossier)")
	}
	ents, err := lirePck(cheminPck)
	if err != nil {
		return err
	}
	brut, err := os.ReadFile(cheminPck)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dossierSortie, 0o755); err != nil {
		return err
	}
	sort.Slice(ents, func(i, j int) bool { return ents[i].ID < ents[j].ID })
	ecrits, ignores, octets := 0, 0, 0
	for _, e := range ents {
		if len(filtre) > 0 && !filtre[e.ID] {
			continue
		}
		fin := e.Offset + int64(e.Taille)
		if e.Offset < 0 || fin > int64(len(brut)) || e.Taille <= 0 {
			fmt.Fprintf(os.Stderr, "  media %d hors borne (offset %d, taille %d) : ignore\n", e.ID, e.Offset, e.Taille)
			ignores++
			continue
		}
		cible := filepath.Join(dossierSortie, fmt.Sprintf("%d.wem", e.ID))
		if err := os.WriteFile(cible, brut[e.Offset:fin], 0o644); err != nil {
			return err
		}
		ecrits++
		octets += e.Taille
	}
	fmt.Printf("pck      : %s\n", filepath.Base(cheminPck))
	fmt.Printf("medias   : %d dans le pack, %d ecrit(s), %d hors borne\n", len(ents), ecrits, ignores)
	fmt.Printf("sortie   : %s (%.2f Mo)\n", dossierSortie, float64(octets)/(1024*1024))
	return nil
}
