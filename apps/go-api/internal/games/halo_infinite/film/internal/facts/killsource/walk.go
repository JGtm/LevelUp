package killsource

// walk.go — LA MARCHE : deroulement sequentiel des records ECS d un paquet.
//
// C est la voie la PLUS CONTRAINTE, et c est pour cela qu elle decide en premier dans l hybride :
// un record qu elle rend a ete atteint par une chaine complete de largeurs connues. Son gate (b)
// vaut 98.2 % contre 78.4 % pour le rattrapage du scan.
//
// ELLE A UNE PROPRIETE QUE LE SCAN N A PAS, ET C EST LE MEILLEUR ARGUMENT POUR LA GARDER : elle
// n a AUCUNE porte de catalogue. Elle lit le tag quel qu il soit. Donc elle voit ce que le test
// T4 du scan cache — un catalogue perime — et l ablation d un tag reel le mesure : une
// architecture scan-d abord perd 224 lignes sur 20 essais, l hybride en perd 20. Facteur 11.2.
//
// LE DEBUT DE LA VUE B. Dans un paquet A EVENTS la boucle de records ne commence pas au bit 2 : son
// debut est la fin de la vue A quand la lecture de la vue A en decide, sinon celui du localisateur
// unique de `grammar` ([grammar.DebutDeLaVueB], ordre [grammar.SignaturePuisLargeurLibre] :
// signature stricte du slot 123 a la generation du monde, sinon repli a largeur libre), le meme
// que celui de la marche des morts d objet. Le paquet qu il ne localise pas se saute.

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
	deads    []deadRecord // tous les Mort=1
	credible []deadRecord // plage bipede + indices dans le roster + categorie dans l enum
	bipLo    int
	bipHi    int
	located  int
	withEv   int
	// LES COMPTES DES REPLIS DE LA MARCHE (lot J8.7), verses a [ReplisDuDecodage] : dead-states jetes par
	// desynchronisation, par le filtre de credibilite (bande, roster, enumeration), et paquets
	// localises a largeur libre.
	desync, horsBande, horsRoster, horsEnum, largeurLibre int
}

// walkFrom : marche la boucle de records depuis le bit `start`, jusqu a `views` vues de
// replication. Les records deja lus quand la chaine casse sont RENDUS : c est le filtre de
// credibilite qui trie, pas le lecteur.
func walkFrom(pl []byte, w *grammar.World, cfg grammar.FrameConfig,
	start, views int) []grammar.FrameRecord {
	br := grammar.LecteurSur(pl)
	br.Skip(start)
	var recs []grammar.FrameRecord
	for v := 0; v < views && len(pl)*8-br.BitPos() >= 8; v++ {
		r2, err := grammar.DecodeFrameRecords(br, w, cfg)
		recs = append(recs, r2...)
		if err != nil {
			break
		}
	}
	return recs
}

// runWalk : la passe de marche complete sur tous les paquets type-0, dans l ordre du temps.
//
// `cal` porte le PROFIL DE MOUVEMENT que la calibration a retenu (lot 2.2.a), et la grammaire de
// la vue A du film (lot VA). Le profil arrive en PARAMETRE depuis le 2.2.a : avant, la marche
// reconstruisait un cadre par defaut et heritait des largeurs calibrees par effet de bord des
// variables de paquet de `grammar`.
func runWalk(f *film, tl *timeline, r *roster, views int, cal *calibration) *walkResult {
	tl.rewind()
	cfg := grammar.DefaultFrameConfig()
	cfg.Profil = cal.Profil
	res := &walkResult{}
	res.bipLo, res.bipHi = tl.bipedRange()
	for i := range f.t0 {
		p := &f.t0[i]
		w := tl.advanceTo(p.ts)
		start := 2
		if hasEvents(p) {
			res.withEv++
			s, aLargeurLibre := grammar.DebutDeLaVueB(p.payload, w, cfg, cal.VueA)
			if s < 0 {
				continue
			}
			res.located++
			res.largeurLibre += unSi(aLargeurLibre)
			start = s
		}
		deads, desync := walkPacket(p, w, cfg, start, views, f.ms(p))
		res.deads = append(res.deads, deads...)
		res.desync += desync
	}
	trierMortsDeLaMarche(res.deads)
	res.selectCredible(r)
	return res
}

// walkPacket : les dead-states d un seul paquet. Le monde est restaure : une marche qui a
// desynchronise ne doit pas laisser de liaison derriere elle (les deux politiques qui les
// conservaient ont ete MESUREES COMME PERDANTES, 330 -> 328 puis 315 sur 372).
//
// Le second rendu compte les dead-states JETES par desynchronisation — `repli_record_desynchronise_jete`
// (lot J8.7).
func walkPacket(p *packet, w *grammar.World, cfg grammar.FrameConfig,
	start, views, ms int) ([]deadRecord, int) {
	snap := w.Snapshot()
	recs := walkFrom(p.payload, w, cfg, start, views)
	w.Restore(snap)
	var out []deadRecord
	jetes := 0
	for i := range recs {
		r := &recs[i]
		if r.Trace.Dead == nil || !r.Trace.Dead.Mort {
			continue
		}
		// SEULS LES RECORDS PROPRES SONT RETENUS. La variante << accepter les
		// desynchronisations TARDIVES (au-dela du composant 11, donc apres la lecture du
		// dead-state) >> existe dans l outil de RE derriere une bascule ; elle n est PAS la
		// configuration mesuree, et c est la configuration mesuree qui est gelee ici.
		if r.DesyncAt != -1 {
			jetes++
			continue
		}
		out = append(out, deadRecord{ms: ms, chunk: p.chunk, pidx: p.idx, slot: int(r.Slot),
			bit: deadStateBit(r), dead: *r.Trace.Dead})
	}
	return out, jetes
}

// deadStateBit : position du composant dead-state dans le record, -1 s il n y figure pas.
func deadStateBit(r *grammar.FrameRecord) int {
	for _, c := range r.Trace.Comps {
		if c.Name == deadStateComponent {
			return c.StartBit
		}
	}
	return -1
}

// deadStateComponent : le nom du composant i11 dans le registre ECS du film. Il est NOMME dans
// le chunk 0 — le schema est dans le film, il n a pas fallu le deviner.
const deadStateComponent = "object-dead-state-component"

// selectCredible : le filtre de credibilite. Trois conditions, toutes structurelles :
// le slot est dans la plage bipede DERIVEE, les deux indices sont dans le roster retenu, et la
// categorie est dans l enum.
func (res *walkResult) selectCredible(r *roster) {
	for _, d := range res.deads {
		if d.slot < res.bipLo || d.slot > res.bipHi {
			res.horsBande++
			continue
		}
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
