/**
 * TacticalPlanFond — LE FOND PLAN : l'image `?variante=plan` est posée quand le réglage « Fond » du rejeu
 * 2D le demande ET que la carte en porte une (`planImage` du sidecar) ; sinon le rendu, sans rien dire
 * (pas d'erreur, pas de requête de l'image plan). Une clé de cache par variante : le plan ne sert
 * jamais l'image rendu, ni l'inverse.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { screen, waitFor } from '@testing-library/react'

import { useAppShellStore } from '@/stores/appShellStore'
import { renderWithProviders } from '@/test/render-utils'

import { TacticalPlanFond } from './TacticalPlanFond'

const get = vi.fn()
const getBlob = vi.fn()
vi.mock('@/lib/api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api/client')>()
  return { ...actual, api: { ...actual.api, get: (p: string) => get(p), getBlob: (p: string) => getBlob(p) } }
})

const CADRAGE = { left: '0%', top: '0%', width: '100%', height: '100%' }
const CALAGE = { metersPerPixel: 1, originX: 0, originY: 60, widthPx: 60, heightPx: 60, convention: 'xy' }
const RENDU = '/players/JGtm/tactical/streets/background.png'
const PLAN = '/players/JGtm/tactical/streets/background.png?variante=plan'

function sidecar(avecPlan: boolean) {
  get.mockImplementation((path: string) => {
    if (path === '/players/JGtm/tactical/streets/background') {
      return Promise.resolve({ calibration: CALAGE, ...(avecPlan ? { planImage: 'plan.png' } : {}) })
    }
    return Promise.reject(new Error(`appel inattendu : ${path}`))
  })
}

function rendre() {
  return renderWithProviders(
    <TacticalPlanFond playerSlug="JGtm" mapId="streets" aspect={2} hauteurMax={400} cadrage={CADRAGE} />,
  )
}

beforeEach(() => {
  window.localStorage.clear()
  get.mockReset()
  getBlob.mockReset().mockResolvedValue(new Blob(['x']))
  useAppShellStore.setState({ currentTitleSlug: 'halo_infinite' })
  let n = 0
  vi.spyOn(URL, 'createObjectURL').mockImplementation(() => `blob:image-${++n}`)
})

describe('TacticalPlanFond — style du fond', () => {
  it('par défaut : l’image rendu, même si la carte a un plan', async () => {
    sidecar(true)
    rendre()
    await waitFor(() => expect(document.querySelector('img')).not.toBeNull())
    expect(getBlob).toHaveBeenCalledWith(RENDU)
    expect(getBlob).not.toHaveBeenCalledWith(PLAN)
  })

  it('réglage plan et carte avec image plan : l’image plan est demandée et posée', async () => {
    window.localStorage.setItem('replay-background-style', 'plan')
    sidecar(true)
    rendre()
    await waitFor(() => expect(getBlob).toHaveBeenCalledWith(PLAN))
    await waitFor(() => expect(document.querySelector('img')).not.toBeNull())
    expect(getBlob).not.toHaveBeenCalledWith(RENDU)
  })

  it('réglage plan mais carte SANS image plan : repli silencieux sur le rendu', async () => {
    window.localStorage.setItem('replay-background-style', 'plan')
    sidecar(false)
    rendre()
    await waitFor(() => expect(getBlob).toHaveBeenCalledWith(RENDU))
    expect(getBlob).not.toHaveBeenCalledWith(PLAN)
    await waitFor(() => expect(document.querySelector('img')).not.toBeNull())
  })

  it('l’image plan qui ne se charge pas : le rendu est posé, aucune erreur à l’écran', async () => {
    window.localStorage.setItem('replay-background-style', 'plan')
    sidecar(true)
    getBlob.mockImplementation((path: string) =>
      path === PLAN ? Promise.reject(new Error('404')) : Promise.resolve(new Blob(['x'])),
    )
    rendre()
    await waitFor(() => expect(getBlob).toHaveBeenCalledWith(RENDU))
    await waitFor(() => expect(document.querySelector('img')).not.toBeNull())
    expect(screen.queryByRole('alert')).toBeNull()
  })

  it('valeur de réglage inconnue : le rendu', async () => {
    window.localStorage.setItem('replay-background-style', 'bizarre')
    sidecar(true)
    rendre()
    await waitFor(() => expect(getBlob).toHaveBeenCalledWith(RENDU))
    expect(getBlob).not.toHaveBeenCalledWith(PLAN)
  })
})
