/**
 * tacticalStrings — les mots de l'onglet Tactique (plan Tactique v2, L7.3).
 *
 * Ce que ces tests cadenassent :
 *   - les titres et les mots de la maquette, en français, tels qu'ils s'affichent (S2 à S11) ;
 *   - les libellés et les unités des sept lectures (V5), en français ET en anglais ;
 *   - le balayage de TOUTES les chaînes françaises du manifeste : aucune des formules retirées
 *     (V9 : conseils impératifs, « La question », « cuisson », « Spawn de départ »…), aucun
 *     impératif adressé au joueur, aucun anglicisme de l'onglet (« spawn », « map », « kill »…).
 */
import { describe, expect, it } from 'vitest'

import { tacticalManifest } from '@/lib/i18n/generated/tactical'

import { getTacticalText } from './i18n'

const fr = getTacticalText('fr')
const en = getTacticalText('en')

describe('les mots de la maquette, en français', () => {
  it('« Cartes jouées » : recherche, liste vide, repli du plancher, vignette', () => {
    expect(fr.mapsTitle).toBe('Cartes jouées')
    expect(fr.mapsSearchPlaceholder).toBe('Carte')
    expect(fr.mapsSearchLabel).toBe('Rechercher une carte')
    expect(fr.mapsNoMatch).toBe('Aucune carte ouvrable ne correspond')
    expect(fr.floorFold(12)).toBe('12 cartes sous le plancher')
    expect(fr.floorFoldFiltered(3, 12)).toBe('3 sur 12 cartes sous le plancher')
    expect(fr.floorCount(7, 10)).toBe('7 sur 10')
    expect(fr.tileSummary(54, 30, 24)).toBe('54 · 30 V / 24 D')
    expect(fr.select('Illusion')).toBe('Sélectionner Illusion')
  })

  it('la carte du plan : pilules, attente d’« Escouade », états, bandeau d’état', () => {
    expect([fr.pillReading, fr.pillPlayers, fr.pillRespawn, fr.pillRespawnAll]).toEqual([
      'Lecture',
      'Joueurs',
      'Réapparition',
      'Toutes',
    ])
    expect([fr.whoMe, fr.whoSquad, fr.whoOpponents]).toEqual(['Moi', 'Escouade', 'Adversaires'])
    expect(fr.planSquadPending).toBe('Aucune composition choisie')
    expect([fr.squadLabel, en.squadLabel, en.planSquadPending]).toEqual(['Escouade', 'Squad', 'No lineup chosen'])
    expect(fr.planEmptyNoMatchTitle).toBe('Aucun match sur cette carte dans le filtre')
    expect(fr.planEmptyTitle).toBe('Pas assez de matchs mesurés sur cette carte')
    expect(fr.planEmptyDensityTitle).toBe('Densité insuffisante pour dessiner un plan')
    expect(fr.statusPending(3)).toBe('3 matchs en attente de traitement')
    expect(fr.statusUnavailable(2)).toBe('2 matchs sans film')
    expect(fr.planInfo(38, 54, fr.planInfoSourceJournal, 2, 3)).toBe(
      '38 matchs mesurés sur 54 · journal des morts · grille 2 m · 3 matchs distincts par zone',
    )
    expect(fr.planLegendLabel('0', '5 morts par match')).toBe('Échelle de la lecture, de 0 à 5 morts par match')
  })

  it('la zone sélectionnée et sa mini-tuile', () => {
    expect(fr.zoneTitle).toBe('Zone sélectionnée')
    expect(fr.zoneNone).toBe('Aucune zone sélectionnée')
    expect(fr.zoneUnnamed).toBe('Zone sans nom')
    expect(fr.zoneReplayHeading).toBe('Rejeu')
    expect(fr.zoneMatches(4)).toBe('4 matchs distincts')
    expect(fr.zoneWinsLosses(3, 1)).toBe('3 victoires, 1 défaite')
    expect(fr.zoneKillsDeaths(2, 5)).toBe('2 frags, 5 morts')
    expect(fr.tileKilledBy('Rival')).toBe('Tué par Rival')
    expect(fr.tileKilled('Cible')).toBe('A tué Cible')
    expect([fr.tileAlone, fr.tileAloneAt('24'), fr.tileNearAt('9')]).toEqual(['seul', 'seul · 24 m', 'près · 9 m'])
    expect(fr.tileOpenReplay('5:07')).toBe('Ouvrir le rejeu à 5:07')
  })
})

describe('les sept lectures (V5), en français et en anglais', () => {
  const libelles = (t: typeof fr) => t.analysisQuestions.map((q) => [q.id, q.label, t.units[q.id]])

  it('libellés et unités français', () => {
    expect(libelles(fr)).toEqual([
      ['morts', 'Morts', 'morts par match'],
      ['kills', 'Frags', 'frags par match'],
      ['solde', 'Solde frags − morts', 'frags − morts par match'],
      ['gagne', 'Victoires − défaites', 'engagements par match'],
      ['temps', 'Temps de présence', 'secondes par match'],
      ['routes', 'Trajets après réapparition', 'passages par match'],
      ['isole', 'Morts seul', 'morts seul par match'],
    ])
  })

  it('libellés anglais', () => {
    expect(en.analysisQuestions.map((q) => q.label)).toEqual([
      'Deaths',
      'Kills',
      'Kills − deaths',
      'Wins − losses',
      'Time on map',
      'Routes after respawn',
      'Deaths alone',
    ])
  })
})

/** Formules retirées par V9, et impératifs adressés au joueur. */
const RETIREES: { nom: string; re: RegExp }[] = [
  { nom: '« La question »', re: /\bLa question\b/ },
  { nom: '« Clique une zone »', re: /\bClique\b/ },
  { nom: 'conseil « Élargis… »', re: /\bÉlargis\b/ },
  { nom: 'conseil « Réessaie… »', re: /\bRéessaie\b/ },
  { nom: 'conseil « Retire… »', re: /\bRetire\b/ },
  { nom: 'conseil « change de… »', re: /\b[Cc]hange de\b/ },
  { nom: 'impératif « Choisis / Ajoute / Ouvre / Sélectionne »', re: /\b(Choisis|Ajoute|Ouvre|Sélectionne)\b/ },
  { nom: '« moins c’est mieux »', re: /moins c['’]est mieux/ },
  { nom: '« échantillon faible »', re: /échantillon faible/ },
  { nom: '« cuisson »', re: /\bcuisson\b/ },
  { nom: '« Données non disponibles »', re: /Données non disponibles/ },
  { nom: '« Spawn de départ »', re: /Spawn de départ/ },
  { nom: '« Cellule sélectionnée »', re: /Cellule sélectionnée/ },
  { nom: '« Voir dans le rejeu »', re: /Voir dans le rejeu/ },
  { nom: 'tutoiement', re: /\b(tes|ton|ta)\b/ },
]

/** Anglicismes de l'onglet, en français. */
const ANGLICISMES: { nom: string; re: RegExp }[] = [
  { nom: 'spawn', re: /\bspawns?\b/i },
  { nom: 'map', re: /\bmaps?\b/i },
  { nom: 'kill', re: /\bkills?\b/i },
  { nom: 'heatmap', re: /\bheatmap\b/i },
  { nom: 'streak', re: /\bstreak\b/i },
]

/**
 * Le texte VISIBLE d'un message ICU : les noms d'arguments (`{map}`, `{n, plural, …`) ne
 * s'affichent pas et ne sont pas balayés ; le texte des branches plurielles, si.
 */
function texteVisible(message: string): string {
  return message.replace(/\{\w+,\s*(plural|select|selectordinal),/g, '{').replace(/\{\w+\}/g, '')
}

describe('le balayage des chaînes françaises du manifeste', () => {
  const chaines = Object.entries(tacticalManifest).map(([cle, v]) => ({ cle, fr: texteVisible(v.fr) }))

  it('le manifeste a bien des chaînes à balayer', () => {
    expect(chaines.length).toBeGreaterThan(80)
  })

  it.each([...RETIREES, ...ANGLICISMES])('aucune chaîne ne contient $nom', ({ re }) => {
    const fautives = chaines.filter((c) => re.test(c.fr)).map((c) => `${c.cle} : ${c.fr}`)
    expect(fautives).toEqual([])
  })
})
