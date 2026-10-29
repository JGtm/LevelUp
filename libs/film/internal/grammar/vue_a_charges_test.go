package grammar

import "testing"

// vue_a_charges_test.go — les vecteurs des charges de la vue A (lot LN), ecrits avec les IMMEDIATS
// du lecteur du jeu (litteraux, jamais les constantes du portage) : une largeur mal portee decale
// la fin de la charge.

// vecteurDeCharge est une charge et le flux que son lecteur du jeu consomme exactement.
type vecteurDeCharge struct {
	nom    string
	charge func(*Lecteur) bool
	ecrire func(w *bitWriter)
}

// TestLesChargesLisentLesLargeursDuJeu : chaque charge s arrete au bit qui suit le flux ecrit
// d apres son lecteur. MUTATIONS : la queue R(1) de `FUN_142f1c6cc` retiree, une position lue au
// mode 2 de `FUN_142ef15e0`, le R(32) de `FUN_142c61310` a 31 bits, la queue R(9) de
// `FUN_140f58324` a 8, la magnitude R(10) de `FUN_14076d528` a 9, le R(3) de `FUN_141037828` a 2
// -> ROUGE.
func TestLesChargesLisentLesLargeursDuJeu(t *testing.T) {
	for _, v := range []vecteurDeCharge{
		{"FUN_142f1c6cc projectile_object_impact_effect", chargeImpactSurObjet, func(w *bitWriter) {
			w.bit(1)            // variante presente : rien ne suit
			w.bits(0, 7+7)      // FUN_1406d84b4 x 2, EBX = 7
			w.bits(0, 0x13)     // FUN_14076dc04, R9D = EBX + 0xc
			w.bits(0, 2)        // R(2)
			w.bits(0, 3*0xc)    // FUN_140c1e9d4 x 3, R9D = 0xc
			w.bits(0, 0x13)     // FUN_14076dc04, R9D = R14D
			w.bits(0, 9+16+1+1) // R(9), R(16), R(1), R(1)
		}},
		{"FUN_142ef8f74 networked_ai_effect, mode 2", chargeEffetDIA, func(w *bitWriter) {
			w.bit(0)     // FUN_14080d6f0 absent
			w.bits(2, 2) // FUN_142ef15e0 : mode 2, aucune position
		}},
		{"FUN_142ef8f74 networked_ai_effect, mode 3", chargeEffetDIA, func(w *bitWriter) {
			w.bit(0)
			w.bits(3, 2) // mode 3, aucune position
		}},
		{"FUN_142c61310 PersonalAILifceycleEffect", chargeCycleDeVieIA, func(w *bitWriter) {
			w.bits(0, 2)  // FUN_142af27f8 R(2)
			w.bit(1)      // FUN_1406cf008 : present
			w.bits(0, 32) // FUN_14080d6f0
			w.bit(0)      // FUN_1406cf008 : absent
			w.bits(0, 32) // FUN_14080dec4 « marker-name »
		}},
		{"FUN_142f1686c TeamGameEvent", chargeEvenementDEquipe, func(w *bitWriter) {
			w.bits(0, 32+8) // FUN_142efb480 : R(32), R(8)
			w.bits(0, 4)    // FUN_142efb634 : aucune propriete
			w.bit(0)        // FUN_14080b034 : sous-sac absent
			w.bits(0, 9)    // FUN_140f58324 R(9)
		}},
		{"FUN_142f16818 TeamGameEventSmall", chargeEvenementDEquipeCourt, func(w *bitWriter) {
			w.bits(0, 32+8) // FUN_14080ae70 : R(32), R(8)
			w.bits(0, 3)    // FUN_14080b1b8 : aucune propriete
			w.bit(0)        // sous-sac absent
			w.bits(0, 9)    // FUN_140f58324 R(9)
		}},
		{"FUN_14116c344 EquipmentKnockbackPlayer", chargePousseeDuJoueur, func(w *bitWriter) {
			w.bit(0)        // FUN_14076d528 : porte a 0
			w.bits(0, 0x13) // septieme argument 0x13
			w.bits(0, 10)   // sixieme argument 10
		}},
		{"FUN_141037828 biped_pickup", chargeRamassage, func(w *bitWriter) {
			w.bits(5, 3)  // R(3)
			w.bit(1)      // FUN_14080d69c : present
			w.bits(0, 32) // R(32)
		}},
	} {
		var w bitWriter
		v.ecrire(&w)
		fin := w.n
		w.bits(0xffff, 16)
		br := LecteurSur(w.buf)
		if !v.charge(br) || br.BitPos() != fin {
			t.Errorf("%s : fin %d, attendu %d", v.nom, br.BitPos(), fin)
		}
	}
}

// TestUneChaineSansNulEstRefusee : `FUN_1407cbc24` lit au plus `n` octets ; sans nul, il leve
// l erreur du flux. MUTATION : `n` octets sans nul acceptes -> ROUGE.
func TestUneChaineSansNulEstRefusee(t *testing.T) {
	var w bitWriter
	for range 16 {
		w.bits(0x41, 8)
	}
	w.bits(0, 8)
	if lireChaine(LecteurSur(w.buf), 16) {
		t.Fatal("seize octets sans nul acceptes")
	}
	var c bitWriter
	c.bits(0x41, 8)
	c.bits(0x42, 8)
	c.bits(0, 8)
	br := LecteurSur(c.buf)
	if !lireChaine(br, 16) || br.BitPos() != 24 {
		t.Fatalf("chaine « AB » : fin %d, attendu 24", br.BitPos())
	}
}
