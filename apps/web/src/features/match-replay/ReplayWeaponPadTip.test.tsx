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

import { REPLAY_TEXT, type ReplayLocale } from './i18n/i18n'
import { ReplayWeaponPadTip } from './ReplayWeaponPadTip'
import type { ReplayWeaponPadReady } from './replayNormalize'
import { padNameFor } from './layers/useReplayWeaponPads'
import type { WeaponPadHover } from './layers/useReplayWeaponPads'

function pad(weapon: string): ReplayWeaponPadReady {
  return { x: 0.26, y: 0, weapon, spawns: [0], presence: [{ t0: 0, tLow: 100, tHigh: 120 }] }
}

function hover(weapon: string, locale: ReplayLocale): WeaponPadHover {
  return {
    pad: pad(weapon),
    at: { x: 10, y: 10 },
    name: padNameFor(weapon, undefined, REPLAY_TEXT[locale], locale),
    state: 'full',
    respawn: null,
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

  it('ne dit QUE le nom : ni état, ni note de lecture (retour du 2026-08-28)', () => {
    const { container } = render(
      <ReplayWeaponPadTip locale="fr" hover={hover('powerup_overshield', 'fr')} width={400} />,
    )
    expect(container.textContent).toBe('Surbouclier')
  })
})

/**
 * LES DEUX COMPTES À REBOURS (D3, 2026-08-27). La carte n'affiche qu'un nombre — à 8 px il n'y a
 * pas la place d'une réserve — et c'est donc l'infobulle, et elle seule, qui doit dire si ce
 * nombre vient d'une apparition VUE dans le film ou d'un cycle qui la PRÉDIT. Les confondre
 * ferait lire une moyenne comme une mesure.
 */
describe('l’infobulle dit D’OÙ VIENT le compte à rebours', () => {
  const vide = (measured: boolean): WeaponPadHover => ({
    ...hover('0x0A1992BC', 'fr'),
    state: 'empty',
    respawn: { seconds: 12.2, measured },
  })

  it('MESURÉ : le chiffre est exact et ne porte AUCUNE réserve', () => {
    const { container } = render(<ReplayWeaponPadTip locale="fr" hover={vide(true)} width={400} />)
    expect(container.textContent).toContain(REPLAY_TEXT.fr.padRespawnMeasuredFmt(12.2))
    expect(container.textContent, 'une mesure ne s’écrit pas « ≈ »').not.toContain('≈')
  })

  it('ATTENDU : le cycle garde son « ≈ », et le texte DIFFÈRE de celui de la mesure', () => {
    const { container } = render(<ReplayWeaponPadTip locale="fr" hover={vide(false)} width={400} />)
    expect(container.textContent).toContain(REPLAY_TEXT.fr.padRespawnExpectedFmt(12.2))
    expect(container.textContent).toContain('≈')
    expect(container.textContent).not.toContain(REPLAY_TEXT.fr.padRespawnMeasuredFmt(12.2))
  })

  it('SANS SOURCE : aucune ligne de réapparition — ni chiffre, ni tiret', () => {
    const rien: WeaponPadHover = { ...hover('0x0A1992BC', 'fr'), state: 'empty', respawn: null }
    const { container } = render(<ReplayWeaponPadTip locale="fr" hover={rien} width={400} />)
    expect(container.textContent).not.toContain('Réapparition')
    // Le nom, et rien d'autre : l'état se lit sur la carte, pas ici.
    expect(container.textContent).toBe('0x0A1992BC')
  })
})

describe('l’infobulle d’un socle d’ARME', () => {
  it('le nom seul quand rien ne date la réapparition', () => {
    const arme: WeaponPadHover = { ...hover('0x0A1992BC', 'fr'), name: 'S7 Sniper' }
    const { container } = render(<ReplayWeaponPadTip locale="fr" hover={arme} width={400} />)
    expect(container.textContent).toBe('S7 Sniper')
  })

  it('le nom PUIS le compte à rebours, et rien de plus, quand une source existe', () => {
    const arme: WeaponPadHover = {
      ...hover('0x0A1992BC', 'fr'),
      name: 'S7 Sniper',
      state: 'empty',
      respawn: { seconds: 12.2, measured: true },
    }
    const { container } = render(<ReplayWeaponPadTip locale="fr" hover={arme} width={400} />)
    expect(container.textContent).toBe(`S7 Sniper${REPLAY_TEXT.fr.padRespawnMeasuredFmt(12.2)}`)
  })
})
