/**
 * Tests — ValueGrid (le rendu de la grille de valeurs).
 *
 * CE QU'ILS PROTÈGENT, et ce sont les exigences non négociables de la forme :
 *   1. L'INFOBULLE EST AU SURVOL *ET* AU FOCUS CLAVIER — une valeur atteignable à la souris
 *      seulement n'est pas atteignable.
 *   2. L'ENSEMBLE DÉFILE DANS SON PROPRE CONTENEUR (`overflow-x`), jamais le corps de la page.
 *   3. LES NOMBRES SONT EN CHIFFRES TABULAIRES : sans quoi les colonnes de valeurs dansent
 *      d'une ligne à l'autre.
 */
import { describe, expect, it } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import { ValueGrid } from './ValueGrid'
import { buildValueGrid } from './valueGridModel'

const MODEL = buildValueGrid({
  rows: [
    { key: 'a', label: 'Alpha', group: 't0', accent: 'var(--ac-team-ally)', emphasis: true },
    { key: 'b', label: 'Bravo', group: 't1', accent: 'var(--ac-team-enemy)' },
  ],
  columns: [{ key: 'grenades', label: 'Grenades', showTotal: true }],
  value: (r) => [40, 13][r],
  format: (v) => String(v),
  color: () => 'var(--ac-frag-grenade)',
  tooltip: (r, _c, text) => `${['Alpha', 'Bravo'][r]} — Grenades : ${text}`,
})

describe('ValueGrid — ce que l’écran montre', () => {
  it('écrit chaque nom, chaque valeur, le total de colonne et les trois graduations', () => {
    const vue = render(<ValueGrid model={MODEL} />)
    expect(vue.getByText('Alpha')).toBeTruthy()
    expect(vue.getByText('Bravo')).toBeTruthy()
    expect(vue.getByText('Grenades')).toBeTruthy()
    expect(vue.getByText('53')).toBeTruthy() // total de colonne
    // « 40 » deux fois : la valeur d'Alpha, et la borne haute de l'axe de sa colonne.
    expect(vue.getAllByText('40')).toHaveLength(2)
    expect(vue.getAllByText('0').length).toBeGreaterThan(0) // graduation de gauche
    expect(vue.getByText('20')).toBeTruthy() // milieu
  })

  it('donne à la plus grande barre toute la largeur de son rail', () => {
    const vue = render(<ValueGrid model={MODEL} />)
    const barres = vue.container.querySelectorAll('[role="img"] > div')
    expect((barres[0] as HTMLElement).style.width).toBe('100%')
    expect((barres[1] as HTMLElement).style.width).toBe('32.5%')
  })

  it('défile HORIZONTALEMENT dans son propre conteneur, jamais le corps de page', () => {
    const vue = render(<ValueGrid model={MODEL} />)
    expect(vue.container.querySelector('.overflow-x-auto')).toBeTruthy()
  })

  it('écrit les nombres en chiffres tabulaires', () => {
    const vue = render(<ValueGrid model={MODEL} />)
    expect(vue.getByText('13').className).toContain('tabular-nums')
  })

  it('sépare deux groupes consécutifs par un filet, un par colonne plus celle des noms', () => {
    const vue = render(<ValueGrid model={MODEL} />)
    // 1 colonne de valeurs + la colonne des noms = 2 cellules de filet.
    expect(vue.container.querySelectorAll('.h-px').length).toBe(2)
  })
})

describe('ValueGrid — l’infobulle', () => {
  it('s’ouvre au SURVOL de la barre', () => {
    const vue = render(<ValueGrid model={MODEL} />)
    fireEvent.mouseEnter(vue.getAllByRole('img')[0].parentElement as Element)
    expect(screen.getByRole('tooltip').textContent).toBe('Alpha — Grenades : 40')
  })

  it('s’ouvre aussi au FOCUS CLAVIER : chaque barre est atteignable au clavier', () => {
    const vue = render(<ValueGrid model={MODEL} />)
    const barre = vue.getAllByRole('img')[1]
    expect(barre.getAttribute('tabindex')).toBe('0')
    fireEvent.focus(barre)
    expect(screen.getByRole('tooltip').textContent).toBe('Bravo — Grenades : 13')
  })

  it('porte le même texte en nom accessible de la barre', () => {
    const vue = render(<ValueGrid model={MODEL} />)
    expect(vue.getAllByRole('img')[0].getAttribute('aria-label')).toBe('Alpha — Grenades : 40')
  })
})
