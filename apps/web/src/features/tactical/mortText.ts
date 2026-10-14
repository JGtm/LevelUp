/**
 * mortText — les mots de la lecture des coéquipiers à la mort (manifeste `tactical.mort.*`) : la vue
 * de la zone, l'instantané, l'échange et la séquence. Fusionné dans `getTacticalText` (i18n.ts) ; un
 * fichier à part pour ne pas grossir celui des autres surfaces de l'onglet.
 */
import type { TacticalManifestKey } from '@/lib/i18n/generated/tactical'

type Formateur = (key: TacticalManifestKey, values?: Record<string, unknown>) => string

/** Les huit secteurs de la rose, du haut dans le sens horaire (cf. `coordination.Rose8`). */
const SECTEURS = [0, 1, 2, 3, 4, 5, 6, 7] as const

export function getMortText(m: Formateur) {
  return {
    mortBilanTitle: m('tactical.mort.bilan_title'),
    mortStatusLabels: {
      accompagnee: m('tactical.mort.status_in_radius'),
      visible_hors_rayon: m('tactical.mort.status_beyond'),
      aucun_visible: m('tactical.mort.status_out_of_sight'),
      equipe_a_terre: m('tactical.mort.status_none'),
    },
    mortBilanRead: (n: number, total: number) => m('tactical.mort.bilan_read', { n, m: total }),
    mortRoseTitle: m('tactical.mort.rose_title'),
    mortRoseLabel: m('tactical.mort.rose_label'),
    mortRoseFrom: (n: number, total: number, dirs: string) => m('tactical.mort.rose_from', { n, m: total, dirs }),
    mortRoseJoin: m('tactical.mort.rose_join') as string,
    mortRoseMedian: (d: string) => m('tactical.mort.rose_median', { d }),
    mortDirections: SECTEURS.map((k) => m(`tactical.mort.dir_${k}` as TacticalManifestKey) as string),
    mortCardAvenged: (s: string) => m('tactical.mort.card_avenged', { s }),
    mortCardReturned: (s: string) => m('tactical.mort.card_returned', { s }),
    mortCardOf: (total: number) => m('tactical.mort.card_of', { m: total }),
    mortCardNearest: m('tactical.mort.card_nearest'),
    mortCardMedian: (d: string) => m('tactical.mort.card_median', { d }),
    mortCardMedianLabel: m('tactical.mort.card_median_label'),
    mortDetailTitle: (date: string, instant: string | undefined, frag: boolean) => {
      const base = `tactical.mort.detail_title${frag ? '_frag' : ''}${instant ? '' : '_no_instant'}`
      return m(base as TacticalManifestKey, { date, instant })
    },
    mortDetailPlayer: m('tactical.mort.detail_player'),
    mortDetailFromDeath: m('tactical.mort.detail_from_death'),
    mortDetailFromKiller: m('tactical.mort.detail_from_killer'),
    mortDetailKiller: m('tactical.mort.detail_killer'),
    mortDetailVictim: m('tactical.mort.detail_victim'),
    mortDetailUnnamed: m('tactical.mort.detail_unnamed'),
    mortDetailWaiting: (name: string) => m('tactical.mort.detail_waiting', { name }),
    mortDetailOutOfSight: (name: string) => m('tactical.mort.detail_out_of_sight', { name }),
    mortDetailSummary: (inside: number, beyond: number, out: number, r: string) =>
      m('tactical.mort.detail_summary', { inside, beyond, out, r }),
    mortDetailSummaryNoRadius: (n: number) => m('tactical.mort.detail_summary_no_radius', { n }),
    mortAvenged: (s: string) => m('tactical.mort.avenged', { s }),
    mortAvengedBy: (who: string, d: string | undefined) =>
      d === undefined ? m('tactical.mort.avenged_by', { who }) : m('tactical.mort.avenged_by_at', { who, d }),
    mortReturned: (s: string) => m('tactical.mort.returned', { s }),
    mortLegend: {
      death: m('tactical.mort.legend_death') as string,
      allyIn: m('tactical.mort.legend_ally_in') as string,
      allyOut: m('tactical.mort.legend_ally_out') as string,
      cloud: m('tactical.mort.legend_cloud') as string,
      riposte: m('tactical.mort.legend_riposte') as string,
      killer: m('tactical.mort.legend_killer') as string,
      radius: (r: string) => m('tactical.mort.legend_radius', { r }),
    },
    seqGroup: m('tactical.mort.seq_group'),
    seqSnapshot: m('tactical.mort.seq_snapshot'),
    seqTrail: (s: string) => m('tactical.mort.seq_trail', { s }),
    seqAt: (clock: string) => m('tactical.mort.seq_at', { clock }),
    seqSummary: (inside: number, total: number, r: string) => m('tactical.mort.seq_summary', { inside, total, r }),
    seqDistanceNote: m('tactical.mort.seq_distance_note'),
    seqStage: (offset: string) => m('tactical.mort.seq_stage', { offset }),
    mortZoneSection: m('tactical.mort.zone_section'),
  }
}
