/**
 * SessionUsageForms.test.tsx — LES DEUX PROMESSES DE LA GRILLE DE JAUGES après la revue
 * de lisibilité du 2026-09-09 (`.ai/PLAN_SESSION_USAGE_LISIBILITE_2026-09-09.md`) :
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

import { UsageGaugeGrid } from './SessionUsageForms'
import { USAGE_TEXT } from './usageI18n'
import { buildGaugeRow } from './usageLogic'

const t = USAGE_TEXT.fr

function rows() {
  return [
    buildGaugeRow({
      key: 'camo',
      label: 'Camouflage',
      shares: {
        player_total: 9,
        team_total: 20,
        lobby_total: 43,
        team_share_of_lobby_pct: 45.6,
        player_share_of_team_pct: 20.5,
        player_share_of_lobby_pct: 9.3,
      },
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
