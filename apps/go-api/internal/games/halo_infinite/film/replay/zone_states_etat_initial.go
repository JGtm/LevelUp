package replay

// zone_states_etat_initial.go — L'ETAT D'UNE ZONE AVANT SON PREMIER CHANGEMENT, lu dans les
// images-cles.
//
// Une trame delta n'emet le canal de propriete qu'a son CHANGEMENT : une zone que la variante
// donne a un camp au coup d'envoi n'y apparait qu'a sa premiere reprise. L'image-cle, vidage de
// l'etat complet, porte la valeur des le depart (`grammar.ManagedPropertyScan.KeyReads`).
//
// CE QUE L'ETAT D'IMAGE-CLE FAIT, ET RIEN D'AUTRE :
//
//	OUVRIR PLUS TOT    les etats d'image-cle anterieurs a la premiere emission delta du canal elu
//	                   precedent ses emissions dans la serie que lisent les intervalles
//	                   ([ownerSpansOf]). Le premier intervalle commence a la frame de la premiere
//	                   image-cle qui dit un CAMP ; un etat neutre en tete n'ouvre rien (la zone
//	                   n'est a personne, comme avant sa premiere emission).
//	CONTROLER          les etats d'image-cle posterieurs se comparent a l'etat que les emissions
//	                   delta reconstituent ([zoneKeyTally]), et l'ecart se journalise.
//
// IL N'ENTRE DANS AUCUNE AUTRE SERIE : l'election du canal de propriete, celle du pousseur, la
// jauge, la colline et la jauge de retour du drapeau lisent les seules emissions delta.

import (
	"cmp"
	"slices"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// zoneKeyOwnerOf pose les etats d'image-cle du canal de propriete (tag 4) sur la grille de
// frames, par slot, dans l'ordre du temps.
//
// UNE IMAGE-CLE ANTERIEURE A L'ORIGINE DE L'AXE se pose a la frame 0 — la sienne n'existe pas —
// a deux conditions : c'est la DERNIERE image-cle avant l'origine pour ce slot, et aucune emission
// delta du canal ne tombe entre elle et l'origine (sinon l'etat a la frame 0 n'est plus le sien).
// Une image-cle au-dela de l'axe est ecartee, comme une emission delta.
func zoneKeyOwnerOf(keyReads, reads []grammar.ManagedPropertyRead, c zoneCtx) map[uint32][]zoneSample {
	out := map[uint32][]zoneSample{}
	if c.step == 0 || c.frames <= 0 {
		return out
	}
	avant := map[uint32]grammar.ManagedPropertyRead{} // derniere image-cle avant l'origine
	var retenues []grammar.ManagedPropertyRead
	for _, r := range keyReads {
		switch {
		case !zoneOwnerRead(r):
		case r.TimestampUS < c.origin:
			if prev, ok := avant[r.Slot]; !ok || r.TimestampUS >= prev.TimestampUS {
				avant[r.Slot] = r
			}
		default:
			retenues = append(retenues, r)
		}
	}
	for slot, k := range avant {
		if !deltaOwnerBetween(reads, slot, k.TimestampUS, c.origin) {
			retenues = append(retenues, k)
		}
	}
	// L'ORDRE EST CELUI DES HORODATAGES : une image-cle anterieure a l'origine et une image-cle de
	// la frame 0 tombent sur la meme frame, et la plus ancienne doit preceder.
	slices.SortStableFunc(retenues, func(a, b grammar.ManagedPropertyRead) int {
		return cmp.Compare(a.TimestampUS, b.TimestampUS)
	})
	for _, r := range retenues {
		t := 0
		if r.TimestampUS >= c.origin {
			f, ok := zoneFrameOf(r.TimestampUS, c)
			if !ok {
				continue
			}
			t = f
		}
		out[r.Slot] = append(out[r.Slot], zoneSample{t: t, v: r.Value})
	}
	return out
}

// zoneOwnerRead dit qu'une lecture porte une valeur du canal de propriete : scalaire, tag 4.
func zoneOwnerRead(r grammar.ManagedPropertyRead) bool {
	return r.Field == grammar.ManagedPropertyScalar && r.HasValue && r.Tag == grammar.ManagedPropertyTagU32
}

// deltaOwnerBetween dit qu'une emission delta du canal de propriete de `slot` tombe dans
// [from, to).
func deltaOwnerBetween(reads []grammar.ManagedPropertyRead, slot uint32, from, to uint64) bool {
	for _, r := range reads {
		if r.Slot == slot && zoneOwnerRead(r) && r.TimestampUS >= from && r.TimestampUS < to {
			return true
		}
	}
	return false
}

// zoneKeyTally compte ce que l'etat d'image-cle a fait au calque, pour le journal.
type zoneKeyTally struct {
	// opened : zones publiees dont le premier intervalle s'ouvre sur un etat d'image-cle.
	opened int
	// checked / agreed : etats d'image-cle posterieurs a la premiere emission delta, compares a
	// l'etat que les emissions reconstituent, et ceux qui s'accordent.
	checked, agreed int
}

// ownerSeriesWithInitialState rend la serie que lisent les intervalles : les etats d'image-cle
// ANTERIEURS a la premiere emission delta, prives de leurs etats neutres de tete, puis les
// emissions delta. Sans etat d'image-cle utile, rend `delta` tel quel.
//
// UN ETAT D'IMAGE-CLE QUI N'EST NI NEUTRE NI UN CAMP DU ROSTER ([zoneOwnerTeam]) ECARTE TOUT
// L'ETAT INITIAL de la zone : la serie redevient celle des emissions, et l'etat d'image-cle ne
// s'ajoute pas aux valeurs inconnues que la couverture compte (`unknownOwner` compte des
// emissions).
func ownerSeriesWithInitialState(delta, key []zoneSample, teams map[uint64]bool) []zoneSample {
	first := -1
	if len(delta) > 0 {
		first = delta[0].t
	}
	var prefix []zoneSample
	for _, k := range key {
		if first >= 0 && k.t >= first {
			break
		}
		if _, known := zoneOwnerTeam(k.v, teams); !known {
			return delta
		}
		if len(prefix) == 0 && k.v == zoneNeutralOwner {
			continue
		}
		prefix = append(prefix, k)
	}
	if len(prefix) == 0 {
		return delta
	}
	return append(prefix, delta...)
}

// tallyKeyAgreement compare chaque etat d'image-cle posterieur a la premiere emission delta a la
// valeur que les emissions reconstituent a sa frame (la derniere emission strictement anterieure).
// Un etat d'image-cle a la frame meme d'une emission n'est pas compare : l'ordre des deux dans la
// frame n'est pas connu.
func tallyKeyAgreement(delta, key []zoneSample, t *zoneKeyTally) {
	if len(delta) == 0 {
		return
	}
	for _, k := range key {
		if k.t < delta[0].t {
			continue
		}
		i, exact := slices.BinarySearchFunc(delta, k.t, func(s zoneSample, f int) int { return cmp.Compare(s.t, f) })
		if exact {
			continue
		}
		t.checked++
		if delta[i-1].v == k.v {
			t.agreed++
		}
	}
}
