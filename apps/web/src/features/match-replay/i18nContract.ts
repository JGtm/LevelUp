/**
 * i18nContract.ts — LE CONTRAT DE TEXTE du rejeu 2D : un champ par string affichée, et la
 * raison d'être de chacune en commentaire.
 *
 * EXTRAIT DE `i18n.ts` LE 2026-08-18 (lot R2-V) : le fichier portait le contrat ET les deux
 * tables de traduction, et il avait franchi le seuil de taille du dépôt (CLAUDE.md n°5, 505
 * lignes). La découpe tombe sur une ligne qui se dit en une phrase — d'un côté ce que l'UI
 * PROMET d'afficher, de l'autre ce que chaque langue en DIT.
 *
 * LA PARITÉ FR/EN RESTE TENUE PAR LE TYPAGE : `Record<ReplayLocale, ReplayText>` dans
 * `i18n.ts` refuse toute langue à laquelle il manque un champ.
 */
import type { PadControlGapKey } from './padControlLogic'
import type { PadEquipmentFamilyKey } from './weaponPadFamilies'

/**
 * LE TABLEAU DES USAGES D'ÉQUIPEMENT de la page match (onglet Chronologie). Il compte, sur tout
 * le match, ce que le rejeu ne montre qu'image par image.
 *
 * QUATRE RÉSERVES SONT PORTÉES PAR CES TEXTES, et aucune ne doit se perdre :
 *
 *  1. `groupActiveHint` — un épisode de camouflage ou de surbouclier est un ÉTAT MESURÉ, pas
 *     un geste : le film dit que l'effet court, il ne dit PAS d'où il vient (bonus ramassé au
 *     socle, ou capacité déclenchée). Compter ces épisodes comme des « utilisations d'objet »
 *     serait affirmer une origine que rien n'établit. LA MÊME PHRASE PORTE DÉSORMAIS LA
 *     RÉSERVE DES COLONNES « FRAGS SOUS <FAMILLE> » (PLAN_RETOURS_UTILISATEUR_2026-08-29
 *     §LOT F.2, DEC-7 révisée) : les bornes de l'épisode sont à la précision de la
 *     retransmission près, et le camo SEUL est sous le seuil de mesure en lecture large
 *     (26,2 % des épisodes avec ≥ 1 frag) — un même groupe, une seule infobulle, parce que les
 *     colonnes de frags sont des SOUS-COLONNES du même état mesuré, pas un calque à part.
 *  2. `powerupPadsHint` — les vidages de socle de bonus sont ANONYMES PAR MESURE
 *     (`padPickups[].xuid` est publié depuis le schéma 30 mais cet écran ne l'exploite pas) :
 *     la ligne reste au niveau du MATCH, et aucun
 *     libellé ne doit laisser croire qu'on connaît le ramasseur.
 *  3. `notMeasured` — répulseur et propulseur n'ont AUCUN canal d'activation dans le film. Pas
 *     de colonne vide (elle se lirait « zéro utilisation ») : une phrase qui le dit.
 *  4. LA CELLULE « — » DES COLONNES DE FRAGS (cf. `equipmentUsageColumns.ts`, `killsCell`) —
 *     un match dont `EquipmentUsageCoverage.killsRead` est faux écrit « — », jamais un zéro :
 *     pas un texte à part, un CARACTÈRE, identique dans les deux langues (même convention que
 *     `lib/formatters` pour toute grandeur non mesurée du dépôt).
 *
 * LES NOMS DE FAMILLE NE SONT PAS ICI, et c'est voulu : ils vivent déjà dans `placementFamily`
 * (règles de rendu) et `padEquipmentFamily` (socles de bonus). Une troisième table de noms
 * divergerait au premier ajout du manifeste.
 */
export interface EquipmentUsageText {
  title: string
  colPlayer: string
  teamTotal: string
  /** Tractions de grappin : la seule ACTIVATION de capacité que le film mesure et attribue. */
  groupGrapple: string
  groupGrappleHint: string
  /** États actifs mesurés (camouflage, surbouclier) : nombre d'épisodes et durée cumulée. */
  groupActive: string
  groupActiveHint: string
  /**
   * LE NOM COURT des deux familles d'état, et pourquoi il ne se prend nulle part ailleurs.
   *
   * `equipmentActive` porte bien ces deux familles, mais ce sont des PHRASES d'infobulle de
   * fiche (« Camouflage actif — le joueur est invisible à l'écran de jeu ») : illisibles en tête
   * de colonne. `padEquipmentFamily` porte bien deux noms courts, mais ce sont ceux des SOCLES
   * de bonus — les employer ici nommerait l'ÉTAT par la source que `groupActiveHint` dit
   * justement ne pas être établie. D'où deux libellés propres, et le typage tient la parité.
   */
  activeFamily: Record<'camo' | 'overshield', string>
  activeCount: string
  activeDuration: string
  /**
   * LES DEUX COLONNES « FRAGS SOUS <FAMILLE> » (PLAN_RETOURS_UTILISATEUR_2026-08-29 §LOT F.2,
   * décision utilisateur 8a/8b) : la somme des frags du PORTEUR pendant ses épisodes de cette
   * famille. En-tête complet plutôt que la composition `activeFamily` + suffixe (contrairement
   * à `activeCount`/`activeDuration`) parce que la phrase se lit seule en tête de colonne
   * étroite — « Frags sous camo » porte son sens, « Camouflage (Frags) » l'aurait fait deviner.
   * La réserve mesurée (source non distinguée, bornes approximatives, camo sous le seuil de
   * mesure en lecture large) est celle de `groupActiveHint`, pas un second texte : ces colonnes
   * sont des sous-colonnes du MÊME état mesuré.
   */
  activeKillsFamily: Record<'camo' | 'overshield', string>
  groupDeployed: string
  groupDeployedHint: string
  groupDropped: string
  groupDroppedHint: string
  groupGrenades: string
  groupGrenadesHint: string
  /** Repli quand le catalogue du titre ne nomme pas ce rang de grenade (le rang reste vrai). */
  grenadeRankFmt: (rank: number) => string
  /** La ligne ANONYME, au niveau du match — jamais rattachée à un joueur. */
  powerupPads: string
  powerupPadsHint: string
  powerupPadsDenomFmt: (pads: number) => string
  /** Dénominateurs repris de `doc.coverage` : « N épisodes » ne se juge pas sans eux. */
  coverageActiveFmt: (lives: number) => string
  coverageGrappleFmt: (pulls: number, lives: number) => string
  /** Gestes mesurés dont le film ne nomme pas l'auteur : comptés hors tableau, jamais versés. */
  unattributedFmt: (count: number) => string
  notMeasured: string
  /**
   * LE BADGE « TEMPS FORT » (`features/match-view/equipmentKillBadges.ts`, LOT F.3) : « N frags
   * sous camouflage » / « N frags sous surbouclier », le nombre RÉEL du meilleur épisode du
   * match — jamais arrondi, jamais une somme sur le joueur. Vit ici (et pas dans
   * `match-view/i18n.ts`) pour la même raison que le reste du vocabulaire d'équipement : les
   * noms de famille appartiennent au dictionnaire du rejeu, `match-view` l'importe (même sens
   * que `MatchEquipmentUsageSection`, réutilisé tel quel par `MatchViewTabChronology`).
   */
  killBadgeFmt: Record<'camo' | 'overshield', (kills: number) => string>
  /** Infobulle du badge : LA MÊME réserve que `groupActiveHint` (source non distinguée, bornes
   * à la précision de la retransmission, camo seul sous le seuil de mesure en lecture large). */
  killBadgeHint: string
}

/**
 * PadControlText — LE VOCABULAIRE DU CONTRÔLE DES ARMES SPÉCIALES (tableau de la page match).
 *
 * IL VIT ICI, PAS DANS `match-view/i18n.ts`, pour la raison qui vaut déjà pour le bilan
 * d'équipement : les noms d'ARME viennent du catalogue du document et de la table des familles
 * de socle (`padNameFor`), tous deux dans le dictionnaire du rejeu. Une seconde table de noms
 * côté `match-view` divergerait au premier ajout du manifeste du titre.
 *
 * CE QUE LA VENTILATION DOIT DIRE, ET POURQUOI ELLE EXISTE. Le tableau ne montre que les
 * occupations dont l'événement natif nomme le ramasseur ; toutes les autres sont réelles et
 * doivent se voir, sans quoi le lecteur croit avoir sous les yeux la totalité des socles pris du
 * match. D'où une note de bas de tableau plutôt qu'un silence, et un libellé par CAUSE — une
 * abstention pour ambiguïté n'est pas une absence de mesure.
 */
export interface PadControlText {
  title: string
  /** Infobulle du titre : d'où vient l'attribution, et ce qu'elle refuse de faire. */
  titleHint: string
  colPlayer: string
  colTotal: string
  teamTotal: string
  /** Le dénominateur : « N prises attribuées sur M occupations de socle ». */
  attributedFmt: (attributed: number, occupations: number) => string
  /** L'annonce du reste, avant la ventilation par cause. */
  missingFmt: (missing: number) => string
  /**
   * UN LIBELLÉ PAR CAUSE, et le typage tient la parité : `PadControlGapKey` énumère les cinq
   * raisons pour lesquelles une occupation reste hors tableau, et aucune ne peut être oubliée
   * dans une langue.
   */
  gapFmt: Record<PadControlGapKey, (count: number) => string>
}

export interface ReplayText {
  title: string
  back: string
  play: string
  pause: string
  restart: string
  loading: string
  empty: string
  speed: string
  time: string
  /**
   * LES SAUTS DE LA BARRE (planche 2a, 2026-08-28) : le libellé PORTE LA DURÉE plutôt que de
   * la répéter à côté du bouton — l'icône dit le sens, le nom accessible dit combien. La
   * valeur vient de `SKIP_SECONDS` (replayCanvasConfig.ts) : changer la convention à un seul
   * endroit change les deux libellés.
   */
  skipBackFmt: (seconds: number) => string
  skipForwardFmt: (seconds: number) => string
  /**
   * LES DEUX NOTES DU MENU DE VITESSE. `speedNormal` marque la vitesse de référence (on
   * cherche « comment je reviens à la normale ? », pas « qu'est-ce que 1× »). `speedMuted`
   * marque celles où le son se tait — le menu ne la pose que sur les vitesses que
   * `soundPlaysAtSpeed` (replaySoundCursor.ts) refuse, donc la borne n'est écrite nulle part
   * dans le texte : elle se lit sur les entrées qui portent la note.
   */
  speedNormal: string
  speedMuted: string
  /**
   * LE NOM DE LA BARRE D'ESPACE, et c'est la SEULE touche du lecteur qui se traduit. Les autres
   * rappels de la barre — R, M, ←, → — sont des touches physiques : leur nom est le glyphe
   * gravé dessus, identique dans les deux langues. « Espace » ne l'est pas ; l'écrire en dur
   * aurait laissé un mot français dans une interface anglaise.
   */
  keySpace: string
  /** Kill feed synchronisé sur l'horloge du rejeu. */
  killFeedTitle: string
  killFeedEmpty: string
  killFeedUnknownWeapon: string
  killFeedNoAssistHint: string
  killFeedAssistHint: string
  killFeedKillerShare: (pct: number) => string
  /**
   * L'ASSISTANCE EST UN PICTOGRAMME, PAS UN MOT (décision utilisateur du 16/08) : le fil
   * n'écrit plus « assisté par », il pose une marque puis le nom. Ce libellé est ce que la
   * marque DIT (infobulle + lecteur d'écran) — elle ne se lit pas toute seule.
   */
  killFeedAssistMark: string
  /** Part de participation de l'assistant, telle que la ligne l'écrit : « 37 % ». */
  killFeedAssistShare: (pct: number) => string
  /** Ligne de mort NEUTRE (suicide, chute, sortie) : le mot affiché et son infobulle. */
  killFeedDeathLabel: string
  killFeedDeathHint: string
  /**
   * Le TYPE de la mort neutre, quand le film l'établit. Clés = les identifiants stables
   * publiés par le document (`kind`) — un type inconnu n'a pas de libellé et n'affiche
   * aucune icône, la ligne garde son repère neutre.
   */
  killFeedDeathKind: Record<'environment' | 'suicide', string>
  /**
   * Lignes d'ENTRÉE/SORTIE de partie du fil (presenceFeed.ts, 2026-09-02). Deux sources,
   * deux vocabulaires : la PARTICIPATION API (drapeaux joined/left_in_progress, précise —
   * libellés affirmatifs) et le REPLI dérivé des bornes de vie du film (matchs sans les
   * colonnes) — dont le libellé de sortie reste au FAIT (« ne reviendra plus ») : le film
   * ne distingue pas un départ d'une élimination définitive, l'infobulle porte la réserve.
   */
  presenceJoined: string
  presenceLeft: string
  presenceJoinedHint: string
  presenceLeftHint: string
  presenceJoinedDerived: string
  presenceLeftDerived: string
  presenceJoinedDerivedHint: string
  presenceLeftDerivedHint: string
  /** Sons du rejeu (lot 5 parité) : COUPÉ PAR DÉFAUT, l'utilisateur l'active. */
  sound: string
  soundHint: string
  soundVolume: string
  /**
   * LE CURSEUR DE VOLUME QUAND LE SON EST COUPÉ (demande utilisateur du 2026-08-25 : « couper
   * le son ne doit plus faire disparaître la barre de volume »). Le curseur RESTE, à zéro et
   * inerte ; ce texte est ce qui l'empêche de se lire comme un réglage cassé — il dit l'état
   * (zéro) ET ce que fera le retour du son (le niveau réglé revient, il n'est pas perdu).
   */
  soundVolumeMutedHint: string
  /** Le son est activé mais tu par la vitesse de lecture — le dire, pas le cacher. */
  soundFastHint: string
  /** Filtre des sons par catégorie (tiroir de réglages, phase 2, décision du 16/08). */
  soundCategoriesTitle: string
  soundCategory: Record<'weapon' | 'grenade' | 'melee' | 'equipment', string>
  /**
   * LA CAPTURE D'IMAGE (2026-08-26) : un bouton en icône dans la barre de lecture, qui
   * télécharge la scène courante en PNG. Le libellé dit le GESTE, pas la technique — « PNG »,
   * « export » ou « canvas » ne veulent rien dire pour qui regarde un match.
   *
   * PAS D'INFOBULLE SÉPARÉE : ce que la commande fait tient dans son nom, et une phrase
   * d'explication sous un bouton d'une évidence pareille serait du bruit. Le
   * bouton d'enregistrement, lui, en a une — parce que son comportement, LUI, surprend.
   */
  captureImage: string
  /**
   * L'ENREGISTREMENT VIDÉO (2026-08-26) : un bouton à état, comme lecture/pause — le nom
   * accessible dit ce que le CLIC va faire, pas où l'on en est.
   */
  recordVideo: string
  stopRecording: string
  /**
   * L'INFOBULLE DU BOUTON, et elle porte deux faits qui SURPRENNENT (décisions 3 et 4) :
   *
   *  1. on filme l'écran tel qu'il défile — changer de vitesse ou déplacer le curseur
   *     pendant l'enregistrement SE VOIT dans le fichier ;
   *  2. mettre en pause, ou laisser le film finir, arrête l'enregistrement ET télécharge.
   *
   * Sans cette phrase, le second point se découvre en perdant un clip qu'on croyait encore
   * en cours. Le nom du bouton, lui, reste dans `recordVideo`/`stopRecording` : cette
   * infobulle explique le COMPORTEMENT, elle ne double pas le libellé (même partage que
   * `sound` / `soundHint`).
   */
  recordHint: string
  /**
   * LES PASTILLES DE SORTIE (planche 2a, 2026-08-28) : les trois commandes portent désormais
   * un TEXTE COURT à côté de leur icône. Il ne remplace pas le nom accessible — `captureImage`,
   * `recordVideo` et `stopRecording` restent en aria-label, et ce sont eux qu'un lecteur
   * d'écran annonce. Ce mot-ci est ce que l'ŒIL lit sans survoler : trois icônes muettes côte
   * à côte se ressemblent toutes, un mot les départage d'un coup.
   */
  /** Le tiroir de réglages (décision utilisateur du 16/08) : bouton et panneau partagent
   *  le même intitulé — ouvrir dit ce qu'on va trouver derrière. */
  settingsButton: string
  settingsClose: string
  /**
   * LA LECTURE : la seule section du tiroir qui parle du LECTEUR et non de ce qu'il montre
   * (demande utilisateur du 2026-08-29, point 22 — « lecture automatique dans les réglages,
   * avec persistance du choix »).
   *
   * L'INFOBULLE PORTE LA RÉSERVE, et elle n'est pas décorative : ce réglage ne commande pas le
   * rejeu ouvert, il décide de son état de DÉPART. Sans cette phrase, on l'essaierait comme un
   * bouton « Lecture » et on conclurait qu'il ne marche pas.
   */
  autoPlay: string
  autoPlayHint: string
  /** Calques que le lecteur peut éteindre. */
  layers: string
  layerAim: string
  layerAimHint: string
  /** Zones nommées (callouts officiels) : calque + libellé de la fiche. */
  layerZones: string
  layerZonesHint: string
  /** Noms des joueurs sous leur marqueur (calque éteignable — un BTB à 24 joueurs). */
  /**
   * TRAÎNÉE derrière chaque marqueur (V1, retour utilisateur du 2026-08-18 : « avoir la
   * traînée en option »). ALLUMÉE par défaut : c'est elle qui dit le SENS d'un déplacement,
   * et le verdict du 16/08 sur le marqueur la comptait dans le « Parfait ».
   */
  layerTrail: string
  layerTrailHint: string
  /**
   * EFFETS D'ÉVÉNEMENT, réglables séparément (décision utilisateur du 16/08) : les éclairs
   * de bouche de TOUS les tirs, et le trait tueur -> victime des éliminations. Le premier
   * porte une RÉSERVE DE MESURE affichée en clair (le film n'enregistre un tir que
   * lorsqu'un dégât est appliqué) : elle ne vit pas dans un commentaire, elle est à l'écran.
   */
  effects: string
  layerShotFx: string
  layerShotFxHint: string
  layerShotFxCoverage: string
  layerKillFx: string
  layerKillFxHint: string
  /**
   * POSES D'ÉQUIPEMENT (schéma 10) : le calque, sa bascule d'objets non identifiés, et les
   * NOMS des familles que le calque sait dessiner. Les clés de `placementFamily` sont les
   * RÈGLES DE RENDU (`PlacementKind`), pas les familles du document : deux familles qui
   * partageraient un tracé partageraient son libellé, et une famille que le calque ne dessine
   * pas n'a besoin d'aucun nom — elle n'a pas d'infobulle.
   *
   * Les noms français des trois familles ajoutées le 2026-08-18 viennent du manifeste du titre
   * (`replay_labels.toml`, `[ability_palettes.ranks]`), la même source que le fil et les
   * fiches : « traqueur de menaces » (rang 12), « champ de réparation » (rang 23),
   * « translocateur quantique » (rang 11) dont la BALISE est l'objet posé.
   *
   * LA RÈGLE `unnamed` N'Y FIGURE PAS : c'est le défaut du serveur, dessiné en point neutre et
   * nommé par `placementUnnamedLabel`. Le garde-rail `placementFamily.guard.test.ts` tient
   * cette liste alignée sur `PLACEMENT_RENDER` (calque) et sur `equipmentFamilies` (Go).
   */
  layerPlacements: string
  layerPlacementsHint: string
  /**
   * OBJETS DE PUISSANCE LÂCHÉS (décision produit du 2026-08-18). La commande apparaît dès que
   * le film en porte, dans TOUS les modes — la restriction « hors Fiesta » a été levée le
   * 2026-08-20. Son libellé n'a donc toujours pas à parler de Fiesta, et pour une raison
   * désormais plus simple : le mode n'entre plus dans la règle.
   */
  layerPlacementsDropped: string
  layerPlacementsDroppedHint: string
  layerPlacementsUnnamed: string
  layerPlacementsUnnamedHint: string
  placementFamily: Record<'wall' | 'sensor' | 'rift' | 'shroud' | 'seeker' | 'field', string>
  /** Ce que dit le point neutre d'un objet dont la nature n'est pas établie. */
  placementUnnamedLabel: string
  /** Ligne « posé par <joueur> » de l'infobulle ; le poseur est une MESURE (proximité). */
  placementOwnerFmt: (name: string) => string
  /** Poseur non mesuré (aucun bipède contemporain à moins de 3 m) : le dire, pas le taire. */
  placementOwnerUnknown: string
  /**
   * L'INFOBULLE D'UN OBJET LÂCHÉ : ce qu'il est, qui l'a lâché, et à quel instant du rejeu.
   *
   * `placementDroppedOwnerFmt` remplace « posé par » : le geste n'est pas le même, et écrire
   * « posé » sur un objet tombé d'un cadavre serait faux. `placementDroppedAtFmt` reçoit un
   * chronomètre déjà formaté (`m:ss`) — un INSTANT et jamais une durée écoulée.
   */
  placementDroppedLabel: string
  placementDroppedOwnerFmt: (name: string) => string
  placementDroppedAtFmt: (clock: string) => string
  /**
   * SOCLES D'ARME (schéma 11) : le calque, ses trois états, son compte à rebours et son cycle.
   *
   * LES TROIS ÉTATS NE SE VALENT PAS, et les libellés doivent le dire : « Disponible » est une
   * présence PROUVÉE, « Pris » une absence PROUVÉE, « Incertain » l'intervalle de ~20 s entre
   * les deux, que le film ne date pas. Aucun de ces libellés ne nomme un joueur : le ramasseur
   * n'est pas AFFICHÉ (`padPickups[].xuid` est publié depuis le schéma 30 ; cet écran ne
   * l'exploite pas), et aucune ligne d'écran ne doit
   * laisser croire qu'on le connaît.
   *
   * NI MÉDIANE NI ÉCARTS À L'ÉCRAN (verdict du 2026-08-18) : le cycle mesuré ne sert plus qu'à
   * DATER le compte à rebours, il ne s'affiche plus. `padCycleFmt` a donc été retiré avec sa
   * ligne d'infobulle — du texte qu'aucun écran ne rend est du code mort (CLAUDE.md n°7).
   */
  layerWeaponPads: string
  layerWeaponPadsHint: string
  /**
   * LES DRAPEAUX DE CTF (schéma 15) : le calque et son infobulle. Titre au PLURIEL — une partie
   * de capture en porte deux, et c'est leur opposition qui se lit.
   *
   * LE CAMP SE DIT « allié » / « adverse », jamais par une couleur ni un nom d'équipe : la page
   * ne connaît que le point de vue du joueur (accessibilité du rejeu), et un film dont la ligne
   * « moi » manque n'a AUCUN camp allié — d'où la troisième clé, qui ne nomme que l'objet.
   *
   * L'ÉTAT « porté, fin non datée » PORTE SA RÉSERVE DANS SON LIBELLÉ, et `flagOpenNote` la
   * développe : l'intervalle court jusqu'à la fin du film, c'est une BORNE HAUTE. Une icône
   * atténuée seule se lirait comme un effet de style.
   */
  layerFlagCarries: string
  layerFlagCarriesHint: string
  flagSide: Record<'ally' | 'enemy' | 'unknown', string>
  flagState: Record<'carried' | 'carried_open' | 'dropped' | 'home', string>
  /** Le porteur que le tableau de bord ne nomme pas — jamais un identifiant brut à l'écran. */
  flagCarrierUnknown: string
  /** Depuis combien de temps l'état courant dure (infobulle). */
  flagSinceFmt: (seconds: number) => string
  /** La réserve de `carried_open`, en toutes lettres. */
  flagOpenNote: string
  /** LA COURONNE VIP (schéma 22) : le nom du calque et sa réserve, identiques FR/EN (« VIP »). */
  layerVipCrown: string
  layerVipCrownHint: string
  /** LE PORTEUR DU CRÂNE d'Oddball (schéma 23) : le nom du calque et sa réserve. */
  layerSkullCarrier: string
  layerSkullCarrierHint: string
  /**
   * LA BOMBE d'Assaut (schéma 30) : le nom du calque et sa réserve — portée elle suit son
   * porteur, lâchée elle reste au dernier point du lâcheur jusqu'à la reprise ou l'explosion.
   */
  layerBombCarrier: string
  layerBombCarrierHint: string
  /**
   * LES ARMES AU SOL (schéma 27) : le nom du calque et sa réserve.
   *
   * LA RÉSERVE EST LE SUJET, pas un ornement : ce calque affiche des objets dont la
   * disparition n'est PAS toujours datée. La réserve doit dire, en toutes lettres, ce que
   * l'estompage veut dire — sans quoi il se lit comme un effet de style, exactement le défaut
   * que `flagOpenNote` corrige sur les portages ouverts.
   */
  layerGroundWeapons: string
  layerGroundWeaponsHint: string
  /**
   * LE NOM D'UN SOCLE QUI NE PORTE PAS UNE ARME (schéma 17). Les clés sont les familles
   * d'équipement publiées par le document (`weaponPads[].weapon`), énumérées une par une dans
   * `weaponPadFamilies.ts` — d'où le typage : une famille ajoutée là-bas sans libellé ici ne
   * compile pas, dans AUCUNE des deux langues.
   *
   * POURQUOI CES LIBELLÉS SONT LOCAUX ET NON SERVIS PAR LE DOCUMENT : le manifeste du titre
   * (`replay_labels.toml`) ne porte, sur ses `[[equipment_objects]]`, que l'identité (famille,
   * provenance, nature) — aucun libellé bilingue, aucune icône — et aucun canal du document ne
   * transporte de libellé d'équipement au client (il n'existe pas de `equipmentLabels`). Les
   * noms des familles DESSINÉES vivent déjà ici, dans `placementFamily` : ceux-ci les
   * rejoignent, avec les mêmes mots que le jeu.
   */
  padEquipmentFamily: Record<PadEquipmentFamilyKey, string>
  /** Compte à rebours COMPACT, celui de la carte (« 12 s ») — il ne dit pas sa source. */
  padCountdownFmt: (seconds: number) => string
  /**
   * LES DEUX COMPTES À REBOURS DE L'INFOBULLE, et leur différence est la seule chose qui compte
   * (D3, 2026-08-27) : `Measured` vise la prochaine apparition VUE dans le film — le rejeu
   * connaît la suite, le chiffre est EXACT et n'a pas à porter de « ≈ » ; `Expected` vise ce que
   * le CYCLE prédit, pour le dernier trou qu'aucune apparition ne ferme, et garde sa réserve.
   *
   * DEUX CLÉS ET NON UN DRAPEAU dans une seule phrase : les deux langues n'insèrent pas la
   * réserve au même endroit, et une phrase à trous se serait figée sur l'ordre du français.
   *
   * SANS PARENTHÈSE D'EXPLICATION depuis le 2026-08-28 (« je ne veux pas de blabla dedans ») :
   * la source du chiffre tient dans le « ≈ », et l'infobulle ne porte plus que deux lignes.
   */
  padRespawnMeasuredFmt: (seconds: number) => string
  padRespawnExpectedFmt: (seconds: number) => string
  /**
   * Carte de chaleur : le calque, ce qu'il mesure, et sa légende. JAMAIS « heatmap » à
   * l'écran (règle FR sans anglicismes) — « carte de chaleur » partout.
   */
  layerHeatmap: string
  layerHeatmapHint: string
  heatmapReading: string
  heatmapMode: Record<'presence' | 'kills', string>
  heatmapModeHint: Record<'presence' | 'kills', string>
  /**
   * PORTÉE DE TEMPS de la carte de chaleur (V2, 2026-08-18) : toute la partie — la lecture
   * d'analyse d'après-match, et le défaut — ou jusqu'à l'image courante, la carte qui se
   * remplit en même temps que le rejeu.
   */
  heatmapSpanTitle: string
  heatmapSpan: Record<'match' | 'live', string>
  heatmapSpanHint: Record<'match' | 'live', string>
  /**
   * COULEUR DES POINTS des joueurs (option du tiroir, 2026-08-24) : par équipe — le défaut,
   * la couleur dit le camp (D1) — ou distincte par joueur, pour suivre quelqu'un dans la
   * mêlée. Les hints disent ce que chaque lecture garde et ce qu'elle déplace.
   */
  markerColorsTitle: string
  markerColorsMode: Record<'team' | 'player', string>
  markerColorsHint: Record<'team' | 'player', string>
  /** Extrémités de la légende : elles nomment la QUANTITÉ, le titre dit de quoi il s'agit. */
  heatLegendLow: string
  heatLegendHigh: string
  heatLegendHint: string
  /** Fiches joueur : ce qui est lu, et ce qui ne l'est pas. */
  rosterEmpty: string
  bridgeDiag: (named: number, total: number, collisions: number) => string
  teamUnknown: string
  /** Libellé d'équipe (cascade `lib/halo/teamLabel.ts`, mêmes textes que la Match View). */
  teamLabelFmt: (name: string) => string
  teamNumberedFmt: (n: number) => string
  /**
   * LE SCORE PERSONNEL À L'INSTANT LU (schéma 12) — celui du joueur sur sa fiche. Ce n'est
   * PAS le score final : il tique avec la lecture. Le score d'ÉQUIPE, lui, ne vit plus
   * qu'au bandeau (les chiffres en tête de colonne sont partis le 2026-08-24).
   */
  playerScoreLive: string
  /** Frags / morts / assistances : deux grandeurs qui ne se confondent pas. */
  countersLive: string
  countersMatch: string
  /**
   * LE FDA du triplet affiché — le net canonique (frags + assistances/3 − morts, cf.
   * `lib/fda.ts`). Le fond coloré du triplet le dit d'un coup d'oeil ; l'infobulle le dit
   * en toutes lettres, parce qu'une couleur seule n'est pas une mesure.
   */
  fdaTooltipFmt: (counters: string, fda: string) => string
  /**
   * LE BANDEAU DE SCORE au-dessus du terrain : deux barres de camp encadrant l'horloge de
   * lecture. Les deux camps s'y nomment par leur RAPPORT au joueur de la page (allié /
   * adverse) et non par leur nom d'équipe, qui est déjà en tête des colonnes de fiches —
   * le bandeau dit un affrontement, la colonne dit une identité.
   *
   * `roundNumberFmt` est le rang seul, affiché sous l'horloge ; `roundOfCountFmt` le
   * situe dans le match pour l'infobulle. Aucun des deux ne porte de valeur : entre deux
   * barres, un nombre sans camp ne se rattacherait à personne.
   */
  scoreBannerLabel: string
  scoreBannerAlly: string
  scoreBannerEnemy: string
  scoreBannerClock: string
  /**
   * La progression de GARDE de la colline (KOTH) : le libelle d'un filet, pas d'un nombre.
   * Il nomme la grandeur — sans lui, un lecteur d'ecran annoncerait un second pourcentage
   * sans dire de quoi, a cote de celui du score.
   */
  hillHoldAlly: string
  hillHoldEnemy: string
  roundNumberFmt: (index: number) => string
  roundOfCountFmt: (index: number, count: number) => string
  /**
   * LES PASTILLES DE MANCHE, au-dessus du score (Oddball et tout mode multi-manche). Une
   * rangée COMMUNE, une pastille par manche jouée, teintée au camp gagnant quand la manche est
   * tranchée — pleine = gagnée, vide = en cours ou à jouer. `roundDotsLabel` nomme la rangée
   * pour les lecteurs d'écran ; les trois formateurs disent l'état de chaque pastille (l'œil lit
   * la couleur, le lecteur d'écran lit le mot). Le camp est nommé par son RAPPORT au joueur de
   * la page (allié / adverse), comme le bandeau — jamais par une couleur ni un nom d'équipe.
   */
  roundDotsLabel: string
  /**
   * LE COMPTE DE MANCHES ÉCRIT EN CLAIR, à côté du rang de manche sous l'horloge (arbitrage
   * utilisateur du 2026-08-29). Les pastilles le disent déjà à l'œil ; ce libellé le dit au
   * lecteur d'écran et à qui compte mal des ronds. Toujours « allié - adverse », l'ordre du
   * bandeau. Il se lit À L'IMAGE COURANTE, comme les pastilles : c'est un compte EN COURS,
   * pas le verdict du match (celui-ci vient de l'API, sur l'écran de fin).
   */
  roundsTallyFmt: (ally: number, enemy: number) => string
  roundsTallyLabel: string
  roundDotAllyFmt: (index: number) => string
  roundDotEnemyFmt: (index: number) => string
  roundDotPendingFmt: (index: number) => string
  /**
   * LE MESSAGE INTER-MANCHE : bref et non bloquant, il paraît à la bascule d'une manche à la
   * suivante et dit la manche qui vient de se TERMINER. Dérivé de la position de lecture comme
   * l'écran de fin — visible dans une courte fenêtre autour de la bascule, il se rejoue si l'on
   * repasse dessus. C'est aussi le point de déclenchement du son « manche terminée ».
   */
  roundOverFmt: (index: number) => string
  /**
   * LE COMPTE À REBOURS DE LA BOMBE (Assaut, schéma 29) : le bandeau affiché pendant la mèche,
   * de l'armement à l'explosion. `remaining` arrive déjà formaté (`formatSeconds`, « 4.9 s ») —
   * le texte ne porte que le fait : la bombe est armée, il reste ce temps.
   */
  bombArmedFmt: (remaining: string) => string
  /**
   * L'ÉCRAN DE FIN DE MATCH, à l'instant où la lecture atteint la fin déclarée.
   *
   * IL N'Y A PAS DE CLÉ DE TITRE ICI, ET C'EST VOULU (amendement utilisateur du 2026-08-26) :
   * le titre est `header.outcome_label`, que le backend sert DÉJÀ localisé (« Victoire » /
   * « Défaite » / « Égalité »). Le fabriquer une deuxième fois côté front, c'est deux mots
   * pour le même verdict sur deux pages du même match — exactement ce que la règle de
   * réutilisation existe pour empêcher. Ne restent ici que les textes PROPRES à cet écran.
   *
   * `victoryPanelLabel` nomme la région pour les lecteurs d'écran. Le panneau apparaît SANS
   * geste de l'utilisateur (la lecture arrive au bout) : sans nom de région, l'annonce
   * `aria-live` tomberait dans le vide.
   *
   * `victoryScoreLabel` désigne la ligne de score FINAL — celui de la fin du match, pas celui
   * de l'image lue : le bandeau au-dessus dit déjà le second, et les deux se ressemblent trop
   * pour rester anonymes l'un à côté de l'autre.
   */
  victoryPanelLabel: string
  victoryScoreLabel: string
  /**
   * LES QUATRE PISTES DE LA FRISE (planche 2a, 2026-08-28). Les trois premières nomment ce
   * qu'on lit sous le curseur : tes éliminations et tes morts, celles de tes alliés, et qui
   * menait à cet instant. Ce sont des ÉTIQUETTES DE LIGNE, pas des titres — d'où des mots
   * seuls, à l'échelle d'une frise haute de quelques pixels.
   *
   * `dominanceOfFmt` date une bande de dominance dans son infobulle : l'équipe y est nommée
   * par la cascade du scoreboard (`labelOf`), la même que les colonnes et le bandeau. Elle
   * dit AUX FRAGS depuis le 2026-08-28 : la piste ne lit plus le compteur du mode (captures,
   * secondes de balle) mais le nombre d'éliminations de chaque camp — « mène » tout court
   * laisserait croire au score du tableau.
   *
   * `dominanceTied` nomme la bande d'ÉGALITÉ (l'encre `outcome-draw`), qui est l'état du coup
   * d'envoi et de tout retour à parité : sans elle, la bande bleue serait la seule de la piste
   * dont le survol ne dirait rien.
   */
  trackYou: string
  trackAllies: string
  trackDominance: string
  dominanceOfFmt: (team: string) => string
  dominanceTied: string
  /**
   * LA PISTE SCORE (2026-08-28) : la même lecture que la dominance, mais sur le compteur du
   * MODE. Elle n'apparaît pas en Slayer, où le score EST le compte des frags — d'où trois
   * chaînes jumelles et non partagées : « mène aux frags » et « mène au score » sont deux
   * affirmations différentes, et les fondre en une seule (« mène ») rendrait les deux rangées
   * indiscernables au survol, précisément là où on les compare.
   *
   * Les séparateurs de manche de cette piste réutilisent `roundOverFmt` — le même mot que
   * l'écran inter-manche, pour la même chose.
   */
  trackScore: string
  scoreOfFmt: (team: string) => string
  scoreTied: string
  /**
   * LA PISTE DES MÉDIAS, son état VIDE, et la lightbox qui ouvre un média.
   *
   * LA DONNÉE EST ARRIVÉE le 2026-08-28 (phase 2 : l'onglet médias du match, recalé sur l'axe
   * du rejeu) : « aucun média » reste un état honnête, mais c'est désormais un fait DU MATCH
   * et non plus l'attente d'une source. `mediaPausedHint` est la pastille de la lightbox :
   * ouvrir un média met le rejeu en pause, et sans ce mot un lecteur qui referme ne saurait
   * pas pourquoi le film n'a pas avancé.
   *
   * LES DEUX ÉCHECS DE LECTURE SONT DISTINCTS, et le lecteur n'y peut pas la même chose : un
   * navigateur sans HLS ne lira JAMAIS ce clip (il faut en changer), tandis qu'un flux en
   * erreur peut être réessayé. Un message unique les confondrait.
   */
  mediaTrack: string
  mediaEmpty: string
  mediaOpen: string
  mediaClose: string
  mediaPausedHint: string
  mediaHlsUnsupported: string
  mediaHlsError: string
  /**
   * REPLI DE LA FRISE (retour utilisateur du 2026-08-28). Les deux libellés portent le GESTE
   * OFFERT, pas l'état courant : c'est ce que le lecteur va déclencher en cliquant, et c'est la
   * convention des noms accessibles de commandes. L'état, lui, se lit dans `aria-expanded`.
   */
  tracksCollapse: string
  tracksExpand: string
  unknownPlayer: string
  /**
   * `markMe` — dernier vestige textuel des glyphes d'identité du fil. Le rond du joueur actif
   * ne se DESSINE plus nulle part depuis le 2026-08-24 ; le glyphe « ami » qui restait au fil
   * (ex-`PlayerMark.tsx`) est retiré à son tour le 2026-09-02 (décision D5) — sa clé
   * `markFriend` part avec lui. `markMe` ne sert donc plus qu'au libellé lecteur d'écran du
   * joueur de la page (retirer un dessin ne doit pas retirer une information).
   */
  markMe: string
  healthLabel: string
  shieldLabel: string
  abilityLabel: string
  loadoutUnread: string
  loadoutAge: string
  loadoutAhead: string
  weaponSecondaryHint: string
  /** Badge de lancer sur la fiche (le `.gic` du POC) : l'auteur est écrit dans le film. */
  grenadeThrown: string
  /** L'encadré de la fiche morte (option 2a) : le mot d'état, puis le décompte lu. */
  eliminatedLabel: string
  respawnIn: string
  goneLabel: string
  goneValue: string
  /** Ligne d'inventaire : grenades, capacité, munitions. */
  inventoryAge: string
  inventoryAhead: string
  grenadeSelected: string
  grenadeSelectedRead: string
  grenadeSelUnknown: string
  /**
   * ÂGE PROPRE À LA BOÎTE DE GRENADES (schéma 20) : l'axe `grenadeReads` ne tombe pas sur les
   * images-clés de l'inventaire, la boîte ne peut donc pas emprunter l'âge de la rangée. Un âge
   * négatif est une lecture À VENIR et se dit comme telle — jamais déguisée en passé.
   */
  grenadeAge: string
  grenadeAhead: string
  /**
   * Capacité absente de la table du titre : la fiche pose un GLYPHE NEUTRE (pas un
   * caractère, décision utilisateur du 16/08) et dit en infobulle ce qu'on ne sait pas —
   * le RANG lu reste écrit, parce qu'il est la seule chose de vraie à cet endroit.
   */
  abilityUnidentified: (rank: number) => string
  abilityAge: string
  abilityAhead: string
  /**
   * État ACTIF d'un équipement, par FAMILLE mesurée (jamais un libellé libre) : le
   * camouflage rend la fiche vitreuse, le surbouclier l'encadre d'or (cahier des
   * charges Notion 21.1, rendu par `ReplayTeams.tsx`). La clé est l'identifiant
   * stable publié par le document (`fam`) — une famille inconnue n'a pas de libellé et
   * ne reçoit aucun effet.
   */
  equipmentActive: Record<'camo' | 'overshield', string>
  /**
   * LE PASSAGE PAR TRANSLOCATEUR (mesuré : deux canaux concordants, cf. placementTeleport.ts) —
   * la phrase de l'éclat violet puis jaune-orangé que la fiche porte brièvement après le saut.
   * Hors de `equipmentActive` PARCE QUE la clé n'est pas une famille d'épisode du document :
   * c'est un ÉVÉNEMENT reconstruit des pistes, pas un état mesuré par le canal des épisodes.
   */
  translocationFlash: string
  /**
   * LE JOUEUR DANS UNE ZONE D'ÉQUIPEMENT (cf. equipmentZones.ts) : les trois états que la
   * fiche sait dire — champ de réparation, écran occultant, capteur adverse. Les clés sont
   * les RÈGLES DE RENDU du calque (`PlacementKind`), la même convention que `placementFamily`
   * ci-dessus : la zone de la fiche et le disque de la carte sont le même objet.
   */
  zonePresence: Record<'field' | 'shroud' | 'sensor', string>
  /**
   * LE PORTEUR D'OBJECTIF (cf. objectiveMark.ts) : la phrase que le filigrane de la fiche dit
   * en toutes lettres dans l'infobulle. Les clés sont les GENRES DE MARQUE, jamais les modes —
   * un même objet se retrouve d'un mode à l'autre, et la fiche ne connaît pas le mode.
   *
   * DEUX RÉGIMES DANS LA MÊME TABLE, ET LES LIBELLÉS DOIVENT LE DIRE : `flag`, `skull`, `vip`
   * et `hill` sont des ÉTATS qui durent — « porte », « est » ; `zone` et `bomb` sont des
   * ÉVÉNEMENTS tenus quelques secondes — « vient de » — parce que la donnée n'attribue à un
   * joueur que l'INSTANT, jamais la durée d'une capture ni le port de la bombe.
   */
  objectiveCarry: Record<'flag' | 'skull' | 'vip' | 'hill' | 'zone' | 'bomb', string>
  /**
   * LE TABLEAU DES USAGES D'ÉQUIPEMENT (page match, onglet Chronologie). Bloc à part parce que
   * ces textes ne servent PAS le rejeu lui-même : ils servent son BILAN, une autre surface.
   */
  equipmentUsage: EquipmentUsageText
  /**
   * LE TABLEAU DU CONTRÔLE DES ARMES SPÉCIALES (onglet Chronologie, sous le bilan d'équipement).
   * Le seul écran du dépôt qui NOMME le ramasseur d'un socle — cf. `PadControlText`.
   */
  padControl: PadControlText
  /** Pictogramme « munitions pleines » (emplacement jamais écrit) : décision produit 4. */
  ammoFullLabel: string
  ammoDrawnHint: string
  drawnUnknown: string
  /**
   * Lecture d'inventaire VIDE (schéma 19, `inventory[].empty`). Le LIBELLÉ se lit à l'écran,
   * l'INDICE porte l'explication et se termine par l'âge de la lecture VIDE — la sienne, pas
   * celle de l'équipement affiché à côté : les deux lectures ne datent pas du même instant, et
   * les confondre ferait passer un état de vingt secondes pour frais.
   *
   * DEUX ÉTATS ET PAS UN : `dead` est corroboré par le fil des éliminations, `empty` ne l'est
   * pas. Les fondre sous un seul mot ferait affirmer une mort qu'aucune pièce n'établit.
   *
   * LA SECONDE MOITIÉ DE L'INFOBULLE DIT CE QUE L'ÉCRAN MONTRE À CÔTÉ, et elle a DEUX formes
   * parce qu'il y a deux situations. `inventoryFallbackHint` (suivi de l'âge de la lecture
   * PLEINE) quand une lecture pleine antérieure est effectivement substituée ;
   * `inventoryNoPriorHint` quand il n'en existe AUCUNE — la fiche n'affiche alors aucun
   * équipement, et promettre « la dernière lecture pleine » serait faux.
   */
  inventoryDeadLabel: string
  inventoryDeadHint: string
  inventoryEmptyLabel: string
  inventoryEmptyHint: string
  inventoryFallbackHint: string
  inventoryNoPriorHint: string
  gaugeLabel: string
  /**
   * L'EXPORT HORS TEMPS REEL — la commande, son dialogue, et ce qu'il annonce pendant qu'il
   * calcule.
   *
   * `exportHint` dit la SEULE chose qui distingue cette commande de l'enregistrement qu'elle
   * remplace : le fichier ne se paie pas en temps de match. `exportRunningHint` prévient que
   * le terrain va défiler très vite sous les yeux — sans cela, un utilisateur croirait à un
   * emballement du rejeu.
   */
  exportVideo: string
  exportHint: string
  exportDialogTitle: string
  exportFrom: string
  exportTo: string
  exportWithSound: string
  exportStart: string
  exportCancel: string
  exportClose: string
  exportRunningHint: string
  /** « Image 4 200 / 18 000 » : ce que la barre de progression dit en toutes lettres. */
  exportProgressFmt: (done: number, total: number) => string
  /**
   * « Plage exportée : 4:32 » — la durée de MATCH demandée, et non celle du fichier : quand la
   * plage va jusqu'au bout, le clip tient sa dernière image quelques secondes de plus, le temps
   * qu'on lise le verdict et que le son s'achève.
   */
  exportLengthFmt: (clock: string) => string
  /** L'export a ECHOUE : le dialogue le dit au lieu de se vider en silence. */
  exportFailed: string
  /**
   * LA PHASE DE PREPARATION, avant la premiere image encodee : polices, logo, decodage des
   * sons, mixage hors ligne, encodage de la piste. Elle dure plusieurs secondes sur un match
   * charge en sons, et la barre y resterait sinon a zero sans rien expliquer.
   */
  exportPreparing: string
  /** « ~1:20 restantes » — l'estimation, absente tant qu'elle danserait. */
  exportEtaFmt: (clock: string) => string
  /** « Fichier depose : rejeu-....mp4 » — la fin se dit, avec le nom du fichier. */
  exportDoneFmt: (filename: string) => string
  /** Le son etait demande, le navigateur l'a refuse : le clip est muet, et on le DIT. */
  exportMutedFallback: string
  /**
   * LES NOMS DES PISTES SONORES du clip. Ils sont ECRITS DANS LE FICHIER : c'est ce qu'un
   * montage affiche dans sa liste de pistes, et la seule chose qui distingue « bruitages » de
   * « voix » pour qui ouvre le clip six mois plus tard.
   *
   * `exportTrackMix` vient TOUJOURS EN PREMIER : un lecteur ordinaire ne joue que la premiere
   * piste, et un navigateur n'expose meme pas les autres.
   */
  exportTrackMix: string
  exportTrackSfx: string
  exportTrackVoice: string
  exportTrackMusic: string
}
