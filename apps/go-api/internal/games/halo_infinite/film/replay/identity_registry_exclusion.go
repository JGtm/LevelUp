package replay

// identity_registry_exclusion.go — L'IDENTITE PAR EXCLUSION TEMPORELLE.
//
// # LE FAIT QU'ELLE FERME (lot P2-bis, temoin `d9781168`, journal RESTES_P2 §B)
//
// L'elimination sur le roster (identity_registry_elimination.go) raisonne sur LE MATCH ENTIER :
// « un seul xuid sans aucune vie, un seul slot sans aucune vie nommee ». Sur `d9781168` elle ne
// s'applique pas, et la mesure est sans appel : les huit joueurs de la feuille ont au moins
// 13 morts, donc ZERO xuid libre, et les 19 vies sans nom sont reparties sur 18 slots muets.
// La premisse du lot R2 (« un joueur qui ne meurt jamais du match ») etait fausse sur ce film.
//
// # CE QUE LA REGLE CONSTATE, ET ELLE NE SUPPOSE RIEN D'AUTRE
//
// UN JOUEUR N'OCCUPE QU'UN SEUL SLOT DE BIPEDE A LA FOIS. Une vie est l'occupation continue
// d'un slot par UN joueur (la decoupe la coupe a chaque trou de `lifeGapUS`, et les slots que
// deux joueurs se partagent sont marques AMBIGUS). Donc : tout joueur qu'une vie NOMMEE place
// ailleurs, ne serait-ce qu'une frame de l'intervalle, ne peut pas etre l'occupant. S'il ne
// reste qu'UN joueur du roster libre sur tout l'intervalle, la vie est la sienne — un
// appariement FORCE, pas une ressemblance ni une geometrie.
//
// # DEUX CANDIDATS = ON SE TAIT. ZERO CANDIDAT = ON SE TAIT AUSSI, ET C'EST UN SIGNAL
//
// Zero candidat signifie que la lecture se CONTREDIT : plus de vies que le film ne peut en
// porter, donc une vie nommee a tort ou une decoupe qui a fusionne deux sejours. Mesure :
// `d9781168` slot 641 `[5836..6170]`, les huit joueurs occupes ailleurs. On ne choisit pas, on
// compte et on alarme.
//
// # ELLE S'ITERE JUSQU'AU POINT FIXE
//
// Nommer une vie retire son occupant des candidats de toutes les vies qui la chevauchent. Deux
// tours suffisent sur `d9781168` (deux vies ne deviennent uniques qu'au second). Le point fixe
// est atteint des qu'un tour ne pose plus rien.
//
// # ELLE AJOUTE UNE PRESENCE, ELLE NE FABRIQUE PAS UNE MORT
//
// `cause` reste celle de la decoupe (`film_end`, `cut`), et les vies posees sont marquees
// DEDUITES : les lecteurs qui prouvent une ABSENCE (le gate de presence des portages) s'en
// abstiennent. Une deduction etablit qu'un joueur etait la, jamais qu'un autre n'y etait pas.

import (
	"log/slog"
	"sort"
)

// NomParExclusionTemporelle : la vie a ete nommee par EXCLUSION TEMPORELLE sur le roster.
// Quatrieme valeur de l'axe `nomPar` (cf. lives.go) — une deduction, et surtout PAS une mort.
const NomParExclusionTemporelle = "exclusion_temporelle"

// toursExclusionMax borne l'iteration. Un tour qui ne pose rien l'arrete, et une vie posee ne
// redevient jamais anonyme : la borne ne peut pas etre atteinte sur un film reel. Elle existe
// pour qu'une regression ne puisse pas faire boucler la cuisson.
const toursExclusionMax = 16

// vieForcee : l'indice d'une vie anonyme et l'unique joueur du roster qui puisse l'occuper.
type vieForcee struct {
	vie  int
	xuid uint64
}

// resolveByTemporalExclusion nomme les vies dont un SEUL joueur du roster est libre sur tout
// l'intervalle, et recommence tant qu'il en pose.
func (r *IdentityRegistry) resolveByTemporalExclusion(in IdentityInput) {
	roster, ok := universDeLExclusion(in, r.Vies())
	if !ok {
		return
	}
	for tour := 0; tour < toursExclusionMax; tour++ {
		if pose := r.poserExclusions(viesForcees(r.Vies(), roster), in); pose == 0 {
			break
		}
	}
	r.excludedContradictions = viesSansAucunCandidat(r.Vies(), roster)
	if r.excluded == 0 && r.excludedContradictions == 0 {
		return
	}
	slog.Info("rejeu : exclusion temporelle sur le roster",
		"match_id", in.MatchID, "viesNommees", r.excluded,
		"viesSansAucunCandidat", r.excludedContradictions)
	if r.excludedContradictions > 0 {
		slog.Warn("rejeu : des vies n'ont AUCUN occupant possible — la lecture se contredit",
			"match_id", in.MatchID, "vies", r.excludedContradictions)
	}
}

// poserExclusions pose les vies forcees d'un tour, apres avoir ecarte les CONFLITS : deux vies
// forcees sur le MEME joueur et qui se chevauchent ne peuvent pas etre toutes deux siennes, et
// rien ne dit laquelle l'est. Rend le nombre de vies posees.
func (r *IdentityRegistry) poserExclusions(forcees []vieForcee, in IdentityInput) int {
	lives, pose := r.Vies(), 0
	for _, f := range forcees {
		if conflitDExclusion(forcees, f, lives) {
			continue
		}
		pi, connu := in.PlayerIndices.ByXUID[f.xuid]
		if r.poserIdentiteDeVie(f.vie, f.xuid, pi, connu, NomParExclusionTemporelle) {
			r.excluded++
			pose++
		}
	}
	return pose
}

// conflitDExclusion dit qu'une autre vie forcee du meme tour revendique le meme joueur SUR UN
// INTERVALLE QUI CHEVAUCHE le sien.
func conflitDExclusion(forcees []vieForcee, f vieForcee, lives []lifeSpan) bool {
	for _, autre := range forcees {
		if autre.vie == f.vie || autre.xuid != f.xuid {
			continue
		}
		if seChevauchent(lives[f.vie], lives[autre.vie]) {
			return true
		}
	}
	return false
}

// viesForcees rend les vies anonymes dont UN SEUL joueur du roster est libre sur l'intervalle.
func viesForcees(lives []lifeSpan, roster []uint64) []vieForcee {
	var out []vieForcee
	for i := range lives {
		if lives[i].xuid != 0 {
			continue
		}
		if seul, n := candidatsLibres(lives, i, roster); n == 1 {
			out = append(out, vieForcee{vie: i, xuid: seul})
		}
	}
	return out
}

// viesSansAucunCandidat compte les vies anonymes qu'AUCUN joueur ne peut occuper. Ce n'est pas
// une abstention ordinaire : c'est une contradiction de la lecture, et elle s'alarme.
func viesSansAucunCandidat(lives []lifeSpan, roster []uint64) int {
	n := 0
	for i := range lives {
		if lives[i].xuid != 0 {
			continue
		}
		if _, libres := candidatsLibres(lives, i, roster); libres == 0 {
			n++
		}
	}
	return n
}

// candidatsLibres rend le nombre de joueurs du roster qu'aucune vie NOMMEE ne place ailleurs
// pendant l'intervalle de la vie `i`, et le dernier d'entre eux (utile quand il n'y en a qu'un).
//
// L'ORDRE DU ROSTER EST STABLE (`rosterCandidat` trie) : le candidat retenu ne depend jamais de
// l'ordre d'iteration d'une map — et de toute facon il n'y en a qu'un quand on le retient.
func candidatsLibres(lives []lifeSpan, i int, roster []uint64) (uint64, int) {
	occupes := map[uint64]bool{}
	for j := range lives {
		if j == i || lives[j].xuid == 0 || !seChevauchent(lives[i], lives[j]) {
			continue
		}
		occupes[lives[j].xuid] = true
	}
	var seul uint64
	libres := 0
	for _, x := range roster {
		if occupes[x] {
			continue
		}
		libres++
		seul = x
	}
	return seul, libres
}

// seChevauchent dit si deux vies partagent au moins un instant (bornes INCLUSIVES).
func seChevauchent(a, b lifeSpan) bool { return a.from <= b.to && b.from <= a.to }

// universDeLExclusion rend le roster sur lequel l'exclusion raisonne, et dit si elle a le droit
// de tourner. TROIS GARDE-FOUS, et ils refusent :
//
//  1. LE ROSTER DE LA FEUILLE EST OBLIGATOIRE. Sans lui, l'univers se reduit aux joueurs que le
//     fil des morts nomme : un joueur absent de l'univers ferait passer un candidat FAUX pour
//     unique. C'est la difference avec l'elimination sur le roster, qui n'ajoute un nom que
//     lorsqu'un xuid n'a AUCUNE vie — un cas que l'incompletude ne peut pas fabriquer.
//  2. AUCUN BOT DECLARE. Un bot occupe un slot de bipede sans porter de xuid : sa vie est
//     « anonyme » au sens du registre, et l'exclusion lui attribuerait un humain.
//  3. L'OCCUPATION NE DEPASSE PAS LE ROSTER. Plus de vies simultanees que de joueurs, c'est que
//     la decoupe ou le roster est faux — la premisse « un joueur, un slot » ne tient plus.
func universDeLExclusion(in IdentityInput, lives []lifeSpan) ([]uint64, bool) {
	if len(in.RosterXUIDs) == 0 || len(in.Bots) > 0 || len(lives) == 0 {
		return nil, false
	}
	roster := rosterCandidat(in)
	if len(roster) == 0 {
		return nil, false
	}
	if simultanees := occupationMaximale(lives); simultanees > len(roster) {
		slog.Warn("rejeu : exclusion temporelle ecartee — plus de vies simultanees que de joueurs",
			"match_id", in.MatchID, "viesSimultanees", simultanees, "roster", len(roster))
		return nil, false
	}
	return roster, true
}

// occupationMaximale rend le plus grand nombre de vies simultanees. Balayage des bornes : une
// vie ouvre a `from` et se ferme APRES `to`, les bornes etant inclusives.
func occupationMaximale(lives []lifeSpan) int {
	debuts := make([]int64, 0, len(lives))
	fins := make([]int64, 0, len(lives))
	for _, l := range lives {
		debuts = append(debuts, l.from)
		fins = append(fins, l.to+1)
	}
	sort.Slice(debuts, func(i, j int) bool { return debuts[i] < debuts[j] })
	sort.Slice(fins, func(i, j int) bool { return fins[i] < fins[j] })
	var cour, plafond int64
	j := 0
	for _, d := range debuts {
		for j < len(fins) && fins[j] <= d {
			cour--
			j++
		}
		cour++
		plafond = maxI64(plafond, cour)
	}
	return int(plafond)
}
