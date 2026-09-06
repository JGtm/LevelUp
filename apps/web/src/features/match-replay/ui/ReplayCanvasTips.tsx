/**
 * ReplayCanvasTips — LES INFOBULLES DE SURVOL DU CANVAS, réunies derrière une seule balise.
 *
 * POURQUOI CE FICHIER EXISTE (CLAUDE.md n°6, « à la 3e copie, centraliser »). Chaque calque
 * survolable posait dans `ReplayCanvas.tsx` le MÊME bloc — un test de présence, un composant,
 * `locale` et `width` recopiés. Deux copies étaient tolérables ; le drapeau de CTF (schéma 15)
 * fait la troisième. Le canvas porte en outre une dette de taille GELÉE par un seuil
 * (`max-lines` eslint, R5) : toute addition s'y fait par EXTRACTION.
 *
 * CE QUE CE COMPOSANT NE FAIT PAS : décider ce qui est survolé. Chaque hook de calque garde son
 * survol, sa donnée et sa règle ; celui-ci ne fait que les afficher. Il ne connaît donc aucune
 * géométrie, et n'a rien à savoir de l'image courante.
 *
 * PLUSIEURS INFOBULLES PEUVENT COEXISTER, et c'est voulu : les calques se superposent (un socle
 * d'arme sous un drapeau lâché), et masquer l'une au profit de l'autre choisirait à la place du
 * lecteur. Chacune se pose à son propre point.
 */
import { ReplayFlagTip } from './ReplayFlagTip'
import { ReplayPlacementTip } from './ReplayPlacementTip'
import { ReplayWeaponPadTip } from './ReplayWeaponPadTip'
import type { ReplayLocale } from '../i18n/i18n'
import type { ReplayWindowBounds } from '../replayWindow'
import type { FlagHover } from '../layers/useReplayFlagCarries'
import type { PlacementHover } from '../layers/usePlacementHover'
import type { WeaponPadHover } from '../layers/useReplayWeaponPads'

interface ReplayCanvasTipsProps {
  locale: ReplayLocale
  /** Largeur du canvas : elle borne chaque infobulle du côté droit. */
  width: number
  /**
   * La fenêtre de gameplay : la seule infobulle qui DATE quelque chose (le lâcher d'un objet)
   * doit le dire sur l'horloge du match, comme le bandeau et le fil (D-A2).
   */
  playWindow: ReplayWindowBounds | null
  /** Une POSE d'équipement survolée : ce que l'objet est, et qui l'a posé. */
  placement: PlacementHover | null
  /**
   * Le nom du poseur, par slot ET par image — la pose ne porte que son numéro. On le résout à
   * l'INSTANT DE LA POSE (`t0`) : le slot peut appartenir à un autre joueur à l'image courante
   * (manche suivante), et le poseur, lui, était vivant quand il a posé.
   */
  ownerNameOf: (slot: number, frame: number) => string | null
  /** Un EMPLACEMENT D'ARME survolé : l'arme, son état, son cycle s'il est établi. */
  pad: WeaponPadHover | null
  /** Un DRAPEAU de CTF survolé : son camp, son état, son porteur, depuis quand. */
  flag: FlagHover | null
}

export function ReplayCanvasTips({
  locale,
  width,
  playWindow,
  placement,
  ownerNameOf,
  pad,
  flag,
}: ReplayCanvasTipsProps) {
  return (
    <>
      {placement && (
        <ReplayPlacementTip
          locale={locale}
          hover={placement}
          ownerName={ownerNameOf(placement.placement.owner, placement.placement.t0)}
          width={width}
          playWindow={playWindow}
        />
      )}
      {/* JAMAIS qui a pris l'arme : le champ existe au contrat et est renseigne depuis le
          schema 30 quand l'evenement natif date l'occupation, mais cette infobulle ne
          l'affiche pas — son dessin n'a pas ete repense. */}
      {pad && <ReplayWeaponPadTip locale={locale} hover={pad} width={width} />}
      {flag && <ReplayFlagTip locale={locale} hover={flag} width={width} />}
    </>
  )
}
