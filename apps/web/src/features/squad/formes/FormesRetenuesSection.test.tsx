/**
 * FormesRetenuesSection.test.tsx — LA SECTION ENTIÈRE, DANS SES DEUX CONTEXTES.
 *
 * CE TEST EXISTE POUR UNE RAISON PRÉCISE : le lot a été demandé parce que ce qui
 * avait été livré ne portait PAS les cartes de l'artefact. Le test vérifie donc la LISTE —
 * les dix-neuf titres, répartis entre le contexte SOLO (neuf cartes, page Timeseries) et
 * le contexte ESCOUADE (dix cartes, page Escouade) depuis le 2026-09-19 — et les trois
 * blocs.
 *
 * MIS À JOUR LE 2026-09-21 (lot A1) : le titre de section et le bandeau de couverture sont
 * retirés (D5), la réserve des occupations sans nom quitte la carte des frises (D9 du plan
 * précédent, arbitrage du 2026-09-21), les phrases de portée passent dans l'infobulle ⓘ du
 * titre de chaque carte (D5/point 5), et un bloc sans objet se masque INTERTITRE COMPRIS.
 */
import { fireEvent, render, screen, within } from '@testing-library/react'
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

  // LA RÉSERVE QUITTE LA CARTE DES FRISES (arbitrage utilisateur 2026-09-21) : elle vivait
  // sous la piste, en gris. Ce que la barre porte vraiment est désormais dit par la note de
  // méthode de la carte, dans son infobulle ⓘ.
  it('ne pose plus la réserve des occupations sans nom sous les frises', () => {
    const { container } = render(
      <FormesRetenuesSection block={formesFixture()} locale="fr" contexte="squad" />,
    )
    const text = container.textContent ?? ''
    expect(text).not.toContain('5 occupations de socle')
    expect(text).not.toContain('9 prises sur 14')
  })

  it('nomme les cinq usages d’équipement, et JAMAIS les grenades', () => {
    const { container } = render(
      <FormesRetenuesSection block={formesFixture()} locale="fr" contexte="solo" />,
    )
    const text = container.textContent ?? ''
    for (const label of Object.values(fr.axes)) expect(text).toContain(label)
    expect(text).not.toContain('Grenades lancées')
  })

  // D5 (2026-09-21) : ni titre de section, ni bandeau de quatre tuiles. Le nom de la
  // section ne vit plus que dans l'ARIA — trois intertitres se suivaient à l'écran.
  it('ne rend NI titre de section NI bandeau de couverture', () => {
    const { container } = render(
      <FormesRetenuesSection block={formesFixture()} locale="fr" contexte="solo" />,
    )
    expect(screen.getByLabelText(fr.sectionTitle)).toBeInTheDocument()
    expect(container.textContent ?? '').not.toContain(fr.sectionTitle)
    expect(screen.queryByText(/Lobbies observ/)).not.toBeInTheDocument()
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
    // LA PORTÉE SE DIT DANS L'INFOBULLE ⓘ DU TITRE, plus sous la forme (2026-09-21) :
    // hors survol, la phrase n'est nulle part dans le corps de la carte.
    expect(grip.textContent).not.toContain('Affichés : les 20 derniers matchs à film décodé')
    fireEvent.mouseEnter(within(grip).getByRole('button', { name: /info/i }))
    const aide = screen.getByRole('tooltip').textContent ?? ''
    expect(aide).toContain('Affichés : les 20 derniers matchs à film décodé')
    expect(aide).toContain('matchs sans film décodé sont hors de cette forme')
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

  // D8 (2026-09-21) : une section SANS OBJET se masque, intertitre compris. L'intertitre
  // « Objectifs » restait seul au bas de la page, annonçant un bloc qui n'arrivait jamais.
  it('masque le bloc d’objectif ENTIER — intertitre compris — quand aucun match n’en porte', () => {
    const block = formesFixture()
    block.matches = (block.matches ?? []).map((m) => ({ ...m, objective: undefined }))
    render(<FormesRetenuesSection block={block} locale="fr" contexte="solo" />)
    expect(screen.queryByText(fr.blocks.objectives.title)).not.toBeInTheDocument()
    expect(screen.queryByText(frCards.cards.objectivesGapRole.title)).not.toBeInTheDocument()
  })

  // D8 : un bloc sans donnée reste affiché et NOMME SA CAUSE.
  it('un bloc sans film décodé garde son intertitre et nomme la cause', () => {
    const block = formesFixture()
    block.matches = (block.matches ?? []).map((m) => ({ ...m, measured: false }))
    block.matches_measured = 0
    render(<FormesRetenuesSection block={block} locale="fr" contexte="squad" />)
    expect(screen.getByText(fr.blocks.equipment.title)).toBeInTheDocument()
    expect(screen.getByText(fr.blocks.weapons.title)).toBeInTheDocument()
    expect(screen.getAllByTestId('formes-block-empty').length).toBe(2)
    expect(screen.getAllByText(fr.empty.noFilm).length).toBe(2)
  })
})
