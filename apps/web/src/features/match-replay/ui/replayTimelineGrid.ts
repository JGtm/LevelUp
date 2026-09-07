/**
 * replayTimelineGrid — LES DEUX COLONNES DE LA FRISE, définies une seule fois (2026-09-06).
 *
 * # POURQUOI UN MODULE POUR DEUX LONGUEURS
 *
 * La frise se pose sur une grille à deux colonnes : la COLONNE DE GAUCHE — le MENU de joueurs
 * en première rangée, puis les libellés « Coéquipiers », « Dominance », « Score », « Médias » —
 * puis les PISTES elles-mêmes. (Elle a porté « Toi » et « Alliés » jusqu'au 2026-09-07 : le lot
 * L3 remplace le premier par le menu et le second par les COÉQUIPIERS du joueur regardé.) Trois
 * lecteurs ont besoin des mêmes nombres, et c'est ce qui les sort des classes Tailwind du
 * composant :
 *
 *  1. la grille des PISTES (`ReplayTimelineTracks`) ;
 *  2. la grille du TRANSPORT — chevron et curseur — qui doit s'aligner au pixel sur la première,
 *     sans quoi le curseur ne lit plus l'axe qu'il commande ;
 *  3. le TRAIT DE LECTURE (`ReplayPlayhead`), qui ne traverse QUE la colonne des pistes et doit
 *     donc savoir où elle commence.
 *
 * Écrits en classes (`grid-cols-[76px_1fr]`, `gap-x-3`) et recopiés dans le calcul du trait, ces
 * deux nombres auraient dérivé au premier élargissement de la colonne. Cet élargissement a eu
 * lieu le 2026-09-07 (lot L3) : la colonne est passée de 76 à 100 px pour loger le MENU de
 * joueurs qui remplace le libellé « Toi » — un gamertag ne tient pas dans la largeur d'un mot de
 * quatre lettres. Une seule ligne a changé ici, et les deux grilles comme le trait ont suivi ;
 * c'est exactement ce que cette centralisation achetait. Une définition, trois lecteurs.
 *
 * # CE QU'IL N'EST PAS
 *
 * Ce n'est pas la géométrie de PISTE : la position d'un instant sur une piste
 * (`calc(8px + (100% - 16px) * r)`, la demi-largeur que réserve le curseur natif) vit dans
 * `model/replayTimelineTracksLogic.ts` et n'a rien à faire ici. Les deux se composent — colonne
 * d'abord, position dans la colonne ensuite — mais elles ne changent pas pour les mêmes raisons.
 */

/**
 * La colonne des LIBELLÉS de piste.
 *
 * 100 px depuis le 2026-09-07 (décision 6 du plan « frise, point de vue »), contre 76 px
 * auparavant : la première rangée n'y porte plus un mot mais un MENU de joueurs, et un gamertag
 * tronqué à 76 px ne se reconnaît pas. La liste ouverte, elle, montre les noms entiers — c'est
 * le navigateur qui la dimensionne, pas cette constante.
 */
export const LABEL_COLUMN = '100px'

/** L'écart entre les libellés et les pistes (l'équivalent exact de `gap-x-3`). */
export const COLUMN_GAP = '0.75rem'

/** Les colonnes, telles que les DEUX grilles de la frise les posent. */
export const TIMELINE_GRID_COLUMNS = {
  gridTemplateColumns: `${LABEL_COLUMN} 1fr`,
  columnGap: COLUMN_GAP,
}

/**
 * LE BORD GAUCHE DE LA COLONNE DES PISTES, vu de la grille entière. Le trait de lecture part de
 * là : posé sur toute la largeur, il barrerait aussi les libellés — un nom traversé d'un trait
 * ne se lit plus.
 */
export const TRACKS_COLUMN_LEFT = `calc(${LABEL_COLUMN} + ${COLUMN_GAP})`
