package replayverite

// violations_vies.go — CE QU'UN JOUEUR NE PEUT PAS FAIRE HORS DE SA VIE OU HORS DU MATCH.
//
// V-3 action hors vie, V-4 deux corps, V-5 acteur absent ou mort, V-8 mort sans fin de vie.
// Ces classes detectent des FAUX : une vie manquante n'en produit pas (l'action n'a alors pas de
// slot vivant et compte ici — c'est voulu : une action publiee sur un slot sans vie est fausse, que
// le faux soit l'action ou l'absence de la vie).

import (
	"fmt"
	"sort"
)

// intervalle est une vie [debut, fin] en images.
type intervalle struct{ a, b int }

// couvre dit si l'intervalle couvre t a `tol` images pres.
func (i intervalle) couvre(t, tol int) bool { return t >= i.a-tol && t <= i.b+tol }

// vies indexe les vies publiees par slot et par xuid.
type vies struct {
	parSlot map[int][]intervalle
	parXUID map[string][]intervalle
}

func indexerVies(d *Document) vies {
	v := vies{parSlot: map[int][]intervalle{}, parXUID: map[string][]intervalle{}}
	for _, t := range d.Tracks {
		i := intervalle{a: t.Debut(), b: t.EndFrame}
		v.parSlot[t.Slot] = append(v.parSlot[t.Slot], i)
		if t.XUID != "" {
			v.parXUID[t.XUID] = append(v.parXUID[t.XUID], i)
		}
	}
	return v
}

func couvertPar(is []intervalle, t, tol int) bool {
	for _, i := range is {
		if i.couvre(t, tol) {
			return true
		}
	}
	return false
}

// slotMuet : le slot 0 d'une action dit « pont muet » — le lanceur n'est pas connu
// (`film/replay/grenades.go`, champ Slot). Ce n'est pas un joueur : l'action n'est pas jugee.
const slotMuet = 0

// violationsHorsVie : V-3, une action dont le slot n'a aucune vie qui couvre son image.
//
// UN TRAJET DU SLOT A BORD D UN VEHICULE FAIT PARTIE DE SA VIE. Le corps d un occupant cesse de
// repliquer sa position a la montee : sa piste s arrete, et il vit pourtant jusqu a la descente ou a
// sa mort. Le trajet se rattache par SLOT, quel que soit le siege, comme le tir d un occupant
// (regle de l utilisateur du 2026-09-21).
func violationsHorsVie(d *Document) Violation {
	v := indexerVies(d)
	trajets := trajetsParSlot(d)
	var out Violation
	familles := []struct {
		nom     string
		actions []Action
	}{
		{"tir", d.Shots}, {"grenade", d.Grenades}, {"ramassage", d.Pickups},
		{"changement d'arme", d.WeaponChanges}, {"capacite", d.Abilities},
		{"changement d'equipement", d.EquipmentChanges},
	}
	for _, f := range familles {
		for _, a := range f.actions {
			if a.Slot == nil || *a.Slot == slotMuet || couvertPar(v.parSlot[*a.Slot], a.T, toleranceVieImages) ||
				couvertPar(trajets[*a.Slot], a.T, toleranceVieImages) {
				continue
			}
			out.Instances = append(out.Instances, fmt.Sprintf("%s slot %d @%d", f.nom, *a.Slot, a.T))
		}
	}
	sort.Strings(out.Instances)
	return out
}

// trajetsParSlot indexe les trajets publies a bord des vehicules par slot de corps.
func trajetsParSlot(d *Document) map[int][]intervalle {
	out := map[int][]intervalle{}
	for _, veh := range d.Vehicles {
		for _, r := range veh.Rides {
			out[r.Slot] = append(out[r.Slot], intervalle{a: r.T0, b: r.T1})
		}
	}
	return out
}

// violationsDeuxCorps : V-4, deux vies d'un meme xuid qui se recouvrent dans le temps.
func violationsDeuxCorps(d *Document) Violation {
	v := indexerVies(d)
	var out Violation
	for _, x := range clesTriees(v.parXUID) {
		is := append([]intervalle(nil), v.parXUID[x]...)
		sort.Slice(is, func(i, j int) bool { return is[i].a < is[j].a || (is[i].a == is[j].a && is[i].b < is[j].b) })
		finMax := is[0].b
		for k := 1; k < len(is); k++ {
			if is[k].a <= finMax {
				out.Instances = append(out.Instances,
					fmt.Sprintf("%s vies @%d et @%d (%d images)", x, is[k-1].a, is[k].a, min(finMax, is[k].b)-is[k].a+1))
			}
			finMax = max(finMax, is[k].b)
		}
	}
	return out
}

// acteur est un joueur nomme a un instant, pour V-5.
type acteur struct {
	famille  string
	xuid     string
	t        int
	vivantEn []int // images ou le joueur doit etre VIVANT (vide = presence seule)
}

// violationsAbsents : V-5, un ramassage, une action d'objectif ou un portage par un joueur hors de sa
// presence au roster ; un portage par un joueur sans vie a sa prise ou a sa fin. Une action
// d'objectif n'exige pas la vie (un kill par grenade apres la mort est legitime), la presence si.
func violationsAbsents(d *Document) Violation {
	v := indexerVies(d)
	presence := map[string][]Presence{}
	for _, r := range d.Roster {
		presence[r.XUID] = append(presence[r.XUID], r.Presence...)
	}
	var out Violation
	for _, a := range acteursNommes(d) {
		if len(d.Roster) > 0 && !present(presence[a.xuid], a.t) {
			out.Instances = append(out.Instances, fmt.Sprintf("%s %s @%d : hors presence", a.famille, a.xuid, a.t))
			continue
		}
		for _, t := range a.vivantEn {
			if !couvertPar(v.parXUID[a.xuid], t, toleranceVieImages) {
				out.Instances = append(out.Instances, fmt.Sprintf("%s %s @%d : sans vie", a.famille, a.xuid, t))
				break
			}
		}
	}
	sort.Strings(out.Instances)
	return out
}

// etatDrapeauPorte : l'etat d'un drapeau qui nomme un porteur.
const etatDrapeauPorte = "carried"

func acteursNommes(d *Document) []acteur {
	var out []acteur
	for _, p := range d.Pickups {
		if p.XUID != "" {
			out = append(out, acteur{famille: "ramassage", xuid: p.XUID, t: p.T})
		}
	}
	for _, o := range d.Objectives {
		if o.XUID != "" {
			out = append(out, acteur{famille: "objectif " + o.Stat, xuid: o.XUID, t: o.T})
		}
	}
	for _, f := range d.FlagCarries {
		for _, s := range f.Spans {
			if s.State == etatDrapeauPorte && s.XUID != nil && *s.XUID != "" {
				out = append(out, acteur{famille: "portage drapeau", xuid: *s.XUID, t: s.T0, vivantEn: []int{s.T0, s.T1}})
			}
		}
	}
	for _, g := range [][]Portage{d.SkullCarries, d.BombCarries} {
		for _, p := range g {
			if p.XUID != "" {
				out = append(out, acteur{famille: "portage", xuid: p.XUID, t: p.T0, vivantEn: []int{p.T0, p.T1}})
			}
		}
	}
	return out
}

func present(ps []Presence, t int) bool {
	for _, p := range ps {
		if t >= p.From && t <= p.Fin() {
			return true
		}
	}
	return false
}

// violationsMortsSansVie : V-8, une mort du statborg (increment de la courbe `deaths`) sans fin de
// vie du meme joueur a ±toleranceMortImages. Un joueur sans AUCUNE vie nommee n'est pas compte :
// c'est un manque de nommage, pas un faux.
func violationsMortsSansVie(d *Document) Violation {
	var out Violation
	if d.ScoreTimeline == nil {
		return out
	}
	fins := map[string][]int{}
	for _, t := range d.Tracks {
		if t.XUID != "" {
			fins[t.XUID] = append(fins[t.XUID], t.EndFrame)
		}
	}
	for _, p := range d.ScoreTimeline.Players {
		f, ok := fins[p.XUID]
		if !ok {
			continue
		}
		pts := p.Deaths.Total
		for i := range pts {
			if (i > 0 && pts[i].V <= pts[i-1].V) || (i == 0 && pts[i].V <= 0) {
				continue
			}
			if !finProche(f, pts[i].T) {
				out.Instances = append(out.Instances, fmt.Sprintf("%s mort @%d", p.XUID, pts[i].T))
			}
		}
	}
	sort.Strings(out.Instances)
	return out
}

func finProche(fins []int, t int) bool {
	for _, e := range fins {
		if t-e >= -toleranceMortImages && t-e <= toleranceMortImages {
			return true
		}
	}
	return false
}
