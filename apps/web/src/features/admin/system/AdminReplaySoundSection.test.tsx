/**
 * Tests AdminReplaySoundSection — les deux curseurs « Sons du rejeu ».
 *
 * CE QUI EST VÉRIFIÉ, ET POURQUOI. Le réglage sert un effet qu'on ne peut pas juger à
 * l'écran : si le curseur enregistre la mauvaise valeur, personne ne le voit — on entend
 * seulement, des jours plus tard, que « les sons ne varient pas ». Ces tests fixent donc
 * les valeurs par défaut (100 / 0, celles du serveur), le fait qu'un geste enregistre la
 * valeur relâchée, et qu'un déplacement en cours n'envoie RIEN au serveur.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import type { SettingsResponse } from '@/lib/api/types'
import { AdminReplaySoundSection } from './AdminReplaySoundSection'

const mutate = vi.fn()
let reponseSettings: Partial<SettingsResponse> | undefined

vi.mock('@/features/settings/queries', () => ({
  useSettings: () => ({ data: reponseSettings, isLoading: false }),
  useUpdateSettings: () => ({ mutate, isPending: false }),
}))

function curseurs(): HTMLInputElement[] {
  return screen.getAllByRole('slider') as HTMLInputElement[]
}

describe('AdminReplaySoundSection', () => {
  beforeEach(() => {
    mutate.mockClear()
    reponseSettings = { replay_sound_variation_percent: 100, replay_sound_distance_percent: 0 }
  })

  it('affiche les deux curseurs, et deux seulement — le plan en exige trois boutons maximum', () => {
    renderWithProviders(<AdminReplaySoundSection />)
    expect(curseurs()).toHaveLength(2)
    expect(screen.getByText('Sons du rejeu')).toBeInTheDocument()
    expect(screen.getByText('Variation')).toBeInTheDocument()
    expect(screen.getByText('Distance')).toBeInTheDocument()
  })

  it('part des valeurs du serveur : variation pleine, aucune distance', () => {
    renderWithProviders(<AdminReplaySoundSection />)
    const [variation, distance] = curseurs()
    expect(variation.value).toBe('100')
    expect(distance.value).toBe('0')
    expect(screen.getByText('100 %')).toBeInTheDocument()
    expect(screen.getByText('0 %')).toBeInTheDocument()
  })

  it('retombe sur les valeurs d’usine quand le serveur ne porte pas encore les clés', () => {
    reponseSettings = {}
    renderWithProviders(<AdminReplaySoundSection />)
    const [variation, distance] = curseurs()
    expect(variation.value).toBe('100')
    expect(distance.value).toBe('0')
  })

  it('un déplacement en cours n’envoie RIEN — un PATCH par pixel noierait le serveur', () => {
    renderWithProviders(<AdminReplaySoundSection />)
    fireEvent.change(curseurs()[0], { target: { value: '45' } })
    expect(screen.getByText('45 %')).toBeInTheDocument()
    expect(mutate).not.toHaveBeenCalled()
  })

  it('le relâchement enregistre la valeur, sur le bon champ', () => {
    renderWithProviders(<AdminReplaySoundSection />)
    const [variation, distance] = curseurs()

    fireEvent.change(variation, { target: { value: '40' } })
    fireEvent.mouseUp(variation)
    expect(mutate).toHaveBeenCalledWith({ replay_sound_variation_percent: 40 })

    fireEvent.change(distance, { target: { value: '75' } })
    fireEvent.mouseUp(distance)
    expect(mutate).toHaveBeenCalledWith({ replay_sound_distance_percent: 75 })
  })

  it('le clavier enregistre aussi : une flèche est déjà un geste complet', () => {
    renderWithProviders(<AdminReplaySoundSection />)
    const [variation] = curseurs()
    fireEvent.change(variation, { target: { value: '95' } })
    fireEvent.keyUp(variation)
    expect(mutate).toHaveBeenCalledWith({ replay_sound_variation_percent: 95 })
  })

  it('les curseurs sont bornés 0-100 : hors de là, le serveur refuse le PATCH', () => {
    renderWithProviders(<AdminReplaySoundSection />)
    for (const curseur of curseurs()) {
      expect(curseur.min).toBe('0')
      expect(curseur.max).toBe('100')
    }
  })
})
