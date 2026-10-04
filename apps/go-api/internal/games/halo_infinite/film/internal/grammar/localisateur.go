package grammar

// localisateur.go — LE LOCALISATEUR UNIQUE DE LA BOUCLE DE RECORDS D UN PAQUET A EVENEMENTS.
//
// Dans un paquet delta qui porte une liste d evenements (bit 1 du payload), la boucle de records
// ne commence pas a l amorce : il faut trouver ou la liste se termine. Trois sites en ont besoin,
// et tous appellent [LocaliserBoucleDeRecords] — il n en existe pas de copie (garde-rail
// `archlint/film_localisateur_unique_test.go`) :
//
//	site                                         ordre
//	cuisson, debut de liste ([localiserLaListe])  [SignatureStricte]
//	marche des morts d objet ([marchDebut])       [SignaturePuisLargeurLibre]
//	marche de killsource (`killsource.runWalk`)    [SignaturePuisLargeurLibre]
//
// LES SITES N ONT PAS LE MEME ORDRE, ET LE PARAMETRE LE DIT. La cuisson prend la premiere
// signature stricte telle quelle, sans controle de generation ni repli : quand elle echoue, c est
// elle qui essaie ensuite la fermeture par NEW de tete. Les deux marches qui lisent les morts
// exigent en plus que la generation de la signature soit celle du monde, et essaient le repli a
// largeur libre quand la signature stricte echoue.
//
// LE BIT NUL QUI PRECEDE LA POSITION EST LU DANS LE JEU : l ecrivain du tick (`FUN_142f2c3b0`)
// ecrit la liste d evenements (vue A, `FUN_142f2c050`) puis un bit 0 (`FUN_1406d49c4`, R8D mis a 0
// en 142f2c57d), et la boucle de records (vue B) commence au bit suivant ; le lecteur de la liste
// (`FUN_14076a1c4`) s arrete sur ce 0. La longueur de la liste n est ecrite nulle part : il faut la
// chercher.
//
// LA SIGNATURE STRICTE. Le premier record est un delta du slot 123, long de 35 bits EXACTEMENT, a
// composant unique. Candidat UNIQUE et VRAI sur 690 paquets sur 690 confrontes a une verite de
// position independante. Le slot, la largeur et le composant unique sont MESURES, pas lus dans le
// jeu.
//
// LE REPLI A LARGEUR LIBRE (`repli_localisation_largeur_libre`, registre des replis) existe parce
// que 35 n est que la largeur MODALE du premier record : sur les paquets que la signature rate, un
// delta du slot 123 decode en 37, 89, 90 ou 28 bits. Il n est essaye qu apres l echec de la
// signature stricte : les paquets deja localises ne bougent pas d un bit.
//
// LE CONTROLE DE GENERATION EST INDISPENSABLE AUX DEUX ETAGES DES MARCHES : la primitive d essai
// ([TryDeltaAt]) ne l applique pas. Sans lui, le localisateur designe des positions ou le slot
// de signature porte une AUTRE generation, et la marche y meurt aussitot.
//
// CHAQUE POSITION ESSAYEE EST UN ESSAI : [TryDeltaAt] traverse l entite pour de vrai, donc les
// deserialiseurs publient. Les trois fonctions du localisateur eteignent la publication des etats
// de mouvement ([Observation.neutraliserEtatsDeMouvement]) ; sans observateur, c est sans effet.

import "levelup/go-api/internal/games/halo_infinite/film/internal/source"

// marchSignatureSlot / marchSignatureBits sont la SIGNATURE du premier record d un paquet a
// evenements : un delta du slot 123, long de 35 bits exactement, a composant unique.
const (
	marchSignatureSlot = uint32(123)
	marchSignatureBits = 35
)

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
	// monde, sinon le repli a largeur libre. Ordre des deux marches qui lisent les morts.
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
// `marchSignatureBits` plus loin, avec un composant unique ?
func marchSignature123(pay []byte, s int, w *World, cfg FrameConfig) bool {
	rec, end, ok := TryDeltaAt(pay, s, w, cfg)
	return ok && rec.Slot == marchSignatureSlot && end == s+marchSignatureBits &&
		len(rec.Trace.Comps) == 1
}

// marchLocateStrict rend la premiere position `s >= 2`, precedee d un bit nul, qui porte la
// signature stricte ; -1 si aucune.
func marchLocateStrict(pay []byte, w *World, cfg FrameConfig) int {
	defer cfg.Obs.neutraliserEtatsDeMouvement()() // essais d offset : aucune lecture publiee
	nb := len(pay) * 8
	for s := 2; s+marchSignatureBits < nb; s++ {
		if source.BitAt(pay, s-1) != 0 {
			continue
		}
		if marchSignature123(pay, s, w, cfg) {
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
