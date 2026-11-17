/**
 * ReplayZoomControl.avant.test.tsx — fixation de la commande de cadrage (± zoom, croix, retour à 1x).
 *
 * Ce que la fixation couvre : deux rendus posés bout à bout, le cadrage à 1x (zoom arrière et
 * déplacement désactivés) puis un palier zoomé (commandes actives), pour qu'une régression sur l'un
 * ou l'autre état se voie.
 *
 * Aucune valeur ne varie : le composant ne lit ni date ni identifiant généré.
 *
 * Régénération : `npx vitest run <ce fichier> -u`, à la création seulement. Jamais pour faire passer une divergence.
 */
import { describe, expect, it, vi } from 'vitest'
import { render, within } from '@testing-library/react'

import { ZOOM_LEVELS } from '../../../lib/replay/replayLogic'
import type { ReplayZoom } from '../hooks/useReplayZoom'
import { ReplayZoomControl } from './ReplayZoomControl'

/**
 * Un zoom figé à un palier donné. Le composant ne lit que ces champs : les actions sont des espions,
 * les capacités suivent le palier comme le fait le hook (zoom arrière impossible à 1x, déplacement
 * seulement au-delà de 1x).
 */
function zoomA(indexPalier: number): ReplayZoom {
  const dernier = ZOOM_LEVELS.length - 1
  return {
    level: ZOOM_LEVELS[indexPalier],
    center: { x: 0.5, y: 0.5 },
    canZoomIn: indexPalier < dernier,
    canZoomOut: indexPalier > 0,
    canPan: indexPalier > 0,
    zoomIn: vi.fn(),
    zoomOut: vi.fn(),
    reset: vi.fn(),
    panStep: vi.fn(),
    zoomAt: vi.fn(),
    panBy: vi.fn(),
    focusOn: vi.fn(),
  } as unknown as ReplayZoom
}

describe('ReplayZoomControl — fixation de la commande de zoom (avant la refonte)', () => {
  it('le HTML de la commande est celui de la fixture prise avant la refonte, à 1x puis zoomée', async () => {
    const aUnX = render(<ReplayZoomControl zoom={zoomA(0)} locale="fr" />)
    const zoomee = render(<ReplayZoomControl zoom={zoomA(1)} locale="fr" />)
    // Les commandes attendues sont là, et l'état désactivé du palier 1x est bien posé : sans cela la
    // fixture comparerait deux rendus qui ne disent rien.
    const a = within(aUnX.container)
    const b = within(zoomee.container)
    expect(a.getByRole('group', { name: 'Cadrage de la carte' })).toBeTruthy()
    expect(a.getByRole('button', { name: 'Réduire (− ou molette)' }).hasAttribute('disabled')).toBe(true)
    expect(b.getByRole('button', { name: 'Réduire (− ou molette)' }).hasAttribute('disabled')).toBe(false)
    const html = aUnX.container.innerHTML + zoomee.container.innerHTML
    await expect(html).toMatchFileSnapshot('./__fixtures__/replayZoomControl.avant.html')
  })
})
