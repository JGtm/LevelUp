/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail : la fermeture d'une couche (Échap, clic dehors) s'écrit UNE fois, dans
 * `lib/hooks/useDismissable`, jamais dans un composant.
 *
 * Motifs interdits dans un `.tsx` de `features/` : un écouteur `pointerdown` ou `mousedown` posé par
 * `addEventListener`, et une comparaison d'Échap (`===`, `!==`, `==`, `!=`). Plafond 0 partout, rejeu
 * compris. Les seuls fichiers admis sont ceux de `EXEMPTIONS`, chacun avec une raison datée : un geste
 * qui n'est pas la fermeture d'une couche, ou une fermeture que le hook ne sait pas exprimer. Une
 * exemption qui ne correspond plus à rien fait échouer le test.
 */
import ts from 'typescript'
import { describe, expect, it } from 'vitest'
import {
  lineOf,
  parseSource,
  scanSources,
  SCAN_TIMEOUT_MS,
  visitAll,
  type ScanHit,
  type ScannedSource,
} from '@/lib/archScan.testutil'

const REPLAY = 'features/match-replay/'

/**
 * Fichiers de `features/` (hors rejeu) admis à porter un motif, et pourquoi. Chaque raison commence par
 * sa date (AAAA-MM-JJ).
 */
const EXEMPTIONS: Record<string, string> = {
  'features/asset-drawer/AssetDockedPanel.tsx':
    "2026-11-17 : Échap est lu par le gestionnaire clavier du volet lui-même et ne ferme que le volet ; un écouteur de document le ferait réagir hors du volet.",
  'features/tactical/TacticalAnalysisView.tsx':
    "2026-11-17 : deux Échap en pile avec condition (refermer la mort choisie, puis écarter la zone), ignorés quand la frappe est déjà traitée ou vient d'un champ de saisie.",
  'features/tactical/TacticalPlanTueurs.tsx':
    "2026-11-17 : appui sur la toile capté pour distinguer un clic d'un glisser ; aucune couche ne se ferme.",
  'features/squad/SquadFocusStrip.tsx':
    "2026-11-17 : Échap annule le renommage dans son champ de saisie ; aucune couche ne se ferme.",
  'features/squad/useSquadPresets.tsx':
    "2026-11-17 : Échap annule le renommage d'un préréglage dans son champ de saisie ; aucune couche ne se ferme.",
}

const GESTES_DEHORS = new Set(['pointerdown', 'mousedown'])

/** Les quatre comparaisons : `===`, `!==`, `==`, `!=`. */
const COMPARAISONS = new Set<ts.SyntaxKind>([
  ts.SyntaxKind.EqualsEqualsEqualsToken,
  ts.SyntaxKind.ExclamationEqualsEqualsToken,
  ts.SyntaxKind.EqualsEqualsToken,
  ts.SyntaxKind.ExclamationEqualsToken,
])

function isLiteral(node: ts.Node, value: string): boolean {
  return ts.isStringLiteralLike(node) && node.text === value
}

/**
 * Un `addEventListener('pointerdown' | 'mousedown', ...)` ou une comparaison d'Échap (`===`, `!==`,
 * `==`, `!=`, dans un sens ou dans l'autre, guillemets simples ou doubles).
 */
function dismissPatterns(source: ScannedSource): ScanHit[] {
  const hits: ScanHit[] = []
  visitAll(source, (node) => {
    if (
      ts.isCallExpression(node) &&
      ts.isPropertyAccessExpression(node.expression) &&
      node.expression.name.text === 'addEventListener' &&
      node.arguments[0] !== undefined &&
      ts.isStringLiteralLike(node.arguments[0]) &&
      GESTES_DEHORS.has(node.arguments[0].text)
    ) {
      hits.push({ file: source.rel, line: lineOf(source, node), detail: `addEventListener('${node.arguments[0].text}')` })
    }
    if (
      ts.isBinaryExpression(node) &&
      COMPARAISONS.has(node.operatorToken.kind) &&
      (isLiteral(node.left, 'Escape') || isLiteral(node.right, 'Escape'))
    ) {
      hits.push({ file: source.rel, line: lineOf(source, node), detail: `${node.operatorToken.getText(node.getSourceFile())} 'Escape'` })
    }
  })
  return hits
}

const tsxDeFeatures = () => scanSources({ under: 'features', tsxOnly: true })

describe('garde-rail : la fermeture d’une couche passe par useDismissable', () => {
  it('plafond 0 : aucun .tsx de features/match-replay ne l’écrit', () => {
    const hits = tsxDeFeatures()
      .filter((s) => s.rel.startsWith(REPLAY))
      .flatMap(dismissPatterns)
    expect(
      hits.map((h) => `${h.file}:${h.line} ${h.detail}`),
      'Fermer une couche : useDismissable (@/lib/hooks/useDismissable).',
    ).toEqual([])
  }, SCAN_TIMEOUT_MS)

  it('hors rejeu : seuls les fichiers exemptés portent un motif, et chaque exemption en porte encore un', () => {
    const files = new Set(
      tsxDeFeatures()
        .filter((s) => !s.rel.startsWith(REPLAY))
        .flatMap(dismissPatterns)
        .map((h) => h.file),
    )
    const nonExemptes = [...files].filter((f) => !(f in EXEMPTIONS)).sort()
    expect(nonExemptes, 'Fermer une couche : useDismissable (@/lib/hooks/useDismissable).').toEqual([])
    const perimees = Object.keys(EXEMPTIONS).filter((f) => !files.has(f)).sort()
    expect(perimees, 'Exemption sans objet : la retirer de EXEMPTIONS.').toEqual([])
  }, SCAN_TIMEOUT_MS)

  it('chaque exemption porte une raison datée', () => {
    for (const [file, raison] of Object.entries(EXEMPTIONS)) {
      expect(raison, file).toMatch(/^\d{4}-\d{2}-\d{2} : \S/)
    }
  })

  it('auto-preuve : le détecteur voit les motifs, pas un autre événement ni un autre nom', () => {
    const n = (code: string): number => dismissPatterns(parseSource('features/x/a.tsx', code)).length
    expect(n("document.addEventListener('pointerdown', f)")).toBe(1)
    expect(n("el.addEventListener('mousedown', f)")).toBe(1)
    expect(n("if (e.key === 'Escape') close()")).toBe(1)
    expect(n("if ('Escape' === e.key) close()")).toBe(1)
    expect(n('if (e.key === "Escape") close()')).toBe(1)
    expect(n("if (e.key !== 'Escape') return")).toBe(1)
    expect(n('if (e.key !== "Escape") return')).toBe(1)
    expect(n("if (e.key == 'Escape') close()")).toBe(1)
    expect(n("if (e.key != 'Escape') return")).toBe(1)
    expect(n("if ('Escape' !== e.key) return")).toBe(1)
    expect(n("document.addEventListener('keydown', f)")).toBe(0)
    expect(n("if (e.key === 'Enter') return")).toBe(0)
    expect(n("const s = \"key === 'Escape'\"")).toBe(0)
  })
})
