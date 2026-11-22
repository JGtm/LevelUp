/**
 * cardGabarit.ts — LES COTES DES FICHES JOUEUR, en objets, par forme de fiche.
 *
 * POURQUOI DES OBJETS. Les cotes d'une fiche (`WEAPON_CELL_W`, `GRENADES_BOX_W`, `COUNT_CELL_W`,
 * `GRENADE_ICON_PX`, `HUD_ICON_PX`, `WATERMARK_PX`...) vivaient dans quatre fichiers, et les hauteurs de
 * jauges étaient écrites dans la fiche. Les composants reçoivent DES NOMBRES ET DES BOOLÉENS, jamais un mot de
 * densité : ils rendent ce qu'on leur donne, et c'est l'appelant qui choisit lequel des objets donner (une
 * prop `compact` drillée avec sa branche `if (compact)` dans chaque rangée est la forme que le commit
 * `10fe3228a` a supprimée).
 *
 * `GABARIT_COMPACT` porte les cotes de la tuile de la rangée des équipes (tuile 115 × 62, corps de 31), seule
 * forme de `CardGabarit`. Le bloc « À l'instant » de la fiche du joueur a ses cotes propres
 * (`COTES_BLOC_INSTANT`), dont la hauteur de l'encadré « Éliminé » sur une ligne.
 *
 * TROIS COTES RESTENT DES CLASSES, PAS DES NOMBRES APPLIQUÉS EN STYLE : la hauteur du corps (`h-[31px]`), la
 * taille du nom (`text-[11.5px]`) et la typographie du triplet (corps, gap, marge du fond FDA) sont des valeurs
 * arbitraires Tailwind, qui n'existent que si le littéral est écrit en clair dans une classe — une interpolation
 * ne produit AUCUNE règle, en silence (`rosterHeight.guard.test.ts` documente le piège). `bodyPx` et `namePx`
 * sont donc ici la VÉRITÉ que les composants traduisent en classe par une TABLE FERMÉE
 * (`Record<CardGabarit['bodyPx']...>`) : leurs types sont des LITTÉRAUX, pour que la table soit close à la
 * compilation — une cote qui n'y figure pas ne compile pas, elle ne rend pas une classe vide. Le garde-rail
 * `ui/cardGabarit.guard.test.ts` interdit qu'une cote revienne en constante dans un composant.
 */

export interface CardGabarit {
  /** Largeur de la cellule d'arme (celle de la main), px. */
  weaponCellW: number
  /**
   * Largeur minimale d'UN compteur du triplet F/M/A, px — et l'ÉCHELLE du triplet : sa typographie (corps mono
   * 9 px, gap 2, fond 3 px) est écrite en littéraux Tailwind dans `ReplayCountersBadge`.
   */
  countCellW: 10
  /** Hauteur de la jauge de bouclier, px (au-dessus de la santé : l'ordre du jeu). */
  gaugeShieldPx: number
  /** Hauteur de la jauge de santé, px. */
  gaugeHealthPx: number
  /**
   * Hauteur FIXE du corps de la fiche, px — la même vivant et mort (règle du 2026-08-24). C'est aussi la cote
   * STRUCTURELLE de l'encadré « Éliminé » sur deux lignes, lue dans une table fermée par cette valeur.
   */
  bodyPx: 31
  /** Côté de la vignette d'un type de grenade, px. */
  iconGrenadePx: number
  /** Côté de la vignette de capacité, px. */
  iconAbilityPx: number
  /** Côté du filigrane de porteur d'objectif, px (critère : « reste un fond »). */
  watermarkPx: number
  /** Nombre d'éclairs de l'écran occultant (largeurs absolues sur une tuile étroite). */
  boltCount: number
  /** Taille du nom, px — une seule valeur aujourd'hui, en classe (`text-[11.5px]`). */
  namePx: 11.5
}

/**
 * La tuile compacte de la maquette A2 (plan 2026-09-06, « Le gabarit tranché ») : 115 × 62,
 * trois lignes (nom 14 · jauges + triplet 12 · arme 48 + grenade 14 + capacité 16 sur 16),
 * corps fixe de 31 px. Ce qui n'est pas sur la tuile passe en infobulle avec sa valeur. Rendu par
 * `ReplayPlayerCard` (trois lignes) et par l'encadré « Éliminé » à deux lignes (`ReplayVitality`).
 */
export const GABARIT_COMPACT: CardGabarit = Object.freeze({
  weaponCellW: 48,
  countCellW: 10,
  gaugeShieldPx: 4,
  gaugeHealthPx: 2,
  bodyPx: 31,
  iconGrenadePx: 14,
  iconAbilityPx: 16,
  watermarkPx: 34,
  boltCount: 2,
  namePx: 11.5,
} as const)

/**
 * LES COTES DE LA RANGÉE (Cinéma, Plein cadre) : deux camps de fiches autour du bandeau de score.
 * Ce sont des cotes de mise en page, pas de fiche : `model/rowLayout.ts` les lit, et rien ne les
 * recopie. `ficheMinW` est le MINIMUM d'une fiche de rangée ; au-delà, la rangée répartit la place
 * du corps à parts égales. `scoreCellW` est la largeur de la cellule du score. `ecartFiches` sépare deux fiches voisines. La marge
 * latérale du corps, de chaque côté, est une cote PAR CADRE (`cotesCorps.ts`).
 */
export const COTES_RANGEE = Object.freeze({
  ficheMinW: 115,
  ecartFiches: 6,
  scoreCellW: 270,
} as const)

/**
 * LES COTES DE LA RANGÉE EN GRANDE ÉQUIPE : par camp, des EMPLACEMENTS de deux fiches compactes empilées
 * (`ReplayCompactCard`, forme `'stacked'`) au lieu d'une fiche par place. `emplacementsParCamp` est le
 * nombre d'emplacements que la règle de tenue (`model/rowLayout.ts`) compte par camp, `fichesParEmplacement`
 * le nombre de fiches empilées dans chacun : un camp se range donc jusqu'à leur produit, au-delà la rangée
 * cède la place à la colonne. `ficheMinW` est le MINIMUM d'une fiche compacte (au-delà, les emplacements se
 * partagent la place à parts égales : 87,5 px à 1440 px de corps), `ecartEmplacementPx` l'écart entre deux
 * fiches d'un même emplacement. L'écart entre emplacements, le score et les marges sont ceux de
 * `COTES_RANGEE` ; la hauteur de la rangée est celle de deux fiches (`GABARIT_FICHE_COMPACTE.stackedPx`) et
 * de cet écart, soit 74 px.
 */
export const COTES_RANGEE_GRANDE_EQUIPE = Object.freeze({
  emplacementsParCamp: 6,
  fichesParEmplacement: 2,
  ficheMinW: 80,
  ecartEmplacementPx: 6,
} as const)

/**
 * LES COTES DU BLOC « À L'INSTANT » de la fiche du joueur (`ui/ReplayPlayerNow.tsx`), en px. Le bloc
 * est une pile de rangées à écart constant : une rangée de jauge est une grille de trois colonnes
 * (libellé, piste, valeur), une rangée d'arme, de grenade ou d'équipement porte une vignette suivie
 * de son texte.
 */
export const COTES_BLOC_INSTANT = Object.freeze({
  /** Écart entre deux rangées, et entre les colonnes d'une rangée. */
  ecart: 8,
  /** Hauteur de la piste d'une jauge et rayon de ses coins. */
  jaugePx: 5,
  jaugeRayonPx: 3,
  /** Largeur de la colonne du libellé d'une jauge, et de celle de sa valeur. */
  jaugeLibelleW: 62,
  jaugeValeurW: 30,
  /** Part de l'encre du thème dans la piste d'une jauge, en pour-cent. */
  pistePct: 10,
  /** Largeur et hauteur de la vignette d'une arme. */
  armeIconeW: 24,
  armeIconeH: 11,
  /** Côté de la vignette d'une grenade et de celle de l'équipement. */
  grenadeIconePx: 13,
  equipementIconePx: 13,
  /** Hauteur du corps de l'encadré « Éliminé » du bloc, sur une ligne. */
  corpsEliminePx: 35,
} as const)

/** Les hauteurs de corps que l'encadré « Éliminé » sait remplir : celle de la tuile compacte et celle du bloc « À l'instant ». */
export type CorpsEncadrePx = CardGabarit['bodyPx'] | (typeof COTES_BLOC_INSTANT)['corpsEliminePx']

/**
 * LE GABARIT « LIGNE » : les cotes d'une ligne de joueur de la colonne des équipes (Standard, et
 * Cinéma quand la rangée ne tient pas), rendue par `ui/ReplayRosterRow.tsx`. Troisième forme d'un
 * joueur après la fiche et la tuile, et la seule qui n'est PAS un `CardGabarit` : une ligne n'a ni
 * corps à hauteur de tuile, ni cellules d'arme à largeur de fiche — elle a une grille de colonnes.
 * Toutes les cotes sont appliquées en `style` (nombres), jamais en classe interpolée.
 *
 * LA GRILLE : nom et jauges (souple), arme, grenades, équipement, triplet F / M / A, séparés par
 * `colGapPx`. L'en-tête de colonnes reprend les mêmes largeurs, pour que les titres tombent sur
 * leurs colonnes.
 */
export interface LigneGabarit {
  /** Hauteur d'une ligne de joueur, et d'une place en attente (même boîte). */
  rowPx: number
  /** Hauteur de l'en-tête de colonnes. */
  headPx: number
  /** Hauteur de l'en-tête d'un camp, côté de sa pastille et écart entre pastille et nom. */
  teamHeadPx: number
  teamDotPx: number
  teamHeadGapPx: number
  /** Écart vertical entre deux blocs de camp. */
  teamsGapPx: number
  /** Écart entre colonnes d'une ligne. */
  colGapPx: number
  weaponColW: number
  grenadeColW: number
  equipmentColW: number
  kdaColW: number
  /** Tailles de texte, px. */
  namePx: number
  mentionPx: number
  teamNamePx: number
  headTextPx: number
  respawnPx: number
  kdaPx: number
  countPx: number
  /** Hauteur de la ligne du compte à rebours, qui prend la place des jauges. */
  respawnLinePx: number
  /** Écart entre le nom et ce qui le suit (jauges ou compte à rebours). */
  nameGapPx: number
  /** Largeur de la pile de jauges ; hauteurs du bouclier (au-dessus) et de la santé ; écart entre les deux. */
  gaugeWidthPx: number
  gaugeShieldPx: number
  gaugeHealthPx: number
  gaugeGapPx: number
  /** Vignette d'arme (largeur, hauteur) et hauteur de sa cellule ; vignettes de grenade et d'équipement. */
  iconWeaponW: number
  iconWeaponH: number
  weaponCellH: number
  iconGrenadePx: number
  iconEquipmentPx: number
  /** Opacité du nom, des icônes et du triplet d'un joueur mort. */
  deadOpacity: number
  /** Côté du filigrane de porteur d'objectif, et nombre d'éclairs de l'écran occultant. */
  watermarkPx: number
  boltCount: number
  /** Le rayon des couches d'effets : celui de la ligne, en classe (un littéral en clair). */
  layerRadius: string
}

/** La ligne du Standard : 34 px, grille 22 / 24 / 14 / 54, jauges de 130 px. */
export const GABARIT_LIGNE: LigneGabarit = Object.freeze({
  rowPx: 34,
  headPx: 16,
  teamHeadPx: 20,
  teamDotPx: 8,
  teamHeadGapPx: 8,
  teamsGapPx: 14,
  colGapPx: 6,
  weaponColW: 22,
  grenadeColW: 24,
  equipmentColW: 14,
  kdaColW: 54,
  namePx: 12.5,
  mentionPx: 11,
  teamNamePx: 12,
  headTextPx: 10.5,
  respawnPx: 11,
  kdaPx: 12.5,
  countPx: 11,
  respawnLinePx: 13,
  nameGapPx: 4,
  gaugeWidthPx: 130,
  gaugeShieldPx: 3,
  gaugeHealthPx: 3,
  gaugeGapPx: 1,
  iconWeaponW: 20,
  iconWeaponH: 9,
  weaponCellH: 16,
  iconGrenadePx: 11,
  iconEquipmentPx: 11,
  deadOpacity: 0.4,
  watermarkPx: 34,
  boltCount: 2,
  layerRadius: 'rounded-md',
} as const)

/**
 * LES COTES DU DÉPLI de la ligne du joueur suivi (`ui/ReplayRosterRowDetail.tsx`) : deux colonnes
 * de rangées (arme en main et arme en réserve, grenades, équipement), écart vertical puis
 * horizontal, marges haute et basse. Reprend le bloc de détail de la ligne.
 */
export const COTES_DEPLI_LIGNE = Object.freeze({
  colonnes: 2,
  ecartV: 4,
  ecartH: 10,
  margeHaute: 2,
  margeBasse: 8,
  textePx: 11.5,
  petitPx: 11,
  /** Écart entre une vignette et son texte. */
  ecartIcone: 6,
  armeIconeW: 20,
  armeIconeH: 9,
  grenadeIconePx: 11,
  equipementIconePx: 11,
} as const)

/**
 * LE GABARIT DE LA FICHE COMPACTE d'une Grande équipe (`ui/ReplayCompactCard.tsx`) : nom, bouclier et
 * santé, sans frags / morts / assistances, sans arme, grenades ni équipement. Une seule fiche, deux
 * formes : SUR DEUX LIGNES (`stackedPx` : le nom, puis les jauges) pour la rangée du Cinéma, SUR UNE
 * LIGNE (`linePx` : pastille, nom, jauges de `gaugeLineW`) pour la colonne du Standard, où les fiches se
 * rangent par paires (`pairColGapPx`, `pairRowGapPx`). Cotes en `style` (nombres), jamais en classe
 * interpolée ; le rayon des couches d'effets est une classe en clair.
 */
export interface FicheCompacteGabarit {
  /** Hauteur de la fiche sur deux lignes, et sur une ligne. */
  stackedPx: number
  linePx: number
  /** Marge latérale intérieure. */
  padX: number
  /** Côté de la pastille de camp, écart entre pastille et nom (deux lignes), écart entre éléments (une ligne). */
  dotPx: number
  dotGapPx: number
  lineGapPx: number
  /** Écart entre la ligne du nom et celle des jauges (deux lignes) et entre jauges et marques (deuxième ligne). */
  stackGapPx: number
  marksGapPx: number
  /** Nom : taille et interligne ; compte à rebours : taille et interligne. */
  namePx: number
  nameLeadingPx: number
  respawnPx: number
  respawnLeadingPx: number
  /** Pile de jauges : largeur sur une ligne, hauteurs bouclier et santé, écart entre les deux. */
  gaugeLineW: number
  gaugeShieldPx: number
  gaugeHealthPx: number
  gaugeGapPx: number
  /** Marque de porteur d'objectif (côté) ; pastille « bot » (hauteur, marge latérale, taille de texte). */
  markPx: number
  capHeightPx: number
  capPadX: number
  capTextPx: number
  /** Opacité du nom d'un joueur éliminé. */
  deadOpacity: number
  /** Nombre d'éclairs de l'écran occultant. */
  boltCount: number
  /** Rayon des couches d'effets : celui de la fiche, en classe (un littéral en clair). */
  layerRadius: string
  /** Écarts entre deux fiches d'une même paire, et entre deux paires. */
  pairColGapPx: number
  pairRowGapPx: number
}

/** La fiche compacte : 34 px sur deux lignes, 24 px sur une ligne, pastille de 6 px, jauges de 3 px. */
export const GABARIT_FICHE_COMPACTE: FicheCompacteGabarit = Object.freeze({
  stackedPx: 34,
  linePx: 24,
  padX: 6,
  dotPx: 6,
  dotGapPx: 4,
  lineGapPx: 6,
  stackGapPx: 3,
  marksGapPx: 5,
  namePx: 11.5,
  nameLeadingPx: 14,
  respawnPx: 11,
  respawnLeadingPx: 13,
  gaugeLineW: 44,
  gaugeShieldPx: 3,
  gaugeHealthPx: 3,
  gaugeGapPx: 1,
  markPx: 11,
  capHeightPx: 13,
  capPadX: 4,
  capTextPx: 9.5,
  deadOpacity: 0.4,
  boltCount: 2,
  layerRadius: 'rounded-md',
  pairColGapPx: 6,
  pairRowGapPx: 3,
} as const)
