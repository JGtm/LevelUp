package replay

import (
	"log/slog"
)

// coverage.go — CE QUE CHAQUE CALQUE A RATTACHÉ, SUR CE QUI EXISTAIT.
//
// LE DÉFAUT QUE CE FICHIER SUPPRIME. `uniqueSlotFor` rendait `(slot, ok)` et l'appelant
// faisait `continue` quand `ok` était faux. Les événements écartés n'apparaissaient nulle
// part : le décodeur ne SAVAIT pas qu'il avait perdu quelque chose, donc personne ne
// pouvait le savoir. Un trou de 34 secondes dans les tirs a ainsi été découvert par
// l'utilisateur en regardant l'écran, alors qu'il était mesurable dès le premier jour.
//
// C'est l'anti-patron « erreur avalée » que le dépôt interdit déjà ailleurs — `_ = f()`,
// `continue` sur erreur sans log ni compteur.
//
// LA RÈGLE POSÉE ICI : tout événement écarté est COMPTÉ et CATÉGORISÉ, et le compte est
// publié AVEC le résultat, pas seulement dans les journaux. L'invariant qui en découle est
// vérifiable et testé : rattachés + rejetés = disponibles, exactement.
//
// POURQUOI LA CATÉGORIE, ET PAS UN SIMPLE TOTAL. Les trois causes appellent des réponses
// opposées : « slot introuvable » dit que le pont est incomplet (chantier de rattachement),
// « slot ambigu » que deux vies se recouvrent (chantier de découpage des vies), « hors
// fenêtre » que la position manque à cet instant (chantier de décodage des positions). Un
// total unique les mélangerait et n'orienterait rien.

// rejectSampleThreshold : au-delà de cette proportion d'une catégorie de rejet, le calque
// émet un avertissement. Un log par événement noierait le journal (519 tirs sur un film) ;
// un seuil global le rend lisible tout en gardant l'alerte.
const rejectSampleThreshold = 0.10

// LayerCoverage est la couverture d'un calque : combien il a rattaché, sur combien
// existaient, et pourquoi il a écarté le reste.
type LayerCoverage struct {
	// Available est le nombre d'événements DISPONIBLES dans le film pour ce calque —
	// le dénominateur sans lequel un compte de rattachés ne se juge pas.
	Available int `json:"available"`
	// Attached est le nombre d'événements effectivement rattachés à un slot.
	Attached int `json:"attached"`
	// NoSlot : aucun slot de ce joueur n'est connu à cet instant (le pont ne le couvre pas).
	NoSlot int `json:"noSlot"`
	// Ambiguous : plusieurs slots de ce joueur couvrent l'instant (vies qui se recouvrent).
	Ambiguous int `json:"ambiguous"`
	// OutOfWindow : le slot est connu, mais aucune position n'est répliquée assez près de
	// l'instant pour poser l'événement sur la carte.
	OutOfWindow int `json:"outOfWindow"`
	// Unpublished : l'événement était rattaché, mais son slot n'a pas de trajectoire publiée
	// (track trop courte). Compté à part : ce n'est pas un échec de rattachement.
	Unpublished int `json:"unpublished"`
}

// Balanced vérifie l'invariant : tout ce qui existait est soit rattaché, soit rejeté sous
// une cause nommée. Une somme fausse signale une fuite — un chemin de rejet non compté.
func (c LayerCoverage) Balanced() bool {
	return c.Attached+c.NoSlot+c.Ambiguous+c.OutOfWindow+c.Unpublished == c.Available
}

// rejectReason nomme la cause d'un rejet, pour le comptage.
type rejectReason int

const (
	reasonAttached rejectReason = iota
	reasonNoSlot
	reasonAmbiguous
	reasonOutOfWindow
)

// count incrémente le compteur de la cause.
func (c *LayerCoverage) count(r rejectReason) {
	switch r {
	case reasonAttached:
		c.Attached++
	case reasonNoSlot:
		c.NoSlot++
	case reasonAmbiguous:
		c.Ambiguous++
	case reasonOutOfWindow:
		c.OutOfWindow++
	}
}

// warnIfLossy émet un avertissement ÉCHANTILLONNÉ — un par calque, pas un par événement —
// quand une catégorie de rejet dépasse le seuil. Le nom du calque est passé en clair pour
// que le journal désigne le chantier concerné.
func (c LayerCoverage) warnIfLossy(layer string) {
	if c.Available == 0 {
		return
	}
	for _, cat := range []struct {
		name string
		n    int
	}{{"slotIntrouvable", c.NoSlot}, {"slotAmbigu", c.Ambiguous}, {"horsFenetre", c.OutOfWindow}} {
		if float64(cat.n)/float64(c.Available) < rejectSampleThreshold {
			continue
		}
		slog.Warn("rejeu : rejets au-dessus du seuil",
			"calque", layer, "cause", cat.name, "rejetes", cat.n, "disponibles", c.Available,
			"rattaches", c.Attached)
	}
}

// Coverage porte la couverture de chaque calque du document, et le VERDICT qui en découle.
//
// ELLE EST PUBLIÉE DANS LE DOCUMENT, pas seulement journalisée : c'est la différence entre
// un décodeur qui SAIT ce qu'il perd et un écran qui le MONTRE. Le POC l'affiche à côté de
// ses autres chiffres.
type Coverage struct {
	Shots    LayerCoverage `json:"shots"`
	Grenades LayerCoverage `json:"grenades"`
	// Objectives est la couverture du calque des actions d'objectif (cf. objectives.go).
	// Son dénominateur est le nombre d'événements identifiés fournis au build : publier
	// 40 actions sans dire que 55 existaient laisserait croire à l'exhaustivité.
	Objectives LayerCoverage `json:"objectives"`
	// Equipment est la couverture des épisodes d'état actif d'équipement (schéma 7, cf.
	// equipment_episodes.go) : combien de vies publiées portent des épisodes, par famille.
	// Sa forme diffère des LayerCoverage : un épisode n'est pas un événement à rattacher
	// (le slot est DANS la lecture), le dénominateur est le nombre de vies publiées.
	// Absente des artefacts antérieurs au schéma 7.
	Equipment *EquipmentCoverage `json:"equipment,omitempty"`
	// Grapple est la couverture des tractions de grappin (schéma 8, cf. grapple_lines.go) :
	// lectures tir/accroche, tractions publiées, ratés et corps non décodables. Même
	// logique qu'Equipment : le slot est DANS la lecture, pas à rattacher. Absente des
	// artefacts antérieurs au schéma 8.
	Grapple *GrappleCoverage `json:"grapple,omitempty"`
	// Placements est la couverture des POSES d'équipement (schéma 9, cf.
	// equipment_placements.go) : le découpage de bloc calibré sur CE film, les vies, les
	// ancres, les poses confirmées par l'oracle, et la part nommée / avec poseur / avec cap.
	// Absente des artefacts antérieurs au schéma 9.
	//
	// ELLE EST PUBLIÉE MÊME QUAND IL N'Y A AUCUNE POSE, et c'est le point : un film dont la
	// calibration échoue et un film sans équipement posé rendent tous deux zéro pose — seule
	// la couverture les distingue (`calibrated: false` contre `calibrated: true`).
	Placements *EquipmentPlacementCoverage `json:"placements,omitempty"`
	// GroundWeapons est la couverture des SOCLES D'ARME (schéma 11, cf. ground_weapon_pads.go) :
	// créations lues, retenues par l'identité et écartées, apparitions classées, grappes,
	// socles publiés, et l'issue de chaque occupation (datée / sans passage / jamais vidée).
	// Absente des artefacts antérieurs au schéma 11.
	//
	// ELLE EST PUBLIÉE MÊME QUAND IL N'Y A AUCUN SOCLE, pour la même raison que `placements` :
	// un film sans socle, un film dont toutes les armes sont des lâchers et un film qu'on n'a
	// pas su balayer rendent tous trois zéro socle — seuls ces compteurs les distinguent.
	GroundWeapons *GroundWeaponCoverage `json:"groundWeapons,omitempty"`
	// Score est la couverture de la courbe de score (schéma 12, cf. document_score.go) :
	// d'où vient l'identité des camps, combien de manches ont été lues, si le mode porte le
	// score de mode, si la lecture a été tronquée, et ce que la courbe MESURE (`displayed`).
	//
	// ELLE EST PUBLIÉE MÊME QUAND AUCUNE COURBE NE L'EST, pour la même raison que `placements`
	// et `groundWeapons` : un film sans enregistrements d'entité et un film dont l'identité des
	// camps n'a pas été résolue rendent tous deux une courbe pauvre, et seuls ces compteurs les
	// distinguent. Son ABSENCE dit autre chose encore — l'appelant n'a rien fourni à lire.
	Score *ScoreCoverage `json:"score,omitempty"`
	// FlagCarries est la couverture du calque du DRAPEAU VIVANT (schéma 14, cf.
	// document_objectives_live.go) : le verdict de mode et les trois signaux qui le fondent, les
	// prises de l'oracle, les portages publiés partagés en fermés / ouverts, les rejets par
	// cause, le contrôle du marqueur et les incohérences.
	//
	// ELLE EST PUBLIÉE MÊME QUAND AUCUN DRAPEAU NE L'EST, pour la même raison que `placements`,
	// `groundWeapons` et `score` : un film d'un autre mode et un film CTF sans aucun portage
	// publié rendent tous deux un calque vide. Son ABSENCE dit encore autre chose — l'appelant
	// n'a rien fourni à lire.
	FlagCarries *FlagCarriesCoverage `json:"flagCarries,omitempty"`
	// Zones est la couverture de L'ÉTAT DES ZONES (schéma 15, cf. document_zones.go) : la
	// MÉTHODE d'appariement employée et les rôles du catalogue qui composent `mapObjectives.zones`
	// (sans quoi `zoneRef` ne serait pas vérifiable), les slots lus, ceux qu'aucune capture n'a
	// rattachés, et le contrôle du propriétaire contre l'équipe du capteur.
	//
	// ELLE EST PUBLIÉE MÊME QUAND AUCUNE ZONE NE L'EST, pour la même raison que les précédentes :
	// un film d'un autre mode, un film à zones dont l'appariement échoue et une carte hors du
	// catalogue rendent tous trois un calque vide. Son ABSENCE dit encore autre chose — l'appelant
	// n'a rien fourni à lire.
	Zones *ZonesCoverage `json:"zones,omitempty"`
	// OriginResolved dit si l'ORIGINE de la frame 0 a été établie (cf. origin.go).
	//
	// ELLE VAUT POUR TOUS LES CALQUES DATÉS DEPUIS L'HORLOGE DU FILM, pas seulement pour un :
	// les actions d'objectif et la courbe de score se posent sur la grille de frames en
	// RETRANCHANT cette origine. Quand elle n'est pas établie (chunk 1 illisible, premier
	// paquet de position antérieur au zéro du film, ou témoin du fil des morts en désaccord),
	// la soustraction se fait avec zéro et ces calques restent décalés de 3,6 s à 50,8 s selon
	// le match — un décalage que RIEN, dans l'artefact, ne signalait avant ce champ.
	//
	// FAUX N'EST PAS « PAS DE DONNÉE » : les calques sont publiés, mais leur axe de temps n'est
	// pas fiable. Le rendu les masque plutôt que de les poser au mauvais instant.
	OriginResolved bool `json:"originResolved"`
	// Verdict dit, calque par calque, si le résultat est publiable. Repris du chantier
	// voisin, qui sait annoncer « 371 couples sur 371, verdict nominal ».
	Verdict map[string]string `json:"verdict,omitempty"`
	// Bridge décrit sur quoi repose le pont slot -> joueur. Un calque peut être complet et
	// néanmoins reposer sur une résolution fragile : les deux se jugent séparément.
	Bridge BridgeHealth `json:"bridge"`
}

// BridgeHealth résume la santé du pont slot -> joueur.
//
// POURQUOI LA PUBLIER PLUTÔT QUE LA GARDER POUR LE JOURNAL : la couverture d'un calque peut
// être excellente alors que le pont qui la porte est fragile. On publie donc, à côté du taux,
// de quoi juger la SOURCE : combien de vies le fil des morts a nommées, et combien de chunks
// ont confirmé la table d'index.
//
// UN CHAMP PAR LIGNE, ET C'EST DÉLIBÉRÉ : en Go, plusieurs champs déclarés sur une même
// ligne PARTAGENT leur tag, ce qui produirait ici plusieurs clés JSON identiques. Le
// compilateur ne dit rien ; seule la sortie sérialisée le révèle. Un test le verrouille
// (TestBridgeHealthJSONKeysAreDistinct).
type BridgeHealth struct {
	// Slots est le nombre d'entrées du pont.
	Slots int `json:"slots"`
	// FromReading : le nombre d'entrées issues de la LECTURE SEULE.
	//
	// CE N'EST PLUS LA SEULE SOURCE, et l'écrire ici l'aurait laissé croire : les FERMETURES
	// (closures.go, 2026-08-08) alimentent le pont par déduction. La règle que `verdictOfBridge`
	// applique réellement est `FromReading + ClosedByShot + ClosedByRespawn == Slots` — un écart
	// à cette somme, et non à `Slots` seul, signale une source non comptée.
	FromReading int `json:"fromReading"`
	// LivesNamed / LivesTotal : ce que le fil des morts a nommé.
	LivesNamed int `json:"livesNamed"`
	LivesTotal int `json:"livesTotal"`
	// IndexReadings : combien de chunks de réplication ont livré la MÊME table identité ->
	// index. Remplace la « marge » de l'ancienne résolution par choix.
	IndexReadings int `json:"indexReadings"`
	// IndexDisagreements : identités lues différemment d'un chunk à l'autre. Non nul, la
	// lecture est fausse.
	IndexDisagreements int `json:"indexDisagreements"`
	// SlotCollisions : slots dont les vies nommées désignent des joueurs différents. Non
	// nul, la table slot -> joueur n'est pas représentable et le verdict le dit.
	SlotCollisions int `json:"slotCollisions"`
	// ClosedByShot : entrées ajoutées par la fermeture A (le corps disponible).
	ClosedByShot int `json:"closedByShot"`
	// ClosedByRespawn : entrées ajoutées par la fermeture B (la réapparition).
	ClosedByRespawn int `json:"closedByRespawn"`
	// ClosedContested : déductions ABANDONNÉES faute d'unicité — deux corps possibles pour un
	// même tir, deux joueurs pour un même corps, deux corps pour un même joueur (un joueur n'a
	// qu'un DERNIER corps), ou deux corps pour une même mort.
	ClosedContested int `json:"closedContested"`
	// ClosedRefused : déductions REJETÉES alors qu'un seul candidat subsistait, pour l'une des
	// causes qui rendent l'attribution IMPOSSIBLE OU NON CORROBORÉE : le corps déduit ne
	// PROLONGE pas le tireur (fermeture A — aucun corps connu pour l'ancrer, ou un corps connu
	// qui ne s'achève pas avant lui ; le recouvrement en est le cas particulier), le
	// recouvrement (fermeture B — un joueur n'a qu'un corps), ou une identité que la table
	// d'index ne porte pas (on saurait quelle victime, pas quel index de film).
	//
	// LES DEUX COMPTEURS DE REFUS SONT PUBLIÉS AU MÊME TITRE QUE LES SUCCÈS, et ce n'est pas
	// de la décoration : un contrôle qui ne rejette jamais rien ne prouve rien. Mesuré sur
	// sept films, 33 attributions pour 17 refus.
	ClosedRefused int `json:"closedRefused"`
}

// VerdictNominal est le verdict d'un calque publiable sans réserve. Nommé plutôt que répété :
// c'est la valeur que les consommateurs comparent, et une faute de frappe dans l'une des six
// occurrences produirait un calque silencieusement « non nominal ».
const VerdictNominal = "nominal"

// Seuils de la porte de publication. Ils sont ÉCRITS ICI, pas dispersés dans les appelants,
// et chacun porte la mesure qui l'a fixé.
const (
	// publishMinRatio : en deçà, le calque publie moins des deux tiers de ce qui existe et
	// se lira comme exhaustif alors qu'il ne l'est pas. Le critère de succès du plan est à
	// 85 % ; 0,66 est le seuil du REFUS, pas celui de la satisfaction.
	publishMinRatio = 0.66
	// bridgeMinReadings : nombre minimal de chunks concordants pour tenir la table d'index
	// pour lue. Mesuré sur 000d5950 : 26 chunks de réplication, tous d'accord. En deçà de 2,
	// on n'a pas de confirmation du tout.
	bridgeMinReadings = 2
)

// verdictOf rend le verdict d'un calque : « nominal », « partiel » ou « non publiable ».
//
// LA PORTE REFUSE PLUTÔT QUE D'AVERTIR. Un calque « non publiable » doit être retiré de
// l'écran, pas affiché avec une note en bas de page : un tir posé sur le mauvais joueur est
// pire qu'un tir absent, et c'est la leçon que ce chantier a déjà payée.
func verdictOf(c LayerCoverage) string {
	switch {
	case c.Available == 0:
		return "aucune donnée"
	case !c.Balanced():
		return "non publiable : fuite dans le comptage"
	case float64(c.Attached)/float64(c.Available) < publishMinRatio:
		return "partiel : moins des deux tiers rattachés"
	default:
		return VerdictNominal
	}
}

// verdictOfBridge rend le verdict du pont lui-même.
//
// LA RÈGLE DE PROVENANCE A CHANGÉ LE 2026-08-08, ET LE COMMENTAIRE AVEC ELLE. Elle exigeait
// `FromReading == Slots` — « une source autre que la lecture a alimenté le pont ». Cette règle
// est née du retrait du repli VOTÉ (2026-07-28) et elle visait juste : rien ne devait entrer au
// pont par un choix. Elle deviendrait FAUSSE telle quelle depuis l'ajout des FERMETURES
// (closures.go), qui ne sont pas des lectures mais des déductions par élimination.
//
// Ce qui est conservé, et c'est l'esprit de la règle : **toute entrée du pont doit être
// justifiée par une source nommée**. La somme lecture + fermetures doit donc rendre EXACTEMENT
// le nombre d'entrées ; un écart signalerait qu'une troisième source, elle non comptée, s'est
// glissée dans le pont — précisément ce que la règle d'origine interdisait.
func verdictOfBridge(b BridgeHealth) string {
	switch {
	case b.Slots == 0:
		return "non publiable : aucun pont"
	case b.SlotCollisions > 0:
		return "non publiable : un slot change de porteur"
	case b.FromReading+b.ClosedByShot+b.ClosedByRespawn != b.Slots:
		return "non publiable : une source non comptée a alimenté le pont"
	case b.IndexDisagreements > 0:
		return "non publiable : une identité est lue de deux façons"
	case b.IndexReadings < bridgeMinReadings:
		return "partiel : la table d'index n'est confirmée par aucun second chunk"
	default:
		return VerdictNominal
	}
}

// buildCoverage assemble la couverture publiée et ses verdicts. `originResolved` dit si
// l'origine de la frame 0 a été établie — elle conditionne la justesse de l'axe de temps de
// TOUS les calques datés depuis l'horloge du film (cf. Coverage.OriginResolved).
func buildCoverage(shots, grenades, objectives LayerCoverage, own OwnerReport,
	originResolved bool, score *ScoreCoverage) *Coverage {
	b := BridgeHealth{
		Slots: len(own.Owner), FromReading: own.FromDeaths,
		LivesNamed: own.DeathsNamed, LivesTotal: own.LivesTotal,
		IndexReadings:      own.IndexReadings,
		IndexDisagreements: own.IndexDisagreements,
		SlotCollisions:     own.SlotCollisions,
		ClosedByShot:       own.Closures.byShot,
		ClosedByRespawn:    own.Closures.byRespawn,
		ClosedContested:    own.Closures.contested,
		ClosedRefused:      own.Closures.refused,
	}
	return &Coverage{
		Shots: shots, Grenades: grenades, Objectives: objectives, Bridge: b,
		OriginResolved: originResolved, Score: score,
		Verdict: map[string]string{
			"shots":      verdictOf(shots),
			"grenades":   verdictOf(grenades),
			"objectives": verdictOf(objectives),
			"bridge":     verdictOfBridge(b),
		},
	}
}

// slotFor rend le slot du joueur pi à l'instant tUS, et la cause du rejet le cas échéant.
//
// C'EST LA MÊME PORTE QUE `uniqueSlotFor`, mais elle DIT pourquoi elle se ferme. L'ancienne
// version rendait un booléen : « slot introuvable » et « slot ambigu » y étaient
// indiscernables, alors qu'ils désignent deux chantiers différents.
func slotFor(tracks map[uint32]slotTrack, owner map[uint32]int, pi int, tUS uint64) (uint32, rejectReason) {
	var found uint32
	n := 0
	for slot, idx := range owner {
		if idx != pi {
			continue
		}
		if _, d := tracks[slot].at(tUS); d <= shotPosToleranceUS {
			found = slot
			n++
		}
	}
	switch {
	case n == 1:
		return found, reasonAttached
	case n > 1:
		return 0, reasonAmbiguous
	default:
		return 0, reasonNoSlot
	}
}

// countUnpublished mesure, après filtrage, combien d'événements rattachés ont été retirés
// faute de trajectoire publiée. Rendu en tant que catégorie propre : ce n'est pas un échec
// du rattachement, et le confondre avec « slot introuvable » orienterait vers le mauvais
// chantier.
func countUnpublished(before, after int) int {
	if d := before - after; d > 0 {
		return d
	}
	return 0
}
