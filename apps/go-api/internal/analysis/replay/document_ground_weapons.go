package replay

// document_ground_weapons.go — CE QUE L'ARTEFACT PUBLIE DES SOCLES D'ARME, et ce qu'il refuse
// d'en publier. Les types du calque, et la chronique du schéma 11 qui les a fait entrer.
//
// Extrait de `document.go` et de `ground_weapon_pads.go` au correctif de revue du 2026-08-17 :
// le lot des socles avait poussé `document.go` de 631 à 673 lignes, au-dessus d'un seuil déjà
// gelé par la baseline. La forme publiée vit donc ici, la RÈGLE qui la remplit reste dans
// `ground_weapon_pads.go`, et le DÉCODAGE dans `build_ground_weapons.go`.
//
// CHRONIQUE — v11 (2026-08-17, plan `.ai/V7.5/replay2d/PLAN_ARMES_AU_SOL_2E_LECTURE.md`,
// phase 3). Le document publie `weaponPads` — les SOCLES D'ARME du match, avec leurs
// apparitions, leurs intervalles de présence bornés par le recensement des images-clés et leur
// cycle de réapparition quand il est établi — et `padPickups`, les occupations qui se sont
// achevées. Les deux champs sont optionnels, mais la version monte : le calque des socles côté
// client N'EXISTE que si l'artefact les porte, et la reprise du backfill se fait par
// SchemaVersion — un artefact v10 doit se voir comme « à re-cuire », pas comme à jour.
//
// CE QUE LA MESURE A REFUSÉ DE PUBLIER, ET C'EST LA MOITIÉ DU RÉSULTAT. Le RAMASSEUR n'est pas
// publié jusqu'au schéma 29 (`PadPickup.XUID` valait `null` partout) : l'oracle indépendant — le loadout d'image-clé
// du ramasseur présumé — donnait 88,1 % en suivant le slot de vie et 79,7 % en suivant le joueur,
// contre >= 90 % exigé, et le seuil n'a pas été rebaissé. Les « ramassages » d'armes LÂCHÉES à
// une mort ne sont pas publiés non plus : l'accord y passe SOUS son propre témoin (32,1 % contre
// 65,0 %), signature d'un critère qui ne mesure rien — une arme lâchée disparaît le plus souvent
// toute seule. Et AUCUN catalogue de carte n'est écrit : sur deux films de Catalyst les dix
// socles sont aux dix mêmes coordonnées au centimètre, mais trois portent une arme différente —
// le socle appartient à la carte, l'arme qui y apparaît appartient au match.
//
// LES DEUX CHAMPS DU DOCUMENT, en détail (ils ne portent qu'un renvoi dans `document.go`) :
//
//	weaponPads   position monde du socle, famille d'arme, instants d'apparition, intervalles de
//	             présence BORNÉS par le recensement des images-clés, cycle SEULEMENT s'il est
//	             établi. Un socle est une RÉCURRENCE MESURÉE, pas une lecture de fichier de
//	             carte : des armes de même famille apparaissent au moins deux fois à moins d'un
//	             mètre, sans qu'aucune vie de joueur ne s'achève à proximité (le négatif de la
//	             règle du lâcher) et sans que l'objet n'ait jamais bougé. Absent quand le film
//	             n'en porte aucun — c'est le cas d'un Super Fiesta sur variante Forge, qui n'a
//	             aucun rack de carte et 82,3 % de lâchers.
//	padPickups   les occupations qui se sont ACHEVÉES : le socle s'est vidé quelque part dans
//	             [tLow, tHigh]. Un INTERVALLE et non un instant. Absent quand aucun socle ne
//	             s'est vidé.

// WeaponPad est UN socle d'arme du match : une position où une arme de la même famille
// réapparaît, avec ses apparitions, ses intervalles de présence et son cycle quand il est établi.
//
// UNE DONNÉE DE MATCH, PAS DE CARTE : la position est celle du match mesuré (elle se retrouve au
// centimètre d'un film à l'autre), la FAMILLE ne l'est pas — deux matchs de la même carte y font
// apparaître deux armes différentes.
type WeaponPad struct {
	// X / Y : la position du socle, en coordonnées monde (mêmes axes que Point.X/Y) — le
	// CENTROÏDE des apparitions agglomérées. Z : l'altitude, gratuite (le même record la
	// porte) et publiée pour la cohérence d'étage.
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z,omitempty"`
	// Weapon est la FAMILLE d'arme (high-32 du weapon-id), en hexadécimal — même écriture que
	// `Loadout.W`, donc même clé dans `weaponLabels`. Le tag brut reste À CÔTÉ du libellé,
	// jamais à sa place : une famille absente de la table garde son hexadécimal à l'écran et
	// n'emprunte pas le nom d'une arme voisine.
	Weapon string `json:"weapon"`
	// Spawns porte les instants d'APPARITION d'une arme sur ce socle, en frames (même axe que
	// Point.T), triés.
	Spawns []int `json:"spawns"`
	// Presence porte, pour chaque apparition, ce que le film permet d'affirmer de sa présence.
	Presence []PadPresence `json:"presence"`
	// Cycle est le délai de réapparition du socle, mesuré DEPUIS la disparition précédente.
	// La CLÉ EST ABSENTE quand le cycle n'est pas ÉTABLI — jamais un chiffre instable publié
	// comme s'il était stable. C'est la décision 3 du plan, et elle coûte : 24 socles sur 57
	// seulement portent un cycle.
	//
	// ABSENT, ET NON `null` : LE COMMENTAIRE DISAIT `null` JUSQU'AU 2026-08-17, ET C'ÉTAIT FAUX
	// (correctif de revue). Le `null` de `PadPickup.XUID` se voit parce qu'un `*string` se décrit
	// `["string", "null"]` au contrat ; un pointeur de STRUCT ne le peut pas — le générateur
	// refuse la nullabilité sur un `$ref` (huma v2.39.1 : « nullable is not supported for field
	// 'Cycle' which is type '#/components/schemas/PadCycle' », vérifié en régénérant). Retirer
	// `omitempty` sans pouvoir dire `null` ferait promettre au contrat un objet TOUJOURS présent
	// là où le Go en écrit un sur deux : un mensonge pire que celui qu'on corrige. La clé absente
	// est donc la vérité publiée, et c'est elle que la note UI et le client doivent lire —
	// `cycle == null` en TypeScript couvre les deux, `cycle?.medianS` aussi.
	Cycle *PadCycle `json:"cycle,omitempty"`
}

// PadPresence est UNE occupation du socle : de l'apparition de l'arme à sa disparition, telle
// que le recensement des images-clés la BORNE.
//
// TROIS INSTANTS, ET LEUR SENS N'EST PAS LE MÊME. `T0` est mesuré (le record de création le
// porte). `TLow` est le dernier instant où l'arme est PROUVÉE présente ; `THigh` le premier où
// son absence est prouvée. Entre les deux, le film ne dit rien : les images-clés sont espacées
// de ~20 s (médiane mesurée 20,00 s), et ce calque ne prétend pas être plus fin que sa source.
// Un rendu honnête montre le socle vide dès `TLow` et incertain jusqu'à `THigh`.
type PadPresence struct {
	// T0 : l'apparition, en frames.
	T0 int `json:"t0"`
	// TLow : dernier instant où l'arme est prouvée présente, en frames.
	TLow int `json:"tLow"`
	// THigh : premier instant où son absence est prouvée, en frames. Vaut la fin du rejeu quand
	// l'arme est encore recensée à la DERNIÈRE image-clé du film — elle n'a alors pas été prise,
	// ou elle l'a été après, ce que rien ne dit.
	THigh int `json:"tHigh"`
}

// PadCycle est le délai de réapparition d'un socle, en secondes, mesuré du moment où il se vide
// à la réapparition suivante.
//
// POURQUOI DEPUIS LA DISPARITION ET NON DEPUIS L'APPARITION : l'horloge d'un socle repart quand
// on prend l'arme, pas quand elle apparaît. La mesure le tranche — 24 socles au cycle établi
// contre 4 pour l'horloge d'apparition, aux MÊMES règles de stabilité, et un pic mesuré à 30,5 s
// (55 écarts sur 142 dans la tranche 30-35 s, tenant dans 0,34 s) que l'horloge d'apparition ne
// montre pas. Les armes de puissance en sortent d'elles-mêmes : S7 Sniper 114 à 134 s, Energy
// Sword 194,5 s, Needler 100,9 s.
type PadCycle struct {
	// MedianS / P10S / P90S : la médiane et les déciles des écarts mesurés, en secondes.
	MedianS float32 `json:"medianS"`
	P10S    float32 `json:"p10S"`
	P90S    float32 `json:"p90S"`
	// Gaps est le nombre d'écarts MESURÉS. Un cycle n'est ÉTABLI qu'à partir de deux : un écart
	// unique n'a pas d'écart-type, et rien ne dit qu'il se répète.
	Gaps int `json:"gaps"`
	// Missing est le nombre de réapparitions dont la disparition PRÉCÉDENTE n'est pas datée :
	// autant d'écarts que le socle offrait et que la mesure n'a pas pu prendre.
	//
	// C'EST L'AUTRE MOITIÉ DU DÉNOMINATEUR (correctif de revue du 2026-08-17). Le calcul jetait
	// ce compte, si bien qu'un cycle établi sur 2 écarts pour 8 occupations se lisait « 2 sur 2 »
	// au lieu de « 2 sur 3 » : la même médiane, mais une confiance très différente. Sans
	// `omitempty`, parce qu'un zéro DIT quelque chose : aucune occasion perdue, le cycle porte
	// tout ce que le socle offrait.
	Missing int `json:"missing"`
}

// PadPickup est une occupation de socle qui S'EST ACHEVÉE : le socle s'est vidé quelque part
// dans [TLow, THigh].
//
// C'EST UN INTERVALLE, PAS UN INSTANT, et c'est délibéré. Le film ne porte aucun événement de
// ramassage (mesuré) et le record de suppression d'entité n'est pas isolable : la seule preuve
// de disparition est le recensement des images-clés, espacé de ~20 s. Publier un instant
// donnerait une précision que la source n'a pas.
type PadPickup struct {
	// Pad est l'index du socle dans `weaponPads`.
	Pad int `json:"pad"`
	// TLow / THigh : les bornes de la disparition, en frames (même axe que Point.T).
	TLow  int `json:"tLow"`
	THigh int `json:"tHigh"`
	// XUID est le joueur qui a pris l'arme. RENSEIGNÉ DEPUIS LE SCHÉMA 29 (2026-08-31) quand
	// l'événement natif `biped_pickup` date l'occupation ; `null` sinon, et c'est alors la
	// vérité : le canal natif ne couvre pas cette fenêtre.
	//
	// POURQUOI IL A ÉTÉ VIDE JUSQU'AU SCHÉMA 29. L'oracle indépendant du ramassage était le loadout d'image-clé du
	// ramasseur présumé : porte-t-il l'arme du socle à l'image-clé suivante ? Mesuré deux fois,
	// sur la seule population que la mesure qualifie (les socles) : 111/126 = 88,1 % en suivant
	// le SLOT de vie, 102/128 = 79,7 % en suivant le JOUEUR à travers ses réapparitions (pont
	// `SlotXUID` du constructeur). Le seuil du plan était >= 90 % et il n'a pas été rebaissé.
	//
	// CE QUI L'A LEVÉ, ET LA CONDITION ÉTAIT ÉCRITE ICI : « un oracle plus RAPPROCHÉ que 20 s ».
	// L'événement natif `biped_pickup` (schéma 29) EST cet oracle — daté à la milliseconde, et
	// il PORTE son ramasseur au lieu de le déduire (`512 + sa référence` vaut le slot du
	// ramasseur, exact sur 32/32 paires de vérité terrain, deux films). Il ne s'agit plus d'un
	// oracle à valider mais d'une donnée lue. Cf. pad_pickup_dating.go.
	//
	// POINTEUR, ET SANS `omitempty` : le champ doit se VOIR à `null`. Un client qui ne le
	// trouve pas pourrait croire qu'il a affaire à un artefact plus ancien.
	XUID *string `json:"xuid"`
	// T est l'instant EXACT du ramassage, en frames, quand l'événement natif `biped_pickup`
	// l'a daté (cf. pad_pickup_dating.go). ABSENT quand aucun événement natif ne correspond :
	// l'intervalle `[tLow, tHigh]` reste alors la seule vérité, et il n'est JAMAIS effacé.
	//
	// C'EST LA LEVÉE DE LA RÉSERVE ÉCRITE SUR `XUID` CI-DESSUS. Le contrat de ce champ disait
	// que ce qui manquait n'était pas un meilleur pont mais « un oracle plus RAPPROCHÉ que
	// 20 s ». L'événement natif EST cet oracle : il date à la milliseconde et il porte le
	// ramasseur, sans inférence — `512 + sa référence` vaut le slot du ramasseur sur 32/32
	// paires de vérité terrain. Quand il date une occupation, `xuid` cesse d'être `null`.
	T *int `json:"t,omitempty"`
}
