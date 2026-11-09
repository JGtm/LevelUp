//go:build research

package mouvement

// Cible est un composant que le lot 5.3 interroge, avec l'etat du depot AVANT la passe : ce que
// le decodeur en lit aujourd'hui. Le rapport met les deux en regard — c'est la seule facon de
// voir ce qui est LU, ce qui est JETE, et ce qui n'est pas porte du tout.
type Cible struct {
	Archetype string
	Nom       string
	Pourquoi  string
	// LuAujourdhui decrit, en une ligne, la grammaire portee par le decodeur du depot.
	LuAujourdhui string
}

// Cibles porte les composants du lot, dans l'ordre de la question posee.
//
// L'ORDRE SUIT LA QUESTION : accroupi (i29), glissade (i62), l'action de mobilite qui est la
// seule candidate connue pour le sprint et l'escalade (i54), la posture physique (i55), puis
// la vitesse (i1) — candidate du SAUT par derivation si aucun composant ne le porte.
var Cibles = []Cible{
	{"ti=35 i29", "unit-crouch-component",
		"l'etat ACCROUPI ; `ecs_table.tsv` le dit porte, niveau 0, valeur jetee",
		"R(1) + dequant R(10) (consumeUnitCrouch, unit_weaponstate.go) — les deux valeurs jetees"},
	{"ti=35 i62", "biped-slide-component",
		"la GLISSADE ; porte depuis le lot 2.x, aucune valeur publiee",
		"R(1) porte ; si 1 : dir/mag quantifies (1 + {0|29}) + R(8) + [R(8) si param_4>=1] + R(8)"},
	{"ti=35 i54", "biped-mobility-action-component",
		"l'ACTION DE MOBILITE : la seule candidate connue pour le SPRINT et l'ESCALADE",
		"R(1) flag1 + R(1) flag2 (MobilityActionHook, sans consommateur) ; si flag1 : handle a largeur variable + corps de 365 a 447 bits"},
	{"ti=35 i55", "biped-posture-physics-component",
		"la POSTURE physique : 2 bits, donc au plus quatre postures",
		"R(2) SAUTE (consumeBipedPosturePhysics = br.Skip(2)) — le tag est lu, sa charge ne l'est pas"},
	{"ti=35 i1", "object-translational-velocity-dynamic-precision-component",
		"la VITESSE : le SAUT n'a pas de composant nomme, il se derive de la composante verticale",
		"R(1) + {R(96) brut | R(19) direction + R(10) magnitude} — decode puis jete"},
}

// MotsDuMouvement est le vocabulaire cherche dans le pool complet des chaines de l'image.
//
// LES MOTS NE SONT PAS CHOISIS AU HASARD : les quatre premiers sont les etats que
// l'utilisateur nomme ; les suivants sont le vocabulaire que le moteur emploie POUR CES MEMES
// GESTES (« clamber » est l'escalade de Halo, « mantle » son nom dans la plupart des moteurs,
// « thrust » la poussee laterale, « vault » le franchissement) ; les derniers sont les mots
// d'ETAT sous lesquels un moteur range ces gestes (posture, stance, airborne, grounded). Un
// mot absent de ce balayage est absent de l'image.
var MotsDuMouvement = []string{
	"sprint", "slide", "crouch", "jump",
	"clamber", "mantle", "vault", "thrust", "dive", "lunge", "boost",
	"posture", "stance", "airborne", "grounded", "locomotion", "mobility", "movement",
}
