/**
 * Tests — ReplayTimelineTracks (la frise habillée et ses quatre pistes).
 *
 * Ce qu'ils protègent :
 *  1. LES QUATRE PISTES SONT NOMMÉES. Une bande de trois pixels sans étiquette ne se lit pas.
 *     La première ne l'est plus par un mot mais par le MENU de point de vue (2026-09-07, L3) :
 *     son nom accessible et son losange d'ami ont leurs propres blocs, en fin de fichier.
 *  2. LA PISTE MÉDIAS RESTE, MÊME VIDE (demande utilisateur du 2026-08-28) — et son vide est
 *     une phrase, pas une bande grise muette. C'est l'exception assumée à la règle « pas de
 *     commande quand il n'y a rien à commander » : ce n'est pas une commande, c'est un lieu.
 *  3. OUVRIR UN MÉDIA MET LE REJEU EN PAUSE. Le composant ne connaît pas la boucle : il DEMANDE
 *     la pause à l'appelant, et seulement si la lecture tourne.
 *  4. AUCUN HEX. Les encres passent par les tokens du thème (règle color-tokens).
 *  5. LE TRAIT DE LECTURE (2026-09-06) : un seul, dans la géométrie des marques, et SEULEMENT
 *     quand la frise est dépliée. Plus le lien non typé qui le nourrit — l'attribut de la
 *     racine où la lecture vient poser la position du curseur.
 *
 * L'OMBRAGE DE PRÉSENCE ET L'ANNEAU DES MÉDAILLES (lot L4) ont leur propre fichier,
 * `ReplayTimelineTracks.presence.test.tsx` : ils forment une responsabilité à part — ce que la
 * frise dit du temps NON JOUÉ — et les garder ici aurait fait franchir à ce fichier le seuil de
 * cinq cents lignes de code. Le montage commun aux deux vit dans `test/timelineTracksHarness`.
 */
import { describe, expect, it } from 'vitest'
import { createRef } from 'react'
import { fireEvent, screen } from '@testing-library/react'

import { trackLeftVar } from '../model/replayTimelineTracksLogic'
import type { PlacedMedia, ReplayMediaItem } from '../model/replayTimelineTracksLogic'
import { CURSOR_HOST_ATTR, CURSOR_RATIO_VAR } from '../hooks/useReplayPlayback'
import { TIMELINE_SHORTCUT_ATTR } from '../hooks/useReplayShortcuts'
import { mark, renderTracks } from '../test/timelineTracksHarness'

function mediaItem(over: Partial<ReplayMediaItem> = {}): ReplayMediaItem {
  return {
    id: 'media-1',
    kind: 'image',
    replayMs: 30_000,
    thumbUrl: '/thumb.png',
    url: '/full.png',
    label: 'Capture Streets',
    ...over,
  }
}

function placed(over: Partial<PlacedMedia> = {}): PlacedMedia {
  return { item: mediaItem(), from: 0.5, to: 0.5, ...over }
}

describe('ReplayTimelineTracks — les quatre pistes sont nommées', () => {
  /**
   * LA PREMIÈRE RANGÉE N'A PLUS D'ÉTIQUETTE (2026-09-07, lot L3) : elle porte le MENU de point
   * de vue, dont le texte visible est le gamertag regardé. « Toi » y serait faux dès qu'on
   * regarde quelqu'un d'autre. Les trois autres restent des mots.
   */
  it('porte le menu de point de vue, puis les étiquettes Coéquipiers, Dominance et Médias', () => {
    renderTracks()
    expect(screen.getByLabelText('Joueur suivi')).toBeTruthy()
    expect(screen.queryByText('Toi')).toBeNull()
    for (const label of ['Coéquipiers', 'Dominance', 'Médias']) {
      expect(screen.getByText(label)).toBeTruthy()
    }
  })

  // LES TROIS BORNES DE L'AXE ONT DISPARU LE 2026-09-02 : le temps suit desormais le point qui
  // avance (demande utilisateur). Ce test garde la BULLE — sa presence, son ancrage sur la meme
  // variable que le remplissage, et le fait qu'elle ne redonne PAS un nom accessible deja pris.
  it('le temps se lit sous le curseur, pas en trois bornes figees', () => {
    const ref = createRef<HTMLSpanElement>()
    renderTracks({ clockRef: ref })
    expect(ref.current).toBeTruthy()
    expect(ref.current?.getAttribute('aria-hidden')).toBe('true')
    // La bulle suit le curseur par la MEME position que le trait et les marques : le ratio nu
    // ecrit par `writeCursor`, passe dans la geometrie de piste (corrige le 2026-09-06 — elle
    // lisait le pourcentage brut, et portait donc jusqu'a 8 px de decalage).
    expect(ref.current?.getAttribute('style') ?? '').toContain(trackLeftVar(CURSOR_RATIO_VAR))
  })

  it('le curseur garde ses bornes et son nom accessible', () => {
    renderTracks({ minFrame: 149, maxFrame: 4_929 })
    const frise = screen.getByLabelText('Temps de match') as HTMLInputElement
    expect(frise.tagName).toBe('INPUT')
    expect(frise).toHaveAttribute('min', '149')
    expect(frise).toHaveAttribute('max', '4929')
  })

  /**
   * GARDE-FOU DU LIEN COMPOSANT <-> CLAVIER (décision utilisateur du 2026-08-28).
   *
   * L'exemption de la garde anti-frappe est NOMINATIVE : `useReplayShortcuts` cherche cet
   * attribut, ce composant le pose. Rien dans le typage ne relie les deux — retirer l'attribut
   * du champ compilerait, passerait tous les autres tests, et rendrait muets Espace et les
   * flèches dès le premier clic sur la frise. C'est exactement ce qu'un garde-rail attrape, et
   * la constante est importée du hook pour que le renommer casse ici aussi.
   */
  it('la frise PORTE l’attribut qui lui rend les raccourcis clavier', () => {
    renderTracks()
    expect(screen.getByLabelText('Temps de match')).toHaveAttribute(TIMELINE_SHORTCUT_ATTR)
  })

  /**
   * MÊME GARDE-FOU, SECOND LIEN NON TYPÉ (2026-09-06) : `useReplayPlayback.writeCursor` remonte
   * du champ jusqu'à la racine de la frise par `CURSOR_HOST_ATTR` pour y poser `--played` et
   * `--played-r`. Retirer l'attribut compilerait et ne casserait aucun autre test — la pose
   * retomberait silencieusement sur la rangée du champ, où les PISTES ne la voient pas : le
   * trait de lecture resterait figé à l'origine pendant que le curseur avance.
   */
  it('la RACINE porte l’attribut où la lecture vient poser la position du curseur', () => {
    const { container } = renderTracks()
    const racine = container.querySelector(`[${CURSOR_HOST_ATTR}]`)
    expect(racine).toBeTruthy()
    // Et c'est bien la racine, pas un noeud interne : le champ doit être un de ses descendants.
    expect(racine?.contains(screen.getByLabelText('Temps de match'))).toBe(true)
  })
})

/**
 * LE TRAIT DE LECTURE (demande utilisateur du 2026-09-06).
 *
 * Ce que ces cas tiennent :
 *  1. IL N'EXISTE QUE DÉPLIÉE. Repliée, la frise n'a aucune piste à traverser : un trait y
 *     serait un trait vers rien, posé sur une hauteur nulle.
 *  2. IL SUIT LA MÊME GÉOMÉTRIE QUE LES MARQUES. C'est tout l'intérêt du lot — un trait qui
 *     désigne un kill à 8 px près ne désigne rien.
 *  3. IL NE CAPTE PAS LE POINTEUR ET NE PARLE PAS. La frise reste saisissable sous lui, et il
 *     ne redonne pas un nom accessible que le champ expose déjà.
 */
describe('ReplayTimelineTracks — le trait de lecture', () => {
  /**
   * Le trait : positionné par la géométrie de piste sur la variable de lecture, et SANS `clamp`.
   *
   * La bulle de temps emploie la même formule — c'est tout l'objet du lot, elles partagent une
   * géométrie — mais elle l'enveloppe d'un `clamp` qui la retient dans la frise à ses deux
   * bouts. Le trait, lui, va jusqu'au bord : c'est ce qui les distingue ici.
   */
  function traits(container: HTMLElement): Element[] {
    return [...container.querySelectorAll('span')].filter((el) => {
      const style = el.getAttribute('style') ?? ''
      return style.includes(trackLeftVar(CURSOR_RATIO_VAR)) && !style.includes('clamp')
    })
  }

  it('dépliée, un trait unique suit la lecture dans la géométrie des marques', () => {
    const { container } = renderTracks({ tracksExpanded: true })
    expect(traits(container)).toHaveLength(1)
  })

  it('REPLIÉE, il n’y a pas de trait : il n’y a plus de piste à traverser', () => {
    const { container } = renderTracks({ tracksExpanded: false })
    expect(traits(container)).toHaveLength(0)
  })

  it('il ne capte pas le pointeur et ne redonne pas un nom déjà pris', () => {
    const { container } = renderTracks({ tracksExpanded: true })
    const conteneur = traits(container)[0].parentElement
    expect(conteneur?.className).toContain('pointer-events-none')
    expect(conteneur?.getAttribute('aria-hidden')).toBe('true')
  })
})

describe('ReplayTimelineTracks — les marques et la dominance', () => {
  it('pose chaque marque avec son horloge en infobulle', () => {
    const { container } = renderTracks({
      own: [mark({ key: 'k1', clock: '1:12' }), mark({ key: 'd1', kind: 'death', clock: '3:40' })],
      teammates: [mark({ key: 'a1', clock: '2:02' })],
    })
    expect(container.querySelectorAll('[title="1:12"]')).toHaveLength(1)
    expect(container.querySelectorAll('[title="3:40"]')).toHaveLength(1)
    expect(container.querySelectorAll('[title="2:02"]')).toHaveLength(1)
  })

  /**
   * UNE MARQUE EST CENTRÉE SUR SON INSTANT (décision utilisateur du 2026-09-06).
   *
   * Ce que ce cas tient, et il ne se voit pas à la relecture : la marque est POSITIONNÉE par
   * `trackLeft(ratio)`, qui rend le point exact du frag — mais un élément de deux à trois pixels
   * posé à ce `left` déborde tout entier vers la DROITE. Sans la translation de sa demi-largeur,
   * son milieu tombe après l'instant qu'elle désigne, et le trait de lecture — qui, lui, passe
   * pile sur le point — la longe au lieu de la couper. Le `left` resterait pourtant juste : rien
   * d'autre que ce cas ne verrait la différence.
   */
  it('CENTRE chaque marque sur son instant, quelle que soit la piste', () => {
    const { container } = renderTracks({
      own: [mark({ key: 'k1', clock: '1:12' })],
      teammates: [mark({ key: 'a1', clock: '2:02' })],
    })
    for (const horloge of ['1:12', '2:02']) {
      const marque = container.querySelector(`[title="${horloge}"]`)
      expect(marque?.className, `la marque ${horloge} n’est pas centrée sur son instant`).toContain(
        '-translate-x-1/2',
      )
    }
  })

  it('nomme le meneur d’une bande de dominance, avec le libellé du scoreboard', () => {
    renderTracks({
      dominance: [{ key: 's1', from: 0, to: 0.6, teamId: 1 }],
      allyOf: () => true,
      labelOf: () => 'Cobalt',
    })
    expect(screen.getByTitle('Cobalt mène aux frags')).toBeTruthy()
  })

  /**
   * L'ÉGALITÉ EST BLEUE (demande utilisateur du 2026-08-28), et du MÊME bleu que partout
   * ailleurs : `outcome-draw`, l'encre des matchs nuls, des tuiles neutres et des barres de
   * bilan. Le test tient le TOKEN, pas la valeur — la palette daltonisme le change, le sens
   * ne change pas.
   */
  it('une bande d’ÉGALITÉ porte l’encre d’égalité du dépôt, et se nomme dans l’infobulle', () => {
    renderTracks({ dominance: [{ key: 'tie', from: 0, to: 0.4, teamId: null }] })
    const bande = screen.getByTitle('Égalité aux frags')
    expect(bande.getAttribute('style')).toContain('--ac-outcome-draw')
  })

  /**
   * LA PISTE SCORE (demande utilisateur du 2026-08-28). Ce qu'elle protège : l'ABSENCE est un
   * état à part entière — `score: null` veut dire « ce mode n'a pas de piste score » (Slayer),
   * pas « la piste est vide ». Une rangée vide se lirait « personne n'a marqué ».
   */
  it('SANS piste score, la rangée n’existe pas — elle n’est pas vide, elle est absente', () => {
    renderTracks({ score: null })
    expect(screen.queryByText('Score')).toBeNull()
  })

  it('AVEC une piste score : sa rangée est nommée, et ses bandes se distinguent des frags', () => {
    renderTracks({
      score: {
        segments: [
          { key: 't', from: 0, to: 0.3, teamId: null },
          { key: 's1', from: 0.3, to: 1, teamId: 1 },
        ],
        rounds: [],
      },
      dominance: [{ key: 'd1', from: 0, to: 1, teamId: 0 }],
      allyOf: (id) => id === 1,
      labelOf: (id) => (id === 1 ? 'Cobalt' : 'Ambre'),
    })
    expect(screen.getByText('Score')).toBeTruthy()
    // Les deux rangées répondent à deux questions : leurs infobulles ne doivent pas se confondre.
    expect(screen.getByTitle('Cobalt mène au score')).toBeTruthy()
    expect(screen.getByTitle('Ambre mène aux frags')).toBeTruthy()
    expect(screen.getByTitle('Égalité au score')).toBeTruthy()
  })

  it('un SÉPARATEUR DE MANCHE se pose sur la piste score et dit quelle manche s’achève', () => {
    renderTracks({
      score: {
        segments: [{ key: 's1', from: 0, to: 1, teamId: 0 }],
        rounds: [{ key: 'r1', endedIndex: 1, ratio: 0.5 }],
      },
    })
    expect(screen.getByTitle('Manche 1 terminée')).toBeTruthy()
  })

  it('N’ÉCRIT AUCUN HEX : toutes les encres passent par les tokens du thème', () => {
    const { container } = renderTracks({
      own: [mark()],
      teammates: [mark({ key: 'a1' })],
      dominance: [{ key: 's1', from: 0, to: 1, teamId: 0 }],
    })
    expect(container.innerHTML).not.toMatch(/#[0-9a-fA-F]{6}/)
  })
})

describe('ReplayTimelineTracks — la piste médias', () => {
  it('reste affichée MÊME VIDE, et le dit en toutes lettres', () => {
    renderTracks({ media: [] })
    expect(screen.getByText('Aucun média sur ce match')).toBeTruthy()
  })

  it('un média porte son libellé et devient un bouton', () => {
    renderTracks({ media: [placed()] })
    expect(screen.getByRole('button', { name: 'Capture Streets' })).toBeTruthy()
    expect(screen.queryByText('Aucun média sur ce match')).toBeNull()
  })

  it('OUVRIR MET LE REJEU EN PAUSE quand il tourne, et montre le média en grand', () => {
    const { onRequestPause } = renderTracks({ media: [placed()], playing: true })
    fireEvent.click(screen.getByRole('button', { name: 'Capture Streets' }))
    expect(onRequestPause).toHaveBeenCalledTimes(1)
    expect(screen.getByRole('dialog', { name: 'Capture Streets' })).toBeTruthy()
    expect(screen.getByText('Rejeu en pause')).toBeTruthy()
  })

  it('déjà en pause : rien à demander, la lightbox s’ouvre quand même', () => {
    const { onRequestPause } = renderTracks({ media: [placed()], playing: false })
    fireEvent.click(screen.getByRole('button', { name: 'Capture Streets' }))
    expect(onRequestPause).not.toHaveBeenCalled()
    expect(screen.getByRole('dialog', { name: 'Capture Streets' })).toBeTruthy()
  })

  it('Échap referme la lightbox', () => {
    renderTracks({ media: [placed()] })
    fireEvent.click(screen.getByRole('button', { name: 'Capture Streets' }))
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  /**
   * VIDE N'EST PAS ABSENTE. Les deux états se ressemblent à l'écran et ne disent pas la même
   * chose : « aucun média sur ce match » est un fait du match, tandis qu'un titre sans médias
   * n'a rien à dire du tout — la rangée y mentirait. Le rejeu n'étant gardé que par
   * `matchmaking`, rien d'autre que cette prop ne retire la piste.
   */
  it('LA RANGÉE DISPARAÎT quand le titre ne porte pas les médias (ni piste, ni phrase de vide)', () => {
    renderTracks({ media: [], showMediaTrack: false })
    expect(screen.queryByText('Médias')).toBeNull()
    expect(screen.queryByText('Aucun média sur ce match')).toBeNull()
    // Les trois autres pistes et le curseur restent intacts.
    expect(screen.getByLabelText('Joueur suivi')).toBeTruthy()
    for (const label of ['Coéquipiers', 'Dominance']) {
      expect(screen.getByText(label)).toBeTruthy()
    }
    expect(screen.getByLabelText('Temps de match')).toBeTruthy()
  })
})

/**
 * LE REPLI (retour utilisateur du 2026-08-28). Ce qu'il protège : replier ne doit RIEN coûter
 * de ce qui fait le lecteur. Le curseur, ses horloges et — surtout — l'attribut qui lui rend
 * les raccourcis clavier restent en place ; seules les pistes s'en vont.
 */
describe('ReplayTimelineTracks — le repli des pistes', () => {
  it('REPLIÉ : plus une seule piste, mais le curseur et son temps restent', () => {
    const ref = createRef<HTMLSpanElement>()
    renderTracks({ tracksExpanded: false, media: [placed()], clockRef: ref })
    expect(screen.queryByLabelText('Joueur suivi')).toBeNull()
    for (const label of ['Coéquipiers', 'Dominance', 'Médias']) {
      expect(screen.queryByText(label)).toBeNull()
    }
    expect(screen.queryByRole('button', { name: 'Capture Streets' })).toBeNull()
    expect(screen.getByLabelText('Temps de match')).toBeTruthy()
    // Le repli emporte les pistes, jamais la lecture du temps.
    expect(ref.current).toBeTruthy()
  })

  /**
   * GARDE-FOU : le repli ne doit pas emporter l'exemption clavier. L'attribut vit sur le
   * curseur, qui survit au repli — mais rien dans le typage ne l'impose, et un repli écrit
   * autrement (en masquant la rangée entière) l'aurait emporté sans qu'un test le voie.
   */
  it('REPLIÉ : le curseur PORTE toujours l’attribut des raccourcis clavier', () => {
    renderTracks({ tracksExpanded: false })
    expect(screen.getByLabelText('Temps de match')).toHaveAttribute(TIMELINE_SHORTCUT_ATTR)
  })

  it('DÉPLIÉ : le chevron propose de replier, et son état se lit dans aria-expanded', () => {
    const { onToggleTracks } = renderTracks()
    const bouton = screen.getByRole('button', { name: 'Replier les pistes' })
    expect(bouton).toHaveAttribute('aria-expanded', 'true')
    fireEvent.click(bouton)
    expect(onToggleTracks).toHaveBeenCalledTimes(1)
  })

  it('REPLIÉ : le même chevron propose de déplier', () => {
    const { onToggleTracks } = renderTracks({ tracksExpanded: false })
    const bouton = screen.getByRole('button', { name: 'Déplier les pistes' })
    expect(bouton).toHaveAttribute('aria-expanded', 'false')
    fireEvent.click(bouton)
    expect(onToggleTracks).toHaveBeenCalledTimes(1)
  })

  /**
   * LE SENS DU CHEVRON (retour utilisateur du 2026-08-28 : « il est dans le mauvais sens »).
   * Les pistes sont AU-DESSUS du bouton : déplié, la flèche doit montrer où elles vont partir
   * (vers le haut) ; replié, d'où elles vont revenir (vers le bas). Le dessin de base pointe
   * vers le bas, c'est donc l'état DÉPLIÉ qui porte la rotation — l'inverse de ce qui était
   * livré, et rien d'autre qu'un test ne tient un sens de flèche.
   */
  it('DÉPLIÉ : le chevron pointe VERS LES PISTES (haut) ; REPLIÉ, vers le bas', () => {
    const { container } = renderTracks()
    expect(container.querySelector('button[aria-expanded="true"] svg')?.getAttribute('class'))
      .toContain('rotate-180')
    const replie = renderTracks({ tracksExpanded: false }).container
    expect(replie.querySelector('button[aria-expanded="false"] svg')?.getAttribute('class'))
      .not.toContain('rotate-180')
  })
})


/**
 * LE MENU DE POINT DE VUE (2026-09-07, lot L3).
 *
 * Ce que ces cas tiennent, et qui ne se voit pas à la relecture :
 *  1. LA VALEUR RENDUE EST CELLE DE LA BASE. Un bot a deux identités — clé film `bot:<nom>`,
 *     xuid `bid(N.0)` en base — et les marques de la frise s'apparient sur la seconde. Rendre la
 *     première changerait bien le point de vue, vers un joueur que rien ne reconnaît, et
 *     laisserait une piste VIDE sans le moindre message.
 *  2. UNE OPTION SANS DONNÉE LE DIT (décision 7 bis). Un joueur sans ligne de tableau de score
 *     n'a ni camp ni kill collecté : son option est inerte, avec sa raison en infobulle.
 *  3. LE CHOIX NE TOUCHE QU'AU POINT DE VUE (décision 1). Ni le curseur, ni la lecture.
 *  4. LE FOCUS PART AVEC LE CHOIX. Sans cela, la barre d'espace qui suit — le geste réflexe pour
 *     mettre en pause — rouvrirait la liste : `useReplayShortcuts` coupe tous les raccourcis
 *     quand l'élément actif est un `SELECT`. Rien dans le typage ne relie les deux.
 */
describe('ReplayTimelineTracks — le menu de point de vue', () => {
  function menu(): HTMLSelectElement {
    return screen.getByLabelText('Joueur suivi') as HTMLSelectElement
  }

  it('s’ouvre sur le joueur REGARDÉ — au montage, celui de la page (décision 8)', () => {
    renderTracks()
    expect(menu().value).toBe('me-1')
    expect(menu().tagName).toBe('SELECT')
  })

  it('range les joueurs par camp, et nomme la section de ceux qui n’en ont pas', () => {
    const { container } = renderTracks()
    const sections = [...container.querySelectorAll('optgroup')].map((g) => g.getAttribute('label'))
    expect(sections).toEqual(['Cobalt', 'Ambre', 'Sans équipe'])
  })

  it('un joueur SANS ligne de tableau de score est listé, INERTE, et dit pourquoi', () => {
    renderTracks()
    const option = screen.getByRole('option', { name: 'Fantome' }) as HTMLOptionElement
    expect(option.disabled).toBe(true)
    expect(option.title).toBe('Aucune donnée de match pour ce joueur')
  })

  it('LE BOT REND SON XUID DE BASE, jamais la clé du film : sinon sa piste resterait vide', () => {
    const { onSelectViewpoint } = renderTracks()
    fireEvent.change(menu(), { target: { value: 'bot-base' } })
    expect(onSelectViewpoint).toHaveBeenCalledWith('bot-base')
  })

  it('choisir NE TOUCHE NI AU CURSEUR NI À LA LECTURE (décision 1)', () => {
    const { onScrub, onRequestPause, onToggleTracks, onSelectViewpoint } = renderTracks()
    const curseur = screen.getByLabelText('Temps de match') as HTMLInputElement
    fireEvent.change(curseur, { target: { value: '240' } })
    onScrub.mockClear()
    fireEvent.change(menu(), { target: { value: 'foe-1' } })
    expect(onSelectViewpoint).toHaveBeenCalledWith('foe-1')
    // Le curseur n'a pas bougé d'une image, et personne n'a demandé de pause ni de repli.
    expect(curseur.value).toBe('240')
    expect(onScrub).not.toHaveBeenCalled()
    expect(onRequestPause).not.toHaveBeenCalled()
    expect(onToggleTracks).not.toHaveBeenCalled()
  })

  it('APRÈS UN CHOIX, le menu n’a plus le focus — sinon Espace rouvrirait la liste', () => {
    renderTracks()
    menu().focus()
    expect(document.activeElement).toBe(menu())
    fireEvent.change(menu(), { target: { value: 'foe-1' } })
    expect(document.activeElement).not.toBe(menu())
  })

  /**
   * L'EXEMPTION CLAVIER DE LA FRISE VISE NOMMÉMENT LE CURSEUR, pas ce menu : tant que la liste
   * est ouverte, ses flèches doivent changer de joueur. Poser l'attribut ici rendrait les
   * flèches au rejeu et casserait la navigation native de la liste.
   */
  it('le menu n’est PAS exempté de la garde anti-frappe', () => {
    renderTracks()
    expect(menu()).not.toHaveAttribute(TIMELINE_SHORTCUT_ATTR)
  })
})

/**
 * LE LOSANGE DES AMIS (décision 4 du plan, 2026-09-07). La forme dit l'identité, la couleur dit
 * le camp : une marque amie tourne de 45°, et son encre — celle qui distingue un frag d'une mort
 * — ne change pas d'un iota. C'est un cas de test parce que rien d'autre ne tient une FORME.
 */
describe('ReplayTimelineTracks — les amis prennent le losange', () => {
  it('une marque AMIE tourne de 45°, une marque ordinaire non', () => {
    const { container } = renderTracks({
      own: [mark({ key: 'k1', clock: '1:12' })],
      teammates: [mark({ key: 'a1', clock: '2:02', friend: true })],
    })
    expect(container.querySelector('[title="1:12"]')?.className).not.toContain('rotate-45')
    expect(container.querySelector('[title="2:02"]')?.className).toContain('rotate-45')
  })

  it('elle garde le CENTRE et l’ENCRE des autres : seule la silhouette change', () => {
    const { container } = renderTracks({
      teammates: [
        mark({ key: 'ami', clock: '2:02', friend: true }),
        mark({ key: 'mort-amie', clock: '3:03', kind: 'death', friend: true }),
      ],
    })
    const ami = container.querySelector('[title="2:02"]')
    const mortAmie = container.querySelector('[title="3:03"]')
    expect(ami?.className).toContain('-translate-x-1/2')
    // L'encre reste celle du TYPE d'événement — un ami tombé garde l'encre des morts.
    expect(ami?.getAttribute('style')).toContain('--ac-team-ally')
    expect(mortAmie?.getAttribute('style')).toContain('--ac-team-enemy')
    expect(mortAmie?.className).toContain('rotate-45')
  })
})
