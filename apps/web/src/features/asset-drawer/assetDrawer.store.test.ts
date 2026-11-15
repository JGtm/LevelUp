/**
 * Store du volet des référentiels : mode épinglé conservé à la fermeture, état « ouvert » mémorisé
 * seulement s'il est épinglé, ancien état « ouvert » repris comme épinglé (version 1 du format).
 */
import { beforeEach, describe, expect, it } from 'vitest'

import { useAssetDrawerStore } from './assetDrawer.store'

const STORAGE_KEY = 'levelup-asset-drawer'

function stored(): { state: Record<string, unknown>; version: number } {
  return JSON.parse(window.localStorage.getItem(STORAGE_KEY) ?? 'null')
}

beforeEach(() => {
  window.localStorage.clear()
  useAssetDrawerStore.setState({ isOpen: false, pinned: false, activeTab: 'maps', search: '' })
})

describe('assetDrawer.store — mode épinglé', () => {
  it('fermer ne change pas le mode ; rouvrir le retrouve', () => {
    const s = useAssetDrawerStore.getState()
    s.setPinned(true)
    s.open()
    useAssetDrawerStore.getState().close()
    expect(useAssetDrawerStore.getState()).toMatchObject({ isOpen: false, pinned: true })
    useAssetDrawerStore.getState().open()
    expect(useAssetDrawerStore.getState()).toMatchObject({ isOpen: true, pinned: true })
  })

  it('épinglé : l’état ouvert et le mode sont mémorisés', () => {
    useAssetDrawerStore.setState({ pinned: true, isOpen: true })
    expect(stored()).toMatchObject({ version: 1, state: { isOpen: true, pinned: true } })
  })

  it('non épinglé : l’état ouvert n’est pas mémorisé (un volet superposé se rouvre fermé)', () => {
    useAssetDrawerStore.setState({ pinned: false, isOpen: true })
    expect(stored().state).toMatchObject({ isOpen: false, pinned: false })
  })

  it('épinglé mais fermé : le mode reste mémorisé', () => {
    useAssetDrawerStore.setState({ pinned: true, isOpen: false })
    expect(stored().state).toMatchObject({ isOpen: false, pinned: true })
  })
})

describe('assetDrawer.store — migration du format', () => {
  it('un ancien état ouvert (version 0) devient épinglé et ouvert', async () => {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify({ state: { isOpen: true, activeTab: 'weapons' }, version: 0 }))
    await useAssetDrawerStore.persist.rehydrate()
    expect(useAssetDrawerStore.getState()).toMatchObject({ isOpen: true, pinned: true, activeTab: 'weapons' })
  })

  it('un ancien état fermé (version 0) reste non épinglé', async () => {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify({ state: { isOpen: false, activeTab: 'medals' }, version: 0 }))
    await useAssetDrawerStore.persist.rehydrate()
    expect(useAssetDrawerStore.getState()).toMatchObject({ isOpen: false, pinned: false, activeTab: 'medals' })
  })

  it('un état de la version 1 est repris tel quel', async () => {
    window.localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({ state: { isOpen: false, pinned: true, activeTab: 'maps' }, version: 1 }),
    )
    await useAssetDrawerStore.persist.rehydrate()
    expect(useAssetDrawerStore.getState()).toMatchObject({ isOpen: false, pinned: true })
  })
})
