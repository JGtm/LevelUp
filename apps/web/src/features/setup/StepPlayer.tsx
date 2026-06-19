/**
 * StepPlayer — étape 2 du wizard SetupPage : création du profil joueur.
 *
 * Multi-titre (Pass C lot 3) : l'utilisateur choisit les jeux à suivre + le
 * nombre de matchs à synchroniser par jeu (au moins un). Un profil est créé par
 * jeu coché (séquentiellement) ; le bootstrap n'est invalidé qu'une fois tous les
 * profils créés (évite un re-render intermédiaire du wizard). Le sélecteur de
 * jeu n'apparaît que s'il y a plus d'un titre actif (single-titre = juste le
 * volume de matchs).
 */
import { useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSetupFlowStore } from '@/stores/setupFlowStore'
import { queryKeys } from '@/lib/query/keys'
import { useCreatePlayer } from './queries'
import { getApiErrorMessage } from './_helpers'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'

const DEFAULT_MAX_MATCHES = 200

function clampMatches(n: number): number {
  if (!Number.isFinite(n) || n < 1) return DEFAULT_MAX_MATCHES
  return Math.min(2000, Math.max(1, Math.trunc(n)))
}

export function StepPlayer() {
  const linkedHaloIdentity = useAppShellStore((s) => s.linkedHaloIdentity)
  const availableTitles = useAppShellStore((s) => s.availableTitles)
  const setSelectedTitles = useSetupFlowStore((s) => s.setSelectedTitles)
  const [gamertagInput, setGamertagInput] = useState('')
  const queryClient = useQueryClient()
  const createPlayer = useCreatePlayer()
  const gamertag = linkedHaloIdentity?.gamertag ?? gamertagInput
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

  // Titres sélectionnables = actifs uniquement. Overrides locaux (robustes au
  // chargement async du store : on lit l'override sinon le défaut du titre).
  const activeTitles = availableTitles.filter((x) => x.status === 'active')
  const [checkedOverride, setCheckedOverride] = useState<Record<string, boolean>>({})
  const [maxOverride, setMaxOverride] = useState<Record<string, number>>({})
  const [creating, setCreating] = useState(false)
  const [createError, setCreateError] = useState<string | null>(null)

  const isChecked = (slug: string, isDefault: boolean) => checkedOverride[slug] ?? isDefault
  const getMax = (slug: string) => maxOverride[slug] ?? DEFAULT_MAX_MATCHES
  const selected = activeTitles.filter((x) => isChecked(x.slug, x.is_default))
  const showCheckboxes = activeTitles.length > 1

  async function handleCreate() {
    if (!gamertag.trim() || selected.length === 0 || creating) return
    setCreating(true)
    setCreateError(null)
    const slugs = selected.map((x) => x.slug)
    const maxByTitle: Record<string, number> = {}
    for (const x of selected) maxByTitle[x.slug] = getMax(x.slug)
    try {
      // Création séquentielle d'un profil par titre choisi.
      for (const slug of slugs) {
        await createPlayer.mutateAsync({
          gamertag: gamertag.trim(),
          xuid: linkedHaloIdentity?.xuid ?? undefined,
          title_slug: slug,
          initial_max_matches: maxByTitle[slug],
        })
      }
      setSelectedTitles(slugs, maxByTitle)
      // Invalidation UNIQUE après tous les profils → le wizard bascule proprement.
      await queryClient.invalidateQueries({ queryKey: queryKeys.bootstrap })
    } catch (e) {
      setCreateError(getApiErrorMessage(e, 'Erreur lors de la création du profil.'))
    } finally {
      setCreating(false)
    }
  }

  return (
    <div className="space-y-4">
      <h2 className="text-lg font-semibold">{t('common.setup.create_player_profile')}</h2>

      {linkedHaloIdentity ? (
        <div className="rounded-lg border border-primary/30 bg-primary/10 p-4">
          <p className="text-xs text-muted-foreground">{t('common.setup.halo_identity_linked')}</p>
          <p className="mt-1 text-2xl font-bold text-primary">{linkedHaloIdentity.gamertag}</p>
          <p className="text-xs text-muted-foreground mt-0.5 font-mono">XUID {linkedHaloIdentity.xuid}</p>
          <p className="mt-2 text-xs text-muted-foreground">{t('common.setup.local_profile_will_be_created')}</p>
        </div>
      ) : (
        <>
          <p className="text-sm text-muted-foreground">{t('common.setup.enter_gamertag')}</p>
          <Input
            value={gamertagInput}
            onChange={(e) => setGamertagInput(e.target.value)}
            placeholder="MonGamertag"
            onKeyDown={(e) => { if (e.key === 'Enter') void handleCreate() }}
          />
        </>
      )}

      {/* Sélection des jeux + volume de matchs */}
      {activeTitles.length > 0 && (
        <div className="rounded-lg border border-border p-3">
          <p className="text-sm font-medium">{t('common.setup.select_titles_title')}</p>
          <p className="mt-0.5 text-xs text-muted-foreground">{t('common.setup.select_titles_desc')}</p>
          <div className="mt-2 space-y-1">
            {activeTitles.map((tt) => {
              const checked = isChecked(tt.slug, tt.is_default)
              return (
                <div key={tt.slug} className="flex items-center justify-between gap-3 py-1">
                  <label className="flex items-center gap-2 text-sm">
                    {showCheckboxes && (
                      <input
                        type="checkbox"
                        checked={checked}
                        onChange={(e) => setCheckedOverride((p) => ({ ...p, [tt.slug]: e.target.checked }))}
                      />
                    )}
                    <span>{tt.name}</span>
                  </label>
                  <div className="flex items-center gap-2">
                    <span className="text-xs text-muted-foreground">{t('common.setup.matches_to_sync')}</span>
                    <Input
                      type="number"
                      min={1}
                      max={2000}
                      value={getMax(tt.slug)}
                      disabled={!checked}
                      onChange={(e) =>
                        setMaxOverride((p) => ({ ...p, [tt.slug]: clampMatches(parseInt(e.target.value, 10)) }))
                      }
                      className="w-20"
                    />
                  </div>
                </div>
              )
            })}
          </div>
        </div>
      )}

      {createError && <p className="text-destructive text-sm">{createError}</p>}

      <Button
        onClick={() => void handleCreate()}
        disabled={!gamertag.trim() || selected.length === 0 || creating}
      >
        {creating
          ? 'Création…'
          : linkedHaloIdentity
          ? 'Confirmer et créer mon profil'
          : 'Ajouter'}
      </Button>
    </div>
  )
}
