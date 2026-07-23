/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail A3.5/A3.8 (DC-9, 2026-07-10) + Lot D (2026-07-23) : le Lab est
 * retiré de l'app. Le dernier survivant, le panneau « Diagnostics d'instance »
 * (parité + garde-fous médailles, endpoint GET /lab/diagnostics), a été
 * supprimé au Lot D — le rapport de parité pointait un script Python de la
 * migration qui n'existe plus, et les garde-fous médailles sont déjà couverts
 * par cmd/refresh-metadata. Ce garde-rail reprend la protection
 * anti-résurrection de l'ex-lab_routes_mounted_test.go Go.
 *
 * Interdit la ré-introduction :
 *   1. des appels front aux endpoints back supprimés (/lab/resources,
 *      /lab/contracts, /lab/waypoint, /lab/diagnostics) ;
 *   2. des modules front supprimés (features/admin/lab, ResourcesPanel,
 *      LabHelp, useLabResources, useLabWaypoint, DiagnosticsPanel,
 *      features/lab/queries, useLabDiagnostics).
 *
 * Vérifie aussi que les redirections des anciennes URLs admin existent
 * (A3.1 : les liens partagés/bookmarks restent valides).
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

const SRC_ROOT = resolve(__dirname, '..', '..')

function walk(dir: string, out: string[] = []): string[] {
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) {
      walk(full, out)
    } else if (/\.(ts|tsx)$/.test(entry.name) && !entry.name.endsWith('.guard.test.ts')) {
      out.push(full)
    }
  }
  return out
}

describe('garde-rail retrait du Lab (A3.5)', () => {
  const files = walk(SRC_ROOT)

  it('aucune référence aux endpoints Lab supprimés', () => {
    const forbidden = ['/lab/resources', '/lab/contracts', '/lab/waypoint', '/lab/diagnostics']
    const offenders: string[] = []
    for (const file of files) {
      const content = readFileSync(file, 'utf8')
      for (const endpoint of forbidden) {
        if (content.includes(endpoint)) {
          offenders.push(`${file} → ${endpoint}`)
        }
      }
    }
    expect(offenders).toEqual([])
  })

  it('aucun import des modules Lab supprimés', () => {
    const forbidden = [
      'features/admin/lab',
      'features/lab/ResourcesPanel',
      'features/lab/LabHelp',
      'features/lab/DiagnosticsPanel',
      'features/lab/queries',
      'useLabResources',
      'useLabWaypoint',
      'useLabDiagnostics',
    ]
    const offenders: string[] = []
    for (const file of files) {
      const content = readFileSync(file, 'utf8')
      for (const name of forbidden) {
        if (content.includes(name)) {
          offenders.push(`${file} → ${name}`)
        }
      }
    }
    expect(offenders).toEqual([])
  })

  it('les anciennes URLs admin redirigent (A3.1)', () => {
    const routesDir = resolve(SRC_ROOT, 'routes', 'admin')
    const redirects: Record<string, string> = {
      'convergence.tsx': '/admin/data',
      'data-quality.tsx': '/admin/data',
      'logs.tsx': '/admin/system',
      'access.tsx': '/admin/management',
      'titles.tsx': '/admin/management',
      'lab.tsx': '/admin/management',
    }
    for (const [file, target] of Object.entries(redirects)) {
      const content = readFileSync(join(routesDir, file), 'utf8')
      expect(content, `${file} doit rediriger vers ${target}`).toContain('redirect(')
      expect(content, `${file} doit cibler ${target}`).toContain(`to: '${target}'`)
    }
  })
})
