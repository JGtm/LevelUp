// Package himodule — resources.go : le BLOB DE RESSOURCES d'un tag.
//
// POURQUOI CE FICHIER EXISTE. La geometrie de rendu (sommets, indices) n'est PAS dans les
// octets du tag : le tag ne porte que des descripteurs, dont le champ `off` est un offset
// d'octets dans la CONCATENATION des entrees-ressource de ce tag. Sans cette concatenation,
// les descripteurs pointent dans le vide et aucune coordonnee n'existe.
//
// Chaque entree de fichier porte l'index de sa PREMIERE ressource (+0x10) ; ses ressources
// vont de la sienne a celle de l'entree suivante. Ces index designent des lignes de la table
// `resourceSlots`, placee juste AVANT la table des blocs, qui donne l'index de l'entree
// portant reellement la donnee.
package himodule

import "fmt"

// offResourceIndex : index de la premiere ressource de l'entree.
const offResourceIndex = 0x10

// resourceCountOffset : nombre total de ressources du module (en-tete).
const resourceCountOffset = 0x28

// ResourceIndex rend l'index de la premiere ressource d'une entree.
func (m *Module) ResourceIndex(i int) int {
	return int(u32(m.data, headerSize+i*entryStride+offResourceIndex))
}

// resourceCount rend le nombre total de ressources declarees par l'en-tete.
func (m *Module) resourceCount() int { return int(u32(m.data, resourceCountOffset)) }

// resourceSlot rend l'index d'entree porteur de la k-ieme ressource.
//
// La table vit juste avant la table des blocs, une entree de 4 octets par ressource.
func (m *Module) resourceSlot(k int) (int, error) {
	n := m.resourceCount()
	if k < 0 || k >= n {
		return 0, fmt.Errorf("himodule: ressource %d hors des %d declarees", k, n)
	}
	base := m.blockOff - n*4
	if base < 0 || base+n*4 > len(m.data) {
		return 0, fmt.Errorf("himodule: table des ressources hors borne (base %d)", base)
	}
	return int(u32(m.data, base+k*4)), nil
}

// ResourceBlob rend la concatenation, dans l'ordre, des ressources d'une entree.
//
// C'est l'espace d'adressage des descripteurs de tampons : un descripteur qui dit
// « offset 4096, taille 512 » designe ces octets-la, pas un fichier a resoudre.
func (m *Module) ResourceBlob(f File) ([]byte, error) {
	debut := m.ResourceIndex(f.Index)
	fin := m.resourceCount()
	if f.Index+1 < m.fileCount {
		fin = m.ResourceIndex(f.Index + 1)
	}
	if debut > fin {
		return nil, fmt.Errorf("himodule: plage de ressources inversee pour l'entree %d (%d..%d)",
			f.Index, debut, fin)
	}
	var out []byte
	for k := debut; k < fin; k++ {
		slot, err := m.resourceSlot(k)
		if err != nil {
			return nil, err
		}
		if slot < 0 || slot >= m.fileCount {
			return nil, fmt.Errorf("himodule: slot de ressource %d hors des %d entrees", slot, m.fileCount)
		}
		buf, err := m.Extract(m.file(slot))
		if err != nil {
			return nil, fmt.Errorf("himodule: ressource %d (entree %d): %w", k, slot, err)
		}
		out = append(out, buf...)
	}
	return out, nil
}
