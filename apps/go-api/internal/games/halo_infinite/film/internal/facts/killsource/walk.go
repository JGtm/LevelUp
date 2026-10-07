package killsource

// walk.go — LA MARCHE : les dead-states des bipedes que la marche des trames lit.
//
// C est la voie la PLUS CONTRAINTE, et c est pour cela qu elle decide en premier dans l hybride :
// un record qu elle rend a ete atteint par une chaine complete de largeurs connues. Son gate (b)
// valait 98.2 % contre 78.4 % pour le rattrapage du scan.
//
// ELLE A UNE PROPRIETE QUE LE SCAN N A PAS, ET C EST LE MEILLEUR ARGUMENT POUR LA GARDER : elle
// n a AUCUNE porte de catalogue. Elle lit le tag quel qu il soit. Donc elle voit ce que le test
// T4 du scan cache — un catalogue perime — et l ablation d un tag reel le mesure : une
// architecture scan-d abord perd 224 lignes sur 20 essais, l hybride en perd 20. Facteur 11.2.
//
// LA MARCHE EST CELLE DE LA GRAMMAIRE ([grammar.LireLesMortsDeLaMarche]) : la marche des trames du
// contexte du film, son monde, ses debuts de vue B (la fin de la vue A, sinon le localisateur), et
// les listes qu elle ne localise pas recuperees par le canal des morts (signature, puis largeur
// libre). killsource y garde sa regle : les records ENTIEREMENT PORTES d un archetype BIPEDE — celui
// que la marche lie au slot —, puis son filtre de credibilite.

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// deadRecord : un dead-state atteint par la marche, avec sa position.
type deadRecord struct {
	ms          int
	chunk, pidx int
	slot        int
	bit         int // position du composant dead-state, -1 si non enregistree
	dead        types.DeadState
}

// walkResult : ce que la passe de marche produit.
type walkResult struct {
	deads    []deadRecord // tous les Mort=1 de bipede entierement portes
	credible []deadRecord // indices dans le roster + categorie dans l enum
	located  int
	withEv   int
	// LES COMPTES DES REPLIS DE LA MARCHE (lot J8.7), verses a [ReplisDuDecodage] : dead-states de
	// bipede jetes parce que leur record a rompu, rejets du filtre de credibilite (roster,
	// enumeration), et listes localisees a largeur libre.
	desync, horsRoster, horsEnum, largeurLibre int
}

// marcherLesMorts lit les dead-states de la marche des trames du contexte `fc` et les range comme
// la marche de killsource les rangeait : dans l ordre du film, credibles a part.
func marcherLesMorts(fc *grammar.FilmContext, f *film, r *roster) (*walkResult, error) {
	lus, err := grammar.LireLesMortsDeLaMarche(fc)
	if err != nil {
		return nil, err
	}
	res := &walkResult{withEv: lus.Stats.EventPackets, located: lus.Stats.LocatedPackets,
		largeurLibre: lus.LargeurLibre}
	for _, m := range lus.Lus {
		if m.TypeIndex != grammar.BipedTypeIndex {
			continue
		}
		// SEULS LES RECORDS PROPRES SONT RETENUS : une rupture, meme apres le dead-state, n est pas
		// la configuration mesuree de cette voie.
		if !m.Propre {
			res.desync++
			continue
		}
		res.deads = append(res.deads, deadRecord{ms: f.msDe(m.TS), chunk: m.PositionDuChunk, pidx: m.Index,
			slot: int(m.Slot), bit: m.Bit, dead: m.Dead})
	}
	trierMortsDeLaMarche(res.deads)
	res.selectCredible(r)
	return res, nil
}

// selectCredible : le filtre de credibilite. Deux conditions, toutes structurelles : les deux
// indices sont dans le roster retenu, et la categorie est dans l enum.
func (res *walkResult) selectCredible(r *roster) {
	for _, d := range res.deads {
		if d.dead.EnumA < 0 || int(d.dead.EnumA) >= r.nPlay {
			res.horsRoster++
			continue
		}
		if d.dead.EnumB < 0 || int(d.dead.EnumB) >= r.nPlay {
			res.horsRoster++
			continue
		}
		if d.dead.Val0c > 9 {
			res.horsEnum++
			continue
		}
		res.credible = append(res.credible, d)
	}
}

// candidates : les dead-states credibles de la marche, convertis dans la forme du scan pour que
// les deux voies produisent des objets COMPARABLES. Un record sans position enregistree garde
// `bit = -1` : il reste utilisable pour l appariement, il est seulement inapte au test de
// redondance avec le scan — et ce cas est COMPTE, jamais ignore.
func (res *walkResult) candidates() ([]candidate, int) {
	out := make([]candidate, 0, len(res.credible))
	noBit := 0
	for _, d := range res.credible {
		if d.bit < 0 {
			noBit++
		}
		out = append(out, candidate{
			chunk: d.chunk, pidx: d.pidx, ms: d.ms, bit: d.bit,
			tag: d.dead.SrcTag0, victim: int(d.dead.EnumA), killer: int(d.dead.EnumB),
			cat: int(d.dead.Val0c),
		})
	}
	return dedup(out), noBit
}
