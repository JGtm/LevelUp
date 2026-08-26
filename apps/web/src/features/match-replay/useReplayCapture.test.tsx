/**
 * Tests — useReplayCapture (la couture React de la capture).
 *
 * CE QU'ILS PROTÈGENT. La commande lit la toile et l'horloge AU MOMENT DU CLIC, pas au rendu :
 * le fichier doit porter l'instant qu'il montre. Et quand la toile ne rend pas d'image — toile
 * absente, encodage refusé — il ne se télécharge RIEN : un fichier vide ferait croire à une
 * capture réussie.
 *
 * LE MODULE DE SORTIE EST DOUBLÉ (`replayCapture`), et c'est la bonne frontière : ce qu'il fait
 * est vérifié chez lui (`replayCapture.test.ts`), ce qu'on vérifie ici est QUI l'appelle, QUAND,
 * et AVEC QUOI. jsdom n'a de toute façon ni `toBlob` ni téléchargement réel.
 */
import { act, renderHook } from '@testing-library/react'
import { createRef, type RefObject } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { captureCanvasImage, triggerDownload } from './replayCapture'
import { testReplayDoc } from './test/testDoc'
import { useReplayCapture } from './useReplayCapture'

vi.mock('./replayCapture', async (importOriginal) => {
  // Le NOMMAGE reste le vrai : c'est lui qui prouve que l'instant lu est le bon.
  const vrai = await importOriginal<typeof import('./replayCapture')>()
  return {
    ...vrai,
    captureCanvasImage: vi.fn(),
    triggerDownload: vi.fn(),
  }
})

/** 20 images par seconde : l'image 40 tombe donc à la 2e seconde de match. */
const DOC = testReplayDoc({ matchId: 'match-témoin', frameIntervalMs: 50 })

function mount(frame: number, canvas: HTMLCanvasElement | null = document.createElement('canvas')) {
  const canvasRef = createRef<HTMLCanvasElement>() as RefObject<HTMLCanvasElement | null>
  canvasRef.current = canvas
  const frameRef = createRef<number>() as RefObject<number>
  frameRef.current = frame
  return renderHook(() => useReplayCapture({ canvasRef, doc: DOC, frameRef }))
}

beforeEach(() => {
  vi.mocked(captureCanvasImage).mockReset()
  vi.mocked(triggerDownload).mockReset()
})

describe('useReplayCapture — capturer l’image', () => {
  it('télécharge un PNG nommé sur l’instant de match affiché', async () => {
    const blob = new Blob(['png'])
    vi.mocked(captureCanvasImage).mockResolvedValue(blob)
    const { result } = mount(40)
    await act(async () => {
      result.current.captureImage()
    })
    expect(captureCanvasImage).toHaveBeenCalledTimes(1)
    // L'identifiant du match est assaini au passage (l'accent ne survit pas au nom de fichier).
    expect(triggerDownload).toHaveBeenCalledWith(blob, 'rejeu-match-t-moin-0m02s.png')
  })

  it('une toile qui ne rend pas d’image ne télécharge RIEN', async () => {
    vi.mocked(captureCanvasImage).mockResolvedValue(null)
    const { result } = mount(40)
    await act(async () => {
      result.current.captureImage()
    })
    expect(triggerDownload).not.toHaveBeenCalled()
  })

  it('sans toile montée, la commande ne fait rien et ne lève pas', async () => {
    const { result } = mount(40, null)
    await act(async () => {
      result.current.captureImage()
    })
    expect(captureCanvasImage).not.toHaveBeenCalled()
    expect(triggerDownload).not.toHaveBeenCalled()
  })
})
