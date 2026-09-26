package replay

// filmfacts_entete_test.go — L EN-TETE DU FICHIER DE FAITS, JALON J3 du
// PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25 (codec 2 : revisions par consommateur, gardes de
// l appelant, empreinte de l entree de catalogue).

import (
	"errors"
	"testing"
)

// TestFaitsDuCodec1SontRefusesSurLePrefixe : un fichier ecrit par le codec d AVANT le jalon J3 est
// PERIME, et il se dit perime sur son PREFIXE — avant que l en-tete, dont la forme a change, soit
// seulement lu. C est la regle du §4.4 du plan : toute montee du codec vient avec le refus de
// l ancien fichier.
func TestFaitsDuCodec1SontRefusesSurLePrefixe(t *testing.T) {
	blob, err := EncodeFilmFactsFile(fichierTemoin(t))
	if err != nil {
		t.Fatalf("encodage : %v", err)
	}
	// Le prefixe est `magie | uvarint(codec) | uvarint(schema) | uvarint(longueur)` ; le codec
	// tient sur un octet.
	ancien := append([]byte{}, blob...)
	ancien[len(magieFaitsDeFilm)] = 1
	if _, err := DecodeFilmFactsEntete(ancien); !errors.Is(err, ErrFilmFactsVersion) {
		t.Fatalf("un fichier du codec 1 est lu : err = %v, attendu ErrFilmFactsVersion — un "+
			"en-tete d une autre forme serait lu de travers", err)
	}
	if _, err := DecodeFilmFactsFile(ancien, goldenEntryPourTest(t)); !errors.Is(err, ErrFilmFactsVersion) {
		t.Fatalf("un fichier du codec 1 est relu en entier : err = %v", err)
	}
}

// TestUtilisable_CompareChaqueRevisionDeConsommateur : depuis le lot J3.3 la couche des faits a
// DEUX revisions (une par consommateur) ; des faits pris sous une autre valeur de L UNE ou de
// L AUTRE sont perimes.
func TestUtilisable_CompareChaqueRevisionDeConsommateur(t *testing.T) {
	entry := goldenEntryPourTest(t)
	e := enteteFrais(t, fichierTemoin(t))
	if err := e.Utilisable(entry, e.Gardes); err != nil {
		t.Fatalf("des faits frais sont refuses : %v", err)
	}
	for nom, muter := range map[string]func(*DecoderCoverage){
		"kill-feed": func(c *DecoderCoverage) { c.KillsourceRev += "-bis" },
		"objectifs": func(c *DecoderCoverage) { c.ObjectivesRev += "-bis" },
	} {
		autre := e
		muter(&autre.Coverage)
		if err := autre.Utilisable(entry, e.Gardes); !errors.Is(err, ErrFilmFactsRevisions) {
			t.Errorf("revision %s differente : err = %v, attendu ErrFilmFactsRevisions", nom, err)
		}
	}
}

// enteteFrais encode `f` sous les revisions COURANTES du binaire et rend son en-tete relu.
func enteteFrais(t *testing.T, f *FilmFactsFile) FilmFactsEntete {
	t.Helper()
	f.Coverage = *couvertureDuDecodeur(nil)
	blob, err := EncodeFilmFactsFile(f)
	if err != nil {
		t.Fatalf("encodage : %v", err)
	}
	e, err := DecodeFilmFactsEntete(blob)
	if err != nil {
		t.Fatalf("en-tete : %v", err)
	}
	return e
}

// fichierAvecGardes : le temoin, encode sous les revisions courantes et les gardes `g`.
func fichierAvecGardes(t *testing.T, g GardesDeCuisson) FilmFactsEntete {
	t.Helper()
	f := fichierTemoin(t)
	f.Gardes = g
	return enteteFrais(t, f)
}

// gardesCompletes : les trois canaux gardes balayes, sous un roster de deux xuids.
func gardesCompletes() GardesDeCuisson {
	return GardesDeCuisson{Drapeau: true, Zones: true, Bombe: true,
		Roster: empreinteDuRoster([]uint64{2533274819954312, 2533274800000001})}
}

// TestUtilisable_RefuseDesFaitsCuitsSansUneGardeDemandee : RA1-1 (P1). Des faits cuits SANS une
// garde — ici sans catalogue de zones, le cas de l ouvrier ou de la base indisponible — ne servent
// pas une cuisson qui la demande : ils ne portent pas le canal, et le rejeu publierait KOTH sans
// `zoneStates`, fige jusqu a la prochaine montee de revision.
func TestUtilisable_RefuseDesFaitsCuitsSansUneGardeDemandee(t *testing.T) {
	entry := goldenEntryPourTest(t)
	demandees := gardesCompletes()
	for nom, retirer := range map[string]func(*GardesDeCuisson){
		"zones":   func(g *GardesDeCuisson) { g.Zones = false },
		"drapeau": func(g *GardesDeCuisson) { g.Drapeau = false },
		"bombe":   func(g *GardesDeCuisson) { g.Bombe = false },
	} {
		cuites := gardesCompletes()
		retirer(&cuites)
		e := fichierAvecGardes(t, cuites)
		if err := e.Utilisable(entry, demandees); !errors.Is(err, ErrFilmFactsGardes) {
			t.Errorf("faits cuits sans la garde %s, garde demandee : err = %v, attendu "+
				"ErrFilmFactsGardes", nom, err)
		}
	}
}

// TestUtilisable_AccepteUnSurEnsembleDeGardes : des faits cuits sous PLUS de gardes que la cuisson
// n en demande la servent — le rejeu retire ce qu elle ne demande pas
// ([FilmInputs.restreindreAuxGardes]).
func TestUtilisable_AccepteUnSurEnsembleDeGardes(t *testing.T) {
	entry := goldenEntryPourTest(t)
	e := fichierAvecGardes(t, gardesCompletes())
	demandees := GardesDeCuisson{Roster: gardesCompletes().Roster}
	if err := e.Utilisable(entry, demandees); err != nil {
		t.Errorf("des faits cuits sous un sur-ensemble de gardes sont refuses : %v", err)
	}
	if err := e.Utilisable(entry, gardesCompletes()); err != nil {
		t.Errorf("des faits cuits sous les MEMES gardes sont refuses : %v", err)
	}
}

// TestUtilisable_RefuseUnRosterDifferent : la table d index de joueur se lit sur le roster de
// l appelant ; des faits cuits sous un autre roster ne sont pas ceux de cette cuisson.
func TestUtilisable_RefuseUnRosterDifferent(t *testing.T) {
	entry := goldenEntryPourTest(t)
	e := fichierAvecGardes(t, gardesCompletes())
	autre := gardesCompletes()
	autre.Roster = empreinteDuRoster([]uint64{2533274819954312})
	if err := e.Utilisable(entry, autre); !errors.Is(err, ErrFilmFactsGardes) {
		t.Errorf("roster different : err = %v, attendu ErrFilmFactsGardes", err)
	}
	vide := gardesCompletes()
	vide.Roster = empreinteDuRoster(nil)
	if err := e.Utilisable(entry, vide); !errors.Is(err, ErrFilmFactsGardes) {
		t.Errorf("roster VIDE (base indisponible) contre faits cuits sous un roster : err = %v, "+
			"attendu ErrFilmFactsGardes", err)
	}
}

// TestGardesDeCuissonVoyagentDansLEnTete : les gardes sont relues a l identique, sur l EN-TETE seul.
func TestGardesDeCuissonVoyagentDansLEnTete(t *testing.T) {
	for _, g := range []GardesDeCuisson{{}, gardesCompletes(), {Zones: true, Roster: empreinteDuRoster(nil)}} {
		if relue := fichierAvecGardes(t, g).Gardes; relue != g {
			t.Errorf("gardes %+v relues %+v", g, relue)
		}
	}
}

// TestEmpreinteDuRosterEstCelleDeLEnsemble : l ordre, les doublons et le xuid nul ne comptent pas
// — ce sont les memes normalisations que `rosterOf`.
func TestEmpreinteDuRosterEstCelleDeLEnsemble(t *testing.T) {
	a := empreinteDuRoster([]uint64{3, 1, 2})
	b := empreinteDuRoster([]uint64{2, 0, 1, 3, 3})
	if a != b {
		t.Error("deux listes du meme ensemble de xuids rendent deux empreintes")
	}
	if a == empreinteDuRoster([]uint64{1, 2}) {
		t.Error("deux ensembles differents rendent la meme empreinte")
	}
}

// TestRejeuDepuisLesFaitsRestreintAuxGardesDemandees : un canal balaye par les faits mais que la
// cuisson ne demande pas est RETIRE avant l assemblage — le rejeu depuis un sur-ensemble rend ce
// qu un decodage sous les gardes demandees rendrait.
func TestRejeuDepuisLesFaitsRestreintAuxGardesDemandees(t *testing.T) {
	in := fichierTemoin(t).Facts.FilmInputs
	in.FlagGauge, in.FlagGaugeScanned = in.ZoneReads, true
	in.restreindreAuxGardes(GardesDeCuisson{})
	if in.ZoneScanned || in.ZoneReads != nil || in.BombReads != nil || in.FlagGaugeScanned ||
		in.FlagGauge != nil || len(in.FlagMarks.Marks) != 0 {
		t.Errorf("canaux non demandes conserves : zones %v/%d, bombe %d, jauge %v/%d, marques %d",
			in.ZoneScanned, len(in.ZoneReads), len(in.BombReads), in.FlagGaugeScanned,
			len(in.FlagGauge), len(in.FlagMarks.Marks))
	}
	complet := fichierTemoin(t).Facts.FilmInputs
	garde := complet
	garde.restreindreAuxGardes(gardesCompletes())
	if !garde.ZoneScanned || len(garde.ZoneReads) != len(complet.ZoneReads) ||
		len(garde.BombReads) != len(complet.BombReads) {
		t.Error("des canaux DEMANDES ont ete retires")
	}
}
