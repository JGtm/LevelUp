/**
 * SessionUsageSection.gate.test.tsx — LES DEUX PORTES DU BLOC « usages d'équipement,
 * armes spéciales et objectifs » (règle du 2026-09-05, registre L4).
 *
 * Le bloc affichait, sur un titre sans décodeur de film, une carte « Ce titre ne publie pas
 * de résumé d'usage des films » : `unsupported` était traité comme `empty` au lieu de
 * `hidden`. Une carte qui annonce une absence définitive occupe une place, ne dit rien
 * d'actionnable et ne disparaîtra jamais — un bloc mort.
 *
 * Ce fichier fixe la distinction, sur le COMPOSANT (usageLogic.test.ts la fixe sur la
 * fonction) :
 *   - `unsupported` (le TITRE ne le publie pas)   → RIEN ;
 *   - `load_failed` (CETTE lecture a échoué)      → carte d'état vide AVEC la raison ;
 *   - 0 match mesuré (le titre publie, pas ici)   → carte d'état vide « aucun film ».
 */
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'

import type { SessionUsageBlock } from '@/lib/api/types'

import { USAGE_TEXT } from '@/features/_shared/usage/usageI18n'

import { SessionUsageSection } from './SessionUsageSection'

const BASE: SessionUsageBlock = { available: true, matches_measured: 4, matches_total: 6 }

describe('SessionUsageSection — porte de titre', () => {
  it("titre qui ne publie pas de résumé d'usage (unsupported) : RIEN n'est rendu", () => {
    const { container } = render(
      <SessionUsageSection
        usage={{ ...BASE, available: false, unavailable_reason: 'unsupported' }}
        meLabel="moi"
      />,
    )
    expect(container).toBeEmptyDOMElement()
  })

  it('bloc absent du payload (vieux serveur) : RIEN non plus', () => {
    const { container } = render(<SessionUsageSection usage={undefined} meLabel="moi" />)
    expect(container).toBeEmptyDOMElement()
  })
})

describe('SessionUsageSection — porte de donnée (le titre publie)', () => {
  it('lecture échouée : la carte dit la raison, elle est transitoire', () => {
    render(
      <SessionUsageSection
        usage={{ ...BASE, available: false, unavailable_reason: 'load_failed' }}
        meLabel="moi"
      />,
    )
    expect(screen.getByText(/La lecture du résumé d'usage a échoué/)).toBeInTheDocument()
  })

  it('aucun match mesuré : la carte le dit, avec le dénominateur', () => {
    render(<SessionUsageSection usage={{ ...BASE, matches_measured: 0 }} meLabel="moi" />)
    expect(screen.getByText(/Aucun match de cette session n'a de film mesuré/)).toBeInTheDocument()
  })
})

/**
 * Colonne divisée (drawer de comparaison ouvert) — demande utilisateur du 2026-09-13 :
 * « il manque des éléments du drawer ».
 *
 * LE COMPACT NE RETIRE PLUS RIEN, il resserre : les mêmes vues sont rendues des deux
 * côtés, avec des rails et des cases plus petits. Ce test empêche le retour de la
 * dérive précédente — masquer la régularité, la piste du lobby ou les grilles dès que
 * la colonne est divisée.
 */
describe('SessionUsageSection — version compacte du drawer', () => {
  const MEASURED: SessionUsageBlock = {
    ...BASE,
    team_parity_pct: 25,
    metrics: [
      {
        key: 'pad_pickups',
        player_total: 9,
        team_total: 20,
        lobby_total: 43,
        matches_above_lobby_parity: 1,
        player_share_of_team_pct: 45,
        per_match: [{ match_id: 'm1', player_share_of_team_pct: 45 }],
      },
    ],
  }

  it('pleine largeur : la piste du lobby et la régularité sont rendues', () => {
    render(<SessionUsageSection usage={MEASURED} meLabel="moi" />)
    expect(screen.getByLabelText(USAGE_TEXT.fr.viewRegularity)).toBeInTheDocument()
    expect(screen.getAllByLabelText(USAGE_TEXT.fr.viewLobbyTrack).length).toBeGreaterThan(0)
  })

  it('compact : les mêmes vues sont rendues, rien ne disparaît', () => {
    render(<SessionUsageSection usage={MEASURED} meLabel="moi" compact />)
    expect(screen.getByLabelText(USAGE_TEXT.fr.viewRegularity)).toBeInTheDocument()
    expect(screen.getAllByLabelText(USAGE_TEXT.fr.viewLobbyTrack).length).toBeGreaterThan(0)
    expect(screen.getAllByLabelText(USAGE_TEXT.fr.viewShares).length).toBeGreaterThan(0)
  })
})

/**
 * E4.5 / E4.3 — bout en bout : la ligne « Objets lâchés » disparaît de la liste des
 * grandeurs (une mort est devenue un segment) et le contrat étendu en E3
 * (`equipment_<famille>.outcomes`) atteint bien la pile de la jauge à l'écran.
 */
describe('SessionUsageSection — E4.5/E4.3 : le bilan équipement remplace le geste', () => {
  const WITH_BILAN: SessionUsageBlock = {
    ...BASE,
    team_parity_pct: 25,
    metrics: [
      {
        key: 'dropped_objects',
        player_total: 4,
        lobby_total: 30,
        matches_above_lobby_parity: 0,
      },
      {
        key: 'deployed_wall',
        player_total: 12,
        team_total: 20,
        lobby_total: 43,
        matches_above_lobby_parity: 1,
      },
      {
        key: 'equipment_wall',
        player_total: 9,
        team_total: 20,
        lobby_total: 43,
        matches_above_lobby_parity: 1,
        player_share_of_team_pct: 45,
        outcomes: { used: 6, dropped: 2, kept: 1, taken: 10 },
      },
    ],
  }

  it("« Objets lâchés » ne s'affiche plus (E4.5)", () => {
    render(<SessionUsageSection usage={WITH_BILAN} meLabel="moi" />)
    expect(screen.queryByText(USAGE_TEXT.fr.metricDropped)).not.toBeInTheDocument()
  })

  it("deployed_wall s'efface derrière son équivalent equipment_wall, dont la pile se rend", () => {
    const { container } = render(<SessionUsageSection usage={WITH_BILAN} meLabel="moi" />)
    // La pile à trois segments (E4.3) vient de equipment_wall, seule survivante :
    // deployed_wall n'ouvre pas de deuxième ligne pour la même famille. Elle est
    // Une seule pile dessinée : la jauge « dans le lobby » n'a pas de part dans ce
    // contrat partiel, donc pas de tranche à remplir.
    expect(container.querySelectorAll('[data-outcome-key]')).toHaveLength(3)
  })
})
