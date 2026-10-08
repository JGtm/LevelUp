package grammar

import (
	"strings"
	"testing"
)

// components_device_ti43_test.go — les lecteurs `device-*` de `ti=43` confrontes a des vecteurs
// ECRITS COMME L ECRIVAIN LES ECRIT (`.ai/V7.5/film_re/campagne_grammaire_2026-10-01/T7_dispositifs_ti43_moteur.md`
// §6.1) : bits dans l ordre d ecriture, chaque champ poids fort en tete. Un vecteur est tenu quand
// le lecteur de production, appele par la chaine de dispatch, consomme EXACTEMENT ses bits — ni
// moins, ni plus : une queue de uns suit chaque vecteur, qu un lecteur qui lirait trop
// consommerait. Les largeurs de reference d entite sont celles du profil par defaut (13 bits ; 9
// apres une sonde a 1, categorie 1).

// vecteurDispositif : un etat ecrit par l ecrivain, ses bits, et ce que le lecteur doit en faire.
type vecteurDispositif struct {
	id, composant string
	bits          string // '0' et '1' ; tout autre caractere est un separateur de lecture
	porte         bool   // faux : le lecteur du jeu echoue, la traversee s arrete au composant
}

// vecteursDispositifs : T7 §6.1. Les DISCRIMINANTS (V35b, V35c, V36, V31b, V31c, V37) font echouer
// une erreur de port plausible : ignorer la condition du poids, traiter le code 1 comme un poids
// nul, recopier le port d `i24` dans `i36`, borner N a 8, refuser N = 8 (le jeu l admet :
// `if (N < 9)`), lire l octet apres la troisieme valeur.
var vecteursDispositifs = []vecteurDispositif{
	{"V35a", compDeviceAnimationLayerState, "0", true},
	{"V35b", compDeviceAnimationLayerState, "1 | 1 0000000000 | 1 1111111111 11111111111111 | 0 0 0 0 0 0", true},
	{"V35c", compDeviceAnimationLayerState, "1 | 1 0000000001 00000000000000 | 0000000", true},
	{"V34a", compDeviceAnimationLayerSettings, "0", true},
	{"V34b", compDeviceAnimationLayerSettings, "1 | 1 00000000000000000000000000000001 11111111111111 00000000000000 1111111111 11111111111111 01 | 0000000", true},
	{"V21a", compDevicePositionGroup, "00000000000000000000000000000111 1 00000011", true},
	{"V21b", compDevicePositionGroup, "00000000000000000000000000000111 0", true},
	{"V19", compDevicePositionAnimationName, "11011110101011011011111011101111 1111111111", true},
	{"V22", compDevicePower, "11111111111111", true},
	{"V23", compDevicePowerGroup, "00000000000000", true},
	{"V24a", compDeviceInteractionInProgress, "0", true},
	{"V24b", compDeviceInteractionInProgress, "1 0 0000000000101 10", true},
	{"V24c", compDeviceInteractionInProgress, "1 1 000000101 10", true},
	{"V29a", compDeviceExclusiveUser, "0", true},
	{"V31a", compDeviceDispenserMonitors, "00000010 | 0 | 1 0 0000000101010 01", true},
	{"V31b", compDeviceDispenserMonitors, "00001001", false},
	{"V31c", compDeviceDispenserMonitors, "00001000 | 0 | 1 0 0000000101010 01 | 0 | 1 1 000000101 10 | 0 | 0 | 1 0 0000000000001 00 | 0", true},
	{"V36", compDeviceDispenserState, "0 101 | 1 1 0000001000010 01 010", true},
	{"V37", compDeviceObjectDispenserTimer, "0 | 1 1111111111 0000000000 00011 1111111111", true},
	{"V39", compDeviceMachineFlags, "110100101", true},
}

// tamponDeVecteur ecrit les bits du vecteur puis une queue de 64 uns ; rend le tampon et le
// nombre de bits du vecteur.
func tamponDeVecteur(bits string) ([]byte, int) {
	var w bitWriter
	for _, c := range bits {
		switch c {
		case '0':
			w.bit(0)
		case '1':
			w.bit(1)
		}
	}
	n := w.n
	w.bits(^uint64(0), 64)
	return w.buf, n
}

// TestLesDispositifsLisentCeQueLEcrivainEcrit : chaque vecteur est consomme au bit pres.
func TestLesDispositifsLisentCeQueLEcrivainEcrit(t *testing.T) {
	for _, v := range vecteursDispositifs {
		buf, n := tamponDeVecteur(v.bits)
		br := LecteurSur(buf)
		_, _, porte := consumeByName(br, v.composant, 43, 1)
		if porte != v.porte {
			t.Errorf("%s (%s) : porte %v, attendu %v", v.id, v.composant, porte, v.porte)
			continue
		}
		if got := br.BitPos(); got != n {
			t.Errorf("%s (%s) : %d bits consommes, l ecrivain en a ecrit %d (%s)", v.id, v.composant,
				got, n, strings.TrimSpace(v.bits))
		}
	}
}
