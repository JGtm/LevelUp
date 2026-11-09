/**
 * Tests — volet des référentiels ancré (application de bureau) : toujours monté, il glisse depuis le bord
 * droit ; épinglé il pousse le contenu, sinon il se superpose sous un voile qui le ferme au clic ; il se
 * ferme au bouton, à Échap (focus rendu au bouton d’ouverture), se bascule au raccourci, se redimensionne
 * à la souris et au clavier, mémorise sa largeur, son état et son mode.
 */
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { act, fireEvent, screen } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'
import { endLayoutTransition, isLayoutSettling } from '@/lib/layout/layoutSettling'
import { useAssetDrawerStore } from './assetDrawer.store'
import { AssetDockedPanel } from './AssetDockedPanel'
import { ASSET_PANEL_TRIGGER_ID } from './assetPanel'
import { PANEL_DEFAULT_WIDTH, PANEL_WIDTH_STORAGE_KEY, loadPanelWidth } from './panelWidth'

beforeEach(() => {
  window.localStorage.clear()
  endLayoutTransition()
  useAppShellStore.setState({ locale: 'fr', currentTitleSlug: 'halo_infinite' })
  useAssetDrawerStore.setState({
    isOpen: false,
    pinned: false,
    activeTab: 'maps',
    search: '',
    panelWidth: PANEL_DEFAULT_WIDTH,
    focusToken: 0,
  })
  Object.defineProperty(window, 'innerWidth', { configurable: true, value: 1600 })
  server.use(http.get('/api/v1/assets/:titleSlug/:kind', () => HttpResponse.json([])))
})
afterEach(() => {
  endLayoutTransition()
})

function panel() {
  return screen.getByRole('complementary', { name: 'Référentiels' })
}

// Masqué (aria-hidden), le panneau n'a plus de nom accessible : on le prend par son rôle seul.
function hiddenPanel() {
  return screen.getByRole('complementary', { hidden: true })
}

function slot() {
  return screen.getByTestId('asset-panel-slot')
}

function handle() {
  return screen.getByRole('separator', { name: 'Redimensionner le volet des référentiels' })
}

function pinButton() {
  return screen.getByRole('button', { name: 'Épingler le volet' })
}

function ctrlShiftL() {
  fireEvent.keyDown(document, { key: 'L', ctrlKey: true, shiftKey: true })
}

describe('AssetDockedPanel — ouverture, fermeture et modes', () => {
  it('fermé : toujours monté mais hors de l’écran, inerte, masqué aux lecteurs d’écran, sans place ni voile', () => {
    renderWithProviders(<AssetDockedPanel />)
    expect(screen.queryByRole('complementary')).not.toBeInTheDocument()
    const aside = hiddenPanel()
    expect(aside).toHaveAttribute('aria-hidden', 'true')
    expect(aside).toHaveAttribute('inert')
    expect(aside.className).toMatch(/\btranslate-x-full\b/)
    expect(slot()).toHaveStyle({ width: '0px' })
    expect(screen.queryByTestId('asset-panel-scrim')).not.toBeInTheDocument()
  })

  it('ouvert et épinglé : il pousse le contenu (fente à sa largeur, panneau hors superposition), sans voile, recherche en tête', () => {
    useAssetDrawerStore.setState({ isOpen: true, pinned: true })
    renderWithProviders(<AssetDockedPanel />)
    const aside = panel()
    expect(aside).not.toHaveAttribute('inert')
    expect(aside.className).toMatch(/\btranslate-x-0\b/)
    expect(aside.className).not.toMatch(/\bfixed\b/)
    expect(aside.className).not.toMatch(/\bz-50\b/)
    expect(slot().className).toMatch(/\bshrink-0\b/)
    expect(slot()).toHaveStyle({ width: `${PANEL_DEFAULT_WIDTH}px` })
    expect(aside).toHaveStyle({ width: `${PANEL_DEFAULT_WIDTH}px` })
    expect(screen.queryByTestId('asset-panel-scrim')).not.toBeInTheDocument()
    // Le champ de recherche précède les onglets.
    const search = screen.getByRole('searchbox')
    const tabs = screen.getByRole('tablist')
    expect(search.compareDocumentPosition(tabs) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
  })

  it('ouvert et non épinglé : il se superpose (fente sans largeur) sous un voile transparent plein écran', () => {
    useAssetDrawerStore.setState({ isOpen: true, pinned: false })
    renderWithProviders(<AssetDockedPanel />)
    expect(slot()).toHaveStyle({ width: '0px' })
    expect(panel()).toHaveStyle({ width: `${PANEL_DEFAULT_WIDTH}px` })
    expect(panel().className).toMatch(/\bz-50\b/)
    const scrim = screen.getByTestId('asset-panel-scrim')
    expect(scrim.className).toMatch(/\bfixed\b/)
    expect(scrim.className).toMatch(/\binset-0\b/)
  })

  it('un clic sur le voile ferme le volet superposé', () => {
    useAssetDrawerStore.setState({ isOpen: true, pinned: false })
    renderWithProviders(<AssetDockedPanel />)
    fireEvent.click(screen.getByTestId('asset-panel-scrim'))
    expect(useAssetDrawerStore.getState().isOpen).toBe(false)
    expect(screen.queryByTestId('asset-panel-scrim')).not.toBeInTheDocument()
    expect(screen.queryByRole('complementary')).not.toBeInTheDocument()
  })

  it('le bouton d’épinglage bascule le mode : la fente prend la largeur du volet, le voile disparaît', () => {
    useAssetDrawerStore.setState({ isOpen: true, pinned: false })
    renderWithProviders(<AssetDockedPanel />)
    expect(pinButton()).toHaveAttribute('aria-pressed', 'false')
    fireEvent.click(pinButton())
    expect(useAssetDrawerStore.getState().pinned).toBe(true)
    expect(pinButton()).toHaveAttribute('aria-pressed', 'true')
    expect(slot()).toHaveStyle({ width: `${PANEL_DEFAULT_WIDTH}px` })
    expect(screen.queryByTestId('asset-panel-scrim')).not.toBeInTheDocument()
    fireEvent.click(pinButton())
    expect(useAssetDrawerStore.getState().pinned).toBe(false)
    expect(slot()).toHaveStyle({ width: '0px' })
    expect(screen.getByTestId('asset-panel-scrim')).toBeInTheDocument()
  })

  it('fermer un volet épinglé conserve le mode : rouvert, il pousse encore le contenu', () => {
    useAssetDrawerStore.setState({ isOpen: true, pinned: true })
    renderWithProviders(<AssetDockedPanel />)
    fireEvent.click(screen.getByRole('button', { name: 'Fermer le référentiel visuel' }))
    expect(useAssetDrawerStore.getState()).toMatchObject({ isOpen: false, pinned: true })
    expect(slot()).toHaveStyle({ width: '0px' })
    act(() => useAssetDrawerStore.getState().open())
    expect(slot()).toHaveStyle({ width: `${PANEL_DEFAULT_WIDTH}px` })
    expect(screen.queryByTestId('asset-panel-scrim')).not.toBeInTheDocument()
  })

  it('le bouton de fermeture ferme le volet : il redevient masqué et inerte', () => {
    useAssetDrawerStore.setState({ isOpen: true })
    renderWithProviders(<AssetDockedPanel />)
    fireEvent.click(screen.getByRole('button', { name: 'Fermer le référentiel visuel' }))
    expect(useAssetDrawerStore.getState().isOpen).toBe(false)
    expect(screen.queryByRole('complementary')).not.toBeInTheDocument()
    expect(hiddenPanel()).toHaveAttribute('inert')
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

describe('AssetDockedPanel — transition de mise en page', () => {
  it('au montage : aucune transition déclarée', () => {
    useAssetDrawerStore.setState({ isOpen: true, pinned: true })
    renderWithProviders(<AssetDockedPanel />)
    expect(isLayoutSettling()).toBe(false)
  })

  it('ouvrir, fermer ou épingler change la place prise : transition déclarée jusqu’à la fin de l’animation de la fente', () => {
    renderWithProviders(<AssetDockedPanel />)
    // Ouvert non épinglé : la place prise reste nulle, rien à attendre.
    act(() => useAssetDrawerStore.getState().open())
    expect(isLayoutSettling()).toBe(false)
    act(() => useAssetDrawerStore.getState().setPinned(true))
    expect(isLayoutSettling()).toBe(true)
    // L’animation du panneau lui-même (transform) ne clôt pas la transition de la fente.
    fireEvent.transitionEnd(panel())
    expect(isLayoutSettling()).toBe(true)
    fireEvent.transitionEnd(slot())
    expect(isLayoutSettling()).toBe(false)
    act(() => useAssetDrawerStore.getState().close())
    expect(isLayoutSettling()).toBe(true)
  })

  it('pendant le glissement de la poignée : la fente ne s’anime pas, aucune transition déclarée', () => {
    useAssetDrawerStore.setState({ isOpen: true, pinned: true })
    renderWithProviders(<AssetDockedPanel />)
    panel().getBoundingClientRect = () => ({ right: 1600 }) as DOMRect
    const h = handle()
    h.setPointerCapture = () => {}
    h.releasePointerCapture = () => {}
    h.hasPointerCapture = () => true
    fireEvent.pointerDown(h, { pointerId: 1, clientX: 1240 })
    fireEvent.pointerMove(h, { pointerId: 1, clientX: 1100 })
    expect(slot()).toHaveStyle({ width: '500px' })
    expect(slot().className).not.toMatch(/transition-\[width\]/)
    expect(isLayoutSettling()).toBe(false)
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
