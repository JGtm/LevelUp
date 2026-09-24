package replay

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

// Coverage porte la couverture de chaque calque du document, et le VERDICT qui en découle.
//
// ELLE EST PUBLIÉE DANS LE DOCUMENT, pas seulement journalisée : c'est la différence entre
// un décodeur qui SAIT ce qu'il perd et un écran qui le MONTRE. Le POC l'affiche à côté de
// ses autres chiffres.
type Coverage struct {
	Shots    LayerCoverage `json:"shots"`
	Grenades LayerCoverage `json:"grenades"`
	// Tracks est ce que le SEUIL DE PUBLICATION des traces a retenu et refusé (schéma 55, cf.
	// TrackCoverage). Absente quand le film ne porte aucune position : il n'y a alors pas de
	// seuil à appliquer, et publier des zéros laisserait croire à une mesure.
	Tracks *TrackCoverage `json:"tracks,omitempty"`
	// Teams est CE QUE LE FILM DIT DES ÉQUIPES et ce que la base en pense (schéma 57, cf.
	// TeamCoverage) : les joueurs dont le film donne l'équipe, ceux qu'il laisse sans, et les
	// trois compteurs du CONTRÔLE (accord / contradiction / silence).
	//
	// ELLE EST LA SEULE FAÇON DE LIRE UN `team: -1`. Le champ ne dit pas si le mode n'a pas de
	// camps ou si le film n'a pas nommé ce joueur ; `noTeam` et `unread` le disent. Absente
	// quand le film ne porte aucune position : il n'y a alors ni roster ni vie à couvrir.
	Teams *TeamCoverage `json:"teams,omitempty"`
	// Seats est CE QUE LA POSE DES SIÈGES A LU ET CE QU'ELLE A APPARIÉ (lot 1.9.14, cf.
	// SeatCoverage) : combien d'entrées tiennent leur siège du film, combien le tiennent du
	// repli ordinal, et — le chiffre qui a ouvert le lot — combien d'occupants le film porte
	// AU PLUS EN MÊME TEMPS face au nombre d'entrées du roster. Absente quand le roster est
	// vide : il n'y a alors aucun siège à couvrir.
	Seats *SeatCoverage `json:"seats,omitempty"`
	// Projectiles est la couverture des TRAJECTOIRES DE PROJECTILE (cf. projectiles.go) :
	// pistes décodées, trajectoires publiées, et celles qu'un PAS IMPOSSIBLE a coupées.
	//
	// `truncated` EST LE CHIFFRE QUI EXISTE POUR ÊTRE VU : la coupure protège le rendu d'une
	// droite en travers de la carte, mais elle ne répare pas la déquantification qui la cause.
	// Tant que ce nombre n'est pas nul, l'artefact porte des vols dont la fin est INCONNUE — et
	// le taire ferait passer un pansement pour un correctif. Mesuré sur le parc du 2026-09-11
	// (schéma 51, avant la coupure) : 947 trajectoires sur 15 735, dont 634 sur les quatre seuls
	// films Live Fire.
	//
	// `omitempty` PARCE QUE LA FORME DU DOCUMENT NE DOIT PAS BOUGER POUR RIEN (même règle que
	// `RefusedByRoster`) : un film sans piste de projectile ne publie rien ici.
	Projectiles *ProjectileCoverage `json:"projectiles,omitempty"`
	// Objectives est la couverture du calque des actions d'objectif (cf. objectives.go).
	// Son dénominateur est le nombre d'événements identifiés fournis au build : publier
	// 40 actions sans dire que 55 existaient laisserait croire à l'exhaustivité.
	//
	// IL NE COMPTE QUE LES FAMILLES D'OBJECTIF (D.2, 2026-09-13), alors que `Objectives` du
	// document publie TOUT ce que le film nommait : les tables nommées portent aussi `kills`
	// (ancre d'identité du balayage) et `assists` (contrôle croisé), qui faisaient 119 des
	// 218 « disponibles » de `8bc6074f` et 93 des 148 de `32d9a94f` — deux artefacts qui
	// annonçaient 100 % de couverture d'objectifs. `Attached` suit la même règle, donc
	// `Balanced()` tient : l'invariant porte sur les familles d'objectif, pas sur la liste
	// publiée. LES ARTEFACTS ANTÉRIEURS GARDENT L'ANCIEN DÉNOMINATEUR jusqu'à leur recuisson —
	// aucun champ ne bouge, seule la valeur change, et `SchemaVersion` ne monte donc pas.
	Objectives LayerCoverage `json:"objectives"`
	// Equipment est la couverture des épisodes d'état actif d'équipement (schéma 7, cf.
	// equipment_episodes.go) : combien de vies publiées portent des épisodes, par famille.
	// Sa forme diffère des LayerCoverage : un épisode n'est pas un événement à rattacher
	// (le slot est DANS la lecture), le dénominateur est le nombre de vies publiées.
	// Absente des artefacts antérieurs au schéma 7.
	Equipment *EquipmentCoverage `json:"equipment,omitempty"`
	// Stances est la couverture des ETATS DE MOUVEMENT (schema 65, cf. document_stances.go) :
	// ce que la marche a lu, ce qu elle a jete, et sous quelles largeurs d axe elle a lu.
	// ABSENTE quand le balayage n a pas tourne du tout (film sans chunk).
	Stances *StanceCoverage `json:"stances,omitempty"`
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
	// VipCrown est la couverture de la COURONNE VIP (schéma 22, cf. document_vip_crown.go) :
	// les sélections `vip_selected`, les périodes publiées partagées en fermées / ouvertes, la
	// cause de fermeture (mort du VIP / sélection suivante) et les rejets par cause. Son ABSENCE
	// dit que l'appelant n'a PAS reconnu un film VIP (la garde de mode est chez `replaybuild`).
	VipCrown *VipCrownCoverage `json:"vipCrown,omitempty"`
	// SkullCarries est la couverture du PORTEUR DU CRANE d'Oddball (schéma 23, cf.
	// document_skull_carries.go) : les prises (`skull_grabs`), les trains de tics détectés, les
	// portages publiés partagés en fermés / ouverts, et les rejets par cause (sans pont, hors
	// fenêtre). Son ABSENCE dit que l'appelant n'a PAS reconnu un film Oddball (la garde de mode
	// est chez `replaybuild`).
	SkullCarries *SkullCarriesCoverage `json:"skullCarries,omitempty"`
	// BombCarries est la couverture du PORTEUR DE LA BOMBE d'Assaut (schéma 30, cf.
	// document_bomb_carries.go) : transitions lues, périodes reconstruites, publiées, et les
	// causes nommées de chaque rejet. Absente = l'appelant n'a pas reconnu un film de la
	// famille bomb ; présente et vide de portages = film d'Assaut où rien n'a pu être nommé.
	BombCarries *BombCarriesCoverage `json:"bombCarries,omitempty"`
	// BombArmings est la couverture de L ARMEMENT DE LA BOMBE d Assaut (schéma 29, cf.
	// document_bomb_armings.go) : lectures de l'anneau, segments, armements retenus, et le
	// verdict de la confrontation locale aux explosions (`suppressed`). Son ABSENCE dit que
	// l'appelant n'a PAS reconnu un film de la FAMILLE bomb (la garde de mode est chez
	// `replaybuild` — One Bomb y entre depuis le 2026-09-04, schéma 39).
	BombArmings *BombArmingsCoverage `json:"bombArmings,omitempty"`
	// WeaponChanges est la couverture des PRISES ET LACHERS d'arme (schéma 25, cf.
	// document_weapon_changes.go) : les changements décodés, ceux publiés, et ce qui a été
	// écarté — les ré-annonces d'une arme déjà portée au spawn (qui ne sont PAS des prises) et
	// les événements antérieurs à la première frame.
	//
	// ELLE EST PUBLIÉE MÊME QUAND AUCUN CHANGEMENT NE L'EST, pour la même raison que
	// `placements` et `groundWeapons` : un film sans ramassage et un film qu'on n'a pas su
	// balayer rendent tous deux zéro changement — seuls ces compteurs les distinguent.
	WeaponChanges *WeaponChangeCoverage `json:"weaponChanges,omitempty"`
	// Keyframes est la SANTÉ DE LA MARCHE D'IMAGE-CLÉ (schéma 69, lot M3.1, cf.
	// coverage_keyframes.go) : comment chaque record a été atteint, et combien de bipèdes la
	// marche a manqués entre deux images-clés qui les portaient.
	Keyframes *KeyframeCoverage `json:"keyframes,omitempty"`
	// BirthLoadouts est la couverture des DOTATIONS DE NAISSANCE (schéma 69, lot M3.2, cf.
	// document_birth_loadouts.go) : les créations lues, celles dont le record ne se ferme pas
	// (par cause), et ce que la publication a posé ou écarté.
	BirthLoadouts *BirthLoadoutCoverage `json:"birthLoadouts,omitempty"`
	// Pickups est la couverture des RAMASSAGES NATIFS (cf. document_pickups.go). Elle porte,
	// comme celle de l'équipement, un TÉMOIN DE CE QU'ELLE NE VOIT PAS : `multiEvent` compte
	// les listes d'événements qui en portent un autre après le ramassage — le balayage ne
	// décode que l'événement de tête, donc un ramassage en deuxième position lui échappe. Sans
	// ce nombre, personne ne peut juger le rappel du canal.
	//
	// PUBLIÉE MÊME QUAND AUCUN RAMASSAGE NE L'EST, comme les autres : un film sans ramassage
	// et un film qu'on n'a pas su balayer rendent tous deux zéro — seuls ces compteurs les
	// distinguent.
	Pickups *PickupCoverage `json:"pickups,omitempty"`
	// PadDating dit ce que l'événement natif a pu dater parmi les occupations de socle : les
	// datées, celles dont le ramasseur est nommé, les AMBIGUËS (plusieurs ramassages de la
	// même arme dans la fenêtre — on s'abstient) et les non couvertes (qui gardent leur
	// intervalle). C'est le compteur qui dit combien de `padPickups` sont sortis de
	// l'intervalle de vingt secondes.
	PadDating *PadDatingStats `json:"padDating,omitempty"`
	// EquipmentChanges est la couverture des RAMASSAGES ET CONSOMMATIONS d'équipement
	// (schéma 26, cf. document_equipment_changes.go). Elle porte, seule de toutes les
	// couvertures du rejeu, un TÉMOIN DE COMPLÉTUDE : le compteur de rotation d'i48 dit
	// combien d'émissions ont été MANQUÉES (`missedEstimate`). Publiée même quand aucun
	// changement ne l'est, pour la même raison que `weaponChanges`.
	EquipmentChanges *EquipmentChangeCoverage `json:"equipmentChanges,omitempty"`
	// Translocations est la couverture des TÉLÉPORTATIONS du translocateur (schéma 38, cf.
	// document_translocations.go) : têtes 117 décodées, publiées, écartées avant l'origine
	// ou sans piste, et PORTANT LEUR VA-ET-VIENT (`positioned` — la charge de l'événement a
	// pu être déquantifiée). Publiée même quand aucune ne l'est, pour la même raison que
	// `weaponChanges` : un film sans translocateur et un film qu'on n'a pas su balayer
	// rendent tous deux zéro téléportation — seuls ces compteurs les distinguent.
	Translocations *TranslocationCoverage `json:"translocations,omitempty"`
	// AbilityImpulses est la couverture des IMPULSIONS DE CAPACITÉ (schéma 38, cf.
	// document_ability_impulses.go) : lectures `tag == 1`, gestes après repliement, publiées,
	// et les CINQ façons d'être écartée — avant l'origine, sans piste, SANS IDENTITÉ (aucun
	// rang i48 dans la vie avant l'instant), FAMILLE NON MESURÉE (le canal n'est prouvé que
	// pour celles que le titre déclare) et ATTRIBUTION INDISPONIBLE (`noResolver` : palette non
	// classée, aucune famille déclarée, ou aucune vie — la chaîne d'identité n'a pas pu
	// tourner du tout, ce qui n'est ni « un autre équipement » ni « le rang n'a pas été lu »).
	// Publiée même quand aucune impulsion ne l'est, pour la même raison que `weaponChanges` —
	// et `componentAbsent` distingue en plus le film qui ne transmet PAS le composant de celui
	// dont personne ne s'est servi.
	AbilityImpulses *AbilityImpulseCoverage `json:"abilityImpulses,omitempty"`
	// AbilityCharges est la couverture des CHARGES D'ÉQUIPEMENT RESTANTES (schéma 38, cf.
	// document_ability_charges.go) : lectures armées d'i56, publiées, et les CINQ refus —
	// avant l'origine, sans piste, SANS IDENTITÉ (aucun rang i48 dans la vie avant
	// l'instant), FAMILLE NON MESURÉE (le canal n'est prouvé que pour celles que le titre
	// déclare — le répulseur reste dehors) et ATTRIBUTION INDISPONIBLE (`noResolver` :
	// palette non classée, aucune famille déclarée, ou aucune vie). La somme des six cases
	// vaut `reads`, exactement. Publiée même quand aucune lecture ne l'est, pour la même
	// raison que `abilityImpulses` — et JAMAIS quand le balayage n'a pas tourné
	// (`Scanned` faux) : publier des zéros affirmerait une lecture qui n'a pas eu lieu.
	// `componentAbsent` distingue en plus le film qui ne transmet PAS le composant de celui
	// dont personne n'use ses charges.
	AbilityCharges *AbilityChargeCoverage `json:"abilityCharges,omitempty"`
	// GroundWeaponItems est la couverture des ARMES AU SOL individuelles (schéma 27, cf.
	// document_ground_weapon_items.go) : combien d'objets, combien liés à leur lâcheur et à
	// leur ramasseur, et comment leurs fins se répartissent entre observé et ouvert. Publiée
	// même vide, même raison que `weaponChanges`.
	GroundWeaponItems *GroundWeaponItemsCoverage `json:"groundWeaponItems,omitempty"`
	// Vehicles est la couverture du calque des VÉHICULES (schéma 29, cf. document_vehicles.go) :
	// vies recensées, publiées, celles dont le châssis est lu et résolu en famille de sprite (et
	// le détail des châssis NON résolus, châssis par châssis), points de trajectoire, et les
	// épisodes d'occupation ventilés par précision de leurs bornes. Publiée même vide, même
	// raison que `placements` et `groundWeapons` : un film d'arène sans véhicule et un film qu'on
	// n'a pas su balayer rendent tous deux zéro véhicule — seul `scanned` les distingue.
	Vehicles *VehicleCoverage `json:"vehicles,omitempty"`
	// ContinuousFire est la couverture du TIR CONTINU (schema 71, lot M4b) : la lecture de la vue
	// de controle (paquets lus, trous par cause) et le sort de chaque rafale lue. Absente quand la
	// marche des trames n a pas tourne (artefact reconstruit sans elle).
	ContinuousFire *ContinuousFireCoverage `json:"continuousFire,omitempty"`
	// ObjectiveObjects est la couverture du calque des objets d'objectif LIBRES : combien le
	// manifeste en déclare de publiables, combien de vies et de points sortent, et ce qui a été
	// écarté hors axe (cf. document_objective_objects.go).
	ObjectiveObjects *ObjectiveObjectsCoverage `json:"objectiveObjects,omitempty"`
	// Inventory est la couverture du calque INVENTAIRE (munitions, grenades, capacité,
	// emplacement dégainé, cf. inventory.go) : lectures décodées, écartées avant l'origine du
	// rejeu, écartées faute de trajectoire publiée, et publiées. TÉLÉMÉTRIE PURE — absente des
	// artefacts antérieurs à ce lot, mais SANS montée de SchemaVersion : aucun rendu n'en
	// dépend (audit AUDIT_AVAL_INVENTAIRE_2026-08-24.md, point 5 ; même règle que
	// Structure/StructureBounds, cf. TestStructureIsOptionalInDocument).
	Inventory *InventoryCoverage `json:"inventory,omitempty"`
	// GrenadeReads dit ce que chaque canal a apporte a l'axe des grenades portees, et si le
	// canal MUNITIONS du film a ete refuse en bloc (cf. grenade_reads.go).
	GrenadeReads *GrenadeReadCoverage `json:"grenadeReads,omitempty"`
	// Abilities est la couverture du calque IDENTITE DE CAPACITE PORTEE (cf. abilities.go) :
	// lectures i48/image-cle disponibles, celles ecartees comme BRUIT DE BALAYAGE (rang hors
	// domaine plausible, RAPPORT_E0_2026-09-10 §3), celles sans trajectoire publiee, et
	// celles publiees. TELEMETRIE PURE, SANS MONTEE DE SCHEMAVERSION (lot 5.6, 2026-09-10) :
	// meme regle qu'Inventory ci-dessus, aucun rendu n'en depend.
	Abilities *AbilityCoverage `json:"abilities,omitempty"`
	// Zones est la couverture de L'ÉTAT DES ZONES (schéma 16, cf. document_zones.go) : la
	// MÉTHODE d'appariement employée et les rôles du catalogue qui composent `mapObjectives.zones`
	// (sans quoi `zoneRef` ne serait pas vérifiable), les slots lus, ceux qu'aucune capture n'a
	// rattachés, le contrôle du propriétaire contre l'équipe du capteur, et depuis le schéma 18
	// le nombre de points de la jauge en direct (`gaugePoints`).
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
	// T0Film est le verdict du détecteur de COUP D'ENVOI (schéma 36, cf. t0_film.go) : détecté
	// ou refusé et pourquoi, avec les deux chiffres qui fondent le verdict — la rafale de
	// départ et la marge depuis la frame 0.
	//
	// ELLE EST PUBLIÉE MÊME QUAND LE DÉTECTEUR REFUSE, pour la raison exacte de `placements`
	// et `groundWeapons` : un film sans mouvement, un film où un seul joueur part et un film
	// qui démarre trop tard rendent tous trois un `t0FilmMs` absent — seule la raison les
	// distingue. Son ABSENCE dit encore autre chose : l'origine de la frame 0 n'étant pas
	// établie, le détecteur n'a même pas été lancé (son résultat vivrait sur une horloge
	// inconnue).
	T0Film *T0FilmCoverage `json:"t0Film,omitempty"`
	// FilmMajorVersion est LA VERSION DU FILM qui a produit cet artefact, lue dans l'en-tête de
	// son registre (`grammar.FilmMajorVersionFromHeader`, u32 LE en tête de `chunk_00`).
	//
	// ELLE EST PUBLIÉE PARCE QUE LE DÉCODAGE EN DÉPEND ET QUE RIEN NE LE DISAIT. Le parc en cache
	// porte sept versions (v31 x3, v33 x3, v37 x10, v38 x1, v39 x26, v40 x185, v41 x1123 au
	// 2026-09-12) ; un seul consommateur la lisait — le découpage du gamertag du parseur d'events
	// — et tout le reste du décodeur travaille sous l'hypothèse implicite « tout est en 41 ».
	// Sans ce champ, un artefact ne dit pas sous quelle grammaire il a été cuit, et une mesure
	// par version (lot H) doit relire les films pour le savoir.
	//
	// TÉLÉMÉTRIE PURE : aucun rendu n'en dépend, et son absence dit « film sans registre, ou
	// artefact antérieur à ce lot » — d'où le pointeur plutôt qu'un zéro qui ressemblerait à une
	// version.
	FilmMajorVersion *int `json:"filmMajorVersion,omitempty"`
	// Verdict dit, calque par calque, si le résultat est publiable. Repris du chantier
	// voisin, qui sait annoncer « 371 couples sur 371, verdict nominal ».
	Verdict map[string]string `json:"verdict,omitempty"`
	// Bridge décrit sur quoi repose le pont slot -> joueur. Un calque peut être complet et
	// néanmoins reposer sur une résolution fragile : les deux se jugent séparément.
	Bridge BridgeHealth `json:"bridge"`
	// Fallbacks dit QUELLE PART DE CE DOCUMENT VIENT D'UN REPLI (schéma 58, décision D14 du plan
	// du décodeur, ADR 0034). Un repli est une décision de secours prise quand la lecture du film
	// ne tranche pas ; chacun porte un nom stable, une condition et un critère de retrait, tous
	// déclarés dans `film/facts/fallback`.
	//
	// UNE LISTE PLATE, ET PAS UN BLOC PAR FAIT. Les replis ne se répartissent pas sur les calques
	// existants — `repli_largeurs_axe_par_defaut_conservees` touche TOUT le décodage, et les
	// trois quarts des faits repliés n'ont pas de bloc de couverture à eux. Une liste
	// `{name, hits}` se lit sans connaître la carte des calques, se joint au registre par le seul
	// nom, et n'oblige aucun des vingt blocs existants à changer de forme.
	//
	// SEULS LES REPLIS DÉCLENCHÉS Y FIGURENT, triés par nom : ce que le document porte est ce qui
	// s'est PRODUIT. Ce qui PEUT se produire est au registre, qui est du code versionné. Absente
	// quand aucun repli ne s'est déclenché, ou quand la cuisson ne portait pas de compteur.
	Fallbacks []FallbackHit `json:"fallbacks,omitempty"`
	// Decoder dit SOUS QUELLES RÉVISIONS cet artefact a été cuit (schéma 61, lot 2.6.3).
	//
	// Un pointeur en `omitempty`, pour la même raison que `FilmMajorVersion` : l'ABSENCE du bloc
	// dit « artefact antérieur au schéma 61 », et c'est une réponse, pas un trou. Tout ce que ce
	// code cuit le porte.
	Decoder *DecoderCoverage `json:"decoder,omitempty"`
	// DeathsPaths dit CE QUE CHAQUE VOIE DE LECTURE DES MORTS A PROPOSE, APPARIE ET PUBLIE
	// (schema 62, lot 4.2.1-b). Meme regime que `Decoder` : pointeur en `omitempty`, l ABSENCE
	// du bloc dit « artefact anterieur au schema 62 » ou « morts non lues » — cf.
	// [DeathsPathsCoverage], qui ecrit les deux sens et ce que `SchemaVersion` tranche.
	DeathsPaths *DeathsPathsCoverage `json:"deathsPaths,omitempty"`
}

// DeathsPathsCoverage dit CE QUE CHAQUE VOIE DE LECTURE DES MORTS A PROPOSE, APPARIE ET PUBLIE
// (schema 62, lot 4.2.1-b ; mesures de M3).
//
// # LES DEUX VOIES, ET POURQUOI LEUR COMPTE ENTRE DANS L ARTEFACT
//
// Le decodage des morts lit le MEME champ par deux localisateurs (`killsource.PathWalk`, la
// MARCHE qui deroule les records depuis le debut du paquet, et `killsource.PathScan`, le SCAN
// DIRECT qui balaie les positions de bit) et, quand les deux repondent, elles repondent au meme
// bit — desaccord ZERO sur la serie de reference. Ce n est donc pas un arbitrage entre deux
// mesures, c est une PREFERENCE entre deux localisateurs, et les deux n ont pas la meme
// precision : 98,2 % contre 78,4 % au gate (b).
//
// JUSQU AU SCHEMA 61, AUCUN COMPTE PAR VOIE N ATTEIGNAIT L ARTEFACT, alors que les deux voies
// existent dans le type publie depuis toujours. Un consommateur ne pouvait donc pas PONDERER ce
// qu il lisait : une ligne de mort venue du scan et une venue de la marche s affichaient pareil,
// et un film ou le scan a tout porte etait indistinguable d un film nominal.
//
// # LE REGIME
//
//	bloc ABSENT, schema < 62   artefact anterieur a ce lot
//	bloc ABSENT, schema >= 62  le decodage des morts n a pas ete lu (`KillsInput.Read` faux :
//	                           source illisible, ou porte de publication ligne-par-ligne fermee)
//	bloc PRESENT               lu ; une voie a zero population est une MESURE (« cette voie n a
//	                           rien propose sur ce film »), pas un trou
//
// # UNE SEULE FORME, ET C EST DELIBERE
//
// Ce type est AUSSI l entree que l appelant remplit (`KillsInput.Paths`) : le producteur du
// compte est `internal/replaybuild`, qui lit `killsource.Result.Stats`. En declarer deux — une
// forme d entree et une forme publiee, converties l une dans l autre — serait la troisieme copie
// d un {population, apparie, publie} que CLAUDE.md regle 6 interdit, pour trois entiers qui ne
// se transforment pas en route.
type DeathsPathsCoverage struct {
	// Walk : la MARCHE. Rappel plus faible, precision superieure ; seule voie sans porte de
	// catalogue, donc seule capable de detecter un catalogue perime.
	Walk DeathsPathTally `json:"walk"`
	// Scan : le SCAN DIRECT. Rappel superieur, precision inferieure — c est la voie de
	// rattrapage.
	// LA CLE N EST PAS `scan`, ET C EST UN RATCHET DU DEPOT QUI LE DECIDE : `"scan"` en litteral
	// brut est interdit hors de `domain/killscope` et de `killsource` (J4R-3,
	// `archlint/no_raw_kill_scope_literal_test.go`) — c est sur cette valeur que se decide la
	// PRESEANCE des ecrivains de `shared.match_kill_events`, et une seconde copie libre de deriver
	// y rendrait la preseance du film aveugle SANS erreur ni compteur. `directScan` est le nom que
	// le code donne deja a cette voie (« le SCAN DIRECT »).
	Scan DeathsPathTally `json:"directScan"`
}

// DeathsPathTally : les trois denominateurs d une voie de lecture des morts.
type DeathsPathTally struct {
	// Population : ce que la voie a PROPOSE.
	Population int `json:"population"`
	// Matched : ce dont le couple exact existe au kill-feed dans la fenetre.
	Matched int `json:"matched"`
	// Published : ce qui a effectivement ete publie. Une mort deja couverte par une voie plus
	// contrainte ne l est pas deux fois : la somme des deux `published` ne se lit donc pas comme
	// un total de morts.
	Published int `json:"published"`
}
