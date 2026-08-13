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
import { useMemo, type CSSProperties } from 'react'

import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { MatchScoreboardRow } from '@/lib/api/types'

import { equippedWeapons } from './equippedLogic'
import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import { formatSeconds, frameToMs, freshness, msToFrames, READING_FADE, trackWindow } from './replayLogic'
import type { ReplayDocumentReady } from './replayNormalize'
import { ReplayInventoryRow } from './ReplayInventoryRow'
import { ReplayWeaponsRow } from './ReplayWeaponsRow'
import {
  buildPlayers,
  groupByTeam,
  playerName,
  playerStateAt,
  vitalityPresence,
  type ReplayPlayer,
  type PlayerState,
  type VitalityPresence,
} from './rosterLogic'

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
 * Durée des DEUX éclats d'événement (coup fatal, réapparition), en temps réel — assez pour
 * être vus sans être subis, calée sur la rémanence des lancers. L'état de mort, lui, est
 * porté en continu par le fond de la fiche.
 */
const FLASH_MS = 1_400
/** Durées CSS des deux animations d'éclat (cf. globals.css) — le délai négatif s'y rapporte. */
const DEATH_FLASH_TOTAL_S = 1.86
const RESPAWN_FLASH_S = 0.55

interface ReplayTeamsProps {
  doc: ReplayDocumentReady
  scoreboard: MatchScoreboardRow[]
  frame: number
  locale: ReplayLocale
}

export function ReplayTeams({ doc, scoreboard, frame, locale }: ReplayTeamsProps) {
  const t = REPLAY_TEXT[locale]
  const groups = useMemo(
    () => groupByTeam(buildPlayers(doc, scoreboard)),
    [doc, scoreboard],
  )
  const vitalityFade = useMemo(() => msToFrames(VITALITY_FADE_MS, doc), [doc])
  const readingFull = useMemo(() => msToFrames(READING_FULL_MS, doc), [doc])
  const flashFrames = useMemo(() => Math.max(1, msToFrames(FLASH_MS, doc)), [doc])
  const presence = useMemo(() => vitalityPresence(doc), [doc])

  if (groups.length === 0) {
    return (
      <div className="rounded-lg border border-border bg-card p-3">
        <p className="text-xs text-muted-foreground">{t.rosterEmpty}</p>
      </div>
    )
  }

  return (
    <div className="grid gap-2" style={{ gridTemplateColumns: `repeat(${groups.length}, 1fr)` }}>
      {groups.map((group, gi) => (
        <div key={group.side ?? `sans-equipe-${gi}`} className="rounded-lg border border-border bg-card p-2">
          <h3
            className="mb-2 flex items-baseline justify-between border-b border-border pb-1 text-[11px] font-semibold uppercase tracking-wide"
            style={{ color: teamColor(gi) }}
          >
            <span>{group.side ?? t.teamUnknown}</span>
            <span className="font-mono text-[10px] font-normal tabular-nums text-muted-foreground">
              {group.players.length}
            </span>
          </h3>
          <div className="flex flex-col">
            {group.players.map((p) => (
              <PlayerCard
                key={p.xuid}
                player={p}
                doc={doc}
                frame={frame}
                presence={presence}
                vitalityFade={vitalityFade}
                readingFull={readingFull}
                flashFrames={flashFrames}
                locale={locale}
              />
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}

/**
 * teamColor distingue les camps sans rien affirmer sur eux. Les tokens de COMPARAISON disent
 * « ceci n'est pas cela » et rien de plus ; `team-ally` / `team-enemy` diraient de quel côté on
 * est, ce que ce rejeu ignore — il se regarde de l'extérieur, pas depuis un joueur.
 */
function teamColor(index: number): string {
  const tokens = ['compare-a', 'compare-b', 'compare-c'] as const
  return tokenCssVar(tokens[index % tokens.length])
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
}

function PlayerCard({ player, doc, frame, presence, vitalityFade, readingFull, flashFrames, locale }: PlayerCardProps) {
  const t = REPLAY_TEXT[locale]
  const state = playerStateAt(player, frame, presence)
  const name = playerName(player) ?? t.unknownPlayer
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
  let flashClass = ''
  const style: CSSProperties = {}
  if (!state.alive) {
    style.boxShadow = `inset 2px 0 0 ${tokenCssVar('destructive')}`
    style.background = `color-mix(in srgb, ${tokenCssVar('destructive')} 12%, transparent)`
    if (deathAge >= 0 && deathAge <= flashFrames) {
      flashClass = 'replay-flash-death'
      style.animationDelay = `${(-(deathAge / flashFrames) * DEATH_FLASH_TOTAL_S).toFixed(3)}s`
    }
  } else if (lifeAge >= 0 && lifeAge <= flashFrames) {
    flashClass = 'replay-flash-respawn'
    style.animationDelay = `${(-(lifeAge / flashFrames) * RESPAWN_FLASH_S).toFixed(3)}s`
  }
  return (
    <div
      className={`flex flex-col gap-0.5 border-t border-border py-1 first:border-t-0 ${flashClass}`}
      style={style}
    >
      <div className="flex items-baseline justify-between gap-1.5">
        <span
          className="truncate text-[11.5px] font-medium"
          style={state.alive ? undefined : { color: tokenCssVar('destructive') }}
          title={name}
        >
          {name}
        </span>
        <KdaBadge board={player.board} />
      </div>
      {/* HAUTEUR CONSTANTE vivant/mort : les trois zones ci-dessous RÉSERVENT leur place
          dans les DEUX états. La mort remplace le contenu d'une zone, jamais la zone —
          une fiche qui se compacte fait sauter toute la colonne à chaque mort. Les
          hauteurs réservées sont celles du contenu vivant : deux barres empilées (14px),
          une rangée d'icônes d'armes (16px + 2px de souligné), une rangée d'icônes de
          HUD (16px). */}
      <div className="flex h-3.5 flex-col justify-center gap-0.5">
        {state.alive ? (
          <>
            {/* Le bouclier AU-DESSUS de la santé : l'ordre dans lequel le jeu les encaisse. */}
            <VitalityBar reading={state.shield} fade={vitalityFade} name={t.shieldLabel} token="info" />
            <VitalityBar reading={state.health} fade={vitalityFade} name={t.healthLabel} token="success" />
          </>
        ) : (
          <RespawnRow state={state} doc={doc} frame={frame} locale={locale} />
        )}
      </div>
      <div className="flex min-h-[18px] items-center">
        {state.alive && (
          <ReplayWeaponsRow
            doc={doc}
            state={state}
            read={equipped}
            frame={frame}
            readingFull={readingFull}
            filmIndex={filmIndex}
            locale={locale}
          />
        )}
      </div>
      <div className="flex min-h-4 items-center">
        {state.alive && state.life && (
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
    </div>
  )
}

/**
 * KdaBadge — FRAGS, MORTS, ASSISTANCES, chacun sa couleur. Trois nombres collés sans
 * distinction se lisent comme un seul nombre à trois chiffres.
 *
 * CES CHIFFRES VIENNENT DE LA BASE, pas du film, et ils valent pour TOUT le match : ce ne sont
 * pas des compteurs à l'instant lu. Sans ligne de scoreboard, on n'affiche rien.
 */
function KdaBadge({ board }: { board?: MatchScoreboardRow }) {
  if (!board) return null
  const parts: [number | null | undefined, string][] = [
    [board.kills, 'success'],
    [board.deaths, 'destructive'],
    [board.assists, 'info'],
  ]
  return (
    <span className="inline-flex shrink-0 items-baseline gap-0.5 font-mono text-[10px] tabular-nums">
      {parts.map(([v, token], i) => (
        <span key={token}>
          {i > 0 && <span className="opacity-50">/</span>}
          <span className="font-semibold" style={{ color: tokenCssVar(token as 'success') }}>
            {v ?? '?'}
          </span>
        </span>
      ))}
    </span>
  )
}

/**
 * VitalityBar — bouclier ou santé, lus dans le MÊME enregistrement que la position.
 *
 * LA BARRE EST TOUJOURS PLEINE AU DÉPART D'UNE VIE : on apparaît vie et bouclier pleins
 * (règle du jeu), et le flux différentiel ne retransmet que ce qui change — l'absence de
 * mesure depuis le spawn veut dire « plein », pas « inconnu » (décision utilisateur
 * 2026-08-12, doctrine du POC). UNE PISTE VIDE reste une MESURE : bouclier brisé, vie
 * entamée. Reading null = le document ne porte pas ce champ (titre sans décodage film) :
 * la ligne n'existe pas — on n'invente pas une jauge pour une donnée qui n'existe nulle
 * part dans le document.
 */
function VitalityBar({
  reading,
  fade,
  name,
  token,
}: {
  reading: { value: number; age: number } | null
  fade: number
  name: string
  token: 'info' | 'success'
}) {
  if (!reading) return null
  const fresh = freshness(reading.age, fade, READING_FADE)
  return (
    <div
      className="h-1 overflow-hidden rounded-sm bg-muted"
      style={{ opacity: fresh }}
      title={name}
      aria-label={name}
    >
      <div
        className="h-full rounded-sm"
        style={{
          width: `${Math.max(0, Math.min(1, reading.value)) * 100}%`,
          background: tokenCssVar(token),
        }}
      />
    </div>
  )
}

/**
 * RespawnRow — ce que la fiche d'un joueur mort a de plus utile à dire.
 *
 * LE RETOUR EST LU, PAS DÉDUIT D'UNE CONSTANTE : c'est l'image de départ de la vie suivante du
 * même joueur. Mesure publiée sur le film de référence : 90 épisodes de mort, 82 avec un retour
 * lisible, médiane 8,0 s, 66 sur 82 exactement à 7,9-8,0 s. Les 8 sans retour affichent une
 * LACUNE — jamais un délai deviné, ce serait remplacer une mesure absente par une moyenne.
 */
function RespawnRow({
  state,
  doc,
  frame,
  locale,
}: {
  state: PlayerState
  doc: ReplayDocumentReady
  frame: number
  locale: ReplayLocale
}) {
  const t = REPLAY_TEXT[locale]
  if (state.respawnFrame < 0) {
    // « retour ? » sans infobulle de méthode : la justification (fin de partie sans vie
    // suivante) vit dans le commentaire de PlayerState.respawnFrame, pas à l'écran.
    return (
      <span className="font-mono text-[9.5px] text-muted-foreground">
        {t.respawnUnknown}
      </span>
    )
  }
  const remainMs = frameToMs(state.respawnFrame - frame, doc)
  // La barre montre l'AVANCEMENT DEPUIS LA MORT : la mort est datée par la fin de la vie
  // précédente, le retour par le départ de la suivante — deux lectures, aucune constante.
  // Quand la mort n'est pas datée (sinceDeath < 0), le compte s'affiche sans barre plutôt
  // qu'avec un avancement faux.
  const span = state.sinceDeath >= 0 ? state.respawnFrame - (frame - state.sinceDeath) : 0
  const progress = span > 0 ? Math.max(0, Math.min(1, state.sinceDeath / span)) : null
  return (
    <span className="inline-flex items-center gap-1.5 font-mono text-[9.5px] text-muted-foreground">
      {progress !== null && (
        <span
          className="inline-block h-1 w-9 overflow-hidden rounded-sm bg-muted"
          role="progressbar"
          aria-label={t.respawnBarLabel}
        >
          <span
            className="block h-full rounded-sm opacity-80"
            style={{ width: `${(progress * 100).toFixed(1)}%`, background: tokenCssVar('success') }}
          />
        </span>
      )}
      {t.respawnIn} <b className="tabular-nums">{formatSeconds(remainMs)}</b>
    </span>
  )
}

// La rangée d'armes (arme en main, secondaire, indicateur de swap) vit dans
// ReplayWeaponsRow.tsx — extraite avec sa logique d'infobulles quand ce fichier a franchi
// le seuil de taille du dépôt.
