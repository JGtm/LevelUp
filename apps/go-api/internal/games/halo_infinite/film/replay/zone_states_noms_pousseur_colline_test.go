package replay

// zone_states_noms_pousseur_colline_test.go — LE POUSSEUR D UNE ZONE ET LE PROPRIETAIRE D UNE
// COLLINE, DESIGNES PAR LE NOM (zone_states_owner_nom.go), sur des series CONSTRUITES.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
)

// nomsDe monte les tables de noms d un cas : slot -> nom.
func nomsDe(parSlot map[uint32]uint32) zoneNoms {
	parNom := map[uint32]uint32{}
	for s, n := range parSlot {
		parNom[n] = s
	}
	return zoneNoms{parSlot: parSlot, parNom: parNom}
}

// blocA est le premier bloc du vocabulaire.
var blocA = zoneBlocsNommes[0]

// TestPousseurLuParLeNomSansElection : une seule rampe aboutie, l election n elit rien ; le nom du
// pousseur du bloc de la jauge designe pourtant son canal.
func TestPousseurLuParLeNomSansElection(t *testing.T) {
	ramps := findZoneRamps(9, rampeDeJauge(10, 100, 0.99))
	owner := []zoneSample{{t: 111, v: 1}}
	pousseur := []zoneSample{{t: 105, v: 1}}
	ser := serieTemoin(map[uint32][]zoneSample{5: owner, 7: pousseur})
	ser.noms = nomsDe(map[uint32]uint32{9: blocA.jauge, 5: blocA.proprietaire, 7: blocA.pousseur})
	choix := zoneCapturerOf(ser, ramps, 9, zoneCapturerCtx{owner: owner, ownerSlot: 5, win: 20})
	if !choix.nomme || choix.slot != 7 || choix.eluOK || len(choix.serie) != 1 {
		t.Fatalf("choix %+v : attendu le canal 7 par le nom, aucun elu", choix)
	}
	cov := &ZonesCoverage{}
	fb := fallback.NouveauCompteur()
	if disc := tallyZoneCapturer(choix, 0, cov, fb, nil); len(disc) != 0 || cov.CapturerNamed != 1 {
		t.Fatalf("discordances %v, nommees %d : attendu aucune et 1", disc, cov.CapturerNamed)
	}
	if n := fb.Compte(fallback.NomZonePousseurParElection); n != 0 {
		t.Errorf("repli par election compte %d fois alors que le nom designe le canal", n)
	}
}

// TestPousseurLeNomPrimeSurLElection : l election elit le canal 6, le nom designe le canal 8. Le
// nom est retenu et la discordance se compte.
func TestPousseurLeNomPrimeSurLElection(t *testing.T) {
	gauge := append(rampeDeJauge(10, 100, 0.99), rampeDeJauge(10, 300, 0.99)...)
	ramps := findZoneRamps(9, gauge)
	owner := []zoneSample{{t: 111, v: 1}, {t: 311, v: 0}}
	elu := []zoneSample{{t: 105, v: 1}, {t: 305, v: 0}}
	nomme := []zoneSample{{t: 104, v: 1}, {t: 304, v: 1}}
	ser := serieTemoin(map[uint32][]zoneSample{5: owner, 6: elu, 8: nomme})
	ser.noms = nomsDe(map[uint32]uint32{9: blocA.jauge, 8: blocA.pousseur})
	choix := zoneCapturerOf(ser, ramps, 9, zoneCapturerCtx{owner: owner, ownerSlot: 5, win: 20})
	if !choix.nomme || choix.slot != 8 || !choix.eluOK || choix.eluSlot != 6 {
		t.Fatalf("choix %+v : attendu le canal 8 par le nom, 6 par l election", choix)
	}
	cov := &ZonesCoverage{}
	disc := tallyZoneCapturer(choix, 2, cov, nil, nil)
	if len(disc) != 1 || disc[0].canal != zoneCanalPousseur || cov.CapturerElectionDisagreed != 1 {
		t.Fatalf("discordances %+v, couverture %+v : attendu une discordance de pousseur", disc, cov)
	}
}

// TestPousseurParElectionFauteDeNom : sans nom au vocabulaire, l election repond, et le repli se
// compte.
func TestPousseurParElectionFauteDeNom(t *testing.T) {
	gauge := append(rampeDeJauge(10, 100, 0.99), rampeDeJauge(10, 300, 0.99)...)
	ramps := findZoneRamps(9, gauge)
	owner := []zoneSample{{t: 111, v: 1}, {t: 311, v: 0}}
	elu := []zoneSample{{t: 105, v: 1}, {t: 305, v: 0}}
	ser := serieTemoin(map[uint32][]zoneSample{5: owner, 6: elu})
	choix := zoneCapturerOf(ser, ramps, 9, zoneCapturerCtx{owner: owner, ownerSlot: 5, win: 20})
	if !choix.parElection || choix.slot != 6 {
		t.Fatalf("choix %+v : attendu le canal 6 par l election", choix)
	}
	fb := fallback.NouveauCompteur()
	cov := &ZonesCoverage{}
	tallyZoneCapturer(choix, 0, cov, fb, nil)
	if n := fb.Compte(fallback.NomZonePousseurParElection); n != 1 || cov.CapturerNamed != 0 {
		t.Fatalf("repli compte %d fois, nommees %d : attendu 1 et 0", n, cov.CapturerNamed)
	}
}

// TestLeNeutreAuSommetNEstPasLaPoussee : le pousseur repasse au neutre dans la frame meme du
// sommet ; la valeur de la rampe est celle de la poussee. Un neutre seul dans la fenetre reste
// une reponse (« personne ne pousse »).
func TestLeNeutreAuSommetNEstPasLaPoussee(t *testing.T) {
	ramps := findZoneRamps(9, rampeDeJauge(10, 100, 0.99))
	r := ramps[0]
	v, ok := zoneValueDuringRamp([]zoneSample{{t: r.t0, v: 1}, {t: r.tPeak, v: zoneNeutralOwner}}, r)
	if !ok || v != 1 {
		t.Fatalf("valeur %d (%v), attendu 1 : le neutre du sommet est la fin de la poussee", v, ok)
	}
	v, ok = zoneValueDuringRamp([]zoneSample{{t: r.tPeak, v: zoneNeutralOwner}}, r)
	if !ok || v != zoneNeutralOwner {
		t.Fatalf("valeur %d (%v), attendu le neutre : seul dans la fenetre, il repond", v, ok)
	}
}

// TestCollineProprietaireParLeNom : le nom du designateur designe le proprietaire de son bloc,
// meme quand ce n est pas le slot voisin ; la discordance avec le voisin se compte.
func TestCollineProprietaireParLeNom(t *testing.T) {
	ser := serieTemoin(map[uint32][]zoneSample{})
	ser.noms = nomsDe(map[uint32]uint32{40: blocA.cle, 42: blocA.proprietaire})
	cov := &ZonesCoverage{}
	fb := fallback.NouveauCompteur()
	if s := hillOwnerSlotOf(ser, hillDesignator{slot: 40}, cov, fb); s != 42 {
		t.Fatalf("canal %d, attendu 42 par le nom", s)
	}
	if cov.OwnerNamed != 1 || cov.OwnerVoteDisagreed != 1 {
		t.Errorf("couverture %+v : attendu 1 nommee, 1 discordance avec le voisin", cov)
	}
	if n := fb.Compte(fallback.NomCollineProprietaireVoisinDuDesignateur); n != 0 {
		t.Errorf("repli compte %d fois alors que le nom designe le canal", n)
	}
}

// TestCollineProprietaireVoisinFauteDeNom : sans nom au vocabulaire, le slot voisin du
// designateur, et le repli se compte.
func TestCollineProprietaireVoisinFauteDeNom(t *testing.T) {
	ser := serieTemoin(map[uint32][]zoneSample{})
	cov := &ZonesCoverage{}
	fb := fallback.NouveauCompteur()
	if s := hillOwnerSlotOf(ser, hillDesignator{slot: 40}, cov, fb); s != 41 {
		t.Fatalf("canal %d, attendu le voisin 41", s)
	}
	if n := fb.Compte(fallback.NomCollineProprietaireVoisinDuDesignateur); n != 1 || cov.OwnerNamed != 0 {
		t.Fatalf("repli compte %d fois, nommees %d : attendu 1 et 0", n, cov.OwnerNamed)
	}
}
