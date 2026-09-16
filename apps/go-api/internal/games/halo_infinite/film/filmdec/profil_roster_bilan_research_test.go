package filmdec

// profil_roster_bilan_research_test.go — PHASE 4, QUESTION 3 : LA COMPTABILITE ET LA SONDE.
//
// Complement de `profil_roster_research_test.go` (meme lot, meme correction). Ce fichier porte
// les deux mesures qui EXPLIQUENT le residu de compte au lieu de le laisser en ecart :
//
//	la SONDE CIBLEE — chaque XUID du roster de la base est cherche dans le flux SANS aucune
//	  contrainte d'en-tete, puis les champs des 85 bits qui le precedent sont releves. C'est
//	  elle qui a designe le coupable : le champ de 2 bits de `slot+0x08`, mesure a 1 sur des
//	  enregistrements bien reels que le balayage d'origine exigeait nul.
//	le BILAN — ce que le balayage lit, ce que la trame annonce, ce que la base connait, et les
//	  deux residus (un XUID de la base absent du flux, un enregistrement inconnu de la base).
//
// Gardes CHUNK00_FILMS et CHUNK00_XUID_EQUIPES. Aucun code de production touche.

import (
	"path/filepath"
	"sort"
	"testing"
)

// TestProfilRosterBilan est la COMPTABILITE COMPLETE de la table, apres correction : ce que le
// balayage lit, ce que la trame annonce (entites ti=9), ce que la base connait, et les deux
// residus — un XUID de la base absent du flux, un enregistrement du flux inconnu de la base.
// Sans cette ventilation, un ecart de compte reste ambigu.
func TestProfilRosterBilan(t *testing.T) {
	oracle := equipeOracleXuid(t)
	if len(oracle) == 0 {
		t.Skip("CHUNK00_XUID_EQUIPES absent : bilan saute")
	}
	var totLu, totTi9, totBase, totInter, totHors, totAbsents int
	for _, dir := range chunk00Films(t, "CHUNK00_FILMS") {
		pref := filepath.Base(dir)
		orc, ok := oracle[pref]
		if !ok {
			continue
		}
		_, d := readChunk00(t, dir)
		hs := profilRosterTable(d)
		lus := map[uint64]bool{}
		for _, h := range hs {
			lus[h.xuid] = true
		}
		inter, hors := 0, 0
		for x := range lus {
			if _, ok := orc[x]; ok {
				inter++
			} else {
				hors++
			}
		}
		absents := 0
		for x := range orc {
			if !lus[x] {
				absents++
			}
		}
		ti9 := profilRosterAttendu(dir)
		t.Logf("%-10s : lu %2d | entites ti=%d %2d | base %2d | lus connus de la base %2d | "+
			"lus INCONNUS de la base %d | de la base ABSENTS du flux %d",
			pref, len(hs), equipeTI, ti9, len(orc), inter, hors, absents)
		totLu += len(hs)
		totTi9 += ti9
		totBase += len(orc)
		totInter += inter
		totHors += hors
		totAbsents += absents
	}
	t.Logf("=== BILAN === lu %d | ti=%d %d | base %d | intersection %d | hors base %d | "+
		"absents du flux %d", totLu, equipeTI, totTi9, totBase, totInter, totHors, totAbsents)
}

// profilRosterChamps est le releve des champs d'en-tete aux 85 bits qui precedent un entier
// de 64 bits trouve dans le flux. C'est la sonde qui dit QUEL critere rejette un slot.
type profilRosterChamps struct {
	Bit        int
	B0, B1, B2 int
	U32        uint64
	Deux       uint64
	Token      uint64
}

// profilRosterLireChamps lit les champs d'en-tete devant la position d'un XUID.
func profilRosterLireChamps(d []byte, bit int) profilRosterChamps {
	s := bit - s3rEnteteBits
	return profilRosterChamps{
		Bit: bit,
		B0:  int(s3rBit(d, s, 1)), B1: int(s3rBit(d, s+1, 1)), B2: int(s3rBit(d, s+2, 1)),
		U32: s3rBit(d, s+3, 32), Deux: s3rBit(d, s+35, 2), Token: s3rBit(d, s+37, 48),
	}
}

// TestProfilRosterXuidIntrouvable est la sonde CIBLEE : pour chaque XUID du roster, chercher
// l'entier de 64 bits SANS aucune contrainte d'en-tete, puis relever les champs devant lui.
// Un XUID absent du flux est une chose ; un XUID present mais rejete par un critere en est une
// autre, et seule cette sonde les distingue.
func TestProfilRosterXuidIntrouvable(t *testing.T) {
	oracle := equipeOracleXuid(t)
	if len(oracle) == 0 {
		t.Skip("CHUNK00_XUID_EQUIPES absent : sonde sautee")
	}
	var absents, rejetes, vus int
	for _, dir := range chunk00Films(t, "CHUNK00_FILMS") {
		pref := filepath.Base(dir)
		orc, ok := oracle[pref]
		if !ok {
			continue
		}
		_, d := readChunk00(t, dir)
		fin := (dernierNonNul(d) + 1) * 8
		xs := make([]uint64, 0, len(orc))
		for x := range orc {
			xs = append(xs, x)
		}
		sort.Slice(xs, func(i, j int) bool { return xs[i] < xs[j] })
		a, r, v := profilRosterSonde(t, d, pref, xs, fin)
		absents, rejetes, vus = absents+a, rejetes+r, vus+v
	}
	t.Logf("=== SONDE === %d XUID vus dans le flux et retenus, %d presents mais REJETES par un "+
		"critere, %d ABSENTS du flux", vus, rejetes, absents)
}

// profilRosterSonde cherche chaque XUID et classe le resultat.
func profilRosterSonde(t *testing.T, d []byte, pref string, xs []uint64, fin int) (a, r, v int) {
	t.Helper()
	for _, x := range xs {
		pos := s3rCherche(d, x, s3rCorpsBit, fin)
		if len(pos) == 0 {
			a++
			t.Logf("%-10s xuid %19d : ABSENT du flux (0 occurrence)", pref, x)
			continue
		}
		for _, p := range pos {
			c := profilRosterLireChamps(d, p)
			ok := c.B0 == 1 && c.B1 == 0 && c.B2 == 0 && c.U32 == 0 && c.Deux == 0 && c.Token != 0
			if ok {
				v++
				continue
			}
			r++
			t.Logf("%-10s xuid %19d bit %9d : REJETE — b=%d/%d/%d u32=%d deux=%d token=%012x",
				pref, x, p, c.B0, c.B1, c.B2, c.U32, c.Deux, c.Token)
		}
	}
	return a, r, v
}
