/**
 * Tests — ReplayLeadMarks (les retournements posés sur la frise).
 *
 * CE QU'ILS PROTÈGENT : une marque désigne un INSTANT, et sa couleur désigne QUI passe
 * devant. Se tromper sur l'un ou sur l'autre transforme un repère en contresens ; et une
 * marque qui capterait le pointeur rendrait la frise inutilisable là où elle se pose.
 */
import { describe, expect, it } from 'vitest'
import { render } from '@testing-library/react'

import { ReplayLeadMarks } from './ReplayLeadMarks'

const CHANGES = [
  { frame: 653, teamId: 0 },
  { frame: 1807, teamId: 1 },
  { frame: 2345, teamId: 0 },
]

/** Le témoin Oddball : 5 137 images à 100 ms, trois retournements. */
function renderMarks(over: Partial<Parameters<typeof ReplayLeadMarks>[0]> = {}) {
  return render(
    <ReplayLeadMarks
      changes={CHANGES}
      frameCount={5137}
      frameIntervalMs={100}
      allyOf={(teamId) => teamId === 0}
      labelOf={(teamId) => `Équipe ${teamId}`}
      locale="fr"
      {...over}
    />,
  )
}

describe('ReplayLeadMarks', () => {
  it('pose une marque par retournement, et pas une de plus', () => {
    const view = renderMarks()
    expect(view.getAllByLabelText('Retournement')).toHaveLength(3)
  })

  it('date chaque marque en mm:ss dans son infobulle, avec le camp qui passe devant', () => {
    const view = renderMarks()
    // 1807 images x 100 ms = 180,7 s = 3:01.
    expect(view.getByTitle('Retournement à 3:01 — Équipe 1 passe devant')).toBeTruthy()
    expect(view.getByTitle('Retournement à 1:05 — Équipe 0 passe devant')).toBeTruthy()
  })

  it('place la marque à la fraction du temps écoulé, pas à un pixel arbitraire', () => {
    const view = renderMarks()
    const first = view.getAllByLabelText('Retournement')[0] as HTMLElement
    // 653 / 5136 = 0,1272 — la marge de curseur est portée par le calc(), pas par le ratio.
    expect(first.style.left).toContain('0.127')
  })

  it('prend le token du NOUVEAU meneur : allié d’un côté, adverse de l’autre', () => {
    const view = renderMarks()
    const marks = view.getAllByLabelText('Retournement') as HTMLElement[]
    expect(marks[0].style.background).toContain('var(--ac-team-ally)')
    expect(marks[1].style.background).toContain('var(--ac-team-enemy)')
  })

  it('camp inconnu : encre NEUTRE — jamais l’une des deux couleurs par défaut', () => {
    const view = renderMarks({ allyOf: () => null })
    const marks = view.getAllByLabelText('Retournement') as HTMLElement[]
    expect(marks[0].style.background).toContain('var(--border)')
    expect(marks[0].style.background).not.toContain('team-')
  })

  it('ne capte JAMAIS le pointeur : la frise reste saisissable sous une marque', () => {
    const view = renderMarks()
    const calque = view.getAllByLabelText('Retournement')[0].parentElement as HTMLElement
    expect(calque.className).toContain('pointer-events-none')
  })

  it('sans retournement, ne rend rien du tout (témoins Slayer et CTF)', () => {
    const view = renderMarks({ changes: [] })
    expect(view.container.firstChild).toBeNull()
  })

  it('sans échelle temporelle, date par le NUMÉRO d’image plutôt qu’une durée fabriquée', () => {
    const view = renderMarks({ frameIntervalMs: undefined })
    expect(view.getByTitle('Retournement à #653 — Équipe 0 passe devant')).toBeTruthy()
  })

  it('document d’une seule image : aucune échelle, donc aucune marque', () => {
    const view = renderMarks({ frameCount: 1 })
    expect(view.container.firstChild).toBeNull()
  })

  it('EN : l’infobulle et le libellé accessible passent en anglais', () => {
    const view = renderMarks({ locale: 'en', labelOf: (id) => `Team ${id}` })
    expect(view.getAllByLabelText('Lead change')).toHaveLength(3)
    expect(view.getByTitle('Lead change at 3:01 — Team 1 moves ahead')).toBeTruthy()
  })
})
