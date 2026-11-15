/**
 * Outils de test du tri par clic sur les en-têtes (étape 7.7) : enregistrent l'ordre des lignes
 * après chaque clic, pour le figer avant et après le passage d'un tableau sur TanStack Table.
 */
import { fireEvent, within } from '@testing-library/react'

/** Texte de la colonne `col` (0-based) de chaque ligne du corps du tableau. */
export function rowTexts(container: HTMLElement, col = 0): string[] {
  return Array.from(container.querySelectorAll('tbody tr')).map((tr) => {
    const cell = tr.querySelectorAll('td, th')[col]
    return (cell?.textContent ?? '').replace(/\s+/g, ' ').trim()
  })
}

/** `aria-sort` de chaque en-tête portant la propriété, dans l'ordre des colonnes. */
export function ariaSorts(container: HTMLElement): string[] {
  return Array.from(container.querySelectorAll('thead th')).map((th) => th.getAttribute('aria-sort') ?? '-')
}

export interface SortHeader {
  /** Texte accessible du bouton d'en-tête. */
  name: string | RegExp
  /** Colonne dont le texte sert d'ordre de lecture des lignes. */
  col?: number
}

export interface SortRecording {
  /** Ordre des lignes avant tout clic. */
  initial: string
  /** Par en-tête : l'ordre des lignes après 1, 2 et 3 clics, puis les aria-sort après le premier clic. */
  [header: string]: string
}

const order = (rows: string[]): string => rows.join(' > ')

/**
 * Clique chaque en-tête trois fois (premier sens, bascule, bascule de retour) et rend l'ordre des lignes
 * à chaque étape, plus l'état `aria-sort` après le premier clic. La clé de chaque entrée est le nom de l'en-tête.
 */
export function recordSortOrders(container: HTMLElement, headers: SortHeader[], readCol = 0): SortRecording {
  const out: SortRecording = { initial: order(rowTexts(container, readCol)) }
  const scope = within(container)
  for (const h of headers) {
    const button = scope.getByRole('button', { name: h.name })
    const steps: string[] = []
    let ariaAfterFirst = ''
    for (let i = 0; i < 3; i++) {
      fireEvent.click(button)
      steps.push(order(rowTexts(container, h.col ?? readCol)))
      if (i === 0) ariaAfterFirst = ariaSorts(container).join(',')
    }
    out[String(h.name)] = `1: ${steps[0]} | 2: ${steps[1]} | 3: ${steps[2]} | aria-sort: ${ariaAfterFirst}`
  }
  return out
}
