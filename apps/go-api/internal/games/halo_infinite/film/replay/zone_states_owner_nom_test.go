package replay

// zone_states_owner_nom_test.go — LE CANAL DE PROPRIETE DESIGNE PAR LE NOM, sur des enregistrements
// CONSTRUITS (cas `bastionCase` de `zone_states_test.go`, que les lectures d'image-cle nomment).

import (
	"context"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// zoneNomAt rend une lecture d'image-cle NOMMEE a la frame 0 : la valeur est neutre (aucun etat
// initial), seul le nom compte.
func zoneNomAt(slot uint32, tag int, nom uint32) grammar.ManagedPropertyRead {
	r := zoneReadAt(slot, 0, tag, zoneNeutralOwner)
	r.Chained, r.Name, r.Named = true, nom, true
	return r
}

// zoneNomsBastion nomme les deux blocs du cas `bastionCase` : jauges 10 et 20, proprietaires 11
// et 21.
func zoneNomsBastion() []grammar.ManagedPropertyRead {
	return []grammar.ManagedPropertyRead{
		zoneNomAt(10, grammar.ManagedPropertyTagQuant, zoneBlocsNommes[0].jauge),
		zoneNomAt(11, grammar.ManagedPropertyTagU32, zoneBlocsNommes[0].proprietaire),
		zoneNomAt(20, grammar.ManagedPropertyTagQuant, zoneBlocsNommes[1].jauge),
		zoneNomAt(21, grammar.ManagedPropertyTagU32, zoneBlocsNommes[1].proprietaire),
	}
}

// TestZoneProprietaireLuParLeNomSansCaptureConcordante : la zone 1 n'a qu'UNE capture concordante,
// sous le seuil du vote ; son nom designe pourtant son proprietaire, et elle est publiee.
func TestZoneProprietaireLuParLeNomSansCaptureConcordante(t *testing.T) {
	in, c := bastionCase()
	in.Reads = zoneReadsWithout(in.Reads, 21, 401)
	c.actions = c.actions[:3]
	in.KeyReads = zoneNomsBastion()
	fb := fallback.NouveauCompteur()
	c.fb = fb
	states, cov := buildZoneStates(context.Background(), in, c)
	if len(states) != 2 || cov.OwnerUnpaired != 0 || cov.OwnerNamed != 2 {
		t.Fatalf("%d zone(s) publiee(s), %d sans proprietaire, %d nommee(s) : attendu 2, 0, 2",
			len(states), cov.OwnerUnpaired, cov.OwnerNamed)
	}
	spans := states[1].Spans
	if got := ownerOf(spans[len(spans)-1]); states[1].ZoneRef != 1 || got != 1 {
		t.Errorf("zone %d, dernier camp %d : attendu zone 1 prise par le camp 1", states[1].ZoneRef, got)
	}
	if n := fb.Compte(fallback.NomZoneProprietaireParVote); n != 0 {
		t.Errorf("repli par vote compte %d fois alors que le nom designe chaque zone", n)
	}
}

// TestZoneProprietaireLeNomPrimeSurLeVote : le vote elit le slot 11 pour la zone 0, le nom designe
// le slot 31. Le nom est retenu, la discordance se compte.
func TestZoneProprietaireLeNomPrimeSurLeVote(t *testing.T) {
	in, c := bastionCase()
	in.Reads = append(in.Reads,
		zoneReadAt(31, 0, grammar.ManagedPropertyTagU32, 1),
		zoneReadAt(31, 150, grammar.ManagedPropertyTagU32, 0))
	in.KeyReads = zoneNomsBastion()
	in.KeyReads[1] = zoneNomAt(31, grammar.ManagedPropertyTagU32, zoneBlocsNommes[0].proprietaire)
	states, cov := buildZoneStates(context.Background(), in, c)
	if cov.OwnerVoteDisagreed != 1 || cov.OwnerNamed != 2 {
		t.Fatalf("discordances %d, nommees %d : attendu 1 et 2", cov.OwnerVoteDisagreed, cov.OwnerNamed)
	}
	if len(states) == 0 || states[0].ZoneRef != 0 || len(states[0].Spans) != 2 ||
		states[0].Spans[1].T0 != 150 {
		t.Fatalf("zone 0 : %+v — attendu les intervalles du slot 31 (bascule a 150)", states)
	}
}

// TestZoneProprietaireBlocIncompletRetombeSurLeVote : la jauge de la zone 0 est nommee mais le
// proprietaire de son bloc est absent du film ; la zone retombe sur le vote, et le repli se compte.
func TestZoneProprietaireBlocIncompletRetombeSurLeVote(t *testing.T) {
	in, c := bastionCase()
	in.KeyReads = zoneNomsBastion()
	in.KeyReads[1] = zoneNomAt(11, grammar.ManagedPropertyTagU32, 0xDEADBEEF)
	fb := fallback.NouveauCompteur()
	c.fb = fb
	states, cov := buildZoneStates(context.Background(), in, c)
	if len(states) != 2 || cov.OwnerNamed != 1 || cov.OwnerUnpaired != 0 {
		t.Fatalf("%d zone(s), %d nommee(s), %d sans proprietaire : attendu 2, 1, 0",
			len(states), cov.OwnerNamed, cov.OwnerUnpaired)
	}
	if n := fb.Compte(fallback.NomZoneProprietaireParVote); n != 1 {
		t.Errorf("repli par vote compte %d fois, attendu 1", n)
	}
}

// TestZoneNomsAmbigusNeDesignentRien : un nom porte par deux slots, ou un slot qui porte deux noms,
// sort des deux tables.
func TestZoneNomsAmbigusNeDesignentRien(t *testing.T) {
	n := zoneNomsDesSlots([]grammar.ManagedPropertyRead{
		zoneNomAt(1, grammar.ManagedPropertyTagU32, 7), zoneNomAt(2, grammar.ManagedPropertyTagU32, 7),
		zoneNomAt(3, grammar.ManagedPropertyTagU32, 8), zoneNomAt(3, grammar.ManagedPropertyTagU32, 9),
		zoneNomAt(4, grammar.ManagedPropertyTagU32, 10), zoneReadAt(5, 0, grammar.ManagedPropertyTagU32, 0),
	})
	if len(n.parSlot) != 1 || n.parSlot[4] != 10 || len(n.parNom) != 1 || n.parNom[10] != 4 {
		t.Fatalf("tables %v / %v : seul le slot 4 (nom 10) est univoque", n.parSlot, n.parNom)
	}
}

// TestZoneProprietaireCanalNommeNonReprisParLeVote : le vote d'une zone sans nom ne reprend pas un
// canal qu'une zone nommee tient deja.
func TestZoneProprietaireCanalNommeNonReprisParLeVote(t *testing.T) {
	noms := zoneNoms{
		parSlot: map[uint32]uint32{10: zoneBlocsNommes[0].jauge, 11: zoneBlocsNommes[0].proprietaire},
		parNom:  map[uint32]uint32{zoneBlocsNommes[0].jauge: 10, zoneBlocsNommes[0].proprietaire: 11},
	}
	p := zoneOwnerSlotsOf(map[int]uint32{0: 10, 1: 20}, map[int]uint32{1: 11}, noms)
	if p.slot[0] != 11 || len(p.slot) != 1 || p.nommees != 1 || p.votees != 0 {
		t.Fatalf("rattachement %+v : la zone 1 ne peut pas reprendre le canal 11", p)
	}
}
