/**
 * SessionUsageFlagGrabsNet.test.tsx — LES PRISES NETTES DE DRAPEAU dans la carte
 * « Objectifs » de la page Sessions.
 *
 * Ce que ces cas fixent, et pourquoi chacun compte :
 *
 *   - L'ÉCART BRUT/NET EST AFFICHÉ. C'est lui qui justifie la grandeur : sans le compteur
 *     officiel à côté, « 12 prises nettes » ressemble à un compteur de plus.
 *   - LA RÈGLE EST ÉCRITE. Une grandeur qui replie du jonglage sans dire dans quelle fenêtre
 *     n'est pas vérifiable par le lecteur.
 *   - LA COUVERTURE EST ÉCRITE. Les matchs sans film décodé ne comptent pour aucune prise, et
 *     l'écran doit le dire au lieu de laisser croire à un total complet.
 *   - DEUX FENÊTRES DANS LE SCOPE ⇒ AUCUNE N'EST ANNONCÉE (le serveur publie alors zéro).
 */
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'

import type { SessionFlagGrabsNetBlock, SessionUsageBlock } from '@/lib/api/types'

import { SessionUsageSection } from './SessionUsageSection'

const NET: SessionFlagGrabsNetBlock = {
  matches_with_objectives: 5,
  matches_measured: 3,
  window_seconds: 1.5,
  player_total: 8,
  team_total: 12,
  lobby_total: 20,
  player_raw_total: 14,
  team_raw_total: 21,
  lobby_raw_total: 33,
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
  it("affiche les prises nettes, le compteur officiel, la règle et la couverture", () => {
    render(<SessionUsageSection usage={usageAvec(NET)} meLabel="moi" />)
    expect(screen.getByRole('region', { name: 'Prises nettes de drapeau' })).toBeInTheDocument()
    // L'écart brut/net : 12 nettes pour 21 au compteur officiel.
    expect(screen.getByText(/8 prises nettes sur les 12 de ton équipe/)).toBeInTheDocument()
    expect(screen.getByText(/le compteur officiel en affiche 21/)).toBeInTheDocument()
    // La règle, avec sa fenêtre.
    expect(screen.getByText(/jonglage replié \(fenêtre 1,5 s\)/i)).toBeInTheDocument()
    // La couverture : 3 des 5 matchs à objectif.
    expect(screen.getByText(/mesuré sur 3 des 5 matchs à objectif/i)).toBeInTheDocument()
  })

  it("n'annonce AUCUNE fenêtre quand le scope en mêle plusieurs (le serveur publie 0)", () => {
    render(<SessionUsageSection usage={usageAvec({ ...NET, window_seconds: 0 })} meLabel="moi" />)
    expect(screen.queryByText(/jonglage replié/i)).not.toBeInTheDocument()
    // Le reste tient : les totaux et la couverture restent lisibles.
    expect(screen.getByText(/mesuré sur 3 des 5 matchs à objectif/i)).toBeInTheDocument()
  })

  it('ne rend RIEN quand aucune prise n’a été lue sur le scope', () => {
    render(<SessionUsageSection usage={usageAvec(undefined)} meLabel="moi" />)
    expect(screen.queryByRole('region', { name: 'Prises nettes de drapeau' })).toBeNull()
  })
})
