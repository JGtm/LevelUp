import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { GrilleForm } from './GrilleForm'

/**
 * Une cellule sans valeur sur une ligne MESURÉE (grandeur optionnelle absente, ex.
 * prises nettes sur un match dont l'artefact n'a pas été lu) doit dire « non mesuré »
 * dans son infobulle, comme une ligne entière non mesurée — et jamais « : — ».
 * Constaté sur données réelles le 2026-09-14 (passe 4).
 */
describe("GrilleForm — infobulle d'une cellule non mesurée", () => {
  const rows = [
    { key: 'm1', label: '21:52 · Capture du drapeau', measured: true },
    { key: 'm2', label: '22:10 · Capture du drapeau', measured: false },
  ]
  const columns = [
    { key: 'grabs', label: 'Drapeaux capturés' },
    { key: 'net', label: 'Prises nettes' },
  ]

  it("nomme l'absence de mesure pour la seule grandeur absente comme pour la ligne entière", () => {
    const { container } = render(
      <GrilleForm
        rows={rows}
        columns={columns}
        value={(r, c) => (r.key === 'm1' && c.key === 'net' ? null : 3)}
        ink={() => 'var(--x)'}
        format={(v) => String(v)}
        tooltip={(r, c, text) => `${r.label} — ${c.label} : ${text}`}
        notMeasuredLabel="non mesuré (pas de film décodé)"
        axisTitle="gestes par match"
      />,
    )
    const labels = Array.from(container.querySelectorAll('[aria-label]')).map((e) =>
      e.getAttribute('aria-label'),
    )
    expect(labels).toContain('21:52 · Capture du drapeau — non mesuré (pas de film décodé)')
    expect(labels).toContain('22:10 · Capture du drapeau — non mesuré (pas de film décodé)')
    expect(labels).toContain('21:52 · Capture du drapeau — Drapeaux capturés : 3')
    expect(labels.some((l) => l != null && l.endsWith(': —'))).toBe(false)
  })
})
