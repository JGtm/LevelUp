/**
 * Tests — SessionPill : libellés d'accessibilité résolus par manifest (compléments i18n 11.C).
 *
 * `formatMessage` est remplacé par un rendu de la clé : un libellé écrit en dur dans le JSX
 * ne correspondrait à aucune clé et ferait échouer ces assertions.
 */
import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import type { SessionOption } from '@/lib/api/types'

vi.mock('@/lib/i18n/format', () => ({
  formatMessage: (_manifest: unknown, key: string) => key,
}))

import { SessionPill } from './SessionPill'

const noop = () => {}

function session(over: Partial<SessionOption> & { session_id: string }): SessionOption {
  return { label: 'Session A', match_count: 4, match_count_filtered: 4, ...over } as SessionOption
}

describe('SessionPill — libellé accessible de la liste', () => {
  it('la liste ouverte porte le libellé du manifest', () => {
    render(
      <SessionPill
        open
        onToggle={noop}
        onClose={noop}
        currentLabel={null}
        allSessions={[session({ session_id: 's1' })]}
        pickedId={null}
        onPick={noop}
      />,
    )
    expect(screen.getByRole('listbox', { name: 'common.filters.session_list_label' })).toBeInTheDocument()
  })
})
