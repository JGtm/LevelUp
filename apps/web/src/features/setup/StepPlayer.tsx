/**
 * StepPlayer — étape 2 du wizard SetupPage : création du profil joueur.
 *
 * P8.4 (revue 2026-04-29) : extrait de SetupPage.tsx (~70L).
 */
import { useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useAppShellStore } from '@/stores/appShellStore'
import { queryKeys } from '@/lib/query/keys'
import { useCreatePlayer } from './queries'
import { getApiErrorMessage } from './_helpers'

export function StepPlayer() {
  const linkedHaloIdentity = useAppShellStore((s) => s.linkedHaloIdentity)
  const [gamertagInput, setGamertagInput] = useState('')
  const queryClient = useQueryClient()
  const createPlayer = useCreatePlayer()
  const gamertag = linkedHaloIdentity?.gamertag ?? gamertagInput

  function handleCreate() {
    if (!gamertag.trim()) return
    createPlayer.mutate(
      {
        gamertag: gamertag.trim(),
        xuid: linkedHaloIdentity?.xuid ?? undefined,
      },
      {
        onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.bootstrap }),
      },
    )
  }

  return (
    <div className="space-y-4">
      <h2 className="text-lg font-semibold">Créer votre profil joueur</h2>

      {linkedHaloIdentity ? (
        /* Carte de confirmation — identité résolue depuis la session */
        <div className="rounded-lg border border-primary/30 bg-primary/10 p-4">
          <p className="text-xs text-muted-foreground">Identité Halo liée à cette session :</p>
          <p className="mt-1 text-2xl font-bold text-primary">
            {linkedHaloIdentity.gamertag}
          </p>
          <p className="text-xs text-muted-foreground mt-0.5 font-mono">
            XUID {linkedHaloIdentity.xuid}
          </p>
          <p className="mt-2 text-xs text-muted-foreground">
            Un profil local sera créé pour ce compte.
          </p>
        </div>
      ) : (
        <>
          <p className="text-sm text-muted-foreground">
            Entrez votre Gamertag Xbox pour créer votre profil.
          </p>
          <Input
            value={gamertagInput}
            onChange={(e) => setGamertagInput(e.target.value)}
            placeholder="MonGamertag"
            onKeyDown={(e) => { if (e.key === 'Enter') handleCreate() }}
          />
        </>
      )}

      {createPlayer.isError && (
        <p className="text-destructive text-sm">
          {getApiErrorMessage(createPlayer.error, 'Erreur lors de la création du profil.')}
        </p>
      )}

      <Button
        onClick={handleCreate}
        disabled={!gamertag.trim() || createPlayer.isPending || createPlayer.isSuccess}
      >
        {createPlayer.isPending
          ? 'Création…'
          : linkedHaloIdentity
          ? 'Confirmer et créer mon profil'
          : 'Ajouter'}
      </Button>
    </div>
  )
}
