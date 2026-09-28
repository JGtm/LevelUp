/**
 * SquadFragBreakdownCard.test.tsx — « Répartition des frags » : le compte de chaque classe
 * s'écrit dans son segment s'il y tient (6 px de marge de chaque côté), sinon sur la ligne de
 * repli au-dessus de la barre, alignée sur le premier segment masqué (règle S3) ; total au
 * bout ; légende des classes en pied de carte.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import { SquadFragBreakdownCard } from './SquadFragBreakdownCard'
import { getSquadText } from './i18n'
import type { FragClassEntry } from '@/lib/api/types'

const t = getSquadText('fr')

function cls(className: string, kills: number): FragClassEntry {
  return { class: className, kills, authoritative: false }
}

const DATA = {
  JGtm: [cls('shoulder', 28), cls('sidearm', 13), cls('grenade', 5), cls('unattributed', 2)],
  Madina97294: [cls('shoulder', 45), cls('grenade', 3)],
}
const ORDER = ['JGtm', 'Madina97294']

function renderCard() {
  return render(
    <SquadFragBreakdownCard
      fragClassesByPlayer={DATA}
      playerOrder={ORDER}
      playerColors={{}}
      classLabel={(c) => `L:${c}`}
      emptyTitle="Aucune donnée"
      t={t}
    />,
  )
}

afterEach(() => {
  vi.restoreAllMocks()
})

/**
 * Largeurs simulées : un segment mesure 4 px par frag, une étiquette 7 px par chiffre.
 * Tient si largeur étiquette + 12 ≤ largeur segment : 28 (112 px), 13 (52 px) et 5 (20 px,
 * pour 7 + 12 = 19) tiennent ; 2 (8 px) et 3 (12 px) non.
 */
function mockWidths() {
  vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(function (this: HTMLElement) {
    let width = 0
    if (this.dataset.fitKey) {
      const kills = Number(this.querySelector('[data-fit-label]')?.textContent ?? 0)
      width = kills * 4
    } else if (this.hasAttribute('data-fit-label')) {
      width = (this.textContent ?? '').length * 7
    }
    return { width, height: 0, top: 0, left: 0, right: width, bottom: 0, x: 0, y: 0, toJSON: () => ({}) } as DOMRect
  })
}

describe('SquadFragBreakdownCard', () => {
  it('sans mesure possible (largeurs nulles) : tous les comptes passent au repli, aucun perdu', () => {
    renderCard()
    const repli = screen.getByTestId('frag-breakdown-repli-JGtm')
    expect(repli.textContent).toBe('281352')
    expect(repli.style.paddingLeft).toBe('0%')
    expect(screen.getByTestId('frag-breakdown-total-JGtm').textContent).toBe('48')
    expect(screen.getByTestId('frag-breakdown-total-Madina97294').textContent).toBe('48')
  })

  it('compte dans le segment quand il tient, repli aligné sur le premier segment masqué', () => {
    mockWidths()
    renderCard()
    const seg = (p: string, c: string) =>
      within(screen.getByTestId(`frag-breakdown-seg-${p}-${c}`)).getByText(String(DATA[p as keyof typeof DATA].find((e) => e.class === c)!.kills))
    expect(seg('JGtm', 'shoulder').style.visibility).toBe('visible')
    expect(seg('JGtm', 'sidearm').style.visibility).toBe('visible')
    expect(seg('JGtm', 'grenade').style.visibility).toBe('visible')
    expect(seg('JGtm', 'unattributed').style.visibility).toBe('hidden')
    const repli = screen.getByTestId('frag-breakdown-repli-JGtm')
    expect(repli.textContent).toBe('2')
    // Premier masqué = unattributed, posé après 28 + 13 + 5 = 46 frags sur une échelle de 48.
    expect(parseFloat(repli.style.paddingLeft)).toBeCloseTo((46 / 48) * 100)
    // Madina97294 : 45 tient, 3 (12 px) ne tient pas.
    expect(screen.getByTestId('frag-breakdown-repli-Madina97294').textContent).toBe('3')
  })

  it('aucun repli quand tout tient', () => {
    vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(function (this: HTMLElement) {
      const width = this.dataset.fitKey ? 500 : this.hasAttribute('data-fit-label') ? 10 : 0
      return { width, height: 0, top: 0, left: 0, right: width, bottom: 0, x: 0, y: 0, toJSON: () => ({}) } as DOMRect
    })
    renderCard()
    expect(screen.queryByTestId('frag-breakdown-repli-JGtm')).toBeNull()
  })

  it('légende des classes en pied de carte, ordre canonique', () => {
    renderCard()
    const legend = screen.getByTestId('chart-card-legend')
    expect(within(legend).getAllByRole('listitem').map((li) => li.textContent)).toEqual([
      'L:shoulder',
      'L:sidearm',
      'L:grenade',
      'L:unattributed',
    ])
  })

  it('sans frag : état vide canonique, pas de légende', () => {
    render(
      <SquadFragBreakdownCard
        fragClassesByPlayer={{}}
        playerOrder={[]}
        playerColors={{}}
        classLabel={(c) => c}
        emptyTitle="Aucune donnée"
        t={t}
      />,
    )
    expect(screen.getByText(t.empty.noBlockData)).toBeTruthy()
    expect(screen.queryByTestId('chart-card-legend')).toBeNull()
  })
})
