package main

import (
	"fmt"
	"math"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/replay"
)

// pipeline.go — LA MESURE SUR LE CHEMIN DE PRODUCTION, et la NON-REGRESSION.
//
// POURQUOI NE PAS SE CONTENTER DES MESURES PRECEDENTES. Les passes 1 a 6 mesurent le repli
// dans cet outil, avec ses propres structures. Le gate de l'etape 2 porte sur ce que le
// REJEU publie, apres decimation des trajectoires et filtrage des tirs dont le slot n'a pas
// de trajectoire publiee. Ce sont deux nombres differents, et c'est le second qui compte.
//
// LA NON-REGRESSION, telle que l'item 2.4 la demande : les tirs que l'ancien pont publiait
// doivent garder leur slot. On construit donc le document DEUX FOIS — avec le fil des morts
// et sans — et on compare tir a tir. Un tir qui change de slot n'est pas un gain de
// couverture, c'est une contradiction entre deux methodes, et il doit se voir.

// runPipeline construit le document de rejeu avec et sans le fil des morts, et compare.
func runPipeline(matchID, filmDir string, rng *filmdec.Vec3Range, deaths []replay.Death) {
	fmt.Printf("\n--- 7. CHEMIN DE PRODUCTION (replay.BuildFromPositions) ---\n")
	opt := filmdec.DefaultScanFilmOptions()
	opt.WorldRange = rng
	opt.CaptureDirs = true
	pos, err := filmdec.ScanFilmBipedPositions(filmDir, opt)
	if err != nil {
		fmt.Println("positions:", err)
		return
	}
	fire, err := filmdec.ScanFilmFireEvents(filmDir)
	if err != nil {
		fmt.Println("tirs:", err)
		return
	}
	grenades, _ := filmdec.ScanFilmGrenadeThrows(filmDir)
	proj, _ := filmdec.ScanFilmProjectiles(filmDir, rng)

	base := replay.Options{WorldRange: rng, Grenades: grenades, Projectiles: proj}
	avant := replay.BuildFromPositions(matchID, "halo_infinite", pos, fire, base)

	apres := base
	apres.Deaths = deaths
	doc := replay.BuildFromPositions(matchID, "halo_infinite", pos, fire, apres)

	fmt.Printf("tirs publies — AVANT (vote seul) : %d\n", len(avant.Shots))
	fmt.Printf("tirs publies — APRES (fil des morts) : %d\n", len(doc.Shots))
	fmt.Printf("records de tir disponibles dans le film : %d\n", len(fire))
	fmt.Printf("COUVERTURE : %.1f %% (gate de l'etape 2 : > 85 %%)\n", pct(len(doc.Shots), len(fire)))

	// NON-REGRESSION : un tir est identifie par (instant de frame, arme) ; son slot doit
	// etre le meme dans les deux documents.
	type key struct {
		t int
		w string
	}
	before := map[key]uint32{}
	for _, s := range avant.Shots {
		before[key{s.T, s.Weapon}] = s.Slot
	}
	same, moved, absent := 0, 0, 0
	var movedList []movedShot
	for _, s := range doc.Shots {
		old, ok := before[key{s.T, s.Weapon}]
		if !ok {
			continue
		}
		if old == s.Slot {
			same++
		} else {
			moved++
			movedList = append(movedList, movedShot{s.T, old, s.Slot, s.Weapon})
		}
	}
	for k := range before {
		found := false
		for _, s := range doc.Shots {
			if s.T == k.t && s.Weapon == k.w {
				found = true
				break
			}
		}
		if !found {
			absent++
		}
	}
	fmt.Printf("NON-REGRESSION : %d tirs communs gardent leur slot, %d en changent, %d ont disparu\n",
		same, moved, absent)
	if moved == 0 && absent == 0 {
		fmt.Printf("  => aucun tir precedemment publie n'est perdu ni deplace.\n")
		return
	}
	// QUI A RAISON SUR LES TIRS DEPLACES ? La question ne se tranche pas par preference pour
	// l'une des deux methodes : on la pose a une TROISIEME source, qui ne partage de piece
	// avec aucune des deux. L'arme du tir appartient-elle au loadout de l'ancien slot, ou a
	// celui du nouveau ? Le loadout est lu dans les records de biped des images-cles ; ni le
	// vote ni le fil des morts n'y touchent.
	fmt.Printf("\nARBITRAGE DES %d TIRS DEPLACES, par le loadout des images-cles :\n", moved)
	known := loadoutKnownFamilies()
	ls, err := filmdec.ScanFilmKeyframeLoadouts(filmDir, known)
	if err != nil || len(ls) == 0 {
		fmt.Printf("  loadouts indisponibles (%v) : ARBITRAGE NON RENDU.\n", err)
		return
	}
	sl := buildSlotLoadouts(ls)
	step := uint64(doc.FrameIntervalMS) * 1000
	origin := frameOrigin(pos)
	forOld, forNew, neither, both := 0, 0, 0, 0
	for _, m := range movedList {
		fam, ok := familyOfHexWeapon(m.weapon)
		if !ok {
			continue
		}
		tUS := origin + uint64(m.t)*step
		oldOK := slotCarries(sl, m.old, tUS, fam)
		newOK := slotCarries(sl, m.new, tUS, fam)
		switch {
		case oldOK && newOK:
			both++
		case newOK:
			forNew++
		case oldOK:
			forOld++
		default:
			neither++
		}
	}
	fmt.Printf("  l'arme est au loadout du NOUVEAU slot seulement : %d\n", forNew)
	fmt.Printf("  l'arme est au loadout de l'ANCIEN slot seulement : %d\n", forOld)
	fmt.Printf("  les deux la portent (non discriminant)           : %d\n", both)
	fmt.Printf("  aucun des deux (loadout absent ou arme inconnue)  : %d\n", neither)

	fmt.Printf("\nSECOND ARBITRE, la visee du record contre le cap du biped :\n")
	arbitrateByAim(pos, fire, origin, step, movedList)
}

// movedShot est un tir que le changement de pont a fait changer de slot.
type movedShot struct {
	t        int
	old, new uint32
	weapon   string
}

// frameOrigin rend l'instant du premier echantillon, qui est l'origine de la grille de
// frames du document (cf. BuildFromPositions).
func frameOrigin(pos []filmdec.BipedPosition) uint64 {
	best := uint64(1) << 62
	for _, p := range pos {
		if p.TimestampUS < best {
			best = p.TimestampUS
		}
	}
	return best
}

// slotCarries dit si le slot porte la famille d'arme a cet instant.
func slotCarries(sl slotLoadouts, slot uint32, tUS uint64, fam uint32) bool {
	fams, ok := sl.familiesAt(slot, tUS)
	return ok && containsFam(fams, fam)
}

// arbitrateByAim — SECOND ARBITRE, tir par tir : la visee du record de degat (cubemap
// 30 bits) contre le cap de visee du biped (composant du record de position, 12 bits).
//
// DEUX CHAMPS DIFFERENTS, DEUX LARGEURS DIFFERENTES, DEUX COMPOSANTS DIFFERENTS : leur
// coincidence n'a aucune raison d'exister si l'attribution est fausse.
//
// PARTIELLEMENT DISJOINT, ET IL FAUT LE DIRE. Cette grandeur est aussi l'une des deux
// sources du vote. Mais le vote l'AGREGE en majorite par slot, alors qu'on l'interroge ici
// TIR PAR TIR : un slot mal elu par la majorite peut etre contredit par chacun de ses
// tirs. Ce n'est donc pas la meme mesure, et ce n'est pas non plus une source neuve.
func arbitrateByAim(pos []filmdec.BipedPosition, fire []filmdec.FireEvent,
	origin uint64, step uint64, moved []movedShot) {
	tracks := map[uint32][]filmdec.BipedPosition{}
	for _, p := range pos {
		tracks[p.Slot] = append(tracks[p.Slot], p)
	}
	for s := range tracks {
		v := tracks[s]
		sort.Slice(v, func(i, j int) bool { return v[i].TimestampUS < v[j].TimestampUS })
		tracks[s] = v
	}
	forOld, forNew, tie, noData := 0, 0, 0, 0
	for _, m := range moved {
		tUS := origin + uint64(m.t)*step
		e, ok := fireAt(fire, tUS, m.weapon)
		if !ok || !e.HasAim {
			noData++
			continue
		}
		head := math.Atan2(float64(e.Aim[1]), float64(e.Aim[0])) * 180 / math.Pi
		dOld, okOld := aimGap(tracks[m.old], tUS, head)
		dNew, okNew := aimGap(tracks[m.new], tUS, head)
		switch {
		case !okOld && !okNew:
			noData++
		case okNew && (!okOld || dNew < dOld-1):
			forNew++
		case okOld && (!okNew || dOld < dNew-1):
			forOld++
		default:
			tie++
		}
	}
	fmt.Printf("  la visee designe le NOUVEAU slot : %d\n", forNew)
	fmt.Printf("  la visee designe l'ANCIEN slot   : %d\n", forOld)
	fmt.Printf("  ecart < 1 deg, non discriminant  : %d\n", tie)
	fmt.Printf("  visee illisible                  : %d\n", noData)
}

// fireAt retrouve l'event de tir d'un instant de frame et d'une arme.
func fireAt(fire []filmdec.FireEvent, tUS uint64, weapon string) (filmdec.FireEvent, bool) {
	for _, e := range fire {
		if absU64(e.TimestampUS, tUS) <= 100_000 && formatWeapon(e.WeaponID) == weapon {
			return e, true
		}
	}
	return filmdec.FireEvent{}, false
}

// aimGap rend l'ecart de cap entre la visee du tir et celle du biped, a cet instant.
func aimGap(pts []filmdec.BipedPosition, tUS uint64, head float64) (float64, bool) {
	best, bestD := -1, uint64(1)<<62
	for i, p := range pts {
		if d := absU64(p.TimestampUS, tUS); d < bestD {
			bestD, best = d, i
		}
	}
	if best < 0 || bestD > 120_000 {
		return 0, false
	}
	h, ok := pts[best].AimHeadingDeg()
	if !ok {
		return 0, false
	}
	d := math.Mod(math.Abs(head-float64(h)), 360)
	if d > 180 {
		d = 360 - d
	}
	return d, true
}

func absU64(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return b - a
}

// formatWeapon reproduit la mise en forme hexadecimale du rejeu (replay.formatWeaponID).
func formatWeapon(id uint64) string {
	const hexDigits = "0123456789ABCDEF"
	buf := make([]byte, 16)
	for i := 15; i >= 0; i-- {
		buf[i] = hexDigits[id&0xF]
		id >>= 4
	}
	return "0x" + string(buf)
}
