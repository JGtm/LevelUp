/**
 * playerCardFx.ts — TOUS LES EFFETS VISUELS D'UNE FICHE JOUEUR, composés en un seul endroit.
 *
 * EXTRAIT DE `ReplayTeams.tsx` LE 2026-08-27, quand les effets de ZONE (champ de réparation,
 * écran occultant, capteur adverse) et l'éclat de TRANSLOCATION ont rejoint les quatre
 * existants (mort, deux éclats d'événement, verre du camouflage, encadré du surbouclier) :
 * la composition dépassait le gabarit d'une fonction de composant. Ici elle est PURE — des
 * chaînes CSS et des noms de classe, pas de React — donc testée directement
 * (playerCardFx.test.ts), et la fiche ne garde que le rendu.
 *
 * LES EFFETS VIVENT SUR UNE COUCHE SOUS LE CONTENU, POSÉE SUR LA BORDURE DE LA TUILE
 * (option 2a du handoff 2026-08-27) : depuis que chaque fiche est une tuile autonome
 * (bordure, coins arrondis au rayon de l'app), la couche absolue est `inset-0 rounded-lg` —
 * cadres et voiles épousent la tuile au lieu de dessiner un second anneau en retrait.
 * C'est elle qui reçoit `underStyle` et la classe d'éclat ; l'INCRUSTATION au-dessus
 * (nuage et éclairs de l'écran, croix du champ, anneau du capteur, fourreau de
 * translocation) vit dans ReplayTeams.ZoneFxOverlay, à la même géométrie.
 *
 * LA GRAMMAIRE DES EFFETS, telle qu'elle s'est fixée lot après lot :
 *  - un ÉTAT continu se porte par le FOND (mort, verre, voile sombre) ou par un CADRE
 *    (surbouclier, champ de réparation) — il dure ce que dure la mesure ;
 *  - un ÉVÉNEMENT se porte par un ÉCLAT bref à délai négatif (coup fatal, réapparition,
 *    translocation) : l'animation reprend à son avancement réel, donc reste juste après un
 *    saut dans le temps de lecture (cf. globals.css) ;
 *  - les ombres s'ACCUMULENT (box-shadow accepte plusieurs couches), jamais ne s'écrasent ;
 *    le fond, lui, se COMPOSE (le voile de l'écran assombrit le verre du camouflage) ;
 *  - aucun littéral de couleur : tokens sémantiques et variables du thème uniquement.
 */
import type { CSSProperties } from 'react'

import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'

import type { ActiveEquipment } from './equipmentFx'
import type { ZonePresence } from './equipmentZones'
import type { ReplayText } from './i18nContract'

/**
 * Durées CSS des animations d'éclat (cf. globals.css) — le délai négatif s'y rapporte.
 * CES NOMBRES SONT LE MIROIR DE LA FEUILLE DE STYLE : les changer ici sans les changer
 * là-bas désaligne le délai négatif, et l'éclat reprend au mauvais endroit.
 *
 * L'ÉCLAT DE RÉAPPARITION EST PASSÉ DE 0,55 s À 1,2 s le 2026-08-17 (« plus lent l'éclat »,
 * planche du 16/08) : à 0,55 s le vert avait disparu avant qu'on ait fini de lire le nom de
 * la fiche. Le FOURREAU de translocation reprend cette durée : même genre d'événement, même
 * temps de lecture.
 */
const DEATH_FLASH_TOTAL_S = 1.86
const RESPAWN_FLASH_S = 1.2
const TRANSLOCATION_FLASH_S = 1.2

/**
 * L'ENCRE SOMBRE-DES-DEUX-THÈMES : `--replay-label-stroke`, définie pour le contour des noms
 * sur la carte (globals.css) — un sombre calibré pour rester lisible sur les deux fonds, là
 * où un « noir » écrit en dur hurlerait en thème clair. La MORT et l'ÉCRAN OCCULTANT s'en
 * servent tous deux pour « éteindre » la fiche.
 */
const DARK_INK = 'var(--replay-label-stroke)'

/**
 * LE CHROME DE LA TUILE — le fond et la bordure que la fiche porte ELLE-MÊME, hors de la
 * couche d'effets (option 2a du handoff 2026-08-27 : « chaque fiche est une tuile
 * autonome »). Les pourcentages APPROCHENT les valeurs de la maquette par les tokens du
 * thème (jamais un littéral, règle color-tokens) :
 *  - vivant : un dégradé très court AUTOUR de `card` — un souffle de `foreground` en haut
 *    (~oklch 0.225), une pointe d'encre sombre en bas (~oklch 0.20) ;
 *  - mort : `card` teinté `destructive` sur une base légèrement éteinte (~oklch 0.19 0.012 25),
 *    bordure au même rouge affaibli. La mort a QUITTÉ la couche d'effets (l'ancien
 *    voile + lavis du 2026-08-27) : c'est la tuile qui la dit, plus l'encadré « Éliminé »
 *    (ReplayVitality) — la couche ne garde que l'éclat du coup fatal.
 */
const TILE_TOP_LIFT_PCT = 3
const TILE_BOTTOM_DIP_PCT = 8
const DEAD_TINT_PCT = 6
const DEAD_DIM_PCT = 10
const DEAD_BORDER_PCT = 28

/**
 * Effet de VERRE TREMPÉ de la fiche pendant un épisode de CAMOUFLAGE actif (cahier des
 * charges Notion, item 21.1 ; densifié « tempered glass » sur planches de référence le
 * 2026-08-27) : translucidité + flou léger + REFLETS DIAGONAUX, PAS une opacité réduite sur
 * toute la fiche — le texte et les icônes restent lisibles, seul le FOND se dépolit.
 *
 * `BLUR_PX` est le flou du verre (`backdrop-filter`, sans effet visible tant que rien de
 * texturé ne passe dessous — la colonne des fiches n'est jamais posée sur la carte — mais la
 * technique canonique d'un « verre » CSS reste correcte et bon marché si ce contexte change).
 * `VEIL_PCT` mélange `--foreground` (PAS `--card`) au fond ambiant : teinter le fond avec SA
 * PROPRE couleur serait invisible (0 contraste), `--foreground` est justement le token conçu
 * pour contraster sur `--card` dans les DEUX thèmes — le voile s'éclaircit en sombre,
 * s'assombrit en clair, toujours visible. `SHEEN_PCT` porte les DEUX BANDES DE REFLET
 * diagonales (115°, la pente des planches de référence) dans la même encre : c'est ce qui
 * fait lire « plaque de verre » plutôt que « fond grisé ». Le liseré reprend `--border` à
 * pleine force, et la TRANCHE HAUTE reçoit un filet plus clair — la lumière sur le chant du
 * verre, le détail que les deux images de référence ont en commun. Aucune animation : une
 * plaque de verre ne bat pas, et rien dans le film ne bat au rythme du camouflage.
 */
const CAMO_GLASS_BLUR_PX = 6
const CAMO_GLASS_VEIL_PCT = 12
const CAMO_GLASS_SHEEN_PCT = 9
const CAMO_GLASS_EDGE_PCT = 28

/**
 * ENCADRÉ DORÉ de la fiche pendant un épisode de SURBOUCLIER (cahier des charges Notion,
 * item 21.1) : un CADRE, pas un fond teinté — `FRAME_PX` (2 px) est le plus fin qui se
 * lise comme un encadrement plutôt qu'un simple contour de sélection ; `GLOW_PCT` un halo
 * externe discret. Token `legendary` (skill color-tokens) : un surbouclier est un état de
 * jeu rare et précieux, la même famille sémantique que le « légendaire » du Battlepass —
 * PAS le token `info` de la jauge de bouclier (qui, lui, reste bleu : cf. VitalityBar).
 * Aucun fond propre : composé avec le verre du camouflage, le cadre doré ne doit pas
 * écraser la translucidité de l'autre effet.
 */
const OVERSHIELD_FRAME_PX = 2
const OVERSHIELD_GLOW_PCT = 60

/**
 * CONTOUR VERT du CHAMP DE RÉPARATION (demande utilisateur du 2026-08-27 : « un contour vert
 * avec des mini croix de pharmacie qui flottent sur la fiche ») : même grammaire de CADRE que
 * le surbouclier — un état protecteur s'encadre — au token `success`, celui de la jauge de
 * SANTÉ (cf. VitalityBar) : l'objet soigne, la couleur dit la même chose des deux côtés de la
 * fiche. Les croix, elles, vivent dans l'incrustation de zone (ReplayTeams.ZoneFxOverlay).
 */
const REPAIR_FRAME_PX = 2
const REPAIR_GLOW_PCT = 45

/**
 * L'ÉCRAN OCCULTANT sur la fiche (demande du 2026-08-27, itérée deux fois le même jour) :
 * un NUAGE NOIR passe AU-DESSUS du contenu, légèrement — c'est l'incrustation qui le porte
 * (ReplayTeams.ZoneFxOverlay, classe `replay-zone-cloud` ; les rayures du premier essai se
 * confondaient avec les reflets du verre) — et la couche d'effets ne garde que le versant
 * discret : un voile sombre LÉGER (le nuage et lui s'additionnent), un flou plus doux que
 * celui du verre (l'écran cache, il ne dépolit pas), et le contour sombre. Les deux effets
 * se COMPOSENT : nuage et voile sombres sur verre.
 */
const SHROUD_CARD_BLUR_PX = 3
const SHROUD_CARD_VEIL_PCT = 14
const SHROUD_FRAME_PX = 2

/** Ce que la composition a besoin de savoir d'une fiche à cette image. */
export interface CardFxInput {
  alive: boolean
  /** Images écoulées depuis la mort (-1 en vie) — cf. PlayerState.sinceDeath. */
  deathAge: number
  /** Images écoulées depuis le départ de la vie courante (-1 : vie initiale ou mort). */
  lifeAge: number
  /** Images écoulées depuis le dernier passage par faille de cette vie (-1 : aucun). */
  teleportAge: number
  /** Fenêtre des éclats, en frames (FLASH_MS convertie par l'appelant). */
  flashFrames: number
  /** État actif d'équipement de la vie courante (null : mort, ou pas de vie). */
  equipment: ActiveEquipment | null
  /** Zones d'équipement sous le joueur (NO_ZONES pour une fiche morte). */
  zones: ZonePresence
  /** Table i18n de la locale courante : l'infobulle se compose ici. */
  text: ReplayText
}

/** Ce que la fiche rend tel quel, réparti sur ses deux couches d'effets. */
export interface CardFx {
  /** Classe d'éclat de la COUCHE D'EFFETS sous le contenu (mort, réapparition) ; '' sinon. */
  flashClass: string
  /** Style de la couche d'effets sous le contenu : fonds, voiles, flou, cadres. */
  underStyle: CSSProperties
  /**
   * Délai négatif du FOURREAU de translocation (la lumière qui court sur la bordure,
   * incrustation) — null hors de la fenêtre de l'éclat.
   */
  translocationDelay: string | null
  title: string | undefined
}

/** La couche d'effets a-t-elle quelque chose à peindre ? (la fiche ne la rend que si oui). */
export function hasUnderLayer(fx: CardFx): boolean {
  return fx.flashClass !== '' || Object.keys(fx.underStyle).length > 0
}

/**
 * cardChrome — le fond et la bordure de la TUILE selon l'état vital (cf. TILE_* / DEAD_*).
 * Rendu par la fiche sur son conteneur (`borderColor` + `background`), pas par la couche
 * d'effets : le chrome est un état PERMANENT de la tuile, il ne s'empile ni ne s'anime.
 */
export function cardChrome(alive: boolean): CSSProperties {
  if (alive) {
    return {
      borderColor: 'var(--border)',
      background:
        `linear-gradient(180deg, color-mix(in srgb, var(--foreground) ${TILE_TOP_LIFT_PCT}%, var(--card)), ` +
        `color-mix(in srgb, ${DARK_INK} ${TILE_BOTTOM_DIP_PCT}%, var(--card)))`,
    }
  }
  const rouge = tokenCssVar('destructive')
  return {
    borderColor: `color-mix(in srgb, ${rouge} ${DEAD_BORDER_PCT}%, transparent)`,
    background:
      `color-mix(in srgb, ${rouge} ${DEAD_TINT_PCT}%, ` +
      `color-mix(in srgb, ${DARK_INK} ${DEAD_DIM_PCT}%, var(--card)))`,
  }
}

/** Le délai négatif d'un éclat : l'animation reprend à son avancement réel. */
function negativeDelay(age: number, flashFrames: number, totalS: number): string {
  return `${(-(age / flashFrames) * totalS).toFixed(3)}s`
}

/**
 * L'ÉCLAT de la couche d'effets, s'il y en a un — et le style reçoit son délai négatif.
 * Mort et réapparition s'excluent par l'état vital ; la translocation, elle, vit sur
 * l'INCRUSTATION (fourreau de bordure), donc ne concourt pas ici.
 */
function flashOf(i: CardFxInput, style: CSSProperties): string {
  if (!i.alive) {
    if (i.deathAge < 0 || i.deathAge > i.flashFrames) return ''
    style.animationDelay = negativeDelay(i.deathAge, i.flashFrames, DEATH_FLASH_TOTAL_S)
    return 'replay-flash-death'
  }
  if (i.lifeAge < 0 || i.lifeAge > i.flashFrames) return ''
  style.animationDelay = negativeDelay(i.lifeAge, i.flashFrames, RESPAWN_FLASH_S)
  return 'replay-flash-respawn'
}

/**
 * LE FOND ET LE FLOU d'une fiche vivante : verre du camouflage, voile de l'écran, ou les
 * deux composés — le voile sombre teinte alors le verre, comme l'écran du jeu assombrit ce
 * qu'on voit au travers. Les reflets du verre partent en `backgroundImage` (une COUCHE, pas
 * une couleur) ; le voile en `backgroundColor` : deux longhands, pour que chacun se lise et
 * se teste seul.
 */
function glassAndVeilOf(i: CardFxInput, style: CSSProperties, shadows: string[]): void {
  const camo = i.equipment?.camo === true
  const shroud = i.zones.shroudSinceMs !== null
  if (!camo && !shroud) return
  const blur = Math.max(camo ? CAMO_GLASS_BLUR_PX : 0, shroud ? SHROUD_CARD_BLUR_PX : 0)
  style.backdropFilter = `blur(${blur}px)`
  style.WebkitBackdropFilter = style.backdropFilter
  let veil = 'var(--card)'
  if (camo) veil = `color-mix(in srgb, var(--foreground) ${CAMO_GLASS_VEIL_PCT}%, ${veil})`
  if (shroud) veil = `color-mix(in srgb, ${DARK_INK} ${SHROUD_CARD_VEIL_PCT}%, ${veil})`
  style.backgroundColor = veil
  if (camo) {
    const sheen = `color-mix(in srgb, var(--foreground) ${CAMO_GLASS_SHEEN_PCT}%, transparent)`
    style.backgroundImage =
      `linear-gradient(115deg, transparent 30%, ${sheen} 30%, ${sheen} 46%, ` +
      `transparent 46%, transparent 58%, ${sheen} 58%, ${sheen} 64%, transparent 64%)`
    shadows.push('inset 0 0 0 1px var(--border)')
    shadows.push(
      `inset 0 1px 0 color-mix(in srgb, var(--foreground) ${CAMO_GLASS_EDGE_PCT}%, transparent)`,
    )
  }
  if (shroud) shadows.push(`inset 0 0 0 ${SHROUD_FRAME_PX}px ${DARK_INK}`)
}

/** L'infobulle : chaque état actif dit sa phrase, dans un ordre stable. */
function titleOf(i: CardFxInput): string | undefined {
  const parts = [
    i.equipment?.camo ? i.text.equipmentActive.camo : null,
    i.equipment?.overshield ? i.text.equipmentActive.overshield : null,
    i.teleportAge >= 0 && i.teleportAge <= i.flashFrames ? i.text.translocationFlash : null,
    i.zones.repair ? i.text.zonePresence.field : null,
    i.zones.shroudSinceMs !== null ? i.text.zonePresence.shroud : null,
    i.zones.sensorSincePingMs !== null ? i.text.zonePresence.sensor : null,
  ].filter(Boolean)
  return parts.length > 0 ? parts.join(' · ') : undefined
}

/**
 * playerCardFx — la composition complète des effets d'une fiche à une image.
 *
 * UNE FICHE MORTE NE PORTE AUCUN EFFET, hors l'éclat du coup fatal dans sa fenêtre : la
 * mort continue est dite par le chrome de la tuile (`cardChrome`) et l'encadré « Éliminé »,
 * pas par cette couche. Les effets d'équipement et de zone, eux, n'existent que sur une
 * fiche vivante (les épisodes se ferment à la mort au plus tard, et les zones sont
 * calculées sur la vie courante).
 */
export function playerCardFx(i: CardFxInput): CardFx {
  const underStyle: CSSProperties = {}
  const flashClass = flashOf(i, underStyle)
  if (!i.alive) {
    return { flashClass, underStyle, translocationDelay: null, title: undefined }
  }
  const shadows: string[] = []
  glassAndVeilOf(i, underStyle, shadows)
  if (i.equipment?.overshield) {
    shadows.push(`inset 0 0 0 ${OVERSHIELD_FRAME_PX}px ${tokenCssVar('legendary')}`)
    shadows.push(
      `0 0 10px color-mix(in srgb, ${tokenCssVar('legendary')} ${OVERSHIELD_GLOW_PCT}%, transparent)`,
    )
  }
  if (i.zones.repair) {
    shadows.push(`inset 0 0 0 ${REPAIR_FRAME_PX}px ${tokenCssVar('success')}`)
    shadows.push(
      `0 0 8px color-mix(in srgb, ${tokenCssVar('success')} ${REPAIR_GLOW_PCT}%, transparent)`,
    )
  }
  if (shadows.length > 0) underStyle.boxShadow = shadows.join(', ')
  const translocationDelay =
    i.teleportAge >= 0 && i.teleportAge <= i.flashFrames
      ? negativeDelay(i.teleportAge, i.flashFrames, TRANSLOCATION_FLASH_S)
      : null
  return { flashClass, underStyle, translocationDelay, title: titleOf(i) }
}
