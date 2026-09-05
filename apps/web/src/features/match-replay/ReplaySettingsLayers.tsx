/**
 * ReplaySettingsLayers — la section CALQUES du tiroir de réglages, et ce que chaque calque
 * demande au film pour seulement s'afficher.
 *
 * EXTRAITE DE `ReplaySettingsDrawer.tsx` LE 2026-08-29, deuxième extraction du tiroir après
 * la carte de chaleur (2026-08-18) et pour la même raison : le tiroir gagnait une section —
 * la LECTURE (lecture automatique) — et repassait au-dessus du seuil de 500 lignes du dépôt
 * (CLAUDE.md n°5). La règle du dépôt est d'extraire, pas de relever le plafond.
 *
 * LA DÉCOUPE TOMBE SUR UNE FRONTIÈRE NETTE : les calques sont la seule section dont l'affichage
 * dépend de CE QUE LE FILM PORTE, et les cinq interfaces `available` qui portent cette question
 * partent avec elle. Le tiroir les réexporte : sa surface d'appel ne change pas d'un octet.
 *
 * PAS DE COMMANDE QUI NE COMMANDE RIEN — c'est la règle du dépôt, née du bouton Zones : un film
 * sans zone nommée, sans socle publié, sans drapeau, sans couronne, sans crâne n'affiche pas la
 * bascule correspondante. Un interrupteur qui ne fait rien trompe plus qu'il n'informe.
 */
import type { ReactNode } from 'react'

import { SettingsToggle } from './ReplaySettingsToggle'

import { REPLAY_TEXT, type ReplayLocale } from './i18n'

/**
 * Ce que le tiroir sait des POSES d'équipement : les deux bascules, et ce que le film porte.
 *
 * `available` et `unnamedAvailable` suivent la même règle que le bouton Zones — pas de
 * commande qui ne commande rien. Un film sans pose publiée (largeur de bloc non tranchée,
 * ou match sans équipement posé) ne montre pas le calque ; un film dont TOUTES les poses
 * sont nommées ne montre pas la bascule des objets non identifiés.
 */
export interface ReplayPlacementControls {
  available: boolean
  show: boolean
  onToggle: () => void
  unnamedAvailable: boolean
  showUnnamed: boolean
  onToggleUnnamed: () => void
  /**
   * Les objets de PUISSANCE lâchés à la mort. `droppedAvailable` ne pose plus qu'UNE
   * condition : le film en porte au moins un. La garde de mode qui l'annulait en Fiesta a été
   * retirée le 2026-08-20 (elle masquait 26 lâchers réels sur le témoin Fiesta) — la commande
   * s'affiche donc dans tous les modes dès qu'elle a de quoi commander.
   */
  droppedAvailable: boolean
  showDropped: boolean
  onToggleDropped: () => void
}

/**
 * Ce que le tiroir sait des EMPLACEMENTS D'ARME : une bascule, et si le film en porte.
 * `available` suit la même règle — un film sans socle publié (Super Fiesta sur variante
 * Forge : zéro socle mesuré) ne montre pas la bascule.
 */
export interface ReplayWeaponPadControls {
  available: boolean
  show: boolean
  onToggle: () => void
}

/**
 * Ce que le tiroir sait des ARMES AU SOL : une bascule, et si le film en porte.
 *
 *  suit la même règle que les socles — un film dont aucune arme ne tombe (ou un
 * artefact antérieur au schéma 27) ne montre pas la bascule. Le calque est SÉPARÉ de celui des
 * socles et ce n'est pas un doublon : un socle est un LIEU qui réapprovisionne, une arme au sol
 * est un OBJET qui ne revient pas. On peut vouloir l'un sans l'autre.
 */
export interface ReplayGroundWeaponControls {
  available: boolean
  show: boolean
  onToggle: () => void
}

/**
 * Ce que le tiroir sait des DRAPEAUX de capture : une bascule, et si le film en porte.
 * `available` suit la même règle que les zones et les socles — un film qui n'est pas reconnu
 * comme de la capture de drapeau ne publie aucun drapeau, et ne montre donc pas la bascule.
 */
export interface ReplayFlagControls {
  available: boolean
  show: boolean
  onToggle: () => void
}

/** La COURONNE VIP (schéma 22) : un seul calque, allumé par défaut, comme les drapeaux. */
export interface ReplayVipCrownControls {
  available: boolean
  show: boolean
  onToggle: () => void
}

/** Le PORTEUR DU CRÂNE d'Oddball (schéma 23) : un seul calque, allumé par défaut. */
export interface ReplaySkullCarrierControls {
  available: boolean
  show: boolean
  onToggle: () => void
}

/** LA BOMBE d'Assaut (schéma 30) : portée et posée, un seul calque, allumé par défaut. */
export interface ReplayBombCarrierControls {
  available: boolean
  show: boolean
  onToggle: () => void
}

/** Les VÉHICULES (schéma 39) : un seul calque, allumé par défaut. */
export interface ReplayVehicleControls {
  available: boolean
  show: boolean
  onToggle: () => void
}

export interface LayersSectionProps {
  locale: ReplayLocale
  showAim: boolean
  onToggleAim: () => void
  showZones: boolean
  onToggleZones: () => void
  showTrail: boolean
  onToggleTrail: () => void
  zonesAvailable: boolean
  placements: ReplayPlacementControls
  weaponPads: ReplayWeaponPadControls
  groundWeapons: ReplayGroundWeaponControls
  flagCarries: ReplayFlagControls
  vipCrown: ReplayVipCrownControls
  skullCarrier: ReplaySkullCarrierControls
  bombCarrier: ReplayBombCarrierControls
  vehicles: ReplayVehicleControls
}

export function LayersSection({
  locale, showAim, onToggleAim, showZones, onToggleZones,
  showTrail, onToggleTrail, zonesAvailable, placements, weaponPads, groundWeapons, flagCarries,
  vipCrown, skullCarrier, bombCarrier, vehicles,
}: LayersSectionProps) {
  const t = REPLAY_TEXT[locale]
  return (
    <section className="space-y-1">
      <h3 className="text-xs font-medium text-muted-foreground">{t.layers}</h3>
      {/* LA FORME DE LA COMMANDE reste l'interrupteur (2026-08-29, « je préfère un toggle
          plutôt que des boutons ») : un rail, libellé à gauche, état à droite. Ce qui a changé
          le 2026-09-02, c'est la LARGEUR dont il dispose — cf. l'en-tête de `LayerGroup`. */}
      {/* LES JOUEURS : ce qui se dit d'une personne à l'écran. Le calque des NOMS n'y figure
          plus (2026-09-02) — il est toujours allumé. Un nom sous un marqueur n'est pas un
          habillage dont on débat, c'est ce qui rend le rejeu lisible. */}
      <LayerGroup title={t.layerGroupPlayers}>
        <SettingsToggle label={t.layerAim} pressed={showAim} onToggle={onToggleAim} hint={t.layerAimHint} />
        <SettingsToggle
          label={t.layerTrail}
          pressed={showTrail}
          onToggle={onToggleTrail}
          hint={t.layerTrailHint}
        />
      </LayerGroup>

      {/* LE TERRAIN : des lieux et des objets, pas des gens. Les EMPLACEMENTS D'ARME sont une
          récurrence spatiale mesurée ; les ARMES AU SOL sont des objets qui gisent là où ils
          sont tombés — on peut vouloir les socles sans le fouillis, et l'inverse. */}
      {(zonesAvailable || weaponPads.available || groundWeapons.available || vehicles.available) && (
        <LayerGroup title={t.layerGroupTerrain}>
          {zonesAvailable && (
            <SettingsToggle
              label={t.layerZones}
              pressed={showZones}
              onToggle={onToggleZones}
              hint={t.layerZonesHint}
            />
          )}
          {weaponPads.available && (
            <SettingsToggle
              label={t.layerWeaponPads}
              pressed={weaponPads.show}
              onToggle={weaponPads.onToggle}
              hint={t.layerWeaponPadsHint}
            />
          )}
          {groundWeapons.available && (
            <SettingsToggle
              label={t.layerGroundWeapons}
              pressed={groundWeapons.show}
              onToggle={groundWeapons.onToggle}
              hint={t.layerGroundWeaponsHint}
            />
          )}
          {/* LES VÉHICULES sont des OBJETS qui bougent, comme les joueurs — mais ce sont des
              meubles du terrain que le conducteur teinte, pas l'enjeu du mode. Ils rejoignent
              donc le groupe des socles et des armes au sol plutôt que celui des drapeaux. */}
          {vehicles.available && (
            <SettingsToggle
              label={t.layerVehicles}
              pressed={vehicles.show}
              onToggle={vehicles.onToggle}
              hint={t.layerVehiclesHint}
            />
          )}
        </LayerGroup>
      )}

      {/* LES POSES RESTENT EN UNE COLONNE, et c'est la seule exception à la grille. Elles ont
          des bascules FILLES qui n'apparaissent qu'avec elles (objets lâchés, non identifiés) :
          en deux colonnes, une fille se retrouverait à côté de sa mère au lieu d'être dessous,
          et la dépendance — qui est toute l'information — cesserait de se voir. */}
      {placements.available && (
        <div className="flex flex-col gap-0.5">
          <SettingsToggle
            label={t.layerPlacements}
            pressed={placements.show}
            onToggle={placements.onToggle}
            hint={t.layerPlacementsHint}
          />
          {placements.show && placements.droppedAvailable && (
            <SettingsToggle
              label={t.layerPlacementsDropped}
              pressed={placements.showDropped}
              onToggle={placements.onToggleDropped}
              hint={t.layerPlacementsDroppedHint}
            />
          )}
          {placements.show && placements.unnamedAvailable && (
            <SettingsToggle
              label={t.layerPlacementsUnnamed}
              pressed={placements.showUnnamed}
              onToggle={placements.onToggleUnnamed}
              hint={t.layerPlacementsUnnamedHint}
            />
          )}
        </div>
      )}

      {/* LES ENJEUX DU MODE : drapeau, couronne, crâne, bombe. Ils bougent, ils changent de
          main, et leur position EST la lecture du match. LE GROUPE DISPARAÎT EN ENTIER hors des
          modes concernés — sur un Slayer, il ne laisse même pas son titre. C'est le gain caché
          du groupement : une liste plate y laissait un trou qu'on ne savait pas nommer. */}
      {(flagCarries.available ||
        vipCrown.available ||
        skullCarrier.available ||
        bombCarrier.available) && (
        <LayerGroup title={t.layerGroupObjectives}>
          {flagCarries.available && (
            <SettingsToggle
              label={t.layerFlagCarries}
              pressed={flagCarries.show}
              onToggle={flagCarries.onToggle}
              hint={t.layerFlagCarriesHint}
            />
          )}
          {vipCrown.available && (
            <SettingsToggle
              label={t.layerVipCrown}
              pressed={vipCrown.show}
              onToggle={vipCrown.onToggle}
              hint={t.layerVipCrownHint}
            />
          )}
          {skullCarrier.available && (
            <SettingsToggle
              label={t.layerSkullCarrier}
              pressed={skullCarrier.show}
              onToggle={skullCarrier.onToggle}
              hint={t.layerSkullCarrierHint}
            />
          )}
          {bombCarrier.available && (
            <SettingsToggle
              label={t.layerBombCarrier}
              pressed={bombCarrier.show}
              onToggle={bombCarrier.onToggle}
              hint={t.layerBombCarrierHint}
            />
          )}
        </LayerGroup>
      )}
    </section>
  )
}

/**
 * LayerGroup — un sous-titre, puis ses bascules SUR DEUX COLONNES.
 *
 * LES DEUX COLONNES SONT REVENUES LE 2026-09-02, ET IL FAUT DIRE POURQUOI : elles avaient été
 * posées le 2026-08-24 puis RETIRÉES le 2026-08-29, parce qu'un interrupteur se lit sur son
 * rail — libellé à gauche, état à droite — et que deux rails dans les 264 px utiles du tiroir
 * `w-72` tronquaient « Objets lâchés au sol ». Cette raison était juste, et elle a cessé de
 * valoir : le tiroir est passé à 26 rem (~392 px utiles) en devenant un panneau flottant, soit
 * ~190 px par colonne. Ce n'est donc pas un retour en arrière, c'est la même décision prise
 * dans une largeur qui n'est plus la même.
 *
 * LE GROUPEMENT, LUI, EST NEUF, et c'est lui qui paie la place du sous-titre : une liste plate
 * de douze bascules ne dit pas de quoi elle parle, et ses trous — un groupe entier absent hors
 * de son mode — n'y étaient pas lisibles.
 */
function LayerGroup({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="space-y-0.5">
      <p className="text-[11px] font-medium uppercase tracking-[0.06em] text-muted-foreground/70">
        {title}
      </p>
      <div className="grid grid-cols-2 gap-x-3 gap-y-0.5">{children}</div>
    </div>
  )
}
