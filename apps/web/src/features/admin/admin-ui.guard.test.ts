/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rails d'harmonisation UI admin (A8.1 / A8.2) — une factorisation sans
 * garde-rail re-diverge (CLAUDE.md règle 6).
 *
 *  1. A8.1 — AdminKpi est LE composant KPI : interdiction de re-déclarer un
 *     composant local `function XxxKpi(` / `SummaryCell` / `DataHealthMetric`
 *     sous features/admin/ (hors composant canonique).
 *  2. A8.2 — useCounterSnapshot est LE hook du pattern « baseline roulante » :
 *     interdiction d'appeler readCountersSnapshot(/writeCountersSnapshot( hors
 *     de countersTrend.ts (définitions), du hook, et des tests.
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

const ADMIN_ROOT = resolve(__dirname)

function walk(dir: string, out: string[] = []): string[] {
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) {
      walk(full, out)
    } else if (/\.(ts|tsx)$/.test(entry.name)) {
      out.push(full)
    }
  }
  return out
}

const isTest = (f: string) => /\.test\.tsx?$/.test(f) || f.endsWith('.guard.test.ts')
const norm = (f: string) => f.replace(/\\/g, '/')

describe('garde-rails UI admin (A8)', () => {
  const files = walk(ADMIN_ROOT)

  it('A8.1 — aucun composant KPI local hors AdminKpi', () => {
    const kpiDecl = /function\s+(\w*Kpi|SummaryCell|DataHealthMetric)\s*\(/
    const offenders: string[] = []
    for (const file of files) {
      if (isTest(file) || norm(file).endsWith('components/AdminKpi.tsx')) continue
      const content = readFileSync(file, 'utf8')
      const m = content.match(kpiDecl)
      if (m) offenders.push(`${norm(file)} → ${m[1]}`)
    }
    expect(offenders).toEqual([])
  })

  it('A8.2 — readCountersSnapshot/writeCountersSnapshot hors du hook interdits', () => {
    const offenders: string[] = []
    for (const file of files) {
      const n = norm(file)
      if (isTest(file) || n.endsWith('/countersTrend.ts') || n.endsWith('/useCounterSnapshot.ts')) {
        continue
      }
      const content = readFileSync(file, 'utf8')
      if (/(readCountersSnapshot|writeCountersSnapshot)\s*\(/.test(content)) {
        offenders.push(n)
      }
    }
    expect(offenders).toEqual([])
  })
})
