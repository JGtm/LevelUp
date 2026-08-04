/**
 * AccessibilityTab — onglet Accessibilité dans les Paramètres.
 *
 * Permet de choisir entre la palette Standard et la palette Okabe-Ito.
 * Inclut un aperçu live des tokens de performance, d'outcomes, divergents et de
 * la quadruple couleur par joueur d'escouade.
 * Permet également de configurer les couleurs d'équipe (outline Halo).
 */
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { useSettingsDraftStore } from '@/stores/settingsDraftStore'
import type { ColorPalette } from '@/stores/settingsDraftStore'
import { tokenCssVar } from '@/lib/accessibility'
import { HALO_OUTLINE_COLORS } from '@/lib/halo/outline-colors'
import type { SettingsText } from '@/features/settings/i18n'
import type { Locale } from '@/lib/i18n/locale'

interface Props {
  t: SettingsText
  locale: Locale
}

const PERF_TOKENS = [
  'perf-tier-1',
  'perf-tier-2',
  'perf-tier-3',
  'perf-tier-4',
  'perf-tier-5',
] as const

const OUTCOME_TOKENS = [
  'outcome-win',
  'outcome-loss',
  'outcome-draw',
  'outcome-dnf',
] as const

const DIVERGENT_TOKENS = [
  'divergent-pos',
  'divergent-neutral',
  'divergent-neg',
] as const

// Quadruple couleur par joueur d'escouade (source unique : features/squad/colors).
// Présente dans les 4 palettes, donc aperçu-able ici au même titre que les autres
// familles — c'est la famille qui décide de la lisibilité des graphes d'escouade.
const SQUAD_PLAYER_TOKENS = [
  'squad-player-1',
  'squad-player-2',
  'squad-player-3',
  'squad-player-4',
] as const

function PaletteOption({
  value,
  label,
  description,
  selected,
  onSelect,
}: {
  value: ColorPalette
  label: string
  description: string
  selected: boolean
  onSelect: (v: ColorPalette) => void
}) {
  return (
    <label
      className={`flex cursor-pointer items-start gap-3 rounded-lg border p-4 transition-colors ${
        selected ? 'border-primary bg-primary/5' : 'border-border hover:border-border/80'
      }`}
    >
      <input
        type="radio"
        name="colorPalette"
        value={value}
        checked={selected}
        onChange={() => onSelect(value)}
        className="mt-1 accent-primary"
      />
      <div>
        <p className="text-sm font-medium text-foreground">{label}</p>
        <p className="mt-0.5 text-xs text-muted-foreground">{description}</p>
      </div>
    </label>
  )
}

// swatchText : légende affichée sous la pastille quand elle diffère du libellé
// accessible (ex. « 1 » sous la pastille, « Joueurs d'escouade 1 » pour le lecteur
// d'écran). Par défaut la légende EST le libellé.
function ColorSwatch({
  token,
  label,
  swatchText,
}: {
  token: string
  label: string
  swatchText?: string
}) {
  return (
    <div className="flex flex-col items-center gap-1">
      <div
        className="h-7 w-7 rounded-full border border-border shadow-sm"
        style={{ backgroundColor: tokenCssVar(token as Parameters<typeof tokenCssVar>[0]) }}
        aria-label={label}
        title={label}
      />
      <span className="text-[9px] text-muted-foreground leading-none">{swatchText ?? label}</span>
    </div>
  )
}

function OutlineColorPicker({
  label,
  value,
  defaultLabel,
  onChange,
  locale,
}: {
  label: string
  value: string | null
  defaultLabel: string
  onChange: (id: string | null) => void
  locale: Locale
}) {
  const selected = HALO_OUTLINE_COLORS.find((c) => c.id === value)
  const selectedName = selected
    ? locale === 'fr'
      ? selected.nameFr
      : selected.nameEn
    : null

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2">
        <p className="text-sm font-medium text-foreground">{label}</p>
        {selectedName && (
          <span className="text-xs text-muted-foreground">— {selectedName}</span>
        )}
      </div>
      <div className="flex flex-wrap gap-2">
        {/* Swatch "défaut palette" */}
        <button
          type="button"
          title={defaultLabel}
          aria-label={defaultLabel}
          aria-pressed={value === null}
          onClick={() => onChange(null)}
          className={`flex h-8 w-8 items-center justify-center rounded-full border-2 text-xs transition-all ${
            value === null
              ? 'border-primary bg-primary/10 text-primary scale-110 shadow-md'
              : 'border-border text-muted-foreground hover:border-border/60'
          }`}
        >
          —
        </button>

        {HALO_OUTLINE_COLORS.map((color) => {
          const name = locale === 'fr' ? color.nameFr : color.nameEn
          const isSelected = value === color.id
          return (
            <button
              key={color.id}
              type="button"
              title={name}
              aria-label={name}
              aria-pressed={isSelected}
              onClick={() => onChange(isSelected ? null : color.id)}
              className={`h-8 w-8 rounded-full border-2 transition-all ${
                isSelected
                  ? 'scale-110 border-white shadow-lg ring-2 ring-white/40'
                  : 'border-transparent hover:scale-105 hover:border-white/30'
              }`}
              style={{ backgroundColor: color.hex }}
            />
          )
        })}
      </div>
    </div>
  )
}

export function AccessibilityTab({ t, locale }: Props) {
  const colorPalette = useSettingsDraftStore((s) => s.localUiPrefs.colorPalette)
  const setColorPalette = useSettingsDraftStore((s) => s.setColorPalette)
  const allyTeamColor = useSettingsDraftStore((s) => s.localUiPrefs.allyTeamColor)
  const enemyTeamColor = useSettingsDraftStore((s) => s.localUiPrefs.enemyTeamColor)
  const setAllyTeamColor = useSettingsDraftStore((s) => s.setAllyTeamColor)
  const setEnemyTeamColor = useSettingsDraftStore((s) => s.setEnemyTeamColor)

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{t.accessibilityTitle}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-6">
        <p className="text-sm text-muted-foreground">{t.accessibilityDescription}</p>

        <div className="space-y-3">
          <p className="text-sm font-medium text-foreground">{t.paletteLabel}</p>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
            <PaletteOption
              value="default"
              label={t.paletteDefault}
              description={t.paletteDefaultDesc}
              selected={colorPalette === 'default'}
              onSelect={setColorPalette}
            />
            <PaletteOption
              value="okabe-ito"
              label={t.paletteOkabeIto}
              description={t.paletteOkabeItoDesc}
              selected={colorPalette === 'okabe-ito'}
              onSelect={setColorPalette}
            />
            <PaletteOption
              value="cividis"
              label={t.paletteCividis}
              description={t.paletteCividisDesc}
              selected={colorPalette === 'cividis'}
              onSelect={setColorPalette}
            />
            <PaletteOption
              value="tol-bright"
              label={t.paletteTolBright}
              description={t.paletteTolBrightDesc}
              selected={colorPalette === 'tol-bright'}
              onSelect={setColorPalette}
            />
          </div>
        </div>

        <div className="space-y-3">
          <p className="text-sm font-medium text-foreground">{t.previewLabel}</p>
          <div className="rounded-lg border border-border bg-muted/40 p-4 space-y-3">
            <div className="flex flex-wrap gap-3">
              {PERF_TOKENS.map((token) => (
                <ColorSwatch key={token} token={token} label={token.replace('perf-tier-', 'P')} />
              ))}
            </div>
            <div className="flex flex-wrap gap-3">
              {OUTCOME_TOKENS.map((token) => (
                <ColorSwatch key={token} token={token} label={token.replace('outcome-', '')} />
              ))}
            </div>
            <div className="flex flex-wrap gap-3">
              {DIVERGENT_TOKENS.map((token) => (
                <ColorSwatch key={token} token={token} label={token.replace('divergent-', 'Δ ')} />
              ))}
            </div>
            <div className="space-y-1.5">
              <p className="text-xs text-muted-foreground">{t.previewSquadPlayersLabel}</p>
              <div className="flex flex-wrap gap-3">
                {SQUAD_PLAYER_TOKENS.map((token) => (
                  <ColorSwatch
                    key={token}
                    token={token}
                    label={`${t.previewSquadPlayersLabel} ${token.replace('squad-player-', '')}`}
                    swatchText={token.replace('squad-player-', '')}
                  />
                ))}
              </div>
            </div>
          </div>
        </div>

        <div className="space-y-4">
          <div>
            <p className="text-sm font-medium text-foreground">{t.teamColorsTitle}</p>
            <p className="mt-0.5 text-xs text-muted-foreground">{t.teamColorsDescription}</p>
          </div>
          <div className="rounded-lg border border-border bg-muted/40 p-4 space-y-5">
            <OutlineColorPicker
              label={t.allyColorLabel}
              value={allyTeamColor}
              defaultLabel={t.teamColorDefault}
              onChange={setAllyTeamColor}
              locale={locale}
            />
            <OutlineColorPicker
              label={t.enemyColorLabel}
              value={enemyTeamColor}
              defaultLabel={t.teamColorDefault}
              onChange={setEnemyTeamColor}
              locale={locale}
            />
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
