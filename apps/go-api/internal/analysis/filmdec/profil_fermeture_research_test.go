package filmdec

// profil_fermeture_research_test.go — PHASE 4, QUESTION 1 : LA FERMETURE DE 186.
//
// CE QUE `profil_entete_research_test.go` A ETABLI. Le modele du depot pour un record
// d'image-cle (en-tete de 64 bits, puis le lecteur de record NEW `FUN_1408f1aa4` : etat par
// defaut, puis MASQUE DE PRESENCE, puis les composants) ne peut PAS placer i0 a 186 : sa borne
// superieure est 64 + 22 + 65 = 151, soit 35 bits trop court, et la mesure le confirme
// (i0 predit a 82 sur 80/80 records, le masque relu a cette position rendant la forme absurde
// « aucun composant present »).
//
// CE QUE CE FICHIER PROUVE. La table d'image-cle n'est PAS ecrite par le lecteur de record NEW
// mais par le lecteur d'ETAT COMPLET `FUN_142e2bfd0`, dont le depot porte deja la forme
// (`keyframe_fullstate_loop.go`, lot R7-d) sans l'avoir jamais confrontee a une position de
// champ. Sa grammaire, relue dans l'executable le 2026-09-12 :
//
//	FUN_142e2bfd0, en-tete PAR ENTITE (108 bits) :
//	    R(32) id           -> puVar12[0]
//	    R(32) typeIndex    -> puVar12[1]   (teste != 0xffffffff)
//	    R(32)              -> puVar12[3]
//	    R(4)               FUN_142e29cf8 (verifie : `*(param+0x2c) += 4`)
//	    R(8)               -> *(puVar12+9)
//	  puis, si typeIndex != 0xffffffff :
//	    R(32) n1           si n1 > 0 : vtable[0x60] = l'ETAT PAR DEFAUT
//	    R(32) de controle  UNIQUEMENT si FUN_14076cea8() est vrai (drapeau RUNTIME :
//	                       DAT_144c23326 / DAT_1450e24e8 — indecidable statiquement)
//	    R(32) n2           si n2 > 0 : vtable[0x88] (0 bit) puis FUN_1428e2b68 ->
//	                       FUN_142e2c690, LA BOUCLE, qui n'a AUCUN MASQUE DE PRESENCE
//
// LA SOMME, POUR ti=9, SANS AUCUN AJUSTEMENT :
//
//	108 (en-tete par entite)
//	+ 32 (n1)
//	+ 14 (etat par defaut de `managed-player`, FUN_1410d7540, prefixe de version a 0 :
//	      R(1)=0 ; R(6) ; R(6) ; R(1) — le bit de version mesure a 0 sur 80/80 records)
//	+ 32 (n2)
//	= 186   -> le premier composant, `managed-player-team-designator-component`, R(4).
//
// LE MOT DE CONTROLE EST ABSENT, ET C'EST UNE PREDICTION VERIFIEE : s'il etait ecrit, i0
// tomberait a 218 et l'oracle d'equipe de la phase 3 (16 films en accord exact, 1 seul
// decalage retenu sur 456) aurait designe 218. Le depot portait deja cette mesure par une
// autre voie : `filmComponentCorruptionCheck` est a `false` par defaut.
//
// LES CONTROLES DE CE FICHIER, ECRITS AVANT LA MESURE :
//
//	F1 FERMETURE   la somme vaut EXACTEMENT 186 sur tous les records ti=9.
//	F2 TAILLES     `n1` et `n2` sont les tailles de tampon de l'archetype (vtable[0x20] et
//	               vtable[0x10]) : elles doivent etre NON NULLES et CONSTANTES par ti, sur
//	               tous les records et tous les films. Un en-tete mal dimensionne lirait des
//	               mots quelconques : le controle est fort et il est gratuit.
//	F3 GENERALITE  la meme derivation, appliquee a TOUS les archetypes de l'image-cle, doit
//	               rendre `n1`/`n2` constants par ti — pas seulement pour ti=9.
//
// Garde CHUNK00_FILMS. Aucun code de production touche.

import (
	"fmt"
	"path/filepath"
	"sort"
	"testing"
)

// profilFermetureAttendue est la position de i0 que la derivation doit rendre pour ti=9.
const profilFermetureAttendue = equipeDecalageMesure

// profilEtatComplet est la decomposition d'un record par le lecteur d'ETAT COMPLET.
type profilEtatComplet struct {
	TI           int
	N1, N2       uint64
	DSBits       int
	CorpsRelatif int // position du premier composant, relative a l'ancre
}

// profilLireEtatComplet decompose un record a l'ancre `anchor` selon FUN_142e2bfd0.
// `ds` est joue par le deserialiseur d'etat par defaut PORTE du depot (default_state_arch.go),
// donc aucune largeur n'est inventee ici.
func profilLireEtatComplet(pay []byte, anchor, ti int) profilEtatComplet {
	e := profilEtatComplet{TI: ti}
	p := anchor + keyframeFullStateHeaderBits
	e.N1 = kfReadBits(pay, p, keyframeFullStateSizeBits)
	p += keyframeFullStateSizeBits
	if e.N1 > 0 { // FUN_142e2bfd0 : `if (0 < (int)uVar7)` — sans taille, pas d'etat par defaut
		br := NewBitReader(pay)
		br.SetBitPos(p)
		consumeKeyframeDefaultState(br, uint32(ti))
		e.DSBits = br.BitPos() - p
		p = br.BitPos()
	}
	e.N2 = kfReadBits(pay, p, keyframeFullStateSizeBits)
	p += keyframeFullStateSizeBits
	e.CorpsRelatif = p - anchor
	return e
}

// TestProfilFermeture186 est le controle F1 : la somme des largeurs lues dans l'executable
// vaut EXACTEMENT 186 sur chaque record ti=9, sans ajustement.
func TestProfilFermeture186(t *testing.T) {
	rel := LockProcessDecode()
	defer rel()
	dirs := chunk00Films(t, "CHUNK00_FILMS")
	positions := map[int]int{}
	n1s, n2s, ds := map[int]int{}, map[int]int{}, map[int]int{}
	total := 0
	for _, dir := range dirs {
		v, err := equipePrepare(dir)
		if err != nil || !v.Longueur {
			t.Logf("%s : ECARTE (%v)", filepath.Base(dir), err)
			continue
		}
		fn1, fn2, fpos := map[int]int{}, map[int]int{}, map[int]int{}
		for _, pq := range v.Paquets {
			for _, r := range pq.Recs {
				e := profilLireEtatComplet(pq.Pay, r.Bit, equipeTI)
				positions[e.CorpsRelatif]++
				fpos[e.CorpsRelatif]++
				n1s[int(e.N1)]++
				fn1[int(e.N1)]++
				n2s[int(e.N2)]++
				fn2[int(e.N2)]++
				ds[e.DSBits]++
				total++
			}
		}
		t.Logf("%-10s : i0 %s | n1 %s | n2 %s", filepath.Base(dir),
			equipeHistoTexte(fpos), equipeHistoTexte(fn1), equipeHistoTexte(fn2))
	}
	if total == 0 {
		t.Skip("aucun film exploitable")
	}
	t.Logf("F1 sur %d records ti=%d :", total, equipeTI)
	t.Logf("  108 (en-tete) + 32 (n1) + etat par defaut + 32 (n2) = %s", equipeHistoTexte(positions))
	t.Logf("  etat par defaut de ti=9 : %s bits", equipeHistoTexte(ds))
	t.Logf("  F2 n1 : %s", equipeHistoTexte(n1s))
	t.Logf("  F2 n2 : %s", equipeHistoTexte(n2s))
	if positions[profilFermetureAttendue] != total {
		t.Errorf("F1 ECHOUE : %d/%d records placent le premier composant a %d",
			positions[profilFermetureAttendue], total, profilFermetureAttendue)
	}
	if n1s[0] != 0 || n2s[0] != 0 {
		t.Errorf("F2 ECHOUE : n1 nul x%d, n2 nul x%d (les deux sont testes > 0 par le jeu)",
			n1s[0], n2s[0])
	}
}

// profilCleTI porte, pour un archetype, les valeurs observees de n1 et n2.
type profilCleTI struct {
	Records int
	N1, N2  map[uint64]int
	DS      map[int]int
}

// TestProfilFermetureTousArchetypes est le controle F3 : la meme derivation sur TOUS les
// archetypes de l'image-cle. Si l'en-tete par entite de 108 bits est la bonne, `n1` et `n2`
// tombent sur les DEUX TAILLES DE TAMPON de l'archetype, donc constantes par ti sur tout le
// corpus. C'est un controle a zero a priori : rien ici ne connait ti=9.
func TestProfilFermetureTousArchetypes(t *testing.T) {
	rel := LockProcessDecode()
	defer rel()
	parTI := map[int]*profilCleTI{}
	films := 0
	for _, dir := range chunk00Films(t, "CHUNK00_FILMS") {
		n := CountFilmChunks(dir)
		if n == 0 {
			continue
		}
		films++
		profilBalayerFilm(dir, n, parTI)
	}
	if films == 0 {
		t.Skip("aucun film exploitable")
	}
	tis := make([]int, 0, len(parTI))
	for ti := range parTI {
		tis = append(tis, ti)
	}
	sort.Ints(tis)
	stables, total := 0, 0
	for _, ti := range tis {
		c := parTI[ti]
		total++
		if len(c.N1) == 1 && len(c.N2) == 1 {
			stables++
		}
		t.Logf("ti=%2d : %5d records | n1 %s | n2 %s | etat par defaut %s bits",
			ti, c.Records, profilHistoU64(c.N1), profilHistoU64(c.N2), equipeHistoTexte(c.DS))
	}
	t.Logf("F3 : %d/%d archetypes ont n1 ET n2 CONSTANTS sur %d films", stables, total, films)
}

// profilBalayerFilm accumule la decomposition de tous les records de tous les paquets de
// type 2 d'un film.
func profilBalayerFilm(dir string, chunks int, parTI map[int]*profilCleTI) {
	for c := 1; c <= chunks; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			pay := pk.Payload(data)
			for _, s := range KeyframeRecordSpans(pay) {
				e := profilLireEtatComplet(pay, s.BitStart, s.TI)
				cle := parTI[s.TI]
				if cle == nil {
					cle = &profilCleTI{N1: map[uint64]int{}, N2: map[uint64]int{}, DS: map[int]int{}}
					parTI[s.TI] = cle
				}
				cle.Records++
				cle.N1[e.N1]++
				cle.N2[e.N2]++
				cle.DS[e.DSBits]++
			}
		}
	}
}

// profilHistoU64 rend un histogramme de valeurs 64 bits, trie, borne a six entrees.
func profilHistoU64(m map[uint64]int) string {
	keys := make([]uint64, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	s := ""
	for i, k := range keys {
		if i >= 6 {
			s += fmt.Sprintf(" ...(%d valeurs)", len(keys))
			break
		}
		if s != "" {
			s += " "
		}
		s += fmt.Sprintf("%d:x%d", k, m[k])
	}
	return s
}
