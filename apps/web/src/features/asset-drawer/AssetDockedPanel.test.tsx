/**
 * Tests — volet des référentiels ancré (application de bureau) : il pousse le contenu au lieu de le
 * recouvrir, se ferme au bouton, à Échap (focus rendu au bouton d'ouverture) et se bascule au raccourci,
 * se redimensionne à la souris et au clavier, mémorise sa largeur et son état.
 */
import { beforeEach, describe, expect, it } from 'vitest'
import { act, fireEvent, screen } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'
import { useAssetDrawerStore } from './assetDrawer.store'
import { AssetDockedPanel } from './AssetDockedPanel'
import { ASSET_PANEL_TRIGGER_ID } from './assetPanel'
import { PANEL_DEFAULT_WIDTH, PANEL_WIDTH_STORAGE_KEY, loadPanelWidth } from './panelWidth'

beforeEach(() => {
  window.localStorage.clear()
  useAppShellStore.setState({ locale: 'fr', currentTitleSlug: 'halo_infinite' })
  useAssetDrawerStore.setState({
    isOpen: false,
    activeTab: 'maps',
    search: '',
    panelWidth: PANEL_DEFAULT_WIDTH,
    focusToken: 0,
  })
  Object.defineProperty(window, 'innerWidth', { configurable: true, value: 1600 })
  server.use(http.get('/api/v1/assets/:titleSlug/:kind', () => HttpResponse.json([])))
})

function panel() {
  return screen.getByRole('complementary', { name: 'Référentiels' })
}

function handle() {
  return screen.getByRole('separator', { name: 'Redimensionner le volet des référentiels' })
}

function ctrlShiftL() {
  fireEvent.keyDown(document, { key: 'L', ctrlKey: true, shiftKey: true })
}

describe('AssetDockedPanel — ouverture et fermeture', () => {
  it('fermé : rien n’est rendu', () => {
    renderWithProviders(<AssetDockedPanel />)
    expect(screen.queryByRole('complementary')).not.toBeInTheDocument()
  })

  it('ouvert : un volet ancré (dans le flux, largeur propre, sans voile) avec la recherche en tête', () => {
    useAssetDrawerStore.setState({ isOpen: true })
    const { container } = renderWithProviders(<AssetDockedPanel />)
    const aside = panel()
    // Il pousse le contenu : dans le flux (pas `fixed`), ne se réduit pas, largeur posée en style.
    expect(aside.className).not.toMatch(/\bfixed\b/)
    expect(aside.className).toMatch(/\bshrink-0\b/)
    expect(aside).toHaveStyle({ width: `${PANEL_DEFAULT_WIDTH}px` })
    // Aucun voile qui recouvre la page.
    expect(container.querySelector('.inset-0')).toBeNull()
    // Le champ de recherche précède les onglets.
    const search = screen.getByRole('searchbox')
    const tabs = screen.getByRole('tablist')
    expect(search.compareDocumentPosition(tabs) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
  })

  it('le bouton de fermeture ferme le volet', () => {
    useAssetDrawerStore.setState({ isOpen: true })
    renderWithProviders(<AssetDockedPanel />)
    fireEvent.click(screen.getByRole('button', { name: 'Fermer le référentiel visuel' }))
    expect(useAssetDrawerStore.getState().isOpen).toBe(false)
    expect(screen.queryByRole('complementary')).not.toBeInTheDocument()
  })

  it('le raccourci Ctrl+Maj+L ouvre puis ferme, et prend la main sur le défaut du moteur', () => {
    renderWithProviders(<AssetDockedPanel />)
    ctrlShiftL()
    expect(useAssetDrawerStore.getState().isOpen).toBe(true)
    expect(panel()).toBeInTheDocument()
    const ev = new KeyboardEvent('keydown', { key: 'L', ctrlKey: true, shiftKey: true, cancelable: true })
    act(() => {
      document.dispatchEvent(ev)
    })
    expect(ev.defaultPrevented).toBe(true)
    expect(useAssetDrawerStore.getState().isOpen).toBe(false)
  })

  it('d’autres combinaisons ne font rien (Ctrl+L, Ctrl+Maj+R, Ctrl+Alt+Maj+L)', () => {
    renderWithProviders(<AssetDockedPanel />)
    fireEvent.keyDown(document, { key: 'l', ctrlKey: true })
    fireEvent.keyDown(document, { key: 'R', ctrlKey: true, shiftKey: true })
    fireEvent.keyDown(document, { key: 'l', ctrlKey: true, shiftKey: true, altKey: true })
    expect(useAssetDrawerStore.getState().isOpen).toBe(false)
  })

  it('une ouverture demandée met le focus dans la recherche ; un volet rouvert au démarrage le laisse', () => {
    useAssetDrawerStore.setState({ isOpen: true })
    renderWithProviders(<AssetDockedPanel />)
    expect(screen.getByRole('searchbox')).not.toHaveFocus()
    act(() => useAssetDrawerStore.getState().close())
    act(() => useAssetDrawerStore.getState().open())
    expect(screen.getByRole('searchbox')).toHaveFocus()
  })

  it('Échap dans le volet le ferme et rend le focus au bouton d’ouverture', () => {
    useAssetDrawerStore.setState({ isOpen: true })
    renderWithProviders(
      <>
        <button type="button" id={ASSET_PANEL_TRIGGER_ID}>
          Ouvrir
        </button>
        <AssetDockedPanel />
      </>,
    )
    screen.getByRole('searchbox').focus()
    fireEvent.keyDown(screen.getByRole('searchbox'), { key: 'Escape' })
    expect(useAssetDrawerStore.getState().isOpen).toBe(false)
    expect(screen.getByRole('button', { name: 'Ouvrir' })).toHaveFocus()
  })

  it('Échap hors du volet ne le ferme pas', () => {
    useAssetDrawerStore.setState({ isOpen: true })
    renderWithProviders(<AssetDockedPanel />)
    fireEvent.keyDown(document.body, { key: 'Escape' })
    expect(useAssetDrawerStore.getState().isOpen).toBe(true)
  })
})

describe('AssetDockedPanel — redimensionnement', () => {
  it('la poignée expose ses bornes et la largeur courante', () => {
    useAssetDrawerStore.setState({ isOpen: true })
    renderWithProviders(<AssetDockedPanel />)
    expect(handle()).toHaveAttribute('aria-valuemin', '280')
    expect(handle()).toHaveAttribute('aria-valuemax', '640')
    expect(handle()).toHaveAttribute('aria-valuenow', String(PANEL_DEFAULT_WIDTH))
  })

  it('les flèches élargissent (gauche) et rétrécissent (droite), Maj pour le grand pas ; la largeur est mémorisée', () => {
    useAssetDrawerStore.setState({ isOpen: true })
    renderWithProviders(<AssetDockedPanel />)
    fireEvent.keyDown(handle(), { key: 'ArrowLeft' })
    expect(panel()).toHaveStyle({ width: `${PANEL_DEFAULT_WIDTH + 16}px` })
    fireEvent.keyDown(handle(), { key: 'ArrowLeft', shiftKey: true })
    expect(panel()).toHaveStyle({ width: `${PANEL_DEFAULT_WIDTH + 16 + 64}px` })
    fireEvent.keyDown(handle(), { key: 'ArrowRight' })
    expect(panel()).toHaveStyle({ width: `${PANEL_DEFAULT_WIDTH + 64}px` })
    expect(window.localStorage.getItem(PANEL_WIDTH_STORAGE_KEY)).toBe(String(PANEL_DEFAULT_WIDTH + 64))
    expect(loadPanelWidth()).toBe(PANEL_DEFAULT_WIDTH + 64)
  })

  it('Début et Fin vont aux bornes, qui ne se franchissent pas', () => {
    useAssetDrawerStore.setState({ isOpen: true })
    renderWithProviders(<AssetDockedPanel />)
    fireEvent.keyDown(handle(), { key: 'End' })
    expect(panel()).toHaveStyle({ width: '640px' })
    fireEvent.keyDown(handle(), { key: 'ArrowLeft' })
    expect(panel()).toHaveStyle({ width: '640px' })
    fireEvent.keyDown(handle(), { key: 'Home' })
    expect(panel()).toHaveStyle({ width: '280px' })
    fireEvent.keyDown(handle(), { key: 'ArrowRight' })
    expect(panel()).toHaveStyle({ width: '280px' })
  })

  it('le glissement de la poignée fixe la largeur d’après le bord droit du volet', () => {
    useAssetDrawerStore.setState({ isOpen: true })
    renderWithProviders(<AssetDockedPanel />)
    panel().getBoundingClientRect = () => ({ right: 1600 }) as DOMRect
    const h = handle()
    h.setPointerCapture = () => {}
    h.releasePointerCapture = () => {}
    h.hasPointerCapture = () => true
    fireEvent.pointerDown(h, { pointerId: 1, clientX: 1240 })
    fireEvent.pointerMove(h, { pointerId: 1, clientX: 1100 })
    expect(panel()).toHaveStyle({ width: '500px' })
    fireEvent.pointerUp(h, { pointerId: 1 })
    fireEvent.pointerMove(h, { pointerId: 1, clientX: 1500 })
    expect(panel()).toHaveStyle({ width: '500px' })
  })

  it('une fenêtre étroite réduit la largeur effective sans toucher à la largeur mémorisée', () => {
    Object.defineProperty(window, 'innerWidth', { configurable: true, value: 1024 })
    useAssetDrawerStore.setState({ isOpen: true, panelWidth: 600 })
    renderWithProviders(<AssetDockedPanel />)
    expect(panel()).toHaveStyle({ width: '304px' })
    expect(useAssetDrawerStore.getState().panelWidth).toBe(600)
  })
})
