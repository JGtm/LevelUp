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
import {
  playerCountersAt,
  scoreTimelineOf,
  type ReplayScoreTimelineReady,
} from '@/lib/replay/scoreTimeline'
import type { XuidMeta } from '@/features/match-view/xuidMeta'
import type { MatchScoreboardRow } from '@/lib/api/types'

import { calloutLabel, zoneAt, type CalloutZoneReady } from './calloutsLayer'
import { deadTimeByPlayer, formatDeadTime } from './deadTimeLogic'
import { NO_MARKS, type PlayerMarkKind } from './playerMarks'
import { PlayerMark } from './PlayerMark'
import { ReplayTeamHeader } from './ReplayTeamHeader'
import { activeEquipmentAt } from './equipmentFx'
import { equippedWeapons } from './equippedLogic'
import { ReplayCountersBadge } from './ReplayCountersBadge'
import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import { altitudeAt, msToFrames, positionAt, trackWindow } from './replayLogic'
import type { ReplayDocumentReady } from './replayNormalize'
import { ReplayInventoryRow } from './ReplayInventoryRow'
import { RespawnRow, VitalityBar } from './ReplayVitality'
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
/**
 * Durées CSS des deux animations d'éclat (cf. globals.css) — le délai négatif s'y rapporte.
 * CES DEUX NOMBRES SONT LE MIROIR DE LA FEUILLE DE STYLE : les changer ici sans les changer
 * là-bas désaligne le délai négatif, et l'éclat reprend au mauvais endroit.
 *
 * L'ÉCLAT DE RÉAPPARITION EST PASSÉ DE 0,55 s À 1,2 s le 2026-08-17 (« plus lent l'éclat »,
 * planche du 16/08) : à 0,55 s le vert avait disparu avant qu'on ait fini de lire le nom de
 * la fiche. La mesure ne change pas — c'est toujours l'image de départ de la vie suivante qui
 * date le retour ; seule la durée du repère à l'écran change.
 */
const DEATH_FLASH_TOTAL_S = 1.86
const RESPAWN_FLASH_S = 1.2
/**
 * Effet de VERRE de la fiche pendant un épisode de CAMOUFLAGE actif (cahier des charges
 * Notion, item 21.1) : translucidité + flou léger, PAS une opacité réduite sur toute la
 * fiche — le texte et les icônes restent lisibles, seul le FOND se dépolit.
 * `BLUR_PX` est le flou du verre (`backdrop-filter`, sans effet visible tant que rien de
 * texturé ne passe dessous — la colonne des fiches n'est jamais posée sur la carte, cf.
 * l'en-tête du fichier — mais la technique canonique d'un "verre" CSS reste correcte et
 * bon marché si ce contexte change). `VEIL_PCT` mélange `--foreground` (PAS `--card`) au
 * fond ambiant : teinter le fond avec SA PROPRE couleur serait invisible (0 contraste),
 * `--foreground` est justement le token conçu pour contraster sur `--card` dans les DEUX
 * thèmes — le voile s'éclaircit en sombre, s'assombrit en clair, toujours visible. Le
 * liseré reprend `--border` À PLEINE FORCE (déjà le ton subtil-mais-visible du système,
 * cf. le même token sur le panneau d'équipe) plutôt que de le diluer encore.
 */
const CAMO_GLASS_BLUR_PX = 6
const CAMO_GLASS_VEIL_PCT = 12
/**
 * ENCADRÉ DORÉ de la fiche pendant un épisode de SURBOUCLIER (cahier des charges Notion,
 * item 21.1) : un CADRE, pas un fond teinté — `FRAME_PX` (2 px) est le plus fin qui se
 * lise comme un encadrement plutôt qu'un simple contour de sélection ; `GLOW_PCT` un halo
 * externe discret. Token `legendary` (skill color-tokens) : un surbouclier est un état de
 * jeu rare et précieux, la même famille sémantique que le "légendaire" du Battlepass —
 * PAS le token `info` de la jauge de bouclier (qui, lui, reste bleu : cf. VitalityBar).
 * Aucun fond propre : composé avec le verre du camouflage, le cadre doré ne doit pas
 * écraser la translucidité de l'autre effet (cf. leur composition ci-dessous).
 */
const OVERSHIELD_FRAME_PX = 2
const OVERSHIELD_GLOW_PCT = 60

interface ReplayTeamsProps {
  doc: ReplayDocumentReady
  scoreboard: MatchScoreboardRow[]
  frame: number
  locale: ReplayLocale
  /**
   * Zones nommées de la carte (normalisées par la route, les mêmes que le calque du
   * canvas) : la fiche affiche la ZONE COURANTE du joueur. Vide = pas de ligne remplie —
   * la place reste réservée (hauteur constante).
   */
  callouts?: CalloutZoneReady[]
  /**
   * Marques d'identité par xuid (« moi », « ami ») — LA MÊME grammaire que la carte et le
   * fil (décision D5). Absentes = aucune fiche marquée, jamais une marque devinée.
   */
  marks?: ReadonlyMap<string, PlayerMarkKind>
  /** Camp de chaque xuid : il donne sa couleur au titre de la colonne (allié / adverse). */
  xuidMeta?: XuidMeta
  /**
   * FICHE COMPACTE (B2/R2-7, option du tiroir — la validée reste le défaut). Trois choses
   * changent, et rien d'autre : la ZONE du joueur disparaît, les armes / grenades /
   * équipement se rangent sur UNE ligne, et seule l'arme en main garde ses munitions. Tout
   * le reste — nom, marque, compteurs, vitalité, réapparition, éclats, verre et encadré —
   * est identique : c'est une fiche plus courte, pas une fiche appauvrie.
   */
  compact?: boolean
}

export function ReplayTeams({
  doc, scoreboard, frame, locale, callouts, marks, xuidMeta, compact = false,
}: ReplayTeamsProps) {
  const t = REPLAY_TEXT[locale]
  const groups = useMemo(
    () => groupByTeam(buildPlayers(doc, scoreboard)),
    [doc, scoreboard],
  )
  // LE TEMPS MORT EST UN TOTAL DE MATCH : il se calcule UNE FOIS pour toute la colonne, sur
  // les mêmes joueurs que les fiches, et ne dépend pas de l'image lue. Le recalculer par
  // fiche et par image ferait balayer toutes les vies de tout le monde à chaque frame.
  const deadTime = useMemo(
    () => deadTimeByPlayer(groups.flatMap((g) => g.players), doc),
    [groups, doc],
  )
  const vitalityFade = useMemo(() => msToFrames(VITALITY_FADE_MS, doc), [doc])
  const readingFull = useMemo(() => msToFrames(READING_FULL_MS, doc), [doc])
  const flashFrames = useMemo(() => Math.max(1, msToFrames(FLASH_MS, doc)), [doc])
  const presence = useMemo(() => vitalityPresence(doc), [doc])
  // LE CALQUE DE SCORE PASSE PAR SA GARDE D'HORLOGE, une seule fois pour toute la colonne :
  // absent = artefact antérieur au schéma 12, mode sans compteur, ou origine non recalée
  // (cf. lib/replay/scoreTimeline.filmClockTrusted). Les fiches et les en-têtes n'ont alors
  // de plus à dire qu'avant — aucune ligne ne se vide, aucun zéro n'apparaît.
  const scoreTimeline = useMemo(() => scoreTimelineOf(doc), [doc])

  if (groups.length === 0) {
    return (
      <div className="rounded-lg border border-border bg-card p-3">
        <p className="text-xs text-muted-foreground">{t.rosterEmpty}</p>
      </div>
    )
  }

  return (
    // LA HAUTEUR VIENT DE LA RANGÉE, JAMAIS DES FICHES (technique du POC) : `h-full min-h-0`
    // laisse la colonne se rétrécir, et le défilement vit À L'INTÉRIEUR de chaque carte. Hors
    // rangée (repli étroit), aucune hauteur n'est imposée : rien ne défile, comportement
    // d'origine.
    <div
      className="grid h-full min-h-0 gap-2"
      style={{ gridTemplateColumns: `repeat(${groups.length}, 1fr)` }}
    >
      {groups.map((group, gi) => (
        <div
          key={group.side ?? `sans-equipe-${gi}`}
          className="flex h-full min-h-0 flex-col rounded-lg border border-border bg-card p-2"
        >
          <ReplayTeamHeader
            players={group.players}
            side={group.side}
            xuidMeta={xuidMeta}
            locale={locale}
            scoreTimeline={scoreTimeline}
            frame={frame}
          />
          <div className="flex min-h-0 flex-1 flex-col overflow-y-auto">
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
                callouts={callouts}
                mark={(marks ?? NO_MARKS).get(p.xuid)}
                scoreTimeline={scoreTimeline}
                deadTimeMs={deadTime.get(p.xuid) ?? null}
                compact={compact}
              />
            ))}
          </div>
        </div>
      ))}
    </div>
  )
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
  callouts?: CalloutZoneReady[]
  /** Marque d'identité du joueur (« moi », « ami »), ou rien. */
  mark?: PlayerMarkKind
  /** Calque de score du film, déjà passé par la garde d'horloge. */
  scoreTimeline?: ReplayScoreTimelineReady
  /**
   * TEMPS MORT CUMULÉ du joueur sur tout le match, en millisecondes (`deadTimeLogic`), ou
   * `null` quand le film ne permet pas de le mesurer. Calculé une fois pour la colonne : la
   * fiche l'affiche, elle ne le mesure pas — et elle ne remplace jamais `null` par zéro.
   */
  deadTimeMs: number | null
  /** Fiche COMPACTE (cf. ReplayTeamsProps.compact). */
  compact?: boolean
}

function PlayerCard({ player, doc, frame, presence, vitalityFade, readingFull, flashFrames, locale, callouts, mark, scoreTimeline, deadTimeMs, compact = false }: PlayerCardProps) {
  const t = REPLAY_TEXT[locale]
  // LES COMPTEURS DU FILM, quand ce joueur est publié. `null` veut dire « pas publié », pas
  // « à zéro » : sur le témoin Slayer 6 joueurs sur 8 en portent, et le mode Oddball n'en
  // publie aucun (0/32 en phase 0). La fiche retombe alors sur les totaux de la BASE, qui
  // valent pour tout le match — c'est ce qu'elle affichait avant ce lot.
  const live = playerCountersAt(scoreTimeline, player.xuid, frame)
  const state = playerStateAt(player, frame, presence)
  const name = playerName(player) ?? t.unknownPlayer
  const equipped = state.life ? equippedWeapons(doc, state.life.slot, frame) : null
  // La ZONE COURANTE, affectée en 3D (zoneAt) : les étages d'une même zone se confondent
  // en 2D — c'est le z qui départage « Fer à cheval » de sa version inférieure.
  // La ZONE ne se calcule pas en fiche compacte : elle n'y est pas affichée.
  const zone = !compact && state.alive && state.life && callouts?.length
    ? currentZone(callouts, state.life, frame)
    : null
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
  let flashClass = ''
  const style: CSSProperties = {}
  if (!state.alive) {
    style.boxShadow = `inset 2px 0 0 ${tokenCssVar('destructive')}`
    style.background = `color-mix(in srgb, ${tokenCssVar('destructive')} 12%, transparent)`
    if (deathAge >= 0 && deathAge <= flashFrames) {
      flashClass = 'replay-flash-death'
      style.animationDelay = `${(-(deathAge / flashFrames) * DEATH_FLASH_TOTAL_S).toFixed(3)}s`
    }
  } else {
    if (lifeAge >= 0 && lifeAge <= flashFrames) {
      flashClass = 'replay-flash-respawn'
      style.animationDelay = `${(-(lifeAge / flashFrames) * RESPAWN_FLASH_S).toFixed(3)}s`
    }
    // Le camouflage rend la fiche VITREUSE (translucidité + flou) ; le surbouclier
    // l'ENCADRE d'or. Les deux peuvent se composer (états indépendants) : les ombres
    // s'ACCUMULENT (box-shadow accepte plusieurs couches) plutôt que de s'écraser, et
    // seul le camouflage pose un fond — le cadre doré n'en a pas de propre — pour
    // qu'une fiche vitreuse ET encadrée dise exactement ce que l'écran de jeu montre.
    const shadows: string[] = []
    if (equipment?.camo) {
      style.backdropFilter = `blur(${CAMO_GLASS_BLUR_PX}px)`
      style.WebkitBackdropFilter = style.backdropFilter
      style.background = `color-mix(in srgb, var(--foreground) ${CAMO_GLASS_VEIL_PCT}%, var(--card))`
      shadows.push('inset 0 0 0 1px var(--border)')
    }
    if (equipment?.overshield) {
      shadows.push(`inset 0 0 0 ${OVERSHIELD_FRAME_PX}px ${tokenCssVar('legendary')}`)
      shadows.push(`0 0 10px color-mix(in srgb, ${tokenCssVar('legendary')} ${OVERSHIELD_GLOW_PCT}%, transparent)`)
    }
    if (shadows.length > 0) {
      style.boxShadow = shadows.join(', ')
    }
  }
  const equipTitle = [
    equipment?.camo ? t.equipmentActive.camo : null,
    equipment?.overshield ? t.equipmentActive.overshield : null,
  ].filter(Boolean).join(' · ')
  return (
    <div
      className={`flex flex-col gap-0.5 border-t border-border py-1 first:border-t-0 ${flashClass}`}
      style={style}
      title={equipTitle || undefined}
    >
      <div className="flex items-baseline gap-1.5">
        {/* LA MARQUE AVANT LE NOM, et elle n'en change pas la couleur : le nom dit déjà
            l'équipe ou la mort, la marque dit l'identité (décision D5). Le glyphe fait 10 px
            dans une ligne de 11,5 px : la hauteur de la fiche ne bouge pas. Il reste FRÈRE du
            nom, jamais son parent — la fiche compte ses rangées par la structure. */}
        <PlayerMark kind={mark} locale={locale} />
        <span
          className="min-w-0 flex-1 truncate text-[11.5px] font-medium"
          style={state.alive ? undefined : { color: tokenCssVar('destructive') }}
          title={name}
        >
          {name}
        </span>
        <ReplayCountersBadge board={player.board} live={live} locale={locale} />
      </div>
      {/* La zone courante RÉSERVE sa ligne dès que la carte a des zones : vivant elle se
          remplit, mort elle reste vide — jamais une fiche qui se compacte (règle 1.1). */}
      {!compact && (callouts?.length ?? 0) > 0 && (
        <div className="flex h-3 items-center">
          {zone && (
            <span
              className="truncate font-mono text-[9px] uppercase tracking-wide text-muted-foreground"
              title={t.zoneLabel}
            >
              {calloutLabel(zone, locale)}
            </span>
          )}
        </div>
      )}
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
      {/* ARMES ET INVENTAIRE : deux rangées réservées en fiche validée, UNE SEULE en
          compacte. `flex-nowrap` + `overflow-hidden` y sont la règle, pas de la mise en
          forme — sans eux la rangée unique se replierait en deux, et la fiche compacte
          serait plus haute que la validée (leçon C1 du même jour). */}
      <div
        className={
          compact
            ? 'flex min-h-[18px] flex-nowrap items-center gap-x-2 overflow-hidden'
            : 'flex min-h-[18px] items-center'
        }
      >
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
        {compact && state.alive && state.life && (
          <ReplayInventoryRow
            doc={doc}
            slot={state.life.slot}
            equipped={equipped}
            frame={frame}
            readingFull={readingFull}
            locale={locale}
            compact
          />
        )}
      </div>
      {!compact && (
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
      )}
      {!compact && <DeadTimeRow ms={deadTimeMs} locale={locale} />}
    </div>
  )
}

/**
 * DeadTimeRow — LE TEMPS MORT CUMULÉ du joueur, en dernière ligne de la fiche.
 *
 * EN BAS, ET C'EST UN CHOIX : c'est la seule ligne de la fiche qui ne change pas avec la
 * lecture. La glisser entre la zone, la vitalité et l'inventaire couperait le bloc de l'état
 * courant par un total de match ; en pied de fiche, elle se lit pour ce qu'elle est.
 *
 * ELLE S'AFFICHE TOUJOURS, ET DIT L'UN DES DEUX : une mesure — « 00:00 » compris, un joueur
 * mort zéro fois est une information — ou son REFUS, écrit d'un tiret, quand le film ne permet
 * pas de conclure (`deadTimeLogic` : une vie non rattachée à un joueur vit dans l'un de ses
 * trous, ou bien il n'a aucune vie publiée). Les deux cas occupent la même hauteur, et aucun ne
 * dépend de l'état vital : la parité de rangées morte/vivante tient (règle 1.1 des fiches).
 *
 * LE REFUS N'EST PAS UNE PANNE D'AFFICHAGE, et l'infobulle le dit en toutes lettres. Sans elle,
 * un tiret se lirait comme un bogue ; avec elle, il se lit comme ce qu'il est — le film manque
 * d'un rattachement, et on préfère le taire que de l'inventer.
 *
 * ABSENTE DE LA FICHE COMPACTE (décision de lot, 2026-08-24) : la compacte n'a pas de rangée
 * libre, et son objet est d'être PLUS COURTE — lui ajouter une ligne annulerait exactement ce
 * qu'elle gagne. La déporter sur la rangée du nom aurait serré le gamertag tronqué contre les
 * compteurs, dans la mise en page la plus étroite du rejeu.
 */
function DeadTimeRow({ ms, locale }: { ms: number | null; locale: ReplayLocale }) {
  const t = REPLAY_TEXT[locale]
  return (
    <div className="flex h-3 items-center">
      <span
        className="truncate font-mono text-[9px] uppercase tracking-wide text-muted-foreground"
        title={ms === null ? t.deadTimeUnmeasurable : t.deadTimeLabel}
      >
        {t.deadTimeLabel} <span className="tabular-nums">{formatDeadTime(ms)}</span>
      </span>
    </div>
  )
}

/**
 * currentZone affecte le joueur à sa zone nommée, à l'instant lu : position interpolée
 * de la vie courante (x, y ET z) puis plus-proche-centre 3D (zoneAt, portage POC).
 * Position non transmise = pas de zone — on n'affecte pas quelqu'un qu'on ne situe pas.
 */
function currentZone(
  zones: CalloutZoneReady[],
  life: NonNullable<PlayerState['life']>,
  frame: number,
): CalloutZoneReady | null {
  const p = positionAt(life.points, frame)
  if (!p) return null
  const z = altitudeAt(life.points, frame)
  return zoneAt(zones, p.x, p.y, z ?? 0)
}

// La rangée d'armes (arme en main, secondaire, indicateur de swap) vit dans
// ReplayWeaponsRow.tsx — extraite avec sa logique d'infobulles quand ce fichier a franchi
// le seuil de taille du dépôt.
