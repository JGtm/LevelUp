/**
 * ReplayViewMenu.test.tsx — LE BOUTON « VUE » de la carte : il nomme la vue active, dit par
 * `aria-expanded` si le panneau est ouvert, l'ouvre et le ferme au clavier comme à la souris, et
 * Échap le ferme en rendant le focus au bouton.
 */
import { useReplaySettings } from './useReplaySettings'
import { useReplayViewMenu } from './useReplayViewMenu'
import { ReplayViewMenu } from './ReplayViewMenu'
import type { FilmLayerAvailability } from './viewPresetLogic'
import type { ReplayLocale } from '../i18n/i18n'
import type { ReplaySound } from '../sound/useReplaySound'

import { fireEvent, render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it } from 'vitest'

const FILM: FilmLayerAvailability = {
  zones: true,
  placements: { drawable: true, unnamed: false, dropped: false },
  weaponPads: true,
  groundWeapons: true,
  modeObjectives: true,
  flagCarries: false,
  vipCrown: false,
  skullCarrier: false,
  bombCarrier: false,
  vehicles: true,
  wallPlan: true,
  lives: { spawns: true, deaths: true, killers: true, followedKnown: true },
  respawnForecast: false,
}

const SOUND = { available: false, categories: {} } as unknown as ReplaySound
const ORIENTATION = { turns: 0 as const, setTurns: () => {}, available: true }

function Harness({ locale, planFeedAvailable = false }: { locale: ReplayLocale; planFeedAvailable?: boolean }) {
  const settings = useReplaySettings()
  const view = useReplayViewMenu({
    settings,
    orientation: ORIENTATION,
    sound: SOUND,
    available: FILM,
    heat: { mode: 'presence', killsAvailable: true },
    planBackground: { available: false, shown: false },
    planFeedAvailable,
    locale,
  })
  return (
    <div className="relative">
      <ReplayViewMenu {...view} />
    </div>
  )
}

describe('ReplayViewMenu', () => {
  beforeEach(() => {
    window.localStorage.clear()
  })

  it('le bouton est fermé au départ et nomme la vue active', () => {
    render(<Harness locale="fr" />)
    const btn = screen.getByRole('button', { name: 'Vue Complet' })
    expect(btn).toHaveAttribute('aria-expanded', 'false')
    expect(screen.queryByRole('region', { name: 'Vue' })).toBeNull()
  })

  it('le clic ouvre le panneau (aria-expanded) et un second clic le referme', () => {
    render(<Harness locale="fr" />)
    const btn = screen.getByRole('button', { name: 'Vue Complet' })
    fireEvent.click(btn)
    expect(btn).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByRole('region', { name: 'Vue' })).toBeTruthy()
    fireEvent.click(btn)
    expect(btn).toHaveAttribute('aria-expanded', 'false')
    expect(screen.queryByRole('region', { name: 'Vue' })).toBeNull()
  })

  it('choisir un préréglage change le nom porté par le bouton, un calque à la main dit Personnalisé', () => {
    render(<Harness locale="fr" />)
    fireEvent.click(screen.getByRole('button', { name: 'Vue Complet' }))
    fireEvent.click(screen.getByRole('button', { name: 'Combat' }))
    expect(screen.getByRole('button', { name: 'Vue Combat' })).toBeTruthy()
    fireEvent.click(screen.getByRole('switch', { name: 'Zones' }))
    expect(screen.getByRole('button', { name: 'Vue Personnalisé' })).toBeTruthy()
  })

  it('Échap ferme le panneau et le focus revient au bouton', () => {
    render(<Harness locale="fr" />)
    const btn = screen.getByRole('button', { name: 'Vue Complet' })
    fireEvent.click(btn)
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(screen.queryByRole('region', { name: 'Vue' })).toBeNull()
    expect(document.activeElement).toBe(btn)
  })

  it('en anglais : View Full', () => {
    render(<Harness locale="en" />)
    expect(screen.getByRole('button', { name: 'View Full' })).toBeTruthy()
  })
})

describe("ReplayViewMenu : l'option « Fil des éliminations »", () => {
  const KEY = 'replay-show-plan-feed'
  const option = () => screen.queryByRole('switch', { name: 'Fil des éliminations' })
  const ouvrir = () => fireEvent.click(screen.getByRole('button', { name: /^(Vue|View) / }))

  beforeEach(() => {
    window.localStorage.clear()
  })

  it("affichée par défaut : l'interrupteur est allumé, sans rien avoir retenu", () => {
    render(<Harness locale="fr" planFeedAvailable />)
    ouvrir()
    expect(option()).toHaveAttribute('aria-checked', 'true')
    expect(window.localStorage.getItem(KEY)).toBeNull()
  })

  it("là où le fil du plan ne peut pas se poser (Standard, repli), l'option n'est pas montrée", () => {
    render(<Harness locale="fr" />)
    ouvrir()
    expect(screen.getByRole('region', { name: 'Vue' })).toBeTruthy()
    expect(option()).toBeNull()
  })

  it("l'éteindre est retenu, et l'allumer aussi", () => {
    render(<Harness locale="fr" planFeedAvailable />)
    ouvrir()
    fireEvent.click(option()!)
    expect(option()).toHaveAttribute('aria-checked', 'false')
    expect(window.localStorage.getItem(KEY)).toBe('false')
    fireEvent.click(option()!)
    expect(option()).toHaveAttribute('aria-checked', 'true')
    expect(window.localStorage.getItem(KEY)).toBe('true')
  })

  it('éteinte, elle le reste après remontage', () => {
    const premier = render(<Harness locale="fr" planFeedAvailable />)
    ouvrir()
    fireEvent.click(option()!)
    premier.unmount()
    render(<Harness locale="fr" planFeedAvailable />)
    ouvrir()
    expect(option()).toHaveAttribute('aria-checked', 'false')
  })

  it("en anglais, l'option porte son libellé anglais", () => {
    render(<Harness locale="en" planFeedAvailable />)
    ouvrir()
    expect(screen.getByRole('switch', { name: 'Elimination feed' })).toBeTruthy()
  })
})
