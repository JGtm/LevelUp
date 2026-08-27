/**
 * Tests — replayCapture (le nom du fichier, le téléchargement, l'image du canvas).
 *
 * CE QU'ILS PROTÈGENT. Trois promesses, et chacune se casse en silence si on ne la tient pas :
 * le nom porte l'INSTANT DU MATCH (un horodatage local ne dit rien à qui reçoit le fichier),
 * le blob remis au navigateur est RÉVOQUÉ tout de suite (sans quoi une vidéo de plusieurs
 * mégaoctets reste retenue jusqu'à la fermeture de l'onglet), et une toile qui ne rend pas
 * d'image ne déclenche AUCUN téléchargement (un fichier vide serait pire que rien).
 *
 * jsdom N'A PAS `canvas.toBlob` (ni `URL.createObjectURL` avant qu'on l'y pose) : ces tests
 * les remplacent par des doublures, exactement comme `useReplayPlayback.test.tsx` remplace la
 * boucle d'animation. Ce n'est pas un contournement — il n'y a rien de RÉEL à mesurer ici, la
 * question posée est « qui est appelé, avec quoi ».
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { buildCaptureFilename, captureCanvasImage, triggerDownload } from './replayCapture'

/** Une horloge FIXE pour le repli : le nom ne doit dépendre d'aucune heure réelle. */
const HORLOGE = new Date(2026, 7, 26, 14, 3, 9)

describe('buildCaptureFilename — le nom dit l’instant du match', () => {
  it('nominal : identifiant du match et temps de match', () => {
    expect(buildCaptureFilename('abc-123', 74_000, 'png', HORLOGE)).toBe('rejeu-abc-123-1m14s.png')
  })

  it('les secondes tiennent toujours sur deux chiffres, les minutes jamais', () => {
    // Le tri alphabétique d'un dossier de captures suit alors l'ordre du match, ce qu'un
    // « 1m4s » ferait mentir dès la dixième seconde.
    expect(buildCaptureFilename('m', 4_000, 'png', HORLOGE)).toBe('rejeu-m-0m04s.png')
  })

  it('au-delà de dix minutes, l’heure de match reste lisible', () => {
    expect(buildCaptureFilename('m', 754_000, 'webm', HORLOGE)).toBe('rejeu-m-12m34s.webm')
  })

  it('sans identifiant de match : repli horodaté (décision 8)', () => {
    expect(buildCaptureFilename(null, 74_000, 'mp4', HORLOGE)).toBe('rejeu-20260826-140309.mp4')
    expect(buildCaptureFilename('   ', 74_000, 'mp4', HORLOGE)).toBe('rejeu-20260826-140309.mp4')
  })

  it('sans instant lisible : repli AUSSI — un nom ne doit pas inventer « 0m00s »', () => {
    // « 0m00s » se lirait « coup d'envoi » : c'est une affirmation, pas une absence.
    expect(buildCaptureFilename('abc', null, 'png', HORLOGE)).toBe('rejeu-20260826-140309.png')
    expect(buildCaptureFilename('abc', Number.NaN, 'png', HORLOGE)).toBe('rejeu-20260826-140309.png')
  })

  it('un identifiant exotique ne s’échappe pas dans le nom de fichier', () => {
    expect(buildCaptureFilename('a/b c:d', 0, 'png', HORLOGE)).toBe('rejeu-a-b-c-d-0m00s.png')
  })
})

describe('triggerDownload — le blob est rendu, puis relâché', () => {
  let created: string[] = []
  let revoked: string[] = []

  beforeEach(() => {
    created = []
    revoked = []
    vi.stubGlobal('URL', {
      createObjectURL: (b: Blob) => {
        created.push(String(b.size))
        return 'blob:rejeu-test'
      },
      revokeObjectURL: (u: string) => revoked.push(u),
    })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('pose une ancre nommée, la déclenche, et révoque l’URL', () => {
    const clic = vi.fn()
    const ancre = { href: '', download: '', rel: '', click: clic } as unknown as HTMLAnchorElement
    const create = vi.spyOn(document, 'createElement').mockReturnValue(ancre)
    triggerDownload(new Blob(['xy']), 'rejeu-m-0m04s.png')
    expect(create).toHaveBeenCalledWith('a')
    expect(ancre.download).toBe('rejeu-m-0m04s.png')
    expect(ancre.href).toBe('blob:rejeu-test')
    expect(clic).toHaveBeenCalledTimes(1)
    // LA RÉVOCATION EST LE POINT DU TEST : sans elle le blob reste retenu tout l'onglet.
    expect(revoked).toEqual(['blob:rejeu-test'])
    expect(created).toHaveLength(1)
    create.mockRestore()
  })
})

describe('captureCanvasImage — les pixels, ou rien', () => {
  it('rend le blob PNG que la toile produit', async () => {
    const blob = new Blob(['png'])
    const canvas = {
      toBlob: (cb: (b: Blob | null) => void, type: string) => {
        expect(type).toBe('image/png')
        cb(blob)
      },
    } as unknown as HTMLCanvasElement
    await expect(captureCanvasImage(canvas)).resolves.toBe(blob)
  })

  it('une toile qui ne rend rien donne `null` — et non une promesse en suspens', async () => {
    const canvas = {
      toBlob: (cb: (b: Blob | null) => void) => cb(null),
    } as unknown as HTMLCanvasElement
    await expect(captureCanvasImage(canvas)).resolves.toBeNull()
  })

  it('un navigateur sans `toBlob` donne `null` au lieu de lever', async () => {
    // C'est le cas de jsdom, et ce serait celui d'un très vieux navigateur : la commande
    // ne doit pas casser la page, elle doit ne rien faire.
    await expect(captureCanvasImage({} as HTMLCanvasElement)).resolves.toBeNull()
  })
})
