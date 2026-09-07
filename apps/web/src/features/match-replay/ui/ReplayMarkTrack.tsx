/**
 * ReplayMarkTrack — UNE PISTE DE MARQUES : les éliminations, les morts et les médailles posées
 * sur l'axe de la frise.
 *
 * EXTRAIT DE `ReplayTimelineTracks.tsx` LE 2026-09-07 (lot L4). L'hôte franchissait le seuil de
 * taille du dépôt en accueillant l'ombrage de présence et l'anneau de médaille, et la règle est
 * d'extraire par RESPONSABILITÉ plutôt que de relever le plafond. La découpe tombe sur une
 * frontière nette : l'hôte compose les rangées, ce fichier dessine ce qui se pose DANS une
 * rangée de marques.
 *
 * # LA GÉOMÉTRIE N'EST PAS ÉCRITE ICI
 *
 * `trackLeft(ratio)` vit dans `model/replayTimelineTracksLogic.ts` et rend la position d'un
 * instant dans la géométrie du curseur natif (qui réserve sa demi-largeur à chaque bout). Le
 * garde-rail `timelineGeometry.guard.test.ts` interdit d'en recopier les littéraux ici — c'est ce
 * qui garantit que la marque, le trait de lecture et la pastille tombent au même endroit.
 *
 * # UNE MARQUE EST CENTRÉE SUR SON INSTANT
 *
 * Décision utilisateur du 2026-09-06, prise en même temps que le trait de lecture. Elle se posait
 * par son BORD GAUCHE : large de deux à trois pixels, elle débordait tout entière vers la droite,
 * et son milieu — ce que l'œil lit comme « l'endroit » de la marque — tombait un pixel et demi
 * après le frag. La translation vaut la demi-largeur de la marque quelle qu'elle soit
 * (`-translate-x-1/2` se mesure sur l'élément) : les marques hautes et les basses n'ont pas la
 * même largeur et n'ont pas à s'en soucier.
 *
 * # UN AMI PREND LE LOSANGE, UNE MÉDAILLE PREND L'ANNEAU
 *
 * Deux grammaires qui ne se marchent pas dessus, parce qu'elles ne parlent pas de la même chose.
 * La FORME dit l'identité (décision 4, transposée de la carte : `MarkerShape = 'diamond'`) ; le
 * CONTOUR dit la médaille (décision 14). La COULEUR, elle, dit le camp — et elle est déjà prise
 * deux fois, kill à l'encre alliée, mort à l'encre adverse, sur deux tokens qui reprennent les
 * couleurs d'équipe choisies par l'utilisateur en jeu. Il n'en restait aucune de libre, d'où
 * deux grandeurs distinctes plutôt qu'une troisième teinte.
 *
 * # CE QUE LES ENFANTS FONT LÀ
 *
 * L'ombrage de présence (`ReplayPresenceShade`) se pose SOUS les marques, dans le même
 * conteneur : il en est le fond, pas un voisin. Le passer en `children` plutôt qu'en prop garde
 * cette piste ignorante de la présence — elle ne sait que « quelque chose se dessine derrière ».
 */
import type { ReactNode } from 'react'

import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'

import { trackLeft, type TrackMark } from '../model/replayTimelineTracksLogic'

/**
 * Une piste de marques. `tall` distingue celle du joueur REGARDÉ (marques pleines, plus hautes,
 * sur une rangée de dix-huit pixels) de celle de ses coéquipiers (plus basses, atténuées, sur
 * quatorze) : deux pistes de même poids se liraient comme une seule.
 */
export function ReplayMarkTrack({
  marks, height, tall, children,
}: {
  marks: readonly TrackMark[]
  height: string
  tall: boolean
  /** L'ombrage de présence, dessiné SOUS les marques (cf. l'en-tête). */
  children?: ReactNode
}) {
  return (
    <div className={`relative ${height} rounded-full bg-muted/40`}>
      {children}
      {marks.map((m) => (
        <span
          key={m.key}
          className={`pointer-events-none absolute -translate-x-1/2 ${markShape(m, tall)}`}
          style={{ left: trackLeft(m.ratio), background: markInk(m.kind) }}
          title={markTitle(m)}
        />
      ))}
    </div>
  )
}

/**
 * LA SILHOUETTE D'UNE MARQUE : barre verticale par défaut, LOSANGE pour un ami (décision 4),
 * ANNEAU CREUX pour une médaille d'objectif (décision 14). Un kill médaillé garde sa silhouette
 * et reçoit l'anneau par-dessus — la marque dit toujours ce qu'elle est, le contour ajoute qu'elle
 * a valu quelque chose.
 *
 * TOUTES LES VARIANTES PARTAGENT LEUR CENTRE VERTICAL. Sur la piste du joueur regardé, y = 9 px
 * sur dix-huit (`top-[5px]` + 8 px de haut, `top-[6px]` + 6 pour le losange, `top-[5px]` + 8 pour
 * l'anneau) ; sur celle des coéquipiers, y = 7 px sur quatorze (`top-1` + 6, `top-[4px]` + 6).
 * Sans cette égalité, une marque amie flotterait un pixel plus haut que ses voisines et la piste
 * se lirait comme deux lignes. Le losange est plus court que la barre parce qu'il occupe sa
 * DIAGONALE une fois tourné (6 px de côté ≈ 8,5 px en travers) : à taille égale il dépasserait.
 *
 * LA PISTE DU JOUEUR REGARDÉ EST PASSÉE DE 14 À 18 px LE 2026-09-07, et les `top` des variantes
 * hautes ont suivi de deux pixels : il fallait loger l'anneau et la porte de présence sans les
 * rogner. La piste des coéquipiers, elle, N'A PAS BOUGÉ — elle ne porte ni l'un ni l'autre, et
 * l'écart de hauteur entre les deux rangées est justement ce qui dit laquelle est le sujet.
 *
 * L'ATTÉNUATION reste celle de la piste, pas de la forme : la piste des coéquipiers est plus
 * discrète que celle du joueur regardé, qu'on y soit ami ou non.
 *
 * L'ANNEAU EST UNE COULEUR STRUCTURELLE (`ring-foreground`, l'encre du texte), exception assumée
 * du skill `color-tokens` et de même nature que le trait de lecture : les deux encres sémantiques
 * du rejeu désignent un CAMP, et « médaillé » n'en désigne aucun.
 */
function markShape(mark: TrackMark, tall: boolean): string {
  const anneau = mark.medals.length > 0 ? ' ring-1 ring-foreground' : ''
  // L'ANNEAU CREUX N'EXISTE QUE SUR LA PISTE DU JOUEUR REGARDÉ (cf. `buildEventTracks`) : les
  // médailles ne décorent pas la piste des coéquipiers, aucune marque `medal` n'y arrive.
  if (mark.kind === 'medal') return 'top-[5px] h-2 w-2 rounded-full ring-1 ring-foreground'
  if (mark.friend) {
    return tall
      ? 'top-[6px] h-1.5 w-1.5 rotate-45 rounded-[2px]' + anneau
      : 'top-[4px] h-1.5 w-1.5 rotate-45 rounded-[2px] opacity-65' + anneau
  }
  return tall
    ? 'top-[5px] h-2 w-[3px] rounded-[2px]' + anneau
    : 'top-1 h-1.5 w-[2px] rounded-[2px] opacity-65' + anneau
}

/**
 * L'ENCRE D'UNE MARQUE : les deux encres du rejeu pour un kill et une mort, RIEN pour une
 * médaille d'objectif — son anneau est tout son dessin, et le remplir d'une des deux couleurs
 * lui ferait dire un camp qu'elle ne dit pas.
 */
function markInk(kind: TrackMark['kind']): string {
  if (kind === 'medal') return 'transparent'
  return tokenCssVar(kind === 'kill' ? 'team-ally' : 'team-enemy')
}

/**
 * L'INFOBULLE D'UNE MARQUE : l'instant, et les médailles quand il y en a — « 0:42 — Double frag,
 * Vengeance ». Aucune image dans la piste : à trois pixels de large une icône de médaille serait
 * une tache, et le fil la montre déjà en grand.
 */
function markTitle(mark: TrackMark): string {
  return mark.medals.length > 0 ? `${mark.clock} — ${mark.medals.join(', ')}` : mark.clock
}
