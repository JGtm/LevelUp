/**
 * FormesRetenuesSection.test.tsx — LA SECTION ENTIÈRE, contexte SOLO (Séries temporelles).
 *
 * CE TEST EXISTE POUR UNE RAISON PRÉCISE : le lot a été demandé parce que ce qui
 * avait été livré ne portait PAS les cartes de l'artefact. Le test vérifie donc la LISTE —
 * les neuf titres du contexte solo (le contexte escouade a été retiré au lot L5.4 du plan
 * PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26 : l'onglet Emprise le remplace) — et les trois
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
  // NEUF depuis le lot L5.4 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26 (les huit
  // cartes du contexte escouade sont retirées avec l'ancien onglet Usages ; les deux cartes
  // d'objectif de l'escouade l'avaient été au lot L3).
  it('les NEUF cartes de l’artefact sont toutes montées', () => {
    expect(ALL_CARDS).toHaveLength(9)
    const { container } = render(<FormesRetenuesSection block={formesFixture()} locale="fr" />)
    const texte = container.textContent ?? ''
    for (const key of ALL_CARDS) {
      expect(texte).toContain(frCards.cards[key].title)
    }
  })

  it('rend les trois blocs, chacun avec son titre — plus aucun intertitre de contexte', () => {
    const { container } = render(
      <FormesRetenuesSection block={formesFixture()} locale="fr" />,
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
      <FormesRetenuesSection block={formesFixture()} locale="fr" />,
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

  it('nomme les cinq usages d’équipement, et JAMAIS les grenades', () => {
    const { container } = render(
      <FormesRetenuesSection block={formesFixture()} locale="fr" />,
    )
    const text = container.textContent ?? ''
    for (const label of Object.values(fr.axes)) expect(text).toContain(label)
    expect(text).not.toContain('Grenades lancées')
  })

  // D5 (2026-09-21) : ni titre de section, ni bandeau de quatre tuiles. Le nom de la
  // section ne vit plus que dans l'ARIA — trois intertitres se suivaient à l'écran.
  it('ne rend NI titre de section NI bandeau de couverture', () => {
    const { container } = render(
      <FormesRetenuesSection block={formesFixture()} locale="fr" />,
    )
    expect(screen.getByLabelText(fr.sectionTitle)).toBeInTheDocument()
    expect(container.textContent ?? '').not.toContain(fr.sectionTitle)
    expect(screen.queryByText(/Lobbies observ/)).not.toBeInTheDocument()
  })

  it('se retire quand le bloc est absent ou indisponible', () => {
    const { container: empty } = render(
      <FormesRetenuesSection block={undefined} locale="fr" />,
    )
    expect(empty).toBeEmptyDOMElement()
    const { container: down } = render(
      <FormesRetenuesSection
        block={{ available: false, matches_total: 3, matches_measured: 0 }}
        locale="fr"
       
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
    render(<FormesRetenuesSection block={block} locale="fr" />)
    const grip = screen.getByLabelText(frCards.cards.equipmentByMatch.title)
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
    render(<FormesRetenuesSection block={formesFixture()} locale="en" />)
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
    render(<FormesRetenuesSection block={block} locale="fr" />)
    expect(screen.queryByText(fr.blocks.objectives.title)).not.toBeInTheDocument()
    expect(screen.queryByText(frCards.cards.objectivesGapRole.title)).not.toBeInTheDocument()
  })

  // D8 : un bloc sans donnée reste affiché et NOMME SA CAUSE.
  it('un bloc sans film décodé garde son intertitre et nomme la cause', () => {
    const block = formesFixture()
    block.matches = (block.matches ?? []).map((m) => ({ ...m, measured: false }))
    block.matches_measured = 0
    render(<FormesRetenuesSection block={block} locale="fr" />)
    expect(screen.getByText(fr.blocks.equipment.title)).toBeInTheDocument()
    expect(screen.getByText(fr.blocks.weapons.title)).toBeInTheDocument()
    expect(screen.getAllByTestId('formes-block-empty').length).toBe(2)
    expect(screen.getAllByText(fr.empty.noFilm).length).toBe(2)
  })
})
