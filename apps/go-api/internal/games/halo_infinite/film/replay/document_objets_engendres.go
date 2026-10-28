package replay

// document_objets_engendres.go — LES OBJETS QU'UN ÉQUIPEMENT PORTÉ ENGENDRE (capteur, traqueur, écran
// occultant, champ de réparation, faille...), AVEC LEUR POSITION DE REPOS.
//
// LA PILE i28 DE L'OBJET PORTÉ NOMME l'objet engendré par sa vie (`types.LifeKey` : slot et
// génération). Cet objet est un projectile (archétype ti=41) : le film écrit sa trajectoire sous la
// MÊME clé (`types.ProjectileTrack{Slot, Gen}`). Sa position de repos est donc le dernier point
// PUBLIÉ de la trajectoire de sa vie ([volPublie], la règle même qui publie `projectiles`) : un vol
// coupé à un pas impossible s'arrête à la coupure, une trajectoire que le rejeu ne publie pas ne
// situe rien.
//
// LA CLÉ NE SUFFIT PAS : la génération d'une vie tient sur quelques bits, le même couple (slot,
// génération) revient donc plus tard dans le match pour un autre projectile. L'objet est situé par la
// trajectoire de sa vie NÉE À SON APPARITION (à [toleranceDeNaissance] frames de la pile qui le nomme,
// l'objet naissant dans ce paquet), la plus proche s'il y en a plusieurs. Une vie sans une telle
// trajectoire publiée est publiée sans position : `land`, `x`, `y`, `z` sont absents ENSEMBLE, et la
// couverture la compte (`coverage.carriedEquipment.trackedUnpositioned`). Aucune position ne vient du
// porteur.
//
// Ce calque ne change ni `projectiles`, ni `equipmentPlacements`, ni `riftPoses`, ni les résumés
// d'usage : il se calcule à la publication, depuis les trajectoires déjà lues des faits.

import "levelup/go-api/internal/games/halo_infinite/film/types"

// CarriedTracked est un objet engendré par un équipement porté. T0 est la frame de la première pile
// i28 qui le nomme (il naît dans ce paquet) ; T1 celle de sa SUPPRESSION, écrite par le film,
// absente quand elle n'est pas lue.
type CarriedTracked struct {
	T0 int  `json:"t0"`
	T1 *int `json:"t1,omitempty"`
	// Land est la frame du dernier point publié de la trajectoire de l'objet ; X/Y/Z sa position à
	// cette frame, en mètres, SOLIDAIRES de Land (z est celle de l'échantillon brut). Absents ensemble
	// quand aucune trajectoire publiée ne porte la vie de l'objet.
	Land *int     `json:"land,omitempty"`
	X    *float32 `json:"x,omitempty"`
	Y    *float32 `json:"y,omitempty"`
	Z    *float32 `json:"z,omitempty"`
}

// toleranceDeNaissance borne l'écart, en frames, entre la frame où la pile i28 nomme un objet engendré
// et la naissance de la trajectoire qui le situe. L'objet naît dans le paquet de la pile ; la
// génération d'une vie ne tient que sur quelques bits, donc le même couple (slot, génération) revient
// plus tard dans le match pour un autre projectile : sans cette borne, la dernière trajectoire de la
// clé situerait l'objet à un autre moment du match.
const toleranceDeNaissance = 3

// reposEngendre est une trajectoire publiée d'une vie : sa frame de naissance, et la position de son
// dernier point publié.
type reposEngendre struct {
	naissance int
	land      int
	x, y, z   float32
}

// reposParVie indexe, pour chaque vie dont une trajectoire est publiée, ses trajectoires (la
// position de repos est le dernier point publié de l'une d'elles), dans l'ordre des pistes.
func reposParVie(tracks []types.ProjectileTrack, origin, step uint64) map[types.LifeKey][]reposEngendre {
	if step == 0 {
		return nil
	}
	out := map[types.LifeKey][]reposEngendre{}
	for _, tr := range tracks {
		v, lu := volPublie(tr, origin, step)
		if !lu || !v.publie() {
			continue
		}
		cle := types.LifeKey{Slot: tr.Slot, Gen: tr.Gen}
		dernier := v.pts[len(v.pts)-1]
		out[cle] = append(out[cle], reposEngendre{naissance: v.t0, land: v.t0 + int(dernier[0]),
			x: round2(v.dernier.X), y: round2(v.dernier.Y), z: round2(v.dernier.Z)})
	}
	return out
}

// reposNeAPeuPres rend la trajectoire de la vie née le plus près de la frame `t0` de l'objet, à
// [toleranceDeNaissance] frames au plus (la première à égalité).
func reposNeAPeuPres(cands []reposEngendre, t0 int) (reposEngendre, bool) {
	var best reposEngendre
	ecart, trouve := toleranceDeNaissance+1, false
	for _, c := range cands {
		e := c.naissance - t0
		if e < 0 {
			e = -e
		}
		if e < ecart {
			best, ecart, trouve = c, e, true
		}
	}
	return best, trouve
}

// situerLesObjetsEngendres pose la position de repos de chaque objet engendré dont une trajectoire
// publiée, née à son apparition, porte la vie, et compte les objets situés et ceux qui ne le sont pas.
func (o *objetLu) situerLesObjetsEngendres(repos map[types.LifeKey][]reposEngendre, cov *CarriedEquipmentCoverage) {
	situes := 0
	for vie, k := range o.suivis {
		t := &o.pub.Tracked[k]
		r, ok := reposNeAPeuPres(repos[vie], t.T0)
		if !ok {
			continue
		}
		land, x, y, z := r.land, r.x, r.y, r.z
		t.Land, t.X, t.Y, t.Z = &land, &x, &y, &z
		situes++
	}
	cov.TrackedPositioned += situes
	cov.TrackedUnpositioned += len(o.pub.Tracked) - situes
}
