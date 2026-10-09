/**
 * cssColorToHex.test — la normalisation de couleur qui sauve l'emphase d'ECharts.
 *
 * Deux mondes à couvrir : le NAVIGATEUR (un contexte 2D existe, il convertit) et JSDOM (pas
 * de contexte : `getContext('2d')` rend `null` et journalise « Not implemented »), où la
 * fonction doit rendre l'entrée INCHANGÉE plutôt que `undefined` — un `undefined` peint en
 * transparent, c'est exactement le bug que ce module corrige.
 *
 * Le contexte 2D est SIMULÉ pour le premier monde : on ne teste pas le moteur de couleur du
 * navigateur, on teste qu'on lui parle correctement (deux sentinelles, deux lectures).
 */
import { afterEach, describe, expect, it, vi } from 'vitest'

/** Contexte 2D factice : il ne connaît que ces quatre couleurs et IGNORE le reste, comme le
 *  vrai `fillStyle`, qui garde sa valeur précédente devant une chaîne qu'il ne comprend pas. */
function fakeContext() {
  const CANONICAL: Record<string, string> = {
    black: '#000000',
    white: '#ffffff',
    'oklch(0.708 0 0)': '#9ca3af',
    'rgb(147, 197, 253)': '#93c5fd',
  }
  let value = '#000000'
  return {
    get fillStyle() {
      return value
    },
    set fillStyle(next: string) {
      const canonical = CANONICAL[next]
      if (canonical) value = canonical
    },
  }
}

/** Recharge le module (la sonde est mémorisée) avec le contexte 2D voulu. */
async function loadWith(context: unknown) {
  vi.resetModules()
  vi.spyOn(document, 'createElement').mockImplementation(
    () => ({ getContext: () => context }) as unknown as HTMLElement,
  )
  return import('./cssColorToHex')
}

afterEach(() => {
  vi.restoreAllMocks()
  vi.resetModules()
})

describe('cssColorToHex — avec un contexte 2D', () => {
  it('convertit un oklch en hex : zrender ne sait parser QUE la seconde forme', async () => {
    const { cssColorToHex } = await loadWith(fakeContext())
    expect(cssColorToHex('oklch(0.708 0 0)')).toBe('#9ca3af')
  })

  it('rend une couleur déjà parsable inchangée dans sa forme canonique', async () => {
    const { cssColorToHex } = await loadWith(fakeContext())
    expect(cssColorToHex('rgb(147, 197, 253)')).toBe('#93c5fd')
  })

  it('rend l’entrée telle quelle si la couleur est invalide — jamais un noir inventé', async () => {
    // Sans la double sentinelle, `fillStyle` garderait « #000000 » et la fonction rendrait du
    // noir pour une valeur qu'elle n'a pas su lire : un rendu faux, silencieux.
    const { cssColorToHex } = await loadWith(fakeContext())
    expect(cssColorToHex('pas-une-couleur')).toBe('pas-une-couleur')
  })

  it('ne crée qu’UNE sonde, même sur plusieurs appels', async () => {
    const { cssColorToHex } = await loadWith(fakeContext())
    cssColorToHex('oklch(0.708 0 0)')
    cssColorToHex('oklch(0.708 0 0)')
    expect(vi.mocked(document.createElement)).toHaveBeenCalledTimes(1)
  })
})

describe('cssColorToHex — sans contexte 2D (jsdom, SSR)', () => {
  it('rend l’entrée inchangée quand getContext donne null', async () => {
    const { cssColorToHex } = await loadWith(null)
    expect(cssColorToHex('oklch(0.708 0 0)')).toBe('oklch(0.708 0 0)')
  })

  it('rend l’entrée inchangée quand getContext LÈVE', async () => {
    vi.resetModules()
    vi.spyOn(document, 'createElement').mockImplementation(
      () =>
        ({
          getContext: () => {
            throw new Error('Not implemented')
          },
        }) as unknown as HTMLElement,
    )
    const { cssColorToHex } = await import('./cssColorToHex')
    expect(cssColorToHex('oklch(0.708 0 0)')).toBe('oklch(0.708 0 0)')
  })

  it('rend la chaîne vide telle quelle', async () => {
    const { cssColorToHex } = await loadWith(fakeContext())
    expect(cssColorToHex('')).toBe('')
  })
})
