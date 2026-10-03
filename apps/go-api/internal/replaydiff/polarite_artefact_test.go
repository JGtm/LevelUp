package replaydiff

// polarite_artefact_test.go — LA TABLE DES POLARITES CONTRE UN ARTEFACT REEL (lot R4).
//
// `testdata/couverture_248972b2_s71.json` est la couverture EXTRAITE telle quelle d'un artefact
// cuit du parc (`data/cache/replays/halo_infinite/248972b2.json`, schema 71, le plus riche du parc
// local au 2026-09-28 : 453 feuilles numeriques de couverture). Le parc local ne porte aucun film
// d'Assaut : `bombStats.coverage` n'est tenu que par l'inventaire de la forme (ratchet).

import (
	"path/filepath"
	"testing"
)

const artefactCouverture = "couverture_248972b2_s71.json"

func empreinteArtefactCouverture(t *testing.T) Empreinte {
	t.Helper()
	doc, err := LireDocument(filepath.Join("testdata", artefactCouverture))
	if err != nil {
		t.Fatal(err)
	}
	return Empreindre(doc)
}

// TestPolarite_ArtefactReelToutClasse : chaque feuille numerique de couverture de l'artefact a
// une polarite, et les quatre classes y sont representees.
func TestPolarite_ArtefactReelToutClasse(t *testing.T) {
	e := empreinteArtefactCouverture(t)
	comptes := map[Polarite]int{}
	total := 0
	for k, m := range e.Mesures {
		_, chemin := decouper(k)
		if !m.EstNum || !estCouverture(chemin) {
			continue
		}
		total++
		p, _ := PolariteDe(chemin)
		comptes[p]++
		if p == PolariteInconnue {
			t.Errorf("feuille reelle sans polarite : %s", chemin)
		}
	}
	if total != 453 {
		t.Errorf("%d feuilles numeriques de couverture, attendu 453 (l'empreinte a change de forme ?)", total)
	}
	for _, p := range []Polarite{PolariteEchec, PolariteSucces, PolariteNeutre, PolariteTelemetrie} {
		if comptes[p] == 0 {
			t.Errorf("classe %v absente de l'artefact", p)
		}
	}
	t.Logf("artefact %s : %d feuilles, echec=%d succes=%d neutre=%d telemetrie=%d", artefactCouverture,
		total, comptes[PolariteEchec], comptes[PolariteSucces], comptes[PolariteNeutre],
		comptes[PolariteTelemetrie])
}

// TestPolarite_ArtefactReelSensDesEcarts : l'artefact reel confronte a une copie ou quelques
// feuilles bougent. Chaque ecart doit sortir dans le camp que sa classe dit.
func TestPolarite_ArtefactReelSensDesEcarts(t *testing.T) {
	a := empreinteArtefactCouverture(t)
	b := Empreinte{Schema: a.Schema, MatchID: a.MatchID, Mesures: map[string]Mesure{}}
	for k, m := range a.Mesures {
		b.Mesures[k] = m
	}
	bouger := func(chemin string, delta float64) string {
		k := cle("couverture", chemin)
		m, ok := a.Mesures[k]
		if !ok {
			t.Fatalf("%s absente de l'artefact", chemin)
		}
		b.Mesures[k] = Mesure{EstNum: true, Num: m.Num + delta}
		return chemin
	}
	attendu := map[string]string{
		bouger("coverage.continuousFire.holes", -46):       SensGain,       // echec qui baisse
		bouger("coverage.placements.byCause.no_owner", +3): SensPerte,      // echec qui monte
		bouger("coverage.groundWeapons.unknown", -2):       SensGain,       // echec qui baisse
		bouger("coverage.shots.attached", -5):              SensPerte,      // succes qui baisse
		bouger("coverage.bridge.livesNamed", +1):           SensGain,       // succes qui monte
		bouger("coverage.shots.available", +10):            SensChangement, // denominateur
		bouger("coverage.tracks.minPoints", +1):            SensChangement, // telemetrie
		bouger("coverage.weaponChanges.dropped", -1):       SensChangement, // ventilation
	}
	rap := Comparer(a, b)
	if len(rap.Differences) != len(attendu) {
		t.Fatalf("%d ecarts, attendu %d : %+v", len(rap.Differences), len(attendu), rap.Differences)
	}
	for _, d := range rap.Differences {
		if s := attendu[d.Metrique]; d.Sens != s {
			t.Errorf("%s %s -> %s : sens %q, attendu %q", d.Metrique, d.Ancien, d.Nouveau, d.Sens, s)
		}
	}
}
