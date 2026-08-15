package main

// lot.go — ETAPE 4, PASSE 1 : les 55 `.pck` traites en UNE SEULE ouverture du module.
//
// POURQUOI PAS 55 APPELS AU MODE `map`. Ce module fait 7,24 Go et `himodule.Open` le lit
// entierement : l'ouvrir 55 fois serait 55 fois 7,24 Go de lecture disque. Une seule
// ouverture, deux sous-passes.
//
// MAITRISE DE LA MEMOIRE (contrainte explicite) :
//
//   - sous-passe A : chaque bank est decompressee, comptee, PUIS RELACHEE. On ne retient
//     qu'un entier par `.pck` (l'index de sa meilleure bank), jamais les octets.
//   - sous-passe B : on ne re-decompresse que les 55 banks retenues, une a la fois, et on
//     ne garde que le resultat d'analyse (des listes d'entiers), pas la bank.
//
// Jamais deux banks vivantes en meme temps. Le pic reste le module lui-meme.
//
// L'appariement `.pck` <-> bank n'utilise aucun nom : on balaie les octets de chaque bank
// et on compte, par `.pck`, combien de ses IDs `.wem` y figurent. La bank d'une arme est
// celle qui en contient le plus.

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"

	"levelup/go-api/internal/himodule"
)

type armeLot struct {
	Pck        string           `json:"pck"`
	Arme       string           `json:"arme"`
	SbnkGlobal uint32           `json:"sbnk_global_id"`
	WemsDuPck  int              `json:"wems_du_pck"`
	Evenements []evenementRendu `json:"evenements"`
}

type rapportLot struct {
	Armes []armeLot `json:"armes"`
}

// motifsPck : les familles de packs traitees (armes, tourelles, sifflements).
var motifsPck = []string{"sb_010_wea_*.pck", "sb_010_tur_*.pck", "sb_010_whizby_*.pck"}

// cartographierLot resout tous les `.pck` d'un dossier en une ouverture du module.
func cartographierLot(cheminModule, dossierPck, sortie string) error {
	chemins, err := listerPcks(dossierPck)
	if err != nil {
		return err
	}
	fmt.Printf("packs    : %d\n", len(chemins))

	idsPar := make([]map[uint32]bool, len(chemins))
	// index inverse : un ID .wem -> le pack qui le contient. Un seul balayage par bank
	// suffira, au lieu de 55 recherches successives.
	proprio := map[uint32]int{}
	for i, c := range chemins {
		set, err := idsPck(c)
		if err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(c), err)
		}
		idsPar[i] = set
		for id := range set {
			proprio[id] = i
		}
	}
	fmt.Printf("ids .wem : %d indexes\n", len(proprio))

	m, err := himodule.Open(cheminModule)
	if err != nil {
		return err
	}
	rapporterMemoire("module charge")

	banks := m.Files("sbnk")
	meilleur, score := associerBanks(m, banks, proprio, len(chemins))
	rapporterMemoire("sous-passe A terminee")

	rap := rapportLot{}
	for i, c := range chemins {
		if meilleur[i] < 0 {
			fmt.Printf("  [%2d/%2d] %-34s AUCUNE bank\n", i+1, len(chemins), nomArme(c))
			continue
		}
		f := banks[meilleur[i]]
		entree, err := analyserPourPck(m, f, idsPar[i], c)
		if err != nil {
			fmt.Printf("  [%2d/%2d] %-34s echec: %v\n", i+1, len(chemins), nomArme(c), err)
			continue
		}
		fmt.Printf("  [%2d/%2d] %-34s sbnk %08x, %2d events, %3d/%3d wem (score %d)\n",
			i+1, len(chemins), entree.Arme, entree.SbnkGlobal, len(entree.Evenements),
			couvertureWems(entree), entree.WemsDuPck, score[i])
		rap.Armes = append(rap.Armes, entree)
		if i%10 == 0 {
			runtime.GC()
		}
	}
	rapporterMemoire("sous-passe B terminee")

	blob, err := json.MarshalIndent(rap, "", " ")
	if err != nil {
		return err
	}
	fmt.Printf("\n%d armes resolues -> %s\n", len(rap.Armes), sortie)
	return os.WriteFile(sortie, blob, 0o644)
}

// listerPcks rend les chemins des packs a traiter, tries.
func listerPcks(dossier string) ([]string, error) {
	var out []string
	for _, motif := range motifsPck {
		trouves, err := filepath.Glob(filepath.Join(dossier, motif))
		if err != nil {
			return nil, err
		}
		out = append(out, trouves...)
	}
	sort.Strings(out)
	if len(out) == 0 {
		return nil, fmt.Errorf("aucun .pck dans %s", dossier)
	}
	return out, nil
}

// associerBanks : SOUS-PASSE A. Pour chaque bank, compte les IDs de chaque pack presents
// dans ses octets, puis RELACHE la bank. Ne retient que des entiers.
func associerBanks(m *himodule.Module, banks []himodule.File, proprio map[uint32]int, nPacks int) ([]int, []int) {
	meilleur := make([]int, nPacks)
	score := make([]int, nPacks)
	for i := range meilleur {
		meilleur[i] = -1
	}
	compte := make([]int, nPacks)
	for k, f := range banks {
		data, err := m.Extract(f)
		if err != nil {
			continue
		}
		for i := range compte {
			compte[i] = 0
		}
		// Balayage octet a octet : les IDs ne sont pas garantis alignes.
		for o := 0; o+4 <= len(data); o++ {
			if p, ok := proprio[binary.LittleEndian.Uint32(data[o:])]; ok {
				compte[p]++
			}
		}
		for i, n := range compte {
			if n > score[i] {
				score[i], meilleur[i] = n, k
			}
		}
		data = nil // la bank ne survit pas a l'iteration
		if k%200 == 0 && k > 0 {
			runtime.GC()
			rapporterMemoire(fmt.Sprintf("banks %d/%d", k, len(banks)))
		}
	}
	return meilleur, score
}

// analyserPourPck : SOUS-PASSE B. Decompresse UNE bank et n'en garde que l'analyse.
func analyserPourPck(m *himodule.Module, f himodule.File, ids map[uint32]bool, cheminPck string) (armeLot, error) {
	var out armeLot
	data, err := m.Extract(f)
	if err != nil {
		return out, err
	}
	debut := indexBKHD(data)
	if debut < 0 {
		return out, fmt.Errorf("chunk BKHD absent")
	}
	b, err := parserBank(data[debut:], func(id uint32) bool { return ids[id] })
	if err != nil {
		return out, err
	}
	out = armeLot{
		Pck: cheminPck, Arme: nomArme(cheminPck),
		SbnkGlobal: f.GlobalID, WemsDuPck: len(ids),
	}
	for id := range b.Events {
		w := b.wemsDeEvent(id)
		if len(w) == 0 {
			continue
		}
		out.Evenements = append(out.Evenements, evenementRendu{IDEvent: id, Nombre: len(w), Wems: w})
	}
	sort.Slice(out.Evenements, func(i, j int) bool {
		return out.Evenements[i].Nombre > out.Evenements[j].Nombre
	})
	return out, nil
}

// couvertureWems compte les `.wem` distincts atteints depuis un evenement.
func couvertureWems(a armeLot) int {
	set := map[uint32]bool{}
	for _, e := range a.Evenements {
		for _, w := range e.Wems {
			set[w] = true
		}
	}
	return len(set)
}
