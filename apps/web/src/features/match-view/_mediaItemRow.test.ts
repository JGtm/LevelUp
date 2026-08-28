/**
 * Repli du `kind` dans `toMediaItemRow` (_mediaItemRow.ts, onglet Medias de la page match).
 *
 * `kind` est REQUIS au contrat servi (`MatchAssociatedMedia`) : le chemin nominal passe
 * toujours par `normalizeMediaKind`. Le repli sur la duree ne sert QUE les reponses en
 * cache navigateur d'avant le fix kind — d'ou les cas « kind absent » ci-dessous.
 *
 * Ce fichier est le garde-rail du correctif `!== null` -> `!= null` : `duration_seconds`
 * est `omitempty` au contrat, donc ABSENT (undefined) et jamais null sur une image. Le
 * test strict classait donc toute image sans kind en 'clip'.
 */
import { describe, it, expect } from 'vitest'

import type { MatchAssociatedMedia } from '@/lib/api/types'
import { toMediaItemRow } from './_mediaItemRow'

const MATCH_ID = 'match-1'

/** Duree relachee a `null` pour pouvoir couvrir aussi une reponse qui servirait null. */
type MediaOverrides = Partial<Omit<MatchAssociatedMedia, 'duration_seconds' | 'kind'>> & {
  duration_seconds?: number | null
  /** OMIS = reponse en cache d'avant le fix kind (le contrat, lui, le rend requis). */
  kind?: string
}

function media(overrides: MediaOverrides = {}): MatchAssociatedMedia {
  return {
    file_id: 'f1',
    file_name: 'capture.png',
    file_path: '/media/capture.png',
    liked: false,
    ...overrides,
  } as MatchAssociatedMedia
}

describe('toMediaItemRow — repli du kind', () => {
  it("kind absent + duree presente => 'clip'", () => {
    expect(toMediaItemRow(media({ duration_seconds: 42 }), MATCH_ID).kind).toBe('clip')
  })

  it("kind absent + duree ABSENTE => 'screenshot'", () => {
    // Cas cible du correctif : `undefined !== null` valait true, donc 'clip' a tort.
    expect(toMediaItemRow(media(), MATCH_ID).kind).toBe('screenshot')
  })

  it("kind absent + duree explicitement null => 'screenshot'", () => {
    // Pinne `!= null` plutot que `!== undefined` : les deux absences sont couvertes.
    expect(toMediaItemRow(media({ duration_seconds: null }), MATCH_ID).kind).toBe('screenshot')
  })

  it('kind present : normalizeMediaKind prime sur la duree', () => {
    expect(toMediaItemRow(media({ kind: 'video' }), MATCH_ID).kind).toBe('clip')
    // Une image reste une image meme si une duree trainait dans la reponse.
    expect(toMediaItemRow(media({ kind: 'image', duration_seconds: 42 }), MATCH_ID).kind).toBe(
      'screenshot',
    )
  })
})
