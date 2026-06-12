/**
 * credentials — déclenche la proposition d'enregistrement du mot de passe par
 * le navigateur après un login admin réussi.
 *
 * Le login passe par `fetch` (TanStack Query) sans navigation native de
 * formulaire : l'heuristique de sauvegarde de Chrome ne se déclenche donc pas
 * toute seule. L'API Credential Management (`navigator.credentials.store` +
 * `PasswordCredential`) provoque explicitement le prompt « Enregistrer le mot
 * de passe ? ».
 *
 * Compatibilité : `PasswordCredential` est Chromium-only et requiert un contexte
 * sécurisé (HTTPS, ou localhost). Feature-detection + dégradation silencieuse
 * ailleurs (Firefox/Safari ignorent simplement l'appel).
 */
export function storePasswordCredential(id: string, password: string): void {
  if (typeof window === 'undefined' || !id || !password) return
  const PasswordCredentialCtor = (window as unknown as {
    PasswordCredential?: new (data: { id: string; password: string }) => Credential
  }).PasswordCredential
  if (!PasswordCredentialCtor || !navigator.credentials?.store) return
  try {
    const cred = new PasswordCredentialCtor({ id, password })
    // Best-effort : on n'attend pas et on avale toute erreur (prompt refusé,
    // policy navigateur, etc.) — ne doit jamais casser le flux de login.
    void navigator.credentials.store(cred).catch(() => {})
  } catch {
    /* API non supportée / arguments invalides : on ignore. */
  }
}
