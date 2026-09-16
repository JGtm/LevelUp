package replay

// document_tracks.go — LES TYPES DE LA TRACE PUBLIEE : la vie d un slot (Track), ce qu elle
// portait (Loadout), ce qu elle a tire (Shot) et l etendue qu elles occupent (Bounds).
//
// DEPLACEMENT PUR depuis `document.go` (lot 2.7 volet publication, 2026-09-16) : ce fichier
// passait 500 lignes en melangeant le type RACINE de l artefact (`ReplayDocument`, son schema et
// sa chronique) avec les types de la couche TRACE, que tous les calques referencent. Aucun octet
// publie ne bouge — memes champs, memes etiquettes JSON, meme ordre de declaration.

// Loadout est l'ensemble des armes PORTÉES par un slot à un instant de référence.
//
// CE QUE LE CHAMP GARANTIT : à l'instant T, ce slot AVAIT ces armes dans son inventaire.
// Témoin croisé sur une source indépendante (l'arme des events de tir) : 98,3 % d'accord
// contre 7,2 % pour le témoin qui ne casse QUE la jointure record->slot. Détail et limites
// en tête de loadouts.go.
//
// CE QU'IL NE GARANTIT PAS, et il faut le dire à l'écran :
//   - QUELLE arme est dégainée. Le loadout est l'inventaire, pas la main. Croiser avec le
//     dernier Shot du même slot désigne l'arme en main ; sans tir récent, on ne sait pas.
//   - la CONTINUITÉ. Un keyframe toutes les ~18-20 s : entre deux instants publiés, un
//     ramassage d'arme est invisible. Un client qui maintient la dernière valeur connue
//     affiche un état de référence, pas une mesure de l'instant.
//   - les GRENADES, la capacité d'armure, les munitions : NON décodées.
type Loadout struct {
	// T est l'index de frame, sur le même axe que Point.T.
	T int `json:"t"`
	// Slot est le slot du biped porteur : il désigne la Track concernée.
	Slot uint32 `json:"slot"`
	// W liste les identifiants de FAMILLE d'arme (high-32 du weapon-id 64 bits) en
	// hexadécimal 8 chiffres, dans l'ordre de lecture du record. La famille est l'identité
	// de l'arme, le suffixe bas ne porte que la variante cosmétique (cf. weaponv3.CanonWeaponID) ;
	// les alias d'un même canon sont repliés — un canon = une entrée.
	W []string `json:"w"`
}

// Shot est un tir décodé, placé à la position de son tireur.
//
// CE QUE LE CHAMP GARANTIT :
//   - le tir a INFLIGÉ DES DÉGÂTS. Le record du film (event type 105) n'existe que quand un
//     dégât est appliqué : il n'y a pas de record de tir manqué, donc pas de notion
//     « touché / raté » à afficher — tous les tirs publiés ici ont touché quelqu'un.
//   - l'origine (X, Y) est la position du biped tireur à l'instant du tir.
//
// CE QU'IL NE GARANTIT PAS :
//   - l'exhaustivité : seuls les tirs dont le tireur a pu être rattaché SANS AMBIGUÏTÉ sont
//     publiés (30 à 57 % des events selon le film) ;
//   - la VICTIME : elle n'est pas décodée du film (champ de largeur runtime). Le trait part
//     du tireur dans la direction visée, il ne relie pas deux joueurs.
type Shot struct {
	// T est l'index de frame, sur le même axe que Point.T.
	T int `json:"t"`
	// Slot est le slot du biped tireur : il désigne la Track d'où part le tir.
	Slot uint32 `json:"slot"`
	// X, Y sont l'origine du tir (position du tireur).
	X float32 `json:"x"`
	Y float32 `json:"y"`
	// H (optionnel) est le CAP DE VISÉE du tir en degrés, même convention que Point.H.
	// Absent quand la visée n'était pas lisible hors ligne (le champ vit après des boucles
	// de longueur variable dans ~80 % des records) : le client dessine alors un simple
	// marqueur, sans direction. Même PIÈGE omitempty que Point.H, même parade (0 -> 360).
	H float32 `json:"h,omitempty"`
	// Weapon est l'identifiant global 64 bits de l'arme, en hexadécimal (un entier 64 bits
	// ne survit pas au `number` JavaScript). Clé de metadata.weapon_labels.weapon_id.
	Weapon string `json:"w,omitempty"`
	// Vehicle est le SLOT DU VÉHICULE d'où part le tir, quand le tireur était EMBARQUÉ.
	//
	// À QUOI IL SERT, ET POURQUOI IL FAUT UN MARQUEUR. Sur un tir à pied, `X`/`Y` sont la
	// position du BIPÈDE : le client peut retrouver le tireur dans ses pistes et y accrocher ce
	// qu'il veut. Sur un tir en véhicule, le bipède ne réplique plus (primitive V1a.4) et
	// `X`/`Y` sont la position INTERPOLÉE DU VÉHICULE : sans ce champ, le client ne saurait pas
	// que la piste du `Slot` est muette à cet instant, et chercherait un pion qui n'existe pas.
	// Il porte le slot plutôt qu'un booléen pour que l'effet puisse s'ancrer sur LE véhicule
	// concerné — c'est la même clé que `VehicleTrack.Slot`.
	//
	// POINTEUR, comme `VehicleRide.Seat` et pour la même raison : un slot vaut zéro en droit, et
	// `omitempty` sur un entier effacerait ce zéro exactement comme une absence de véhicule.
	// Nil = tir à pied, et c'est le cas nominal.
	Vehicle *uint32 `json:"v,omitempty"`
}

// Bounds est l'étendue alignée sur les axes de tous les points de trajectoire, dans le
// repère monde partagé. Permet au client d'ajuster la scène au viewport (le range monde
// absolu est inutile au rendu — seule la disposition relative importe).
type Bounds struct {
	MinX float32 `json:"minX"`
	MinY float32 `json:"minY"`
	MaxX float32 `json:"maxX"`
	MaxY float32 `json:"maxY"`
	// MinZ / MaxZ (optionnels) donnent l'amplitude verticale, pour colorer les étages.
	// PIÈGE omitempty : une borne exactement nulle est omise — les valeurs sont issues
	// d'une déquantification à mi-bucket (min + step*(q+0.5)), un zéro exact est donc
	// hors d'atteinte en pratique ; le client lit une borne absente comme 0.
	MinZ float32 `json:"minZ,omitempty"`
	MaxZ float32 `json:"maxZ,omitempty"`
}

// Track est la trajectoire d'une entité (slot biped) sur la timeline du rejeu.
//
// ATTENTION : un slot est réattribué aux respawns — une Track = UNE VIE, pas un joueur.
// Le regroupement des vies par joueur se fait par XUID.
type Track struct {
	Slot uint32 `json:"slot"`
	// Team est le DÉSIGNATEUR D'ÉQUIPE QUE LE FILM ÉCRIT (schéma 57, lot 1.7) : `0..8` sont les
	// huit camps de l'énumération `mp_team_designator` du jeu, `-1` est « aucune équipe ».
	//
	// D'OÙ IL VIENT : le composant i0 de l'archétype ti=9 de la trame d'état, sur quatre bits,
	// à une position DÉRIVÉE de la grammaire (cf. `filmdec.ScanPlayerTeams`). Le film est la
	// SEULE source — décision utilisateur du 2026-09-13 : « si le décodeur est fiable, pas
	// besoin du repli ». La base ne pose aucune équipe ; elle CONTRÔLE, et
	// `coverage.teams.{accord, contradiction, silence}` disent ce qu'elle en pense.
	//
	// LES ARTEFACTS ANTÉRIEURS AU SCHÉMA 57 PORTENT -1 PARTOUT, et c'était la vérité du moment :
	// le dépôt tenait que l'équipe n'était pas dans le film. Elle y est.
	//
	// `-1` NE DIT PAS DEUX CHOSES À LA FOIS, il en dit une : « cette vie n'a pas d'équipe dans
	// l'artefact ». Que ce soit parce que le mode n'en a pas (FFA) ou parce que le film n'a pas
	// nommé ce joueur se lit dans `coverage.teams` (`noTeam` contre `unread`), pas ici.
	Team int `json:"team"`
	// Name est TOUJOURS VIDE, et c'est délibéré : le film ne porte aucun gamertag. Le remplir
	// exigerait de lire la base depuis un outil hors ligne dont toute la valeur est de n'en
	// dépendre pas. Le client joint le nom par XUID.
	Name string `json:"name,omitempty"`
	// XUID est l'IDENTITÉ du porteur de cette vie, en décimal (un entier 64 bits ne survit pas
	// au `number` JavaScript ; le décimal est aussi la forme employée par la base).
	//
	// POURQUOI LE XUID ET PAS UN INDEX. Un index est un ORDRE, jamais une identité — la leçon
	// a coûté une fausse découverte à ce chantier (un tri alphabétique publié comme une
	// permutation du format). Le xuid est stable, global, et indépendant de tout tri : c'est
	// la seule clé sur laquelle un client peut joindre sans rien supposer.
	//
	// D'OÙ IL VIENT : le fil des morts du film nomme chaque vie par le xuid de sa victime
	// (cf. lives.go). Vide quand la vie n'a pas été nommée — 15 vies sur 105 sur le film de
	// référence, dont 4 antérieures au début réel du match et 6 survivants de fin de partie,
	// que le film ne clôt par aucun événement.
	XUID string `json:"xuid,omitempty"`
	// Bot est le NOM du bot qui porte cette vie (schéma 36), suffixe « [bot] » compris — un
	// bot n'a pas de xuid, et c'est le seul cas où une vie est nommée sans en avoir un.
	//
	// D'OÙ IL VIENT : BOT_METADATA (paquet type 12) déclare slot de roster + nom, et le pont
	// des fermetures attribue un slot de biped à cet index quand l'unicité le permet
	// (cf. nameBotTracks). Une vie de bot que le pont ne peut pas attribuer reste anonyme —
	// même règle que les humains : rien plutôt que faux.
	Bot    string  `json:"bot,omitempty"`
	Points []Point `json:"points"`
	// StartFrame / EndFrame (optionnels) bornent la vie de la track sur l'axe de temps :
	// le client peut masquer l'entité hors de cette fenêtre au lieu de la figer.
	StartFrame int `json:"startFrame,omitempty"`
	EndFrame   int `json:"endFrame,omitempty"`
}
