/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail A3.5/A3.8 (DC-9, 2026-07-10) : le Lab est retiré de l'app.
 *
 * Interdit la ré-introduction :
 *   1. des appels front aux endpoints back supprimés (/lab/resources,
 *      /lab/contracts, /lab/waypoint) — seul /lab/diagnostics survit
 *      (panneau Diagnostics de l'onglet Données) ;
 *   2. des modules front supprimés (features/admin/lab, ResourcesPanel,
 *      LabHelp, useLabResources, useLabWaypoint).
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
    const forbidden = ['/lab/resources', '/lab/contracts', '/lab/waypoint']
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
      'useLabResources',
      'useLabWaypoint',
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
