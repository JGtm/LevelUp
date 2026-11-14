/**
 * useReplayLives — LES TRACES DES VIES, câblées au canvas : les marques bâties une fois par
 * document (`model/livesMarks`), le filtre « qui / quand » des réglages, et les trois peintres
 * (`livesLayer`) que la scène ordonne sous les pions.
 *
 * LES MARQUES NE DÉPENDENT NI DES INTERRUPTEURS NI DES FILTRES : elles se construisent pour que le
 * panneau sache quel calque a de quoi se peindre (`available`) — « pas de commande qui ne commande
 * rien » — et le filtre s'applique au dessin, image par image. Les lectures d'identité (couleur,
 * fiche, camp, joueur suivi) sont celles des pions : une trace porte l'encre de son équipe et se
 * tait quand le pion se tairait.
 *
 * Trois calques, une seule construction : chacun porte son `id` à côté de son `paint`
 * (`bindPainters`), comme ceux des autres hooks.
 */
import { useMemo } from 'react'

import type { FilmAllegiance } from '@/lib/replay/filmAllegiance'

import type { PlayerMarkKind } from '../../../lib/replay/playerMarks'
import { frameToMs } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { useCarrierPosAt } from '../model/carrierPosition'
import { buildLivesByXuid, deathWindowFrames } from '../model/livesPosition'
import type { ReplayFeedEntry } from '../model/killFeedLogic'
import { buildLivesMarks, effectiveLivesWho, lifeSlotAt, type LivesResolvers, type LivesWhen, type LivesWho } from '../model/livesMarks'
import type { CanvasView } from '../model/replayView'
import type { ReplaySettings } from '../settings/useReplaySettings'
import { drawDeathMarks, drawKillerMarks, drawSpawnMarks, type LivesPaint } from './livesLayer'
import type { LivesFlags, NamedLayerPainter } from './replayCompose'
import type { SlotIdentity } from './useSlotIdentity'

export interface LivesOptions {
  doc: ReplayDocumentReady
  view: CanvasView
  /** Le fil ALIGNÉ de la page : une seule horloge pour le fil et la carte. */
  feedEntries: readonly ReplayFeedEntry[]
  /** Les réglages : trois interrupteurs et deux affinages (le reste n'est pas lu). */
  settings: Pick<ReplaySettings, 'showSpawns' | 'showDeaths' | 'showKillers' | 'livesWho' | 'livesWhen'>
  allegiance: FilmAllegiance
  /** Marques d'identité par xuid : « moi » désigne le joueur suivi. */
  marks?: ReadonlyMap<string, PlayerMarkKind>
  /** Les lectures de pion du canvas (`useSlotIdentity`). */
  identity: Pick<SlotIdentity, 'colorOfSlot' | 'colorOfXuid' | 'campOfSlot' | 'markOfSlot' | 'seatedOfXuid'>
  /** Encre du liseré sombre sous les traits de couleur. */
  ink: string
  /** Seul et Masquer (`useReplayPlayerFocus`) : une trace se tait quand le pion de sa vie se tairait. */
  shownAtSlot?: (slot: number, frame: number) => boolean
}

export interface ReplayLivesLayers {
  /** Les trois peintres, à passer à `bindPainters`. */
  layers: readonly [
    NamedLayerPainter<'vies-apparitions'>,
    NamedLayerPainter<'vies-morts'>,
    NamedLayerPainter<'vies-tueurs'>,
  ]
  /** Les interrupteurs des trois calques. */
  show: LivesFlags
  /** Les calques qui ont au moins une marque, et s'il y a un joueur suivi (hors calque). */
  available: LivesFlags & { followedKnown: boolean }
}

/** Les lectures d'identité du canvas, assemblées en résolveurs de marques. */
function resolversOf(
  posOf: LivesResolvers['posOf'],
  identity: LivesOptions['identity'],
  allegiance: FilmAllegiance,
  marks: LivesOptions['marks'],
  slotAt: LivesResolvers['slotAt'],
): LivesResolvers {
  return {
    slotAt,
    posOf,
    seated: identity.seatedOfXuid,
    colorOfXuid: identity.colorOfXuid,
    colorOfSlot: identity.colorOfSlot,
    mineOfXuid: (xuid) => allegiance.ofXuid(xuid) === true,
    mineOfSlot: (slot, frame) => allegiance.ofTeam(identity.campOfSlot(slot, frame)) === true,
    followedOfXuid: (xuid) => marks?.get(xuid) === 'me',
    followedOfSlot: (slot, frame) => identity.markOfSlot(slot, frame) === 'me',
  }
}

export function useReplayLives(o: LivesOptions): ReplayLivesLayers {
  const { doc, view, feedEntries, settings, allegiance, marks, identity, ink, shownAtSlot } = o
  const posOf = useCarrierPosAt(doc)
  const slotAt = useMemo<LivesResolvers['slotAt']>(() => {
    const lives = buildLivesByXuid(doc.tracks)
    const window = deathWindowFrames(doc)
    return (xuid, frame) => lifeSlotAt(lives.get(xuid), frame, window)
  }, [doc])
  // Les lectures d'identité, une à une : l'objet `identity` est neuf à chaque rendu, ses fonctions non.
  const { colorOfSlot, colorOfXuid, campOfSlot, markOfSlot, seatedOfXuid } = identity
  const built = useMemo(
    () =>
      buildLivesMarks(
        doc,
        feedEntries,
        resolversOf(posOf, { colorOfSlot, colorOfXuid, campOfSlot, markOfSlot, seatedOfXuid }, allegiance, marks, slotAt),
      ),
    [doc, feedEntries, posOf, colorOfSlot, colorOfXuid, campOfSlot, markOfSlot, seatedOfXuid, allegiance, marks, slotAt],
  )
  const frameMs = useMemo(() => frameToMs(1, doc), [doc])
  const who: LivesWho = effectiveLivesWho(settings.livesWho, built.followedKnown)
  const when: LivesWhen = settings.livesWhen
  const layers = useMemo(() => {
    const paint = (frame: number, k: number): LivesPaint => ({ who, time: { when, frame, frameMs }, underlay: ink, k, shownAtSlot })
    return [
      { id: 'vies-apparitions', paint: (ctx, f, k) => drawSpawnMarks(ctx, built.spawns, view, paint(f, k)) },
      { id: 'vies-morts', paint: (ctx, f, k) => drawDeathMarks(ctx, built.deaths, view, paint(f, k)) },
      { id: 'vies-tueurs', paint: (ctx, f, k) => drawKillerMarks(ctx, built.killers, view, paint(f, k)) },
    ] as const satisfies ReplayLivesLayers['layers']
  }, [built, view, who, when, frameMs, ink, shownAtSlot])
  const { showSpawns, showDeaths, showKillers } = settings
  return useMemo(
    () => ({
      layers,
      show: { spawns: showSpawns, deaths: showDeaths, killers: showKillers },
      available: {
        spawns: built.spawns.length > 0,
        deaths: built.deaths.length > 0,
        killers: built.killers.length > 0,
        followedKnown: built.followedKnown,
      },
    }),
    [layers, built, showSpawns, showDeaths, showKillers],
  )
}
