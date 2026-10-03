package main

// weaptags.go — LA CLE UNIVERSELLE DE L ICONE : le tag `weap`, pas un nom, pas une cle produit.
//
// CE QUE ÇA REMPLACE. La resolution d image du produit est aujourd hui keyee par
// `weapon_labels.name_en` — dernier reste du domaine arme keye par un nom EN brut, et il DIVERGE
// deja de la source canonique (« Mk51 Sidekick » contre « Mk50 Sidekick »). Le remplacer par le
// `weapon_key` du registre paraissait l evidence, mais la mesure la refute : **7 etiquettes sur
// 42 n ont pas de weapon_key**, dont MA5K Avenger et Mutilator qui ONT une icone aujourd hui.
//
// LA VRAIE CLE ETAIT DEJA DOCUMENTEE (§1.1 de l etat de l art) : les **32 bits HAUTS** d un
// identifiant filmshell sont le global tag id du `weap`. Verifie sur les donnees reelles :
//
//	Mutilator        0xd7915565_42c9679f -> tag d7915565 -> index 37
//	Sandwich         0x880fe0bc_42c9679f -> tag 880fe0bc -> index 35
//	Mythic Sandwich  0xb7262ca1_c8fb11d0 -> tag b7262ca1 -> index 35
//	MA5K Avenger     0xf5c335df_e7232c0b -> tag f5c335df -> index 36
//
// Elle ne depend ni d un nom, ni du registre, ni d une table produit : elle couvre les armes
// enregistrees, leurs variantes, ET celles qui ne sont pas au registre. Seules les sentinelles
// (weapon_id 0/1/2 = grenade / melee / vehicule generiques) restent hors de sa portee — ce sont
// des concepts produit, pas des tags du jeu.
//
// POURQUOI TOUS LES GROUPES. Le bloc `UI display info` n est pas propre au `weap` : c est une
// structure partagee du systeme de tags. On balaie donc tous les groupes, l auto-validation par
// l atlas servant de filtre — un bloc n est retenu que si son champ `sprite` porte un atlas
// connu. C est ce balayage qui a fait sortir l index 29 (revendique hors `weap`).

import (
	"fmt"
	"sort"
)

// weapTagsByIndex rend, par index d'atlas, les tags qui le revendiquent — tries, dedoublonnes.
func weapTagsByIndex(ix *tagIndex) (map[int][]string, error) {
	fields, err := loadWeapPlugin()
	if err != nil {
		return nil, err
	}
	offSprite, offIndex, _, ok := uiBlockOffsets(fields)
	if !ok {
		return nil, fmt.Errorf("champs `sprite` / `sprite index` absents du plugin")
	}
	vus := map[uint32]bool{}
	parIndex := map[int]map[string]bool{}
	for groupe, refs := range ix.byGroup {
		if groupe == "bitm" {
			continue // les bitmaps ne portent pas de bloc d'interface
		}
		for _, r := range refs {
			if vus[r.ID] {
				continue
			}
			vus[r.ID] = true
			data, err := ix.extract(r)
			if err != nil {
				continue
			}
			wt, err := openWeapTag(data)
			if err != nil {
				continue
			}
			abs, _, _, found := findUIBlock(wt, data, offSprite)
			if !found {
				continue
			}
			idx := int(readU32LE(data, abs+offIndex))
			if idx < 0 || idx >= maxAtlasIndex {
				continue
			}
			if parIndex[idx] == nil {
				parIndex[idx] = map[string]bool{}
			}
			parIndex[idx][fmt.Sprintf("%08x", r.ID)] = true
		}
	}
	out := make(map[int][]string, len(parIndex))
	for idx, set := range parIndex {
		tags := make([]string, 0, len(set))
		for t := range set {
			tags = append(tags, t)
		}
		sort.Strings(tags)
		out[idx] = tags
	}
	return out, nil
}

// maxAtlasIndex borne les index retenus : au-dela, la valeur lue n'est pas un index d'atlas
// mais un champ d'un bloc qui a passe le filtre par hasard.
const maxAtlasIndex = 128
