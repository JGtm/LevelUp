package replay

// document_birth_loadouts.go — LES DOTATIONS DE NAISSANCE, publiées dans `loadouts` (schéma 69,
// lot M3.2 de la campagne « retours rejeu », 2026-09-23).
//
// # CE QU'ELLES CORRIGENT
//
// Les armes d'une fiche ne se lisaient qu'aux images-clés, une toutes les ~20 s : un début de vie
// affichait « armes non lues », ou la lecture À VENIR de la première image-clé — qui peut montrer
// une arme ramassée entre-temps. Le record de création du corps transmet sa dotation À LA
// NAISSANCE (cf. `grammar/birth_loadouts.go`) : elle devient le premier relevé PASSÉ de la vie.
//
// # SUR QUELLE VIE ELLE SE POSE
//
// Un slot porte plusieurs vies, et une piste ne commence pas toujours à la création de son corps :
// elle peut s'ouvrir un peu avant (sonde P3 : 4,6 s sur un témoin) ou après, quand les positions
// arrivent en retard. Chaque vie publiée est donc APPARIÉE à la création du même slot la plus
// proche de son début — parmi celles qui ne lui sont pas postérieures (un corps créé après la fin
// d'une vie ne l'a pas ouverte) —, et une dotation se pose sur la vie appariée à SA création,
// jamais sur celle d'une autre. Le relevé prend l'instant de la création ramené dans la fenêtre
// de la vie ; un relevé ramené est COMPTÉ (`snapped`). Sans vie appariée, rien n'est publié
// (`noLife`).
//
// # CE QUI N'EST PAS UNE ARME
//
// Un emplacement vide n'est pas publié. Une famille hors du catalogue d'armes (`weaponv3`) non
// plus : c'est le cas de l'objet de départ `00007CA9`, troisième emplacement au coup d'envoi, que
// le catalogue ne nomme pas — il n'est PAS affiché comme une arme, et il est COMPTÉ (`nonWeapon`)
// en attendant la décision de l'utilisateur (le masquer ou le nommer).

import (
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/weaponv3"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// LoadoutSrcBirth est la provenance d'un relevé lu dans le record de création du corps.
const LoadoutSrcBirth = "birth"

// BirthLoadoutCoverage dit ce que la lecture des dotations de naissance a vu, refusé et publié.
type BirthLoadoutCoverage struct {
	// Creations est le nombre de créations de bipède soumises à la lecture.
	Creations int `json:"creations"`
	// Closed est le nombre de records de création qui se sont FERMÉS ; Read ceux d'entre eux dont
	// au moins un emplacement porte une arme du catalogue — une DOTATION LUE. Une fermeture sans
	// arme n'en est pas une (`noDisplayable`) : sur les builds antérieurs à HI_1_12_0, les
	// quelques records qui se ferment ne portent aucune arme (revue adverse du lot M3.2,
	// 2026-09-24, constat F3). Partition : `closed` = `read` + `noDisplayable`, et `read` =
	// `published` + `noLife` + `beforeOrigin`.
	Closed int `json:"closed"`
	Read   int `json:"read"`
	// Desync / Overflow / Unconfirmed : les records qui ne se ferment pas, par cause. Aucune
	// lecture de repli ne les remplace.
	Desync      int `json:"desync"`
	Overflow    int `json:"overflow"`
	Unconfirmed int `json:"unconfirmed"`
	// NoWeaponComponent : record fermé dont le masque n'annonce aucun emplacement d'arme.
	NoWeaponComponent int `json:"noWeaponComponent"`
	// Published : entrées `loadouts` de provenance `birth`. Snapped : celles dont l'instant a été
	// ramené dans la fenêtre de leur vie.
	Published int `json:"published"`
	Snapped   int `json:"snapped"`
	// NoLife : dotations qu'aucune vie publiée du slot ne porte ; BeforeOrigin : antérieures à la
	// première frame du rejeu.
	NoLife       int `json:"noLife"`
	BeforeOrigin int `json:"beforeOrigin"`
	// NonWeapon : emplacements d'une famille hors du catalogue d'armes, non publiés comme armes.
	NonWeapon int `json:"nonWeapon"`
	// NoDisplayable : records fermés dont aucun emplacement ne porte d'arme publiable.
	NoDisplayable int `json:"noDisplayable"`
}

// birthInputs porte ce que la publication des dotations lit du balayage : les dotations, leurs
// refus comptés, et TOUTES les créations de bipède — lues ou non, c'est la création voisine qui
// borne la vie qu'une dotation peut ouvrir.
type birthInputs struct {
	births    []types.BirthLoadout
	stats     types.BirthLoadoutStats
	creations []grammar.BipedCreation
}

// buildBirthLoadouts projette les dotations de naissance sur les vies publiées. Nil quand le film
// ne porte aucune création : un film sans corps ne publie pas une ligne de zéros.
func buildBirthLoadouts(in birthInputs, tracks []Track, origin, step uint64) ([]Loadout, *BirthLoadoutCoverage) {
	st := in.stats
	if st.Creations == 0 && len(in.births) == 0 {
		return nil, nil
	}
	cov := &BirthLoadoutCoverage{Creations: st.Creations, Closed: st.Read, Desync: st.Desync,
		Overflow: st.Overflow, Unconfirmed: st.Unconfirmed, NoWeaponComponent: st.NoWeaponComponent}
	if step == 0 {
		return nil, cov
	}
	creees := map[uint32][]int{}
	for _, c := range in.creations {
		if c.TimestampUS >= origin {
			creees[c.Slot] = append(creees[c.Slot], int((c.TimestampUS-origin)/step))
		}
	}
	vies := apparierLesVies(fenetresParSlot(tracks), creees)
	var out []Loadout
	for _, b := range in.births {
		// L'ARME D'ABORD : une fermeture sans arme du catalogue n'est pas une dotation lue, où
		// qu'elle tombe sur l'axe du rejeu.
		w, k, nonArme := armesPubliables(b.Weapons)
		cov.NonWeapon += nonArme
		if len(w) == 0 {
			cov.NoDisplayable++
			continue
		}
		cov.Read++
		if b.TimestampUS < origin {
			cov.BeforeOrigin++
			continue
		}
		fb := int((b.TimestampUS - origin) / step)
		v, ok := vies[vieCreee{b.Slot, fb}]
		t := min(max(fb, v.debut), v.fin)
		if !ok {
			cov.NoLife++
			continue
		}
		if t != fb {
			cov.Snapped++
		}
		out = append(out, Loadout{T: t, Slot: b.Slot, W: w, Src: LoadoutSrcBirth, K: k})
	}
	cov.Published = len(out)
	return out, cov
}

// fenetre est la fenêtre de vie d'une piste, sur l'axe des frames.
type fenetre struct{ debut, fin int }

// fenetresParSlot rend les fenêtres de vie des pistes publiées, par slot, dans l'ordre des débuts.
// La fin d'une piste sans `EndFrame` est son dernier point — la règle du client (`trackWindow`).
func fenetresParSlot(tracks []Track) map[uint32][]fenetre {
	out := map[uint32][]fenetre{}
	for _, tr := range tracks {
		f := fenetre{debut: tr.StartFrame, fin: tr.EndFrame}
		if f.fin == 0 && len(tr.Points) > 0 {
			f.fin = tr.Points[len(tr.Points)-1].T
		}
		out[tr.Slot] = append(out[tr.Slot], f)
	}
	for slot := range out {
		l := out[slot]
		sort.SliceStable(l, func(i, j int) bool { return l[i].debut < l[j].debut })
	}
	return out
}

// vieCreee désigne une création : son slot et sa frame.
type vieCreee struct {
	slot  uint32
	frame int
}

// apparierLesVies apparie chaque vie publiée à la création du même slot la plus proche de son
// début, parmi celles qui ne sont pas postérieures à sa fin ; quand deux vies se disputent une
// même création, la plus proche l'emporte. Rend la vie de chaque création appariée.
func apparierLesVies(vies map[uint32][]fenetre, creees map[uint32][]int) map[vieCreee]fenetre {
	out := map[vieCreee]fenetre{}
	ecarts := map[vieCreee]int{}
	for slot, fs := range vies {
		for _, v := range fs {
			c, ecart, ok := creationLaPlusProche(v, creees[slot])
			if !ok {
				continue
			}
			cle := vieCreee{slot, c}
			if e, deja := ecarts[cle]; !deja || ecart < e {
				out[cle], ecarts[cle] = v, ecart
			}
		}
	}
	return out
}

// creationLaPlusProche rend la création la plus proche du début de la vie `v`, parmi celles qui
// ne sont pas postérieures à sa fin, et son écart en frames.
func creationLaPlusProche(v fenetre, creees []int) (c, ecart int, ok bool) {
	for _, f := range creees {
		if f > v.fin {
			continue
		}
		d := f - v.debut
		if d < 0 {
			d = -d
		}
		if !ok || d < ecart {
			c, ecart, ok = f, d, true
		}
	}
	return c, ecart, ok
}

// armesPubliables rend, dans l'ordre des emplacements, les armes d'une dotation que le document
// publie (familles du catalogue, alias repliés sur leur nom), leur emplacement, et le nombre
// d'emplacements écartés parce que leur famille n'est pas une arme du catalogue.
func armesPubliables(weapons []types.BirthWeapon) (w []string, k []int, nonArme int) {
	vus := map[string]bool{}
	for _, a := range weapons {
		if a.Family == grammar.NoWeaponVariant {
			continue
		}
		nom := weaponv3.WeaponName(a.Family)
		if nom == "" {
			nonArme++
			continue
		}
		if vus[nom] {
			continue
		}
		vus[nom] = true
		w = append(w, formatWeaponFamily(a.Family))
		k = append(k, a.Emplacement)
	}
	return w, k, nonArme
}

// mergeLoadouts range dans une seule liste les relevés d'image-clé et les dotations de naissance,
// par frame puis par slot ; à frame et slot égaux, le relevé d'IMAGE-CLÉ d'abord — c'est lui que
// l'inventaire du même instant suit (`inventory[].am` est « dans l'ordre de `Loadout.W` »), et
// les lecteurs qui cherchent le relevé d'un instant prennent le premier.
func mergeLoadouts(images, naissances []Loadout) []Loadout {
	if len(naissances) == 0 {
		return images
	}
	out := make([]Loadout, 0, len(images)+len(naissances))
	out = append(out, images...)
	out = append(out, naissances...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].T != out[j].T {
			return out[i].T < out[j].T
		}
		return out[i].Slot < out[j].Slot
	})
	return out
}
