package main

// chaines.go — le mode `chaines` : LES CHAINES LISIBLES D'UN TAG, et son en-tete en octets.
//
// POURQUOI CE MODE EXISTE. Les modules du jeu ont `stringsSize = 0` : aucun nom de tag n'y
// survit, et c'est ce qui a rendu tout le chantier sonore dependant du hachage. MAIS un tag de
// SCRIPT (`hsc*`) n'est pas un tag de donnees : un script compile garde souvent les noms de ses
// fonctions, de ses variables globales et de ses constantes de chaine, la ou une table binaire
// n'a rien a garder.
//
// C'EST LA SONDE S1 DU HANDOFF DU 2026-08-27, et la seule piste de NOMMAGE encore ouverte pour
// les sons de zone. La remontee depuis la banque `1c609526` la designe sans ambiguite :
//
//	niveau 1 : 60 `snd!` + 20 `lsnd`
//	niveau 2 : 58 `sgrp` + 3 `effe` + 1 `hsc*`   <- a35c6ce9, LE script du chemin
//	niveau 3 : 6 `hsc*` + 1 `sdzg` + 1 `weap`
//
// CE QUE LA SORTIE MONTRE, DANS CET ORDRE IMPOSE : d'abord la TAILLE et les premiers octets
// (si le tag est compresse ou chiffre, ca se voit la et rien de ce qui suit ne vaut) ; ensuite
// seulement les chaines. Une sonde qui listerait des chaines sans montrer qu'elle a bien lu du
// texte ferait passer du bruit pour un resultat.

import (
	"fmt"
	"sort"
	"strings"

	"levelup/go-api/internal/himodule"
)

// longueurMinChaine : en dessous, une suite d'octets imprimables est du hasard. Quatre
// caracteres est la convention de `strings(1)`.
const longueurMinChaine = 4

// extraireChaines est le mode `chaines`.
func extraireChaines(cheminModule string, cibles []uint32) error {
	m, err := himodule.Open(cheminModule)
	if err != nil {
		return err
	}
	rapporterMemoire("module charge")
	voulu := map[uint32]bool{}
	for _, id := range cibles {
		voulu[id] = true
	}

	trouves := 0
	for _, f := range m.Files("") {
		if !voulu[f.GlobalID] {
			continue
		}
		trouves++
		data, err := m.Extract(f)
		if err != nil {
			fmt.Printf("\n=== tag %08x (%s) : extraction impossible : %v\n", f.GlobalID, f.Group, err)
			continue
		}
		fmt.Printf("\n=== tag %08x, groupe '%s', %d octets ===\n", f.GlobalID, f.Group, len(data))
		fmt.Println("  -- les 64 premiers octets, avant toute interpretation --")
		for o := 0; o < 64 && o < len(data); o += 16 {
			fin := o + 16
			if fin > len(data) {
				fin = len(data)
			}
			fmt.Printf("    +%04x  % x   %s\n", o, data[o:fin], imprimable(data[o:fin]))
		}
		chaines := chainesLisibles(data)
		fmt.Printf("  -- %d chaine(s) de %d caracteres ou plus --\n", len(chaines), longueurMinChaine)
		for _, s := range chaines {
			fmt.Printf("    %s\n", s)
		}
		if len(chaines) == 0 {
			fmt.Println("    (aucune) — ce tag ne porte aucun texte : le nommage ne viendra pas d'ici.")
		}
	}
	if trouves == 0 {
		fmt.Println("aucun des identifiants demandes n'existe dans ce module")
	}
	return nil
}

// chainesLisibles rend les suites de caracteres imprimables du tag, dedoublonnees et triees.
func chainesLisibles(data []byte) []string {
	set := map[string]bool{}
	var courante []byte
	fermer := func() {
		if len(courante) >= longueurMinChaine {
			set[string(courante)] = true
		}
		courante = courante[:0]
	}
	for _, b := range data {
		if b >= 0x20 && b < 0x7f {
			courante = append(courante, b)
			continue
		}
		fermer()
	}
	fermer()
	out := make([]string, 0, len(set))
	for s := range set {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// imprimable rend la vue texte d'une tranche d'octets, les non-imprimables en points.
func imprimable(b []byte) string {
	var sb strings.Builder
	for _, c := range b {
		if c >= 0x20 && c < 0x7f {
			sb.WriteByte(c)
			continue
		}
		sb.WriteByte('.')
	}
	return sb.String()
}
