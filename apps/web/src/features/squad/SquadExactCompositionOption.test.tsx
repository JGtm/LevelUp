/**
 * La case « composition stricte » explique sa règle au survol (décision du 2026-10-21 : garder
 * le libellé et le défaut coché, ajouter l'infobulle). Le test fige : l'infobulle s'ouvre au
 * survol et au focus avec le composant `Tooltip` de la page, le texte est le texte FR/EN du
 * dictionnaire, et la case reste pilotée par ses props.
 */
import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'

import { SquadExactCompositionOption } from './SquadExactCompositionOption'
import { getSquadText } from './i18n'

describe('SquadExactCompositionOption', () => {
  it.each(['fr', 'en'] as const)('ouvre l’infobulle explicative au survol (%s)', (locale) => {
    const t = getSquadText(locale)
    render(<SquadExactCompositionOption locale={locale} checked onChange={() => {}} />)

    expect(screen.queryByRole('tooltip')).toBeNull()
    fireEvent.mouseEnter(screen.getByText(t.filter.exactComposition))
    expect(screen.getByRole('tooltip')).toHaveTextContent(t.filter.exactCompositionTitle)

    fireEvent.mouseLeave(screen.getByText(t.filter.exactComposition))
    expect(screen.queryByRole('tooltip')).toBeNull()
  })

  it('ouvre aussi l’infobulle au focus de la case (clavier)', () => {
    render(<SquadExactCompositionOption locale="fr" checked onChange={() => {}} />)
    fireEvent.focus(screen.getByRole('checkbox'))
    expect(screen.getByRole('tooltip')).toBeInTheDocument()
  })

  it('dit la règle : coéquipiers cochés, autre joueur suivi écarté, ami déclaré écarté, autres joueurs ignorés', () => {
    const fr = getSquadText('fr').filter.exactCompositionTitle
    const en = getSquadText('en').filter.exactCompositionTitle
    expect(fr).toContain('autre joueur suivi')
    expect(fr).toContain('ami déclaré')
    expect(en).toContain('another tracked player')
    expect(en).toContain('declared friend')
  })

  it('garde le libellé, l’état coché et la bascule pilotée par la prop', () => {
    const onChange = vi.fn()
    render(<SquadExactCompositionOption locale="fr" checked onChange={onChange} />)
    const box = screen.getByRole('checkbox', { name: getSquadText('fr').filter.exactComposition })
    expect(box).toBeChecked()
    fireEvent.click(box)
    expect(onChange).toHaveBeenCalledWith(false)
  })
})
