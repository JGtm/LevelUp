package main

// variation_test.go — la lecture du paquet RANGED et l'agregation par couche dominante.
//
// POURQUOI CES TESTS. Le paquet RANGED etait deja localise et valide, mais son contenu
// etait jete : `lirePaquetProps` ne rendait que la PREMIERE composante de chaque propriete.
// Une regression y serait invisible a l'oeil (la sortie resterait plausible, simplement
// amputee d'une borne), et impossible a contre-verifier sans relancer une extraction de
// 7,24 Go. Ces tests fixent le layout sur des octets construits a la main.

import (
	"encoding/binary"
	"math"
	"testing"
)

// paquet fabrique un AkPropBundle : u8 n, n x u8 idProp, n x largeur x float32.
func paquet(ids []byte, valeurs [][]float32) []byte {
	out := []byte{byte(len(ids))}
	out = append(out, ids...)
	for _, composantes := range valeurs {
		for _, v := range composantes {
			var b [4]byte
			binary.LittleEndian.PutUint32(b[:], math.Float32bits(v))
			out = append(out, b[:]...)
		}
	}
	return out
}

// Le lecteur large rend les DEUX composantes ; c'est exactement ce que l'ancien lecteur
// perdait.
func TestLirePaquetLargeRendLesDeuxComposantes(t *testing.T) {
	d := paquet([]byte{propVolume, propPitch}, [][]float32{{-2, 1}, {-50, 50}})
	vals, fin, ok := lirePaquetLarge(d, 0, 2)
	if !ok {
		t.Fatal("paquet juge implausible alors qu'il est bien forme")
	}
	if fin != len(d) {
		t.Fatalf("offset de suite %d, attendu %d", fin, len(d))
	}
	if got := vals[propVolume]; got[0] != -2 || got[1] != 1 {
		t.Fatalf("volume %v, attendu [-2 1]", got)
	}
	if got := vals[propPitch]; got[0] != -50 || got[1] != 50 {
		t.Fatalf("hauteur %v, attendu [-50 50]", got)
	}
}

// Le lecteur historique reste inchange pour la largeur 1 : les gains deja mesures ne
// doivent pas bouger sous pretexte qu'on ajoute la variation.
func TestLirePaquetPropsRestePremiereComposante(t *testing.T) {
	d := paquet([]byte{propVolume}, [][]float32{{-6}})
	vals, _, ok := lirePaquetProps(d, 0, 1)
	if !ok || vals[propVolume] != -6 {
		t.Fatalf("lecture simple = %v (ok=%v), attendu -6", vals[propVolume], ok)
	}
	large := paquet([]byte{propVolume}, [][]float32{{-2, 1}})
	vals, _, ok = lirePaquetProps(large, 0, 2)
	if !ok || vals[propVolume] != -2 {
		t.Fatalf("lecture largeur 2 = %v (ok=%v), attendu la premiere composante -2", vals[propVolume], ok)
	}
}

// Les deux composantes sont rendues ORDONNEES : le format ne dit pas laquelle est le
// minimum, une fourchette inversee produirait un tirage vide cote app.
func TestLireVariationOrdonneLesBornes(t *testing.T) {
	d := paquet([]byte{propVolume, propPitch}, [][]float32{{1, -2}, {50, -50}})
	v := lireVariation(d, 0)
	if !v.Lu {
		t.Fatal("variation non lue")
	}
	if v.VolumeDB.Bas != -2 || v.VolumeDB.Haut != 1 {
		t.Fatalf("volume %v, attendu bas=-2 haut=1", v.VolumeDB)
	}
	if v.PitchCts.Bas != -50 || v.PitchCts.Haut != 50 {
		t.Fatalf("hauteur %v, attendu bas=-50 haut=50", v.PitchCts)
	}
}

// Un paquet illisible ne doit pas rendre une fourchette inventee : sans variation, l'app
// joue le son pur, ce qui est correct — une fourchette fausse, non.
func TestLireVariationRefuseUnPaquetTronque(t *testing.T) {
	d := paquet([]byte{propVolume}, [][]float32{{-2, 1}})
	if v := lireVariation(d[:len(d)-2], 0); v.Lu {
		t.Fatal("paquet tronque accepte")
	}
	if v := lireVariation(nil, 0); v.Lu {
		t.Fatal("paquet vide accepte")
	}
}

// Bout en bout : un NodeBaseParams complet rend la valeur nominale ET sa fourchette.
// C'est le chemin reellement emprunte par le parseur de bank.
func TestLireProprietesRendNominalEtVariation(t *testing.T) {
	d := make([]byte, 12) // bIsOverrideParentFX, uNumFx=0, puis 10 octets fixes
	d = append(d, paquet([]byte{propVolume}, [][]float32{{-6}})...)
	d = append(d, paquet([]byte{propVolume, propPitch}, [][]float32{{-2, 1}, {-50, 50}})...)
	p := lireProprietesConteneur(d)
	if !p.Lu || p.VolumeDB != -6 {
		t.Fatalf("nominal = %v (lu=%v), attendu -6", p.VolumeDB, p.Lu)
	}
	if !p.Variation.Lu || p.Variation.VolumeDB.Haut != 1 || p.Variation.PitchCts.Bas != -50 {
		t.Fatalf("variation = %+v, attendue volume [-2 1] hauteur [-50 50]", p.Variation)
	}
}

// Les ecarts s'ADDITIONNENT le long du chemin (chaque noeud traverse tire le sien) et
// s'ENVELOPPENT entre variantes d'un meme point de choix (le moteur n'en joue qu'une).
func TestSommeEtEnveloppeDesFourchettes(t *testing.T) {
	a := fourchetteSon{VolumeDB: fourchette{Bas: -2, Haut: 1}, Lu: true}
	b := fourchetteSon{VolumeDB: fourchette{Bas: -3, Haut: 4}, Lu: true}
	if s := sommeFourchettes(a, b); s.VolumeDB.Bas != -5 || s.VolumeDB.Haut != 5 {
		t.Fatalf("somme %v, attendue [-5 5]", s.VolumeDB)
	}
	if e := enveloppeFourchettes(a, b); e.VolumeDB.Bas != -3 || e.VolumeDB.Haut != 4 {
		t.Fatalf("enveloppe %v, attendue [-3 4]", e.VolumeDB)
	}
}

// L'agregation retient la couche DOMINANTE : une couche de renfort 20 dB en arriere ne
// dicte pas la variation du coup, sa propre variation ne s'entend pas.
func TestVariationDeCouchesRetientLaDominante(t *testing.T) {
	couches := []brancheRendue{
		{Cible: "renfort", Variation: &variationRendue{
			VolumeDB: fourchette{Bas: -9, Haut: 9}, Couche: "renfort", GainDB: -20}},
		{Cible: "principale", Variation: &variationRendue{
			VolumeDB: fourchette{Bas: -1, Haut: 1}, Couche: "principale", GainDB: 0}},
		{Cible: "muette"},
	}
	v := variationDeCouches(couches)
	if v == nil || v.Couche != "principale" {
		t.Fatalf("couche retenue = %+v, attendue « principale »", v)
	}
	if variationDeCouches([]brancheRendue{{Cible: "muette"}}) != nil {
		t.Fatal("une fourchette a ete publiee alors qu'aucune couche n'en declare")
	}
}

// La remontee vers un mode de tir suit la meme regle, et rend une COPIE : le rapport ne
// doit pas partager un pointeur avec l'evenement dont il derive.
func TestVariationDominanteRendUneCopie(t *testing.T) {
	fort := &variationRendue{Couche: "fort", GainDB: -3}
	faible := &variationRendue{Couche: "faible", GainDB: -30}
	v := variationDominante([]*variationRendue{nil, faible, fort})
	if v == nil || v.Couche != "fort" {
		t.Fatalf("dominante = %+v, attendue « fort »", v)
	}
	if v == fort {
		t.Fatal("la dominante partage son pointeur avec la source")
	}
	if variationDominante([]*variationRendue{nil, nil}) != nil {
		t.Fatal("une dominante a ete rendue alors qu'aucune source n'existe")
	}
}
