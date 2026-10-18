import { afterEach, describe, expect, it } from 'vitest'

import { useAppShellStore } from '@/stores/appShellStore'

import { buildWeaponAccuracyOption } from './WeaponAccuracyChart'

afterEach(() => useAppShellStore.setState({ locale: 'fr' }))

type Formatter = (p: unknown) => string
const point = { name: 'BR75', value: 53.456, data: { shotsLanded: 1234, shotsFired: 2300, className: 'Précision' } }

describe('infobulle de la précision par arme', () => {
  it('écrit le pourcentage et le décompte à la langue de l’application, avec l’unité traduite', () => {
    const series = [{ key: 'a', datapoints: [{ label: 'BR75', accuracyPct: 53.456, shotsFired: 2300, shotsLanded: 1234 }] }]
    const fr = buildWeaponAccuracyOption(series, 'fr') as unknown as { tooltip: { formatter: Formatter } }
    const htmlFr = fr.tooltip.formatter(point).normalize('NFKC')
    expect(htmlFr).toContain('53,5 %')
    expect(htmlFr).toContain('1 234 / 2 300 tirs')
    useAppShellStore.setState({ locale: 'en' })
    const en = buildWeaponAccuracyOption(series, 'en') as unknown as { tooltip: { formatter: Formatter } }
    const htmlEn = en.tooltip.formatter(point).normalize('NFKC')
    expect(htmlEn).toContain('53.5 %')
    expect(htmlEn).toContain('1,234 / 2,300 shots')
    expect(htmlEn).not.toContain('tirs')
  })
})
