/**
 * FormesRetenuesSection.test.tsx — LA SECTION ENTIÈRE.
 *
 * CE TEST EXISTE POUR UNE RAISON PRÉCISE : le lot a été demandé parce que ce qui
 * avait été livré ne portait PAS les cartes de l'artefact. Le test vérifie donc
 * la LISTE : les dix-neuf titres, les trois blocs, les deux contextes répétés,
 * et les textes qui n'ont pas le droit de disparaître (constats, lexiques,
 * réserve des occupations sans nom).
 */
import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { FormesRetenuesSection } from './FormesRetenuesSection'
import { FORMES_CARDS_TEXT, type FormesCardKey } from './cardsI18n'
import { formesFixture } from './formes.fixtures'
import { FORMES_TEXT } from './i18n'

const fr = FORMES_TEXT.fr
const frCards = FORMES_CARDS_TEXT.fr

const ALL_CARDS = Object.keys(frCards.cards) as FormesCardKey[]

describe('FormesRetenuesSection', () => {
  it('rend les DIX-NEUF cartes de l’artefact', () => {
    render(<FormesRetenuesSection block={formesFixture()} locale="fr" />)
    expect(ALL_CARDS).toHaveLength(19)
    for (const key of ALL_CARDS) {
      const title = frCards.cards[key].title
      // Deux cartes portent le même titre dans deux contextes (« Écart à la
      // parité, par famille d'arme ») : on exige AU MOINS une occurrence.
      expect(screen.getAllByText(title).length).toBeGreaterThan(0)
    }
  })

  it('rend les trois blocs et les deux contextes de chacun', () => {
    render(<FormesRetenuesSection block={formesFixture()} locale="fr" />)
    expect(screen.getByText(fr.blocks.equipment.title)).toBeInTheDocument()
    expect(screen.getByText(fr.blocks.weapons.title)).toBeInTheDocument()
    expect(screen.getByText(fr.blocks.objectives.title)).toBeInTheDocument()
    // Trois blocs, deux contextes chacun.
    expect(screen.getAllByText(fr.contexts.solo)).toHaveLength(3)
    expect(screen.getAllByText(fr.contexts.squad)).toHaveLength(3)
  })

  it('rend le lexique de chaque bloc et la réserve des occupations sans nom', () => {
    const { container } = render(<FormesRetenuesSection block={formesFixture()} locale="fr" />)
    const text = container.textContent ?? ''
    expect(text).toContain('Six familles')
    expect(text).toContain('Un socle')
    expect(text).toContain('La table rôle')
    // La réserve : 5 occupations sans ramasseur, 9 prises nommées sur 14.
    expect(text).toContain('5 occupations de socle')
    expect(text).toContain('9 prises sur 14')
  })

  it('nomme les cinq gestes d’équipement, et JAMAIS les grenades', () => {
    const { container } = render(<FormesRetenuesSection block={formesFixture()} locale="fr" />)
    const text = container.textContent ?? ''
    for (const label of Object.values(fr.axes)) expect(text).toContain(label)
    expect(text).not.toContain('Grenades lancées')
  })

  it('affiche la couverture « mesurés sur total » du bandeau', () => {
    render(<FormesRetenuesSection block={formesFixture()} locale="fr" />)
    expect(screen.getByText(fr.header.scopeMeasuredFmt(2, 3))).toBeInTheDocument()
  })

  it('se retire quand le bloc est absent ou indisponible', () => {
    const { container: empty } = render(<FormesRetenuesSection block={undefined} locale="fr" />)
    expect(empty).toBeEmptyDOMElement()
    const { container: down } = render(
      <FormesRetenuesSection
        block={{ available: false, matches_total: 3, matches_measured: 0 }}
        locale="fr"
      />,
    )
    expect(down).toBeEmptyDOMElement()
  })

  it('rend aussi en anglais, sans clé manquante', () => {
    render(<FormesRetenuesSection block={formesFixture()} locale="en" />)
    expect(screen.getByText(FORMES_TEXT.en.blocks.weapons.title)).toBeInTheDocument()
    expect(
      screen.getAllByText(FORMES_CARDS_TEXT.en.cards.equipmentShares.title).length,
    ).toBeGreaterThan(0)
  })

  it('retire les cartes d’objectif quand aucun match n’en porte', () => {
    const block = formesFixture()
    block.matches = (block.matches ?? []).map((m) => ({ ...m, objective: undefined }))
    render(<FormesRetenuesSection block={block} locale="fr" />)
    // Le bloc reste, avec son constat d'absence — les cartes, elles, partent.
    expect(screen.getByText(fr.blocks.objectives.title)).toBeInTheDocument()
    expect(screen.queryByText(frCards.cards.objectivesGapRole.title)).not.toBeInTheDocument()
    expect(screen.getAllByText(fr.contexts.solo)).toHaveLength(2)
  })
})
