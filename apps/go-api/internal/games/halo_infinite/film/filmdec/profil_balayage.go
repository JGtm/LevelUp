package filmdec

// profil_balayage.go — CE QU UN BALAYAGE POSE SUR LES LECTEURS DE BITS QU IL CONSTRUIT
// (lot 2.3 du PLAN_DECODEUR_FILM ; remplace `profil_herite.go`, l heritage PAR L ETAT DU
// PROCESSUS que les lots 2.2.a, 2.2.b et 2.2.e avaient regroupe en attendant ce lot).
//
// # CE QUE CE FICHIER FERME
//
// Jusqu au 2.3, les valeurs ci-dessous vivaient dans UNE variable de paquet (`herite`) que la
// calibration de `killsource` ecrivait et que `NewBitReader` relisait. La consequence etait
// ecrite noir sur blanc dans `profil_herite.go` : `replaybuild.BuildBytes` decode `killsource`
// PUIS appelle `replay.BuildFromFilm` dans le MEME processus, et la cuisson du rejeu decodait
// donc ses composants aux largeurs que la calibration du kill-feed avait retenues — sans que
// personne ne le demande, et sans qu aucun commentaire ne le declare.
//
// LES LARGEURS CALIBREES SUR LE FILM SONT VOULUES (la grammaire mesuree prime sur le defaut) ;
// L HERITAGE PAR L ETAT NE L EST PAS. Le lot 2.3 separe les deux : la calibration RETOURNE son
// resultat ([killsource.Result.ProfilCalibre]), `replaybuild` le PASSE a
// `replay.BuildFromFilm`, et `BuildFromFilm` le pose sur le contexte du film. Le chemin est
// explicite d un bout a l autre, et deux films peuvent se decoder en parallele : plus rien
// n est partage par le processus.
//
// # CE QUE LE PROFIL DE BALAYAGE N EST PAS
//
// Ce n est pas [Profile] — le profil du FILM, resolu une fois a la construction du contexte et
// IMMUABLE. Le profil de balayage est ce qu un balayage POSE : l invariant du profil par
// defaut, puis ce que la carte du match (largeurs d axe objets du monde), la version de format
// (decoupage MPP) ou une calibration (descripteur de traversee, largeur d axe absolue,
// `param_4`) y substituent pour la duree de CE decodage.

// ProfilDeBalayage porte tout ce qu un lecteur de bits doit savoir avant de lire son premier
// bit. Il voyage par VALEUR : un lecteur qui le recoit en tient sa propre copie, et une passe
// ne peut donc pas modifier celle d une autre.
type ProfilDeBalayage struct {
	// Mouvement : descripteur de traversee, largeur d axe absolue, descripteur world-object,
	// range de dequantification, quantum et largeur du delta, drapeaux de contexte.
	Mouvement MovementProfile
	// Cadre : l en-tete par entite et la largeur d un mot de taille des images-cles d etat
	// complet. Invariants du format, relus chez l ecrivain — rien ne les installe par film.
	Cadre KeyframeProfile
	// MPP : le decoupage des deux champs de largeur variable du bloc
	// `object-multiplayer-properties`, pose par la VERSION DE FORMAT du film.
	MPP MPPWidths
	// ParamEtat / ParamEtatImpose : le `param_4` du moteur qu un harnais de balayage a force,
	// et le drapeau qui dit qu il l a force. Hors balayage, la table par composant decide seule.
	ParamEtat       uint32
	ParamEtatImpose bool
}

// ProfilDeBalayageParDefaut rend l INVARIANT — la valeur qu un lecteur porte quand aucun
// balayage ne lui a rien pose. Ses trois composantes viennent des MEMES fonctions que
// [ResolveProfile] : il n existe pas de seconde table de valeurs.
func ProfilDeBalayageParDefaut() ProfilDeBalayage {
	return ProfilDeBalayage{Mouvement: mouvementDuProfil(), Cadre: cadreDuProfil(), MPP: mppDuProfil()}
}

// LargeursObjetDuMonde rend les largeurs d axe du chemin world-object de ce profil.
func (p ProfilDeBalayage) LargeursObjetDuMonde() PrecisionDescriptor { return p.Mouvement.WorldObject }

// PoserLargeursObjetDuMonde installe des largeurs world-object brutes. Les instruments et les
// garde-rails qui sauvent puis restaurent un profil passent par la.
func (p *ProfilDeBalayage) PoserLargeursObjetDuMonde(d PrecisionDescriptor) {
	p.Mouvement.WorldObject = d
}

// PoserLargeursObjetDuMondeDepuisDecoupage installe les largeurs d axe de la CARTE pour le
// chemin world-object. Les axes sont partages avec l absolu du bipede : c est le meme AABB de
// BSP qui les fixe — hypothese verifiee par ses consequences le 2026-08-15 (cf.
// [BitReader.worldObjectPrecision]).
//
// SOURCE ATTENDUE : `MapQuantEntry.AxisWidths`, deduit des bornes par la loi du moteur. Le
// decoupage lu dans le film (`DetectI0Layout`) sert de controle : s il contredit le catalogue,
// ce sont les BORNES qui sont fausses.
func (p *ProfilDeBalayage) PoserLargeursObjetDuMondeDepuisDecoupage(l I0Layout) {
	if l.AxisW[0] == 0 || l.AxisW[1] == 0 || l.AxisW[2] == 0 {
		return // decoupage non detecte : garder le defaut plutot qu installer des zeros
	}
	p.Mouvement.WorldObject.AxisW = l.AxisW
	// La largeur de l INDEX DE REGION est elle aussi une constante par carte
	// (ceilLog2(nb de regions) — 2 bits sur Live Fire, lot C catalogues 2026-08-27). Un
	// decoupage sans gate (appels historiques qui ne posent que AxisW) laisse le defaut.
	// LIMITE ASSUMEE : le lecteur world-object dequantifie tous les records aux largeurs
	// de LA region cataloguee ; un record d une autre region (rarissime — l ordre des
	// 3/291 288 de Cliffhanger) consommerait des largeurs differentes et desalignerait
	// SON record.
	if l.GateBits > i0SpineBits+i0UseDefaultBits {
		p.Mouvement.WorldObject.IndexW = uint(l.GateBits - i0SpineBits - i0UseDefaultBits)
	}
	// LA REGION ATTENDUE SUIT LES LARGEURS, par le meme chemin et dans le meme appel
	// (lot B-bis, 2026-09-12). Sans elle, le lecteur world-object exigeait un index de region
	// NUL — vrai partout sauf sur Live Fire, dont la region jouee est la 1 sur 2 bits. Il y
	// lisait donc ses trois axes un bit trop tot, et le bit de poids fort de chaque axe
	// devenait le bit de poids faible du champ precedent : un pas de la moitie de l etendue
	// de l axe a chaque bascule (31,89 m sur Y, mesure sur quatre films).
	p.Mouvement.WorldObject.Region = l.Region
}

// PoserParamEtat force le `param_4` du moteur (l actor-tick / weapon-set count que le
// descripteur d un composant rend a FUN_14076cb60) pour les balayages qui prennent ce profil.
// Cf. [paramForComponent] : le repli n est consulte que HORS de la table par composant.
func (p *ProfilDeBalayage) PoserParamEtat(v uint32) { p.ParamEtat, p.ParamEtatImpose = v, true }
