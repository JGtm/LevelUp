//go:build research

package grammar

// mouvement_5_16_gate_research_test.go — LE GATE DU LOT 5.16, ET SON TABLEAU AVANT/APRES.
//
// Il reprend le gate du 5.14 (`TestClasses514Bourrage` : reste dans [0 ; 7] ET tous ses bits a
// ZERO, parce que le bourrage d octet est ecrit a zero) et lui ajoute, en UNE passe, tout ce que
// le lot doit ne pas abimer :
//
//	paquets fermes a reste NUL     le gate proprement dit
//	debordements                   curseur AU-DELA du payload — jamais tolere
//	records fantomes               un record dont la trace FINIT au-dela du payload
//	records `ti=35` et desyncs     l oracle de CONTENU (reference 5.14.3 : 129 572 et 4)
//	rejets HORS DATUM / DE VUE     les deux sorties par rejet, ventilees (cf. `rejetDeVue`)
//	liaisons de table de datums    ce que `LierTableDeDatums` ajoute, et ses slots ambigus
//	liaisons par anticipation      le REPLI du lot 5.23, par archetype (A/B `MOUV523_ANTICIPE=0`)
//	records par archetype          l oracle de contenu : ti=40, ti=37, ti=32 avant et apres
//
// Rejouable :
//
//	MOUV511_FILM=<dir> MOUV511_BORNES=<catalogue> MOUV511_CARTE=<carte> \
//	  go test -tags=research -count=1 -v -timeout 60m -run '^TestGate516$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
)

// g516Bilan est le tableau d une passe.
type g516Bilan struct {
	paquets, nonLocalise            int
	fermes, nul, debordements       int
	fantomes, ti35, desync35        int
	datums, ambigus                 int
	blocPosees, blocVivantes        int
	blocInconnus, blocAmbigus       int
	rejetsHorsDatum, rejetsDeVue    int
	recordsTotal, resteHorsBourrage int
	// anticipees : les liaisons du REPLI du lot 5.23, par archetype. L A/B est
	// `MOUV523_ANTICIPE=0`, qui rejoue le gate SANS la table anticipee.
	anticipees map[uint32]int
	// parTI : les records rendus par archetype — l oracle de contenu du lot 5.23, qui doit
	// dire ce que ti=40 (vehicules), ti=37 (equipements) et ti=32 gagnent ou perdent.
	parTI map[uint32]int
}

// TestGate516 joue le gate du lot sur le film courant et publie son tableau.
func TestGate516(t *testing.T) {
	tc := t516Cadre(t)
	obs := NouvelleObservation()
	cfg := tc.cfg
	cfg.Obs = obs
	w := NewWorld(tc.reg)
	// L A/B DU LOT 5.23 : `MOUV523_ANTICIPE=0` rejoue le gate SANS la table anticipee, donc
	// sous le modele du 5.21 — c est la colonne « avant » du tableau, et elle doit rester
	// rejouable pour que le gain soit un ECART mesure et non un chiffre isole.
	if os.Getenv("MOUV523_ANTICIPE") != "0" {
		tab := ConstruireTableAnticipee(tc.fc)
		if tis := a523TIDemandes(); tis != nil {
			tab = a523Filtrer(tab, tis) // recherche de cause : un archetype a la fois
		}
		w.PoserTableAnticipee(tab)
	}
	b := g516Bilan{anticipees: map[uint32]int{}, parTI: map[uint32]int{}}
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		w.PoserChunkCourant(c)
		kf := map[uint32]uint32{}
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				//nolint:gosec // slot, TI et Gen viennent du walker, bornes par construction
				slot, ti := uint32(r.Slot), uint32(r.TI)
				w.BindImageCle(uint32(r.Gen), slot, ti) //nolint:gosec // idem
				kf[slot] = ti
			}
		}
		if bloc, present := b521Bloc(t, c, data, pks); present {
			// L A/B DU LOT 5.21 : le bloc de type 1 LIE avant la marche. Mesure : 0 liaison
			// posee sur `bfecd02b`, 1 sur `dad793c7`, et le gate ne bouge d aucun paquet.
			l := b521Lier(w, bloc, kf)
			b.blocVivantes += l.vivantes
			b.blocPosees += l.posees
			b.blocInconnus += l.masqueInconnu
			b.blocAmbigus += l.ambigus
		}
		if os.Getenv("MOUV516_DATUMS") != "0" {
			// L A/B DU LOT : `MOUV516_DATUMS=0` rejoue la marche SANS la table de datums, donc
			// sous le modele du 5.15 (la garde de la branche 0 transcrite). C est la colonne
			// « avant » du tableau, et elle doit rester rejouable pour que « +13 paquets » soit
			// un ECART mesure et non un chiffre isole.
			posees, ambigus := LierTableDeDatums(w, data, pks)
			b.datums += posees
			b.ambigus += ambigus
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			g516Paquet(pk.Payload(data), w, cfg, &b)
		}
	}
	b.rejetsHorsDatum, b.rejetsDeVue = obs.RejetsHorsDatum, obs.RejetsDeVue
	b.anticipees = obs.LiaisonsParAnticipation
	t.Logf("PAQUETS %d (%d non localises) · FERMES %d dont a reste NUL %d · debordements %d",
		b.paquets, b.nonLocalise, b.fermes, b.nul, b.debordements)
	t.Logf("  reste hors bourrage (paquets) : %d", b.resteHorsBourrage)
	t.Logf("RECORDS %d · ti=35 %d (dont %d desynchronises) · FANTOMES %d",
		b.recordsTotal, b.ti35, b.desync35, b.fantomes)
	t.Logf("REJETS DE LA VUE B : hors datum %d · de vue (repli) %d",
		b.rejetsHorsDatum, b.rejetsDeVue)
	t.Logf("TABLE DE DATUMS : %d liaisons posees · %d slots ambigus ecartes",
		b.datums, b.ambigus)
	t.Logf("BLOC DE TYPE 1 : %d entrees vivantes · %d liaisons posees · %d masques inconnus · "+
		"%d masques ambigus", b.blocVivantes, b.blocPosees, b.blocInconnus, b.blocAmbigus)
	g516Anticipees(t, &b)
}

// g516Anticipees publie le REPLI du lot 5.23 : combien de liaisons il a posees, par archetype,
// et ce que la marche rend par archetype (l oracle de contenu).
func g516Anticipees(t *testing.T, b *g516Bilan) {
	t.Helper()
	total := 0
	for _, n := range b.anticipees {
		total += n
	}
	t.Logf("LIAISONS PAR ANTICIPATION (repli 5.23) : %d au total · %s", total,
		g516HistTI(b.anticipees))
	t.Logf("RECORDS PAR ARCHETYPE : %s", g516HistTI(b.parTI))
}

// g516HistTI rend un histogramme `ti=N:compte` par compte decroissant.
func g516HistTI(m map[uint32]int) string {
	cles := make([]int, 0, len(m))
	for k := range m {
		cles = append(cles, int(k))
	}
	sort.Slice(cles, func(i, j int) bool {
		//nolint:gosec // les cles viennent de uint32 bornes par le registre
		if m[uint32(cles[i])] != m[uint32(cles[j])] {
			return m[uint32(cles[i])] > m[uint32(cles[j])]
		}
		return cles[i] < cles[j]
	})
	var sb strings.Builder
	for i, k := range cles {
		if i > 0 {
			sb.WriteString(" · ")
		}
		//nolint:gosec // idem
		fmt.Fprintf(&sb, "ti=%d:%d", k, m[uint32(k)])
	}
	return sb.String()
}

// g516Paquet mesure UN paquet delta et cumule dans le bilan.
func g516Paquet(pay []byte, w *World, cfg FrameConfig, b *g516Bilan) {
	debut := DefaultPacketPreambleBits
	if _, present := PacketHeadEventType(pay); present {
		if debut = marchLocateStrict(pay, w, cfg); debut < 0 {
			b.nonLocalise++
			return
		}
	}
	b.paquets++
	frameLen := len(pay) * 8
	recs, _, curseur := DecodeFrameViewsCurseur(pay, w, cfg, 3, debut)
	switch reste := frameLen - curseur; {
	case reste < 0:
		b.debordements++
	case reste > m5116GateOctet:
		b.resteHorsBourrage++
	default:
		b.fermes++
		if c514ResteNul(pay, curseur) {
			b.nul++
		} else {
			b.resteHorsBourrage++
		}
	}
	for _, r := range recs {
		b.recordsTotal++
		b.parTI[r.TypeIndex]++
		if r.Trace.EndBit > frameLen {
			b.fantomes++
		}
		if r.TypeIndex != BipedTypeIndex {
			continue
		}
		b.ti35++
		if r.DesyncAt >= 0 {
			b.desync35++
		}
	}
}

// TestGate516Contenu dit CE QUE LE TROU PORTAIT, en clair : les records et les COMPOSANTS que la
// marche lit, ventiles par archetype — a comparer avec `MOUV516_DATUMS=0`, qui rejoue la meme
// passe sans la table de datums.
//
// C est l item 5.16.5 : un tableau « par composant, avant/apres ». Sans l A/B il n aurait qu une
// colonne, et le lot 5.15 avait refuse de le publier pour cette raison.
func TestGate516Contenu(t *testing.T) {
	tc := t516Cadre(t)
	w := NewWorld(tc.reg)
	if os.Getenv("MOUV523_ANTICIPE") != "0" {
		// MEME A/B QUE `TestGate516` : sans la table, ce tableau est la colonne « avant ».
		w.PoserTableAnticipee(ConstruireTableAnticipee(tc.fc))
	}
	parTI := map[int]int{}
	parComposant := map[string]int{}
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		w.PoserChunkCourant(c)
		kf := map[uint32]uint32{}
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				//nolint:gosec // slot, TI et Gen viennent du walker, bornes par construction
				slot, ti := uint32(r.Slot), uint32(r.TI)
				w.BindImageCle(uint32(r.Gen), slot, ti) //nolint:gosec // idem
				kf[slot] = ti
			}
		}
		if bloc, present := b521Bloc(t, c, data, pks); present {
			b521Lier(w, bloc, kf)
		}
		if os.Getenv("MOUV516_DATUMS") != "0" {
			LierTableDeDatums(w, data, pks)
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			debut := DefaultPacketPreambleBits
			if _, present := PacketHeadEventType(pay); present {
				if debut = marchLocateStrict(pay, w, tc.cfg); debut < 0 {
					continue
				}
			}
			recs, _, _ := DecodeFrameViewsCurseur(pay, w, tc.cfg, 3, debut)
			for _, r := range recs {
				parTI[int(r.TypeIndex)]++
				for _, cp := range r.Trace.Comps {
					parComposant[cp.Name]++
				}
			}
		}
	}
	t.Logf("RECORDS PAR ARCHETYPE : %s", t516HistTI(parTI))
	t.Logf("COMPOSANTS LUS : %d etiquettes", len(parComposant))
	for _, n := range g516Trier(parComposant) {
		t.Logf("  %-60s %7d", n, parComposant[n])
	}
}

// g516Trier rend les etiquettes de composant par compte DECROISSANT.
func g516Trier(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool {
		if m[out[i]] != m[out[j]] {
			return m[out[i]] > m[out[j]]
		}
		return out[i] < out[j]
	})
	return out
}
