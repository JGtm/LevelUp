package lecture

import "levelup/go-api/internal/games/halo_infinite/film/types"

// Genre est le genre d'un record : ce que son en-tête annonce (trame delta), ou l'état complet
// d'une entité (image-clé). Le terminateur d'une vue n'est pas un record : c'est la sortie de la
// vue ([SortieVueB]).
type Genre uint8

// Les genres de record.
const (
	// GenreNonRenseigne : sentinelle, jamais posée par la marche.
	GenreNonRenseigne Genre = iota
	// GenreNeuf : record NEW — archétype `R(6)`, état par défaut, porte, masque, composants.
	GenreNeuf
	// GenreDelta : record DELTA — sélecteur de base, masque, composants présents ; l'archétype vient
	// de la table d'entités.
	GenreDelta
	// GenreSuppression : record DEL — mot de 32 bits, la liaison du slot est retirée.
	GenreSuppression
	// GenreEtatComplet : record d'image-clé — tous les composants de l'archétype, dans l'ordre du
	// registre, sans masque.
	GenreEtatComplet
)

// Etat est l'état d'une OCCURRENCE de composant (ADR 0037 IR-4). Le statut de
// `ecs_table.tsv` (`porte`, `partiel`, `non_porte`) est une capacité statique ; l'état dit ce que
// la marche a fait de cette occurrence-ci.
type Etat uint8

// Les états d'une occurrence de composant.
const (
	// EtatNonRenseigne : sentinelle, jamais posée par la marche.
	EtatNonRenseigne Etat = iota
	// EtatInterprete : un canal de la marche interprète l'occurrence — son archétype et son index
	// sont dans l'union des intérêts des canaux — et elle a été traversée (ADR 0037 IR-4). Sa valeur
	// est publiée au crochet du canal et reste hors de la structure (IR-8).
	EtatInterprete
	// EtatDelimite : l'étendue est connue — le composant a été traversé — et aucun canal ne
	// l'interprète.
	EtatDelimite
	// EtatInfranchissable : la largeur est inconnue (lecteur non porté) ; la traversée s'arrête au
	// début de l'occurrence, et le reste de la vue est une queue opaque ([QueueOpaque]).
	EtatInfranchissable
	// EtatArrete : le lecteur du jeu de l'occurrence est porté, et il ÉCHOUE (cause
	// [Composant.Arret]) : ce qu'il lit fait échouer le jeu, ou le film n'établit pas une largeur
	// qu'il lirait. La boucle d'état complet du jeu s'arrête sur lui, et la traversée s'arrête au
	// début de l'occurrence. Ce n'est pas un composant non porté (plan LK, découverte D-12). Seuls les
	// records d'image-clé le portent : ses causes naissent sous la portée de l'état complet, que ni le
	// record NEW ni le DELTA ne posent.
	EtatArrete
)

// CauseDArret nomme l'ÉCHEC d'un lecteur de composant du jeu ([EtatArrete]) : son désérialiseur rend
// faux, et la boucle de composants s'arrête sur lui. Sa largeur est lue ; c'est ce qu'il lit qui fait
// échouer le jeu, ou ce que le film n'établit pas.
type CauseDArret uint8

// Les causes d'échec d'un lecteur de composant.
const (
	// ArretAucun : aucun lecteur n'a échoué.
	ArretAucun CauseDArret = iota
	// ArretPositionNonFinie : la position absolue lue sous la portée porte un flottant non fini
	// (`FUN_140492128`) ; le lecteur du jeu rend faux.
	ArretPositionNonFinie
	// ArretLargeurHandleMoteurUn : la position absolue lue sous la portée annonce la queue de son
	// handle dans un film qui n'exclut pas le type de moteur 1 : la largeur de l'index du handle n'est
	// pas établie.
	ArretLargeurHandleMoteurUn
	// ArretJeuDArmesRefuse : le jeu d armes lu par un thunk qui rend le retour de `FUN_1406d01fc`
	// comme celui du désérialiseur (i42 du bipède, i38 du véhicule) désigne le même emplacement en
	// main principale et en seconde main (param[1] ≠ −1 et param[1] = param[2]) : le lecteur du jeu rend
	// faux.
	ArretJeuDArmesRefuse
)

// ProvenanceLargeur dit d'où vient la largeur avec laquelle une occurrence a été traversée. Elle
// n'est renseignée que pour une occurrence traversée ([EtatInterprete] ou [EtatDelimite]).
type ProvenanceLargeur uint8

// Les provenances d'une largeur de composant.
const (
	// LargeurNonRenseignee : aucune largeur — l'occurrence n'a pas été traversée.
	LargeurNonRenseignee ProvenanceLargeur = iota
	// LargeurEcrivain : le lecteur porté depuis l'écrivain du jeu (ADR 0034 D-3, règle 2).
	LargeurEcrivain
	// LargeurPresumee : une largeur trouvée par fermeture pour un composant de taille fixe qu'on ne
	// fait que sauter, inscrite comme présumée (ADR 0034, amendement D-3 règle 2).
	LargeurPresumee
	// LargeurExceptionDatee : un site qui garde son ancien lecteur, en exception datée
	// (`lecteur_position_exceptions.go`).
	LargeurExceptionDatee
	// LargeurCalibree : une largeur de saut posée par un harnais (`LargeursCalibrees` du profil de
	// balayage) ; jamais en production.
	LargeurCalibree
	// LargeurBouchon : une largeur provisoire posée par un harnais sur un composant non porté
	// (`LargeursBouchon` du profil de balayage) ; jamais en production.
	LargeurBouchon
)

// Preuve dit ce qui prouve qu'un record a été lu au bit près.
type Preuve uint8

// Les preuves d'un record.
const (
	// PreuveNonRenseignee : sentinelle, jamais posée par la marche.
	PreuveNonRenseignee Preuve = iota
	// PreuveFerme : le record a été lu par la marche et sa fermeture tient — le paquet delta se
	// ferme ([VerdictFerme]), ou la traversée de l'état complet d'un record d'image-clé finit
	// exactement sur le record suivant.
	PreuveFerme
	// PreuveNonProuve : le record a été lu par la marche, mais rien ne prouve sa frontière : son
	// paquet ne se ferme pas (fermeture refusée ou queue opaque), ou sa traversée n'atteint pas le
	// record suivant.
	PreuveNonProuve
	// PreuveRecupere : le record n'a pas été lu par la marche ; la couche de récupération l'a
	// trouvé, par une méthode nommée et comptée au registre des replis (ADR 0037 IR-6).
	PreuveRecupere
)

// Liaison dit d'où vient la liaison slot -> archétype d'une entité (ADR 0037 IR-5) : pour un
// record delta, la liaison sous laquelle il a été lu ; pour un record d'image-clé, comment son
// identité a été établie. Les liaisons marquées « récupération » ne sont pas des lectures de la
// marche (ADR 0037 IR-6).
type Liaison uint8

// Les provenances d'une liaison.
const (
	// LiaisonAucune : le record n'est lu sous aucune liaison, ou n'en pose aucune (un NEW refusé ou
	// désynchronisé, un DEL).
	LiaisonAucune Liaison = iota
	// LiaisonLueNeuf : un record NEW lu, traversé sans désynchronisation.
	LiaisonLueNeuf
	// LiaisonImageCle : la chaîne des records d'une image-clé, ancre atteinte de proche en proche.
	LiaisonImageCle
	// LiaisonImageCleElue : récupération — un record d'image-clé dont l'ancre a été ÉLUE
	// (`repli_ancre_d_image_cle_par_election`).
	LiaisonImageCleElue
	// LiaisonDatum : récupération — la table de datums d'une image-clé, lue à position libre.
	LiaisonDatum
	// LiaisonAnticipation : récupération — l'archétype qu'une image-clé ULTÉRIEURE donne à l'eid
	// (`repli_liaison_par_anticipation`).
	LiaisonAnticipation
	// LiaisonInference : récupération — un archétype inféré par la chaîne des records qui suivent.
	LiaisonInference
	// LiaisonJoker : un archétype établi par une contrainte structurelle, génération inconnue.
	LiaisonJoker
)

// TINonResolu est l'archétype d'un record dont l'archétype n'a pas été résolu.
const TINonResolu int16 = -1

// SansDesynchronisation est le [Record.Desync] d'un record traversé jusqu'au bout.
const SansDesynchronisation int16 = -1

// CorpsNonParcouru est le [Record.Desync] d'un record d'image-clé dont la marche n'a pas parcouru
// l'état complet : aucun canal de la distribution ne lit les composants de son archétype. Le record
// garde son identité, son ancre et sa liaison ; il n'a ni composant, ni longueur, ni preuve.
const CorpsNonParcouru int16 = -2

// Composant est UNE occurrence de composant dans un record : son index d'itération dans
// l'archétype, son état, la provenance de sa largeur et son étendue dans le payload.
//
// Douze octets : sa taille est gelée par `tailles_test.go`.
type Composant struct {
	// Index est l'index d'itération du composant dans son archétype, c'est-à-dire son bit de masque
	// (moins de 64).
	Index uint8
	// Etat est l'état de l'occurrence.
	Etat Etat
	// Prov est la provenance de la largeur, [LargeurNonRenseignee] pour une occurrence
	// infranchissable ou arrêtée.
	Prov ProvenanceLargeur
	// Arret est la cause de l'échec du lecteur d'une occurrence [EtatArrete], [ArretAucun] sinon. Il
	// tient dans l'octet de bourrage qui précède [Composant.Debut] : la taille reste celle gelée.
	Arret CauseDArret
	// Debut est le premier bit de l'occurrence.
	Debut uint32
	// Bits est sa longueur, zéro pour une occurrence infranchissable ou arrêtée.
	Bits uint32
}

// Record est UN record lu : son genre, son identité, son archétype, sa liaison, sa preuve, son
// étendue et ses composants. Ses composants sont la tranche `Comps[0]:Comps[1]` de
// [Paquet.Comps] : un record ne porte pas de tranche à lui, l'arène du paquet les porte tous.
//
// Quarante octets : sa taille est gelée par `tailles_test.go`.
type Record struct {
	// Genre est le genre du record.
	Genre Genre
	// Vue est le rang de la vue qui le porte, dans la numérotation du film : 1 pour un record de
	// trame delta (la vue B) ; pour un record d'image-clé, le rang lu dans les deux bits de tête de
	// son identifiant.
	Vue uint8
	// Liaison est la provenance de la liaison sous laquelle le record a été lu ou qu'il établit.
	Liaison Liaison
	// Preuve est ce qui prouve sa frontière.
	Preuve Preuve
	// TI est l'archétype, [TINonResolu] quand il n'a pas été résolu.
	TI int16
	// Desync est l'index d'itération où la traversée s'est arrêtée, [SansDesynchronisation] pour
	// un record traversé jusqu'au bout, [CorpsNonParcouru] pour un record d'image-clé dont le
	// corps n'a pas été parcouru. Quand le dernier composant du record est [EtatInfranchissable] ou
	// [EtatArrete], c'est son index ; sinon la traversée s'est arrêtée avant tout composant (archétype
	// hors du registre, slot non lié).
	Desync int16
	// Vie est l'identité du record (ADR 0034, `LifeKey`) : son slot et les deux bits de tête de son
	// identifiant, la « tête » que la table d'entités compare — la génération du handle d'un record
	// de trame delta ; pour un record d'image-clé, le monde lit ces deux bits comme le rang de sa
	// vue ([Record.Vue]).
	Vie types.LifeKey
	// Debut est le premier bit de l'en-tête du record, mot facultatif d'en-tête compris.
	Debut uint32
	// Bits est sa longueur, jusqu'au dernier bit lu de son corps.
	Bits uint32
	// Masque est le masque de présence des composants (zéro pour un DEL et pour un record dont le
	// masque n'a pas été lu).
	Masque uint64
	// Comps borne les composants du record dans [Paquet.Comps] : `[début, fin)`.
	Comps [2]uint32
}
