/**
 * Tests — replayMediaLogic (où un média tombe sur l'axe du rejeu).
 *
 * Ce qu'ils protègent, dans l'ordre des règles produit :
 *  1. `capture_time` EST UNE FIN. Un clip posé dessus apparaîtrait sa propre durée trop tard —
 *     c'est le décalage que le lot 1 est allé chercher en base pour pouvoir le corriger ici.
 *  2. LE DÉBUT PUBLIÉ PRIME sur toute reconstitution : quand la base connaît le début, on ne
 *     retranche rien.
 *  3. UNE CAPTURE EST UN INSTANT, sans durée, et n'est jamais reculée.
 *  4. ON N'INVENTE AUCUNE POSE. Sans horodatage, le média sort de la frise ; sans heure de
 *     début de match, la piste entière est vide.
 *  5. L'IDENTITÉ EST `file_id`. `file_path` est muté par la conversion et le transcodage HLS —
 *     un identifiant qui bouge d'un rendu à l'autre casserait tout rapprochement.
 */
import { describe, expect, it } from 'vitest'

import type { MatchMediaTab } from '@/lib/api/types'

import { replayClock } from './replayClock'
import { buildReplayMedia, type ReplayMediaHeader } from './replayMediaLogic'
import { testReplayDoc } from '../test/testDoc'

type MediaTabItem = NonNullable<MatchMediaTab['media_items']>[number]

/** Le match commence à 12:00:00 UTC ; l'image zéro du film tombe 10 s plus tard. */
const MATCH_START = '2026-08-28T12:00:00Z'
const ORIGIN_MS = 10_000
const HEADER: ReplayMediaHeader = { start_time: MATCH_START }

/**
 * L'horloge de la page, etablie sur l'origine ci-dessus : c'est elle qui porte desormais la
 * soustraction, et c'est par elle que la piste se tait quand l'artefact n'a pas d'origine.
 */
const CLOCK = replayClock(
  testReplayDoc({ originMs: ORIGIN_MS, frameIntervalMs: 100, frameCount: 6_000 }),
)

function item(over: Partial<MediaTabItem> = {}): MediaTabItem {
  return {
    file_id: '42',
    file_name: 'clip.mp4',
    file_path: '/api/v1/players/Hero/media/files/clip.mp4',
    kind: 'video',
    liked: false,
    ...over,
  }
}

function tab(...items: MediaTabItem[]): MatchMediaTab {
  return { media_items: items }
}

describe('buildReplayMedia — le recalage temporel', () => {
  it("UN CLIP SANS DÉBUT SE RECULE DE SA DURÉE : capture_time est la FIN de la capture", () => {
    // Fin de capture à 12:01:00 (60 s de match), clip de 30 s → il a commencé à 30 s de match.
    const [media] = buildReplayMedia(
      tab(item({ capture_time: '2026-08-28T12:01:00Z', duration_seconds: 30 })),
      HEADER,
      CLOCK,
    )
    expect(media.replayMs).toBe(30_000 - ORIGIN_MS)
    expect(media.durationMs).toBe(30_000)
  })

  it('LE DÉBUT PUBLIÉ PRIME : rien n’est retranché quand la base le connaît', () => {
    const [media] = buildReplayMedia(
      tab(
        item({
          capture_start_time: '2026-08-28T12:00:50Z',
          capture_time: '2026-08-28T12:01:00Z',
          duration_seconds: 10,
        }),
      ),
      HEADER,
      CLOCK,
    )
    expect(media.replayMs).toBe(50_000 - ORIGIN_MS)
  })

  it("UNE CAPTURE EST UN INSTANT : pas de durée, et aucun recul", () => {
    const [media] = buildReplayMedia(
      tab(item({ kind: 'image', file_name: 'shot.png', capture_time: '2026-08-28T12:00:20Z' })),
      HEADER,
      CLOCK,
    )
    expect(media.kind).toBe('image')
    expect(media.replayMs).toBe(20_000 - ORIGIN_MS)
    expect(media.durationMs).toBeUndefined()
  })

  it("UNE DURÉE ABSENTE POSE LE CLIP SUR SA FIN plutôt que de le retirer de la frise", () => {
    const [media] = buildReplayMedia(
      tab(item({ capture_time: '2026-08-28T12:00:40Z' })),
      HEADER,
      CLOCK,
    )
    expect(media.replayMs).toBe(40_000 - ORIGIN_MS)
    expect(media.durationMs).toBeUndefined()
  })

  it('LES MÉDIAS SORTENT DANS L’ORDRE DE L’AXE, quel que soit celui de l’API', () => {
    const media = buildReplayMedia(
      tab(
        item({ file_id: '2', capture_start_time: '2026-08-28T12:01:00Z' }),
        item({ file_id: '1', capture_start_time: '2026-08-28T12:00:30Z' }),
      ),
      HEADER,
      CLOCK,
    )
    expect(media.map((m) => m.id)).toEqual(['1', '2'])
  })
})

describe('buildReplayMedia — ce qu’on refuse d’inventer', () => {
  it('UN MÉDIA SANS AUCUN HORODATAGE EST ÉCARTÉ (jamais une pose au hasard)', () => {
    const media = buildReplayMedia(
      tab(item({ file_id: '1' }), item({ file_id: '2', capture_time: '2026-08-28T12:00:30Z' })),
      HEADER,
      CLOCK,
    )
    expect(media.map((m) => m.id)).toEqual(['2'])
  })

  it("UN HORODATAGE ILLISIBLE VAUT UNE ABSENCE, pas un NaN posé sur la frise", () => {
    expect(buildReplayMedia(tab(item({ capture_time: 'pas une date' })), HEADER, CLOCK))
      .toEqual([])
  })

  it("SANS HEURE DE DÉBUT DE MATCH, aucun média n'est plaçable", () => {
    expect(
      buildReplayMedia(tab(item({ capture_start_time: MATCH_START })), { start_time: undefined }, CLOCK),
    ).toEqual([])
  })

  it("SANS HORLOGE ÉTABLIE, LA PISTE SE TAIT — jamais une pose à l'origine zéro", () => {
    // L'artefact qui ne publie pas d'origine ne dit pas « zéro », il ne dit RIEN : poser les
    // captures à zéro les éloignait du fil des éliminations, lui recalé sur un décalage
    // MESURÉ, de l'écart réel entre les deux (3,6 à 50,8 s selon le match).
    const sansOrigine = replayClock(testReplayDoc({ frameIntervalMs: 100, frameCount: 6_000 }))
    expect(sansOrigine).toBeNull()
    expect(
      buildReplayMedia(tab(item({ capture_start_time: MATCH_START })), HEADER, sansOrigine),
    ).toEqual([])
  })

  it('UN ONGLET VIDE OU ABSENT NE FAIT PAS TOMBER LE MAPPEUR', () => {
    expect(buildReplayMedia(tab(), HEADER, CLOCK)).toEqual([])
    expect(buildReplayMedia(null, HEADER, CLOCK)).toEqual([])
    expect(buildReplayMedia({ media_items: null }, HEADER, CLOCK)).toEqual([])
  })

  it('UNE DURÉE NULLE OU NÉGATIVE NE DEVIENT PAS UNE LARGEUR', () => {
    const [media] = buildReplayMedia(
      tab(item({ capture_start_time: '2026-08-28T12:00:30Z', duration_seconds: 0 })),
      HEADER,
      CLOCK,
    )
    expect(media.durationMs).toBeUndefined()
    expect(media.replayMs).toBe(30_000 - ORIGIN_MS)
  })
})

describe('buildReplayMedia — identité et rendu', () => {
  it("L'IDENTITÉ EST file_id, JAMAIS file_path (chemin muté par conversion et HLS)", () => {
    const [media] = buildReplayMedia(
      tab(
        item({
          file_id: '77',
          file_path: '/api/v1/players/Hero/media/files/hls/clip/master.m3u8',
          capture_start_time: '2026-08-28T12:00:30Z',
        }),
      ),
      HEADER,
      CLOCK,
    )
    expect(media.id).toBe('77')
    expect(media.url).toBe('/api/v1/players/Hero/media/files/hls/clip/master.m3u8')
  })

  it('UNE IMAGE SANS VIGNETTE SE SERT D’ELLE-MÊME ; un clip n’a rien d’immobile à montrer', () => {
    const [image] = buildReplayMedia(
      tab(item({ kind: 'image', file_path: '/shot.png', capture_time: '2026-08-28T12:00:30Z' })),
      HEADER,
      CLOCK,
    )
    expect(image.thumbUrl).toBe('/shot.png')

    const [clip] = buildReplayMedia(
      tab(item({ capture_time: '2026-08-28T12:00:30Z' })),
      HEADER,
      CLOCK,
    )
    expect(clip.thumbUrl).toBe('')
  })

  it('LA VIGNETTE PUBLIÉE EST REPRISE TELLE QUELLE (WebP animé du pipeline)', () => {
    const [media] = buildReplayMedia(
      tab(
        item({
          thumbnail_url: '/api/v1/players/Hero/media/files/thumbs/clip.webp',
          capture_start_time: '2026-08-28T12:00:30Z',
        }),
      ),
      HEADER,
      CLOCK,
    )
    expect(media.thumbUrl).toBe('/api/v1/players/Hero/media/files/thumbs/clip.webp')
    expect(media.label).toBe('clip.mp4')
  })

  it("UN KIND INCONNU SE MONTRE EN IMAGE plutôt que de disparaître", () => {
    const [media] = buildReplayMedia(
      tab(item({ kind: 'autre', capture_time: '2026-08-28T12:00:30Z' })),
      HEADER,
      CLOCK,
    )
    expect(media.kind).toBe('image')
  })
})
