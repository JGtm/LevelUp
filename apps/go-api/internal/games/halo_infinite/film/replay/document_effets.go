package replay

// document_effets.go — L OBJET D EFFET d un objet engendre : la bulle de l ecran occultant, avec sa
// naissance, sa suppression et sa position (plan des durees d effets, etape B.2).
//
// POURQUOI. La grenade de l ecran occultant est detruite a l impact (son `t1` lu date la fin du PROJECTILE,
// pas celle de l effet). L effet est une autre vie, ti=38, que la pile i28 ne nomme pas : le film la cree a
// l impact et la supprime 7,507 s plus tard (38 ecrans sur 39, cinq films de 2025-11 a 2026-07, duree a
// +-10 ms). Elle porte une position a la creation, celle du repos de la grenade (a 0,03 m de mediane).
//
// LE LIEN N EST PAS ECRIT. Le record de creation de la ti=38 (i0, i2, i14, i15) ne designe ni la grenade, ni
// le porteur, ni l objet porte. La grammaire rend donc, a la suppression de chaque objet suivi, les vies ti=38
// non nommees nees 100 a 700 ms avant elle (`effect-birth`), puis leur suppression (`effect-end`) ; CE FICHIER
// choisit. Le choix est une ASSOCIATION, nommee et comptee (repli `repli_effet_d_ecran_par_la_fin_de_la_grenade`) :
//
//   - seule la famille de l ecran occultant est lue (la palette est connue ici, pas dans la grammaire) ;
//   - quand le repos de la grenade est publie, ne restent que les candidates a [effetRayonDeReposM] de lui
//     (mesure : 1,96 m au plus sur 34 ecrans ; les rivales mesurees sont a 9,5 et 30 m) ;
//   - sans repos publie, la candidate doit etre la SEULE de la fenetre ;
//   - une seule candidate retenue : l effet est publie ; plusieurs : ambigu, rien n est publie ; aucune : rien.

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// familleDeLEcranOccultant est la famille de la palette dont l effet se lit.
const familleDeLEcranOccultant = "shroud_screen"

// effetRayonDeReposM est la distance maximale, en metres monde, entre la position a la creation de la
// candidate et le repos PUBLIE de la grenade (le dernier point de sa trajectoire a la grille de 100 ms, pas son
// dernier echantillon). Mesure sur 34 ecrans de 4 films : 0 a 1,96 m (mediane 0,6 m) ; la rivale la plus proche
// mesuree est a 9,5 m. 3 m laisse une marge sur la mesure sans approcher la rivale.
const effetRayonDeReposM = 3.0

// CarriedEffect est l objet d effet d un objet engendre : sa naissance, sa suppression quand elle est lue, et sa
// position a la creation quand elle est lue (X, Y, Z absents ensemble).
type CarriedEffect struct {
	T0 int      `json:"t0"`
	T1 *int     `json:"t1,omitempty"`
	X  *float32 `json:"x,omitempty"`
	Y  *float32 `json:"y,omitempty"`
	Z  *float32 `json:"z,omitempty"`
}

// effetCandidat est une vie ti=38 non nommee, nee peu avant la suppression d un objet suivi. `fin` est la frame de
// cette suppression : elle rattache la candidate a CETTE vie de l objet suivi, la cle (slot, generation) revenant
// dans le match (et dans le meme objet porte, pour un second lancer).
type effetCandidat struct {
	vie  types.LifeKey
	ts   uint64
	t    int
	fin  int
	pos  [3]float32
	aPos bool
}

// noterLEffet retient une lecture `effect-birth` : la candidate de l objet suivi `Tracked[0]`, avec la frame de la
// suppression de cet objet (l instant de la lecture).
func (o *objetLu) noterLEffet(l types.CarriedEquipmentRead, in entreeDEquipementPorte) {
	if l.Kind != grammar.CarriedEffectBirth || len(l.Tracked) == 0 || len(l.Spawned) == 0 ||
		l.Spawned[0].TimestampUS < in.origin || l.TimestampUS < in.origin {
		return
	}
	n := l.Spawned[0]
	if o.effets == nil {
		o.effets = map[types.LifeKey][]effetCandidat{}
	}
	o.effets[l.Tracked[0]] = append(o.effets[l.Tracked[0]], effetCandidat{vie: n.Life, ts: n.TimestampUS,
		t: in.frame(n.TimestampUS), fin: in.frame(l.TimestampUS), pos: [3]float32{n.X, n.Y, n.Z}, aPos: n.HasPosition})
}

// finsDEffetLues indexe les suppressions `effect-end` par la vie de l effet, TOUS objets portes confondus : la
// grammaire rend la fin d une candidate une fois, sous le premier objet qui l a retenue, et un second objet suivi qui
// la retient aussi doit la retrouver (la lecture n est pas publiee sous un objet qui n existe peut-etre pas).
func finsDEffetLues(reads []types.CarriedEquipmentRead, in entreeDEquipementPorte) map[types.LifeKey][]finDEffet {
	out := map[types.LifeKey][]finDEffet{}
	for _, l := range reads {
		if l.Kind != grammar.CarriedEffectEnd || len(l.Tracked) == 0 || l.TimestampUS < in.origin {
			continue
		}
		out[l.Tracked[0]] = append(out[l.Tracked[0]], finDEffet{ts: l.TimestampUS, t: in.frame(l.TimestampUS)})
	}
	return out
}

// finDEffet est la suppression lue d une vie d effet.
type finDEffet struct {
	ts uint64
	t  int
}

// situerLesEffets publie l effet de chaque objet suivi de la famille de l ecran occultant, et compte ce qu il a
// retenu, ce qu il a trouve ambigu et ce qu il n a pas pu lier.
func (o *objetLu) situerLesEffets(cov *CarriedEquipmentCoverage) {
	if o.famille != familleDeLEcranOccultant {
		return
	}
	for vie, k := range o.suivis {
		t := &o.pub.Tracked[k]
		retenues := retenirLesCandidates(deCetObjet(o.effets[vie], t), t)
		switch len(retenues) {
		case 0:
			cov.TrackedEffectsUnlinked++
			continue
		case 1:
		default:
			cov.TrackedEffectsAmbiguous++
			continue
		}
		c := retenues[0]
		e := &CarriedEffect{T0: c.t}
		if c.aPos {
			x, y, z := round2(c.pos[0]), round2(c.pos[1]), round2(c.pos[2])
			e.X, e.Y, e.Z = &x, &y, &z
		}
		for _, f := range o.finsDEffet[c.vie] {
			if f.ts > c.ts {
				fin := f.t
				e.T1 = &fin
				cov.TrackedEffectEnds++
				break
			}
		}
		t.Effect = e
		cov.TrackedEffects++
	}
}

// deCetObjet garde les candidates lues a la suppression de CET objet suivi. Une cle (slot, generation) recyclee dans le
// meme objet porte (un second lancer) rend ses candidates sous la meme cle : sans cette garde la bulle du second
// serait attribuee au premier. Meme esprit que la garde de naissance du repos (`toleranceDeNaissance`). Un objet suivi
// dont la suppression n est pas lue n a aucune candidate : elles sont toutes nees d une suppression.
func deCetObjet(cands []effetCandidat, t *CarriedTracked) []effetCandidat {
	if t.T1 == nil {
		return nil
	}
	var out []effetCandidat
	for _, c := range cands {
		if c.fin == *t.T1 {
			out = append(out, c)
		}
	}
	return out
}

// retenirLesCandidates garde les candidates d un objet suivi : celles a [effetRayonDeReposM] de son repos publie,
// ou toutes quand le repos n est pas publie.
func retenirLesCandidates(cands []effetCandidat, t *CarriedTracked) []effetCandidat {
	if t.X == nil || t.Y == nil || t.Z == nil {
		return cands
	}
	var out []effetCandidat
	for _, c := range cands {
		if !c.aPos {
			continue
		}
		if dist3(c.pos, [3]float32{*t.X, *t.Y, *t.Z}) <= effetRayonDeReposM {
			out = append(out, c)
		}
	}
	return out
}
