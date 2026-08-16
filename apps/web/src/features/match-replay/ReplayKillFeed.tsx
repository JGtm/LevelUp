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
 * SA HAUTEUR EST CELLE DE LA RANGÉE, JAMAIS LA SIENNE (mise en page du 2026-08-16) : le fil
 * vit dans une colonne à côté de la carte, et c'est la CARTE qui impose la hauteur. Le
 * composant ne borne donc plus sa liste (`max-h-64` supprimé) ; il occupe ce qu'on lui donne
 * et fait défiler à l'intérieur. Le sens de lecture ne change pas : le plus récent en tête,
 * descendre va chercher les événements anciens.
 *
 * LES MARQUES D'IDENTITÉ (« moi », « ami ») précèdent chaque nom, comme dans les fiches et
 * sur la carte — une seule grammaire sur les trois panneaux (décision D5).
 *
 * L'ASSISTANCE NE S'AFFICHE QUE NOMMÉE (décision utilisateur 2026-08-12) : « + Nom
 * (part %) · tueur part % », fond BLEUTÉ sur la ligne — le fond affirme une contribution.
 * « Aucun » (mesuré) garde sa précision en infobulle ; « inconnu » n'écrit RIEN. Les
 * trois états restent distincts dans la DONNÉE (assist_state).
 */
import { useEffect, useMemo, useRef } from 'react'

import { WeaponIcon } from '@/components/ui/WeaponIcon'
import type { KillEvent } from '@/features/match-view/_momentum'
import { teamColorResolver, type TeamColorResolver } from '@/features/match-view/teamColor'
import type { XuidMeta } from '@/features/match-view/xuidMeta'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { MatchScoreboardRow } from '@/lib/api/types'
import { parseTeamSideID } from '@/lib/halo/teamNames'
import { displayPlayerName, normalizeGamertagKey } from '@/lib/players/displayName'

import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import { MedalBadges } from './MedalBadges'
import { NO_MARKS, type PlayerMarkKind } from './playerMarks'
import { PlayerMark } from './PlayerMark'
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
  /**
   * Marques d'identité par xuid (« moi », « ami ») — la même grammaire que les fiches et la
   * carte. Absentes = aucun glyphe, jamais une marque devinée.
   */
  marks?: ReadonlyMap<string, PlayerMarkKind>
  /**
   * Résolveur de couleur d'équipe. Par défaut celui du scoreboard (cascade d'identité :
   * couleur backend, puis couleur officielle du jeu) — la Match View ne change pas. La page
   * de rejeu, elle, passe le résolveur des tokens d'accessibilité (D1) pour que le fil, les
   * fiches et les points parlent d'une seule voix.
   */
  colorOf?: TeamColorResolver
}

export function ReplayKillFeed({
  kills, medals, t0Ms, nowMs, doc, scoreboard, xuidMeta, locale, marks, colorOf,
}: Props) {
  const t = REPLAY_TEXT[locale]
  // L'assemblage ne dépend PAS de l'image courante : le refaire soixante fois par
  // seconde coûterait le budget d'animation pour un résultat identique.
  const entries = useMemo(
    () => buildFeedEntries(kills, medals, t0Ms, doc),
    [kills, medals, t0Ms, doc],
  )
  const fallbackColorOf = useMemo(() => teamColorResolver(scoreboard), [scoreboard])
  const colorOfTeam = colorOf ?? fallbackColorOf
  // team_id par xuid, pour colorer le défunt d'une mort neutre — la piste ne porte pas
  // l'équipe de la base, le scoreboard si.
  const teamIDByXuid = useMemo(() => {
    const m = new Map<string, number | null>()
    for (const r of scoreboard ?? []) m.set(r.xuid, parseTeamSideID(r.team_side))
    return m
  }, [scoreboard])
  // L'ASSISTANT N'A PAS DE XUID dans l'événement du film : il n'est nommé que par son
  // gamertag (cf. `KillEvent`). Pour qu'il porte la MÊME marque que les autres, les marques
  // sont réindexées par gamertag normalisé — via le scoreboard, la seule table qui porte les
  // deux clés à la fois. Sans lui, l'assistant serait le seul nom du fil sans glyphe.
  const marksByGamertag = useMemo(() => {
    const byName = new Map<string, PlayerMarkKind>()
    for (const r of scoreboard ?? []) {
      const kind = (marks ?? NO_MARKS).get(r.xuid)
      if (kind) byName.set(normalizeGamertagKey(r.gamertag), kind)
    }
    return byName
  }, [scoreboard, marks])
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
    <div className="flex h-full min-h-0 flex-col rounded-lg border border-border bg-card px-3 py-2">
      <div className="flex shrink-0 items-baseline justify-between">
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
        className="mt-1 flex min-h-[4.5rem] flex-1 flex-col gap-0.5 overflow-y-auto"
        aria-live="off"
      >
        {count === 0 && <li className="text-xs text-muted-foreground">{t.killFeedEmpty}</li>}
        {visibles.map((entry) => (
          <FeedLine
            key={entry.key}
            entry={entry}
            colorOf={colorOfTeam}
            xuidMeta={xuidMeta}
            teamIDByXuid={teamIDByXuid}
            marks={marks ?? NO_MARKS}
            marksByGamertag={marksByGamertag}
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
  marks,
  marksByGamertag,
  locale,
}: {
  entry: ReplayFeedEntry
  colorOf: TeamColorResolver
  xuidMeta: XuidMeta
  teamIDByXuid: ReadonlyMap<string, number | null>
  marks: ReadonlyMap<string, PlayerMarkKind>
  marksByGamertag: ReadonlyMap<string, PlayerMarkKind>
  locale: ReplayLocale
}) {
  if (entry.death) {
    return (
      <DeathLine
        death={entry.death}
        replayMs={entry.replayMs}
        colorOf={colorOf}
        xuidMeta={xuidMeta}
        teamIDByXuid={teamIDByXuid}
        marks={marks}
        locale={locale}
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
          <PlayerMark kind={marks.get(m.xuid)} locale={locale} />
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
  return (
    <KillLine
      kill={k}
      replayMs={entry.replayMs}
      colorOf={colorOf}
      xuidMeta={xuidMeta}
      marks={marks}
      marksByGamertag={marksByGamertag}
      locale={locale}
    />
  )
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
  marks,
  locale,
}: {
  death: ReplayDeath
  replayMs: number
  colorOf: TeamColorResolver
  xuidMeta: XuidMeta
  teamIDByXuid: ReadonlyMap<string, number | null>
  marks: ReadonlyMap<string, PlayerMarkKind>
  locale: ReplayLocale
}) {
  const t = REPLAY_TEXT[locale]
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
        <PlayerMark kind={marks.get(death.xuid)} locale={locale} />
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
  marks,
  marksByGamertag,
  locale,
}: {
  kill: ReplayKill
  replayMs: number
  colorOf: TeamColorResolver
  xuidMeta: XuidMeta
  marks: ReadonlyMap<string, PlayerMarkKind>
  marksByGamertag: ReadonlyMap<string, PlayerMarkKind>
  locale: ReplayLocale
}) {
  const t = REPLAY_TEXT[locale]
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
        <PlayerMark kind={marks.get(k.xuid)} locale={locale} />
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
          <>
            <PlayerMark kind={marks.get(k.victimXuid)} locale={locale} />
            <span
              className="truncate"
              style={{ color: colorOf(k.victimTeamID, allyOf(xuidMeta, k.victimXuid, !k.ally)) }}
            >
              {k.victimGamertag}
            </span>
          </>
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
        <div className="flex items-baseline gap-1 pl-7 text-3xs text-muted-foreground" title={t.killFeedAssistHint}>
          <span className="font-semibold" style={{ color: colorOf(k.assistTeamID, k.ally) }}>
            +
          </span>
          <PlayerMark kind={marksByGamertag.get(normalizeGamertagKey(k.assistGamertag))} locale={locale} />
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

