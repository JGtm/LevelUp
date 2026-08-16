package main

// stid.go — lecture du chunk `STID` : la seule source de VRAIS noms dans les banks.
//
// POURQUOI. L'etape 3 doit retrouver des noms d'evenements par hachage, ce qui suppose de
// connaitre la CONVENTION de nommage. Le calibrage sur l'ID de bank prouve que la fonction
// est bien FNV-1 sur le nom complet en minuscules, mais ne dit rien de la forme des noms
// d'evenements. `STID` (present sur 2 banks sur 1305) porte des noms en clair : c'est
// l'echantillon qui permet de deduire la convention au lieu de la deviner.
//
// Format : u32 inconnu | u32 nombre | par entree { u32 idBank, u8 longueur, octets nom }.

import (
	"encoding/binary"
	"fmt"

	"levelup/go-api/internal/himodule"
)

// nomsSTID decode les couples {identifiant, nom} d'un chunk STID.
func nomsSTID(stid []byte) map[uint32]string {
	out := map[uint32]string{}
	if len(stid) < 8 {
		return out
	}
	n := int(binary.LittleEndian.Uint32(stid[4:]))
	off := 8
	for i := 0; i < n; i++ {
		if off+5 > len(stid) {
			break
		}
		id := binary.LittleEndian.Uint32(stid[off:])
		longueur := int(stid[off+4])
		debut := off + 5
		if debut+longueur > len(stid) {
			break
		}
		out[id] = string(stid[debut : debut+longueur])
		off = debut + longueur
	}
	return out
}

// listerNoms parcourt les `sbnk` du module et rend tous les noms lisibles trouves,
// en verifiant a chaque fois que le nom hache bien vers l'identifiant annonce.
func listerNoms(cheminModule string) error {
	m, err := himodule.Open(cheminModule)
	if err != nil {
		return err
	}
	banks := m.Files("sbnk")
	fmt.Printf("module : %d tags sbnk\n\n", len(banks))

	total, coherents := 0, 0
	for _, f := range banks {
		data, err := m.Extract(f)
		if err != nil {
			continue
		}
		stid, ok := chunks(data)["STID"]
		if !ok {
			continue
		}
		noms := nomsSTID(stid)
		if len(noms) == 0 {
			continue
		}
		fmt.Printf("--- sbnk gid %08x : %d noms ---\n", f.GlobalID, len(noms))
		affiches := 0
		for id, nom := range noms {
			total++
			marque := "  "
			if fnv1(nom) == id {
				coherents++
				marque = "ok"
			}
			if affiches < 40 {
				fmt.Printf("  %s %08x  %s\n", marque, id, nom)
				affiches++
			}
		}
		if len(noms) > 40 {
			fmt.Printf("  ... et %d autres\n", len(noms)-40)
		}
	}
	fmt.Printf("\nnoms lus : %d, dont %d dont le hachage FNV-1 redonne l'identifiant\n",
		total, coherents)
	if total > 0 && coherents == 0 {
		fmt.Println("ATTENTION : aucun nom ne se re-hache — la convention n'est pas celle supposee")
	}
	return nil
}
