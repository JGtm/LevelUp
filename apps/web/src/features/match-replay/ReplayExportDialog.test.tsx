/**
 * ReplayExportDialog.test.tsx — LES DEUX BORNES, LA CASE SON, ET CE QUI SE PASSE PENDANT.
 *
 * Ce qui se verrouille ici est ce qu'un utilisateur peut CASSER : croiser les deux curseurs,
 * lancer un export muet en croyant l'avoir demandé sonore, ou perdre le bouton « Annuler » au
 * milieu d'un calcul de plusieurs minutes.
 */
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'

import { ReplayExportDialog } from './ReplayExportDialog'
import type { ReplayExport } from './useReplayExport'

function makeExport(over: Partial<ReplayExport> = {}): ReplayExport {
  return {
    supported: true,
    state: { running: false, done: 0, total: 0, pct: 0 },
    defaultBounds: () => ({ startFrame: 0, endFrame: 100 }),
    run: vi.fn(async () => {}),
    cancel: vi.fn(),
    clockOf: (f) => `frame:${f}`,
    lengthClock: (b) => `${b.startFrame}-${b.endFrame}`,
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
    expect(exporter.run).toHaveBeenCalledWith({ startFrame: 0, endFrame: 100 }, { sound: true })
  })

  it('exporte muet quand la case est décochée', async () => {
    const user = userEvent.setup()
    const { exporter } = setup()
    await user.click(screen.getByLabelText('Inclure le son'))
    await user.click(screen.getByRole('button', { name: 'Exporter' }))
    expect(exporter.run).toHaveBeenCalledWith(expect.anything(), { sound: false })
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

describe('ReplayExportDialog — pendant le calcul', () => {
  const enCours = { running: true, done: 300, total: 1200, pct: 25 }

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
