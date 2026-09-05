package replay

// vehicules_v4_comobilite_test.go — INSTRUMENT DE MESURE (lot V4, etage 3) : UN BIPEDE
// SUIT-IL UN VEHICULE EN MOUVEMENT ? LECTURE SEULE, garde par V4_ROOT / V4_FILMS.
//
// POURQUOI CET ETAGE. L etage 1 a mesure que la primitive du TROU de position est deja au
// PLAFOND de ce qu elle peut rendre : sur `0d76e8f1`, 10 trous seulement sont confirmes par un
// evenement, et la production en publie 12 — elargir le rayon ou baisser le seuil de trou
// n ajoute AUCUN embarquement atteste, seulement du bruit. Si des trajets manquent, ils
// manquent parce qu ils N OUVRENT PAS DE TROU.
//
// L HYPOTHESE QUE CET ETAGE TESTE, et c est la seule qui reste : l occupant CONTINUE de
// repliquer sa position pendant tout ou partie du trajet. Dans ce cas il n y a pas de trou a
// trouver — mais il y a une signature bien plus forte : UN BIPEDE QUI RESTE COLLE A UN VEHICULE
// QUI ROULE. Un pieton ne suit pas un Warthog a 20 m/s ; s il y a un bipede a moins de quelques
// metres d un vehicule EN MOUVEMENT pendant plusieurs secondes, il est DEDANS.
//
// LE TEMOIN, ecrit avant mesure : les memes vehicules, les memes bipedes, decales de 60 s. Il
// mesure exactement la chance de trouver un bipede pres d un vehicule qui roule.

import (
	"fmt"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// Seuils de l etage 3, ecrits avant mesure.
const (
	// v4CoPasUS : sous-echantillonnage des instants de vehicule interroges (5 Hz). Le nuage
	// replique a ~60 Hz : l interroger a chaque record couterait 12 fois plus pour la meme
	// reponse — un occupant ne change pas de vehicule en 200 ms.
	v4CoPasUS = uint64(200_000)
	// v4CoBipTolUS : ecart temporel maximal accepte entre l instant du vehicule et
	// l echantillon de bipede compare. Au-dela, on compare deux instants differents.
	v4CoBipTolUS = uint64(150_000)
	// v4CoRayonM : distance EN PLAN sous laquelle un bipede est « sur » le vehicule. 3 m couvre
	// l ecart entre le repere d un Warthog et le siege de son tourelleur.
	v4CoRayonM = 3.0
	// v4CoRunMin : nombre d instants consecutifs colles exiges pour parler de trajet (5 Hz, donc
	// 5 s). Un croisement dure une fraction de seconde ; cinq secondes a moins de 3 m d un
	// vehicule qui roule ne s obtient qu a bord.
	v4CoRunMin = 25
)

// v4CoPaire agrege un couple (vie de vehicule, slot de bipede).
type v4CoPaire struct {
	vehSlot, vehGen, bipSlot uint32
	colles                   int
	plusLongRun              int
	premierUS, dernierUS     uint64
}

// v4CoMesureFilm publie l etage 3 pour UN film. Appele depuis `TestV4Diagnostic`, qui detient
// deja le decodage : ce fichier ne porte AUCUN test a lui, pour ne pas payer une seconde fois
// les trois minutes de decodage d un film.
func v4CoMesureFilm(t *testing.T, ctx v4Ctx) {
	t.Helper()
	bip := indexBySlot(ctx.bip)
	reel, instants := v4CoMesure(ctx, bip, 0)
	temoin, _ := v4CoMesure(ctx, bip, v4TemoinUS)
	t.Logf("V4-COMO %s (%s) — instants de vehicule EN MOUVEMENT interroges : %d",
		ctx.film.ID, ctx.film.Carte, instants)
	t.Logf("V4-COMO %s REEL   — couples colles>=%d : %d · trajets (run>=%d) : %d · %s",
		ctx.film.ID, v4CoRunMin, v4CoCouples(reel), v4CoRunMin, v4CoTrajets(reel),
		v4CoTop(reel, ctx))
	t.Logf("V4-COMO %s TEMOIN +60s — couples colles>=%d : %d · trajets : %d",
		ctx.film.ID, v4CoRunMin, v4CoCouples(temoin), v4CoTrajets(temoin))
}

// v4CoMesure parcourt les instants de vehicule EN MOUVEMENT et releve, pour chaque bipede colle,
// le couple correspondant. `decal` decale la lecture des bipedes (temoin).
func v4CoMesure(
	ctx v4Ctx, bip map[uint32]slotTrack, decal uint64,
) (map[[3]uint32]*v4CoPaire, int) {
	out := map[[3]uint32]*v4CoPaire{}
	runs := map[[3]uint32]int{}
	instants := 0
	slots := make([]uint32, 0, len(bip))
	for s := range bip {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	for _, l := range ctx.lives {
		vus := map[[3]uint32]bool{}
		for _, p := range v4CoInstants(ctx.vehBySlot[l.key.Slot], l) {
			instants++
			v4CoUnInstant(ctx, bip, slots, v4CoArg{life: l, veh: p, decal: decal},
				out, runs, vus)
			for k := range runs {
				if !vus[k] {
					runs[k] = 0
				}
			}
			for k := range vus {
				delete(vus, k)
			}
		}
	}
	return out, instants
}

// v4CoArg porte les entrees d UN instant (regle des 5 parametres du depot).
type v4CoArg struct {
	life  vehicleLife
	veh   filmdec.BipedPosition
	decal uint64
}

// v4CoUnInstant releve les bipedes colles au vehicule a CET instant.
func v4CoUnInstant(
	ctx v4Ctx, bip map[uint32]slotTrack, slots []uint32, a v4CoArg,
	out map[[3]uint32]*v4CoPaire, runs map[[3]uint32]int, vus map[[3]uint32]bool,
) {
	for _, s := range slots {
		q, d := bip[s].at(a.veh.TimestampUS + a.decal)
		if d > v4CoBipTolUS || !q.HasWorld {
			continue
		}
		if planDist(a.veh.X, a.veh.Y, q.X, q.Y) > v4CoRayonM {
			continue
		}
		k := [3]uint32{a.life.key.Slot, a.life.key.Gen, s}
		vus[k] = true
		runs[k]++
		p := out[k]
		if p == nil {
			p = &v4CoPaire{vehSlot: a.life.key.Slot, vehGen: a.life.key.Gen, bipSlot: s,
				premierUS: a.veh.TimestampUS}
			out[k] = p
		}
		p.colles++
		p.dernierUS = a.veh.TimestampUS
		if runs[k] > p.plusLongRun {
			p.plusLongRun = runs[k]
		}
	}
}

// v4CoInstants rend les instants EN MOUVEMENT d une vie, sous-echantillonnes.
func v4CoInstants(pts []filmdec.BipedPosition, l vehicleLife) []filmdec.BipedPosition {
	var out []filmdec.BipedPosition
	var dernier uint64
	for _, p := range pts {
		if p.TimestampUS < l.loUS || p.TimestampUS > l.hiUS {
			continue
		}
		if _, ok := vehicleHeadingOf(p); !ok {
			continue // sous 5 m/s : le vehicule est a l arret, un badaud n est pas un occupant
		}
		if dernier != 0 && p.TimestampUS-dernier < v4CoPasUS {
			continue
		}
		dernier = p.TimestampUS
		out = append(out, p)
	}
	return out
}

// v4CoCouples compte les couples dont le total d instants colles atteint le seuil.
func v4CoCouples(m map[[3]uint32]*v4CoPaire) int {
	n := 0
	for _, p := range m {
		if p.colles >= v4CoRunMin {
			n++
		}
	}
	return n
}

// v4CoTrajets compte les couples dont un run CONTINU atteint le seuil.
func v4CoTrajets(m map[[3]uint32]*v4CoPaire) int {
	n := 0
	for _, p := range m {
		if p.plusLongRun >= v4CoRunMin {
			n++
		}
	}
	return n
}

// v4CoTop rend les cinq couples les plus fournis, pour lire ce que la mesure a trouve.
func v4CoTop(m map[[3]uint32]*v4CoPaire, ctx v4Ctx) string {
	all := make([]*v4CoPaire, 0, len(m))
	for _, p := range m {
		all = append(all, p)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].colles > all[j].colles })
	s := "top:"
	for i, p := range all {
		if i >= 5 {
			break
		}
		nom := ""
		if x, ok := ctx.own.SlotXUID[p.bipSlot]; ok {
			nom = fmt.Sprintf("/%d", x)
		}
		s += fmt.Sprintf(" veh%d.%d<-bip%d%s(%d,run%d,%.0fs)",
			p.vehSlot, p.vehGen, p.bipSlot, nom, p.colles, p.plusLongRun,
			float64(p.dernierUS-p.premierUS)/1e6)
	}
	return s
}
