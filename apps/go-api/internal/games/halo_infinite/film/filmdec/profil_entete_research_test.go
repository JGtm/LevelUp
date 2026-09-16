package filmdec

// profil_entete_research_test.go — PHASE 4, QUESTION 1 : EXPLIQUER 186, PAS LE MESURER.
//
// CE QUE LA PHASE 3 A LAISSE OUVERT. Le champ d'equipe est a 186 bits du debut du record
// d'image-cle (`equipeDecalageMesure`), et ce decalage est MESURE par balayage, pas derive.
// Le dossier porte trois largeurs concurrentes pour l'en-tete par entite (47 du fork
// chasewoodhams, 64 de `keyframeHeaderBits`, 108 de `keyframeFullStateHeaderBits`), et la
// largeur du bloc d'etat par defaut de ti=9 n'avait jamais ete sommee.
//
// CE QUE CE FICHIER FAIT. Il SOMME, terme a terme, les largeurs LUES DANS L'EXECUTABLE le
// 2026-09-12 (Ghidra, lecture seule, aucun renommage) le long de la chaine du lecteur de
// record NEW, et compare la somme a 186 sur les films. Chaque terme porte son adresse :
//
//	FUN_1406cbaa0(param_1=1)  la boucle de records : en mode film, R(1) porte puis R(8)
//	                          version AVANT le lecteur de record (FUN_1406cf008 = R(1),
//	                          verifie : `*(param+0x2c) += 1`).
//	FUN_1408f1aa4             le lecteur de record NEW : R(6) typeIndex, puis
//	                          vtable[0x60] (etat par defaut), vtable[0x88], vtable[0x30],
//	                          puis la boucle de composants.
//	vtable de ti=9 @0x1436fff28 (relue octet a octet) :
//	    +0x30 = 0x14111fd9c   `XORPS XMM0,XMM0 ; MOVUPS [RDX],XMM0 ; MOVUPS [RDX+0x10],XMM0 ;
//	                           MOV dword [RDX],1 ; RET`  -> 0 BIT
//	    +0x60 = 0x1410d7540   l'etat par defaut de `managed-player` :
//	                           FUN_1406cf008 = R(1) ; si 1 -> R(8) ; R(6) ; R(6) ; R(1)
//	                           -> 14 ou 22 BITS
//	    +0x88 = 0x141071a58   `memset(dst,0,0x88)` et des constantes -> 0 BIT
//	FUN_14076cb60             la boucle de composants : commence par FUN_1406d7610.
//	FUN_1406d7610             LE MASQUE : R(1) ; si le bit vaut 1 -> R(64) (donc 65 bits) ;
//	                          sinon R(3) = compte, puis compte x R(6) (donc 4 + 6c bits).
//	                          C'est exactement `consumeMask` du depot.
//	i0 de ti=9                `managed-player-team-designator-component`, lecteur
//	                          FUN_140f581e8 -> FUN_1407ef804 = R(4).
//
// LA SOMME NE PEUT PAS ATTEINDRE 186 : 64 + 22 + 65 = 151 au maximum. Ce fichier le PROUVE sur
// les films au lieu de l'affirmer, en decoupant chaque record ti=9 champ par champ et en
// publiant l'ecart. Il rend donc une DONNEE DE PROFIL — la largeur d'en-tete par entite
// IMPLIQUEE par la mesure (`KeyframeLayout` de l'architecture cible, section 5) — et le nombre
// de bits qui manquent au modele du depot.
//
// Garde CHUNK00_FILMS. Aucun code de production touche.

import (
	"fmt"
	"path/filepath"
	"sort"
	"testing"
)

// profilTI9VersionBits est la largeur du champ de version du prefixe commun des etats par
// defaut : `FUN_1406cf008` (R(1)) puis, si le bit est mis, R(8).
const profilTI9VersionBits = 8

// profilTI9CorpsBits est la partie FIXE de l'etat par defaut de ti=9 apres le prefixe de
// version : R(6) index de joueur, R(6), R(1) (lu dans FUN_1410d7540).
const profilTI9CorpsBits = 6 + 6 + 1

// profilMaskCountBits est la largeur du compte de la forme courte du masque (FUN_1406d7610,
// `*(param+0x2c) += 3`), et profilMaskIndexBits celle d'un index de composant.
const (
	profilMaskCountBits = 3
	profilMaskIndexBits = 6
)

// profilDecoupe est la decomposition d'UN record ti=9, terme a terme.
type profilDecoupe struct {
	Anchor, LenBits int
	Field26         uint32
	VerGate         bool
	DSBits          int
	MaskPlein       bool
	MaskCompte      int
	MaskBits        int
	I0Predit        int // 64 + DSBits + MaskBits, le modele du depot
	I0Reel          int // ce que TraverseEntity place comme StartBit du composant 0
	EnteteImpliquee int // 186 - DSBits - MaskBits : la largeur d'en-tete qu'il faudrait
	Val186, ValI0   int
	Desync          int
}

// profilDecouper decompose un record ti=9 en sommant les largeurs lues dans l'executable.
func profilDecouper(pay []byte, anchor, lenBits int, reg *Registry) profilDecoupe {
	d := profilDecoupe{Anchor: anchor, LenBits: lenBits, Desync: -1}
	d.Field26 = uint32(kfReadBits(pay, anchor+32, 26))
	p := anchor + keyframeHeaderBits
	d.VerGate = kfReadBits(pay, p, 1) == 1
	p++
	if d.VerGate {
		p += profilTI9VersionBits
	}
	p += profilTI9CorpsBits
	d.DSBits = p - (anchor + keyframeHeaderBits)
	d.MaskPlein = kfReadBits(pay, p, 1) == 1
	if d.MaskPlein {
		d.MaskBits = 1 + 64
	} else {
		d.MaskCompte = int(kfReadBits(pay, p+1, profilMaskCountBits))
		d.MaskBits = 1 + profilMaskCountBits + d.MaskCompte*profilMaskIndexBits
	}
	d.I0Predit = keyframeHeaderBits + d.DSBits + d.MaskBits
	d.EnteteImpliquee = equipeDecalageMesure - d.DSBits - d.MaskBits
	d.Val186 = int(kfReadBits(pay, anchor+equipeDecalageMesure, equipeDesignatorBits))
	d.ValI0 = int(kfReadBits(pay, anchor+d.I0Predit, equipeDesignatorBits))
	d.I0Reel = profilI0Reel(pay, anchor, reg, &d)
	return d
}

// profilI0Reel rejoue le lecteur de record NEW de PRODUCTION (`TraverseEntity`) sur le record
// et rend la position, RELATIVE a l'ancre, ou il place le composant 0. -1 s'il n'y arrive pas.
func profilI0Reel(pay []byte, anchor int, reg *Registry, d *profilDecoupe) int {
	if reg == nil {
		return -1
	}
	br := LecteurSur(pay)
	br.SetBitPos(anchor + keyframeRecordTIBit)
	tr := TraverseEntity(br, reg, 0)
	d.Desync = tr.DesyncAt
	for _, c := range tr.Comps {
		if c.Index == 0 {
			return c.StartBit - anchor
		}
	}
	return -1
}

// profilRegistre rend le registre ECS du film, ou nil si chunk_00 est illisible.
func profilRegistre(t *testing.T, dir string) *Registry {
	t.Helper()
	defer func() { _ = recover() }()
	_, data := readChunk00(t, dir)
	return parseRegistry(data)
}

// TestProfilEnteteTI9Derivation somme les largeurs de l'executable sur chaque record ti=9 et
// publie l'ecart a 186. C'est la mesure qui repond a « expliquer 186 ».
func TestProfilEnteteTI9Derivation(t *testing.T) {
	dirs := chunk00Films(t, "CHUNK00_FILMS")
	entetes := map[int]int{}
	predits := map[int]int{}
	reels := map[int]int{}
	total := 0
	for _, dir := range dirs {
		v, err := equipePrepare(dir)
		if err != nil || !v.Longueur {
			t.Logf("%s : ECARTE (%v)", filepath.Base(dir), err)
			continue
		}
		reg := profilRegistre(t, dir)
		pq := v.Paquets[0]
		var lignes []profilDecoupe
		for _, r := range pq.Recs {
			d := profilDecouper(pq.Pay, r.Bit, r.LenBits, reg)
			lignes = append(lignes, d)
			entetes[d.EnteteImpliquee]++
			predits[d.I0Predit]++
			reels[d.I0Reel]++
			total++
		}
		profilJournalFilm(t, filepath.Base(dir), lignes)
	}
	if total == 0 {
		t.Skip("aucun film exploitable")
	}
	t.Logf("SOMME SUR %d records ti=%d :", total, equipeTI)
	t.Logf("  i0 PREDIT par le modele du depot (64 + etat par defaut + masque) : %s",
		equipeHistoTexte(predits))
	t.Logf("  i0 REEL place par TraverseEntity                                 : %s",
		equipeHistoTexte(reels))
	t.Logf("  en-tete IMPLIQUEE par 186 (186 - etat par defaut - masque)       : %s",
		equipeHistoTexte(entetes))
	t.Logf("  candidats du dossier : 47 (fork) / %d (keyframeHeaderBits) / %d (FUN_142e2bfd0)",
		keyframeHeaderBits, keyframeFullStateHeaderBits)
}

// profilJournalFilm publie la decoupe d'un film : une ligne de synthese, puis les records.
func profilJournalFilm(t *testing.T, nom string, lignes []profilDecoupe) {
	t.Helper()
	if len(lignes) == 0 {
		return
	}
	ecarts := map[int]int{}
	for _, d := range lignes {
		ecarts[equipeDecalageMesure-d.I0Predit]++
	}
	t.Logf("%s : %d records ti=%d ; ecart 186 - i0predit : %s",
		nom, len(lignes), equipeTI, equipeHistoTexte(ecarts))
	for i, d := range lignes {
		if i >= profilRecordsJournalises {
			t.Logf("    (... %d records de plus, meme forme)", len(lignes)-i)
			break
		}
		t.Logf("    ancre %6d len %3d field26=%d | ver=%v ds=%2d | masque %s = %2d bits"+
			" | i0predit %3d (val %d) i0reel %3d | val@186 = %d | entete impliquee %d",
			d.Anchor, d.LenBits, d.Field26, d.VerGate, d.DSBits,
			profilMaskTexte(d), d.MaskBits, d.I0Predit, d.ValI0, d.I0Reel, d.Val186,
			d.EnteteImpliquee)
	}
}

// profilRecordsJournalises borne le journal par film : quatre records suffisent a voir la
// forme, le reste est du volume.
const profilRecordsJournalises = 4

// profilMaskTexte decrit la forme du masque lue dans le flux.
func profilMaskTexte(d profilDecoupe) string {
	if d.MaskPlein {
		return "plein(64)"
	}
	return fmt.Sprintf("court(c=%d)", d.MaskCompte)
}

// TestProfilEnteteTI9Bornes publie la BORNE SUPERIEURE de la somme des largeurs lues dans
// l'executable, et la confronte a 186. C'est un controle arithmetique qui ne depend d'aucun
// film : si la borne est sous 186, aucune donnee ne peut fermer la derivation.
func TestProfilEnteteTI9Bornes(t *testing.T) {
	dsMin := 1 + profilTI9CorpsBits
	dsMax := 1 + profilTI9VersionBits + profilTI9CorpsBits
	maskMin := 1 + profilMaskCountBits
	maskMax := 1 + 64
	for _, h := range equipeEntetesCandidates() {
		t.Logf("en-tete %3d : i0 dans [%d, %d] ; 186 %s",
			h, h+dsMin+maskMin, h+dsMax+maskMax,
			profilVerdictBorne(h+dsMin+maskMin, h+dsMax+maskMax))
	}
	t.Logf("etat par defaut de ti=9 (FUN_1410d7540) : %d bits sans version, %d avec",
		dsMin, dsMax)
	t.Logf("masque (FUN_1406d7610) : %d bits (forme courte, c=0) a %d (forme pleine)",
		maskMin, maskMax)
	t.Logf("la forme courte vaut 4 + 6c pour c dans 0..7, soit au plus %d bits",
		1+profilMaskCountBits+7*profilMaskIndexBits)
}

// profilVerdictBorne dit si 186 tient dans l'intervalle atteignable.
func profilVerdictBorne(lo, hi int) string {
	switch {
	case equipeDecalageMesure < lo:
		return "SOUS la borne inferieure"
	case equipeDecalageMesure > hi:
		return fmt.Sprintf("AU-DESSUS de la borne superieure de %d bits", equipeDecalageMesure-hi)
	}
	return "ATTEIGNABLE"
}

// TestProfilEnteteTI9Voisinage cherche, dans le record, la position ou le modele du depot
// place i0 ET celle ou l'oracle a trouve l'equipe, puis mesure la DISTANCE entre les deux sur
// tout le corpus. Un ecart CONSTANT est une donnee de profil ; un ecart qui varie avec la
// forme du masque dit que le modele se trompe de terme.
func TestProfilEnteteTI9Voisinage(t *testing.T) {
	parForme := map[string]map[int]int{}
	for _, dir := range chunk00Films(t, "CHUNK00_FILMS") {
		v, err := equipePrepare(dir)
		if err != nil || !v.Longueur {
			continue
		}
		reg := profilRegistre(t, dir)
		for _, pq := range v.Paquets {
			for _, r := range pq.Recs {
				d := profilDecouper(pq.Pay, r.Bit, r.LenBits, reg)
				k := profilMaskTexte(d) + fmt.Sprintf(" ver=%v", d.VerGate)
				if parForme[k] == nil {
					parForme[k] = map[int]int{}
				}
				parForme[k][equipeDecalageMesure-d.I0Predit]++
			}
		}
	}
	if len(parForme) == 0 {
		t.Skip("aucun film exploitable")
	}
	formes := make([]string, 0, len(parForme))
	for k := range parForme {
		formes = append(formes, k)
	}
	sort.Strings(formes)
	for _, k := range formes {
		t.Logf("forme %-24s : ecart 186 - i0predit %s", k, equipeHistoTexte(parForme[k]))
	}
}
