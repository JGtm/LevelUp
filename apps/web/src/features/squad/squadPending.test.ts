/**
 * Tests deriveSquadPending — propagation des sessions multi-select vers le
 * preview filter.
 *
 * Régression : avant ce fix, le SessionMultiSelect de l'écran Escouade vivait
 * dans un useState local et n'était envoyé qu'à la requête teammates. Le POST
 * filters/resolve (qui alimente le compteur sticky « N matchs » + les counts
 * cascade post-filtres) ignorait totalement la sélection — d'où l'impression
 * que les autres filtres n'avaient aucun effet sur les sessions et inversement.
 */
import { describe, expect, it } from 'vitest'
import {
  deriveSquadPending,
  decideCompositionReanchor,
  mergeSessionCounts,
} from './squadPending'
import {
  reconcileSquadSessionLabels,
  stripSessionCountSuffix,
} from '@/lib/sessions/sessionLabels'
import type { FilterContextInput, SessionLabelEntry } from '@/lib/api/types'

function session(label: string): SessionLabelEntry {
  return { label, started_at: '2026-04-02T19:00:00Z', ended_at: '2026-04-02T23:45:00Z' }
}

const basePending: FilterContextInput = {
  filter_mode: 'period',
  period: { start_date: '2026-04-01', end_date: '2026-04-30' },
  sessions: { picked_sessions: [], gap_minutes: 120 },
  cascade: { experience_types: [], playlists: [], modes: [], maps: [] },
}

describe('deriveSquadPending', () => {
  it('sans squad sessions : ajoute uniquement match_context=squad', () => {
    const out = deriveSquadPending(basePending, [])
    expect(out.match_context).toBe('squad')
    expect(out.filter_mode).toBe('period')
    expect(out.period).toEqual(basePending.period)
    expect(out.sessions?.picked_sessions).toEqual([])
  })

  it('avec sessions cochées : bascule en filter_mode=sessions et injecte les labels', () => {
    const labels = ['30/04/2026 18:30 (12)', '01/05/2026 14:00 (8)']
    const out = deriveSquadPending(basePending, labels)
    expect(out.filter_mode).toBe('sessions')
    expect(out.sessions?.picked_sessions).toEqual(labels)
  })

  it('avec sessions cochées : la période est neutralisée pour le preview', () => {
    const out = deriveSquadPending(basePending, ['session-A'])
    expect(out.period?.start_date).toBeNull()
    expect(out.period?.end_date).toBeNull()
  })

  it('match_context reste squad indépendamment du filter_mode dérivé', () => {
    const withSessions = deriveSquadPending(basePending, ['session-A'])
    const withoutSessions = deriveSquadPending(basePending, [])
    expect(withSessions.match_context).toBe('squad')
    expect(withoutSessions.match_context).toBe('squad')
  })

  it('préserve gap_minutes des sessions courantes', () => {
    const pending = {
      ...basePending,
      sessions: { picked_sessions: [], gap_minutes: 60 },
    }
    const out = deriveSquadPending(pending, ['session-A'])
    expect(out.sessions?.gap_minutes).toBe(60)
  })

  it('préserve la cascade quel que soit le mode', () => {
    const pending = {
      ...basePending,
      cascade: { experience_types: ['PVE'], playlists: [], modes: [], maps: [] },
    }
    const withSessions = deriveSquadPending(pending, ['session-A'])
    const withoutSessions = deriveSquadPending(pending, [])
    expect(withSessions.cascade?.experience_types).toEqual(['PVE'])
    expect(withoutSessions.cascade?.experience_types).toEqual(['PVE'])
  })
})

describe('stripSessionCountSuffix', () => {
  it('retire le suffixe de comptage', () => {
    expect(stripSessionCountSuffix('02/04/2026 19:00–23:45 (13)')).toBe('02/04/2026 19:00–23:45')
    expect(stripSessionCountSuffix('02/04/2026 19:00 (5)')).toBe('02/04/2026 19:00')
  })

  it('idempotent quand aucun suffixe', () => {
    expect(stripSessionCountSuffix('02/04/2026 19:00–23:45')).toBe('02/04/2026 19:00–23:45')
  })
})

describe('reconcileSquadSessionLabels', () => {
  it('remappe un label dont le suffixe a dérivé (zombie de sync) vers la forme courante', () => {
    // L'utilisateur avait pické "(13)" ; un sync a porté le compte à "(15)".
    const picked = ['02/04/2026 19:00–23:45 (13)']
    const current = [session('02/04/2026 19:00–23:45 (15)')]
    expect(reconcileSquadSessionLabels(picked, current)).toEqual(['02/04/2026 19:00–23:45 (15)'])
  })

  it('droppe un label introuvable (session disparue) — fix du compteur "minimum 2"', () => {
    const picked = ['session-zombie (3)', '02/04/2026 19:00–23:45 (15)']
    const current = [session('02/04/2026 19:00–23:45 (15)')]
    expect(reconcileSquadSessionLabels(picked, current)).toEqual(['02/04/2026 19:00–23:45 (15)'])
  })

  it('droppe le dernier zombie → liste vide (fix du "1 session" après tout désélectionner)', () => {
    const picked = ['session-zombie (3)']
    const current = [session('02/04/2026 19:00–23:45 (15)')]
    expect(reconcileSquadSessionLabels(picked, current)).toEqual([])
  })

  it('déduplique deux labels pointant sur la même session', () => {
    const picked = ['02/04/2026 19:00–23:45 (13)', '02/04/2026 19:00–23:45 (15)']
    const current = [session('02/04/2026 19:00–23:45 (15)')]
    expect(reconcileSquadSessionLabels(picked, current)).toEqual(['02/04/2026 19:00–23:45 (15)'])
  })

  it('préserve une sélection déjà à jour (no-op)', () => {
    const picked = ['02/04/2026 19:00–23:45 (15)']
    const current = [session('02/04/2026 19:00–23:45 (15)')]
    expect(reconcileSquadSessionLabels(picked, current)).toEqual(picked)
  })

  it('ne touche à rien si la liste de sessions est vide (chargement en cours)', () => {
    const picked = ['02/04/2026 19:00–23:45 (13)']
    expect(reconcileSquadSessionLabels(picked, [])).toEqual(picked)
  })

  it('préserve l\'ordre des labels valides', () => {
    const picked = ['B (2)', 'A (1)']
    const current = [session('A (9)'), session('B (9)')]
    expect(reconcileSquadSessionLabels(picked, current)).toEqual(['B (9)', 'A (9)'])
  })
})

describe('decideCompositionReanchor', () => {
  const base = {
    hasTeammates: true,
    followLatest: true,
    compositionSessionLabels: [] as string[],
    lastAnchoredLatestSession: '',
  }

  it('aucun coéquipier → none (ancrage non piloté par la composition)', () => {
    expect(
      decideCompositionReanchor({
        ...base,
        hasTeammates: false,
        latestCompositionSession: 'S_new (3)',
        pickedSessions: ['X (2)'],
      }),
    ).toEqual({ kind: 'none' })
  })

  it('follow-latest + composition jamais jouée ensemble + session pickée → clear', () => {
    expect(
      decideCompositionReanchor({ ...base, latestCompositionSession: '', pickedSessions: ['today (3)'] }),
    ).toEqual({ kind: 'clear' })
  })

  it('follow-latest + composition jamais jouée ensemble + rien de pické → none', () => {
    expect(
      decideCompositionReanchor({ ...base, latestCompositionSession: '', pickedSessions: [] }),
    ).toEqual({ kind: 'none' })
  })

  it('déjà ancré sur la dernière (suffixe « (N) » volatil ignoré) → none', () => {
    expect(
      decideCompositionReanchor({
        ...base,
        latestCompositionSession: 'S_new (5)',
        pickedSessions: ['S_new (3)'],
        compositionSessionLabels: ['S_new (5)'],
      }),
    ).toEqual({ kind: 'none' })
  })

  it('session courante ≠ dernière de la composition → snap (cœur du fix Choco/Madina)', () => {
    // On ajoute un coéquipier, "today" n'est pas une session de la composition →
    // on retombe sur S_old (dernière session où TOUS ont joué ensemble).
    expect(
      decideCompositionReanchor({
        ...base,
        latestCompositionSession: 'S_old (4)',
        pickedSessions: ['today (3)'],
        compositionSessionLabels: ['S_old (4)'],
      }),
    ).toEqual({ kind: 'snap', label: 'S_old (4)' })
  })

  it('aucune session pickée mais composition non vide → snap (atterrissage initial)', () => {
    expect(
      decideCompositionReanchor({
        ...base,
        latestCompositionSession: 'S_new (3)',
        pickedSessions: [],
        compositionSessionLabels: ['S_new (3)'],
      }),
    ).toEqual({ kind: 'snap', label: 'S_new (3)' })
  })

  it('sélection MANUELLE encore valide + dernière session DÉJÀ ancrée → none (respect du choix, ex. reload)', () => {
    expect(
      decideCompositionReanchor({
        ...base,
        followLatest: false,
        latestCompositionSession: 'S_new (5)',
        lastAnchoredLatestSession: 'S_new (5)', // on a déjà atterri sur S_new
        pickedSessions: ['S_old (2)'],
        compositionSessionLabels: ['S_new (5)', 'S_old (3)'], // S_old reste une session de la compo
      }),
    ).toEqual({ kind: 'none' })
  })

  it('sélection manuelle INVALIDE pour la nouvelle composition → snap sur la dernière', () => {
    expect(
      decideCompositionReanchor({
        ...base,
        followLatest: false,
        latestCompositionSession: 'S_new (5)',
        lastAnchoredLatestSession: 'S_new (5)',
        pickedSessions: ['ghost (2)'], // absente de compositionSessionLabels
        compositionSessionLabels: ['S_new (5)'],
      }),
    ).toEqual({ kind: 'snap', label: 'S_new (5)' })
  })

  // ── Bug « la page ne s'ouvre pas sur la dernière soirée » ────────────────
  // followLatest est faux dès qu'un chemin TECHNIQUE a appelé setSessions /
  // setFilterContext (bouton Analyser, resync au montage, réconciliation des
  // suffixes « (N) ») — pas seulement sur un choix délibéré. Sans clé de
  // détection, la page restait ancrée sur la session précédente indéfiniment.
  it('sélection épinglée valide MAIS nouvelle session jamais ancrée → snap (cœur du fix autosnap)', () => {
    expect(
      decideCompositionReanchor({
        ...base,
        followLatest: false,
        latestCompositionSession: '31/07/2026 20:12–23:40 (14)',
        lastAnchoredLatestSession: '23/07/2026 19:32–19:55 (3)', // dernier atterrissage connu
        pickedSessions: ['23/07/2026 19:32–19:55 (3)'],
        compositionSessionLabels: ['31/07/2026 20:12–23:40 (14)', '23/07/2026 19:32–19:55 (3)'],
      }),
    ).toEqual({ kind: 'snap', label: '31/07/2026 20:12–23:40 (14)' })
  })

  it('même dernière session avec un suffixe « (N) » qui a grossi au sync → none (pas de re-snap)', () => {
    expect(
      decideCompositionReanchor({
        ...base,
        followLatest: false,
        latestCompositionSession: '31/07/2026 20:12–23:40 (14)',
        lastAnchoredLatestSession: '31/07/2026 20:12–23:40 (11)',
        pickedSessions: ['23/07/2026 19:32–19:55 (3)'],
        compositionSessionLabels: ['31/07/2026 20:12–23:40 (14)', '23/07/2026 19:32–19:55 (3)'],
      }),
    ).toEqual({ kind: 'none' })
  })

  it('aucun ancrage connu (premier atterrissage) + sélection épinglée → snap sur la dernière', () => {
    expect(
      decideCompositionReanchor({
        ...base,
        followLatest: false,
        latestCompositionSession: 'S_new (5)',
        lastAnchoredLatestSession: '',
        pickedSessions: ['S_old (2)'],
        compositionSessionLabels: ['S_new (5)', 'S_old (2)'],
      }),
    ).toEqual({ kind: 'snap', label: 'S_new (5)' })
  })

  it('sélection épinglée DÉJÀ sur la dernière session nouvellement arrivée → none', () => {
    expect(
      decideCompositionReanchor({
        ...base,
        followLatest: false,
        latestCompositionSession: 'S_new (5)',
        lastAnchoredLatestSession: 'S_old (2)',
        pickedSessions: ['S_new (5)'],
        compositionSessionLabels: ['S_new (5)', 'S_old (2)'],
      }),
    ).toEqual({ kind: 'none' })
  })

  it('nouvelle session jamais ancrée mais AUCUN coéquipier → none (règle inchangée)', () => {
    expect(
      decideCompositionReanchor({
        ...base,
        hasTeammates: false,
        followLatest: false,
        latestCompositionSession: 'S_new (5)',
        lastAnchoredLatestSession: 'S_old (2)',
        pickedSessions: ['S_old (2)'],
        compositionSessionLabels: ['S_new (5)', 'S_old (2)'],
      }),
    ).toEqual({ kind: 'none' })
  })
})

// Compteurs de sessions unifiés (le « 11/8/6/5 ») : le sélecteur doit afficher le
// compte « commencés ensemble » servi par teammates, pas celui de /filters/resolve.
describe('mergeSessionCounts', () => {
  const fallback = [
    { label: 'S1 (11)', match_count_filtered: 8 },
    { label: 'S2 (4)', match_count_filtered: 4 },
  ]

  it('le compte « ensemble » de la composition prime sur celui de filters/resolve', () => {
    const merged = mergeSessionCounts(fallback, [{ label: 'S1 (11)', match_count: 6 }])
    expect(merged.get('S1 (11)')).toBe(6)
  })

  it('sessions non couvertes par la composition : repli sur filters/resolve', () => {
    const merged = mergeSessionCounts(fallback, [{ label: 'S1 (11)', match_count: 6 }])
    expect(merged.get('S2 (4)')).toBe(4)
  })

  it('réponse teammates pas encore arrivée → uniquement le repli', () => {
    const merged = mergeSessionCounts(fallback, [])
    expect(merged.get('S1 (11)')).toBe(8)
    expect(merged.size).toBe(2)
  })

  it('match_count absent ou nul → on garde le repli (pas de session affichée à 0)', () => {
    const merged = mergeSessionCounts(fallback, [
      { label: 'S1 (11)' },
      { label: 'S2 (4)', match_count: 0 },
    ])
    expect(merged.get('S1 (11)')).toBe(8)
    expect(merged.get('S2 (4)')).toBe(4)
  })

  it('session connue de la composition seule (absente du repli) → exposée', () => {
    const merged = mergeSessionCounts(fallback, [{ label: 'S3 (2)', match_count: 2 }])
    expect(merged.get('S3 (2)')).toBe(2)
  })
})
