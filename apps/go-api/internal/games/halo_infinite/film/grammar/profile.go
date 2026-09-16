package grammar

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
// ([profile.FilmIdentity.TypeVersions]) est CLONE par son accesseur : sans cela « immuable » serait un
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

import (
	"errors"

	"levelup/go-api/internal/games/halo_infinite/film/profile"
	"levelup/go-api/internal/games/halo_infinite/film/source"
)

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
	Implantation implantationGamertag
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
// APRES depend de l archetype ([KeyframeProfile.EtatParDefautPorte]).
func (k KeyframeProfile) CadreBits() int { return k.EnTeteBits + 2*k.MotDeTailleBits }

// EtatParDefautPorte dit si la grammaire du depot porte le deserialiseur d etat par defaut de
// l archetype `ti`. Faux = 0 bit consomme, et c est soit un STUB du jeu, soit un REPLI
// (cf. l en-tete de default_state_arch.go, qui les distingue population par population).
func (k KeyframeProfile) EtatParDefautPorte(ti uint32) bool {
	if ti == BipedTypeIndex {
		return true // traite a part par TraverseEntity (consumeBipedDefaultState)
	}
	_, ok := defaultStateDeserByTI[ti]
	return ok
}

// MovementProfile : les quantums, la largeur d axe absolue et les drapeaux de queue du chemin
// de position. Toutes ces valeurs sont aujourd hui des variables de paquet de `filmdec` ; le lot
// 2.1 les RECOPIE ici avec leur provenance et PROUVE l egalite, le lot 2.2 les fait lire ici.
type MovementProfile struct {
	// Traversal est le descripteur de quantification du chemin de TRAVERSEE.
	Traversal profile.PrecisionDescriptor
	// WorldObject est le descripteur du chemin WORLD-OBJECT (projectiles, armes au sol,
	// equipement, corps rigides), dont les largeurs sont celles de la CARTE du match. Son
	// defaut n est pas un repli neutre : c est l entree `cliffhanger` du catalogue.
	// L installateur de `replay` y pose les largeurs de la carte jouee (lot 2.2.b).
	// Provenance et preuve : ligne `Movement.WorldObject` de [TableProfil].
	WorldObject profile.PrecisionDescriptor
	// AbsoluteAxisW est la largeur d axe uniforme du chemin ABSOLU, a defaut de table par
	// index de plage.
	AbsoluteAxisW uint
	// DeltaQuantum est le pas, en unites monde, d UN cran de position repliquee en delta.
	DeltaQuantum float32
	// DeltaAxisWidth est la largeur d axe du chemin delta axis-width.
	DeltaAxisWidth uint
	// Range est la range de dequantification par defaut du paquet. La range du MATCH vient de
	// la carte ([Profile.Map]) ; celle-ci ne sert que lorsqu aucune carte n est fournie.
	Range profile.Vec3Range
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
	identity  profile.FilmIdentity
	mapEntry  MapQuantEntry
	highlight HighlightProfile
	keyframe  KeyframeProfile
	movement  MovementProfile
	slots     SlotsProfile
	mpp       profile.MPPWidths
	format    int
	build     string
	err       error
}

// Identity rend la section 2 de `chunk_00`, CLONEE : sa table par type est une tranche, et la
// rendre telle quelle laisserait un lecteur reecrire le profil du film.
func (p Profile) Identity() profile.FilmIdentity {
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
func (p Profile) MPP() profile.MPPWidths { return p.mpp }

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
// C EST UNE FONCTION PURE de (octets du film, entree de carte) : aucune variable de paquet n est
// lue ni ecrite, aucun fichier n est ouvert. Elle peut donc etre appelee deux fois sans que les
// deux resultats different — ce qui est exactement ce que la double ecriture du lot 2.1 exige,
// et ce que le lot 2.2 supprimera en ne la faisant appeler qu une fois par cuisson.
//
// `film` nil et `entry` nil sont ACCEPTES : le profil rend alors ses invariants, sa carte nulle,
// et [Profile.Err] porte les cles manquantes.
func ResolveProfile(film *source.Film, entry *MapQuantEntry) Profile {
	p := Profile{
		keyframe: cadreDuProfil(),
		movement: mouvementDuProfil(),
		format:   FilmFormatVersionUnknown,
	}
	if entry != nil {
		p.mapEntry = *entry
	}
	reg, ok := FilmRegistryChunk(film)
	if !ok {
		// Bobine partielle ou fixture sans `chunk_00` : AUCUNE cle n est lisible. Les trois
		// erreurs le disent, dans le vocabulaire des cles et non dans celui du fichier.
		p.highlight = highlightDuProfil(FilmMajorVersionUnknown, false)
		p.err = errors.Join(erreurFormatInconnu(FilmFormatVersionUnknown), erreurBuildInconnu(""))
		return p
	}
	p.highlight = HighlightProfileFromHeader(reg)
	format, formatLu := FilmFormatVersionFromHeader(reg)
	if formatLu {
		p.format = format
	}
	var errs []error
	if w, connu := mppWidthsPourFormat(p.format); connu {
		p.mpp = w
	} else {
		errs = append(errs, erreurFormatInconnu(p.format))
	}
	// L IDENTITE EST LUE APRES LE FORMAT, et l ordre compte : un `chunk_00` sans section
	// d identification (5 films du cache) porte quand meme sa version de format. Lire le build
	// d abord ferait perdre le format de ces cinq-la.
	id, errID := ReadFilmIdentity(reg)
	if errID == nil {
		p.identity, p.build = id, id.Build
	}
	if octets, connu := personnalisationOctets(p.build); connu {
		p.slots = SlotsProfile{PersoBytes: octets, DeltaBits: persoDeltaBits(octets), Connu: true}
	} else {
		errs = append(errs, erreurBuildInconnu(p.build))
	}
	p.err = errors.Join(errs...)
	return p
}

// HighlightProfileOfFilm rend l implantation du gamertag d un film DEJA CHARGE : la seule part
// du profil dont les lecteurs de temps forts aient besoin.
//
// POURQUOI UNE PORTE ETROITE, ET PAS [ResolveProfile] : les trois sites qui decoupent un bloc
// d evenement de temps fort n ont pas de carte, et deux d entre eux n ont pas de film complet.
// Leur faire resoudre le profil ENTIER — donc lire la section d identification et la table par
// type — couterait une analyse de registre de plus par appel pour une valeur qui tient dans les
// quatre premiers octets. La VALEUR est la meme : c est la meme fonction qui la compose ici et
// dans [ResolveProfile], et [Profile.Highlight] la rend a l identique.
func HighlightProfileOfFilm(f *source.Film) HighlightProfile {
	reg, ok := FilmRegistryChunk(f)
	if !ok {
		return highlightDuProfil(FilmMajorVersionUnknown, false)
	}
	return HighlightProfileFromHeader(reg)
}

// HighlightProfileFromHeader rend l implantation du gamertag depuis un `chunk_00` DECOMPRESSE.
//
// C est la forme des appelants qui tiennent des octets bruts et pas un film (le backfill du flux
// de medailles lit son registre au cache). Un chunk trop court rend l implantation historique
// avec [HighlightProfile.Lue] a faux : la degradation se NOMME, elle ne se tait pas.
func HighlightProfileFromHeader(chunk0 []byte) HighlightProfile {
	majeure, lue := FilmMajorVersionFromHeader(chunk0)
	return highlightDuProfil(majeure, lue)
}

// highlightDuProfil compose l implantation du gamertag depuis la version majeure.
func highlightDuProfil(majeure int, lue bool) HighlightProfile {
	impl, off := implantationDuGamertag(majeure)
	return HighlightProfile{
		MajorVersion:        majeure,
		Lue:                 lue,
		Implantation:        impl,
		GamertagOffsetBytes: off,
	}
}

// cadreDuProfil rend le CADRE d un record d image-cle d etat complet.
//
// C est la SOURCE UNIQUE des deux largeurs, et elle sert aux DEUX bouts depuis le lot 2.2.c :
// [ResolveProfile] la pose dans le profil, et [LecteurSur] la pose sur le lecteur — les
// lecteurs d etat complet ne lisent donc plus les constantes du paquet, ils lisent le profil.
// La regle reste `172 + etat(ti)`, jamais un nombre : les 172 se composent ici.
func cadreDuProfil() KeyframeProfile {
	return KeyframeProfile{
		EnTeteBits:      keyframeFullStateHeaderBits,
		MotDeTailleBits: keyframeFullStateSizeBits,
	}
}

// mouvementDuProfil rend les quantums, la largeur d axe absolue et les drapeaux de queue.
//
// LES VALEURS SONT DES LITTERAUX, PAS UNE LECTURE DES GLOBALES, et c est le point du lot : un
// profil qui recopierait `DeltaQuantum` au moment de sa resolution mirroiterait ce qu une
// calibration vient d y ecrire, au lieu de dire ce que la GRAMMAIRE pose. Leur provenance est en
// table ([tableProfilInvariants]) et leur egalite avec les globales AU REPOS est prouvee par
// [TestProfilEgaleGlobales] — c est ce qui autorise le lot 2.2 a basculer les lecteurs.
func mouvementDuProfil() MovementProfile {
	return MovementProfile{
		Traversal:               profile.PrecisionDescriptor{IndexW: 1, AxisW: [3]uint{6, 6, 6}},
		WorldObject:             profile.PrecisionDescriptor{IndexW: 1, AxisW: [3]uint{13, 13, 14}},
		AbsoluteAxisW:           14,
		DeltaQuantum:            0.01383,
		DeltaAxisWidth:          14,
		Range:                   profile.QuantRangeCEBiped,
		FullPrecision:           false,
		DeltaHasHandleTail:      false,
		CalibratedSkip:          false,
		MobilityActionExtraBits: 0,
	}
}
