package replay

import (
	"fmt"
	"log/slog"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/filmsource"
)

// equipment_placements.go — LES POSES d'équipement sur la carte : le mur de protection, le
// capteur de menaces, et tout ce que l'archétype d'équipement porte sans qu'on le nomme.
//
// D'OÙ VIENT LA DONNÉE (2026-08-17/18, PLAN_IDENTITE_TI37 gates 0-1 puis
// PLAN_POSES_EQUIPEMENT_PUBLICATION) : le record de CRÉATION d'une entité `ti=37` porte, dans
// son bloc `object-multiplayer-properties`, le GlobalID du tag `eqip` de l'objet — les
// 21 valeurs du corpus se résolvent toutes dans le groupe `eqip` du jeu. Le MÊME record porte
// la position i0, c'est-à-dire le lieu exact de la pose. `t1` vient de la trajectoire décodée
// des paquets delta. `filmdec.ScanFilmEquipmentPlacements` rend le tout.
//
// `t1` N'EST PAS LA DISPARITION, et c'est mesuré (2026-08-18, `filmdec/equipment_life_end_test
// .go`) : le décodage ne suit que les records qui portent une position, donc `t1` date l'instant
// où l'objet S'IMMOBILISE. Le recensement des keyframes prouve que l'entité survit à ce moment
// (101 poses sur 295 du film 000d5950, 228 sur 537 de 00ba2e1c, encore recensées plus d'une
// seconde après), et aucune fin explicite n'est isolable — ni record de suppression, ni queue
// de records sans position (les deux sont du bruit au témoin). Le film porte donc une BORNE
// INFÉRIEURE ; ce qu'un rendu en fait est une décision de rendu, jamais une lecture de `t1`.
//
// CE QUE CETTE COUCHE AJOUTE, et pourquoi c'est ici : le POSEUR et son CAP. Ni l'un ni l'autre
// n'est écrit dans le record — le champ de référence d'entité du default-state est une porte
// FERMÉE sur 503 records sur 503. Le poseur se MESURE : c'est le bipède le plus proche à
// l'instant de la pose. La mesure du corpus le justifie et le borne — médiane 0,52 à 0,60 m
// sur 11 films, contre 11 à 36 m pour le témoin (un autre bipède vivant au même instant).
// Le cap est celui de la VISÉE de ce poseur au même instant (i21, déjà décodé par CaptureDirs).
//
// CE QU'ON NE PRÉTEND PAS. `poseHeading` n'est PAS l'orientation de l'objet : le record n'en
// porte aucune. C'est là où le poseur REGARDAIT quand il a posé — une mesure, dont le rendu
// peut se servir pour orienter un mur, en sachant ce qu'elle est.

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

// EquipmentPlacement est UNE pose d'équipement, datée et située.
type EquipmentPlacement struct {
	// T0 est l'instant de CRÉATION de l'objet — le geste de pose — sur le même axe que
	// Point.T. T1 est le DERNIER POINT DE POSITION de sa vie décodée : la fin de son
	// MOUVEMENT RÉPLIQUÉ, c'est-à-dire une BORNE INFÉRIEURE de sa durée de vie, PAS sa
	// disparition. Le film ne date la disparition d'AUCUN objet d'équipement (mesure du
	// 2026-08-18) ; un client qui efface la pose à T1 affirme une disparition que rien ne
	// mesure.
	//
	// LE COMMENTAIRE A DIT « la disparition » JUSQU'AU 2026-08-18, ET C'ÉTAIT FAUX. Un
	// encodage delta ne transmet que ce qui CHANGE : un objet posé qui s'immobilise cesse
	// d'être transmis, et sa dernière position transmise n'a rien à voir avec sa fin de vie.
	// La preuve est dans les grenades, et elle corrobore leur identification — spike 1,2 s
	// (elle colle à l'impact) < dynamo 1,9 < plasma 3,5 < frag 4,1 (elle rebondit ET roule) ;
	// idem l'appareil du mur 0,7-0,9 s (son vol) contre ses panneaux 0,5 s (déployés sur
	// place). Conséquence pour le rendu : dessiner une pose sur le seul [T0, T1] affiche un
	// détecteur ~2 s là où le jeu le garde 15 s. La durée RÉELLE demanderait le record de
	// suppression de l'entité, cherché le 2026-08-18 et NON isolable (registre des reports).
	T0 int `json:"t0"`
	T1 int `json:"t1"`
	// X / Y : la position de la pose, en coordonnées monde (mêmes axes que Point.X/Y).
	X float32 `json:"x"`
	Y float32 `json:"y"`
	// Z : l'altitude de la pose. Gratuite (le même record la porte), publiée pour la
	// cohérence d'étage. PIÈGE omitempty accepté, même argument que GrappleLine.AZ.
	Z float32 `json:"z,omitempty"`
	// Family est la famille de rendu : `wall`, `sensor`, ou `other`. Identifiant STABLE du
	// document (même règle que NeutralDeath.Kind) — les libellés vivent dans l'i18n du
	// client, jamais ici. `other` est un RÉSULTAT, pas un défaut d'analyse : l'objet est
	// bien posé, sa nature n'est pas établie.
	Family string `json:"family"`
	// ID est le GlobalID du tag `eqip`, en hexadécimal. Publié même pour `other` : c'est
	// l'identité que le jeu donne à l'objet, et c'est ce qui permettra de nommer plus tard
	// sans re-cuire les artefacts (le client peut regrouper par identifiant).
	ID string `json:"id"`
	// Owner est le SLOT du poseur — donc une VIE, pas un joueur (même règle que les autres
	// calques : le slot migre aux réapparitions). Vaut -1 quand aucun bipède contemporain
	// n'est assez proche : c'est le cas des objets du monde, et c'est une mesure.
	Owner int `json:"owner"`
	// H est le cap de VISÉE du poseur à l'instant de la pose, en degrés [0,360[ — même
	// convention que Point.H. Absent quand aucune lecture de visée n'est contemporaine, ou
	// quand la pose n'a pas de poseur. POINTEUR, PAS float32 : un cap de zéro est une
	// valeur, et omitempty l'effacerait (même piège qu'OriginMs).
	H *float32 `json:"h,omitempty"`
	// Origin est l'ORIGINE MESURÉE de la pose. Identifiant STABLE du document (même règle
	// que Family et NeutralDeath.Kind) :
	//
	//	deployed  l'objet a été créé EN COURS DE VIE du poseur : il l'a déployé. C'est la
	//	          seule origine qui décrit un geste, donc la seule que le rendu dessine.
	//	dropped   l'objet a été créé à l'instant ET à l'endroit où la vie du poseur
	//	          S'ACHÈVE : ce sont les objets qu'il PORTAIT, relâchés. Une mort en Halo
	//	          libère les grenades et l'équipement du joueur, et c'est ce que la mesure
	//	          voit — pas une pose sur la carte.
	//	unknown   aucun poseur mesuré (aucun bipède contemporain à moins de 3 m). L'origine
	//	          n'est pas établie, et elle ne se devine pas.
	//
	// CE CHAMP EXISTE PARCE QUE `equipmentPlacements` N'EST PAS CE QUE SON NOM DIT : sur les
	// 11 films calibrés, 3 242 des 3 661 poses à poseur mesuré (88,6 %) naissent dans les
	// 2 frames qui suivent le dernier point de leur poseur. Dessiner un arc de mur à ces
	// positions dessinerait un mur là où personne n'en a déployé.
	//
	// OPTIONNEL, ET CE N'EST PAS UNE FAIBLESSE DU CONTRAT — C'EST LA VÉRITÉ SUR LE PARC.
	// Les artefacts antérieurs au schéma 10 portent des poses SANS origine (ils sont encore sur
	// disque et en production jusqu'à la re-cuisson complète). Déclarer le champ REQUIS aurait
	// fait promettre au contrat une clé que ces artefacts n'ont pas. Le builder, lui, le
	// renseigne TOUJOURS — jamais la chaîne vide — donc un artefact de schéma 10 le porte
	// systématiquement. **Absent = origine non mesurée : le client lit `unknown`, JAMAIS
	// `deployed`.** Un repli sur `deployed` ferait dessiner, sur tout le parc non re-cuit,
	// exactement le mur fantôme que ce champ existe pour supprimer.
	Origin string `json:"origin,omitempty"`
	// Until / UntilMax / End (schéma 28) : la FIN D'AFFICHAGE OBSERVÉE — ce que T1 n'a jamais
	// été (T1 est la fin du MOUVEMENT, cf. son contrat). Même sémantique que
	// `GroundWeapon.T1/T1Max/End` : pour End == "seen", la disparition est un INTERVALLE
	// mesuré — Until est la dernière image-clé qui recense l'objet (dernière preuve de
	// présence, à défaut sa création) et UntilMax la première qui ne le recense plus. Le
	// client affiche plein jusqu'à Until, dégradé jusqu'à UntilMax, jamais au-delà. Pour
	// End == "open", rien ne prouve la disparition : Until == UntilMax == dernière frame.
	//
	// PAS DE FIN "pickup" POUR L'ÉQUIPEMENT, et c'est une RÉFUTATION mesurée (2026-08-30,
	// ground_link_research_test.go, mesure D) : l'équipement tombe à la mort AVEC les
	// grenades du mort — plusieurs objets au mètre carré — et le lien spatial vers la prise
	// i48 attrape le mauvais objet (matrice GlobalID x rang non diagonale : un même objet lié
	// à trois rangs ; à candidat unique il reste 0 à 2 paires par film, incohérentes). Le
	// ramassage d'équipement reste publié par `equipmentChanges`, qui dit QUI et QUAND — il
	// ne dit juste pas QUEL objet du sol.
	//
	// End vide = artefact antérieur au schéma 28 : aucune fin observée, même règle de lecture
	// qu'Origin absent.
	Until    int    `json:"until,omitempty"`
	UntilMax int    `json:"untilMax,omitempty"`
	End      string `json:"end,omitempty"`
}

// EquipmentPlacementCoverage dit ce que le calque a lu et ce qu'il en a publié.
//
// LA CALIBRATION Y FIGURE, et c'est le point : la largeur du bloc `object-multiplayer-
// properties` varie d'un film à l'autre et se MESURE dans le film. Publier les poses sans
// publier la mesure qui les rend lisibles laisserait croire qu'elles tombent d'une constante.
type EquipmentPlacementCoverage struct {
	// Scanned dit que le film a été BALAYÉ jusqu'au bout. Faux : il n'a pas pu l'être du tout
	// (chunks illisibles, bornes de carte absentes, archétype absent des keyframes) — ou il
	// n'y a pas eu de film (assemblage sur positions figées). Sans lui, `calibrated: false`
	// couvrait DEUX pannes distinctes qui se lisaient pareil : un film illisible et un film
	// dont la calibration refuse de trancher rendent tous deux zéro pose.
	Scanned bool `json:"scanned"`
	// Widths est le découpage retenu (« lead/index »), vide si la calibration a échoué.
	Widths string `json:"widths,omitempty"`
	// Calibrated dit si le film a tranché. Faux : aucune pose n'est publiée, et c'est
	// délibéré — un balayage à la mauvaise largeur ne rend pas des poses, il rend du bruit.
	Calibrated bool `json:"calibrated"`
	// Lives est le nombre de vies d'objet d'équipement décodées ; Anchors le nombre
	// d'en-têtes de création reconnus (bruit compris) ; Confirmed ceux que l'oracle de
	// position a validés. L'écart entre les trois dit la sélectivité réelle.
	Lives     int `json:"lives"`
	Anchors   int `json:"anchors"`
	Confirmed int `json:"confirmed"`
	// Placements est le nombre de poses publiées ; Named celles dont la famille est établie
	// (wall ou sensor) ; Other les autres.
	Placements int `json:"placements"`
	Named      int `json:"named"`
	Other      int `json:"other"`
	// WithOwner / WithHeading : poses dont le poseur, puis le cap, ont été mesurés.
	WithOwner   int `json:"withOwner"`
	WithHeading int `json:"withHeading"`
	// ByFamily compte les poses par famille — le détail que `Named` résume.
	ByFamily map[string]int `json:"byFamily,omitempty"`
	// Deployed / Dropped / Unknown : les poses par ORIGINE MESURÉE (schéma 10, cf.
	// EquipmentPlacement.Origin). Les trois sont publiés, et l'invariant qui les rend
	// vérifiables est testé : `Deployed + Dropped + Unknown == Placements`, exactement.
	//
	// POURQUOI LES TROIS ET PAS SEULEMENT `Deployed` : c'est le seul endroit qui dit ce que
	// le rendu ÉCARTE. Publier « 120 poses déployées » sans les 91 lâchers et les 11 sans
	// poseur laisserait lire 120 comme un total, alors que le mur en porte 222.
	Deployed int `json:"deployed"`
	Dropped  int `json:"dropped"`
	Unknown  int `json:"unknown"`
	// EndSeen / EndOpen (schema 28) : les fins d affichage observees — disparition bornee par
	// le recensement, ou rien ne prouve la disparition. Somme == Placements sur un artefact 27.
	EndSeen int `json:"endSeen"`
	EndOpen int `json:"endOpen"`
	// ByFamilyOrigin est le CROISEMENT famille x origine, clé `"<famille>/<origine>"` — la
	// table que le lot de rendu doit voir avant de dessiner. Une clé composée plutôt qu'une
	// carte de cartes : le contrat reste `additionalProperties: integer`, et la lecture reste
	// une seule indirection côté client.
	ByFamilyOrigin map[string]int `json:"byFamilyOrigin,omitempty"`
}

// equipmentFamilyOther est la famille par défaut : un objet dont la nature n'est pas établie.
const equipmentFamilyOther = "other"

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

// originDropWindowUS est l'écart MAXIMAL entre la création de l'objet et le dernier point de
// position de son poseur pour que la pose soit un LÂCHER. 200 ms, c'est-à-dire deux frames de
// la grille du document — le seuil du plan, écrit avant la mesure.
//
// LA MESURE LE VALIDE PAR SES DEUX CÔTÉS, et c'est ce qui le rend autre chose qu'un réglage.
// Les lâchers tombent à 20-40 ms du dernier point (médiane par identifiant : 20,5 à 38,3 ms) ;
// les déploiements, eux, sont à 14 à 42 SECONDES de la fin de vie. Trois ordres de grandeur
// séparent les deux populations : n'importe quel seuil entre 1 s et 10 s rendrait le même
// classement, donc celui-ci ne se règle pas, il se constate.
const originDropWindowUS = 200_000

// originDropMaxDist est la distance MAXIMALE, en mètres, entre la pose et la dernière position
// du poseur pour que la pose soit un LÂCHER. 1,5 m — le seuil du plan, écrit avant la mesure,
// et là encore validé des deux côtés : les lâchers sont à 0,63 m de médiane, les déploiements
// à 5,6 à 21,3 m. Un objet lâché tombe aux pieds de celui qui le portait.
const originDropMaxDist = 1.5

// equipLife est une vie de bipède : les positions d'un même slot sans trou majeur, réduites à
// ce dont l'origine a besoin — quand elle finit, et où.
type equipLife struct {
	from, to uint64
	// x, y, z est la DERNIÈRE position répliquée de la vie : là où le poseur s'arrête.
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
func equipmentLives(positions []filmdec.BipedPosition) map[uint32][]equipLife {
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

// equipmentOrigin classe une pose : lâchée à la fin de la vie du poseur, ou déployée.
//
// LA VIE RETENUE EST CELLE QUI CONTIENT L'INSTANT DE LA POSE, à défaut la plus proche en
// temps : écarter silencieusement une pose dont la vie ne couvre pas l'instant biaiserait la
// mesure vers les cas faciles.
func equipmentOrigin(lives []equipLife, p filmdec.EquipmentPlacement) string {
	if len(lives) == 0 {
		return OriginUnknown
	}
	best, bestGap := equipLife{}, ^uint64(0)
	for _, v := range lives {
		gap := uint64(0)
		switch {
		case p.T0US < v.from:
			gap = v.from - p.T0US
		case p.T0US > v.to:
			gap = p.T0US - v.to
		}
		if gap == 0 { // la vie CONTIENT l'instant : aucune autre ne fera mieux
			best = v
			break
		}
		if gap < bestGap {
			best, bestGap = v, gap
		}
	}
	if equipTimeGap(p.T0US, best.to) > originDropWindowUS {
		return OriginDeployed
	}
	if dist3([3]float32{p.X, p.Y, p.Z}, [3]float32{best.x, best.y, best.z}) >= originDropMaxDist {
		return OriginDeployed
	}
	return OriginDropped
}

// decodeFilmPlacements décode les poses du film et JOURNALISE ce qu'il en est.
//
// TROIS SORTIES, TROIS PHRASES — et la distinction est le point. Un balayage impossible (film
// illisible) n'est pas un film sans équipement, et un film dont le découpage du bloc de
// réplication n'a pas été tranché n'est ni l'un ni l'autre : il aurait des poses, mais les lire
// à une largeur devinée rendrait du bruit. Les trois se lisent au journal, et la troisième se
// relit ensuite dans l'artefact (`coverage.placements.calibrated`).
//
// HORS LIGNE — appelée par BuildFromFilm, sous LockProcessDecode.
func decodeFilmPlacements(
	film *filmsource.Film, matchID string, worldRange *filmdec.Vec3Range,
) ([]filmdec.EquipmentPlacement, filmdec.EquipmentPlacementStats) {
	pl, st, err := filmdec.ScanEquipmentPlacements(film, worldRange)
	switch {
	case err != nil:
		slog.Warn("poses d'equipement illisibles — rejeu sans equipement pose",
			"err", err, "match_id", matchID)
		return nil, st
	case !st.Calibration.Widths.Valid():
		slog.Warn("poses d'equipement : le decoupage du bloc de replication n'a pas ete tranche"+
			" sur ce film — AUCUNE pose publiee plutot que du bruit",
			"match_id", matchID, "ancres", st.Calibration.Anchors,
			"vies", st.Calibration.Lives, "chunksLus", st.Calibration.Chunks)
	default:
		slog.Info("poses d'equipement : records de creation ti=37",
			"decoupage", st.Calibration.Widths.String(), "accords", st.Calibration.Agree,
			"ancres", st.Anchors, "acceptes", st.Accepted,
			"confirmes", st.Confirmed, "poses", st.Placements)
	}
	return pl, st
}

// logPlacementCoverage publie au journal ce que le calque a rendu — les mêmes dénominateurs
// que l'artefact, pour qu'un build se juge sans ouvrir le JSON.
func logPlacementCoverage(c *EquipmentPlacementCoverage) {
	if c == nil {
		return
	}
	slog.Info("rejeu : poses d'equipement",
		"balaye", c.Scanned, "calibre", c.Calibrated, "decoupage", c.Widths, "poses", c.Placements,
		"nommees", c.Named, "autres", c.Other,
		"avecPoseur", c.WithOwner, "avecCap", c.WithHeading,
		"deployees", c.Deployed, "lachees", c.Dropped, "origineInconnue", c.Unknown,
		"finVue", c.EndSeen, "finOuverte", c.EndOpen)
}

// buildEquipmentPlacements assemble les poses : famille par le manifeste, poseur par
// proximité mesurée, cap par la visée du poseur — et, depuis le schéma 28, la FIN OBSERVÉE
// par le recensement des images-clés (`census`, la lecture `ti=37` de la chaîne des socles).
//
// `positions` doit être TRIÉ par instant (c'est le cas de `sorted` dans BuildFromFilm) : la
// recherche du poseur est une fenêtre glissante, pas un balayage complet par pose.
func buildEquipmentPlacements(
	raw []filmdec.EquipmentPlacement, st filmdec.EquipmentPlacementStats,
	positions []filmdec.BipedPosition, clock replayClock, census filmdec.WorldObjectKeyframes,
) ([]EquipmentPlacement, *EquipmentPlacementCoverage) {
	cov := &EquipmentPlacementCoverage{
		Scanned:        st.Scanned,
		Calibrated:     st.Calibration.Widths.Valid(),
		Lives:          st.Lives,
		Anchors:        st.Anchors,
		Confirmed:      st.Confirmed,
		ByFamily:       map[string]int{},
		ByFamilyOrigin: map[string]int{},
	}
	if cov.Calibrated {
		cov.Widths = st.Calibration.Widths.String()
	}
	if len(raw) == 0 || clock.step == 0 {
		return nil, cov
	}
	lives := equipmentLives(positions)
	ends := placementEnds(raw, census, clock)
	out := make([]EquipmentPlacement, 0, len(raw))
	for i, p := range raw {
		t0 := frameOf(p.T0US, clock.origin, clock.step)
		t1 := frameOf(p.T1US, clock.origin, clock.step)
		if t1 < 0 || t0 >= clock.frames {
			continue // hors de l'axe publié : rien à dessiner
		}
		pl := EquipmentPlacement{
			T0: clampFrame(t0, clock.frames), T1: clampFrame(t1, clock.frames),
			X: p.X, Y: p.Y, Z: p.Z,
			Family: clock.families[p.GlobalID],
			ID:     fmt.Sprintf("0x%08x", p.GlobalID),
			Owner:  -1,
			Origin: OriginUnknown,
			Until:  ends[i].until, UntilMax: ends[i].untilMax, End: ends[i].end,
		}
		if pl.Family == "" {
			pl.Family = equipmentFamilyOther
		}
		if slot, h, ok := equipmentOwner(positions, p); ok {
			pl.Owner, pl.H = int(slot), h
			pl.Origin = equipmentOrigin(lives[slot], p)
		}
		out = append(out, pl)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].T0 != out[j].T0 {
			return out[i].T0 < out[j].T0
		}
		return out[i].ID < out[j].ID
	})
	tallyEquipmentPlacements(out, cov)
	return out, cov
}

// placEnd est la fin d'affichage observée d'UNE pose, sur l'axe du document.
type placEnd struct {
	until, untilMax int
	end             string
}

// placementEnds borne la disparition de chaque pose par le recensement des images-clés —
// mêmes règles que la chaîne des socles (`gwPickupBoundsFrom`, un seul exemplaire), la vie
// d'une clé étant fermée par la pose SUIVANTE de la même clé (le pool de clés reboucle).
//
// Rendu INDEXÉ sur `raw` : l'appelant filtre les poses hors axe après coup, l'index doit
// survivre au filtre.
func placementEnds(
	raw []filmdec.EquipmentPlacement, census filmdec.WorldObjectKeyframes, clock replayClock,
) []placEnd {
	byLife := map[filmdec.EquipmentLifeKey][]int{}
	for i, p := range raw {
		byLife[p.Life] = append(byLife[p.Life], i)
	}
	// +1 : la fenêtre de recensement (`gwPickupSeenWithin`) est EXCLUSIVE sur sa borne haute.
	// À la fin de film exacte, la DERNIÈRE image-clé serait retranchée et une pose encore
	// recensée à cette image-clé sortirait « disparue » au lieu d'« ouverte ».
	filmEnd := census.LastTimeUS() + 1
	out := make([]placEnd, len(raw))
	for life, idxs := range byLife {
		sort.Slice(idxs, func(a, b int) bool { return raw[idxs[a]].T0US < raw[idxs[b]].T0US })
		for j, i := range idxs {
			lifeEnd := filmEnd
			if j+1 < len(idxs) {
				lifeEnd = raw[idxs[j+1]].T0US
			}
			seen := gwPickupSeenWithin(census.SeenUS[life], raw[i].T0US, lifeEnd)
			b := gwPickupBoundsFrom(raw[i].T0US, lifeEnd, filmEnd, census.TimesUS, seen)
			switch {
			case b.NeverPicked || b.NoLaterKF:
				out[i] = placEnd{
					until: clock.frames - 1, untilMax: clock.frames - 1,
					end: GroundWeaponEndOpen,
				}
			default:
				e := placEnd{
					until:    clampFrame(frameOf(b.LowUS, clock.origin, clock.step), clock.frames),
					untilMax: clampFrame(frameOf(b.HighUS, clock.origin, clock.step), clock.frames),
					end:      GroundWeaponEndSeen,
				}
				if e.untilMax < e.until {
					e.untilMax = e.until
				}
				out[i] = e
			}
		}
	}
	return out
}

// replayClock porte l'axe de temps et la table de familles du document (règle des
// 5 paramètres — ces quatre-là voyagent toujours ensemble).
type replayClock struct {
	origin, step uint64
	frames       int
	families     map[uint32]string
}

func clampFrame(t, frames int) int {
	switch {
	case t < 0:
		return 0
	case t >= frames:
		return frames - 1
	}
	return t
}

func tallyEquipmentPlacements(out []EquipmentPlacement, cov *EquipmentPlacementCoverage) {
	cov.Placements = len(out)
	for _, p := range out {
		cov.ByFamily[p.Family]++
		cov.ByFamilyOrigin[p.Family+"/"+p.Origin]++
		if p.Family == equipmentFamilyOther {
			cov.Other++
		} else {
			cov.Named++
		}
		if p.Owner >= 0 {
			cov.WithOwner++
		}
		if p.H != nil {
			cov.WithHeading++
		}
		switch p.Origin {
		case OriginDeployed:
			cov.Deployed++
		case OriginDropped:
			cov.Dropped++
		default:
			cov.Unknown++
		}
		switch p.End {
		case GroundWeaponEndSeen:
			cov.EndSeen++
		case GroundWeaponEndOpen:
			cov.EndOpen++
		}
	}
}

// equipmentOwner rend le bipède le plus proche de la pose dans la fenêtre temporelle, à
// condition qu'il soit à moins d'equipOwnerMaxDist mètres. Rend aussi, quand elle existe, la
// lecture de VISÉE la plus proche en temps du même slot — c'est elle qui porte le cap.
//
// UN ÉCHANTILLON PAR SLOT, LE PLUS PROCHE EN TEMPS : plusieurs records d'un même bipède
// tombent dans la fenêtre, et retenir le plus proche en ESPACE au lieu du plus proche en
// TEMPS ferait gagner le joueur qui passe par là au bon moment plutôt que celui qui pose.
func equipmentOwner(
	positions []filmdec.BipedPosition, p filmdec.EquipmentPlacement,
) (slot uint32, heading *float32, ok bool) {
	lo := sort.Search(len(positions), func(k int) bool {
		return positions[k].TimestampUS+equipOwnerWindowUS >= p.T0US
	})
	best := map[uint32]filmdec.BipedPosition{}
	aim := map[uint32]filmdec.BipedPosition{}
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
	var near filmdec.BipedPosition
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
func equipCloser(a, b filmdec.BipedPosition, at uint64) bool {
	return equipTimeGap(a.TimestampUS, at) < equipTimeGap(b.TimestampUS, at)
}

// equipDist n'est qu'un ADAPTATEUR de types vers la distance canonique du paquet (`dist3`) : la
// formule ne se réécrit pas ici, elle n'est écrite qu'une fois.
func equipDist(p filmdec.EquipmentPlacement, s filmdec.BipedPosition) float32 {
	return float32(dist3([3]float32{p.X, p.Y, p.Z}, [3]float32{s.X, s.Y, s.Z}))
}

func equipTimeGap(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return b - a
}
