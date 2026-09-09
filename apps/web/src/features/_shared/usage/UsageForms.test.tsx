/**
 * UsageForms.test.tsx — LES DEUX PROMESSES DE LA GRILLE DE JAUGES après la revue
 * de lisibilité du 2026-09-09 (`.ai/PLAN_SESSION_USAGE_LISIBILITE_2026-09-09.md`).
 *
 * Déménagé (et renommé) de `session-detail/SessionUsageForms.test.tsx` le 2026-09-09
 * (étape E5.1, PLAN_EQUIPEMENT_GACHIS_2026-09-09.md) — déplacement pur, assertions
 * inchangées, seuls les chemins d'import bougent.
 *
 *   - D4 — UNE colonne rendue par défaut (« ma part dans mon équipe »), les deux autres
 *     dénominateurs derrière le repli. Trois rails par ligne rendaient la grille illisible ;
 *     ce test empêche leur retour silencieux (un `expanded` initialisé à `true`, un repli
 *     débranché) ;
 *   - D2 — le compte brut n'est PAS dans une cellule, il est dans l'infobulle. Le
 *     pourcentage seul est ce qui se lit ; la fraction reste disponible au survol. Le test
 *     vérifie les DEUX moitiés : absente du texte rendu, présente dans l'aide du rail.
 */
import { describe, expect, it } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import type { SessionUsageOutcomes } from '@/lib/api/types'

import { UsageGaugeGrid } from './UsageForms'
import { buildGaugeRow } from './usageGaugeModel'
import { USAGE_TEXT } from './usageI18n'

const t = USAGE_TEXT.fr

const baseShares = {
  player_total: 9,
  team_total: 20,
  lobby_total: 43,
  team_share_of_lobby_pct: 45.6,
  player_share_of_team_pct: 20.5,
  player_share_of_lobby_pct: 9.3,
}

function rows(outcomes?: SessionUsageOutcomes) {
  return [
    buildGaugeRow({
      key: 'camo',
      label: 'Camouflage',
      shares: baseShares,
      outcomes,
      teamParityPct: 25,
      lobbyParityPct: 12.5,
      teamOfLobbyParityPct: 50,
      t,
      locale: 'fr',
    }),
  ]
}

describe('UsageGaugeGrid — repli des dénominateurs secondaires (D4)', () => {
  it('replié : seule « ma part dans mon équipe » est rendue', () => {
    render(<UsageGaugeGrid rows={rows()} t={t} />)
    expect(screen.getByText(t.gaugePlayerOfTeam)).toBeInTheDocument()
    expect(screen.queryByText(t.gaugePlayerOfLobby)).not.toBeInTheDocument()
    expect(screen.queryByText(t.gaugeTeamOfLobby)).not.toBeInTheDocument()
    // Une seule valeur affichée : celle de la colonne rendue.
    expect(screen.getByText('20,5 %')).toBeInTheDocument()
    expect(screen.queryByText('45,6 %')).not.toBeInTheDocument()
  })

  it('déplié : les trois dénominateurs reviennent, rien n avait été retiré du calcul', () => {
    render(<UsageGaugeGrid rows={rows()} t={t} />)
    fireEvent.click(screen.getByRole('button', { name: t.sharesShowMoreFmt(2) }))
    expect(screen.getByText(t.gaugeTeamOfLobby)).toBeInTheDocument()
    expect(screen.getByText(t.gaugePlayerOfTeam)).toBeInTheDocument()
    expect(screen.getByText(t.gaugePlayerOfLobby)).toBeInTheDocument()
    expect(screen.getByText('45,6 %')).toBeInTheDocument()
  })
})

describe('UsageGaugeGrid — le compte brut vit dans l infobulle (D2)', () => {
  it('la cellule porte le pourcentage seul ; le rail porte la fraction', () => {
    render(<UsageGaugeGrid rows={rows()} t={t} />)
    expect(screen.queryByText(/9 sur 20/)).not.toBeInTheDocument()
    const rail = screen.getByRole('img', { name: /Camouflage/ })
    expect(rail).toHaveAttribute('aria-label', expect.stringContaining('9 sur 20'))
  })
})

describe('UsageGauge — E4.1/E4.2 : la pile des trois issues et les deux repères DANS la tranche', () => {
  const outcomes: SessionUsageOutcomes = {
    used: 6,
    dropped: 3,
    kept: 1,
    taken: 11,
    teammates_used_rate_pct: 62,
    opponents_used_rate_pct: 40,
  }

  it('remplit la tranche de trois segments, dans l ordre utilisé -> lâché -> gardé (P1)', () => {
    const { container } = render(<UsageGaugeGrid rows={rows(outcomes)} t={t} />)
    const rail = screen.getByRole('img', { name: /Camouflage/ })
    const segments = rail.querySelectorAll('[data-outcome-key]')
    expect(Array.from(segments).map((el) => el.getAttribute('data-outcome-key'))).toEqual([
      'used',
      'dropped',
      'kept',
    ])
    // Zéro fond uni ALLY_INK quand une pile existe : la tranche n'est QUE des segments.
    expect(container.querySelector('[data-outcome-fill]')).not.toBeInTheDocument()
  })

  it('une famille sans troisième issue (gardé = 0) rend deux segments', () => {
    const twoOutcomes: SessionUsageOutcomes = { used: 6, dropped: 4, kept: 0, taken: 10 }
    render(<UsageGaugeGrid rows={rows(twoOutcomes)} t={t} />)
    const rail = screen.getByRole('img', { name: /Camouflage/ })
    const segments = rail.querySelectorAll('[data-outcome-key]')
    expect(segments).toHaveLength(2)
  })

  it('sans outcomes (grandeur hors bilan) : rendu inchangé, un seul aplat, aucun segment', () => {
    render(<UsageGaugeGrid rows={rows()} t={t} />)
    const rail = screen.getByRole('img', { name: /Camouflage/ })
    expect(rail.querySelectorAll('[data-outcome-key]')).toHaveLength(0)
    expect(rail.querySelector('[data-outcome-fill]')).toBeInTheDocument()
  })

  it('pose les deux repères de taux DANS la tranche, sans chiffre affiché (E4.2)', () => {
    render(<UsageGaugeGrid rows={rows(outcomes)} t={t} />)
    const rail = screen.getByRole('img', { name: /Camouflage/ })
    const teammatesRef = rail.querySelector('[data-outcome-ref="teammates"]')
    const opponentsRef = rail.querySelector('[data-outcome-ref="opponents"]')
    expect(teammatesRef).toBeInTheDocument()
    expect(opponentsRef).toBeInTheDocument()
    expect(teammatesRef).toHaveStyle({ left: '62%' })
    expect(opponentsRef).toHaveStyle({ left: '40%' })
    // Aucun chiffre imprimé dans le rail : le texte du taux ne vit que dans l infobulle.
    expect(rail.textContent).toBe('')
  })

  it('repères absents (scope à camp inconnu) : aucune marque, jamais posée à 0 %', () => {
    const noRef: SessionUsageOutcomes = { used: 6, dropped: 3, kept: 1, taken: 11 }
    render(<UsageGaugeGrid rows={rows(noRef)} t={t} />)
    const rail = screen.getByRole('img', { name: /Camouflage/ })
    expect(rail.querySelector('[data-outcome-ref="teammates"]')).not.toBeInTheDocument()
    expect(rail.querySelector('[data-outcome-ref="opponents"]')).not.toBeInTheDocument()
  })

  it('le compte brut des trois issues reste en infobulle (E4.6)', () => {
    render(<UsageGaugeGrid rows={rows(outcomes)} t={t} />)
    const rail = screen.getByRole('img', { name: /Camouflage/ })
    const label = rail.getAttribute('aria-label') ?? ''
    expect(label).toContain('utilisé 6')
    expect(label).toContain('gardé 1')
    expect(label).toContain('lâché 3')
  })
})
