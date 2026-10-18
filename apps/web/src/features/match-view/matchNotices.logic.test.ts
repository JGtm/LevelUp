/**
 * matchNotices.logic — QUELS messages, dans QUEL ordre, avec QUELS libellés, pour la zone de
 * messages de la vue match.
 */
import { describe, expect, it } from 'vitest'

import { MATCH_VIEW_TEXT } from './i18n'
import {
  analysisNoticeBody,
  analysisNoticeTitle,
  analysisSlotLabel,
  selectMatchNotices,
  translatePartialReason,
} from './matchNotices.logic'

const header = (over: Record<string, unknown> = {}) => over as never

describe('selectMatchNotices — sélection', () => {
  it('vue complète : aucun message', () => {
    expect(selectMatchNotices({ is_partial: false, partial_reasons: [], header: header() })).toEqual([])
    expect(selectMatchNotices({})).toEqual([])
  })

  it("lecture en échec : un seul message d'erreur, sans la liste des données absentes", () => {
    expect(
      selectMatchNotices({ is_partial: true, partial_reasons: ['events_empty', 'medals_unavailable'] }),
    ).toEqual([{ kind: 'unavailable' }])
  })

  it('données absentes : la liste des raisons', () => {
    expect(selectMatchNotices({ is_partial: true, partial_reasons: ['scoreboard_empty'] })).toEqual([
      { kind: 'partial', reasons: ['scoreboard_empty'] },
    ])
  })

  it("analyse : seules les causes connues, dans l'ordre servi", () => {
    expect(
      selectMatchNotices({
        header: header({ analysis_pending: true, analysis_pending_parts: ['replay', 'inconnue', 'kill_source'] }),
      }),
    ).toEqual([{ kind: 'analysis', inProgress: false, parts: ['replay', 'kill_source'] }])
  })

  it('analyse sans causes servies : le message existe, sans corps', () => {
    expect(selectMatchNotices({ header: header({ analysis_pending: true }) })).toEqual([
      { kind: 'analysis', inProgress: false, parts: [] },
    ])
  })

  it('« en cours » seulement pour analysis_state == in_progress', () => {
    const etat = (analysis_state?: string) =>
      selectMatchNotices({ header: header({ analysis_pending: true, analysis_state }) })[0]
    expect(etat('in_progress')).toMatchObject({ inProgress: true })
    for (const autre of ['queued', undefined, '', 'autre']) expect(etat(autre)).toMatchObject({ inProgress: false })
  })

  it('analysis_pending faux : aucun message, même avec un état résiduel', () => {
    expect(
      selectMatchNotices({ header: header({ analysis_pending: false, analysis_state: 'in_progress' }) }),
    ).toEqual([])
  })

  it('confidentialité : présente sauf niveau « none » ou absente', () => {
    const warning = { level: 'partial', message: 'Profil partiellement privé.' } as never
    expect(selectMatchNotices({ privacy_warning: warning })).toEqual([{ kind: 'privacy', warning }])
    expect(selectMatchNotices({ privacy_warning: { level: 'none', message: '' } as never })).toEqual([])
    expect(selectMatchNotices({ privacy_warning: null })).toEqual([])
  })
})

describe('selectMatchNotices — ordre', () => {
  it('erreur > données absentes > analyse > confidentialité', () => {
    const warning = { level: 'full', message: 'Profil privé.' } as never
    const kinds = (reasons: string[]) =>
      selectMatchNotices({
        is_partial: true,
        partial_reasons: reasons,
        header: header({ analysis_pending: true, analysis_pending_parts: ['citations'] }),
        privacy_warning: warning,
      }).map((n) => n.kind)

    expect(kinds(['scoreboard_unavailable'])).toEqual(['unavailable', 'analysis', 'privacy'])
    expect(kinds(['scoreboard_empty'])).toEqual(['partial', 'analysis', 'privacy'])
  })
})

describe("libellés de l'analyse", () => {
  it.each([
    ['fr', 'Analyse du film en cours', 'Analyse du film en attente', 'Analyse en cours', 'Analyse en attente'],
    ['en', 'Film analysis in progress', 'Film analysis pending', 'Analysis in progress', 'Analysis pending'],
  ] as const)("%s : titre et emplacement selon l'état", (loc, titreEnCours, titreAttente, slotEnCours, slotAttente) => {
    const t = MATCH_VIEW_TEXT[loc]
    expect(analysisNoticeTitle(true, t)).toBe(titreEnCours)
    expect(analysisNoticeTitle(false, t)).toBe(titreAttente)
    expect(analysisSlotLabel(true, t)).toBe(slotEnCours)
    expect(analysisSlotLabel(false, t)).toBe(slotAttente)
  })

  it('corps : noms courts des seules parties en attente', () => {
    const fr = MATCH_VIEW_TEXT.fr
    expect(analysisNoticeBody(['kill_source'], fr)).toBe('Frags assistés, arme de chaque frag')
    expect(analysisNoticeBody(['citations'], fr)).toBe('Citations')
    expect(analysisNoticeBody(['replay'], fr)).toBe('Rejeu 2D')
    expect(analysisNoticeBody(['kill_source', 'replay'], fr)).toBe('Frags assistés, arme de chaque frag · Rejeu 2D')
    expect(analysisNoticeBody([], fr)).toBe('')
    const en = MATCH_VIEW_TEXT.en
    expect(analysisNoticeBody(['kill_source', 'citations', 'replay'], en)).toBe(
      'Assisted frags, weapon per frag · Citations · 2D replay',
    )
  })

  it('aucun libellé de message ne promet un délai, ne dit « à venir » ni « réessayez plus tard »', () => {
    for (const loc of ['fr', 'en'] as const) {
      const t = MATCH_VIEW_TEXT[loc]
      const textes = [
        t.noticeUnavailable,
        t.noticeUnavailableRetry,
        t.noticePartialLead,
        t.analysisPendingTitleInProgress,
        t.analysisPendingTitleQueued,
        t.analysisPendingAutoUpdate,
        t.analysisPendingSlotInProgress,
        t.analysisPendingSlotQueued,
        ...Object.values(t.analysisPendingParts),
      ].join(' | ')
      expect(textes).not.toMatch(/à venir|coming|upcoming|d'ici|within|moments|quelques|plus tard|try again later|instants/i)
    }
  })
})

describe('translatePartialReason', () => {
  it('traduit les codes connus et rend un code inconnu tel quel', () => {
    expect(translatePartialReason('medals_empty', 'fr')).toBe('Les médailles et citations ne sont pas disponibles')
    expect(translatePartialReason('medals_empty', 'en')).toBe('Medals and commendations are unavailable')
    expect(translatePartialReason('autre_code', 'fr')).toBe('autre_code')
  })
})
