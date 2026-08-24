package main

// backfill_memlimit_test.go — le plafond memoire.
//
// NOTE : ces tests n'ARMENT jamais un vrai plafond. `armerPlafondMemoire` appelle
// `debug.SetMemoryLimit` (etat global du processus) et lance une sentinelle qui fait
// `os.Exit` — l'armer pour de bon ici donnerait a un test le pouvoir de tuer le binaire de
// test. On teste donc les pieces, et le cas DESARME.

import "testing"

func TestMargeDure(t *testing.T) {
	// Le plafond dur se pose 25 % au-dessus du souple : sous cette marge, le GC a le droit
	// de travailler dur sans etre abattu — c'est son role.
	if got := margeDure(4 * octetsParGiB); got != 5*octetsParGiB {
		t.Fatalf("margeDure(4 GiB) = %d, veut 5 GiB", got)
	}
	if got := margeDure(0); got != 0 {
		t.Fatalf("margeDure(0) = %d, veut 0 (desarme reste desarme)", got)
	}
}

// TestEmpreinteMemoire_Mesure : la mesure doit rendre un chiffre plausible, sinon la
// sentinelle surveillerait le vide et ne couperait jamais.
func TestEmpreinteMemoire_Mesure(t *testing.T) {
	v := empreinteMemoire()
	if v == 0 {
		t.Fatal("empreinteMemoire() = 0 — les compteurs runtime ne repondent pas")
	}
	// Un processus de test Go tient au moins quelques centaines de Kio.
	if v < 128*1024 {
		t.Fatalf("empreinteMemoire() = %d octets — invraisemblablement bas", v)
	}
}

func TestSentinelleMemoire_NotePic(t *testing.T) {
	var s sentinelleMemoire
	for _, v := range []uint64{10, 500, 42, 500, 3} {
		s.noterPic(v)
	}
	if got := s.pic.Load(); got != 500 {
		t.Fatalf("pic = %d, veut 500 (le maximum, jamais la derniere valeur)", got)
	}
}

// TestSentinelleMemoire_PicObserve : un enfant qui meurt avant le premier echantillon doit
// tout de meme rendre un pic — celui de l'instant.
func TestSentinelleMemoire_PicObserve(t *testing.T) {
	var s sentinelleMemoire
	if got := s.picObserve(); got == 0 {
		t.Fatal("picObserve() = 0 alors qu aucun echantillon n a encore eu lieu")
	}
}

// TestArmerPlafondMemoire_Desarme : `--mem-limit-gib 0` est l'echappatoire de l'operateur.
// Elle doit desarmer la COUPURE, sans desarmer la MESURE.
func TestArmerPlafondMemoire_Desarme(t *testing.T) {
	s := armerPlafondMemoire(0)
	if s.plafondDur != 0 {
		t.Fatalf("plafondDur = %d, veut 0 (aucune coupure quand le plafond est desarme)", s.plafondDur)
	}
	if got := s.picObserve(); got == 0 {
		t.Fatal("le pic doit rester mesure meme plafond desarme")
	}
}

func TestLibellePlafond(t *testing.T) {
	if got := libellePlafond(0); got != "DESARME" {
		t.Fatalf("libellePlafond(0) = %q", got)
	}
	if got := libellePlafond(3); got != "3 GiB" {
		t.Fatalf("libellePlafond(3) = %q", got)
	}
}

func TestLibelleOctets(t *testing.T) {
	if got := libelleOctets(0); got != "inconnu" {
		t.Fatalf("libelleOctets(0) = %q — 0 veut dire NON MESURE, pas zero octet", got)
	}
	if got := libelleOctets(2 * octetsParGiB); got != "2.00 GiB" {
		t.Fatalf("libelleOctets(2 GiB) = %q", got)
	}
	if got := libelleOctets(512 * 1024 * 1024); got != "512 MiB" {
		t.Fatalf("libelleOctets(512 MiB) = %q", got)
	}
}
