package replay

import (
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// flag_retour_auto_test.go — LE RETOUR AUTOMATIQUE DU DRAPEAU, sans film.
//
// La regle qu'ils figent : un drapeau reste au sol RENTRE CHEZ LUI quand l'OBJET renait a son
// socle, sans qu'aucun joueur ne soit credite. C'est la seule chaine qui date ce retour-la — le
// statborg n'en porte rien, puisque personne ne le provoque. Verite terrain sur films reels :
// `ctf_retour_zone_research_test.go`, sous garde d'environnement.

// TestFlagRetourAutomatiqueParLObjet — le drapeau au sol rentre a la naissance de l'objet AU SOCLE.
func TestFlagRetourAutomatiqueParLObjet(t *testing.T) {
	tracks := []Track{flagTestTrack(10, "1", 0, 99, 95, 95)}
	deaths := []Death{{XUID: 1, TimeMS: 4000}}
	scan := FlagCarryScan{
		Scanned: true, Signals: flagTestSignals(),
		Events: []objectiveevents.NamedEvent{
			{TimeMS: 1000, Slot: 12, Stat: objectiveevents.StatFlagSteals},
		},
		Identity: objectiveevents.FlatRoundIdentity(map[int]string{12: "1"}),
		Spawns:   []FlagSpawn{{Team: 0, X: 0, Y: 0}, {Team: 1, X: 100, Y: 100}},
		// La vie libre nait A 7 s AU SOCLE de l'equipe 1 : c'est le drapeau qui rentre.
		Free: []flagFreeLife{{
			T0US: 7_000_000, T1US: 7_000_000,
			Pts: []flagFreeSample{{TUS: 7_000_000, X: 100, Y: 100}},
		}},
	}
	got, cov := buildFlagCarries(scan, flagTestCtx(tracks, deaths, 100))
	if cov.HomeByObject != 1 {
		t.Fatalf("couverture %+v : une rentree par l'objet attendue", *cov)
	}
	f := flagOfTeam(t, got, 1)
	assertFlagStates(t, f, []string{FlagStateHome, FlagStateCarried, FlagStateDropped, FlagStateHome})
	last := f.Spans[len(f.Spans)-1]
	if last.T0 != 70 {
		t.Fatalf("le retour commence a la frame %d, attendu 70 (7 s a 100 ms/frame)", last.T0)
	}
	if last.X != 100 || last.Y != 100 {
		t.Fatalf("le drapeau rentre en (%g,%g), attendu son socle (100,100)", last.X, last.Y)
	}
}

// TestFlagRetourAutomatiqueNAgitPasSurUnDrapeauPorte — une naissance au socle pendant qu'un
// joueur PORTE le drapeau ne change rien : c'est le re-spawn normal, pas un retour.
func TestFlagRetourAutomatiqueNAgitPasSurUnDrapeauPorte(t *testing.T) {
	tracks := []Track{flagTestTrack(10, "1", 0, 99, 95, 95)}
	scan := FlagCarryScan{
		Scanned: true, Signals: flagTestSignals(),
		Events: []objectiveevents.NamedEvent{
			{TimeMS: 1000, Slot: 12, Stat: objectiveevents.StatFlagSteals},
		},
		Identity: objectiveevents.FlatRoundIdentity(map[int]string{12: "1"}),
		Spawns:   []FlagSpawn{{Team: 0, X: 0, Y: 0}, {Team: 1, X: 100, Y: 100}},
		Free: []flagFreeLife{{
			T0US: 5_000_000, T1US: 5_000_000,
			Pts: []flagFreeSample{{TUS: 5_000_000, X: 100, Y: 100}},
		}},
	}
	got, cov := buildFlagCarries(scan, flagTestCtx(tracks, nil, 100))
	if cov.HomeByObject != 0 {
		t.Fatalf("couverture %+v : aucune rentree ne doit s'appliquer a un drapeau porte", *cov)
	}
	f := flagOfTeam(t, got, 1)
	assertFlagStates(t, f, []string{FlagStateHome, FlagStateCarriedOpen})
}

// TestFlagRetourAutomatiqueSAbstientSiUnAutreDrapeauGitLa — l'ABSTENTION : le drapeau adverse
// tombe au pied du socle produit la MEME naissance ; rien ne les depart, donc on ne renvoie rien.
func TestFlagRetourAutomatiqueSAbstientSiUnAutreDrapeauGitLa(t *testing.T) {
	// « 2 » vole le drapeau de l'equipe 0 A SON SOCLE, le porte jusqu'au socle de l'equipe 1 et
	// y meurt : le drapeau 0 git DONC exactement la ou le drapeau 1 renaitrait.
	porteur := flagTestTrack(12, "2", 0, 20, 0, 0)
	porteur.Points = append(porteur.Points, flagTestTrack(12, "2", 21, 99, 100, 100).Points...)
	porteur.EndFrame = 99
	tracks := []Track{
		flagTestTrack(10, "1", 0, 99, 95, 95), // vole le drapeau de l'equipe 1, meurt sur place
		porteur,
	}
	deaths := []Death{{XUID: 1, TimeMS: 4000}, {XUID: 2, TimeMS: 4000}}
	scan := FlagCarryScan{
		Scanned: true, Signals: flagTestSignals(),
		Events: []objectiveevents.NamedEvent{
			{TimeMS: 1000, Slot: 12, Stat: objectiveevents.StatFlagSteals},
			{TimeMS: 1000, Slot: 14, Stat: objectiveevents.StatFlagSteals},
		},
		Identity: objectiveevents.FlatRoundIdentity(map[int]string{12: "1", 14: "2"}),
		Spawns:   []FlagSpawn{{Team: 0, X: 0, Y: 0}, {Team: 1, X: 100, Y: 100}},
		Free: []flagFreeLife{{
			T0US: 7_000_000, T1US: 7_000_000,
			Pts: []flagFreeSample{{TUS: 7_000_000, X: 100, Y: 100}},
		}},
	}
	_, cov := buildFlagCarries(scan, flagTestCtx(tracks, deaths, 100))
	if cov.HomeByObject != 0 || cov.AmbiguousHomecomings != 1 {
		t.Fatalf("couverture %+v : la rentree doit s'abstenir et se compter", *cov)
	}
}
