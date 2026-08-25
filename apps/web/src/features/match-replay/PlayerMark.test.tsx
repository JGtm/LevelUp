import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { PlayerMark } from './PlayerMark'

describe('PlayerMark', () => {
  it('« moi » ne dessine plus RIEN (demande utilisateur du 2026-08-24 : le point du joueur actif est supprimé partout)', () => {
    const { container } = render(<PlayerMark kind="me" locale="fr" />)
    expect(container.innerHTML).toBe('')
  })
  it('« ami » porte son libellé accessible, dans la langue', () => {
    render(<PlayerMark kind="friend" locale="fr" />)
    expect(screen.getByRole('img', { name: 'Ami' })).toBeTruthy()
  })
  it('friend en anglais', () => {
    render(<PlayerMark kind="friend" locale="en" />)
    expect(screen.getByRole('img', { name: 'Friend' })).toBeTruthy()
  })
  it('sans marque : rien du tout', () => {
    const { container } = render(<PlayerMark kind={undefined} locale="fr" />)
    expect(container.innerHTML).toBe('')
  })
})
