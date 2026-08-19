/**
 * Tests — ReplayWeaponPadTip : CE QUE L'INFOBULLE MONTRE VRAIMENT, à l'écran.
 *
 * POURQUOI UN RENDU ET PAS SEULEMENT LES RÉSOLVEURS. Le défaut du 2026-08-19 était visible
 * sur l'ÉCRAN et nulle part ailleurs : une clé de famille (`powerup_overshield`) s'affichait
 * telle quelle dans l'infobulle du socle central de Catalyst. Un test qui n'éprouverait que
 * `padNameFor` laisserait repasser la même faute par un autre chemin (un champ oublié, une
 * réserve de lecture qui recopie la clé). Ici on lit le texte rendu, comme le lecteur.
 */
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'

import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import { ReplayWeaponPadTip } from './ReplayWeaponPadTip'
import type { ReplayWeaponPadReady } from './replayNormalize'
import { padNameFor } from './useReplayWeaponPads'
import type { WeaponPadHover } from './useReplayWeaponPads'

function pad(weapon: string): ReplayWeaponPadReady {
  return { x: 0.26, y: 0, weapon, spawns: [0], presence: [{ t0: 0, tLow: 100, tHigh: 120 }] }
}

function hover(weapon: string, locale: ReplayLocale): WeaponPadHover {
  return {
    pad: pad(weapon),
    at: { x: 10, y: 10 },
    name: padNameFor(weapon, undefined, REPLAY_TEXT[locale], locale),
    state: 'full',
    respawnS: null,
  }
}

describe('l’infobulle d’un socle de POWER-UP', () => {
  it('nomme le surbouclier dans les deux langues, et n’écrit JAMAIS la clé brute', () => {
    for (const locale of ['fr', 'en'] as const) {
      const { container, unmount } = render(
        <ReplayWeaponPadTip locale={locale} hover={hover('powerup_overshield', locale)} width={400} />,
      )
      expect(container.textContent, `clé brute affichée en ${locale}`).not.toContain('powerup_')
      expect(screen.getByRole('tooltip').textContent).toContain(
        locale === 'fr' ? 'Surbouclier' : 'Overshield',
      )
      unmount()
    }
  })

  it('sa réserve de lecture ne parle PAS de râtelier mural — un power-up ne s’y accroche pas', () => {
    const { container } = render(
      <ReplayWeaponPadTip locale="fr" hover={hover('powerup_overshield', 'fr')} width={400} />,
    )
    expect(container.textContent).toContain(REPLAY_TEXT.fr.padPlacementNotePowerUp)
    expect(container.textContent).not.toContain('râtelier')
  })
})

describe('l’infobulle d’un socle d’ARME — inchangée', () => {
  it('garde le nom servi par le survol et la réserve socle / râtelier', () => {
    const arme: WeaponPadHover = { ...hover('0x0A1992BC', 'fr'), name: 'S7 Sniper' }
    const { container } = render(<ReplayWeaponPadTip locale="fr" hover={arme} width={400} />)
    expect(container.textContent).toContain('S7 Sniper')
    expect(container.textContent).toContain(REPLAY_TEXT.fr.padPlacementNote)
  })
})
