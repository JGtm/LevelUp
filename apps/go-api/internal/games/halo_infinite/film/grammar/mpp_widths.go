package grammar

// mpp_widths.go — LE DECOUPAGE DU BLOC `object-multiplayer-properties`, ET RIEN D AUTRE.
//
// SORTI DE `default_state.go` PAR DEPLACEMENT PUR au lot 2.2.e : pas une ligne de logique ne
// change. La coupure est forcee par le seuil de 500 lignes du depot, que la migration des deux
// largeurs vers le profil a fait franchir a `default_state.go` (503) — et elle suit une
// responsabilite, pas une arithmetique : ce fichier porte la LARGEUR des deux champs variables
// du bloc, sa structure, son installation et sa lecture ; `default_state.go` garde les
// deserialiseurs qui les emploient.

import "fmt"

// mppLeadBits est la largeur du PREMIER champ du bloc MPP (FUN_141fd72c0). Le décompile la
// donne à 9, et 9 est bit-exact sur les films d'arène — mais elle VARIE d'un film à l'autre :
// mesuré le 2026-08-17, le default-state de ti=37 fait 60 bits sur `000d5950` et `00162144`
// (largeur 9) contre 57 sur `06dfe6d9` et `00ba2e1c`. C'est le même genre de largeur de
// configuration de réplication que les largeurs d axe du chemin world-object (mais PAS que
// la plage de `FUN_1406d3140` : celle-la est une CONSTANTE du binaire, cf. `varwidth.go`,
// releve du 2026-09-15) : posée au chargement de la carte, absente de l exécutable, et donc
// DÉTECTÉE dans le film (cf. CalibrateMPPWidths) plutôt que devinée.
//
// Le défaut 9 est celui du chemin bipède, validé en live (rep = 166 ou 198 bits) : il ne bouge
// restaurer la valeur précédente — c'est un état de processus (le profil hérité).
//
// C'ÉTAIT LA VARIABLE DE PAQUET `mppLeadBits` JUSQU'AU LOT 2.2.e, puis l'héritage de
// processus jusqu'au lot 2.3 : la largeur vit dans le PROFIL que le lecteur porte.
const mppLeadParDefaut = 9

// mppIndexBits est la largeur du champ inline `R(5) -> DST+0x1a`, qui suit l'identifiant de
// 32 bits. Le décompile la donne à 5.
//
// POURQUOI ELLE EST PARAMÉTRABLE, ET CE QUE ÇA CHANGE : le default-state de ti=37 perd 3 bits
// sur certains films, et DEUX champs inconditionnels du chemin minimal peuvent le porter — le
// premier du bloc (mppLeadBits) et celui-ci. Les deux donnent le même TOTAL, mais pas la même
// lecture : rétrécir le premier décale l'identifiant de 32 bits de 3 bits et rend un identifiant
// FAUX, rétrécir celui-ci le laisse en place. Seule la mesure tranche, et elle le fait sur le
// nombre de records que chaque découpage fait tomber sur l'oracle de position
// (cf. CalibrateMPPWidths) — jamais sur une préférence d'écriture.
//
// C'ÉTAIT LA VARIABLE DE PAQUET `mppIndexBits` JUSQU'AU LOT 2.2.e.
const mppIndexParDefaut = 5

// mppDuProfil rend le découpage MPP par défaut — la SOURCE UNIQUE de l'invariant, comme
// [mouvementDuProfil] l'est pour le mouvement et [cadreDuProfil] pour le cadre d'image-clé.
func mppDuProfil() MPPWidths { return MPPWidths{Lead: mppLeadParDefaut, Index: mppIndexParDefaut} }

// mppWidths rend le découpage du bloc MPP que ce lecteur porte.
func (b *Lecteur) mppWidths() MPPWidths { return b.p.MPP }

// MPPWidths est le découpage des deux champs de largeur variable du bloc MPP.
type MPPWidths struct {
	// Lead est la largeur du premier champ du bloc (FUN_141fd72c0).
	Lead int
	// Index est la largeur du champ inline qui suit l'identifiant de 32 bits.
	Index int
}

// String rend le découpage sous la forme « lead/index ».
func (w MPPWidths) String() string { return fmt.Sprintf("%d/%d", w.Lead, w.Index) }

// Valid dit si le découpage est renseigné.
func (w MPPWidths) Valid() bool { return w.Lead > 0 && w.Index > 0 }
