/**
 * Tests — MatchEquipmentUsageSection (le bilan d'équipement de la page match).
 *
 * CE QU'ILS PROTÈGENT, et ce sont les cinq promesses de la section :
 *   1. LA DOUBLE PORTE. Sans artefact — la quasi-totalité des matchs en production — la
 *      section ne rend RIEN ; avec un artefact qui ne porte aucune grandeur, non plus. Un cadre
 *      vide répété sur chaque page de match est une promesse non tenue à l'infini.
 *   2. LES DEUX VUES, et les colonnes que LA DONNÉE justifie — jamais une liste en dur.
 *   3. LA PART D'UNE ÉQUIPE est la somme des gestes de ses joueurs, pas celle du match, et le
 *      COMPTE BRUT est écrit à côté du pourcentage.
 *   4. AUCUN TEXTE DE PIED (décision utilisateur 2026-09-14) : ce qui n'entre pas dans les deux
 *      vues — gestes sans propriétaire, poses d'origine inconnue — se dit en UNE phrase au
 *      survol du TITRE, et les socles de bonus vidés se lisent dans le bloc voisin.
 *   5. CE QUI N'EST PAS MESURÉ SE DIT. Aucune colonne pour le répulseur ni le propulseur, et une
 *      phrase le dit au survol du groupe « Équipement » — mais pas la même raison pour les deux
 *      depuis le 2026-09-03 : le répulseur n'a AUCUN canal (une colonne de zéros se lirait « zéro usage »),
 *      le propulseur en a un (schéma 38) mais son geste se lit sur la CARTE, pas ici. Et une
 *      grandeur non mesurée écrit « — » là où un zéro se lirait comme une mesure.
 *
 * LES VALEURS SE LISENT PAR LE NOM ACCESSIBLE DES BARRES (`gridTipFmt` : joueur — grandeur :
 * valeur). C'est ce que porte l'écran depuis que la section est un graphe, et c'est aussi ce
 * qu'entend un lecteur d'écran : l'éprouver ici éprouve les deux d'un coup.
 *
 * Le calcul est éprouvé chez `equipmentUsageLogic.test.ts` et `valueGridModel.test.ts` ; ici on
 * éprouve le RENDU.
 */
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import type { MatchScoreboardRow, ReplayDocument } from '@/lib/api/types'

import { REPLAY_TEXT } from './i18n/i18n'
import { MatchEquipmentUsageSection } from './MatchEquipmentUsageSection'
import { testReplayDoc } from './test/testDoc'

// La lecture de l'artefact est la SEULE frontière réseau du composant : on la pilote, et tout
// le reste (agrégation, colonnes, libellés) reste le vrai code. Même patron que
// `MatchScoreCurveChart.test.tsx`.
const artefact = vi.hoisted(() => ({ current: undefined as unknown }))
vi.mock('../../lib/replay/queries', () => ({ useMatchReplay: () => ({ data: artefact.current }) }))

const t = REPLAY_TEXT.fr

const SCOREBOARD = [
  { xuid: 'a1', gamertag: 'Alpha', team_side: 't0', is_me: true },
  { xuid: 'a2', gamertag: 'Bravo', team_side: 't0' },
  { xuid: 'b1', gamertag: 'Charlie', team_side: 't1' },
] as unknown as MatchScoreboardRow[]

/** Une vie du film : le slot, son propriétaire, et de quoi donner une fenêtre. */
function vie(slot: number, xuid: string) {
  return {
    slot,
    xuid,
    team: -1,
    startFrame: 0,
    endFrame: 100,
    points: [
      { t: 0, x: 0, y: 0 },
      { t: 100, x: 1, y: 1 },
    ],
  }
}

/**
 * LE TÉMOIN DE RENDU : quatre joueurs (dont un que le scoreboard ignore), un geste de chaque
 * canal, et deux vidages de socle de bonus. Frames à 100 ms.
 */
const TEMOIN: Partial<ReplayDocument> = {
  frameCount: 200,
  frameIntervalMs: 100,
  roster: [
    { filmIndex: 0, xuid: 'a1', name: 'Alpha' },
    { filmIndex: 1, xuid: 'a2', name: 'Bravo' },
    { filmIndex: 2, xuid: 'b1', name: 'Charlie' },
    { filmIndex: 3, xuid: 'orphelin', name: 'Delta' },
  ],
  tracks: [vie(1, 'a1'), vie(2, 'a2'), vie(3, 'b1'), vie(4, 'orphelin')],
  grappleLines: [
    { slot: 1, t0: 1, t1: 5, ax: 0, ay: 0 },
    { slot: 2, t0: 2, t1: 6, ax: 0, ay: 0 },
    { slot: 3, t0: 3, t1: 7, ax: 0, ay: 0 },
  ],
  equipmentEpisodes: [
    { slot: 1, fam: 'camo', t0: 10, t1: 60 },
    { slot: 2, fam: 'overshield', t0: 0, t1: 30 },
  ],
  equipmentPlacements: [
    { family: 'sensor', origin: 'deployed', owner: 1, id: '0x1', t0: 5, t1: 9, x: 0, y: 0 },
    { family: 'sensor', origin: 'deployed', owner: 2, id: '0x1', t0: 5, t1: 9, x: 0, y: 0 },
    { family: 'repair_field', origin: 'dropped', owner: 3, id: '0x2', t0: 5, t1: 9, x: 0, y: 0 },
  ],
  grenades: [{ slot: 1, rank: 0, t: 5, i: 0, s: 'x', x: 0, y: 0 }],
  grenadeLabels: [{ fr: 'Fragmentation', en: 'Frag' }],
  weaponPads: [{ weapon: 'powerup_overshield', x: 0, y: 0, spawns: [], presence: [] }],
  padPickups: [
    { pad: 0, tLow: 10, tHigh: 20, xuid: null },
    { pad: 0, tLow: 60, tHigh: 70, xuid: null },
  ],
  coverage: {
    equipment: { tracksTotal: 40, camoLives: 1, camoEpisodes: 1, overshieldLives: 1, overshieldEpisodes: 1 },
    grapple: { pulls: 3, pullLives: 3, lightReads: 4, heavyReads: 3, unpairedFires: 1, brokenBodies: 0 },
    groundWeapons: { powerupPads: 1 },
  },
} as unknown as Partial<ReplayDocument>

function poserArtefact(over: Partial<ReplayDocument> | null) {
  artefact.current = over ? testReplayDoc(over) : undefined
}

function afficher(locale: 'fr' | 'en' = 'fr') {
  return render(
    <MatchEquipmentUsageSection
      playerSlug="joueur"
      matchId="m1"
      replayAvailable
      scoreboard={SCOREBOARD}
      locale={locale}
    />,
  )
}

/**
 * Déplie le repli « game changers » (plan 2026-09-05). Sur le TÉMOIN, deux colonnes sont
 * repliées par défaut : le grappin et le champ de réparation lâché.
 */
function deplier(vue: ReturnType<typeof afficher>, count = 2) {
  fireEvent.click(vue.getByRole('button', { name: t.collapsedColumnsShowFmt(count) }))
}

/**
 * survolerTitre — ouvre (ou tente d'ouvrir) l'infobulle du TITRE de la carte, seul endroit où
 * la réserve de couverture se dit depuis le 2026-09-14. Sans réserve, rien ne s'ouvre.
 */
function survolerTitre(vue: ReturnType<typeof afficher>, titre = t.equipmentUsage.title) {
  fireEvent.mouseEnter(vue.getByText(titre).parentElement as Element)
}

describe('MatchEquipmentUsageSection — la double porte', () => {
  it('ne rend RIEN sans artefact : 404 = pas de film, pas de cadre vide', () => {
    poserArtefact(null)
    expect(afficher().container.firstChild).toBeNull()
  })

  it('ne rend rien quand l’artefact existe mais ne porte AUCUNE grandeur mesurée', () => {
    poserArtefact({ tracks: [vie(1, 'a1')] } as Partial<ReplayDocument>)
    expect(afficher().container.firstChild).toBeNull()
  })

  it('ne rend rien quand le film ne porte que des gestes SANS propriétaire', () => {
    poserArtefact({
      tracks: [{ slot: 9, team: -1, startFrame: 0, endFrame: 10, points: [{ t: 0, x: 0, y: 0 }] }],
      grappleLines: [{ slot: 9, t0: 1, t1: 5, ax: 0, ay: 0 }],
    } as unknown as Partial<ReplayDocument>)
    expect(afficher().container.firstChild).toBeNull()
  })
})

describe('MatchEquipmentUsageSection — les deux vues', () => {
  it('rend la carte et ses deux vues, chacune nommée', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    expect(vue.getByRole('region', { name: t.equipmentUsage.title })).toBeTruthy()
    expect(vue.getByRole('region', { name: t.equipmentUsage.viewByPlayer })).toBeTruthy()
    expect(vue.getByRole('region', { name: t.equipmentUsage.viewTeamShare })).toBeTruthy()
  })

  it('montre une FAMILLE par canal mesuré, en légende comme en part d’équipe (déplié)', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    // Le grappin et les objets lâchés du témoin sont repliés par défaut (vote du 05/09).
    deplier(vue)
    for (const famille of [
      t.equipmentUsage.groupGrapple,
      t.equipmentUsage.groupActive,
      // E2 (2026-09-09) : `groupDeployed`/`groupDropped` ont fusionné en UNE colonne
      // « équipement » par famille (P2/P3) — plus de section séparée à chercher ici.
      t.equipmentUsage.groupEquipment,
    ]) {
      expect(vue.getAllByText(famille).length).toBeGreaterThan(0)
    }
  })

  it('nomme les colonnes par les tables EXISTANTES du rejeu, jamais un nom en dur', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    // Le champ de réparation lâché est replié par défaut : on déplie pour lire son nom.
    deplier(vue)
    // Familles de pose : libellés de `placementFamily` (par règle de rendu).
    expect(vue.getByText(t.placementFamily.sensor)).toBeTruthy()
    expect(vue.getByText(t.placementFamily.field)).toBeTruthy()
    // AUCUNE colonne de grenade depuis le 2026-09-13 (retrait demandé par l'utilisateur).
    expect(vue.queryByText('Fragmentation')).toBeNull()
    // États actifs : UNE colonne par famille, nommée par la famille seule — plus de suffixe
    // « (épisodes) » ni « (durée) », plus de colonne « Frags sous … ».
    expect(vue.getByText(t.equipmentUsage.activeColumnFmt(t.equipmentUsage.activeFamily.camo))).toBeTruthy()
    expect(vue.getByText(t.equipmentUsage.activeColumnFmt(t.equipmentUsage.activeFamily.overshield))).toBeTruthy()
  })

  it('rend une ligne par joueur, y compris celui que le scoreboard ignore', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    for (const nom of ['Alpha', 'Bravo', 'Charlie', 'Delta']) {
      expect(vue.getByText(nom)).toBeTruthy()
    }
  })

  it('écrit la valeur de chaque barre dans son nom accessible : joueur, grandeur, valeur', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    // La colonne du grappin est repliée par défaut : on la déplie pour lire ses barres.
    deplier(vue)
    const tip = t.equipmentUsage.gridTipFmt
    expect(vue.getByLabelText(tip('Alpha', t.equipmentUsage.groupGrapple, '1'))).toBeTruthy()
    expect(vue.getByLabelText(tip('Delta', t.equipmentUsage.groupGrapple, '0'))).toBeTruthy()
  })

  it('la durée cumulée et les frags passent dans l’INFOBULLE de la cellule, plus dans une colonne', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    const u = t.equipmentUsage
    const tip = u.gridTipFmt
    expect(
      vue.getByLabelText(tip('Alpha', u.activeColumnFmt(u.activeFamily.camo), u.activeCellTipFmt(1, '0:05', null))),
    ).toBeTruthy()
    // Bravo n'a aucun épisode de camouflage : la mesure a eu lieu et vaut zéro.
    expect(
      vue.getByLabelText(tip('Bravo', u.activeColumnFmt(u.activeFamily.camo), u.activeCellTipFmt(0, '0:00', null))),
    ).toBeTruthy()
  })

  it('un épisode MESURÉ de durée nulle s’écrit « 0:00 », pas « — »', () => {
    // t1 === t0 : l'épisode a bien été observé, sa durée vaut zéro. Le nombre d'épisodes
    // dit « 1 » ; la durée doit dire « 0:00 » et non un repli d'absence.
    poserArtefact({
      ...TEMOIN,
      equipmentEpisodes: [
        { slot: 1, fam: 'camo', t0: 40, t1: 40 },
        { slot: 2, fam: 'overshield', t0: 0, t1: 30 },
      ],
    } as unknown as Partial<ReplayDocument>)
    const vue = afficher()
    const u = t.equipmentUsage
    expect(
      vue.getByLabelText(
        u.gridTipFmt('Alpha', u.activeColumnFmt(u.activeFamily.camo), u.activeCellTipFmt(1, '0:00', null)),
      ),
    ).toBeTruthy()
  })

  it('range le joueur HORS SCOREBOARD sous « équipe inconnue », jamais dans un camp nommé', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    // Son nom porte son camp en infobulle de ligne — et ce camp est l'inconnu.
    expect(vue.getByText('Delta').closest('[title]')?.getAttribute('title')).toContain(
      t.teamUnknown,
    )
  })
})

describe('MatchEquipmentUsageSection — la part de chaque équipe', () => {
  it('somme les gestes du CAMP, pas ceux du match, et écrit le compte brut avec la part', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    // La famille du grappin est repliée par défaut : sa ligne de part n'existe que dépliée.
    deplier(vue)
    // Grappin : Alpha + Bravo = 2 tractions pour le camp t0, sur les 3 du match.
    const segment = vue.getByLabelText(
      t.equipmentUsage.shareTipFmt('Équipe Eagle', t.equipmentUsage.groupGrapple, 2, 3),
    )
    expect(segment.textContent).toBe('2 · 67 %')
    expect(
      vue.getByLabelText(
        t.equipmentUsage.shareTipFmt('Équipe Cobra', t.equipmentUsage.groupGrapple, 1, 3),
      ).textContent,
    ).toBe('1 · 33 %')
  })

  it('ne rend aucune ligne pour une famille qu’aucun camp n’a employée', () => {
    poserArtefact({ ...TEMOIN, grappleLines: [] } as Partial<ReplayDocument>)
    const vue = afficher()
    // Plus de tractions : ni colonne, ni ligne de part, ni entrée de légende.
    expect(vue.queryByText(t.equipmentUsage.groupGrapple)).toBeNull()
  })

  it('porte la RÉSERVE d’une famille en infobulle de son NOM, dans la vue des parts', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    // Le nom de la famille apparaît en légende de la vue 1 PUIS en tête de sa ligne de part :
    // c'est cette dernière occurrence qui porte la réserve de mesure.
    const noms = vue.getAllByText(t.equipmentUsage.groupActive)
    fireEvent.mouseEnter(noms[noms.length - 1].parentElement as Element)
    expect(screen.getByRole('tooltip').textContent).toBe(t.equipmentUsage.groupActiveHint)
  })
})

describe('MatchEquipmentUsageSection — ce que l’écran DIT de sa mesure', () => {
  it('ne pose AUCUN texte de pied sous le bloc (décision utilisateur 2026-09-14)', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    // Le pied de carte a porté successivement le paragraphe répulseur/propulseur, la ligne des
    // socles de bonus vidés, les dénominateurs de couverture et les deux réserves. Il ne porte
    // plus rien : aucune de ces phrases ne doit revenir sous les deux vues.
    expect(vue.queryByText(/Socles de bonus|États actifs mesurés|traction.* de grappin lue/)).toBeNull()
    expect(vue.queryByText(/hors des deux vues|origine inconnue/)).toBeNull()
  })

  it('dit la RÉSERVE au survol du TITRE, en une phrase', () => {
    poserArtefact({
      ...TEMOIN,
      coverage: {
        ...TEMOIN.coverage,
        placements: { byFamilyOrigin: { 'sensor/deployed': 2, 'wall/unknown': 3 } },
      },
    } as unknown as Partial<ReplayDocument>)
    const vue = afficher()
    survolerTitre(vue)
    expect(screen.getByRole('tooltip').textContent).toBe(t.equipmentUsage.coverageReserveFmt(3))
  })

  it('ne dit AUCUNE réserve quand rien ne la justifie (titre nu)', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    survolerTitre(vue)
    expect(screen.queryByRole('tooltip')).toBeNull()
  })

  it('ne rend JAMAIS les objets pris sans famille connue (décision utilisateur 2026-09-09)', () => {
    poserArtefact({
      ...TEMOIN,
      // Aucune table `abilityLabels` : le rang 5 n'a aucun nom dans ce film. Il est compté par la
      // logique (`unnamedTaken`, outillage d'investigation) mais aucune ligne ni note ne le montre.
      equipmentChanges: [{ t: 5, slot: 1, kind: 'taken', r: 5, from: -1 }],
    } as unknown as Partial<ReplayDocument>)
    const vue = afficher()
    expect(vue.queryByText(/sans famille connue|without a known family/i)).toBeNull()
  })

  it('n’ouvre AUCUNE colonne pour le répulseur ni le propulseur', () => {
    poserArtefact({
      ...TEMOIN,
      equipmentPlacements: [
        { family: 'repulsor', origin: 'deployed', owner: 1, id: '0x3', t0: 5, t1: 9, x: 0, y: 0 },
        { family: 'thruster', origin: 'dropped', owner: 1, id: '0x4', t0: 5, t1: 9, x: 0, y: 0 },
      ],
    } as unknown as Partial<ReplayDocument>)
    const vue = afficher()
    // Le témoin garde les épisodes camo/surbouclier de TEMOIN (non écrasés) : la colonne
    // « équipement » existe donc toujours pour EUX (E2, décision D9 amendée), mais ni le
    // répulseur ni le propulseur n'y ouvrent de colonne — c'est ce que ce test protège.
    expect(vue.queryByText('repulsor')).toBeNull()
    expect(vue.queryByText('thruster')).toBeNull()
  })

  it('compte à part les gestes mesurés sans propriétaire, hors des vues', () => {
    poserArtefact({
      ...TEMOIN,
      equipmentPlacements: [
        { family: 'sensor', origin: 'deployed', owner: 1, id: '0x1', t0: 5, t1: 9, x: 0, y: 0 },
        { family: 'sensor', origin: 'deployed', owner: -1, id: '0x1', t0: 5, t1: 9, x: 0, y: 0 },
      ],
    } as unknown as Partial<ReplayDocument>)
    const vue = afficher()
    survolerTitre(vue)
    expect(screen.getByRole('tooltip').textContent).toBe(
      REPLAY_TEXT.fr.equipmentUsage.coverageReserveFmt(1),
    )
  })
})

describe('MatchEquipmentUsageSection — le repli « game changers » (plan 2026-09-05)', () => {
  it('REPLIE PAR DÉFAUT les familles hors vote : ni colonne, ni légende, ni ligne de part', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    // Le grappin (voté replié) est invisible d'emblée ; le capteur (élu) est là.
    expect(vue.queryByText(t.equipmentUsage.groupGrapple)).toBeNull()
    expect(vue.queryByText(t.placementFamily.field)).toBeNull()
    expect(vue.getByText(t.placementFamily.sensor)).toBeTruthy()
    // Le bouton porte le COMPTE des colonnes masquées : grappin + champ de réparation lâché.
    expect(vue.getByRole('button', { name: t.collapsedColumnsShowFmt(2) })).toBeTruthy()
  })

  it('« Voir plus (N) » révèle les colonnes repliées APRÈS les élues, puis « Replier »', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    deplier(vue)
    expect(vue.getAllByText(t.equipmentUsage.groupGrapple).length).toBeGreaterThan(0)
    expect(vue.getByText(t.placementFamily.field)).toBeTruthy()
    // Le bouton s'est renversé, et un clic replie tout à nouveau.
    fireEvent.click(vue.getByRole('button', { name: t.collapsedColumnsHide }))
    expect(vue.queryByText(t.equipmentUsage.groupGrapple)).toBeNull()
  })

  it('zéro colonne repliée = AUCUN bouton', () => {
    // Tout ce que ce film mesure est élu (capteur, épisodes) ou hors vote (grenades).
    poserArtefact({
      ...TEMOIN,
      grappleLines: [],
      equipmentPlacements: [
        { family: 'sensor', origin: 'deployed', owner: 1, id: '0x1', t0: 5, t1: 9, x: 0, y: 0 },
      ],
    } as unknown as Partial<ReplayDocument>)
    const vue = afficher()
    expect(vue.queryByRole('button', { name: t.collapsedColumnsHide })).toBeNull()
    expect(vue.queryByRole('button', { name: /Voir plus/ })).toBeNull()
  })

  it('les GRENADES n’ont plus aucune colonne (retrait utilisateur du 2026-09-13)', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    deplier(vue)
    expect(vue.queryByText('Fragmentation')).toBeNull()
  })
})

describe('MatchEquipmentUsageSection — parité FR/EN', () => {
  it('EN : titre, vues et familles passent en anglais', () => {
    poserArtefact(TEMOIN)
    const vue = afficher('en')
    const en = REPLAY_TEXT.en
    expect(vue.getByRole('region', { name: en.equipmentUsage.title })).toBeTruthy()
    expect(vue.getByRole('region', { name: en.equipmentUsage.viewByPlayer })).toBeTruthy()
    expect(vue.getAllByText(en.equipmentUsage.groupEquipment).length).toBeGreaterThan(0)
    expect(vue.getByText(en.placementFamily.sensor)).toBeTruthy()
    // États actifs : en-tête EN, pas la clé FR.
    expect(vue.getByText(en.equipmentUsage.activeColumnFmt(en.equipmentUsage.activeFamily.camo))).toBeTruthy()
  })
})

describe('MatchEquipmentUsageSection — frags sous effet actif (LOT F.2)', () => {
  it('dit « non mesurés », jamais 0, quand la jointure n’a pas pu être tentée (killsRead faux)', () => {
    // TEMOIN ne pose pas `coverage.equipment.killsRead` : la jointure est réputée non tentée.
    poserArtefact(TEMOIN)
    const vue = afficher()
    const u = t.equipmentUsage
    expect(
      vue.getByLabelText(
        u.gridTipFmt('Alpha', u.activeColumnFmt(u.activeFamily.camo), u.activeCellTipFmt(1, '0:05', null)),
      ),
    ).toBeTruthy()
  })

  it('écrit le compte réel, kills sommés sur les épisodes de la famille, quand killsRead est vrai', () => {
    poserArtefact({
      ...TEMOIN,
      equipmentEpisodes: [
        { slot: 1, fam: 'camo', t0: 10, t1: 60, k: 2 },
        { slot: 2, fam: 'overshield', t0: 0, t1: 30, k: 0 },
      ],
      coverage: {
        equipment: {
          tracksTotal: 40,
          camoLives: 1,
          camoEpisodes: 1,
          overshieldLives: 1,
          overshieldEpisodes: 1,
          killsRead: true,
        },
      },
    } as unknown as Partial<ReplayDocument>)
    const vue = afficher()
    const u = t.equipmentUsage
    expect(
      vue.getByLabelText(
        u.gridTipFmt('Alpha', u.activeColumnFmt(u.activeFamily.camo), u.activeCellTipFmt(1, '0:05', 2)),
      ),
    ).toBeTruthy()
  })
})
