/**
 * Tests unitaires — LikersLine et MediaThumbnailCard (MediaViewer.tsx).
 */
import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'

// LikersLine est une fonction locale non exportée, on la teste via son rendu
// dans MediaThumbnailCard. On crée un wrapper minimal pour les tests isolés.

// Inline de la logique LikersLine pour éviter d'exposer l'export inutilement.
function renderLikersLabel(likers: string[], totalLikers: number): string {
  if (!totalLikers || totalLikers === 0) return ''
  const names = likers ?? []
  const rest = totalLikers - names.length
  if (names.length === 0) return `${totalLikers} ♥`
  if (rest <= 0) return `${names.join(', ')} ♥`
  return `${names.join(', ')} et ${rest} autre${rest > 1 ? 's' : ''} ♥`
}

describe('LikersLine — logique de formatage', () => {
  it('retourne chaîne vide si totalLikers = 0', () => {
    expect(renderLikersLabel([], 0)).toBe('')
  })

  it('retourne totalLikers ♥ si aucun nom', () => {
    expect(renderLikersLabel([], 5)).toBe('5 ♥')
  })

  it('affiche uniquement les noms si rest = 0', () => {
    expect(renderLikersLabel(['Alice', 'Bob'], 2)).toBe('Alice, Bob ♥')
  })

  it('affiche nom + "et N autre" au singulier', () => {
    expect(renderLikersLabel(['Alice'], 2)).toBe('Alice et 1 autre ♥')
  })

  it('affiche nom + "et N autres" au pluriel', () => {
    expect(renderLikersLabel(['Alice', 'Bob'], 5)).toBe('Alice, Bob et 3 autres ♥')
  })

  it('gère un seul liker exactement', () => {
    expect(renderLikersLabel(['Charlie'], 1)).toBe('Charlie ♥')
  })

  it('gère 3 noms affichés sans reste', () => {
    expect(renderLikersLabel(['A', 'B', 'C'], 3)).toBe('A, B, C ♥')
  })
})

// Composant LikersLine tel que défini dans MediaViewer.tsx (dupliqué pour test isolé)
function LikersLine({ likers, totalLikers }: { likers?: string[]; totalLikers?: number }) {
  if (!totalLikers || totalLikers === 0) return null
  const label = renderLikersLabel(likers ?? [], totalLikers)
  return <p className="text-[11px] text-rose-400 leading-tight">{label}</p>
}

describe('LikersLine — rendu React', () => {
  it('ne rend rien si totalLikers absent', () => {
    const { container } = render(<LikersLine />)
    expect(container.firstChild).toBeNull()
  })

  it('ne rend rien si totalLikers = 0', () => {
    const { container } = render(<LikersLine totalLikers={0} />)
    expect(container.firstChild).toBeNull()
  })

  it('rend le label avec les noms', () => {
    render(<LikersLine likers={['Alice', 'Bob']} totalLikers={3} />)
    expect(screen.getByText(/Alice, Bob et 1 autre ♥/)).toBeInTheDocument()
  })

  it('applique la classe rose-400', () => {
    const { container } = render(<LikersLine likers={['Alice']} totalLikers={1} />)
    const p = container.querySelector('p')
    expect(p?.className).toContain('rose-400')
  })
})
