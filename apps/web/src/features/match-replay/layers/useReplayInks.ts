/**
 * useReplayInks — LES ENCRES DU REJEU, résolues une fois par palette.
 *
 * POURQUOI CE FICHIER EXISTE. Huit valeurs de couleur du canvas partageaient EXACTEMENT le même
 * corps — `void paletteVersion` puis un `resolveToken` ou un `readInk`, mémoïsé sur la version
 * de palette — recopié huit fois dans `ReplayCanvas.tsx`. C'est la 3e copie de la règle
 * CLAUDE.md n°6 largement dépassée, et c'est aussi ce qui a fait grossir le canvas jusqu'à son
 * seuil de taille (`max-lines` eslint, R5) : l'extraction est la façon prescrite d'y
 * ajouter un calque, pas le relèvement du plafond.
 *
 * UN SEUL MÉMO POUR LES HUIT, ET C'EST MIEUX QUE HUIT : elles dépendaient toutes de la même
 * chose, elles changent donc toutes en même temps. Les consommateurs y gagnent des références
 * stables entre deux rendus — ce dont les tableaux de dépendances du canvas ont besoin.
 *
 * DEUX SOURCES DE COULEUR, ET LEUR DIFFÉRENCE EST DU SENS. `resolveToken` sert les couleurs qui
 * DISENT quelque chose (une équipe, un lancer, une mort) : ce sont des tokens sémantiques, et
 * ils suivent la palette d'accessibilité que l'utilisateur a réglée. `readInk` sert la mise en
 * page (arête d'une marche, contour d'une étiquette) : ce sont les variables du système de
 * design, jamais un littéral (cf. canvasInk.ts). Aucune valeur écrite en dur ici.
 */
import { useMemo } from 'react'

import { resolveToken } from '@/lib/accessibility/resolveToken'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'

import { readInk } from './canvasInk'
import { readFxInk, type FxInk } from './fxInk'
import type { RiftInk } from './placementRift'
import type { PadFamily } from '../weaponPadFamilies'

/** Fond de carte : token neutre, sans connotation directionnelle (le sujet = les joueurs). */
const GEOMETRY_TOKEN: SemanticToken = 'divergent-neutral'
/**
 * L'ENCRE DU « AUCUN CAMP », et c'est un TOKEN parce que ça DIT quelque chose.
 *
 * Le rejeu servait jusqu'ici `floor.edge` — c'est-à-dire `--muted-foreground` via `readInk` —
 * partout où il n'y avait pas de camp. C'était l'entorse que `canvasInk.ts` s'interdit
 * lui-même en toutes lettres : « toute couleur qui DIT quelque chose (une équipe, un tir, une
 * mort) passe par un token ». « Personne ne tient cette zone » dit quelque chose — c'est même
 * une MESURE du film (valeur neutre du canal de propriété), pas de la mise en page.
 *
 * `divergent-neutral` EST DÉJÀ le neutre sémantique de la feature : le fil des éliminations
 * l'emploie pour une mort que personne ne revendique. Les objectifs sans camp parlent donc
 * maintenant la même langue que le fil, et suivent la palette d'accessibilité de
 * l'utilisateur — ce que `readInk` ne fait pas.
 *
 * MÊME TOKEN QUE `GEOMETRY_TOKEN`, ET DEUX CONSTANTES QUAND MÊME : le fond de carte est neutre
 * parce qu'il n'est pas le sujet, une zone est neutre parce que personne ne la tient. Deux
 * raisons distinctes, qui doivent pouvoir diverger sans qu'on ait à démêler laquelle on change.
 */
const NEUTRAL_TOKEN: SemanticToken = 'divergent-neutral'
/**
 * Événements ponctuels. Le LANCER emprunte un token d'information ; le TIR, lui, ne prend plus
 * aucun token de données : sa couleur dit la NATURE DE LA DÉCHARGE et vient des teintes
 * diégétiques du thème (fxInk.ts, décision utilisateur du 2026-08-15). Le token d'alerte reste
 * employé par les effets de MORT, qui n'ont pas changé.
 */
const SHOT_TOKEN: SemanticToken = 'destructive'
const GRENADE_TOKEN: SemanticToken = 'info'

/**
 * LE MARQUEUR DU JOUEUR DE LA PAGE (lot R2-V, retour utilisateur du 2026-08-18 : « avoir
 * l'icône du joueur actif qui se démarque de tous les autres »).
 *
 * `success` EST LE VERT DU DÉPÔT, et c'est le vert qui a été demandé — « j'aurais bien aimé du
 * vert, mais pour l'accessibilité je sais pas si ça peut le faire ». La réponse est : oui, à
 * condition que la couleur ne soit JAMAIS SEULE. Elle ne teint ici que le DOUBLE CONTOUR et le
 * halo, deux signes de FORME qui se lisent sans elle ; le noyau garde la couleur d'ÉQUIPE, qui
 * dit le camp. Un lecteur qui ne distingue pas ce vert voit toujours le seul marqueur de la
 * carte qui porte deux anneaux.
 */
const SELF_TOKEN: SemanticToken = 'success'

/**
 * LE MUR DE PROTECTION (W1/R2-5, verdict du 2026-08-18 : « je préférerais un orange doré »).
 *
 * DEUX TOKENS ÉTAIENT CANDIDATS et la planche les a montrés côte à côte : `legendary` (l'or,
 * déjà porté par l'encadré du surbouclier) et `warning` (l'orange d'alerte). `warning` est
 * retenu. Ce qu'on PERD est écrit : la couleur d'ÉQUIPE du poseur — le mur ne dit plus QUI
 * l'a posé. Ce qu'on gagne : il dit QUOI, et un objet du terrain qui garde sa teinte quel que
 * soit le camp se reconnaît d'un coup d'œil au milieu de huit trajectoires colorées.
 */
const WALL_TOKEN: SemanticToken = 'warning'

/**
 * LA FAILLE DU TRANSLOCATEUR (2026-08-27 : « fais en sorte que ça ressemble plus à un portail
 * interdimensionnel au niveau des couleurs »).
 *
 * MÊME RAISONNEMENT QUE LE MUR ci-dessus, poussé d'un cran : la faille n'est pas un objet du
 * jeu qui appartiendrait à un camp, c'est une DÉCHIRURE. Lui donner la teinte d'équipe du
 * poseur la ferait lire comme un marqueur tactique de plus. Le camp reste lisible à
 * l'infobulle ; ce qu'on gagne, c'est qu'elle ne ressemble à RIEN d'autre sur la carte.
 *
 * DEUX TOKENS, parce qu'un portail a un bord et une lumière qui en sort :
 *  - `extreme` (fuchsia) pour les LÈVRES — c'est le token le plus saturé de la palette, et le
 *    seul dont aucune donnée du rejeu ne se sert : il n'entre en conflit avec aucune lecture ;
 *  - `bonus` (violet, plus CLAIR) pour le CŒUR et le halo — plus clair que les lèvres, ce qui
 *    fait lire le centre comme une ouverture éclairée plutôt qu'un trait plein.
 *
 * Les deux sont voisins en teinte (271° et 292°), donc l'objet reste UNE couleur vue de loin,
 * et se sépare en bord + lumière quand on s'approche.
 */
const RIFT_RIM_TOKEN: SemanticToken = 'extreme'
const RIFT_CORE_TOKEN: SemanticToken = 'bonus'


/**
 * LES ENCRES DES TROIS NATURES DE SOCLE (retour utilisateur du 2026-08-26 : « une couleur pour
 * chaque type, en respectant les couleurs accessibles »).
 *
 * TROIS TOKENS DÉJÀ EN SERVICE DANS LE REJEU, aucun nouveau — c'est la condition pour que la
 * palette d'accessibilité de l'utilisateur (et ses variantes daltonisme) continue de valoir
 * sans qu'on ait à la ré-étalonner :
 *   - `legendary` pour un POWER-UP : c'est déjà l'or de l'encadré de surbouclier sur les fiches
 *     (cf. ReplayTeams). Un bonus au sol et l'état qu'il donne se disent de la même couleur ;
 *   - `warning` pour une ARME DE PUISSANCE : c'est déjà le token du mur de protection, retenu
 *     le 2026-08-18 comme la teinte des objets de terrain à fort enjeu ;
 *   - `divergent-neutral` pour un RÂTELIER ordinaire : le neutre sémantique du rejeu, celui qui
 *     recule — un socle classique ne doit pas concurrencer les deux autres.
 *
 * LA HIÉRARCHIE EST LE POINT, pas la teinte exacte : deux natures à fort enjeu qui se
 * distinguent l'une de l'autre, et une troisième qui s'efface. L'utilisateur tranche à l'écran.
 */
const PAD_FAMILY_TOKENS: Readonly<Record<PadFamily, SemanticToken>> = {
  powerup: 'legendary',
  power: 'warning',
  classic: 'divergent-neutral',
}

/** Les encres de mise en page du sol reconstruit : son aplat et l'arête de ses marches. */
export interface ReplayFloorInk {
  fill: string
  edge: string
}

export interface ReplayInks {
  /** Les DEUX couleurs d'équipe telles que l'utilisateur les a réglées (décision D1). */
  teamColorOf: (isAlly: boolean) => string
  /** Props Forge du fond de carte, quand aucun sol reconstruit n'est disponible. */
  geometry: string
  /** Effets de MORT (le tir, lui, prend sa teinte de la décharge : cf. `fx`). */
  shot: string
  /** Lancers de grenade. */
  grenade: string
  /** Sol reconstruit ; `edge` sert aussi d'encre de mise en page à ce qui n'a pas de rôle. */
  floor: ReplayFloorInk
  /** Encre SÉMANTIQUE du « aucun camp » : objectifs neutres, zone que personne ne tient. */
  neutral: string
  /** Encre par NATURE de socle (cf. PAD_FAMILY_TOKENS) : power-up, arme de puissance, râtelier. */
  pad: Readonly<Record<PadFamily, string>>
  /** Teintes des éclairs de bouche, lues une fois par thème (jamais par image). */
  fx: FxInk
  /** Ligne de grappin : l'encre la plus claire du thème (`--foreground`), jamais un hex. */
  grapple: string
  /** Contour des étiquettes : SOMBRE dans les deux thèmes (cf. globals.css). */
  labelStroke: string
  /** Double contour et halo du joueur de la page (cf. SELF_TOKEN). */
  self: string
  /** Arc du mur de protection : un token FIXE, plus la couleur d'équipe (cf. WALL_TOKEN). */
  wall: string
  /** Les deux encres FIXES de la faille du translocateur (cf. RIFT_RIM_TOKEN). */
  rift: RiftInk
  /**
   * LE MARQUAGE DES SOCLES : ce qui est REMPLI, et ce qui le CERNE. Ce sont les deux encres du
   * THÈME, pas des tokens de données : en sombre le remplissage est clair et le liseré sombre,
   * en clair l'inverse — ce qu'un blanc et un noir écrits en dur n'auraient pas fait.
   *
   * `outline` NE CERNE PLUS LES VIGNETTES depuis le 2026-08-28, seulement le COMPTE À REBOURS :
   * un liseré sombre autour d'une arme se confondait avec les contours noirs des fonds de carte
   * en niveaux de gris (retour utilisateur). Les vignettes sont cernées de l'encre de leur
   * NATURE (`pad`), la même que leur losange.
   */
  mark: { fill: string; outline: string }
}

/**
 * useReplayInks résout toutes les encres du rejeu pour la palette courante.
 *
 * `paletteVersion` vient de `useColorPaletteVersion()` chez l'appelant — il l'observe déjà pour
 * ses propres couleurs de série. Le passer plutôt que de le relire ici garde UN observateur de
 * style pour la page, et rend la dépendance visible à la lecture.
 */
export function useReplayInks(paletteVersion: number): ReplayInks {
  return useMemo(() => {
    void paletteVersion
    const ally = resolveToken('team-ally')
    const enemy = resolveToken('team-enemy')
    return {
      teamColorOf: (isAlly: boolean) => (isAlly ? ally : enemy),
      geometry: resolveToken(GEOMETRY_TOKEN),
      shot: resolveToken(SHOT_TOKEN),
      grenade: resolveToken(GRENADE_TOKEN),
      floor: { fill: resolveToken(GEOMETRY_TOKEN), edge: readInk('--muted-foreground') },
      neutral: resolveToken(NEUTRAL_TOKEN),
      pad: {
        powerup: resolveToken(PAD_FAMILY_TOKENS.powerup),
        power: resolveToken(PAD_FAMILY_TOKENS.power),
        classic: resolveToken(PAD_FAMILY_TOKENS.classic),
      },
      fx: readFxInk(),
      grapple: readInk('--foreground'),
      labelStroke: readInk('--replay-label-stroke'),
      self: resolveToken(SELF_TOKEN),
      wall: resolveToken(WALL_TOKEN),
      rift: { rim: resolveToken(RIFT_RIM_TOKEN), core: resolveToken(RIFT_CORE_TOKEN) },
      mark: { fill: readInk('--foreground'), outline: readInk('--background') },
    }
  }, [paletteVersion])
}
