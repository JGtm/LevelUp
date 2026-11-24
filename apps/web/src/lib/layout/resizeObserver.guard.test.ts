/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail : UN OBSERVATEUR DE TAILLE SE POSE À UN SEUL ENDROIT, `lib/layout/observeSize.ts` (règle <= 2 copies,
 * CLAUDE.md n° 6).
 *
 * Quatre mesures du rejeu (largeur du corps du cadre, largeur utile d'une zone, taille du plan de position, espace
 * libre du terrain) reposaient chacune sur leur propre `new ResizeObserver(...)`, avec le même repli « jsdom n'en a
 * pas » recopié quatre fois. Elles passent par `observeSize` ; ce garde interdit d'en reposer un à côté.
 *
 * LES AUTRES POSEURS sont les mesures hors du rejeu ou d'un autre genre (défilement d'un carrousel, ajustement d'un
 * graphique, lignes de la barre de lecture). Chacun est inscrit ici avec sa date ; la liste ne fait que rétrécir (un
 * inscrit qui n'en pose plus fait échouer le test, un poseur neuf aussi).
 */
import ts from 'typescript'
import { describe, expect, it } from 'vitest'
import {
  formatHits,
  lineOf,
  parseSource,
  scanSources,
  SCAN_TIMEOUT_MS,
  visitAll,
  type ScanHit,
  type ScannedSource,
} from '@/lib/archScan.testutil'

const FOYER = 'lib/layout/observeSize.ts'

const POSEURS_HORS_REJEU: Readonly<Record<string, string>> = {
  'components/charts/segmentLabelFit.ts': '2026-11-24 : ajustement des libellés d’un graphique, pas une largeur de zone',
  'components/foundations/useTabStripScroll.ts': '2026-11-24 : défilement de la barre d’onglets',
  'components/ui/carousel.tsx': '2026-11-24 : état de défilement du carrousel',
  'components/ui/combat-yield-display.tsx': '2026-11-24 : largeur d’un graphique de rendement, avec son propre repli',
  'features/match-replay/hooks/useTransportBarLines.ts': '2026-11-24 : lignes de la barre de lecture, mesure de plusieurs éléments',
  'features/palmares/BattlePassRewardCarousel.tsx': '2026-11-24 : état de défilement du carrousel de récompenses',
  'features/session-detail/SessionDetailPage.tsx': '2026-11-24 : mesure propre à la page de session',
  'features/synthesis/WeaponRecordsRuler.tsx': '2026-11-24 : largeur de la règle, avec un repli fixe (`useMeasuredWidth`)',
  'features/tactical/useTailleDuCanvas.ts': '2026-11-24 : taille de la toile de la Tactique',
  'features/_shared/assists/useElementWidth.ts': '2026-11-24 : largeur d’un diagramme, avec un repli fixe',
  'lib/echarts/followContainerSize.ts': '2026-11-24 : suivi de taille des graphiques ECharts',
}

/** Les `new ResizeObserver(...)` du fichier. */
function posesDObservateur(source: ScannedSource): ScanHit[] {
  const hits: ScanHit[] = []
  visitAll(source, (node) => {
    if (ts.isNewExpression(node) && ts.isIdentifier(node.expression) && node.expression.text === 'ResizeObserver') {
      hits.push({ file: source.rel, line: lineOf(source, node), detail: 'new ResizeObserver(...)' })
    }
  })
  return hits
}

function poseursParFichier(sources: readonly ScannedSource[]): Map<string, ScanHit[]> {
  const byFile = new Map<string, ScanHit[]>()
  for (const s of sources) {
    const hits = posesDObservateur(s)
    if (hits.length > 0) byFile.set(s.rel, hits)
  }
  return byFile
}

describe('garde-rail : un seul foyer pour l’observateur de taille', () => {
  const byFile = poseursParFichier(scanSources())

  it('le foyer pose l’observateur, une seule fois', () => {
    expect(byFile.get(FOYER)?.length).toBe(1)
  }, SCAN_TIMEOUT_MS)

  it('aucun autre poseur que le foyer et la liste datée (plafond)', () => {
    const hits = [...byFile.entries()]
      .filter(([file]) => file !== FOYER && !(file in POSEURS_HORS_REJEU))
      .flatMap(([, h]) => h)
    expect(hits, formatHits(hits, 'Suivre la taille par `observeSize` (lib/layout) ; sinon inscrire le fichier, daté, avec sa raison.')).toEqual([])
  }, SCAN_TIMEOUT_MS)

  it('cliquet : chaque inscrit pose encore un observateur', () => {
    const stale = Object.keys(POSEURS_HORS_REJEU).filter((file) => !byFile.has(file))
    expect(stale, `à retirer de POSEURS_HORS_REJEU : ${stale.join(', ')}`).toEqual([])
  }, SCAN_TIMEOUT_MS)

  it('auto-preuve : le détecteur voit une pose, pas un commentaire, un type ni une simple mention', () => {
    const n = (code: string): number => posesDObservateur(parseSource('features/x/a.ts', code)).length
    expect(n('const ro = new ResizeObserver(lire)')).toBe(1)
    expect(n('const ro = typeof ResizeObserver === "undefined" ? null : new ResizeObserver(settle)')).toBe(1)
    expect(n('// new ResizeObserver(lire)\nlet x: ResizeObserver | null = null\nvi.stubGlobal("ResizeObserver", Faux)')).toBe(0)
  })
})
