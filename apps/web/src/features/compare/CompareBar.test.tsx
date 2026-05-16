/**
 * Tests unitaires — CompareBar.
 *
 * Vérifie que la barre composite est rendue avec le bon style gradient
 * et les bonnes valeurs de gauche/droite.
 */
import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { CompareBar } from './CompareBar'

describe('CompareBar', () => {
  it('rend le label centré', () => {
    render(<CompareBar label="Frags/partie" valueA="9,54" valueB="16,81" rawA={9.54} rawB={16.81} winner="b" />)
    expect(screen.getByText('Frags/partie')).toBeInTheDocument()
  })

  it('affiche valueA et valueB', () => {
    render(<CompareBar label="KDA" valueA="0,93" valueB="1,82" rawA={0.93} rawB={1.82} winner="b" />)
    expect(screen.getByText('0,93')).toBeInTheDocument()
    expect(screen.getByText('1,82')).toBeInTheDocument()
  })

  it('la barre a un style linear-gradient', () => {
    const { container } = render(
      <CompareBar label="KDA" valueA="0,93" valueB="1,82" rawA={0.93} rawB={1.82} winner="b" />,
    )
    // La barre est le div avec class flex-1 h-3
    const barDiv = container.querySelector('[data-testid="compare-bar-track"]') as HTMLElement
    expect(barDiv).toBeTruthy()

    const style = barDiv.getAttribute('style') ?? ''
    console.log('Style attribute:', style)
    expect(style).toContain('linear-gradient')
  })

  it('calcule le bon ratio proportionnel (A=0.93, B=1.82 → ratio≈33.8%)', () => {
    const { container } = render(
      <CompareBar label="KDA" valueA="0,93" valueB="1,82" rawA={0.93} rawB={1.82} winner="b" />,
    )
    const barDiv = container.querySelector('[data-testid="compare-bar-track"]') as HTMLElement
    const style = barDiv.getAttribute('style') ?? ''

    // ratio = 0.93 / (0.93 + 1.82) = 0.93 / 2.75 ≈ 0.338 → 33.8%
    expect(style).toContain('33.8%')
  })

  it('ratio = 50% quand rawA = rawB', () => {
    const { container } = render(
      <CompareBar label="Test" valueA="1" valueB="1" rawA={1} rawB={1} winner="tie" />,
    )
    const barDiv = container.querySelector('[data-testid="compare-bar-track"]') as HTMLElement
    const style = barDiv.getAttribute('style') ?? ''
    expect(style).toContain('50.0%')
  })

  it('ratio = 50% quand rawA = rawB = 0 (fallback)', () => {
    const { container } = render(
      <CompareBar label="Test" valueA="0" valueB="0" rawA={0} rawB={0} winner="tie" />,
    )
    const barDiv = container.querySelector('[data-testid="compare-bar-track"]') as HTMLElement
    const style = barDiv.getAttribute('style') ?? ''
    expect(style).toContain('50.0%')
  })

  it('ratio min = 5% quand B >> A', () => {
    const { container } = render(
      <CompareBar label="Test" valueA="1" valueB="1000" rawA={1} rawB={1000} winner="b" />,
    )
    const barDiv = container.querySelector('[data-testid="compare-bar-track"]') as HTMLElement
    const style = barDiv.getAttribute('style') ?? ''
    // 1/(1+1000) ≈ 0.1% → clamped to 5%
    expect(style).toContain('5.0%')
  })

  it('ratio max = 95% quand A >> B', () => {
    const { container } = render(
      <CompareBar label="Test" valueA="1000" valueB="1" rawA={1000} rawB={1} winner="a" />,
    )
    const barDiv = container.querySelector('[data-testid="compare-bar-track"]') as HTMLElement
    const style = barDiv.getAttribute('style') ?? ''
    // 1000/(1000+1) ≈ 99.9% → clamped to 95%
    expect(style).toContain('95.0%')
  })

  it('gère NaN gracieusement (fallback ratio 50%)', () => {
    const { container } = render(
      <CompareBar label="Test" valueA="—" valueB="—" rawA={NaN} rawB={NaN} winner={null} />,
    )
    const barDiv = container.querySelector('[data-testid="compare-bar-track"]') as HTMLElement
    const style = barDiv.getAttribute('style') ?? ''
    expect(style).toContain('50.0%')
  })

  it('affiche sampleNote si fourni', () => {
    render(
      <CompareBar label="Headshots" valueA="3,21" valueB="6,33" rawA={3.21} rawB={6.33} winner="b" sampleNote="(sur 5 parties)" />,
    )
    expect(screen.getByText('(sur 5 parties)')).toBeInTheDocument()
  })

  it("n'affiche pas sampleNote si absent", () => {
    render(<CompareBar label="KDA" valueA="0,93" valueB="1,82" rawA={0.93} rawB={1.82} winner="b" />)
    expect(screen.queryByText(/sur \d+ parties/)).not.toBeInTheDocument()
  })

  it('affiche N/A et neutralise la barre quand availableB=false', () => {
    const { container } = render(
      <CompareBar
        label="Perf. record" valueA="98" valueB="N/A" rawA={98} rawB={0} winner="a"
        availableA availableB={false}
      />,
    )
    expect(screen.getByText('N/A')).toBeInTheDocument()
    const bar = container.querySelector('[data-testid="compare-bar-track"]') as HTMLElement
    const style = bar.getAttribute('style') ?? ''
    // Ratio neutre 50% + opacité réduite quand une valeur est N/A.
    expect(style).toContain('50.0%')
    expect(style).toMatch(/opacity:\s*0\.35/)
  })

  it('met le côté N/A en italique muted, ignore le winner pointant dessus', () => {
    render(
      <CompareBar
        label="Perf. record" valueA="98" valueB="N/A" rawA={98} rawB={0} winner="a"
        availableA availableB={false}
      />,
    )
    const naSpan = screen.getByText('N/A')
    expect(naSpan.className).toContain('italic')
  })
})
