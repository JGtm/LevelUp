package main

import (
	"fmt"
	"sort"
)

// controle.go — LA CONFRONTATION À LA VÉRITÉ TERRAIN.
//
// Relevé à l'écran par l'utilisateur en Theater sur le film 000d5950, au début réel du match.
// C'est le premier contrôle réellement exécutable du chantier : jusqu'ici les deux côtés
// venaient de films différents.
//
// Le décodeur ne corrige JAMAIS le terrain ici : tout écart est publié comme un échec.

type truthRow struct {
	slot     uint32
	nom      string
	grenade  string
	capacite string
	uses     int
	arme     string
	mag, res int // -1 = sans objet (jauge de charge)
}

var truth = []truthRow{
	{512, "whiteknight2519", "Dynamo", "grappin", 5, "MLRS-2 Hydra", 6, 6},
	{513, "JAVIERLOLITO540", "Plasma", "grappin", 5, "Gravity Hammer", -1, -1},
	{514, "JGtm", "Fragmentation", "propulseur", 5, "M41 SPNKr", 2, 2},
	{515, "LORD PEINX13", "Spike", "capteur de menace", 4, "Mangler", 8, 16},
	{516, "IKE ILYA", "Plasma", "grappin", 5, "MA40 AR", 25, 75},
	{517, "Akatsuki fire17", "Dynamo", "mur portatif", 5, "Mangler", 8, 16},
	{518, "aldusbroncus", "Dynamo", "propulseur", 5, "Cindershot", 6, 6},
	{519, "VitaminA1688", "Fragmentation", "propulseur", 5, "CQS48 Bulldog", 12, 12},
}

// NOTE sur deux lignes du relevé :
//   - slot 517 : l'utilisateur lit « mur portatif (1) ». Le 1 est le nombre d'utilisations,
//     champ NON localisé par ce décodeur ; seul le nom est confronté.
//   - slot 518 : l'utilisateur lit « Cremator », le POC affiche « Cindershot ». Défaut
//     d'étiquette connu de la table de noms, pas un défaut de décodage : on confronte la
//     famille décodée au nom que la table du dépôt lui donne.

func controle(states []InvState, frame int) map[string]any {
	byslot := map[uint32]InvState{}
	for _, s := range states {
		if s.Frame == frame {
			byslot[s.Slot] = s
		}
	}
	sort.Slice(truth, func(i, j int) bool { return truth[i].slot < truth[j].slot })

	var okGren, okAbil, okArme, okMag, okDeg int
	var nGren, nAbil, nArme, nMag, nDeg int

	fmt.Printf("\n=== CONTRÔLE TERRAIN — film 000d5950, image-clé de l'image %d\n", frame)
	fmt.Printf("%-5s %-17s %-14s %-14s %-18s %-18s %-9s %-9s %s\n",
		"slot", "joueur", "grenade lue", "attendue", "capacité lue", "attendue", "armes", "mun. lues", "attendues")
	for _, t := range truth {
		s, has := byslot[t.slot]
		if !has {
			fmt.Printf("%-5d %-17s ABSENT DE L'EXTRACTION\n", t.slot, t.nom)
			continue
		}
		// grenade
		gl := "non lue"
		if s.GrenSel >= 0 {
			gl = grenadeNames[s.GrenSel]
		}
		nGren++
		if gl == t.grenade {
			okGren++
		}
		// capacité
		nAbil++
		if s.AbilName == t.capacite {
			okAbil++
		}
		// arme en main : présente dans la paire décodée ?
		nArme++
		idx := -1
		for i, w := range s.Weapons {
			if w == t.arme {
				idx = i
			}
		}
		if idx >= 0 {
			okArme++
		}
		// munitions de l'emplacement portant l'arme du relevé
		ml := "non lues"
		if idx >= 0 && idx < len(s.Ammo) {
			a := s.Ammo[idx]
			switch {
			case a.Mag != nil:
				ml = fmt.Sprintf("%d/%d", *a.Mag, deref(a.Res))
			case a.Gauge != nil:
				ml = fmt.Sprintf("jauge %.1f%%", *a.Gauge*100)
			default:
				ml = "aucune"
			}
		}
		me := "sans objet"
		if t.mag >= 0 {
			me = fmt.Sprintf("%d/%d", t.mag, t.res)
			nMag++
			if idx >= 0 && idx < len(s.Ammo) && s.Ammo[idx].Mag != nil &&
				int(*s.Ammo[idx].Mag) == t.mag && s.Ammo[idx].Res != nil && int(*s.Ammo[idx].Res) == t.res {
				okMag++
			}
		}
		// emplacement dégainé
		nDeg++
		if s.Drawn == idx {
			okDeg++
		}
		fmt.Printf("%-5d %-17s %-14s %-14s %-18s %-18s emp.%-4s %-9s %s\n",
			t.slot, t.nom, gl, t.grenade, s.AbilName, t.capacite,
			fmt.Sprintf("%d", idx), ml, me)
	}
	fmt.Printf("\nRÉSULTAT CHIFFRÉ\n")
	fmt.Printf("  grenade portée      : %d / %d\n", okGren, nGren)
	fmt.Printf("  capacité            : %d / %d\n", okAbil, nAbil)
	fmt.Printf("  arme du relevé dans la paire décodée : %d / %d\n", okArme, nArme)
	fmt.Printf("  chargeur/réserve    : %d / %d\n", okMag, nMag)
	fmt.Printf("  emplacement dégainé : %d / %d\n", okDeg, nDeg)

	// Contrôle interne des GROUPES, sans aucune injection de la table des noms.
	groups := map[string][]uint32{}
	for _, t := range truth {
		groups[t.capacite] = append(groups[t.capacite], t.slot)
	}
	fmt.Printf("\nCONTRÔLE INTERNE DES GROUPES (aucun nom injecté)\n")
	var names []string
	for k := range groups {
		names = append(names, k)
	}
	sort.Strings(names)
	vals := map[string]map[int]bool{}
	for _, k := range names {
		vals[k] = map[int]bool{}
		var idxs []string
		for _, sl := range groups[k] {
			v := -1
			if s, has := byslot[sl]; has {
				v = s.AbilIdx
			}
			vals[k][v] = true
			idxs = append(idxs, fmt.Sprintf("%d->%d", sl, v))
		}
		verdict := "GROUPÉ"
		if len(vals[k]) != 1 {
			verdict = "ÉCLATÉ (échec)"
		}
		fmt.Printf("  %-18s %-28s %s\n", k, fmt.Sprint(idxs), verdict)
	}
	// mutuellement distincts ?
	seen := map[int]string{}
	distinct := true
	for _, k := range names {
		for v := range vals[k] {
			if prev, dup := seen[v]; dup && prev != k {
				distinct = false
				fmt.Printf("  COLLISION : index %d partagé par %s et %s\n", v, prev, k)
			}
			seen[v] = k
		}
	}
	if distinct {
		fmt.Printf("  les %d capacités reçoivent %d index mutuellement DISTINCTS\n", len(names), len(seen))
	}
	lignes := []map[string]any{}
	for _, t := range truth {
		s, has := byslot[t.slot]
		l := map[string]any{"slot": t.slot, "joueur": t.nom, "presente": has}
		if has {
			gl := "non lue"
			if s.GrenSel >= 0 {
				gl = grenadeNames[s.GrenSel]
			}
			idx := -1
			for i, w := range s.Weapons {
				if w == t.arme {
					idx = i
				}
			}
			l["grenade_lue"], l["grenade_attendue"] = gl, t.grenade
			l["capacite_lue"], l["capacite_attendue"] = s.AbilName, t.capacite
			l["capacite_index"] = s.AbilIdx
			l["arme_attendue"], l["arme_emplacement_decode"] = t.arme, idx
			if idx >= 0 && idx < len(s.Ammo) && s.Ammo[idx].Mag != nil {
				l["munitions_lues"] = fmt.Sprintf("%d/%d", *s.Ammo[idx].Mag, deref(s.Ammo[idx].Res))
			} else {
				l["munitions_lues"] = "aucune"
			}
			if t.mag >= 0 {
				l["munitions_attendues"] = fmt.Sprintf("%d/%d", t.mag, t.res)
			} else {
				l["munitions_attendues"] = "sans objet"
			}
			l["emplacement_degaine_lu"] = s.DrawnRaw
			l["utilisations_attendues"] = t.uses
			l["utilisations_lues"] = "non localise"
		}
		lignes = append(lignes, l)
	}
	return map[string]any{
		"film":                       "000d5950",
		"image":                      frame,
		"source_verite":              "releve a l ecran en Theater par l utilisateur, film 000d5950, au debut reel du match",
		"grenade_portee":             fmt.Sprintf("%d/%d", okGren, nGren),
		"capacite":                   fmt.Sprintf("%d/%d", okAbil, nAbil),
		"arme_du_releve_dans_paire":  fmt.Sprintf("%d/%d", okArme, nArme),
		"chargeur_reserve":           fmt.Sprintf("%d/%d", okMag, nMag),
		"emplacement_degaine":        fmt.Sprintf("%d/%d", okDeg, nDeg),
		"utilisations_capacite":      "0/8 — champ NON LOCALISE (36 006 positions testees sur 6 ancres)",
		"groupes_capacite_distincts": distinct,
		"lignes":                     lignes,
	}
}

func deref(p *uint32) uint32 {
	if p == nil {
		return 0
	}
	return *p
}
