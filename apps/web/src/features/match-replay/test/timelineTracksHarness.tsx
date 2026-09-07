/**
 * timelineTracksHarness.tsx — LE MONTAGE PARTAGÉ DE LA FRISE, pour ses fichiers de tests.
 *
 * # POURQUOI CE MODULE EXISTE (2026-09-07, lot L4)
 *
 * `ReplayTimelineTracks` prend vingt-cinq props. Les monter à la main dans chaque cas noierait
 * ce qu'un test vérifie sous ce qu'il doit seulement fournir, et l'ajout d'une prop obligerait à
 * repasser dans tous les fichiers. Le montage vit donc à UN endroit, et chaque cas ne nomme que
 * ce qui l'intéresse.
 *
 * Il est sorti de `ui/ReplayTimelineTracks.test.tsx` le jour où l'ombrage de présence lui a donné
 * un SECOND lecteur (`ui/ReplayTimelineTracks.presence.test.tsx`) — même règle que `fakeAudio.ts`
 * et `testDoc.ts` : un double partagé rejoint `test/` dès sa deuxième copie, avant de diverger.
 * Sans cela, la copie qui n'aurait pas reçu la prop nouvelle serait restée verte en testant une
 * frise que plus personne ne rend.
 *
 * # CE QU'IL REND, ET POURQUOI CE N'EST PAS QUE `container`
 *
 * Les cinq espions de rappel sont rendus avec le résultat de `render`. C'est ce qui permet à un
 * test de vérifier une NON-action — qu'un clic sur une porte de présence ne met pas en pause et
 * ne change pas de point de vue (décision 1 du plan « frise, point de vue » : le curseur
 * appartient à l'utilisateur). Un espion créé dans le test ne pourrait pas le faire sans que
 * l'appelant repasse toutes les props à la main.
 *
 * # CE QU'IL NE FAIT PAS
 *
 * Aucune assertion, aucun `beforeEach`, aucun état entre deux montages : ce module FOURNIT, il ne
 * juge pas. Les valeurs par défaut sont toutes VIDES (pas de marque, pas d'ombre, pas de média) —
 * un défaut peuplé ferait passer des cas pour de mauvaises raisons.
 */
import { createRef } from 'react'
import { render } from '@testing-library/react'
import { vi } from 'vitest'

import { ReplayTimelineTracks } from '../ui/ReplayTimelineTracks'
import type { TrackMark } from '../model/replayTimelineTracksLogic'

/** Une marque de piste, au frag par défaut — les cas ne surchargent que ce qu'ils regardent. */
export function mark(over: Partial<TrackMark> = {}): TrackMark {
  return { key: 'm1', ratio: 0.5, kind: 'kill', clock: '2:30', medals: [], friend: false, ...over }
}

/**
 * LE MENU DE POINT DE VUE : deux camps nommés, plus le groupe des joueurs sans ligne de tableau
 * de score — dont l'option est INERTE (décision 7 bis). `bot-base` porte le xuid de la BASE, pas
 * la clé film : c'est le piège que le lot devait éviter (cf. `viewpointOptions`).
 */
export const GROUPES = [
  {
    key: 't0',
    label: 'Cobalt',
    options: [
      { value: 'me-1', label: 'JGtm', disabled: false, title: 'JGtm' },
      { value: 'bot-base', label: 'Cortana', disabled: false, title: 'Cortana' },
    ],
  },
  { key: 't1', label: 'Ambre', options: [{ value: 'foe-1', label: 'Rival', disabled: false, title: 'Rival' }] },
  {
    key: '',
    label: 'Sans équipe',
    options: [
      { value: 'bot:Fantome', label: 'Fantome', disabled: true, title: 'Aucune donnée de match pour ce joueur' },
    ],
  },
]

export function renderTracks(over: Partial<Parameters<typeof ReplayTimelineTracks>[0]> = {}) {
  const onRequestPause = vi.fn()
  const onScrub = vi.fn()
  const onToggleTracks = vi.fn()
  const onSelectViewpoint = vi.fn()
  const onSeekFrame = vi.fn()
  const utils = render(
    <ReplayTimelineTracks
      sliderRef={createRef<HTMLInputElement>()}
      minFrame={0}
      maxFrame={600}
      onScrub={onScrub}
      own={[]}
      teammates={[]}
      shades={[]}
      absence={[]}
      identity={new Map()}
      onSeekFrame={onSeekFrame}
      viewpoint="me-1"
      viewpointGroups={GROUPES}
      onSelectViewpoint={onSelectViewpoint}
      dominance={[]}
      score={null}
      allyOf={() => null}
      labelOf={(id) => `Équipe ${id}`}
      media={[]}
      showMediaTrack
      tracksExpanded
      onToggleTracks={onToggleTracks}
      playing
      onRequestPause={onRequestPause}
      clockRef={createRef<HTMLSpanElement>()}
      locale="fr"
      {...over}
    />,
  )
  return { ...utils, onRequestPause, onScrub, onToggleTracks, onSelectViewpoint, onSeekFrame }
}
