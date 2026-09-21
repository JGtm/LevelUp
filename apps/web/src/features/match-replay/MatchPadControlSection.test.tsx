/**
 * Tests — MatchPadControlSection (le contrôle des armes spéciales de la page match).
 *
 * CE QU'ILS PROTÈGENT, et ce sont les promesses de la section :
 *   1. LA DOUBLE PORTE. Sans artefact — la quasi-totalité des matchs en production — la section
 *      ne rend RIEN ; avec un artefact dont aucune occupation n'est attribuée, non plus. Un
 *      cadre vide répété sur chaque page de match est une promesse non tenue à l'infini.
 *   2. LES NOMS D'ARME viennent des tables EXISTANTES du rejeu (catalogue du document, familles
 *      de socle) — jamais une clé brute, jamais une seconde table de noms.
 *   3. UNE COLONNE PAR SOCLE (2026-09-21, D18) : hauteur = les prises nommées, échelle commune,
 *      colonnes rangées PUISSANCE puis TERRAIN, un trait entre les deux groupes et le nom de
 *      chaque groupe avec son sous-total écrit DANS le graphe.
 *   4. CE QUI N'A PAS DE RAMASSEUR NOMMÉ N'EST VERSÉ À PERSONNE : il est annoté sous sa colonne,
 *      hors de la pile.
 *   5. LES ARMES DE BASE SONT DANS UN DÉPLIABLE FERMÉ, et rien d'autre n'est caché.
 *
 * LE GRAPHE EST UN DOUBLE (`BarStackedChart` bouchonné) : ECharts peint un canvas que jsdom ne
 * sait pas lire. On éprouve donc ce que la section LUI DEMANDE — l'ordre des colonnes, les
 * groupes, les totaux, les notes —, et le dessin lui-même est éprouvé chez
 * `BarStackedChart.test.ts` et `barStackedGroups.test.ts`. Le calcul, lui, est éprouvé chez
 * `padControlLogic.test.ts` et `padControlColumns.test.ts`.
 */
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import type { MatchScoreboardRow, ReplayDocument } from '@/lib/api/types'

import { REPLAY_TEXT } from './i18n/i18n'
import { MatchPadControlSection } from './MatchPadControlSection'
import { testReplayDoc } from './test/testDoc'

// La lecture de l'artefact est la SEULE frontière réseau du composant : on la pilote, et tout
// le reste (agrégation, colonnes, libellés) reste le vrai code. Même patron que
// `MatchEquipmentUsageSection.test.tsx`.
const artefact = vi.hoisted(() => ({ current: undefined as unknown }))
vi.mock('../../lib/replay/queries', () => ({ useMatchReplay: () => ({ data: artefact.current }) }))

interface GrapheProps {
  series: { datapoints: { category: string; components: Record<string, number> }[] }[]
  categoryGroups: { label: string; span: number }[]
  valueLabels: { segments?: boolean; totals?: readonly number[] }
  categoryNote: (category: string) => string | undefined
  componentOrder: string[]
  emptyMessage: string
  showLegend?: boolean
}

const graphes = vi.hoisted(() => ({ appels: [] as unknown[] }))
vi.mock('@/components/charts/BarStackedChart', () => ({
  BarStackedChart: (props: GrapheProps) => {
    graphes.appels.push(props)
    const points = props.series[0]?.datapoints ?? []
    return (
      <div data-testid="bar-stacked">
        {points.length === 0 && <p>{props.emptyMessage}</p>}
        {points.map((d) => (
          <span key={d.category} data-testid="colonne">
            {d.category}
          </span>
        ))}
        {props.categoryGroups
          .filter((g) => g.label !== '')
          .map((g) => (
            <span key={g.label} data-testid="titre-groupe">
              {g.label}
            </span>
          ))}
      </div>
    )
  },
}))

/** Les props du dernier graphe monté (le principal, ou celui du dépliable s'il est ouvert). */
function dernierGraphe(): GrapheProps {
  return graphes.appels[graphes.appels.length - 1] as GrapheProps
}

const t = REPLAY_TEXT.fr
const SNIPER = '0xAAAA1111'

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
 * LE TÉMOIN DE RENDU : trois joueurs, un socle d'arme nommé au catalogue, un socle de bonus, et
 * quatre occupations dont deux seulement portent un ramasseur.
 */
const TEMOIN: Partial<ReplayDocument> = {
  frameCount: 200,
  frameIntervalMs: 100,
  roster: [
    { filmIndex: 0, xuid: 'a1', name: 'Alpha' },
    { filmIndex: 1, xuid: 'a2', name: 'Bravo' },
    { filmIndex: 2, xuid: 'b1', name: 'Charlie' },
  ],
  tracks: [vie(1, 'a1'), vie(2, 'a2'), vie(3, 'b1')],
  weaponLabels: { [SNIPER]: { fr: 'S7 Sniper', en: 'S7 Sniper', key: 'hinf_s7_sniper' } },
  weaponPads: [
    { weapon: SNIPER, x: 0, y: 0, spawns: [], presence: [] },
    { weapon: 'powerup_overshield', x: 1, y: 1, spawns: [], presence: [] },
  ],
  padPickups: [
    { pad: 0, t: 10, tLow: 5, tHigh: 15, xuid: 'a1' },
    { pad: 0, t: 40, tLow: 35, tHigh: 45, xuid: 'a1' },
    { pad: 0, tLow: 60, tHigh: 70, xuid: null },
    { pad: 1, tLow: 80, tHigh: 90, xuid: null },
  ],
  coverage: {
    padDating: {
      occupations: 4,
      dated: 2,
      named: 2,
      ambiguous: 1,
      uncovered: 0,
      powerupOccupations: 1,
    },
  },
} as unknown as Partial<ReplayDocument>

function poserArtefact(over: Partial<ReplayDocument> | null) {
  artefact.current = over ? testReplayDoc(over) : undefined
  graphes.appels.length = 0
}

function afficher(locale: 'fr' | 'en' = 'fr') {
  return render(
    <MatchPadControlSection
      playerSlug="joueur"
      matchId="m1"
      replayAvailable
      scoreboard={SCOREBOARD}
      locale={locale}
    />,
  )
}

/**
 * L'AIDE DU TITRE, ouverte. Depuis le 2026-09-21 (lot D) une SEULE infobulle (i) porte la
 * provenance de l'attribution ET les deux réserves qui s'écrivaient au-dessus du graphe
 * (niveaux non établis, départs aléatoires), plus le compte des prises non classées.
 */
function aideDuTitre(vue: ReturnType<typeof afficher>): string {
  fireEvent.mouseEnter(vue.getByRole('button', { name: /informations|more info/i }))
  return screen.getByRole('tooltip').textContent ?? ''
}

describe('MatchPadControlSection — la double porte', () => {
  it('ne rend RIEN sans artefact : 404 = pas de film, pas de cadre vide', () => {
    poserArtefact(null)
    expect(afficher().container.firstChild).toBeNull()
  })

  it('ne rend rien quand le film ne porte AUCUNE occupation de socle', () => {
    poserArtefact({ tracks: [vie(1, 'a1')] } as Partial<ReplayDocument>)
    expect(afficher().container.firstChild).toBeNull()
  })

  it('ne rend rien quand des socles se sont vidés mais qu’AUCUNE prise n’est attribuée', () => {
    poserArtefact({
      ...TEMOIN,
      padPickups: [
        { pad: 0, tLow: 5, tHigh: 15, xuid: null },
        { pad: 1, tLow: 80, tHigh: 90, xuid: null },
      ],
    } as unknown as Partial<ReplayDocument>)
    expect(afficher().container.firstChild).toBeNull()
  })
})

describe('MatchPadControlSection — le graphe', () => {
  it('affiche le titre et une colonne par socle réellement pris', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    expect(vue.getByRole('region', { name: t.padControl.title })).toBeTruthy()
    expect(vue.getAllByTestId('colonne').map((n) => n.textContent)).toEqual(['S7 Sniper'])
    // Le socle de bonus n'a été pris par personne : aucune colonne pour lui.
    expect(vue.queryByText(t.padEquipmentFamily.powerup_overshield)).toBeNull()
  })

  it('nomme le socle par le CATALOGUE du document, jamais par sa clé brute', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    expect(vue.getByText('S7 Sniper')).toBeTruthy()
    expect(vue.queryByText(SNIPER)).toBeNull()
  })

  it('la hauteur d’une colonne est le total NOMMÉ, et les segments portent les joueurs', () => {
    poserArtefact(TEMOIN)
    afficher()
    const graphe = dernierGraphe()
    expect(graphe.valueLabels.totals).toEqual([2])
    expect(graphe.valueLabels.segments).toBe(true)
    expect(graphe.series[0].datapoints[0].components).toEqual({ Alpha: 2 })
  })

  it('nomme chaque joueur dans la légende DOM, camp du joueur de la page en tête', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    const legende = vue.getByTestId('chart-legend')
    expect([...legende.querySelectorAll('li')].map((li) => li.textContent)).toEqual(['Alpha'])
    // UNE SEULE légende : celle d'ECharts est éteinte.
    expect(dernierGraphe().showLegend).toBe(false)
  })

  it('annote SOUS la colonne les occupations sans ramasseur nommé, hors de la pile', () => {
    poserArtefact(TEMOIN)
    afficher()
    // Trois occupations du socle S7, deux nommées : la troisième s'annonce en note.
    expect(dernierGraphe().categoryNote('S7 Sniper')).toBe(t.padControl.unnamedFmt(1))
  })
})

describe('MatchPadControlSection — le repli « game changers » (plan 2026-09-05)', () => {
  const BR = '0xDDDD4444'

  /**
   * LE TÉMOIN MIXTE : le Sniper (élu) et un BR dont le label n'a PAS de clé canonique —
   * replié par la dégradation D6. Trois prises attribuées sur cinq occupations mesurées.
   */
  const TEMOIN_MIXTE: Partial<ReplayDocument> = {
    ...TEMOIN,
    weaponLabels: {
      [SNIPER]: { fr: 'S7 Sniper', en: 'S7 Sniper', key: 'hinf_s7_sniper' },
      [BR]: { fr: 'BR75', en: 'BR75' },
    },
    weaponPads: [
      { weapon: SNIPER, x: 0, y: 0, spawns: [], presence: [] },
      { weapon: 'powerup_overshield', x: 1, y: 1, spawns: [], presence: [] },
      { weapon: BR, x: 2, y: 2, spawns: [], presence: [] },
    ],
    padPickups: [
      { pad: 0, t: 10, tLow: 5, tHigh: 15, xuid: 'a1' },
      { pad: 0, t: 40, tLow: 35, tHigh: 45, xuid: 'a1' },
      { pad: 2, t: 50, tLow: 45, tHigh: 55, xuid: 'a2' },
      { pad: 0, tLow: 60, tHigh: 70, xuid: null },
      { pad: 1, tLow: 80, tHigh: 90, xuid: null },
    ],
    coverage: {
      padDating: {
        occupations: 5,
        dated: 3,
        named: 3,
        ambiguous: 1,
        uncovered: 0,
        powerupOccupations: 1,
      },
    },
  } as unknown as Partial<ReplayDocument>

  it('AFFICHE TOUTES LES ARMES au chargement : le vote ordonne, il ne cache plus rien', () => {
    poserArtefact(TEMOIN_MIXTE)
    const vue = afficher()
    expect(vue.getAllByTestId('colonne').map((n) => n.textContent).sort()).toEqual([
      'BR75',
      'S7 Sniper',
    ])
    expect(vue.queryByRole('button', { name: /Voir plus|Replier/ })).toBeNull()
  })

  it('le TOTAL ne ment pas : la prise du socle non élu est bien à sa colonne', () => {
    poserArtefact(TEMOIN_MIXTE)
    afficher()
    const graphe = dernierGraphe()
    const parSocle = Object.fromEntries(
      graphe.series[0].datapoints.map((d) => [d.category, d.components]),
    )
    expect(parSocle.BR75).toEqual({ Bravo: 1 })
  })
})

describe('MatchPadControlSection — ce que l’écran dit de sa mesure', () => {
  // LE PIED DE CARTE (« N prises attribuées sur N occupations… ») A ÉTÉ RETIRÉ le 2026-09-13
  // sur demande de l'utilisateur : l'annotation de colonne est le seul aveu qui reste.
  it('n’écrit AUCUN pied de carte : rien sous le graphe et sa légende', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    expect(vue.container.textContent).not.toContain('attribuée')
    expect(vue.container.textContent).not.toContain('occupation')
  })

  it('rend les mêmes libellés en anglais, sans laisser une string française', () => {
    poserArtefact(TEMOIN)
    const vue = afficher('en')
    const en = REPLAY_TEXT.en.padControl
    expect(vue.getByRole('region', { name: en.title })).toBeTruthy()
    expect(dernierGraphe().categoryNote('S7 Sniper')).toBe(en.unnamedFmt(1))
  })
})

/**
 * LES NIVEAUX D'ARMES (2026-09-14). Le bloc range ses colonnes en base / terrain / puissance /
 * non classé, et le niveau vient de la CARTE — l'emplacement Forge qui confirme le socle — plus
 * de l'équipement de départ du film. Jamais du nom ni du rôle de l'arme.
 */
describe('MatchPadControlSection — les niveaux d’armes', () => {
  const AR = '0xBBBB2222'

  /** Le témoin des niveaux : le sniper sur un socle de PUISSANCE, l'AR sur un RÂTELIER. */
  function temoinNiveaux(over: Partial<ReplayDocument> = {}) {
    return {
      ...TEMOIN,
      weaponLabels: {
        [SNIPER]: { fr: 'S7 Sniper', en: 'S7 Sniper', key: 'hinf_s7_sniper' },
        [AR]: { fr: 'MA40 AR', en: 'MA40 AR', key: 'hinf_ma40_ar' },
      },
      weaponPads: [
        { weapon: SNIPER, x: 0, y: 0, spawns: [], presence: [] },
        { weapon: AR, x: 1, y: 0, spawns: [], presence: [] },
      ],
      mapWeaponPads: {
        catalogN: 4,
        pads: [
          { x: 0, y: 0, pad: 0, family: 'power' },
          { x: 1, y: 0, pad: 1, family: 'rack' },
        ],
      },
      padPickups: [
        { pad: 0, t: 10, tLow: 5, tHigh: 15, xuid: 'a1' },
        { pad: 1, t: 40, tLow: 35, tHigh: 45, xuid: 'b1' },
      ],
      ...over,
    } as unknown as Partial<ReplayDocument>
  }

  it('range les colonnes PUISSANCE puis TERRAIN, et nomme chaque groupe avec son sous-total', () => {
    poserArtefact(temoinNiveaux())
    const vue = afficher('fr')
    expect(vue.getAllByTestId('colonne').map((n) => n.textContent)).toEqual([
      'S7 Sniper',
      'MA40 AR',
    ])
    expect(vue.getAllByTestId('titre-groupe').map((n) => n.textContent)).toEqual([
      `${t.padControl.tierShortLabels.power} · ${t.padControl.tierSubtotalFmt(1)}`,
      `${t.padControl.tierShortLabels.ground} · ${t.padControl.tierSubtotalFmt(1)}`,
    ])
    // LE TRAIT DE SÉPARATION : une colonne par groupe, donc une frontière après la première.
    expect(dernierGraphe().categoryGroups.map((g) => g.span)).toEqual([1, 1])
  })

  it('promeut en « base » l’arme de l’équipement de départ, et la met dans un dépliable FERMÉ', () => {
    poserArtefact(
      temoinNiveaux({
        loadouts: Array.from({ length: 20 }, (_, i) => ({ t: 74, slot: 512 + i, w: [AR] })),
      } as unknown as Partial<ReplayDocument>),
    )
    const vue = afficher('fr')
    // LE NIVEAU « BASE » EST UN DÉPLIABLE FERMÉ depuis le 2026-09-21 (D2) : son bouton porte le
    // compte, et sa colonne n'est pas dans le document tant qu'il n'est pas ouvert.
    const bouton = vue.getByRole('button', { name: /Armes de base/ })
    expect(bouton.textContent).toContain(t.padControl.baseToggleFmt(1))
    expect(bouton.getAttribute('aria-expanded')).toBe('false')
    expect(vue.queryByText('MA40 AR')).toBeNull()
    fireEvent.click(bouton)
    expect(bouton.getAttribute('aria-expanded')).toBe('true')
    expect(vue.getByText('MA40 AR')).toBeTruthy()
    // L'AR quitte le râtelier pour la base ; il n'y a plus de groupe « terrain ».
    expect(vue.queryByText(new RegExp(t.padControl.tierShortLabels.ground))).toBeNull()
  })

  it('un match sans socle de puissance ni de terrain montre l’ÉTAT VIDE de la carte', () => {
    poserArtefact(
      temoinNiveaux({
        loadouts: Array.from({ length: 20 }, (_, i) => ({ t: 74, slot: 512 + i, w: [AR] })),
        mapWeaponPads: { catalogN: 4, pads: [{ x: 1, y: 0, pad: 1, family: 'rack' }] },
        padPickups: [{ pad: 1, t: 40, tLow: 35, tHigh: 45, xuid: 'b1' }],
      } as unknown as Partial<ReplayDocument>),
    )
    const vue = afficher('fr')
    expect(vue.getByText(t.padControl.chartEmpty)).toBeTruthy()
    // La carte reste affichée, et les armes de base gardent leur dépliable.
    expect(vue.getByRole('region', { name: t.padControl.title })).toBeTruthy()
    expect(vue.getByRole('button', { name: /Armes de base/ })).toBeTruthy()
  })

  it('écrit la note « départs aléatoires » quand le SERVEUR le dit, et n’y publie aucun niveau de base', () => {
    // Le caractère aléatoire vient de la RÉPONSE (`weaponTiers.randomStarts`), plus d'une liste
    // de catégories tenue côté web — celle-ci a divergé en une semaine (revue 2026-09-14).
    poserArtefact(
      temoinNiveaux({
        loadouts: Array.from({ length: 20 }, (_, i) => ({ t: 74, slot: 512 + i, w: [AR] })),
        weaponTiers: { randomStarts: true },
      } as unknown as Partial<ReplayDocument>),
    )
    const vue = afficher('fr')
    expect(aideDuTitre(vue)).toContain(t.padControl.randomStartsNote)
    expect(vue.queryByRole('button', { name: /Armes de base/ })).toBeNull()
    // Les deux autres niveaux restent lisibles.
    expect(vue.getAllByTestId('titre-groupe').map((n) => n.textContent)).toEqual([
      `${t.padControl.tierShortLabels.power} · ${t.padControl.tierSubtotalFmt(1)}`,
      `${t.padControl.tierShortLabels.ground} · ${t.padControl.tierSubtotalFmt(1)}`,
    ])
  })

  it('dit « niveaux non établis » quand la carte n’est pas dans la référence, et ne nomme AUCUN groupe', () => {
    poserArtefact(temoinNiveaux({ mapWeaponPads: undefined } as unknown as Partial<ReplayDocument>))
    const vue = afficher('fr')
    expect(aideDuTitre(vue)).toContain(t.padControl.tiersUnmeasuredNote)
    // Surtout pas un titre « Non identifié » au-dessus du graphe : une absence de mesure n'est
    // pas un résultat de mesure.
    expect(vue.queryAllByTestId('titre-groupe')).toHaveLength(0)
    // Les colonnes, elles, restent toutes rendues : rien n'est retiré.
    expect(vue.getAllByTestId('colonne').map((n) => n.textContent).sort()).toEqual([
      'MA40 AR',
      'S7 Sniper',
    ])
  })

  it('n’écrit aucune de ces notes quand la carte est connue et le mode régulier', () => {
    poserArtefact(temoinNiveaux())
    const vue = afficher('fr')
    const aide = aideDuTitre(vue)
    expect(aide).not.toContain(t.padControl.tiersUnmeasuredNote)
    expect(aide).not.toContain(t.padControl.randomStartsNote)
  })

  // 2026-09-21, décision utilisateur amendant D2 : un socle de BONUS est un équipement, il se
  // lit dans « Usages d'équipement » et n'a rien à faire dans le contrôle des ARMES.
  it('ne rend PAS le niveau des socles de bonus', () => {
    poserArtefact(
      temoinNiveaux({
        weaponPads: [
          { weapon: SNIPER, x: 0, y: 0, spawns: [], presence: [] },
          { weapon: 'powerup_overshield', x: 1, y: 1, spawns: [], presence: [] },
        ],
        mapWeaponPads: {
          catalogN: 4,
          pads: [
            { x: 0, y: 0, pad: 0, family: 'power' },
            { x: 1, y: 1, pad: 1, family: 'powerup' },
          ],
        },
        padPickups: [
          { pad: 0, t: 10, tLow: 5, tHigh: 15, xuid: 'a1' },
          { pad: 1, t: 40, tLow: 35, tHigh: 45, xuid: 'b1' },
        ],
      } as unknown as Partial<ReplayDocument>),
    )
    const vue = afficher('fr')
    expect(vue.getAllByTestId('titre-groupe').map((n) => n.textContent)).toEqual([
      `${t.padControl.tierShortLabels.power} · ${t.padControl.tierSubtotalFmt(1)}`,
    ])
  })

  it('nomme les groupes en anglais aussi', () => {
    poserArtefact(temoinNiveaux())
    const vue = afficher('en')
    const en = REPLAY_TEXT.en.padControl
    expect(vue.getAllByTestId('titre-groupe').map((n) => n.textContent)).toEqual([
      `${en.tierShortLabels.power} · ${en.tierSubtotalFmt(1)}`,
      `${en.tierShortLabels.ground} · ${en.tierSubtotalFmt(1)}`,
    ])
  })
})
