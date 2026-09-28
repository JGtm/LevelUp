package squademprise

// objects.go — LES OBJETS : un bonus ou une arme de socle, compté par camp, et chez nous par
// joueur (le reste du camp à part). Le même accumulateur sert au match et à la soirée.

import (
	"sort"

	"levelup/go-api/internal/domain"
)

// objectAcc — un objet en cours de comptage.
type objectAcc struct {
	resource, key string
	us, them      int
	// taken / kept / dropped : chez nous, par xuid ; la clé "" est le reste du camp.
	taken, kept, dropped map[string]int
	padsEmptied          int
	hasPads              bool
}

func newObjectAcc(resource, key string) *objectAcc {
	return &objectAcc{
		resource: resource, key: key,
		taken: map[string]int{}, kept: map[string]int{}, dropped: map[string]int{},
	}
}

// add ajoute une ligne : n prises (et, pour un bonus, kept / dropped perdus) par ce joueur.
// who est le xuid s'il est un joueur des fiches, "" pour le reste du camp ; ignoré côté eux.
func (o *objectAcc) add(us bool, who string, n, kept, dropped int) {
	if !us {
		o.them += n
		return
	}
	o.us += n
	o.taken[who] += n
	o.kept[who] += kept
	o.dropped[who] += dropped
}

func (o *objectAcc) merge(other *objectAcc) {
	o.us += other.us
	o.them += other.them
	for k, v := range other.taken {
		o.taken[k] += v
	}
	for k, v := range other.kept {
		o.kept[k] += v
	}
	for k, v := range other.dropped {
		o.dropped[k] += v
	}
	o.padsEmptied += other.padsEmptied
	o.hasPads = o.hasPads || other.hasPads
}

// objets — les objets d'un match ou d'une soirée.
type objets struct {
	byKey map[string]*objectAcc
}

func newObjets() *objets { return &objets{byKey: map[string]*objectAcc{}} }

func (s *objets) get(resource, key string) *objectAcc {
	id := resource + "|" + key
	o := s.byKey[id]
	if o == nil {
		o = newObjectAcc(resource, key)
		s.byKey[id] = o
	}
	return o
}

func (s *objets) merge(other *objets) {
	for _, o := range other.byKey {
		s.get(o.resource, o.key).merge(o)
	}
}

// total — les prises de chaque camp d'une ressource.
func (s *objets) total(resource string) domain.SquadEmpriseCount {
	var c domain.SquadEmpriseCount
	for _, o := range s.byKey {
		if o.resource == resource {
			c.Us += o.us
			c.Them += o.them
		}
	}
	return c
}

// has dit si la ressource a une trace : une prise, ou un socle de bonus vidé.
func (s *objets) has(resource string) bool {
	for _, o := range s.byKey {
		if o.resource == resource && (o.us+o.them > 0 || o.padsEmptied > 0) {
			return true
		}
	}
	return false
}

// publier rend les objets d'une ressource ("" = toutes), rangés : ressource, prises de notre
// camp, prises totales, clé.
func (s *objets) publier(resource string, players []domain.SessionUsageSquadPlayer, in *Input) []domain.SquadEmpriseObject {
	list := make([]*objectAcc, 0, len(s.byKey))
	for _, o := range s.byKey {
		if (resource == "" || o.resource == resource) && (o.us+o.them > 0 || o.padsEmptied > 0) {
			list = append(list, o)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		a, b := list[i], list[j]
		if ra, rb := resourceRank(a.resource), resourceRank(b.resource); ra != rb {
			return ra < rb
		}
		if a.us != b.us {
			return a.us > b.us
		}
		if a.us+a.them != b.us+b.them {
			return a.us+a.them > b.us+b.them
		}
		return a.key < b.key
	})
	out := make([]domain.SquadEmpriseObject, 0, len(list))
	for _, o := range list {
		out = append(out, publierObjet(o, players, in))
	}
	return out
}

// publierObjet projette un objet au contrat.
func publierObjet(o *objectAcc, players []domain.SessionUsageSquadPlayer, in *Input) domain.SquadEmpriseObject {
	bonus := o.resource == domain.EmpriseResourcePowerup
	obj := domain.SquadEmpriseObject{
		Resource: o.resource, Key: o.key,
		Taken: domain.SquadEmpriseCount{Us: o.us, Them: o.them},
		Squad: make([]domain.SquadEmpriseObjectShare, 0, len(players)+1),
	}
	if info, ok := in.Weapons[o.key]; ok && !bonus {
		obj.WeaponKey, obj.Label = info.WeaponKey, info.Label
	}
	if bonus && o.hasPads {
		n := o.padsEmptied
		obj.PadsEmptied = &n
	}
	for _, p := range players {
		obj.Squad = append(obj.Squad, partDe(o, p.XUID, bonus))
	}
	obj.Squad = append(obj.Squad, partDe(o, "", bonus))
	return obj
}

// partDe — la part d'un joueur des fiches (ou du reste du camp, who = "").
func partDe(o *objectAcc, who string, bonus bool) domain.SquadEmpriseObjectShare {
	part := domain.SquadEmpriseObjectShare{XUID: who, Taken: o.taken[who]}
	if bonus {
		kept, dropped := o.kept[who], o.dropped[who]
		part.Kept, part.Dropped = &kept, &dropped
	}
	return part
}
