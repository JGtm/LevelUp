package profile

// profile.go — LE PROFIL D UN FILM, RESOLU UNE FOIS, IMMUABLE (D1 du PLAN_DECODEUR_FILM,
// D-3 d ADR 0034).
//
// # CE QU IL EST
//
// TOUT ce qui, dans la lecture d un film, ne se lit pas dans le flux de bits : les largeurs, les
// cadres, les quantums, les drapeaux. Il se resout UNE fois, a partir des trois cles que le film
// ECRIT (version de format, build, version majeure — cf. profile_table.go) et de l entree de
// catalogue de la CARTE du match, et il ne change plus.
//
// # POURQUOI IL EST IMMUABLE, ET COMMENT
//
// Les champs sont PRIVES et les accesseurs rendent des VALEURS. Un [Profile] se copie a chaque
// lecture, donc un appelant qui modifie ce qu il a recu ne modifie pas le profil du film — c est
// le meme geste que [FilmContext.ImposedLayout]. Le seul champ porteur d une tranche
// ([FilmIdentity.TypeVersions]) est CLONE par son accesseur : sans cela « immuable » serait un
// mot, pas une propriete.
//
// # CE QU IL N EST PAS ENCORE, ET C EST LE TITRE DU LOT
//
// Au lot 2.1 le profil est RESOLU et PROUVE EGAL aux globales de paquet
// ([TestProfilEgaleGlobales]), mais aucun lecteur de bits ne le lit encore : les globales sont
// toujours ecrites, et ce sont elles qui decident. C est la DOUBLE ECRITURE de D7, avec son
// kill-switch date (`replay/world_object_precision.go`). Le lot 2.2 branche les lecteurs famille
// par famille, le lot 2.3 retire les globales.
//
// # LE PROFIL NE MET AUCUN FILM DE COTE
//
// D-4 d ADR 0034 interdit de lire un film au profil du build voisin — pas de le lire du tout.
// Une cle absente de la table rend une ERREUR TYPEE ([Profile.Err], `errors.Is` sur
// [ErrUnknownFormat] / [ErrUnknownBuild]) que l appelant consulte, et le reste du profil est
// pose : les invariants et la carte ne dependent d aucune cle. Les replis existants
// (calibration MPP, largeurs d axe par defaut) continuent de tourner, comptes comme aujourd hui.

//
// # CE QUE CE FICHIER PORTE DEPUIS LE LOT 2.5.b, ET CE QUE `grammar` GARDE
//
// La couche `profile` porte le TYPE et la partie de la resolution qui lit LA TABLE et LE
// CATALOGUE ([Resoudre]). La partie qui lit LE FILM — ouvrir le `chunk_00`, y prendre la
// version de format, la version majeure et la section d identification — reste en `grammar`
// (`profile.ResolveProfile`) et appelle [Resoudre] avec les cles DEJA LUES ([ClesDuFilm]).
//
// C EST L INVERSION DE DEPENDANCE DU LOT, ET ELLE N EST PAS COSMETIQUE : tant que la lecture et
// la table vivaient dans la meme fonction, la couche `profile` aurait importe `grammar` — un
// import VERS LE HAUT que le ratchet des couches refuse (R1). Le profil ne sait pas ouvrir un
// film ; c est ce qui le garde honnete.
//
// UNE METHODE EST PARTIE AVEC LA GRAMMAIRE : `KeyframeProfile.EtatParDefautPorte` interrogeait
// la table des deserialiseurs d etat par defaut (`grammar/default_state_arch.go`) sans jamais
// lire son recepteur. Elle y est devenue la fonction `grammar.EtatParDefautPorte` — un profil
// ne sait pas quels archetypes ce depot sait decoder ; la grammaire, si.

import "errors"

// HighlightProfile : l implantation du gamertag dans un bloc d evenement de temps fort, keyee
// par la VERSION MAJEURE du film (`chunk_00+0`).
//
// LA BRANCHE QUI L APPLIQUE EST AILLEURS, ET C EST VOULU : `analysis.decodeEventBytes` est le
// seul site qui decoupe le bloc, et le dedoubler ici ferait deux grammaires a maintenir. Ce que
// le profil apporte est la CLE (la version, lue une fois au lieu d etre devinee ou repassee de
// main en main) et le NOM de l implantation qu elle selectionne. Le garde-rail
// [TestProfilHighlightEgaleLeParseur] prouve que les deux disent la meme chose.
type HighlightProfile struct {
	// MajorVersion est la cle : l u32 little-endian de `chunk_00+0`, ou
	// [FilmMajorVersionUnknown] quand le film ne porte pas son registre.
	MajorVersion int
	// Lue dit si la cle a ete LUE dans le film. Faux = version inconnue : l implantation
	// ci-dessous est le comportement historique, et l appelant doit consigner la degradation.
	Lue bool
	// Implantation nomme le decoupage que cette version selectionne.
	Implantation ImplantationGamertag
	// GamertagOffsetBytes est le premier octet du gamertag dans le bloc de 60 octets.
	GamertagOffsetBytes int
}

// KeyframeProfile : le CADRE d un record d image-cle d etat complet.
//
// LA REGLE EST `172 + etat(ti)`, JAMAIS UN NOMBRE. Les 172 bits sont fixes et relus chez
// l ecrivain (108 d en-tete par entite, plus les deux mots de taille de 32 bits que
// `FUN_142e2bfd0` lit autour de l etat par defaut) ; ce qui suit depend de l ARCHETYPE, et ce
// n est pas une largeur mais un DESERIALISEUR (`vtable[0x60]`, table `defaultStateDeserByTI`).
// Un archetype dont la grammaire n est pas resolue consomme 0 bit — repli nomme, D14.
type KeyframeProfile struct {
	// EnTeteBits : l en-tete par entite (108).
	EnTeteBits int
	// MotDeTailleBits : la largeur d UN mot de taille (32). Le cadre en porte DEUX.
	MotDeTailleBits int
}

// CadreBits rend les 172 bits du cadre : en-tete plus les deux mots de taille. Ce qui vient
// APRES depend de l archetype (`grammar.EtatParDefautPorte`).
func (k KeyframeProfile) CadreBits() int { return k.EnTeteBits + 2*k.MotDeTailleBits }

// MovementProfile : les quantums, la largeur d axe absolue et les drapeaux de queue du chemin
// de position. Toutes ces valeurs sont aujourd hui des variables de paquet de `filmdec` ; le lot
// 2.1 les RECOPIE ici avec leur provenance et PROUVE l egalite, le lot 2.2 les fait lire ici.
type MovementProfile struct {
	// Traversal est le descripteur de quantification du chemin de TRAVERSEE.
	Traversal PrecisionDescriptor
	// WorldObject est le descripteur du chemin WORLD-OBJECT (projectiles, armes au sol,
	// equipement, corps rigides), dont les largeurs sont celles de la CARTE du match. Son
	// defaut n est pas un repli neutre : c est l entree `cliffhanger` du catalogue.
	// L installateur de `replay` y pose les largeurs de la carte jouee (lot 2.2.b).
	// Provenance et preuve : ligne `Movement.WorldObject` de [TableProfil].
	WorldObject PrecisionDescriptor
	// AbsoluteAxisW est la largeur d axe uniforme du chemin ABSOLU, a defaut de table par
	// index de plage.
	AbsoluteAxisW uint
	// DeltaQuantum est le pas, en unites monde, d UN cran de position repliquee en delta.
	DeltaQuantum float32
	// DeltaAxisWidth est la largeur d axe du chemin delta axis-width.
	DeltaAxisWidth uint
	// Range est la range de dequantification par defaut du paquet. La range du MATCH vient de
	// la carte ([Profile.Map]) ; celle-ci ne sert que lorsqu aucune carte n est fournie.
	Range Vec3Range
	// FullPrecision mirroite le global de configuration `DAT_145121140`.
	FullPrecision bool
	// DeltaHasHandleTail mirroite le champ d execution `bVar16 = (precIndex != -1)`. Ce n est
	// PAS un bit du flux.
	DeltaHasHandleTail bool
	// CalibratedSkip active la calibration d i0 (saut de 47 ou 101 bits).
	CalibratedSkip bool
	// MobilityActionExtraBits est le nombre de bits supplementaires d une action de mobilite.
	MobilityActionExtraBits int
}

// SlotsProfile : la transposition d un enregistrement de slot de joueur, keyee par le BUILD
// (lot 1.5.2).
type SlotsProfile struct {
	// PersoBytes est la largeur du bloc de personnalisation `sub+0xcc0`, en octets.
	PersoBytes int
	// DeltaBits est la transposition par rapport au build de reference, en bits.
	DeltaBits int
	// Connu dit si le build est dans la table. Faux = [ErrUnknownBuild] dans [Profile.Err] et
	// les deux champs ci-dessus valent zero.
	Connu bool
}

// Profile porte TOUT ce qui ne se lit pas dans le flux de bits. Cf. l en-tete du fichier.
//
// Les champs sont PRIVES : un profil se lit par ses accesseurs, qui rendent des valeurs.
type Profile struct {
	identity  FilmIdentity
	mapEntry  MapQuantEntry
	highlight HighlightProfile
	keyframe  KeyframeProfile
	movement  MovementProfile
	slots     SlotsProfile
	mpp       MPPWidths
	format    int
	build     string
	err       error
}

// Identity rend la section 2 de `chunk_00`, CLONEE : sa table par type est une tranche, et la
// rendre telle quelle laisserait un lecteur reecrire le profil du film.
func (p Profile) Identity() FilmIdentity {
	id := p.identity
	if id.TypeVersions != nil {
		id.TypeVersions = append([]uint32(nil), id.TypeVersions...)
	}
	return id
}

// Map rend l entree de catalogue de la CARTE du match. Entree nulle quand l appelant n en a pas
// fourni (instruments, enveloppes hors production).
func (p Profile) Map() MapQuantEntry { return p.mapEntry }

// Highlight rend l implantation du gamertag des blocs d evenement de temps fort.
func (p Profile) Highlight() HighlightProfile { return p.highlight }

// Keyframe rend le cadre des records d image-cle d etat complet.
func (p Profile) Keyframe() KeyframeProfile { return p.keyframe }

// Movement rend les quantums, la largeur d axe absolue et les drapeaux de queue.
func (p Profile) Movement() MovementProfile { return p.movement }

// Slots rend la transposition d un enregistrement de slot de joueur.
func (p Profile) Slots() SlotsProfile { return p.slots }

// MPP rend le decoupage du bloc `object-multiplayer-properties`. Non valide quand la version de
// format est connue mais sa largeur indeterminee (formats 20, 21, 24, 25).
func (p Profile) MPP() MPPWidths { return p.mpp }

// FormatVersion rend la cle de FORMAT (`chunk_00+4`), ou [FilmFormatVersionUnknown].
func (p Profile) FormatVersion() int { return p.format }

// Build rend la cle de BUILD (section 2), ou la chaine vide.
func (p Profile) Build() string { return p.build }

// Err rend l erreur TYPEE des cles absentes de la table : [ErrUnknownFormat] enveloppee avec la
// version refusee, [ErrUnknownBuild] enveloppee avec le nom refuse, ou les deux jointes.
//
// ELLE N EST PAS UN REFUS DE LIRE (D-4) : le reste du profil est pose, les replis existants
// tournent et sont comptes comme aujourd hui. L appelant la journalise et continue.
func (p Profile) Err() error { return p.err }

// ResolveProfile resout le profil d un film DEJA CHARGE, sous l entree de catalogue de sa carte.
//

// ClesDuFilm : LES TROIS CLES QUE LE FILM ECRIT, DEJA LUES.
//
// C est la frontiere entre les deux moities de la resolution (lot 2.5.b). `grammar` sait ouvrir
// un `chunk_00` et y prendre ces valeurs ; ce paquet sait ce qu elles designent dans la table.
// Le passage se fait par une STRUCTURE et non par six parametres : la regle des cinq parametres
// du depot, et surtout le fait qu une cle qui s ajoutera demain (le bit `DAT_144706104` de la
// piste D2) se pose ici sans toucher aucune signature.
type ClesDuFilm struct {
	// RegistrePresent dit si le `chunk_00` a pu etre lu. Faux = bobine partielle ou fixture
	// sans registre : AUCUNE cle n est lisible, et [Resoudre] le dit dans le vocabulaire des
	// cles plutot que dans celui du fichier.
	RegistrePresent bool
	// Format est la version de format (`chunk_00+4`) TELLE QUE LE LECTEUR LA REND — donc
	// [grammar.FilmFormatVersionUnknown] quand l en-tete est trop court. Le profil ne connait
	// pas la sentinelle et n a pas a la connaitre : il range ce qu on lui donne.
	Format int
	// Majeure est la version majeure (`chunk_00+0`), meme regle, et Lue dit si elle a ete LUE.
	// Ce second champ compte : l implantation du gamertag qu une version non lue selectionne
	// est le comportement HISTORIQUE, et l appelant doit consigner la degradation.
	Majeure    int
	MajeureLue bool
	// Identite est la section 2, et IdentiteLue dit si sa lecture a abouti. Un film sans
	// section d identification (5 au cache) laisse les deux a leur zero : le build reste vide,
	// et [Profile.Err] porte [ErrUnknownBuild].
	Identite    FilmIdentity
	IdentiteLue bool
}

// Resoudre compose le profil d un film a partir de ses cles DEJA LUES et de l entree de
// catalogue de sa carte.
//
// C EST UNE FONCTION PURE de (cles, entree de carte) : aucune variable de paquet n est lue ni
// ecrite, aucun fichier n est ouvert, AUCUN OCTET DE FILM N EST TOUCHE. Elle peut donc etre
// appelee deux fois sans que les deux resultats different.
//
// `entry` nil est ACCEPTE : le profil rend alors ses invariants et sa carte nulle.
//
// LE PROFIL NE MET AUCUN FILM DE COTE (D-4 d ADR 0034) : une cle absente de la table rend une
// ERREUR TYPEE dans [Profile.Err], et le reste du profil est pose.
func Resoudre(cles ClesDuFilm, entry *MapQuantEntry) Profile {
	p := Profile{
		keyframe:  CadreParDefaut(),
		movement:  MouvementParDefaut(),
		format:    cles.Format,
		highlight: HighlightDepuisMajeure(cles.Majeure, cles.MajeureLue),
	}
	if entry != nil {
		p.mapEntry = *entry
	}
	if !cles.RegistrePresent {
		p.err = errors.Join(ErreurFormatInconnu(cles.Format), ErreurBuildInconnu(""))
		return p
	}
	var errs []error
	if w, connu := MPPPourFormat(p.format); connu {
		p.mpp = w
	} else {
		errs = append(errs, ErreurFormatInconnu(p.format))
	}
	if cles.IdentiteLue {
		p.identity, p.build = cles.Identite, cles.Identite.Build
	}
	if octets, connu := PersonnalisationOctets(p.build); connu {
		p.slots = SlotsProfile{PersoBytes: octets, DeltaBits: PersoDeltaBits(octets), Connu: true}
	} else {
		errs = append(errs, ErreurBuildInconnu(p.build))
	}
	p.err = errors.Join(errs...)
	return p
}

// HighlightDepuisMajeure compose l implantation du gamertag depuis la version majeure.
//
// EXPORTEE AU LOT 2.5.b (elle s appelait `highlightDuProfil`) : les trois sites qui decoupent un
// bloc d evenement de temps fort tiennent des octets, pas un profil, et leur porte etroite
// (`grammar.HighlightProfileFromHeader`) vit donc de l autre cote de la frontiere. C est LA MEME
// fonction qui compose la valeur ici et la-bas — pas deux tables.
func HighlightDepuisMajeure(majeure int, lue bool) HighlightProfile {
	impl, off := implantationDuGamertag(majeure)
	return HighlightProfile{
		MajorVersion:        majeure,
		Lue:                 lue,
		Implantation:        impl,
		GamertagOffsetBytes: off,
	}
}

// CadreParDefaut rend le CADRE d un record d image-cle d etat complet.
//
// C est la SOURCE UNIQUE des deux largeurs, et elle sert aux DEUX bouts depuis le lot 2.2.c :
// [Resoudre] la pose dans le profil, et `grammar.LecteurSur` la pose sur le lecteur — les
// lecteurs d etat complet ne lisent donc plus les constantes du paquet, ils lisent le profil.
// La regle reste `172 + etat(ti)`, jamais un nombre : les 172 se composent ici.
//
// LES DEUX LARGEURS VIVENT ICI DEPUIS LE LOT 2.5.b : elles ont DEMENAGE de
// `grammar/keyframe_fullstate_loop.go`, avec leur provenance, parce qu elles n avaient plus
// qu un lecteur — cette fonction.
func CadreParDefaut() KeyframeProfile {
	return KeyframeProfile{
		EnTeteBits:      KeyframeEnTeteBits,
		MotDeTailleBits: KeyframeMotDeTailleBits,
	}
}

// DESCENDUES DE `grammar/keyframe_fullstate_loop.go` AU LOT 2.5.b, AVEC LEUR PROVENANCE. Elles
// n avaient plus qu un lecteur — la composition du cadre ci-dessus — et le lecteur d etat complet
// prend le cadre sur le PROFIL depuis le lot 2.2.c. Les recopier de ce cote aurait fait deux
// verites pour une largeur du jeu (CLAUDE.md regle 6) ; elles ont donc DEMENAGE, exportees parce
// que la doc de `keyframe_fullstate_loop.go` les cite.
// KeyframeEnTeteBits est l'en-tete PAR ENTITE d'un etat complet tel que
// `FUN_142e2bfd0` le lit : `R(32)` id, `R(32)` typeIndex, `R(32)`, `R(4)`, `R(8)`.
//
// Il ne CONTREDIT pas l'en-tete de 64 bits `[id:32][field:26][ti:6]` valide par R3/R5 : le
// balayeur oracle (`kfValidAnchor`) n'accepte une ancre que si le mot de 32 bits a `q+32`
// vaut moins de 50, ce qui veut dire `field26 == 0` sous une lecture et `typeIndex < 50`
// sous l'autre. Les deux sont INDISCERNABLES sur les ancres acceptees, et les 6 bits de
// `typeIndex` lus en `+58` valent la meme chose dans les deux cas.
const KeyframeEnTeteBits = 108

// KeyframeMotDeTailleBits est la largeur des deux mots de taille que `FUN_142e2bfd0` lit
// autour de l'etat par defaut (`n1` avant, `n2` apres) : ce sont des comptes testes `> 0`,
// pas des longueurs de saut.
const KeyframeMotDeTailleBits = 32

// MouvementParDefaut rend les quantums, la largeur d axe absolue et les drapeaux de queue.
//
// LES VALEURS SONT DES LITTERAUX, PAS UNE LECTURE DES GLOBALES, et c est le point du lot 2.1 :
// un profil qui recopierait `DeltaQuantum` au moment de sa resolution mirroiterait ce qu une
// calibration vient d y ecrire, au lieu de dire ce que la GRAMMAIRE pose. Leur provenance est en
// table ([tableProfilInvariants]).
func MouvementParDefaut() MovementProfile {
	return MovementProfile{
		Traversal:               PrecisionDescriptor{IndexW: 1, AxisW: [3]uint{6, 6, 6}},
		WorldObject:             PrecisionDescriptor{IndexW: 1, AxisW: [3]uint{13, 13, 14}},
		AbsoluteAxisW:           14,
		DeltaQuantum:            0.01383,
		DeltaAxisWidth:          14,
		Range:                   QuantRangeCEBiped,
		FullPrecision:           false,
		DeltaHasHandleTail:      false,
		CalibratedSkip:          false,
		MobilityActionExtraBits: 0,
	}
}
