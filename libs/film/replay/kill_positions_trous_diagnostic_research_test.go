//go:build research

package replay

// kill_positions_trous_diagnostic_research_test.go — le DIAGNOSTIC d un cote non place de
// l enquete E8.1 (`kill_positions_trous_research_test.go`), sorti par deplacement pur pour tenir le
// seuil de 500 lignes : dernier et prochain echantillon du siege, embarquement, teleportation,
// immobilite, mort anterieure du tueur.

import (
	"math"
	"sort"

	"github.com/JGtm/LevelUp/libs/film/internal/grammar"
)

// cote : le diagnostic d un cote (tueur ou victime) d une mort a l instant tUS.
type cote struct {
	place        bool
	cause        string
	ecartMS      int64
	avantMS      int64 // tUS - dernier echantillon <= tUS du siege nomme (ms), -1 inconnu
	apresMS      int64 // prochain echantillon > tUS du siege nomme - tUS (ms), -1 inconnu
	tirsDansTrou int
	finDeVieMS   int64
	aFinDeVie    bool
	saut         float64
	aSaut        bool
}

func (c *contexteFilm) diagnostiquer(xuid uint64, tUS uint64, estTueur bool, mortMS int64) cote {
	sieges := siegesDe(c.reg, c.sieges, xuid, tUS)
	if v, _ := positionOf(c.pistes, sieges, tUS); v != nil {
		r := cote{place: true}
		r.finDeVieMS, r.aFinDeVie = c.finDeVie(sieges, tUS)
		return r
	}
	r := cote{avantMS: -1, apresMS: -1, ecartMS: -1}
	if len(sieges) == 0 {
		r.cause = c.causeSansSiege(xuid, mortMS, estTueur)
		return r
	}
	proches := 0
	for _, s := range sieges {
		if p, d := c.pistes[s].at(tUS); d <= killPosToleranceUS && p.HasWorld {
			proches++
		}
	}
	if proches > 1 {
		r.cause = "ambigu"
		return r
	}
	r.finDeVieMS, r.aFinDeVie = c.finDeVie(sieges, tUS)
	// Le siege le plus recemment echantillonne avant tUS.
	var dernier, prochain grammar.BipedPosition
	var avDernier grammar.BipedPosition
	var aDernier, aProchain, aAvDernier bool
	var siegeDernier uint32
	for _, s := range sieges {
		pts := c.pistes[s].pts
		i := sort.Search(len(pts), func(k int) bool { return pts[k].TimestampUS > tUS })
		if i > 0 && (!aDernier || pts[i-1].TimestampUS > dernier.TimestampUS) {
			dernier, aDernier, siegeDernier = pts[i-1], true, s
			aAvDernier = i > 1
			if aAvDernier {
				avDernier = pts[i-2]
			}
		}
		if i < len(pts) && (!aProchain || pts[i].TimestampUS < prochain.TimestampUS) {
			prochain, aProchain = pts[i], true
		}
	}
	mini := int64(math.MaxInt64)
	if aDernier {
		r.avantMS = int64(tUS-dernier.TimestampUS) / 1000
		mini = r.avantMS
	}
	if aProchain {
		r.apresMS = int64(prochain.TimestampUS-tUS) / 1000
		if r.apresMS < mini {
			mini = r.apresMS
		}
	}
	r.ecartMS = mini
	if idx, ok := c.g.PlayerIndices.ByXUID[xuid]; ok && aDernier {
		for _, ts := range c.tirsPar[idx] {
			if ts > dernier.TimestampUS && ts <= tUS {
				r.tirsDansTrou++
			}
		}
	}
	r.cause = c.classer(xuid, estTueur, mortMS, tUS, aDernier, aAvDernier, dernier, avDernier, siegeDernier)
	if aDernier && aProchain && dernier.HasWorld && prochain.HasWorld {
		r.saut, r.aSaut = distPos(prochain, dernier), true
	}
	return r
}

func tranche(ms int64) string {
	switch {
	case ms <= 1000:
		return "<=1s"
	case ms <= 5000:
		return "1-5s"
	case ms <= 15000:
		return "5-15s"
	}
	return ">15s"
}

func (c *contexteFilm) classer(xuid uint64, estTueur bool, mortMS int64, tUS uint64, aDernier, aAvDernier bool,
	dernier, avDernier grammar.BipedPosition, siegeDernier uint32,
) string {
	if estTueur {
		for _, m := range c.mortsPar[xuid] {
			if m < mortMS && mortMS-m <= 10_000 {
				return "tueur-deja-mort(<=10 s)"
			}
		}
	}
	if !aDernier {
		return "aucun-echantillon-avant"
	}
	av := tranche(int64(tUS-dernier.TimestampUS) / 1000)
	vitesse := -1.0
	if aAvDernier && dernier.TimestampUS > avDernier.TimestampUS {
		vitesse = distPos(dernier, avDernier) / (float64(dernier.TimestampUS-avDernier.TimestampUS) / 1e6)
	}
	switch {
	case c.embarqueEntre(siegeDernier, dernier.TimestampUS, tUS):
		return "embarque(evt 8 sans 22) " + av
	case c.presVehicule(dernier):
		return "dernier-point-a-<2m-d-un-vehicule " + av
	case c.evtProche(c.telep[siegeDernier], dernier.TimestampUS, 1_000_000):
		return "teleporte " + av
	case aAvDernier && (avDernier.Q == dernier.Q || vitesse >= 0 && vitesse < 0.5):
		return "immobile(<0,5 m/s) " + av
	case aAvDernier && float64(dernier.Z-avDernier.Z)/(float64(dernier.TimestampUS-avDernier.TimestampUS)/1e6) < -8:
		return "chute(vz<-8m/s) " + av
	}
	return "en-mouvement " + av
}

func (c *contexteFilm) presVehicule(p grammar.BipedPosition) bool {
	v := c.vehicules
	i := sort.Search(len(v), func(k int) bool { return v[k].TimestampUS+1_000_000 >= p.TimestampUS })
	for ; i < len(v) && v[i].TimestampUS <= p.TimestampUS+1_000_000; i++ {
		if !v[i].HasWorld {
			continue
		}
		if distPos(v[i], p) < 2 {
			return true
		}
	}
	return false
}

// finDeVie : dernier echantillon des sieges nommes, cherche jusqu a tUS + 2 s, moins tUS (ms).
func (c *contexteFilm) finDeVie(sieges []uint32, tUS uint64) (int64, bool) {
	var best uint64
	ok := false
	for _, s := range sieges {
		pts := c.pistes[s].pts
		i := sort.Search(len(pts), func(k int) bool { return pts[k].TimestampUS > tUS+2_000_000 })
		if i > 0 && pts[i-1].TimestampUS+2_000_000 >= tUS && (!ok || pts[i-1].TimestampUS > best) {
			best, ok = pts[i-1].TimestampUS, true
		}
	}
	if !ok {
		return 0, false
	}
	return (int64(best) - int64(tUS)) / 1000, true
}

func (c *contexteFilm) embarqueEntre(slot uint32, depuis, jusqua uint64) bool {
	for _, b := range c.boards[slot] {
		if b+500_000 >= depuis && b <= jusqua {
			ferme := false
			for _, e := range c.exits[slot] {
				if e > b && e <= jusqua {
					ferme = true
				}
			}
			if !ferme {
				return true
			}
		}
	}
	return false
}

func (c *contexteFilm) evtProche(ts []uint64, t uint64, fen uint64) bool {
	for _, x := range ts {
		if absDiffU64(x, t) <= fen {
			return true
		}
	}
	return false
}

// causeSansSiege : aucun siege ne porte ce xuid a l instant. Pour un tueur, la cause la plus
// probable est qu il est DEJA mort (frag posthume : grenade, projectile en vol).
func (c *contexteFilm) causeSansSiege(xuid uint64, mortMS int64, estTueur bool) string {
	if estTueur {
		for _, m := range c.mortsPar[xuid] {
			if m <= mortMS && mortMS-m <= 10_000 {
				return "sans-siege:tueur-deja-mort(<=10 s)"
			}
		}
	}
	// Le joueur a-t-il un siege nomme dans les 3 s autour ?
	tUS := uint64(mortMS*1000 + c.origine)
	for _, dt := range []int64{-3_000_000, -1_000_000, 1_000_000, 3_000_000} {
		if len(siegesDe(c.reg, c.sieges, xuid, uint64(int64(tUS)+dt))) > 0 {
			return "sans-siege:nomme-a-+-3s"
		}
	}
	return "sans-siege:aucun-a-+-3s"
}
