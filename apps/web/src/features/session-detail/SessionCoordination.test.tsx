/**
 * SessionCoordination.test — LE LOT O sur la colonne de session : la carte « Appui reçu » de la
 * section « Coordination » (D22-6) et la carte « Portée des engagements » (D22-4). « Riposte » a
 * quitté la page (plan PLAN_SESSIONS_EMPRISE_2026-10-06, V3).
 *
 * Ce que ces cas verrouillent, et pourquoi chacun compte :
 *
 *   - DEUX JAUGES, DEUX EN-TÊTES PROPRES, AUCUNE HACHURE DE LOBBY. `UsageGaugeGrid` lisait
 *     ses en-têtes dans une table de TROIS dénominateurs du bloc « usages » et reconnaissait
 *     la colonne d'équipe par sa POSITION : le cas fixe les en-têtes ET l'absence de hachure ;
 *   - UNE CASE SANS DÉNOMINATEUR RESTE GRISE. Un match sans appui dans le camp ne vaut pas 0 % —
 *     il n'est pas mesuré. Le tri par `tone` est la seule chose qui distingue les deux à l'écran ;
 *   - `available = false` GARDE LA RANGÉE et nomme sa cause (D8) : un bloc escamoté laisse
 *     la rangée bancale et se lit comme un bug ;
 *   - PORTÉE (lot W, D23-4) : le nuage porte la PÉRIODE, la session s'y surligne sur les
 *     bons matchs, les bandes sont les SEUILS SERVIS (jamais recalculés côté web), la
 *     référence absente replie sur la session seule sans bandes ni surbrillance, et en
 *     comparaison les deux colonnes surlignent DEUX fenêtres du MÊME nuage.
 */
import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import { buildSquadRangeRolesOption } from '@/features/squad/charts/squadRangeRolesChart'
import type {
  CoordinationBlock,
  MatchRangeBlock,
  RangeReferenceBlock,
} from '@/lib/api/types'

import { SessionCoordinationSection } from './SessionCoordinationSection'
import { SessionRangeCard } from './SessionRangeCard'
import { COORDINATION_TEXT } from './coordinationI18n'
import { bandCaption, buildAppuiBand, buildAppuiGaugeRows } from './coordinationModel'
import { nuagePortee, pointDeLaSession, seuilsServis } from './sessionRange.logic'

// ECharts (canvas) ne peint pas dans jsdom : la carte de portée se lit par son cadre et sa légende.
vi.mock('echarts-for-react', () => ({ default: () => null }))

const t = COORDINATION_TEXT.fr

function couverture(taux: number, brut: number, n: number, faible = false) {
  return { taux, brut, n, par_match: brut / Math.max(1, n), echantillon_faible: faible }
}

function matchPoint(id: string, over: Partial<Record<string, number>> = {}) {
  return {
    match_id: id,
    assists_to_me: 2,
    my_assisted_kills: 4,
    my_kills: 10,
    team_assists: 12,
    parity_pct: 25,
    assist_share_of_team_pct: 25,
    team_size: 4,
    ...over,
  }
}

/** Le bloc tel que la carte le lit : l'appui et la couverture (la riposte n'a plus de lecteur). */
const BLOC = {
  available: true,
  matches_measured: 3,
  matches_total: 4,
  appui: {
    on_me_prepare: couverture(0.44, 22, 50),
    ma_part_des_appuis: couverture(0.34, 17, 50),
    parity_pct: 25,
  },
  per_match: [
    matchPoint('m1', { assist_share_of_team_pct: 40 }),
    // Aucun appui dans le camp : la case doit rester GRISE malgré une part servie.
    matchPoint('m2', { team_assists: 0, assist_share_of_team_pct: 0 }),
    matchPoint('m3', { assist_share_of_team_pct: 10 }),
  ],
} as unknown as CoordinationBlock

describe('Section Coordination : « Appui reçu » seule (D22-6, V3)', () => {
  it('rend deux jauges nommées par le lot, sans hachure de lobby, sans carte Riposte', () => {
    const { container } = render(<SessionCoordinationSection coordination={BLOC} />)

    expect(screen.getByText(t.gaugePrepared)).toBeInTheDocument()
    expect(screen.getByText(t.gaugeAssistShare)).toBeInTheDocument()
    // Aucune colonne rapportée au lobby → aucune tranche hachurée.
    expect(container.querySelectorAll('[data-gauge-denominator="lobby"]')).toHaveLength(0)
    // Plus de Riposte : ni sa carte, ni son chiffre d'appel.
    expect(container.textContent).not.toContain('Riposte')
    expect(container.querySelectorAll('[data-coordination-callout]')).toHaveLength(0)
  })

  it('grise la case d’un match sans appui dans le camp et ne la compte pas dans le pied', () => {
    const cells = buildAppuiBand(BLOC.per_match ?? [], t, 'fr')
    expect(cells.map((c) => c.tone)).toEqual(['above', 'unmeasured', 'below'])
    // 1 case au-dessus sur 2 MESURÉES — la case grise sort du dénominateur.
    expect(bandCaption(cells, t)).toBe('1/2')
    // La case grise dit ce qu'elle est, jamais « non mesuré ».
    expect(cells[1].tooltip).toBe('Match #2 · aucun appui d’équipe')
  })

  it('n’écrit « non mesuré » nulle part sur la carte (légende, aide)', () => {
    const { container } = render(<SessionCoordinationSection coordination={BLOC} />)
    expect(container.textContent).not.toMatch(/mesur/i)
    expect(t.infoAppui2).toContain('bots')
  })

  it('garde la rangée et nomme la cause quand le bloc est indisponible (D8)', () => {
    const { container } = render(
      <SessionCoordinationSection
        coordination={{ ...BLOC, available: false, matches_measured: 0 }}
      />,
    )
    expect(container.querySelector('[data-session-coordination="empty"]')).not.toBeNull()
    // La carte reste, avec son titre et une cause nommée.
    expect(screen.getAllByText(t.cardAppui).length).toBeGreaterThan(0)
    expect(container.querySelectorAll('[data-usage-empty]')).toHaveLength(1)
  })
})

describe('Repère d’habituel de « on me prépare » (lot S)', () => {
  it('pose l’habituel de la période sur la jauge sans parité, et le nomme', () => {
    const bloc = { ...BLOC, appui: { ...BLOC.appui, habituel_pct: 38 } } as CoordinationBlock
    const [appui] = buildAppuiGaugeRows(bloc, t, 'fr')
    // Le trait de la jauge sans parité EST l'habituel...
    expect(appui.gauges[0].parityPct).toBe(38)
    // ...et l'infobulle le NOMME, faute de quoi il se lirait comme une parité.
    expect(appui.gauges[0].tooltip).toContain('habituel')
    // La jauge voisine garde SA parité, et son infobulle ne parle pas d'habituel.
    expect(appui.gauges[1].parityPct).toBe(25)
    expect(appui.gauges[1].tooltip).not.toContain('habituel')
  })

  it('n’invente aucun repère quand le contrat ne sert pas d’habituel', () => {
    const [appui] = buildAppuiGaugeRows(BLOC, t, 'fr')
    expect(appui.gauges[0].parityPct).toBeNull()
    expect(appui.gauges[0].tooltip).not.toContain('habituel')
  })
})

const profil = (id: string, ordreIso: string, mediane: number, lobby: number, mesures: number) => ({
  match_id: id,
  played_at: ordreIso,
  map_name: 'Streets',
  lobby_median_m: lobby,
  lobby_measured: 40,
  players: [
    {
      xuid: 'x1',
      gamertag: 'Moi',
      median_m: mediane,
      lobby_delta_m: mediane - lobby,
      measured: mesures,
    },
  ],
})

/** Quatre matchs de la période, servis EN DÉSORDRE : le tri chronologique les remet en place. */
const PROFILS_PERIODE = [
  profil('p2', '2026-04-20T20:00:00Z', 16, 20, 11),
  profil('m1', '2026-04-21T19:30:00Z', 14, 20, 12),
  profil('p1', '2026-04-19T18:00:00Z', 24, 20, 10),
  profil('m2', '2026-04-21T20:15:00Z', 18, 20, 3), // sous le plancher → creux
]

const REFERENCE: RangeReferenceBlock = {
  profiles: PROFILS_PERIODE,
  role_low_m: -5,
  role_high_m: 1,
  period_median_delta_m: -2,
  matches_measured: 4,
  matches_total: 5,
}

/** La session affichée : les deux derniers matchs de la période. */
const BLOC_SESSION: MatchRangeBlock = {
  kills_measured: 15,
  kills_total: 20,
  profiles: [
    profil('m1', '2026-04-21T19:30:00Z', 14, 20, 12),
    profil('m2', '2026-04-21T20:15:00Z', 18, 20, 3),
  ],
}

const LIBELLES = {
  xAxis: t.rangeXAxisPeriod,
  yAxis: t.rangeAxis,
  lobbyLine: t.rangeLobbyLine,
  bandes: { front: t.roleFront, polyvalent: t.roleVersatile, sniper: t.roleSniper },
  tooltipMedian: t.rangeTipMedian,
  tooltipDelta: t.rangeTipDelta,
  tooltipMeasured: t.rangeTipMeasured,
}

interface OptionNuage {
  series: [
    {
      data: { value: [number, number]; itemStyle: Record<string, unknown> }[]
      markArea?: { data: Record<string, unknown>[][] }
      markLine?: { label: { position: string } }
    },
  ]
  xAxis: { axisLabel: { hideOverlap?: boolean } }
  legend?: unknown
}

/** Monte l'option du nuage avec deux encres reconnaissables (appartenance, pas identité). */
function option(nuage: ReturnType<typeof nuagePortee>, encreSession = '#111', encrePeriode = '#999') {
  return buildSquadRangeRolesOption([nuage.serie!], {
    categories: nuage.categories,
    seuils: nuage.seuils,
    couleurs: { Moi: encreSession },
    mesuresMin: 3,
    mesuresMax: 12,
    masquerLegende: true,
    encrePoint: (p) => (pointDeLaSession(nuage, p) ? encreSession : encrePeriode),
    surbrillance: nuage.surbrillance
      ? { ...nuage.surbrillance, label: t.rangeThisSession, couleur: encreSession }
      : undefined,
    libelles: LIBELLES,
    fmtM: (v: number) => String(v),
  }) as unknown as OptionNuage
}

describe('Carte Portée des engagements (lot W, D23-4)', () => {
  it('pose l’axe sur la période, dans l’ordre chronologique, et surligne les matchs de la session', () => {
    const nuage = nuagePortee(REFERENCE, BLOC_SESSION, 'moi')
    expect(nuage.periode).toBe(true)
    expect(nuage.profils.map((p) => p.match_id)).toEqual(['p1', 'p2', 'm1', 'm2'])
    expect(nuage.categories[0]).toBe('#1 · Streets')
    // La session, ce sont les indices 2 et 3 — surtout pas les deux premiers.
    expect(nuage.surbrillance).toEqual({ debut: 2, fin: 3 })
    expect(nuage.serie!.points.map((p) => p.plein)).toEqual([true, true, true, false])
  })

  it('peint la session à l’encre du joueur et la période en gris, le point creux reste creux', () => {
    const nuage = nuagePortee(REFERENCE, BLOC_SESSION, 'moi')
    const data = option(nuage).series[0].data
    expect(data.map((d) => d.value[1])).toEqual([4, -4, -6, -2])
    expect(data[0].itemStyle.color).toBe('#999')
    expect(data[2].itemStyle.color).toBe('#111')
    // Le dernier point est de la session ET sous le plancher : creux, contour à l'encre.
    expect(data[3].itemStyle.color).toBe('transparent')
    expect(data[3].itemStyle.borderColor).toBe('#111')
  })

  it('assied les bandes sur les SEUILS SERVIS, sans jamais les recalculer', () => {
    const nuage = nuagePortee(REFERENCE, BLOC_SESSION, 'moi')
    expect(nuage.seuils).toEqual({ bas: -5, haut: 1 })
    // Les tiers des écarts mesurés vaudraient (-4, -4) : la carte ne les invente pas.
    const opt = option(nuage)
    const zones = opt.series[0].markArea!.data
    // Trois bandes de rôle + la fenêtre de surbrillance.
    expect(zones).toHaveLength(4)
    expect(zones[1][0].yAxis).toBe(-5)
    expect(zones[2][0].yAxis).toBe(1)
    expect(zones[3][0].xAxis).toBe(1.5)
    expect(zones[3][1].xAxis).toBe(3.5)
    expect(zones[3][0].name).toBe(t.rangeThisSession)
    // Une seule série : la légende du graphe n'a rien à nommer.
    expect(opt.legend).toBeUndefined()
  })

  it('lisibilité en colonne étroite : étiquettes de match sans chevauchement, médiane du lobby à droite des libellés de bande', () => {
    const opt = option(nuagePortee(REFERENCE, BLOC_SESSION, 'moi'))
    expect(opt.xAxis.axisLabel.hideOverlap).toBe(true)
    // Les libellés de bande s’écrivent à gauche : la médiane du lobby va en fin de ligne.
    expect(opt.series[0].markLine!.label.position).toBe('insideEndTop')
  })

  it('les libellés des zones s’écrivent AU-DESSUS des points, sur le fond de la carte', () => {
    const opt = option(nuagePortee(REFERENCE, BLOC_SESSION, 'moi')) as unknown as {
      series: Array<{
        type: string
        z?: number
        markArea?: { label: { show: boolean; backgroundColor?: string }; data: Record<string, unknown>[][] }
      }>
    }
    const fonds = opt.series[0]
    const calque = opt.series[opt.series.length - 1]
    // Les remplissages restent sous le nuage, sans libellé : un point ne couvre plus « Ligne de front ».
    expect(fonds.markArea!.label.show).toBe(false)
    expect(calque.type).toBe('custom')
    expect(calque.z).toBeGreaterThan(fonds.z ?? 0)
    expect(calque.markArea!.label.show).toBe(true)
    expect(calque.markArea!.label.backgroundColor).toBeTruthy()
    // Les mêmes zones, sans remplissage : trois bandes + la surbrillance.
    expect(calque.markArea!.data).toHaveLength(4)
    expect(calque.markArea!.data.every(([debut]) => (debut.itemStyle as { color: string }).color === 'transparent')).toBe(true)
    expect(calque.markArea!.data[0][0].name).toBe(LIBELLES.bandes.front)
  })

  it('n’assied aucune bande quand le serveur ne sert pas les seuils', () => {
    const sansSeuils: RangeReferenceBlock = {
      ...REFERENCE,
      role_low_m: undefined,
      role_high_m: undefined,
    }
    expect(seuilsServis(sansSeuils)).toBeNull()
    const nuage = nuagePortee(sansSeuils, BLOC_SESSION, 'moi')
    expect(nuage.seuils).toBeNull()
    // Reste la seule surbrillance.
    expect(option(nuage).series[0].markArea!.data).toHaveLength(1)
  })

  it('replie sur la session seule sans référence : pas de bandes, pas de surbrillance', () => {
    const nuage = nuagePortee(null, BLOC_SESSION, 'moi')
    expect(nuage.periode).toBe(false)
    expect(nuage.profils.map((p) => p.match_id)).toEqual(['m1', 'm2'])
    expect(nuage.seuils).toBeNull()
    expect(nuage.surbrillance).toBeNull()
    const opt = option(nuage)
    expect(opt.series[0].markArea).toBeUndefined()
    // Tous les points sont ceux de la session : aucun gris de population.
    expect(opt.series[0].data.map((d) => d.itemStyle.color)).toEqual(['#111', 'transparent'])
  })

  it('en comparaison, les deux colonnes surlignent deux fenêtres du MÊME nuage', () => {
    const comparee: MatchRangeBlock = {
      kills_measured: 20,
      kills_total: 22,
      profiles: [profil('p1', '2026-04-19T18:00:00Z', 24, 20, 10)],
    }
    const a = nuagePortee(REFERENCE, BLOC_SESSION, 'moi')
    const b = nuagePortee(REFERENCE, comparee, 'moi')
    expect(b.categories).toEqual(a.categories)
    expect(b.surbrillance).toEqual({ debut: 0, fin: 0 })
    const dataB = option(b).series[0].data
    expect(dataB[0].itemStyle.color).toBe('#111')
    expect(dataB[2].itemStyle.color).toBe('#999')
  })
})

describe('Carte « Portée des engagements » : graphe et légende, rien d’autre', () => {
  it('aucune mention de couverture sous le graphe ; légende centrée en bas de la carte', () => {
    const { container } = render(<SessionRangeCard block={BLOC_SESSION} reference={REFERENCE} meLabel="Moi" />)
    expect(container.textContent).not.toMatch(/mesurés sur|measured/)
    expect(container.querySelector('[data-testid="session-portee-couverture"]')).toBeNull()
    const legende = container.querySelector('[data-testid="session-portee-legende"]')
    expect(legende?.className).toContain('justify-center')
    expect(legende?.parentElement?.lastElementChild).toBe(legende)
  })
})
