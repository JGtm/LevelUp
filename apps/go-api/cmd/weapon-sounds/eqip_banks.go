package main

// eqip_banks.go — PASSE 2 de la chaine son d'equipement : `sbnk` -> `.wem` -> PACK NOMME.
//
// CE QU'ELLE ETABLIT. Les tags n'ont pas de nom (`stringsSize` = 0 sur tous les modules),
// mais les 841 `.pck` du jeu en ont un en clair — `sb_007_abl_repairfield.pck`. Faire tomber
// les `.wem` d'une bank dans un pack nomme, c'est donc NOMMER la bank, et par elle
// l'equipement qui l'atteint.
//
// LE TEMOIN EST LA CONDITION, PAS UN SUPPLEMENT (decision 2c du plan). Une bank atteinte par
// vingt et un objets d'equipement ne nomme rien : c'est la bank commune du systeme. Seule
// une bank atteinte par UN SEUL `eqip` peut porter un nom, et la passe 1 rend precisement ce
// denominateur (`--- banks atteintes, et QUI les atteint ---`).
//
// MEMOIRE : ce mode ouvre `pc/globals/globals-rtx-new.module` (7,24 Go). JAMAIS en meme temps
// qu'un autre gros chargement — c'est toute la raison du decoupage en deux passes.

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"levelup/go-api/internal/himodule"
)

// BankResolue est ce qu'une bank rend une fois ses medias rattaches aux packs nommes.
type BankResolue struct {
	Bank      string         `json:"bank"`
	Embarques int            `json:"wem_embarques"`
	Sons      int            `json:"wem_references"`
	Packs     map[string]int `json:"packs"`
	Eqip      []string       `json:"eqip"`
}

// RapportEqipBanks est la sortie de la passe 2.
type RapportEqipBanks struct {
	Module string        `json:"module"`
	Banks  []BankResolue `json:"banks"`
}

// banquesDEquipement est la PASSE 2 : elle lit le JSON de la passe 1 et nomme les banks.
func banquesDEquipement(cheminModule, entree, sortie string) error {
	rap, err := chargerEqipSons(entree)
	if err != nil {
		return err
	}
	parBank := map[string][]string{}
	for _, e := range rap.Equipement {
		for _, b := range e.Banks {
			parBank[b] = append(parBank[b], e.Eqip)
		}
	}
	fmt.Printf("passe 1 : %d eqip sonores, %d banks a resoudre\n", len(rap.Equipement), len(parBank))

	m, err := himodule.Open(cheminModule)
	if err != nil {
		return err
	}
	rapporterMemoire("module charge")

	out := RapportEqipBanks{Module: cheminModule}
	cles := trierCles(parBank)
	for _, b := range cles {
		r := resoudreBank(m, b, parBank[b])
		out.Banks = append(out.Banks, r)
	}
	afficherBanksResolues(out)
	if sortie == "" {
		return nil
	}
	j, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(sortie, j, 0o644)
}

// chargerEqipSons relit le rapport de la passe 1.
func chargerEqipSons(chemin string) (RapportEqipSons, error) {
	var rap RapportEqipSons
	if chemin == "" {
		return rap, fmt.Errorf("le mode eqip-banks exige -json (la sortie du mode eqip-sons)")
	}
	b, err := os.ReadFile(chemin)
	if err != nil {
		return rap, err
	}
	return rap, json.Unmarshal(b, &rap)
}

// resoudreBank rattache les `.wem` d'une bank aux packs nommes du jeu.
func resoudreBank(m *himodule.Module, hex string, eqips []string) BankResolue {
	r := BankResolue{Bank: hex, Packs: map[string]int{}, Eqip: eqips}
	var gid uint32
	if _, err := fmt.Sscanf(hex, "%08x", &gid); err != nil {
		return r
	}
	_, brut, err := bankParIdentifiant(m, gid)
	if err != nil {
		fmt.Printf("  sbnk %s : %v\n", hex, err)
		return r
	}
	ch := chunks(brut)
	emb := mediasEmbarques(ch)
	r.Embarques = len(emb)
	estWem := func(id uint32) bool {
		if _, ok := emb[id]; ok {
			return true
		}
		_, ok := indexLarge[id]
		return ok
	}
	bk, err := parserBank(brut, estWem)
	if err != nil {
		fmt.Printf("  sbnk %s : bank illisible : %v\n", hex, err)
		return r
	}
	vus := map[uint32]bool{}
	for _, wem := range bk.Sons {
		vus[wem] = true
	}
	for id := range emb {
		vus[id] = true
	}
	r.Sons = len(vus)
	for id := range vus {
		if chemin, ok := indexLarge[id]; ok {
			r.Packs[nomFichierSansExt(chemin)]++
		}
	}
	return r
}

// afficherBanksResolues rend le tableau final : une bank, son temoin, ses packs nommes.
func afficherBanksResolues(rap RapportEqipBanks) {
	fmt.Printf("\n--- %d banks resolues ---\n", len(rap.Banks))
	for _, b := range rap.Banks {
		noms := make([]string, 0, len(b.Packs))
		for p, n := range b.Packs {
			noms = append(noms, fmt.Sprintf("%s(%d)", p, n))
		}
		sort.Strings(noms)
		verdict := "PARTAGEE"
		if len(b.Eqip) == 1 {
			verdict = "SELECTIVE"
		}
		if len(noms) == 0 {
			noms = []string{"aucun pack nomme"}
		}
		fmt.Printf("  sbnk %s  %-9s  %2d eqip · %4d wem (%d embarques)  packs: %s\n",
			b.Bank, verdict, len(b.Eqip), b.Sons, b.Embarques, strings.Join(noms, " "))
	}
}

func trierCles(m map[string][]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
