package replay

// document_faille_pose.go — LA FAILLE, LUE DE SA POSE A SA FIN (doctrine en tete de
// `document_equipement_porte.go`) : l objet que la pose cree la situe, sinon la piste du porteur ; la
// suppression lue de cet objet la clot.

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// situerLesPoses pose la position et la fin de chaque faille. La position est celle de l'objet que la
// pose CRÉE (une vie engendrée née dans le paquet même de la pose, position i0 lue), sinon celle du
// porteur lue dans sa piste à l'image de la pose. La fin est la frame où le film écrit la SUPPRESSION
// de cet objet (`tracked-end` de la pile i28 qui le suivait), absente quand elle n'est pas lue : le
// client se replie alors sur la consommation ou la mort du porteur. `instants` porte l'horodatage du
// paquet de chaque pose.
func (o *objetLu) situerLesPoses(poses []RiftPose, instants []uint64, tracks []Track) {
	for k := range poses {
		n, cree := o.objetCreeA(instants[k])
		if cree {
			poses[k].T1 = o.finLue(n.Life)
		}
		if cree && n.HasPosition {
			x, y, z := n.X, n.Y, n.Z
			poses[k].X, poses[k].Y, poses[k].Z, poses[k].Src = &x, &y, &z, sourceFailleObjet
			continue
		}
		if p, ok := pointDuPorteur(tracks, poses[k].Slot, poses[k].T); ok {
			x, y, z := p.X, p.Y, p.Z
			poses[k].X, poses[k].Y, poses[k].Z, poses[k].Src = &x, &y, &z, sourceFaillePorteur
		}
	}
}

// objetCreeA rend l'objet engendré né dans le paquet d'horodatage `ts` : le premier positionné, sinon
// le premier lu.
func (o *objetLu) objetCreeA(ts uint64) (types.SpawnedObject, bool) {
	var premier types.SpawnedObject
	trouve := false
	for _, l := range o.lectures {
		if l.Kind == grammar.CarriedEffectBirth {
			continue // une candidate a l effet n est pas l objet que la pose cree
		}
		for _, n := range l.Spawned {
			if n.TimestampUS != ts {
				continue
			}
			if n.HasPosition {
				return n, true
			}
			if !trouve {
				premier, trouve = n, true
			}
		}
	}
	return premier, trouve
}

// finLue rend la frame de la suppression lue de l'objet engendré `vie`, ou nil.
func (o *objetLu) finLue(vie types.LifeKey) *int {
	k, ok := o.suivis[vie]
	if !ok || o.pub.Tracked[k].T1 == nil {
		return nil
	}
	t := *o.pub.Tracked[k].T1
	return &t
}

// pointDuPorteur rend le point de piste du slot à l'image `t` exactement.
func pointDuPorteur(tracks []Track, slot uint32, t int) (Point, bool) {
	for _, tr := range tracks {
		if tr.Slot != slot {
			continue
		}
		for _, p := range tr.Points {
			if p.T == t {
				return p, true
			}
		}
	}
	return Point{}, false
}
