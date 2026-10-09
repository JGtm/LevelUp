package signaux

// statborg_domaine_branche_test.go — LE BRANCHEMENT DU FILTRE DE DOMAINE DANS LE BALAYAGE
// (revue 6.R, constat C3, 2026-09-11).
//
// CE QUE CE FICHIER FERME. `statborg_domaine_test.go` couvre le PREDICAT
// ([statCountersInDomain]) sous toutes ses faces, et il l'ecrivait lui-meme : retirer
// `!statCountersInDomain(comps)` de [scanFrameForRecords] ne rougissait AUCUN test. Le predicat
// etait donc prouve, et son BRANCHEMENT ne l'etait pas — un debranchement accidentel serait passe
// vert jusqu'a la prochaine cuisson de parc, c'est-a-dire jusqu'a ce que les dix prises fausses
// de `fb1a1a72` reviennent.
//
// LE VECTEUR EST SYNTHETIQUE, ET C'EST NECESSAIRE : le cas reel vit a une position precise d'un
// paquet de film que le depot ne versionne pas. Il est construit a la grammaire de la production
// — en-tete d'enregistrement, liste creuse d'un composant, deux canaux a longueur variable — et
// le TEMOIN POSITIF le prouve : le MEME vecteur, avec un canal A dans le domaine, ressort bien du
// balayage. Seule la VALEUR du canal separe les deux cas.
//
// MUTATION JOUEE : retirer `!statCountersInDomain(comps)` de [scanFrameForRecords] rougit
// `TestLeBalayageJETTELEnregistrementHorsDomaine`, et lui seul.

import (
	"levelup/go-api/internal/games/halo_infinite/film/types"
	"testing"
)

// statBrancheComp est le composant que le vecteur porte — celui du cas reel de `fb1a1a72`.
const statBrancheComp = 22

// statVecteurUnCompo fabrique un paquet FRAME portant UN enregistrement d'entite ancre au bit 1 :
// slot 24, manche 0, liste creuse d'un seul composant [statBrancheComp] dont le canal A vaut `a`
// (ecrit sur 32 bits) et le canal B vaut 8 (sur 8 bits).
func statVecteurUnCompo(a uint64) []byte {
	pay := make([]byte, 48)                 // large devant [statTailBits] : l'ancrage au bit 1 reste balaye.
	pay = setBitsBE(pay, 0, 1, 1)           // le bit marqueur qui precede l'en-tete
	pay = setBitsBE(pay, 1, statIDBits, 24) // slot d'entite
	pay = setBitsBE(pay, 1+statIDBits, statGenBits, statGenValue)
	m := 1 + statIDBits + statGenBits
	pay = setBitsBE(pay, m, 1, 0)   // forme CREUSE
	pay = setBitsBE(pay, m+1, 3, 1) // un seul composant
	pay = setBitsBE(pay, m+4, statCompIndexBits, statBrancheComp)
	at := m + 4 + statCompIndexBits
	pay = setBitsBE(pay, at, statHdrBits, 0) // les deux en-tetes de manche, egaux
	pay = setBitsBE(pay, at+statHdrBits, statHdrBits, 0)
	q := at + 2*statHdrBits
	pay = setBitsBE(pay, q, 2, 2) // canal A : selecteur 2 -> 32 bits
	pay = setBitsBE(pay, q+2, 32, a)
	q += 34
	pay = setBitsBE(pay, q, 2, 0) // canal B : selecteur 0 -> 8 bits
	pay = setBitsBE(pay, q+2, 8, 8)
	q += 10
	pay = setBitsBE(pay, q, 2, 0) // les deux drapeaux conditionnels : aucun canal C ni D
	return pay
}

// statPorteComp dit si l'un des enregistrements rendus porte le composant du vecteur avec la
// valeur `a` — c'est-a-dire si le balayage a laisse passer CE vecteur-la et pas un autre ancrage.
func statPorteComp(recs []types.StatRecord, a int64) bool {
	for _, r := range recs {
		if v, ok := r.Comps[statBrancheComp]; ok && v.A == a {
			return true
		}
	}
	return false
}

// TestLeBalayageGARDELEnregistrementDansLeDomaine — LE TEMOIN POSITIF. Sans lui, le test suivant
// serait satisfait par un vecteur que le balayage refuse pour une tout autre raison (ancrage
// rate, liste mal formee) et ne prouverait rien du filtre.
func TestLeBalayageGARDELEnregistrementDansLeDomaine(t *testing.T) {
	recs := scanFrameForRecords(statVecteurUnCompo(10), 764967)
	if !statPorteComp(recs, 10) {
		t.Fatalf("le vecteur de controle (comp %d A = 10) ne ressort pas du balayage : "+
			"%d enregistrement(s) rendus — revoir le vecteur avant de conclure sur le filtre",
			statBrancheComp, len(recs))
	}
}

// TestLeBalayageJETTELEnregistrementHorsDomaine — LE POINT DU CONSTAT. Le MEME vecteur, dont le
// seul canal A sort du domaine : l'enregistrement ENTIER doit etre absent de la sortie. C'est la
// regle de `fb1a1a72` — un ancrage fortuit ne produit pas une valeur fausse, il produit un
// enregistrement qui n'existe pas.
func TestLeBalayageJETTELEnregistrementHorsDomaine(t *testing.T) {
	const hors = 2_000_000 // au-dela de [statMaxCounter], et loin sous le bord des 32 bits signes
	if hors <= statMaxCounter {
		t.Fatalf("le vecteur n'est plus hors domaine : %d <= %d", hors, statMaxCounter)
	}
	recs := scanFrameForRecords(statVecteurUnCompo(hors), 764967)
	if statPorteComp(recs, hors) {
		t.Errorf("le balayage rend un enregistrement dont le canal A vaut %d : le filtre de "+
			"domaine n'est plus branche sur scanFrameForRecords", hors)
	}
}

// TestLeBalayageDuStatborgCompteSesAbandons : un enregistrement dont un compteur sort du domaine
// est abandonne ET compte ; un composant non decodable arrete la boucle et le dit (lot J8.7). Les
// deux comptes voyagent dans [LectureDuStatborg] ; la couche des faits les porte jusqu au
// versement (`objectives.ComptesDesReplis`).
func TestLeBalayageDuStatborgCompteSesAbandons(t *testing.T) {
	var c LectureDuStatborg
	if recs := scanFrameAvecReplis(statVecteurUnCompo(statMaxCounter+1), 764967, &c); len(recs) != 0 {
		t.Fatalf("enregistrement hors domaine publie : %v", recs)
	}
	if c.EnregistrementsAbandonnes == 0 {
		t.Error("l enregistrement hors domaine est abandonne sans etre compte")
	}
	var sain LectureDuStatborg
	if recs := scanFrameAvecReplis(statVecteurUnCompo(10), 764967, &sain); len(recs) == 0 {
		t.Fatal("l enregistrement sain n est plus lu")
	}
	pay := statVecteurUnCompo(10)
	_, idx, at, ok := matchRecordHeader(pay, 1)
	if !ok {
		t.Fatal("le vecteur sain ne porte plus d en-tete a son bit d ancrage")
	}
	if comps, _, arrete := decodeComponentsAvecArret(pay, at, idx); len(comps) == 0 || arrete {
		t.Errorf("vecteur sain : %d composant(s), arret %v — attendu une lecture complete", len(comps), arrete)
	}
	if _, _, arrete := decodeComponentsAvecArret(make([]byte, 2), 0, []int{0, 1}); !arrete { // tampon trop court : le premier composant deborde
		t.Error("un composant non decodable doit arreter la boucle ET le dire")
	}
}
