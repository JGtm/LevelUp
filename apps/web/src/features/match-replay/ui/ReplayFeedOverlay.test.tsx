/**
 * Tests — LE FIL DES ÉLIMINATIONS POSÉ SUR LE PLAN (`ReplayFeedOverlay`).
 *
 * Ce qu'ils protègent : 0, 3, 5 et 8 éliminations passées (rien ; 3 lignes sans estompage ; 5 lignes dont les
 * deux plus anciennes sont estompées ; les 5 dernières) ; la plus récente EN BAS ; seules les éliminations y
 * entrent ; il suit l'instant lu sans remonter ses lignes ; il ne capte pas le pointeur et passe sous les
 * surcouches de la carte ; ligne nue (pas de médailles), aux colonnes et aux cotes du fil.
 */
import { render } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { filmAllegianceOf } from '@/lib/replay/filmAllegiance'

import { COTES_FIL_PLAN } from '../model/cotesFilPlan'
import { COTES_LIGNE_FIL } from '../model/cotesFeedLine'
import type { MedalEvent, ReplayFeedEntry, ReplayKill } from '../model/killFeedLogic'
import { testReplayDoc } from '../test/testDoc'
import { ReplayFeedOverlay } from './ReplayFeedOverlay'

vi.mock('@/lib/accessibility', () => ({
  resolveToken: (token: string) => `var(${token})`,
  tokenCssVar: (token: string) => `var(--ac-${token})`,
}))

const DOC = testReplayDoc({
  frameIntervalMs: 100,
  roster: [
    { xuid: 'me', filmIndex: 0, name: 'JGtm', team: 0 },
    { xuid: 'foe', filmIndex: 1, name: 'Cobra01', team: 1 },
  ],
})
const ALLEGIANCE = filmAllegianceOf(DOC, [], 'me')
const META = new Map([
  ['me', { gamertag: 'JGtm' }],
  ['foe', { gamertag: 'Cobra01' }],
])

/** La couche des surcouches de la carte (écran de fin, message inter-manche, compte à rebours de la bombe). */
const COUCHE_SURCOUCHES_CARTE = 10
const FIN = 10_000_000

function medaille(): MedalEvent {
  return { tMs: 1_000, xuid: 'me', gamertag: 'JGtm', name: 'Medal', label: 'Médaille', description: '', imageUrl: '/m.png' }
}

/** Une élimination à `secondes`, la clé `k<secondes>`, avec une médaille rattachée. */
function elimination(secondes: number): ReplayFeedEntry {
  const replayMs = secondes * 1_000
  const kill = {
    tMs: replayMs, xuid: 'me', teamID: 0, weaponKey: 'br75', weaponLabel: 'BR75', weaponImageUrl: '/br75.png',
    weaponTinted: true, assistState: '', assistGamertag: '', assistTeamID: null, killerDamagePct: null,
    assistDamagePct: null, victimXuid: 'foe', victimGamertag: 'Cobra01', replayMs, medals: [medaille()],
  } as ReplayKill
  return { key: `k${secondes}`, replayMs, kill, medal: null, death: null }
}

const fil = (n: number) => Array.from({ length: n }, (_, i) => elimination(i + 1))

function afficher(entries: ReplayFeedEntry[], nowMs = FIN, zoneOf?: (key: string) => string) {
  const props = {
    playWindow: null, xuidMeta: META, allegiance: ALLEGIANCE, locale: 'fr' as const, zoneOf,
  }
  const view = render(<ReplayFeedOverlay {...props} entries={entries} nowMs={nowMs} />)
  return {
    ...view,
    avec: (suivant: number) => view.rerender(<ReplayFeedOverlay {...props} entries={entries} nowMs={suivant} />),
  }
}

const lignes = (c: HTMLElement) => [...c.querySelectorAll<HTMLElement>('li')]
const instants = (c: HTMLElement) => lignes(c).map((li) => li.querySelector('[data-feed-col="instant"]')?.textContent)
const colonnesDe = (li: Element) => [...li.children].map((c) => c.getAttribute('data-feed-col') ?? 'nom')

describe('le nombre de lignes', () => {
  it("aucune élimination passée : rien n'est rendu", () => {
    expect(afficher([]).container.firstChild).toBeNull()
    expect(afficher(fil(4), 500).container.firstChild).toBeNull()
  })

  it('3 éliminations passées : 3 lignes, aucune estompée', () => {
    const { container } = afficher(fil(3))
    expect(lignes(container)).toHaveLength(3)
    expect(lignes(container).map((li) => li.style.opacity)).toEqual(['1', '1', '1'])
  })

  it('5 éliminations passées : 5 lignes, les deux plus anciennes estompées (70 %, puis 85 %)', () => {
    const { container } = afficher(fil(5))
    expect(lignes(container)).toHaveLength(5)
    expect(lignes(container).map((li) => li.style.opacity)).toEqual(['0.7', '0.85', '1', '1', '1'])
  })

  it('8 éliminations passées : les 5 dernières, avec le même estompage', () => {
    const { container } = afficher(fil(8))
    expect(instants(container)).toEqual(['0:04', '0:05', '0:06', '0:07', '0:08'])
    expect(lignes(container).map((li) => li.style.opacity)).toEqual(['0.7', '0.85', '1', '1', '1'])
  })

  it("4 lignes parmi 8 : pas d'estompage, même quand le fil a plus de lignes plus loin dans le match", () => {
    const { container } = afficher(fil(8), 4_000)
    expect(instants(container)).toEqual(['0:01', '0:02', '0:03', '0:04'])
    expect(lignes(container).map((li) => li.style.opacity)).toEqual(['1', '1', '1', '1'])
  })
})

describe("l'ordre, l'instant lu et les entrées", () => {
  it('la plus récente est EN BAS', () => {
    const { container } = afficher(fil(5))
    expect(instants(container)).toEqual(['0:01', '0:02', '0:03', '0:04', '0:05'])
  })

  it('seules les éliminations y entrent : médaille seule, mort sans tueur, entrée de partie et événement sont écartés', () => {
    const autres: ReplayFeedEntry[] = [
      { key: 'm', replayMs: 1_500, kill: null, medal: medaille(), death: null },
      { key: 'd', replayMs: 1_600, kill: null, medal: null, death: { xuid: 'foe', replayMs: 1_600 } as never },
      { key: 'p', replayMs: 1_700, kill: null, medal: null, death: null, presence: { xuid: 'foe' } as never },
      { key: 'e', replayMs: 1_800, kill: null, medal: null, death: null, event: { family: 'objectives' } as never },
    ]
    const { container } = afficher([...fil(2), ...autres].sort((a, b) => a.replayMs - b.replayMs))
    expect(instants(container)).toEqual(['0:01', '0:02'])
  })

  it("il suit l'instant lu : ses lignes avancent et reculent avec lui", () => {
    const { container, avec } = afficher(fil(8), 3_000)
    expect(instants(container)).toEqual(['0:01', '0:02', '0:03'])
    avec(8_000)
    expect(instants(container)).toEqual(['0:04', '0:05', '0:06', '0:07', '0:08'])
    avec(2_000)
    expect(instants(container)).toEqual(['0:01', '0:02'])
  })

  it("rien n'est remonté pendant la lecture : la liste et les lignes qui restent gardent leur nœud", () => {
    const { container, avec } = afficher(fil(8), 6_000)
    const liste = container.querySelector('ul')
    const quatre = lignes(container)[2]
    expect(quatre.querySelector('[data-feed-col="instant"]')?.textContent).toBe('0:04')
    avec(6_500)
    avec(7_000)
    expect(container.querySelector('ul')).toBe(liste)
    // La ligne de 4 s est restée dans la fenêtre des 5 dernières (2 s à 6 s, puis 3 s à 7 s) : le même nœud.
    expect(lignes(container)).toContain(quatre)
  })
})

describe('la forme posée sur le plan', () => {
  it('il ne capte pas le pointeur : le zoom et le déplacement de la carte passent à travers', () => {
    const { container } = afficher(fil(3))
    expect(container.querySelector('ul')?.className).toContain('pointer-events-none')
  })

  it('il passe SOUS les surcouches de la carte, et il est ancré en bas à gauche, aux cotes de la maquette', () => {
    const style = afficher(fil(3)).container.querySelector('ul')?.style
    expect(COTES_FIL_PLAN.couche).toBeLessThan(COUCHE_SURCOUCHES_CARTE)
    expect(style?.zIndex).toBe(String(COTES_FIL_PLAN.couche))
    expect(style?.left).toBe('16px')
    expect(style?.bottom).toBe('12px')
    expect(style?.width).toBe('350px')
  })

  it("posé au-dessus de la barre de lecture quand elle est sur le plan : il monte de la hauteur qu'on lui donne", () => {
    const view = render(
      <ReplayFeedOverlay entries={fil(3)} nowMs={FIN} playWindow={null} xuidMeta={META} allegiance={ALLEGIANCE} locale="fr" liftPx={56} />,
    )
    expect(view.container.querySelector('ul')?.style.bottom).toBe(`${COTES_FIL_PLAN.margeBasPx + 56}px`)
  })

  it("l'ombre portée vient du jeton de voile, aucune couleur écrite en dur", () => {
    const classe = afficher(fil(3)).container.querySelector('ul')?.className ?? ''
    expect(classe).toContain('text-shadow')
    expect(classe).toContain('drop-shadow')
    expect(classe).toContain('var(--scrim)')
    expect(classe).not.toMatch(/#[0-9a-f]{3,8}|rgba?\(|oklch\(/i)
  })

  it('ni en-tête, ni menu, ni bouton : une liste de lignes nues', () => {
    const { container } = afficher(fil(3))
    expect(container.querySelector('button')).toBeNull()
    expect(container.querySelector('h1, h2, h3, [role="menu"]')).toBeNull()
    expect(container.querySelector('ul')?.getAttribute('aria-label')).toBe('Éliminations')
  })

  it("ligne nue, sans médailles (même quand l'élimination en porte), aux colonnes du fil : instant, lieu, tueur, arme, victime", () => {
    const { container } = afficher([elimination(1)], FIN, () => 'Fer à cheval')
    const [ligne] = lignes(container)
    expect(colonnesDe(ligne)).toEqual(['instant', 'lieu', 'nom', 'arme', 'nom'])
    expect(ligne.querySelector('[data-feed-col="medailles"]')).toBeNull()
    expect([...ligne.children].map((c) => c.textContent).filter((t) => t)).toEqual(['0:01', 'Fer à cheval', 'JGtm', 'Cobra01'])
    const c = COTES_LIGNE_FIL.colonnes
    expect(ligne.style.gridTemplateColumns).toBe(
      `${c.instantPx}px ${c.lieuPx}px ${c.tueurPx}px ${c.armePx}px ${c.victimePx}px minmax(0, 1fr)`,
    )
  })

  it('carte sans zone nommée : pas de colonne du lieu, les autres gardent leur alignement', () => {
    const { container } = afficher([elimination(1)])
    expect(colonnesDe(lignes(container)[0])).toEqual(['instant', 'nom', 'arme', 'nom'])
  })

  it('le lieu de la victime suit la ligne : un lieu inconnu laisse la colonne, vide', () => {
    const { container } = afficher([elimination(1), elimination(2)], FIN, (cle) => (cle === 'k1' ? 'Pont Est' : ''))
    const lieux = lignes(container).map((li) => li.querySelector('[data-feed-col="lieu"]')?.textContent)
    expect(lieux).toEqual(['Pont Est', ''])
  })
})
