package objectiveevents

// statborg_domaine_test.go — LE DOMAINE DES COMPTEURS, AU NIVEAU DE L'ENREGISTREMENT
// (lot 6.11, item 3).
//
// CE QUE CE FILTRE FERME. Sur `fb1a1a72`, un ancrage fortuit du slot 24 (t = 764 967) portait
// DOUZE composants, dont `comp 22 A = 10` — publie en DIX prises de drapeau pour ZERO a
// l'oracle, alors que les sept autres slots du film sont exacts a l'unite. Le meme
// enregistrement portait `comp 7 B = 2 415 919 104` et `comp 46 A = -30 456` : des valeurs
// qu'aucun compteur de match ne prend. Le pas de 10, lui, passait SOUS la borne de deroulage
// recalibree au lot 6.7-B1 (16) — la borne PAR PAS ne pouvait pas le voir.
//
// C'EST LA DECOUVERTE D5 DE B1, ENFIN POSEE : « le filtre juste serait au niveau de
// l'ENREGISTREMENT, pas du pas ». Un ancrage fortuit ne produit pas une valeur fausse, il
// produit un enregistrement qui n'existe pas ; ses autres composants sont du bruit au meme
// titre, et c'est l'enregistrement ENTIER qu'il faut refuser.
//
// LE SEUIL EST MESURE, PAS CHOISI (11 films du parc, tous modes — CTF, Oddball, Strongholds,
// KOTH, Slayer, BTB) : la plus grande valeur d'un enregistrement dont aucun canal n'atteint
// 2^20 vaut 102 934, et 32 518 sur dix films sur onze ; la population aberrante commence a
// 2 415 919 104. Entre les deux, RIEN.

import "testing"

// TestDomaineAccepteUnEnregistrementREEL — LE TEMOIN POSITIF, et il est indispensable : un
// filtre qui refuserait tout « n'aurait plus de faux positifs » sans rien prouver. Le vecteur
// est un enregistrement reel du corpus, decode par la production.
func TestDomaineAccepteUnEnregistrementREEL(t *testing.T) {
	for nom, v := range map[string]statVector{"creux": vecRound0, "dense": vecDense} {
		if !statCountersInDomain(v.comps) {
			t.Errorf("%s : l'enregistrement REEL %v est refuse par le domaine", nom, v.comps)
		}
		_, idx, at, ok := matchRecordHeader(v.data, v.bits)
		if !ok {
			t.Fatalf("%s : le vecteur ne s'ancre plus — revoir les vecteurs avant ce test", nom)
		}
		comps, _ := decodeComponents(v.data, at, idx)
		if len(comps) == 0 || !statCountersInDomain(comps) {
			t.Errorf("%s : %d composant(s) decode(s), domaine %v — le chemin de production doit "+
				"garder cet enregistrement", nom, len(comps), statCountersInDomain(comps))
		}
	}
}

// TestDomaineRefuseLEnregistrementDeFb1a1a72 — LE CAS REEL, reproduit a l'identique : les douze
// composants du slot 24 a t = 764 967, avec leurs valeurs mesurees. `comp 22 A = 10` y est
// PARFAITEMENT plausible pris isolement — c'est le reste de l'enregistrement qui le condamne.
//
// MUTATION : ce test porte sur le PREDICAT, et lui seul. Le BRANCHEMENT du predicat dans
// [scanFrameForRecords] a ses propres temoins depuis la revue 6.R — `statborg_domaine_branche_test.go`,
// ou retirer `!statCountersInDomain(comps)` du balayage rougit.
func TestDomaineRefuseLEnregistrementDeFb1a1a72(t *testing.T) {
	rec := map[int]StatValue{
		7:  {A: 1, B: 2415919104},
		14: {A: 0, B: 0},
		19: {A: 0, B: 0},
		22: {A: 10, B: 8},
		23: {A: 0, B: 105},
		28: {A: 0, B: 4},
		32: {A: 3, B: 8192},
		36: {A: 56, B: 14},
		38: {A: 0, B: 0},
		39: {A: 0, B: 0},
		43: {A: 8, B: 79},
		46: {A: -30456, B: 0},
	}
	if statCountersInDomain(rec) {
		t.Error("l'enregistrement fortuit de `fb1a1a72` (comp 7 B = 2 415 919 104) est accepte : " +
			"ses dix prises de drapeau fausses reviendraient")
	}
	// Le MEME enregistrement prive de son seul canal hors domaine redevient recevable : le
	// filtre ne juge que le domaine, jamais la forme ni le nombre de composants.
	rec[7] = StatValue{A: 1, B: 2}
	if !statCountersInDomain(rec) {
		t.Error("le filtre refuse un enregistrement dont tous les canaux tiennent dans le " +
			"domaine : il juge autre chose que le domaine")
	}
}

// TestDomaineEstUneBorneEtPasUnSigne — les deux bords, et le signe. Un canal EXACTEMENT a la
// borne passe (c'est une borne, pas un plafond exclusif du domaine sain mesure a 102 934) ; un
// canal au-dela ne passe pas, positif comme NEGATIF — `comp 46 A = -30 456` du cas reel montre
// que l'aberration se lit aussi en dessous de zero.
func TestDomaineEstUneBorneEtPasUnSigne(t *testing.T) {
	for _, cas := range []struct {
		nom    string
		v      StatValue
		accept bool
	}{
		{"pire valeur saine mesuree", StatValue{A: 102934}, true},
		{"exactement a la borne", StatValue{A: statMaxCounter}, true},
		{"un cran au-dela", StatValue{A: statMaxCounter + 1}, false},
		{"negatif au-dela", StatValue{B: -(statMaxCounter + 1)}, false},
		{"negatif plausible", StatValue{B: -333}, true},
	} {
		if got := statCountersInDomain(map[int]StatValue{22: cas.v}); got != cas.accept {
			t.Errorf("%s : %+v accepte=%v, attendu %v", cas.nom, cas.v, got, cas.accept)
		}
	}
}
