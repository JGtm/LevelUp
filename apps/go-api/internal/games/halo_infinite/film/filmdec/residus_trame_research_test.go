package filmdec

// residus_trame_research_test.go — PHASE 5b, RESIDUS 3 et 4 DE LA TRAME D'ETAT.
//
//	R3 `HI_1_9_0` SUR LA TRAME. La phase 4 a mesure cinq builds sur sept ; le film unique
//	   `11de8353` n'avait jamais ete passe a la derivation de 186 ni a l'oracle d'equipe.
//	R4 LE DRAPEAU DE CONTROLE PAR BUILD. `FUN_14076cea8` gate un mot de 32 bits ecrit entre
//	   l'etat par defaut et `n2` ; s'il etait actif, le premier composant tomberait a 218 et
//	   non a 186. La phase 4 le disait « mesure faux partout » PAR DEDUCTION (le balayage de la
//	   phase 3 ne retenait que 186). Ce fichier le mesure DIRECTEMENT.
//
// CE QUE L'EXECUTABLE DIT DU DRAPEAU, ET QUI REND LA MESURE DECISIVE. L'ECRIVAIN d'etat complet
// `FUN_142e2d08c` (le symetrique du lecteur `FUN_142e2bfd0`) montre que le mot gate n'est pas un
// champ quelconque : c'est une SENTINELLE CONSTANTE.
//
//	(**(code **)(*plVar14 + 0x58))(...)        <- l'etat par defaut de l'archetype
//	if (DAT_1450e24e8 != '\0') {
//	    ... ecriture de 32 bits de la valeur 0xffddcba ...
//	}
//	... n2 ...
//
// Deux consequences, toutes deux testables sur les films :
//
//	S1 SI LE DRAPEAU ETAIT ACTIF, la valeur `0x0FFDDCBA` serait presente dans la trame, une fois
//	   par record d'entite a etat par defaut. On la CHERCHE a tout decalage de bit dans les
//	   paquets de type 2. Attendu : zero.
//	S2 LE TEMOIN POSITIF DU MEME INSTRUMENT : a la place ou la sentinelle tomberait — les
//	   32 bits qui precedent immediatement le premier composant, soit `i0 - 32` — on doit lire
//	   `n2`, c'est-a-dire 136 sur le build courant et 88 sur les builds anterieurs. Un negatif
//	   ne se publie pas seul (methode, regle 4) : c'est ce temoin qui prouve qu'on a cherche au
//	   bon endroit.
//
// Cote LECTEUR, le drapeau n'est pas la meme globale : `FUN_14076cea8` rend `DAT_144c23326` si
// `FUN_1404f2b4c()` est vrai, sinon `DAT_1450e24e8`. Et `FUN_1404f2b4c` lit un champ de la
// STRUCTURE DE SESSION — `*(uint *)(index * 0x1134f0 + 0xea71c + DAT_145121d28) == 2` — c'est-a-dire
// exactement `jeu+0xEA71C`, l'un des champs que l'ecrivain du corps de `chunk_00`
// (`FUN_1407ec560` @`1407ec828`) serialise, sur 2 bits (`SHL RDX,0x2`). Le pas de la table de
// sessions, `0x1134F0`, est la taille de la structure que `FUN_14095944c` recopie pour le film :
// les deux chemins parlent bien du meme objet.
//
// Gardes CHUNK00_FILMS (+ CHUNK00_XUID_EQUIPES pour l'oracle). Lecture seule, aucun code de
// production touche.

import (
	"path/filepath"
	"testing"
)

const (
	// rtSentinelle : la constante ecrite par `FUN_142e2d08c` quand le drapeau de controle est
	// actif. Lue dans le decompile, pas devinee.
	rtSentinelle = uint64(0x0FFDDCBA)
	// rtI0Actif : la position du premier composant SI le drapeau etait actif — 186 plus les
	// 32 bits de la sentinelle.
	rtI0Actif = equipeDecalageMesure + 32
)

// rtCompteMotif compte les occurrences d'un motif de 32 bits a TOUT decalage de bit dans un
// tampon. C'est la recherche de S1 : elle ne suppose aucune position.
func rtCompteMotif(pay []byte, motif uint64) int {
	n, fin := 0, len(pay)*8-32
	for p := 0; p <= fin; p++ {
		if kfReadBits(pay, p, 32) == motif {
			n++
		}
	}
	return n
}

// TestResidusTrameSentinelle execute S1 et S2 : la sentinelle du drapeau de controle est-elle
// dans les films, et que lit-on a sa place ?
func TestResidusTrameSentinelle(t *testing.T) {
	totSent, totRec, totBons := 0, 0, 0
	for _, dir := range chunk00Films(t, "CHUNK00_FILMS") {
		_, d := readChunk00(t, dir)
		build, _ := s3bBuild(d)
		v, err := equipePrepare(dir)
		if err != nil || !v.Longueur {
			t.Logf("%-10s %-10q : ECARTE (%v)", filepath.Base(dir), build, err)
			continue
		}
		sent, avant, recs := rtMesureFilm(v)
		bons := avant[136] + avant[88]
		totSent += sent
		totRec += recs
		totBons += bons
		t.Logf("%-10s %-10q : S1 %d occurrence(s) de 0x%07X dans %d paquet(s) de type 2 | "+
			"S2 les 32 bits a i0-32 valent %s sur %d record(s) ti=%d (n2 attendu)",
			filepath.Base(dir), build, sent, rtSentinelle, len(v.Paquets),
			equipeHistoTexte(avant), recs, equipeTI)
	}
	t.Logf("=== BILAN S1 === %d occurrence(s) de la sentinelle 0x%07X sur l'ensemble des "+
		"paquets de type 2 examines", totSent, rtSentinelle)
	t.Logf("=== BILAN S2 (temoin positif) === %d/%d record(s) lisent n2 (136 ou 88) a la "+
		"position ou la sentinelle tomberait", totBons, totRec)
	if totSent > 0 {
		t.Errorf("S1 : la sentinelle du drapeau de controle apparait %d fois", totSent)
	}
}

// rtMesureFilm rend le compte de sentinelles du film, la distribution des 32 bits qui precedent
// le premier composant, et le nombre de records examines.
func rtMesureFilm(v equipeVue) (sentinelles int, avant map[int]int, recs int) {
	avant = map[int]int{}
	for _, pq := range v.Paquets {
		sentinelles += rtCompteMotif(pq.Pay, rtSentinelle)
		for _, r := range pq.Recs {
			e := profilLireEtatComplet(pq.Pay, r.Bit, equipeTI)
			avant[int(kfReadBits(pq.Pay, r.Bit+e.CorpsRelatif-32, 32))]++
			recs++
		}
	}
	return sentinelles, avant, recs
}

// TestResidusTrameDeuxFormes execute S3 : sur chaque film, le designateur lu a 186 et celui lu
// a 218 (la position que le drapeau actif imposerait) sont confrontes au meme critere interne —
// domaine 1..9 et partage en deux moities egales. Un decodeur robuste doit savoir departager
// les deux formes ; ce test dit si la question se pose sur ce cache.
func TestResidusTrameDeuxFormes(t *testing.T) {
	bons186, bons218, vus := 0, 0, 0
	for _, dir := range chunk00Films(t, "CHUNK00_FILMS") {
		v, err := equipePrepare(dir)
		if err != nil || !v.Longueur {
			continue
		}
		_, d := readChunk00(t, dir)
		build, _ := s3bBuild(d)
		a := equipeVecteurA(v.Paquets[0], equipeDecalageMesure)
		b := equipeVecteurA(v.Paquets[0], rtI0Actif)
		okA := equipeDomaineValide(a) && equipeFormeEquilibree(a)
		okB := equipeDomaineValide(b) && equipeFormeEquilibree(b)
		vus++
		if okA {
			bons186++
		}
		if okB {
			bons218++
		}
		t.Logf("%-10s %-10q : d=186 domaine+partage %-5v %v | d=218 domaine+partage %-5v %v",
			filepath.Base(dir), build, okA, a, okB, b)
	}
	t.Logf("=== BILAN S3 === %d/%d film(s) passent le critere interne a d=186, %d/%d a d=218",
		bons186, vus, bons218, vus)
}

// rtAlignements rend, pour un film dont la table des slots porte `n` rangs et dont la trame
// porte `card` entites ti=9, le nombre de recalages possibles du premier sur le second.
//
// POURQUOI CE TEST EXISTE. L'oracle terme a terme de la phase 3 exige `n == card` et ECARTE les
// films ou les deux comptes different — or c'est exactement le cas de `11de8353` (`HI_1_9_0`),
// le seul film de son build. La table de `chunk_00` est le roster A L'ECRITURE du chunk ; la
// trame compte les entites PRESENTES a l'image-cle, et un joueur arrive en cours de partie
// n'est que dans la seconde. Plutot que d'ecarter le film, on essaie TOUS les recalages et on
// publie le compte de ceux qui rendent l'accord EXACT : si un seul passe sur `card-n+1`
// essais, la lecture n'est pas fortuite, et le chiffre est le plancher de bruit MESURE.
// `pred` est indexe PAR RANG et porte -1 aux rangs dont le XUID n'est pas dans l'oracle : un
// rang manquant ne doit pas DECALER la suite (c'est ce que fait `equipeAttendu`, qui compacte),
// il doit juste ne pas etre compare.
func rtAlignements(pred, lu equipeVecteur) (bons []int, essais, compares int) {
	if len(pred) == 0 || len(lu) < len(pred) {
		return nil, 0, 0
	}
	for k := 0; k+len(pred) <= len(lu); k++ {
		essais++
		bonsK, vus := 0, 0
		for i, p := range pred {
			if p < 0 {
				continue
			}
			vus++
			if p == lu[k+i] {
				bonsK++
			}
		}
		if k == 0 {
			compares = vus
		}
		if vus > 0 && bonsK == vus {
			bons = append(bons, k)
		}
	}
	return bons, essais, compares
}

// rtPredParRang construit le vecteur attendu INDEXE PAR RANG : `team_id + 1` quand le XUID du
// rang est dans l'oracle, -1 sinon.
func rtPredParRang(xuids []uint64, orc map[uint64]int) (pred equipeVecteur, apparies int) {
	pred = make(equipeVecteur, 0, len(xuids))
	for _, x := range xuids {
		if eq, ok := orc[x]; ok {
			pred = append(pred, eq+1)
			apparies++
			continue
		}
		pred = append(pred, -1)
	}
	return pred, apparies
}

// TestResidusTrameOracleRecale execute R3-ORACLE : l'oracle externe sur les films que la
// confrontation terme a terme ecarte pour cardinaux differents, en essayant tous les recalages.
// L'ordre des rangs vient du lecteur CORRIGE de R2 (`rsChaine`), qui enjambe les slots vacants.
func TestResidusTrameOracleRecale(t *testing.T) {
	oracle := equipeOracleXuid(t)
	if len(oracle) == 0 {
		t.Skip("CHUNK00_XUID_EQUIPES absent : instrument saute")
	}
	uniques, filmsTot, essaisTot, bonsTot, slots := 0, 0, 0, 0, 0
	for _, dir := range chunk00Films(t, "CHUNK00_FILMS") {
		orc, ok := oracle[filepath.Base(dir)]
		if !ok {
			continue
		}
		v, err := equipePrepare(dir)
		if err != nil || !v.Longueur {
			continue
		}
		_, d := readChunk00(t, dir)
		build, _ := s3bBuild(d)
		delta, _, _ := rsDelta(d)
		enrs, vac := rsChaine(d, delta)
		pred, app := rtPredParRang(rtXuids(enrs), orc)
		lu := equipeVecteurA(v.Paquets[0], equipeDecalageMesure)
		bons, essais, compares := rtAlignements(pred, lu)
		filmsTot++
		essaisTot += essais
		bonsTot += len(bons)
		if len(bons) == 1 {
			uniques++
			slots += compares
		}
		t.Logf("%-10s %-10q : %d rang(s) lus (+%d vacants), %d apparies a l'oracle, %d "+
			"entites ti=%d | recalages en accord EXACT %v sur %d essais (%d rang(s) "+
			"compares) | predit %v | lu %v", filepath.Base(dir), build, len(enrs), vac, app,
			v.Card, equipeTI, bons, essais, compares, pred, lu)
	}
	t.Logf("=== BILAN R3-ORACLE === %d/%d film(s) ou UN SEUL recalage rend l'accord exact "+
		"(%d slots en accord) ; %d recalage(s) en accord sur %d essais au total (plancher de "+
		"bruit MESURE)", uniques, filmsTot, slots, bonsTot, essaisTot)
}

// rtXuids rend les XUID d'une suite d'enregistrements de slot, dans l'ordre du flux.
func rtXuids(es []*s3sEnr) []uint64 {
	out := make([]uint64, 0, len(es))
	for _, e := range es {
		out = append(out, e.xuid)
	}
	return out
}

// TestResidusTrameCardinalite execute D-CARD : la CARDINALITE REELLE des designateurs d'equipe,
// mesuree DANS LE FILM, par film et tous films confondus.
//
// La table `mp_team_designator` compte 9 entrees (`First`..`Eighth`, `Neutral`) et le lecteur de
// la composante globale accepte le domaine -1..8. Savoir combien de valeurs distinctes un film
// porte REELLEMENT dit si ce domaine est exerce — et c'est une mesure, pas une lecture de table.
func TestResidusTrameCardinalite(t *testing.T) {
	global := map[int]int{}
	cards := map[int]int{}
	for _, dir := range chunk00Films(t, "CHUNK00_FILMS") {
		v, err := equipePrepare(dir)
		if err != nil || !v.Longueur {
			continue
		}
		_, d := readChunk00(t, dir)
		build, _ := s3bBuild(d)
		vals := map[int]int{}
		for _, pq := range v.Paquets {
			for _, x := range equipeVecteurA(pq, equipeDecalageMesure) {
				vals[x]++
				global[x]++
			}
		}
		cards[len(vals)]++
		t.Logf("%-10s %-10q : %d entite(s) ti=%d, valeurs brutes distinctes %s",
			filepath.Base(dir), build, v.Card, equipeTI, equipeHistoTexte(vals))
	}
	t.Logf("=== BILAN D-CARD === valeurs brutes vues, tous films confondus : %s",
		equipeHistoTexte(global))
	t.Logf("=== BILAN D-CARD === nombre de valeurs distinctes PAR FILM : %s",
		equipeHistoTexte(cards))
}
