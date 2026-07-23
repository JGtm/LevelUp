/**
 * Tests AppearanceDiagSection — diagnostic apparence à la demande.
 * Couvre : état initial (aucun diagnostic), état en cours, et le rendu des 5
 * verdicts depuis des fixtures (mock API MSW), CTA de réauthentification inclus.
 */
import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { screen, fireEvent } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'
import type { AppearanceDiagnosisResponse, PlayerSummary } from '@/lib/api/types'
import { AppearanceDiagSection } from './AppearanceDiagSection'

function player(slug: string, gamertag: string): PlayerSummary {
  return { player_slug: slug, gamertag, xuid: 'x1' } as PlayerSummary
}

function baseResponse(
  components: AppearanceDiagnosisResponse['components'],
): AppearanceDiagnosisResponse {
  return {
    player_slug: 'jgtm',
    gamertag: 'JGtm',
    xuid: 'x1',
    title_slug: 'halo_infinite',
    generated_at: '2026-07-23T10:00:00Z',
    last_fetch_status: 'ok',
    components,
  }
}

// Bannière/emblème/arrière-plan/indicatif : 4 verdicts distincts en une réponse.
const FOUR_VERDICTS = baseResponse([
  { component: 'banner', served_value: 'https://example.test/banner.png', served_from: 'carry', verdict: 'upstream_missing', detail: 'no_positive_cfg' },
  { component: 'emblem', served_value: 'https://example.test/emblem.png', served_from: 'live', verdict: 'ok', detail: 'image_resolved' },
  { component: 'backdrop', served_value: 'https://example.test/backdrop.png', served_from: 'carry', verdict: 'transient', detail: 'image_unresolved' },
  { component: 'service_tag', served_value: '117', served_from: 'carry', verdict: 'auth_required', detail: '' },
])

// 5e verdict : titre à pipeline dédié (H5) → 4 composants not_supported.
const NOT_SUPPORTED = baseResponse([
  { component: 'banner', served_value: '', served_from: 'carry', verdict: 'not_supported', detail: '' },
  { component: 'emblem', served_value: '', served_from: 'carry', verdict: 'not_supported', detail: '' },
  { component: 'backdrop', served_value: '', served_from: 'carry', verdict: 'not_supported', detail: '' },
  { component: 'service_tag', served_value: '', served_from: 'carry', verdict: 'not_supported', detail: '' },
])

function mockDiag(resp: AppearanceDiagnosisResponse, opts?: { delayMs?: number }) {
  server.use(
    http.get('/api/v1/admin/diag/appearance/:slug', async () => {
      if (opts?.delayMs) await new Promise((r) => setTimeout(r, opts.delayMs))
      return HttpResponse.json(resp)
    }),
  )
}

async function selectAndDiagnose() {
  const select = screen.getByRole('combobox')
  fireEvent.change(select, { target: { value: 'jgtm' } })
  fireEvent.click(screen.getByRole('button', { name: /Diagnostiquer/i }))
}

describe('AppearanceDiagSection', () => {
  beforeEach(() => {
    useAppShellStore.setState({ locale: 'fr', availablePlayers: [player('jgtm', 'JGtm')] })
  })
  afterEach(() => {
    useAppShellStore.setState({ locale: 'fr', availablePlayers: [] })
  })

  it('état initial : aucun diagnostic lancé', () => {
    renderWithProviders(<AppearanceDiagSection />)
    expect(screen.getByText('Aucun diagnostic lancé')).toBeInTheDocument()
    // Le bouton est désactivé tant qu'aucun joueur n'est sélectionné.
    expect(screen.getByRole('button', { name: /Diagnostiquer/i })).toBeDisabled()
  })

  it('état en cours : le diagnostic tourne', async () => {
    mockDiag(FOUR_VERDICTS, { delayMs: 80 })
    renderWithProviders(<AppearanceDiagSection />)
    await selectAndDiagnose()
    expect((await screen.findAllByText(/Diagnostic en cours/i)).length).toBeGreaterThan(0)
    // Puis le résultat arrive.
    await screen.findByText('Bannière')
  })

  it('rend 4 verdicts distincts + composants + CTA de réauthentification', async () => {
    mockDiag(FOUR_VERDICTS)
    renderWithProviders(<AppearanceDiagSection />)
    await selectAndDiagnose()

    // Libellés des 4 composants.
    await screen.findByText('Bannière')
    expect(screen.getByText('Emblème')).toBeInTheDocument()
    expect(screen.getByText('Arrière-plan')).toBeInTheDocument()
    expect(screen.getByText('Indicatif de service')).toBeInTheDocument()

    // Badges de verdict.
    expect(screen.getByText('Absent côté Microsoft')).toBeInTheDocument()
    expect(screen.getByText('À jour')).toBeInTheDocument()
    expect(screen.getByText('Temporaire')).toBeInTheDocument()
    expect(screen.getByText('Réauthentification requise')).toBeInTheDocument()

    // « rien à faire, servi par design » (upstream_missing).
    expect(
      screen.getByText(/servie par design/i),
    ).toBeInTheDocument()

    // CTA de réauthentification (auth_required) → flux SSO existant.
    const cta = screen.getByText('Se connecter via Xbox').closest('a')
    expect(cta).not.toBeNull()
    expect(cta?.getAttribute('href')).toContain('/auth/xbox/login')

    // Indicatif de service : valeur texte servie.
    expect(screen.getByText('117')).toBeInTheDocument()
  })

  it('rend le verdict not_supported (titre à pipeline dédié)', async () => {
    mockDiag(NOT_SUPPORTED)
    renderWithProviders(<AppearanceDiagSection />)
    await selectAndDiagnose()

    const badges = await screen.findAllByText('Non pris en charge')
    expect(badges.length).toBe(4)
    // Valeur servie absente → état vide explicite (pas d'image cassée).
    expect(screen.getAllByText('Aucune valeur servie').length).toBe(4)
  })
})
