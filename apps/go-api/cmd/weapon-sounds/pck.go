package main

// pck.go — lecteur de Wwise File Package (`.pck`, magic `AKPK`).
//
// A QUOI IL SERT ICI. Un `.pck` par arme : `sb_010_wea_un_assaultrifle.pck` ne contient
// que du fusil d'assaut. L'ensemble de ses IDs `.wem` est donc un VALIDATEUR : il
// designe le `sbnk` de l'arme parmi 1305, et il tranche les ambiguites de layout du
// parseur de bank (un `sourceID` lu au mauvais offset n'appartient pas a cet ensemble).
//
// En-tete : magic `AKPK` | +0x04 tailleEntete | +0x08 version | +0x0C tailleTableLangues
//
//	+0x10 tailleTableBanks | +0x14 tailleTableStreams | +0x18 tailleTableExternes
//
// Chaque table : u32 nombre, puis des entrees {id, tailleBloc, tailleFichier,
// blocDepart, idLangue}. L'id est un u64 pour la table des externes, un u32 ailleurs.

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const magicAKPK = "AKPK"

// entreePck decrit un fichier embarque dans un `.pck`.
type entreePck struct {
	ID     uint32
	Offset int64
	Taille int
}

// lirePck rend les entrees d'un `.pck` (banks + streams + externes).
func lirePck(chemin string) ([]entreePck, error) {
	b, err := os.ReadFile(chemin)
	if err != nil {
		return nil, err
	}
	if len(b) < 0x1C || string(b[:4]) != magicAKPK {
		return nil, fmt.Errorf("pck: magic AKPK absent (%s)", chemin)
	}
	tailleLangues := int(binary.LittleEndian.Uint32(b[0x0C:]))
	tailleBanks := int(binary.LittleEndian.Uint32(b[0x10:]))
	tailleStreams := int(binary.LittleEndian.Uint32(b[0x14:]))
	tailleExternes := int(binary.LittleEndian.Uint32(b[0x18:]))

	debutBanks := 0x1C + tailleLangues
	debutStreams := debutBanks + tailleBanks
	debutExternes := debutStreams + tailleStreams

	var out []entreePck
	for _, t := range []struct {
		offset, taille int
		idLarge        bool
	}{
		{debutBanks, tailleBanks, false},
		{debutStreams, tailleStreams, false},
		{debutExternes, tailleExternes, true},
	} {
		e, err := lireTablePck(b, t.offset, t.taille, t.idLarge)
		if err != nil {
			return nil, err
		}
		out = append(out, e...)
	}
	return out, nil
}

// lireTablePck decode une table d'entrees du `.pck`.
func lireTablePck(b []byte, offset, taille int, idLarge bool) ([]entreePck, error) {
	if taille <= 4 {
		return nil, nil
	}
	if offset+4 > len(b) {
		return nil, fmt.Errorf("pck: table hors borne (offset %d)", offset)
	}
	n := int(binary.LittleEndian.Uint32(b[offset:]))
	p := offset + 4
	largeurID := 4
	if idLarge {
		largeurID = 8
	}
	stride := largeurID + 16
	if p+n*stride > len(b) {
		return nil, fmt.Errorf("pck: %d entrees debordent le fichier", n)
	}
	out := make([]entreePck, 0, n)
	for i := 0; i < n; i++ {
		var id uint32
		if idLarge {
			id = uint32(binary.LittleEndian.Uint64(b[p:]))
		} else {
			id = binary.LittleEndian.Uint32(b[p:])
		}
		q := p + largeurID
		tailleBloc := int64(binary.LittleEndian.Uint32(b[q:]))
		tailleFichier := int(binary.LittleEndian.Uint32(b[q+4:]))
		blocDepart := int64(binary.LittleEndian.Uint32(b[q+8:]))
		if tailleBloc == 0 {
			tailleBloc = 1
		}
		out = append(out, entreePck{ID: id, Offset: blocDepart * tailleBloc, Taille: tailleFichier})
		p += stride
	}
	return out, nil
}

// idsPck rend l'ensemble des IDs `.wem` d'un `.pck`.
func idsPck(chemin string) (map[uint32]bool, error) {
	ents, err := lirePck(chemin)
	if err != nil {
		return nil, err
	}
	set := make(map[uint32]bool, len(ents))
	for _, e := range ents {
		set[e.ID] = true
	}
	return set, nil
}

// indexTousPcks rend, pour TOUS les `.pck` du dossier, l'identifiant `.wem` -> son pack.
//
// POURQUOI CET INDEX EXISTE. Le parseur de bank validait chaque `sourceID` contre le seul
// pack de l'arme. C'etait un garde-fou efficace contre les mauvais offsets, mais il
// REJETAIT SILENCIEUSEMENT toute couche dont le son vit dans un autre pack — mecanique
// commune, queues d'environnement. Mesure : 2 des 4 couches du fusil d'assaut et 1 des 3
// du Skewer disparaissaient ainsi. Ce sont precisement les couches qui manquaient a
// l'oreille. L'index large (environ 90 000 identifiants sur 2^32) reste un filtre tres
// selectif, tout en laissant passer les couches partagees.
func indexTousPcks(dossier string) (map[uint32]string, error) {
	chemins, err := filepath.Glob(filepath.Join(dossier, "*.pck"))
	if err != nil {
		return nil, err
	}
	if len(chemins) == 0 {
		return nil, fmt.Errorf("aucun .pck dans %s", dossier)
	}
	out := make(map[uint32]uint32, 1<<17)
	source := make(map[uint32]string, 1<<17)
	for _, c := range chemins {
		ents, err := lirePck(c)
		if err != nil {
			continue
		}
		for _, e := range ents {
			if _, vu := out[e.ID]; !vu {
				out[e.ID] = 1
				source[e.ID] = c
			}
		}
	}
	return source, nil
}

// dossierSFXParDefaut deduit le dossier des `.pck` de la racine `deploy` du jeu.
func dossierSFXParDefaut(racineDeploy string) string {
	return filepath.Join(filepath.Dir(racineDeploy), "Sound", "win", "SFX")
}

// nomFichierSansExt rend le nom de fichier prive de son extension.
func nomFichierSansExt(chemin string) string {
	return strings.TrimSuffix(filepath.Base(chemin), filepath.Ext(chemin))
}

// nomArme rend le nom d'arme lisible depuis le nom de fichier d'un `.pck`.
// Exemple : sb_010_wea_un_assaultrifle.pck -> un_assaultrifle
func nomArme(chemin string) string {
	base := nomFichierSansExt(chemin)
	for _, p := range []string{"sb_010_wea_", "sb_010_tur_", "sb_010_whizby_"} {
		if strings.HasPrefix(base, p) {
			return strings.TrimPrefix(base, p)
		}
	}
	return base
}
