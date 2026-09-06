/**
 * useReplayWeaponPads — TOUT LE CÂBLAGE DU CALQUE DES SOCLES, en un seul point.
 *
 * POURQUOI UN HOOK ET NON QUATRE MORCEAUX DANS `ReplayCanvas`. Le canvas du rejeu porte une
 * dette de taille GELÉE par un cliquet (cf. placementFamily.guard.test.ts) : toute addition s'y
 * fait par extraction, jamais par empilement. Ce hook réunit donc les quatre préoccupations du
 * calque — la cuisson des vignettes, le tracé par image, le survol et l'infobulle — et n'en
 * rend au canvas que trois lignes utiles : `paint`, les deux gestionnaires de pointeur, et le
 * survol courant. Même parti que `usePlacementHover` et `useSlotIdentity` avant lui.
 *
 * LES VIGNETTES SONT CUITES UNE FOIS PAR THÈME, pas par image. Ce sont les MÊMES icônes que
 * les fiches joueur — celles du document (`weaponLabels[id].img`, 168 PNG extraits du jeu) —
 * et la plupart sont des MASQUES à teindre : un canvas ne connaît pas le `mask-image` du CSS,
 * la teinte se fait donc hors écran par composition (`tintedIconCanvas`, le même utilitaire que
 * les vignettes de grenade). Une famille sans visuel n'en emprunte aucun : le calque lui dessine
 * un glyphe neutre.
 *
 * UN SOCLE N'EST PAS TOUJOURS UNE ARME (schéma 17, 2026-08-19). Un socle de POWER-UP publie une
 * famille d'ÉQUIPEMENT (`powerup_overshield`) et non l'hexadécimal d'une arme : elle n'est
 * dans AUCUNE table du document, ni pour la taille, ni pour le nom, ni pour la vignette. Les
 * trois résolutions passent donc par des fonctions pures (`padScaleFor`, `padNameFor`,
 * `padIconRefFor`) qui interrogent d'abord la table des familles non-arme — une table écrite,
 * jamais un test de préfixe — avant de retomber sur le catalogue d'armes. Sans elles, le socle
 * du centre de Catalyst restait petit, sans image, et son infobulle affichait sa clé brute.
 *
 * L'IMAGE COURANTE EST LUE DANS UNE RÉFÉRENCE, jamais dans un état React (même règle et même
 * conséquence assumée que `usePlacementHover`) : si l'état d'un socle change SOUS un pointeur
 * immobile, son infobulle attend le prochain mouvement. À l'arrêt — le cas où l'on inspecte —
 * la lecture est exacte.
 */
import { useCallback, useEffect, useMemo, useRef, useState, type PointerEvent, type RefObject } from 'react'

import { staticAssetURL } from '@/lib/staticAssets'
import { useTitleSlug } from '@/lib/title-routing'

import { catalogText } from '../i18n/catalogLabel'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { refinePadPresence } from '../padPresenceRefine'
import type { ReplayText } from '../i18n/i18nContract'
import type { PlacementView } from './placementShapes'
import { tintedIconCanvas } from './replayDraw'
import { weaponFullIcon } from '../weaponFullIcon'
import { frameToMs, type XY } from '../replayLogic'
import type { ReplayDocumentReady, ReplayWeaponPadReady } from '../replayNormalize'
import {
  PAD_EQUIPMENT_FAMILIES,
  padEquipmentFamilyOf,
  padFamilyOf,
  padScaleOf,
  type PadFamily,
  type PadScale,
} from '../weaponPadFamilies'
import {
  padRespawnAt,
  padStateAt,
  type PadRespawn,
  type PadState,
} from '../weaponPadTime'
import { drawWeaponPadsLayer, padAt, type PadIcon } from './weaponPadsLayer'

/** Le catalogue d'ARMES du document, tel qu'il arrive : une table par identifiant, ou rien. */
type PadLabels = ReplayDocumentReady['weaponLabels']

/**
 * padScaleFor — la TAILLE d'un socle, quelle que soit la nature de ce qu'il porte.
 *
 * DEUX VOCABULAIRES, UNE SEULE RÈGLE. Un socle d'ARME publie l'hexadécimal d'une famille : sa
 * clé canonique se lit dans `weaponLabels[id].key` (posée à la requête par le service). Un
 * socle de POWER-UP publie DIRECTEMENT sa famille d'équipement, qui EST déjà la clé — la
 * chercher dans `weaponLabels`, table d'armes, ne rendait rien et laissait le socle petit.
 * `POWER_PAD_KEYS` porte les deux vocabulaires, donc la règle de taille ne se dédouble pas.
 */
export function padScaleFor(weapon: string, labels: PadLabels): PadScale {
  return padScaleOf(padEquipmentFamilyOf(weapon) ?? labels?.[weapon]?.key)
}

/**
 * padNameFor — CE QUE L'INFOBULLE ÉCRIT, et l'ordre est une règle.
 *
 * 1. une famille d'équipement connue rend son libellé bilingue local (`padEquipmentFamily`) ;
 * 2. sinon le libellé bilingue du document, s'il nomme cet identifiant ;
 * 3. sinon l'identifiant lui-même — et c'est VOULU pour une arme hors catalogue du titre
 *    (l'hexadécimal est alors la seule chose vraie qu'on puisse écrire, cf. la famille
 *    `0xD7915565` du registre des reports). Une famille d'équipement, elle, ne peut jamais
 *    tomber dans ce cas : elle est nommée à l'étape 1 ou elle n'est pas dans la table.
 */
export function padNameFor(
  weapon: string,
  labels: PadLabels,
  t: ReplayText,
  locale: ReplayLocale,
): string {
  const family = padEquipmentFamilyOf(weapon)
  if (family) return t.padEquipmentFamily[family]
  return catalogText(labels?.[weapon], locale) ?? weapon
}

/** Une vignette à charger : son URL, son mode (masque à teindre ou image finie), son sens. */
export interface PadIconRef {
  url: string
  tinted: boolean
  /** Vrai = image d'atlas, à retourner pour le sens du kill feed du jeu (cf. weaponFullIcon). */
  mirrored: boolean
}

/**
 * padIconRefFor — QUELLE IMAGE pour ce socle, ou null (le calque pose alors un glyphe neutre).
 *
 * UNE ARME PREND SA VERSION PLEINE, LA MÊME QUE LES FICHES ET LE KILL FEED (retour utilisateur
 * du 2026-08-28 : « pour les armes il faut utiliser les icônes comme pour les fiches de joueurs
 * et le kill feed »). Le document cuit l'atlas `contour` — un trait d'arme à vide, qui se perd
 * sur un fond de carte en niveaux de gris déjà cerné de noir ; la `silhouette` est une forme
 * PLEINE, lisible à 8 px. L'échange se fait côté client (`weaponFullIcon`), pour la même raison
 * que sur les fiches : l'URL est figée dans l'artefact au build. Et comme sur les fiches,
 * l'image d'atlas se rend RETOURNÉE — les deux atlas pointent à gauche, le jeu à droite.
 *
 * LES POWER-UPS SE RÉSOLVENT CÔTÉ CLIENT, comme la version pleine des icônes d'arme des
 * fiches (cf. `weaponFullIcon.ts`) : leur famille n'entre dans AUCUN catalogue du document — le
 * manifeste du titre ne déclare d'icône que sur ses grenades et ses capacités, jamais sur ses
 * `[[equipment_objects]]`. Le masque de HUD existe pourtant, livré sous
 * `static/weapons-assets/{slug}/hud/`, et c'est celui-là même que le manifeste nomme pour la
 * capacité de même nom. Un garde-rail rejoue le lien ; rien n'est deviné à l'exécution.
 */
export function padIconRefFor(weapon: string, labels: PadLabels, titleSlug: string): PadIconRef | null {
  const family = padEquipmentFamilyOf(weapon)
  if (family) {
    const url = staticAssetURL('weapon', `hud/${PAD_EQUIPMENT_FAMILIES[family].icon}`, '.png', titleSlug)
    // Le masque de HUD d'un power-up n'est pas une image d'atlas : il garde son sens.
    return url ? { url, tinted: true, mirrored: false } : null
  }
  const label = labels?.[weapon]
  if (!label?.img) return null
  const full = weaponFullIcon(label.img)
  return { url: full.url, tinted: !!label.tinted, mirrored: full.mirrored }
}

/**
 * crossedWeaponPads — LES SOCLES À DESSINER, une fois le catalogue de la carte croisé.
 *
 * CE QUE LE CROISEMENT APPORTE, et il n'apporte que ça : la POSITION. Le serveur ne sert
 * que les emplacements qu'un socle du match CONFIRME (à moins d'un mètre), et il les sert
 * à la position du SPAWNER telle que le fichier de carte la pose — au centimètre, connue
 * dès la première image. Le film, lui, ne donne que le centroïde des apparitions qu'il a
 * vues, donc rien avant la première.
 *
 * CE QUE LE CROISEMENT NE TOUCHE PAS : la PRÉSENCE. Les apparitions, les intervalles, le
 * cycle, la famille d'arme et donc les états plein / incertain / vide et le compte à
 * rebours restent EXACTEMENT ceux du match. Le catalogue ne sait pas ce qui apparaît sur
 * un socle (un même objet porte l'épée ou le marteau selon le match), ni quand.
 *
 * UN SOCLE DU FILM QUE LE CATALOGUE IGNORE RESTE DESSINÉ, à sa position d'origine : le
 * film fait foi, le catalogue complète. Et un emplacement du catalogue que le film ne
 * confirme pas n'arrive jamais ici — le serveur ne l'envoie pas, et rien de ce qui suit ne
 * pourrait l'inventer, puisque tout part de `weaponPads`.
 *
 * SANS CROISEMENT (carte hors catalogue, artefact d'un autre titre, réponse d'une version
 * antérieure) : la liste du film, inchangée. C'est le repli, et c'est le comportement
 * d'avant le 2026-08-19.
 */
export function crossedWeaponPads(
  pads: readonly ReplayWeaponPadReady[],
  cross: ReplayDocumentReady['mapWeaponPads'],
): readonly ReplayWeaponPadReady[] {
  const spots = cross?.pads ?? []
  if (spots.length === 0) return pads
  // PREMIÈRE CITATION GAGNE : deux emplacements ne peuvent pas déplacer le même socle à
  // deux endroits. Le serveur ne le fait pas, le client ne s'y fie pas.
  const parSocle = new Map<number, (typeof spots)[number]>()
  for (const spot of spots) {
    if (!parSocle.has(spot.pad)) parSocle.set(spot.pad, spot)
  }
  return pads.map((pad, i) => {
    const spot = parSocle.get(i)
    if (!spot) return pad
    return { ...pad, x: spot.x, y: spot.y, z: spot.z ?? pad.z }
  })
}

/** Ce qui est survolé : le socle, son état LU À CET INSTANT, et où poser l'infobulle. */
export interface WeaponPadHover {
  pad: ReplayWeaponPadReady
  at: XY
  /** Nom de l'arme dans la langue du lecteur, ou son identifiant quand rien ne la nomme. */
  name: string
  state: PadState
  /**
   * Le compte à rebours ET SA PROVENANCE, ou null (socle non vide, ou aucune source).
   *
   * L'OBJET PLUTÔT QU'UN NOMBRE depuis D3 (2026-08-27) : l'infobulle doit distinguer la
   * prochaine apparition VUE dans le film de celle que le cycle PRÉDIT. Le nombre seul
   * mélangeait les deux et faisait lire une prédiction comme une mesure.
   */
  respawn: PadRespawn | null
}

export interface WeaponPadsInput {
  doc: ReplayDocumentReady
  view: PlacementView
  /** L'image courante, telle que la boucle de lecture la tient. */
  frameRef: RefObject<number>
  /** Faux quand le calque est éteint : rien n'est dessiné, rien ne se survole. */
  enabled: boolean
  /**
   * Les encres du calque : le NEUTRE du terrain (le point) et le couple REMPLISSAGE /
   * LISERÉ du marquage (la vignette et le compte à rebours). Toutes viennent du canvas.
   */
  ink: {
    neutral: string
    fill: string
    outline: string
    /** Encre par NATURE de socle (A13) : power-up, arme de puissance, râtelier ordinaire. */
    family: Readonly<Record<PadFamily, string>>
  }
  locale: ReplayLocale
  /** Repeindre la scène : les vignettes arrivent après coup (chargement asynchrone). */
  redraw: () => void
}

export interface WeaponPads {
  /** Le film porte-t-il des socles ? Une bascule qui ne commande rien ne s'affiche pas. */
  available: boolean
  /** Trace le calque à l'image demandée ; ne fait rien quand il est éteint. */
  paint: (ctx: CanvasRenderingContext2D, frame: number, k: number) => void
  hover: WeaponPadHover | null
  onPointerMove: (event: PointerEvent<HTMLCanvasElement>) => void
  onPointerLeave: () => void
}

export function useReplayWeaponPads({
  doc,
  view,
  frameRef,
  enabled,
  ink,
  locale,
  redraw,
}: WeaponPadsInput): WeaponPads {
  // LES SOCLES DU MATCH, POSÉS AUX EMPLACEMENTS DE LA CARTE quand le serveur les a croisés
  // (cf. crossedWeaponPads), PUIS leur incertitude ramenée à ce qu'un joueur a pu faire
  // (cf. refinePadPresence, décision utilisateur du 2026-08-27).
  //
  // L'ORDRE DES DEUX EST LE BON, et il n'est pas indifférent : le raffinement mesure des
  // DISTANCES au socle, il doit donc travailler sur la position FINALE. Le croisement ne la
  // déplace que de moins d'un mètre par construction — le serveur ne sert un emplacement de
  // carte que si un socle du match le confirme à moins d'un mètre — mais c'est justement
  // l'échelle du rayon d'approche, et l'inversion se paierait aux limites.
  //
  // UNE SEULE PASSE PAR DOCUMENT, jamais par image : le raffinement parcourt les segments de
  // trace de chaque fenêtre d'incertitude, ce qui est trop cher à refaire soixante fois par
  // seconde. Mémoïsé sur les trois sources, comme le tableau qu'il rend.
  const pads = useMemo(
    () =>
      refinePadPresence(crossedWeaponPads(doc.weaponPads, doc.mapWeaponPads), doc.tracks),
    [doc.weaponPads, doc.mapWeaponPads, doc.tracks],
  )
  const [hover, setHover] = useState<WeaponPadHover | null>(null)
  const iconsRef = useRef<Map<string, PadIcon>>(new Map())
  const titleSlug = useTitleSlug()
  const t = REPLAY_TEXT[locale]

  // LA TAILLE ET LE NOM suivent la CLÉ CANONIQUE de ce que porte le socle — jamais son
  // hexadécimal, jamais un libellé. Les trois résolutions vivent au-dessus, en fonctions
  // pures : elles sont les mêmes pour une arme et pour un power-up de socle, et c'est là
  // qu'est écrit lequel des deux vocabulaires s'applique.
  const labels = doc.weaponLabels
  const scaleOf = useCallback((weapon: string) => padScaleFor(weapon, labels), [labels])
  // L'ENCRE DE LA NATURE (A13) : la famille se résout de la même clé canonique que la taille,
  // et l'appelant a déjà résolu les trois tokens — ce hook ne fait que les apparier.
  const inkOf = useCallback(
    (weapon: string) => ink.family[padFamilyOf(weapon, labels?.[weapon]?.key)],
    [labels, ink.family],
  )
  const nameOf = useCallback(
    (weapon: string) => padNameFor(weapon, labels, t, locale),
    [labels, t, locale],
  )

  // LES VIGNETTES, cuites une fois par document ET par encre : un masque se teint hors écran,
  // une image finie se pose telle quelle (même contrat `tinted` que WeaponIcon).
  //
  // DEUX TEINTURES PAR VIGNETTE depuis le 2026-08-18 : le CORPS à l'encre du texte, le LISERÉ
  // à celle du fond. Un canvas ne sait pas cerner une image ; le calque repose donc la même
  // forme tout autour, et il lui faut les deux.
  //
  // LE LISERÉ N'EST PLUS NOIR (retour utilisateur du 2026-08-28 : « pour les contours des armes
  // et power-up, le noir ça va pas — les dessins de carte sont en niveaux de gris avec contours
  // noirs, c'est difficile de distinguer les armes »). Il prend l'encre de la NATURE du socle,
  // celle-là même que porte son losange : un halo coloré autour d'une silhouette claire se
  // détache d'un fond gris, là où le noir sur noir se confondait. L'encre du fond (`ink.outline`)
  // reste celle du COMPTE À REBOURS, qui est du texte et non une forme.
  //
  // LE LISERÉ EST SERVI POUR TOUTES LES IMAGES depuis le 2026-08-27, images FINIES comprises
  // (retour utilisateur : « icône AVEC contour »). Il était refusé aux non-masques au motif
  // qu'on ne peut pas les reteindre — vrai pour le CORPS, faux pour le liseré : cerner ne
  // demande que la SILHOUETTE, et `tintedIconCanvas` la rend de n'importe quelle image à alpha
  // (composition `source-in`, qui ne garde que l'alpha et le remplit de l'encre). Seul le
  // corps distingue donc encore les deux cas.
  useEffect(() => {
    const map = new Map<string, PadIcon>()
    iconsRef.current = map
    const seen = new Set<string>()
    for (const pad of pads) {
      const ref = padIconRefFor(pad.weapon, labels, titleSlug)
      if (!ref || seen.has(pad.weapon)) continue
      seen.add(pad.weapon)
      const weapon = pad.weapon
      const tinted = ref.tinted
      const mirrored = ref.mirrored
      const outlineInk = inkOf(weapon)
      const im = new Image()
      im.onload = () => {
        map.set(weapon, {
          fill: tinted || mirrored ? tintedIconCanvas(im, ink.fill, { mirrored, tinted }) : im,
          outline: tintedIconCanvas(im, outlineInk, { mirrored }),
        })
        redraw()
      }
      im.src = ref.url
    }
  }, [pads, labels, titleSlug, ink.fill, inkOf, redraw])

  const frameMs = useMemo(() => frameToMs(1, doc), [doc])

  const paint = useCallback(
    (ctx: CanvasRenderingContext2D, frame: number, k: number) => {
      if (!enabled || pads.length === 0) return
      drawWeaponPadsLayer(
        ctx,
        pads,
        view,
        { frame, frameMs, k },
        {
          ink: ink.neutral,
          fill: ink.fill,
          outline: ink.outline,
          iconOf: (weapon) => iconsRef.current.get(weapon) ?? null,
          scaleOf,
          inkOf,
          countdownLabel: t.padCountdownFmt,
        },
      )
    },
    [enabled, pads, view, frameMs, ink.neutral, ink.fill, ink.outline, scaleOf, inkOf, t.padCountdownFmt],
  )

  const onPointerMove = useCallback(
    (event: PointerEvent<HTMLCanvasElement>) => {
      if (!enabled || pads.length === 0 || view.width === 0) {
        setHover((prev) => (prev === null ? prev : null))
        return
      }
      const rect = event.currentTarget.getBoundingClientRect()
      // Le contexte dessine en pixels CSS ; le rapport ne vaut 1 que si la mise en page ne
      // remet pas le canevas à l'échelle — on le calcule plutôt que de le supposer (même
      // amorce que le survol des poses).
      const kx = rect.width > 0 ? view.width / rect.width : 1
      const ky = rect.height > 0 ? view.height / rect.height : 1
      const at = { x: (event.clientX - rect.left) * kx, y: (event.clientY - rect.top) * ky }
      const style = {
        ink: ink.neutral,
        fill: ink.fill,
        outline: ink.outline,
        iconOf: () => null,
        scaleOf,
        inkOf,
        countdownLabel: t.padCountdownFmt,
      }
      const found = padAt(pads, view, style, window.devicePixelRatio || 1, at)
      setHover((prev) => {
        if (!found) return prev === null ? prev : null
        const frame = frameRef.current
        const next: WeaponPadHover = {
          pad: found,
          at,
          name: nameOf(found.weapon),
          state: padStateAt(found, frame),
          respawn: padRespawnAt(found, frame, frameMs),
        }
        // LE COMPTE SE COMPARE CHAMP À CHAMP depuis qu'il est un objet : `padRespawnAt` en
        // construit un neuf à chaque appel, et une comparaison d'identité rendrait TOUJOURS un
        // état différent — l'infobulle se recréerait à chaque mouvement de pointeur.
        if (
          prev &&
          prev.pad === next.pad &&
          prev.state === next.state &&
          prev.respawn?.seconds === next.respawn?.seconds &&
          prev.respawn?.measured === next.respawn?.measured &&
          prev.at.x === at.x &&
          prev.at.y === at.y
        ) {
          return prev
        }
        return next
      })
    },
    [enabled, pads, view, ink.neutral, ink.fill, ink.outline, scaleOf, inkOf, t.padCountdownFmt, frameRef, frameMs, nameOf],
  )

  const onPointerLeave = useCallback(() => {
    setHover((prev) => (prev === null ? prev : null))
  }, [])

  return { available: pads.length > 0, paint, hover, onPointerMove, onPointerLeave }
}
