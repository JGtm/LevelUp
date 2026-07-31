package main

import (
	"fmt"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/weaponv3"
)

// extract.go — EXTRACTION de l'inventaire des records de biped des images-clés du film.
//
// RÈGLES D'ANCRAGE, toutes énoncées AVANT toute confrontation à la vérité terrain :
//
//  R1 capacité  : ancre 28 bits 0x8CAC57A puis, dans les 60 bits suivants, le motif 20 bits
//                 0b00000000000000010010 ; les 3 bits qui suivent = l'index.
//                 Retenue seulement si l'ancre est UNIQUE dans le record.
//  R2 grenades  : PREMIER motif i22 (R(3)=4 puis 4 x R(8) <= grenMax, somme > 0) situé APRÈS
//                 l'ancre capacité. La position de l'ancre capacité est établie par R1, donc
//                 sans aucune information de grenade.
//  R3 armes     : familles connues (weaponv3.KnownWeaponHigh32) trouvées dans le record, dans
//                 l'ORDRE DES BITS. E0 = position de la première, E1 = de la seconde.
//  R4 munitions : le bloc i30..i42 se termine EXACTEMENT sur le bit de porte d'i43, à E0-1.
//                 On énumère les débuts possibles et on ne garde que ceux dont le parse
//                 (4 x [union chargeur/jauge + réserve R(11) + surchauffe 9 bits] puis i42)
//                 ATTERRIT au bit près sur E0-1. Aucune valeur n'entre dans le critère : c'est
//                 un critère de LARGEUR, pas de contenu.

// grenadeNames — RECETTE §1 et §8, deux chaînes indépendantes (35 lancers sur 35 appariés aux
// décréments unitaires ; table grenade_types du binaire, ordre = typeId-1).
var grenadeNames = [4]string{"Fragmentation", "Plasma", "Dynamo", "Spike"}

// abilityNames — RECETTE §2. TABLE PARTIELLE : 4 index observés, 11 capacités existent.
// Un index absent de cette table s'affiche « inconnu », jamais deviné.
var abilityNames = map[uint32]string{
	3: "mur portatif",
	4: "grappin",
	5: "propulseur",
	6: "capteur de menace",
}

// ammoTable — RECETTE §4 : chargeur/réserve par arme, 16 armes. Sert UNIQUEMENT de témoin
// croisé (la réserve est un multiple entier du chargeur pour les 16). Aucun champ produit ne
// vient de cette table : elle ne sert qu'à mesurer l'accord de la lecture.
var ammoTable = map[string][2]uint32{
	"BR75": {30, 90}, "CQS48 Bulldog": {12, 12}, "Cindershot": {6, 6}, "Disruptor": {7, 14},
	"Heatwave": {8, 8}, "M41 SPNKr": {2, 2}, "MA40 AR": {25, 75}, "MLRS-2 Hydra": {6, 6},
	"Mangler": {8, 16}, "Mk51 Sidekick": {14, 42}, "Needler": {30, 30}, "S7 Sniper": {10, 10},
	"Sentinel Beam": {80, 80}, "Shock Rifle": {15, 30}, "Skewer": {1, 3}, "VK78 Commando": {40, 120},
}

// SlotAmmo est l'état de munitions d'un emplacement d'arme.
type SlotAmmo struct {
	Mag     *uint32  `json:"chargeur,omitempty"`
	Res     *uint32  `json:"reserve,omitempty"`
	Gauge   *float64 `json:"jauge,omitempty"`
	Overh   uint32   `json:"surchauffe"`
	Flags   uint32   `json:"drapeaux"`
	Present bool     `json:"lu"`
}

// InvState est l'état d'inventaire d'un biped à une image-clé.
type InvState struct {
	Film      string     `json:"film"`
	Frame     int        `json:"f"`
	TimeMS    int        `json:"ms"`
	Slot      uint32     `json:"s"`
	Gren      [4]uint32  `json:"g"`
	GrenNames [4]string  `json:"gn"`
	GrenSel   int        `json:"gs"`  // rang 0..3 du type sélectionné, -1 = indéterminé
	GrenSelBy string     `json:"gsb"` // comment il a été déterminé
	AbilIdx   int        `json:"a"`   // -1 = non lu
	AbilName  string     `json:"an"`  // "inconnu" si l'index n'est pas dans la table partielle
	AbilUses  int        `json:"au"`  // -1 = non localisé
	Weapons   []string   `json:"w"`
	WeaponFam []string   `json:"wf"`
	Drawn     int        `json:"d"`  // emplacement dégainé : 0, 1, 2 = aucun, -1 = non lu
	DrawnRaw  int        `json:"dr"` // valeur brute d'i42 (sel), -1 si absente
	Ammo      []SlotAmmo `json:"am"`
	Parsed    bool       `json:"ok"` // le bloc munitions/sélecteur a-t-il un parse unique ?
	ParseN    int        `json:"pn"` // nombre de parses candidats
}

type slotState struct {
	mag, res *uint32
	gauge    *float64
	overh    uint32
	flags    uint32
}

// parseAmmoBlock parse le bloc i30..i42 depuis le bit s et rend l'état des 4 emplacements,
// la valeur du sélecteur, et le bit d'arrivée. ok=false si un champ déborde de `limit`.
func parseAmmoBlock(pay []byte, s, limit int) (st [4]slotState, sel int, end int, ok bool) {
	p := s
	rd := func(n int) uint32 {
		v := bits(pay, p, n)
		p += n
		return v
	}
	sel = -1
	for k := 0; k < 4; k++ {
		if p+2 > limit {
			return st, sel, p, false
		}
		if rd(1) == 0 { // branche chargeur
			if p+8 > limit {
				return st, sel, p, false
			}
			m := rd(8)
			st[k].mag = &m
		}
		if p+1 > limit {
			return st, sel, p, false
		}
		if rd(1) == 0 { // branche jauge : dequant(R(12), 0, 1)
			if p+12 > limit {
				return st, sel, p, false
			}
			g := float64(rd(12)) / 4095.0
			st[k].gauge = &g
		}
		if p+11+9 > limit {
			return st, sel, p, false
		}
		r := rd(11)
		st[k].res = &r
		st[k].flags = rd(2)
		st[k].overh = rd(7)
	}
	// i42 : R(3) puis deux fois [R(1) porte active-bas + R(2) optionnel].
	if p+3 > limit {
		return st, sel, p, false
	}
	rd(3)
	for g := 0; g < 2; g++ {
		if p+1 > limit {
			return st, sel, p, false
		}
		if rd(1) == 0 {
			if p+2 > limit {
				return st, sel, p, false
			}
			sel = int(rd(2))
		}
	}
	return st, sel, p, true
}

// solveAmmoBlock cherche les débuts s tels que le parse atterrisse EXACTEMENT sur end0 ET
// que les emplacements 2 et 3 soient VIDES au bit près.
//
// La contrainte « emplacements 2 et 3 vides » est STRUCTURELLE, pas sémantique : un Spartan
// ne porte que deux armes, donc les deux emplacements restants des quatre décrits par la
// carte mémoire (0x7F0 + s*0x90, quatre entrées) ne portent ni chargeur, ni jauge, ni
// réserve, ni surchauffe. Elle impose 44 bits nuls et n'utilise AUCUNE valeur des
// emplacements 0 et 1 — ceux que la confrontation à la vérité terrain mesure.
func solveAmmoBlock(pay []byte, end0, lo int) (sols []int) {
	for s := lo; s < end0; s++ {
		st, _, e, ok := parseAmmoBlock(pay, s, end0+1)
		if !ok || e != end0 {
			continue
		}
		bad := false
		for k := 0; k < 4; k++ {
			// R4a : la RECETTE §7 énumère les largeurs possibles de l'union — 2 (vide),
			// 10 (chargeur entier), 14 (jauge fractionnaire). La largeur 22, qui porterait
			// les DEUX branches, n'existe pas. Un parse qui la produit est invalide.
			if st[k].mag != nil && st[k].gauge != nil {
				bad = true
				break
			}
			// R4b : union vide = l'emplacement ne porte aucun état d'arme vivant, donc ni
			// réserve, ni drapeaux, ni surchauffe. Vérifié sur les emplacements du marteau
			// (qui n'émet aucune des deux branches) comme sur les emplacements 2 et 3, que
			// nul Spartan ne remplit.
			if st[k].mag == nil && st[k].gauge == nil {
				if st[k].res == nil || *st[k].res != 0 || st[k].flags != 0 || st[k].overh != 0 {
					bad = true
					break
				}
			}
		}
		if !bad {
			sols = append(sols, s)
		}
	}
	return
}

func u32p(v uint32) *uint32 { return &v }

func runExtract(dir, film string, grenMax uint32, originUS uint64, stepUS uint64, outPath string) {
	known := map[uint32]string{}
	for f, n := range weaponv3.KnownWeaponHigh32 {
		known[f] = n
	}
	n := filmdec.CountFilmChunks(dir)
	var out []InvState
	kf := 0
	// dénominateurs
	var nBiped, nAbil, nGren, nW2, nParse1, nParseN, nParse0 int
	var nAmmoAgree, nAmmoTested int
	for c := 1; c <= n; c++ {
		chunk, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeKeyframe {
				continue
			}
			kf++
			pay := p.Payload(chunk)
			for _, sp := range spans(pay) {
				if sp.ti != bipedTI {
					continue
				}
				nBiped++
				st := InvState{Film: film, Slot: uint32(sp.slot), AbilIdx: -1, GrenSel: -1,
					AbilUses: -1, Drawn: -1, DrawnRaw: -1, AbilName: "inconnu"}
				if p.TimestampUS >= originUS {
					st.TimeMS = int((p.TimestampUS - originUS) / 1000)
					st.Frame = int((p.TimestampUS - originUS) / stepUS)
				} else {
					st.TimeMS = -int((originUS - p.TimestampUS) / 1000)
					st.Frame = -1
				}
				// R1 capacité
				ah := abilityIn(pay, sp.from, sp.to)
				if len(ah) == 1 {
					nAbil++
					st.AbilIdx = int(ah[0].idx3)
					if nm, okk := abilityNames[ah[0].idx3]; okk {
						st.AbilName = nm
					}
					// R2 grenades
					for _, g := range grenIn(pay, ah[0].anchorBit, sp.to, grenMax) {
						nGren++
						st.Gren = g.c
						nz, cnt := -1, 0
						for i := 0; i < 4; i++ {
							st.GrenNames[i] = grenadeNames[i]
							if g.c[i] > 0 {
								nz, cnt = i, cnt+1
							}
						}
						if cnt == 1 {
							st.GrenSel, st.GrenSelBy = nz, "unique compteur non nul"
						} else {
							st.GrenSelBy = "indetermine (plusieurs types portes)"
						}
						break
					}
				}
				// R3 armes
				wpos, wfam := famsIn(pay, sp.from, sp.to, known)
				seen := map[string]bool{}
				var ord []int
				for i, f := range wfam {
					nm := known[f]
					if nm == "" || seen[nm] {
						continue
					}
					seen[nm] = true
					st.Weapons = append(st.Weapons, nm)
					st.WeaponFam = append(st.WeaponFam, fmt.Sprintf("%08x", f))
					ord = append(ord, wpos[i])
				}
				if len(ord) >= 2 {
					nW2++
				}
				// R4 munitions + sélecteur
				if len(ord) >= 1 {
					end0 := ord[0] - 1
					lo := end0 - 300
					if lo < sp.from {
						lo = sp.from
					}
					sols := solveAmmoBlock(pay, end0, lo)
					st.ParseN = len(sols)
					switch len(sols) {
					case 0:
						nParse0++
					case 1:
						nParse1++
					default:
						nParseN++
					}
					if len(sols) >= 1 {
						// DÉPARTAGE : on retient le bloc le PLUS LONG (début le plus petit).
						// Une solution plus courte s'obtient en réinterprétant des bits réels
						// comme appartenant au composant précédent ; le début réel est donc le
						// premier qui parse. Le nombre de candidats est publié (champ pn) pour
						// que le départage reste visible.
						sl, sel, _, _ := parseAmmoBlock(pay, sols[0], end0+1)
						st.Parsed = len(sols) == 1
						st.DrawnRaw = sel
						st.Drawn = sel
						for k := 0; k < 4; k++ {
							a := SlotAmmo{Mag: sl[k].mag, Res: sl[k].res, Gauge: sl[k].gauge,
								Overh: sl[k].overh, Flags: sl[k].flags,
								Present: sl[k].mag != nil || sl[k].gauge != nil}
							st.Ammo = append(st.Ammo, a)
						}
						// TÉMOIN CROISÉ, deux mesures distinctes :
						//  - la BORNE : hors ramassage, une munition ne peut pas dépasser la
						//    dotation pleine de l'arme. Vaut à tout instant du match.
						//  - l'ÉGALITÉ : au spawn seulement, la dotation est pleine.
						for k := 0; k < 2 && k < len(st.Weapons); k++ {
							ref, has := ammoTable[st.Weapons[k]]
							if !has || st.Ammo[k].Mag == nil || st.Ammo[k].Res == nil {
								continue
							}
							nAmmoTested++
							if *st.Ammo[k].Mag <= ref[0] && *st.Ammo[k].Res <= ref[1] {
								nAmmoAgree++
							}
						}
					}
				}
				out = append(out, st)
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Frame != out[j].Frame {
			return out[i].Frame < out[j].Frame
		}
		return out[i].Slot < out[j].Slot
	})
	fmt.Printf("=== EXTRACTION film %s : %d images-clés, %d records biped, %d états produits\n",
		film, kf, nBiped, len(out))
	fmt.Printf("capacité lue (ancre unique)      : %d / %d\n", nAbil, nBiped)
	fmt.Printf("grenades lues (motif après ancre): %d / %d\n", nGren, nBiped)
	fmt.Printf("records à >= 2 armes             : %d / %d\n", nW2, nBiped)
	fmt.Printf("bloc munitions : parse unique %d · plusieurs %d · aucun %d\n", nParse1, nParseN, nParse0)
	fmt.Printf("témoin croisé (borne : munitions <= dotation pleine) : %d / %d\n", nAmmoAgree, nAmmoTested)
	selDist := map[int]int{}
	for _, s := range out {
		selDist[s.DrawnRaw]++
	}
	fmt.Printf("sélecteur i42 (valeur brute) : %v\n", fmtIntMap(selDist))
	frames := map[int]int{}
	for _, s := range out {
		frames[s.Frame]++
	}
	var fs []int
	for f := range frames {
		fs = append(fs, f)
	}
	sort.Ints(fs)
	fmt.Printf("images-clés (image du rejeu -> nb de bipeds) : ")
	for _, f := range fs {
		fmt.Printf("%d:%d ", f, frames[f])
	}
	fmt.Println()
	temoinI42(out)
	var ctrl any
	if len(fs) > 0 {
		ctrl = controle(out, fs[0])
	}
	couv := map[string]any{
		"images_cles":                    kf,
		"records_biped":                  nBiped,
		"capacite_lue":                   fmt.Sprintf("%d/%d", nAbil, nBiped),
		"grenades_lues":                  fmt.Sprintf("%d/%d", nGren, nBiped),
		"records_a_deux_armes":           fmt.Sprintf("%d/%d", nW2, nBiped),
		"munitions_parse_unique":         fmt.Sprintf("%d/%d", nParse1, nParse1+nParseN+nParse0),
		"munitions_parse_multiple":       nParseN,
		"munitions_borne_respectee":      fmt.Sprintf("%d/%d", nAmmoAgree, nAmmoTested),
		"compteur_utilisations_capacite": "NON LOCALISE",
	}
	if outPath != "" {
		if err := writeJSON(outPath, buildLivrable(out, film, couv, ctrl)); err != nil {
			fmt.Fprintf(os.Stderr, "écriture: %v\n", err)
		}
	}
}

func fmtIntMap(m map[int]int) string {
	var ks []int
	for k := range m {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	s := ""
	for _, k := range ks {
		s += fmt.Sprintf("%d:%d ", k, m[k])
	}
	return s
}
