package lecture

// Les rangs des trois vues d'une trame delta, dans la numérotation du film (le frame-processeur
// `FUN_142987460` les parcourt dans cet ordre).
const (
	// RangVueA : la vue des messages.
	RangVueA uint8 = 0
	// RangVueB : la vue des entités, la seule qui porte des records.
	RangVueB uint8 = 1
	// RangVueC : la vue de contrôle.
	RangVueC uint8 = 2
)

// DebutDeVueB dit comment la marche a trouvé le début de la vue B d'une trame delta. Un début
// LOCALISÉ est de la récupération qui vit dans la marche (ADR 0037 IR-6) : la marche ne l'a pas
// atteint en lisant le paquet depuis sa tête. Un début LU ([DebutEnTete], [DebutParVueA]) n'en est
// pas : la marche l'a atteint en lisant la vue A jusqu'à son terminateur.
type DebutDeVueB uint8

// Les débuts de vue B.
const (
	// DebutNonRenseigne : sans objet — un paquet d'image-clé.
	DebutNonRenseigne DebutDeVueB = iota
	// DebutEnTete : lu depuis la tête du paquet (bit de configuration, puis la vue A jusqu'à son
	// terminateur).
	DebutEnTete
	// DebutParSignature : paquet à liste d'événements, liste localisée par la signature du slot
	// 123 (`marchLocateStrict`).
	DebutParSignature
	// DebutParChaine : paquet à liste d'événements, liste ouverte par ses records NEW de tête, dont
	// la chaîne finit au bit près sur le début que la signature a localisé (`localiserLaListe`).
	DebutParChaine
	// DebutParFermeture : paquet à liste d'événements que la signature ne localise pas, liste
	// ouverte par un record NEW de tête depuis lequel la marche ferme le paquet (`localiserLaListe`).
	DebutParFermeture
	// DebutParFermetureAuBit : comme [DebutParFermeture], mais la marche depuis ce record ne ferme
	// le paquet qu'au bit près, une règle de l'écrivain contredite : le repli nommé
	// `repli_debut_de_liste_ferme_au_bit`.
	DebutParFermetureAuBit
	// DebutNonLocalise : paquet à liste d'événements dont le début n'a pas été trouvé ; aucune vue
	// n'est lue.
	DebutNonLocalise
	// DebutParVueA : paquet à liste d'événements dont la vue A a été LUE jusqu'à son terminateur ;
	// la vue B commence au bit qui le suit, chez l'écrivain (`FUN_142f2c3b0`) comme chez le lecteur
	// (`FUN_142987460`). C'est une LECTURE, pas une localisation : aucune position n'est cherchée.
	// Posé quand la table des genres du film est ÉGALE à celle du jeu, ou quand elle en est un
	// préfixe et que la marche depuis cette fin ferme le paquet.
	DebutParVueA
)

// EtatDeVue dit jusqu'où la marche a lu la vue A ou la vue C d'une trame delta.
type EtatDeVue uint8

// Les états d'une vue.
const (
	// VueNonLue : la marche n'a pas lu la vue — la vue C derrière une vue B qui ne s'est pas
	// terminée ; les vues d'un paquet d'image-clé, qui n'en a pas.
	VueNonLue EtatDeVue = iota
	// VueTerminee : la vue a été lue jusqu'à son terminateur.
	VueTerminee
	// VueArretee : la lecture s'est arrêtée dans la vue. Une vue A s'arrête après le genre du
	// message qu'elle ne sait pas lire (charge non portée, film dont la table des genres ne se lit
	// pas, fin du payload) ; une vue C arrêtée porte sa cause dans [Fermeture.Queue].
	VueArretee
)

// SortieVueB dit comment la boucle de records de la vue B s'est terminée (ADR 0037 IR-4).
type SortieVueB uint8

// Les sorties de la vue B.
const (
	// SortieNonAtteinte : la marche n'est pas entrée dans la vue B.
	SortieNonAtteinte SortieVueB = iota
	// SortieTerminateur : le terminateur de la boucle de records.
	SortieTerminateur
	// SortieRejetHorsDatum : un delta dont l'eid n'est dans aucune table de datums connue ; la vue
	// s'arrête après son en-tête, comme chez l'écrivain.
	SortieRejetHorsDatum
	// SortieRejetDeVue : un delta dont le slot appartient à une autre vue.
	SortieRejetDeVue
	// SortieRecordInfranchissable : un record dont la traversée s'est arrêtée ([Record.Desync]).
	SortieRecordInfranchissable
	// SortieFinDePayload : la fin du payload, sans terminateur.
	SortieFinDePayload
	// SortiePlafond : le plafond de records de la boucle hors ligne.
	SortiePlafond
)

// Verdict est le verdict de fermeture d'une trame delta (ADR 0037 IR-4). Les trois verdicts
// rendus ne se confondent jamais.
type Verdict uint8

// Les verdicts de fermeture.
const (
	// VerdictNonRendu : aucun verdict — un paquet d'image-clé, dont chaque record porte sa preuve,
	// ou une trame marchée sans les classes de vue (la vue C n'y est pas lue).
	VerdictNonRendu Verdict = iota
	// VerdictFerme : le prédicat de fermeture de la grammaire tient ; chaque record du paquet est
	// prouvé avec lui.
	VerdictFerme
	// VerdictRefuse : la vue C a été lue jusqu'à son terminateur mais le paquet ne se ferme pas :
	// une largeur est fausse quelque part devant, à une position INCONNUE, ou la lecture contredit
	// une règle de l'écrivain ([Fermeture.Regle]). Les records ne sont pas prouvés.
	VerdictRefuse
	// VerdictQueueOpaque : la marche s'est arrêtée à une position CONNUE, pour une cause typée
	// ([Fermeture.Queue]).
	VerdictQueueOpaque
)

// CauseDeQueue est la cause typée d'une queue opaque : ce qui a arrêté la marche d'une trame
// delta à une position connue.
type CauseDeQueue uint8

// Les causes d'une queue opaque.
const (
	// CauseAucune : la marche ne s'est pas arrêtée avant la fin des vues.
	CauseAucune CauseDeQueue = iota
	// CauseListeNonLocalisee : le début de la liste d'événements n'a pas été trouvé.
	CauseListeNonLocalisee
	// CauseMessageVueANonPorte : la marche partie de la tête du paquet s'est arrêtée sur le premier
	// message de la vue A — elle ne traverse qu'une vue A vide.
	CauseMessageVueANonPorte
	// CauseFinDePayloadVueA : la vue A a atteint la fin du payload avant son terminateur.
	CauseFinDePayloadVueA
	// CauseComposantNonPorte : un composant sans lecteur ([Record.Desync] désigne son index).
	CauseComposantNonPorte
	// CauseArchetypeHorsRegistre : un record dont l'archétype est hors du registre du film.
	CauseArchetypeHorsRegistre
	// CauseSlotNonLie : un delta dont le slot n'est pas lié et dont l'archétype n'a pas été inféré.
	CauseSlotNonLie
	// CauseFinDePayloadVueB : la vue B a atteint la fin du payload sans terminateur.
	CauseFinDePayloadVueB
	// CausePlafondVueB : la vue B a atteint le plafond de records de la boucle hors ligne.
	CausePlafondVueB
	// CauseDebordementVueC : la vue C a atteint la fin du payload avant son terminateur.
	CauseDebordementVueC
	// CauseKindVueCNonPorte : un sélecteur de la vue C (1 ou 2) dont la charge n'est pas portée.
	CauseKindVueCNonPorte
	// CauseBlocBCVueC : le bloc de 0xbc octets d'une entrée de contrôle, non porté.
	CauseBlocBCVueC
	// CausePlafondVueC : la vue C a atteint le plafond de tours de la boucle hors ligne.
	CausePlafondVueC
)

// SansRecord est le [QueueOpaque.Record] d'une cause qui ne désigne aucun record.
const SansRecord int32 = -1

// SansComposant est le [QueueOpaque.Composant] d'une cause qui ne désigne aucun composant.
const SansComposant int16 = -1

// QueueOpaque est le reste d'une trame delta que la marche n'a pas traversé, à partir d'une
// position connue.
type QueueOpaque struct {
	// Debut est le premier bit non traversé.
	Debut uint32
	// Cause est ce qui a arrêté la marche.
	Cause CauseDeQueue
	// Record est le rang, dans [Paquet.Records], du record qui a arrêté la marche ; [SansRecord]
	// sinon.
	Record int32
	// Composant est l'index du composant infranchissable dans son archétype ; [SansComposant]
	// sinon.
	Composant int16
}

// AucuneRegle est la [Fermeture.Regle] d'une lecture qui ne contredit aucune règle de l'écrivain.
const AucuneRegle uint8 = 0

// Fermeture est le verdict de fermeture d'une trame delta et ce qu'elle a consommé. Le prédicat
// appartient à la grammaire (`ecrivain_invariants.go`) ; la structure porte son verdict, jamais
// une seconde définition (ADR 0037 IR-4).
type Fermeture struct {
	// Verdict est le verdict ; [VerdictNonRendu] pour un paquet d'image-clé.
	Verdict Verdict
	// AuBit dit que la vue C s'est lue jusqu'à son terminateur et que le reste du payload fait de 0
	// à 7 bits nuls : la moitié « au bit près » du prédicat.
	AuBit bool
	// Regle est la première règle de l'écrivain que la lecture contredit, codée comme
	// l'`InvariantEcrivain` de la grammaire ; [AucuneRegle] sinon. Le paquet n'est fermé que si
	// [Fermeture.AuBit] tient ET qu'aucune règle n'est contredite.
	Regle uint8
	// Consommes est la position du curseur de la marche à son arrêt, en bits depuis le début du
	// payload.
	Consommes uint32
	// Longueur est la longueur du payload, en bits.
	Longueur uint32
	// Queue est la queue opaque quand le verdict est [VerdictQueueOpaque] ; sa cause est
	// [CauseAucune] sinon.
	Queue QueueOpaque
}

// VueA est la vue des messages (rang 0) d'une trame delta, telle que la lecture unique de la vue A
// l'a lue, une fois par trame, avant que la marche en décide la suite : message par message jusqu'à
// son terminateur quand le film la rend lisible, sa tête seule sinon (la continuation et, quand elle
// annonce un message, son genre). Terminée, son étendue finit sur le bit qui suit le terminateur.
type VueA struct {
	// Debut et Bits sont son étendue ; zéro bit pour une vue non lue.
	Debut, Bits uint32
	// Etat dit jusqu'où la marche l'a lue.
	Etat EtatDeVue
	// PremierPresume est le rang, dans Genres, du premier genre dont la numérotation est PRÉSUMÉE
	// et non lue : sur un film dont la fin de vue A ne vaut que prouvée (classe PRÉFIXE), le premier
	// genre au-delà du dernier dont la version native diffère de 1 (`grammar/vue_a_versions.go`).
	// Ce genre et ceux qui le suivent sont lus sous cette présomption. len(Genres) quand aucun ne
	// l'est — la valeur zéro d'une vue vide.
	PremierPresume uint16
	// Genres sont les sélecteurs `R(7)` des messages lus, dans l'ordre, puis celui du message qui a
	// arrêté la lecture ; le premier est le genre de la tête.
	Genres []uint8
	// Kills sont les messages `PlayerKilledEvent` (genre 85) lus, dans l'ordre.
	Kills []MessageDeKill
}

// RefAbsente est la valeur d'une référence d'entité qu'un message ne porte pas (garde à 1).
const RefAbsente int8 = -1

// MessageDeKill est un message `PlayerKilledEvent` (genre 85) de la vue A : sa position et ses
// champs (`FUN_14104bd08`), lus sans la queue optionnelle.
//
// Seize octets : sa taille est gelée par `tailles_test.go`.
type MessageDeKill struct {
	// Debut est le premier bit du message : le bit de continuation qui l'annonce.
	Debut uint32
	// PartDuTueur et PartDeLAssistant sont ses deux `R(32)` (`[+8]` et `[+0x14]`).
	PartDuTueur, PartDeLAssistant uint32
	// Victime, Tueur et Assistant sont ses trois références d'entité (`FUN_1407f2058`, 5 bits),
	// dans l'ordre de la lecture ; [RefAbsente] pour une référence que le message ne porte pas.
	Victime, Tueur, Assistant int8
	// Drapeau est le `R(1)` qui suit la part du tueur (`[+0xc]`).
	Drapeau uint8
}

// VueB est la vue des entités (rang 1) d'une trame delta : son étendue et sa sortie. Ses records
// sont [Paquet.Records].
type VueB struct {
	// Debut et Bits sont son étendue ; zéro bit pour une vue non atteinte. Un en-tête rejeté est
	// compris dans l'étendue : l'écrivain l'a écrit, et la vue s'arrête après lui.
	Debut, Bits uint32
	// Sortie dit comment sa boucle de records s'est terminée.
	Sortie SortieVueB
	// EIDRejete est l'eid complet de l'en-tête rejeté, quand la sortie est un rejet ; zéro sinon.
	EIDRejete uint32
}

// IndexDeControleAbsent est l'[EntreeVueC.Index] d'une entrée qui ne porte pas d'index de
// contrôle.
const IndexDeControleAbsent int8 = -1

// EntreeVueC est UN tour de la boucle de la vue C : le sélecteur qu'il annonce et son étendue.
//
// Douze octets : sa taille est gelée par `tailles_test.go`.
type EntreeVueC struct {
	// Debut est le premier bit du tour : le bit de continuation qui l'annonce.
	Debut uint32
	// Bits est sa longueur, sélecteur et charge compris ; pour un tour qui arrête la marche,
	// jusqu'à la position d'arrêt.
	Bits uint32
	// Kind est le sélecteur `R(2)` : 0 l'entrée de contrôle d'un participant, 1 et 2 des charges
	// non portées, 3 aucune charge.
	Kind uint8
	// Index est l'index de contrôle `R(5)` d'une entrée `kind 0` — la place du joueur dans le film ;
	// [IndexDeControleAbsent] sinon.
	Index int8
}

// VueC est la vue de contrôle (rang 2) d'une trame delta, telle que la marche l'a lue.
type VueC struct {
	// Debut et Bits sont son étendue ; zéro bit pour une vue non lue.
	Debut, Bits uint32
	// Etat dit jusqu'où la marche l'a lue.
	Etat EtatDeVue
	// Entrees sont ses tours, dans l'ordre du flux.
	Entrees []EntreeVueC
}

// Paquet est UN paquet du film tel que la marche l'a lu : son en-tête, ses vues, ses records et
// leurs composants, et son verdict de fermeture.
//
// IL EST RÉUTILISÉ D'UN TOUR À L'AUTRE (cf. la documentation du paquet) : il n'est valide que le
// temps du tour d'itération qui le rend.
type Paquet struct {
	// Chunk est le numéro du chunk, celui que `FilmContext.ChunkNumbers` énumère.
	Chunk int
	// Index est le rang du paquet dans son chunk.
	Index int
	// Type est le type du paquet (0 une trame delta, 2 une image-clé).
	Type uint16
	// TS est l'horodatage du paquet, en microsecondes, horloge du film.
	TS uint64
	// Payload est le payload du paquet, sous-tranche du chunk décompressé : jamais une copie.
	Payload []byte
	// Debut dit comment la marche a trouvé le début de la vue B ; [DebutNonRenseigne] pour une
	// image-clé.
	Debut DebutDeVueB
	// VueA, VueB et VueC sont les trois vues d'une trame delta ; vides pour une image-clé.
	VueA VueA
	VueB VueB
	VueC VueC
	// Fermeture est le verdict de fermeture d'une trame delta.
	Fermeture Fermeture
	// Records sont les records lus, dans l'ordre du flux : ceux de la vue B pour une trame delta,
	// les records d'état complet pour une image-clé.
	Records []Record
	// Comps est l'arène des composants de tous les records du paquet ([Record.Comps]).
	Comps []Composant
	// Entites est la table d'entités de la marche, telle que la lecture de ce paquet l'a laissée
	// (ADR 0037 IR-5) : en lecture seule, valide le temps du tour.
	Entites Entites
}
