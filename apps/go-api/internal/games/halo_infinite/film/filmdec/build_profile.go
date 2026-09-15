package filmdec

// build_profile.go — LE PROFIL PAR BUILD : TOUT CE QUI VARIE D UN BUILD DU JEU A L AUTRE.
//
// UN SEUL LIEU (renomme depuis `player_table_profile.go` le 2026-09-15, lot 1.9.1 bis pas 3).
// Le fichier portait la largeur du bloc de personnalisation (lot 1.5.2) ; il porte desormais
// AUSSI les largeurs du bloc `object-multiplayer-properties`, parce que c est le meme objet :
// une donnee immuable par build, avec sa provenance ligne a ligne. Deux tables par build dans
// deux fichiers auraient diverge au premier build ajoute.
//
// # LA PROVENANCE EST UNE COLONNE, PAS UNE POLITESSE (regle du pilote, 2026-09-15)
//
// Une valeur de profil porte D OU ELLE VIENT, et il y a exactement deux provenances :
//
//	RELU     l executable de CE build est ouvert, la fonction est citee. Ici : `HI_1_12_0` et
//	         `HI_1_13_0`, dont l exe desassemble est le build.
//	MESURE   l executable de ce build n est PAS disponible. La valeur se MESURE alors sur le
//	         film par un oracle INTERNE au film (`n2` constant, fermeture), et la ligne porte
//	         l oracle et son score. Ce n est pas « relu chez l ecrivain », et le dire serait
//	         faux : c est l esprit de D-3/D-4 d ADR 0034, ou le profil est une DONNEE avec sa
//	         provenance, pas une inference deguisee.
//
// # POURQUOI CE FICHIER EST UNE DONNEE DE PROFIL, ET PAS UNE MESURE
//
// D-3 d'ADR 0034 : le profil est resolu une fois, immuable, et sa cle est le BUILD lu en clair
// dans la section 2 de `chunk_00`. Une largeur par build est une donnee de profil — elle ne
// s'auto-detecte pas sur le film. Le calibrage lu sur le film reste possible et il reste
// PRECIEUX, mais comme CONTROLE : `PlayerTableReport` compte les accords et les contradictions
// entre ce que le profil dit et ce que le film mesure (player_table.go). Un calibrage qui
// decide serait exactement l'inference que D-3 interdit.
//
// D-4 : un build absent de cette table donne [ErrUnknownBuild] et un compteur expvar par build.
// Il n'est JAMAIS lu au profil du build le plus proche. Les 5 films du cache sans section
// d'identification se comportent comme `HI_1_4_1` (transposition +1 600 bits mesuree sur les
// cinq) — se comporter comme n'est pas etre, et ils sont donc mis de cote.
//
// # LA MESURE QUI DONNE CES QUATRE VALEURS
//
// L'ordre d'ecriture de `FUN_1407edea8` place TOUS les champs de longueur variable AVANT le
// gamertag et tous les champs de largeur fixe APRES. Le bloc de queue `sub+0x1400` porte le
// MEME gamertag en UTF-16 petit-boutiste : il est donc localisable dans le flux sans rien
// supposer de ce qui le precede, ce qui coupe l'enregistrement en deux moities mesurables
// separement (`residus_slots_research_test.go`, `NOTE_RESIDUS_CHUNK00_2026-09-13.md` section 2).
// Resultat, sur un film par build : la moitie APRES est invariante sur les sept builds, la plus
// longue suite de bits NULS de la moitie AVANT vaut exactement le bloc de personnalisation, et
// le RESTE hors bloc nul vaut **270 bits sur les huit groupes** — c'est-a-dire qu'AUCUN autre
// champ n'a change de largeur. La transposition par build est ce bloc, et rien d'autre.

import (
	"fmt"
	"strings"
	"unicode"

	"levelup/go-api/internal/analysis/filmsource"
)

// LES SEPT BUILDS CONNUS DU CACHE, NOMMES UNE FOIS. Ils servent DEUX fois chacun — la table
// de personnalisation et la table MPP — et la regle 6 du depot veut qu a la troisieme copie on
// centralise. Les TESTS, eux, gardent leurs litteraux : un test qui relit la constante qu il
// verifie ne verifie rien.
const (
	buildHI1131 = "HI_1_13_0"
	buildHI1120 = "HI_1_12_0"
	buildHI1110 = "HI_1_11_0"
	buildHI1100 = "HI_1_10_0"
	buildHI190  = "HI_1_9_0"
	buildHI180  = "HI_1_8_0"
	buildHI141  = "HI_1_4_1"
)

// ErrUnknownBuild : le build du film n'est pas dans la table de profil ci-dessous.
//
// L'erreur rendue par [ReadPlayerTable] enveloppe cette sentinelle avec le nom du build, pour
// qu'un journal dise LEQUEL : `errors.Is(err, ErrUnknownBuild)` reste vrai.
const ErrUnknownBuild = chunk00Error("filmdec: build absent de la table de profil")

// persoOctetsRef : la largeur du bloc `sub+0xcc0` sur le build de l'executable desassemble.
// Elle sert de REFERENCE a la transposition : un build dont le bloc est plus court decale
// l'enregistrement d'autant, bit pour bit (cf. persoDeltaBits).
const persoOctetsRef = 1852

// personnalisationOctets rend la largeur, en octets, du bloc de personnalisation `sub+0xcc0`
// pour un build donne. UN COMMENTAIRE DE PROVENANCE PAR LIGNE : une valeur sans preuve n'est
// pas une valeur de profil (D-3, regle 1).
func personnalisationOctets(build string) (int, bool) {
	switch build {
	case buildHI1131:
		// 1 852 = 0x73C, LU CHEZ L'ECRIVAIN : `FUN_1407edea8` ecrit `LEA R8,[RDI+0xcc0] ;
		// R9D=0x39e0` (14 816 bits). Fermeture independante chez le serialiseur DELTA
		// `FUN_1407ec27c`, qui ecrit la MEME structure champ par champ : `actionPose` a
		// `+0x738`, plus 4 = `0x73C`. Ghidra, 2026-09-12 — c'est le build de l'exe desassemble.
		return 1852, true
	case buildHI1120:
		// 1 852 aussi, et c'est MESURE, pas suppose : transposition `mesure - predit` de 0 bit,
		// valeur unique sur les 146 films `HI_1_12_0` du cache (2026-09-14, oracle
		// `TestResidusSlotChaineCorpus`). La bascule du bloc se fait entre `HI_1_11_0` et
		// `HI_1_12_0`, au meme endroit que celle de `n2` dans la trame.
		return 1852, true
	case buildHI1110:
		// 1 492 = 1 852 - 360. Transposition mesuree -2 880 bits (= -360 octets), valeur unique
		// sur les 39 films `HI_1_11_0` du cache ; la coupe autour du bloc de queue sur
		// `00ba2e1c` place ces 360 octets DANS le bloc nul, RESTE hors bloc nul inchange a
		// 270 bits. 360 = 10 x 0x24, le pas d'une attache d'armure (`FUN_1407ebf44`) — appui,
		// pas preuve : sans executable de ce build, la CAUSE reste une hypothese, la LARGEUR
		// est une mesure.
		return 1492, true
	case buildHI1100:
		// 1 492, transposition -2 880 bits, valeur unique sur les 26 films `HI_1_10_0` du
		// cache ; coupe verifiee sur `084a804d` (2026-09-13, note section 2.1).
		return 1492, true
	case buildHI190:
		// 1 312 = 1 852 - 540. Transposition -4 320 bits, valeur unique sur l'unique film
		// `HI_1_9_0` du cache (`11de8353`), coupe verifiee sur ce film. 540 = 15 x 0x24.
		return 1312, true
	case buildHI180:
		// 1 312, transposition -4 320 bits, valeur unique sur les 10 films `HI_1_8_0` du
		// cache ; coupe verifiee sur `0a247154`.
		return 1312, true
	case buildHI141:
		// 2 052 = 1 852 + 200. Transposition +1 600 bits, valeur unique sur l'unique film
		// `HI_1_4_1` du cache (`a521164d`), coupe verifiee sur ce film. CE +200 NE FERME SUR
		// AUCUN PAS CONNU (ni 0x24 = 36, ni 0x58 = 88) : la largeur est mesuree, sa cause n'est
		// pas etablie, et elle est au registre des reports du plan.
		return 2052, true
	}
	return 0, false
}

// persoDeltaBits rend la transposition d'un enregistrement de slot pour une largeur de bloc de
// personnalisation donnee : la difference avec la reference, en bits. C'est la SEULE constante
// par laquelle la grammaire du slot se transpose d'un build a l'autre.
func persoDeltaBits(octets int) int { return (octets - persoOctetsRef) * 8 }

// unknownBuildCounterName rend le nom expvar du compteur de build inconnu.
//
// NOMMAGE (ADR 0009) : `<categorie>_<sous_cle>` en snake_case. Le plan ecrit
// `filmdec.unknown_build.<build>` ; la forme physique du depot est
// `filmdec_unknown_build_<build>`, parce que c'est la convention que tout `/debug/vars` de ce
// service suit deja (`killsource_*`, `replay_artifact_*`, `filmdec_keyframe_ti12_*`).
//
// Le build entre dans le nom EN MINUSCULES et purge de tout ce qui n'est ni lettre, ni chiffre,
// ni `_` : un nom de build vient du film, donc d'une source que ce processus ne controle pas, et
// un nom de compteur bati sur une chaine arbitraire salirait `/debug/vars`. Un build VIDE (film
// sans section d'identification) se compte sous `sans_section` — il y en a 5 au cache.
func unknownBuildCounterName(build string) string {
	nettoye := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_':
			return r
		case unicode.IsUpper(r) && r < unicode.MaxASCII:
			return unicode.ToLower(r)
		default:
			return '_'
		}
	}, build)
	if nettoye == "" {
		nettoye = "sans_section"
	}
	return "filmdec_unknown_build_" + nettoye
}

// UnknownBuildExpvarPairs rend le compteur a publier quand [ReadPlayerTable] refuse un film pour
// build inconnu (D-4 d'ADR 0034 : erreur typee ET compteur, le film est mis de cote).
//
// `filmdec` NOMME ses compteurs et ne depend PAS d'`internal/observability` : meme patron que
// `KillSourceHealth.ExpvarPairs` (cable par `killcollector`) et que
// `NavpointRadialScan.KeyframeExpvarPairs` (cable par `replay.decodeFilmBombReads`). C'est ce
// qui garde le decodeur sans dependance interne et cette fonction testable sans expvar.
//
// LE CABLEUR N'EXISTE PAS ENCORE, ET C'EST DATE : le lot 1.5 ne livre que des LECTEURS, sans
// consommateur ; le premier appelant de production de [ReadPlayerTable] est le registre
// d'identite du lot 1.6, et c'est lui qui publiera cette paire. Si ce lot passe sans qu'elle
// soit cablee, c'est un defaut de 1.6, pas un compteur « au cas ou » : le refus d'un film serait
// alors invisible en production, ce que D-4 interdit.
func UnknownBuildExpvarPairs(build string) []ExpvarPair {
	return []ExpvarPair{{Name: unknownBuildCounterName(build), Value: 1}}
}

// erreurBuildInconnu enveloppe [ErrUnknownBuild] avec le nom du build refuse.
func erreurBuildInconnu(build string) error {
	return fmt.Errorf("%w : %q", ErrUnknownBuild, build)
}

// mppWidthsPourBuild rend le decoupage du bloc `object-multiplayer-properties` (FUN_14080cfe8)
// pour un build donne. UN COMMENTAIRE DE PROVENANCE PAR LIGNE, comme pour la personnalisation.
//
// # POURQUOI CETTE TABLE EXISTE, ET CE QU ELLE REMPLACE
//
// Le bloc MPP vit dans l etat par defaut de `ti=36, 37, 38, 39, 42, 43` — donc AVANT le premier
// composant. Sa largeur decale tout le record. Le depot en lisait UNE seule pour les sept
// builds, et compensait la variabilite par une CALIBRATION sur le film (`CalibrateMPPWidths`,
// installee en production par `equipment_placements.go` et `build_ground_weapons.go`). C etait
// le profil manquant, ecrit en heuristique : la calibration devient un CONTROLE.
//
// # LA BASCULE EST A `HI_1_12_0`, ET ELLE EST MESUREE
//
// Balayage 16 x 16 des deux largeurs contre l oracle `n2` (le mot de taille que
// `FUN_142e2bfd0` lit APRES l etat par defaut : une taille de tampon, donc CONSTANTE par
// archetype et par build ; un decoupage faux le fait atterrir sur du bruit). Part des records
// dont `n2` prend la valeur modale, `ti=37` — `ti=38` et `ti=42` donnent la MEME coupure :
//
//	HI_1_4_1   8/3 -> 0,995    9/5 -> 0,304
//	HI_1_8_0   8/3 -> 0,988    9/5 -> 0,522
//	HI_1_9_0   8/3 -> 0,990    9/5 -> 0,492
//	HI_1_10_0  8/3 -> 0,993    9/5 -> 0,432
//	HI_1_11_0  8/3 -> 0,996    9/5 -> 0,472
//	HI_1_12_0  9/5 -> 0,949
//	HI_1_13_0  9/5 -> 1,000
//
// LA VERSION MAJEURE DU FILM NE DISCRIMINE PAS : `e5adf7b2` (HI_1_11_0) et `bcb6d393`
// (HI_1_12_0) portent tous deux `v=40` et tombent de part et d autre. Le cardinal de la table
// par type, lui, suit la coupure (116/121/121/121/122 contre 123/123) — mais c est un PROXY,
// garde comme controle, jamais comme cle : la cle est le BUILD.
func mppWidthsPourBuild(build string) (MPPWidths, bool) {
	switch build {
	case buildHI1131:
		// RELU — c est le build de l executable desassemble. `FUN_141fd72c0` lit le champ de
		// tete par un litteral (`141fd72de : ADD dword ptr [RCX + 0x2c],0x9`) et le champ
		// d index est un `R(5)` inline de `FUN_14080cfe8`. Aucune branche de version dans le
		// bloc : son seul `if` runtime (`DAT_145121140 == 1`) ne consomme aucun bit.
		// L oracle `n2` le confirme independamment : part modale 1,000 sur les trois archetypes.
		return MPPWidths{Lead: 9, Index: 5}, true
	case buildHI1120:
		// RELU pour la grammaire (meme executable que `HI_1_13_0` : la bascule est SOUS ce
		// build, pas au-dessus), MESURE pour la confirmation : part modale 0,949 (ti=37),
		// 0,998 (ti=38), 0,961 (ti=42) sur `bcb6d393`.
		return MPPWidths{Lead: 9, Index: 5}, true
	case buildHI1110, buildHI1100, buildHI190, buildHI180, buildHI141:
		// INDETERMINE, ET LES DEUX ORACLES SE CONTREDISENT — voir l en-tete de section
		// ci-dessous. La largeur N EST PAS POSEE : `MPPWidths{}` n est pas valide, donc
		// `InstallBuildProfileMPP` n installe RIEN et le decodeur garde son defaut. Le build
		// est CONNU (ce n est pas `ErrUnknownBuild`), c est sa largeur MPP qui ne l est pas.
		return MPPWidths{}, true
	}
	return MPPWidths{}, false
}

// BuildProfile porte TOUT ce que le profil d un build donne. Il se resout UNE FOIS, a partir du
// nom de build lu en clair dans la section 2 de `chunk_00`, et il est IMMUABLE (D-3 d ADR 0034).
type BuildProfile struct {
	// Build est la cle : le nom en clair, tel que `FilmIdentity.Build` le rend.
	Build string
	// PersoBytes est la largeur du bloc de personnalisation (lot 1.5.2).
	PersoBytes int
	// MPP est le decoupage du bloc `object-multiplayer-properties` (lot 1.9.1 bis).
	MPP MPPWidths
}

// BuildProfileFor rend le profil d un build, ou [ErrUnknownBuild] enveloppe avec son nom.
//
// D-4 : un build absent de la table n est JAMAIS lu au profil du build le plus proche, et jamais
// calibre en silence. L appelant publie le compteur ([UnknownBuildExpvarPairs]) et met le film
// de cote.
func BuildProfileFor(build string) (BuildProfile, error) {
	perso, okP := personnalisationOctets(build)
	mpp, okM := mppWidthsPourBuild(build)
	if !okP || !okM {
		return BuildProfile{}, erreurBuildInconnu(build)
	}
	return BuildProfile{Build: build, PersoBytes: perso, MPP: mpp}, nil
}

// BuildProfileFromFilm resout le profil d un film DEJA CHARGE : il lit la section 2 de
// `chunk_00` et rend le profil de son build.
//
// Un film SANS section d identification (les cinq du cache : `03af54c3`, `13b00e35`, `47d20b5d`,
// `50247b26`, `a349fea8`) rend [ErrUnknownBuild] avec un build VIDE — et le compteur
// [UnknownBuildExpvarPairs] le range sous `sans_section`. Se comporter comme `HI_1_4_1` n est
// pas etre `HI_1_4_1` : leur profil se declare au REGISTRE des replis (1.9.0), pas ici.
func BuildProfileFromFilm(f *filmsource.Film) (BuildProfile, error) {
	reg, ok := FilmRegistryChunk(f)
	if !ok {
		return BuildProfile{}, erreurBuildInconnu("")
	}
	id, err := ReadFilmIdentity(reg)
	if err != nil {
		return BuildProfile{}, erreurBuildInconnu("")
	}
	return BuildProfileFor(id.Build)
}

// InstallBuildProfileMPP installe les largeurs MPP du profil et rend leur restauration.
//
// L APPELANT DOIT DETENIR [LockProcessDecode] : `SetMPPWidths` ecrit des globaux de paquet, le
// meme contrat que `replay.installWorldObjectPrecision`. Un film dont le build est inconnu ne
// change RIEN — le defaut de paquet reste en place et l erreur est rendue a l appelant, qui
// decide (mettre le film de cote, ou compter un repli nomme).
func InstallBuildProfileMPP(f *filmsource.Film) (func(), error) {
	p, err := BuildProfileFromFilm(f)
	if err != nil {
		return func() {}, err
	}
	if !p.MPP.Valid() {
		// Build CONNU dont la largeur MPP est INDETERMINEE (les cinq builds <= HI_1_11_0) :
		// on n installe rien plutot qu un decoupage nul, qui ne lirait aucune identite du tout.
		return func() {}, nil
	}
	prev := SetMPPWidths(p.MPP)
	return func() { SetMPPWidths(prev) }, nil
}

// POURQUOI LES CINQ BUILDS ANCIENS N ONT PAS DE LARGEUR MPP POSEE (2026-09-15).
//
// Les deux oracles internes au film SE CONTREDISENT, et il faut le dire plutot que de choisir.
//
//	L ORACLE `n2` designe `8/3` sur ces cinq builds, nettement : part modale 0,988 a 0,996
//	contre 0,304 a 0,522 pour `9/5`. Et il est VALIDE la ou l ecrivain est connu — sur
//	`fb1a1a72` (HI_1_13_0, le build de l executable desassemble) il rend 1,000 a `9/5`,
//	c est-a-dire exactement ce que `FUN_141fd72c0` ecrit.
//
//	L ORACLE FERMETURE dit autre chose, et il est DISQUALIFIE ICI, par sa propre mesure :
//	sur ce meme `fb1a1a72`, la fermeture prefere `10/5` (140 records) a `9/5` (11) — donc
//	elle contredit l ecrivain la ou l ecrivain est certain. Sur `bcb6d393` (HI_1_12_0) elle
//	prefere `8/3` (69) a `9/5` (33), meme desaccord. A moins de 5 % de fermeture, maximiser
//	un compte de fermetures revient a chercher des coincidences : la fermeture ne devient un
//	oracle de largeur que pres de 100 %.
//
// POSER `8/3` FERAIT DESCENDRE LE RATCHET DE COUVERTURE de 246 a 182 records fermes (mesure du
// 2026-09-15, `TestE191cPrefixeObjet`), et D14 interdit de le faire descendre. Poser `9/5`
// partout serait affirmer une largeur que `n2` refute sur ces builds. Les deux gestes seraient
// une decision deguisee en mesure : la table laisse donc la case VIDE, ce qui est l etat reel
// de la connaissance, et l arbitrage remonte au pilote.
//
// CE QUI LEVERAIT L INDETERMINATION, par ordre de force : l executable d un build <= HI_1_11_0
// (alors la ligne devient RELU) ; ou une fermeture qui vaille quelque chose sur ces archetypes,
// c est-a-dire la suite du chantier — l oracle fermeture redeviendra utilisable quand la marche
// fermera, et il tranchera alors sans ambiguite.

// CORRECTION DU 2026-09-16 — LES TROIS BITS NE SONT PAS DANS LES LARGEURS MPP.
//
// Fait tranche par l utilisateur (mecanique de jeu, il fait autorite) : « les films sont
// independants des builds ; ils sont enregistres a l instant T et jamais touches ensuite ; le
// film ne depend que de lui-meme pour expliquer au mode Theater comment le lire ». L executable
// OUVERT lit donc les films anciens, et tout ce qui varie est ECRIT DANS LE FILM.
//
// CE QUE LA RELECTURE A ALORS ETABLI, ET QUI CORRIGE LE PAS 3 :
//
//	`FUN_14080cfe8` N A AUCUNE BRANCHE DE VERSION. Toutes ses largeurs sont des litteraux
//	(9, 32, 1[+32], 1[+18], 2, 5, 3, la boucle, la queue) et son seul `if` runtime
//	(`DAT_145121140 == 1`) ne consomme AUCUN bit. `FUN_141fd72c0` (le champ de tete, R(9))
//	n a qu UN SEUL appelant, ce bloc. Et `FUN_1428e1c0c`, l accesseur de version du film, n a
//	que six sites d appel, AUCUN dans la chaine des etats par defaut.
//
// Le bloc MPP lit donc les MEMES bits pour tous les films. Les trois bits d ecart que l oracle
// `n2` mesure sur les films anciens sont AILLEURS dans l etat par defaut — le balayage les avait
// attribues a `lead`/`index` parce que c etaient les deux seules molettes qu il avait.
//
// CE QUI RESTE VRAI : `n2` mesure bien que l etat par defaut des films anciens est plus court de
// trois bits, et la case de ces builds reste donc VIDE — non parce que la largeur MPP serait
// inconnue (elle ne l est pas : 9/5, relue), mais parce que l ENDROIT des trois bits ne l est
// pas. Poser 9/5 pour ces builds retirerait la calibration sans avoir explique l ecart.
//
// CE QUE LA PROCHAINE SESSION DOIT CHERCHER, ET OU : la cle est dans le film, et la mesure la
// designe — la table par type de la section 2, ALIGNEE PAR LA FIN, porte DOUZE positions dont la
// version vaut 1 sur les cinq films courts et 2 a 5 sur les deux autres (alignee par le DEBUT
// elle n en porte aucune : les types s ajoutent en tete). Le discriminant est l une de ces douze
// ; la fonction qui la consulte est dans l executable ouvert, et elle reste a trouver.
