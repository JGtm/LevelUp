package grammar

// frame_closure_detail_records.go — CE QUE LA CARTE V2 DIT DES RECORDS D UN PAQUET
// (`frame_closure_detail.go`) : le dernier composant lu avant la fin de la vue B, les records NEW
// lus, et les lectures qui depassent la fin du payload (le MODE BORNE). Fonctions de ce que la
// marche a DEJA rendu : aucune lecture de bits.

import "fmt"

// decrireLesRecords remplit, dans `d`, ce que les records de la vue B disent du paquet.
func decrireLesRecords(reg *Registry, recs []FrameRecord, frameLen int, d *PaquetDeCarte) {
	d.RecordsLus = len(recs)
	d.DernierLu = dernierLu(reg, recs)
	for _, r := range recs {
		propre := r.DesyncAt < 0
		if r.Type == recNew && propre {
			d.NeufsLus = append(d.NeufsLus, r.Slot)
		}
		if r.Trace.EndBit > frameLen {
			d.RecordsDebordants++
			if r.Type == recNew && propre {
				d.NeufsPropresDebordants++
			}
		}
		d.ComposantsDebordants += composantsDebordants(r.Trace, frameLen)
	}
}

// composantsDebordants compte les composants d une trace dont la lecture finit apres le dernier
// bit du payload. La fin d un composant est le debut du suivant ; celle du dernier, la fin du
// corps.
func composantsDebordants(t EntityTrace, frameLen int) int {
	n := 0
	for k := range t.Comps {
		fin := t.EndBit
		if k+1 < len(t.Comps) {
			fin = t.Comps[k+1].StartBit
		}
		if fin > frameLen {
			n++
		}
	}
	return n
}

// nomDeTypeDeRecord rend le nom d un type de record de la vue B.
func nomDeTypeDeRecord(typ int) string {
	switch typ {
	case recNew:
		return "NEW"
	case recDelta:
		return "DELTA"
	case recDel:
		return "DEL"
	}
	return fmt.Sprintf("type %d", typ)
}

// dernierLu decrit le DERNIER record rendu par la vue B : son type, son archetype et son dernier
// composant lu (`ti=<a> i<idx> <nom>`). Un DELTA dont l archetype a ete INFERE n a pas de trace
// (son corps est saute) ; un DEL n a pas de corps.
func dernierLu(reg *Registry, recs []FrameRecord) string {
	if len(recs) == 0 {
		return "aucun record"
	}
	r := recs[len(recs)-1]
	typ := nomDeTypeDeRecord(r.Type)
	ti := cleArchetype(r)
	switch {
	case r.Type == recDel:
		return typ
	case ti == ArchetypeNonResolu:
		return typ + " slot non lie"
	case len(r.Trace.Comps) > 0:
		c := r.Trace.Comps[len(r.Trace.Comps)-1]
		return fmt.Sprintf("%s ti=%d %s", typ, ti, nomComposantBloquant(reg, ti, c.Index))
	case r.Type == recDelta && r.Trace.EndBit == 0:
		return fmt.Sprintf("%s infere ti=%d", typ, ti)
	}
	return fmt.Sprintf("%s ti=%d sans composant", typ, ti)
}
