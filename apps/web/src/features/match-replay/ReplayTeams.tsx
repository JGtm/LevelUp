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

import { NO_MARKS, type PlayerMarkKind } from './playerMarks'
import { PlayerMark } from './PlayerMark'
import { ReplayTeamHeader } from './ReplayTeamHeader'
import { activeEquipmentAt } from './equipmentFx'
import { equippedWeapons } from './equippedLogic'
import { ReplayCountersBadge } from './ReplayCountersBadge'
import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import { msToFrames, trackWindow } from './replayLogic'
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
   * Marques d'identité par xuid (« ami » — le glyphe « moi » ne se dessine plus, cf.
   * PlayerMark) — LA MÊME grammaire que la carte et le fil (décision D5). Absentes =
   * aucune fiche marquée, jamais une marque devinée.
   */
  marks?: ReadonlyMap<string, PlayerMarkKind>
  /** Camp de chaque xuid : il donne sa couleur au titre de la colonne (allié / adverse). */
  xuidMeta?: XuidMeta
}

// LA FICHE COMPACTE EST DEVENUE LA FICHE (décision utilisateur du 2026-08-24 : « fiches
// compactes va devenir la seule et unique option ») : la variante longue — zone du joueur,
// deux rangées d'inventaire, munitions des armes rangées — est SUPPRIMÉE avec son réglage.

export function ReplayTeams({
  doc, scoreboard, frame, locale, marks, xuidMeta,
}: ReplayTeamsProps) {
  const t = REPLAY_TEXT[locale]
  const groups = useMemo(
    () => groupByTeam(buildPlayers(doc, scoreboard)),
    [doc, scoreboard],
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
                mark={(marks ?? NO_MARKS).get(p.xuid)}
                scoreTimeline={scoreTimeline}
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
  /** Marque d'identité du joueur (« ami »), ou rien. */
  mark?: PlayerMarkKind
  /** Calque de score du film, déjà passé par la garde d'horloge. */
  scoreTimeline?: ReplayScoreTimelineReady
}

function PlayerCard({ player, doc, frame, presence, vitalityFade, readingFull, flashFrames, locale, mark, scoreTimeline }: PlayerCardProps) {
  const t = REPLAY_TEXT[locale]
  // LES COMPTEURS DU FILM, quand ce joueur est publié. `null` veut dire « pas publié », pas
  // « à zéro » : sur le témoin Slayer 6 joueurs sur 8 en portent, et le mode Oddball n'en
  // publie aucun (0/32 en phase 0). La fiche retombe alors sur les totaux de la BASE, qui
  // valent pour tout le match — c'est ce qu'elle affichait avant ce lot.
  const live = playerCountersAt(scoreTimeline, player.xuid, frame)
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
      {/* HAUTEUR CONSTANTE vivant/mort : les deux zones ci-dessous ont une hauteur FIXE
          dans les DEUX états (h-3.5 puis h-[18px]). La mort remplace le CONTENU d'une
          zone, jamais la zone — une fiche qui change de hauteur fait sauter toute la
          colonne à chaque mort (retour utilisateur du 2026-08-24 : la fiche morte
          paraissait plus haute). `overflow-hidden` est la garantie, pas un ornement. */}
      <div className="flex h-3.5 flex-col justify-center gap-0.5 overflow-hidden">
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
      {/* ARMES ET INVENTAIRE SUR UNE GRILLE À CELLULES FIXES (demande utilisateur du
          2026-08-24) : chaque rangée émet des cellules à largeur constante — deux armes,
          munitions de la main, grenades, capacité — pour que les fiches s'alignent en
          colonnes. `flex-nowrap` + `overflow-hidden` : la rangée ne se replie jamais. */}
      <div className="flex h-[18px] flex-nowrap items-center gap-x-1 overflow-hidden">
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

// La rangée d'armes (arme en main, secondaire, indicateur de swap) vit dans
// ReplayWeaponsRow.tsx — extraite avec sa logique d'infobulles quand ce fichier a franchi
// le seuil de taille du dépôt.
