package main

// eqip_arbre.go — la STRUCTURE des evenements d'une banque d'equipement.
//
// CE QUE LES DEUX PASSES `eqip-sons`/`eqip-banks` PERDAIENT. `gestesDeBank` rend, pour un
// evenement, l'ENSEMBLE plat de ses `.wem` (`wemsDeEvent`). Un geste « 3 wem » peut donc
// etre UNE couche de trois variantes — le moteur en joue une — ou TROIS couches
// simultanees — le moteur les additionne. Ecouter les trois fichiers isoles ne tranche pas,
// et c'est exactement le retour de l'utilisateur du 2026-08-19 : aucun des `.wem` de la
// banque du translocateur ne ressemble au son du jeu.
//
// CE MODE-CI rend la structure : par evenement, ses COUCHES (`couchesDeEvent`, la recette
// prouvee du chantier armes), leur type de conteneur, leurs gains de chemin, leur delai.
// Il enumere TOUS les Events de la banque, pas seulement ceux qu'un `snd!` designe — la
// couverture (`.wem` embarques qu'aucun evenement n'atteint) est le chiffre qui dit si une
// ecoute portait sur toute la banque ou sur un tiers.
//
// MEMOIRE : il ouvre `pc/globals/globals-rtx-new.module` (7,24 Go). JAMAIS en parallele
// d'un autre gros chargement — meme regle que `eqip-banks`.

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"levelup/go-api/internal/himodule"
)

// evenementStructure : un evenement de la banque, avec la maniere dont il se joue.
type evenementStructure struct {
	Event string `json:"event"`
	// Designe : tags `snd!` de la passe 1 dont le corps contient cet identifiant. Vide =
	// evenement que la chaine `eqip -> effe -> snd!` ne designe pas ; il existe quand meme.
	Designe []string `json:"designe_par,omitempty"`
	// Nature : la phrase qui repond a « joues comment » — le seul champ que l'oreille lit.
	Nature  string          `json:"nature"`
	Couches []brancheRendue `json:"couches"`
	Wems    []uint32        `json:"wems"`
}

// banqueStructure : une banque, ses evenements, et ce que ses evenements laissent de cote.
type banqueStructure struct {
	Bank        string               `json:"bank"`
	Eqip        []string             `json:"eqip,omitempty"`
	Embarques   int                  `json:"wem_embarques"`
	NoeudsDelai int                  `json:"noeuds_avec_delai"`
	Evenements  []evenementStructure `json:"evenements"`
	// Orphelins : `.wem` embarques qu'AUCUN evenement de la banque n'atteint.
	Orphelins []uint32 `json:"wem_orphelins"`
}

// RapportStructure est la sortie du mode.
type RapportStructure struct {
	Module string            `json:"module"`
	Banks  []banqueStructure `json:"banks"`
}

// structureDesBanques est le mode `eqip-arbre`.
//
// `banques` : identifiants de `sbnk` a ouvrir. `entree` : rapport de la passe 1
// (`eqip-sons`), facultatif — il sert a dire quel `snd!` designe quel evenement et quels
// `eqip` atteignent la banque. `dossierEmb` : si non vide, TOUS les `.wem` embarques y sont
// ecrits (un sous-dossier par banque), pas seulement ceux des gestes designes.
func structureDesBanques(cheminModule string, banques []uint32, entree, sortie, dossierEmb string) error {
	if len(banques) == 0 {
		return fmt.Errorf("le mode eqip-arbre exige -banks (identifiants de sbnk, hexa, virgules)")
	}
	var parBank map[string][]string
	var sndsParBank map[string][]SndSon
	if entree != "" {
		rap, err := chargerEqipSons(entree)
		if err != nil {
			return err
		}
		parBank, sndsParBank = indexerPasse1(rap)
		fmt.Printf("passe 1 relue : %d eqip sonores, %d banks connues\n", len(rap.Equipement), len(parBank))
	}

	m, err := himodule.Open(cheminModule)
	if err != nil {
		return err
	}
	rapporterMemoire("module charge")

	out := RapportStructure{Module: cheminModule}
	for _, gid := range banques {
		hex := fmt.Sprintf("%08x", gid)
		b := structureDUneBanque(m, gid, parBank[hex], sndsParBank[hex], dossierEmb)
		out.Banks = append(out.Banks, b)
	}
	afficherStructure(out)
	if sortie == "" {
		return nil
	}
	j, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	fmt.Printf("\nstructure ecrite : %s\n", sortie)
	return os.WriteFile(sortie, j, 0o644)
}

// indexerPasse1 rend, depuis le rapport de `eqip-sons` : les `eqip` par banque (le
// denominateur de selectivite) et les `snd!` par banque (les designations d'evenements).
//
// Une seule construction pour les deux modes qui en ont besoin : la dupliquer les faisait
// deja diverger sur le dedoublonnage des `snd!`.
func indexerPasse1(rap RapportEqipSons) (map[string][]string, map[string][]SndSon) {
	parBank := map[string][]string{}
	snds := map[string][]SndSon{}
	vus := map[string]bool{}
	for _, e := range rap.Equipement {
		for _, b := range e.Banks {
			parBank[b] = append(parBank[b], e.Eqip)
		}
		for _, s := range e.Sons {
			for _, b := range s.Banks {
				if cle := b + "/" + s.Tag; !vus[cle] {
					vus[cle] = true
					snds[b] = append(snds[b], s)
				}
			}
		}
	}
	return parBank, snds
}

// structureDUneBanque ouvre une banque et rend tous ses evenements avec leurs couches.
func structureDUneBanque(
	m *himodule.Module, gid uint32, eqips []string, snds []SndSon, dossierEmb string,
) banqueStructure {
	hex := fmt.Sprintf("%08x", gid)
	r := banqueStructure{Bank: hex, Eqip: eqips}
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
	r.NoeudsDelai = len(bk.DelaiNoeud)
	atteints := map[uint32]bool{}
	for id := range bk.Events {
		ev := evenementDeBanque(bk, id, snds)
		if len(ev.Wems) == 0 {
			continue
		}
		for _, w := range ev.Wems {
			atteints[w] = true
		}
		r.Evenements = append(r.Evenements, ev)
	}
	sort.Slice(r.Evenements, func(i, j int) bool { return r.Evenements[i].Event < r.Evenements[j].Event })
	for id := range emb {
		if !atteints[id] {
			r.Orphelins = append(r.Orphelins, id)
		}
	}
	sort.Slice(r.Orphelins, func(i, j int) bool { return r.Orphelins[i] < r.Orphelins[j] })
	if dossierEmb != "" {
		ecrireTousEmbarques(ch, emb, hex, dossierEmb)
	}
	return r
}

// evenementDeBanque assemble la vue d'un evenement : ses couches, sa nature, ses `.wem`.
func evenementDeBanque(bk *bank, id uint32, snds []SndSon) evenementStructure {
	couches := bk.couchesDeEvent(id)
	ev := evenementStructure{
		Event:   fmt.Sprintf("%08x", id),
		Couches: couches,
		Wems:    bk.wemsDeEvent(id),
		Nature:  natureEvenement(couches),
	}
	for _, s := range snds {
		for _, mot := range s.Mots {
			if mot == id {
				ev.Designe = append(ev.Designe, s.Tag)
				break
			}
		}
	}
	sort.Strings(ev.Designe)
	return ev
}

// natureEvenement dit, en une phrase, comment l'evenement se joue.
//
// C'est la seule sortie que l'oreille lit, et elle ne doit rien affirmer que les couches
// ne portent : le nombre de couches vient du parcours, le nombre de variantes du point de
// choix, et « simultanees » de la regle prouvee (les actions d'un Event partent ensemble).
func natureEvenement(couches []brancheRendue) string {
	if len(couches) == 0 {
		return "aucune couche"
	}
	if len(couches) == 1 {
		c := couches[0]
		if len(c.Wems) == 1 {
			return "1 couche, 1 son"
		}
		return fmt.Sprintf("1 couche, UN son tire parmi %d (%s)", len(c.Wems), c.TypeNoeud)
	}
	var parts []string
	for _, c := range couches {
		if len(c.Wems) == 1 {
			parts = append(parts, "1 son")
			continue
		}
		parts = append(parts, fmt.Sprintf("1 parmi %d", len(c.Wems)))
	}
	return fmt.Sprintf("%d couches SIMULTANEES [%s]", len(couches), strings.Join(parts, " + "))
}

// ecrireTousEmbarques ecrit l'INTEGRALITE des medias d'une banque, orphelins compris.
//
// `ecrireGestes` (passe 2) ne sort que les `.wem` des gestes designes : c'est ce qui a fait
// ecouter 32 fichiers sur les 70 d'une banque sans que le compte soit dit.
func ecrireTousEmbarques(ch map[string][]byte, emb map[uint32][2]uint32, hex, racine string) {
	if len(emb) == 0 {
		return
	}
	dossier := racine + string(os.PathSeparator) + "sbnk_" + hex
	n, err := ecrireEmbarques(ch, emb, dossier)
	if err != nil {
		fmt.Printf("  sbnk %s : ecriture : %v\n", hex, err)
		return
	}
	fmt.Printf("  sbnk %s : %d .wem ecrits dans %s\n", hex, n, dossier)
}

// afficherStructure rend le tableau lisible : une banque, ses evenements, leur nature.
func afficherStructure(rap RapportStructure) {
	for _, b := range rap.Banks {
		fmt.Printf("\n=== sbnk %s : %d evenement(s), %d .wem embarques, %d orphelin(s), %d noeud(s) a delai ===\n",
			b.Bank, len(b.Evenements), b.Embarques, len(b.Orphelins), b.NoeudsDelai)
		if len(b.Eqip) > 0 {
			fmt.Printf("    atteinte par %d eqip : %s\n", len(b.Eqip), strings.Join(b.Eqip, " "))
		}
		for _, ev := range b.Evenements {
			designe := "(non designe)"
			if len(ev.Designe) > 0 {
				designe = "snd!:" + strings.Join(ev.Designe, ",")
			}
			fmt.Printf("  event %s  %-18s  %s\n", ev.Event, designe, ev.Nature)
			for _, c := range ev.Couches {
				fmt.Printf("      couche %s %-14s gains=%v delai=%.3fs wems=%v\n",
					c.Cible, c.TypeNoeud, c.Gains, c.DelaiS, c.Wems)
			}
		}
		if len(b.Orphelins) > 0 {
			fmt.Printf("    orphelins : %v\n", b.Orphelins)
		}
	}
}
