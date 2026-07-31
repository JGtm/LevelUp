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
import type { MatchScoreboardRow } from '@/lib/api/types'

import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import { frameToMs, freshness, msToFrames } from './replayLogic'
import type { ReplayDocumentReady } from './replayNormalize'
import {
  buildPlayers,
  grenadesCarried,
  groupByTeam,
  inventoryAt,
  loadoutAt,
  playerName,
  playerStateAt,
  selectedGrenadeRank,
  type ReplayPlayer,
  type PlayerState,
} from './rosterLogic'

/** Maintien du bouclier : même valeur que sur la carte, pour que les deux disent la même chose. */
const SHIELD_HOLD_MS = 2_000
/**
 * Au-delà, une lecture d'inventaire est au plancher d'opacité. 20 s est l'écart médian entre
 * deux images-clés du film : c'est donc l'âge maximal ordinaire d'une lecture.
 */
const READING_FULL_MS = 20_000
/** Une lecture au plancher garde la moitié de son opacité : elle s'estompe, elle ne disparaît pas. */
const READING_FADE = 0.5

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
  const shieldHold = useMemo(() => msToFrames(SHIELD_HOLD_MS, doc), [doc])
  const readingFull = useMemo(() => msToFrames(READING_FULL_MS, doc), [doc])

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
                shieldHold={shieldHold}
                readingFull={readingFull}
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
  shieldHold: number
  readingFull: number
  locale: ReplayLocale
}

function PlayerCard({ player, doc, frame, shieldHold, readingFull, locale }: PlayerCardProps) {
  const t = REPLAY_TEXT[locale]
  const state = playerStateAt(player, frame, shieldHold)
  const name = playerName(player) ?? t.unknownPlayer
  return (
    <div
      className="flex flex-col gap-0.5 border-t border-border py-1 first:border-t-0"
      style={state.alive ? undefined : { boxShadow: `inset 2px 0 0 ${tokenCssVar('destructive')}` }}
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
        <ShieldBar state={state} label={t.shieldUnread} />
      ) : (
        <RespawnRow state={state} doc={doc} frame={frame} locale={locale} />
      )}
      <WeaponsRow
        doc={doc}
        state={state}
        frame={frame}
        readingFull={readingFull}
        locale={locale}
      />
      {state.life && (
        <InventoryRow
          doc={doc}
          slot={state.life.slot}
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
  frame,
  readingFull,
  locale,
}: {
  doc: ReplayDocumentReady
  slot: number
  frame: number
  readingFull: number
  locale: ReplayLocale
}) {
  const t = REPLAY_TEXT[locale]
  const read = inventoryAt(doc, slot, frame)
  if (!read) return null
  const { state } = read
  const grenades = grenadesCarried(state, doc.grenadeLabels)
  const selected = selectedGrenadeRank(state)
  const ability = abilityText(doc, state.a, t.abilityUnknown)
  const ammo = state.am ?? []
  if (grenades.length === 0 && !ability && ammo.length === 0) return null

  return (
    <div
      className="flex flex-wrap items-center gap-x-2 gap-y-0.5 font-mono text-[9.5px] text-muted-foreground"
      style={{ opacity: freshness(read.age, readingFull, READING_FADE) }}
      title={`${t.inventoryAge} ${formatSeconds(frameToMs(read.age, doc))}`}
    >
      {grenades.map((g) => (
        <span
          key={g.rank}
          className={g.rank === selected ? 'font-semibold' : undefined}
          style={g.rank === selected ? { color: tokenCssVar('warning') } : undefined}
          title={g.rank === selected ? t.grenadeSelected : undefined}
        >
          {g.name} ×{g.count}
        </span>
      ))}
      {ability && (
        <span
          className={ability.known ? undefined : 'border-b border-dashed border-border'}
          title={ability.known ? undefined : t.abilityUnknownHint}
        >
          {ability.text}
        </span>
      )}
      {ammo.map((a, i) => (
        <AmmoCell
          key={i}
          index={i}
          ammo={a}
          noneLabel={t.ammoNone}
          noneHint={t.ammoNoneHint}
          hint={t.ammoSlotHint}
          gaugeLabel={t.gaugeLabel}
        />
      ))}
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
): { text: string; known: boolean } | null {
  if (index === undefined) return null
  const name = doc.abilityLabels?.[String(index)]
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
  noneLabel,
  noneHint,
  hint,
  gaugeLabel,
}: {
  index: number
  ammo: { mag?: number; res?: number; gauge?: number }
  noneLabel: string
  noneHint: string
  hint: string
  gaugeLabel: string
}) {
  return (
    <span className="inline-flex items-center gap-1 tabular-nums" title={hint}>
      <i className="not-italic text-[8px] opacity-45">{index}</i>
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
function ShieldBar({ state, label }: { state: PlayerState; label: string }) {
  if (!state.shield) {
    return <span className="font-mono text-[9.5px] italic text-muted-foreground/70">{label}</span>
  }
  const fresh = freshness(state.shield.age, 20, READING_FADE)
  return (
    <div className="h-1 overflow-hidden rounded-sm bg-muted" style={{ opacity: fresh }}>
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
  return (
    <span className="font-mono text-[9.5px] text-muted-foreground">
      {t.respawnIn} <b className="tabular-nums">{formatSeconds(remainMs)}</b>
    </span>
  )
}

/** formatSeconds rend un délai court en secondes avec une décimale (virgule décimale FR/EN). */
function formatSeconds(ms: number): string {
  return `${(Math.max(0, ms) / 1000).toFixed(1)} s`
}

/**
 * WeaponsRow — les armes portées, lues aux images-clés.
 *
 * DEUX CHOSES QUE CETTE RANGÉE NE DIT PAS, et qu'il faut se garder de laisser croire :
 *   - QUELLE arme est en main. Le loadout est l'inventaire, pas la main : le sélecteur
 *     d'emplacement n'est pas dans ce document.
 *   - la CONTINUITÉ. Une image-clé toutes les ~20 s : un ramassage entre deux est invisible.
 * D'où l'estompage avec l'âge, et l'âge exact dans l'infobulle.
 *
 * UNE ARME NON CATALOGUÉE GARDE SON IDENTIFIANT plutôt que d'emprunter un nom voisin.
 */
function WeaponsRow({
  doc,
  state,
  frame,
  readingFull,
  locale,
}: {
  doc: ReplayDocumentReady
  state: PlayerState
  frame: number
  readingFull: number
  locale: ReplayLocale
}) {
  const t = REPLAY_TEXT[locale]
  if (!state.life) return null
  const read = loadoutAt(doc, state.life.slot, frame)
  if (!read) {
    return <span className="font-mono text-[9.5px] italic text-muted-foreground/70">{t.loadoutUnread}</span>
  }
  const ageMs = frameToMs(read.age, doc)
  const hint = `${t.loadoutAge} ${formatSeconds(ageMs)}`
  return (
    <div
      className="flex flex-wrap items-center gap-x-1.5 gap-y-0.5 font-mono text-[9.5px] text-muted-foreground"
      style={{ opacity: freshness(read.age, readingFull, READING_FADE) }}
      title={hint}
    >
      {read.weapons.map((id) => (
        <span key={id} className={doc.weaponLabels?.[id] ? '' : 'border-b border-dashed border-border'}>
          {doc.weaponLabels?.[id] ?? id}
        </span>
      ))}
    </div>
  )
}
