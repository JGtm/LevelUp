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
 * # UN AMI PREND LE LOSANGE, UNE MÉDAILLE PREND SON BADGE
 *
 * Deux grammaires qui ne se marchent pas dessus, parce qu'elles ne parlent pas de la même chose.
 * La FORME dit l'identité (décision 4, transposée de la carte : `MarkerShape = 'diamond'`) ; la
 * MÉDAILLE, elle, EN IMAGE depuis le 2026-09-09 (décision D7, plan « vague C, les formes ») — un
 * badge du jeu vaut mieux qu'un anneau d'un pixel, invisible en pratique. La COULEUR, elle, dit le
 * camp — et elle est déjà prise deux fois, kill à l'encre alliée, mort à l'encre adverse, sur deux
 * tokens qui reprennent les couleurs d'équipe choisies par l'utilisateur en jeu. Il n'en restait
 * aucune de libre pour la médaille, d'où un badge séparé plutôt qu'une troisième teinte.
 *
 * CE QUE L'ANCIEN ANNEAU (décision 14 du plan « frise, point de vue ») FAISAIT ET NE FAIT PLUS.
 * Il décorait la marque de kill/mort EN PLACE : un contour d'un pixel autour d'une marque de
 * trois par huit pixels. Mesuré illisible à l'écran (retour utilisateur du 2026-09-08). Le badge
 * ne décore plus la marque, il se pose EN SURIMPRESSION à côté d'elle (cf. `MedalBadges`, déjà
 * employé par le fil des éliminations) : la marque garde sa silhouette nue, le badge dit la
 * médaille.
 *
 * # LE BADGE EST LA SECONDE EXCEPTION À « LES PISTES NE CAPTENT PAS LE POINTEUR »
 *
 * La vignette de média (`ReplayTimelineTracks`) est la première : un bouton doit recevoir le
 * clic. Le badge de médaille est la seconde, pour une autre raison — son infobulle (titre ET
 * description, décision D7) est un `title` natif du navigateur, qui n'apparaît qu'au SURVOL d'un
 * élément qui reçoit le pointeur. `pointer-events` est une propriété HÉRITÉE : l'envelopper dans
 * un span `pointer-events-none` (comme les marques) aurait éteint le survol de l'image qu'il
 * contient, sans qu'aucun test de rendu ne le voie. Le risque symétrique — un badge qui recouvre
 * la porte de présence au même pixel — est écarté en pratique : les deux ne se posent jamais au
 * même instant (la porte marque une frontière d'ombre, le badge un kill ou une médaille).
 *
 * # CE QUE LES ENFANTS FONT LÀ
 *
 * L'ombrage de présence (`ReplayPresenceShade`) se pose SOUS les marques, dans le même
 * conteneur : il en est le fond, pas un voisin. Le passer en `children` plutôt qu'en prop garde
 * cette piste ignorante de la présence — elle ne sait que « quelque chose se dessine derrière ».
 */
import type { ReactNode } from 'react'

import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'

import { MedalBadges } from './MedalBadges'
import { trackLeft, type TrackMark } from '../model/replayTimelineTracksLogic'

/**
 * Une piste de marques. `tall` distingue celle du joueur REGARDÉ (marques pleines, plus hautes,
 * sur une rangée de vingt-quatre pixels) de celle de ses coéquipiers (plus basses, atténuées, sur
 * quatorze) : deux pistes de même poids se liraient comme une seule.
 *
 * SEULE LA PISTE `tall` REÇOIT DES BADGES : `buildEventTracks` ne pose de médaille QUE sur la
 * piste du point de vue (cf. son en-tête) — la piste des coéquipiers ne porte donc jamais de
 * `TrackMark.medals` non vide, et le badge n'est dessiné que là où il peut apparaître.
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
      {/* LES MARQUES DE KILL/MORT NUES : une médaille orpheline (`kind === 'medal'`) n'a pas de
          silhouette propre depuis que le badge remplace l'anneau — son seul dessin est le badge
          ci-dessous. Un kill médaillé, lui, garde sa marque ET reçoit le badge à côté. */}
      {marks.filter((m) => m.kind !== 'medal').map((m) => (
        <span
          key={m.key}
          className={`pointer-events-none absolute -translate-x-1/2 ${markShape(m, tall)}`}
          style={{ left: trackLeft(m.ratio), background: markInk(m.kind) }}
          title={markTitle(m)}
        />
      ))}
      {tall && marks.filter((m) => m.medals.length > 0).map((m) => (
        <span
          key={`${m.key}-medal`}
          // `pointer-events` N'EST PAS RÉGLÉ ICI (cf. l'en-tête) : le défaut `auto` laisse
          // l'image du badge recevoir le survol qui déclenche son infobulle native.
          className="absolute top-1 flex -translate-x-1/2 items-center gap-0.5"
          style={{ left: trackLeft(m.ratio) }}
        >
          <MedalBadges medals={m.medals} />
        </span>
      ))}
    </div>
  )
}

/**
 * LA SILHOUETTE D'UNE MARQUE : barre verticale par défaut, LOSANGE pour un ami (décision 4). Une
 * médaille ne passe plus par cette fonction (cf. le filtre `kind !== 'medal'` du composant) : son
 * seul dessin est désormais le badge, jamais un contour posé ici.
 *
 * TOUTES LES VARIANTES PARTAGENT LEUR CENTRE VERTICAL. Sur la piste du joueur regardé, y = 12 px
 * sur vingt-quatre (`top-2` + 8 px de haut pour la barre, `top-[9px]` + 6 pour le losange) ; sur
 * celle des coéquipiers, y = 7 px sur quatorze (`top-1` + 6, `top-[4px]` + 6). Sans cette égalité,
 * une marque amie flotterait un pixel plus haut que ses voisines et la piste se lirait comme deux
 * lignes. Le losange est plus court que la barre parce qu'il occupe sa DIAGONALE une fois tourné
 * (6 px de côté ≈ 8,5 px en travers) : à taille égale il dépasserait.
 *
 * LA PISTE DU JOUEUR REGARDÉ EST PASSÉE DE 18 À 24 px LE 2026-09-09 (décision D7) : c'est
 * désormais elle qui loge le badge de médaille (16 px) ET la porte de présence, sans les rogner.
 * Elle avait déjà quitté ses quatorze pixels d'origine le 2026-09-07 pour les deux premières. La
 * piste des coéquipiers, elle, N'A PAS BOUGÉ — elle ne porte ni présence ni médaille, et l'écart
 * de hauteur entre les deux rangées est justement ce qui dit laquelle est le sujet.
 *
 * L'ATTÉNUATION reste celle de la piste, pas de la forme : la piste des coéquipiers est plus
 * discrète que celle du joueur regardé, qu'on y soit ami ou non.
 */
function markShape(mark: TrackMark, tall: boolean): string {
  if (mark.friend) {
    return tall
      ? 'top-[9px] h-1.5 w-1.5 rotate-45 rounded-[2px]'
      : 'top-[4px] h-1.5 w-1.5 rotate-45 rounded-[2px] opacity-65'
  }
  return tall
    ? 'top-2 h-2 w-[3px] rounded-[2px]'
    : 'top-1 h-1.5 w-[2px] rounded-[2px] opacity-65'
}

/** L'ENCRE D'UNE MARQUE : les deux encres du rejeu, pour un kill et pour une mort. */
function markInk(kind: TrackMark['kind']): string {
  return tokenCssVar(kind === 'kill' ? 'team-ally' : 'team-enemy')
}

/**
 * L'INFOBULLE DE LA MARQUE NUE : l'instant, et les noms des médailles rattachées quand il y en a
 * — « 0:42 — Double frag, Vengeance ». Portée par un span `pointer-events-none` (cf. l'en-tête),
 * elle ne se déclenche pas au survol de la souris ; elle reste utile à l'accessible-name que les
 * lecteurs d'écran calculent depuis l'attribut `title`, indépendamment du survol. L'infobulle QUI
 * COMPTE, titre ET description (décision D7), est celle du badge lui-même (`MedalBadges`).
 */
function markTitle(mark: TrackMark): string {
  if (mark.medals.length === 0) return mark.clock
  const noms = mark.medals.map((m) => m.label || m.name).join(', ')
  return `${mark.clock} — ${noms}`
}
