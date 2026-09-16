package profile

// mpp_widths.go — LE DECOUPAGE DU BLOC `object-multiplayer-properties`, COMME VALEUR.
//
// EXTRAIT DE `grammar/mpp_widths.go` AU LOT 2.5.b : le TYPE, les deux largeurs par defaut et
// leur provenance descendent ici ; ce qui LIT (`Lecteur.mppWidths`, l installation par le
// contexte de film, la calibration sur le film) reste en `grammar`. La coupe suit la couche,
// pas le fichier : le decoupage est une DONNEE du format, sa lecture est de la grammaire.

import "fmt"

// mppLeadParDefaut est la largeur du PREMIER champ du bloc MPP (FUN_141fd72c0). Le décompile la
// donne à 9, et 9 est bit-exact sur les films d'arène — mais elle VARIE d'un film à l'autre :
// mesuré le 2026-08-17, le default-state de ti=37 fait 60 bits sur `000d5950` et `00162144`
// (largeur 9) contre 57 sur `06dfe6d9` et `00ba2e1c`. C'est le même genre de largeur de
// configuration de réplication que les largeurs d axe du chemin world-object (mais PAS que
// la plage de `FUN_1406d3140` : celle-la est une CONSTANTE du binaire, cf. `varwidth.go`,
// releve du 2026-09-15) : posée au chargement de la carte, absente de l exécutable, et donc
// DÉTECTÉE dans le film (cf. CalibrateMPPWidths) plutôt que devinée.
//
// Le défaut 9 est celui du chemin bipède, validé en live (rep = 166 ou 198 bits).
//
// C'ÉTAIT LA VARIABLE DE PAQUET `mppLeadBits` JUSQU'AU LOT 2.2.e, puis l'héritage de
// processus jusqu'au lot 2.3 : la largeur vit dans le PROFIL que le lecteur porte.
const mppLeadParDefaut = 9

// mppIndexParDefaut est la largeur du champ inline `R(5) -> DST+0x1a`, qui suit l'identifiant de
// 32 bits. Le décompile la donne à 5.
//
// POURQUOI ELLE EST PARAMÉTRABLE, ET CE QUE ÇA CHANGE : le default-state de ti=37 perd 3 bits
// sur certains films, et DEUX champs inconditionnels du chemin minimal peuvent le porter — le
// premier du bloc (mppLeadParDefaut) et celui-ci. Les deux donnent le même TOTAL, mais pas la même
// lecture : rétrécir le premier décale l'identifiant de 32 bits de 3 bits et rend un identifiant
// FAUX, rétrécir celui-ci le laisse en place. Seule la mesure tranche, et elle le fait sur le
// nombre de records que chaque découpage fait tomber sur l'oracle de position
// (cf. CalibrateMPPWidths) — jamais sur une préférence d'écriture.
//
// C'ÉTAIT LA VARIABLE DE PAQUET `mppIndexBits` JUSQU'AU LOT 2.2.e.
const mppIndexParDefaut = 5

// MPPParDefaut rend le découpage MPP par défaut — la SOURCE UNIQUE de l'invariant, comme
// [MouvementParDefaut] l'est pour le mouvement et [CadreParDefaut] pour le cadre d'image-clé.
//
// EXPORTEE AU LOT 2.5.b : elle s appelait `mppDuProfil` et son second appelant,
// `grammar.ProfilDeBalayageParDefaut`, vit maintenant de l autre cote de la frontiere. Recopier
// les deux littéraux là-bas aurait fait deux invariants pour une seule largeur.
func MPPParDefaut() MPPWidths { return MPPWidths{Lead: mppLeadParDefaut, Index: mppIndexParDefaut} }

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
