package replay

// zone_states_hill_activation_test.go — LA 1RE COLLINE APPARAIT AU COUP D ENVOI, ramene dans la
// fenetre des images-cles (zone_states_hill_activation.go), sur des enregistrements construits.

import (
	"context"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// keyReadAt fabrique une lecture d image-cle du slot a la frame donnee.
func keyReadAt(slot uint32, frame int) grammar.ManagedPropertyRead {
	return zoneChainedReadAt(slot, frame, grammar.ManagedPropertyTagU32, 0)
}

// hillActivationCase : designateur 40 (bascule a 400), proprietaire 41 (camp 0 a 150), le bloc
// voisin 41-43. Images-cles ti=13 aux frames 20 (un autre objet, slot 7) et 220 (le bloc) ; le
// premier contact est a 150. Le gardien se tient dans la zone 1 de 150 a 399.
func hillActivationCase(kickoff int, withKickoff bool) (ZoneInput, zoneCtx) {
	reads := []grammar.ManagedPropertyRead{
		zoneChainedReadAt(40, 400, grammar.ManagedPropertyTagStringID, 0x78F81557),
		zoneReadAt(41, 150, grammar.ManagedPropertyTagU32, 0),
		zoneReadAt(41, 160, grammar.ManagedPropertyTagU32, 0),
	}
	in := zoneTestInput(reads)
	in.Hill = true
	in.KeyReads = []grammar.ManagedPropertyRead{keyReadAt(7, 20), keyReadAt(7, 220), keyReadAt(40, 220), keyReadAt(41, 220)}
	gardien := Track{XUID: "2533", Team: 0, Points: pointsIn(150, 399, 20.5)}
	autre := Track{XUID: "2535", Team: 1, Points: pointsIn(400, 599, -19.5)}
	c := zoneTestCtx(nil, []Track{gardien, autre})
	c.fb = fallback.NouveauCompteur()
	c.kickoff, c.hasKickoff = kickoff, withKickoff
	return in, c
}

// firstSpanT0 rend le debut du premier intervalle publie, toutes zones confondues.
func firstSpanT0(states []ZoneState) int {
	first := -1
	for _, s := range states {
		for _, sp := range s.Spans {
			if first < 0 || sp.T0 < first {
				first = sp.T0
			}
		}
	}
	return first
}

func TestCollinePremiereAuCoupDEnvoi(t *testing.T) {
	in, c := hillActivationCase(60, true)
	states, _ := buildZoneStates(context.Background(), in, c)
	if got := firstSpanT0(states); got != 60 {
		t.Errorf("1re periode a %d, attendu 60 (le coup d envoi)", got)
	}
	if n := c.fb.Compte(fallback.NomCollinePremiereAuCoupDEnvoi); n != 1 {
		t.Errorf("repli du coup d envoi compte %d fois, attendu 1", n)
	}
	if refOfPeriod(states, 100) != 1 {
		t.Errorf("la 1re colline doit etre la zone 1 des le coup d envoi : %+v", states)
	}
}

func TestCollinePremiereRameneeDansLaFenetreDesImagesCles(t *testing.T) {
	// Coup d envoi AVANT l image-cle sans bloc (frame 20) : la colline n existait pas encore.
	in, c := hillActivationCase(10, true)
	states, _ := buildZoneStates(context.Background(), in, c)
	if got := firstSpanT0(states); got != 21 {
		t.Errorf("1re periode a %d, attendu 21 (apres l image-cle qui ne porte pas le bloc)", got)
	}
	// Coup d envoi APRES l image-cle qui porte le bloc (220) et le premier contact (150) : jamais
	// au-dela du premier contact.
	in, c = hillActivationCase(300, true)
	states, _ = buildZoneStates(context.Background(), in, c)
	if got := firstSpanT0(states); got != 150 {
		t.Errorf("1re periode a %d, attendu 150 (jamais apres le premier contact)", got)
	}
}

func TestCollinePremiereSansCoupDEnvoiALImageCle(t *testing.T) {
	in, c := hillActivationCase(0, false)
	in.KeyReads = []grammar.ManagedPropertyRead{keyReadAt(7, 20), keyReadAt(40, 120)}
	states, _ := buildZoneStates(context.Background(), in, c)
	if got := firstSpanT0(states); got != 120 {
		t.Errorf("1re periode a %d, attendu 120 (premiere image-cle qui porte le bloc)", got)
	}
	if n := c.fb.Compte(fallback.NomCollinePremiereAuCoupDEnvoi); n != 0 {
		t.Errorf("repli du coup d envoi compte %d fois sans coup d envoi", n)
	}
}
