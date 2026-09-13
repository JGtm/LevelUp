package filmdec

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
// La production ne ferme AUCUN record aujourd'hui (mesure du 2026-09-12 : 62 686 records, 6 films,
// 3 builds) parce qu'elle lit l'image-cle avec le cadre du delta. Le cadre juste existe depuis
// R7-d — `WalkKeyframeFullState` — et n'est branche nulle part : c'est le lot 1.4. Ce fichier ne
// branche rien non plus ; il MESURE, pour que le port des composants manquants (lot 3.6) sache
// par ou commencer et pour qu'un ratchet interdise a la couverture de redescendre.
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
	"fmt"
	"sort"
)

// KeyframeClosureStat est la fermeture d'UN archetype sur un film.
type KeyframeClosureStat struct {
	// Closed : records dont la marche atterrit exactement sur la frontiere visee.
	Closed int
	// Total : records bornes examines (le dernier record d'un payload n'a pas de suivant, donc
	// pas de frontiere : il n'est pas compte).
	Total int
	// Blocking nomme le composant non porte LE PLUS FREQUENT, sous la forme `i<idx> <nom>`, ou
	// l'index nu si le registre ne porte pas son nom. Vide quand aucun record n'a desynchronise.
	//
	// LE PLUS FREQUENT, PAS LE PREMIER VU : un archetype bute a des endroits differents selon le
	// film, et c'est celui qui arrete le plus de records qu'il faut porter d'abord. A egalite, le
	// nom le plus petit tranche — la sortie nourrit un golden, elle doit etre reproductible.
	Blocking string
}

// keyframeClosureOpt : LA FORME JUSTE, celle de `FUN_142e2bfd0`, et rien d'autre.
//
// En-tete de 108 bits, les deux mots de taille autour de l'etat par defaut, et l'etat par defaut
// joue par le deserialiseur de l'archetype. `LevelShift` reste FAUX : c'est une variable de
// recherche encore ouverte (plan R7-e), et une mesure de reference ne se prend pas sous une
// option non tranchee. Ce sont EXACTEMENT les options de l'instrument de recherche
// (`imagecle_fermeture_research_test.go`, `imcOptEtatComplet`) : la mesure de production et
// l'oracle de recherche doivent dire la meme chose du meme film, sinon aucun des deux ne prouve
// rien.
func keyframeClosureOpt() KeyframeFullStateOpt {
	return KeyframeFullStateOpt{
		HeaderBits:   keyframeFullStateHeaderBits,
		SizeWords:    true,
		DefaultState: true,
	}
}

// KeyframeClosure mesure, archetype par archetype, la fermeture des records d'image-cle d'un
// film, sous le cadre d'etat complet.
//
// Elle ne change RIEN au chemin de production des images-cles (`navpoint_radial_scan.go` et
// `objective_scan.go` restent sur `TraverseEntity`) : c'est une mesure, et son branchement est le
// lot 1.4.
func KeyframeClosure(fc *FilmContext) (map[uint32]KeyframeClosureStat, error) {
	if fc == nil {
		return nil, fmt.Errorf("filmdec: contexte de film nil — aucune fermeture a mesurer")
	}
	reg, err := fc.Registry()
	if err != nil {
		return nil, fmt.Errorf("filmdec: registre illisible, la fermeture n'a pas de grammaire: %w", err)
	}
	opt := keyframeClosureOpt()
	stats := map[uint32]KeyframeClosureStat{}
	// bloquants compte, par archetype, combien de records chaque composant non porte a arretes.
	bloquants := map[uint32]map[string]int{}
	for _, num := range fc.ChunkNumbers() {
		data, packets, ok := fc.ChunkAt(num)
		if !ok {
			continue
		}
		for _, pk := range packets {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			accumulerFermeture(pk.Payload(data), reg, opt, stats, bloquants)
		}
	}
	for ti, parComposant := range bloquants {
		s := stats[ti]
		s.Blocking = composantLePlusBloquant(parComposant)
		stats[ti] = s
	}
	return stats, nil
}

// accumulerFermeture classe les records BORNES d'un payload dans les comptes par archetype.
//
// Le dernier record d'un payload est ecarte : sans record suivant il n'a pas de frontiere visee,
// donc la question « ferme-t-il ? » ne se pose pas. Le compter en echec gonflerait le
// denominateur d'un record par payload sans qu'aucun port ne puisse jamais le fermer.
func accumulerFermeture(pay []byte, reg *Registry, opt KeyframeFullStateOpt,
	stats map[uint32]KeyframeClosureStat, bloquants map[uint32]map[string]int,
) {
	recs := WalkKeyframeWorld(pay)
	sort.Slice(recs, func(i, j int) bool { return recs[i].Bit < recs[j].Bit })
	for i := 0; i+1 < len(recs); i++ {
		ti := uint32(recs[i].TI) //nolint:gosec // TI est un index d'archetype, jamais negatif
		tr := WalkKeyframeFullState(pay, recs[i].Bit, reg, opt)
		s := stats[ti]
		s.Total++
		switch {
		case tr.DesyncAt >= 0:
			nom := nomComposantBloquant(reg, recs[i].TI, tr.DesyncAt)
			if bloquants[ti] == nil {
				bloquants[ti] = map[string]int{}
			}
			bloquants[ti][nom]++
		case tr.EndBit == recs[i+1].Bit:
			s.Closed++
		}
		stats[ti] = s
	}
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
