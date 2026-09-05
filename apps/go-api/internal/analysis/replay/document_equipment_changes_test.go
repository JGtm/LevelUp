package replay

// document_equipment_changes_test.go — la projection des ramassages et des consommations
// d'équipement sur l'axe du document.
//
// POURQUOI CES TESTS EXISTENT. Comme pour les changements d'arme, le golden d'assemblage ne
// couvre PAS ce calque : son fixture d'entrées a été figé avant lui et ne porte aucun
// changement d'équipement. Sans les tests ci-dessous, le filtrage des réapparitions, la
// conversion en frames et le report du témoin de complétude n'auraient aucune couverture de
// non-régression.

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// ecOrigin / ecStep : une origine et un pas ronds, pour que les frames attendues se lisent.
const (
	ecOrigin = uint64(1_000_000)
	ecStep   = uint64(100_000) // 10 frames par seconde
)

func TestBuildEquipmentChangesEcarteLesReapparitions(t *testing.T) {
	in := []filmdec.EquipmentChange{
		{TimestampUS: ecOrigin, Slot: 7, Rank: 4, Previous: filmdec.AbilitySetNoRank,
			Kind: filmdec.EquipmentSpawned},
		{TimestampUS: ecOrigin + 2_000_000, Slot: 7, Rank: 6, Previous: 4,
			Kind: filmdec.EquipmentTaken},
	}
	got, cov := buildEquipmentChanges(in, filmdec.EquipmentChangeStats{Lives: 1}, ecOrigin, ecStep)
	if len(got) != 1 {
		t.Fatalf("publiés = %d, attendu 1 : une réapparition équipée n'est PAS un ramassage et "+
			"ne doit pas gonfler le compte", len(got))
	}
	if cov.Spawned != 1 || cov.Published != 1 || cov.Taken != 1 || cov.Decoded != 2 {
		t.Errorf("couverture = %+v, attendu decoded=2 published=1 taken=1 spawned=1", cov)
	}
	if got[0].T != 20 {
		t.Errorf("T = %d, attendu 20 (2 s après l'origine au pas de 100 ms)", got[0].T)
	}
	if got[0].R != 6 || got[0].From != 4 {
		t.Errorf("r=%d from=%d, attendu r=6 from=4 : le rang précédent doit voyager avec "+
			"l'événement, c'est lui qui dit ce qui a été remplacé", got[0].R, got[0].From)
	}
}

func TestBuildEquipmentChangesConsommation(t *testing.T) {
	in := []filmdec.EquipmentChange{
		{TimestampUS: ecOrigin, Slot: 3, Rank: filmdec.AbilitySetNoRank, Previous: 9,
			Kind: filmdec.EquipmentSpent},
	}
	got, cov := buildEquipmentChanges(in, filmdec.EquipmentChangeStats{Lives: 1}, ecOrigin, ecStep)
	if len(got) != 1 || got[0].Kind != EquipmentSpent {
		t.Fatalf("publiés = %v, attendu une consommation", got)
	}
	if got[0].R != NoAbilityRank {
		t.Errorf("r = %d, attendu %d : après une consommation le joueur ne porte plus rien, et "+
			"le rang zéro existe — la sentinelle est publiée plutôt que le champ omis",
			got[0].R, NoAbilityRank)
	}
	if got[0].From != 9 {
		t.Errorf("from = %d, attendu 9 : ce qui vient d'être usé est la seule chose "+
			"intéressante d'une consommation", got[0].From)
	}
	if cov.Spent != 1 {
		t.Errorf("couverture spent = %d, attendu 1", cov.Spent)
	}
}

func TestBuildEquipmentChangesEcarteAvantOrigine(t *testing.T) {
	in := []filmdec.EquipmentChange{
		{TimestampUS: ecOrigin - 1, Slot: 3, Rank: 4, Previous: 6, Kind: filmdec.EquipmentTaken},
	}
	got, cov := buildEquipmentChanges(in, filmdec.EquipmentChangeStats{}, ecOrigin, ecStep)
	if len(got) != 0 || cov.BeforeOrigin != 1 {
		t.Fatalf("publiés=%d beforeOrigin=%d : un rejeu ne montre pas ce qui précède sa "+
			"première frame", len(got), cov.BeforeOrigin)
	}
}

func TestBuildEquipmentChangesReporteLeTemoinDeCompletude(t *testing.T) {
	// Le témoin de complétude ne se recalcule PAS ici : il est lu par le décodeur, qui seul
	// voit le compteur de rotation. Ce test verrouille son passage jusqu'à la couverture —
	// une couverture qui ne le porterait pas laisserait croire que tout a été vu.
	st := filmdec.EquipmentChangeStats{
		Lives: 44, MissedEstimate: 3, CounterJumps: 2, LivesFirstOffSpec: 1, Repeats: 0,
	}
	_, cov := buildEquipmentChanges(nil, st, ecOrigin, ecStep)
	if cov.Lives != 44 || cov.MissedEstimate != 3 || cov.CounterJumps != 2 ||
		cov.LivesFirstOffSpec != 1 {
		t.Errorf("couverture = %+v : le témoin de complétude du décodeur doit arriver intact", cov)
	}
}

func TestBuildEquipmentChangesPublieRecuperationEtGap(t *testing.T) {
	in := []filmdec.EquipmentChange{
		// Une émission RÉCUPÉRÉE (schéma 38) : la provenance voyage jusqu'au document.
		{TimestampUS: ecOrigin + 1_000_000, Slot: 7, Rank: 11, Previous: 4,
			Kind: filmdec.EquipmentTaken, Recovered: true},
		// Une émission sous GAP résiduel : deux émissions manquent encore juste avant elle,
		// son `from` n'est pas une identité fiable et le document doit le dire.
		{TimestampUS: ecOrigin + 3_000_000, Slot: 7, Rank: filmdec.AbilitySetNoRank,
			Previous: 11, Kind: filmdec.EquipmentSpent, Gap: 2},
	}
	st := filmdec.EquipmentChangeStats{Lives: 1, Recovered: 1, CounterJumps: 1, MissedEstimate: 2}
	got, cov := buildEquipmentChanges(in, st, ecOrigin, ecStep)
	if len(got) != 2 {
		t.Fatalf("publiés = %d, attendu 2", len(got))
	}
	if !got[0].Recovered || got[0].Gap != 0 {
		t.Errorf("émission récupérée : %+v — recovered doit voyager, gap rester nul", got[0])
	}
	if got[1].Recovered || got[1].Gap != 2 {
		t.Errorf("émission sous gap : %+v — gap=2 doit voyager tel quel, y compris après "+
			"récupération partielle", got[1])
	}
	if cov.Recovered != 1 {
		t.Errorf("couverture recovered = %d, attendu 1 : les récupérées se comptent À PART", cov.Recovered)
	}
}

func TestBirthOfLivesRendLePremierEchantillon(t *testing.T) {
	born := birthOfLives([]filmdec.BipedPosition{
		{Slot: 9, TimestampUS: 500},
		{Slot: 9, TimestampUS: 100},
		{Slot: 4, TimestampUS: 300},
	})
	if born == nil {
		t.Fatal("prédicat nil alors que des positions existent")
	}
	if at, ok := born(9); !ok || at != 100 {
		t.Errorf("naissance du slot 9 = %d (%v), attendu 100 : c'est le PREMIER échantillon, "+
			"et l'ordre d'arrivée ne le garantit pas", at, ok)
	}
	if _, ok := born(42); ok {
		t.Error("un slot sans position doit rendre ok=false : sans témoin de naissance, le " +
			"décodeur ne peut pas distinguer une réapparition d'un ramassage, et il doit le savoir")
	}
	if birthOfLives(nil) != nil {
		t.Error("sans position, le prédicat doit être nil — pas un prédicat qui répond toujours non")
	}
}
