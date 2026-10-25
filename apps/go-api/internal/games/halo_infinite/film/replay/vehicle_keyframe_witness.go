package replay

// vehicle_keyframe_witness.go — UNE IMAGE-CLE QUI SE VIDE D UN COUP N EST PAS UN TEMOIN D ABSENCE.
//
// `goneByUS` d une vie est la premiere image-cle apres son dernier recensement : sa premiere preuve
// d absence. Cette preuve suppose que l image-cle recense ce qu elle porte. Une image-cle qui passe de
// N >= 2 vehicules recenses a ZERO ne le prouve pas : les vehicules ne disparaissent pas tous dans la
// meme fenetre de vingt secondes, et ceux d une image-cle voisine sont revenus a la suivante (marche
// d ancres qui saute le corps des vehicules, `grammar/keyframe_world_attestation.go`). Lue comme une
// preuve, elle coupait la fenetre de chaque vie a cet instant : les echantillons s arretaient,
// la mort ecrite par le film tombait hors fenetre, le trajet du conducteur etait ferme trop tot.
//
// LA REGLE : la premiere preuve d absence est la premiere image-cle APRES le dernier recensement qui
// n est pas une image-cle videe d un coup. Une vie qu aucune image-cle temoin ne suit garde la
// premiere image-cle suivante : la regle ne fabrique jamais une fin de film. Le passage de UN
// vehicule a zero reste une fin (c est le plus souvent la destruction du dernier vehicule) ; il se
// distingue d une marche perdue par le seul recensement, que les faits persistes portent.
//
// Les deux effets se COMPTENT : `VehicleCoverage.KeyframesEmptiedAtOnce` (images-cles ignorees comme
// preuve) et `VehicleCoverage.EndsPastEmptiedKeyframe` (vies dont la preuve d absence a ete reportee).

import (
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// vehicleEmptiedAtOnceMinPrev est le recensement minimal de l image-cle precedente pour qu un
// recensement nul soit lu comme une marche perdue : un seul vehicule qui disparait est une fin
// plausible, deux ou plus ensemble ne le sont pas.
const vehicleEmptiedAtOnceMinPrev = 2

// vehicleCensusTally compte ce que la regle des images-cles videes a fait.
type vehicleCensusTally struct {
	// emptied : images-cles ignorees comme preuve d absence ; deferred : vies dont la premiere
	// preuve d absence est une autre image-cle que la premiere suivant leur dernier recensement.
	emptied, deferred int
}

// keyframesEmptiedAtOnce rend les instants des images-cles dont le recensement tombe de
// [vehicleEmptiedAtOnceMinPrev] vehicules ou plus a zero, et ceux des images-cles vides qui les
// suivent jusqu au prochain recensement non nul.
func keyframesEmptiedAtOnce(kf grammar.WorldObjectKeyframes) map[uint64]bool {
	count := make(map[uint64]int, len(kf.TimesUS))
	for _, seen := range kf.SeenUS {
		for _, t := range seen {
			count[t]++
		}
	}
	out := map[uint64]bool{}
	last := 0 // dernier recensement non nul : une suite d images-cles vides le garde
	for _, t := range kf.TimesUS {
		switch n := count[t]; {
		case n > 0:
			last = n
		case last >= vehicleEmptiedAtOnceMinPrev:
			out[t] = true
		}
	}
	return out
}

// firstWitnessAfter rend la premiere image-cle de `times` posterieure a `at` qui n est pas dans
// `emptied`, ou la premiere image-cle posterieure quand toutes le sont, ou zero quand il n en est
// aucune.
func firstWitnessAfter(times []uint64, emptied map[uint64]bool, at uint64) uint64 {
	first := firstTimeAfter(times, at)
	for i := sort.Search(len(times), func(k int) bool { return times[k] > at }); i < len(times); i++ {
		if !emptied[times[i]] {
			return times[i]
		}
	}
	return first
}
