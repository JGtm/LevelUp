package main

// probe.go — ETAPE 1 du plan : statuer sur le format REEL de la charge utile d'un `sbnk`.
//
// La question a trancher est binaire : le contenu decompresse est-il une bank Wwise
// verbatim (chunks `BKHD`/`HIRC`) ou une structure maison ? Le reste du plan en depend :
// avec `HIRC` on remonte Evenement -> Conteneur -> Son -> `.wem` ; sans lui on se rabat
// sur un regroupement par structure interne.
//
// La sonde ne postule rien : elle decompresse, cherche les signatures, et cherche les
// `.wem` temoins d'une arme connue pour pouvoir DESIGNER le sbnk de cette arme sans nom.

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sort"

	"levelup/go-api/internal/himodule"
)

// tagMagicUCSH est le magic d'un fichier de tag Halo Infinite ('ucsh' en little-endian).
const tagMagicUCSH = 0x68736375

// signaturesWwise : chunks d'une bank Wwise standard.
var signaturesWwise = []string{"BKHD", "HIRC", "DIDX", "DATA", "STID", "STMG", "ENVS"}

type resultatBank struct {
	GlobalID uint32
	Taille   int
	TailleR  int // taille du blob de ressources lie
	UCSH     bool
	Chunks   []string // signatures vues dans le tag OU dans ses ressources
	DansRes  bool     // les signatures viennent du blob de ressources
	Temoins  []uint32
	TemoinsR bool // les temoins viennent du blob de ressources
}

// sonder decompresse les `sbnk` du module et rend le verdict de l'etape 1.
func sonder(chemin string, temoins []uint32, limite int) error {
	fmt.Printf("module   : %s\n", chemin)
	m, err := himodule.Open(chemin)
	if err != nil {
		return err
	}
	banks := m.Files("sbnk")
	fmt.Printf("tags sbnk: %d\n", len(banks))
	if len(banks) == 0 {
		return fmt.Errorf("aucun tag sbnk dans ce module")
	}

	// Les plus petits d'abord : une bank d'arme ne porte que des metadonnees (l'audio vit
	// dans le `.pck`), donc elle est petite. Les gros sbnk sont la musique et l'ambiance.
	sort.Slice(banks, func(i, j int) bool { return banks[i].UncompSize < banks[j].UncompSize })
	if limite <= 0 || limite > len(banks) {
		limite = len(banks)
	}

	var res []resultatBank
	var echecs int
	for _, f := range banks[:limite] {
		data, err := m.Extract(f)
		if err != nil {
			echecs++
			continue
		}
		// La donnee utile d'un tag n'est pas forcement dans ses octets : les descripteurs
		// pointent souvent dans le BLOB DE RESSOURCES. On sonde donc les deux espaces.
		blob, errRes := m.ResourceBlob(f)
		if errRes != nil {
			blob = nil
		}
		res = append(res, analyserBank(f.GlobalID, data, blob, temoins))
	}
	afficherVerdict(res, echecs, limite)
	return nil
}

// analyserBank cherche les signatures Wwise et les `.wem` temoins dans le tag et son blob.
func analyserBank(gid uint32, data, blob []byte, temoins []uint32) resultatBank {
	r := resultatBank{GlobalID: gid, Taille: len(data), TailleR: len(blob)}
	if len(data) >= 4 && binary.LittleEndian.Uint32(data) == tagMagicUCSH {
		r.UCSH = true
	}
	for _, s := range signaturesWwise {
		switch {
		case bytes.Contains(data, []byte(s)):
			r.Chunks = append(r.Chunks, s)
		case bytes.Contains(blob, []byte(s)):
			r.Chunks = append(r.Chunks, s)
			r.DansRes = true
		}
	}
	var aiguille [4]byte
	for _, t := range temoins {
		binary.LittleEndian.PutUint32(aiguille[:], t)
		switch {
		case bytes.Contains(data, aiguille[:]):
			r.Temoins = append(r.Temoins, t)
		case bytes.Contains(blob, aiguille[:]):
			r.Temoins = append(r.Temoins, t)
			r.TemoinsR = true
		}
	}
	return r
}

// afficherVerdict rend compte de la sonde et statue OUI/NON sur le format Wwise.
func afficherVerdict(res []resultatBank, echecs, limite int) {
	fmt.Printf("sondes   : %d decompresses, %d echecs (sur %d demandes)\n\n", len(res), echecs, limite)

	compte := map[string]int{}
	var avecUCSH, avecTemoin int
	for _, r := range res {
		for _, c := range r.Chunks {
			compte[c]++
		}
		if r.UCSH {
			avecUCSH++
		}
		if len(r.Temoins) > 0 {
			avecTemoin++
		}
	}

	fmt.Println("--- signatures rencontrees ---")
	if len(compte) == 0 {
		fmt.Println("  aucune signature Wwise")
	}
	for _, s := range signaturesWwise {
		if n := compte[s]; n > 0 {
			fmt.Printf("  %-5s present dans %d / %d tags\n", s, n, len(res))
		}
	}
	fmt.Printf("\nen-tete ucsh : %d / %d\n", avecUCSH, len(res))
	fmt.Printf("porteurs de .wem temoins : %d\n", avecTemoin)

	fmt.Println("\n--- tags portant les temoins ---")
	for _, r := range res {
		if len(r.Temoins) == 0 {
			continue
		}
		fmt.Printf("  gid %08x  tag %7d o  res %9d o  chunks=%v  temoins=%v (dans ressources=%v)\n",
			r.GlobalID, r.Taille, r.TailleR, r.Chunks, r.Temoins, r.TemoinsR)
	}

	fmt.Println("\n=== VERDICT ETAPE 1 ===")
	if compte["HIRC"] > 0 {
		fmt.Println("HIRC present : hierarchie Wwise exploitable, etape 2 nominale.")
		return
	}
	fmt.Println("HIRC ABSENT sur cet echantillon : bifurcation prevue au plan")
	fmt.Println("(regroupement des .wem par structure interne du sbnk).")
}
