/**
 * replay.gate.test.tsx — LA PORTE DE TITRE DE LA PAGE DE REJEU (v2 D.14).
 *
 * CE QU'IL VÉRIFIE, et c'est le point produit : sur un titre qui ne déclare pas la capability
 * `replay`, la page n'est pas montée — donc AUCUNE requête de rejeu n'est émise — et l'écran
 * dit pourquoi, en FR comme en EN. L'artefact pèse de 1,5 à 2,7 Mio : une page qui se monte
 * pour afficher un vide aurait payé ce téléchargement pour rien, et le 404 aurait ressemblé à
 * une panne plutôt qu'à un titre sans rejeu.
 *
 * DEUX ASSERTIONS QUI SE TIENNENT :
 *  1. le COMPORTEMENT des portes, sur la composition exacte de la route (les deux
 *     `RouteCapabilityGate` imbriqués), avec un enfant qui appelle le vrai hook de chargement ;
 *  2. la SOURCE de la route, qui doit bien monter cette composition — sans quoi le test 1
 *     vérifierait une composition que personne n'utilise.
 */
/// <reference types="node" />
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'

import { afterEach, describe, expect, it, vi } from 'vitest'
import { screen, waitFor } from '@testing-library/react'

import { api } from '@/lib/api/client'
import { RouteCapabilityGate } from '@/lib/capabilities/RouteCapabilityGate'
import { useMatchReplay } from '@/lib/replay/queries'
import { useAppShellStore } from '@/stores/appShellStore'
import { renderWithProviders } from '@/test/render-utils'

vi.mock('@/lib/api/client', () => ({
  api: { get: vi.fn() },
  getApiTitleSlug: () => 'halo_infinite',
  setApiTitleSlug: vi.fn(),
  setApiLocale: vi.fn(),
}))

const apiGet = vi.mocked(api.get)

/** Le titre courant du magasin d'app, avec la liste de capabilities qu'on veut éprouver. */
function titreAvecCapabilities(caps: string[]) {
  useAppShellStore.setState({
    currentTitleSlug: 'titre_test',
    availableTitles: [
      {
        slug: 'titre_test',
        name: 'Titre de test',
        status: 'active',
        capabilities: caps,
        is_default: true,
        effective_hp_to_kill: 225,
        provides_damage_taken: true,
        provides_team_mmr: true,
        provides_max_killing_spree: true,
        offensive_conversion_p80: 0.9,
        defensive_resistance_p80: 1.65,
      },
    ],
  })
}

/** L'enfant que la route monte : ce qui compte ici est qu'il DEMANDE l'artefact. */
function PageEspionne() {
  useMatchReplay('jgtm', 'match-1')
  return <div>contenu du rejeu</div>
}

/** La composition EXACTE de `Route.options.component`. */
function SousLesDeuxPortes() {
  return (
    <RouteCapabilityGate capability="matchmaking">
      <RouteCapabilityGate capability="replay">
        <PageEspionne />
      </RouteCapabilityGate>
    </RouteCapabilityGate>
  )
}

afterEach(() => {
  apiGet.mockReset()
  useAppShellStore.setState({ currentTitleSlug: 'halo_infinite', availableTitles: [] })
})

describe('page de rejeu — porte de titre `replay`', () => {
  it('titre SANS la clé : aucune requête de rejeu, et l’écran dit pourquoi', async () => {
    titreAvecCapabilities(['matchmaking'])
    apiGet.mockResolvedValue({})

    renderWithProviders(<SousLesDeuxPortes />)

    expect(screen.getByText('Indisponible pour ce titre')).toBeInTheDocument()
    expect(screen.getByText('Ce titre ne fournit pas le rejeu 2D des matchs.')).toBeInTheDocument()
    expect(screen.queryByText('contenu du rejeu')).not.toBeInTheDocument()
    // La page n'est pas montée : son hook de chargement n'a jamais tourné.
    await waitFor(() => expect(apiGet).not.toHaveBeenCalled())
  })

  it('titre SANS la clé, en anglais : le même état, dans la langue du lecteur', () => {
    titreAvecCapabilities(['matchmaking'])
    useAppShellStore.setState({ locale: 'en' })

    renderWithProviders(<SousLesDeuxPortes />)

    expect(screen.getByText('Not available for this title')).toBeInTheDocument()
    expect(screen.getByText('This title does not provide 2D match replay.')).toBeInTheDocument()
    useAppShellStore.setState({ locale: 'fr' })
  })

  it('titre AVEC la clé : la page est montée et demande son artefact', async () => {
    titreAvecCapabilities(['matchmaking', 'replay'])
    apiGet.mockResolvedValue({ schemaVersion: 4, tracks: [] })

    renderWithProviders(<SousLesDeuxPortes />)

    expect(screen.getByText('contenu du rejeu')).toBeInTheDocument()
    await waitFor(() =>
      expect(apiGet).toHaveBeenCalledWith('/players/jgtm/matches/match-1/replay'),
    )
  })

  it('la route monte bien cette composition — sans quoi ce test ne garderait rien', () => {
    const source = readFileSync(resolve(dirname(__filename), 'replay.tsx'), 'utf8')
    expect(source).toMatch(/<RouteCapabilityGate capability="matchmaking">/)
    expect(source).toMatch(/<RouteCapabilityGate capability="replay">/)
    // La porte enveloppe la page, elle ne la suit pas.
    const gateReplay = source.indexOf('capability="replay"')
    const page = source.indexOf('<ReplayPage />')
    expect(gateReplay).toBeGreaterThan(0)
    expect(page).toBeGreaterThan(gateReplay)
  })
})
