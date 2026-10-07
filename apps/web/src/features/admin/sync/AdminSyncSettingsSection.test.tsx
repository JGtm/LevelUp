/**
 * AdminSyncSettingsSection — réglages de synchronisation de l'instance.
 *
 * Non-régression : réglages DÉJÀ en cache au montage (la coquille les lit avant la page). Les
 * contrôles montrent la valeur enregistrée, pas leur défaut.
 */
import { screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import type { SettingsResponse } from '@/lib/api/types'
import { renderWithProviders } from '@/test/render-utils'

import { AdminSyncSettingsSection } from './AdminSyncSettingsSection'

const reponseSettings: Partial<SettingsResponse> = {
  spnkr_auto_sync_enabled: true,
  spnkr_auto_sync_interval_minutes: 45,
}

vi.mock('@/features/settings/queries', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/features/settings/queries')>()),
  useSettings: () => ({ data: reponseSettings, isLoading: false }),
  useUpdateSettings: () => ({ mutate: vi.fn(), isPending: false }),
}))

describe('AdminSyncSettingsSection', () => {
  it('réglages déjà servis au montage : l’intervalle enregistré (45), pas le défaut (360)', () => {
    renderWithProviders(<AdminSyncSettingsSection />)
    expect(screen.getByDisplayValue('45')).toBeInTheDocument()
    expect(screen.queryByDisplayValue('360')).not.toBeInTheDocument()
  })
})
