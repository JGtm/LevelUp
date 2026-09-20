/**
 * FormesRetenuesSection.test.tsx — LA SECTION ENTIÈRE, DANS SES DEUX CONTEXTES.
 *
 * CE TEST EXISTE POUR UNE RAISON PRÉCISE : le lot a été demandé parce que ce qui
 * avait été livré ne portait PAS les cartes de l'artefact. Le test vérifie donc la LISTE —
 * les dix-neuf titres, répartis entre le contexte SOLO (neuf cartes, page Timeseries) et
 * le contexte ESCOUADE (dix cartes, page Escouade) depuis le 2026-09-19 — les trois blocs,
 * et les textes qui n'ont pas le droit de disparaître (réserve des occupations sans nom).
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
  it('les DIX-NEUF cartes de l’artefact se répartissent entre les deux contextes', () => {
    expect(ALL_CARDS).toHaveLength(19)
    const { container: solo } = render(
      <FormesRetenuesSection block={formesFixture()} locale="fr" contexte="solo" />,
    )
    const { container: squad } = render(
      <FormesRetenuesSection block={formesFixture()} locale="fr" contexte="squad" />,
    )
    const textes = `${solo.textContent ?? ''}${squad.textContent ?? ''}`
    for (const key of ALL_CARDS) {
      expect(textes).toContain(frCards.cards[key].title)
    }
  })

  it('le contexte SOLO ne monte AUCUNE carte du contexte escouade, et réciproquement', () => {
    const { container: solo } = render(
      <FormesRetenuesSection block={formesFixture()} locale="fr" contexte="solo" />,
    )
    const texteSolo = solo.textContent ?? ''
    expect(texteSolo).toContain(frCards.cards.equipmentShares.title)
    expect(texteSolo).not.toContain(frCards.cards.equipmentSquadGrid.title)
    expect(texteSolo).not.toContain(frCards.cards.padsTwoFriezes.title)

    const { container: squad } = render(
      <FormesRetenuesSection block={formesFixture()} locale="fr" contexte="squad" />,
    )
    const texteSquad = squad.textContent ?? ''
    expect(texteSquad).toContain(frCards.cards.equipmentSquadGrid.title)
    expect(texteSquad).not.toContain(frCards.cards.equipmentShares.title)
  })

  it('rend les trois blocs, chacun avec son titre — plus aucun intertitre de contexte', () => {
    const { container } = render(
      <FormesRetenuesSection block={formesFixture()} locale="fr" contexte="solo" />,
    )
    expect(screen.getByText(fr.blocks.equipment.title)).toBeInTheDocument()
    expect(screen.getByText(fr.blocks.weapons.title)).toBeInTheDocument()
    expect(screen.getByText(fr.blocks.objectives.title)).toBeInTheDocument()
    const texte = container.textContent ?? ''
    expect(texte).not.toContain('Contexte Solo')
    expect(texte).not.toContain('Contexte Escouade')
  })

  it('l’aide de chaque bloc reste CONCISE (trois phrases au plus) et n’est plus un pavé à l’écran', () => {
    const { container } = render(
      <FormesRetenuesSection block={formesFixture()} locale="fr" contexte="solo" />,
    )
    const texte = container.textContent ?? ''
    // Les deux pavés (constat, lexique) ont quitté l'écran : l'aide vit dans l'infobulle.
    expect(texte).not.toContain('Six familles')
    expect(texte).not.toContain('Ce que le décodeur prend en charge')
    for (const bloc of [fr.blocks.equipment, fr.blocks.weapons, fr.blocks.objectives]) {
      const phrases = bloc.aide.split('.').filter((p) => p.trim().length > 0)
      expect(phrases.length).toBeLessThanOrEqual(3)
    }
  })

  it('garde la réserve des occupations sans ramasseur nommé (contexte escouade)', () => {
    const { container } = render(
      <FormesRetenuesSection block={formesFixture()} locale="fr" contexte="squad" />,
    )
    const text = container.textContent ?? ''
    // La réserve : 5 occupations sans ramasseur, 9 prises nommées sur 14.
    expect(text).toContain('5 occupations de socle')
    expect(text).toContain('9 prises sur 14')
  })

  it('nomme les cinq gestes d’équipement, et JAMAIS les grenades', () => {
    const { container } = render(
      <FormesRetenuesSection block={formesFixture()} locale="fr" contexte="solo" />,
    )
    const text = container.textContent ?? ''
    for (const label of Object.values(fr.axes)) expect(text).toContain(label)
    expect(text).not.toContain('Grenades lancées')
  })

  it('affiche la couverture « mesurés sur total » du bandeau', () => {
    render(<FormesRetenuesSection block={formesFixture()} locale="fr" contexte="solo" />)
    expect(screen.getByText(fr.header.scopeMeasuredFmt(2, 3))).toBeInTheDocument()
  })

  it('se retire quand le bloc est absent ou indisponible', () => {
    const { container: empty } = render(
      <FormesRetenuesSection block={undefined} locale="fr" contexte="solo" />,
    )
    expect(empty).toBeEmptyDOMElement()
    const { container: down } = render(
      <FormesRetenuesSection
        block={{ available: false, matches_total: 3, matches_measured: 0 }}
        locale="fr"
        contexte="solo"
      />,
    )
    expect(down).toBeEmptyDOMElement()
  })

  // DÉFAUT MESURÉ LE 2026-09-13 : sur une portée de 1 147 matchs, les formes par
  // match rendaient une ligne par match — quinze écrans, presque tous hachurés.
  it('replie les formes par match au-delà de vingt, et le dit', () => {
    const block = formesFixture()
    const base = (block.matches ?? [])[1]
    // 60 matchs mesurés + les 3 de la fixture.
    block.matches = [
      ...Array.from({ length: 60 }, (_, i) => ({
        ...base,
        match_id: `bulk-${i}`,
        start_time: new Date(Date.UTC(2026, 6, 1) - i * 3600_000).toISOString(),
      })),
      ...(block.matches ?? []),
    ]
    block.matches_total = block.matches.length
    block.matches_measured = block.matches.filter((m) => m.measured).length
    render(<FormesRetenuesSection block={block} locale="fr" contexte="squad" />)
    // Une ligne par match mesuré affiché, jamais une par match du scope.
    const grip = screen.getByLabelText(frCards.cards.padsSquadByMatch.title)
    expect(grip.querySelectorAll('[role="img"][aria-label*="Prises de socle"]').length).toBe(0)
    expect(grip.textContent).toContain('Affichés : les 20 derniers matchs à film décodé')
    expect(grip.textContent).toContain('matchs sans film décodé sont hors de cette forme')
    // Le match sans film n'a plus de ligne du tout.
    expect(grip.textContent).not.toContain(fr.common.noFilm)
  })

  it('rend aussi en anglais, sans clé manquante', () => {
    render(<FormesRetenuesSection block={formesFixture()} locale="en" contexte="solo" />)
    expect(screen.getByText(FORMES_TEXT.en.blocks.weapons.title)).toBeInTheDocument()
    expect(
      screen.getAllByText(FORMES_CARDS_TEXT.en.cards.equipmentShares.title).length,
    ).toBeGreaterThan(0)
  })

  it('retire les cartes d’objectif quand aucun match n’en porte', () => {
    const block = formesFixture()
    block.matches = (block.matches ?? []).map((m) => ({ ...m, objective: undefined }))
    render(<FormesRetenuesSection block={block} locale="fr" contexte="solo" />)
    // Le bloc garde son titre et son aide — les cartes, elles, partent.
    expect(screen.getByText(fr.blocks.objectives.title)).toBeInTheDocument()
    expect(screen.queryByText(frCards.cards.objectivesGapRole.title)).not.toBeInTheDocument()
  })
})
