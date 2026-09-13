/**
 * Tests — ReplaySchemaBadge (badge admin « version de schéma », lot A, 2026-09-11).
 *
 * CE QU'ILS PROTÈGENT : rien n'apparaît pour un non-admin (pas même un élément vide) ; le
 * libellé dit « à jour » quand l'artefact porte déjà la version courante du producteur,
 * « à recuire (dernier : N) » sinon, se limite à la version lue quand aucune comparaison
 * n'est possible (en-tête absent — artefact antérieur à ce lot), et NOMME le manquement quand
 * le document ne respecte pas le contrat (lot 0.B, 2026-09-13).
 *
 * AUCUN NUMÉRO DE SCHÉMA ÉCRIT EN DUR : les versions viennent de ce que Go publie
 * (`test/goFixtures.ts`) et du seuil déclaré par le web — cf. le ratchet `testDoc.guard.test.ts`.
 */
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'

import { MIN_RENDERABLE_SCHEMA_VERSION } from '../model/replaySchemaStatusLogic'
import { goFixtureSchemaVersion } from '../test/goFixtures'
import { ReplaySchemaBadge } from './ReplaySchemaBadge'

/** La version que le producteur écrit aujourd'hui, et une version en retard sur elle. */
const PRODUCTEUR = goFixtureSchemaVersion()
const EN_RETARD = MIN_RENDERABLE_SCHEMA_VERSION + 1

describe('ReplaySchemaBadge', () => {
  it("ne rend RIEN pour un non-admin, même si l'artefact est périmé", () => {
    const { container } = render(
      <ReplaySchemaBadge
        isAdmin={false}
        schemaVersion={EN_RETARD}
        latestSchemaVersion={PRODUCTEUR}
        locale="fr"
      />,
    )
    expect(container).toBeEmptyDOMElement()
  })

  it('dit "à jour" quand la version lue égale la version courante du producteur', () => {
    render(
      <ReplaySchemaBadge
        isAdmin
        schemaVersion={PRODUCTEUR}
        latestSchemaVersion={PRODUCTEUR}
        locale="fr"
      />,
    )
    expect(screen.getByText(`Schéma ${PRODUCTEUR} · à jour`)).toBeInTheDocument()
  })

  it('dit "à recuire (dernier : N)" quand l\'artefact est en retard', () => {
    render(
      <ReplaySchemaBadge
        isAdmin
        schemaVersion={EN_RETARD}
        latestSchemaVersion={PRODUCTEUR}
        locale="fr"
      />,
    )
    expect(
      screen.getByText(`Schéma ${EN_RETARD} · à recuire (dernier : ${PRODUCTEUR})`),
    ).toBeInTheDocument()
  })

  it("se limite à la version lue quand aucune comparaison n'est possible (en-tête absent)", () => {
    render(
      <ReplaySchemaBadge
        isAdmin
        schemaVersion={EN_RETARD}
        latestSchemaVersion={undefined}
        locale="fr"
      />,
    )
    expect(screen.getByText(`Schéma ${EN_RETARD}`)).toBeInTheDocument()
  })

  it('dit "à recuire" SANS cible sous le seuil d\'affichage, en-tête absent', () => {
    render(
      <ReplaySchemaBadge
        isAdmin
        schemaVersion={MIN_RENDERABLE_SCHEMA_VERSION - 1}
        latestSchemaVersion={undefined}
        locale="fr"
      />,
    )
    expect(
      screen.getByText(`Schéma ${MIN_RENDERABLE_SCHEMA_VERSION - 1} · à recuire`),
    ).toBeInTheDocument()
  })
  it('NOMME le manquement quand le document ne respecte pas le contrat', () => {
    render(
      <ReplaySchemaBadge
        isAdmin
        schemaVersion={PRODUCTEUR}
        latestSchemaVersion={PRODUCTEUR}
        contractIssue={{ kind: 'unknownKeys', keys: ['shotz'] }}
        locale="fr"
      />,
    )
    expect(
      screen.getByText(`Schéma ${PRODUCTEUR} · contrat non respecté (clé(s) inconnue(s) : shotz)`),
    ).toBeInTheDocument()
  })

  /**
   * LE MANQUEMENT SE DIT EN ANGLAIS EN LOCALE ANGLAISE (ronde 2 de la revue, constat R2-2).
   * Il était fabriqué côté schéma, en français : le badge anglais affichait
   * `contract violated (cle(s) inconnue(s) : shotz)` — un fragment FR hors d'`i18n.ts`, au
   * milieu d'une phrase EN, ce que la règle n° 1 du dépôt interdit. La LISTE DES CLÉS, elle,
   * reste brute des deux côtés : c'est une donnée, pas de la langue.
   */
  it('et il se dit en ANGLAIS en locale anglaise, liste de clés comprise', () => {
    render(
      <ReplaySchemaBadge
        isAdmin
        schemaVersion={PRODUCTEUR}
        latestSchemaVersion={PRODUCTEUR}
        contractIssue={{ kind: 'unknownKeys', keys: ['shotz'] }}
        locale="en"
      />,
    )
    expect(
      screen.getByText(`Schema ${PRODUCTEUR} · contract violated (unknown key(s): shotz)`),
    ).toBeInTheDocument()
  })

  it('dit le CHAMP fautif quand ce n’est pas une clé inconnue', () => {
    render(
      <ReplaySchemaBadge
        isAdmin
        schemaVersion={PRODUCTEUR}
        latestSchemaVersion={PRODUCTEUR}
        contractIssue={{ kind: 'invalidField', path: 'bounds.maxX', detail: 'expected number' }}
        locale="en"
      />,
    )
    expect(
      screen.getByText(`Schema ${PRODUCTEUR} · contract violated (bounds.maxX: expected number)`),
    ).toBeInTheDocument()
  })

  it('et sait le dire quand il ne sait rien de plus (document informe)', () => {
    render(
      <ReplaySchemaBadge
        isAdmin
        schemaVersion={PRODUCTEUR}
        latestSchemaVersion={PRODUCTEUR}
        contractIssue={{ kind: 'malformed' }}
        locale="en"
      />,
    )
    expect(
      screen.getByText(
        `Schema ${PRODUCTEUR} · contract violated (document does not match the contract)`,
      ),
    ).toBeInTheDocument()
  })

  it('bascule en anglais avec locale="en"', () => {
    render(
      <ReplaySchemaBadge
        isAdmin
        schemaVersion={PRODUCTEUR}
        latestSchemaVersion={PRODUCTEUR}
        locale="en"
      />,
    )
    expect(screen.getByText(`Schema ${PRODUCTEUR} · up to date`)).toBeInTheDocument()
  })
})
