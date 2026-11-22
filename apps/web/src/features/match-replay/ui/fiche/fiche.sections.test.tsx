/**
 * fiche.sections.test.tsx — LE RENDU DE CHAQUE SECTION DE LA FICHE, sur des valeurs données à la main : la section
 * écrit ce qu'on lui dit, dans la langue de la page, avec les jetons de l'application ; une ligne sans valeur n'est
 * pas écrite ; un zéro mesuré l'est quand la section le prévoit.
 */
import { cleanup, render, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'

import { COTES_FICHE_INSTANT } from '../../model/cotesFicheInstant'
import type { KillsView, ShotsView, SinceView } from '../../model/playerInstantView'
import type { MedalGroup } from '../../model/playerInstantMedals'
import { FicheDepuisDebut } from './FicheDepuisDebut'
import { FicheEliminations } from './FicheEliminations'
import { FicheEquipement } from './FicheEquipement'
import { FicheGestes, FicheTotaux } from './FicheGroupes'
import { FicheIdentite } from './FicheIdentite'
import { FicheGrenades, FicheRamassages } from './FicheLancers'
import { FicheMedailles } from './FicheMedailles'
import { FicheTirs } from './FicheTirs'

afterEach(cleanup)

describe('FicheIdentite', () => {
  const props = {
    name: 'Madina97294', camp: 'Équipe Eagle', ink: 'var(--ac-team-ally)',
    counters: { kills: 8, deaths: 5, assists: 2, fromFilm: true }, surface: 'outer' as const, locale: 'fr' as const,
  }

  it('la pastille de camp, le nom en gras, le camp, frags / morts / assistances', () => {
    const { container } = render(<FicheIdentite {...props} />)
    const nom = screen.getByText('Madina97294')
    expect(nom.tagName).toBe('B')
    expect(nom.className).toContain('font-bold')
    expect(screen.getByText('Équipe Eagle')).toBeTruthy()
    expect(screen.getByTitle("Frags / morts / assistances à l'instant lu").textContent).toBe('8/5/2')
    const pastille = container.querySelector('span[aria-hidden]') as HTMLElement
    expect(pastille.style.background).toBe('var(--ac-team-ally)')
  })

  it('aucune bordure colorée sur le bord gauche', () => {
    const { container } = render(<FicheIdentite {...props} />)
    expect(container.innerHTML).not.toMatch(/border-left|border-l-|borderLeft/)
  })

  it('le nom est à 17 px dans la colonne, à 16 px dans la fenêtre', () => {
    const { unmount } = render(<FicheIdentite {...props} surface="outer" />)
    expect(screen.getByText('Madina97294').style.fontSize).toBe(`${COTES_FICHE_INSTANT.identite.nomPx.outer}px`)
    unmount()
    render(<FicheIdentite {...props} surface="internal" />)
    expect(screen.getByText('Madina97294').style.fontSize).toBe(`${COTES_FICHE_INSTANT.identite.nomPx.internal}px`)
    expect(COTES_FICHE_INSTANT.identite.nomPx).toEqual({ outer: 17, internal: 16 })
  })

  it('les totaux de la feuille le disent en infobulle ; sans compteurs, ni triplet ni camp si le film ne l’écrit pas', () => {
    const { unmount } = render(<FicheIdentite {...props} counters={{ kills: 7, deaths: 3, assists: 2, fromFilm: false }} />)
    expect(screen.getByTitle('Frags / morts / assistances du match').textContent).toBe('7/3/2')
    unmount()
    const { container } = render(<FicheIdentite {...props} counters={null} camp={null} />)
    expect(container.querySelector('[title^="Frags"]')).toBeNull()
    expect(container.querySelector('small')).toBeNull()
  })

  it('ce que la surface pose à droite suit l’identité', () => {
    render(<FicheIdentite {...props} headEnd={<button type="button">Fermer</button>} />)
    expect(screen.getByRole('button', { name: 'Fermer' })).toBeTruthy()
  })
})

const SINCE: SinceView = {
  fda: 3.67, currentStreak: 2, bestStreak: 4, headshots: 3, perfects: 1, respawns: 5,
  currentLifeMs: 27_000, averageLifeMs: 41_000, shield: { breaks: 11, recharges: 4, differential: -7, matchBreaks: 20, matchRecharges: 9, matchDifferential: -11 }, score: 800, rank: { place: 2, of: 8 },
}

const lignes = (el: HTMLElement) =>
  [...el.querySelectorAll('dt')].map((dt) => [dt.textContent, (dt.nextElementSibling as HTMLElement).textContent])

describe('FicheDepuisDebut', () => {
  it('toutes les lignes, dans l’ordre de la maquette, valeurs dans la langue de la page', () => {
    const { container } = render(<FicheDepuisDebut since={SINCE} locale="fr" />)
    expect(screen.getByRole('region', { name: 'Depuis le début du match' })).toBeTruthy()
    expect(lignes(container)).toEqual([
      ['FDA', '3,67'],
      ['Série en cours', '2'],
      ['Meilleure série', '4'],
      ['Tirs à la tête', '3'],
      ['Éliminations parfaites', '1'],
      ['Réapparitions', '5'],
      ['Vie en cours', '0:27'],
      ["Durée moyenne d'une vie", '0:41'],
      ['Boucliers brisés', '11 · 20 sur le match'],
      ['Recharges complètes', '4 · 9 sur le match'],
      ['Différentiel', '−7 · −11 sur le match'],
      ['Score', '800 · 2e sur 8'],
    ])
  })

  it('en anglais : KDA, rang en ordinal anglais', () => {
    const { container } = render(<FicheDepuisDebut since={SINCE} locale="en" />)
    const l = lignes(container)
    expect(l[0]).toEqual(['KDA', '3.67'])
    expect(l.at(-1)).toEqual(['Score', '800 · 2nd of 8'])
    expect(l.slice(-4, -1)).toEqual([
      ['Shields broken', '11 · 20 in the match'],
      ['Full recharges', '4 · 9 in the match'],
      ['Differential', '−7 · −11 in the match'],
    ])
  })

  it('le différentiel porte son signe : plus, moins, ou ±0 quand recharges et bris s’égalent', () => {
    const egal = { breaks: 3, recharges: 3, differential: 0, matchBreaks: 5, matchRecharges: 6, matchDifferential: 1 }
    const { container } = render(<FicheDepuisDebut since={{ ...SINCE, shield: egal }} locale="fr" />)
    expect(lignes(container).find(([l]) => l === 'Différentiel')).toEqual(['Différentiel', '±0 · +1 sur le match'])
  })

  it('une valeur absente n’a pas de ligne : ni tiret ni texte de remplacement', () => {
    const since: SinceView = { ...SINCE, fda: null, currentLifeMs: null, averageLifeMs: null, headshots: null, score: null, rank: null, shield: null }
    const { container } = render(<FicheDepuisDebut since={since} locale="fr" />)
    const labels = lignes(container).map(([l]) => l)
    expect(labels).toEqual(['Série en cours', 'Meilleure série', 'Éliminations parfaites', 'Réapparitions'])
    expect(container.textContent).not.toMatch(/—/)
  })

  it('le zéro mesuré d’un compteur s’écrit, un bouclier non mesuré n’écrit rien ; le score sans rang n’écrit pas de rang', () => {
    const since: SinceView = {
      fda: 0, currentStreak: 0, bestStreak: 0, headshots: null, perfects: 0, respawns: 0,
      currentLifeMs: null, averageLifeMs: null, shield: null, score: 0, rank: null,
    }
    const { container } = render(<FicheDepuisDebut since={since} locale="fr" />)
    expect(lignes(container)).toEqual([
      ['FDA', '0,00'], ['Série en cours', '0'], ['Meilleure série', '0'], ['Éliminations parfaites', '0'],
      ['Réapparitions', '0'], ['Score', '0'],
    ])
  })

  it('le rang vaut « 1er » en tête', () => {
    const { container } = render(<FicheDepuisDebut since={{ ...SINCE, rank: { place: 1, of: 8 } }} locale="fr" />)
    expect(lignes(container).at(-1)).toEqual(['Score', '800 · 1er sur 8'])
  })
})

const KILLS: KillsView = {
  classes: [
    { classKey: 'shoulder', atInstant: 4, share: 0.5, atMatch: 9, weapons: [{ name: "Fusil d'assaut", atInstant: 4, share: 0.5, atMatch: 9 }] },
    { classKey: 'heavy', atInstant: 2, share: 0.25, atMatch: 3, weapons: [{ name: 'SPNKr', atInstant: 2, share: 0.25, atMatch: 3 }] },
  ],
  unclassified: { atInstant: 2, share: 0.25, atMatch: 2 },
  bar: [{ classKey: 'shoulder', share: 0.5 }, { classKey: 'heavy', share: 0.25 }, { classKey: 'unattributed', share: 0.25 }],
}

const cellules = (row: HTMLElement) => [...row.querySelectorAll('[role="cell"],[role="columnheader"]')].map((c) => c.textContent)

describe('FicheEliminations', () => {
  it('une barre aux jetons des classes de l’application, puis un tableau par classe et par arme', () => {
    const { container } = render(<FicheEliminations kills={KILLS} locale="fr" />)
    const barre = screen.getByRole('img')
    expect(barre.getAttribute('aria-label')).toBe('Arme d\'épaule 50 %, Lourde 25 %, Non attribué 25 %')
    const segments = [...container.querySelectorAll('[data-fiche-segment]')] as HTMLElement[]
    expect(segments.map((s) => s.style.background)).toEqual([
      'var(--ac-frag-shoulder)', 'var(--ac-frag-heavy)', 'var(--ac-frag-unattributed)',
    ])
    const rows = screen.getAllByRole('row')
    expect(rows.map(cellules)).toEqual([
      ['', "À l'instant", 'Part', 'Match'],
      ["Arme d'épaule", '4', '50 %', '9'],
      ["Fusil d'assaut", '4', '50 %', '9'],
      ['Lourde', '2', '25 %', '3'],
      ['SPNKr', '2', '25 %', '3'],
      ['Non attribué', '2', '25 %', '2'],
    ])
  })

  it('la pastille de chaque classe est de son jeton ; la classe non classée est la dernière ligne, au jeton neutre', () => {
    const { container } = render(<FicheEliminations kills={KILLS} locale="fr" />)
    const classes = [...container.querySelectorAll('[data-fiche-classe]')] as HTMLElement[]
    expect(classes.map((r) => r.getAttribute('data-fiche-classe'))).toEqual(['shoulder', 'heavy', 'unattributed'])
    const pastilles = classes.map((r) => (r.querySelector('i') as HTMLElement).style.background)
    expect(pastilles).toEqual(['var(--ac-frag-shoulder)', 'var(--ac-frag-heavy)', 'var(--ac-frag-unattributed)'])
  })

  it('« Lourde » en français, « Heavy » en anglais : les libellés de la Synthèse', () => {
    const { unmount } = render(<FicheEliminations kills={KILLS} locale="fr" />)
    expect(screen.getByText('Lourde')).toBeTruthy()
    unmount()
    render(<FicheEliminations kills={KILLS} locale="en" />)
    expect(screen.getByText('Heavy')).toBeTruthy()
    expect(screen.getByRole('region', { name: 'Kills by weapon' })).toBeTruthy()
  })

  it('sans élimination à l’instant : pas de barre, le tableau écrit ses zéros et aucune part', () => {
    const zero: KillsView = {
      classes: [{ classKey: 'shoulder', atInstant: 0, share: null, atMatch: 3, weapons: [{ name: 'BR75', atInstant: 0, share: null, atMatch: 3 }] }],
      unclassified: null, bar: [],
    }
    render(<FicheEliminations kills={zero} locale="fr" />)
    expect(screen.queryByRole('img')).toBeNull()
    expect(screen.getAllByRole('row').slice(1).map(cellules)).toEqual([["Arme d'épaule", '0', '', '3'], ['BR75', '0', '', '3']])
  })

  it('une arme sans nom : « Sans arme nommée », jamais sa clé', () => {
    const k: KillsView = {
      classes: [{ classKey: 'melee', atInstant: 1, share: 1, atMatch: 2, weapons: [{ name: null, atInstant: 1, share: 1, atMatch: 2 }, { name: 'Marteau', atInstant: 0, share: 0, atMatch: 0 }] }],
      unclassified: null, bar: [{ classKey: 'melee', share: 1 }],
    }
    render(<FicheEliminations kills={k} locale="fr" />)
    expect(screen.getByText('Sans arme nommée')).toBeTruthy()
  })

  it('une classe à armes vides (la seule arme sans nom) n’écrit que la ligne de la classe', () => {
    const k: KillsView = {
      classes: [{ classKey: 'melee', atInstant: 1, share: 1, atMatch: 1, weapons: [] }], unclassified: null,
      bar: [{ classKey: 'melee', share: 1 }],
    }
    render(<FicheEliminations kills={k} locale="fr" />)
    expect(screen.getAllByRole('row')).toHaveLength(2)
  })
})

const SHOTS: ShotsView = {
  accuracy: { ratio: 61 / 142, hit: 61, fired: 142 },
  damageDealt: 1480, damageTaken: 1120,
  weapons: [
    { key: '0xAAAA', name: "Fusil d'assaut", fired: 118, damage: 1050 },
    { key: '0xBBBB', name: 'SPNKr', fired: 6, damage: 430 },
  ],
}

describe('FicheTirs', () => {
  it('la capsule « estimation » à côté du titre, précision, dégâts, une ligne par arme', () => {
    const { container } = render(<FicheTirs shots={SHOTS} locale="fr" />)
    const section = screen.getByRole('region', { name: 'Tirs et dégâts' })
    expect(within(section).getByText('estimation')).toBeTruthy()
    expect(lignes(container)).toEqual([
      ['Précision', '43 % · 61 / 142'],
      ['Dégâts infligés / subis', '1 480 / 1 120'],
      ["Fusil d'assaut", '118 tirs · 1 050 dégâts'],
      ['SPNKr', '6 tirs · 430 dégâts'],
    ])
  })

  it('précision non établie (aucun tir estimé, ou plus de touches que de tirs) : pas de ligne de précision, les dégâts demeurent', () => {
    const { container } = render(<FicheTirs shots={{ ...SHOTS, accuracy: null, damageDealt: 0, damageTaken: 0, weapons: [] }} locale="fr" />)
    expect(lignes(container)).toEqual([['Dégâts infligés / subis', '0 / 0']])
  })

  it('en anglais', () => {
    const { container } = render(<FicheTirs shots={SHOTS} locale="en" />)
    expect(screen.getByText('estimate')).toBeTruthy()
    expect(lignes(container)[0]).toEqual(['Accuracy', '43 % · 61 / 142'])
  })
})

describe('FicheGrenades et FicheRamassages', () => {
  it('une ligne par type de grenade', () => {
    const { container } = render(<FicheGrenades throws={[{ name: 'Fragmentation', count: 4 }, { name: 'Plasma', count: 1 }]} locale="fr" />)
    expect(screen.getByRole('region', { name: 'Grenades lancées' })).toBeTruthy()
    expect(lignes(container)).toEqual([['Fragmentation', '4'], ['Plasma', '1']])
  })

  it('les ramassages portent la capsule « estimation » ; une ligne absente n’est pas écrite', () => {
    const { container, unmount } = render(<FicheRamassages pickups={{ weapons: 2, grenades: 3 }} locale="fr" />)
    expect(within(screen.getByRole('region', { name: 'Ramassages' })).getByText('estimation')).toBeTruthy()
    expect(lignes(container)).toEqual([['Armes', '2'], ['Grenades', '3']])
    unmount()
    const seul = render(<FicheRamassages pickups={{ weapons: 5, grenades: null }} locale="fr" />)
    expect(lignes(seul.container)).toEqual([['Armes', '5']])
  })
})

describe('FicheEquipement', () => {
  it('un tableau par famille : pris, utilisé, perdu, zéros compris', () => {
    render(
      <FicheEquipement
        families={[{ family: 'sensor', taken: 2, used: 1, dropped: 0 }, { family: 'overshield', taken: 1, used: 1, dropped: 0 }]}
        locale="fr"
      />,
    )
    const rows = screen.getAllByRole('row').map(cellules)
    expect(rows[0]).toEqual(['', 'Pris', 'Utilisé', 'Perdu'])
    expect(rows[1].slice(1)).toEqual(['2', '1', '0'])
    expect(rows[2].slice(1)).toEqual(['1', '1', '0'])
    expect(rows).toHaveLength(3)
  })
})

const MEDALS: MedalGroup[] = [
  { name: 'Double Kill', label: 'Doublé', description: 'Deux éliminations', count: 2, imageUrl: '/m/double.png', sprite: null, difficulty: 'Normal' },
  { name: 'Perfect', label: 'Parfait', description: '', count: 1, imageUrl: '', difficulty: 'Legendary',
    sprite: { sheet: '/m/sprites.png', left: 74, top: 148, width: 74, height: 74 } },
  { name: 'Mystery Medal', label: 'Mystery Medal', description: '', count: 3, imageUrl: '', sprite: null, difficulty: null },
]

describe('FicheMedailles', () => {
  it('l’icône de MedalIcon, le nom dessous, « ×n »', () => {
    const { container } = render(<FicheMedailles medals={MEDALS} locale="fr" />)
    const doublé = container.querySelector('[data-fiche-medaille="Double Kill"]') as HTMLElement
    expect(within(doublé).getByRole('img').getAttribute('src')).toBe('/m/double.png')
    expect(within(doublé).getByText('Doublé')).toBeTruthy()
    expect(within(doublé).getByText('×2')).toBeTruthy()
    expect(doublé.title).toBe('Deux éliminations')
  })

  it('le halo de la rareté, comme sur les tuiles de match ; aucun halo sans rareté', () => {
    const { container } = render(<FicheMedailles medals={MEDALS} locale="fr" />)
    const halo = (name: string) => (container.querySelector(`[data-fiche-medaille="${name}"] [role="img"],[data-fiche-medaille="${name}"] img`) as HTMLElement | null)?.style.filter
    expect(halo('Double Kill')).toContain('drop-shadow')
    expect(halo('Perfect')).toContain('drop-shadow')
  })

  it('une médaille à feuille de sprites : la découpe de MedalIcon', () => {
    const { container } = render(<FicheMedailles medals={MEDALS} locale="fr" />)
    const parfait = container.querySelector('[data-fiche-medaille="Perfect"]') as HTMLElement
    const icone = within(parfait).getByRole('img')
    expect(icone.tagName).toBe('DIV')
    expect(icone.getAttribute('aria-label')).toBe('Parfait')
  })

  it('une médaille sans visuel garde son nom et son compte, sans image inventée', () => {
    const { container } = render(<FicheMedailles medals={MEDALS} locale="fr" />)
    const inconnue = container.querySelector('[data-fiche-medaille="Mystery Medal"]') as HTMLElement
    expect(inconnue.querySelector('img,[role="img"]')).toBeNull()
    expect(within(inconnue).getByText('Mystery Medal')).toBeTruthy()
    expect(within(inconnue).getByText('×3')).toBeTruthy()
  })

  it('la rangée est centrée et passe à la ligne', () => {
    const { container } = render(<FicheMedailles medals={MEDALS} locale="fr" />)
    const rangee = (container.querySelector('[data-fiche-medaille]') as HTMLElement).parentElement as HTMLElement
    expect(rangee.className).toContain('justify-center')
    expect(rangee.className).toContain('flex-wrap')
  })
})

describe('FicheTotaux et FicheGestes', () => {
  it('les totaux : précision avec touches et tirs, dégâts, score, éliminations à la tête — pas d’armes', () => {
    const { container } = render(
      <FicheTotaux
        totals={{ accuracy: { pct: 41, hit: 118, fired: 288 }, damage: { dealt: 2140, taken: 1610 }, score: 1450, headshotKills: 6 }}
        locale="fr"
      />,
    )
    expect(screen.getByRole('region', { name: 'Totaux du match' })).toBeTruthy()
    expect(lignes(container)).toEqual([
      ['Précision', '41 % · 118 / 288'],
      ['Dégâts infligés / subis', '2 140 / 1 610'],
      ['Score', '1 450'],
      ['Éliminations à la tête', '6'],
    ])
  })

  it('le groupe est séparé d’un filet ; une ligne que la feuille ne porte pas n’est pas écrite', () => {
    const { container } = render(
      <FicheTotaux totals={{ accuracy: { pct: 40, hit: null, fired: null }, damage: null, score: 0, headshotKills: null }} locale="fr" />,
    )
    expect(container.firstElementChild?.className).toContain('border-t')
    expect(lignes(container)).toEqual([['Précision', '40 %'], ['Score', '0']])
  })

  it('les gestes : l’instant lu à côté du titre, sprint, sauts avec la capsule « estimation »', () => {
    const { container } = render(<FicheGestes gestures={{ sprintMs: 64_000, moveMs: null, moveEstimated: false, jumps: 12, reloads: null, reloadsMinimum: false }} clock="3:51" locale="fr" />)
    const section = screen.getByRole('region', { name: 'Gestes' })
    expect(within(section).getByText('3:51')).toBeTruthy()
    expect(container.firstElementChild?.className).toContain('border-t')
    expect(lignes(container)).toEqual([['Sprint, temps cumulé', '1:04'], ['Sautsestimation', '12']])
    expect(within(section).getByText('estimation')).toBeTruthy()
  })

  it('le temps de déplacement et les rechargements : deux lignes, dans les deux langues, sans jargon', () => {
    const g = { sprintMs: 64_000, moveMs: 125_000, moveEstimated: false, jumps: null, reloads: 18, reloadsMinimum: false }
    const fr = render(<FicheGestes gestures={g} clock="3:51" locale="fr" />)
    expect(lignes(fr.container)).toEqual([['Sprint, temps cumulé', '1:04'], ['Déplacement, temps cumulé', '2:05'], ['Rechargements', '18']])
    fr.unmount()
    const en = render(<FicheGestes gestures={g} clock="3:51" locale="en" />)
    expect(lignes(en.container)).toEqual([['Sprint, total time', '1:04'], ['Movement, total time', '2:05'], ['Reloads', '18']])
  })

  it('un compte de rechargements qui est un plancher porte la capsule « estimation » et son infobulle', () => {
    const g = { sprintMs: null, moveMs: null, moveEstimated: false, jumps: null, reloads: 7, reloadsMinimum: true }
    const { container } = render(<FicheGestes gestures={g} clock="1:00" locale="fr" />)
    expect(lignes(container)).toEqual([['Rechargementsestimation', '7']])
    expect(screen.getByTitle('Le film ne montre pas tous les rechargements : ce compte est un minimum')).toBeTruthy()
  })

  it('un temps de déplacement estimé porte la capsule « estimation » et son infobulle, dans les deux langues', () => {
    const g = { sprintMs: null, moveMs: 5000, moveEstimated: true, jumps: null, reloads: null, reloadsMinimum: false }
    const fr = render(<FicheGestes gestures={g} clock="1:00" locale="fr" />)
    expect(lignes(fr.container)).toEqual([['Déplacement, temps cumuléestimation', '0:05']])
    expect(screen.getByTitle('Le film ne montre pas toutes les commandes de déplacement : ce temps est une estimation')).toBeTruthy()
    fr.unmount()
    render(<FicheGestes gestures={g} clock="1:00" locale="en" />)
    expect(screen.getByTitle('The film does not show every movement command: this time is an estimate')).toBeTruthy()
  })

  it('un geste absent n’a pas de ligne', () => {
    const { container } = render(<FicheGestes gestures={{ sprintMs: null, moveMs: null, moveEstimated: false, jumps: 4, reloads: null, reloadsMinimum: false }} clock="0:10" locale="fr" />)
    expect(lignes(container)).toEqual([['Sautsestimation', '4']])
  })
})
