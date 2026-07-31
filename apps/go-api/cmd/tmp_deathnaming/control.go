package main

import (
	"fmt"
	"sort"
	"strconv"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/weaponv3"
)

// control.go — LE CONTROLE PAR SOURCE DISJOINTE.
//
// POURQUOI IL EST NECESSAIRE ICI. Le pont index de tir -> xuid est resolu par une
// affectation de cout minimal, et sa marge est etroite : 33 contradictions contre 39 pour
// la deuxieme permutation. Une marge de 6 sur 519 events ne suffit pas a fonder une
// publication. Le repli serait alors une inference de plus, mieux habillee.
//
// LA SOURCE DISJOINTE. L'arme d'un tir (identifiant 64 bits du record de degat) doit
// appartenir au loadout lu pour ce slot dans les images-cles. Les deux chaines n'ont
// AUCUNE piece commune : l'une part des records de degat du flux de trames, l'autre du
// balayage des familles d'arme dans les records de biped des images-cles. Aucun parametre,
// aucun seuil, aucune horloge ne leur est partage.
//
// LE TEMOIN. La meme mesure, mais sur un AUTRE slot vivant au meme instant. Sans lui, un
// taux d'accord eleve ne dirait rien : si tous les joueurs portaient les memes armes,
// n'importe quelle attribution concorderait.

// loadoutWindow est l'ecart maximal accepte entre un tir et l'image-cle qui fournit le
// loadout de reference. Les images-cles arrivent toutes les ~20 s ; au-dela d'une demi
// periode, la reference est plus proche de l'image suivante.
const loadoutWindowUS = 15_000_000

// slotLoadouts indexe, par slot, les familles d'arme lues a chaque image-cle.
type slotLoadouts map[uint32][]filmdec.KeyframeLoadout

// buildSlotLoadouts regroupe les loadouts par slot, tries par instant.
func buildSlotLoadouts(ls []filmdec.KeyframeLoadout) slotLoadouts {
	out := slotLoadouts{}
	for _, l := range ls {
		out[l.Slot] = append(out[l.Slot], l)
	}
	for s := range out {
		v := out[s]
		sort.Slice(v, func(i, j int) bool { return v[i].TimestampUS < v[j].TimestampUS })
		out[s] = v
	}
	return out
}

// familiesAt rend les familles lues pour un slot a l'image-cle la plus proche de tUS.
func (sl slotLoadouts) familiesAt(slot uint32, tUS uint64) ([]uint32, bool) {
	v := sl[slot]
	if len(v) == 0 {
		return nil, false
	}
	best, bestD := -1, uint64(1)<<62
	for i, l := range v {
		d := l.TimestampUS - tUS
		if l.TimestampUS < tUS {
			d = tUS - l.TimestampUS
		}
		if d < bestD {
			bestD, best = d, i
		}
	}
	if best < 0 || bestD > loadoutWindowUS {
		return nil, false
	}
	return v[best].Families, true
}

// reportLoadoutControl mesure l'accord entre l'arme d'un tir et le loadout du slot auquel
// le repli le rattache, puis le compare au meme test sur un autre slot vivant.
func reportLoadoutControl(lives []Life, fire []filmdec.FireEvent, b bridgeResult, filmDir string) {
	known := map[uint32]bool{}
	for f := range weaponv3.KnownWeaponHigh32 {
		known[f] = true
	}
	ls, err := filmdec.ScanFilmKeyframeLoadouts(filmDir, known)
	if err != nil || len(ls) == 0 {
		fmt.Printf("loadouts d'images-cles indisponibles (%v) : CONTROLE NON EXECUTE.\n", err)
		return
	}
	sl := buildSlotLoadouts(ls)
	fmt.Printf("loadouts lus : %d, sur %d slots\n", len(ls), len(sl))

	agree, tested := 0, 0
	ctrlAgree, ctrlTested := 0, 0
	for _, e := range fire {
		if e.WeaponID == 0 {
			continue
		}
		fam := uint32(e.WeaponID >> 32)
		slot, ok := uniqueLifeSlot(lives, b, e)
		if !ok {
			continue
		}
		if fams, ok := sl.familiesAt(slot, e.TimestampUS); ok {
			tested++
			if containsFam(fams, fam) {
				agree++
			}
		}
		// TEMOIN : un autre slot VIVANT au meme instant, choisi de facon deterministe (le
		// plus petit slot different). Meme arme, meme instant, mauvais porteur.
		if other, ok := otherLiveSlot(lives, slot, int64(e.TimestampUS)); ok {
			if fams, ok := sl.familiesAt(other, e.TimestampUS); ok {
				ctrlTested++
				if containsFam(fams, fam) {
					ctrlAgree++
				}
			}
		}
	}
	fmt.Printf("l'arme du tir appartient au loadout du slot RATTACHE : %d / %d (%.1f %%)\n",
		agree, tested, pct(agree, tested))
	fmt.Printf("TEMOIN, meme arme sur un AUTRE slot vivant au meme instant : %d / %d (%.1f %%)\n",
		ctrlAgree, ctrlTested, pct(ctrlAgree, ctrlTested))
}

// uniqueLifeSlot rend le slot de LA vie de l'identite du tireur qui couvre l'instant, si
// elle est unique — la meme regle que le rattachement lui-meme.
func uniqueLifeSlot(lives []Life, b bridgeResult, e filmdec.FireEvent) (uint32, bool) {
	x, ok := b.IndexToXUID[e.FilmIndex]
	if !ok {
		return 0, false
	}
	var found uint32
	n := 0
	t := int64(e.TimestampUS)
	for _, l := range lives {
		if l.XUID == x && t >= l.StartUS && t <= l.EndUS {
			found, n = l.Slot, n+1
		}
	}
	return found, n == 1
}

// otherLiveSlot rend le plus petit slot, different de `slot`, dont une vie couvre t.
func otherLiveSlot(lives []Life, slot uint32, t int64) (uint32, bool) {
	best, found := uint32(0), false
	for _, l := range lives {
		if l.Slot == slot || t < l.StartUS || t > l.EndUS {
			continue
		}
		if !found || l.Slot < best {
			best, found = l.Slot, true
		}
	}
	return best, found
}

func containsFam(fams []uint32, f uint32) bool {
	for _, v := range fams {
		if v == f {
			return true
		}
	}
	return false
}

// loadoutKnownFamilies rend le catalogue de familles d'arme qui donne sa selectivite au
// balayage des images-cles (cf. filmdec.ScanFilmKeyframeLoadouts).
func loadoutKnownFamilies() map[uint32]bool {
	out := map[uint32]bool{}
	for f := range weaponv3.KnownWeaponHigh32 {
		out[f] = true
	}
	return out
}

// familyOfHexWeapon extrait la famille (high-32) d'un identifiant d'arme publie en
// hexadecimal par le rejeu (« 0x… » sur 16 chiffres, cf. replay.formatWeaponID).
func familyOfHexWeapon(s string) (uint32, bool) {
	if len(s) != 18 || s[:2] != "0x" {
		return 0, false
	}
	v, err := strconv.ParseUint(s[2:], 16, 64)
	if err != nil {
		return 0, false
	}
	return uint32(v >> 32), true
}
