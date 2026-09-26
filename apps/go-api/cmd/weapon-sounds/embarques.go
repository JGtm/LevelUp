package main

// embarques.go — extrait les `.wem` EMBARQUES dans une bank (chunks `DIDX` + `DATA`).
//
// POURQUOI CE MODE EXISTE. Les `.pck` du jeu ne contiennent pas tous les sons : une bank
// peut porter ses propres medias. Mesure sur le fusil d'assaut : deux des quatre couches
// de l'evenement de tir ne resolvaient AUCUN son tant qu'on ne regardait que les packs, et
// se remplissent des qu'on lit `DIDX` (0 -> 1 et 0 -> 14 sons ; l'evenement passe de 103 a
// 133 `.wem`). Ce sont exactement les couches qui manquaient a l'oreille.
//
// `DIDX` indexe {identifiant, offset, taille} ; les octets vivent dans `DATA` a cet offset.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"levelup/go-api/internal/himodule"
)

// extraireEmbarques ecrit les medias embarques d'une bank dans un dossier.
//
// La bank est designee soit par le `.pck` de l'arme (intersection des IDs), soit
// DIRECTEMENT par son identifiant. Le second cas existe pour les armes dont aucun pack ne
// correspond — la Carabine Vestige a un `weap` et une `sbnk`, mais pas de `.pck` parmi les
// 55 traites : sans acces par identifiant, ses sons resteraient hors de portee.
func extraireEmbarques(cheminModule, cheminPck, dossierSortie string, gidBank uint32) error {
	if dossierSortie == "" {
		return fmt.Errorf("le mode embarques exige -out (dossier de sortie)")
	}
	var ids map[uint32]bool
	if cheminPck != "" {
		var err error
		if ids, err = idsPck(cheminPck); err != nil {
			return err
		}
	} else if gidBank == 0 {
		return fmt.Errorf("le mode embarques exige -pck ou -sbnk")
	}
	m, err := himodule.Open(cheminModule)
	if err != nil {
		return err
	}
	rapporterMemoire("module charge")

	var f himodule.File
	var brut []byte
	score := 0
	if cheminPck != "" {
		if f, brut, score, err = trouverSbnk(m, ids); err != nil {
			return err
		}
	} else {
		if f, brut, err = bankParIdentifiant(m, gidBank); err != nil {
			return err
		}
	}
	ch := chunks(brut)
	index := mediasEmbarques(ch)
	data, ok := ch["DATA"]
	fmt.Printf("arme     : %s\n", nomArme(cheminPck))
	fmt.Printf("sbnk     : %08x (score %d)\n", f.GlobalID, score)
	fmt.Printf("DIDX     : %d media(s) embarque(s) | DATA present : %v\n", len(index), ok)
	if len(index) == 0 || !ok {
		fmt.Println("rien a extraire")
		return nil
	}
	if err := os.MkdirAll(dossierSortie, 0o755); err != nil {
		return err
	}

	_ = data
	n, err := ecrireEmbarques(ch, index, dossierSortie)
	if err != nil {
		return err
	}
	fmt.Printf("ecrits   : %d fichiers .wem dans %s\n", n, dossierSortie)
	return nil
}

// bankParIdentifiant rend une bank par son global tag id, sans passer par un `.pck`.
func bankParIdentifiant(m *himodule.Module, gid uint32) (himodule.File, []byte, error) {
	for _, f := range m.Files("sbnk") {
		if f.GlobalID != gid {
			continue
		}
		data, err := m.Extract(f)
		if err != nil {
			return f, nil, err
		}
		debut := indexBKHD(data)
		if debut < 0 {
			return f, nil, fmt.Errorf("sbnk %08x sans chunk BKHD", gid)
		}
		return f, data[debut:], nil
	}
	return himodule.File{}, nil, fmt.Errorf("sbnk %08x absente de ce module", gid)
}

// ecrireEmbarques decoupe le chunk `DATA` selon l'index `DIDX` et ecrit un `.wem` par media.
func ecrireEmbarques(ch map[string][]byte, index map[uint32][2]uint32, dossier string) (int, error) {
	data, ok := ch["DATA"]
	if !ok || len(index) == 0 {
		return 0, nil
	}
	if err := os.MkdirAll(dossier, 0o755); err != nil {
		return 0, err
	}
	cles := make([]uint32, 0, len(index))
	for id := range index {
		cles = append(cles, id)
	}
	sort.Slice(cles, func(i, j int) bool { return cles[i] < cles[j] })

	ecrits := 0
	for _, id := range cles {
		e := index[id]
		debut, taille := int(e[0]), int(e[1])
		if debut < 0 || taille <= 0 || debut+taille > len(data) {
			continue // entree hors des bornes de DATA : on saute plutot que d'ecrire du bruit
		}
		nom := filepath.Join(dossier, fmt.Sprintf("%d.wem", id))
		if err := os.WriteFile(nom, data[debut:debut+taille], 0o644); err != nil {
			return ecrits, err
		}
		ecrits++
	}
	return ecrits, nil
}
