package main

// hirc_texte.go — la sortie LISIBLE du mode `hirc-event`.
//
// Elle est faite pour etre recopiee telle quelle dans un rapport de chantier : chaque
// noeud avec ses proprietes, chaque couche avec son gain de chemin complet, et le releve
// des identifiants `AkPropID` rencontres — celui qui tranche la table de `hirc_noeuds.go`.

import (
	"fmt"
	"sort"
)

// afficherEvenementV3E imprime le dump complet d'un evenement.
func afficherEvenementV3E(ev v3eEvent) {
	etat := ""
	if ev.Etat != 0 {
		etat = fmt.Sprintf(" / etat %d", ev.Etat)
	}
	fmt.Printf("\n=== sbnk %s / event %s%s : %d action(s), %d couche(s) ===\n",
		ev.Bank, ev.Event, etat, len(ev.Actions), len(ev.Couches))
	for _, a := range ev.Actions {
		fmt.Printf("  action %s  %-8s (0x%04x)  -> %s  delai=%.3fs\n",
			a.ID, a.Type, a.Brut, a.Cible, a.DelaiS)
	}
	for i, c := range ev.Couches {
		bus := c.BusEffectif
		if bus == "" {
			bus = "(herite, non declare dans la banque)"
		} else if !c.BusResolu {
			bus += " HORS BANQUE"
		}
		fmt.Printf("  couche %d : %s %s  amont=%+.2f dB  bus=%s\n",
			i+1, c.Cible, c.TypeNoeud, c.GainAmont, bus)
		fmt.Printf("      chemin : %s\n", c.Chemin)
		if c.RangedVolume != nil {
			fmt.Printf("      RANGED volume : %+.2f .. %+.2f dB\n", c.RangedVolume[0], c.RangedVolume[1])
		}
		if c.RangedPitch != nil {
			fmt.Printf("      RANGED hauteur : %+.0f .. %+.0f cents\n", c.RangedPitch[0], c.RangedPitch[1])
		}
		if c.Repetitions != nil {
			fmt.Printf("      repetitions : %d (0 = boucle infinie) mode_transition=%d duree=%.3fs\n",
				*c.Repetitions, c.ModeTransition, c.TransitionS)
		}
		fmt.Printf("      conteneur : sequence=%v continu=%v\n", c.Sequence, c.Continu)
		for _, v := range c.Variantes {
			fmt.Printf("      wem %-11d  noeud %s  gain=%+.2f dB  delai=%.3f s  pitch=%+.0f cents\n",
				v.Wem, v.Noeud, v.GainDB, v.DelaiS, v.PitchCts)
		}
	}
	fmt.Printf("  --- noeuds (%d) ---\n", len(ev.Noeuds))
	for _, n := range ev.Noeuds {
		afficherNoeudV3E(n)
	}
}

// afficherNoeudV3E imprime un noeud : bus, parent, et TOUTES ses proprietes.
func afficherNoeudV3E(n v3eNoeud) {
	wem := ""
	if n.Wem != 0 {
		wem = fmt.Sprintf(" wem=%d", n.Wem)
	}
	if !n.Base.Lu {
		fmt.Printf("    %s %-14s NON LU : %s%s\n", n.ID, n.Type, n.Base.Echec, wem)
		return
	}
	fmt.Printf("    %s %-14s gain_propre=%+.2f dB  bus=%08x  parent=%08x  nFx=%d%s\n",
		n.ID, n.Type, n.GainPropre, n.Base.BusOverride, n.Base.ParentDirect, n.Base.NumFx, wem)
	if n.Base.Echec != "" {
		fmt.Printf("        RESERVE : %s\n", n.Base.Echec)
	}
	for _, p := range n.Base.Props {
		fmt.Printf("        prop %-3d %-22s = %-12g  (u32 %-10d)  octets [%s]\n",
			p.ID, p.Nom, p.Valeur, p.Bits, p.Octets)
	}
	for _, p := range n.Base.Ranged {
		fmt.Printf("        RANGED %-3d %-20s = %g .. %g  octets [%s]\n", p.ID, p.Nom, p.Min, p.Max, p.Octets)
	}
	if len(n.SwitchEtats) > 0 {
		fmt.Printf("        switch groupe=%d defaut=%d\n", n.SwitchGroupe, n.SwitchDefaut)
		for _, e := range n.SwitchEtats {
			marque := ""
			if e.Etat == n.SwitchDefaut {
				marque = "  (DEFAUT)"
			}
			fmt.Printf("          etat %-12d enfants %v wems %v%s\n", e.Etat, e.Enfants, e.Wems, marque)
		}
	}
}

// afficherProfilProps imprime le releve des identifiants de proprietes rencontres, et la
// liste des bus vus. C'est la mesure qui autorise (ou non) a nommer une propriete.
func afficherProfilProps(rap v3eRapport) {
	fmt.Printf("\n=== releve des AkPropID rencontres (%d distincts) ===\n", len(rap.ProfilProps))
	cles := make([]string, 0, len(rap.ProfilProps))
	for k := range rap.ProfilProps {
		cles = append(cles, k)
	}
	sort.Slice(cles, func(i, j int) bool { return rap.ProfilProps[cles[i]] > rap.ProfilProps[cles[j]] })
	for _, k := range cles {
		fmt.Printf("  %-28s %d occurrence(s)\n", k, rap.ProfilProps[k])
	}
	fmt.Printf("=== bus presents dans les banques ouvertes : %d ===\n", len(rap.Bus))
	for id, g := range rap.Bus {
		fmt.Printf("  bus %s gain %+.2f dB\n", id, g)
	}
	fmt.Printf("=== proprietes NON DECODEES : %d ===\n", len(rap.Inconnues))
	for _, s := range rap.Inconnues {
		fmt.Printf("  %s\n", s)
	}
}

// nomActionV3E nomme un type d'Event Action a partir de son octet de poids fort.
func nomActionV3E(brut uint16) string {
	noms := map[byte]string{
		0x01: "Stop", 0x02: "Pause", 0x03: "Resume", 0x04: "Play", 0x05: "Trigger",
		0x06: "Mute", 0x07: "UnMute", 0x08: "SetVolume", 0x09: "SetPitch",
		0x0a: "SetLPF", 0x0b: "SetHPF", 0x0c: "SetBusVolume", 0x0d: "SetState",
		0x0e: "SetSwitch", 0x1c: "Seek", 0x1e: "Break",
	}
	if n, ok := noms[byte(brut>>8)]; ok {
		return n
	}
	return fmt.Sprintf("type0x%02x", byte(brut>>8))
}
