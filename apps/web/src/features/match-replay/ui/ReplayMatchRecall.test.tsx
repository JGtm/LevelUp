/**
 * Tests — ReplayMatchRecall (le complément du match, sur la ligne du fil d'Ariane).
 *
 * CE QU'ILS PROTÈGENT : ce rappel lit une vue qui peut être EN VOL. Une ligne qui
 * s'afficherait vide (ou avec sa ponctuation orpheline) mentirait sur ce qu'on sait du match.
 *
 * CE QU'ILS NE PROTÈGENT PLUS (2026-09-02) : la composition « mode sur carte ». Elle a quitté
 * ce composant, qui la recalculait avec `buildMatchHeadingStr` alors que la page passe DÉJÀ
 * cette même phrase au fil d'Ariane — le libellé s'imprimait deux fois. Le fil la garde, ce
 * composant ne porte plus que ce qu'il ajoutait vraiment : la date et la playlist.
 */
import { describe, expect, it } from 'vitest'
import { render } from '@testing-library/react'

import { ReplayMatchRecall } from './ReplayMatchRecall'

/** Le témoin : un match daté, joué en playlist classée. */
function renderRecall(over: Partial<Parameters<typeof ReplayMatchRecall>[0]> = {}) {
  return render(
    <ReplayMatchRecall
      startTimeLabel="12 août 2026, 21:14"
      playlistLabel="Arène classée"
      {...over}
    />,
  )
}

describe('ReplayMatchRecall', () => {
  it('dit la date, et la playlist en badge', () => {
    const view = renderRecall()
    expect(view.getByText('12 août 2026, 21:14')).toBeTruthy()
    expect(view.getByText('Arène classée')).toBeTruthy()
  })

  // LE DOUBLON SUPPRIMÉ, TENU PAR UN TEST : la phrase du fil d'Ariane ne doit pas réapparaître
  // ici. Sans cette assertion, un futur ajout « pour redonner du contexte » la réintroduirait.
  it('ne redit PAS le mode sur la carte — le fil d\'Ariane le porte', () => {
    const view = renderRecall()
    expect(view.container.textContent).not.toContain('sur')
  })

  it("n'affiche RIEN tant que la vue du match n'est pas là", () => {
    const view = renderRecall({ startTimeLabel: undefined, playlistLabel: undefined })
    expect(view.container.firstChild).toBeNull()
  })

  it('sans playlist, aucun badge ne se pose', () => {
    const view = renderRecall({ playlistLabel: undefined })
    expect(view.queryByText('Arène classée')).toBeNull()
    // Le reste de la ligne, lui, tient debout.
    expect(view.getByText('12 août 2026, 21:14')).toBeTruthy()
  })

  // LE POINT MÉDIAN SÉPARE DE LA FEUILLE DU FIL (« Rejeu 2D · 12 août… »), toujours présente
  // quand ce composant se rend. Il ne doit donc paraître QUE s'il a une date à introduire :
  // sans date, la playlist suit la feuille directement, sans ponctuation qui pende.
  it('sans date, la ponctuation ne reste pas orpheline', () => {
    const view = renderRecall({ startTimeLabel: undefined })
    expect(view.container.textContent).not.toContain('·')
    expect(view.getByText('Arène classée')).toBeTruthy()
  })

  it('avec une date, le point médian la sépare de ce qui précède', () => {
    const view = renderRecall()
    expect(view.container.textContent).toContain('·')
  })
})
