/**
 * Tests — SessionMultiSelect.
 *
 * Couvre : label trigger, ouverture panel, filtre fuzzy, filtre date,
 * toggle checkboxes, sélection tout/rien, validation différée, fermeture
 * click-outside, locale EN.
 */
import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { SessionMultiSelect } from './SessionMultiSelect'
import type { SessionLabelEntry } from '@/lib/api/types'
import type { ManifestLocale } from '@/lib/i18n/format'

// ─── Fixtures ────────────────────────────────────────────────────────────────

// Timestamps à minuit UTC pour éviter les ambiguïtés dans les comparaisons
// date-only (new Date('YYYY-MM-DD') = minuit UTC).
const SESSIONS: SessionLabelEntry[] = [
  {
    label: 'Session Ranked A',
    started_at: '2026-04-20T00:00:00Z',
    ended_at:   '2026-04-20T02:00:00Z',
  },
  {
    label: 'Session Casual B',
    started_at: '2026-04-21T00:00:00Z',
    ended_at:   '2026-04-21T02:00:00Z',
  },
  {
    label: 'Session Ranked C',
    started_at: '2026-04-22T00:00:00Z',
    ended_at:   '2026-04-22T02:00:00Z',
  },
]

function setup(
  selected: string[] = [],
  onChange = vi.fn(),
  locale: ManifestLocale = 'fr',
) {
  const utils = render(
    <SessionMultiSelect
      sessions={SESSIONS}
      selected={selected}
      onChange={onChange}
      locale={locale}
    />,
  )
  const trigger = () => screen.getByRole('button', { name: /session/i })
  const openPanel = () => fireEvent.click(trigger())
  return { ...utils, onChange, trigger, openPanel }
}

// ─── Tests ───────────────────────────────────────────────────────────────────

describe('SessionMultiSelect — label du bouton déclencheur', () => {
  it('affiche "Toutes les sessions" quand aucune sélection (fr)', () => {
    setup()
    expect(screen.getByRole('button', { name: /Toutes les sessions/i })).toBeTruthy()
  })

  it('affiche le compte quand des sessions sont sélectionnées (fr)', () => {
    setup(['Session Ranked A', 'Session Casual B'])
    expect(screen.getByRole('button', { name: /2 sessions/i })).toBeTruthy()
  })

  it('affiche "All sessions" en locale EN', () => {
    setup([], vi.fn(), 'en')
    expect(screen.getByRole('button', { name: /All sessions/i })).toBeTruthy()
  })

  it('affiche le placeholder fourni quand vide', () => {
    render(
      <SessionMultiSelect
        sessions={SESSIONS}
        selected={[]}
        onChange={vi.fn()}
        locale="fr"
        placeholder="Choisir une session…"
      />,
    )
    expect(screen.getByRole('button', { name: /Choisir une session/i })).toBeTruthy()
  })
})

describe('SessionMultiSelect — ouverture et fermeture du panel', () => {
  it('ouvre le dropdown au clic sur le bouton', () => {
    const { openPanel } = setup()
    expect(screen.queryByPlaceholderText(/Rechercher/i)).toBeNull()
    openPanel()
    expect(screen.getByPlaceholderText(/Rechercher/i)).toBeTruthy()
  })

  it("affiche toutes les sessions à l'ouverture", () => {
    const { openPanel } = setup()
    openPanel()
    SESSIONS.forEach((s) => {
      expect(screen.getByText(s.label)).toBeTruthy()
    })
  })

  it('ferme sans propager au mousedown en dehors du composant', () => {
    const { onChange, openPanel } = setup()
    openPanel()
    fireEvent.mouseDown(document.body)
    expect(screen.queryByPlaceholderText(/Rechercher/i)).toBeNull()
    expect(onChange).not.toHaveBeenCalled()
  })
})

describe('SessionMultiSelect — filtre fuzzy', () => {
  it('filtre la liste par la valeur de recherche textuelle', () => {
    const { openPanel } = setup()
    openPanel()
    const searchInput = screen.getByPlaceholderText(/Rechercher/i)
    fireEvent.change(searchInput, { target: { value: 'Ranked' } })
    expect(screen.getByText('Session Ranked A')).toBeTruthy()
    expect(screen.getByText('Session Ranked C')).toBeTruthy()
    expect(screen.queryByText('Session Casual B')).toBeNull()
  })

  it('affiche "Aucune session" quand le filtre ne correspond à rien', () => {
    const { openPanel } = setup()
    openPanel()
    fireEvent.change(screen.getByPlaceholderText(/Rechercher/i), {
      target: { value: 'xyzinexistant' },
    })
    expect(screen.getByText(/Aucune session/i)).toBeTruthy()
  })
})

describe('SessionMultiSelect — filtre date', () => {
  it('masque les sessions antérieures à dateFrom', () => {
    const { container, openPanel } = setup()
    openPanel()
    const [dateFromInput] = Array.from(container.querySelectorAll<HTMLInputElement>('input[type="date"]'))
    fireEvent.change(dateFromInput, { target: { value: '2026-04-22' } })
    // Sessions A+B (ended 20+21 avril) filtrées car ended_at < dateFrom
    expect(screen.queryByText('Session Ranked A')).toBeNull()
    expect(screen.queryByText('Session Casual B')).toBeNull()
    expect(screen.getByText('Session Ranked C')).toBeTruthy()
  })

  it('masque les sessions postérieures à dateTo', () => {
    const { container, openPanel } = setup()
    openPanel()
    const dateInputs = Array.from(container.querySelectorAll<HTMLInputElement>('input[type="date"]'))
    const dateToInput = dateInputs[1]
    // dateTo = 2026-04-20 → started_at > 2026-04-20T00:00Z → B et C filtrés
    // Session A démarrée à 2026-04-20T00:00Z (=dateTo) → not strictly > → visible
    fireEvent.change(dateToInput, { target: { value: '2026-04-20' } })
    expect(screen.getByText('Session Ranked A')).toBeTruthy()
    expect(screen.queryByText('Session Casual B')).toBeNull()
    expect(screen.queryByText('Session Ranked C')).toBeNull()
  })
})

describe('SessionMultiSelect — validation différée', () => {
  it('ne propage pas onChange avant de cliquer sur Valider', () => {
    const { onChange, openPanel } = setup()
    openPanel()
    // Cocher une session
    fireEvent.click(screen.getByLabelText(/Session Ranked A/i) ?? screen.getAllByRole('checkbox')[0])
    expect(onChange).not.toHaveBeenCalled()
  })

  it('propage onChange(pending) au clic sur Valider', () => {
    const { onChange, openPanel } = setup()
    openPanel()
    const checkboxes = screen.getAllByRole('checkbox')
    fireEvent.click(checkboxes[0]) // sélectionne la 1ère session
    fireEvent.click(screen.getByText(/Valider|Apply/i))
    expect(onChange).toHaveBeenCalledOnce()
    const payload = onChange.mock.calls[0][0] as string[]
    expect(payload).toContain(SESSIONS[0].label)
  })

  it('ferme le panel après validation', () => {
    const { openPanel } = setup()
    openPanel()
    fireEvent.click(screen.getByText(/Valider|Apply/i))
    expect(screen.queryByPlaceholderText(/Rechercher/i)).toBeNull()
  })

  it('réinitialise le pending sur la sélection active si on rouvre sans valider', () => {
    const { openPanel } = setup(['Session Ranked A'])
    openPanel()
    // Décocher la session sélectionnée sans valider
    const checkbox = screen.getAllByRole('checkbox').find(
      (cb) => (cb as HTMLInputElement).checked,
    ) as HTMLInputElement
    fireEvent.click(checkbox) // décoche
    fireEvent.mouseDown(document.body) // ferme sans valider

    // Rouvrir : la session doit être recheckée (pending = selected)
    openPanel()
    const recheckBoxes = screen.getAllByRole('checkbox')
    const recheckA = recheckBoxes.find(
      (cb) => (cb as HTMLInputElement).checked,
    ) as HTMLInputElement | undefined
    expect(recheckA).toBeTruthy()
  })
})

describe('SessionMultiSelect — toggle tout sélectionner / désélectionner', () => {
  it('sélectionne toutes les sessions filtrées au clic sur "Tout sélectionner"', () => {
    const { openPanel } = setup()
    openPanel()
    fireEvent.click(screen.getByText(/Tout sélectionner/i))
    const checkboxes = screen.getAllByRole('checkbox')
    checkboxes.forEach((cb) => {
      expect((cb as HTMLInputElement).checked).toBe(true)
    })
  })

  it('affiche "Tout désélectionner" quand tout est coché', () => {
    const { openPanel } = setup(['Session Ranked A', 'Session Casual B', 'Session Ranked C'])
    openPanel()
    expect(screen.getByText(/Tout désélectionner/i)).toBeTruthy()
  })

  it('désélectionne toutes les sessions filtrées', () => {
    const { onChange, openPanel } = setup([
      'Session Ranked A',
      'Session Casual B',
      'Session Ranked C',
    ])
    openPanel()
    fireEvent.click(screen.getByText(/Tout désélectionner/i))
    fireEvent.click(screen.getByText(/Valider|Apply/i))
    const payload = onChange.mock.calls[0][0] as string[]
    expect(payload).toHaveLength(0)
  })

  it('"Tout sélectionner" ne sélectionne que les sessions filtrées par la recherche', () => {
    const { onChange, openPanel } = setup()
    openPanel()
    fireEvent.change(screen.getByPlaceholderText(/Rechercher/i), {
      target: { value: 'Ranked' },
    })
    fireEvent.click(screen.getByText(/Tout sélectionner/i))
    fireEvent.click(screen.getByText(/Valider|Apply/i))
    const payload = onChange.mock.calls[0][0] as string[]
    // Uniquement Session Ranked A et Session Ranked C
    expect(payload).toContain('Session Ranked A')
    expect(payload).toContain('Session Ranked C')
    expect(payload).not.toContain('Session Casual B')
  })
})

describe('SessionMultiSelect — lien Réinitialiser', () => {
  it('masque le lien Réinitialiser quand aucune session sélectionnée', () => {
    const { openPanel } = setup()
    openPanel()
    expect(screen.queryByText(/Réinitialiser/i)).toBeNull()
  })

  it('affiche le lien Réinitialiser dès qu\'une session est cochée', () => {
    const { openPanel } = setup(['Session Ranked A'])
    openPanel()
    expect(screen.getByText(/Réinitialiser/i)).toBeTruthy()
  })

  it('vide une sélection partielle au clic sur Réinitialiser', () => {
    const { onChange, openPanel } = setup(['Session Ranked A'])
    openPanel()
    // Sélection partielle (1/3) → le toggle reste "Tout sélectionner".
    expect(screen.getByText(/Tout sélectionner/i)).toBeTruthy()
    fireEvent.click(screen.getByText(/Réinitialiser/i))
    fireEvent.click(screen.getByText(/Valider|Apply/i))
    const payload = onChange.mock.calls[0][0] as string[]
    expect(payload).toHaveLength(0)
  })

  it('vide TOUTE la sélection, même les sessions masquées par la recherche', () => {
    const { onChange, openPanel } = setup(['Session Ranked A', 'Session Casual B'])
    openPanel()
    // Filtrer pour ne montrer que les "Ranked" : Casual B est masquée mais
    // reste sélectionnée → Réinitialiser doit aussi la retirer.
    fireEvent.change(screen.getByPlaceholderText(/Rechercher/i), {
      target: { value: 'Ranked' },
    })
    fireEvent.click(screen.getByText(/Réinitialiser/i))
    fireEvent.click(screen.getByText(/Valider|Apply/i))
    const payload = onChange.mock.calls[0][0] as string[]
    expect(payload).toHaveLength(0)
  })
})
