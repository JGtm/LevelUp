package replay

// vehicules_v4_couverture_test.go — INSTRUMENT DE MESURE (lot V4, etage 1) : POURQUOI si peu
// d EPISODES D OCCUPATION. LECTURE SEULE, garde par V4_ROOT / V4_FILMS.
//
// LE CONSTAT QUI OUVRE LE LOT. L artefact `fccc61cd` publie 2 episodes pour 21 vies de vehicule
// (1 seul nomme), `0d76e8f1` en publie 12 pour 45 vies recensees / 30 publiees. L utilisateur a
// conduit bien davantage. La primitive ne ment pas — elle est simplement FERMEE par trois portes
// successives, et cet instrument mesure LAQUELLE ferme.
//
// LES TROIS PORTES DE `buildVehicleRides`, dans l ordre ou elles s appliquent :
//
//	P1. LE TROU (`vehicleGapMinMS` = 3 s) : un embarquement plus court n ouvre aucun trou.
//	P2. LA GEOMETRIE (`vehicleNearestTo` : un echantillon de vehicule a moins de 1 s de l instant,
//	    ET a moins de 1,5 m EN PLAN). Deux exigences, pas une : la FRAICHEUR et la DISTANCE.
//	P3. LA VIE (`vehicleLifeAt`) : l instant doit tomber dans la fenetre d une vie du slot.
//
// CE QUE L INSTRUMENT MESURE, ET SON ORACLE. Il n existe pas de verite terrain « ce joueur a
// conduit ». Le plus proche dont on dispose est l EVENEMENT : un `biped_board_vehicle` ou un
// `unit_exit_vehicle` de la liste, dont la grammaire est portee et validee (V3 : occupant en
// bande 22/22 et 68/68, la sortie ferme le trou a +/-2 s dans 69/69 des cas). Un trou CONFIRME
// PAR EVENEMENT est donc un embarquement REEL, et la distribution des distances SUR CES
// TROUS-LA est ce qui calibre le rayon. Les trous non confirmes donnent le denominateur.
//
// LE TEMOIN, ecrit avant mesure : les MEMES positions de bipede, decalees de v4TemoinUS dans le
// temps. Un rayon qui rattache autant au temoin qu au reel ne mesure rien — il mesure la densite
// de vehicules sur la carte.

import (
	"fmt"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// v4TemoinUS est le decalage du temoin. 60 s : plus d une vie de joueur, moins qu une partie —
// le vehicule le plus proche a cet instant-la n a aucune raison d etre celui qu on a pris.
const v4TemoinUS = uint64(60_000_000)

// v4EventTolUS est la tolerance d appariement trou <-> evenement. MEME valeur que la production
// (`vehicleEventTolMS`) : l oracle doit se lire aux memes bornes que ce qu il calibre.
const v4EventTolUS = uint64(vehicleEventTolMS) * 1000

// v4SeuilsTrouMS sont les seuils de trou compares. 3 s est la production ; les deux autres
// disent ce qu elle rate.
var v4SeuilsTrouMS = []uint64{800, 1500, 3000}

// v4RayonsM sont les rayons compares, en plan. 1,5 m est la production.
var v4RayonsM = []float64{1.5, 3, 5, 8, 12}

// v4Classe agrege une population de trous.
type v4Classe struct {
	n         int
	sansVeh   int
	distances []float64
	// ageFrais compte les trous dont le vehicule le plus proche avait un echantillon de moins
	// d une seconde (la porte de FRAICHEUR de la production) ; ageSpawn ceux dont la position
	// tenue vient de la NAISSANCE (le vehicule n avait jamais bouge).
	ageFrais, ageSpawn int
	// parRayon compte, par rayon candidat, les trous rattaches.
	parRayon map[float64]int
}

func (c *v4Classe) init() {
	if c.parRayon == nil {
		c.parRayon = map[float64]int{}
	}
}

func (c *v4Classe) ajoute(d float64, age uint64, ok bool) {
	c.init()
	c.n++
	if !ok {
		c.sansVeh++
		return
	}
	c.distances = append(c.distances, d)
	switch {
	case age == v4AgeSpawn:
		c.ageSpawn++
	case age <= vehicleNearestSampleUS:
		c.ageFrais++
	}
	for _, r := range v4RayonsM {
		if d <= r {
			c.parRayon[r]++
		}
	}
}

func (c v4Classe) ligne(nom string) string {
	p := v4Percentiles(c.distances, 0.25, 0.5, 0.75, 0.9)
	s := fmt.Sprintf("%-22s n=%-5d sansVeh=%-4d frais<1s=%-4d parNaissance=%-4d "+
		"d25=%.1f d50=%.1f d75=%.1f d90=%.1f |", nom, c.n, c.sansVeh, c.ageFrais, c.ageSpawn,
		p[0], p[1], p[2], p[3])
	for _, r := range v4RayonsM {
		s += fmt.Sprintf(" R%.1f=%d", r, c.parRayon[r])
	}
	return s
}

// TestV4CouvertureEpisodes — ETAGE 1 : quelle porte ferme, et de combien.
func TestV4CouvertureEpisodes(t *testing.T) {
	root := v4Root(t)
	for _, f := range v4Corpus(t) {
		v4CouvertureUnFilm(t, root, f)
	}
}

func v4CouvertureUnFilm(t *testing.T, root string, f v0Film) {
	t.Helper()
	release := filmdec.LockProcessDecode()
	defer release()
	prev := filmdec.WorldObjectPrecision
	defer func() { filmdec.WorldObjectPrecision = prev }()
	ctx, ok := v4Decode(t, root, f)
	if !ok {
		return
	}
	boards, exits := vehicleEventsByOccupant(ctx.scan.Events)
	t.Logf("V4-COUV %s (%s) — vies=%d publiables=%d evenements=%d (embarquements=%d sorties=%d)",
		f.ID, f.Carte, len(ctx.lives), len(ctx.vehBySlot), len(ctx.scan.Events),
		v4Compte(boards), v4Compte(exits))
	for _, seuil := range v4SeuilsTrouMS {
		v4MesureSeuil(t, ctx, boards, exits, seuil)
	}
	v4MesureProduction(t, ctx)
}

// v4Compte somme les evenements d une table par occupant.
func v4Compte(m map[uint32][]filmdec.VehicleEvent) int {
	n := 0
	for _, v := range m {
		n += len(v)
	}
	return n
}

// v4MesureSeuil publie, pour UN seuil de trou, la distribution des distances des trous confirmes
// par evenement, des trous non confirmes, et du temoin decale.
func v4MesureSeuil(
	t *testing.T, ctx v4Ctx, boards, exits map[uint32][]filmdec.VehicleEvent, seuilMS uint64,
) {
	t.Helper()
	var confirme, autre, temoin v4Classe
	var horsVie int
	for _, g := range v4Gaps(ctx.bip, seuilMS) {
		_, d, age, ok := v4NearestHeld(ctx, g.last)
		if !ok {
			horsVie++
		}
		if v4Confirme(g, boards[g.slot], exits[g.slot]) {
			confirme.ajoute(d, age, ok)
		} else {
			autre.ajoute(d, age, ok)
		}
		dec := g.last
		dec.TimestampUS += v4TemoinUS
		_, dt, aget, okt := v4NearestHeld(ctx, dec)
		temoin.ajoute(dt, aget, okt)
	}
	t.Logf("V4-COUV %s trou>=%dms  %s", ctx.film.ID, seuilMS, confirme.ligne("CONFIRME-EVENEMENT"))
	t.Logf("V4-COUV %s trou>=%dms  %s", ctx.film.ID, seuilMS, autre.ligne("NON CONFIRME"))
	t.Logf("V4-COUV %s trou>=%dms  %s (aucune vie a cet instant : %d)",
		ctx.film.ID, seuilMS, temoin.ligne("TEMOIN +60s"), horsVie)
}

// v4Gaps releve les trous du flux bipede a un seuil DONNE (la production fige le sien).
func v4Gaps(bipeds []filmdec.BipedPosition, seuilMS uint64) []vehicleGap {
	bySlot := map[uint32][]filmdec.BipedPosition{}
	for _, b := range bipeds {
		if b.HasWorld {
			bySlot[b.Slot] = append(bySlot[b.Slot], b)
		}
	}
	slots := make([]uint32, 0, len(bySlot))
	for s := range bySlot {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	var out []vehicleGap
	for _, s := range slots {
		ech := bySlot[s]
		sort.SliceStable(ech, func(i, j int) bool { return ech[i].TimestampUS < ech[j].TimestampUS })
		for i := 1; i < len(ech); i++ {
			if (ech[i].TimestampUS-ech[i-1].TimestampUS)/1000 < seuilMS {
				continue
			}
			out = append(out, vehicleGap{
				slot: s, startUS: ech[i-1].TimestampUS, endUS: ech[i].TimestampUS, last: ech[i-1],
			})
		}
	}
	return out
}

// v4Confirme dit si un trou porte un evenement d embarquement a son ouverture ou de sortie a sa
// fermeture — l ORACLE du lot.
func v4Confirme(g vehicleGap, boards, exits []filmdec.VehicleEvent) bool {
	for _, ev := range boards {
		if gapUS(ev.TimestampUS, g.startUS) <= v4EventTolUS {
			return true
		}
	}
	for _, ev := range exits {
		if gapUS(ev.TimestampUS, g.endUS) <= v4EventTolUS {
			return true
		}
	}
	return false
}

// v4MesureProduction rejoue la chaine de production TELLE QUELLE et publie ses compteurs, pour
// que l ecart mesure ci-dessus se lise a cote du chiffre publie.
func v4MesureProduction(t *testing.T, ctx v4Ctx) {
	t.Helper()
	tracks, cov := buildVehicleTracks(ctx.scan, ctx.bip, ctx.own, ctx.clock)
	t.Logf("V4-COUV %s PRODUCTION — vies=%d publiees=%d episodes=%d nommes=%d avecSiege=%d"+
		" ambigus=%d (evenement=%d mixte=%d trou=%d)",
		ctx.film.ID, cov.Lives, len(tracks), cov.Rides, cov.RidesNamed, cov.RidesWithSeat,
		cov.Ambiguous, cov.RidesFromEvent, cov.RidesMixed, cov.RidesFromGap)
}
