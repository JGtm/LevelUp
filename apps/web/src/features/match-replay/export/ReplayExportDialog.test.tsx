/**
 * ReplayExportDialog.test.tsx — LES DEUX BORNES, LA CASE SON, ET CE QUI SE PASSE PENDANT.
 *
 * Ce qui se verrouille ici est ce qu'un utilisateur peut CASSER : croiser les deux curseurs,
 * lancer un export muet en croyant l'avoir demandé sonore, ou perdre le bouton « Annuler » au
 * milieu d'un calcul de plusieurs minutes.
 */
import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { ReplayExportDialog } from './ReplayExportDialog'
import type { ReplayExport } from './useReplayExport'
import { EXPORT_FORMAT_KEY } from '../settings/replayPreferences'

function makeExport(over: Partial<ReplayExport> = {}): ReplayExport {
  return {
    supported: true,
    state: { phase: 'idle', done: 0, total: 0, pct: 0 },
    defaultBounds: () => ({ startFrame: 0, endFrame: 100 }),
    run: vi.fn(async () => {}),
    cancel: vi.fn(),
    clockOf: (f) => `frame:${f}`,
    lengthClock: (b) => `${b.startFrame}-${b.endFrame}`,
    zoomLevel: 1,
    ...over,
  }
}

function setup(over: Partial<ReplayExport> = {}) {
  const exporter = makeExport(over)
  const onClose = vi.fn()
  render(<ReplayExportDialog exporter={exporter} locale="fr" onClose={onClose} />)
  return { exporter, onClose }
}

describe('ReplayExportDialog — au repos', () => {
  it('propose la plage entière par défaut', () => {
    setup()
    const debut = screen.getByLabelText('Début') as HTMLInputElement
    const fin = screen.getByLabelText('Fin') as HTMLInputElement
    expect(debut.value).toBe('0')
    expect(fin.value).toBe('100')
  })

  it('lance l’export avec les bornes et le son, inclus PAR DÉFAUT', async () => {
    const user = userEvent.setup()
    const { exporter } = setup()
    await user.click(screen.getByRole('button', { name: 'Exporter' }))
    expect(exporter.run).toHaveBeenCalledWith({ startFrame: 0, endFrame: 100 }, { sound: true, format: '1080p', framing: 'whole' })
  })

  it('exporte muet quand la case est décochée', async () => {
    const user = userEvent.setup()
    const { exporter } = setup()
    await user.click(screen.getByLabelText('Inclure le son'))
    await user.click(screen.getByRole('button', { name: 'Exporter' }))
    expect(exporter.run).toHaveBeenCalledWith(expect.anything(), { sound: false, format: '1080p', framing: 'whole' })
  })

  it('EMPÊCHE les deux bornes de se croiser', () => {
    setup()
    const debut = screen.getByLabelText('Début') as HTMLInputElement
    const fin = screen.getByLabelText('Fin') as HTMLInputElement
    // Tirer le début au-delà de la fin ne produit pas un intervalle vide : il est borné.
    fireChange(debut, '80')
    fireChange(fin, '40')
    expect(Number(debut.value)).toBeLessThanOrEqual(Number(fin.value))
  })

  it('referme sur « Fermer », sans rien lancer', async () => {
    const user = userEvent.setup()
    const { exporter, onClose } = setup()
    await user.click(screen.getByRole('button', { name: 'Fermer' }))
    expect(onClose).toHaveBeenCalled()
    expect(exporter.run).not.toHaveBeenCalled()
  })
})

describe('ReplayExportDialog — le format du fichier', () => {
  afterEach(() => {
    window.localStorage.removeItem(EXPORT_FORMAT_KEY)
  })

  it('propose 1080p et 720p avec leurs dimensions, 1080p coche par defaut', () => {
    setup()
    expect(screen.getByRole('radiogroup', { name: 'Format' })).toBeInTheDocument()
    expect((screen.getByLabelText('1080p (1920 × 1080)') as HTMLInputElement).checked).toBe(true)
    expect((screen.getByLabelText('720p (1280 × 720)') as HTMLInputElement).checked).toBe(false)
  })

  it('exporte au format choisi, et le RETIENT pour le prochain dialogue', async () => {
    const user = userEvent.setup()
    const { exporter } = setup()
    await user.click(screen.getByLabelText('720p (1280 × 720)'))
    await user.click(screen.getByRole('button', { name: 'Exporter' }))
    expect(exporter.run).toHaveBeenCalledWith(expect.anything(), { sound: true, format: '720p', framing: 'whole' })
    expect(window.localStorage.getItem(EXPORT_FORMAT_KEY)).toBe('720p')
    // Un NOUVEAU dialogue (page rechargee, autre match) repart du choix retenu.
    cleanup()
    const second = setup()
    expect((screen.getByLabelText('720p (1280 × 720)') as HTMLInputElement).checked).toBe(true)
    await user.click(screen.getByRole('button', { name: 'Exporter' }))
    expect(second.exporter.run).toHaveBeenCalledWith(expect.anything(), { sound: true, format: '720p', framing: 'whole' })
  })

  it('une valeur retenue hors catalogue retombe sur 1080p', () => {
    window.localStorage.setItem(EXPORT_FORMAT_KEY, '4k')
    setup()
    expect((screen.getByLabelText('1080p (1920 × 1080)') as HTMLInputElement).checked).toBe(true)
  })

  it('ne se change pas pendant le calcul : le choix n’est plus rendu', () => {
    setup({ state: { phase: 'encode', done: 1, total: 10, pct: 10 } })
    expect(screen.queryByRole('radiogroup')).not.toBeInTheDocument()
  })
})

describe('ReplayExportDialog — le cadrage (D6)', () => {
  it('n’est PAS propose sur une carte a 1x', () => {
    setup({ zoomLevel: 1 })
    expect(screen.queryByRole('radiogroup', { name: 'Cadrage' })).not.toBeInTheDocument()
  })

  it('zoomee : propose la carte entiere, cochee par defaut, et le cadrage actuel avec son palier', async () => {
    const user = userEvent.setup()
    const { exporter } = setup({ zoomLevel: 1.5 })
    expect(screen.getByRole('radiogroup', { name: 'Cadrage' })).toBeInTheDocument()
    expect((screen.getByLabelText('Carte entière') as HTMLInputElement).checked).toBe(true)
    expect(screen.getByLabelText('Cadrage actuel (1,5x)')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Exporter' }))
    expect(exporter.run).toHaveBeenCalledWith(expect.anything(), { sound: true, format: '1080p', framing: 'whole' })
  })

  it('exporte au cadrage actuel quand on le choisit, sans rien retenir', async () => {
    const user = userEvent.setup()
    const cles = () => Array.from({ length: window.localStorage.length }, (_, i) => window.localStorage.key(i)).sort()
    const avant = cles()
    const { exporter } = setup({ zoomLevel: 2 })
    await user.click(screen.getByLabelText('Cadrage actuel (2x)'))
    await user.click(screen.getByRole('button', { name: 'Exporter' }))
    expect(exporter.run).toHaveBeenCalledWith(expect.anything(), { sound: true, format: '1080p', framing: 'current' })
    // NON MEMORISE : aucune cle nouvelle, et un nouveau dialogue repart de la carte entiere.
    expect(cles()).toEqual(avant)
    cleanup()
    setup({ zoomLevel: 2 })
    expect((screen.getByLabelText('Carte entière') as HTMLInputElement).checked).toBe(true)
  })
})

describe('ReplayExportDialog — pendant le calcul', () => {
  const enCours = { phase: 'encode', done: 300, total: 1200, pct: 25 } as const

  it('montre la progression en toutes lettres, et la barre', () => {
    setup({ state: enCours })
    expect(screen.getByText('Image 300 / 1200')).toBeInTheDocument()
    expect(screen.getByRole('progressbar')).toHaveAttribute('aria-valuenow', '25')
  })

  it('n’offre plus QUE l’annulation', () => {
    setup({ state: enCours })
    // Relancer un export pendant un export, ou refermer le dialogue en laissant le calcul
    // tourner sans rien qui le dise : les deux sont retirés.
    expect(screen.queryByRole('button', { name: 'Exporter' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Fermer' })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Annuler' })).toBeInTheDocument()
  })

  it('annule au clic', async () => {
    const user = userEvent.setup()
    const { exporter } = setup({ state: enCours })
    await user.click(screen.getByRole('button', { name: 'Annuler' }))
    expect(exporter.cancel).toHaveBeenCalled()
  })
})

/** Un `<input type="range">` ne se pilote pas au clavier dans jsdom : on pose la valeur. */
function fireChange(el: HTMLInputElement, value: string) {
  const setter = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')?.set
  setter?.call(el, value)
  el.dispatchEvent(new Event('change', { bubbles: true }))
}

describe('ReplayExportDialog — les phases du calcul', () => {
  it('en PRÉPARATION, dit ce qu’on fait au lieu de compter des images', () => {
    setup({ state: { phase: 'prepare', done: 0, total: 1200, pct: 0 } })
    // C'est la phase qui affichait « Image 0 / 1200 » sans bouger pendant plusieurs secondes.
    expect(screen.getByText('Préparation du son et des images…')).toBeInTheDocument()
    expect(screen.queryByText(/Image 0/)).not.toBeInTheDocument()
    // Barre INDÉTERMINÉE : rien n'est encore encodé, aucun pourcentage n'est vrai.
    expect(screen.getByRole('progressbar')).not.toHaveAttribute('aria-valuenow')
  })

  it('en ENCODAGE, compte les images ET annonce le temps restant', () => {
    setup({ state: { phase: 'encode', done: 300, total: 1200, pct: 25, etaMs: 80_000 } })
    expect(screen.getByText('Image 300 / 1200')).toBeInTheDocument()
    expect(screen.getByText(/environ 1:20 restantes/)).toBeInTheDocument()
  })

  it('ne montre PAS une estimation qui ne dit rien', () => {
    // « environ 0:00 restantes » se lit « c'est fini » alors que le calcul tourne encore.
    setup({ state: { phase: 'encode', done: 300, total: 1200, pct: 25, etaMs: 900 } })
    expect(screen.queryByText(/restantes/)).not.toBeInTheDocument()
    expect(screen.getByRole('progressbar')).toHaveAttribute('aria-valuenow', '25')
  })

  it('sans estimation fiable, n’en invente pas', () => {
    setup({ state: { phase: 'encode', done: 3, total: 1200, pct: 0.25 } })
    expect(screen.queryByText(/restantes/)).not.toBeInTheDocument()
  })

  it('prévient du défilement du terrain PENDANT le calcul, pas avant', () => {
    setup({ state: { phase: 'encode', done: 300, total: 1200, pct: 25 } })
    // La clé s'appelle `exportRunningHint` et s'affichait pourtant dans le formulaire.
    expect(screen.getByText(/Le terrain défile dans le cadre de la vidéo pendant le calcul/)).toBeInTheDocument()
  })

  it('à la FIN, nomme le fichier déposé et rend le formulaire', () => {
    setup({ state: { phase: 'done', done: 10, total: 10, pct: 100, filename: 'rejeu-m-0m00s-5m00s.mp4' } })
    expect(screen.getByText('Fichier déposé : rejeu-m-0m00s-5m00s.mp4')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Exporter' })).toBeInTheDocument()
  })
})
