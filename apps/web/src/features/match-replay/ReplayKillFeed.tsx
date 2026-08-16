/**
 * ReplayKillFeed — le fil des éliminations de la page de rejeu, SYNCHRONISÉ sur
 * l'horloge du rejeu. PORTAGE du fil du POC (spec de rendu) :
 *
 *  - FIL PERMANENT : toutes les lignes depuis le début du match, la plus récente en
 *    tête, dans une liste qui DÉFILE — les événements passés restent lisibles. On ne
 *    remonte en tête que si le lecteur y était déjà : une nouvelle ligne ne lui arrache
 *    pas sa position de lecture (règle du POC).
 *  - CHAQUE LIGNE : TUEUR (couleur de son équipe) — ICÔNE de l'arme, TEINTE de la
 *    couleur d'équipe du tueur (masque + currentColor, la technique de `WeaponIcon`) —
 *    VICTIME (couleur de SON équipe), horodatage à droite ; puis les MÉDAILLES du kill
 *    (image + libellé et description en infobulle), puis l'ASSISTANCE quand elle est
 *    nommée.
 *  - MÉDAILLE SEULE : une médaille sans kill du même acteur à ±500 ms fait sa propre
 *    ligne plutôt que d'être forcée sur un mauvais kill.
 *  - MORT NEUTRE : une fin de vie qu'aucun kill ne revendique (suicide, chute, sortie)
 *    fait une ligne SANS tueur ni arme — comme le jeu affiche un suicide (décision produit
 *    2026-08-13). Elle vient des pistes déjà servies, côté client ; le TYPE de la mort
 *    (chute / sortie de zone, ou sa propre arme) vient du document quand le film l'établit,
 *    et donne son PICTOGRAMME à la ligne. Type non établi = repère neutre, jamais l'icône
 *    d'une autre mort.
 *
 * CE QU'IL RÉUTILISE : `collectKillEvents`/`collectMedalEvents` (lecture des highlight
 * events), `teamColorResolver` (cascade de couleur d'identité du scoreboard),
 * `WeaponIcon` (masque teint par CSS). L'alignement des deux horloges est traité dans
 * `killFeedLogic.ts`, avec sa mesure.
 *
 * L'ASSISTANCE NE S'AFFICHE QUE NOMMÉE (décision utilisateur 2026-08-12) : marque
 * d'assistance, Nom, « - part % », puis la part du tueur ; fond BLEUTÉ sur la ligne — le
 * fond affirme une contribution. « Aucun » (mesuré) garde sa précision en infobulle ;
 * « inconnu » n'écrit RIEN. Les trois états restent distincts dans la DONNÉE (assist_state).
 *
 * AUCUN MOT « ASSISTÉ PAR » (planche du 16/08) : la marque le dit, et elle le dit en
 * pictogramme — le mot coûtait une demi-ligne à chaque élimination assistée.
 */
import { useEffect, useMemo, useRef } from 'react'

import { WeaponIcon } from '@/components/ui/WeaponIcon'
import type { KillEvent } from '@/features/match-view/_momentum'
import { teamColorResolver, type TeamColorResolver } from '@/features/match-view/teamColor'
import type { XuidMeta } from '@/features/match-view/xuidMeta'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { MatchScoreboardRow } from '@/lib/api/types'
import { parseTeamSideID } from '@/lib/halo/teamNames'
import { displayPlayerName } from '@/lib/players/displayName'

import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import {
  buildFeedEntries,
  feedAt,
  type MedalEvent,
  type ReplayDeath,
  type ReplayFeedEntry,
  type ReplayKill,
} from './killFeedLogic'
import { formatClock } from './replayLogic'
import type { ReplayDocumentReady } from './replayNormalize'

/** Gabarit d'une icône d'arme : le format bandeau de l'atlas kill feed. */
const ICON_W = 22
const ICON_H = 9
/** Diamètre du repère servi quand l'arme est inconnue — jamais l'icône d'une autre arme. */
const DOT_PX = 7
/**
 * Côté du pictogramme de TYPE DE MORT. Il ne suit PAS le gabarit bandeau des armes : les
 * pictogrammes du jeu sont quasi carrés (44×42 pour la chute, 32×40 pour le suicide, mesuré
 * à l'extraction) là où une arme fait ~108×36. Les mettre dans le bandeau les réduirait à la
 * hauteur d'une lettre — `contain` ajuste sur la plus petite dimension.
 */
const DEATH_ICON_PX = 12
/** Côté d'un badge de médaille, en px (POC : 15 px pour un raster de 45 px). */
const MEDAL_PX = 15
/** Seuil sous lequel le lecteur est considéré « en tête de fil » (POC : 4 px). */
const AT_TOP_PX = 4

interface Props {
  /** Kills du match, tels que `collectKillEvents` les a lus (horloge gameplay). */
  kills: KillEvent[]
  /** Médailles du match, telles que `collectMedalEvents` les a lues. */
  medals: MedalEvent[]
  /** Offset du countdown pré-match, en ms (`header.t0_ms`). 0 = inconnu. */
  t0Ms: number
  /** Instant courant du rejeu, en ms depuis le début du match. */
  nowMs: number
  /**
   * Le document du rejeu : ses pistes datent les fins de vie, le référentiel sur lequel
   * le fil se recale (cf. killFeedLogic.ts) et dont sortent les lignes de mort neutre.
   * Absent : recalage brut `+t0Ms`, aucune mort neutre.
   */
  doc?: ReplayDocumentReady | null
  scoreboard: MatchScoreboardRow[] | null | undefined
  xuidMeta: XuidMeta
  locale: ReplayLocale
}

export function ReplayKillFeed({ kills, medals, t0Ms, nowMs, doc, scoreboard, xuidMeta, locale }: Props) {
  const t = REPLAY_TEXT[locale]
  // L'assemblage ne dépend PAS de l'image courante : le refaire soixante fois par
  // seconde coûterait le budget d'animation pour un résultat identique.
  const entries = useMemo(
    () => buildFeedEntries(kills, medals, t0Ms, doc),
    [kills, medals, t0Ms, doc],
  )
  const colorOf = useMemo(() => teamColorResolver(scoreboard), [scoreboard])
  // team_id par xuid, pour colorer le défunt d'une mort neutre — la piste ne porte pas
  // l'équipe de la base, le scoreboard si.
  const teamIDByXuid = useMemo(() => {
    const m = new Map<string, number | null>()
    for (const r of scoreboard ?? []) m.set(r.xuid, parseTeamSideID(r.team_side))
    return m
  }, [scoreboard])
  const visibles = feedAt(entries, nowMs)

  // POSITION DE LECTURE (règle du POC) : on ne colle en tête que si le lecteur y était.
  const listRef = useRef<HTMLUListElement>(null)
  const atTopRef = useRef(true)
  const count = visibles.length
  useEffect(() => {
    const el = listRef.current
    if (el && atTopRef.current) el.scrollTop = 0
  }, [count])

  return (
    <div className="rounded-lg border border-border bg-card px-3 py-2">
      <div className="flex items-baseline justify-between">
        <span className="text-3xs uppercase tracking-wider text-muted-foreground">
          {t.killFeedTitle}
        </span>
        {/* LE TOTAL EST DIT : un fil court en début de match ne doit pas se lire comme
            une panne de décodage. */}
        <span className="font-mono text-3xs tabular-nums text-muted-foreground">
          {`${count} / ${entries.length}`}
        </span>
      </div>
      <ul
        ref={listRef}
        onScroll={(e) => {
          atTopRef.current = e.currentTarget.scrollTop <= AT_TOP_PX
        }}
        className="mt-1 flex max-h-64 min-h-[4.5rem] flex-col gap-0.5 overflow-y-auto"
        aria-live="off"
      >
        {count === 0 && <li className="text-xs text-muted-foreground">{t.killFeedEmpty}</li>}
        {visibles.map((entry) => (
          <FeedLine
            key={entry.key}
            entry={entry}
            colorOf={colorOf}
            xuidMeta={xuidMeta}
            teamIDByXuid={teamIDByXuid}
            locale={locale}
          />
        ))}
      </ul>
    </div>
  )
}

/**
 * deathKindLabel traduit l'identifiant STABLE du type de mort publié par le document. Un
 * identifiant que le produit ne connaît pas rend une chaîne vide — jamais l'identifiant brut
 * ni le libellé d'un type voisin : le document peut publier demain un type que cette version
 * du front n'a pas.
 */
function deathKindLabel(kind: string, t: (typeof REPLAY_TEXT)['fr']): string {
  return kind === 'environment' || kind === 'suicide' ? t.killFeedDeathKind[kind] : ''
}

/** allyOf : le camp d'un joueur, lu du scoreboard indexé ; repli fourni par l'appelant. */
function allyOf(xuidMeta: XuidMeta, xuid: string, fallback: boolean): boolean {
  return xuidMeta.get(xuid)?.ally ?? fallback
}

/**
 * FeedLine — UNE ligne du fil, au format du POC : tueur, arme, victime, horodatage,
 * médailles puis assistance en dessous. Le liseré gauche porte la couleur d'équipe de
 * l'acteur (tueur, ou décoré pour une médaille seule) — NEUTRE pour une mort sans tueur.
 */
function FeedLine({
  entry,
  colorOf,
  xuidMeta,
  teamIDByXuid,
  locale,
}: {
  entry: ReplayFeedEntry
  colorOf: TeamColorResolver
  xuidMeta: XuidMeta
  teamIDByXuid: ReadonlyMap<string, number | null>
  locale: ReplayLocale
}) {
  const t = REPLAY_TEXT[locale]
  if (entry.death) {
    return (
      <DeathLine
        death={entry.death}
        replayMs={entry.replayMs}
        colorOf={colorOf}
        xuidMeta={xuidMeta}
        teamIDByXuid={teamIDByXuid}
        t={t}
      />
    )
  }
  const k = entry.kill
  if (!k) {
    // MÉDAILLE SEULE : le décoré et son badge — pas de croix, pas d'arme.
    const m = entry.medal
    if (!m) return null
    const color = colorOf(m.teamID, allyOf(xuidMeta, m.xuid, true))
    return (
      <li
        className="flex flex-col rounded-sm py-0.5 pl-2 text-xs"
        style={{ borderLeft: `3px solid ${color}` }}
      >
        <div className="flex items-center gap-2">
          <span className="truncate font-medium" style={{ color }}>
            {displayPlayerName(m.gamertag || xuidMeta.get(m.xuid)?.gamertag, m.xuid)}
          </span>
          <MedalBadges medals={[m]} />
          <span className="ml-auto font-mono tabular-nums text-muted-foreground">
            {formatClock(entry.replayMs)}
          </span>
        </div>
      </li>
    )
  }
  return <KillLine kill={k} replayMs={entry.replayMs} colorOf={colorOf} xuidMeta={xuidMeta} t={t} />
}

/**
 * DeathLine — une mort SANS tueur crédité (suicide, chute, sortie) : le défunt à la
 * couleur de SON équipe, le mot « mort », l'horodatage — ni arme ni tueur. L'instant est
 * la fin de vie de la piste : le MÊME que le flash de la fiche.
 *
 * À LA PLACE DE L'ARME, LE PICTOGRAMME DU **TYPE** DE MORT quand le film l'établit (chute
 * ou sortie de zone / tué par sa propre arme). Ce n'est pas l'icône d'une arme : sur une
 * ligne sans tueur, montrer une arme donnerait à lire un kill. Le jeu fait la même
 * distinction — son propre fil affiche un pictogramme de suicide.
 *
 * TEINTE : la MÊME technique que les icônes d'arme du fil (`WeaponIcon`, masque + couleur
 * portée par le parent), à l'encre NEUTRE du repère — personne n'a tué, aucune couleur
 * d'équipe n'a de sens ici. Un type non établi garde le repère rond, JAMAIS l'icône d'une
 * autre mort.
 */
function DeathLine({
  death,
  replayMs,
  colorOf,
  xuidMeta,
  teamIDByXuid,
  t,
}: {
  death: ReplayDeath
  replayMs: number
  colorOf: TeamColorResolver
  xuidMeta: XuidMeta
  teamIDByXuid: ReadonlyMap<string, number | null>
  t: (typeof REPLAY_TEXT)['fr']
}) {
  const color = colorOf(teamIDByXuid.get(death.xuid) ?? null, allyOf(xuidMeta, death.xuid, true))
  const kindLabel = deathKindLabel(death.kind, t)
  return (
    <li
      className="flex flex-col rounded-sm py-0.5 pl-2 text-xs"
      style={{ borderLeft: `3px solid ${tokenCssVar('divergent-neutral')}` }}
      title={kindLabel ? `${kindLabel} — ${t.killFeedDeathHint}` : t.killFeedDeathHint}
    >
      <div className="flex items-center gap-2">
        {death.img ? (
          /* LE TYPE DE MORT, à l'encre neutre : personne n'a tué, aucune couleur d'équipe
             n'a de sens sur cette ligne. */
          <WeaponIcon
            imageUrl={death.img}
            tinted={death.tinted}
            label={kindLabel || t.killFeedDeathLabel}
            width={DEATH_ICON_PX}
            height={DEATH_ICON_PX}
            style={{ color: tokenCssVar('divergent-neutral'), flex: 'none' }}
          />
        ) : (
          /* Repère NEUTRE : le type de mort n'est pas établi, on ne montre rien d'autre. */
          <span
            aria-hidden
            className="rounded-full"
            style={{
              width: DOT_PX,
              height: DOT_PX,
              backgroundColor: tokenCssVar('divergent-neutral'),
              opacity: 0.7,
              flex: 'none',
            }}
          />
        )}
        <span className="truncate font-medium" style={{ color }}>
          {displayPlayerName(xuidMeta.get(death.xuid)?.gamertag, death.xuid)}
        </span>
        <span className="text-3xs text-muted-foreground">{t.killFeedDeathLabel}</span>
        <span className="ml-auto font-mono tabular-nums text-muted-foreground">
          {formatClock(replayMs)}
        </span>
      </div>
    </li>
  )
}

/** KillLine — une mort : tueur (sa couleur), arme, victime (SA couleur), médailles, assistance. */
function KillLine({
  kill: k,
  replayMs,
  colorOf,
  xuidMeta,
  t,
}: {
  kill: ReplayKill
  replayMs: number
  colorOf: TeamColorResolver
  xuidMeta: XuidMeta
  t: (typeof REPLAY_TEXT)['fr']
}) {
  const assisted = k.assistState === 'named'
  const killerColor = colorOf(k.teamID, k.ally)
  const lineHint =
    k.assistState === 'none'
      ? k.killerDamagePct != null
        ? `${t.killFeedNoAssistHint} ${t.killFeedKillerShare(k.killerDamagePct)}.`
        : t.killFeedNoAssistHint
      : undefined
  return (
    <li
      className="flex flex-col rounded-sm py-0.5 pl-2 text-xs"
      style={{
        borderLeft: `3px solid ${killerColor}`,
        background: assisted ? `color-mix(in srgb, ${tokenCssVar('info')} 10%, transparent)` : undefined,
      }}
      title={lineHint}
    >
      <div className="flex items-center gap-2">
        <span className="truncate font-medium" style={{ color: killerColor }}>
          {displayPlayerName(xuidMeta.get(k.xuid)?.gamertag, k.xuid)}
        </span>
        {/* L'ARME entre le tueur et la victime — elle remplace la croix (POC). L'icône
            extraite du jeu est un masque teint par currentColor (cf. WeaponIcon) : poser
            la couleur d'équipe du TUEUR ici, c'est la technique du kill feed de la carte
            « Dominance » (MatchKillFeed pose color sur le parent de l'icône). */}
        {k.weaponImageUrl ? (
          <WeaponIcon
            imageUrl={k.weaponImageUrl}
            tinted={k.weaponTinted}
            label={k.weaponLabel || t.killFeedUnknownWeapon}
            width={ICON_W}
            height={ICON_H}
            style={{ color: killerColor }}
          />
        ) : (
          /* Repli assumé : la source du dégât n'est pas identifiable sans ambiguïté. */
          <span
            aria-hidden
            className="rounded-full"
            style={{
              width: DOT_PX,
              height: DOT_PX,
              backgroundColor: killerColor,
              opacity: 0.7,
              flex: 'none',
            }}
          />
        )}
        {k.victimGamertag && (
          <span
            className="truncate"
            style={{ color: colorOf(k.victimTeamID, allyOf(xuidMeta, k.victimXuid, !k.ally)) }}
          >
            {k.victimGamertag}
          </span>
        )}
        <span className="ml-auto font-mono tabular-nums text-muted-foreground">
          {formatClock(replayMs)}
        </span>
      </div>
      {k.medals.length > 0 && (
        <div className="flex flex-wrap items-center gap-1 pl-7">
          <MedalBadges medals={k.medals} />
        </div>
      )}
      {assisted && (
        <div className="flex items-center gap-1 pl-7 text-3xs text-muted-foreground" title={t.killFeedAssistHint}>
          <AssistMark label={t.killFeedAssistMark} color={colorOf(k.assistTeamID, k.ally)} />
          <span className="truncate">{k.assistGamertag}</span>
          {k.assistDamagePct != null && (
            <span className="font-mono tabular-nums">{t.killFeedAssistShare(k.assistDamagePct)}</span>
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

/**
 * AssistMark — LA MARQUE D'ASSISTANCE, en SVG : une flèche qui entre dans un point.
 *
 * POURQUOI PAS L'ICÔNE DU JEU. Elle a été cherchée avant d'être dessinée : ni l'atlas du
 * kill feed (`weapons-assets/{slug}/jeu/index.json`, 168 vignettes — aucune entrée
 * d'assistance), ni les vignettes de HUD (capacités et grenades seulement), ni le jeu de
 * médailles (Halo Infinite n'a pas de médaille d'assistance) ne la portent. Le plan prévoit
 * ce cas : à défaut, un glyphe neutre — et JAMAIS un caractère, qui dépendrait de la police
 * de la machine (c'est le « + » qu'il remplace).
 *
 * CE QU'ELLE DESSINE : une contribution qui aboutit sur la cible. La couleur d'équipe de
 * l'assistant la porte, comme le « + » avant elle — l'assistant reste identifiable à sa
 * couleur avant même qu'on lise son nom.
 */
function AssistMark({ label, color }: { label: string; color: string }) {
  return (
    <span role="img" aria-label={label} title={label} className="inline-flex shrink-0 items-center">
      <svg
        width="11"
        height="11"
        viewBox="0 0 12 12"
        fill="none"
        stroke={color}
        strokeWidth="1.3"
        strokeLinecap="round"
        strokeLinejoin="round"
        aria-hidden="true"
      >
        <path d="M1.2 6h4.2" />
        <path d="M4 4.3 5.8 6 4 7.7" />
        <circle cx="9.4" cy="6" r="1.7" fill={color} stroke="none" />
      </svg>
    </span>
  )
}

/**
 * MedalBadges — les badges de médaille EN IMAGES (POC) : le badge du jeu se reconnaît
 * d'un coup d'œil, le libellé et la description vivent dans l'infobulle où ils ne
 * coûtent rien. PAS d'inversion ni de teinte : une médaille est une pièce EN COULEUR.
 * REPLI : une médaille sans visuel garde son TEXTE — jamais le badge d'une autre.
 */
function MedalBadges({ medals }: { medals: MedalEvent[] }) {
  return (
    <>
      {medals.map((m, i) => {
        const label = m.label || m.name
        const tooltip = m.description ? `${label} — ${m.description}` : label
        return m.imageUrl ? (
          <img
            key={`${m.name}-${m.tMs}-${i}`}
            src={m.imageUrl}
            alt={label}
            title={tooltip}
            width={MEDAL_PX}
            height={MEDAL_PX}
            className="inline-block object-contain"
            loading="lazy"
          />
        ) : (
          <span
            key={`${m.name}-${m.tMs}-${i}`}
            title={tooltip}
            className="rounded-sm border border-border px-1 text-3xs text-muted-foreground"
          >
            {label}
          </span>
        )
      })}
    </>
  )
}
