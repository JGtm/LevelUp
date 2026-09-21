package types

// grammar_mouvement.go — LES ETATS DE MOUVEMENT DU SPARTAN, TELS QUE LE FILM LES ECRIT
// (lot 5.3.6, 2026-09-21).
//
// # CE QUE CES TYPES PORTENT, ET CE QU ILS NE PORTENT PAS
//
// Trois etats, et TROIS SEULEMENT, parce que trois seulement sont LUS dans le flux :
//
//	MovementCrouch    `ti=35 i29 unit-crouch-component`            un booleen + une progression
//	MovementSlide     `ti=35 i62 biped-slide-component`            la porte du composant
//	MovementMobility  `ti=35 i54 biped-mobility-action-component`  le drapeau d amorce
//
// LE SPRINT ET LE SAUT N Y SONT PAS, ET C EST UNE MESURE, pas un oubli (lot 5.3.5) :
//
//   - le SPRINT est REFUTE comme observable par la vitesse — la loi de dequantification d `i1`
//     est exacte (ecrivain relu, constantes relues), un oracle independant la valide sur deux
//     films (dispersion du rapport deplacement/vitesse : 1,7 et 2,3 ; meme facteur d unite
//     0,240 et 0,236), et la distribution au sol n a QU UN SEUL mode, a 2-3 m/s (61,3 % et
//     56,1 % de la population) ; au-dela de 4 m/s il reste 0,08 %. S il existe dans le film,
//     c est un ETAT, pas un second regime de vitesse ;
//   - le SAUT est LU mais PAS PROUVE : sa segmentation en episodes repose sur deux seuils
//     d instrument, et la signature ne tient pas d un film a l autre (duree mediane des
//     impulsions a pic >= 3 m/s : 0,632 s sur `bfecd02b`, 1,567 s sur `4f77afc1`, p90 48 s).
//
// Publier l un des deux reviendrait a publier un SEUIL comme une DONNEE. Les deux sont au § 6
// de la note 5.3 avec leurs chiffres.

// Les trois genres d etat de mouvement. Ce sont les valeurs publiees telles quelles au
// document (`stances[].kind`) : une etiquette STABLE, jamais un index de composant.
const (
	// MovementCrouch : accroupi. Le booleen d `i29` EST l etat a l instant ; sa progression
	// (10 bits) est le geste, et elle sert de garde (cf. `MovementStateRead.Progress`).
	MovementCrouch = "crouch"
	// MovementSlide : glissade. La porte de tete d `i62` dit a elle seule si cet instant en
	// porte une.
	MovementSlide = "slide"
	// MovementMobility : action de mobilite. Le drapeau d amorce d `i54` dit « une action est
	// transmise a cet instant ». CE QUE L ACTION EST n est PAS etabli : l enum a trois
	// candidats (escalade, saut de rebord, propulsion) et le domaine mesure de ses champs
	// contredit l hypothese a quatre valeurs (§ 2.8 et D9 de la note 5.3). Le document publie
	// donc l ACTION, pas son nom.
	MovementMobility = "mobility"
)

// MovementStateRead est UNE lecture d etat de mouvement, datee et attribuee a une vie.
//
// C EST UNE LECTURE, PAS UN INTERVALLE. Un delta ne porte ces composants que quand l etat
// CHANGE : la liste est une suite de TRANSITIONS, et c est l assembleur du document qui les
// replie en intervalles (`replay/document_stances.go`). Les garder ici sous leur forme brute
// est ce qui permet a la couverture de dire combien de lectures ont produit combien
// d intervalles.
type MovementStateRead struct {
	// Slot est l identifiant bas du bipede porteur — le meme que celui des trajectoires, donc
	// UNE VIE et non un joueur (le slot migre aux reapparitions).
	Slot uint32
	// Kind est l un des trois genres ci-dessus.
	Kind string
	// TimestampUS est l horodatage du paquet porteur — MEME horloge que `BipedPosition`.
	TimestampUS uint64
	// Chunk / PacketIndex localisent la lecture dans le film.
	Chunk, PacketIndex int
	// On dit si l etat est POSE a cet instant. C est le booleen d `i29`, la porte d `i62`, le
	// drapeau d amorce d `i54`.
	On bool
	// Progress est la progression d animation de l accroupi, en 0..1023 (le quantum brut de
	// 10 bits d `i29`). ZERO sur les deux autres genres, qui n en portent pas.
	Progress uint32
}

// MovementStateStats compte ce que la marche a rencontre. Sans ces denominateurs, « N
// lectures » ne se juge pas : il faut savoir sur combien de records ANNONCANT le composant.
type MovementStateStats struct {
	// Records est le nombre de records delta bipede reconnus par la marche.
	Records int
	// Read est le nombre de lectures ABOUTIES (le deserialiseur a publie, slot lie au bipede,
	// transition non deja vue).
	Read int
	// Absent dit que le film ne transmet AUCUN des trois composants (leur etiquette est
	// absente de l archetype bipede de son registre). C est un fait mesure, distinct d un
	// film ou personne ne s accroupit.
	Absent bool
	// Scanned dit que la marche a eu lieu. Faux = le balayage a refuse (film sans chunk, sans
	// registre, archetype bipede illisible), et TOUT le reste est sans valeur.
	Scanned bool
	// Packets est le nombre de paquets delta effectivement decodes.
	Packets int
	// EventPackets / EventPacketsLocated / EventPacketsUnlocated : les paquets a liste
	// d evenements PLEINE, et ce que la signature du depot en a localise. Un paquet non
	// localise est SAUTE — sauter sa liste bit-exactement demanderait la grammaire de charge de
	// chaque type d evenement. Sans ces trois-la, « N lectures » ne dit pas sur quelle part du
	// film elles portent.
	EventPackets, EventPacketsLocated, EventPacketsUnlocated int
	// Desyncs est le nombre de records `ti=35` desynchronises pendant la marche. Mesure de
	// reference (lot 5.3.5) : 3 sur 97 447 sur `bfecd02b`, 51 sur 315 251 sur `4f77afc1`.
	Desyncs int
	// SlotUnbound compte les lectures ECARTEES parce que leur slot n est pas lie au bipede :
	// l attribution de slot du chemin d inference est partielle pour un record NEW (D13 de la
	// note 5.3). Les attribuer a tort serait pire que de les jeter, et ce compteur dit le prix.
	SlotUnbound int
	// Duplicates compte les re-publications de la MEME transition (meme slot, meme genre, meme
	// instant) : le chemin d inference re-parcourt un record quand une chaine de transitoires le
	// demande. Deduplique, pas compte deux fois.
	Duplicates int
	// MapWidths est le triplet de largeurs d axe employe par le chemin absolu d `i0` pendant la
	// marche. PUBLIE parce que c est le pre-requis le plus facile a oublier : un triplet qui
	// n est pas celui de la carte du match rend la marche muette (cf. l en-tete de
	// `grammar/movement_states.go`).
	MapWidths [3]uint
}
