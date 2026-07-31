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
// LE LOCALISATEUR. Dans un paquet A EVENTS la boucle de records ne commence pas au bit 2 : il
// faut trouver ou la liste d evenements se termine. Signature structurelle : tout paquet type-0
// commence sa boucle par un delta sur le slot 123 de 35 bits EXACTEMENT — candidat UNIQUE et VRAI
// sur 690/690 paquets confrontes a une verite de position independante. Ce n est pas une
// heuristique.
//
// LE REPLI EXISTE PARCE QUE LA SIGNATURE STRICTE EST SUR-CONTRAINTE : la largeur 35 n est que le
// cas MODAL du premier record. Sur les paquets qu elle rate, un delta du slot 123 decode bien —
// mais en 37, 89/90 ou 28 bits. Le repli accepte donc n importe quelle largeur, et il n est
// essaye QUE lorsque la signature stricte echoue : les 92 % deja localises ne bougent pas d un
// bit. Il est falsifiable comme le reste — s il designait du bruit, le gate (b) baisserait.
//
// LE TEST DE GENERATION EST INDISPENSABLE des deux cotes : la primitive d essai ne l applique
// pas, alors que la marche le fait. Sans lui, le localisateur designe des positions ou le slot
// 123 porte une AUTRE generation, et la marche y meurt aussitot (mesure : 3 morts perdues sur un
// film, dont un double kill).

import (
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

// deadRecord : un dead-state atteint par la marche, avec sa position.
type deadRecord struct {
	ms          int
	chunk, pidx int
	slot        int
	bit         int // position du composant dead-state, -1 si non enregistree
	dead        filmdec.DeadState
}

// walkResult : ce que la passe de marche produit.
type walkResult struct {
	deads    []deadRecord // tous les Mort=1
	credible []deadRecord // plage bipede + indices dans le roster + categorie dans l enum
	bipLo    int
	bipHi    int
	located  int
	withEv   int
}

// walkFrom : marche la boucle de records depuis le bit `start`, jusqu a `views` vues de
// replication. Les records deja lus quand la chaine casse sont RENDUS : c est le filtre de
// credibilite qui trie, pas le lecteur.
func walkFrom(pl []byte, w *filmdec.World, cfg filmdec.FrameConfig,
	start, views int) []filmdec.FrameRecord {
	br := filmdec.NewBitReader(pl)
	br.Skip(start)
	var recs []filmdec.FrameRecord
	for v := 0; v < views && len(pl)*8-br.BitPos() >= 8; v++ {
		r2, err := filmdec.DecodeFrameRecords(br, w, cfg)
		recs = append(recs, r2...)
		if err != nil {
			break
		}
	}
	return recs
}

// signature123 : un delta sur le slot 123 decode-t-il proprement en `s` et finit-il 35 bits plus
// loin, avec un unique composant ?
func signature123(pl []byte, s int, w *filmdec.World, cfg filmdec.FrameConfig) bool {
	rec, end, ok := filmdec.TryDeltaAt(pl, s, w, cfg)
	return ok && rec.Slot == 123 && end == s+35 && len(rec.Trace.Comps) == 1
}

// locateStrict : premiere position S >= 2 telle que le bit S-1 vaille 0 (fin de la liste
// d evenements) et que la signature stricte y decode. -1 si aucune.
func locateStrict(pl []byte, w *filmdec.World, cfg filmdec.FrameConfig) int {
	nb := len(pl) * 8
	for s := 2; s+35 < nb; s++ {
		if bitAt(pl, s-1) != 0 {
			continue
		}
		if signature123(pl, s, w, cfg) {
			return s
		}
	}
	return -1
}

// locateFallback : meme condition, LARGEUR LIBRE. N est essaye qu apres l echec de la signature
// stricte.
func locateFallback(pl []byte, w *filmdec.World, cfg filmdec.FrameConfig) int {
	nb := len(pl) * 8
	for s := 2; s+16 < nb; s++ {
		if bitAt(pl, s-1) != 0 {
			continue
		}
		rec, _, ok := filmdec.TryDeltaAt(pl, s, w, cfg)
		if !ok || rec.Slot != 123 || !w.GenerationMatches(rec.ID) {
			continue
		}
		return s
	}
	return -1
}

// locateRecords : le localisateur complet — signature stricte, puis repli. -1 si aucune position.
func locateRecords(pl []byte, w *filmdec.World, cfg filmdec.FrameConfig) int {
	if s := locateStrict(pl, w, cfg); s >= 0 {
		if rec, _, ok := filmdec.TryDeltaAt(pl, s, w, cfg); ok && w.GenerationMatches(rec.ID) {
			return s
		}
	}
	return locateFallback(pl, w, cfg)
}

// runWalk : la passe de marche complete sur tous les paquets type-0, dans l ordre du temps.
func runWalk(f *film, tl *timeline, r *roster, views int) *walkResult {
	tl.rewind()
	cfg := filmdec.DefaultFrameConfig()
	res := &walkResult{}
	res.bipLo, res.bipHi = tl.bipedRange()
	for i := range f.t0 {
		p := &f.t0[i]
		w := tl.advanceTo(p.ts)
		start := 2
		if hasEvents(p) {
			res.withEv++
			s := locateRecords(p.payload, w, cfg)
			if s < 0 {
				continue
			}
			res.located++
			start = s
		}
		res.deads = append(res.deads, walkPacket(p, w, cfg, start, views, f.ms(p))...)
	}
	sort.Slice(res.deads, func(i, j int) bool { return res.deads[i].ms < res.deads[j].ms })
	res.selectCredible(r)
	return res
}

// walkPacket : les dead-states d un seul paquet. Le monde est restaure : une marche qui a
// desynchronise ne doit pas laisser de liaison derriere elle (les deux politiques qui les
// conservaient ont ete MESUREES COMME PERDANTES, 330 -> 328 puis 315 sur 372).
func walkPacket(p *packet, w *filmdec.World, cfg filmdec.FrameConfig,
	start, views, ms int) []deadRecord {
	snap := w.Snapshot()
	recs := walkFrom(p.payload, w, cfg, start, views)
	w.Restore(snap)
	var out []deadRecord
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
			continue
		}
		out = append(out, deadRecord{ms: ms, chunk: p.chunk, pidx: p.idx, slot: int(r.Slot),
			bit: deadStateBit(r), dead: *r.Trace.Dead})
	}
	return out
}

// deadStateBit : position du composant dead-state dans le record, -1 s il n y figure pas.
func deadStateBit(r *filmdec.FrameRecord) int {
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
			continue
		}
		if d.dead.EnumA < 0 || int(d.dead.EnumA) >= r.nPlay {
			continue
		}
		if d.dead.EnumB < 0 || int(d.dead.EnumB) >= r.nPlay {
			continue
		}
		if d.dead.Val0c > 9 {
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
