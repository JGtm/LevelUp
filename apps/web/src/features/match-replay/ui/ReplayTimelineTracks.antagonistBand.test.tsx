/**
 * Tests — la bande des titres dans l'hôte de la frise.
 *
 * L'hôte ne connaît pas les antagonistes : il reçoit une bande toute faite (`antagonistBand`) et la
 * pose AU-DESSUS de la rangée du joueur regardé, dans la grille des pistes. Ce que ces tests
 * protègent :
 *  1. la bande est AVANT la première rangée (le menu de point de vue et la piste du joueur) ;
 *  2. sans bande fournie, la frise est celle d'avant : rien n'est posé ;
 *  3. frise repliée, la bande n'existe pas : elle ne vit que dans les pistes.
 */
import { describe, expect, it } from 'vitest'

import { renderTracks } from '../test/timelineTracksHarness'

const BANDE = <div data-testid="bande-test" />

describe('ReplayTimelineTracks — l emplacement de la bande des titres', () => {
  it('pose la bande avant le menu de point de vue', () => {
    const { getByTestId, getByLabelText } = renderTracks({ antagonistBand: BANDE })
    const bande = getByTestId('bande-test')
    const menu = getByLabelText('Point de vue')
    // DOCUMENT_POSITION_FOLLOWING : le menu suit la bande dans l'ordre du document.
    expect(bande.compareDocumentPosition(menu) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
  })

  it('sans bande fournie, rien n est posé', () => {
    const { queryByTestId } = renderTracks()
    expect(queryByTestId('bande-test')).toBeNull()
  })

  it('frise repliée : pas de bande', () => {
    const { queryByTestId } = renderTracks({ antagonistBand: BANDE, tracksExpanded: false })
    expect(queryByTestId('bande-test')).toBeNull()
  })
})
