package main

// audit.go — INVENTAIRE EXHAUSTIF du format, confronte a ce que le parseur lit vraiment.
//
// POURQUOI CE MODE EXISTE. Deux fois de suite, une specificite du format Wwise a ete
// manquee parce que le parseur n'implementait que le strict necessaire a l'objectif du
// moment : d'abord les medias embarques (`DIDX`/`DATA`, soit plus de la moitie des sons),
// ensuite le fait que « Weapon Fire Sounds » est un TABLEAU (un mode de tir par entree).
// Dans les deux cas le defaut etait invisible : le code rendait un resultat plausible.
//
// Ce mode inverse la demarche. Il ENUMERE ce que les banks contiennent — types de chunks,
// types d'objets HIRC, types d'Action, presence de proprietes — et affiche en regard ce que
// le parseur consomme et ce qu'il IGNORE. Un trou devient visible sans qu'il faille le
// soupconner d'abord.

import (
	"encoding/binary"
	"fmt"
	"sort"

	"levelup/go-api/internal/himodule"
)

// chunksLus : les chunks que `parserBank` exploite reellement aujourd'hui.
var chunksLus = map[string]string{
	"BKHD": "en-tete, identifiant de bank",
	"HIRC": "hierarchie (objets)",
	"DIDX": "index des medias embarques",
	"DATA": "octets des medias embarques",
}

// typesLus : les types d'objets HIRC que `parserBank` traite specifiquement. Tous les
// autres tombent dans la branche par defaut, qui ne cherche qu'une liste d'enfants.
var typesLus = map[byte]string{
	typeSound:  "sourceID lu",
	typeAction: "cible lue (mais PAS le type d'action)",
	typeEvent:  "liste d'actions lue",
}

// nomsActions : nomenclature des types d'Action Wwise. Le type est un u16 en tete de la
// charge utile — que le parseur ne lit PAS aujourd'hui, alors qu'il conditionne le sens de
// la cible : empiler la cible d'un `Stop` comme une couche produirait un mixage faux.
var nomsActions = map[uint16]string{
	0x0100: "Stop", 0x0200: "Pause", 0x0300: "Resume", 0x0400: "Play",
	0x0500: "Trigger", 0x0600: "Mute", 0x0700: "UnMute", 0x0800: "SetVolume",
	0x0900: "SetPitch", 0x0A00: "SetLPF", 0x0B00: "SetState", 0x0C00: "SetSwitch",
	0x0D00: "SetRTPC", 0x0E00: "SetGameParameter", 0x0F00: "UseState",
	0x1000: "Bypass", 0x1100: "ResetBypass", 0x1200: "Break", 0x1300: "Seek",
}

func nomAction(t uint16) string {
	if n, ok := nomsActions[t&0xFF00]; ok {
		return n
	}
	return fmt.Sprintf("inconnu(%04x)", t)
}

// auditFormat parcourt les banks et rend l'inventaire complet.
func auditFormat(cheminModule string, limite int) error {
	m, err := himodule.Open(cheminModule)
	if err != nil {
		return err
	}
	rapporterMemoire("module charge")
	banks := m.Files("sbnk")
	if limite <= 0 || limite > len(banks) {
		limite = len(banks)
	}
	fmt.Printf("banks examinees : %d sur %d\n\n", limite, len(banks))

	chunks_ := map[string]int{}
	types := map[byte]int{}
	actions := map[uint16]int{}
	// Taille de charge utile d'un Sound : si elle depasse largement les 13 octets lus,
	// c'est qu'il reste des donnees non exploitees (proprietes, effets, positionnement).
	var sommeSound, nSound, maxSound int
	var propsLues, avecVolume, avecDelai, avecPitch int
	sommeVol := 0.0
	minVol := 0.0
	var sansHIRC, sansBKHD int
	var nActionsProps, nActionsDelai int
	conteneurs := nouvellesStatsConteneurs()

	for i, f := range banks[:limite] {
		data, err := m.Extract(f)
		if err != nil {
			continue
		}
		debut := indexBKHD(data)
		if debut < 0 {
			sansBKHD++
			continue
		}
		ch := chunks(data[debut:])
		for nom := range ch {
			chunks_[nom]++
		}
		h, ok := ch["HIRC"]
		if !ok {
			sansHIRC++
			continue
		}
		objs, err := objetsHIRC(h)
		if err != nil {
			continue
		}
		// `connu` doit porter sur LA bank courante : un identifiant d'enfant ne se valide
		// que contre les objets de sa propre bank.
		presents := make(map[uint32]bool, len(objs))
		for _, o := range objs {
			presents[o.ID] = true
		}
		connu := func(id uint32) bool { return presents[id] }
		for _, o := range objs {
			types[o.Type]++
			switch o.Type {
			case typeSound:
				nSound++
				sommeSound += len(o.Data)
				if len(o.Data) > maxSound {
					maxSound = len(o.Data)
				}
				if p := lireProprietes(o.Data); p.Lu {
					propsLues++
					if p.VolumeDB != 0 {
						avecVolume++
						sommeVol += float64(p.VolumeDB)
						if float64(p.VolumeDB) < minVol {
							minVol = float64(p.VolumeDB)
						}
					}
					if p.DelaiS != 0 {
						avecDelai++
					}
					if p.PitchCts != 0 {
						avecPitch++
					}
				}
			case typeAction:
				if len(o.Data) >= 2 {
					actions[binary.LittleEndian.Uint16(o.Data)]++
				}
			case typeEvent:
			default:
				conteneurs.ajouter(o, connu)
			}
		}
		if i%300 == 0 && i > 0 {
			rapporterMemoire(fmt.Sprintf("banks %d/%d", i, limite))
		}
	}

	afficherInventaire(chunks_, types, actions, nSound, sommeSound, maxSound, sansBKHD, sansHIRC)
	conteneurs.afficher()
	fmt.Printf("\n=== ACTIONS Play : proprietes ===\n")
	fmt.Printf("  paquet de proprietes lisible : %d | avec un delai non nul : %d\n",
		nActionsProps, nActionsDelai)
	fmt.Printf("\n=== PROPRIETES DES OBJETS Sound (nouveau lecteur) ===\n")
	fmt.Printf("  paquet lu de facon plausible : %d / %d (%.0f %%)\n",
		propsLues, nSound, 100*float64(propsLues)/float64(max(nSound, 1)))
	fmt.Printf("  avec un volume non nul     : %d (moyenne %.1f dB, minimum %.1f dB)\n",
		avecVolume, sommeVol/float64(max(avecVolume, 1)), minVol)
	fmt.Printf("  avec un delai non nul      : %d\n", avecDelai)
	fmt.Printf("  avec une hauteur non nulle : %d\n", avecPitch)
	return nil
}

func afficherInventaire(chunks_ map[string]int, types map[byte]int, actions map[uint16]int,
	nSound, sommeSound, maxSound, sansBKHD, sansHIRC int) {
	fmt.Println("=== CHUNKS presents, et ce qu'on en fait ===")
	nomsCh := make([]string, 0, len(chunks_))
	for n := range chunks_ {
		nomsCh = append(nomsCh, n)
	}
	sort.Slice(nomsCh, func(i, j int) bool { return chunks_[nomsCh[i]] > chunks_[nomsCh[j]] })
	for _, n := range nomsCh {
		usage, lu := chunksLus[n]
		marque := "IGNORE"
		if lu {
			marque = "lu : " + usage
		}
		fmt.Printf("  %-6s %5d bank(s)   %s\n", n, chunks_[n], marque)
	}
	if sansBKHD > 0 || sansHIRC > 0 {
		fmt.Printf("  (%d sans BKHD, %d sans HIRC)\n", sansBKHD, sansHIRC)
	}

	fmt.Println("\n=== TYPES D'OBJETS HIRC, et ce qu'on en fait ===")
	tps := make([]byte, 0, len(types))
	for t := range types {
		tps = append(tps, t)
	}
	sort.Slice(tps, func(i, j int) bool { return types[tps[i]] > types[tps[j]] })
	for _, t := range tps {
		marque := "enfants seulement (aucune propriete lue)"
		if u, ok := typesLus[t]; ok {
			marque = u
		}
		fmt.Printf("  %-16s %7d   %s\n", nomType(t), types[t], marque)
	}

	fmt.Println("\n=== TYPES D'ACTION (jamais lus par le parseur) ===")
	acts := make([]uint16, 0, len(actions))
	for a := range actions {
		acts = append(acts, a)
	}
	sort.Slice(acts, func(i, j int) bool { return actions[acts[i]] > actions[acts[j]] })
	for _, a := range acts {
		fmt.Printf("  %-18s %7d  (valeur brute %04x)\n", nomAction(a), actions[a], a)
	}

	if nSound > 0 {
		fmt.Printf("\n=== CHARGE UTILE DES OBJETS Sound ===\n")
		fmt.Printf("  %d objets, taille moyenne %.0f octets, maximum %d\n",
			nSound, float64(sommeSound)/float64(nSound), maxSound)
		fmt.Printf("  le parseur n'en lit que les 13 premiers (pluginID, streamType, sourceID)\n")
		fmt.Printf("  => le reste (proprietes de volume, hauteur, filtrage, positionnement,\n")
		fmt.Printf("     effets, aleas par lecture) n'est PAS exploite\n")
	}
}
