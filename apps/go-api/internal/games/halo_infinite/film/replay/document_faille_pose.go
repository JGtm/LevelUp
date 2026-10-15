package replay

// document_faille_pose.go — LA POSITION DE LA FAILLE, LUE A LA POSE (doctrine en tete de
// `document_equipement_porte.go`) : l objet que la pose cree, sinon la piste du porteur.

import "levelup/go-api/internal/games/halo_infinite/film/types"

// situerLesPoses pose la position de chaque faille : celle de l'objet que la pose CRÉE (une vie
// engendrée née dans le paquet même de la pose, position i0 lue), sinon celle du porteur lue dans sa
// piste à l'image de la pose. `instants` porte l'horodatage du paquet de chaque pose.
func (o *objetLu) situerLesPoses(poses []RiftPose, instants []uint64, tracks []Track) {
	for k := range poses {
		if n, ok := o.objetCreeA(instants[k]); ok {
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

// objetCreeA rend l'objet engendré, positionné, né dans le paquet d'horodatage `ts`.
func (o *objetLu) objetCreeA(ts uint64) (types.SpawnedObject, bool) {
	for _, l := range o.lectures {
		for _, n := range l.Spawned {
			if n.TimestampUS == ts && n.HasPosition {
				return n, true
			}
		}
	}
	return types.SpawnedObject{}, false
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
