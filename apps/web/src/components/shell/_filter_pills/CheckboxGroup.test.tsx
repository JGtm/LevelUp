/**
 * Tests — CheckboxGroup avec progressive disclosure des options à count=0.
 *
 * Couvre : affichage des counts, repli/expansion des options indisponibles,
 * désactivation des options à 0, désactivation du pliage pour Experience types.
 */
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import type { LabelValue } from '@/lib/api/types'
import { CheckboxGroup } from './CheckboxGroup'

function makeOptions(): LabelValue[] {
  return [
    { label: 'Slayer', value: 'Slayer', count: 120 },
    { label: 'CTF', value: 'CTF', count: 45 },
    { label: 'Strongholds', value: 'Strongholds', count: 12 },
    { label: 'Oddball', value: 'Oddball', count: 0 },
    { label: 'KOTH', value: 'KOTH', count: 0 },
    { label: 'Assassin', value: 'Assassin', count: 0 },
    { label: 'Stockpile', value: 'Stockpile', count: 0 },
  ]
}

describe('CheckboxGroup — affichage de base', () => {
  it('affiche chaque option active avec son count', () => {
    render(
      <CheckboxGroup
        title="Modes"
        options={makeOptions()}
        selected={[]}
        onToggle={vi.fn()}
      />,
    )
    expect(screen.getByText('Slayer')).toBeInTheDocument()
    expect(screen.getByText('120')).toBeInTheDocument()
    expect(screen.getByText('CTF')).toBeInTheDocument()
    expect(screen.getByText('45')).toBeInTheDocument()
  })

  it("ne rend rien si aucune option et aucun zombie", () => {
    const { container } = render(
      <CheckboxGroup title="Vide" options={[]} selected={[]} onToggle={vi.fn()} />,
    )
    expect(container.firstChild).toBeNull()
  })
})

describe('CheckboxGroup — progressive disclosure', () => {
  it('cache les options à count=0 derrière un compteur cliquable', () => {
    render(
      <CheckboxGroup
        title="Modes"
        options={makeOptions()}
        selected={[]}
        onToggle={vi.fn()}
      />,
    )
    // Les 4 options indisponibles ne sont pas visibles initialement.
    expect(screen.queryByText('Oddball')).not.toBeInTheDocument()
    expect(screen.queryByText('KOTH')).not.toBeInTheDocument()
    // Mais le compteur les annonce.
    expect(screen.getByText(/4 options indisponibles/)).toBeInTheDocument()
  })

  it('déplie les options à 0 au clic sur le compteur', () => {
    render(
      <CheckboxGroup
        title="Modes"
        options={makeOptions()}
        selected={[]}
        onToggle={vi.fn()}
      />,
    )
    fireEvent.click(screen.getByText(/4 options indisponibles/))
    expect(screen.getByText('Oddball')).toBeInTheDocument()
    expect(screen.getByText('KOTH')).toBeInTheDocument()
    // Et un bouton "Masquer" apparaît.
    expect(screen.getByText(/Masquer/i)).toBeInTheDocument()
  })

  it("rend les options à 0 disabled (incochables)", () => {
    const onToggle = vi.fn()
    render(
      <CheckboxGroup
        title="Modes"
        options={makeOptions()}
        selected={[]}
        onToggle={onToggle}
      />,
    )
    fireEvent.click(screen.getByText(/4 options indisponibles/))

    // La checkbox d'Oddball doit être disabled.
    const oddball = screen.getByText('Oddball').closest('label')!
    const checkbox = oddball.querySelector('input[type="checkbox"]') as HTMLInputElement
    expect(checkbox.disabled).toBe(true)

    // Cliquer dessus ne déclenche pas onToggle.
    fireEvent.click(checkbox)
    expect(onToggle).not.toHaveBeenCalled()
  })

  it('ne replie pas les options si disableCollapse=true (Experience types)', () => {
    // Liste très courte simulant Experience types : 3 valeurs dont 1 à 0.
    const exp: LabelValue[] = [
      { label: 'PVP non classé', value: 'PVP non classé', count: 200 },
      { label: 'PVP classé', value: 'PVP classé', count: 50 },
      { label: 'PVE', value: 'PVE', count: 0 },
    ]
    render(
      <CheckboxGroup
        title="Type d'expérience"
        options={exp}
        selected={[]}
        onToggle={vi.fn()}
        disableCollapse
      />,
    )
    // PVE (count=0) reste visible directement, sans pliage.
    expect(screen.getByText('PVE')).toBeInTheDocument()
    expect(screen.queryByText(/options indisponibles/)).not.toBeInTheDocument()
    // Mais elle est disabled.
    const pve = screen.getByText('PVE').closest('label')!
    const checkbox = pve.querySelector('input[type="checkbox"]') as HTMLInputElement
    expect(checkbox.disabled).toBe(true)
  })

  it('garde une option à count=0 visible si déjà cochée', () => {
    // Cas zombie partiel : l'option existe mais l'utilisateur l'a déjà cochée
    // → doit rester visible et cliquable pour qu'il puisse la décocher.
    render(
      <CheckboxGroup
        title="Modes"
        options={makeOptions()}
        selected={['Oddball']}
        onToggle={vi.fn()}
      />,
    )
    expect(screen.getByText('Oddball')).toBeInTheDocument()
    // Une seule option indisponible reste cachée (KOTH/Assassin/Stockpile=3).
    expect(screen.getByText(/3 options indisponibles/)).toBeInTheDocument()
  })
})

describe('CheckboxGroup — toggle', () => {
  it("appelle onToggle pour les options actives", () => {
    const onToggle = vi.fn()
    render(
      <CheckboxGroup
        title="Modes"
        options={makeOptions()}
        selected={[]}
        onToggle={onToggle}
      />,
    )
    const slayer = screen.getByText('Slayer').closest('label')!
    const checkbox = slayer.querySelector('input[type="checkbox"]') as HTMLInputElement
    fireEvent.click(checkbox)
    expect(onToggle).toHaveBeenCalledWith('Slayer')
  })
})
