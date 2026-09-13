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
  it("affiche les prises nettes, le compteur officiel, la règle et la couverture", () => {
    render(<SessionUsageSection usage={usageAvec(NET)} meLabel="moi" />)
    expect(screen.getByRole('region', { name: 'Prises nettes de drapeau' })).toBeInTheDocument()
    // L'écart net / brut DU FILM : 12 nettes pour 21 ramassages lus.
    expect(screen.getByText(/8 prises nettes sur les 12 de ton équipe/)).toBeInTheDocument()
    expect(screen.getByText(/le film lit 21 ramassages pour ce camp/)).toBeInTheDocument()
    // La règle, avec sa fenêtre.
    expect(screen.getByText(/jonglage replié \(fenêtre 1,5 s\)/i)).toBeInTheDocument()
    // LE DÉNOMINATEUR DU BRUT : les ouvertures lues par l'oracle du film.
    expect(
      screen.getByText(/le film a lu 40 ouvertures de portage, dont 33 attribuées/i),
    ).toBeInTheDocument()
    // La couverture : 3 des 5 matchs DE CAPTURE DU DRAPEAU (pas « à objectif »).
    expect(
      screen.getByText(/mesuré sur 3 des 5 matchs de capture du drapeau/i),
    ).toBeInTheDocument()
  })

  it("dit le périmètre réduit quand des matchs mesurés n'ont pas de camp connu", () => {
    render(
      <SessionUsageSection usage={usageAvec({ ...NET, matches_team_known: 2 })} meLabel="moi" />,
    )
    expect(
      screen.getByText(/la comparaison avec ton camp porte sur 2 de ces 3 matchs/i),
    ).toBeInTheDocument()
  })

  it("ne dit rien du périmètre d'équipe quand tous les matchs mesurés ont un camp", () => {
    render(<SessionUsageSection usage={usageAvec(NET)} meLabel="moi" />)
    expect(screen.queryByText(/la comparaison avec ton camp porte sur/i)).not.toBeInTheDocument()
  })

  it("n'annonce AUCUNE fenêtre quand le scope en mêle plusieurs (le serveur publie 0)", () => {
    render(<SessionUsageSection usage={usageAvec({ ...NET, window_seconds: 0 })} meLabel="moi" />)
    expect(screen.queryByText(/jonglage replié/i)).not.toBeInTheDocument()
    // Le reste tient : les totaux et la couverture restent lisibles.
    expect(screen.getByText(/mesuré sur 3 des 5 matchs de capture du drapeau/i)).toBeInTheDocument()
  })

  it('ne rend RIEN quand aucune prise n’a été lue sur le scope', () => {
    render(<SessionUsageSection usage={usageAvec(undefined)} meLabel="moi" />)
    expect(screen.queryByRole('region', { name: 'Prises nettes de drapeau' })).toBeNull()
  })
})
