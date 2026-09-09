/**
 * Tests — CE QUE LA FRISE DIT DU TEMPS NON JOUÉ (2026-09-07, lot L4).
 *
 * Séparé de `ReplayTimelineTracks.test.tsx` parce que c'est une autre question. Celui-là tient
 * la CHARPENTE de la frise — ses rangées, ses étiquettes, son curseur, son trait de lecture.
 * Celui-ci tient ce que le lot L4 y a ajouté : l'ombrage de présence et sa porte cliquable. Le
 * montage commun vit dans `test/timelineTracksHarness`.
 *
 * LE BADGE DE MÉDAILLE (ex-anneau) N'EST PLUS TESTÉ ICI depuis le 2026-09-09 (décision D7) : il
 * a migré avec son dessin dans `ReplayMarkTrack.test.tsx`, qui monte le composant directement
 * plutôt qu'au travers de toute la frise. Ce fichier ne garde que la hauteur de la piste du
 * sujet, qui reste un fait STRUCTUREL de `ReplayTimelineTracks` (cf. plus bas).
 *
 * Ce qu'ils protègent :
 *  1. UNE PISTE VIDE AVAIT DEUX CAUSES ET UN SEUL DESSIN — le joueur n'était pas là, ou il n'a
 *     rien fait. L'ombre les sépare, et c'est elle qui explique une dominance qui s'effondre à
 *     la huitième minute : l'effectif a baissé.
 *  2. LE BORD DIT LA CONFIANCE. Franc quand l'API date l'événement, dégradé quand le film le
 *     DÉDUIT d'une borne de vie à 10-20 s près. Une frontière au pixel sur une déduction serait
 *     un mensonge de précision.
 *  3. LA PORTE EST LE BOUTON, et elle ne fait qu'une chose : poser le curseur.
 */
import { describe, expect, it } from 'vitest'
import { fireEvent } from '@testing-library/react'

import { trackLeft, trackWidth } from '../model/replayTimelineTracksLogic'
import type { AbsenceStep, PresenceShade } from '../model/presenceTrackLogic'
import { renderTracks } from '../test/timelineTracksHarness'

/**
 * L'OMBRE DE PRÉSENCE ET SA PORTE (2026-09-07, lot L4).
 *
 * Ce que ces cas tiennent, et que la relecture ne verrait pas :
 *  1. LE SENS DE L'OMBRE. Une inversion `joined`/`left` griserait le temps JOUÉ et laisserait
 *     clair le temps absent — un dessin cohérent, entièrement faux, qu'aucune erreur de type ne
 *     signalerait. Les deux formes sont donc vérifiées sur leur `left` et leur `width` réels,
 *     calculés par les mêmes fonctions que les marques (jamais un littéral recopié — cf. le
 *     garde-rail `timelineGeometry.guard.test.ts`).
 *  2. LA PORTE VIT DU CÔTÉ OMBRÉ. L'arrivant recule de sa propre largeur, le partant non. Sans
 *     cette translation, la porte d'un arrivant déborde sur la zone jouée et peut recouvrir le
 *     premier kill — précisément la marque que l'utilisateur cherche après une entrée en cours.
 *  3. LE CLIC NE FAIT QU'UNE CHOSE (décision 1). Il pose le curseur. Il ne met pas en pause et
 *     ne change pas de point de vue : le curseur appartient à l'utilisateur.
 *  4. LE BORD DIT LA CONFIANCE. Franc quand l'API date l'événement, dégradé quand le film le
 *     déduit — et le dégradé part de la FRONTIÈRE, donc son sens dépend de l'espèce d'ombre.
 *  5. LA PISTE DES COÉQUIPIERS N'A PAS DE PORTE. Elle a des paliers, dont l'opacité est la part
 *     de l'effectif absente. Un glyphe y apparaîtrait le jour où l'on brancherait `shades` sur
 *     la mauvaise rangée, et rien d'autre ne le dirait.
 */
describe('ReplayTimelineTracks — l’ombre de présence et sa porte', () => {
  function shade(over: Partial<PresenceShade> = {}): PresenceShade {
    return {
      key: 'p1',
      from: 0,
      to: 0.3,
      kind: 'joined',
      source: 'api',
      edge: 0.3,
      frame: 180,
      clock: '1:00',
      xuid: 'me-1',
      ...over,
    }
  }

  /** L'ombre de QUEUE — celle d'un partant : sa frontière est à gauche, l'ombre court jusqu'au bout. */
  function depart(over: Partial<PresenceShade> = {}): PresenceShade {
    return shade({
      key: 'p2', from: 0.8, to: 1, kind: 'left', edge: 0.8, frame: 480, clock: '8:00', ...over,
    })
  }

  function absence(over: Partial<AbsenceStep> = {}): AbsenceStep {
    return { key: 'a1', from: 0.2, to: 0.6, absent: 1, total: 4, ...over }
  }

  /**
   * LES DEUX RANGÉES DE MARQUES, reconnues à leur FOND et à leur HAUTEUR. Le fond `bg-muted/40`
   * écarte la piste de dominance (même fond, mais `h-2.5`) et la vignette de média ; la hauteur
   * distingue les deux qui restent. C'est aussi ce qui rend le dernier cas de ce bloc — l'écart
   * de hauteur entre le sujet et ses coéquipiers — vérifiable sans un second repère.
   */
  function piste(container: HTMLElement, hauteur: string): HTMLElement {
    const rangees = [...container.querySelectorAll('div')].filter(
      (d) => d.className.includes('bg-muted/40') && d.className.includes(hauteur),
    )
    expect(rangees, `une seule piste doit avoir la hauteur ${hauteur}`).toHaveLength(1)
    return rangees[0] as HTMLElement
  }

  const pisteJoueur = (c: HTMLElement) => piste(c, 'h-[24px]')
  const pisteCoequipiers = (c: HTMLElement) => piste(c, 'h-3.5')

  /** Les bandes ombrées d'une rangée : des spans muets, sans infobulle (les marques en ont une). */
  function ombres(rangee: HTMLElement): HTMLElement[] {
    return [...rangee.querySelectorAll('span[aria-hidden="true"]')].filter(
      (el) => !el.hasAttribute('title'),
    ) as HTMLElement[]
  }

  const portes = (rangee: HTMLElement) => [...rangee.querySelectorAll('button')]

  /**
   * LA VALEUR ATTENDUE PASSE PAR LE MÊME MOTEUR QUE LA VALEUR RENDUE. jsdom RÉÉCRIT ce qu'on lui
   * donne : il réordonne les opérandes d'un `calc()` (`8px + 0.3 * (100% - 16px)` pour un
   * `8px + (100% - 16px) * 0.3` posé) et minuscule `currentColor`. Comparer la chaîne du DOM à
   * celle que rend `trackLeft` échouerait donc sur une différence de SÉRIALISATION, jamais de
   * position — un faux rouge qui pousserait à recopier des littéraux ici, précisément ce que le
   * garde-rail `timelineGeometry.guard.test.ts` interdit. Ce qui est vérifié reste la géométrie,
   * calculée par la fonction du dépôt.
   */
  function css(propriete: string, valeur: string): string {
    const temoin = document.createElement('div')
    temoin.style.setProperty(propriete, valeur)
    return temoin.style.getPropertyValue(propriete)
  }

  it('UN ARRIVANT ombre la TÊTE de la frise, un PARTANT en ombre la QUEUE', () => {
    const { container } = renderTracks({ shades: [shade(), depart()] })
    const [tete, queue] = ombres(pisteJoueur(container))
    // L'ombre de tête part du coup d'envoi et s'arrête à l'entrée en jeu.
    expect(tete.style.left).toBe(css('left', trackLeft(0)))
    expect(tete.style.width).toBe(css('width', trackWidth(0, 0.3)))
    expect(tete.className).toContain('rounded-l-full')
    // Celle de queue part du départ et court jusqu'à la fin.
    expect(queue.style.left).toBe(css('left', trackLeft(0.8)))
    expect(queue.style.width).toBe(css('width', trackWidth(0.8, 1)))
    expect(queue.className).toContain('rounded-r-full')
  })

  /**
   * LA PORTE SE POSE À LA FRONTIÈRE, DU CÔTÉ OMBRÉ. Les deux ont le même `left` — celui de la
   * frontière — et c'est la TRANSLATION qui les sépare : l'arrivant recule d'une largeur pour
   * rentrer dans son ombre, le partant part du bord et déborde vers la sienne.
   */
  it('la porte d’un ARRIVANT recule dans son ombre, celle d’un PARTANT non', () => {
    const { container } = renderTracks({ shades: [shade(), depart()] })
    const [entree, sortie] = portes(pisteJoueur(container))
    expect(entree.style.left).toBe(css('left', trackLeft(0.3)))
    expect(entree.className).toContain('-translate-x-full')
    expect(sortie.style.left).toBe(css('left', trackLeft(0.8)))
    expect(sortie.className).not.toContain('-translate-x-full')
  })

  it('CLIQUER UNE PORTE pose le curseur à l’image de l’événement, et rien d’autre', () => {
    const { container, onSeekFrame, onRequestPause, onSelectViewpoint, onScrub, onToggleTracks } =
      renderTracks({ shades: [depart()] })
    fireEvent.click(portes(pisteJoueur(container))[0])
    expect(onSeekFrame).toHaveBeenCalledWith(480)
    // Décision 1 : le curseur appartient à l'utilisateur — la porte le déplace, elle ne
    // met pas en pause, ne change pas de joueur et ne replie pas la frise.
    expect(onRequestPause).not.toHaveBeenCalled()
    expect(onSelectViewpoint).not.toHaveBeenCalled()
    expect(onScrub).not.toHaveBeenCalled()
    expect(onToggleTracks).not.toHaveBeenCalled()
  })

  it('SOURCE API : le bord est FRANC, et les mots AFFIRMENT', () => {
    const { container } = renderTracks({ shades: [shade({ source: 'api' })] })
    const rangee = pisteJoueur(container)
    expect(ombres(rangee)[0].style.background).not.toContain('linear-gradient')
    const porte = portes(rangee)[0]
    expect(porte.getAttribute('aria-label')).toBe('1:00 — a rejoint la partie')
    expect(porte.title).toContain('Horodatage de participation')
  })

  /**
   * SOURCE FILM : LE DESSIN AVOUE CE QUE LA DONNÉE NE SAIT PAS. Le repli déduit l'instant d'une
   * borne de vie à 10-20 s près : une frontière au pixel serait un mensonge de précision. Le
   * dégradé part toujours de la frontière — `to left` pour une ombre de tête, `to right` pour
   * une ombre de queue — et l'infobulle porte la réserve.
   */
  it('SOURCE FILM : le bord se DÉGRADE depuis la frontière, et les mots restent au fait', () => {
    const { container } = renderTracks({
      shades: [shade({ source: 'film' }), depart({ source: 'film' })],
    })
    const rangee = pisteJoueur(container)
    const [tete, queue] = ombres(rangee)
    expect(tete.style.background).toContain('linear-gradient(to left')
    expect(queue.style.background).toContain('linear-gradient(to right')
    const [entree, sortie] = portes(rangee)
    expect(entree.getAttribute('aria-label')).toBe('1:00 — entre en partie')
    expect(sortie.getAttribute('aria-label')).toBe('8:00 — ne reviendra plus')
    expect(sortie.title).toContain('le film ne les distingue pas')
  })

  /**
   * L'ENCRE DE LA PORTE DIT LE CAMP, celle de l'ombre non. L'ombre est structurelle — une absence
   * ne désigne aucun camp. Le glyphe, lui, nomme quelqu'un : il prend son encre d'équipe quand
   * elle est connue, et l'encre COURANTE sinon. Jamais un camp par défaut : ce serait désigner
   * une équipe au hasard sur le seul repère de la frise qui parle d'une personne.
   */
  it('LE GLYPHE PREND L’ENCRE DU CAMP, et l’encre courante quand le camp est inconnu', () => {
    const encre = (c: HTMLElement) =>
      (pisteJoueur(c).querySelector('button svg') as SVGElement | null)?.style.color

    const allie = renderTracks({
      shades: [shade({ xuid: 'me-1' })],
      identity: new Map([['me-1', { ally: true }]]),
    })
    expect(encre(allie.container)).toContain('--ac-team-ally')
    allie.unmount()

    const adverse = renderTracks({
      shades: [shade({ xuid: 'foe-1' })],
      identity: new Map([['foe-1', { ally: false }]]),
    })
    expect(encre(adverse.container)).toContain('--ac-team-enemy')
    adverse.unmount()

    // Bot sans ligne de tableau de score : aucun camp connu, aucun camp deviné.
    const inconnu = renderTracks({ shades: [shade({ xuid: 'bot:Fantome' })], identity: new Map() })
    expect(encre(inconnu.container)).toBe(css('color', 'currentColor'))
  })

  it('L’OMBRE NE DIT AUCUN CAMP : son encre est structurelle, pas une couleur d’équipe', () => {
    const { container } = renderTracks({
      shades: [shade()],
      identity: new Map([['me-1', { ally: true }]]),
    })
    const fond = ombres(pisteJoueur(container))[0].style.background
    expect(fond).toContain('--muted-foreground')
    expect(fond).not.toContain('team-ally')
  })

  it('DEUX BORNES = DEUX OMBRES ET DEUX PORTES : arrivé PUIS reparti', () => {
    const { container } = renderTracks({ shades: [shade(), depart()] })
    const rangee = pisteJoueur(container)
    expect(ombres(rangee)).toHaveLength(2)
    expect(portes(rangee)).toHaveLength(2)
  })

  /**
   * LA PISTE DES COÉQUIPIERS S'ASSOMBRIT PAR PALIERS, sans une seule porte : à quatre absents
   * possibles, quatre glyphes empilés sur quatorze pixels ne se liraient pas, et la question de
   * cette rangée n'est pas « qui est parti » mais « combien manquaient ». Le fil nomme chacun.
   */
  it('LA PISTE COÉQUIPIERS n’a AUCUNE porte, et son opacité est la part manquante', () => {
    const { container } = renderTracks({ absence: [absence({ absent: 2, total: 4 })] })
    const rangee = pisteCoequipiers(container)
    expect(portes(rangee)).toHaveLength(0)
    const palier = ombres(rangee)[0]
    // Deux absents sur quatre : la moitié de l'alpha maximal (0,5), soit 0,25.
    expect(Number(palier.style.opacity)).toBeCloseTo(0.25, 5)
    expect(palier.style.left).toBe(css('left', trackLeft(0.2)))
    expect(palier.style.width).toBe(css('width', trackWidth(0.2, 0.6)))
  })

  it('les paliers restent sur les COÉQUIPIERS et les ombres sur le SUJET : jamais l’inverse', () => {
    const { container } = renderTracks({ shades: [shade()], absence: [absence()] })
    expect(ombres(pisteJoueur(container))).toHaveLength(1)
    expect(portes(pisteJoueur(container))).toHaveLength(1)
    expect(ombres(pisteCoequipiers(container))).toHaveLength(1)
    expect(portes(pisteCoequipiers(container))).toHaveLength(0)
  })

  /**
   * L'ÉCART DE HAUTEUR EST CE QUI DIT LAQUELLE DES DEUX RANGÉES EST LE SUJET. Celle du joueur
   * regardé est passée de quatorze à dix-huit pixels le 2026-09-07 pour loger l'ex-anneau de
   * médaille et la porte de présence, puis à vingt-quatre le 2026-09-09 (décision D7) pour loger
   * le badge EN IMAGE qui a remplacé cet anneau (cf. `ReplayMarkTrack`, seize pixels de côté).
   * Celle des coéquipiers, qui ne porte ni l'un ni l'autre, n'a pas bougé. Les égaliser rendrait
   * la frise muette sur ce point — et `piste()` ci-dessus ne saurait même plus les distinguer.
   */
  it('la piste du SUJET est plus haute que celle des coéquipiers', () => {
    const { container } = renderTracks()
    expect(pisteJoueur(container).className).toContain('h-[24px]')
    expect(pisteCoequipiers(container).className).toContain('h-3.5')
  })

  it('SANS ÉVÉNEMENT DE PRÉSENCE, aucune ombre : le cas nominal ne se décore pas', () => {
    const { container } = renderTracks({ shades: [], absence: [] })
    expect(ombres(pisteJoueur(container))).toHaveLength(0)
    expect(ombres(pisteCoequipiers(container))).toHaveLength(0)
  })
})
