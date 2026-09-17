package replay

import (
	"fmt"
	"log/slog"
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// equipment_placements.go — LES POSES d'équipement sur la carte : le mur de protection, le
// capteur de menaces, et tout ce que l'archétype d'équipement porte sans qu'on le nomme.
//
// D'OÙ VIENT LA DONNÉE (2026-08-17/18, PLAN_IDENTITE_TI37 gates 0-1 puis
// PLAN_POSES_EQUIPEMENT_PUBLICATION) : le record de CRÉATION d'une entité `ti=37` porte, dans
// son bloc `object-multiplayer-properties`, le GlobalID du tag `eqip` de l'objet — les
// 21 valeurs du corpus se résolvent toutes dans le groupe `eqip` du jeu. Le MÊME record porte
// la position i0, c'est-à-dire le lieu exact de la pose. `t1` vient de la trajectoire décodée
// des paquets delta. `grammar.ScanFilmEquipmentPlacements` rend le tout.
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
	// Origin est l'ORIGINE de la pose. Identifiant STABLE du document (même règle que Family
	// et NeutralDeath.Kind).
	//
	// # LE VOCABULAIRE DU JEU, ET LE NÔTRE
	//
	// Le jeu n'écrit AUCUN événement « equipment drop » — il n'en existe pas (seul
	// `weapon_drop`, type 46, existe, et il est pour les armes). Ce qu'il écrit sur un objet
	// d'équipement, ce sont des COMPOSANTS D'ÉTAT de l'archétype 37
	// (`filmdec/testdata/ecs_table.tsv`) :
	//
	//	i20  equipment-deployed-component     l'objet est DÉPLOYÉ
	//	i21  equipment-activated-component    l'objet est ACTIF
	//	i18  item-at-rest-component           l'objet est POSÉ, immobile
	//	i10  object-parent-state-component    son PORTEUR
	//	i23  equipment-creator-component      qui l'a créé
	//
	// Plus un ÉVÉNEMENT, et un seul, pour l'apparition : `EquipmentSpawnedObject` (type 103),
	// « une PIÈCE a été engendrée ». Les ramassages sont `biped_pickup` (9) et
	// `biped_pickup_item_request` (57) ; les activations `biped_equipment_activation` (30) et
	// `activate_spartan_ability` (93).
	//
	// NOTRE VOCABULAIRE SE CALQUE SUR CELUI-LÀ. La définition tranchée par l'utilisateur le
	// 2026-09-15 : **« déployé »** = le mot du jeu, `deployed` ; **« lâché »** = l'objet QUITTE
	// SON PORTEUR SANS ÊTRE DÉPLOYÉ — à la mort ou à mi-vie par échange, les deux sont « lâché »,
	// et la cause va dans `coverage.placements.byCause`, jamais dans l'étiquette.
	//
	//	deployed  le film écrit que cette pose est un DÉPLOIEMENT. Aujourd'hui : l'événement 103
	//	          qui désigne la pièce engendrée — les panneaux du mur. Le jour où un autre
	//	          signal écrit de déploiement est lu, il entre ici par la même porte.
	//	dropped   l'objet quitte son porteur sans avoir été déployé. UN APPAREIL PORTÉ N'A
	//	          AUJOURD'HUI AUCUN DÉPLOIEMENT ÉCRIT, et c'est mesuré : le 103 en désigne 0 sur
	//	          91 et 0 sur 4 853 (rapport F.0). Il sort donc toujours `dropped`.
	//	unknown   le film ne dit RIEN de cette pose : ni 103, ni mort écrite du poseur, ni prise
	//	          écrite — ou aucun poseur mesuré. Rien ne se devine.
	//
	// CE QUE LA DÉCISION DU 2026-09-15 A SUPPRIMÉ. Un lâcher à mi-vie sortait `deployed` et
	// faisait dessiner un geste qui n'avait pas eu lieu ; et une pose dont le film ne disait rien
	// recevait quand même une étiquette, par une fenêtre temporelle de 200 ms depuis la fin de la
	// vie du poseur. Les deux ont cessé : la fenêtre ne classe plus AUCUNE pose d'équipement.
	//
	// LA CASCADE ET SON UNIQUE REPLI vivent dans `equipment_origin.go`. Ce qui reste
	// d'heuristique — une pièce engendrée au manifeste qu'aucun 103 ne désigne — est un REPLI
	// NOMMÉ au registre `fallback`, compté, et publié dans `byCause`.
	//
	// OPTIONNEL, ET CE N'EST PAS UNE FAIBLESSE DU CONTRAT — C'EST LA VÉRITÉ SUR LE PARC.
	// Les artefacts antérieurs au schéma 10 portent des poses SANS origine (ils sont encore sur
	// disque et en production jusqu'à la re-cuisson complète). Déclarer le champ REQUIS aurait
	// fait promettre au contrat une clé que ces artefacts n'ont pas. Le builder, lui, le
	// renseigne TOUJOURS — jamais la chaîne vide. **Absent = origine non mesurée : le client lit
	// `unknown`, JAMAIS `deployed`.**
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
	// Deployed / Dropped / Unknown : les poses par ORIGINE (schéma 10, cf.
	// EquipmentPlacement.Origin ; LUE et non plus mesurée depuis le schéma 59). Les trois sont
	// publiés, et l'invariant qui les rend vérifiables est testé :
	// `Deployed + Dropped + Unknown == Placements`, exactement.
	//
	// ILS DISENT LE VERDICT, JAMAIS SA PROVENANCE : deux documents aux mêmes trois comptes
	// peuvent venir l'un d'une lecture du film et l'autre d'un repli. C'est [ByCause] qui les
	// sépare, et c'est la raison d'être du schéma 59.
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
	// SpawnEvents est le nombre d'événements 103 `EquipmentSpawnedObject` LUS dans le film —
	// « une PIÈCE a été engendrée ». C'est le DÉNOMINATEUR de la désignation : sans lui, un
	// `byCause.spawn_event` à zéro ne se distingue pas d'un film que le lecteur ne sait pas lire
	// (mesuré : 0 événement sur `a521164d` HI_1_4_1 et 2 sur `50247b26` v31, contre 30 à 86 sur
	// les builds récents).
	SpawnEvents int `json:"spawnEvents"`
	// SpawnLists est le nombre de LISTES D'ÉVÉNEMENTS NON VIDES traversées par ce balayage.
	//
	// C'EST LUI QUI SÉPARE DEUX ZÉROS QUI NE SE RESSEMBLENT PAS. `spawnEvents: 0` sur un film
	// qui porte 7 850 listes dit « aucune pièce n'a été engendrée » ; le même zéro sur un film
	// qui n'en porte aucune dirait « le lecteur n'a rien pu lire ». Les deux se traitent
	// autrement, et l'artefact ne pouvait pas les distinguer sans ce dénominateur — c'est la
	// question ouverte D2 (1.9.1) du plan, posée en chiffres dans le document.
	SpawnLists int `json:"spawnLists"`
	// ByCause est la PROVENANCE de l'origine de chaque pose (schéma 59, lot 1.9.1) : ce que le
	// document dit de LUI-MÊME. Clés fermées, cf. les constantes `CausePose*` :
	//
	//	spawn_event     un événement 103 désigne la pièce engendrée      LECTURE  -> deployed
	//	death_written   une mort écrite du poseur couvre l'instant       LECTURE  -> dropped
	//	taken_written   une prise écrite du poseur couvre l'instant      LECTURE  -> dropped
	//	both            les deux à la fois : contradiction COMPTÉE       LECTURE  -> dropped
	//	manifest_piece  pièce engendrée qu'aucun 103 ne désigne          REPLI    -> deployed
	//	none            un poseur est mesuré, le film ne dit RIEN                 -> unknown
	//	no_owner        aucun poseur mesuré : on ne sait pas de qui parler        -> unknown
	//
	// LES DEUX DERNIÈRES SONT DEUX SILENCES DIFFÉRENTS, et les confondre coûterait le
	// diagnostic : `none` dit que le poseur est connu et que le film se tait sur lui,
	// `no_owner` qu'aucun bipède contemporain n'était assez proche.
	//
	// SA SOMME VAUT `Placements`, EXACTEMENT, et l'invariant est testé. C'est ce qui rend la
	// table lisible : « 3 255 poses décidées par une mort écrite » ne veut rien dire sans le
	// total qu'elle partage avec les autres.
	ByCause map[string]int `json:"byCause,omitempty"`
}

// equipmentFamilyOther est la famille par défaut : un objet dont la nature n'est pas établie.
const equipmentFamilyOther = "other"

// decodeFilmPlacements décode les poses du film et JOURNALISE ce qu'il en est.
//
// TROIS SORTIES, TROIS PHRASES — et la distinction est le point. Un balayage impossible (film
// illisible) n'est pas un film sans équipement, et un film dont le découpage du bloc de
// réplication n'a pas été tranché n'est ni l'un ni l'autre : il aurait des poses, mais les lire
// à une largeur devinée rendrait du bruit. Les trois se lisent au journal, et la troisième se
// relit ensuite dans l'artefact (`coverage.placements.calibrated`).
//
// HORS LIGNE — appelée par BuildFromFilm.
func decodeFilmPlacements(
	fc *grammar.FilmContext, matchID string, worldRange *profile.Vec3Range,
) ([]types.EquipmentPlacement, grammar.EquipmentPlacementStats) {
	pl, st, err := grammar.ScanEquipmentPlacements(fc, worldRange)
	if st.FormatSansProfil {
		// SITE 1 DU REPLI `repli_largeurs_mpp_calibrees_sur_le_film` : le compteur, pas le
		// journal — l avertissement est emis UNE FOIS PAR FILM par `avertirFormatSansProfil`.
		publierFormatSansProfil(st.FormatVersion)
	}
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

// decodeFilmSpawnEvents lit les evenements 103 « une PIECE a ete engendree » et JOURNALISE ce
// que le balayage a vu.
//
// UN FILM SANS AUCUN EVENEMENT N'EST PAS UNE ERREUR, ET CE N'EST PAS NON PLUS UN SILENCE : les
// builds les plus anciens du corpus n'en rendent aucun (`a521164d`, HI_1_4_1 : 0 sur 4 956
// listes non vides) la ou les recents en rendent des dizaines. Le journal publie donc les DEUX
// denominateurs — listes traversees et evenements lus —, et la couverture de l'artefact les
// publie TOUS DEUX (`coverage.placements.spawnLists` et `.spawnEvents`).
//
// HORS LIGNE — appelee par le balayage, sous le meme verrou que le reste de la cuisson.
func decodeFilmSpawnEvents(
	fc *grammar.FilmContext, matchID string,
) ([]types.EquipmentSpawnEvent, types.EquipmentSpawnStats) {
	ev, st, err := grammar.ScanEquipmentSpawnEvents(fc)
	if err != nil {
		slog.Warn("evenements de piece engendree illisibles — l'origine des poses retombe sur ses replis",
			"err", err, "match_id", matchID)
		return nil, st
	}
	slog.Info("poses d'equipement : evenements 103 (piece engendree)",
		"match_id", matchID, "chunks", st.Chunks, "paquetsDelta", st.Packets,
		"listesNonVides", st.Lists, "evenements", st.Events,
		"refSource", st.WithSource, "refEngendree", st.WithSpawned, "ref2", st.Ref2)
	return ev, st
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
		"finVue", c.EndSeen, "finOuverte", c.EndOpen,
		// LA PROVENANCE, AU JOURNAL COMME A L'ARTEFACT (lot 1.9.1) : sans elle, un operateur lit
		// « 51 deployees » sans savoir si le film l'a dit ou si un repli l'a decide.
		"evenements103", c.SpawnEvents, "listesEvenements", c.SpawnLists, "parCause", c.ByCause)
}

// equipmentInputs porte TOUT ce que l'assemblage des poses lit : le balayage, le nuage de
// bipèdes, le recensement d'images-clés et les TROIS SIGNAUX ÉCRITS de l'origine (lot 1.9.1).
//
// UN STRUCT PARCE QUE LA LIMITE DU DÉPÔT EST DE CINQ PARAMÈTRES, et que la cascade d'origine en
// demande trois de plus (événements 103, vies nommées, changements d'équipement). Il ne porte
// AUCUN réglage : l'horloge et le compteur de replis restent dans `replayClock`.
type equipmentInputs struct {
	// Raw / Stats : le balayage des créations `ti=37` et sa calibration.
	Raw   []types.EquipmentPlacement
	Stats grammar.EquipmentPlacementStats
	// Positions est le nuage NON décimé, TRIÉ par instant : la recherche du poseur est une
	// fenêtre glissante, pas un balayage complet par pose.
	Positions []grammar.BipedPosition
	// Census est le recensement `ti=37` des images-clés — la FIN OBSERVÉE (schéma 28).
	Census grammar.WorldObjectKeyframes
	// Spawns sont les événements 103 `EquipmentSpawnedObject` : « une PIÈCE a été engendrée » ;
	// SpawnStats porte les DÉNOMINATEURS de leur balayage, sans lesquels un zéro d'événement ne
	// se distingue pas d'un film que le lecteur n'a pas su lire.
	Spawns     []types.EquipmentSpawnEvent
	SpawnStats types.EquipmentSpawnStats
	// Lives sont les vies NOMMÉES du registre d'identité : leur `cause` porte la mort ÉCRITE.
	Lives []lifeSpan
	// Changes sont les ramassages et consommations d'équipement : leurs `taken` portent la prise
	// ÉCRITE du poseur.
	Changes []types.EquipmentChange
}

// buildEquipmentPlacements assemble les poses : famille par le manifeste, poseur par
// proximité mesurée, cap par la visée du poseur, ORIGINE par ce que le film écrit
// (cf. `equipment_origin.go`) — et, depuis le schéma 28, la FIN OBSERVÉE par le recensement
// des images-clés.
func buildEquipmentPlacements(
	in equipmentInputs, clock replayClock,
) ([]EquipmentPlacement, *EquipmentPlacementCoverage) {
	st := in.Stats
	cov := &EquipmentPlacementCoverage{
		Scanned:        st.Scanned,
		Calibrated:     st.Calibration.Widths.Valid(),
		Lives:          st.Lives,
		Anchors:        st.Anchors,
		Confirmed:      st.Confirmed,
		ByFamily:       map[string]int{},
		ByFamilyOrigin: map[string]int{},
		ByCause:        map[string]int{},
	}
	if cov.Calibrated {
		cov.Widths = st.Calibration.Widths.String()
	}
	if len(in.Raw) == 0 || clock.step == 0 {
		return nil, cov
	}
	src := nouvelleSourceOrigine(in.Spawns, in.Lives, in.Changes)
	cov.SpawnEvents, cov.SpawnLists = src.evenements, in.SpawnStats.Lists
	ends := placementEnds(in.Raw, in.Census, clock)
	out := make([]EquipmentPlacement, 0, len(in.Raw))
	causes := make([]string, 0, len(in.Raw))
	for i, p := range in.Raw {
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
			Until:  ends[i].until, UntilMax: ends[i].untilMax, End: ends[i].end,
		}
		if pl.Family == "" {
			pl.Family = equipmentFamilyOther
		}
		o := poseOwner{src: src}
		if slot, h, ok := equipmentOwner(in.Positions, p); ok {
			pl.Owner, pl.H = int(slot), h
			o.slot, o.avecPoseur = slot, true
		}
		var cause string
		pl.Origin, cause = origineDeLaPose(p, pl.ID, o, clock.fb)
		out, causes = append(out, pl), append(causes, cause)
	}
	tallyEquipmentPlacements(out, causes, cov)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].T0 != out[j].T0 {
			return out[i].T0 < out[j].T0
		}
		return out[i].ID < out[j].ID
	})
	return out, cov
}

// replayClock porte l'axe de temps et la table de familles du document (règle des
// 5 paramètres — ces quatre-là voyagent toujours ensemble).
type replayClock struct {
	origin, step uint64
	frames       int
	families     map[uint32]string
	// fb compte les REPLIS de cette cuisson (D14, cf. le paquet `fallback`). Il voyage ici parce
	// que `replayClock` est deja CE QUE L'ASSEMBLAGE PARTAGE — la grille, les frames et la table
	// des familles —, et que les calques qui le recoivent sont exactement ceux qui se replient.
	// Nil ne compte rien.
	fb *fallback.Compteur
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

// tallyEquipmentPlacements ventile les poses publiees dans la couverture.
//
// `causes` est PARALLELE a `out`, et c'est pourquoi l'appelant compte AVANT de trier : le tri
// d'affichage reordonne les poses, jamais les causes.
func tallyEquipmentPlacements(
	out []EquipmentPlacement, causes []string, cov *EquipmentPlacementCoverage,
) {
	cov.Placements = len(out)
	for i, p := range out {
		cov.ByCause[causes[i]]++
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
