/**
 * IssueSections — formulaires de traduction : une saisie par langue de `config/locales.toml`
 * (KNOWN_LOCALES), étiquetée du nom de la langue ; l'envoi construit `names: { langue: texte }`.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import { languageName } from '@/lib/formatters'
import { formatMessage } from '@/lib/i18n/format'
import { adminManifest } from '@/lib/i18n/generated/admin'
import { KNOWN_LOCALES } from '@/lib/i18n/locale'
import type { AdminDataQualityIssues } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

const modeMutate = vi.fn()
const assetMutate = vi.fn()
let issues: AdminDataQualityIssues

vi.mock('./queries', () => ({
  DATA_QUALITY_LOCALE: 'fr',
  useDataQualityIssues: () => ({ data: issues }),
}))
vi.mock('./mutations', () => ({
  useResolveModeTranslation: () => ({ mutate: modeMutate, isPending: false }),
  useResolveAssetTranslation: () => ({ mutate: assetMutate, isPending: false }),
}))

import { RawAssetsSection, UntranslatedModesSection } from './IssueSections'

const tr = (key: Parameters<typeof formatMessage<typeof adminManifest>>[1]) =>
  formatMessage(adminManifest, key, 'fr')

/** Libellé attendu de la saisie d'une langue dans l'interface française. */
const inputLabel = (code: (typeof KNOWN_LOCALES)[number]) =>
  formatMessage(adminManifest, 'admin.dq.form_name_in', 'fr', { language: languageName(code, 'fr') })

beforeEach(() => {
  useAppShellStore.setState({ locale: 'fr' })
  modeMutate.mockReset()
  assetMutate.mockReset()
})

afterEach(() => {
  useAppShellStore.setState({ currentPlayer: null, availablePlayers: [], locale: 'fr' })
})

function openForm(buttonKey: 'admin.dq.translate_btn' | 'admin.dq.resolve_btn') {
  fireEvent.click(screen.getByRole('button', { name: tr(buttonKey) }))
}

describe('formulaire de traduction d\'un mode', () => {
  beforeEach(() => {
    issues = {
      title_slug: 'halo_infinite',
      kind: 'untranslated_modes',
      locale: 'fr',
      total: 1,
      items: [{ kind: 'untranslated_mode', id: 'Husky Raid', occurrences: 3 }],
    } as AdminDataQualityIssues
  })

  it('affiche une saisie par langue listée, étiquetée du nom de la langue', () => {
    render(<UntranslatedModesSection />)
    openForm('admin.dq.translate_btn')
    const inputs = screen.getAllByRole('textbox')
    expect(inputs).toHaveLength(KNOWN_LOCALES.length)
    for (const code of KNOWN_LOCALES) {
      expect(screen.getByLabelText(inputLabel(code))).toBeInTheDocument()
    }
  })

  it('envoie names avec les seules langues renseignées (noms rognés)', () => {
    render(<UntranslatedModesSection />)
    openForm('admin.dq.translate_btn')
    fireEvent.change(screen.getByLabelText(inputLabel('fr')), { target: { value: '  Raid Husky ' } })
    fireEvent.click(screen.getByRole('button', { name: tr('admin.dq.form_submit') }))
    expect(modeMutate).toHaveBeenCalledTimes(1)
    expect(modeMutate.mock.calls[0][0]).toEqual({ mode_key: 'Husky Raid', names: { fr: 'Raid Husky' } })

    // Deux langues : une entrée par langue.
    modeMutate.mockReset()
    fireEvent.change(screen.getByLabelText(inputLabel('en')), { target: { value: 'Husky Raid CTF' } })
    fireEvent.click(screen.getByRole('button', { name: tr('admin.dq.form_submit') }))
    expect(modeMutate.mock.calls[0][0]).toEqual({
      mode_key: 'Husky Raid',
      names: { fr: 'Raid Husky', en: 'Husky Raid CTF' },
    })
  })

  it('désactive l\'envoi tant qu\'aucune langue n\'est renseignée', () => {
    render(<UntranslatedModesSection />)
    openForm('admin.dq.translate_btn')
    expect(screen.getByRole('button', { name: tr('admin.dq.form_submit') })).toBeDisabled()
    expect(modeMutate).not.toHaveBeenCalled()
  })
})

describe('formulaire de traduction d\'un asset', () => {
  beforeEach(() => {
    issues = {
      title_slug: 'halo_infinite',
      kind: 'raw_uuids',
      locale: 'fr',
      total: 1,
      items: [{ kind: 'raw_uuid', id: 'pl-9', asset_kind: 'playlist', occurrences: 2 }],
    } as AdminDataQualityIssues
  })

  it('envoie asset_kind, asset_id et names construit par langue', () => {
    render(<RawAssetsSection />)
    openForm('admin.dq.resolve_btn')
    const form = screen.getAllByRole('textbox')
    expect(form).toHaveLength(KNOWN_LOCALES.length)
    for (const code of KNOWN_LOCALES) {
      fireEvent.change(screen.getByLabelText(inputLabel(code)), {
        target: { value: `nom ${code}` },
      })
    }
    fireEvent.click(screen.getByRole('button', { name: tr('admin.dq.form_submit') }))
    expect(assetMutate).toHaveBeenCalledTimes(1)
    const expected: Record<string, string> = {}
    for (const code of KNOWN_LOCALES) expected[code] = `nom ${code}`
    expect(assetMutate.mock.calls[0][0]).toEqual({
      asset_kind: 'playlist',
      asset_id: 'pl-9',
      names: expected,
    })
  })
})
