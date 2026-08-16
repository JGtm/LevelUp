import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { PlayerMark } from './PlayerMark'

describe('PlayerMark', () => {
  it('« moi » et « ami » portent leur libellé accessible, dans la langue', () => {
    render(<PlayerMark kind="me" locale="fr" />)
    expect(screen.getByRole('img', { name: 'Moi' })).toBeTruthy()
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
