package replay

// equipment_episode_kills.go — LA JOINTURE PURE entre les épisodes d'état actif
// (equipment_episodes.go : camouflage, surbouclier) et les frags/assistances du film.
//
// PLAN_RETOURS_UTILISATEUR_2026-08-29 §LOT F, sous-lot F.1. Décision utilisateur 8a/8b,
// DEC-7 (révisée après contre-lecture de la session équipement) : GO à petite population —
// camo 35,2 % (25/71), surbouclier 55,6 % (10/18), global 39,3 % (35/89) en lecture STRICTE
// (`LineByLinePublishable`, la population qui affiche réellement des chiffres). Re-mesure
// obligatoire après la cuisson de masse (n=149 → ~10×, cf. F.0).
//
// FRONTIÈRE, ET ELLE EST LA MÊME QUE PARTOUT AILLEURS DANS CE PAQUET (Options.Deaths,
// Options.Vip, Options.Skull, ...) : ce fichier ne lit NI film NI base. `EquipmentKillRef` est une
// identité déjà résolue (xuid) et un instant déjà daté sur l'horloge « depuis le début du
// film » (`killsource.Kill.TimeMS`) — la résolution gamertag -> xuid vit dans
// `internal/replaybuild` (le seul paquet du dépôt qui ouvre `killsource`, cf.
// replaybuild/kills.go), jamais ici. `analysis/replay` reste title-agnostic et pur.
//
// L'HORLOGE : `killsource.Kill.TimeMS` est sur l'horloge « début du film », PAS celle du fil
// des éliminations (`event_time_ms + t0_ms`) — mesuré à 99,9 % d'exactitude sur 1 822 kills
// (F.0). La conversion vers l'axe des frames du rejeu est donc une SOUSTRACTION SIMPLE,
// `replayMs = TimeMS - OriginMs` (cf. document.go, `ReplayDocument.OriginMs`), suivie d'une
// division entière par le pas de la grille — exactement la même famille de calcul que
// `frameOf` (juste au-dessus, en microsecondes) mais en millisecondes : les deux horloges
// s'accordent au même zéro par construction (cf. origin.go).

// EquipmentKillRef est un frag résolu en identité, sur l'horloge « début du film ». MÊME FRONTIÈRE
// QUE LES AUTRES ENTRÉES DE DONNÉES D'OPTIONS : l'appelant résout, cette jointure consomme.
type EquipmentKillRef struct {
	// XUID est l'identité créditée du frag par le kill-feed (le TUEUR).
	XUID uint64
	// TimeMS est l'instant du frag SUR L'HORLOGE DU FILM — verbatim killsource.Kill.TimeMS,
	// PAS encore recalé sur l'origine de la frame 0 (cette jointure fait la soustraction
	// elle-même, une fois l'origine du document connue).
	TimeMS int
	// AssistXUID / AssistKnown : l'assistant NOMMÉ, si la résolution a abouti.
	//
	// AssistKnown=false NE VEUT PAS DIRE « pas d'assistant » (même règle que
	// killsource.Assist.Known) : ça veut dire qu'aucune identité résolue n'est disponible
	// pour ce frag — silence, jamais un zéro imposé à l'assistant.
	AssistXUID  uint64
	AssistKnown bool
}

// KillsInput porte les EquipmentKillRef d'un match ET si la mesure a pu être tentée. `Read` distingue
// LA MESURE de son ABSENCE : faux = killsource.Decode a échoué, sa source était illisible, ou
// sa porte de publication ligne-par-ligne (`Result.LineByLinePublishable`) était fermée pour
// ce match — DISTINCT d'une liste VIDE à `Read=true`, qui dit « lu, zéro frag sous effet actif
// mesuré ». Sans cette distinction, `EquipmentEpisode.K/A` à zéro et une mesure non tentée
// s'afficheraient pareil (cf. EquipmentCoverage.KillsRead).
type KillsInput struct {
	Read  bool
	Kills []EquipmentKillRef
}

// attachAllEquipmentKills pose les compteurs sur les épisodes DÉJÀ CONSTRUITS et rend si la
// mesure est fiable — appelée depuis BuildFromPositions juste après buildEquipmentEpisodes,
// UNE SEULE LIGNE au site d'appel pour ne pas alourdir davantage un fichier déjà long.
//
// L'ORIGINE CONDITIONNE AUTANT QUE LA LECTURE : sans `originMs` établi, convertir
// `EquipmentKillRef.TimeMS` en frame ne veut rien dire (cf. origin.go — le client retomberait sur un
// décalage de 3,6 s à 50,8 s selon le match) ; aucun frag n'est donc joint, et c'est compté
// comme une mesure NON tentée, pas comme un match sans aucun frag sous effet actif.
func attachAllEquipmentKills(
	episodes []EquipmentEpisode, kills KillsInput, slotXUID map[uint32]uint64,
	originMs *int64, frameIntervalMS int,
) bool {
	read := kills.Read && originMs != nil
	if read {
		attachEpisodeKills(episodes, kills.Kills, slotXUID, *originMs, frameIntervalMS)
	}
	return read
}

// attachEpisodeKills pose, sur chaque épisode, les frags et assistances du PORTEUR pendant SA
// fenêtre [T0, T1]. PURE — aucune lecture de film ni de base ici, testable sur données
// synthétiques (equipment_episode_kills_test.go).
//
// slotXUID EST LE MÊME PONT QUE CELUI DÉJÀ PUBLIÉ DANS L'ARTEFACT (Track.XUID, cf.
// document.go) : à un instant donné, au plus UNE vie d'un xuid est active, et les épisodes
// d'un slot sont déjà bornés à SA fenêtre de vie (equipment_episodes.go, `episodeAccum`) —
// chercher parmi TOUS les slots d'un xuid ne peut donc jamais faire matcher deux vies
// distinctes au même instant, sans qu'il soit nécessaire de rejouer ici les fenêtres de vie.
//
// UN FRAG APRÈS LA FIN DE L'ÉPISODE NE COMPTE PAS, QUE LA FIN SOIT MESURÉE OU LA MORT
// (`EndRead`) : l'épisode est déjà borné à T1 dans les deux cas — [T0, T1] est la SEULE
// vérité consultée ici, bornes INCLUSES (un frag exactement à T0 ou à T1 compte).
//
// UN FRAG COMPTE POUR CHAQUE FAMILLE ACTIVE EN MÊME TEMPS : camo et surbouclier peuvent être
// ouverts simultanément pour le même porteur, et le frag crédite alors LES DEUX épisodes —
// ce n'est pas une avance qui s'arrête au premier trouvé.
func attachEpisodeKills(
	episodes []EquipmentEpisode, kills []EquipmentKillRef, slotXUID map[uint32]uint64,
	originMs int64, frameIntervalMS int,
) {
	if len(episodes) == 0 || len(kills) == 0 || frameIntervalMS <= 0 {
		return
	}
	for _, k := range kills {
		// Troncature simple (pas de division plancher signée comme `frameOf`) : T0 est
		// toujours >= 0 (fenêtre clampée au début de la vie publiée), donc un frag dont le
		// temps recalé est négatif ne matchera jamais aucun épisode, quel que soit
		// l'arrondi exact retenu pour la partie négative.
		frame := (k.TimeMS - int(originMs)) / frameIntervalMS
		if k.XUID != 0 {
			creditFrame(episodes, slotXUID, k.XUID, frame, false)
		}
		if k.AssistKnown && k.AssistXUID != 0 {
			creditFrame(episodes, slotXUID, k.AssistXUID, frame, true)
		}
	}
}

// creditFrame incrémente K (ou A si assist) sur chaque épisode du xuid donné dont la fenêtre
// [T0, T1] couvre frame.
func creditFrame(episodes []EquipmentEpisode, slotXUID map[uint32]uint64, xuid uint64, frame int, assist bool) {
	for i := range episodes {
		if slotXUID[episodes[i].Slot] != xuid {
			continue
		}
		if frame < episodes[i].T0 || frame > episodes[i].T1 {
			continue
		}
		if assist {
			episodes[i].A++
		} else {
			episodes[i].K++
		}
	}
}
