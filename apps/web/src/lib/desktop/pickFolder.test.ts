import { afterEach, describe, expect, it, vi } from 'vitest'
import { PICK_FOLDER_COMMAND, pickFolder } from './pickFolder'

function stubShell(invoke: unknown) {
  vi.stubGlobal('__TAURI_INTERNALS__', { invoke })
}

describe('pickFolder', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  it('appelle la commande de la coquille avec la valeur du champ et rend le chemin tel quel', async () => {
    const invoke = vi.fn().mockResolvedValue('  D:/un chemin/opaque \\ avec  espaces ')
    stubShell(invoke)

    await expect(pickFolder('E:/avant')).resolves.toBe('  D:/un chemin/opaque \\ avec  espaces ')

    expect(invoke).toHaveBeenCalledTimes(1)
    expect(invoke).toHaveBeenCalledWith(PICK_FOLDER_COMMAND, { initial: 'E:/avant' })
    expect(PICK_FOLDER_COMMAND).toBe('pick_folder')
  })

  it('rend null quand l’utilisateur annule (la coquille répond null), sans rien journaliser', async () => {
    stubShell(vi.fn().mockResolvedValue(null))
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
    const error = vi.spyOn(console, 'error').mockImplementation(() => {})

    await expect(pickFolder('')).resolves.toBeNull()

    expect(warn).not.toHaveBeenCalled()
    expect(error).not.toHaveBeenCalled()
  })

  it('rend null et journalise quand la coquille refuse ou échoue : le champ reste inchangé', async () => {
    stubShell(vi.fn().mockRejectedValue('pick_folder not allowed on origin'))
    const error = vi.spyOn(console, 'error').mockImplementation(() => {})

    await expect(pickFolder('x')).resolves.toBeNull()

    expect(error).toHaveBeenCalledTimes(1)
    expect(String(error.mock.calls[0]?.[0])).toContain('pickFolder')
  })

  it('rend null et journalise quand le pont est absent (page ouverte hors de l’application)', async () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})

    await expect(pickFolder('x')).resolves.toBeNull()

    expect(warn).toHaveBeenCalledTimes(1)
  })

  it('ignore une réponse qui n’est pas un chemin (vide, nombre, objet)', async () => {
    for (const answer of ['', 42, { path: 'x' }, undefined]) {
      stubShell(vi.fn().mockResolvedValue(answer))
      await expect(pickFolder('x')).resolves.toBeNull()
    }
  })

  it('ne lit pas un pont dont invoke n’est pas une fonction', async () => {
    stubShell('pas une fonction')
    vi.spyOn(console, 'warn').mockImplementation(() => {})
    await expect(pickFolder('x')).resolves.toBeNull()
  })
})
