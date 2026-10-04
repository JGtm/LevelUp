package grammar

// localisateur.go — LE LOCALISATEUR UNIQUE DE LA BOUCLE DE RECORDS D UN PAQUET A EVENEMENTS.
//
// Dans un paquet delta qui porte une liste d evenements (bit 1 du payload), la boucle de records
// ne commence pas a l amorce : il faut trouver ou la liste se termine. Trois sites en ont besoin,
// et tous appellent [LocaliserBoucleDeRecords] — il n en existe pas de copie (garde-rail
// `archlint/film_localisateur_unique_test.go`) :
//
//	site                                         ordre
//	cuisson, debut de liste ([localiserLaListe])  [SignatureStricte], puis la fermeture par NEW
//	                                              de tete, puis [SignatureHauteFrequence]
//	marche des morts d objet ([marchDebut])       [SignaturePuisLargeurLibre]
//	marche de killsource (`killsource.runWalk`)    [SignaturePuisLargeurLibre]
//
// LES SITES N ONT PAS LE MEME ORDRE, ET LE PARAMETRE LE DIT. La cuisson prend la premiere
// signature du slot 123 telle quelle, sans controle de generation ni repli, et place entre elle et
// la signature haute frequence la fermeture par NEW de tete ([localiserLaListe]). Les deux marches
// qui lisent les morts exigent que la generation de la signature du slot 123 soit celle du monde,
// essaient la signature haute frequence quand le slot 123 n en porte aucune, et le repli a largeur
// libre en dernier. A tous les sites, la signature haute frequence exige la generation du monde
// ([signeLaHauteFrequence]). Ces ordres ne sont pas ecrits par le jeu, qui ne dit pas
// quelle preuve essayer d abord : ce sont les ordres mesures du lot LS
// (`.ai/V7.5/film_re/campagne_grammaire_2026-10-01/LOT_LS.md`).
//
// LE BIT NUL QUI PRECEDE LA POSITION EST LU DANS LE JEU : l ecrivain du tick (`FUN_142f2c3b0`)
// ecrit la liste d evenements (vue A, `FUN_142f2c050`) puis un bit 0 (`FUN_1406d49c4`, R8D mis a 0
// en 142f2c57d), et la boucle de records (vue B) commence au bit suivant ; le lecteur de la liste
// (`FUN_14076a1c4`) s arrete sur ce 0. La longueur de la liste n est ecrite nulle part : il faut la
// chercher.
//
// LA VUE B EST ECRITE PAR GENRE PUIS PAR SLOT CROISSANT. La liste des entites de la vue
// (`FUN_142f2e174`) parcourt la table de vue par index croissant ; l ecrivain (`FUN_142f2cc78`)
// range chaque entree dans le sous-ecrivain de son genre, et `FUN_14076b9c8` les concatene : tous
// les NEW, puis tous les DELTA, puis tous les DEL. Le premier DELTA de la vue B est donc celui du
// plus petit slot qui en ecrit un ; les NEW de tete le precedent, et la cuisson les retrouve par la
// chaine ([debutParChaine]). La signature haute frequence localise le premier delta HAUTE
// FREQUENCE, pas le premier delta de la vue B : les deltas des slots inferieurs (statborg `ti=6`,
// joueur `ti=5`) le precedent, et la chaine, qui ne remonte que des NEW, ne les lit pas.
//
// LA SIGNATURE DU SLOT 123. Le premier delta est celui du slot 123, long de 35 bits EXACTEMENT, a
// composant unique. Candidat UNIQUE et VRAI sur 690 paquets sur 690 confrontes a une verite de
// position independante. Le slot, la largeur et le composant unique sont MESURES, pas lus dans le
// jeu.
//
// LA SIGNATURE HAUTE FREQUENCE. La meme forme sur un AUTRE slot, que le monde lie a l archetype
// `high-frequency` ([archetypeHauteFrequence] : `FUN_140e462d8` ecrit `+0x4754 = 4` ; composant
// unique de table 0x143d06a60, lecteur `FUN_14076d034`, ecrivain `FUN_142eda680`). Tous les objets
// de cet archetype ont le meme ecrivain, donc la meme forme de delta ; dans les modes a objectif
// porte, les images-cles lient d autres slots a cet archetype, et le slot 123 ne l est que dans une
// partie des chunks. L archetype se reconnait par la cle qui route sa table au dispatch, jamais par
// le nom de son composant. La signature exige la generation du monde. Elle n est cherchee que dans
// un paquet ou le slot 123 ne porte aucune signature.
//
// LE REPLI A LARGEUR LIBRE (`repli_localisation_largeur_libre`, registre des replis) existe parce
// que 35 n est que la largeur MODALE du premier record : sur les paquets que la signature rate, un
// delta du slot 123 decode en 37, 89, 90 ou 28 bits. Il n est essaye qu apres l echec des
// signatures : les paquets deja localises ne bougent pas d un bit.
//
// LE CONTROLE DE GENERATION EST INDISPENSABLE AUX ETAGES DES MARCHES : la primitive d essai
// ([TryDeltaAt]) ne l applique pas. Sans lui, le localisateur designe des positions ou le slot
// de signature porte une AUTRE generation, et la marche y meurt aussitot.
//
// CHAQUE POSITION ESSAYEE EST UN ESSAI : [TryDeltaAt] traverse l entite pour de vrai, donc les
// deserialiseurs publient. Les fonctions du localisateur eteignent la publication des etats de
// mouvement ([Observation.neutraliserEtatsDeMouvement]) ; sans observateur, c est sans effet.

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
	// SignatureStricte : la premiere signature du slot 123 telle quelle, sans controle de
	// generation ni repli. Premier etage de la cuisson.
	SignatureStricte OrdreDeLocalisation = iota
	// SignaturePuisLargeurLibre : la premiere signature du slot 123 si sa generation est celle du
	// monde ; sans signature du slot 123, la signature haute frequence ; sinon le repli a largeur
	// libre. Ordre des deux marches qui lisent les morts.
	SignaturePuisLargeurLibre
	// SignatureHauteFrequence : la premiere signature haute frequence d un paquet ou le slot 123
	// n en porte aucune, -1 sinon. Dernier etage de la cuisson.
	SignatureHauteFrequence
)

// LocaliserBoucleDeRecords rend le bit de depart de la boucle de records d un paquet a
// evenements, ou -1, et si la position vient du repli a largeur libre. L appelant compte ce
// verdict au registre des replis ; seul l ordre [SignaturePuisLargeurLibre] peut le rendre vrai.
func LocaliserBoucleDeRecords(pay []byte, w *World, cfg FrameConfig, ordre OrdreDeLocalisation) (int, bool) {
	defer cfg.Obs.neutraliserEtatsDeMouvement()() // son TryDeltaAt de controle est un essai
	s, hauteFrequence := marchLocateSignatures(pay, w, cfg)
	switch ordre {
	case SignatureStricte:
		return s, false
	case SignatureHauteFrequence:
		return hauteFrequence, false
	}
	if s >= 0 {
		if rec, _, ok := TryDeltaAt(pay, s, w, cfg); ok && aLaGenerationDuMonde(rec, w, cfg) {
			return s, false
		}
	} else if hauteFrequence >= 0 {
		return hauteFrequence, false
	}
	s = marchLocateFallback(pay, w, cfg)
	return s, s >= 0
}

// formeDeSignature : le delta decode a `s` finit-il exactement `marchSignatureBits` plus loin, a
// composant unique ?
func formeDeSignature(rec FrameRecord, s, fin int) bool {
	return fin == s+marchSignatureBits && len(rec.Trace.Comps) == 1
}

// signeLaHauteFrequence : un delta de la forme de la signature porte-t-il un objet que le monde
// lie a l archetype `high-frequency`, a sa generation ?
func signeLaHauteFrequence(rec FrameRecord, w *World, cfg FrameConfig) bool {
	return rec.TypeIndex == archetypeHauteFrequence && aLaGenerationDuMonde(rec, w, cfg)
}

// aLaGenerationDuMonde : le record porte-t-il la generation que le monde lie a son slot (sous la
// generation stricte du profil) ? Le controle de generation du localisateur, que [TryDeltaAt]
// n applique pas.
func aLaGenerationDuMonde(rec FrameRecord, w *World, cfg FrameConfig) bool {
	return w.GenerationMatches(rec.ID, cfg.Profil.Grammaire.GenerationStricte)
}

// marchLocateSignatures rend, en un balayage, la premiere position `s >= 2`, precedee d un bit
// nul, qui porte la signature du slot 123 ; quand il n y en a aucune (-1), la premiere qui porte
// la signature haute frequence, -1 si aucune. Une signature du slot 123 arrete le balayage : la
// seconde valeur est alors -1.
func marchLocateSignatures(pay []byte, w *World, cfg FrameConfig) (int, int) {
	defer cfg.Obs.neutraliserEtatsDeMouvement()() // essais d offset : aucune lecture publiee
	hauteFrequence := -1
	nb := len(pay) * 8
	for s := 2; s+marchSignatureBits < nb; s++ {
		if source.BitAt(pay, s-1) != 0 {
			continue
		}
		rec, fin, ok := TryDeltaAt(pay, s, w, cfg)
		if !ok || !formeDeSignature(rec, s, fin) {
			continue
		}
		if rec.Slot == marchSignatureSlot {
			return s, -1
		}
		if hauteFrequence < 0 && signeLaHauteFrequence(rec, w, cfg) {
			hauteFrequence = s
		}
	}
	return -1, hauteFrequence
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
		if !ok || rec.Slot != marchSignatureSlot || !aLaGenerationDuMonde(rec, w, cfg) {
			continue
		}
		return s
	}
	return -1
}
