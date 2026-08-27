/**
 * Tests — ReplayMatchRecall (le rappel du match en tête de la page de rejeu).
 *
 * CE QU'ILS PROTÈGENT : ce rappel lit une vue qui peut être EN VOL. Une ligne qui
 * s'afficherait vide (ou avec sa ponctuation orpheline) mentirait sur ce qu'on sait du
 * match ; et une phrase composée ici plutôt que par le helper de la page match ferait
 * diverger les deux pages — c'est le même match, il se dit avec les mêmes mots.
 */
import { describe, expect, it } from 'vitest'
import { render } from '@testing-library/react'

import { ReplayMatchRecall } from './ReplayMatchRecall'

/** Le témoin : un CTF sur Forbidden, daté, joué en playlist classée. */
function renderRecall(over: Partial<Parameters<typeof ReplayMatchRecall>[0]> = {}) {
  return render(
    <ReplayMatchRecall
      mapUI="Forbidden"
      modeUI="CTF"
      startTimeLabel="12 août 2026, 21:14"
      playlistLabel="Arène classée"
      locale="fr"
      {...over}
    />,
  )
}

describe('ReplayMatchRecall', () => {
  it('dit le mode sur la carte, la date, et la playlist en badge', () => {
    const view = renderRecall()
    expect(view.getByText('CTF sur Forbidden')).toBeTruthy()
    expect(view.getByText('12 août 2026, 21:14')).toBeTruthy()
    expect(view.getByText('Arène classée')).toBeTruthy()
  })

  it('compose la phrase avec le helper de la page match, langue comprise', () => {
    const view = renderRecall({ locale: 'en' })
    expect(view.getByText('CTF on Forbidden')).toBeTruthy()
  })

  it("n'affiche RIEN tant que la vue du match n'est pas là", () => {
    const view = renderRecall({
      mapUI: undefined,
      modeUI: undefined,
      startTimeLabel: undefined,
      playlistLabel: undefined,
    })
    expect(view.container.firstChild).toBeNull()
  })

  it('sans playlist, aucun badge ne se pose', () => {
    const view = renderRecall({ playlistLabel: undefined })
    expect(view.queryByText('Arène classée')).toBeNull()
    expect(view.container.querySelector('span.rounded-full')).toBeNull()
    // Le reste de la ligne, lui, tient debout.
    expect(view.getByText('CTF sur Forbidden')).toBeTruthy()
  })

  it('sans date, la ponctuation ne reste pas orpheline', () => {
    const view = renderRecall({ startTimeLabel: undefined })
    expect(view.container.textContent).not.toContain('·')
    expect(view.getByText('CTF sur Forbidden')).toBeTruthy()
  })

  // L'orphelin SYMÉTRIQUE (revue R1) : des libellés de carte/mode vides existent en
  // production (la page match les teste) — la date ne doit pas arriver derrière un point
  // médian en tête de ligne.
  it("sans carte ni mode, la date s'affiche sans point médian en tête", () => {
    const view = renderRecall({ mapUI: undefined, modeUI: undefined })
    expect(view.getByText('12 août 2026, 21:14')).toBeTruthy()
    expect(view.container.textContent).not.toContain('·')
  })
})
