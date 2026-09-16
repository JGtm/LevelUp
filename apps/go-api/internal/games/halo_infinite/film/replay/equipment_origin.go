package replay

// equipment_origin.go — L'ORIGINE D'UNE POSE D'ÉQUIPEMENT : ce que le film ÉCRIT, et le silence
// publié tel quel.
//
// # CE QUE CE FICHIER DÉCIDE
//
// Une pose publiée porte une ORIGINE — `deployed`, `dropped`, `unknown` — et c'est la seule
// chose qu'un rendu lise pour savoir s'il dessine un geste ou un objet tombé. Jusqu'au
// 2026-09-15 elle se décidait par DEUX règles de secours : le manifeste (`kind = "deployed"` ->
// `deployed` sans mesure, item H.2) puis une FENÊTRE TEMPORELLE de 200 ms entre la création de
// l'objet et la fin de la vie de son poseur. Aucune des deux ne lisait ce que le film écrit.
//
// # LE VOCABULAIRE, TRANCHÉ PAR L'UTILISATEUR LE 2026-09-15
//
// IL SE CALQUE SUR LE MOT DU JEU. Le jeu n'écrit AUCUN événement « equipment drop » — il n'en
// existe pas ; il écrit des COMPOSANTS D'ÉTAT sur l'objet (archétype 37 : `i20
// equipment-deployed`, `i21 equipment-activated`, `i18 item-at-rest`, `i10
// object-parent-state`, `i23 equipment-creator`) plus l'événement d'apparition
// `EquipmentSpawnedObject` (103). La table complète et ce que le décodeur en lit (7 composants
// sur 31) sont au contrat d'[EquipmentPlacement.Origin] et au §4 du plan.
//
//	deployed  une pose que le film DÉSIGNE comme une PIÈCE ENGENDRÉE (événement 103) — les
//	          panneaux du mur, et eux seuls. C'est le seul geste de déploiement que le film
//	          écrive.
//	dropped   un appareil PORTÉ qui tombe. Peu importe POURQUOI : mort de son porteur, ou
//	          échange (il ramasse autre chose et lâche celui-ci). Les deux sont des lâchers, et
//	          l'étiquette ne les distingue pas — la CAUSE se publie dans la couverture
//	          (`coverage.placements.byCause`), jamais dans l'étiquette.
//	unknown   le film ne dit rien de cette pose. Rien ne se devine.
//
// CE QUE CETTE DÉCISION SUPPRIME : un lâcher volontaire à mi-vie sortait `deployed` et faisait
// dessiner un geste qui n'a pas eu lieu (108 poses sur 4 583 au corpus mesuré) ; et une pose dont
// le film ne disait rien recevait quand même une étiquette, par corrélation temporelle
// (649 poses). Les deux ont cessé.
//
// # LES TROIS LECTURES
//
//  1. L'ÉVÉNEMENT 103 `EquipmentSpawnedObject` désigne la vie de l'objet -> `deployed`. Le film
//     dit littéralement « une PIÈCE a été engendrée » ([grammar.ScanEquipmentSpawnEvents]).
//  2. Une MORT ÉCRITE du poseur à l'instant de la pose -> `dropped`. Le film date la mort dans
//     son fil des morts, et le registre d'identité (lots 1.6 / 1.8) la rattache au siège.
//  3. Une PRISE ÉCRITE (`taken` d'`equipmentChanges`) du poseur à l'instant -> `dropped` aussi.
//     Le porteur ramasse autre chose, donc il lâche ce qu'il tenait (rapport E0, question 5).
//
// (2) et (3) simultanées sont une CONTRADICTION, comptée — mais elle n'arbitre rien, les deux
// rendant `dropped` : elle est publiée pour être vue. Mesure : 1 pose sur 4 583.
//
// # LES TOLÉRANCES NE SONT PAS CHOISIES, ELLES SONT LUES (mesure du 2026-09-15, 13 films,
// 4 583 poses, `e191_origine_mesure_research_test.go`)
//
//   - MORT, 200 ms : les poses que la fenêtre publiait `dropped` et dont le poseur a une mort
//     écrite l'ont TOUTES à 171,7 ms au plus (min 5,6 ; p50 37,1) ; la mort la plus proche d'une
//     pose que la fenêtre publiait `deployed` est à 205,3 ms. Aucun recouvrement, un intervalle
//     VIDE de 33,6 ms.
//   - PRISE, 50 ms : 108 poses portent un `taken` du poseur à 50 ms au plus — dont 103 à MOINS
//     D'UNE MILLISECONDE, l'émission est simultanée. À 100 ms la population parasite passe de
//     1 à 8 : la coupure est entre les deux.
//   - DÉSIGNATION 103, 200 ms APRÈS la création : les 115 poses de panneau désignées le sont
//     entre +32,2 et +70,2 ms, toutes POSITIVES (l'événement SUIT la création), et aucun autre
//     objet n'est désigné (0 sur 4 459).
//
// # POURQUOI UNE FENÊTRE POUR UNE DÉSIGNATION, ALORS QUE LA RÉFÉRENCE PORTE UNE CLÉ
//
// La clé d'une vie est la paire (slot, génération) et la GÉNÉRATION NE FAIT QUE DEUX BITS : un
// slot repasse par la même paire plusieurs fois dans un match. Un appariement par clé SEULE fait
// « désigner » 83 poses par 3 événements (mesuré sur `d9781168` le 2026-09-15). Le temps est
// donc la seconde moitié de la clé, et il est lu, pas réglé.
//
// # CE QUI RESTE UN REPLI, NOMMÉ ET COMPTÉ (D14) — UN SEUL
//
// `repli_piece_engendree_sans_evenement` : une pièce engendrée au manifeste qu'aucun 103 ne
// désigne sort quand même `deployed` — elle n'existe QUE déployée, et le manifeste du titre est
// une donnée ÉCRITE, pas une heuristique. 9 poses sur 124, toutes sur les DEUX films de build les
// plus anciens (`a521164d` HI_1_4_1 : 0 événement 103 lu sur 4 956 listes ; `50247b26` v31 sans
// section : 2). C'est une limite de BUILD, pas une incertitude, et son critère de retrait est
// qu'elle tombe à zéro.
//
// LES DEUX AUTRES ONT DISPARU AVEC LEUR CODE, et leurs entrées sont sorties du registre le même
// jour (D14 d) : `repli_origine_pose_fenetre_temporelle` et `repli_origine_pose_vie_la_plus_proche`
// n'existaient que pour faire trancher la fenêtre. Elle ne tranche plus rien.

import (
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/grammar"
)

// equipOwnerWindowUS est la fenêtre dans laquelle un échantillon de bipède est jugé
// contemporain de la pose : 250 ms, la même borne que celle qui sépare deux vies de projectile.
const equipOwnerWindowUS = 250_000

// equipOwnerMaxDist est la distance MAXIMALE, en mètres, entre une pose et son poseur. Le
// seuil est énoncé avant la mesure (plan, décision 2) et la mesure le confirme largement :
// médiane 0,56 m, p90 0,73 à 0,86 m sur les films d'arène. Au-delà de 3 m, la proximité ne
// veut plus rien dire — c'est le cas des objets du monde (bonus, socles), qui n'ont pas de
// poseur et ne doivent pas s'en voir attribuer un.
const equipOwnerMaxDist = 3.0

// equipHeadingWindowUS borne l'écart entre la pose et la lecture de visée qui lui donne son
// cap. 200 ms : plus court que la fenêtre du poseur, parce qu'un cap vieilli est faux là où
// une position vieillie reste juste (on tourne plus vite qu'on ne se déplace).
const equipHeadingWindowUS = 200_000

// Les trois ORIGINES publiées. Liste fermée, vocabulaire du document (cf.
// EquipmentPlacement.Origin) : un client qui lit une quatrième valeur doit la traiter comme
// inconnue, jamais la rapprocher d'une voisine.
const (
	// OriginDeployed : créé en cours de vie du poseur — le geste.
	OriginDeployed = "deployed"
	// OriginDropped : créé à la fin de la vie du poseur — les objets qu'il portait.
	OriginDropped = "dropped"
	// OriginUnknown : aucun poseur mesuré.
	OriginUnknown = "unknown"
)

// originDropWindowUS est un écart de DEUX FRAMES de la grille du document — 200 ms.
//
// ELLE NE CLASSE PLUS AUCUNE POSE D'ÉQUIPEMENT (décision utilisateur du 2026-09-15, lot 1.9.1).
// Elle l'a fait jusqu'à ce jour : « créé à moins de 200 ms de la fin de la vie du poseur » valait
// `dropped`, au-delà `deployed`. C'était une CORRÉLATION, et l'étiquette qu'elle posait n'était
// pas gagnée : l'origine se lit désormais sur ce que le film ÉCRIT (cf. l'en-tête de ce fichier),
// et une pose dont il ne dit rien sort `unknown`.
//
// ELLE SURVIT AU PAQUET, avec sa mesure, parce que DEUX AUTRES chaînes posent leur propre
// question avec elle : les ARMES AU SOL (`gwPickupTrackTolUS`, `gwPadsClass` — l'arme tombée
// d'un porteur qui meurt) et les instruments de recherche qui rejouent l'ancienne règle. La
// mesure qui la fonde LÀ-BAS est la même : les lâchers tombent à 20-40 ms du dernier point,
// les créations de socle à des dizaines de secondes.
const originDropWindowUS = 200_000

// originDropMaxDist est la distance MAXIMALE, en mètres, entre un objet créé et la dernière
// position de celui qui le portait. 1,5 m — le seuil du plan des poses, écrit avant la mesure,
// validé des deux côtés à l'époque (lâchers à 0,63 m de médiane, déploiements à 5,6 à 21,3 m).
//
// ELLE NE SERT PLUS À CLASSER UNE POSE D'ÉQUIPEMENT (item F.1, 2026-09-13) :
// [equipmentOrigineParFenetre] ne pose plus qu'une question temporelle — et depuis le lot 1.9.1
// elle n'est même plus la décision —, parce que cette clause promouvait `deployed` des
// lâchers à la mort dont le corps avait glissé (8 appareils de mur sur 295 poses, mesure E0 du
// 2026-09-10 ; 15 poses sur 5 761 au corpus de F.0). La constante reste parce que TROIS AUTRES
// chaînes du paquet posent leur propre question avec elle — les ARMES AU SOL
// (`ground_weapon_rules.go`, le socle d'où l'arme vient), le DRAPEAU (`flag_objects.go`,
// `flag_carries_lives.go`) et le CRÂNE. Aucune ne classe une pose d'équipement.
const originDropMaxDist = 1.5

// equipLife est une vie de bipède : les positions d'un même slot sans trou majeur, réduites à
// quand elle finit et où.
type equipLife struct {
	from, to uint64
	// x, y, z est la DERNIÈRE position répliquée de la vie : là où le poseur s'arrête.
	//
	// PLUS AUCUN LECTEUR CÔTÉ ÉQUIPEMENT depuis le 2026-09-13 (item F.1) : l'origine d'une pose
	// d'équipement est une question purement TEMPORELLE. Ces trois champs servent la chaîne des
	// ARMES AU SOL (`gwPadsClass`, `ground_weapon_objects.go`), qui pose une autre question — le
	// socle d'où l'arme vient — et pour laquelle le lieu de la fin de vie est le fait même.
	x, y, z float32
}

// equipmentLives découpe le nuage des bipèdes en vies par slot.
//
// LE SEUIL N'EST PAS INVENTÉ ICI : `lifeGapUS` (5 s) est celui de lives.go, très au-dessus du
// pas de réplication (~16 ms) et bien en deçà du temps de réapparition mesuré (médiane 8,0 s).
// Le découpage reproduit d'ailleurs le compte de lives.go sur le film de référence (105 vies
// pour 99 slots) — un contrôle gratuit qu'un second découpage du même fait ne divergeait pas.
//
// `positions` doit être TRIÉ par instant (c'est le cas de `sorted` dans BuildFromFilm).
func equipmentLives(positions []grammar.BipedPosition) map[uint32][]equipLife {
	out := make(map[uint32][]equipLife)
	for _, p := range positions {
		if !p.HasWorld {
			continue // sans bornes de carte, ce n'est pas une position
		}
		v := out[p.Slot]
		if n := len(v); n > 0 && p.TimestampUS-v[n-1].to <= lifeGapUS {
			v[n-1].to = p.TimestampUS
			v[n-1].x, v[n-1].y, v[n-1].z = p.X, p.Y, p.Z
			out[p.Slot] = v
			continue
		}
		out[p.Slot] = append(v, equipLife{
			from: p.TimestampUS, to: p.TimestampUS, x: p.X, y: p.Y, z: p.Z,
		})
	}
	return out
}

// Les PROVENANCES publiées d'une origine — `coverage.placements.byCause`. Vocabulaire STABLE du
// document (même règle que `Origin`) : un client qui lit une valeur inconnue la traite comme
// telle, jamais en la rapprochant d'une voisine.
//
// LEUR SOMME VAUT `Placements`, EXACTEMENT, et c'est testé : une provenance qui manquerait
// laisserait une pose sans explication, ce qui est précisément ce que ce lot supprime.
const (
	// CausePoseEvenementEngendre : un événement 103 désigne la vie de l'objet. LECTURE.
	CausePoseEvenementEngendre = "spawn_event"
	// CausePoseMortEcrite : une mort ÉCRITE du poseur couvre l'instant de la pose. LECTURE.
	CausePoseMortEcrite = "death_written"
	// CausePosePriseEcrite : une prise ÉCRITE (`taken`) du poseur couvre l'instant. LECTURE.
	CausePosePriseEcrite = "taken_written"
	// CausePoseContradiction : la mort ET la prise couvrent l'instant. D14 (b) : ce n'est pas un
	// repli, c'est une CONTRADICTION — elle se compte, elle ne disparaît pas. Les deux causes
	// rendent `dropped` de toute façon (décision utilisateur du 2026-09-15), donc la
	// contradiction ne change AUCUNE étiquette : elle est publiée pour être vue, pas arbitrée.
	CausePoseContradiction = "both"
	// CausePoseManifeste : pièce engendrée qu'aucun 103 ne désigne — REPLI compté, le manifeste
	// décide (une pièce engendrée n'existe que déployée).
	CausePoseManifeste = "manifest_piece"
	// CausePoseAucunSignal : un poseur EST mesuré, et le film ne dit RIEN de cette pose — ni
	// événement 103, ni mort écrite, ni prise écrite. L'origine sort `unknown`.
	//
	// ELLE REMPLACE UNE DÉCISION (décision utilisateur du 2026-09-15). Jusqu'à ce jour une
	// fenêtre de 200 ms depuis la fin de la vie du poseur tranchait ces poses entre `deployed`
	// et `dropped` ; c'était une corrélation, pas une lecture, et l'étiquette qu'elle posait
	// n'était pas gagnée. Ce silence se publie désormais tel quel.
	CausePoseAucunSignal = "none"
	// CausePoseSansPoseur : aucun bipède contemporain assez proche. Second silence, distinct du
	// précédent : ici on ne sait même pas DE QUI parler. L'origine sort `unknown` aussi.
	CausePoseSansPoseur = "no_owner"
)

// equipmentSpawnMatchUS borne l'écart entre la CRÉATION d'un objet et l'événement 103 qui porte
// la même clé de vie. 200 ms, APRÈS la création seulement.
//
// ELLE EST LUE, PAS RÉGLÉE (mesure du 2026-09-15) : les 115 poses de panneau désignées du corpus
// le sont entre +32,2 et +70,2 ms, toutes positives ; aucune ne se présente avant la création, et
// l'écart suivant d'une clé rebouclée se compte en dizaines de secondes. Elle existe PARCE QUE la
// génération ne fait que 2 bits (cf. l'en-tête de ce fichier) : sans elle, 3 événements
// « désignent » 83 poses.
const equipmentSpawnMatchUS = 200_000

// equipmentDeathMatchUS borne l'écart entre la pose et la MORT ÉCRITE de son poseur. 200 ms,
// mesurés : max 171,7 ms du côté des lâchers, min 205,3 ms du côté des déploiements — un
// intervalle VIDE de 33,6 ms sépare les deux populations sur 4 583 poses.
const equipmentDeathMatchUS = 200_000

// equipmentTakenMatchUS borne l'écart entre la pose et la PRISE ÉCRITE de son poseur. 50 ms :
// 103 des 108 prises retenues sont à MOINS D'UNE MILLISECONDE (l'émission est simultanée), et la
// première prise d'une pose `dropped` n'apparaît qu'au-delà — à 100 ms la population parasite
// passe de 1 à 8.
const equipmentTakenMatchUS = 50_000

// poseOrigineSource porte les trois signaux ÉCRITS que la cascade lit, indexés pour la lecture.
//
// UN STRUCT PLUTÔT QUE TROIS PARAMÈTRES : la limite du dépôt est de cinq, et `origineDeLaPose` en
// porterait sept. Il est construit une fois par cuisson ([nouvelleSourceOrigine]) et partagé.
type poseOrigineSource struct {
	// spawns : par clé de vie d'objet, les instants des événements 103 qui la désignent.
	spawns map[grammar.EquipmentLifeKey][]uint64
	// morts : par SIÈGE, les instants des morts ÉCRITES (fil des morts apparié aux vies par le
	// registre d'identité — lots 1.6 et 1.8, seul producteur de liens).
	morts map[uint32][]uint64
	// prises : par SIÈGE, les instants des `taken` d'`equipmentChanges`.
	prises map[uint32][]uint64
	// evenements est le nombre d'événements 103 lus dans le film — le DÉNOMINATEUR sans lequel
	// un zéro de désignation ne se distingue pas d'un film muet.
	evenements int
}

// nouvelleSourceOrigine indexe les trois signaux écrits d'une cuisson.
func nouvelleSourceOrigine(
	spawns []grammar.EquipmentSpawnEvent, vies []lifeSpan, changes []grammar.EquipmentChange,
) poseOrigineSource {
	src := poseOrigineSource{
		spawns: map[grammar.EquipmentLifeKey][]uint64{},
		morts:  map[uint32][]uint64{},
		prises: map[uint32][]uint64{},
	}
	for _, e := range spawns {
		src.evenements++
		if e.SpawnedValid {
			src.spawns[e.Spawned] = append(src.spawns[e.Spawned], e.TimestampUS)
		}
	}
	for _, v := range vies {
		if v.cause == CauseVieMort && v.to >= 0 {
			src.morts[v.slot] = append(src.morts[v.slot], uint64(v.to))
		}
	}
	for _, c := range changes {
		if c.Kind == grammar.EquipmentTaken {
			src.prises[c.Slot] = append(src.prises[c.Slot], c.TimestampUS)
		}
	}
	return src
}

// designeParUnEvenement dit si un événement 103 désigne la vie de CET objet : même clé
// (slot, génération), et l'événement SUIT la création d'au plus [equipmentSpawnMatchUS].
func (s poseOrigineSource) designeParUnEvenement(p grammar.EquipmentPlacement) bool {
	for _, at := range s.spawns[p.Life] {
		if at >= p.T0US && at-p.T0US <= equipmentSpawnMatchUS {
			return true
		}
	}
	return false
}

// couvreUnInstant dit si l'un des instants donnés est à moins de `tol` de `at`.
func couvreUnInstant(instants []uint64, at, tol uint64) bool {
	for _, t := range instants {
		if equipTimeGap(t, at) <= tol {
			return true
		}
	}
	return false
}

// poseOwner regroupe ce que la cascade sait du POSEUR d'une pose, plus les signaux écrits de la
// cuisson. Un struct plutôt que quatre paramètres de plus (limite de cinq du dépôt).
type poseOwner struct {
	src        poseOrigineSource
	slot       uint32
	avecPoseur bool
}

// origineDeLaPose rend l'ORIGINE d'une pose et la PROVENANCE de cette décision — la cascade que
// l'en-tête de ce fichier décrit : LIRE, et publier le silence quand il n'y a rien à lire.
//
// L'ORDRE N'EST PAS ARBITRAIRE, ET CHACUNE DE SES MARCHES A SA RAISON :
//
//  1. LA DÉSIGNATION PAR LE 103 NE DEMANDE AUCUN POSEUR : le film parle de l'OBJET, pas de qui
//     l'a posé. C'est ce qui lui permet de trancher les poses de panneau SANS poseur mesuré,
//     que la mesure temporelle rendait `unknown` (4 cas sur le corpus de F.0).
//  2. UNE PIÈCE ENGENDRÉE SORT AVANT LES DEUX AUTRES LECTURES, et pas après : « mort du
//     porteur » et « prise du porteur » répondent à une question qu'un panneau NE POSE PAS —
//     il n'entre jamais dans un inventaire. Le manifeste du titre en décide, et c'est le SEUL
//     endroit où `deployed` se publie sans un 103 : repli NOMMÉ et COMPTÉ, 9 poses sur 124 au
//     corpus, toutes sur les deux builds les plus anciens.
//  3. ENSUITE SEULEMENT les deux lectures de l'APPAREIL PORTÉ, qui exigent un poseur — et qui
//     rendent TOUTES DEUX `dropped` (décision utilisateur du 2026-09-15).
//  4. AUCUN SIGNAL : `unknown`. Rien ne se devine.
func origineDeLaPose(
	p grammar.EquipmentPlacement, id string, o poseOwner, fb *fallback.Compteur,
) (origine, cause string) {
	if o.src.designeParUnEvenement(p) {
		return OriginDeployed, CausePoseEvenementEngendre
	}
	if equipmentIsSpawnedPiece(id) {
		fb.Declenche(fallback.NomPieceEngendreeSansEvenement)
		return OriginDeployed, CausePoseManifeste
	}
	if !o.avecPoseur {
		return OriginUnknown, CausePoseSansPoseur
	}
	mort := couvreUnInstant(o.src.morts[o.slot], p.T0US, equipmentDeathMatchUS)
	prise := couvreUnInstant(o.src.prises[o.slot], p.T0US, equipmentTakenMatchUS)
	switch {
	case mort && prise:
		return OriginDropped, CausePoseContradiction
	case mort:
		return OriginDropped, CausePoseMortEcrite
	case prise:
		return OriginDropped, CausePosePriseEcrite
	}
	return OriginUnknown, CausePoseAucunSignal
}

// equipmentOwner rend le bipède le plus proche de la pose dans la fenêtre temporelle, à
// condition qu'il soit à moins d'equipOwnerMaxDist mètres. Rend aussi, quand elle existe, la
// lecture de VISÉE la plus proche en temps du même slot — c'est elle qui porte le cap.
//
// UN ÉCHANTILLON PAR SLOT, LE PLUS PROCHE EN TEMPS : plusieurs records d'un même bipède
// tombent dans la fenêtre, et retenir le plus proche en ESPACE au lieu du plus proche en
// TEMPS ferait gagner le joueur qui passe par là au bon moment plutôt que celui qui pose.
func equipmentOwner(
	positions []grammar.BipedPosition, p grammar.EquipmentPlacement,
) (slot uint32, heading *float32, ok bool) {
	lo := sort.Search(len(positions), func(k int) bool {
		return positions[k].TimestampUS+equipOwnerWindowUS >= p.T0US
	})
	best := map[uint32]grammar.BipedPosition{}
	aim := map[uint32]grammar.BipedPosition{}
	for k := lo; k < len(positions) && positions[k].TimestampUS <= p.T0US+equipOwnerWindowUS; k++ {
		s := positions[k]
		if !s.HasWorld {
			continue // sans bornes de carte, la distance n'est pas une distance
		}
		if b, seen := best[s.Slot]; !seen || equipCloser(s, b, p.T0US) {
			best[s.Slot] = s
		}
		if !s.HasYaw || equipTimeGap(s.TimestampUS, p.T0US) > equipHeadingWindowUS {
			continue
		}
		if b, seen := aim[s.Slot]; !seen || equipCloser(s, b, p.T0US) {
			aim[s.Slot] = s
		}
	}
	// LE PLUS PROCHE, ET À ÉGALITÉ LE PLUS PETIT SLOT (correction du 2026-09-02, item 0.4bis
	// étendu de PLAN_CUISSON_PERF). `best` est une MAP : sans le second critère, deux bipèdes à
	// la MÊME distance de la pose — des coordonnées quantifiées, donc des égalités exactes, et un
	// film BTB à 26 joueurs en réveille — laissaient l'ordre d'itération, tiré au sort à chaque
	// exécution, nommer le poseur publié. Le départage vient du slot, une donnée de l'élément.
	var near grammar.BipedPosition
	for _, s := range best {
		d := equipDist(p, s)
		if d > equipOwnerMaxDist {
			continue
		}
		if nd := equipDist(p, near); !ok || d < nd || (d == nd && s.Slot < near.Slot) {
			near, ok = s, true
		}
	}
	if !ok {
		return 0, nil, false
	}
	// Le CAP vient de la lecture de visée du MÊME slot la plus proche en temps. Jamais d'un
	// autre slot, et jamais d'une lecture trop vieille : on tourne plus vite qu'on ne marche.
	if a, seen := aim[near.Slot]; seen {
		if h, valid := a.AimHeadingDeg(); valid {
			heading = &h
		}
	}
	return near.Slot, heading, true
}

// equipCloser dit si a est plus proche de `at` en TEMPS que b.
func equipCloser(a, b grammar.BipedPosition, at uint64) bool {
	return equipTimeGap(a.TimestampUS, at) < equipTimeGap(b.TimestampUS, at)
}

// equipDist n'est qu'un ADAPTATEUR de types vers la distance canonique du paquet (`dist3`) : la
// formule ne se réécrit pas ici, elle n'est écrite qu'une fois.
func equipDist(p grammar.EquipmentPlacement, s grammar.BipedPosition) float32 {
	return float32(dist3([3]float32{p.X, p.Y, p.Z}, [3]float32{s.X, s.Y, s.Z}))
}

func equipTimeGap(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return b - a
}
