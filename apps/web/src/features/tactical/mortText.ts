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
    mortRoseTitle: m('tactical.mort.rose_title'),
    mortRoseLabel: m('tactical.mort.rose_label'),
    mortRoseFrom: (n: number, total: number, dirs: string) => m('tactical.mort.rose_from', { n, m: total, dirs }),
    mortRoseMedian: (d: string) => m('tactical.mort.rose_median', { d }),
    mortDirections: SECTEURS.map((k) => m(`tactical.mort.dir_${k}` as TacticalManifestKey) as string),
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
    mortLegend: {
      death: m('tactical.mort.legend_death') as string,
      allyIn: m('tactical.mort.legend_ally_in') as string,
      allyOut: m('tactical.mort.legend_ally_out') as string,
      cloud: m('tactical.mort.legend_cloud') as string,
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
    seqPlayReplay: m('tactical.mort.seq_play_replay'),
    seqPlayPause: m('tactical.mort.seq_play_pause'),
    seqPlayResume: m('tactical.mort.seq_play_resume'),
    seqLoading: m('tactical.mort.seq_loading'),
    seqPending: m('tactical.mort.seq_pending'),
    // Pourquoi une séquence manque (une paire titre / description par cause, `CauseDeSequence`), et
    // l'état d'une contribution qui n'est pas une mort.
    seqMissing: {
      film_non_analyse: {
        title: m('tactical.mort.seq_missing_film_title') as string,
        description: m('tactical.mort.seq_missing_film_desc') as string,
      },
      film_sans_horloge: {
        title: m('tactical.mort.seq_missing_clock_title') as string,
        description: m('tactical.mort.seq_missing_clock_desc') as string,
      },
      carte_sans_rayon: {
        title: m('tactical.mort.seq_missing_radius_title') as string,
        description: m('tactical.mort.seq_missing_radius_desc') as string,
      },
      sans_position: {
        title: m('tactical.mort.seq_missing_position_title') as string,
        description: m('tactical.mort.seq_missing_position_desc') as string,
      },
      inconnue: {
        title: m('tactical.mort.seq_missing_unknown_title') as string,
        description: m('tactical.mort.seq_missing_unknown_desc') as string,
      },
    },
    seqOnlyDeaths: {
      title: m('tactical.mort.seq_only_deaths_title') as string,
      description: m('tactical.mort.seq_only_deaths_desc') as string,
    },
    mortZoneSection: m('tactical.mort.zone_section'),
  }
}
