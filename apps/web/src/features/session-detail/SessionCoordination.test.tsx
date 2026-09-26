/**
 * SessionCoordination.test — LE LOT O sur la colonne de session : les deux cartes de la
 * section « Coordination » (D22-1, D22-6) et la carte « Portée des engagements » (D22-4).
 *
 * Ce que ces cas verrouillent, et pourquoi chacun compte :
 *
 *   - DEUX JAUGES, DEUX EN-TÊTES PROPRES, UN SEUL TRAIT DE PARITÉ. `UsageGaugeGrid` lisait
 *     ses en-têtes dans une table de TROIS dénominateurs du bloc « usages » et reconnaissait
 *     la colonne d'équipe par sa POSITION : montée telle quelle, la carte Riposte aurait
 *     affiché « Mon équipe dans le lobby » au-dessus de « Je suis couvert » et rayé une
 *     colonne sur deux. Le cas fixe les en-têtes ET l'absence de hachure ;
 *   - UNE CASE SANS DÉNOMINATEUR RESTE GRISE. Un match sans mort de camp ne vaut pas 0 % de
 *     ripostes — il n'est pas mesuré. Le tri par `tone` est la seule chose qui distingue les
 *     deux à l'écran ;
 *   - `available = false` GARDE LA RANGÉE et nomme sa cause (D8) : un bloc escamoté laisse
 *     la rangée bancale et se lit comme un bug ;
 *   - PORTÉE (lot W, D23-4) : le nuage porte la PÉRIODE, la session s'y surligne sur les
 *     bons matchs, les bandes sont les SEUILS SERVIS (jamais recalculés côté web), la
 *     référence absente replie sur la session seule sans bandes ni surbrillance, et en
 *     comparaison les deux colonnes surlignent DEUX fenêtres du MÊME nuage.
 */
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'

import { buildSquadRangeRolesOption } from '@/features/squad/charts/squadRangeRolesChart'
import type {
  CoordinationBlock,
  MatchRangeBlock,
  RangeReferenceBlock,
} from '@/lib/api/types'

import { SessionCoordinationSection } from './SessionCoordinationSection'
import { COORDINATION_TEXT } from './coordinationI18n'
import {
  bandCaption,
  buildAppuiGaugeRows,
  buildRiposteBand,
  buildRiposteGaugeRows,
} from './coordinationModel'
import { nuagePortee, pointDeLaSession, seuilsServis } from './sessionRange.logic'

const t = COORDINATION_TEXT.fr

function couverture(taux: number, brut: number, n: number, faible = false) {
  return { taux, brut, n, par_match: brut / Math.max(1, n), echantillon_faible: faible }
}

function matchPoint(id: string, over: Partial<Record<string, number>> = {}) {
  return {
    match_id: id,
    assists_to_me: 2,
    my_assisted_kills: 4,
    my_deaths: 5,
    my_deaths_avenged: 3,
    my_measured_kills: 10,
    my_ripostes: 3,
    team_assists: 12,
    team_deaths: 20,
    team_deaths_avenged: 12,
    parity_pct: 25,
    riposte_share_pct: 40,
    assist_share_of_team_pct: 25,
    team_size: 4,
    ...over,
  }
}

const BLOC: CoordinationBlock = {
  available: true,
  fenetre_ms: 5000,
  matches_measured: 3,
  matches_total: 4,
  riposte: {
    je_suis_couvert: couverture(0.62, 31, 50),
    je_riposte: couverture(0.31, 18, 58),
    parity_pct: 25,
    team_deaths: 58,
    team_deaths_avenged: 30,
    delai_median_ms: 4200,
  },
  appui: {
    on_me_prepare: couverture(0.44, 22, 50),
    ma_part_des_appuis: couverture(0.34, 17, 50),
    parity_pct: 25,
  },
  per_match: [
    matchPoint('m1'),
    // Aucune mort de camp : la case doit rester GRISE malgré une part servie.
    matchPoint('m2', { team_deaths: 0, riposte_share_pct: 0 }),
    matchPoint('m3', { riposte_share_pct: 10 }),
  ],
}

describe('Section Coordination (D22-1 / D22-6)', () => {
  it('rend deux jauges nommées par le lot, sans hachure de lobby, et un seul trait de parité', () => {
    const { container } = render(<SessionCoordinationSection coordination={BLOC} />)

    // Les en-têtes sont ceux du lot, pas les trois dénominateurs du bloc « usages ».
    expect(screen.getByText(t.gaugeCovered)).toBeInTheDocument()
    expect(screen.getByText(t.gaugeIRiposte)).toBeInTheDocument()
    expect(screen.getByText(t.gaugePrepared)).toBeInTheDocument()
    expect(screen.getByText(t.gaugeAssistShare)).toBeInTheDocument()

    // Aucune colonne rapportée au lobby → aucune tranche hachurée.
    expect(container.querySelectorAll('[data-gauge-denominator="lobby"]')).toHaveLength(0)

    // Le chiffre d'appel (délai médian) est écrit ; aucune autre phrase (D22-verbosité).
    expect(container.querySelectorAll('[data-coordination-callout]')).toHaveLength(1)
    expect(screen.getByText(`${t.delaiMedian} : 4,2 s`)).toBeInTheDocument()
  })

  it('grise la case d’un match sans mort de camp et ne la compte pas dans le pied', () => {
    const cells = buildRiposteBand(BLOC.per_match ?? [], t, 'fr')
    expect(cells.map((c) => c.tone)).toEqual(['above', 'unmeasured', 'below'])
    // 1 case au-dessus sur 2 MESURÉES — la case grise sort du dénominateur.
    expect(bandCaption(cells, t)).toBe('1/2')
  })

  it('garde la rangée et nomme la cause quand le bloc est indisponible (D8)', () => {
    const { container } = render(
      <SessionCoordinationSection
        coordination={{ ...BLOC, available: false, matches_measured: 0 }}
      />,
    )
    expect(container.querySelector('[data-session-coordination="empty"]')).not.toBeNull()
    // Les deux cartes restent, avec leur titre et une cause nommée.
    expect(screen.getByText(t.cardRiposte)).toBeInTheDocument()
    expect(screen.getByText(t.cardAppui)).toBeInTheDocument()
    expect(container.querySelectorAll('[data-usage-empty]')).toHaveLength(2)
  })
})

describe('Repère d’habituel des jauges sans parité (lot S)', () => {
  it('pose l’habituel de la période sur « je suis couvert » et « on me prépare »', () => {
    const bloc: CoordinationBlock = {
      ...BLOC,
      riposte: { ...BLOC.riposte, habituel_pct: 55 },
      appui: { ...BLOC.appui, habituel_pct: 38 },
    }
    const [riposte] = buildRiposteGaugeRows(bloc, t, 'fr')
    const [appui] = buildAppuiGaugeRows(bloc, t, 'fr')

    // Le trait des deux jauges sans parité EST l'habituel...
    expect(riposte.gauges[0].parityPct).toBe(55)
    expect(appui.gauges[0].parityPct).toBe(38)
    // ...et l'infobulle le NOMME, faute de quoi il se lirait comme une parité.
    expect(riposte.gauges[0].tooltip).toContain('habituel')
    expect(appui.gauges[0].tooltip).toContain('habituel')
    // La jauge voisine garde SA parité, et son infobulle ne parle pas d'habituel.
    expect(riposte.gauges[1].parityPct).toBe(25)
    expect(riposte.gauges[1].tooltip).not.toContain('habituel')
  })

  it('n’invente aucun repère quand le contrat ne sert pas d’habituel', () => {
    const [riposte] = buildRiposteGaugeRows(BLOC, t, 'fr')
    const [appui] = buildAppuiGaugeRows(BLOC, t, 'fr')
    expect(riposte.gauges[0].parityPct).toBeNull()
    expect(appui.gauges[0].parityPct).toBeNull()
    expect(riposte.gauges[0].tooltip).not.toContain('habituel')
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
    },
  ]
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
