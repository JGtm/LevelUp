/**
 * ReplayKillFeed — le kill feed de la page de rejeu, SYNCHRONISÉ sur l'horloge du rejeu.
 *
 * CE QU'IL RÉUTILISE, ET CE QU'IL N'A PAS REFAIT. Trois briques déjà livrées pour le kill
 * feed de la carte « Dominance » servent ici telles quelles :
 *   - `collectKillEvents` (`match-view/_momentum.ts`) — ce qui fait qu'un event est un kill,
 *     et comment son arme voyage (les trois champs `weapon*` vides ensemble) ;
 *   - `teamColorResolver` (`match-view/teamColor.ts`) — la cascade de couleur d'identité de
 *     l'en-tête du scoreboard ;
 *   - `WeaponIcon` — le masque d'arme teint par CSS, qui rend l'icône lisible dans les deux
 *     thèmes et dans la couleur de l'équipe du tueur.
 *
 * CE QUI DIFFÈRE, ET POURQUOI CE N'EST PAS LE MÊME COMPOSANT. `MatchKillFeed` pose TOUS les
 * kills du match sur deux lanes alignées, au pixel, sur l'axe de catégories d'un graphe
 * ECharts : sa position horizontale se calcule à partir de `binIdx` et de l'encart de la
 * grille du graphe. Il n'y a pas de graphe dans la page de rejeu, donc pas de bin et pas
 * d'encart — et surtout on n'y veut pas tout le match, mais ce qui vient de se passer. Le
 * monter ici aurait affiché une frise figée sous un rejeu qui avance.
 *
 * L'alignement des deux horloges (le film part du début du match, les events du début du
 * gameplay) est traité dans `killFeedLogic.ts`, avec sa mesure.
 */
import { useMemo } from 'react'

import { WeaponIcon } from '@/components/ui/WeaponIcon'
import type { KillEvent } from '@/features/match-view/_momentum'
import { teamColorResolver } from '@/features/match-view/teamColor'
import type { XuidMeta } from '@/features/match-view/xuidMeta'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { MatchScoreboardRow } from '@/lib/api/types'
import { displayPlayerName } from '@/lib/players/displayName'

import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import { attachVictims, freshnessOf, killsAt, toReplayKills, type ReplayKill, type VictimPair } from './killFeedLogic'
import { formatClock } from './replayLogic'

/**
 * Fenêtre du feed, en temps de MATCH. 8 s est la durée pendant laquelle un kill reste utile
 * à la lecture d'une action ; au-delà il appartient à l'histoire du match, que la carte
 * « Dominance » raconte déjà en entier.
 */
const WINDOW_MS = 8_000
/** Au-delà de 6 lignes, le feed devient un mur : les plus anciennes sortent. */
const MAX_LINES = 6

/** Gabarit d'une icône d'arme : le format bandeau de l'atlas kill feed. */
const ICON_W = 22
const ICON_H = 9
/** Diamètre du repère servi quand l'arme est inconnue — jamais l'icône d'une autre arme. */
const DOT_PX = 7

interface Props {
  /** Kills du match, tels que `collectKillEvents` les a lus (horloge gameplay). */
  kills: KillEvent[]
  /** Paires tueur→victime datées (contrat `killer_victim`), pour NOMMER la victime. */
  victims?: VictimPair[] | null
  /** Offset du countdown pré-match, en ms (`header.t0_ms`). 0 = inconnu. */
  t0Ms: number
  /** Instant courant du rejeu, en ms depuis le début du match. */
  nowMs: number
  scoreboard: MatchScoreboardRow[] | null | undefined
  xuidMeta: XuidMeta
  locale: ReplayLocale
}

export function ReplayKillFeed({ kills, victims, t0Ms, nowMs, scoreboard, xuidMeta, locale }: Props) {
  const t = REPLAY_TEXT[locale]
  // Le recalage et le tri ne dépendent PAS de l'image courante : les refaire soixante fois
  // par seconde coûterait le budget d'animation pour un résultat identique.
  const recales = useMemo(
    () => toReplayKills(attachVictims(kills, victims), t0Ms),
    [kills, victims, t0Ms],
  )
  const colorOf = useMemo(() => teamColorResolver(scoreboard), [scoreboard])
  const visibles = killsAt(recales, nowMs, WINDOW_MS, MAX_LINES)

  return (
    <div className="rounded-lg border border-border bg-card px-3 py-2">
      <div className="flex items-baseline justify-between">
        <span className="text-3xs uppercase tracking-wider text-muted-foreground">
          {t.killFeedTitle}
        </span>
        {/* LE TOTAL EST DIT : un feed vide au début du match ne doit pas se lire comme
            une panne de décodage. */}
        <span className="text-3xs text-muted-foreground">
          {t.killFeedTotal(recales.length)}
        </span>
      </div>
      <ul className="mt-1 flex min-h-[4.5rem] flex-col gap-0.5" aria-live="off">
        {visibles.length === 0 && (
          <li className="text-xs text-muted-foreground">{t.killFeedEmpty}</li>
        )}
        {visibles.map((k) => (
          <FeedLine
            key={`${k.xuid}-${k.replayMs}`}
            kill={k}
            nowMs={nowMs}
            colorOf={colorOf}
            xuidMeta={xuidMeta}
            locale={locale}
          />
        ))}
      </ul>
    </div>
  )
}

/**
 * FeedLine — UNE mort du feed, au format du POC : arme, tueur → victime, horodatage, puis
 * l'ASSISTANCE en dessous quand elle a quelque chose à dire.
 *
 * L'ASSISTANCE NE S'AFFICHE QUE NOMMÉE (décision utilisateur 2026-08-12) : « + Nom
 * (part %) · tueur part % », fond BLEUTÉ sur la ligne — le fond affirme une contribution.
 * « Aucun » (mesuré) garde sa précision en infobulle ; « inconnu » n'écrit RIEN — la
 * plupart des kills n'ont pas d'assistant, marteler « assistant inconnu » ne renseignait
 * personne. Les trois états restent distincts dans la DONNÉE (assist_state).
 */
function FeedLine({
  kill: k,
  nowMs,
  colorOf,
  xuidMeta,
  locale,
}: {
  kill: ReplayKill
  nowMs: number
  colorOf: (teamID: number | null, ally: boolean) => string
  xuidMeta: XuidMeta
  locale: ReplayLocale
}) {
  const t = REPLAY_TEXT[locale]
  const assisted = k.assistState === 'named'
  const lineHint =
    k.assistState === 'none'
      ? k.killerDamagePct != null
        ? `${t.killFeedNoAssistHint} ${t.killFeedKillerShare(k.killerDamagePct)}.`
        : t.killFeedNoAssistHint
      : undefined
  return (
    <li
      className="flex flex-col rounded-sm text-xs"
      style={{
        opacity: freshnessOf(k, nowMs, WINDOW_MS),
        background: assisted ? `color-mix(in srgb, ${tokenCssVar('info')} 10%, transparent)` : undefined,
      }}
      title={lineHint}
    >
      <div className="flex items-center gap-2" style={{ color: colorOf(k.teamID, k.ally) }}>
        {k.weaponImageUrl ? (
          <WeaponIcon
            imageUrl={k.weaponImageUrl}
            tinted={k.weaponTinted}
            label={k.weaponLabel || t.killFeedUnknownWeapon}
            width={ICON_W}
            height={ICON_H}
          />
        ) : (
          /* Repli assumé : la source du dégât n'est pas identifiable sans ambiguïté. */
          <span
            aria-hidden
            className="rounded-full"
            style={{
              width: DOT_PX,
              height: DOT_PX,
              backgroundColor: 'currentColor',
              opacity: 0.7,
              flex: 'none',
            }}
          />
        )}
        <span className="truncate font-medium">
          {displayPlayerName(xuidMeta.get(k.xuid)?.gamertag, k.xuid)}
        </span>
        {k.victimGamertag && (
          <>
            <span aria-hidden className="opacity-60">
              →
            </span>
            <span className="truncate">{k.victimGamertag}</span>
          </>
        )}
        <span className="ml-auto font-mono tabular-nums text-muted-foreground">
          {formatClock(k.replayMs)}
        </span>
      </div>
      {assisted && (
        <div className="flex items-baseline gap-1 pl-7 text-3xs text-muted-foreground" title={t.killFeedAssistHint}>
          <span className="font-semibold" style={{ color: colorOf(k.assistTeamID, k.ally) }}>
            +
          </span>
          <span className="truncate">{k.assistGamertag}</span>
          {k.assistDamagePct != null && (
            <span className="font-mono tabular-nums">{k.assistDamagePct} %</span>
          )}
          {k.killerDamagePct != null && (
            <span className="font-mono tabular-nums opacity-70">
              · {t.killFeedKillerShare(k.killerDamagePct)}
            </span>
          )}
        </div>
      )}
    </li>
  )
}
