/**
 * ReplayTeams — LA COLONNE DES FICHES, à CÔTÉ de la carte et jamais dessus.
 *
 * CE QUE CETTE COLONNE APPORTE, et que la carte ne peut pas dire : qui est qui. La carte
 * montre des traces ; les fiches montrent des gens, avec leur état à l'instant lu — vivant
 * ou mort, bouclier, armes portées, temps avant le retour. Ce fichier ne garde que ce qui est
 * PROPRE À LA COLONNE : les camps et leurs sièges, la scène des effets construite une fois
 * par document, et la grille dans laquelle les tuiles se rangent. La tuile elle-même vit dans
 * `ReplayPlayerCard.tsx`, ses lectures dans `model/playerCardReadings.ts` (extraction du
 * 2026-09-06, plan fiches compactes).
 *
 * DEUX GABARITS, UN SEUL JEU DE COMPOSANTS (2026-09-06, décision D1 de l'utilisateur : « les
 * matchs de type 4v4 on touche pas »). La densité se lit sur le TYPE DE MATCH — la catégorie
 * de mode de l'en-tête (`header.mode_category === 'BTB'`, `model/cardDensity.ts`) — et sur
 * RIEN d'autre : ni le nombre de sièges, ni de joueurs, ni de lignes du tableau. Une Grande
 * équipe range ses sièges en deux colonnes de tuiles de 115 × 62 ; tout autre match garde la
 * colonne de tuiles de 235 px, nœud DOM pour nœud DOM (fixation
 * `__fixtures__/replayTeams.4v4.html`). Il n'y a NI réglage NI drapeau : les deux gabarits sont
 * atteints par des matchs réels. Historique : la variante longue (zone du joueur, deux rangées
 * d'inventaire) a été supprimée avec son réglage le 2026-08-24 (« fiches compactes va devenir
 * la seule et unique option ») ; ce qui revient ici n'est pas cette option mais un second jeu
 * de COTES (`model/cardGabarit.ts`), et les composants ne reçoivent que des nombres et des
 * booléens — jamais le mot « compacte ».
 *
 * TROIS RÈGLES QUI NE SE NÉGOCIENT PAS ICI :
 *   1. Une valeur non lue s'affiche comme une lacune, jamais comme un zéro ni une moyenne.
 *   2. Une lecture ancienne PÂLIT et dit son âge — l'inventaire ne se lit qu'aux images-clés,
 *      une toutes les ~20 s, et le faire passer pour l'instant courant était un défaut réel.
 *   3. Aucun littéral de couleur : les rôles passent par des tokens sémantiques.
 */
import { useMemo } from 'react'

import { scoreTimelineOf } from '@/lib/replay/scoreTimeline'
import type { XuidMeta } from '@/features/match-view/xuidMeta'
import type { MatchScoreboardRow } from '@/lib/api/types'

import { ReplayPlayerCard } from './ReplayPlayerCard'
import { ReplayTeamHeader } from './ReplayTeamHeader'
import { cardDensity } from '../model/cardDensity'
import { teleportMoments } from '../model/placementTeleport'
import type { CardFxScene } from '../model/playerCardReadings'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { frameToMs, msToFrames } from '../../../lib/replay/replayLogic'
import type { PresenceHeader } from '../model/presenceFeed'
import { buildSeats, groupSeatsByTeam, seatOccupantAt } from '../model/seatLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import {
  buildPlayers,
  buildSlotOwnership,
  sideResolver,
  vitalityPresence,
} from '../../../lib/replay/rosterLogic'

/**
 * Estompage COMPLET d'une lecture de vitalité à 6 s : la même graduation que le bouclier
 * sur la carte. Le REPORT, lui, n'a pas de limite dans une vie — le flux est différentiel,
 * non retransmis veut dire inchangé — et les points appartiennent à la vie, donc il ne
 * franchit jamais une mort. Ce qui vieillit pâlit ; ce qui n'a jamais été mesuré reste
 * une lacune dite.
 */
const VITALITY_FADE_MS = 6_000
/**
 * Au-delà, une lecture d'inventaire est au plancher d'opacité. 20 s est l'écart médian entre
 * deux images-clés du film : c'est donc l'âge maximal ordinaire d'une lecture.
 */
const READING_FULL_MS = 20_000
/**
 * Durée des éclats d'événement (coup fatal, réapparition, translocation), en temps réel —
 * assez pour être vus sans être subis, calée sur la rémanence des lancers. L'état de mort,
 * lui, est porté en continu par le fond de la fiche. LA COMPOSITION DES EFFETS (éclats,
 * verre du camouflage, encadrés, voile de l'écran occultant) vit dans `playerCardFx.ts`
 * depuis le 2026-08-27 : la colonne ne fait plus que donner les âges.
 */
const FLASH_MS = 1_400

/**
 * Les sièges d'un camp : une colonne simple (gabarit normal — la chaîne d'aujourd'hui, à
 * l'octet), ou une grille à remplissage automatique de tuiles de 115 px minimum (gabarit
 * compact). ÉCRITES EN CLASSES SANS ESPACE : une valeur arbitraire Tailwind qui contient un
 * espace ou un `calc(` ne produit aucune règle, en silence (`rosterHeight.guard.test.ts`).
 */
const SEATS_COLUMN_CLASS = 'flex min-h-0 flex-1 flex-col gap-1 overflow-y-auto'
const SEATS_GRID_CLASS =
  'grid min-h-0 flex-1 auto-rows-max grid-cols-[repeat(auto-fill,minmax(115px,1fr))] gap-1 overflow-y-auto'

interface ReplayTeamsProps {
  doc: ReplayDocumentReady
  scoreboard: MatchScoreboardRow[]
  frame: number
  locale: ReplayLocale
  /** Camp de chaque xuid : il donne sa couleur au titre de la colonne (allié / adverse). */
  xuidMeta?: XuidMeta
  /**
   * En-tête du match : l'heure de début (le repère des relais de siège, cf. seatLogic) et la
   * catégorie de mode (le gabarit des fiches, cf. cardDensity). Absent = pas de relais,
   * gabarit normal.
   */
  header?: PresenceHeader | null
}

export function ReplayTeams({
  doc, scoreboard, frame, locale, xuidMeta, header,
}: ReplayTeamsProps) {
  const t = REPLAY_TEXT[locale]
  const players = useMemo(() => buildPlayers(doc, scoreboard), [doc, scoreboard])
  // LA FICHE EST UN SIÈGE, PAS UN JOUEUR (retour user 2026-09-02) : un remplacé cède sa
  // fiche à son remplaçant à l'image du relais — un 4v4 garde huit fiches, quel que soit le
  // nombre de relais. L'appariement vient de la participation API (cf. seatLogic.ts) ; sans
  // elle, chaque joueur garde son siège — l'affichage d'avant.
  const seats = useMemo(() => buildSeats(players, header, doc), [players, header, doc])
  const groups = useMemo(() => groupSeatsByTeam(seats), [seats])
  // LE GABARIT DU MATCH, un seul pour toute la colonne (D1) : il ne dépend que de l'en-tête.
  const gabarit = cardDensity(header)
  const vitalityFade = useMemo(() => msToFrames(VITALITY_FADE_MS, doc), [doc])
  const readingFull = useMemo(() => msToFrames(READING_FULL_MS, doc), [doc])
  const flashFrames = useMemo(() => Math.max(1, msToFrames(FLASH_MS, doc)), [doc])
  const presence = useMemo(() => vitalityPresence(doc), [doc])
  // LA SCÈNE DES EFFETS D'ÉQUIPEMENT : les camps par vie (le capteur adverse en a besoin,
  // même contrat que le calque), l'axe de temps des fenêtres de pose, et les instants de
  // translocation — une lecture de document, donc UNE FOIS par document et jamais par image
  // (le canvas lit le même calque de son côté, par son propre besoin : lui veut les POSITIONS,
  // la fiche ne veut que les INSTANTS).
  // LE CAMP D'UNE VIE PAR SLOT ET PAR IMAGE (résolveur frame-aware) : un slot de biped est
  // réattribué entre manches, le camp doit suivre l'occupant. Le capteur adverse le lit à
  // l'image du joueur interrogé / à la pose du capteur (cf. equipmentZones).
  const sideOfSlot = useMemo(() => sideResolver(buildSlotOwnership(players)), [players])
  // L'ÉCLAT DE TRANSLOCATION EST DATÉ PAR L'ÉVÉNEMENT DU FILM (schéma 38, 2026-09-03) :
  // `translocations[]` porte l'instant EXACT de chaque usage — plus jamais le `spent`, qui date
  // la FIN de l'équipement avec jusqu'à 16,5 s de retard mesuré, ni l'heuristique spatiale
  // supprimée le même jour. Sur un artefact antérieur au schéma 38, `teleportMoments` retombe
  // sur le repli daté du `spent` (kill-switch dans `placementTeleport.ts`). La fiche ne
  // consomme que (slot, frame) : une translocation sans position l'allume comme les autres.
  const teleports = useMemo(() => teleportMoments(doc), [doc])
  const fxScene = useMemo<CardFxScene>(
    () => ({
      zones: {
        placements: doc.equipmentPlacements,
        sideOfSlot,
      },
      time: { frameMs: frameToMs(1, doc), frames: doc.frameCount },
      teleports,
    }),
    [doc, sideOfSlot, teleports],
  )
  // LE CALQUE DE SCORE PASSE PAR SA GARDE D'HORLOGE, une seule fois pour toute la colonne :
  // absent = artefact antérieur au schéma 12, mode sans compteur, ou origine non recalée
  // (cf. lib/replay/scoreTimeline.filmClockTrusted). Les fiches et les en-têtes n'ont alors
  // de plus à dire qu'avant — aucune ligne ne se vide, aucun zéro n'apparaît.
  const scoreTimeline = useMemo(() => scoreTimelineOf(doc), [doc])

  if (groups.length === 0) {
    // LE DIAGNOSTIC DU PONT S'AFFICHE AVEC LE CONSTAT (coverage.bridge, consommé depuis le
    // 2026-09-02) : « aucune vie rattachée » sans ses dénominateurs se lisait comme un bug
    // muet — avec eux, on voit si le film n'a rien nommé (0/N) ou si la table d'index est
    // tombée (collisions).
    const bridge = doc.coverage?.bridge
    return (
      <div className="rounded-lg border border-border bg-card p-3">
        <p className="text-xs text-muted-foreground">{t.rosterEmpty}</p>
        {bridge && (
          <p className="mt-1 text-3xs text-muted-foreground">
            {t.bridgeDiag(bridge.livesNamed, bridge.livesTotal, bridge.slotCollisions)}
          </p>
        )}
      </div>
    )
  }

  return (
    // LA HAUTEUR VIENT DE LA RANGÉE, JAMAIS DES FICHES (technique du POC) : `h-full min-h-0`
    // laisse la colonne se rétrécir, et le défilement vit À L'INTÉRIEUR de chaque colonne.
    // Sous `xl`, la page borne la pile à 60 vh (`replay.tsx`) : la colonne défile dans cette
    // borne.
    //
    // PLUS DE CARTE DE COLONNE (option 2a du handoff 2026-08-27) : chaque fiche est une
    // TUILE autonome, le bandeau d'équipe est posé AU-DESSUS de la pile — la boîte qui les
    // enfermait ne disait rien de plus. Gaps de la maquette : 10 px entre colonnes, 6 px
    // sous le bandeau, 4 px entre tuiles. LES COLONNES D'ÉQUIPE NE CHANGENT PAS avec le
    // gabarit (`repeat(groups.length, 1fr)`) : c'est À L'INTÉRIEUR d'un camp que les sièges
    // passent en grille (D2 : pas de groupe « sans équipe » à traiter ; D3 : le FFA garde ses
    // N colonnes d'un siège).
    <div
      className="grid h-full min-h-0 gap-2.5"
      style={{ gridTemplateColumns: `repeat(${groups.length}, 1fr)` }}
    >
      {groups.map((group, gi) => (
        <div
          key={group.side ?? `sans-equipe-${gi}`}
          className="flex h-full min-h-0 flex-col gap-1.5"
        >
          <ReplayTeamHeader
            players={group.seats.map((s) => seatOccupantAt(s, frame))}
            side={group.side}
            xuidMeta={xuidMeta}
            locale={locale}
          />
          <div className={gabarit.seatGrid ? SEATS_GRID_CLASS : SEATS_COLUMN_CLASS}>
            {group.seats.map((seat) => (
              <ReplayPlayerCard
                key={seat.key}
                player={seatOccupantAt(seat, frame)}
                doc={doc}
                frame={frame}
                presence={presence}
                vitalityFade={vitalityFade}
                readingFull={readingFull}
                flashFrames={flashFrames}
                locale={locale}
                scoreTimeline={scoreTimeline}
                fxScene={fxScene}
                gabarit={gabarit}
              />
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}
