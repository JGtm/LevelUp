//go:build research

package grammar

// mouvement_5_15_minimal_research_test.go — LE TEMOIN LE PLUS COURT DU TROU (lot 5.15.1).
//
// Sur `dad793c7`, douze paquets fautifs font EXACTEMENT 96 bits, portent UN seul record
// (`ti=4 i0 high-frequency`, slot 123) et laissent EXACTEMENT 42 bits. Ils sont identiques bit
// pour bit sauf un compteur de 16 bits qui croit d un pas constant. C est la population la plus
// propre du trou du rang 1 : un paquet de douze octets entierement lisible a l oeil, la ou le
// film dense en offre 23 852 de 2 226 largeurs differentes.
//
// Cet instrument les dump INTEGRALEMENT — amorce, record, en-tete rejete, reste — pour que la
// lecture de l ecrivain se confronte a des bits nommes et non a une moyenne.
//
// Rejouable :
//
//	MOUV511_FILM=<dir> MOUV511_BORNES=<catalogue> MOUV511_CARTE=<carte> \n//	  go test -tags=research -count=1 -v -timeout 20m \n//	  -run '^TestTrou515Minimalx27 \n//	  ./internal/games/halo_infinite/film/internal/grammar/

import "testing"

// TestTrou515Minimal dump INTEGRALEMENT les paquets fautifs les plus courts : c est la
// population la plus propre du trou (un seul record, un reste de largeur CONSTANTE).
func TestTrou515Minimal(t *testing.T) {
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
	vus := 0
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
			if _, present := PacketHeadEventType(pay); present {
				continue
			}
			m := t515Marcher(pay, w, cfg, DefaultPacketPreambleBits)
			reste := len(pay)*8 - m.FinVueC
			if reste >= 0 && reste <= m5116GateOctet && c514ResteNul(pay, m.FinVueC) {
				continue
			}
			if !m.PorteA || !m.HitEndB || !m.PorteC || len(pay)*8 > 128 {
				continue
			}
			recs, _, _ := t515Records(pay, w, cfg, DefaultPacketPreambleBits)
			vus++
			if vus > 3 {
				return
			}
			t.Logf("PAQUET de %d bits · A %d · B %d · C %d · reste %d",
				len(pay)*8, m.FinVueA, m.FinVueB, m.FinVueC, reste)
			t.Logf("  TOTALITE      : %s", m511Bits(pay, 0, len(pay)*8))
			t.Logf("  amorce  [0,2) : %s", m511Bits(pay, 0, 2))
			for _, r := range recs {
				t.Logf("  record type %d slot %d ti %d masque %#x comps %d fin %d",
					r.Type, r.Slot, r.TypeIndex, r.Trace.Mask, len(r.Trace.Comps), r.Trace.EndBit)
				for _, cp := range r.Trace.Comps {
					t.Logf("      i%-2d %-34s depuis le bit %d", cp.Index, cp.Name,
						cp.StartBit)
				}
			}
			t.Logf("  en-tete REJETE : %s", m511Bits(pay, m.FinVueB-16, 16))
			t.Logf("  RESTE          : %s", m511Bits(pay, m.FinVueC, reste))
		}
	}
}
