package grammar

// localisateur.go — LE LOCALISATEUR UNIQUE DE LA BOUCLE DE RECORDS D UN PAQUET A EVENEMENTS.
//
// Dans un paquet delta qui porte une liste d evenements (bit 1 du payload), la boucle de records
// ne commence pas a l amorce : il faut trouver ou la liste se termine. Trois sites en ont besoin,
// et tous appellent [LocaliserBoucleDeRecords] — il n en existe pas de copie (garde-rail
// `archlint/film_localisateur_unique_test.go`) :
//
//	site                                                         ordre
//	cuisson, debut de liste ([localiserLaListe])                  [SignatureStricte]
//	canal des morts, liste que la cuisson n a pas localisee        [SignaturePuisLargeurLibre]
//	  ([debutRecupere])
//	marche de killsource (`killsource.runWalk`)                    [SignaturePuisLargeurLibre]
//
// LES SITES N ONT PAS LE MEME ORDRE, ET LE PARAMETRE LE DIT. La cuisson prend la premiere
// signature stricte telle quelle, sans controle de generation ni repli : quand elle echoue, c est
// elle qui essaie ensuite la fermeture par NEW de tete. Les deux sites qui lisent les morts exigent
// en plus que la generation de la signature soit celle du monde, et essaient le repli a largeur
// libre quand la signature stricte echoue ; le canal des morts n y vient que pour une liste que le
// debut de liste de la cuisson n a pas localisee.
//
// LE BIT NUL QUI PRECEDE LA POSITION. Le jeu ne l ecrit que devant le PREMIER record de la vue B :
// l ecrivain du tick (`FUN_142f2c3b0`) ecrit la liste d evenements (vue A, `FUN_142f2c050`) puis un
// bit 0 (`FUN_1406d49c4`, R8D mis a 0 en 142f2c57d) dans le tampon du premier flux, et la vue B
// (`FUN_142f2cc78`) dans un tampon a part ; `FUN_14299d2c8` les met bout a bout au bit pres
// (`FUN_1406d5d14`, sans prefixe), donc la vue B commence au bit qui suit ce 0, sur lequel le
// lecteur de la liste (`FUN_14076a1c4`) s arrete. Les records de la vue B se suivent sans
// separateur : devant un record qui n est pas le premier, le bit precedent est le dernier bit de
// donnees du record d avant. Le localisateur exige ce bit nul devant toute position qu il essaie :
// c est une condition de RECHERCHE, lue dans le jeu pour la tete de la vue B seulement — quand des
// records NEW precedent la signature (la chaine de tete les retrouve, [debutParChaine]), le bit
// nul devant elle n est pas le terminateur. La longueur de la liste n est ecrite nulle part : il
// faut la chercher.
//
// LA SIGNATURE STRICTE. Le premier record est un delta du slot 123 a composant unique, de la
// largeur EXACTE que ses ecrivains lui donnent sous le cadre du film ([largeurDeSignature] : 35
// bits au cadre par defaut). Candidat UNIQUE et VRAI sur 690 paquets sur 690 confrontes a une
// verite de position independante. Le slot 123 et le composant unique sont MESURES, pas lus dans
// le jeu ; la largeur est derivee des ecrivains.
//
// LE REPLI A LARGEUR LIBRE (`repli_localisation_largeur_libre`, registre des replis) existe parce
// que la largeur de la signature n est que celle du record MODAL : sur les paquets que la
// signature rate, un delta du slot 123 decode en 37, 89, 90 ou 28 bits (cadre par defaut). Il n est
// essaye qu apres l echec de la signature stricte : les paquets deja localises ne bougent pas d un
// bit.
//
// LE CONTROLE DE GENERATION DES SITES QUI LISENT LES MORTS ([TryDeltaAt] ne l applique pas) n agit
// que sous la generation stricte du profil (`Profil.Grammaire.GenerationStricte`) : sans elle,
// [World.GenerationMatches] rend vrai et le controle est vide. La production tourne sous elle :
// killsource la leve dans son profil de depart, et la cuisson pose le profil que killsource calibre
// (`replaybuild` -> `replay.Options.ProfilDeBalayage`). Sous la generation stricte, sans lui, le
// localisateur designerait des positions ou le slot de signature porte une AUTRE generation, et la
// lecture y mourrait aussitot.
//
// CHAQUE POSITION ESSAYEE EST UN ESSAI : [TryDeltaAt] traverse l entite pour de vrai, donc les
// deserialiseurs publient. Les trois fonctions du localisateur eteignent la publication des etats
// de mouvement ([Observation.neutraliserEtatsDeMouvement]) ; sans observateur, c est sans effet.

import "levelup/go-api/internal/games/halo_infinite/film/internal/source"

// marchSignatureSlot est le slot de la SIGNATURE du premier record d un paquet a evenements (slot
// MESURE, cf. l en-tete).
const marchSignatureSlot = uint32(123)

// Largeurs ecrites par le jeu d un delta a composant unique d un objet de l archetype
// `high-frequency` ([archetypeHauteFrequence]), hors mot facultatif et identifiant bas.
const (
	// baselineSansReferenceBits : `FUN_1406cdc04`, R(1) = 0 (aucune reference : le R(7) n est pas
	// ecrit).
	baselineSansReferenceBits = 1
	// masqueEparsUniqueBits : `FUN_142e2da44`, masque epars d un seul composant — R(1) = 0 (pas
	// dense), le compte en R(3), un index en R(6).
	masqueEparsUniqueBits = 1 + 3 + 6
	// composantHauteFrequenceBits : `FUN_142eda680` ecrit 8 bits, `FUN_14076d034` les lit.
	composantHauteFrequenceBits = 8
)

// largeurDeSignature rend la largeur EXACTE de la signature sous le cadre `cfg`, telle que les
// ecrivains du jeu l ecrivent : le mot facultatif de `HasExtraFields` ([motFacultatifDEnTete]), le
// prefixe DELTA ([readRecordType]), l identifiant (`FUN_1406d3140`, [readRecordID] : `IDLowBits`
// bits puis la generation), la baseline, le masque et le composant. 35 bits au cadre par defaut
// (`IDLowBits` 13, sans mot facultatif) ; un cadre a une autre largeur d identifiant bas cherche la
// signature qui lui correspond.
func largeurDeSignature(cfg FrameConfig) int {
	return motFacultatifDEnTete(cfg) + prefixeDeltaBits + cfg.IDLowBits + handleGenBits +
		baselineSansReferenceBits + masqueEparsUniqueBits + composantHauteFrequenceBits
}

// resteMinimalALargeurLibre : la passe a largeur libre n essaie pas une position dont il reste 16
// bits ou moins.
const resteMinimalALargeurLibre = 16

// OrdreDeLocalisation est l ordre des essais du localisateur, propre a chaque site (cf. l en-tete).
type OrdreDeLocalisation uint8

const (
	// SignatureStricte : la premiere signature stricte telle quelle, sans controle de generation
	// ni repli. Ordre de la cuisson.
	SignatureStricte OrdreDeLocalisation = iota
	// SignaturePuisLargeurLibre : la premiere signature stricte si sa generation est celle du
	// monde, sinon le repli a largeur libre. Ordre des deux sites qui lisent les morts.
	SignaturePuisLargeurLibre
)

// LocaliserBoucleDeRecords rend le bit de depart de la boucle de records d un paquet a
// evenements, ou -1, et si la position vient du repli a largeur libre. L appelant compte ce
// verdict au registre des replis ; l ordre [SignatureStricte] ne le rend jamais vrai.
func LocaliserBoucleDeRecords(pay []byte, w *World, cfg FrameConfig, ordre OrdreDeLocalisation) (int, bool) {
	defer cfg.Obs.neutraliserEtatsDeMouvement()() // son TryDeltaAt de controle est un essai
	s := marchLocateStrict(pay, w, cfg)
	if ordre == SignatureStricte {
		return s, false
	}
	if s >= 0 {
		if rec, _, ok := TryDeltaAt(pay, s, w, cfg); ok &&
			w.GenerationMatches(rec.ID, cfg.Profil.Grammaire.GenerationStricte) {
			return s, false
		}
	}
	s = marchLocateFallback(pay, w, cfg)
	return s, s >= 0
}

// marchSignature123 : un delta du slot de signature decode-t-il en `s`, finit-il exactement
// `largeur` plus loin ([largeurDeSignature]), avec un composant unique ?
func marchSignature123(pay []byte, s, largeur int, w *World, cfg FrameConfig) bool {
	rec, end, ok := TryDeltaAt(pay, s, w, cfg)
	return ok && rec.Slot == marchSignatureSlot && end == s+largeur &&
		len(rec.Trace.Comps) == 1
}

// marchLocateStrict rend la premiere position `s >= 2`, precedee d un bit nul, qui porte la
// signature stricte ; -1 si aucune.
func marchLocateStrict(pay []byte, w *World, cfg FrameConfig) int {
	defer cfg.Obs.neutraliserEtatsDeMouvement()() // essais d offset : aucune lecture publiee
	nb, largeur := len(pay)*8, largeurDeSignature(cfg)
	for s := 2; s+largeur < nb; s++ {
		if source.BitAt(pay, s-1) != 0 {
			continue
		}
		if marchSignature123(pay, s, largeur, w, cfg) {
			return s
		}
	}
	return -1
}

// marchLocateFallback reprend la meme condition a LARGEUR LIBRE, avec le controle de generation :
// la premiere position `s >= 2`, precedee d un bit nul, ou un delta du slot de signature decode a
// la generation du monde ; -1 si aucune.
func marchLocateFallback(pay []byte, w *World, cfg FrameConfig) int {
	defer cfg.Obs.neutraliserEtatsDeMouvement()() // essais d offset : aucune lecture publiee
	nb := len(pay) * 8
	for s := 2; s+resteMinimalALargeurLibre < nb; s++ {
		if source.BitAt(pay, s-1) != 0 {
			continue
		}
		rec, _, ok := TryDeltaAt(pay, s, w, cfg)
		if !ok || rec.Slot != marchSignatureSlot ||
			!w.GenerationMatches(rec.ID, cfg.Profil.Grammaire.GenerationStricte) {
			continue
		}
		return s
	}
	return -1
}
