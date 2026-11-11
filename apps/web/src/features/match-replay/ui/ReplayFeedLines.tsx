/**
 * ReplayFeedLines — LES FORMES DE LIGNE du fil des éliminations : frag (`KillLine`), mort sans
 * tueur (`DeathLine`), médaille seule, entrée/sortie de partie, et les lignes d'événement
 * (objectif, prise d'arme ou de bonus, montée ou destruction de véhicule). `FeedLine` choisit la
 * forme d'une entrée ; le fil (`ReplayKillFeed`) ne fait plus que la liste, le menu et les états vides.
 *
 * EXTRAIT DE `ReplayKillFeed.tsx` (lot 1 du plan « maquettes du rejeu »), sans changement de
 * rendu : le fichier était au plafond de taille et le menu de filtre s'ajoute à côté. Le contrat
 * de chaque forme (une seule ligne, pictogrammes `shrink-0`, assistance nommée seulement, encre
 * de camp = allégeance du film) est écrit au-dessus de la forme.
 */
import { WeaponIcon } from '@/components/ui/WeaponIcon'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import { displayPlayerName } from '@/lib/players/displayName'

import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { FEED_ROW, FEED_ROW_BUTTON, FEED_ROW_ITEM } from '../model/feedRow'
import type { ReplayDeath, ReplayFeedEntry, ReplayKill } from '../model/killFeedLogic'
import type { KillerZoneOf } from '../hooks/useKillerZones'
import type { PlayerMarkKind } from '../../../lib/replay/playerMarks'
import { MedalBadges } from './MedalBadges'
import type { FeedEventLine } from '../model/feedEventLines'
import { ReplayEventLine } from './ReplayEventLine'
import { ReplayObjectiveLine } from './ReplayObjectiveLine'
import { KillAssist, KillWeapon, KillZone, DOT_PX } from './ReplayKillLineParts'
import { FeedClock } from './ReplayFeedClock'
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
 * FeedLine — UNE ligne du fil, au format du POC : tueur, arme, victime, horodatage,
 * médailles puis assistance en dessous.
 *
 * PLUS DE LISERÉ GAUCHE (demande utilisateur du 2026-08-25, la même que pour la fiche morte) :
 * les trois formes de ligne portaient un trait de 3 px à la couleur d'équipe de leur acteur.
 * Le camp est DÉJÀ dit par la couleur du nom du tueur et par celle de sa victime, sur la même
 * ligne — le liseré en était la troisième copie, et il empilait des barres colorées sur toute
 * la hauteur d'un fil qui défile.
 */
export function FeedLine({
  entry,
  inkOf,
  xuidMeta,
  marks,
  marksByGamertag,
  locale,
  zoneOf,
  focus,
}: {
  entry: ReplayFeedEntry
  inkOf: InkOf
  xuidMeta: GamertagIndex
  marks: ReadonlyMap<string, PlayerMarkKind>
  marksByGamertag: ReadonlyMap<string, PlayerMarkKind>
  locale: ReplayLocale
  zoneOf?: KillerZoneOf
  /** Seule la ligne de FRAG s'en sert : les autres formes de ligne ne sont pas cliquables. */
  focus?: FeedLineFocus
}) {
  if (entry.event) {
    return (
      <EventFeedLine
        event={entry.event}
        replayMs={entry.replayMs}
        inkOf={inkOf}
        xuidMeta={xuidMeta}
        marks={marks}
        locale={locale}
      />
    )
  }
  if (entry.presence) {
    // ENTRÉE/SORTIE DE PARTIE (presenceFeed.ts) : même encre que les autres formes de ligne —
    // l'allégeance du film de l'arrivant ou du partant.
    const p = entry.presence
    return (
      <ReplayPresenceLine
        presence={p}
        replayMs={entry.replayMs}
        color={inkOf(p.xuid)}
        mark={marks.get(p.xuid)}
        locale={locale}
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
      />
    )
  }
  const k = entry.kill
  if (!k) {
    // MÉDAILLE SEULE : le décoré et son badge — pas de croix, pas d'arme.
    const m = entry.medal
    if (!m) return null
    const color = inkOf(m.xuid)
    return (
      <li className={FEED_ROW}>
        <FeedClock ms={entry.replayMs} />
        <FeedName kind={marks.get(m.xuid)} color={color} locale={locale} className="font-medium"
          name={displayPlayerName(m.gamertag || xuidMeta.get(m.xuid)?.gamertag, m.xuid)} />
        <MedalBadges medals={[m]} />
      </li>
    )
  }
  return (
    <KillLine
      kill={k}
      replayMs={entry.replayMs}
      inkOf={inkOf}
      xuidMeta={xuidMeta}
      marks={marks}
      marksByGamertag={marksByGamertag}
      locale={locale}
      zone={zoneOf?.(entry.key)}
      focus={focus}
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
  inkOf,
  xuidMeta,
  marks,
  locale,
}: {
  death: ReplayDeath
  replayMs: number
  inkOf: InkOf
  xuidMeta: GamertagIndex
  marks: ReadonlyMap<string, PlayerMarkKind>
  locale: ReplayLocale
}) {
  const t = REPLAY_TEXT[locale]
  const color = inkOf(death.xuid)
  const kindLabel = deathKindLabel(death.kind, t)
  return (
    <li
      className={FEED_ROW}
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
      <FeedName kind={marks.get(death.xuid)} color={color} locale={locale} className="font-medium"
        name={displayPlayerName(xuidMeta.get(death.xuid)?.gamertag, death.xuid)} />
      <span className="shrink-0 text-3xs text-muted-foreground">{t.killFeedDeathLabel}</span>
    </li>
  )
}

/**
 * KillLine — une mort : zone du tueur (si la carte en a), tueur (son encre), arme, victime (SON
 * encre), médailles, assistance. `zone` : le nom de la zone du tueur, vide quand elle n'est pas
 * établie, `undefined` quand la carte n'a aucune zone (pas de gouttière).
 *
 * AVEC `focus`, LA LIGNE EST UN BOUTON (plan « maquettes du rejeu », lot 2) : le clic pose la
 * lecture à l'instant du frag et met le tueur et la victime en valeur sur la carte. La ligne choisie
 * se lit par un fond de jeton (`bg-muted`) ET deux filets aux couleurs de camp du tueur (à gauche) et de la
 * victime (à droite) — la couleur ne porte jamais seule. La souris ne prend pas le focus
 * (`mousedown` annulé) : après un clic, Espace reste « Lecture » ; au clavier, Entrée et Espace
 * activent la ligne, et Espace ne remonte pas au raccourci global de lecture.
 */
function KillLine({
  kill: k,
  replayMs,
  inkOf,
  xuidMeta,
  marks,
  marksByGamertag,
  locale,
  zone,
  focus,
}: {
  kill: ReplayKill
  replayMs: number
  inkOf: InkOf
  xuidMeta: GamertagIndex
  marks: ReadonlyMap<string, PlayerMarkKind>
  marksByGamertag: ReadonlyMap<string, PlayerMarkKind>
  locale: ReplayLocale
  zone?: string
  focus?: FeedLineFocus
}) {
  const t = REPLAY_TEXT[locale]
  const assisted = k.assistState === 'named'
  const killerColor = inkOf(k.xuid)
  const lineHint =
    k.assistState === 'none'
      ? k.killerDamagePct != null
        ? `${t.killFeedNoAssistHint} ${t.killFeedKillerShare(k.killerDamagePct)}.`
        : t.killFeedNoAssistHint
      : undefined
  const assistTint = assisted ? `color-mix(in srgb, ${tokenCssVar('stat-assists')} 9%, transparent)` : undefined
  const content = (
    <>
      <FeedClock ms={replayMs} />
      {zone !== undefined && <KillZone name={zone} locale={locale} />}
      <FeedName kind={marks.get(k.xuid)} color={killerColor} locale={locale} className="font-medium"
        name={displayPlayerName(xuidMeta.get(k.xuid)?.gamertag, k.xuid)} />
      <KillWeapon kill={k} color={killerColor} locale={locale} />
      {k.victimGamertag && (
        <FeedName kind={marks.get(k.victimXuid)} name={displayPlayerName(k.victimGamertag, k.victimXuid)} locale={locale}
          color={inkOf(k.victimXuid)} />
      )}
      {/* LES MÉDAILLES DANS LA RANGÉE, plus en dessous : `shrink-0` parce qu'un badge rogné
          ne se reconnaît plus, alors qu'un nom tronqué se lit encore. */}
      {k.medals.length > 0 && (
        <span className="flex shrink-0 items-center gap-1">
          <MedalBadges medals={k.medals} />
        </span>
      )}
      {assisted && <KillAssist kill={k} marksByGamertag={marksByGamertag} locale={locale} />}
    </>
  )
  if (!focus) {
    return (
      <li className={FEED_ROW} style={{ background: assistTint }} title={lineHint}>
        {content}
      </li>
    )
  }
  return (
    <li
      className={focus.selected ? `${FEED_ROW_ITEM} bg-muted` : FEED_ROW_ITEM}
      data-selected={focus.selected ? 'true' : undefined}
      style={{
        background: focus.selected ? undefined : assistTint,
        boxShadow: focus.selected
          ? `inset 3px 0 ${killerColor}, inset -3px 0 ${inkOf(k.victimXuid)}`
          : undefined,
      }}
    >
      <button
        type="button"
        className={FEED_ROW_BUTTON}
        aria-pressed={focus.selected}
        title={lineHint ? `${t.killFeedFocusHint} — ${lineHint}` : t.killFeedFocusHint}
        onMouseDown={(e) => e.preventDefault()}
        onKeyDown={(e) => {
          if (e.key === ' ') e.stopPropagation()
        }}
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
  locale,
}: {
  event: FeedEventLine
  replayMs: number
  inkOf: InkOf
  xuidMeta: GamertagIndex
  marks: ReadonlyMap<string, PlayerMarkKind>
  locale: ReplayLocale
}) {
  // UNE DESTRUCTION SANS AUTEUR n'a pas d'acteur : ni nom ni encre de camp (neutre), la phrase seule.
  const actor = event.xuid
  const shared = {
    replayMs,
    color: actor ? inkOf(actor) : inkOf(''),
    mark: actor ? marks.get(actor) : undefined,
    name: actor ? displayPlayerName(xuidMeta.get(actor)?.gamertag, actor) : '',
    locale,
  }
  if (event.family === 'objectives') return <ReplayObjectiveLine line={event} {...shared} />
  const occupant = 'destroyed' in event ? event.occupantXuid : null
  return (
    <ReplayEventLine
      line={event}
      {...shared}
      occupantName={occupant ? displayPlayerName(xuidMeta.get(occupant)?.gamertag, occupant) : undefined}
    />
  )
}
