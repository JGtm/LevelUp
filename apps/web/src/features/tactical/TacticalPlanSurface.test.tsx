/**
 * Tests — TacticalPlanSurface : la toile du relief se pose au calage de l'image (même style de placement, rotation
 * comprise) et recopie sa surface à chaque nouvelle frame.
 */
import { render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type { ReliefFrame } from '@/lib/replay/relief'
import type { ReliefSurface } from '@/lib/replay/useReliefView'

import { TacticalPlanSurface } from './TacticalPlanSurface'

const CADRAGE = { left: '10%', top: '20%', width: '80%', height: '60%', transform: 'rotate(90deg)' }

function surface(): ReliefSurface {
  const image = document.createElement('canvas')
  image.width = 40
  image.height = 10
  return { image, frame: {} as ReliefFrame }
}

afterEach(() => vi.restoreAllMocks())

describe('TacticalPlanSurface', () => {
  it('prend la place de l’image : mêmes pourcentages et même rotation', () => {
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue({ clearRect: vi.fn(), drawImage: vi.fn() } as never)
    render(<TacticalPlanSurface surface={surface()} cadrage={CADRAGE} />)
    expect(screen.getByTestId('tactical-plan-surface')).toHaveStyle(CADRAGE)
  })

  it('recopie la surface à la taille du relief', () => {
    const drawImage = vi.fn()
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue({ clearRect: vi.fn(), drawImage } as never)
    const s = surface()
    render(<TacticalPlanSurface surface={s} cadrage={CADRAGE} />)
    const canvas = screen.getByTestId('tactical-plan-surface') as HTMLCanvasElement
    expect([canvas.width, canvas.height]).toEqual([40, 10])
    expect(drawImage).toHaveBeenCalledWith(s.image, 0, 0, 40, 10)
  })

  it('repeint quand la frame change, même si le canvas réutilisé est le même', () => {
    const drawImage = vi.fn()
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue({ clearRect: vi.fn(), drawImage } as never)
    const s = surface()
    const { rerender } = render(<TacticalPlanSurface surface={s} cadrage={CADRAGE} />)
    rerender(<TacticalPlanSurface surface={{ image: s.image, frame: {} as ReliefFrame }} cadrage={CADRAGE} />)
    expect(drawImage).toHaveBeenCalledTimes(2)
  })

  it('ne dessine rien d’une source sans taille', () => {
    const drawImage = vi.fn()
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue({ clearRect: vi.fn(), drawImage } as never)
    const vide = document.createElement('canvas')
    vide.width = 0
    vide.height = 0
    render(<TacticalPlanSurface surface={{ image: vide, frame: {} as ReliefFrame }} cadrage={CADRAGE} />)
    expect(drawImage).not.toHaveBeenCalled()
  })
})
