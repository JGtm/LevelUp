/**
 * Tests — ReplayVehicleCycleTip : CE QUE L'INFOBULLE D'UN EMPLACEMENT DE VÉHICULE MONTRE, à
 * l'écran.
 *
 * POURQUOI UN RENDU ET PAS SEULEMENT LES RÉSOLVEURS : la réserve du cycle (médiane, déciles,
 * nombre de cycles mesurés) n'existe QUE là. Un test qui n'éprouverait que la lecture temporelle
 * laisserait passer une infobulle qui l'oublie, et c'est précisément ce qui distingue un chiffre
 * mesuré d'un chiffre prédit pour le lecteur.
 *
 * LES DEUX LANGUES SONT ÉPROUVÉES : la parité FR/EN est tenue par le typage, mais le CONTENU
 * (« Cycle ≈ », « Mesuré sur ») ne l'est par rien d'autre que ce fichier.
 */
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'

import { REPLAY_TEXT } from '../i18n/i18n'
import { ReplayVehicleCycleTip } from './ReplayVehicleCycleTip'
import type { VehicleCycleHover } from '../layers/useReplayVehicles'

const CYCLE = { x: 12, y: 8, family: 'warthog', medianS: 95, p10S: 75, p90S: 120, gaps: 4, missing: 1 }

function hover(over: Partial<VehicleCycleHover> = {}): VehicleCycleHover {
  return {
    cycle: CYCLE,
    at: { x: 10, y: 10 },
    name: 'Warthog',
    occupied: false,
    respawn: null,
    ...over,
  }
}

describe('l’infobulle d’un emplacement de naissance de véhicule', () => {
  it('nomme la famille et dit la réserve du cycle, dans les deux langues', () => {
    for (const locale of ['fr', 'en'] as const) {
      const { unmount } = render(
        <ReplayVehicleCycleTip locale={locale} hover={hover()} width={400} />,
      )
      const texte = screen.getByRole('tooltip').textContent ?? ''
      expect(texte, `famille en ${locale}`).toContain('Warthog')
      expect(texte, `cycle en ${locale}`).toContain(
        REPLAY_TEXT[locale].vehicleCycleFmt(95, 75, 120),
      )
      expect(texte, `écarts en ${locale}`).toContain(REPLAY_TEXT[locale].vehicleCycleGapsFmt(4))
      unmount()
    }
  })

  it('un compte MESURÉ se dit sans réserve ; un compte PRÉDIT garde son « ≈ »', () => {
    const mesure = render(
      <ReplayVehicleCycleTip
        locale="fr"
        hover={hover({ respawn: { seconds: 12.2, measured: true } })}
        width={400}
      />,
    )
    expect(screen.getByRole('tooltip').textContent).toContain(
      REPLAY_TEXT.fr.padRespawnMeasuredFmt(12.2),
    )
    mesure.unmount()
    render(
      <ReplayVehicleCycleTip
        locale="fr"
        hover={hover({ respawn: { seconds: 12.2, measured: false } })}
        width={400}
      />,
    )
    expect(screen.getByRole('tooltip').textContent).toContain('≈')
  })

  it('un emplacement OCCUPÉ dit pourquoi il n’a pas de compte', () => {
    render(<ReplayVehicleCycleTip locale="fr" hover={hover({ occupied: true })} width={400} />)
    expect(screen.getByRole('tooltip').textContent).toContain(REPLAY_TEXT.fr.vehicleCycleOccupied)
  })

  it('libre et SANS source : ni compte, ni « véhicule présent » (aucun tiret non plus)', () => {
    render(<ReplayVehicleCycleTip locale="fr" hover={hover()} width={400} />)
    const texte = screen.getByRole('tooltip').textContent ?? ''
    expect(texte).not.toContain(REPLAY_TEXT.fr.vehicleCycleOccupied)
    expect(texte).not.toContain('Réapparition')
    expect(texte).not.toContain('—')
  })
})
