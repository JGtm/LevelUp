package replay

// killpos_test.go — LES POSITIONS DE MORT, ET SURTOUT CE QUI RESTE NIL.
//
// Comme pour les fermetures, l'essentiel des cas testés sont des ABSTENTIONS : une position
// absente doit rester absente. Une ligne `kill_positions` posée au mauvais endroit serait
// invisible à l'écran et crédible — c'est exactement le genre d'erreur que ce chantier a déjà
// payé une fois.

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

func TestKillPositionsPlaceLesDeuxJoueurs(t *testing.T) {
	pos := []filmdec.BipedPosition{
		posAt(1, 10_000_000, 1, 2, 3),
		posAt(2, 10_000_000, 4, 5, 6),
	}
	slotXUID := map[uint32]uint64{1: 111, 2: 222}
	kills := []KillRef{{KillerXUID: 111, VictimXUID: 222, TimeMS: 10_000}}
	got, rep := BuildKillPositions(pos, slotXUID, kills, 0)
	if len(got) != 1 || got[0].Killer == nil || got[0].Victim == nil {
		t.Fatalf("les deux positions devaient être trouvées : %+v", got)
	}
	if got[0].Killer.X != 1 || got[0].Victim.X != 4 {
		t.Errorf("coordonnées inattendues : %+v / %+v", got[0].Killer, got[0].Victim)
	}
	if rep.Both != 1 || rep.Dropped != 0 {
		t.Errorf("compte rendu inattendu : %+v", rep)
	}
}

// TestKillPositionsLaisseNilHorsTolerance : une position trop lointaine n'est PAS une position.
func TestKillPositionsLaisseNilHorsTolerance(t *testing.T) {
	pos := []filmdec.BipedPosition{
		posAt(1, 10_000_000, 1, 2, 3),
		posAt(2, 5_000_000, 4, 5, 6), // 5 s avant la mort : hors tolérance
	}
	slotXUID := map[uint32]uint64{1: 111, 2: 222}
	kills := []KillRef{{KillerXUID: 111, VictimXUID: 222, TimeMS: 10_000}}
	got, rep := BuildKillPositions(pos, slotXUID, kills, 0)
	if len(got) != 1 || got[0].Victim != nil {
		t.Fatalf("la victime devait rester nil : %+v", got)
	}
	if rep.KillerOnly != 1 {
		t.Errorf("compte rendu inattendu : %+v", rep)
	}
}

// TestKillPositionsNEcritRienSansAucunePosition : une ligne vide ne vaut rien.
func TestKillPositionsNEcritRienSansAucunePosition(t *testing.T) {
	pos := []filmdec.BipedPosition{posAt(1, 1_000_000, 1, 2, 3)}
	slotXUID := map[uint32]uint64{1: 111}
	kills := []KillRef{{KillerXUID: 111, VictimXUID: 999, TimeMS: 60_000}}
	got, rep := BuildKillPositions(pos, slotXUID, kills, 0)
	if len(got) != 0 {
		t.Fatalf("aucune position : rien ne devait être écrit, %+v", got)
	}
	if rep.Dropped != 1 || rep.NoBridge != 1 {
		t.Errorf("compte rendu inattendu : %+v (le xuid 999 n'est pas au pont)", rep)
	}
}

// TestKillPositionsRefuseDeTrancherEntreDeuxCorps : l'ambiguïté n'est pas arbitrée.
func TestKillPositionsRefuseDeTrancherEntreDeuxCorps(t *testing.T) {
	// Deux slots du MÊME joueur portent un échantillon à l'instant de la mort : le découpage
	// des vies est faux ici, et choisir serait un coup de dé.
	pos := []filmdec.BipedPosition{
		posAt(1, 10_000_000, 1, 2, 3),
		posAt(2, 10_000_000, 9, 9, 9),
		posAt(3, 10_000_000, 4, 5, 6),
	}
	slotXUID := map[uint32]uint64{1: 111, 2: 111, 3: 222}
	kills := []KillRef{{KillerXUID: 111, VictimXUID: 222, TimeMS: 10_000}}
	got, rep := BuildKillPositions(pos, slotXUID, kills, 0)
	if len(got) != 1 || got[0].Killer != nil {
		t.Fatalf("le tueur a deux corps : sa position devait rester nil, %+v", got)
	}
	if rep.VictimOnly != 1 {
		t.Errorf("compte rendu inattendu : %+v", rep)
	}
}

// TestKillPositionsAppliqueLeDecalageDHorloge : le fil des morts et le film n'ont pas la même
// origine, et l'oublier placerait toutes les morts ailleurs.
func TestKillPositionsAppliqueLeDecalageDHorloge(t *testing.T) {
	pos := []filmdec.BipedPosition{posAt(1, 14_000_000, 1, 2, 3), posAt(2, 14_000_000, 4, 5, 6)}
	slotXUID := map[uint32]uint64{1: 111, 2: 222}
	kills := []KillRef{{KillerXUID: 111, VictimXUID: 222, TimeMS: 10_000}}
	if _, rep := BuildKillPositions(pos, slotXUID, kills, 0); rep.Dropped != 1 {
		t.Fatalf("sans décalage, la mort tombe à 10 s et rien n'est trouvé : %+v", rep)
	}
	_, rep := BuildKillPositions(pos, slotXUID, kills, 4_000_000)
	if rep.Both != 1 {
		t.Fatalf("avec le décalage de 4 s, les deux positions devaient être trouvées : %+v", rep)
	}
}
