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

/**
 * LA COUCHE NOMMÉE (schéma 62, lot 4.4.2) — et elle se dit dans LES DEUX LANGUES.
 *
 * CE QUE CES CAS PROTÈGENT, ET LE DÉFAUT QU'ILS FERMENT : le badge ne doit nommer une couche que
 * lorsque le module la PROUVE. Trois situations, et une seule doit produire la phrase — l'artefact
 * qui déclare sa couche de publication ET qui est en retard. Un artefact sans `layers` (antérieur
 * à 62) doit garder le libellé d'avant ce lot, sans quoi le badge attribuerait au producteur une
 * déclaration qu'il n'a pas faite ; un artefact à jour ne dit rien d'une couche, puisque rien n'a
 * bougé.
 */
describe('ReplaySchemaBadge — la couche de publication', () => {
  /** La table des calques telle que la cuisson l'écrit : la publication y porte SA version. */
  const layersDeLArtefact = (schemaVersion: number) => ({
    matchId: `publication-${schemaVersion}`,
    tracks: 'grammar-2026-09-15.42',
  })

  it('nomme la publication, en FR, quand elle est la seule couche prouvée', () => {
    render(
      <ReplaySchemaBadge
        isAdmin
        schemaVersion={EN_RETARD}
        latestSchemaVersion={PRODUCTEUR}
        locale="fr"
        layers={layersDeLArtefact(EN_RETARD)}
      />,
    )
    expect(screen.getByText(/seule la publication a changé/)).toBeInTheDocument()
  })

  it('la nomme en EN par la même donnée — parité tenue par le typage', () => {
    render(
      <ReplaySchemaBadge
        isAdmin
        schemaVersion={EN_RETARD}
        latestSchemaVersion={PRODUCTEUR}
        locale="en"
        layers={layersDeLArtefact(EN_RETARD)}
      />,
    )
    expect(screen.getByText(/publication layer only/)).toBeInTheDocument()
  })

  it('ne nomme AUCUNE couche sans `layers` — le libellé d’avant ce lot, à la lettre', () => {
    render(
      <ReplaySchemaBadge
        isAdmin
        schemaVersion={EN_RETARD}
        latestSchemaVersion={PRODUCTEUR}
        locale="fr"
      />,
    )
    expect(screen.queryByText(/publication/)).not.toBeInTheDocument()
    expect(
      screen.getByText(`Schéma ${EN_RETARD} · à recuire (dernier : ${PRODUCTEUR})`),
    ).toBeInTheDocument()
  })

  it('ne nomme aucune couche sur un artefact À JOUR, même s’il déclare la sienne', () => {
    render(
      <ReplaySchemaBadge
        isAdmin
        schemaVersion={PRODUCTEUR}
        latestSchemaVersion={PRODUCTEUR}
        locale="fr"
        layers={layersDeLArtefact(PRODUCTEUR)}
      />,
    )
    expect(screen.queryByText(/publication/)).not.toBeInTheDocument()
    expect(screen.getByText(/à jour/)).toBeInTheDocument()
  })
})
