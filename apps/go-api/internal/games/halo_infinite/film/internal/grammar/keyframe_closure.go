package grammar

// keyframe_closure.go — LA FERMETURE DES RECORDS D'IMAGE-CLE, PAR ARCHETYPE (lot 0.A.3).
//
// # CE QUE « FERMER » VEUT DIRE
//
// Un record d'image-cle porte l'etat COMPLET d'une entite. La grammaire le dit lisible de bout
// en bout : une fois ses composants deroules, la marche doit atterrir EXACTEMENT sur le premier
// bit du record suivant. Quand elle y atterrit, le record est FERME — et c'est la seule preuve
// qu'on ait que toutes ses largeurs sont justes, sans capture live ni oracle externe. Une marche
// qui s'arrete sur un composant sans lecteur (DESYNC), qui tombe avant (SOUS) ou apres (SUR) la
// frontiere ne ferme pas : elle a lu quelque chose, mais rien ne dit quoi.
//
// # POURQUOI C'EST LA MESURE QUI COMPTE
//
// La production ne fermait AUCUN record jusqu'au 2026-09-14 (mesure du 2026-09-12 : 0 sur
// 62 686 records, 6 films, 3 builds) parce qu'elle lisait l'image-cle avec le cadre du delta.
// LE LOT 1.4 A BRANCHE LE CADRE JUSTE : les deux balayages qui parsent le corps d'un record
// d'image-cle (`navpoint_radial_scan.go`, `objective_scan.go`) appellent desormais
// `WalkKeyframeFullState`, c'est-a-dire EXACTEMENT ce que ce fichier mesure.
//
// CONSEQUENCE POUR LE RATCHET : il ne bouge PAS au lot 1.4, et c'est le resultat attendu. Ce
// golden a toujours mesure le cadre d'etat complet (lot 0.A.3) ; ce qui change au lot 1.4, c'est
// que la PRODUCTION le rejoint. Le plan prevoyait « 0 -> 14 % » ici : c'etait une erreur de
// citation — ce 0 etait celui des compteurs de production, pas celui de ce golden, qui vaut
// 30,8 % depuis le lot 1.3. Le ratchet continue de servir a ce pour quoi il existe : interdire a
// la couverture de redescendre quand le lot 3.6 portera les composants manquants.
//
// # LE BLOQUANT
//
// Dans une image-cle il n'y a pas de masque de presence : TOUS les composants de l'archetype sont
// la, dans l'ordre du registre. Un seul composant sans lecteur bloque donc tout ce qui le suit —
// et c'est pour cela que `Blocking` nomme le composant non porte LE PLUS FREQUENT — celui qui
// arrete le plus de records, donc celui dont le port en debloque le plus. Ce n'est PAS « le
// premier rencontre » : un archetype bute a des endroits differents selon le film, et prendre la
// premiere occurrence vue ferait dependre la reponse de l'ordre de parcours.

import (
	"cmp"
	"fmt"
	"slices"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
)

// KeyframeClosureStat est la fermeture d'UN archetype sur un film.
type KeyframeClosureStat struct {
	// Closed : records dont la marche atterrit exactement sur la frontiere visee.
	Closed int
	// Total : records bornes examines (le dernier record d'un payload n'a pas de suivant, donc
	// pas de frontiere : il n'est pas compte).
	Total int
	// Blocking nomme le composant non porte LE PLUS FREQUENT, sous la forme `i<idx> <nom>`, ou
	// l'index nu si le registre ne porte pas son nom. Vide quand aucun record n'a desynchronise. Un
	// lecteur du jeu en ECHEC ([lecture.EtatArrete]) n'est pas un composant non porte : il est nomme
	// `i<idx> <nom> (arret <cause>)` (plan LK, D-12).
	//
	// LE PLUS FREQUENT, PAS LE PREMIER VU : un archetype bute a des endroits differents selon le
	// film, et c'est celui qui arrete le plus de records qu'il faut porter d'abord. A egalite, le
	// nom le plus petit tranche — la sortie nourrit un golden, elle doit etre reproductible.
	Blocking string
}

// keyframeBorne est un record d'image-cle BORNE : celui dont le balayeur d'ancres rend un
// voisin suivant, donc le seul dont la question « ferme-t-il ? » ait un sens.
type keyframeBorne struct {
	// Bit est le premier bit du record ; Want la frontiere visee (premier bit du suivant).
	Bit, Want int
	// Slot et TI identifient le record.
	Slot, TI int
	// Voisin restreint aux paires de slots CONSECUTIFS (`slotSuivant == slot + 1`), ou le
	// balayeur ne peut avoir saute aucun record entre les deux ancres.
	Voisin bool
}

// keyframeBornesToutes rend TOUS les records d'un payload d'image-cle, tries par bit, chacun
// avec la frontiere visee — SAUF LE DERNIER, qui rend `Want = -1` : sans record suivant il n'a
// pas de frontiere, donc la question « ferme-t-il ? » ne se pose pas pour lui. Il est rendu
// quand meme parce qu'un balayage de production doit le LIRE : il porte des donnees comme les
// autres, et l'ecarter serait perdre une lecture en silence.
//
// UNE SEULE COPIE DE L'APPARIEMENT, ET C'EST LA REGLE 6 DU DEPOT. Le meme « record i,
// frontiere i+1 » etait ecrit dans `accumulerFermeture` (mesure) et dans l'instrument de
// recherche (`imcBornes`), et il en fallait deux de plus dans les balayages de production au
// lot 1.4 : a la troisieme copie on centralise. La mesure, l'oracle et la production comptent
// desormais sur la MEME population — sans quoi aucun des trois ne prouve rien des deux autres.
//
// SANS PREUVE : c'est la forme des instruments. Un balayage de production passe par
// [keyframeBornesDe] sur les records de la marche de son film ([FilmContext.MarcheDImageCle]).
func keyframeBornesToutes(pay []byte) []keyframeBorne {
	return keyframeBornesDe(WalkKeyframeWorld(pay))
}

// keyframeBornesDe rend les bornes de records DEJA marches (cf. [keyframeBornesToutes]).
func keyframeBornesDe(recs []KeyframeRec) []keyframeBorne {
	// Tri total (J12.1, DT-9) : Bit unique, la marche avance strictement (ancre suivante >= Bit+64).
	slices.SortFunc(recs, func(a, b KeyframeRec) int { return cmp.Compare(a.Bit, b.Bit) })
	out := make([]keyframeBorne, 0, len(recs))
	for i := range recs {
		b := keyframeBorne{Bit: recs[i].Bit, Want: -1, Slot: recs[i].Slot, TI: recs[i].TI}
		if i+1 < len(recs) {
			b.Want = recs[i+1].Bit
			b.Voisin = recs[i+1].Slot == recs[i].Slot+1
		}
		out = append(out, b)
	}
	return out
}

// keyframeBornes rend les seuls records BORNES : le denominateur de toute mesure de fermeture.
// Sans preuve (instruments) ; la mesure de production borne les records de la marche de son film.
func keyframeBornes(pay []byte) []keyframeBorne {
	return seulesBornees(keyframeBornesToutes(pay))
}

// seulesBornees garde les records qui ont une frontiere visee.
func seulesBornees(toutes []keyframeBorne) []keyframeBorne {
	out := make([]keyframeBorne, 0, len(toutes))
	for _, b := range toutes {
		if b.Want >= 0 {
			out = append(out, b)
		}
	}
	return out
}

// KeyframeClosure mesure, archetype par archetype, la fermeture des records d'image-cle d'un
// film, sous le cadre d'etat complet — celui que la production lit depuis le lot 1.4.
//
// LA MESURE CONSOMME LA PHASE IMAGES-CLES DE LA STRUCTURE DE LECTURE ([FilmContext.ImagesCles]) :
// les records sont les ancres de la marche du film ([FilmContext.MarcheDImageCle], qui refuse
// l'elu qu'un record prouve contredit), chacun avec sa preuve — la population que les balayages
// lisent. Le decoupage MPP du format du film est pose pour la duree de la marche (lot 1.9.1 bis) :
// sans lui, la fermeture des archetypes qui portent ce bloc (ti=36, 37, 38, 39, 42, 43) serait
// mesuree au decoupage d un AUTRE build.
func KeyframeClosure(fc *FilmContext) (map[uint32]KeyframeClosureStat, error) {
	if fc == nil {
		return nil, fmt.Errorf("filmdec: contexte de film nil — aucune fermeture a mesurer")
	}
	reg, err := fc.Registry()
	if err != nil {
		return nil, fmt.Errorf("filmdec: registre illisible, la fermeture n'a pas de grammaire: %w", err)
	}
	m := mesureDeFermeture{reg: reg, stats: map[uint32]KeyframeClosureStat{}, bloquants: map[uint32]map[string]int{}}
	for p, err := range fc.ImagesCles() {
		if err != nil {
			return nil, err
		}
		m.accumuler(p)
	}
	for ti, parComposant := range m.bloquants {
		s := m.stats[ti]
		s.Blocking = composantLePlusBloquant(parComposant)
		m.stats[ti] = s
	}
	return m.stats, nil
}

// mesureDeFermeture porte ce que la mesure accumule d'un paquet a l'autre.
type mesureDeFermeture struct {
	reg *Registry
	// stats : les comptes par archetype ; bloquants : par archetype, combien de records chaque
	// composant non porte, ou chaque lecteur en echec avec sa cause, a arretes.
	stats     map[uint32]KeyframeClosureStat
	bloquants map[uint32]map[string]int
}

// accumuler classe les records BORNES d'un paquet d'image-cle dans les comptes par archetype.
//
// Le dernier record d'un paquet est ecarte : sans record suivant il n'a pas de frontiere visee,
// donc la question « ferme-t-il ? » ne se pose pas. Le compter en echec gonflerait le
// denominateur d'un record par paquet sans qu'aucun port ne puisse jamais le fermer.
func (m *mesureDeFermeture) accumuler(p *lecture.Paquet) {
	for i := 0; i+1 < len(p.Records); i++ {
		r := &p.Records[i]
		ti := uint32(r.TI) //nolint:gosec // TI est l archetype d une ancre, jamais negatif
		s := m.stats[ti]
		s.Total++
		switch {
		case r.Desync >= 0:
			nom := nomComposantBloquant(m.reg, int(r.TI), int(r.Desync))
			if c, ok := composantDArret(p, r); ok {
				nom += " (arret " + nomDeLArret(c.Arret) + ")" // un lecteur en echec n est pas un composant non porte
			}
			if m.bloquants[ti] == nil {
				m.bloquants[ti] = map[string]int{}
			}
			m.bloquants[ti][nom]++
		case r.Preuve == lecture.PreuveFerme:
			s.Closed++
		}
		m.stats[ti] = s
	}
}

// composantDArret rend la derniere occurrence du record `r` quand le lecteur du jeu y a echoue
// ([lecture.EtatArrete]) : la marche s y est arretee, cause nommee.
func composantDArret(p *lecture.Paquet, r *lecture.Record) (lecture.Composant, bool) {
	if r.Comps[1] == r.Comps[0] {
		return lecture.Composant{}, false
	}
	c := p.Comps[r.Comps[1]-1]
	return c, c.Etat == lecture.EtatArrete
}

// nomComposantBloquant nomme le composant qui a arrete la marche, ou l'index nu si le registre ne
// porte pas son nom — un index sans nom reste exploitable, un silence ne l'est pas.
func nomComposantBloquant(reg *Registry, ti, idx int) string {
	if arch, ok := reg.Archetype(ti); ok && idx >= 0 && idx < len(arch.Components) {
		return fmt.Sprintf("i%d %s", idx, arch.Components[idx])
	}
	return fmt.Sprintf("i%d", idx)
}

// composantLePlusBloquant rend le composant qui a arrete le plus de records. A egalite, le nom le
// plus petit tranche : la sortie doit etre REPRODUCTIBLE (elle nourrit un golden), et l'ordre de
// parcours d'une map ne l'est pas.
func composantLePlusBloquant(parComposant map[string]int) string {
	meilleur, meilleurN := "", -1
	for nom, n := range parComposant {
		if n > meilleurN || (n == meilleurN && nom < meilleur) {
			meilleur, meilleurN = nom, n
		}
	}
	return meilleur
}
