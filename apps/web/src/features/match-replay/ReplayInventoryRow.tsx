/**
 * ReplayInventoryRow — grenades portées, capacité d'armure, munitions d'une fiche joueur.
 * Extrait de ReplayTeams.tsx (seuil de taille du dépôt), comme ReplayWeaponsRow avant lui :
 * la ligne d'inventaire et ses cellules de munitions forment une unité.
 *
 * TOUT CE QUI N'EST PAS LU S'AFFICHE COMME LACUNE, jamais comme une valeur par défaut :
 * - une capacité hors table garde son NUMÉRO, marquée non interprétable — la table est
 *   partielle (4 index observés pour 11 capacités), et la combler se lirait comme une certitude ;
 * - un compteur d'utilisations n'est jamais affiché : il n'est pas localisé dans le film
 *   (36 006 positions testées, aucune ne reproduit le relevé) ;
 * - un emplacement dont le film n'écrit RIEN affiche « aucune » : pour une arme à charge,
 *   cela veut dire PLEIN, le plein étant la valeur par défaut d'un flux différentiel.
 *
 * L'ensemble pâlit avec l'âge de la lecture, comme les armes portées et pour la même raison.
 */
import { WeaponIcon } from '@/components/ui/WeaponIcon'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'

import { catalogText, type CatalogLabel } from './catalogLabel'
import type { EquippedReading } from './equippedLogic'
import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import { formatSeconds, frameToMs, freshness, READING_FADE } from './replayLogic'
import type { ReplayDocumentReady } from './replayNormalize'
import { grenadesCarried, inventoryAt, selectedGrenade } from './rosterLogic'

/** Boîte d'une vignette de HUD (grenade, capacité) : la hauteur de la ligne. */
const HUD_ICON_PX = 16

export function ReplayInventoryRow({
  doc,
  slot,
  equipped,
  frame,
  readingFull,
  locale,
}: {
  doc: ReplayDocumentReady
  slot: number
  equipped: EquippedReading | null
  frame: number
  readingFull: number
  locale: ReplayLocale
}) {
  const t = REPLAY_TEXT[locale]
  const read = inventoryAt(doc, slot, frame)
  if (!read) return null
  const { state } = read
  const grenades = grenadesCarried(state, doc.grenadeLabels, locale)
  const selected = selectedGrenade(state)
  const ability = abilityText(doc, state.a, t.abilityUnknown, locale)
  const ammo = state.am ?? []
  if (grenades.length === 0 && !ability && ammo.length === 0) return null

  // LA LIGNE DES MUNITIONS SUIT L'ORDRE DES ARMES AU-DESSUS (l'arme dégainée d'abord) : les
  // deux lignes partagent la même lecture d'ordre, sinon chaque cellule se rattacherait à la
  // mauvaise arme. Le numéro d'emplacement lève l'ambiguïté quand l'ordre bascule.
  const ammoOrder = (equipped?.order ?? ammo.map((_, i) => i)).filter((i) => i < ammo.length)
  // Âge NÉGATIF = lecture d'une image-clé À VENIR (début de vie, cf. inventoryAt) :
  // l'infobulle le dit — l'estompage porte déjà sur la valeur absolue.
  const ageMs = formatSeconds(frameToMs(Math.abs(read.age), doc))
  const ageTitle = read.age < 0 ? `${t.inventoryAhead} ${ageMs}` : `${t.inventoryAge} ${ageMs}`

  return (
    <div
      className="flex flex-wrap items-center gap-x-2 gap-y-0.5 font-mono text-[9.5px] text-muted-foreground"
      style={{ opacity: freshness(read.age, readingFull, READING_FADE) }}
      title={ageTitle}
    >
      {grenades.map((g) => {
        const isSel = typeof selected === 'object' && selected !== null && g.rank === selected.rank
        // La vignette du HUD porte l'identité, le nom passe en infobulle : à cette taille
        // le dessin est plus lisible que le mot (POC). Sans vignette, le libellé reste.
        const lbl: CatalogLabel | undefined = doc.grenadeLabels?.[g.rank]
        return (
          <span
            key={g.rank}
            className={`inline-flex items-center gap-0.5 ${isSel ? 'rounded-sm px-0.5 font-semibold' : ''}`}
            style={
              isSel
                ? {
                    color: tokenCssVar('warning'),
                    boxShadow: `0 0 0 1px ${tokenCssVar('warning')}`,
                    background: `color-mix(in srgb, ${tokenCssVar('warning')} 13%, transparent)`,
                  }
                : undefined
            }
            title={
              isSel
                ? `${g.name} — ${selected.read ? t.grenadeSelectedRead : t.grenadeSelected}`
                : g.name
            }
          >
            {lbl?.img ? (
              <WeaponIcon
                imageUrl={lbl.img}
                tinted={lbl.tinted}
                label={g.name}
                width={HUD_ICON_PX}
                height={HUD_ICON_PX}
              />
            ) : (
              g.name
            )}
            <span className="tabular-nums">×{g.count}</span>
          </span>
        )
      })}
      {selected === 'indeterminate' && (
        <span className="border-b border-dashed border-border opacity-80" title={t.grenadeSelUnknownHint}>
          {t.grenadeSelUnknown}
        </span>
      )}
      {ability && (
        <span
          className={
            ability.known
              ? 'inline-flex items-center'
              : 'border-b border-dashed border-border'
          }
          title={ability.known ? `${t.abilityLabel} — ${ability.text}` : t.abilityUnknownHint}
        >
          {ability.img ? (
            <WeaponIcon
              imageUrl={ability.img}
              tinted={ability.tinted}
              label={ability.text}
              width={HUD_ICON_PX}
              height={HUD_ICON_PX}
            />
          ) : (
            ability.text
          )}
        </span>
      )}
      {ammoOrder.map((i) => (
        <AmmoCell
          key={i}
          index={i}
          ammo={ammo[i]}
          drawn={equipped?.drawn === i}
          noneLabel={t.ammoNone}
          noneHint={t.ammoNoneHint}
          hint={equipped?.drawn === i ? `${t.ammoSlotHint} ${t.ammoDrawnHint}` : t.ammoSlotHint}
          gaugeLabel={t.gaugeLabel}
        />
      ))}
      {ammo.length > 0 && equipped && equipped.drawn === null && (
        <span
          className="border-b border-dashed border-border opacity-80"
          title={equipped.holstered ? t.weaponsHolstered : t.drawnUnknownHint}
        >
          {equipped.holstered ? t.weaponsHolsteredShort : t.drawnUnknown}
        </span>
      )}
    </div>
  )
}

/**
 * abilityText nomme la capacité — et porte sa vignette de HUD quand le document en sert
 * une. Un index hors table garde son numéro : ni nom ni vignette empruntés à un voisin.
 * Renvoie null quand rien n'a été lu — l'absence de capacité et une capacité inconnue sont
 * deux états différents.
 */
function abilityText(
  doc: ReplayDocumentReady,
  index: number | undefined,
  unknownLabel: string,
  locale: ReplayLocale,
): { text: string; known: boolean; img?: string; tinted?: boolean } | null {
  if (index === undefined) return null
  const lbl: CatalogLabel | undefined = doc.abilityLabels?.[String(index)]
  const name = catalogText(lbl, locale)
  if (name) return { text: name, known: true, img: lbl?.img, tinted: lbl?.tinted }
  return { text: `${unknownLabel} (${index})`, known: false }
}

/**
 * AmmoCell — une cellule par EMPLACEMENT du record, numérotée.
 *
 * L'EMPLACEMENT k EST L'ARME k, mais seulement quand la lecture du bloc est UNIQUE : 197
 * appariements sur 198 concordent dans ce cas. Sur une lecture à plusieurs candidats, la
 * correspondance se brouille — c'est ce bruit qui avait fait conclure, à tort, que le
 * rattachement échouait.
 *
 * LE NUMÉRO RESTE AFFICHÉ parce qu'il coûte deux caractères et qu'il lève l'ambiguïté sans
 * rien affirmer. La réserve, elle, est portée par l'infobulle et par le marquage des lectures
 * ambiguës.
 */
function AmmoCell({
  index,
  ammo,
  drawn,
  noneLabel,
  noneHint,
  hint,
  gaugeLabel,
}: {
  index: number
  ammo: { mag?: number; res?: number; gauge?: number }
  /** L'emplacement est DÉGAINÉ selon le sélecteur : encre plus franche, index accentué. */
  drawn: boolean
  noneLabel: string
  noneHint: string
  hint: string
  gaugeLabel: string
}) {
  return (
    <span
      className={`inline-flex items-center gap-1 tabular-nums ${drawn ? 'text-foreground' : ''}`}
      title={hint}
    >
      <i
        className={`not-italic text-[8px] ${drawn ? '' : 'opacity-45'}`}
        style={drawn ? { color: tokenCssVar('warning') } : undefined}
      >
        {index}
      </i>
      {ammo.mag !== undefined ? (
        <span>
          {ammo.mag}
          {ammo.res !== undefined && <span className="opacity-60">/{ammo.res}</span>}
        </span>
      ) : ammo.gauge !== undefined ? (
        // LA BARRE MONTRE CE QUI RESTE, ET LA DONNÉE DIT CE QUI A ÉTÉ CONSOMMÉ : d'où le
        // complément. L'afficher tel quel donnait une barre INVERSÉE — un marteau à 10 %
        // consommé s'affichait presque vide alors qu'il lui reste 90 %.
        //
        // Deux témoins fondent cette lecture (cf. ReplayAmmoSlot) : à la première image-clé du
        // match, quand personne n'a tiré, les armes à charge n'émettent AUCUN champ ; et dans
        // une même vie la valeur ne redescend jamais (6 hausses, 0 baisse).
        //
        // PAS DE COMPTEUR « n / N » : le quantum est propre à l'arme et le film ne dit pas
        // combien de charges font un plein.
        <span
          className="inline-block h-1 w-5 overflow-hidden rounded-sm bg-muted align-middle"
          aria-label={gaugeLabel}
        >
          <span
            className="block h-full rounded-sm"
            style={{
              width: `${Math.max(0, 1 - ammo.gauge) * 100}%`,
              background: tokenCssVar('warning'),
            }}
          />
        </span>
      ) : (
        <span className="italic opacity-70" title={noneHint}>
          {noneLabel}
        </span>
      )}
    </span>
  )
}
