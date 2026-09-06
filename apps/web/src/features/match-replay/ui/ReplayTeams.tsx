/**
 * ReplayTeams — les fiches joueur, à CÔTÉ de la carte et jamais dessus.
 *
 * CE QUE CETTE COLONNE APPORTE, et que la carte ne peut pas dire : qui est qui. La carte
 * montre des traces ; huit fiches montrent des gens, avec leur état à l'instant lu — vivant ou
 * mort, bouclier, armes portées, temps avant le retour.
 *
 * TROIS RÈGLES QUI NE SE NÉGOCIENT PAS ICI :
 *   1. Une valeur non lue s'affiche comme une lacune, jamais comme un zéro ni une moyenne.
 *   2. Une lecture ancienne PÂLIT et dit son âge — l'inventaire ne se lit qu'aux images-clés,
 *      une toutes les ~20 s, et le faire passer pour l'instant courant était un défaut réel.
 *   3. Aucun littéral de couleur : les rôles passent par des tokens sémantiques.
 */
import { useMemo } from 'react'

import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import {
  playerCountersAt,
  scoreTimelineOf,
  type ReplayScoreTimelineReady,
} from '@/lib/replay/scoreTimeline'
import type { XuidMeta } from '@/features/match-view/xuidMeta'
import type { MatchScoreboardRow } from '@/lib/api/types'
import { stripBotSuffix } from '@/lib/players/displayName'

import { ReplayTeamHeader } from './ReplayTeamHeader'
import { activeEquipmentAt } from '../equipmentFx'
import type { PlacementWindowTime } from '../layers/equipmentPlacementsLayer'
import { NO_ZONES, zonePresenceAt, type ZonePresence, type ZoneScene } from '../equipmentZones'
import { objectiveMarkAt } from '../objectiveMark'
import { ReplayObjectiveMark } from './ReplayObjectiveMark'
import { equippedWeapons } from '../equippedLogic'
import { lastTeleportAge, teleportMoments, type TranslocationMoment } from '../placementTeleport'
import { cardChrome, hasUnderLayer, playerCardFx } from '../playerCardFx'
import { ReplayCountersBadge } from './ReplayCountersBadge'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { frameToMs, msToFrames, positionAt, trackWindow } from '../replayLogic'
import type { PresenceHeader } from '../presenceFeed'
import { buildSeats, groupSeatsByTeam, seatOccupantAt } from '../seatLogic'
import type { ReplayDocumentReady } from '../replayNormalize'
import { ReplayInventoryRow } from './ReplayInventoryRow'
import { EliminatedBox, VitalityBar } from './ReplayVitality'
import { ReplayWeaponsRow } from './ReplayWeaponsRow'
import {
  buildPlayers,
  buildSlotOwnership,
  playerName,
  playerStateAt,
  sideResolver,
  vitalityPresence,
  type ReplayPlayer,
  type VitalityPresence,
} from '../rosterLogic'

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
 * depuis le 2026-08-27 : ce composant ne fait plus que lui donner les âges et rendre.
 */
const FLASH_MS = 1_400

interface ReplayTeamsProps {
  doc: ReplayDocumentReady
  scoreboard: MatchScoreboardRow[]
  frame: number
  locale: ReplayLocale
  /** Camp de chaque xuid : il donne sa couleur au titre de la colonne (allié / adverse). */
  xuidMeta?: XuidMeta
  /** En-tête du match (heure de début) : le repère des relais de siège (cf. seatLogic). */
  header?: PresenceHeader | null
}

// LA FICHE COMPACTE EST DEVENUE LA FICHE (décision utilisateur du 2026-08-24 : « fiches
// compactes va devenir la seule et unique option ») : la variante longue — zone du joueur,
// deux rangées d'inventaire, munitions des armes rangées — est SUPPRIMÉE avec son réglage.

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
    // Hors rangée (repli étroit), aucune hauteur n'est imposée : rien ne défile,
    // comportement d'origine.
    //
    // PLUS DE CARTE DE COLONNE (option 2a du handoff 2026-08-27) : chaque fiche est une
    // TUILE autonome, le bandeau d'équipe est posé AU-DESSUS de la pile — la boîte qui les
    // enfermait ne disait rien de plus. Gaps de la maquette : 10 px entre colonnes, 6 px
    // sous le bandeau, 4 px entre tuiles.
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
          <div className="flex min-h-0 flex-1 flex-col gap-1 overflow-y-auto">
            {group.seats.map((seat) => (
              <PlayerCard
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
              />
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}

/**
 * Ce que les EFFETS d'une fiche lisent en dehors d'elle-même : la scène des zones (poses +
 * camps), l'axe de temps des fenêtres de pose, et les passages par faille. Construite UNE
 * FOIS par la colonne — les fiches la consomment, aucune ne la recalcule.
 */
interface CardFxScene {
  zones: ZoneScene
  time: PlacementWindowTime
  teleports: readonly TranslocationMoment[]
}

interface PlayerCardProps {
  player: ReplayPlayer
  doc: ReplayDocumentReady
  frame: number
  presence: VitalityPresence
  vitalityFade: number
  readingFull: number
  flashFrames: number
  locale: ReplayLocale
  /** Calque de score du film, déjà passé par la garde d'horloge. */
  scoreTimeline?: ReplayScoreTimelineReady
  fxScene: CardFxScene
}

function PlayerCard({ player, doc, frame, presence, vitalityFade, readingFull, flashFrames, locale, scoreTimeline, fxScene }: PlayerCardProps) {
  const t = REPLAY_TEXT[locale]
  // LES COMPTEURS DU FILM, quand ce joueur est publié. `null` veut dire « pas publié », pas
  // « à zéro » : sur le témoin Slayer 6 joueurs sur 8 en portent, et le mode Oddball n'en
  // publie aucun (0/32 en phase 0). La fiche retombe alors sur les totaux de la BASE, qui
  // valent pour tout le match — c'est ce qu'elle affichait avant ce lot.
  const live = playerCountersAt(scoreTimeline, player.xuid, frame)
  const state = playerStateAt(player, frame, presence)
  // Suffixe « [bot] » = marqueur de donnée killsource (schéma 36), pas d'affichage —
  // retiré ici sans toucher au repli `t.unknownPlayer` (playerName() reste `null`-able).
  const rawName = playerName(player)
  const name = (rawName ? stripBotSuffix(rawName) : null) ?? t.unknownPlayer
  const equipped = state.life ? equippedWeapons(doc, state.life.slot, frame) : null
  // L'index de FILM du joueur : la clé des lancers de grenade (l'auteur y est écrit).
  const filmIndex = doc.roster.find((r) => r.xuid === player.xuid)?.filmIndex ?? null
  // Les DEUX éclats d'événement : le coup fatal et la réapparition. Ils durent le temps de
  // leur animation ; le délai NÉGATIF la fait reprendre à son avancement réel, donc elle
  // reste juste après un saut dans le temps de lecture (cf. globals.css).
  const deathAge = state.sinceDeath
  const lifeAge = state.alive && state.life && trackWindow(state.life).start > 0
    ? frame - trackWindow(state.life).start
    : -1
  // L'ÉTAT ACTIF d'équipement de la vie courante : même chaîne slot -> fiche que le flash
  // de mort. L'effet couvre TOUTE la fiche (demande utilisateur du 14/08) et dure
  // exactement l'épisode mesuré — une fiche morte n'en porte jamais (les épisodes se
  // ferment à la mort au plus tard).
  const equipment = state.alive && state.life
    ? activeEquipmentAt(doc, state.life.slot, frame)
    : null
  // LES ZONES SOUS LE JOUEUR et le dernier PASSAGE par faille de cette vie : mêmes portes
  // que la carte (equipmentZones.ts), position interpolée de la vie courante. Sans position
  // lisible — et sur une fiche morte — aucune zone n'est affirmée.
  const pos = state.alive && state.life ? positionAt(state.life.points, frame) : null
  const zones = pos && state.life
    ? zonePresenceAt(fxScene.zones, { slot: state.life.slot, x: pos.x, y: pos.y, frame }, fxScene.time)
    : NO_ZONES
  // L'OBJECTIF PORTÉ : drapeau, crâne, VIP (des périodes attribuées, donc un état qui dure),
  // ou la prise de base (un instant attribué, tenu quelques secondes — cf. objectiveMark.ts).
  // Comme pour l'équipement et les zones, une fiche morte n'en porte aucun : un mort a lâché
  // ce qu'il tenait, et la tuile ne dit plus que la mort.
  const objective = state.alive ? objectiveMarkAt(doc, player.xuid, frame) : null
  const teleportAge = state.alive && state.life
    ? lastTeleportAge(fxScene.teleports, state.life.slot, frame)
    : -1
  // LA COMPOSITION DES EFFETS vit dans playerCardFx.ts (mort, éclats à délai négatif,
  // verre trempé du camouflage, encadrés, voile de l'écran occultant) : la fiche lui donne
  // les âges et rend ce qu'il dit, réparti sur ses DEUX couches (dessous / incrustation).
  const fx = playerCardFx({
    alive: state.alive,
    deathAge,
    lifeAge,
    teleportAge,
    flashFrames,
    equipment,
    zones,
    objective,
    text: t,
  })
  return (
    // LA TUILE (option 2a du handoff 2026-08-27) : chaque fiche porte sa bordure et son
    // fond — dégradé court autour de `card` en vie, `card` teinté destructive en mort
    // (cf. playerCardFx.cardChrome). `shrink-0` : une tuile ne se tasse jamais, la colonne
    // défile.
    <div
      className="relative flex shrink-0 flex-col rounded-lg border px-2.5 py-2"
      style={cardChrome(state.alive)}
      title={fx.title}
    >
      {/* LA COUCHE D'EFFETS ÉPOUSE LA TUILE (option 2a) : fonds, voiles, flou et cadres
          vivent sur cette couche `inset-0 rounded-lg`, SOUS le contenu — les rangées sont
          en `relative` pour peindre au-dessus d'elle. Les éclats de mort/réapparition
          animent SON fond, jamais celui de la fiche. */}
      {hasUnderLayer(fx) && (
        <div
          aria-hidden
          className={`replay-card-fx pointer-events-none absolute inset-0 rounded-lg ${fx.flashClass}`}
          style={fx.underStyle}
        />
      )}
      {/* LE FILIGRANE DE PORTEUR : la couche du seul canal qui restait libre sur la fiche —
          derrière le contenu, sans toucher ni la bordure (trois cadres d'équipement) ni le
          fond (verre, voile, teinte de mort). Déclarée AVANT les rangées, qui sont
          `relative` : l'ordre de peinture du DOM les met au-dessus, comme la couche d'effets. */}
      {objective && <ReplayObjectiveMark kind={objective} />}
      {/* AUCUNE MARQUE D'IDENTITÉ SUR LA FICHE (demande utilisateur du 2026-08-25) : le glyphe
          « ami » a été retiré de la colonne. Il reste au FIL des éliminations, où il sert à
          reconnaître un nom au milieu d'événements qui défilent ; sur une fiche, la colonne
          d'équipe et le nom disent déjà tout ce qu'il y a à savoir. */}
      <div className="relative flex items-baseline gap-1.5">
        <span
          className={`min-w-0 flex-1 truncate text-[11.5px] font-bold uppercase tracking-[.06em] ${
            state.alive ? 'text-foreground' : 'text-muted-foreground'
          }`}
          title={name}
        >
          {name}
        </span>
        <ReplayCountersBadge board={player.board} live={live} locale={locale} />
      </div>
      {/* HAUTEUR CONSTANTE vivant/mort : le CORPS de la fiche est une zone à hauteur FIXE
          (35 px = barres 11 + marge 6 + inventaire 18) dans les DEUX états. La mort
          remplace son CONTENU — l'encadré « Éliminé » remplit toute la zone — jamais la
          zone : une fiche qui change de hauteur fait sauter toute la colonne à chaque mort
          (retour utilisateur du 2026-08-24). `overflow-hidden` est la garantie, pas un
          ornement. */}
      <div className="relative mt-[7px] h-[35px] overflow-hidden">
        {state.alive ? (
          <>
            {/* Le bouclier (5 px) AU-DESSUS de la santé (3 px) : l'ordre dans lequel le jeu
                les encaisse, dit aussi par l'épaisseur. */}
            <div className="flex flex-col gap-[3px]">
              <VitalityBar reading={state.shield} fade={vitalityFade} name={t.shieldLabel} token="info" heightPx={5} />
              <VitalityBar reading={state.health} fade={vitalityFade} name={t.healthLabel} token="success" heightPx={3} />
            </div>
            {/* ARMES ET INVENTAIRE SUR UNE GRILLE À CELLULES FIXES (demande utilisateur du
                2026-08-24) : chaque rangée émet des cellules à largeur constante — deux
                armes, munitions de la main, grenades, capacité — pour que les fiches
                s'alignent en colonnes. `flex-nowrap` + `overflow-hidden` : la rangée ne se
                replie jamais. */}
            <div className="mt-[6px] flex h-[18px] flex-nowrap items-center gap-x-[5px] overflow-hidden">
              <ReplayWeaponsRow
                doc={doc}
                state={state}
                read={equipped}
                frame={frame}
                readingFull={readingFull}
                filmIndex={filmIndex}
                locale={locale}
              />
              {state.life && (
                <ReplayInventoryRow
                  doc={doc}
                  slot={state.life.slot}
                  equipped={equipped}
                  frame={frame}
                  readingFull={readingFull}
                  locale={locale}
                />
              )}
            </div>
          </>
        ) : (
          <EliminatedBox state={state} doc={doc} frame={frame} locale={locale} />
        )}
      </div>
      <ZoneFxOverlay zones={zones} translocationDelay={fx.translocationDelay} />
    </div>
  )
}

/**
 * Décalages des trois croix du champ de réparation : désynchronisés (délais NÉGATIFS, donc
 * déjà en vol au premier rendu) pour qu'elles ne montent pas au pas. Trois, pas plus : la
 * fiche reste une fiche, l'effet un signe.
 */
const REPAIR_CROSSES = [
  { left: '16%', delay: '-0.3s' },
  { left: '46%', delay: '-1.1s' },
  { left: '74%', delay: '-1.9s' },
] as const

/**
 * Les TROIS ÉCLAIRS de l'écran occultant (option 2a du handoff 2026-08-27) : abscisse,
 * largeur, et décalage de scintillement dans le cycle de 2,6 s (cf. globals.css,
 * `replay-zone-bolt`). Le décalage se SOUSTRAIT à l'horloge de la pose (`shroudSinceMs`) :
 * à la pose, les délais valent exactement 0 / −1,3 / −2,05 s — la composition de la
 * maquette — et après un saut de lecture chaque éclair reprend à son avancement réel,
 * jamais sur un rythme inventé (même contrat que le capteur et les éclats).
 */
const SHROUD_BOLTS = [
  { left: '10%', width: 42, offsetS: 0 },
  { left: '56%', width: 34, offsetS: 1.3 },
  { left: '33%', width: 28, offsetS: 2.05 },
] as const

/**
 * ZoneFxOverlay — l'INCRUSTATION au-dessus du contenu : le nuage noir et les ÉCLAIRS de
 * l'écran occultant (« par-dessus les infos, mais légèrement »), le contour « détecté » du
 * capteur adverse, les mini croix du champ de réparation, et le FOURREAU de translocation —
 * la lumière qui court sur la bordure, en rotation continue jusqu'à son fondu. (Les fonds,
 * voiles et cadres, eux, vivent sur la couche d'effets SOUS le contenu :
 * playerCardFx.underStyle.)
 *
 * SUR SA PROPRE COUCHE, ET C'EST LE POINT : la couche du dessous anime déjà son fond (une
 * seule animation par élément et par propriété) — la pulsation du capteur, les éclairs et
 * le fourreau vivent donc chacun sur leur enfant. MÊME GÉOMÉTRIE que la couche du dessous
 * (`inset-0 rounded-lg`, la tuile elle-même — option 2a) : les deux épousent la fiche,
 * jamais deux cadres décalés. `pointer-events-none` : l'infobulle et le survol restent
 * ceux de la fiche. `aria-hidden` : tout ce que l'incrustation montre est déjà dit en
 * texte par l'infobulle (title de la fiche).
 *
 * LE DÉLAI NÉGATIF DU CAPTEUR cale la pulsation sur l'horloge des pings du capteur le plus
 * fraîchement pingé (equipmentZones.sensorSincePingMs) : le contour de la fiche bat AVEC
 * l'onde de la carte, à la cadence officielle — jamais un rythme inventé. Ceux des éclairs
 * suivent l'horloge de la POSE de l'écran (shroudSinceMs) ; celui du fourreau, le contrat
 * des éclats (reprise à l'avancement réel après un saut de lecture).
 */
function ZoneFxOverlay({
  zones,
  translocationDelay,
}: {
  zones: ZonePresence
  translocationDelay: string | null
}) {
  const rien =
    !zones.repair && zones.shroudSinceMs === null && zones.sensorSincePingMs === null &&
    translocationDelay === null
  if (rien) return null
  return (
    <div aria-hidden className="pointer-events-none absolute inset-0 overflow-hidden rounded-lg">
      {zones.shroudSinceMs !== null && (
        <>
          <div className="replay-zone-cloud absolute inset-0" />
          {SHROUD_BOLTS.map((b) => (
            <span
              key={b.left}
              className="replay-zone-bolt absolute top-[-8%] h-[116%]"
              style={{
                left: b.left,
                width: b.width,
                animationDelay: `${(-((zones.shroudSinceMs ?? 0) / 1000 + b.offsetS)).toFixed(3)}s`,
              }}
            />
          ))}
        </>
      )}
      {zones.sensorSincePingMs !== null && (
        <div
          className="replay-zone-sensor absolute inset-0 rounded-lg border-[1.5px] border-dashed"
          style={{
            borderColor: tokenCssVar('destructive'),
            animationDelay: `${(-zones.sensorSincePingMs / 1000).toFixed(3)}s`,
          }}
        />
      )}
      {zones.repair &&
        REPAIR_CROSSES.map((c) => (
          <span
            key={c.left}
            className="replay-zone-cross absolute top-[40%] text-[10px] font-bold leading-none"
            style={{ left: c.left, color: tokenCssVar('success'), animationDelay: c.delay }}
          >
            +
          </span>
        ))}
      {translocationDelay !== null && (
        <div
          className="replay-flash-translocation absolute inset-0 rounded-lg"
          style={{ animationDelay: translocationDelay }}
        />
      )}
    </div>
  )
}

// La rangée d'armes (arme en main, secondaire, indicateur de swap) vit dans
// ReplayWeaponsRow.tsx — extraite avec sa logique d'infobulles quand ce fichier a franchi
// le seuil de taille du dépôt.
