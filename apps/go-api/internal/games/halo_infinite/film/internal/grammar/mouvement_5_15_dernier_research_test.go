//go:build research

package grammar

// mouvement_5_15_dernier_research_test.go — LE DERNIER RECORD AVANT LE REJET (lot 5.15.1).
//
// Les deux instruments precedents ont montre que la vue B clot sa liste sur un REJET de table de
// vue (23 452 paquets sur 23 852) et que le slot rejete n est lie par rien (21 988 sur 22 112),
// avec une etendue de slots quasi uniforme sur les treize bits : un en-tete lu a une position
// FAUSSE ressemble a cela. Cet instrument nomme le RECORD PRECEDENT — son archetype et son
// DERNIER composant lu — pour dire OU la marche a quitte la piste. Il ne conclut rien.
//
// Rejouable :
//
//	MOUV511_FILM=<dir> MOUV511_BORNES=<catalogue> MOUV511_CARTE=<carte> \
//	  go test -tags=research -count=1 -v -timeout 60m \
//	  -run '^TestTrou515Dernier$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"sort"
	"testing"
)

// TestTrou515Dernier ventile, pour les paquets fautifs comme pour les paquets FERMES (temoin), le
// dernier record de la vue B : archetype, dernier composant lu, nombre de composants.
func TestTrou515Dernier(t *testing.T) {
	film := m511Film(t)
	entry := m511Entree(t)
	fc := m511Contexte(film, entry)
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	if restore, errMPP := InstallFilmFormatMPP(fc); errMPP == nil {
		defer restore()
	}
	bal := fc.ProfilDeBalayage()
	bal.Grammaire.ClassesDeVue = true
	fc.PoserProfilDeBalayage(bal)
	cfg := fc.CadreDeBalayage()
	w := NewWorld(reg)

	fautif := map[string]int{}
	temoin := map[string]int{}
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		m533bLierMonde(w, data, pks)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			debut := DefaultPacketPreambleBits
			if _, present := PacketHeadEventType(pay); present {
				if debut = marchLocateStrict(pay, w, cfg); debut < 0 {
					continue
				}
			}
			m := t515Marcher(pay, w, cfg, debut)
			recs, _, _ := t515Records(pay, w, cfg, debut)
			reste := len(pay)*8 - m.FinVueC
			ferme := reste >= 0 && reste <= m5116GateOctet && c514ResteNul(pay, m.FinVueC)
			cle := t515CleDernier(recs)
			if ferme {
				temoin[cle]++
				continue
			}
			if m.PorteA && m.HitEndB && m.PorteC {
				fautif[cle]++
			}
		}
	}
	t.Logf("DERNIER RECORD DE LA VUE B — paquets FAUTIFS (`ti` / dernier composant lu) :")
	t515Top20(t, fautif)
	t.Logf("DERNIER RECORD DE LA VUE B — paquets FERMES (temoin) :")
	t515Top20(t, temoin)
}

// t515CleDernier nomme le dernier record d une liste : `ti=<n> i<dernier composant> (<n> comps)`.
func t515CleDernier(recs []FrameRecord) string {
	if len(recs) == 0 {
		return "aucun record"
	}
	r := recs[len(recs)-1]
	if len(r.Trace.Comps) == 0 {
		return fmt.Sprintf("type %d ti=%d masque vide", r.Type, r.TypeIndex)
	}
	d := r.Trace.Comps[len(r.Trace.Comps)-1]
	return fmt.Sprintf("type %d ti=%2d i%-2d %-34s (%d comps)",
		r.Type, r.TypeIndex, d.Index, d.Name, len(r.Trace.Comps))
}

// t515Top20 publie les vingt classes les plus peuplees d une ventilation.
func t515Top20(t *testing.T, m map[string]int) {
	t.Helper()
	type kv struct {
		k string
		v int
	}
	l := make([]kv, 0, len(m))
	tot := 0
	for k, v := range m {
		l = append(l, kv{k, v})
		tot += v
	}
	sort.Slice(l, func(i, j int) bool {
		if l[i].v != l[j].v {
			return l[i].v > l[j].v
		}
		return l[i].k < l[j].k
	})
	for i := 0; i < 20 && i < len(l); i++ {
		t.Logf("  %6d (%5.1f %%)  %s", l[i].v, m533bPart(l[i].v, tot), l[i].k)
	}
	t.Logf("  TOTAL %d paquets · %d classes", tot, len(l))
}
