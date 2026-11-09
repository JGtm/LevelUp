package grammar

// etats_mouvement_hooks.go — LA PUBLICATION DES ETATS DE MOUVEMENT DU SPARTAN A L INSTANT
// (lot 5.3.4, 2026-09-21).
//
// # CE QUE CE FICHIER AJOUTE, ET CE QU IL NE TOUCHE PAS
//
// Quatre deserialiseurs de l archetype bipede LISAIENT deja leurs champs et les JETAIENT :
// `i29 unit-crouch`, `i62 biped-slide`, `i55 biped-posture-physics` et la tete d
// `i18 unit-control`. Ce fichier leur donne une PORTE DE PUBLICATION, et rien d autre :
//
//	AUCUN BIT N EST LU AUTREMENT. Les quatre fonctions consomment exactement les memes
//	largeurs qu avant ; seul ce qu elles RENDENT change, et seulement quand un observateur
//	est branche (`nil` en production, cf. l en-tete d `observateur.go`).
//
// # POURQUOI UNE SEULE PORTE POUR QUATRE COMPOSANTS
//
// C est l idiome de `components_probe.go` : une enumeration STABLE (jamais un index de
// registre), un publieur, un hook. Quatre champs d observation pour quatre composants du MEME
// sujet — l etat de mouvement a l instant — se seraient copies quatre fois dans chaque
// instrument. Les valeurs voyagent dans une tranche dont la FORME est documentee par composant
// ci-dessous, et le lecteur d instrument lit la forme ici plutot que de la deviner.
//
// # LA DOCTRINE DE CE LOT, ECRITE OU ELLE SERT
//
// Ces etats sont a L INSTANT, pas aux images-cles : un delta ne porte `i29` que quand
// l accroupi CHANGE (l autorite du sujet, 2026-09-21). Un instrument qui compte des lectures
// compte donc des TRANSITIONS, pas des images — et c est ce qui rend leur cadence lisible en
// records par seconde et par slot.

import "fmt"

// compUnitControl : L ETIQUETTE DE REGISTRE d `i18`, ecrite UNE fois.
//
// Elle etait en dur a trois endroits (`component_param4.go` pour son niveau, `dispatch_biped.go`
// pour son aiguillage, et ici pour son nom publie) ; la quatrieme copie a fait mordre `goconst`,
// et c est la regle 6 de CLAUDE.md qui tranche : a la troisieme, on centralise. Le garde-rail
// est le linter lui-meme — il compte les occurrences.
const compUnitControl = "unit-control-component"

// EtatMouvementComposant designe le composant d etat de mouvement publie. Enumeration STABLE,
// pas un index de registre (meme raison que [ProbeComponent] et `GameEngineField`).
type EtatMouvementComposant int

// Les quatre composants d etat, et leur compte.
const (
	// EtatAccroupi : `ti=35 i29 unit-crouch-component` (FUN_1406d0f24). Valeurs, TOUJOURS deux :
	//   [0] le booleen d accroupissement (R(1)) ;
	//   [1] le quantum de PROGRESSION de l animation (R(10), dequantifie sur [0,1]).
	// La progression est ce qui distingue « s accroupit » de « est accroupi » : un lot qui
	// cherche l etat A L INSTANT lit le booleen, un lot qui cherche le GESTE lit la progression.
	EtatAccroupi EtatMouvementComposant = iota
	// EtatGlissade : `ti=35 i62 biped-slide-component` (FUN_142f26ce8). Valeurs, TOUJOURS sept —
	// les absentes valent zero, et la porte dit laquelle :
	//   [0] la porte du composant (R(1)) ; a zero, tout le reste est zero et la glissade est
	//       ABSENTE de cet instant ;
	//   [1] la porte du bloc quantifie (R(1)) ; a un, la direction et la magnitude sont ABSENTES
	//       (vecteur constant) ;
	//   [2] la direction empaquetee (R(19)) ; [3] la magnitude (R(10)) ;
	//   [4] la premiere fraction (R(8)) ; [5] la seconde, lue seulement si `param_4 >= 1`
	//       (0 sinon) ; [6] la troisieme (R(8), en ligne).
	EtatGlissade
	// EtatPosture : `ti=35 i55 biped-posture-physics-component` (FUN_142f0293c ->
	// FUN_142f1f630). Valeurs, TOUJOURS six — l UNION d etat physique du bipede, dont les
	// quatre charges sont portees depuis le lot 5.7 (`components_biped_posture.go`) :
	//   [0] le TAG de l union (R(2)) : le repartiteur `FUN_141fd997c` pose un octet de genre
	//       1, 2 ou 3 pour les tags 1, 2, 3, et prend une quatrieme voie pour le tag 0 ;
	//   [1] la PORTE de tete de la branche (1 = la charge lourde est presente) ;
	//   [2] le SOUS-GENRE : le second discriminant du tag 0 (0..3) ou le premier champ court
	//       du tag 1 (0 = charge absente) ; zero sur les tags 2 et 3 ;
	//   [3] le HANDLE court R(15) (tags 0 sous-genres 1/2/3, 1 charge presente, 2 et 3) ;
	//   [4] la DIRECTION empaquetee R(19), presente avec la queue quantifiee ;
	//   [5] le MOT de 32 bits : l identifiant de l objet porteur du tag 2 (brut ou resolu par
	//       reference d entite), la queue du tag 0 sous-genres 2/3, le champ du tag 3.
	// LA CORRESPONDANCE TAG -> CLASSE N EST PAS ETABLIE : l image ne porte que trois classes
	// d etat de bipede (`c_biped_ground_state`, `c_biped_airborne_state`,
	// `c_biped_vehicle_state`) et aucune chaine n est attachee a l octet de genre.
	EtatPosture
	// EtatControleUnite : la TETE d `i18 unit-control-component` (FUN_14080d69c). Valeurs,
	// TOUJOURS quatre :
	//   [0] la porte de tete (R(1)) ; a zero, les trois suivantes valent zero ;
	//   [1] le premier index (R(5)) ; [2] la porte du second (R(1)) ; [3] le second index (R(6)).
	// LE MOT DE 32 BITS DE QUEUE N EST PAS ICI : il est deja publie par `UnitRefHook`
	// (`UnitRefWord32`, cf. `consumeOpt32`) — une seconde porte pour la meme valeur serait la
	// troisieme copie que la regle 6 interdit.
	EtatControleUnite
	// EtatVitesse : `ti=35 i1 object-translational-velocity` (FUN_14076d45c). Valeurs,
	// TOUJOURS quatre :
	//   [0] le mode (R(1)) ; a un, la vitesse est en PLEINE PRECISION (R(96)) et les trois
	//       suivantes valent zero — ce chemin n est pas dequantifie ici ;
	//   [1] la porte du bloc quantifie (R(1)) ; a un, la vitesse est ABSENTE de cet instant ;
	//   [2] la direction empaquetee (R(19)) ; [3] le mot d echelle (R(10)).
	// `DecodeVelocity(dir, echelle)` en rend le vecteur en m/s : sa composante VERTICALE est le
	// candidat du SAUT, et la norme de ses deux composantes horizontales la vitesse au sol —
	// celle ou un sprint ferait une seconde bosse.
	EtatVitesse
	// EtatMobilite : `ti=35 i54 biped-mobility-action-component` (FUN_1408f0264). Valeurs,
	// TOUJOURS deux :
	//   [0] le drapeau d AMORCE (R(1)) — « une action est transmise a cet instant » ;
	//   [1] le second drapeau (R(1)).
	// LE HOOK HISTORIQUE (`MobilityActionHook`) RESTE, et il publie les memes deux drapeaux :
	// ses appelants (le balayage des capacites) n ont pas besoin du slot. Cette porte-ci
	// l ajoute, et c est la seule raison de son existence — un intervalle par VIE l exige.
	EtatMobilite
	// EtatCapaciteActive : `ti=35 i57 biped-spartan-ability-component` (FUN_142f02810 ->
	// FUN_142f268c4). Valeurs, TOUJOURS une :
	//   [0] l INDEX DE LA FENTE DE CAPACITE ACTIVE, decale de +1 comme le flux l ecrit —
	//       le flux lit `R(2)` et l ecrivain pose `bloc+3 = valeur - 1`, donc la valeur BRUTE
	//       `0` signifie « aucune fente active » et `1`, `2`, `3` designent les fentes `0`,
	//       `1`, `2`. La porte publie le BRUT : convertir ici cacherait le decalage.
	//
	// ET LES TROIS FENTES SONT NOMMEES PAR L IMAGE (lot 5.9.5). `FUN_1407e9ce4` aiguille sur le
	// GROUPE DE TAG de la definition de capacite et appelle, pour chacun, un desenregistreur qui
	// teste l index actif contre SA fente :
	//
	//	'saev' (0x73616576, esquive)  -> FUN_14319d0ac : fente `comp+0x1c`, index actif **0**
	//	'sasp' (0x73617370, SPRINT)   -> FUN_14319d1ec : fente `comp+0x20`, index actif **1**
	//	'sagh' (0x73616768, grappin)  -> FUN_14319d14c : fente `comp+0x24`, index actif **2**
	//
	// D ou : valeur brute **2** = LE SPRINT EST ACTIF. C est ce que publie `movement_states.go`
	// sous le genre `sprint`, et c est une LECTURE, pas une derivation.
	EtatCapaciteActive
	// EtatMouvementCount est le nombre de composants publies.
	EtatMouvementCount = 7
)

// String rend l etiquette de registre du composant d etat.
func (c EtatMouvementComposant) String() string {
	switch c {
	case EtatAccroupi:
		return "unit-crouch-component"
	case EtatGlissade:
		return "biped-slide-component"
	case EtatPosture:
		return "biped-posture-physics-component"
	case EtatControleUnite:
		return compUnitControl
	case EtatVitesse:
		return "object-translational-velocity-component"
	case EtatMobilite:
		return "biped-mobility-action-component"
	case EtatCapaciteActive:
		return "biped-spartan-ability-component"
	}
	return fmt.Sprintf("etat de mouvement inconnu (%d)", int(c))
}

// publishEtatMouvement rend au hook, s il y en a un, les valeurs d un composant d etat, AVEC LE
// SLOT du record en cours de decodage.
//
// LE SLOT VIENT DU LECTEUR, et c est ce qui rend la mesure par VIE possible : la boucle de
// records le pose (`poserSlotDeCapture`) avant de traverser l entite, exactement comme pour les
// echantillons de position. Sans lui, une cadence « par slot » ne serait qu un total.
func (b *Lecteur) publishEtatMouvement(comp EtatMouvementComposant, values ...uint64) {
	if b.obs == nil || b.obs.EtatMouvementHook == nil {
		return
	}
	b.obs.EtatMouvementHook(comp, b.cap.accumSlot, values)
}
