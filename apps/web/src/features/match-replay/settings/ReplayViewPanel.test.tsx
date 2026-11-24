/**
 * ReplayViewPanel.test.tsx — LE PANNEAU « VUE » en onglets verticaux, sur les vrais réglages : sa structure
 * (rôles, onglets, clavier, fermeture), le contenu de chaque rubrique selon ce que le match propose, les interrupteurs
 * maîtres et leur « N sur M », les épingles, la remise à zéro. Les vues (préréglages souples, Mes vues) sont dans
 * `ReplayViewPanel.views.test.tsx`.
 */
import { act, fireEvent, screen, within } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { makeReplayRelief, THREE_FLOORS } from '../test/reliefFixtures'
import { FULL_FILM, SLAYER_FILM } from '../test/settingsContextFixtures'
import { makeSound } from '../test/soundFixtures'
import { renderOpenPanel } from './replayViewPanelHarness'

const tab = (name: string | RegExp) => screen.getByRole('tab', { name })
const sw = (name: string | RegExp) => screen.getByRole('switch', { name })
const radio = (name: string) => screen.getByRole('radio', { name })

beforeEach(() => window.localStorage.clear())

describe('la structure du panneau', () => {
  it('s’ouvre en dialogue « Vue » avec cinq rubriques en onglets verticaux, la première retenue', () => {
    renderOpenPanel()
    expect(screen.getByRole('dialog', { name: 'Vue' })).toBeInTheDocument()
    const tablist = screen.getByRole('tablist')
    expect(tablist).toHaveAttribute('aria-orientation', 'vertical')
    expect(within(tablist).getAllByRole('tab').map((t) => t.textContent)).toEqual([
      'Joueurs5 sur 8', 'Carte2 sur 3', 'Objets7 sur 10', 'Fil1 sur 1',
    ])
    expect(tab(/Joueurs/)).toHaveAttribute('aria-selected', 'true')
    expect(screen.getAllByRole('tabpanel')).toHaveLength(1)
  })

  it('le son est la cinquième rubrique quand la piste en porte', () => {
    renderOpenPanel({ sound: makeSound() })
    expect(screen.getAllByRole('tab').map((t) => t.textContent?.replace(/\d+ sur \d+/, ''))).toEqual(['Joueurs', 'Carte', 'Objets', 'Fil', 'Son'])
  })

  it('une rubrique à la fois : choisir Carte montre ses lignes et cache celles de Joueurs', () => {
    renderOpenPanel()
    expect(screen.getByRole('switch', { name: 'Visée' })).toBeInTheDocument()
    fireEvent.click(tab(/Carte/))
    expect(tab(/Carte/)).toHaveAttribute('aria-selected', 'true')
    expect(screen.queryByRole('switch', { name: 'Visée' })).toBeNull()
    expect(sw('Zones nommées')).toBeInTheDocument()
  })

  it('les flèches changent de rubrique et lui rendent le focus, Début et Fin vont aux bouts, le tour se referme', () => {
    renderOpenPanel({ sound: makeSound() })
    tab(/Joueurs/).focus()
    fireEvent.keyDown(tab(/Joueurs/), { key: 'ArrowDown' })
    expect(tab(/Carte/)).toHaveAttribute('aria-selected', 'true')
    expect(tab(/Carte/)).toHaveFocus()
    fireEvent.keyDown(tab(/Carte/), { key: 'End' })
    expect(tab(/Son/)).toHaveAttribute('aria-selected', 'true')
    fireEvent.keyDown(tab(/Son/), { key: 'ArrowDown' })
    expect(tab(/Joueurs/)).toHaveAttribute('aria-selected', 'true')
    fireEvent.keyDown(tab(/Joueurs/), { key: 'ArrowUp' })
    expect(tab(/Son/)).toHaveAttribute('aria-selected', 'true')
    fireEvent.keyDown(tab(/Son/), { key: 'Home' })
    expect(tab(/Joueurs/)).toHaveAttribute('aria-selected', 'true')
  })

  it('seul l’onglet retenu se tabule, les autres s’atteignent aux flèches', () => {
    renderOpenPanel()
    expect(tab(/Joueurs/)).toHaveAttribute('tabindex', '0')
    expect(tab(/Carte/)).toHaveAttribute('tabindex', '-1')
  })

  it('retient la dernière rubrique ouverte pour la prochaine ouverture', () => {
    const view = renderOpenPanel()
    fireEvent.click(tab(/Objets/))
    view.unmount()
    renderOpenPanel()
    expect(tab(/Objets/)).toHaveAttribute('aria-selected', 'true')
  })

  it('la rubrique retenue qui n’existe pas pour ce match laisse la place à la première', () => {
    window.localStorage.setItem('replay-view-tab', 'son')
    renderOpenPanel()
    expect(tab(/Joueurs/)).toHaveAttribute('aria-selected', 'true')
  })

  it('le bouton de fermeture, Échap et un clic dehors ferment ; un clic dedans ne ferme rien', () => {
    renderOpenPanel()
    fireEvent.click(sw('Visée'))
    expect(screen.getByRole('dialog', { name: 'Vue' })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Fermer la vue' }))
    expect(screen.queryByRole('dialog')).toBeNull()
    fireEvent.click(screen.getByRole('button', { name: /^Vue\b/ }))
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(screen.queryByRole('dialog')).toBeNull()
    fireEvent.click(screen.getByRole('button', { name: /^Vue\b/ }))
    fireEvent.pointerDown(document.body)
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  it('le focus entre au panneau à l’ouverture, et revient au bouton à la fermeture', () => {
    renderOpenPanel()
    expect(screen.getByRole('dialog', { name: 'Vue' })).toHaveFocus()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(screen.getByRole('button', { name: /^Vue\b/ })).toHaveFocus()
  })

  it('déclare la couche d’Échap pour que le Plein cadre ne parte pas avant le panneau', () => {
    renderOpenPanel()
    expect(screen.getByRole('dialog', { name: 'Vue' })).toHaveAttribute('data-replay-escape-layer')
  })
})

describe('la rubrique Joueurs', () => {
  it('range les lignes dans l’ordre : regard, traînée, couleur, effets, puis le sous-titre Vies', () => {
    renderOpenPanel()
    const panel = screen.getByRole('tabpanel')
    const names = within(panel).getAllByRole('switch').map((s) => s.getAttribute('aria-label'))
    expect(names).toEqual([
      'Tout activer ou désactiver : Joueurs', 'Visée', 'Traînée', 'Effets de tirs', 'Effets de mort',
      'Apparitions', 'Morts', 'Tueurs', 'Probabilité de réapparition',
    ])
    expect(within(panel).getByText('Vies')).toBeInTheDocument()
    expect(within(panel).getByRole('radiogroup', { name: 'Couleur des points' })).toBeInTheDocument()
  })

  it('« qui » et « quand » ne se montrent qu’avec une trace allumée, « point de vue » sous leur nom neuf', () => {
    renderOpenPanel()
    expect(screen.queryByRole('radiogroup', { name: 'Qui' })).toBeNull()
    fireEvent.click(sw('Morts'))
    const qui = screen.getByRole('radiogroup', { name: 'Qui' })
    expect(within(qui).getAllByRole('radio').map((r) => r.textContent)).toEqual(['Tous', 'Mon équipe', 'Point de vue'])
    expect(screen.getByRole('radiogroup', { name: 'Quand' })).toBeInTheDocument()
  })

  it('« point de vue » n’est proposé que s’il y a un joueur suivi dans ce match', () => {
    renderOpenPanel({ film: { ...FULL_FILM, lives: { ...FULL_FILM.lives, followedKnown: false } } })
    fireEvent.click(sw('Morts'))
    expect(within(screen.getByRole('radiogroup', { name: 'Qui' })).getAllByRole('radio').map((r) => r.textContent)).toEqual(['Tous', 'Mon équipe'])
  })

  it('le dessin de la probabilité suit son interrupteur et propose Teintes et points, Teintes, Points', () => {
    renderOpenPanel()
    const dessin = screen.getByRole('radiogroup', { name: 'Dessin' })
    expect(within(dessin).getAllByRole('radio').map((r) => r.textContent)).toEqual(['Teintes et points', 'Teintes', 'Points'])
    expect(radio('Teintes et points')).toHaveAttribute('aria-checked', 'true')
    fireEvent.click(radio('Points'))
    expect(radio('Points')).toHaveAttribute('aria-checked', 'true')
    fireEvent.click(sw('Probabilité de réapparition'))
    expect(screen.queryByRole('radiogroup', { name: 'Dessin' })).toBeNull()
  })

  it('un interrupteur écrit son réglage, qui garde sa clé d’avant', () => {
    renderOpenPanel()
    expect(sw('Visée')).toHaveAttribute('aria-checked', 'true')
    fireEvent.click(sw('Visée'))
    expect(sw('Visée')).toHaveAttribute('aria-checked', 'false')
    expect(window.localStorage.getItem('replay-show-aim')).toBe('false')
    fireEvent.click(sw('Morts'))
    expect(window.localStorage.getItem('replay-show-deaths')).toBe('true')
  })

  it('cliquer le libellé bascule aussi', () => {
    renderOpenPanel()
    fireEvent.click(screen.getByText('Traînée'))
    expect(sw('Traînée')).toHaveAttribute('aria-checked', 'false')
  })

  it('sur un Assassin : ni vies ni probabilité, mais regard, traînée et effets', () => {
    renderOpenPanel({ film: SLAYER_FILM })
    const names = within(screen.getByRole('tabpanel')).getAllByRole('switch').map((s) => s.getAttribute('aria-label'))
    expect(names).toEqual(['Tout activer ou désactiver : Joueurs', 'Visée', 'Traînée', 'Effets de tirs', 'Effets de mort'])
    expect(screen.queryByText('Vies')).toBeNull()
  })

  it('la probabilité refusée dit son motif dans un bloc pointillé, sans interrupteur', () => {
    renderOpenPanel({ film: { ...FULL_FILM, respawnForecast: false, respawnForecastRefusal: 'Cette carte n’a pas de zones nommées.' } })
    expect(screen.queryByRole('switch', { name: 'Probabilité de réapparition' })).toBeNull()
    expect(screen.getByText('Cette carte n’a pas de zones nommées.')).toBeInTheDocument()
  })
})

describe('l’interrupteur maître d’une rubrique', () => {
  const master = () => sw('Tout activer ou désactiver : Joueurs')

  it('dit tout, une partie ou rien, et le compte « N sur M » suit', () => {
    renderOpenPanel()
    expect(master()).toHaveAttribute('aria-checked', 'mixed')
    expect(within(screen.getByRole('tabpanel')).getByText('5 sur 8')).toBeInTheDocument()
    fireEvent.click(master())
    expect(master()).toHaveAttribute('aria-checked', 'true')
    expect(within(screen.getByRole('tabpanel')).getByText('8 sur 8')).toBeInTheDocument()
    fireEvent.click(master())
    expect(master()).toHaveAttribute('aria-checked', 'false')
    expect(within(screen.getByRole('tabpanel')).getByText('0 sur 8')).toBeInTheDocument()
    expect(sw('Visée')).toHaveAttribute('aria-checked', 'false')
  })

  it('n’écrit que ce que le match propose : sur un Assassin, il ne touche pas aux objets', () => {
    window.localStorage.setItem('replay-show-vehicles', 'false')
    renderOpenPanel({ film: SLAYER_FILM })
    fireEvent.click(master())
    expect(window.localStorage.getItem('replay-show-vehicles')).toBe('false')
    // Tout était allumé : le maître éteint les quatre interrupteurs que ce match propose, et rien d'autre.
    expect(sw('Visée')).toHaveAttribute('aria-checked', 'false')
    expect(window.localStorage.getItem('replay-show-zones')).toBeNull()
    expect(window.localStorage.getItem('replay-show-weapon-pads')).toBeNull()
  })
})

describe('la rubrique Carte', () => {
  const relief = (over = {}) =>
    makeReplayRelief({
      options: ['plan', 'rendu', 'relief'], selected: 'plan', meshPercent: 46,
      lines: { available: true, show: true, onToggle: vi.fn() },
      floor: { state: { state: 'ready', chips: THREE_FLOORS, min: 0, max: 4.4 }, cut: null, selectAll: vi.fn(), selectRank: vi.fn(), setHeight: vi.fn(), follow: { available: false, on: false, onToggle: vi.fn() } },
      ...over,
    })
  const open = (o = {}) => {
    const view = renderOpenPanel({ relief: relief(), orientation: { retained: null, setTurns: vi.fn(), available: true }, ...o })
    fireEvent.click(tab(/Carte/))
    return view
  }

  it('propose Plan, Rendu, Relief dans cet ordre, le fond retenu pressé, l’infobulle du Relief dit le maillage', () => {
    open()
    const fond = screen.getByRole('radiogroup', { name: 'Fond' })
    expect(within(fond).getAllByRole('radio').map((r) => r.textContent)).toEqual(['Plan', 'Rendu', 'Relief'])
    expect(radio('Plan')).toHaveAttribute('aria-checked', 'true')
    expect(radio('Relief').title).toContain('46 % de la carte')
  })

  it('un clic sur un fond le retient dans la clé globale partagée avec la Tactique', () => {
    open()
    fireEvent.click(radio('Relief'))
    expect(window.localStorage.getItem('replay-background-style')).toBe('relief')
  })

  it('sans choix de fond : pas de segmenté', () => {
    open({ relief: relief({ options: [] }) })
    expect(screen.queryByRole('radiogroup', { name: 'Fond' })).toBeNull()
  })

  it('« Traits du relief » (celui du fond affiché) se lit sous le fond ; « Plan des murs » n’est pas proposé sur le fond Plan', () => {
    const { unmount } = open({ drawn: 'plan' })
    expect(sw('Traits du relief')).toBeInTheDocument()
    expect(screen.queryByRole('switch', { name: 'Plan des murs' })).toBeNull()
    unmount()
    open({ drawn: 'rendu' })
    expect(sw('Plan des murs')).toHaveAttribute('aria-checked', 'true')
  })

  it('l’étage : Tout puis un rang par étage, sans curseur tant qu’aucune coupe n’est active', () => {
    const r = relief()
    open({ relief: r })
    const etage = screen.getByRole('radiogroup', { name: 'Étage' })
    expect(within(etage).getAllByRole('radio').map((x) => x.textContent)).toEqual(['Tout', 'Bas', 'Étage 1', 'Étage 2'])
    expect(radio('Tout')).toHaveAttribute('aria-checked', 'true')
    expect(screen.queryByRole('slider', { name: 'Hauteur de la coupe' })).toBeNull()
    fireEvent.click(radio('Étage 1'))
    expect(r.floor.selectRank).toHaveBeenCalledWith(2.2)
  })

  it('une coupe active montre le curseur « Hauteur », qui dit le rang le plus proche', () => {
    const r = relief({ floor: { ...relief().floor, cut: 2.2 } })
    open({ relief: r })
    const curseur = screen.getByRole('slider', { name: 'Hauteur de la coupe' })
    expect(curseur).toHaveAttribute('step', '0.25')
    expect(screen.getByText('Étage 1', { selector: 'output' })).toBeInTheDocument()
    fireEvent.change(curseur, { target: { value: '3' } })
    expect(r.floor.setHeight).toHaveBeenCalledWith(3)
    expect(radio('Étage 1')).toHaveAttribute('aria-checked', 'true')
  })

  it('carte sans maillage : un bloc d’explication, aucun choix d’étage', () => {
    open({ relief: relief({ floor: { ...relief().floor, state: { state: 'noMesh' } } }) })
    expect(screen.queryByRole('radiogroup', { name: 'Étage' })).toBeNull()
    expect(screen.getByText(/sans maillage de navigation/)).toBeInTheDocument()
    expect(screen.queryByRole('switch', { name: 'Étage de la fiche ouverte' })).toBeNull()
  })

  it('« Étage de la fiche ouverte » : grisé et expliqué sans fiche, actif avec une fiche, la note dit le rang suivi', () => {
    const off = relief()
    const { unmount } = open({ relief: off })
    expect(sw('Étage de la fiche ouverte')).toBeDisabled()
    expect(screen.getByText(/Aucun joueur ouvert/)).toBeInTheDocument()
    unmount()
    const on = relief({ floor: { ...relief().floor, cut: 2.2, follow: { available: true, on: true, onToggle: vi.fn() } } })
    open({ relief: on })
    expect(sw('Étage de la fiche ouverte')).toBeEnabled()
    expect(screen.getByText('Étage suivi : Étage 1')).toBeInTheDocument()
    fireEvent.click(sw('Étage de la fiche ouverte'))
    expect(on.floor.follow.onToggle).toHaveBeenCalledTimes(1)
  })

  it('l’orientation : Auto puis les quatre quarts de tour ; Auto efface le choix retenu', () => {
    const setTurns = vi.fn()
    open({ orientation: { retained: 1, setTurns, available: true } })
    const groupe = screen.getByRole('radiogroup', { name: 'Orientation' })
    expect(within(groupe).getAllByRole('radio').map((r) => r.textContent)).toEqual(['Auto', '0°', '90°', '180°', '270°'])
    expect(radio('90°')).toHaveAttribute('aria-checked', 'true')
    fireEvent.click(radio('Auto'))
    expect(setTurns).toHaveBeenLastCalledWith(null)
    fireEvent.click(radio('270°'))
    expect(setTurns).toHaveBeenLastCalledWith(3)
  })

  it('sans carte connue, pas d’orientation', () => {
    open({ orientation: { retained: null, setTurns: vi.fn(), available: false } })
    expect(screen.queryByRole('radiogroup', { name: 'Orientation' })).toBeNull()
  })

  it('la carte de chaleur : Mesure et Durée sous elle, allumée seulement ; pas d’« éliminations » sans mort localisable', () => {
    open({ heatKillsAvailable: false })
    expect(screen.queryByRole('radiogroup', { name: 'Durée' })).toBeNull()
    fireEvent.click(sw('Carte de chaleur'))
    expect(screen.getByRole('radiogroup', { name: 'Durée' })).toBeInTheDocument()
    expect(screen.queryByRole('radiogroup', { name: 'Mesure' })).toBeNull()
  })

  it('la mesure propose Présence et Éliminations quand une mort est localisable, et la choisit', () => {
    open()
    fireEvent.click(sw('Carte de chaleur'))
    expect(within(screen.getByRole('radiogroup', { name: 'Mesure' })).getAllByRole('radio').map((r) => r.textContent)).toEqual(['Présence', 'Éliminations'])
    fireEvent.click(radio('Éliminations'))
    expect(window.localStorage.getItem('replay-heatmap-mode')).toBe('kills')
  })
})

describe('la rubrique Objets', () => {
  const open = (o = {}) => {
    const view = renderOpenPanel(o)
    fireEvent.click(tab(/Objets/))
    return view
  }

  it('les filles se lisent sous leur mère, et disparaissent quand elle est éteinte', () => {
    open()
    expect(sw('Armes spéciales seulement')).toBeInTheDocument()
    expect(sw('Y compris les armes de départ')).toBeInTheDocument()
    expect(sw('Objets lâchés')).toBeInTheDocument()
    expect(sw('Objets non identifiés')).toBeInTheDocument()
    fireEvent.click(sw('Armes au sol'))
    fireEvent.click(sw('Équipements posés'))
    for (const fille of ['Armes spéciales seulement', 'Y compris les armes de départ', 'Objets lâchés', 'Objets non identifiés']) {
      expect(screen.queryByRole('switch', { name: fille })).toBeNull()
    }
  })

  it('« Drapeaux » reste atteignable quand « Objectifs du mode » est éteint : sa couche reste dessinée', () => {
    open()
    fireEvent.click(sw('Objectifs du mode'))
    expect(sw('Drapeaux')).toHaveAttribute('aria-checked', 'true')
    fireEvent.click(sw('Drapeaux'))
    expect(window.localStorage.getItem('replay-show-flag-carries')).toBe('false')
  })

  it('un film qui porte un VIP, un crâne et une bombe les propose sous les objectifs', () => {
    open({ film: { ...FULL_FILM, vipCrown: true, skullCarrier: true, bombCarrier: true } })
    for (const nom of ['VIP', 'Porteur du crâne', 'Bombe']) expect(sw(nom)).toHaveAttribute('aria-checked', 'true')
  })

  it('un film sans pose dessinable : ni le calque ni ses filles ; sans socle : pas de bascule', () => {
    open({ film: { ...FULL_FILM, placements: { drawable: false, unnamed: true, dropped: true }, weaponPads: false } })
    for (const nom of ['Équipements posés', 'Objets lâchés', 'Objets non identifiés', 'Emplacements d’arme']) {
      expect(screen.queryByRole('switch', { name: nom })).toBeNull()
    }
  })

  it('un Assassin n’a pas de rubrique Objets', () => {
    renderOpenPanel({ film: SLAYER_FILM })
    expect(screen.queryByRole('tab', { name: /Objets/ })).toBeNull()
  })
})

describe('la rubrique Fil', () => {
  const open = (o = {}) => {
    const view = renderOpenPanel(o)
    fireEvent.click(tab(/^Fil/))
    return view
  }

  it('le filtre du fil est une liste avec le nombre de lignes de chaque filtre ; Objectifs n’y est pas sans ligne d’objectif', () => {
    open()
    const liste = screen.getByRole('combobox', { name: 'Filtre du fil' })
    const options = within(liste).getAllByRole('option').map((o) => o.textContent)
    expect(options[0]).toBe('Tout (0)')
    expect(options.join(' ')).not.toContain('Objectifs')
  })

  it('choisir un filtre le retient pour la colonne du fil, et la liste le relit', () => {
    open()
    fireEvent.change(screen.getByRole('combobox', { name: 'Filtre du fil' }), { target: { value: 'kills' } })
    expect(window.localStorage.getItem('replay-feed-filter')).toBe('kills')
    expect(screen.getByRole('combobox', { name: 'Filtre du fil' })).toHaveValue('kills')
  })

  it('« Fil posé sur le plan » est proposé là où le cadre le pose, sinon une phrase dit où le régler', () => {
    const { unmount } = open({ planFeedAvailable: true })
    expect(sw('Fil posé sur le plan')).toHaveAttribute('aria-checked', 'true')
    unmount()
    open({ planFeedAvailable: false })
    expect(screen.queryByRole('switch', { name: 'Fil posé sur le plan' })).toBeNull()
    expect(screen.getByText('Le fil posé sur le plan se règle au Cinéma et au Plein cadre.')).toBeInTheDocument()
  })
})

describe('la rubrique Son', () => {
  const open = (sound = makeSound()) => {
    const view = renderOpenPanel({ sound })
    fireEvent.click(tab(/Son/))
    return { view, sound }
  }

  it('sans piste sonore : ni rubrique ni commande', () => {
    renderOpenPanel()
    expect(screen.queryByRole('tab', { name: /Son/ })).toBeNull()
  })

  it('l’interrupteur du son, le volume et les quatre catégories, au même état que la barre', () => {
    const { sound } = open()
    expect(sw('Son')).toHaveAttribute('aria-checked', 'false')
    fireEvent.click(sw('Son'))
    expect(sound.toggle).toHaveBeenCalledTimes(1)
    for (const c of ['Armes', 'Grenades', 'Mêlée', 'Équipements']) expect(sw(c)).toHaveAttribute('aria-checked', 'true')
    fireEvent.click(sw('Mêlée'))
    expect(sound.toggleCategory).toHaveBeenCalledWith('melee')
  })

  it('le volume est grisé et à zéro tant que le son est coupé ; allumé, il écrit le niveau', () => {
    const { view, sound } = open()
    expect(screen.getByRole('slider', { name: /Volume/ })).toBeDisabled()
    expect(screen.getByText('0 %')).toBeInTheDocument()
    view.unmount()
    const on = makeSound({ on: true, volume: 0.4 })
    open(on)
    const curseur = screen.getByRole('slider', { name: /Volume/ })
    expect(curseur).toBeEnabled()
    expect(curseur).toHaveValue('40')
    fireEvent.change(curseur, { target: { value: '65' } })
    expect(on.setVolume).toHaveBeenCalledWith(0.65)
    expect(sound.setVolume).not.toHaveBeenCalled()
  })

  it('le maître du son compte l’interrupteur du son et les quatre catégories', () => {
    open()
    expect(within(screen.getByRole('tabpanel')).getByText('4 sur 5')).toBeInTheDocument()
  })
})

describe('la barre de lecture fine', () => {
  it('absente quand le cadre ne peut pas poser la barre sur le plan', () => {
    renderOpenPanel({ thinBarAvailable: false })
    expect(screen.queryByRole('switch', { name: 'Barre de lecture fine' })).toBeNull()
  })

  it('au Plein cadre : une ligne « Lecture » au pied du panneau, avec l’état retenu, qui bascule', () => {
    renderOpenPanel({ thinBarAvailable: true })
    expect(screen.getByText('Lecture')).toBeInTheDocument()
    expect(sw('Barre de lecture fine')).toHaveAttribute('aria-checked', 'true')
    fireEvent.click(sw('Barre de lecture fine'))
    expect(window.localStorage.getItem('replay-thin-bar-full-frame')).toBe('false')
  })
})

describe('l’accès rapide : épingler depuis une ligne', () => {
  it('les épingles d’usine sont posées ; la punaise d’une ligne l’épingle ou la retire', () => {
    renderOpenPanel()
    expect(screen.getByRole('button', { name: 'Retirer « Visée » de l’accès rapide' })).toHaveAttribute('aria-pressed', 'true')
    fireEvent.click(screen.getByRole('button', { name: 'Retirer « Visée » de l’accès rapide' }))
    expect(screen.getByRole('button', { name: 'Épingler « Visée » à l’accès rapide' })).toHaveAttribute('aria-pressed', 'false')
    expect(JSON.parse(window.localStorage.getItem('replay-pins') ?? '[]')).toEqual(['trail', 'heatmap', 'planFeed'])
  })

  it('une cinquième épingle passe, une sixième est refusée avec la raison', () => {
    renderOpenPanel()
    fireEvent.click(screen.getByRole('button', { name: 'Épingler « Effets de tirs » à l’accès rapide' }))
    expect(JSON.parse(window.localStorage.getItem('replay-pins') ?? '[]')).toHaveLength(5)
    fireEvent.click(screen.getByRole('button', { name: 'Épingler « Effets de mort » à l’accès rapide' }))
    expect(screen.getByRole('status')).toHaveTextContent('5 réglages au plus dans l’accès rapide')
    expect(JSON.parse(window.localStorage.getItem('replay-pins') ?? '[]')).toHaveLength(5)
  })

  it('les filles ne s’épinglent pas : pas de punaise sur « Armes spéciales seulement »', () => {
    renderOpenPanel()
    fireEvent.click(tab(/Objets/))
    expect(screen.queryByRole('button', { name: /Armes spéciales seulement.*accès rapide/ })).toBeNull()
    expect(screen.getByRole('button', { name: 'Épingler « Armes au sol » à l’accès rapide' })).toBeInTheDocument()
  })
})

describe('réinitialiser', () => {
  it('demande confirmation, remet les réglages d’usine, garde Mes vues et les épingles', () => {
    renderOpenPanel()
    fireEvent.click(sw('Visée'))
    fireEvent.click(screen.getByRole('button', { name: 'Épingler « Effets de tirs » à l’accès rapide' }))
    fireEvent.click(screen.getByRole('button', { name: 'Réinitialiser' }))
    expect(screen.getByText('Remettre les réglages d’usine ?')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Confirmer' }))
    expect(sw('Visée')).toHaveAttribute('aria-checked', 'true')
    expect(screen.getByRole('status')).toHaveTextContent('Vos vues et vos épingles sont conservées')
    expect(JSON.parse(window.localStorage.getItem('replay-pins') ?? '[]')).toContain('shotFx')
  })

  it('Annuler et Échap défont la confirmation sans rien changer ; Échap ne ferme pas le panneau cette fois', () => {
    renderOpenPanel()
    fireEvent.click(sw('Visée'))
    fireEvent.click(screen.getByRole('button', { name: 'Réinitialiser' }))
    fireEvent.click(screen.getByRole('button', { name: 'Annuler' }))
    expect(screen.queryByText('Remettre les réglages d’usine ?')).toBeNull()
    expect(sw('Visée')).toHaveAttribute('aria-checked', 'false')
    fireEvent.click(screen.getByRole('button', { name: 'Réinitialiser' }))
    act(() => {
      fireEvent.keyDown(screen.getByRole('button', { name: 'Confirmer' }), { key: 'Escape' })
    })
    expect(screen.queryByText('Remettre les réglages d’usine ?')).toBeNull()
    expect(screen.getByRole('dialog', { name: 'Vue' })).toBeInTheDocument()
  })
})

describe('en anglais', () => {
  it('les rubriques, les lignes renommées et le pied portent leur libellé anglais', () => {
    renderOpenPanel({ locale: 'en', sound: makeSound() })
    expect(screen.getByRole('dialog', { name: 'View' })).toBeInTheDocument()
    expect(screen.getAllByRole('tab').map((t) => t.textContent?.replace(/\d+ of \d+/, ''))).toEqual(['Players', 'Map', 'Objects', 'Feed', 'Sound'])
    expect(sw('Aim')).toBeInTheDocument()
    expect(sw('Turn all on or off: Players')).toBeInTheDocument()
    fireEvent.click(tab(/Map/))
    expect(sw('Named zones')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Reset' })).toBeInTheDocument()
  })
})
