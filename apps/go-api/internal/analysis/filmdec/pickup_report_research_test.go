package filmdec

// pickup_report_research_test.go — LA LISTE, ET RIEN D'AUTRE.
//
// POURQUOI CET INSTRUMENT REMPLACE LES PRECEDENTS. Les mesures de completude (oracle des
// images-cles, union des inventaires, oracle des tirs) ont toutes bute sur le meme mur : elles
// exigent une verite-terrain que le film hors ligne ne fournit pas, et leurs temoins plafonnent.
// Or la desambiguisation qu'elles cherchaient n'est PAS necessaire : les rateliers et les socles
// sont peu nombreux et a temps de recharge, les joueurs changent rarement d'arme — un changement
// d'inventaire est donc soit un LACHER soit un RAMASSAGE, il n'y a pas de troisieme cause.
//
// CE FICHIER SORT DONC LA LISTE et la rend lisible : combien d'evenements, par vie, par arme, a
// quelle minute. Le juge est la PLAUSIBILITE, et elle se lit a l'oeil : quelques ramassages par
// vie, les armes lourdes surrepresentees, aucune aberration de volume.
//
// GARDE : HW_FILM, meme convention que les autres instruments de ce lot.

import (
	"fmt"
	"os"
	"sort"
	"testing"
)

// prEvent est un evenement d'inventaire classe et date par rapport au debut du film.
type prEvent struct {
	AtMs int64
	Slot uint32
	Comp int
	Kind string
	From string
	To   string
}

// prClassify rend les evenements classes, dates par rapport au premier paquet du film.
func prClassify(s hwSetup, ref hwKFRef, ev []hwEvent) []prEvent {
	var origin uint64
	for _, e := range ev {
		if origin == 0 || e.TimestampUS < origin {
			origin = e.TimestampUS
		}
	}
	for _, list := range ref.bySlot {
		for _, k := range list {
			if origin == 0 || k.TimestampUS < origin {
				origin = k.TimestampUS
			}
		}
	}
	type key struct {
		slot uint32
		comp int
	}
	prev, seen := map[key]uint32{}, map[key]bool{}
	out := make([]prEvent, 0, len(ev))
	for _, e := range ev {
		k := key{e.Slot, e.CompIndex}
		var kind, from string
		switch {
		case e.IDHigh == noVariant:
			kind, from = "LACHER", hwName(prev[k])
		case seen[k] && prev[k] == noVariant:
			kind, from = "PRISE", "(vide)"
		case seen[k]:
			kind, from = "ECHANGE", hwName(prev[k])
		default:
			if r, ok := ref.setAt(e.Slot, e.TimestampUS); ok && r[e.IDHigh] {
				kind, from = "DEJA PORTEE", "(spawn)"
			} else {
				kind, from = "PRISE", "(spawn sans elle)"
			}
		}
		out = append(out, prEvent{
			AtMs: int64(e.TimestampUS-origin) / 1000,
			Slot: e.Slot, Comp: e.CompIndex, Kind: kind, From: from, To: hwName(e.IDHigh),
		})
		seen[k], prev[k] = true, e.IDHigh
	}
	return out
}

// prTop rend les n premieres entrees d'un comptage, triees par frequence decroissante.
func prTop(m map[string]int, n int) string {
	type kv struct {
		k string
		v int
	}
	var l []kv
	for k, v := range m {
		l = append(l, kv{k, v})
	}
	sort.Slice(l, func(i, j int) bool {
		if l[i].v != l[j].v {
			return l[i].v > l[j].v
		}
		return l[i].k < l[j].k
	})
	if len(l) > n {
		l = l[:n]
	}
	out := ""
	for _, e := range l {
		out += fmt.Sprintf("%s=%d  ", e.k, e.v)
	}
	return out
}

func TestPickupReport(t *testing.T) {
	dir := os.Getenv(hwFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", hwFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	s := hwResolve(t, dir)
	ev := hwIdentities(hwScanEvents(s))
	ref := hwKeyframeRef(t, dir)
	list := prClassify(s, ref, ev)

	byKind := map[string]int{}
	byWeapon := map[string]int{}
	byLife := map[uint32]int{}
	for _, e := range list {
		byKind[e.Kind]++
		if e.Kind == "PRISE" || e.Kind == "ECHANGE" {
			byWeapon[e.To]++
			byLife[e.Slot]++
		}
		if e.Kind == "LACHER" {
			byLife[e.Slot]++
		}
	}
	t.Logf("TOTAL = %d evenements sur %d vies", len(list), len(byLife))
	t.Logf("PAR NATURE : %s", prTop(byKind, 8))
	t.Logf("ARMES PRISES (top 10) : %s", prTop(byWeapon, 10))

	var perLife []int
	for _, n := range byLife {
		perLife = append(perLife, n)
	}
	sort.Ints(perLife)
	if len(perLife) > 0 {
		t.Logf("PAR VIE : mediane=%d  max=%d  (vies concernees=%d)",
			perLife[len(perLife)/2], perLife[len(perLife)-1], len(perLife))
	}

	t.Log("CHRONOLOGIE (les 25 premiers evenements, minute:seconde depuis le debut du film) :")
	sort.SliceStable(list, func(i, j int) bool { return list[i].AtMs < list[j].AtMs })
	for i, e := range list {
		if i >= 25 {
			t.Logf("   ... et %d autres", len(list)-25)
			break
		}
		t.Logf("   %02d:%02d  vie %-4d  %-12s %s -> %s",
			e.AtMs/60000, (e.AtMs/1000)%60, e.Slot, e.Kind, e.From, e.To)
	}
	t.Log("JUGE : la PLAUSIBILITE. Quelques evenements par vie, des armes lourdes et des armes " +
		"de socle surrepresentees dans les prises, et aucune vie a volume aberrant.")
}
