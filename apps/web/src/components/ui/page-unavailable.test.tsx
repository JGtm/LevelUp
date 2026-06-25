/**
 * Tests unitaires — PageUnavailable + apiErrorCode (ADR 0029).
 */
import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { PageUnavailable } from './page-unavailable'
import { apiErrorCode } from '@/lib/api/client'

describe('PageUnavailable', () => {
  it('affiche le titre et la description', () => {
    render(<PageUnavailable title="Match indisponible" description="Tu n'as pas participé." />)
    expect(screen.getByText('Match indisponible')).toBeTruthy()
    expect(screen.getByText("Tu n'as pas participé.")).toBeTruthy()
  })

  it('rend les actions et déclenche onClick', () => {
    const onHome = vi.fn()
    render(
      <PageUnavailable title="t" description="d" actions={[{ label: 'Accueil', onClick: onHome }]} />,
    )
    fireEvent.click(screen.getByText('Accueil'))
    expect(onHome).toHaveBeenCalledTimes(1)
  })

  it('ne rend aucun bouton sans action', () => {
    const { container } = render(<PageUnavailable title="t" description="d" />)
    expect(container.querySelectorAll('button').length).toBe(0)
  })
})

describe('apiErrorCode', () => {
  it('extrait le code d\'un objet ApiError', () => {
    expect(apiErrorCode({ code: 'match_not_participant', status: 404 })).toBe('match_not_participant')
    expect(apiErrorCode({ code: 'session_not_found', status: 404 })).toBe('session_not_found')
  })

  it('retourne undefined hors ApiError', () => {
    expect(apiErrorCode(null)).toBeUndefined()
    expect(apiErrorCode({})).toBeUndefined()
    expect(apiErrorCode(new Error('boom'))).toBeUndefined()
  })
})
