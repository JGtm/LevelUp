/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail : LA JOINTURE FILM <-> BASE SE CONSTRUIT UNE FOIS PAR PAGE (règle <= 2 copies, CLAUDE.md n° 6).
 *
 * `buildPlayers` (`lib/replay/rosterLogic.ts`) joint les vies du film à la feuille du match. Sur la
 * page du rejeu, `model/replayModel.ts` la construit UNE fois et la donne à tous ses lecteurs (colonne
 * des équipes, identité des pions, zone du tueur, allégeance) : aucun ne la refait. Cinq modules la
 * refaisaient chacun (et la jointure d'un bot dépend de la feuille, d'où des copies qui divergent).
 *
 * LES AUTRES APPELANTS sont les surfaces de la page MATCH (onglets Arsenal et Nombres) : elles chargent
 * le document elles-mêmes (`useMatchReplay`) et n'ont pas de `ReplayModel`. Chacun est inscrit ici avec
 * sa raison ; la liste ne fait que rétrécir (un appelant inscrit qui n'appelle plus fait échouer le test,
 * un appelant neuf aussi).
 */
import { describe, expect, it } from 'vitest'
import {
  callsNamed,
  formatHits,
  parseSource,
  scanSources,
  SCAN_TIMEOUT_MS,
  type ScanHit,
  type ScannedSource,
} from '@/lib/archScan.testutil'

const NAME = new Set(['buildPlayers'])
const DEFINITION = 'lib/replay/rosterLogic.ts'

/** Le seul appelant de la page du rejeu. */
const REPLAY_MODEL = 'features/match-replay/model/replayModel.ts'

/** Les surfaces de la page Match (pas de `ReplayModel`), une entrée par fichier, datée. */
const MATCH_PAGE_CALLERS: Readonly<Record<string, string>> = {
  'features/match-replay/model/equipmentUsageLogic.ts':
    '2026-11-16 : bilan d’équipement de l’onglet Arsenal (document et feuille de la page Match, ~40 tests à signature)',
  'features/match-view/equipmentKillBadges.ts': '2026-11-16 : badges d’impact de la page Match, document et feuille propres',
  'features/match-view/MatchNumbersAdvantageChart.tsx':
    '2026-11-16 : graphe de l’onglet Nombres, qui charge son document par `useMatchReplay`',
  'lib/replay/filmAllegiance.ts':
    '2026-11-16 : `filmAllegianceOf`, l’allégeance des tables de l’onglet Arsenal montées sur la page Match',
}

/** Les dossiers `test/` portent le soutien des tests (montages, fixtures) : ils ne sont pas du code de page. */
const TEST_SUPPORT = /(^|\/)test\//

function buildPlayersCalls(source: ScannedSource): ScanHit[] {
  return source.rel === DEFINITION || TEST_SUPPORT.test(source.rel) ? [] : callsNamed(source, NAME)
}

function callsByFile(sources: readonly ScannedSource[]): Map<string, ScanHit[]> {
  const byFile = new Map<string, ScanHit[]>()
  for (const s of sources) {
    const hits = buildPlayersCalls(s)
    if (hits.length > 0) byFile.set(s.rel, hits)
  }
  return byFile
}

describe('garde-rail : buildPlayers n’est appelé que par son foyer et par les surfaces de la page Match', () => {
  const byFile = callsByFile(scanSources())

  it('la page du rejeu : un seul appel, dans model/replayModel.ts', () => {
    const replay = [...byFile.entries()].filter(
      ([file]) => file.startsWith('features/match-replay/') && file !== REPLAY_MODEL && !(file in MATCH_PAGE_CALLERS),
    )
    expect(
      replay.map(([file]) => file),
      'Lire `model.players` (ReplayModel) au lieu de refaire la jointure.',
    ).toEqual([])
    expect(byFile.get(REPLAY_MODEL)?.length).toBe(1)
  }, SCAN_TIMEOUT_MS)

  it('aucun appelant hors du modèle et de la liste des surfaces de la page Match (plafond)', () => {
    const hits = [...byFile.entries()]
      .filter(([file]) => file !== REPLAY_MODEL && !(file in MATCH_PAGE_CALLERS))
      .flatMap(([, h]) => h)
    expect(hits, formatHits(hits, 'Recevoir la jointure en paramètre ; sinon inscrire la surface (datée, avec sa raison).')).toEqual([])
  }, SCAN_TIMEOUT_MS)

  it('cliquet : chaque surface inscrite appelle encore buildPlayers, une fois', () => {
    const stale = Object.keys(MATCH_PAGE_CALLERS).filter((file) => byFile.get(file)?.length !== 1)
    expect(stale, `à retirer de MATCH_PAGE_CALLERS (ou à corriger) : ${stale.join(', ')}`).toEqual([])
  }, SCAN_TIMEOUT_MS)

  it('auto-preuve : le détecteur voit un appel, pas un commentaire, un import ni la définition', () => {
    const n = (rel: string, code: string): number => buildPlayersCalls(parseSource(rel, code)).length
    expect(n('features/x/a.ts', 'const p = buildPlayers(doc, board)')).toBe(1)
    expect(n('features/x/a.ts', 'const p = useMemo(() => rosterLogic.buildPlayers(d, b), [])')).toBe(1)
    expect(n('features/x/a.ts', "import { buildPlayers } from '@/lib/replay/rosterLogic'\n// buildPlayers(doc, board)")).toBe(0)
    expect(n(DEFINITION, 'export function buildPlayers(a: number) { return buildPlayers(a) }')).toBe(0)
  })
})
