/**
 * PresenceGlyph — LA PORTE FRANCHIE PAR UNE FLÈCHE : le repère d'entrée ou de sortie de partie.
 *
 * EXTRAIT DE `ReplayPresenceLine.tsx` LE 2026-09-07 (lot L4). Il y vivait comme fonction privée,
 * et la frise en a désormais besoin elle aussi — posé à la frontière entre la zone ombrée et la
 * zone jouée, où il sert de bouton (décision 2 bis du plan « frise, point de vue »). Recopier le
 * SVG aurait donné deux dessins à faire diverger : le montant de porte aurait changé de côté
 * dans l'un sans que rien ne rougisse dans l'autre. Le dessin n'existe qu'ICI, ses deux lecteurs
 * l'appellent (règle n° 6 du dépôt ; précédents d'extraction : `useTeamCascades`,
 * `useSlotIdentity`).
 *
 * LE SENS EST DANS LE DESSIN, PAS DANS UNE COULEUR : montant de porte à gauche et flèche qui
 * entre pour une arrivée, montant à droite et flèche qui sort pour un départ. Il reste VECTORIEL
 * et à l'encre courante (même arbitrage que le crâne et la bombe : aucune vignette d'atlas dont
 * l'index bouge par saison), teinté par l'équipe de l'acteur quand elle est connue — un bot sans
 * camp joint garde l'encre du repli, jamais un camp deviné.
 *
 * LA TAILLE EST UNE PROP, pas un second dessin : le fil lui donne 14 × 12 (sa taille d'origine,
 * et le défaut ci-dessous), la frise la réduit pour tenir dans une piste de dix-huit pixels. Le
 * `viewBox` ne bouge pas — c'est ce qui garantit que les deux rendus sont le MÊME glyphe à deux
 * échelles.
 *
 * CE FICHIER N'EXPORTE QUE LE COMPOSANT, et ce n'est pas un hasard — ses deux dimensions de
 * référence sont PRIVÉES depuis le 2026-09-07 (revue F6), personne ne les lisait du dehors.
 * `presenceWording` — la règle
 * « l'API affirme, le film reste au fait » — y a vécu quelques heures le 2026-09-07 avant de
 * partir dans `model/presenceWording.ts` : un module qui exporte un composant ET autre chose
 * casse le rafraîchissement à chaud de Vite, et la dette lint du dépôt est gelée. Même remède que
 * le lot L1 avec `replayTimelineGrid.ts`. Ne rien rapatrier ici.
 */

import type { PresenceEvent } from '../model/presenceFeed'

/**
 * Les dimensions du glyphe dans le FIL — la taille d'origine, et le défaut de ce composant.
 *
 * PRIVÉES DEPUIS LE 2026-09-07 (revue F6) : elles étaient exportées, et aucun autre fichier ne
 * les importait — l'en-tête pouvait donc dire « ce fichier n'exporte que le composant » tout en
 * étant faux. Rendues privées plutôt que la phrase corrigée : la frise passe sa taille en props,
 * elle n'a jamais eu besoin de lire celle du fil (règle n° 7, on ne garde rien « au cas où »).
 */
const PRESENCE_GLYPH_W = 14
const PRESENCE_GLYPH_H = 12

export function PresenceGlyph({
  kind,
  color,
  width = PRESENCE_GLYPH_W,
  height = PRESENCE_GLYPH_H,
}: {
  kind: PresenceEvent['kind']
  /** Encre du glyphe : celle de l'équipe quand elle est connue, celle du repli sinon. */
  color: string
  width?: number
  height?: number
}) {
  const entering = kind === 'joined'
  return (
    <svg
      viewBox={`0 0 ${PRESENCE_GLYPH_W} ${PRESENCE_GLYPH_H}`}
      width={width}
      height={height}
      aria-hidden
      className="shrink-0"
      style={{ color }}
    >
      {/* Le montant de la porte : côté gauche pour entrer, côté droit pour sortir. */}
      <path
        d={entering ? 'M1.5 1v10' : 'M12.5 1v10'}
        stroke="currentColor"
        strokeWidth="1.6"
        strokeLinecap="round"
      />
      <path
        d={entering ? 'M4 6h7M8.4 3.4 11 6l-2.6 2.6' : 'M3 6h7M7.4 3.4 10 6l-2.6 2.6'}
        stroke="currentColor"
        strokeWidth="1.6"
        strokeLinecap="round"
        strokeLinejoin="round"
        fill="none"
      />
    </svg>
  )
}
