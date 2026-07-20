/**
 * FragSunburst.test.tsx — rendu SVG (arcs classe + rôle, lignes de rappel des rôles,
 * légende des classes) + contrat du builder pur `buildSunburstModel` (hiérarchie
 * classe→rôle, feuilles sans ligne de rappel, ordre conservé). Pas de mock ECharts :
 * le composant est du SVG inline, testable directement en jsdom.
 */
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'

import {
  FragSunburst,
  buildSunburstModel,
  type FragSunburstColors,
  type FragSunburstLabels,
} from './FragSunburst'
import type { FragDistribution } from '@/lib/api/types'

const COLORS: FragSunburstColors = {
  classColor: (c) => `col:${c}`,
  roleColor: (c, i) => `role:${c}:${i}`,
  leafColor: (c) => `leaf:${c}`,
}

const LABELS: FragSunburstLabels = {
  classLabel: (c) => `class:${c}`,
  roleLabel: (r) => `role:${r}`,
  formatValue: (n) => String(n),
  formatShare: (n) => `${n}%`,
}

const DIST: FragDistribution = {
  total_kills: 18,
  classes: [
    {
      class: 'shoulder',
      kills: 10,
      authoritative: false,
      roles: [
        { role: 'precision', kills: 6 },
        { role: 'automatic', kills: 4 },
      ],
    },
    { class: 'melee', kills: 5, authoritative: true },
    { class: 'unattributed', kills: 3, authoritative: false },
  ],
}

describe('buildSunburstModel (builder pur)', () => {
  it('produit arcs classe + rôle + feuille, lignes de rappel des rôles, légende des classes', () => {
    const model = buildSunburstModel(DIST.classes ?? [], DIST.total_kills, COLORS, LABELS)

    // Arcs : 3 classes (anneau interne) + 2 rôles (shoulder) + 2 feuilles (melee, unattributed).
    expect(model.arcs.filter((a) => a.kind === 'class')).toHaveLength(3)
    expect(model.arcs.filter((a) => a.kind === 'role')).toHaveLength(2)
    expect(model.arcs.filter((a) => a.kind === 'leaf')).toHaveLength(2)

    // Lignes de rappel : uniquement pour les rôles (2), jamais pour les feuilles.
    expect(model.callouts).toHaveLength(2)
    expect(model.callouts.map((c) => c.label).sort()).toEqual(['role:automatic', 'role:precision'])
    // Chaque callout a une polyline (point → coude → genou → bord) et un texte valeur.
    for (const co of model.callouts) {
      expect(co.points.split(' ')).toHaveLength(4) // point → coude → genou → bord
      expect(co.valueLabel).toContain('%')
    }

    // FIX 1 — répartition gauche/droite (reprise de buildSun) : precision (haut-gauche,
    // mid < 90°) ancré 'start' au bord GAUCHE (x proche de 0) ; automatic (droite,
    // mid ≥ 90°) ancré 'end' au bord DROIT (x proche du viewBox). Les DEUX ancres présentes.
    const anchors = model.callouts.map((c) => c.anchor)
    expect(anchors).toContain('start')
    expect(anchors).toContain('end')
    const left = model.callouts.find((c) => c.anchor === 'start')!
    const right = model.callouts.find((c) => c.anchor === 'end')!
    expect(left.tx).toBeLessThan(20) // plaqué à gauche (x ≈ 6)
    expect(right.tx).toBeGreaterThan(420) // plaqué à droite (x ≈ W-6, viewBox 440)

    // Légende : 1 entrée par classe, ordre conservé, couleur de classe (pas de rôle).
    expect(model.legend.map((l) => l.classKey)).toEqual(['shoulder', 'melee', 'unattributed'])
    expect(model.legend[0].color).toBe('col:shoulder')
  })

  it('total 0 → modèle vide (rien à tracer)', () => {
    const model = buildSunburstModel([], 0, COLORS, LABELS)
    expect(model.arcs).toHaveLength(0)
    expect(model.callouts).toHaveLength(0)
    expect(model.legend).toHaveLength(0)
  })
})

describe('FragSunburst (composant SVG)', () => {
  it('total > 0 → rend le SVG, les lignes de rappel des rôles et la légende', () => {
    render(<FragSunburst distribution={DIST} />)
    expect(screen.getByTestId('frag-sunburst')).toBeInTheDocument()
    // 2 lignes de rappel (precision + automatic), aucune pour les feuilles.
    expect(screen.getAllByTestId('frag-callout')).toHaveLength(2)
    // Légende des classes présente.
    expect(screen.getByTestId('frag-legend')).toBeInTheDocument()
    // Centre = total.
    expect(screen.getByText('18')).toBeInTheDocument()
  })

  it('total 0 → rend null (aucune carte)', () => {
    const { container } = render(<FragSunburst distribution={{ total_kills: 0, classes: [] }} />)
    expect(container).toBeEmptyDOMElement()
  })

  it('distribution absente → rend null', () => {
    const { container } = render(<FragSunburst distribution={null} />)
    expect(container).toBeEmptyDOMElement()
  })
})
