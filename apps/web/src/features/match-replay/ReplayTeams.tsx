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

import { catalogText } from './catalogLabel'
import { equippedWeapons, type EquippedReading } from './equippedLogic'
import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import { formatSeconds, frameToMs, freshness, msToFrames, READING_FADE, trackWindow } from './replayLogic'
import type { ReplayDocumentReady } from './replayNormalize'
import { ReplayWeaponsRow } from './ReplayWeaponsRow'
import {
  buildPlayers,
  grenadesCarried,
  groupByTeam,
  inventoryAt,
  playerName,
  playerStateAt,
  selectedGrenade,
  type ReplayPlayer,
  type PlayerState,
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
  vitalityFade: number
  readingFull: number
  flashFrames: number
  locale: ReplayLocale
}

function PlayerCard({ player, doc, frame, vitalityFade, readingFull, flashFrames, locale }: PlayerCardProps) {
  const t = REPLAY_TEXT[locale]
  const state = playerStateAt(player, frame)
  const name = playerName(player) ?? t.unknownPlayer
  const equipped = state.life ? equippedWeapons(doc, state.life.slot, frame) : null
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
      {state.alive ? (
        <>
          {/* Le bouclier AU-DESSUS de la santé : l'ordre dans lequel le jeu les encaisse. */}
          <ShieldBar state={state} fade={vitalityFade} label={t.shieldUnread} name={t.shieldLabel} />
          <HealthBar state={state} fade={vitalityFade} label={t.healthUnread} name={t.healthLabel} />
        </>
      ) : (
        <RespawnRow state={state} doc={doc} frame={frame} locale={locale} />
      )}
      <ReplayWeaponsRow
        doc={doc}
        state={state}
        read={equipped}
        frame={frame}
        readingFull={readingFull}
        locale={locale}
      />
      {state.life && (
        <InventoryRow
          doc={doc}
          slot={state.life.slot}
          equipped={equipped}
          frame={frame}
          readingFull={readingFull}
          locale={locale}
        />
      )}
    </div>
  )
}

/**
 * InventoryRow — grenades portées, capacité d'armure, munitions.
 *
 * TOUT CE QUI N'EST PAS LU S'AFFICHE COMME LACUNE, jamais comme une valeur par défaut :
 * - une capacité hors table garde son NUMÉRO, marquée non interprétable — la table est
 *   partielle (4 index observés pour 11 capacités), et la combler se lirait comme une certitude ;
 * - un compteur d'utilisations n'est jamais affiché : il n'est pas localisé dans le film
 *   (36 006 positions testées, aucune ne reproduit le relevé) ;
 * - un emplacement dont le film n'écrit RIEN affiche « aucune » : pour une arme à charge,
 *   cela veut dire PLEIN, le plein étant la valeur par défaut d'un flux différentiel.
 *
 * L'ensemble pâlit avec l'âge de la lecture, comme les armes portées et pour la même raison.
 */
function InventoryRow({
  doc,
  slot,
  equipped,
  frame,
  readingFull,
  locale,
}: {
  doc: ReplayDocumentReady
  slot: number
  equipped: EquippedReading | null
  frame: number
  readingFull: number
  locale: ReplayLocale
}) {
  const t = REPLAY_TEXT[locale]
  const read = inventoryAt(doc, slot, frame)
  if (!read) return null
  const { state } = read
  const grenades = grenadesCarried(state, doc.grenadeLabels, locale)
  const selected = selectedGrenade(state)
  const ability = abilityText(doc, state.a, t.abilityUnknown, locale)
  const ammo = state.am ?? []
  if (grenades.length === 0 && !ability && ammo.length === 0) return null

  // LA LIGNE DES MUNITIONS SUIT L'ORDRE DES ARMES AU-DESSUS (l'arme dégainée d'abord) : les
  // deux lignes partagent la même lecture d'ordre, sinon chaque cellule se rattacherait à la
  // mauvaise arme. Le numéro d'emplacement lève l'ambiguïté quand l'ordre bascule.
  const ammoOrder = (equipped?.order ?? ammo.map((_, i) => i)).filter((i) => i < ammo.length)

  return (
    <div
      className="flex flex-wrap items-center gap-x-2 gap-y-0.5 font-mono text-[9.5px] text-muted-foreground"
      style={{ opacity: freshness(read.age, readingFull, READING_FADE) }}
      title={`${t.inventoryAge} ${formatSeconds(frameToMs(read.age, doc))}`}
    >
      {grenades.map((g) => {
        const isSel = typeof selected === 'object' && selected !== null && g.rank === selected.rank
        return (
          <span
            key={g.rank}
            className={isSel ? 'rounded-sm px-0.5 font-semibold' : undefined}
            style={
              isSel
                ? {
                    color: tokenCssVar('warning'),
                    boxShadow: `0 0 0 1px ${tokenCssVar('warning')}`,
                    background: `color-mix(in srgb, ${tokenCssVar('warning')} 13%, transparent)`,
                  }
                : undefined
            }
            title={isSel ? (selected.read ? t.grenadeSelectedRead : t.grenadeSelected) : undefined}
          >
            {g.name} ×{g.count}
          </span>
        )
      })}
      {selected === 'indeterminate' && (
        <span className="border-b border-dashed border-border opacity-80" title={t.grenadeSelUnknownHint}>
          {t.grenadeSelUnknown}
        </span>
      )}
      {ability && (
        <span
          className={ability.known ? undefined : 'border-b border-dashed border-border'}
          title={ability.known ? t.abilityLabel : t.abilityUnknownHint}
        >
          {ability.text}
        </span>
      )}
      {ammoOrder.map((i) => (
        <AmmoCell
          key={i}
          index={i}
          ammo={ammo[i]}
          drawn={equipped?.drawn === i}
          noneLabel={t.ammoNone}
          noneHint={t.ammoNoneHint}
          hint={equipped?.drawn === i ? `${t.ammoSlotHint} ${t.ammoDrawnHint}` : t.ammoSlotHint}
          gaugeLabel={t.gaugeLabel}
        />
      ))}
      {ammo.length > 0 && equipped && equipped.drawn === null && (
        <span
          className="border-b border-dashed border-border opacity-80"
          title={equipped.holstered ? t.weaponsHolstered : t.drawnUnknownHint}
        >
          {equipped.holstered ? t.weaponsHolsteredShort : t.drawnUnknown}
        </span>
      )}
    </div>
  )
}

/**
 * abilityText nomme la capacité, ou rend son numéro quand la table ne la connaît pas.
 * Renvoie null quand rien n'a été lu — l'absence de capacité et une capacité inconnue sont
 * deux états différents.
 */
function abilityText(
  doc: ReplayDocumentReady,
  index: number | undefined,
  unknownLabel: string,
  locale: ReplayLocale,
): { text: string; known: boolean } | null {
  if (index === undefined) return null
  const name = catalogText(doc.abilityLabels?.[String(index)], locale)
  if (name) return { text: name, known: true }
  return { text: `${unknownLabel} (${index})`, known: false }
}

/**
 * AmmoCell — une cellule par EMPLACEMENT du record, numérotée.
 *
 * L'EMPLACEMENT k EST L'ARME k, mais seulement quand la lecture du bloc est UNIQUE : 197
 * appariements sur 198 concordent dans ce cas. Sur une lecture à plusieurs candidats, la
 * correspondance se brouille — c'est ce bruit qui avait fait conclure, à tort, que le
 * rattachement échouait.
 *
 * LE NUMÉRO RESTE AFFICHÉ parce qu'il coûte deux caractères et qu'il lève l'ambiguïté sans
 * rien affirmer. La réserve, elle, est portée par l'infobulle et par le marquage des lectures
 * ambiguës.
 */
function AmmoCell({
  index,
  ammo,
  drawn,
  noneLabel,
  noneHint,
  hint,
  gaugeLabel,
}: {
  index: number
  ammo: { mag?: number; res?: number; gauge?: number }
  /** L'emplacement est DÉGAINÉ selon le sélecteur : encre plus franche, index accentué. */
  drawn: boolean
  noneLabel: string
  noneHint: string
  hint: string
  gaugeLabel: string
}) {
  return (
    <span
      className={`inline-flex items-center gap-1 tabular-nums ${drawn ? 'text-foreground' : ''}`}
      title={hint}
    >
      <i
        className={`not-italic text-[8px] ${drawn ? '' : 'opacity-45'}`}
        style={drawn ? { color: tokenCssVar('warning') } : undefined}
      >
        {index}
      </i>
      {ammo.mag !== undefined ? (
        <span>
          {ammo.mag}
          {ammo.res !== undefined && <span className="opacity-60">/{ammo.res}</span>}
        </span>
      ) : ammo.gauge !== undefined ? (
        // LA BARRE MONTRE CE QUI RESTE, ET LA DONNÉE DIT CE QUI A ÉTÉ CONSOMMÉ : d'où le
        // complément. L'afficher tel quel donnait une barre INVERSÉE — un marteau à 10 %
        // consommé s'affichait presque vide alors qu'il lui reste 90 %.
        //
        // Deux témoins fondent cette lecture (cf. ReplayAmmoSlot) : à la première image-clé du
        // match, quand personne n'a tiré, les armes à charge n'émettent AUCUN champ ; et dans
        // une même vie la valeur ne redescend jamais (6 hausses, 0 baisse).
        //
        // PAS DE COMPTEUR « n / N » : le quantum est propre à l'arme et le film ne dit pas
        // combien de charges font un plein.
        <span
          className="inline-block h-1 w-5 overflow-hidden rounded-sm bg-muted align-middle"
          aria-label={gaugeLabel}
        >
          <span
            className="block h-full rounded-sm"
            style={{
              width: `${Math.max(0, 1 - ammo.gauge) * 100}%`,
              background: tokenCssVar('warning'),
            }}
          />
        </span>
      ) : (
        <span className="italic opacity-70" title={noneHint}>
          {noneLabel}
        </span>
      )}
    </span>
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
 * ShieldBar — le bouclier lu dans le MÊME enregistrement que la position.
 *
 * UNE PISTE VIDE EST UNE MESURE : bouclier brisé. Une piste ABSENTE est une lacune. Les deux ne
 * doivent jamais se ressembler, d'où le libellé explicite quand rien n'a été lu.
 */
function ShieldBar({ state, fade, label, name }: { state: PlayerState; fade: number; label: string; name: string }) {
  if (!state.shield) {
    return <span className="font-mono text-[9.5px] italic text-muted-foreground/70">{label}</span>
  }
  const fresh = freshness(state.shield.age, fade, READING_FADE)
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
          width: `${Math.max(0, Math.min(1, state.shield.value)) * 100}%`,
          background: tokenCssVar('info'),
        }}
      />
    </div>
  )
}

/**
 * HealthBar — la santé, sur le MÊME patron que le bouclier : maintien court, fondu de
 * fraîcheur, LACUNE explicite quand rien n'a été lu. JAMAIS un plein par défaut.
 *
 * POURQUOI CE PATRON ET PAS UNE JAUGE PERMANENTE. La santé est répliquée AU CHANGEMENT et
 * presque jamais transmise (0,56 % des points, un tiers des vies sur le film de référence,
 * médiane zéro échantillon par vie) : une barre permanente serait vide ou fausse la plupart
 * du temps. Le report AVANT est honnête — valeur absolue inchangée depuis la mesure — mais
 * reporter la seule mesure d'une vie en ARRIÈRE peindrait faux tout le début de vie.
 */
function HealthBar({ state, fade, label, name }: { state: PlayerState; fade: number; label: string; name: string }) {
  if (!state.health) {
    return <span className="font-mono text-[9.5px] italic text-muted-foreground/70">{label}</span>
  }
  const fresh = freshness(state.health.age, fade, READING_FADE)
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
          width: `${Math.max(0, Math.min(1, state.health.value)) * 100}%`,
          background: tokenCssVar('success'),
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
    return (
      <span className="font-mono text-[9.5px] text-muted-foreground" title={t.respawnUnknownHint}>
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
