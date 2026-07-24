/**
 * FragSunburst.test.tsx — rendu SVG (arcs classe + rôle, lignes de rappel des rôles,
 * légende des classes) + contrat du builder pur `buildSunburstModel` (hiérarchie
 * classe→rôle, feuilles sans ligne de rappel, ordre conservé). Pas de mock ECharts :
 * le composant est du SVG inline, testable directement en jsdom.
 */
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'

import { FragSunburst } from './FragSunburst'
import {
  buildSunburstModel,
  type FragSunburstColors,
  type FragSunburstLabels,
} from './fragSunburstModel'
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

    // Légende : 1 entrée par classe, ordre conservé, couleur de classe (pas de rôle).
    expect(model.legend.map((l) => l.classKey)).toEqual(['shoulder', 'melee', 'unattributed'])
    expect(model.legend[0].color).toBe('col:shoulder')
    // I5 (V7.1) : chaque entrée de légende affiche le POURCENTAGE (« — NN % »), pas le
    // seul décompte brut — via labels.formatShare (déjà utilisé par le tooltip des arcs).
    for (const row of model.legend) {
      expect(row.valueLabel.startsWith('— ')).toBe(true)
      expect(row.valueLabel).toContain('%')
    }
    expect(model.legend[0].valueLabel).toBe(`— ${LABELS.formatShare(10)}`)
  })

  it('côté du callout = position HORIZONTALE (cos) de l\'arc — le trait ne traverse jamais le donut', () => {
    // 4 rôles en quadrants (repère polaire interne : q1 45° HAUT-DROITE, q2 135° BAS-DROITE,
    // q3 225° BAS-GAUCHE, q4 315° HAUT-GAUCHE). Côté = cos(mid-90) = signe de X : moitié
    // DROITE (x >= centre) → étiquette à droite ; moitié GAUCHE → à gauche. POINT CLÉ : q1 et
    // q4 sont tous deux en HAUT mais partent de côtés OPPOSÉS → prouve le split horizontal
    // (X), pas vertical (Y) ; c'est ce que sin ratait quand tous les rôles étaient en haut.
    const Q_DIST: FragDistribution = {
      total_kills: 20,
      classes: [
        { class: 'shoulder', kills: 5, authoritative: false, roles: [{ role: 'q1', kills: 5 }] },
        { class: 'sidearm', kills: 5, authoritative: false, roles: [{ role: 'q2', kills: 5 }] },
        { class: 'heavy', kills: 5, authoritative: false, roles: [{ role: 'q3', kills: 5 }] },
        { class: 'melee', kills: 5, authoritative: false, roles: [{ role: 'q4', kills: 5 }] },
      ],
    }
    const model = buildSunburstModel(Q_DIST.classes ?? [], Q_DIST.total_kills, COLORS, LABELS)
    expect(model.callouts).toHaveLength(4)
    const byLabel = (l: string) => model.callouts.find((c) => c.label === l)!

    // Moitié DROITE (q1 haut-droite, q2 bas-droite) → DROITE (ancre 'end', texte à droite).
    expect(byLabel('role:q1').anchor).toBe('end')
    expect(byLabel('role:q2').anchor).toBe('end')
    expect(byLabel('role:q1').tx).toBeGreaterThan(420)
    // Moitié GAUCHE (q3 bas-gauche, q4 haut-gauche) → GAUCHE (ancre 'start', texte à gauche).
    expect(byLabel('role:q3').anchor).toBe('start')
    expect(byLabel('role:q4').anchor).toBe('start')
    expect(byLabel('role:q4').tx).toBeLessThan(20)
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

  it('I5 (V7.1) : heightPx fixe la hauteur du conteneur SVG (h-full, pas h-auto) — aligne la carte sur un graphe voisin de hauteur fixe', () => {
    const { container } = render(<FragSunburst distribution={DIST} heightPx={320} />)
    const svg = container.querySelector('svg')!
    expect(svg.getAttribute('class')).toContain('h-full')
    expect(svg.getAttribute('class')).not.toContain('h-auto')
    // Le conteneur direct du SVG porte la hauteur fixe demandée.
    const svgContainer = svg.parentElement!
    expect(svgContainer.style.height).toBe('320px')
  })

  it('sans heightPx : comportement historique inchangé (h-auto, pas de hauteur fixée)', () => {
    const { container } = render(<FragSunburst distribution={DIST} />)
    const svg = container.querySelector('svg')!
    expect(svg.getAttribute('class')).toContain('h-auto')
    const svgContainer = svg.parentElement!
    expect(svgContainer.style.height).toBe('')
  })
})
