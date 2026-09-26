/**
 * SessionUsageFlagGrabsNet.test.tsx — LES PRISES NETTES DE DRAPEAU dans la carte
 * « Objectifs » de la page Sessions.
 *
 * LA VUE NE PORTE PLUS QUE SON TITRE depuis le 2026-09-21 (D1, décision utilisateur). Ses
 * cinq paragraphes — l'écart brut/net, la fenêtre de jonglage repliée, les ouvertures lues
 * par le film, la couverture en matchs de Capture du drapeau et le périmètre d'équipe — ont
 * été retirés : cinq phrases de méthode pour deux chiffres, sous une carte qui portait déjà
 * trois vues. La grandeur est dite par la jauge du rôle « prendre », qui reste.
 *
 * Ce que ces cas fixent désormais :
 *   - LE TITRE DE VUE RESTE, et il reste CONDITIONNÉ À UNE MESURE : un intertitre seul, sans
 *     rien dessous ET sans prise lue, serait un bloc mort ;
 *   - AUCUNE des cinq phrases retirées ne revient (ratchet anti-résurrection : elles ont
 *     été retirées du dictionnaire, ces cas le vérifient à l'écran).
 */
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'

import type { SessionFlagGrabsNetBlock, SessionUsageBlock } from '@/lib/api/types'

import { SessionUsageSection } from './SessionUsageSection'

const NET: SessionFlagGrabsNetBlock = {
  matches_with_flag_family: 5,
  matches_measured: 3,
  matches_team_known: 3,
  openings_total: 40,
  window_seconds: 1.5,
  player_total: 8,
  lobby_total: 20,
  player_raw_total: 14,
  lobby_raw_total: 33,
  player_team_scope_total: 8,
  team_total: 12,
  player_team_scope_raw_total: 14,
  team_raw_total: 21,
  player_share_of_team_pct: 66.7,
}

/** Un bloc d'usage minimal qui fait rendre la carte « Objectifs ». */
function usageAvec(net: SessionFlagGrabsNetBlock | undefined): SessionUsageBlock {
  return {
    available: true,
    matches_measured: 3,
    matches_total: 5,
    objectives: {
      matches_with_objectives: 5,
      roles: [
        { role: 'take', player_total: 6, team_total: 10, lobby_total: 18 },
        { role: 'defend', player_total: 2, team_total: 5, lobby_total: 9 },
      ],
      flag_grabs_net: net,
    },
  }
}

describe('SessionUsageSection — prises nettes de drapeau', () => {
  it('garde le titre de vue quand le film a lu des prises', () => {
    render(<SessionUsageSection usage={usageAvec(NET)} meLabel="moi" />)
    expect(screen.getByRole('region', { name: 'Prises nettes de drapeau' })).toBeInTheDocument()
  })

  it("n'écrit AUCUNE des cinq phrases explicatives retirées (D1)", () => {
    render(<SessionUsageSection usage={usageAvec(NET)} meLabel="moi" />)
    expect(screen.queryByText(/prises nettes sur les/i)).not.toBeInTheDocument()
    expect(screen.queryByText(/jonglage replié/i)).not.toBeInTheDocument()
    expect(screen.queryByText(/ouvertures de portage/i)).not.toBeInTheDocument()
    expect(screen.queryByText(/matchs de capture du drapeau/i)).not.toBeInTheDocument()
    expect(screen.queryByText(/la comparaison avec ton camp porte sur/i)).not.toBeInTheDocument()
    expect(screen.queryByText(/ta part :/i)).not.toBeInTheDocument()
  })

  it('la jauge du rôle « prendre » reste, elle : c’est elle qui dit la grandeur', () => {
    render(<SessionUsageSection usage={usageAvec(NET)} meLabel="moi" />)
    expect(screen.getByText('Prendre')).toBeInTheDocument()
  })

  it('ne rend RIEN quand aucune prise n’a été lue sur le scope', () => {
    render(<SessionUsageSection usage={usageAvec(undefined)} meLabel="moi" />)
    expect(screen.queryByRole('region', { name: 'Prises nettes de drapeau' })).toBeNull()
  })
})
