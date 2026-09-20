package geo

// types.go — LES ENTREES, LES REGLAGES PHYSIQUES ET LES NOEUDS.
//
// Deux jeux de constantes cohabitent et ne se confondent pas :
//
//   - `Parametres` sont des grandeurs PHYSIQUES (hauteur des yeux, marche maximale, hauteur
//     libre) : elles decrivent un Spartan et une grille, pas une opinion sur ce qu'est une
//     bonne position. Elles ne changent pas d'une carte a l'autre.
//   - `Reglage` (score.go) porte les POIDS et les SEUILS de selection : c'est lui qui se fige
//     sur les cartes de calibrage, et lui seul.

import "levelup/go-api/internal/analysis/tactical"

// Parametres sont les grandeurs physiques de la mesure.
type Parametres struct {
	// PasVoxelXYM, PasVoxelZM : cote et hauteur d'un voxel d'occlusion, en metres. Plus fins
	// que la cellule tactique (0,5 m) : un mur qui effleure le coin d'une cellule ne doit pas
	// la condamner, et une marche doit se distinguer d'une caisse.
	PasVoxelXYM float64
	PasVoxelZM  float64
	// HauteurYeuxM : hauteur des yeux au-dessus du sol du noeud, source et cible des rayons.
	HauteurYeuxM float64
	// HauteurLibreM : hauteur libre exigee au-dessus d'un sol. 1,5 m et non la taille d'un
	// Spartan debout : LE MAILLAGE DE RENDU N'EST PAS LE MAILLAGE DE COLLISION. Mesure du
	// 2026-09-20 sur Recharge : les joueurs des artefacts de rejeu montent l'escalier de la
	// fosse (points a 0,5 / 1,0 / 1,5 m) la ou une corniche de rendu a 1,6-2,0 m au-dessus
	// des marches les rejetait a 1,9 m. Un Spartan accroupi passe sous 1,5 m ; le decor bas
	// sans collision, lui, n'est pas plus bas que ca.
	HauteurLibreM float64
	// MarcheMaxM : denivele maximal franchi entre deux cellules voisines (orthogonales ou
	// diagonales : c'est une hauteur de marche, pas une pente). C'est aussi la ZONE DE
	// MARCHE au-dessus d'un sol, dont l'occupation est ignoree par le controle de hauteur
	// libre : ce qu'on enjambe ne bloque pas, et la face d'une dalle en contact avec la
	// derniere cellule d'une rampe ne la tue plus.
	MarcheMaxM float64
	// SautMaxM : montee maximale entre deux noeuds voisins, saut et clamber compris ; au-dela
	// de la marche elle coute son denivele en detour. ChuteMaxM : descente maximale, sans
	// cout (un Spartan ne se blesse pas en tombant). Les deux rendent le graphe ORIENTE.
	SautMaxM  float64
	ChuteMaxM float64
	// HauteurPassageM : hauteur, au-dessus du sol, du rayon qui verifie qu'aucun mur ne
	// separe deux noeuds voisins (la poitrine : au-dessus de ce qu'on enjambe, sous les
	// rambardes). Sans lui, deux centres de cellule de part et d'autre d'un mur mince sont
	// relies A TRAVERS le mur. HauteurSautM : hauteur du second rayon, lance quand le
	// premier est bloque — libre, l'obstacle est un muret que l'on saute.
	HauteurPassageM float64
	HauteurSautM    float64
	// PenteMaxCos : cosinus minimal de la normale (composante Z) pour qu'un triangle soit un
	// sol. cos 50 deg = 0,64 : une rampe de Halo est praticable, un mur non.
	PenteMaxCos float64
	// RayonHM : rayon du voisinage, en metres, dont la moyenne d'altitude donne H.
	RayonHM float64
	// PasCiblesCellules : sous-echantillonnage des CIBLES de visibilite, en cellules — 2
	// cellules = une cible tous les metres. Les sources restent tous les noeuds.
	PasCiblesCellules int
	// PorteeVueM : au-dela, un rayon n'est pas lance et la cible est tenue pour invisible.
	PorteeVueM float64
	// NbSecteurs : nombre de secteurs angulaires de E (36 = secteurs de 10 deg).
	NbSecteurs int
	// PorteeCouvertM : rayon maximal, en distance de deplacement, de la recherche du couvert.
	PorteeCouvertM float64
	// RayonPlacementM : distance XY maximale entre une ancre (ou une ressource) et le noeud
	// qui la porte ; DeltaZPlacementM la tolerance en altitude.
	RayonPlacementM  float64
	DeltaZPlacementM float64
}

// ParametresParDefaut rend les grandeurs physiques retenues le 2026-09-20.
func ParametresParDefaut() Parametres {
	return Parametres{
		PasVoxelXYM:       0.25,
		PasVoxelZM:        0.25,
		HauteurYeuxM:      1.7,
		HauteurLibreM:     1.5,
		MarcheMaxM:        0.75,
		SautMaxM:          1.6,
		ChuteMaxM:         5.0,
		HauteurPassageM:   1.2,
		HauteurSautM:      1.9,
		PenteMaxCos:       0.64,
		RayonHM:           6.0,
		PasCiblesCellules: 2,
		PorteeVueM:        60.0,
		NbSecteurs:        36,
		PorteeCouvertM:    12.0,
		RayonPlacementM:   1.5,
		DeltaZPlacementM:  2.0,
	}
}

// Ressource est un point d'interet dont on mesure la distance de deplacement.
type Ressource struct {
	X, Y, Z float64
	// Nature : `arme_forte`, `arme`, `powerup` ou `objectif`.
	Nature string
}

// Natures de ressource. Les armes de ratelier comptent moins qu'une arme de pouvoir ; les
// deux sont mesurees a part.
const (
	NatureArmeForte = "arme_forte"
	NatureArme      = "arme"
	NaturePowerup   = "powerup"
	NatureObjectif  = "objectif"
)

// Noeud est une cellule praticable A UN NIVEAU DONNE : une cellule a etages porte un
// noeud par etage. `Index` est l'index lineaire dans le cadre, `Niveau` le rang du sol
// dans la cellule (0 = le plus bas).
type Noeud struct {
	Cellule tactical.Cellule
	Index   int
	Niveau  int
	X, Y, Z float64
}

// Variables sont les cinq variables d'un noeud, brutes ou normalisees.
type Variables struct {
	H, V, E, R, M float64
}

// NoeudMesure est un noeud avec ses mesures.
type NoeudMesure struct {
	Noeud
	// Brut : H en metres, V et E dans [0, 1], R et M en PROXIMITE dans [0, 1].
	Brut Variables
	// Norm : les memes, ramenees dans [0, 1] par carte (cf. Normalise).
	Norm Variables
	// Les distances de deplacement, en metres ; +Inf quand rien n'est atteignable.
	DArmeForte float64
	DArme      float64
	DObjectif  float64
	// DCouvert : distance de deplacement au premier noeud cache de la majorite des
	// cibles qui voient celui-ci.
	DCouvert float64
	// NbVisibles : cibles visibles depuis le noeud (a hauteur d'yeux).
	NbVisibles int
	// Score : la combinaison ponderee des variables normalisees.
	Score float64
}
