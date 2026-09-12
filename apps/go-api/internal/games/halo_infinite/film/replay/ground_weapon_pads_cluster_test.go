package replay

// ground_weapon_pads_cluster_test.go — LES TESTS DE LA REGLE DE GRAPPE ET DE LA REGLE DE CYCLE,
// isoles de tout film pour qu'elles soient TESTEES et non seulement executees.
//
// LA REGLE ELLE-MEME EST EN PRODUCTION depuis la phase 3 du plan (`ground_weapon_rules.go`) :
// c'est elle que l'artefact de rejeu publie, et c'est elle que ces tests et les instruments sous
// garde appellent. Une seconde copie de la regle de grappe ou du verdict de stabilite aurait
// diverge au premier correctif — le critere de catalogue du plan (item 1.4) a tranche entretemps
// (PAR MATCH, aucun catalogue versionne), et c'est ce que le document publie.
//
// CE QUI RESTE ICI : les tests sans garde (ils tombent sous le gate ordinaire du depot) et les
// deux RESUMES de mesure (`gwPadsSpread`, `gwPadsCycleSummary`), qui n'ecrivent que des lignes
// de journal et n'ont rien a faire en production.

import (
	"fmt"
	"math"
	"testing"
)

// gwPadsSpread resume la taille des grappes — le temoin de Notion 11 se lit la : un socle
// recurre, une grappe d'une seule apparition est un lacher isole.
func gwPadsSpread(in []gwPadCluster) string {
	if len(in) == 0 {
		return "aucune grappe"
	}
	sizes := make([]float64, 0, len(in))
	singles, maxN := 0, 0
	for _, c := range in {
		sizes = append(sizes, float64(len(c.TS)))
		if len(c.TS) == 1 {
			singles++
		}
		if len(c.TS) > maxN {
			maxN = len(c.TS)
		}
	}
	return fmt.Sprintf("grappes %d · a une seule apparition %d · mediane %.0f · max %d",
		len(in), singles, gwPadsQuantile(sizes, 0.5), maxN)
}

// gwPadsCycleSummary compte les socles dont le cycle est ETABLI, et donne la mediane des
// medianes — la seule facon honnete de resumer des cycles qui n'ont pas tous la meme stabilite.
func gwPadsCycleSummary(socles []gwPadCluster) string {
	if len(socles) == 0 {
		return "aucun socle : aucun cycle"
	}
	var med []float64
	etablis := 0
	for _, p := range socles {
		c := gwPadsCycle(p.TS)
		if !c.Established {
			continue
		}
		etablis++
		med = append(med, c.MedianS)
	}
	if etablis == 0 {
		return fmt.Sprintf("socles %d · cycle ETABLI 0 · aucun cycle publiable", len(socles))
	}
	return fmt.Sprintf("socles %d · cycle ETABLI %d (%s) · mediane des medianes %.1f s",
		len(socles), etablis, gwPadsPart(etablis, len(socles)), gwPadsQuantile(med, 0.5))
}

// --- TESTS DE LA REGLE (sans garde : ils tournent avec le paquet) ------------------------

// TestGwPadsClusterSepareDeuxSoclesVoisins : la regle « le plus proche dans le rayon » ne doit
// pas fondre deux socles distants de plus du rayon, meme quand les apparitions alternent.
func TestGwPadsClusterSepareDeuxSoclesVoisins(t *testing.T) {
	app := []gwPadApparition{
		{Kind: gwPadKindWeapon, Family: "A", X: 0, TUS: 1},
		{Kind: gwPadKindWeapon, Family: "A", X: 3, TUS: 2},
		{Kind: gwPadKindWeapon, Family: "A", X: 0.4, TUS: 3},
		{Kind: gwPadKindWeapon, Family: "A", X: 3.3, TUS: 4},
	}
	got, _ := gwPadsClusterAssign(app)
	if len(got) != 2 {
		t.Fatalf("2 grappes attendues, %d obtenues : %+v", len(got), got)
	}
	for _, c := range got {
		if len(c.TS) != 2 {
			t.Fatalf("chaque grappe doit porter 2 apparitions, %d : %+v", len(c.TS), c)
		}
	}
}

// TestGwPadsClusterNeMelangePasLesFamilles : deux armes differentes au MEME endroit sont deux
// grappes — un socle est une famille a une position, jamais une position seule.
func TestGwPadsClusterNeMelangePasLesFamilles(t *testing.T) {
	app := []gwPadApparition{
		{Kind: gwPadKindWeapon, Family: "A", TUS: 1},
		{Kind: gwPadKindWeapon, Family: "B", TUS: 2},
		{Kind: gwPadKindPowerup, Family: "A", TUS: 3},
	}
	got, _ := gwPadsClusterAssign(app)
	if len(got) != 3 {
		t.Fatalf("3 grappes attendues (2 familles + 1 nature), %d obtenues : %+v", len(got), got)
	}
}

// TestGwPadsClusterAssignRendLaGrappeDeChaqueEntree : l'assignation doit designer la grappe
// RENDUE (apres l'ordre total), pas celle de l'ordre de decouverte — l'erreur silencieuse que
// la permutation evite.
func TestGwPadsClusterAssignRendLaGrappeDeChaqueEntree(t *testing.T) {
	app := []gwPadApparition{
		{Kind: gwPadKindWeapon, Family: "Z", X: 10, TUS: 1},
		{Kind: gwPadKindWeapon, Family: "A", X: 0, TUS: 2},
		{Kind: gwPadKindWeapon, Family: "Z", X: 10.3, TUS: 3},
	}
	pads, assign := gwPadsClusterAssign(app)
	if len(pads) != 2 || len(assign) != 3 {
		t.Fatalf("2 grappes et 3 assignations attendues : %d / %d", len(pads), len(assign))
	}
	for i, a := range app {
		p := pads[assign[i]]
		if p.Family != a.Family {
			t.Fatalf("apparition %d (%s) assignee a la grappe %q", i, a.Family, p.Family)
		}
		if dist3([3]float32{p.X, p.Y, p.Z}, [3]float32{a.X, a.Y, a.Z}) > gwPadRadiusM {
			t.Fatalf("apparition %d assignee a une grappe hors du rayon : %+v", i, p)
		}
	}
}

// TestGwPadsCycleRefuseDeConclureSurUnCycleInstable : le seuil de la decision 3 ne se contourne
// pas — un cycle disperse se publie « non etabli », pas arrondi.
func TestGwPadsCycleRefuseDeConclureSurUnCycleInstable(t *testing.T) {
	stable := gwPadsCycle([]uint64{0, 100_000_000, 200_000_000, 300_000_000})
	if !stable.Established || math.Abs(stable.MedianS-100) > 0.01 {
		t.Fatalf("cycle regulier de 100 s attendu etabli : %+v", stable)
	}
	instable := gwPadsCycle([]uint64{0, 10_000_000, 200_000_000, 205_000_000})
	if instable.Established {
		t.Fatalf("cycle disperse ne doit PAS etre etabli : %+v", instable)
	}
	court := gwPadsCycle([]uint64{0, 100_000_000})
	if court.Established || court.Gaps != 1 || math.Abs(court.MedianS-100) > 0.01 {
		t.Fatalf("un ecart unique se MESURE (100 s) mais n'ETABLIT rien : %+v", court)
	}
	if vide := gwPadsCycle([]uint64{42}); vide.Gaps != 0 || vide.MedianS != 0 {
		t.Fatalf("une apparition unique n'a aucun ecart : %+v", vide)
	}
}
