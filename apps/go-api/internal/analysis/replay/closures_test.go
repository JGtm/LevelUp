package replay

// closures_test.go — LES FERMETURES, ET SURTOUT CE QU'ELLES REFUSENT.
//
// L'ESSENTIEL DE CE FICHIER TESTE DES ABSTENTIONS. Une fermeture qui attribue est banale ; ce
// qui la distingue du vote retiré le 2026-07-28, c'est qu'elle se TAIT dès que deux candidats
// subsistent. Un test qui ne vérifierait que les succès laisserait passer un retour au vote sans
// que rien ne casse.

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// at construit un échantillon de position au seul instant qui nous intéresse. Les helpers
// `posAt` (shots_test.go) et `tracksOf` (lives_test.go) sont RÉUTILISÉS tels quels : une
// troisième copie divergerait, et la règle du dépôt l'interdit.
func at(slot uint32, tUS uint64) filmdec.BipedPosition { return posAt(slot, tUS, 0, 0, 0) }

// TestFermetureAAttribueQuandUnSeulCorpsEstPossible : le cas nominal de la fermeture A.
func TestFermetureAAttribueQuandUnSeulCorpsEstPossible(t *testing.T) {
	// Slot 1 appartient au joueur 0 (lu). Slot 2 est une vie anonyme, seule vivante à t=1s.
	tracks := tracksOf(at(1, 500_000), at(2, 1_000_000))
	owner := map[uint32]int{1: 0}
	lives := []lifeSpan{{slot: 1, from: 400_000, to: 600_000, xuid: 111},
		{slot: 2, from: 900_000, to: 1_100_000}}
	var rep closureReport
	closeByAvailableBody(tracks, owner, lives, []FireEventRef{{FilmIndex: 3, TimestampUS: 1_000_000}}, &rep)
	if owner[2] != 3 {
		t.Fatalf("le slot 2 devait revenir au joueur 3, pont = %v", owner)
	}
	if rep.byShot != 1 || rep.contested != 0 || rep.refused != 0 {
		t.Fatalf("compte rendu inattendu : %+v", rep)
	}
}

// TestFermetureASAbstientQuandDeuxCorpsSontPossibles : L'ABSTENTION, ET C'EST LE TEST QUI COMPTE.
func TestFermetureASAbstientQuandDeuxCorpsSontPossibles(t *testing.T) {
	tracks := tracksOf(at(2, 1_000_000), at(3, 1_000_000))
	owner := map[uint32]int{}
	lives := []lifeSpan{{slot: 2, from: 900_000, to: 1_100_000}, {slot: 3, from: 900_000, to: 1_100_000}}
	var rep closureReport
	closeByAvailableBody(tracks, owner, lives, []FireEventRef{{FilmIndex: 3, TimestampUS: 1_000_000}}, &rep)
	if len(owner) != 0 {
		t.Fatalf("deux corps possibles : rien ne devait être attribué, pont = %v", owner)
	}
	if rep.byShot != 0 {
		t.Fatalf("aucune attribution attendue, compte rendu %+v", rep)
	}
}

// TestFermetureARefuseQuandDeuxJoueursRevendiquentLeMemeCorps.
func TestFermetureARefuseQuandDeuxJoueursRevendiquentLeMemeCorps(t *testing.T) {
	tracks := tracksOf(at(2, 1_000_000), at(2, 1_200_000))
	owner := map[uint32]int{}
	lives := []lifeSpan{{slot: 2, from: 900_000, to: 1_300_000}}
	var rep closureReport
	closeByAvailableBody(tracks, owner, lives, []FireEventRef{
		{FilmIndex: 3, TimestampUS: 1_000_000},
		{FilmIndex: 5, TimestampUS: 1_200_000},
	}, &rep)
	if len(owner) != 0 {
		t.Fatalf("corps contesté : rien ne devait être attribué, pont = %v", owner)
	}
	if rep.contested != 1 {
		t.Fatalf("une contestation attendue, compte rendu %+v", rep)
	}
}

// TestLeControleDeRecouvrementRejette : un joueur n'a qu'un corps.
//
// C'EST LE GARDE-FOU QUI PEUT RÉFUTER LA MÉTHODE, et il doit donc mordre au moins une fois dans
// les tests — sinon rien ne prouve qu'il est branché.
func TestLeControleDeRecouvrementRejette(t *testing.T) {
	// Le joueur 3 possède déjà le slot 1, dont la VIE ENCADRE l'instant du tir sans qu'aucun
	// échantillon n'y soit assez proche (500 ms, au-delà de la tolérance de 120 ms). Le tir est
	// donc orphelin — la fermeture A s'active — mais le corps déduit chevauche dans le temps un
	// corps déjà attribué au même joueur : c'est impossible, et le contrôle doit le dire.
	tracks := tracksOf(at(1, 500_000), at(1, 2_000_000), at(2, 1_000_000))
	owner := map[uint32]int{1: 3}
	lives := []lifeSpan{{slot: 1, from: 500_000, to: 2_000_000, xuid: 111},
		{slot: 2, from: 900_000, to: 1_100_000}}
	var rep closureReport
	closeByAvailableBody(tracks, owner, lives, []FireEventRef{{FilmIndex: 3, TimestampUS: 1_000_000}}, &rep)
	if _, ok := owner[2]; ok {
		t.Fatalf("le joueur 3 a déjà un corps à cet instant : l'attribution devait être rejetée, pont = %v", owner)
	}
	if rep.refused != 1 {
		t.Fatalf("un rejet par recouvrement attendu, compte rendu %+v", rep)
	}
}

// TestFermetureBAttribueSurUneSeuleMortDansLaFenetre : le cas nominal de la fermeture B.
func TestFermetureBAttribueSurUneSeuleMortDansLaFenetre(t *testing.T) {
	// Une vie nommée calibre la fenêtre à 8 000 ms ; la vie anonyme commence 8 000 ms après une
	// mort unique, celle du joueur 222.
	tracks := tracksOf(at(1, 9_000_000), at(2, 20_000_000))
	lives := []lifeSpan{
		{slot: 1, from: 9_000_000, to: 10_000_000, xuid: 111},
		{slot: 2, from: 20_000_000, to: 21_000_000},
	}
	deaths := []Death{{XUID: 111, TimeMS: 1_000}, {XUID: 222, TimeMS: 12_000}}
	owner := map[uint32]int{1: 0}
	var rep closureReport
	closeByRespawn(tracks, owner, lives, deaths, 0, map[uint64]int{111: 0, 222: 1}, &rep)
	if owner[2] != 1 {
		t.Fatalf("le slot 2 devait revenir au joueur 1 (xuid 222), pont = %v", owner)
	}
	if rep.byRespawn != 1 {
		t.Fatalf("une fermeture par réapparition attendue, compte rendu %+v", rep)
	}
}

// TestFermetureBSAbstientQuandDeuxMortsSontDansLaFenetre : L'ABSTENTION DE B.
func TestFermetureBSAbstientQuandDeuxMortsSontDansLaFenetre(t *testing.T) {
	tracks := tracksOf(at(1, 9_000_000), at(2, 20_000_000))
	lives := []lifeSpan{
		{slot: 1, from: 9_000_000, to: 10_000_000, xuid: 111},
		{slot: 2, from: 20_000_000, to: 21_000_000},
	}
	// Deux morts distinctes tombent dans la même fenêtre de réapparition.
	deaths := []Death{{XUID: 111, TimeMS: 1_000}, {XUID: 222, TimeMS: 12_000}, {XUID: 333, TimeMS: 12_100}}
	owner := map[uint32]int{1: 0}
	var rep closureReport
	closeByRespawn(tracks, owner, lives, deaths, 0, map[uint64]int{111: 0, 222: 1, 333: 2}, &rep)
	if _, ok := owner[2]; ok {
		t.Fatalf("deux victimes possibles : rien ne devait être attribué, pont = %v", owner)
	}
	if rep.contested != 1 {
		t.Fatalf("une contestation attendue, compte rendu %+v", rep)
	}
}

// TestVerdictDuPontRefuseUneSourceNonComptee : la règle de provenance, après les fermetures.
func TestVerdictDuPontRefuseUneSourceNonComptee(t *testing.T) {
	base := BridgeHealth{Slots: 10, FromReading: 8, ClosedByShot: 1, ClosedByRespawn: 1,
		IndexReadings: 26}
	if got := verdictOfBridge(base); got != VerdictNominal {
		t.Fatalf("lecture + fermetures = entrées : verdict attendu %q, obtenu %q", VerdictNominal, got)
	}
	orphelin := base
	orphelin.ClosedByRespawn = 0 // une entrée n'est plus justifiée par aucune source
	if got := verdictOfBridge(orphelin); got == VerdictNominal {
		t.Fatal("une entrée non justifiée doit rendre le pont NON PUBLIABLE")
	}
}
