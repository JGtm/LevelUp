/**
 * useSiteImportForm — saisie et progression de l'import depuis le site (lot L22).
 *
 * Le jeton d'export vit dans l'état du composant, le temps de la saisie : il est vidé dès l'envoi et
 * n'entre dans aucune clé de requête, aucune URL, aucun stockage du navigateur, aucun journal. À la
 * fin d'un import réussi, les données de l'application sont relues (amorçage, joueurs).
 */
import { useEffect, useState, type FormEvent } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { apiErrorCode } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { isImportRunning, useSiteImportStatus, useStartSiteImport } from './siteImportQueries'
import type { SiteImportRequest } from './siteImportQueries'
import type { SiteImportText } from './siteImportText'

/** Code de refus quand plusieurs joueurs sont suivis et qu'aucun gamertag n'a été donné. */
const AMBIGUOUS_PLAYER_CODE = 'import_player_ambiguous'

/** Le texte d'un échec (code fixe du serveur) ; un code inconnu donne le texte générique. */
export function importErrorText(t: SiteImportText, code: string | undefined): string {
  return (code !== undefined ? t.errors[code] : undefined) ?? t.unknownError
}

export function useSiteImportForm(t: SiteImportText) {
  const qc = useQueryClient()
  const { data: status } = useSiteImportStatus(true)
  const start = useStartSiteImport()
  const [url, setUrl] = useState('')
  const [token, setToken] = useState('')
  const [gamertag, setGamertag] = useState('')
  const [askGamertag, setAskGamertag] = useState(false)

  const phase = status?.phase
  useEffect(() => {
    if (phase !== 'done') return
    void qc.invalidateQueries({ queryKey: queryKeys.bootstrap })
    void qc.invalidateQueries({ queryKey: queryKeys.players })
  }, [phase, qc])

  function submit(e: FormEvent) {
    e.preventDefault()
    const req: SiteImportRequest = { url: url.trim(), token: token.trim() }
    if (gamertag.trim() !== '') req.gamertag = gamertag.trim()
    // Le jeton est à usage unique : il quitte le champ dès l'envoi, réussi ou non.
    setToken('')
    start.mutate(req, {
      onError: (err) => {
        if (apiErrorCode(err) === AMBIGUOUS_PLAYER_CODE) setAskGamertag(true)
      },
    })
  }

  const running = isImportRunning(phase) || start.isPending
  const errorMessage = start.isError
    ? importErrorText(t, apiErrorCode(start.error))
    : phase === 'failed'
      ? importErrorText(t, status?.error_code)
      : undefined

  return {
    status,
    running,
    starting: start.isPending,
    errorMessage,
    url,
    token,
    gamertag,
    askGamertag,
    setUrl,
    setToken,
    setGamertag,
    canSubmit: !running && url.trim() !== '' && token.trim() !== '',
    submit,
  }
}
