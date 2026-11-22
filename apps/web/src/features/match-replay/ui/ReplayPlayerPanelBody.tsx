/**
 * ReplayPlayerPanelBody — LE CONTENU COMMUN DE LA FICHE D'UN JOUEUR, indépendant de ce qui la porte : la colonne du
 * Standard et la fenêtre du Cinéma et du Plein cadre le montent à l'identique. Il ne fait qu'ORDONNER des sections ;
 * chacune est un composant de rendu (`ui/fiche/`) qui reçoit ce qu'il écrit, et tout ce qui se lit vient d'UN hook
 * (`usePlayerInstant`), lu une fois par document et par joueur puis interrogé à chaque image.
 *
 * DE HAUT EN BAS :
 *  - l'IDENTITÉ : pastille de camp, nom, camp, frags / morts / assistances, puis `headEnd` ;
 *  - les TROIS ACTIONS sur la carte — Suivre, Seul, Masquer — en un segment de boutons à bascule (`FicheActions`) ;
 *    leurs exclusions et leur remise à zéro sont dans `playerPanelLogic` ;
 *  - une région qui DÉFILE (par défaut : `scroll: 'internal'`) : « À l'instant » (`ReplayPlayerNow`), « Depuis le
 *    début du match », « Éliminations par arme », « Tirs et dégâts », « Grenades lancées », « Ramassages »,
 *    « Équipement », « Médailles », puis, séparés d'un filet, « Totaux du match » et « Gestes ».
 *
 * UNE SECTION SANS VALEUR N'EXISTE PAS : ni titre, ni texte de remplacement. Un joueur sans ligne de feuille n'a pas de
 * totaux ; un joueur dont les tirs ne sont pas lus en entier n'a ni « Tirs et dégâts », ni score, ni rang.
 *
 * CE QUE LE CORPS NE PORTE PAS : tout ce qui change d'une surface à l'autre — le retour au fil, la fermeture, le
 * sélecteur de joueur, l'écoute d'Échap, le nom accessible de la surface, sa hauteur. Le conteneur donne une colonne
 * flex de hauteur bornée (`min-h-0`) et, s'il veut un bouton à droite de l'identité, le passe par `headEnd`. Le corps
 * rend un fragment : ses trois blocs sont les enfants directs du conteneur. Quand le conteneur fait défiler TOUTE la
 * fiche (`scroll: 'outer'`), la région basse ne défile plus d'elle-même et c'est le conteneur qui défile.
 *
 * Le corps reçoit tout par props. Ses boutons suivent la règle des boutons du rejeu : la souris ne prend pas le
 * focus, Espace reste « Lecture » (`PRESS_GUARDS`). Aucun littéral de couleur : l'encre du camp et les neutres sont
 * des jetons sémantiques.
 */
import { useMemo, type ReactNode } from 'react'

import type { FilmAllegiance } from '@/lib/replay/filmAllegiance'
import { formatClock } from '@/lib/replay/replayLogic'
import type { ReplayDocumentReady } from '@/lib/replay/replayNormalize'
import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import { useReadingAges } from '../hooks/useCardScene'
import { usePlayerInstant } from '../hooks/usePlayerInstant'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { campInk } from '../model/campInk'
import { COTES_FICHE_INSTANT } from '../model/cotesFicheInstant'
import type { ReplayFeedEntry } from '../model/killFeedLogic'
import type { WeaponIconsByKey } from '../model/killSourceIcons'
import { playerCampLabel } from '../model/playerCampLabel'
import { cardName } from '../model/playerCardReadings'
import { playerMatchTotals } from '../model/playerMatchTotals'
import type { PlayerPanelAction } from '../model/playerPanelLogic'
import { displayClockMs, type ReplayWindowBounds } from '../model/replayWindow'
import { FicheActions } from './fiche/FicheActions'
import { FicheDepuisDebut } from './fiche/FicheDepuisDebut'
import { FicheEliminations } from './fiche/FicheEliminations'
import { FicheEquipement } from './fiche/FicheEquipement'
import { FicheGestes, FicheTotaux } from './fiche/FicheGroupes'
import { FicheGrenades, FicheRamassages } from './fiche/FicheLancers'
import { FicheMedailles } from './fiche/FicheMedailles'
import { FicheIdentite } from './fiche/FicheIdentite'
import { FicheTirs } from './fiche/FicheTirs'
import { ReplayPlayerNow } from './ReplayPlayerNow'


export interface ReplayPlayerPanelBodyProps {
  player: ReplayPlayer
  /** Les trois actions et leur état : l'état est tenu par la page (`reducePlayerPanel`). */
  actions: Record<PlayerPanelAction, boolean>
  onToggle: (action: PlayerPanelAction) => void
  doc: ReplayDocumentReady
  /** L'image publiée : tout ce que la fiche écrit est lu à cet instant. */
  frame: number
  /** L'instant courant sur l'axe BRUT du film, et la fenêtre qui le recale pour l'affichage. */
  nowMs: number
  playWindow: ReplayWindowBounds | null
  allegiance: FilmAllegiance
  locale: ReplayLocale
  /** Le lobby du match (`ReplayModel.players`) : le rang, les lancers et l'équipement se lisent parmi eux. */
  players: readonly ReplayPlayer[]
  /** Le fil du rejeu (`ReplayModel.feed`) : éliminations, tirs à la tête et médailles datées. */
  feed: readonly ReplayFeedEntry[]
  /** Les pictogrammes d'arme du titre, par clé : ils nomment les armes que le fil ne nomme pas. */
  weaponIcons: WeaponIconsByKey
  /** Ce que le conteneur pose à droite de l'identité (retour, fermeture) ; rien par défaut. */
  headEnd?: ReactNode
  /**
   * Qui fait défiler la partie basse : le corps lui-même (`'internal'`, défaut : l'identité et les actions restent en
   * place) ou son conteneur (`'outer'` : la région garde sa hauteur naturelle).
   */
  scroll?: 'internal' | 'outer'
}

/** La région basse du corps, selon qui la fait défiler. */
const LOWER_CLASS: Record<NonNullable<ReplayPlayerPanelBodyProps['scroll']>, string> = {
  internal: 'flex min-h-0 flex-1 flex-col overflow-y-auto pr-1',
  outer: 'flex flex-col',
}

export function ReplayPlayerPanelBody({
  player, actions, onToggle, doc, frame, nowMs, playWindow, allegiance, locale,
  players, feed, weaponIcons, headEnd, scroll = 'internal',
}: ReplayPlayerPanelBodyProps) {
  const t = REPLAY_TEXT[locale]
  const name = cardName(player, t)
  const ages = useReadingAges(doc)
  const clock = formatClock(displayClockMs(nowMs, playWindow))
  const view = usePlayerInstant(
    { doc, player, players, feed, presence: ages.presence, matchStartFrame: playWindow?.startFrame ?? null, weaponIcons, locale },
    frame,
  )
  const camp = useMemo(() => playerCampLabel(player, players, t), [player, players, t])
  const totals = useMemo(() => playerMatchTotals(player.board), [player.board])

  return (
    <>
      <FicheIdentite
        name={name}
        camp={camp}
        ink={campInk(allegiance.ofXuid(player.xuid))}
        counters={view.identity}
        surface={scroll}
        locale={locale}
        headEnd={headEnd}
      />
      <FicheActions name={name} actions={actions} onToggle={onToggle} locale={locale} />
      <div className={LOWER_CLASS[scroll]} style={{ gap: COTES_FICHE_INSTANT.ecartSectionsPx[scroll] }}>
        <ReplayPlayerNow player={player} doc={doc} frame={frame} clock={clock} ages={ages} locale={locale} />
        {view.since && <FicheDepuisDebut since={view.since} locale={locale} />}
        {view.kills && <FicheEliminations kills={view.kills} locale={locale} />}
        {view.shots && <FicheTirs shots={view.shots} locale={locale} />}
        {view.grenadesThrown.length > 0 && <FicheGrenades throws={view.grenadesThrown} locale={locale} />}
        {view.pickups && <FicheRamassages pickups={view.pickups} locale={locale} />}
        {view.equipment.length > 0 && <FicheEquipement families={view.equipment} locale={locale} />}
        {view.medals.length > 0 && <FicheMedailles medals={view.medals} locale={locale} />}
        {totals && <FicheTotaux totals={totals} locale={locale} />}
        {view.gestures && <FicheGestes gestures={view.gestures} clock={clock} locale={locale} />}
      </div>
    </>
  )
}
