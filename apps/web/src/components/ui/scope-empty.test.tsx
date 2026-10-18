import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { ScopeEmptyCard, ScopeEmptySection, ScopeEmptyState } from './scope-empty'
import { scopeEmptyText } from './scope-empty-text'

describe('scope-empty', () => {
  it('le constat des matchs à objectif, en français et en anglais', () => {
    expect(scopeEmptyText('objective', 'fr').title).toBe('Aucun match à objectif sur ce périmètre')
    expect(scopeEmptyText('objective', 'en').title).toBe('No objective match in this scope')
  })

  it('le constat des frags reprend la phrase du manifeste partagé', () => {
    expect(scopeEmptyText('frags', 'fr').title).toBe('Aucun frag sur ce périmètre')
    expect(scopeEmptyText('frags', 'en').title).toBe('No kills in this scope')
  })

  it('le constat seul porte le titre et la description', () => {
    render(<ScopeEmptyState kind="objectiveActions" locale="fr" />)
    expect(screen.getByText('Aucune action d’objectif sur ce périmètre')).toBeInTheDocument()
  })

  it('le bloc parent entoure le constat', () => {
    render(<ScopeEmptyCard kind="objective" locale="fr" testId="bloc" />)
    expect(screen.getByTestId('bloc')).toContainElement(screen.getByText('Aucun match à objectif sur ce périmètre'))
  })

  it('la section garde son intertitre au-dessus du bloc', () => {
    render(<ScopeEmptySection title="Objectif" kind="objective" locale="en" testId="sec" />)
    const section = screen.getByTestId('sec')
    expect(section).toContainElement(screen.getByRole('heading', { name: 'Objectif' }))
    expect(section).toContainElement(screen.getByText('No objective match in this scope'))
  })
})
