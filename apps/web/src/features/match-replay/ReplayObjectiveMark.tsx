/**
 * ReplayObjectiveMark — LE FILIGRANE DE PORTEUR D'OBJECTIF sur une fiche joueur.
 *
 * LE CANAL A ÉTÉ CHOISI PARCE QU'IL ÉTAIT LE SEUL LIBRE (planche du 2026-08-29). La fiche a un
 * nombre fini d'endroits où poser un signe, et les effets d'équipement les occupent presque
 * tous : la BORDURE porte déjà trois cadres (surbouclier `legendary`, champ de réparation
 * `success`, capteur adverse en pointillés), le FOND ambiant porte le verre du camouflage, le
 * voile de l'écran occultant et la teinte de mort, l'INCRUSTATION porte le nuage, les éclairs,
 * les croix et le fourreau de translocation, et les ÉCLATS brefs sont réservés aux événements
 * de vie. Restait le FILIGRANE DERRIÈRE LE CONTENU : personne dessus, et rien à négocier avec
 * les autres effets — d'où cette couche, et pas une rangée de plus (contrainte utilisateur :
 * « je ne veux pas rajouter de lignes/rangées sur les fiches »).
 *
 * LE NOM N'EST PAS RECOLORÉ, et c'est un arbitrage, pas un oubli (décision utilisateur du
 * 2026-08-29). La couleur du gamertag CODE DÉJÀ vivant/mort (`text-foreground` / `text-
 * muted-foreground`) : lui faire porter en plus l'objectif surchargerait un canal qui dit
 * autre chose, et entrerait en concurrence avec la couleur de camp de la colonne.
 *
 * CE QU'IL COÛTE EN LISIBILITÉ, dit franchement : le filigrane vit SOUS le contenu, donc il se
 * COMPOSE avec les fonds — sous le voile de l'écran occultant, il perd la moitié de sa présence.
 * C'est le prix du seul canal libre ; l'infobulle de la fiche, elle, dit la phrase en toutes
 * lettres dans tous les cas (`playerCardFx.titleOf`), et c'est elle qui tient la promesse
 * d'accessibilité — d'où `aria-hidden` ici.
 *
 * LES GLYPHES REPRENNENT CEUX DE LA CARTE, forme pour forme : hampe + fanion pour le drapeau,
 * boule à deux orbites pour le crâne (cf. `skullGlyph.ts`), couronne pour le VIP. Ils sont
 * RÉÉCRITS en SVG et non partagés parce que les calques de la carte tracent sur un canvas
 * impératif : aucune géométrie n'y est exposée sous une forme que le DOM saurait lire. Deux
 * écritures d'une même forme, donc — la limite de la règle « ≤ 2 copies », à ne pas franchir
 * sans publier un chemin SVG commun.
 */
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'

import type { ObjectiveMarkKind } from './objectiveMark'

/**
 * Opacité du filigrane. 0,22 est le plus haut palier qui laisse les barres de vitalité et la
 * rangée d'armes se lire par-dessus sans halo — au-delà, le glyphe cesse d'être un fond et
 * devient un objet de la fiche, ce que la place ne permet pas.
 */
const WATERMARK_OPACITY = 0.22

/** Côté du glyphe, en pixels : la hauteur du corps de la fiche (35 px), moins l'air autour. */
const WATERMARK_PX = 46

/**
 * Les six glyphes, en repère 16×16.
 *
 * LE CRÂNE ET LA BASE SE CREUSENT PAR `fill-rule="evenodd"` plutôt qu'en repeignant leurs trous
 * à la couleur du fond : un trou peint tiendrait de la couleur exacte de la tuile, qui varie
 * (dégradé en vie, teinte rouge en mort, verre du camouflage par-dessus). Évidé, le glyphe
 * laisse voir CE QU'IL Y A derrière, quel que soit ce que les autres effets y ont mis.
 */
const GLYPH_PATHS: Record<ObjectiveMarkKind, string> = {
  // Hampe + fanion — la silhouette du drapeau porté de la carte (cf. flagCarriesLayer).
  flag: 'M4 15.2V1.6a.8.8 0 0 1 1.6 0v13.6a.8.8 0 0 1-1.6 0Z M5.6 2.4h7.6l-1.9 2.9 1.9 2.9H5.6Z',
  // Boule + deux orbites creusées — la forme canonique du crâne (cf. skullGlyph.ts).
  skull: 'M8 1.5a6.5 6.5 0 1 0 0 13 6.5 6.5 0 0 0 0-13Z M5.7 6.2a1.6 1.6 0 1 0 0 3.2 1.6 1.6 0 0 0 0-3.2Z M10.3 6.2a1.6 1.6 0 1 0 0 3.2 1.6 1.6 0 0 0 0-3.2Z',
  // Couronne — le même dessin que la couronne de la carte (cf. vipCrownLayer).
  vip: 'M1.8 4.9 4.6 7.3 8 2.6l3.4 4.7 2.8-2.4L12.9 12.6H3.1Z',
  // Colline : un mamelon posé sur son socle, distinct de la couronne à petite taille.
  hill: 'M1.4 12.6 6.4 3.9a1.8 1.8 0 0 1 3.2 0l5 8.7Z',
  // Base : hexagone évidé, la forme des zones du calque d'objectifs.
  zone: 'M8 1.4l5.7 3.3v6.6L8 14.6 2.3 11.3V4.7Z M8 4.6l3 1.7v3.4L8 11.4 5 9.7V6.3Z',
  // Bombe : corps rond, collier, et mèche partant en biais. La mèche est ce qui la distingue
  // du crâne à cette taille — c'est elle, pas le corps, qui porte l'identification.
  bomb: 'M7.4 5.9a4.6 4.6 0 1 0 0 9.2 4.6 4.6 0 0 0 0-9.2Z M6.3 4.3h2.2v1.6H6.3Z M8.5 4.9c1.4-.6 2.5-1.5 2.9-2.9l1.5.5c-.6 2-2 3.2-3.7 3.9Z',
}

/**
 * ReplayObjectiveMark — le filigrane, aligné à droite de la tuile et débordant légèrement :
 * un objet POSÉ SUR la fiche plutôt qu'un motif centré, pour que le regard le prenne comme un
 * fond et non comme une colonne de plus.
 *
 * `absolute` sans `relative` autour : la couche est déclarée AVANT les rangées de contenu, qui
 * sont `relative` — l'ordre de peinture du DOM les met donc au-dessus, exactement comme la
 * couche d'effets de `playerCardFx`. Aucun `z-index` à arbitrer.
 */
export function ReplayObjectiveMark({ kind }: { kind: ObjectiveMarkKind }) {
  return (
    <span
      aria-hidden
      className="pointer-events-none absolute inset-y-0 right-[-6px] flex items-center overflow-hidden rounded-lg"
      style={{ color: tokenCssVar('extreme'), opacity: WATERMARK_OPACITY }}
    >
      <svg width={WATERMARK_PX} height={WATERMARK_PX} viewBox="0 0 16 16" fill="currentColor">
        <path d={GLYPH_PATHS[kind]} fillRule="evenodd" />
      </svg>
    </span>
  )
}
