package filmdec

// profil_builds_research_test.go — PHASE 4, QUESTION 2 : LES BUILDS ANCIENS.
//
// CE QUE LA PHASE 3 A LAISSE OUVERT. Le decalage du designateur d'equipe (186) et la grammaire
// de la table des slots n'avaient ete essayes que sur `HI_1_12_0` et `HI_1_13_0`. Le cache en
// porte SEPT (`TestD1Builds`), et la phase 2 a mesure que la longueur d'un enregistrement de
// slot se transpose par UNE CONSTANTE par build (-2 880, -4 320, +1 600 bits). Il fallait
// s'attendre a la meme chose sur la trame.
//
// CE QUE CET INSTRUMENT MESURE, film par film, SANS verdict :
//
//	la chaine de build lue dans l'en-tete de `chunk_00` (`s3bBuild`) ;
//	le nombre d'entites ti=9 par image-cle et la longueur d'un record ;
//	la DERIVATION de la position du premier composant (`profilLireEtatComplet`) : en-tete de
//	  108 bits, `n1`, etat par defaut, `n2` — et les valeurs de `n1` / `n2`, qui sont les deux
//	  tailles de tampon de l'archetype et doivent donc etre constantes DANS un build ;
//	les decalages qui passent les controles internes C1+C2+C3 de la phase 3 ;
//	le vecteur brut lu a 186 ;
//	la table des slots : nombre d'enregistrements rendus par le balayage et par le lecteur
//	  canonique, et la constante de transposition `mesure - predit` du build.
//
// LE CONTROLE QUI TRANCHE, ecrit avant la mesure : si la grammaire de la trame etait, comme
// celle de la table des slots, decalee d'une constante par build, le balayage C1+C2+C3 de la
// phase 3 rendrait sur les builds anciens un decalage DIFFERENT de 186. S'il rend 186, la
// grammaire de la trame est STABLE la ou celle de `chunk_00` ne l'est pas — et c'est une donnee
// de profil, pas une impression.
//
// Garde CHUNK00_FILMS. Aucun code de production touche.

import (
	"path/filepath"
	"sort"
	"testing"
)

// profilBuildLigne est la ligne de profil d'UN film.
type profilBuildLigne struct {
	Film, Build  string
	OffsetBuild  int
	Card         int // entites ti=9 (le mode)
	Long         int // longueur d'un record ti=9, en bits
	Paquets      int
	DSBits       int
	N1, N2       uint64
	I0Derive     int
	Retenus      []int
	Brut         equipeVecteur
	Enrs, Chaine int   // enregistrements de la table des slots : balayage / lecteur canonique
	Ecarts       []int // `mesure - predit` distincts sur le film
}

// profilBuildMesure remplit la ligne de profil d'un film. Aucune valeur n'est supposee : tout
// est lu, soit dans `chunk_00`, soit dans la trame.
func profilBuildMesure(t *testing.T, dir string) (profilBuildLigne, bool) {
	t.Helper()
	l := profilBuildLigne{Film: filepath.Base(dir)}
	_, d := readChunk00(t, dir)
	l.Build, l.OffsetBuild = s3bBuild(d)
	profilBuildSlots(d, &l)
	v, err := equipePrepare(dir)
	if err != nil || !v.Longueur {
		t.Logf("%-10s build %-10q : PAS DE TRAME ti=%d exploitable (%v)",
			l.Film, l.Build, equipeTI, err)
		return l, false
	}
	l.Card, l.Long, l.Paquets = v.Card, v.Long, len(v.Paquets)
	e := profilLireEtatComplet(v.Paquets[0].Pay, v.Paquets[0].Recs[0].Bit, equipeTI)
	l.DSBits, l.N1, l.N2, l.I0Derive = e.DSBits, e.N1, e.N2, e.CorpsRelatif
	retenus, _ := equipeDecalagesRetenus(v)
	for dd := range retenus {
		l.Retenus = append(l.Retenus, dd)
	}
	sort.Ints(l.Retenus)
	l.Brut = equipeVecteurA(v.Paquets[0], equipeDecalageMesure)
	return l, true
}

// profilBuildSlots remplit la partie « table des slots » de la ligne : combien
// d'enregistrements le balayage trouve, combien le lecteur canonique en chaine, et quelle
// constante de transposition le film porte.
func profilBuildSlots(d []byte, l *profilBuildLigne) {
	es := s3bEnrs(d)
	l.Enrs = len(es)
	l.Chaine = len(s3sChaine(d))
	vus := map[int]bool{}
	for i := 0; i+1 < len(es); i++ {
		if !s3sImprimable(es[i].gamertag) || !s3sImprimable(es[i+1].gamertag) {
			continue
		}
		vus[(es[i+1].debut-es[i].debut)-s3sPredite(es[i])] = true
	}
	for ec := range vus {
		l.Ecarts = append(l.Ecarts, ec)
	}
	sort.Ints(l.Ecarts)
}

// TestProfilBuildsTrame est le releve par build : une ligne par film, tout mesure.
func TestProfilBuildsTrame(t *testing.T) {
	var lignes []profilBuildLigne
	for _, dir := range chunk00Films(t, "CHUNK00_FILMS") {
		l, ok := profilBuildMesure(t, dir)
		lignes = append(lignes, l)
		if !ok {
			continue
		}
		t.Logf("%-10s build %-10q : %2d entites ti=%d, record %3d bits, %3d paquets | "+
			"etat par defaut %2d bits, n1=%d n2=%d -> i0 DERIVE a %3d | "+
			"C1+C2+C3 retient %v | brut@186 %v",
			l.Film, l.Build, l.Card, equipeTI, l.Long, l.Paquets,
			l.DSBits, l.N1, l.N2, l.I0Derive, l.Retenus, l.Brut)
		t.Logf("%-10s   table des slots : balayage %2d enregistrement(s), lecteur canonique %2d ;"+
			" transposition `mesure - predit` %v",
			l.Film, l.Enrs, l.Chaine, l.Ecarts)
	}
	profilBuildsBilan(t, lignes)
}

// profilBuildsBilan agrege le releve par build : c'est la TABLE DE PROFIL.
func profilBuildsBilan(t *testing.T, lignes []profilBuildLigne) {
	t.Helper()
	type agg struct {
		films                     int
		i0, n1, n2, ds, long, ecs map[int]int
		retient186, aTrame        int
	}
	par := map[string]*agg{}
	var ordre []string
	for _, l := range lignes {
		a := par[l.Build]
		if a == nil {
			a = &agg{i0: map[int]int{}, n1: map[int]int{}, n2: map[int]int{},
				ds: map[int]int{}, long: map[int]int{}, ecs: map[int]int{}}
			par[l.Build] = a
			ordre = append(ordre, l.Build)
		}
		a.films++
		for _, ec := range l.Ecarts {
			a.ecs[ec]++
		}
		if l.Card == 0 {
			continue
		}
		a.aTrame++
		a.i0[l.I0Derive]++
		a.n1[int(l.N1)]++
		a.n2[int(l.N2)]++
		a.ds[l.DSBits]++
		a.long[l.Long]++
		for _, dd := range l.Retenus {
			if dd == equipeDecalageMesure {
				a.retient186++
			}
		}
	}
	sort.Strings(ordre)
	t.Logf("=== TABLE DE PROFIL PAR BUILD ===")
	for _, b := range ordre {
		a := par[b]
		t.Logf("%-10s %d film(s), %d a trame ti=%d | i0 derive %s | n1 %s | n2 %s | "+
			"etat par defaut %s | record %s | C1+C2+C3 retient 186 sur %d/%d | "+
			"transposition du slot %s",
			b, a.films, a.aTrame, equipeTI, equipeHistoTexte(a.i0), equipeHistoTexte(a.n1),
			equipeHistoTexte(a.n2), equipeHistoTexte(a.ds), equipeHistoTexte(a.long),
			a.retient186, a.aTrame, equipeHistoTexte(a.ecs))
	}
}

// TestProfilBuildsOracle confronte, sur les films dont le roster tient dans le lecteur de
// `chunk_00`, le vecteur lu a 186 a l'oracle `match_participants.team_id`. C'est la meme
// confrontation que la phase 3, rejouee sur les builds anciens.
func TestProfilBuildsOracle(t *testing.T) {
	oracle := equipeOracleXuid(t)
	if len(oracle) == 0 {
		t.Skip("CHUNK00_XUID_EQUIPES absent : instrument saute")
	}
	var filmsBons, filmsTotal, slotsBons, slotsTotal, cardsDifferents int
	for _, dir := range chunk00Films(t, "CHUNK00_FILMS") {
		pref := filepath.Base(dir)
		orc, ok := oracle[pref]
		if !ok {
			continue
		}
		v, err := equipePrepare(dir)
		if err != nil || !v.Longueur {
			continue
		}
		_, d := readChunk00(t, dir)
		build, _ := s3bBuild(d)
		rangs, repli := equipeRangsXuid(t, dir, orc)
		pred, rgs := equipeAttendu(rangs, orc)
		lu := equipeVecteurA(v.Paquets[0], equipeDecalageMesure)
		if len(rgs) != v.Card || len(pred) == 0 {
			cardsDifferents++
			t.Logf("%-10s build %-10q : %d rangs apparies contre %d entites ti=%d — "+
				"CARDINAUX DIFFERENTS, compte a part", pref, build, len(rgs), v.Card, equipeTI)
			continue
		}
		bons := equipeCompteAccord(pred, lu)
		filmsTotal++
		slotsTotal += len(pred)
		slotsBons += bons
		if bons == len(pred) {
			filmsBons++
		}
		t.Logf("%-10s build %-10q : accord %d/%d a d=%d (repli=%v) ; predit %v ; lu %v",
			pref, build, bons, len(pred), equipeDecalageMesure, repli, pred, lu)
	}
	t.Logf("=== ORACLE SUR LES BUILDS ANCIENS === %d/%d films en accord TOTAL, %d/%d slots ; "+
		"%d films a cardinaux differents (lecteur de chunk_00 sur gros roster)",
		filmsBons, filmsTotal, slotsBons, slotsTotal, cardsDifferents)
}

// TestProfilBuildsOracleCorrige rejoue la confrontation de la phase 3 en prenant l'ordre des
// rangs dans le lecteur de `chunk_00` CORRIGE (`profilRosterTable`, question 3) au lieu du
// lecteur de la phase 2. C'est ce qui ferme les films de gros roster que la phase 3 avait du
// compter a part, et l'ordre reste celui du flux : les enregistrements sont rendus par
// positions de bit croissantes, rien n'est reordonne.
func TestProfilBuildsOracleCorrige(t *testing.T) {
	oracle := equipeOracleXuid(t)
	if len(oracle) == 0 {
		t.Skip("CHUNK00_XUID_EQUIPES absent : instrument saute")
	}
	var filmsBons, filmsTotal, slotsBons, slotsTotal, ecartes, voisEssais, voisTouches int
	for _, dir := range chunk00Films(t, "CHUNK00_FILMS") {
		pref := filepath.Base(dir)
		orc, ok := oracle[pref]
		if !ok {
			continue
		}
		v, err := equipePrepare(dir)
		if err != nil || !v.Longueur {
			continue
		}
		_, d := readChunk00(t, dir)
		build, _ := s3bBuild(d)
		pred, rgs := equipeAttendu(profilRosterXuids(profilRosterTable(d)), orc)
		if len(rgs) != v.Card || len(pred) == 0 {
			ecartes++
			t.Logf("%-10s build %-10q : %d rangs apparies contre %d entites ti=%d — compte a part",
				pref, build, len(rgs), v.Card, equipeTI)
			continue
		}
		lu := equipeVecteurA(v.Paquets[0], equipeDecalageMesure)
		bons := equipeCompteAccord(pred, lu)
		filmsTotal++
		slotsTotal += len(pred)
		slotsBons += bons
		if bons == len(pred) {
			filmsBons++
		}
		e, h := equipeControleVoisins(v, pred)
		voisEssais += e
		voisTouches += h
		t.Logf("%-10s build %-10q : accord %d/%d a d=%d ; voisins %d/%d ; predit %v ; lu %v",
			pref, build, bons, len(pred), equipeDecalageMesure, h, e, pred, lu)
	}
	t.Logf("=== ORACLE, LECTEUR CORRIGE === %d/%d films en accord TOTAL, %d/%d slots ; "+
		"CONTROLE NEGATIF %d touche(s) sur %d decalages voisins ; %d film(s) a cardinaux "+
		"differents", filmsBons, filmsTotal, slotsBons, slotsTotal, voisTouches, voisEssais,
		ecartes)
}
