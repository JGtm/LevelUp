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
//
// BALAYAGE STRUCTUREL (`-banks all`, lot L6, phase 5.1 de `PLAN_BALISE_MIX_WWISE.md`). La
// chaine `eqip -> effe -> snd! -> sbnk` n'atteint que 17 des 1305 banques du module (1,3 %) :
// si le son cherche vit ailleurs, aucune liste explicite ne le trouvera. `-banks all` enumere
// TOUTES les banques `sbnk` (precedent : `probe.go`/`sonder`, `audit.go`/`auditFormat`, qui
// balaient deja `m.Files("sbnk")` sans liste) et applique EXACTEMENT la meme structuration
// qu'une banque nommee — aucun parseur nouveau, seul le denominateur change. Estimation du
// plan : 20 a 30 minutes en un seul processus une fois le module charge (~2 min, ~7,2 Go).

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

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
	// Conditionnelles : les sous-arbres que les filtres du parcours ECARTENT, chacun avec
	// la condition sous laquelle le jeu les joue. Ce ne sont pas des dechets : ce sont les
	// PHASES du geste que le rendu au point de reference ne peut pas servir.
	Conditionnelles []varianteConditionnelle `json:"variantes_conditionnelles,omitempty"`
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
// `banques` : identifiants de `sbnk` a ouvrir ; ignore si `toutes`. `entree` : rapport de la
// passe 1 (`eqip-sons`), facultatif — il sert a dire quel `snd!` designe quel evenement et
// quels `eqip` atteignent la banque. `dossierEmb` : si non vide, TOUS les `.wem` embarques y
// sont ecrits (un sous-dossier par banque), pas seulement ceux des gestes designes. `toutes` :
// `-banks all` — `banques` est ALORS remplace par l'enumeration complete du module
// (`tousLesGidsSbnk`), et la sortie console bascule du tableau detaille (illisible sur 1305
// banques) a une progression compacte, cf. `afficherProgres`.
func structureDesBanques(cheminModule string, banques []uint32, entree, sortie, dossierEmb string, toutes bool) error {
	if !toutes && len(banques) == 0 {
		return fmt.Errorf(`le mode eqip-arbre exige -banks (identifiants de sbnk, hexa, virgules, ou "all")`)
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

	if toutes {
		banques = tousLesGidsSbnk(m)
		fmt.Printf("balayage structurel : %d banques sbnk (toutes, hors liste explicite)\n", len(banques))
	}

	out := RapportStructure{Module: cheminModule}
	debut := time.Now()
	totalEvenements := 0
	for i, gid := range banques {
		hex := fmt.Sprintf("%08x", gid)
		b := structureDUneBanque(m, gid, parBank[hex], sndsParBank[hex], dossierEmb)
		out.Banks = append(out.Banks, b)
		totalEvenements += len(b.Evenements)
		if toutes {
			afficherProgres(i+1, len(banques), totalEvenements, debut)
		}
	}
	if toutes {
		fmt.Printf("\nbalayage termine : %d banques, %d evenements, en %s\n",
			len(banques), totalEvenements, time.Since(debut).Round(time.Second))
	} else {
		afficherStructure(out)
	}
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

// tousLesGidsSbnk enumere TOUS les tags `sbnk` du module, tries par identifiant croissant —
// determinisme : deux executions du balayage traitent les banques dans le meme ordre.
//
// Precedent qui valide l'appel : `probe.go` (`sonder`) et `audit.go` (`auditFormat`)
// enumerent deja `m.Files("sbnk")` sans liste explicite pour parcourir le module entier.
func tousLesGidsSbnk(m *himodule.Module) []uint32 {
	fichiers := m.Files("sbnk")
	gids := make([]uint32, 0, len(fichiers))
	for _, f := range fichiers {
		gids = append(gids, f.GlobalID)
	}
	sort.Slice(gids, func(i, j int) bool { return gids[i] < gids[j] })
	return gids
}

// afficherProgres imprime un point d'avancement toutes les 50 banques (et sur la derniere).
// Le balayage complet est estime a 20-30 minutes (plan, phase 5.1) : un processus muet tout
// ce temps ne permet pas de distinguer un traitement normal d'un blocage.
func afficherProgres(fait, total, evenements int, debut time.Time) {
	if fait%50 != 0 && fait != total {
		return
	}
	ecoule := time.Since(debut).Round(time.Second)
	fmt.Printf("  progression : %d/%d banques, %d evenements cumules (%s ecoule)\n",
		fait, total, evenements, ecoule)
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
	cond := bk.variantesConditionnelles(id)
	ev := evenementStructure{
		Event:           fmt.Sprintf("%08x", id),
		Couches:         couches,
		Wems:            bk.wemsDeEvent(id),
		Nature:          natureEvenement(couches),
		Conditionnelles: cond,
	}
	if len(cond) > 0 {
		ev.Nature += fmt.Sprintf(" + %d PHASE(S) SOUS CONDITION", len(cond))
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
		return "1 couche, " + descriptionCouche(couches[0])
	}
	var parts []string
	enchaine := false
	for _, c := range couches {
		if c.DelaiS > 0 {
			enchaine = true
		}
		parts = append(parts, descriptionCouche(c))
	}
	liaison := "SIMULTANEES"
	if enchaine {
		liaison = "ENCHAINEES"
	}
	return fmt.Sprintf("%d couches %s [%s]", len(couches), liaison, strings.Join(parts, " + "))
}

// descriptionCouche dit ce qu'une couche joue, et ce qu'elle en fait : combien de variantes,
// a quel instant elle demarre, si elle se repete. Rien qui ne soit porte par la couche.
func descriptionCouche(c brancheRendue) string {
	base := "1 son"
	if len(c.Wems) != 1 {
		base = fmt.Sprintf("1 parmi %d", len(c.Wems))
	}
	if c.DelaiS > 0 {
		base += fmt.Sprintf(" a +%.2fs", c.DelaiS)
	}
	if c.Repetitions != nil {
		if *c.Repetitions == 0 {
			base += " EN BOUCLE"
		} else {
			base += fmt.Sprintf(" x%d", *c.Repetitions)
		}
		base += " " + nomEnchainement(c.ModeEnchainement, c.TransitionS)
	}
	return base
}

// nomEnchainement dit, en clair, comment les lectures successives se suivent.
func nomEnchainement(mode int, transitionS float32) string {
	switch mode {
	case transitionCadence:
		return fmt.Sprintf("(toutes les %.2fs)", transitionS)
	case transitionDelai:
		return fmt.Sprintf("(silence de %.2fs entre)", transitionS)
	case transitionFonduAmp, transitionFonduPuiss:
		return fmt.Sprintf("(fondu enchaine %.2fs)", transitionS)
	default:
		return "(bout a bout)"
	}
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
