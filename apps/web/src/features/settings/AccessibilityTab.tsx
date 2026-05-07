/**
 * AccessibilityTab — onglet Accessibilité dans les Paramètres.
 *
 * Permet de choisir entre la palette Standard et la palette Okabe-Ito.
 * Inclut un aperçu live des tokens de performance et d'outcomes.
 */
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { useSettingsDraftStore } from '@/stores/settingsDraftStore'
import type { ColorPalette } from '@/stores/settingsDraftStore'
import { tokenCssVar } from '@/lib/accessibility'
import type { SettingsText } from '@/features/settings/i18n'

interface Props {
  t: SettingsText
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

function ColorSwatch({ token, label }: { token: string; label: string }) {
  return (
    <div className="flex flex-col items-center gap-1">
      <div
        className="h-7 w-7 rounded-full border border-border shadow-sm"
        style={{ backgroundColor: tokenCssVar(token as Parameters<typeof tokenCssVar>[0]) }}
        aria-label={label}
        title={label}
      />
      <span className="text-[9px] text-muted-foreground leading-none">{label}</span>
    </div>
  )
}

export function AccessibilityTab({ t }: Props) {
  const colorPalette = useSettingsDraftStore((s) => s.localUiPrefs.colorPalette)
  const setColorPalette = useSettingsDraftStore((s) => s.setColorPalette)

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
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
