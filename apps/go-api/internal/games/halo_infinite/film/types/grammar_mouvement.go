package types

// grammar_mouvement.go — LES ETATS DE MOUVEMENT DU SPARTAN, TELS QUE LE FILM LES ECRIT
// (lots 5.3.6, 5.9.4 et 5.9.5, 2026-09-21).
//
// # CE QUE CES TYPES PORTENT, ET DE QUELLE NATURE
//
// QUATRE etats LUS dans le flux, et UN CINQUIEME qui est CALCULE. La difference n est pas un
// detail de documentation : elle est dans les NOMS, pour qu un client ne puisse pas confondre
// une lecture avec une derivation.
//
//	MovementCrouch       `ti=35 i29 unit-crouch-component`            un booleen + une progression
//	MovementSlide        `ti=35 i62 biped-slide-component`            la porte du composant
//	MovementMobility     `ti=35 i54 biped-mobility-action-component`  le drapeau d amorce
//	MovementSprint       `ti=35 i57 biped-spartan-ability-component`  l index de la fente ACTIVE
//	MovementJumpDerived  (aucun composant)                            l integrale de `i1`, par sa
//	                                                                  HAUTEUR
//
// LE SPRINT A ETE PUBLIE LE 2026-09-21 (lot 5.9.5), ET CE N EST PAS UN REVIREMENT SUR LA MESURE
// DU 5.3.5. Ce que le 5.3.5 avait refute reste vrai mot pour mot : le sprint n est PAS observable
// PAR LA VITESSE — la distribution au sol n a qu un seul mode a 2-3 m/s, et le lot 5.9.5 l a
// re-mesure (Vm 2,31 contre Vs 2,85 sur `bfecd02b` ; 2,55 contre 2,84 sur `4f77afc1`). La
// conclusion d alors le disait deja : « S il existe dans le film, c est un ETAT, pas un second
// regime de vitesse. » C est exactement ce qu il est, et `i57` le porte.
//
// LE SAUT, LUI, RESTE UNE DERIVATION, et son nom le dit. Ce que le 5.3.5 refusait — « publier un
// SEUIL comme une DONNEE » — est evite par une constante qui n est pas un seuil d instrument mais
// un FAIT DE JEU : `SpartanJumpHeightM`, un pic etroit mesure sur TROIS films. Et la chaine du
// declencheur n est plus « non trouvee » : le lot 5.11 la FERME (`FUN_140d988c8` lit le bit 18 de
// `obj+0x308` du contact, pas un champ du flux) et mesure sur un film temoin controle qu AUCUN
// champ replique ne bascule au saut hors `i0`, `i1` et `i25`. Il n y aura pas de genre `jump` LU :
// il n y a rien a lire. Details dans l en-tete de `grammar/movement_states_jump.go`.

// Les genres d etat de mouvement. Ce sont les valeurs publiees telles quelles au document
// (`stances[].kind`) : une etiquette STABLE, jamais un index de composant.
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
	// MovementJumpDerived : le SAUT, et il est DERIVE — jamais lu. Le genre porte le mot
	// `Derived` dans sa valeur meme (`jumpDerived`) parce qu un client qui l affiche doit
	// pouvoir le distinguer d un bit du film sans consulter de documentation : les trois
	// autres genres sont des composants deserialises, celui-ci est l integrale de la vitesse
	// verticale d `i1` reconnue a sa HAUTEUR (cf. `SpartanJumpHeightM`).
	//
	// LE JOUR OU LE CHAMP REPLIQUE SERA NOMME, un genre `jump` LU le remplacera, avec sa
	// propre montee de revision. La chaine du declencheur est remontee jusqu au compteur de
	// ticks sans contact `u+0x89b` et NON TROUVEE a `FUN_1408b2f90` (lot 5.9.2) : c est un
	// maillon manquant, pas un refus.
	MovementJumpDerived = "jumpDerived"
	// MovementSprint : le SPRINT, et il est LU — `ti=35 i57 biped-spartan-ability-component`
	// porte l INDEX DE LA FENTE DE CAPACITE ACTIVE, et la fente 1 est le sprint.
	//
	// LES TROIS FENTES SONT NOMMEES PAR L IMAGE, PAS PAR UN SCORE (lot 5.9.5). `FUN_1407e9ce4`
	// aiguille sur le GROUPE DE TAG de la definition de capacite et appelle, pour chacun, un
	// desenregistreur qui teste l index actif contre SA fente :
	//
	//	'saev' (esquive) -> FUN_14319d0ac : fente `comp+0x1c`, index **0**
	//	'sasp' (SPRINT)  -> FUN_14319d1ec : fente `comp+0x20`, index **1**
	//	'sagh' (grappin) -> FUN_14319d14c : fente `comp+0x24`, index **2**
	//
	// Le flux ecrit `bloc+3 = R(2) - 1` (`FUN_142f268c4`), donc le brut `2` designe la fente 1.
	// La chaine complete du consommateur au champ est au lot 5.9.1 : de la condition
	// d animation `is_sprinting_tlg` au bit 45 des drapeaux d unite, pose par `Sprint::Update`
	// (`FUN_1431a2474`) depuis une fraction rampee localement — et ce qui vient du film est
	// l ACTIVATION, pas la fraction.
	MovementSprint = "sprint"
)

// SpartanJumpHeightM est la hauteur de la montee d un saut de Spartan, EN METRES.
//
// MESURE, PAS CHOISIE (lot 5.7.5, 2026-09-21) : la distribution des hauteurs d episode aerien,
// obtenue par integration de la vitesse verticale tenue d `i1`, porte un pic etroit a cette
// valeur sur DEUX films — `bfecd02b` (snowbound) pic x 10,7 au-dessus de ses voisins, montee
// 0,467 s ; `4f77afc1` (flood gulch) pic x 3,9, montee 0,466 s. C est un fait de jeu (tous les
// Spartans sautent la meme hauteur), donc une constante et non un reglage.
//
// TROISIEME MESURE, SUR UN SAUT UNIQUE ET CONTROLE (lot 5.11.1, 2026-09-21) : le film
// `dad793c7` porte UN joueur qui n a fait QUE sauter, une fois. Son unique episode donne
// vz maximale +3,446 m/s, montee a vz >= 0,5 m/s de **0,470 s**, hauteur integree
// **0,8649 m** — 1,8 % de cette constante — et une vitesse au sol de 0,000 m/s sur tout le vol.
// C est la mesure la plus propre du depot : une seule vie, une seule action, aucun autre
// mouvement dans le film.
const SpartanJumpHeightM = 0.85

// SpartanJumpHeightTol est la demi-largeur de la fenetre d acceptation, en fraction de
// `SpartanJumpHeightM`. Dix pour cent : la valeur de l etiquetage de l oracle du lot 5.7.5, sous
// laquelle le pic se separe de ses voisins sur les deux films.
const SpartanJumpHeightTol = 0.10

// SpartanJumpRiseMinMS est la vitesse verticale, en m/s, au-dela de laquelle une montee commence.
// Un demi-metre par seconde ecarte le bruit du quantum log/exp au voisinage de zero sans couper
// un saut, dont l amorce est de plusieurs m/s.
const SpartanJumpRiseMinMS = 0.5

// SpartanJumpHoldMaxUS borne la duree qu une lecture de vitesse peut representer. `i1` ne voyage
// que sur CHANGEMENT : au-dela, le silence n est pas une vitesse tenue, c est une vie qui ne
// transmet plus. Integrer un tel silence fabriquerait des hauteurs.
const SpartanJumpHoldMaxUS = 250000

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
	// VelocityReads est le nombre de lectures d `i1` RETENUES (mode quantifie, vitesse presente,
	// slot lie au bipede) — le denominateur de la derivation du saut.
	VelocityReads int
	// JumpEpisodes est le nombre de montees FERMEES examinees, JumpsDerived celles dont la
	// hauteur integree tombe dans la fenetre `SpartanJumpHeightM` +/- `SpartanJumpHeightTol`.
	// Le rapport des deux est la SELECTIVITE de la derivation, et sans lui « N sauts » ne se
	// juge pas.
	JumpEpisodes, JumpsDerived int
	// MapWidths est le triplet de largeurs d axe employe par le chemin absolu d `i0` pendant la
	// marche. PUBLIE parce que c est le pre-requis le plus facile a oublier : un triplet qui
	// n est pas celui de la carte du match rend la marche muette (cf. l en-tete de
	// `grammar/movement_states.go`).
	MapWidths [3]uint
}
