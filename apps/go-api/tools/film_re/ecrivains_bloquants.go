//go:build research

package filmre

// ecrivains_bloquants.go — le releve du 2026-09-16, une entree par composant NON PORTE des
// archetypes utiles au lot 3.6. Voir doc.go pour ce que ce paquet est et n'est pas.

// Grammaire dit ce qu'on sait de la suite de champs d'un composant, et a quel point on le sait.
type Grammaire int

const (
	// GrammaireInconnue : l'ecrivain est nomme, sa suite de champs n'a pas encore ete lue.
	GrammaireInconnue Grammaire = iota
	// GrammaireComplete : la suite de champs est lue au bit pres, sans inconnue.
	GrammaireComplete
	// GrammairePartielle : la structure est lue, une largeur ou une branche reste ouverte.
	GrammairePartielle
)

// Ecrivain est un composant ECS dont l'ecrivain a ete localise dans l'executable.
//
// Descripteur et Ecrivain sont des adresses ABSOLUES dans l'image de base 0x140000000.
// L'invariant de la famille relevee est Ecrivain == *(Descripteur + 0x40) ; SlotNom vaut
// Descripteur + 0x18. Les trois sont consignes parce que le lot 3.6 devra rejouer la chaine sur
// un autre binaire le jour ou le jeu bouge, et qu'un seul des trois ne suffit pas a la refaire.
type Ecrivain struct {
	TI       int    // index d'archetype du film (== le type d'objet de l'executable)
	I        int    // index du composant dans l'archetype
	Nom      string // nom exact, celui de la chaine de .rdata et de ecs_table.tsv
	Chaine   uint64 // adresse de la chaine de caracteres dans .rdata (0 si non relevee)
	SlotNom  uint64 // adresse du slot d'accesseur de nom = Descripteur + 0x18
	Adresse  uint64 // adresse de l'ecrivain = *(Descripteur + 0x40)
	Etat     Grammaire
	Champs   string // la suite de champs, quand elle est lue ; sinon ""
	BitsTyp  string // largeur typique, quand elle est connue ; sinon ""
	Depend   string // ce dont la grammaire depend hors du flux ; "" = rien
	Remarque string // ce qu'il faut savoir avant de porter
}

// Descripteur rend l'adresse du descripteur du composant, deduite du slot du nom.
func (e Ecrivain) Descripteur() uint64 { return e.SlotNom - 0x18 }

// TI9 — un seul composant non porte, grammaire complete. Note : NOTE_3_6_TI9_2026-09-16.md.
var TI9 = []Ecrivain{{
	TI: 9, I: 4, Nom: "managed-player-forge-weather-effect-overrides-component",
	Chaine: 0x143c95460, SlotNom: 0x143d08858, Adresse: 0x142ed5bc8,
	Etat: GrammaireComplete, Champs: "R(32) ; R(32)", BitsTyp: "64",
	Remarque: "inconditionnel, aucune porte ; meme forme que i5 managed-player-active-mission-name-component, deja porte",
}}

// TI11 — un seul composant non porte ; l'en-tete est lu, le corps est derriere un appel virtuel.
// Note : NOTE_3_6_TI11_2026-09-16.md.
var TI11 = []Ecrivain{{
	TI: 11, I: 4, Nom: "managed-objective-interaction-filter-component",
	Adresse: 0x142c7023c, Etat: GrammairePartielle,
	Champs:   "R(4) masque ; R(1) ; pour chaque bit pose des 4 emplacements : R(4) etiquette puis dispatch FUN_141e99630 (case 0 = 0 bit ; case 1..5 = R(1) puis appel virtuel vtable[+0x08])",
	BitsTyp:  "5 minimum ; 5 + 4 x (4 + corps) maximum",
	Depend:   "la largeur du corps depend du type dynamique de l'objet filtre (vtable[+0x08])",
	Remarque: "l'adresse etait deja au depot (FUN_142edb5cc -> FUN_142c7023c) ; ce releve ajoute la table d'etiquettes",
}}

// TI12 — 26 composants non portes, 18 ecrivains nommes ici, 8 instances de
// visual-state-groups non resolues. Note : NOTE_3_6_TI12_2026-09-16.md.
var TI12 = []Ecrivain{
	{TI: 12, I: 1, Nom: "managed-navpoint-flags-component", SlotNom: 0x143d08038, Adresse: 0x141094130,
		Etat: GrammaireComplete, Champs: "R(8)", BitsTyp: "8",
		Remarque: "FUN_141094130 delegue a FUN_14109414c ; un seul champ, inconditionnel"},
	{TI: 12, I: 2, Nom: "managed-navpoint-visibility-distance-filters-component", SlotNom: 0x143d07fe8, Adresse: 0x140dbde1c},
	{TI: 12, I: 3, Nom: "managed-navpoint-visible-offscreen-filters-component", SlotNom: 0x143d07f98, Adresse: 0x140dbdf80},
	{TI: 12, I: 4, Nom: "managed-navpoint-can-be-occluded-filters-component", SlotNom: 0x143d07f48, Adresse: 0x140dbdfac},
	{TI: 12, I: 5, Nom: "managed-navpoint-visibility-filter-component", SlotNom: 0x143d083f8, Adresse: 0x140dbe194},
	{TI: 12, I: 6, Nom: "managed-navpoint-docking-filter-component", SlotNom: 0x143d083a8, Adresse: 0x140dbdf34},
	{TI: 12, I: 7, Nom: "managed-navpoint-docking-order-component", SlotNom: 0x143d08358, Adresse: 0x142ed5050},
	{TI: 12, I: 8, Nom: "managed-navpoint-docking-group-name-component", SlotNom: 0x143d08308, Adresse: 0x142ed5028},
	{TI: 12, I: 9, Nom: "managed-navpoint-formatted-text-component", SlotNom: 0x143d08538, Adresse: 0x1410e7b90,
		Remarque: "nom jumeau de ti=11 i2 (consumeObjectiveFormattedText) : liste taguee attendue"},
	{TI: 12, I: 10, Nom: "managed-navpoint-timers-component", SlotNom: 0x143d084e8, Adresse: 0x1410d9040,
		Remarque: "nom jumeau de ti=11 i0 (2 x R(7), valeur+1)"},
	{TI: 12, I: 11, Nom: "managed-navpoint-manual-timer-initial-duration-component", SlotNom: 0x143d08498, Adresse: 0x142ed5194},
	{TI: 12, I: 12, Nom: "managed-navpoint-manual-timer-current-duration-component", SlotNom: 0x143d08448, Adresse: 0x142ed512c},
	{TI: 12, I: 13, Nom: "managed-navpoint-top-progress", SlotNom: 0x143d08178, Adresse: 0x142ed51d8,
		Remarque: "jumeau de i14 radial-progress, PORTE : R(8) dequantifie dans [-1, 1]"},
	{TI: 12, I: 15, Nom: "managed-navpoint-bottom-progress", SlotNom: 0x143d080d8, Adresse: 0x142ed4fe4,
		Remarque: "meme famille que i13 et i14"},
	{TI: 12, I: 16, Nom: "managed-navpoint-override-flags", SlotNom: 0x143d08088, Adresse: 0x140ebf834},
	{TI: 12, I: 17, Nom: "managed-navpoint-object-marker", SlotNom: 0x143d082b8, Adresse: 0x141169e68},
	{TI: 12, I: 18, Nom: "managed-navpoint-position-offset", SlotNom: 0x143d08268, Adresse: 0x140f04f68,
		Remarque: "bloc de vecteur attendu (FUN_142ee2194 = R(16) dequantifie dans [-100, 100])"},
	{TI: 12, I: 19, Nom: "managed-navpoint-visual-states-component", SlotNom: 0x143d08218, Adresse: 0x142ed521c},
}

// TI35 — aucun composant non porte ; le bloquant est un `partiel` dont le predicat de queue est
// resolu. Note : NOTE_3_6_TI35_2026-09-16.md.
var TI35 = []Ecrivain{{
	TI: 35, I: 60, Nom: "simulation-state-component", Adresse: 0x142ed6d88,
	Etat:     GrammairePartielle,
	Champs:   "R(1) drapeau ; si 0 : rien de plus. Si 1 : 2 x FUN_1407f2058 (porte inversee R(1) puis R(5)) ; 4 x R(16) dequantifie [-100, 100] ; 2 x R(2) ; 4 x R(16) ; FUN_140c1e79c (largeur A RELEVER) ; puis QUEUE si et seulement si le predicat d'orthonormalite est vrai",
	BitsTyp:  "1 (branche vide) ; 168 de structure connue + queue sinon",
	Depend:   "aucune configuration : le predicat FUN_140501798 ne lit que les deux vecteurs deja decodes et trois constantes de .rdata (0.0f, 1.0f, epsilon 1.0e-3 a 0x143cd84bc)",
	Remarque: "le MEME composant est ti=40 i43 : le porter une fois le porte deux fois. Restent deux largeurs a relever : FUN_140c1e79c et FUN_14076e494(..., 0x10, ...)",
}}

// TI40 — 16 composants non portes, tous nommes ici. Le golden 0.A.3 ne designe AUCUN bloquant
// pour cet archetype : ce sont des candidats, pas une cause. Note : NOTE_3_6_TI40_2026-09-16.md.
var TI40 = []Ecrivain{
	{TI: 40, I: 30, Nom: "vehicle-auto-turret-triggers-component", SlotNom: 0x143d0b680, Adresse: 0x142f04994},
	{TI: 40, I: 31, Nom: "vehicle-auto-turret-aiming-vector-component", SlotNom: 0x143d0b6d0, Adresse: 0x14115f33c,
		Remarque: "vecteur de visee : R(16) dequantifies attendus"},
	{TI: 40, I: 32, Nom: "vehicle-transformed-or-desired-open-state-changed-component", SlotNom: 0x143d0b630, Adresse: 0x142f04b70},
	{TI: 40, I: 33, Nom: "vehicle-type-state-component", SlotNom: 0x143d0b3f8, Adresse: 0x142f02474,
		Remarque: "resolu a la main : la chaine a deux references, dont une table a 0x143b8c860"},
	{TI: 40, I: 34, Nom: "vehicle-type-physics-component", SlotNom: 0x143d0b308, Adresse: 0x142f02498,
		Remarque: "resolu a la main, meme raison que i33"},
	{TI: 40, I: 35, Nom: "vehicle-auto-turret-target-component", SlotNom: 0x143d0b770, Adresse: 0x142f0496c,
		Remarque: "une cible est probablement une reference d'entite (FUN_1406d3140)"},
	{TI: 40, I: 36, Nom: "vehicle-sentry-state-component", SlotNom: 0x143d0b720, Adresse: 0x142f04b34},
	{TI: 40, I: 37, Nom: "vehicle-emp-timer-component", SlotNom: 0x143d0b498, Adresse: 0x142f049dc},
	{TI: 40, I: 38, Nom: "vehicle-weapon-set-component", SlotNom: 0x143d0b4e8, Adresse: 0x14116d3cc},
	{TI: 40, I: 39, Nom: "vehicle-auto-turret-component", SlotNom: 0x143d0b448, Adresse: 0x142f04884},
	{TI: 40, I: 40, Nom: "vehicle-equipment-turret-parent-component", SlotNom: 0x143d0b5d8, Adresse: 0x142f04a00,
		Remarque: "un parent est probablement une reference d'entite"},
	{TI: 40, I: 41, Nom: "vehicle-seats-override-pitch-component", SlotNom: 0x143d0b538, Adresse: 0x142f04a4c},
	{TI: 40, I: 42, Nom: "vehicle-seats-override-yaw-component", SlotNom: 0x143d0b588, Adresse: 0x142f04ac0},
	{TI: 40, I: 45, Nom: "air-drop-flight-component", SlotNom: 0x143d0b1c8, Adresse: 0x142f02508},
	{TI: 40, I: 46, Nom: "warp-component", SlotNom: 0x143d0b2b8, Adresse: 0x142f04bcc},
	{TI: 40, I: 47, Nom: "vehicle-low-frequency-component", SlotNom: 0x143d0b218, Adresse: 0x142f04a20,
		Remarque: "suspect n 1 : la famille *-low-frequency est celle des composants les plus larges (D1 du lot 1.9.1 bis)"},
}

// TI43 — 22 composants non portes, tous nommes ici ; i19 porte deja sa grammaire, relevee au
// lot 1.9.1 bis. Note : NOTE_3_6_TI43_2026-09-16.md.
var TI43 = []Ecrivain{
	{TI: 43, I: 19, Nom: "device-position-animation-name-component", SlotNom: 0x143d0ceb0, Adresse: 0x1410156e4,
		Etat: GrammaireComplete, Champs: "R(32) identifiant (FUN_141015740) ; R(10) dequantifie dans [0, 10] (14101571c)", BitsTyp: "42",
		Remarque: "grammaire relevee au lot 1.9.1 bis ; sert de calibration de la chaine de resolution sur cette famille"},
	{TI: 43, I: 20, Nom: "device-position-animation-control-component", SlotNom: 0x143d0cff0, Adresse: 0x142f02d04},
	{TI: 43, I: 21, Nom: "device-position-group-component", SlotNom: 0x143d0cf50, Adresse: 0x1407f0678},
	{TI: 43, I: 22, Nom: "device-power-component", SlotNom: 0x143d0cfa0, Adresse: 0x14100d310},
	{TI: 43, I: 23, Nom: "device-power-group-component", SlotNom: 0x143d0c218, Adresse: 0x14100d2d0},
	{TI: 43, I: 24, Nom: "device-interaction-in-progress-component", SlotNom: 0x143d0c1c0, Adresse: 0x141167910},
	{TI: 43, I: 25, Nom: "device-interaction-hold-time-component", SlotNom: 0x143d0c308, Adresse: 0x142f02c54},
	{TI: 43, I: 26, Nom: "device-control-action-string-override-component", SlotNom: 0x143d0c268, Adresse: 0x142f029e4},
	{TI: 43, I: 27, Nom: "device-health-station-charges-component", SlotNom: 0x143d0c0c8, Adresse: 0x140bee524},
	{TI: 43, I: 28, Nom: "device-health-station-in-use-component", SlotNom: 0x143d0c020, Adresse: 0x142f02bec},
	{TI: 43, I: 29, Nom: "device-exclusive-user-component", SlotNom: 0x143d0c070, Adresse: 0x14116fcb0,
		Remarque: "un utilisateur exclusif est probablement une reference d'entite (FUN_1406d3140)"},
	{TI: 43, I: 30, Nom: "device-in-primary-mode-component", SlotNom: 0x143d0c170, Adresse: 0x142f02c20},
	{TI: 43, I: 31, Nom: "device-dispenser-monitors-changed-component", SlotNom: 0x143d0c118, Adresse: 0x142f02a48},
	{TI: 43, I: 32, Nom: "device-dispenser-state-flags-component", SlotNom: 0x143d0c4f0, Adresse: 0x142f02bcc},
	{TI: 43, I: 33, Nom: "device-dispenser-require-los-component", SlotNom: 0x143d0c540, Adresse: 0x142f02b7c},
	{TI: 43, I: 34, Nom: "device-animation-layer-settings-component", SlotNom: 0x143d0c4a0, Adresse: 0x140f44104},
	{TI: 43, I: 35, Nom: "device-animation-layer-state-component", SlotNom: 0x143d0c590, Adresse: 0x141076f68},
	{TI: 43, I: 36, Nom: "device-dispenser-state-component", SlotNom: 0x143d0c3a8, Adresse: 0x142f02bb0},
	{TI: 43, I: 37, Nom: "device-object-dispenser-timer-component", SlotNom: 0x143d0c358, Adresse: 0x142f02c94},
	{TI: 43, I: 38, Nom: "device-position-transition-velocity-component", SlotNom: 0x143d0c400, Adresse: 0x142f02d28},
	{TI: 43, I: 39, Nom: "device-machine-flags-component", SlotNom: 0x143d0c450, Adresse: 0x14107bb68},
	{TI: 43, I: 40, Nom: "device-interaction-start-time-override-component", SlotNom: 0x143d0c2b8, Adresse: 0x141fd7bc0},
}

// Releve rend toutes les entrees du 2026-09-16, dans l'ordre des archetypes du plan.
//
// ti=42 n'y figure pas, et c'est un resultat : ses 21 composants sont TOUS portes, le golden
// 0.A.3 ne nomme aucun bloquant, et sa fermeture de 10/2 087 n'a donc aucune cause portable
// (cf. NOTE_3_6_TI42_2026-09-16.md et la decouverte D8 du lot 1.9.1 bis pas 2 bis).
func Releve() []Ecrivain {
	out := make([]Ecrivain, 0, len(TI9)+len(TI11)+len(TI12)+len(TI35)+len(TI40)+len(TI43))
	for _, g := range [][]Ecrivain{TI9, TI11, TI12, TI35, TI40, TI43} {
		out = append(out, g...)
	}
	return out
}

// ARelever rend les entrees dont la grammaire n'a pas encore ete lue chez l'ecrivain — la file
// de travail du lot 3.6, et la seule mesure honnete de ce qu'il reste a faire.
func ARelever() []Ecrivain {
	var out []Ecrivain
	for _, e := range Releve() {
		if e.Etat == GrammaireInconnue {
			out = append(out, e)
		}
	}
	return out
}
