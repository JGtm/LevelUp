/**
 * Tests — ReplaySchemaBadge (badge admin « version de schéma », lot A, 2026-09-11).
 *
 * CE QU'ILS PROTÈGENT : rien n'apparaît pour un non-admin (pas même un élément vide) ; le
 * libellé dit « à jour » quand l'artefact porte déjà la version courante du producteur,
 * « à recuire (dernier : N) » sinon, et se limite à la version lue quand aucune comparaison
 * n'est possible (en-tête absent — artefact antérieur à ce lot).
 */
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'

import { ReplaySchemaBadge } from './ReplaySchemaBadge'

describe('ReplaySchemaBadge', () => {
  it("ne rend RIEN pour un non-admin, même si l'artefact est périmé", () => {
    const { container } = render(
      <ReplaySchemaBadge isAdmin={false} schemaVersion={48} latestSchemaVersion={51} locale="fr" />,
    )
    expect(container).toBeEmptyDOMElement()
  })

  it('dit "à jour" quand la version lue égale la version courante du producteur', () => {
    render(<ReplaySchemaBadge isAdmin schemaVersion={51} latestSchemaVersion={51} locale="fr" />)
    expect(screen.getByText('Schéma 51 · à jour')).toBeInTheDocument()
  })

  it('dit "à recuire (dernier : N)" quand l\'artefact est en retard', () => {
    render(<ReplaySchemaBadge isAdmin schemaVersion={48} latestSchemaVersion={51} locale="fr" />)
    expect(screen.getByText('Schéma 48 · à recuire (dernier : 51)')).toBeInTheDocument()
  })

  it("se limite à la version lue quand aucune comparaison n'est possible (en-tête absent)", () => {
    render(
      <ReplaySchemaBadge isAdmin schemaVersion={42} latestSchemaVersion={undefined} locale="fr" />,
    )
    expect(screen.getByText('Schéma 42')).toBeInTheDocument()
  })

  it('bascule en anglais avec locale="en"', () => {
    render(<ReplaySchemaBadge isAdmin schemaVersion={51} latestSchemaVersion={51} locale="en" />)
    expect(screen.getByText('Schema 51 · up to date')).toBeInTheDocument()
  })
})
