/**
 * ReplayFeedMedals — LES MARQUES ET LES MÉDAILLES D'UNE LIGNE D'ÉLIMINATION, dans la dernière colonne : les marques
 * vraies du tableau des scores (tir à la tête, frag parfait), puis deux médailles au plus, puis « +N » quand il y en a
 * davantage. Le nom (et la description) de chaque médaille vit en infobulle ; celle du « +N » liste les médailles
 * qu'il remplace. L'icône du frag parfait s'efface quand la ligne porte la médaille « Parfait » : elle le dit déjà.
 *
 * Les médailles sont celles du document de rejeu (`readMatchMedals`) : libellé, description et image
 * viennent de sa table. Une médaille sans visuel garde son TEXTE, tronqué à la largeur d'une pastille :
 * jamais le pictogramme d'une autre. Les tailles sont `COTES_LIGNE_FIL.medailles`.
 */
import { COTES_LIGNE_FIL } from '../model/cotesFeedLine'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import type { FragMarks } from '../model/eliminationMarks'
import type { MedalEvent } from '../model/killFeedLogic'
import { PERFECT_MEDAL_NAME } from '../model/playerInstantStats'
import { FragMarkIcons } from './ReplayFragMarks'

const COTES = COTES_LIGNE_FIL.medailles

const medalLabel = (m: MedalEvent) => m.label || m.name
const medalTooltip = (m: MedalEvent) => (m.description ? `${medalLabel(m)} — ${m.description}` : medalLabel(m))

export function FeedMedals({
  medals,
  marks,
  locale,
}: {
  medals: readonly MedalEvent[]
  /** Les marques vraies de l'élimination (tir à la tête, frag parfait), en tête de la colonne. */
  marks?: FragMarks
  locale: ReplayLocale
}) {
  const t = REPLAY_TEXT[locale].feedLines
  const shown = medals.slice(0, COTES.pictosMax)
  const hidden = medals.slice(COTES.pictosMax)
  return (
    <span data-feed-col="medailles" className="flex min-w-0 items-center" style={{ gap: COTES.ecartPx }}>
      <FragMarkIcons marks={marks} perfect={!medals.some((m) => m.name === PERFECT_MEDAL_NAME)} locale={locale} />
      {shown.map((m, i) =>
        m.imageUrl ? (
          <img
            key={`${m.name}-${m.tMs}-${i}`}
            src={m.imageUrl}
            alt={medalLabel(m)}
            title={medalTooltip(m)}
            width={COTES.pictoPx}
            height={COTES.pictoPx}
            className="inline-block shrink-0 object-contain"
            loading="lazy"
          />
        ) : (
          <span
            key={`${m.name}-${m.tMs}-${i}`}
            title={medalTooltip(m)}
            className="truncate rounded-sm border border-border px-1 text-3xs text-muted-foreground"
            style={{ maxWidth: COTES.texteMaxPx }}
          >
            {medalLabel(m)}
          </span>
        ),
      )}
      {hidden.length > 0 && (
        <span
          className="shrink-0 text-muted-foreground"
          style={{ fontSize: COTES.tailleResteDemiPx }}
          title={t.medalsMoreHint(hidden.map(medalLabel).join(', '))}
        >
          {t.medalsMore(hidden.length)}
        </span>
      )}
    </span>
  )
}
