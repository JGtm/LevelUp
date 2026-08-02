/**
 * Tests formatDataIssues — les dégradations de chargement remontées par l'API
 * doivent produire un message lisible (FR et EN), y compris pour un code inconnu.
 *
 * Régression visée : avant, un LoadMainTeamParticipants / LoadFor en échec était
 * avalé côté serveur (slog.Warn) et la page affichait des chiffres amputés sans
 * rien dire — d'où des compteurs non reproductibles d'une requête à l'autre.
 */
import { describe, expect, it } from 'vitest'
import { formatDataIssues } from './squadDataIssues'
import { getSquadText } from './i18n'

const fr = getSquadText('fr')
const en = getSquadText('en')

describe('formatDataIssues', () => {
  it('aucune dégradation → aucun message', () => {
    expect(formatDataIssues(undefined, fr)).toEqual([])
    expect(formatDataIssues([], fr)).toEqual([])
  })

  it('matchs d\'un coéquipier non chargés : le gamertag apparaît dans le message', () => {
    const [msg] = formatDataIssues([{ code: 'teammate_matches', detail: 'Chocoboflor' }], fr)
    expect(msg).toContain('Chocoboflor')
  })

  it('heatmap : message distinct de celui des matchs du coéquipier', () => {
    const [tm] = formatDataIssues([{ code: 'teammate_matches', detail: 'Madina97294' }], fr)
    const [hm] = formatDataIssues([{ code: 'heatmap_teammate', detail: 'Madina97294' }], fr)
    expect(hm).not.toEqual(tm)
    expect(hm).toContain('Madina97294')
  })

  it('codes sans détail : message fixe', () => {
    expect(formatDataIssues([{ code: 'main_team_participants' }], fr)[0]).toBe(fr.dataIssues.mainTeamParticipants)
    expect(formatDataIssues([{ code: 'map_stats' }], fr)[0]).toBe(fr.dataIssues.mapStats)
  })

  it('code inconnu (backend plus récent) → message générique, jamais de ligne vide', () => {
    const [msg] = formatDataIssues([{ code: 'nouvelle_cause' }], fr)
    expect(msg).toContain('nouvelle_cause')
    expect(msg.trim().length).toBeGreaterThan(0)
  })

  it('parité EN : chaque code produit un message anglais non vide et distinct du FR', () => {
    const issues = [
      { code: 'teammate_matches', detail: 'Chocoboflor' },
      { code: 'heatmap_teammate', detail: 'Chocoboflor' },
      { code: 'main_team_participants' },
      { code: 'map_stats' },
    ]
    const frMsgs = formatDataIssues(issues, fr)
    const enMsgs = formatDataIssues(issues, en)
    expect(enMsgs).toHaveLength(4)
    for (let i = 0; i < enMsgs.length; i++) {
      expect(enMsgs[i].trim().length).toBeGreaterThan(0)
      expect(enMsgs[i]).not.toEqual(frMsgs[i])
    }
  })

  it('conserve l\'ordre et le nombre de dégradations', () => {
    const msgs = formatDataIssues(
      [{ code: 'map_stats' }, { code: 'teammate_matches', detail: 'A' }, { code: 'heatmap_teammate', detail: 'B' }],
      fr,
    )
    expect(msgs).toHaveLength(3)
    expect(msgs[0]).toBe(fr.dataIssues.mapStats)
    expect(msgs[1]).toContain('A')
    expect(msgs[2]).toContain('B')
  })
})
