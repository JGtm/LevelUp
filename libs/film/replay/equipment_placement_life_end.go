package replay

// equipment_placement_life_end.go — LA FIN LUE D UNE POSE : la suppression de l objet que la pose a cree
// (`EquipmentPlacement.LifeEnd`, schema 106).
//
// LE LIEN EST STRUCTUREL, PAS TEMPOREL. Le panneau du mur est une vie ti=37 (slot, generation) que le
// balayage des poses rend (`EquipmentPlacement.Life`) et que l evenement 103 designe. La pile i28 de
// l objet porte qui l a pose NOMME cette MEME vie (`CarriedEquipmentRead.Tracked`) dans un paquet voisin :
// mesure du plan des durees d effets (etape B.1, trois films, 67 murs suivis) : meme vie pour 66 sur 67,
// nommee 16 a 45 ms APRES la creation de la pose ; le seul mur suivi sans pose designee n a pas de 103.
// La generation ne tient que sur 2 bits et la cle revient dans le match : le lien exige donc, en plus de la
// cle, que le nommage SUIVE la creation de la pose d au plus [equipmentSpawnMatchUS] (la borne de la
// designation par le 103, meme raison). Aucune jointure par proximite : une pose sans cle nommee n a pas de
// fin lue. Plusieurs nommages dans la fenetre pour la meme cle : AMBIGU, rien n est publie.
//
// LA FIN est la suppression ecrite de l objet suivi (`tracked-end`) : la premiere qui suit le nommage.

import (
	"github.com/JGtm/LevelUp/libs/film/internal/grammar"
	"github.com/JGtm/LevelUp/libs/film/types"
)

// finDePose est le resultat du lien d UNE pose brute.
type finDePose struct {
	// etat : 0 aucun objet suivi, 1 lie, 2 ambigu.
	etat int
	// ts : l instant de la suppression lue, 0 quand elle n est pas lue.
	ts uint64
}

const (
	liaisonAucune = iota
	liaisonUnique
	liaisonAmbigue
)

// vieNommee est une vie que la pile i28 d un objet porte a designee : l instant du nommage, et celui de sa
// suppression (0 : non lue).
type vieNommee struct{ nomme, fin uint64 }

// finsLuesDesPoses lie chaque pose brute a l objet suivi de meme vie, parallelement a `raw`.
func finsLuesDesPoses(raw []types.EquipmentPlacement, reads []types.CarriedEquipmentRead) []finDePose {
	out := make([]finDePose, len(raw))
	if len(reads) == 0 {
		return out
	}
	vies := map[types.LifeKey][]*vieNommee{}
	ouvertes := map[types.LifeKey]*vieNommee{}
	for _, l := range reads {
		switch l.Kind {
		case grammar.CarriedEffectBirth, grammar.CarriedEffectEnd:
			continue
		case types.CarriedTrackedEnd:
			for _, k := range l.Tracked {
				if v := ouvertes[k]; v != nil {
					v.fin = l.TimestampUS
					delete(ouvertes, k)
				}
			}
		default:
			for _, k := range l.Tracked {
				if ouvertes[k] == nil {
					v := &vieNommee{nomme: l.TimestampUS}
					ouvertes[k] = v
					vies[k] = append(vies[k], v)
				}
			}
		}
	}
	for i, p := range raw {
		var lies []*vieNommee
		for _, v := range vies[p.Life] {
			if v.nomme >= p.T0US && v.nomme-p.T0US <= equipmentSpawnMatchUS {
				lies = append(lies, v)
			}
		}
		switch len(lies) {
		case 0:
		case 1:
			out[i] = finDePose{etat: liaisonUnique, ts: lies[0].fin}
		default:
			out[i] = finDePose{etat: liaisonAmbigue}
		}
	}
	return out
}

// publier compte le lien dans la couverture et rend la frame de fin lue (nil quand le lien manque, est
// ambigu ou que la suppression n est pas lue), bornee a l axe publie.
func (f finDePose) publier(cov *EquipmentPlacementCoverage, clock replayClock) *int {
	switch f.etat {
	case liaisonAmbigue:
		cov.LifeAmbiguous++
		return nil
	case liaisonUnique:
		cov.LifeLinked++
	default:
		return nil
	}
	if f.ts == 0 || f.ts < clock.origin {
		return nil
	}
	t := clampFrame(frameOf(f.ts, clock.origin, clock.step), clock.frames)
	cov.LifeEnds++
	return &t
}
