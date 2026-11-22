/**
 * ReplayFeedLines — LES FORMES DE LIGNE du fil des éliminations : frag (`KillLine`), mort sans
 * tueur (`DeathLine`), médaille seule, entrée/sortie de partie, et les lignes d'événement
 * (objectif, prise d'arme ou de bonus, montée ou destruction de véhicule). `FeedLine` choisit la
 * forme d'une entrée ; le fil (`ReplayKillFeed`) ne fait plus que la liste, le menu et les états vides.
 * Le contrat de chaque forme (une seule ligne de 28 px, pictogrammes `shrink-0`, encre de camp =
 * allégeance du film) est écrit au-dessus de la forme.
 */
import { WeaponIcon } from '@/components/ui/WeaponIcon'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import { displayPlayerName } from '@/lib/players/displayName'

import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { feedLineColumns } from '../model/cotesFeedLine'
import { concernsFollowed } from '../model/feedFollowed'
import { FEED_GRID_BUTTON, FEED_GRID_ROW, FEED_ROW, FEED_ROW_ITEM, feedRowClass, followedAttr } from '../model/feedRow'
import type { ReplayDeath, ReplayFeedEntry, ReplayKill } from '../model/killFeedLogic'
import type { MedalEvent } from '../model/medalEvent'
import { killLineHint } from '../model/killLineHint'
import type { FeedZoneOf } from '../hooks/useFeedZones'
import type { ReplayTeamCascades } from '../hooks/useTeamCascades'
import type { PlayerMarkKind } from '../../../lib/replay/playerMarks'
import { MedalBadges } from './MedalBadges'
import { isObjectiveStateLine, type FeedEventLine } from '../model/feedEventLines'
import { ReplayEventLine } from './ReplayEventLine'
import { ReplayObjectiveLine, type CaptureAssister } from './ReplayObjectiveLine'
import { ReplayObjectiveStateLine } from './ReplayObjectiveStateLine'
import { KillWeapon, KillZone, DOT_PX } from './ReplayKillLineParts'
import { FeedClock } from './ReplayFeedClock'
import { FeedMedals } from './ReplayFeedMedals'
import { PRESS_GUARDS } from './replayPressGuards'
import { FeedName } from './ReplayFeedName'
import { ReplayPresenceLine } from './ReplayPresenceLine'

/**
 * Côté d'un pictogramme quasi carré de l'atlas kill feed — TYPE DE MORT ou ASSISTANCE. Ces
 * vignettes ne suivent PAS le gabarit bandeau des armes (44×42, 32×40, 40×40 mesurés) : les
 * mettre dans le bandeau les réduirait à la hauteur d'une lettre.
 */
const PICTOGRAM_PX = 13

/** Les noms de la feuille par xuid (`XuidMeta` s'y conforme) : ce que le fil en lit, et rien d'autre. */
export type GamertagIndex = ReadonlyMap<string, { gamertag: string }>

/** L'encre de camp d'un nom du fil, désigné par son xuid : son allégeance du film, en tokens. */
export type InkOf = (xuid: string) => string

/**
 * Ce qui rend une ligne de frag CLIQUABLE : si elle est la ligne choisie, et le geste du clic. Absent
 * = ligne en lecture seule (pas de magasin de lecture, ou pas de document où viser une image).
 */
export interface FeedLineFocus {
  selected: boolean
  onSelect: () => void
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

/**
 * Le poids d'un nom : l'acteur d'une ligne est en `font-medium`, un nom qui est celui du joueur mis en avant
 * (`strongXuids`, la fiche ouverte) est en gras, quel que soit son emplacement. `base` : le poids de l'emplacement.
 */
function nameWeight(xuid: string, strongXuids: ReadonlySet<string> | undefined, base?: string): string | undefined {
  return strongXuids?.has(xuid) ? 'font-bold' : base
}

/** Ce que reçoit une ligne du fil : l'entrée, l'encre et les noms, et ce qui rend un frag cliquable. */
interface FeedLineProps {
  entry: ReplayFeedEntry
  inkOf: InkOf
  xuidMeta: GamertagIndex
  marks: ReadonlyMap<string, PlayerMarkKind>
  locale: ReplayLocale
  zoneOf?: FeedZoneOf
  /** Le nom et l'encre d'un camp du film : les lignes de manche et de colline écrivent des camps, pas des joueurs. */
  teams: ReplayTeamCascades
  /** Seule la ligne de FRAG s'en sert : les autres formes de ligne ne sont pas cliquables. */
  focus?: FeedLineFocus
  /**
   * Les clés du joueur dont le nom s'écrit en gras là où il apparaît (acteur d'un frag, victime, mort, médaillé) : la
   * fiche ouverte dans le pied de la colonne. Les lignes d'événement (objectif, prise, véhicule) ne le lisent pas.
   */
  strongXuids?: ReadonlySet<string>
  /**
   * Les clés du joueur que la caméra suit : une ligne qui le concerne (auteur ou cible, `concernsFollowed`) prend le fond
   * léger de suivi, quelle que soit sa forme. Absent ou vide : aucune ligne marquée.
   */
  followedKeys?: ReadonlySet<string>
}

/**
 * FeedLine — UNE ligne du fil, de 28 px : l'élimination est une grille à colonnes fixes (instant, lieu,
 * tueur, arme, victime, médailles), les autres formes une rangée qui s'ouvre sur la même gouttière
 * d'instant.
 *
 * AUCUN LISERÉ DE CAMP : le camp est dit par la couleur du nom du tueur et par celle de sa victime,
 * sur la même ligne. Une ligne n'a ni fond ni bordure.
 */
export function FeedLine({
  entry,
  inkOf,
  xuidMeta,
  marks,
  locale,
  zoneOf,
  teams,
  focus,
  strongXuids,
  followedKeys,
}: FeedLineProps) {
  const followed = followedKeys ? concernsFollowed(entry, followedKeys) : false
  if (entry.event) {
    return (
      <EventFeedLine
        event={entry.event}
        replayMs={entry.replayMs}
        inkOf={inkOf}
        xuidMeta={xuidMeta}
        marks={marks}
        teams={teams}
        locale={locale}
        followed={followed}
      />
    )
  }
  if (entry.presence) {
    // ENTRÉE/SORTIE DE PARTIE (presenceFeed.ts) : même encre que les autres formes de ligne —
    // l'allégeance du film de l'arrivant ou du partant.
    const p = entry.presence
    return (
      <ReplayPresenceLine
        presence={p} replayMs={entry.replayMs} color={inkOf(p.xuid)} mark={marks.get(p.xuid)} locale={locale}
        followed={followed}
      />
    )
  }
  if (entry.death) {
    return (
      <DeathLine
        death={entry.death}
        replayMs={entry.replayMs}
        inkOf={inkOf}
        xuidMeta={xuidMeta}
        marks={marks}
        locale={locale}
        strongXuids={strongXuids}
        followed={followed}
      />
    )
  }
  const k = entry.kill
  if (!k) {
    const m = entry.medal
    if (!m) return null
    return (
      <MedalLine
        medal={m} replayMs={entry.replayMs} color={inkOf(m.xuid)} mark={marks.get(m.xuid)} locale={locale}
        name={displayPlayerName(m.gamertag || xuidMeta.get(m.xuid)?.gamertag, m.xuid)}
        strong={strongXuids?.has(m.xuid) === true} followed={followed}
      />
    )
  }
  return (
    <KillLine
      kill={k}
      replayMs={entry.replayMs}
      inkOf={inkOf}
      xuidMeta={xuidMeta}
      marks={marks}
      locale={locale}
      zone={zoneOf?.(entry.key)}
      carrier={entry.flagCarrierKill === true}
      focus={focus}
      strongXuids={strongXuids}
      followed={followed}
    />
  )
}

/** MÉDAILLE SEULE : le décoré et son badge — pas de croix, pas d'arme. */
function MedalLine({
  medal, replayMs, color, mark, locale, name, strong, followed,
}: {
  medal: MedalEvent
  replayMs: number
  color: string
  mark: PlayerMarkKind | undefined
  locale: ReplayLocale
  name: string
  strong: boolean
  followed: boolean
}) {
  return (
    <li className={feedRowClass(FEED_ROW, followed)} data-followed={followedAttr(followed)}>
      <FeedClock ms={replayMs} />
      <FeedName kind={mark} color={color} locale={locale} className={strong ? 'font-bold' : 'font-medium'} name={name} />
      <MedalBadges medals={[medal]} />
    </li>
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
  inkOf,
  xuidMeta,
  marks,
  locale,
  strongXuids,
  followed,
}: {
  death: ReplayDeath
  replayMs: number
  inkOf: InkOf
  xuidMeta: GamertagIndex
  marks: ReadonlyMap<string, PlayerMarkKind>
  locale: ReplayLocale
  strongXuids?: ReadonlySet<string>
  followed: boolean
}) {
  const t = REPLAY_TEXT[locale]
  const color = inkOf(death.xuid)
  const kindLabel = deathKindLabel(death.kind, t)
  return (
    <li
      className={feedRowClass(FEED_ROW, followed)}
      data-followed={followedAttr(followed)}
      title={kindLabel ? `${kindLabel} — ${t.killFeedDeathHint}` : t.killFeedDeathHint}
    >
      <FeedClock ms={replayMs} />
      {death.img ? (
        /* LE TYPE DE MORT, à l'encre neutre : personne n'a tué, aucune couleur d'équipe
           n'a de sens sur cette ligne. */
        <WeaponIcon
          imageUrl={death.img}
          tinted={death.tinted}
          label={kindLabel || t.killFeedDeathLabel}
          width={PICTOGRAM_PX}
          height={PICTOGRAM_PX}
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
      <FeedName kind={marks.get(death.xuid)} color={color} locale={locale} className={nameWeight(death.xuid, strongXuids, 'font-medium')}
        name={displayPlayerName(xuidMeta.get(death.xuid)?.gamertag, death.xuid)} />
      <span className="shrink-0 text-3xs text-muted-foreground">{t.killFeedDeathLabel}</span>
    </li>
  )
}

/** Ce que reçoit une ligne de frag : `zone` est vide quand elle n'est pas établie, absente quand la carte n'en a aucune. */
interface KillLineProps {
  kill: ReplayKill
  replayMs: number
  inkOf: InkOf
  xuidMeta: GamertagIndex
  marks: ReadonlyMap<string, PlayerMarkKind>
  locale: ReplayLocale
  zone?: string
  /** Vrai quand la victime portait le drapeau : la ligne d'objectif s'est repliée en infobulle. */
  carrier?: boolean
  focus?: FeedLineFocus
  strongXuids?: ReadonlySet<string>
  /** La ligne concerne le joueur suivi par la caméra : fond léger de suivi, cumulable avec la ligne choisie. */
  followed?: boolean
  /**
   * La forme posée sur le plan : ligne NUE (sans médailles), en lecture seule, à l'opacité donnée. Les
   * mêmes colonnes et les mêmes cellules que la ligne du fil ; `focus` et `carrier` n'y ont pas d'effet.
   */
  onPlan?: { opacity: number }
}

/**
 * KillLine — une mort, en grille à colonnes fixes : instant, lieu de la victime (si la carte en a),
 * tueur (son encre), arme, victime (SON encre), médailles. `zone` : le nom de la zone de la victime,
 * vide quand elle n'est pas établie, `undefined` quand la carte n'a aucune zone (la colonne du lieu
 * disparaît, les autres gardent leur alignement). L'assistance et la victime porteuse du drapeau n'ont
 * pas de colonne : elles sont dans l'infobulle de la ligne (`killLineHint`).
 *
 * AVEC `focus`, LA LIGNE EST UN BOUTON : le clic pose la lecture à l'instant du frag et met le tueur
 * et la victime en valeur sur la carte. La ligne choisie se lit par un fond de jeton (`bg-muted`) et
 * `aria-pressed` — la couleur ne porte jamais seule. La souris ne prend pas le focus (`mousedown`
 * annulé) : après un clic, Espace reste « Lecture » ; au clavier, Entrée et Espace activent la ligne,
 * et Espace ne remonte pas au raccourci global de lecture.
 *
 * AVEC `onPlan`, c'est la ligne du fil posé sur le plan (`ReplayFeedOverlay`) : nue, sans médailles ni
 * infobulle, jamais cliquable, estompée à l'opacité donnée.
 */
export function KillLine({
  kill: k,
  replayMs,
  inkOf,
  xuidMeta,
  marks,
  locale,
  zone,
  carrier,
  focus,
  strongXuids,
  followed,
  onPlan,
}: KillLineProps) {
  const t = REPLAY_TEXT[locale]
  const killerColor = inkOf(k.xuid)
  const columns = { gridTemplateColumns: feedLineColumns(zone !== undefined) }
  const content = (
    <>
      <FeedClock ms={replayMs} />
      {zone !== undefined && <KillZone name={zone} locale={locale} />}
      <FeedName kind={marks.get(k.xuid)} color={killerColor} locale={locale} className={nameWeight(k.xuid, strongXuids, 'font-medium')}
        name={displayPlayerName(xuidMeta.get(k.xuid)?.gamertag, k.xuid)} />
      <KillWeapon kill={k} color={killerColor} locale={locale} />
      {k.victimGamertag ? (
        <FeedName kind={marks.get(k.victimXuid)} name={displayPlayerName(k.victimGamertag, k.victimXuid)} locale={locale}
          color={inkOf(k.victimXuid)} className={nameWeight(k.victimXuid, strongXuids)} />
      ) : (
        <span aria-hidden />
      )}
      {!onPlan && <FeedMedals medals={k.medals} locale={locale} />}
    </>
  )
  if (onPlan) {
    return (
      <li className={feedRowClass(FEED_GRID_ROW, followed)} data-followed={followedAttr(followed)} style={{ ...columns, opacity: onPlan.opacity }}>
        {content}
      </li>
    )
  }
  if (!focus) {
    return (
      <li
        className={feedRowClass(FEED_GRID_ROW, followed)} data-followed={followedAttr(followed)} style={columns}
        title={killLineHint(k, carrier === true, t, false)}
      >
        {content}
      </li>
    )
  }
  return (
    <li
      className={feedRowClass(focus.selected ? `${FEED_ROW_ITEM} bg-muted` : FEED_ROW_ITEM, followed)}
      data-selected={focus.selected ? 'true' : undefined}
      data-followed={followedAttr(followed)}
    >
      <button
        type="button"
        className={FEED_GRID_BUTTON}
        style={columns}
        aria-pressed={focus.selected}
        title={killLineHint(k, carrier === true, t, true)}
        {...PRESS_GUARDS}
        onClick={focus.onSelect}
      >
        {content}
      </button>
    </li>
  )
}

/**
 * EventFeedLine — la forme d'une ligne d'ÉVÉNEMENT : l'objectif en bandeau, la prise d'arme ou de
 * bonus et la montée en véhicule en ligne simple. Le joueur est nommé comme partout dans le fil
 * (gamertag de la feuille, repli sur le xuid) et pris à l'encre de son allégeance du film.
 */
function EventFeedLine({
  event,
  replayMs,
  inkOf,
  xuidMeta,
  marks,
  teams,
  locale,
  followed,
}: {
  event: FeedEventLine
  replayMs: number
  inkOf: InkOf
  xuidMeta: GamertagIndex
  marks: ReadonlyMap<string, PlayerMarkKind>
  teams: ReplayTeamCascades
  locale: ReplayLocale
  followed: boolean
}) {
  // MANCHE, CRÂNE, COLLINE : des lignes d'objectif sans acteur nommé par `xuid` (un camp, ou le porteur).
  if (isObjectiveStateLine(event)) {
    return (
      <ReplayObjectiveStateLine
        line={event}
        replayMs={replayMs}
        inkOf={inkOf}
        xuidMeta={xuidMeta}
        marks={marks}
        teams={teams}
        locale={locale}
        followed={followed}
      />
    )
  }
  // UNE DESTRUCTION SANS AUTEUR n'a pas d'acteur : ni nom ni encre de camp (neutre), la phrase seule.
  // Un destructeur BOT n'a pas de xuid : son nom est celui du roster, à l'encre neutre.
  const actor = event.xuid
  const botName = 'destroyed' in event ? event.botName : null
  const shared = {
    replayMs,
    color: actor ? inkOf(actor) : inkOf(''),
    mark: actor ? marks.get(actor) : undefined,
    name: actor ? displayPlayerName(xuidMeta.get(actor)?.gamertag, actor) : botName ? displayPlayerName(botName) : '',
    locale,
    followed,
  }
  if (event.family === 'objectives') {
    const assisters: CaptureAssister[] = (event.assistXuids ?? []).map((xuid) => ({
      key: xuid,
      name: displayPlayerName(xuidMeta.get(xuid)?.gamertag, xuid),
      mark: marks.get(xuid),
    }))
    return <ReplayObjectiveLine line={event} {...shared} assisters={assisters} />
  }
  const killed = 'destroyed' in event ? event.killedXuids : []
  return (
    <ReplayEventLine
      line={event}
      {...shared}
      killedNames={killed.map((xuid) => displayPlayerName(xuidMeta.get(xuid)?.gamertag, xuid))}
    />
  )
}
