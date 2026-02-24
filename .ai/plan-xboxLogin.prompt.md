# Plan : Connexion Xbox via OAuth dans Streamlit

> Statut : **À implémenter** — plan rédigé le 24/02/2026, non démarré.

## Résumé

Ajouter un flux de connexion Xbox (OAuth Microsoft) directement dans l'app Streamlit.
L'utilisateur clique "Se connecter avec Xbox", s'authentifie sur la page Microsoft, et est
automatiquement redirigé vers l'app qui crée son profil. Chaque joueur stocke son propre
`refresh_token` dans sa DB (`sync_meta`), rendant l'app multi-utilisateurs sans fichier
`.env` partagé.

**Mécanisme clé :** Microsoft redirige vers `http://localhost:8501/?code=XXXX` → Streamlit
re-execute le script → `st.query_params["code"]` est lu avant d'être effacé → échange
contre des tokens SPNKr.

---

## Prérequis Azure (1 action manuelle, une seule fois)

Aller dans Azure Portal → App Registration → Authentication → Redirect URIs et ajouter :
- `http://localhost:8501` (local)
- Plus tard : `http://ton-nas:8501`

La variable `SPNKR_AZURE_REDIRECT_URI` (déjà supportée dans le code) devra pointer vers
`http://localhost:8501` dans `.env.local`.

---

## Étapes

1. **Créer `src/ui/xbox_login.py`** — Module central avec :
   - `build_xbox_auth_url(client_id, redirect_uri, state) → str` — réutilise la logique de
     `_build_authorize_url()` de `scripts/spnkr_get_refresh_token.py`
   - `exchange_code_for_refresh_token(code, redirect_uri) → str` — réutilise
     `_oauth2_token_request_v2()` du même script
   - `resolve_player_identity(spartan_token, clearance_token) → tuple[str, str]` — appelle
     l'API Halo (SPNKr `get_current_user()`) pour obtenir gamertag + XUID
   - `create_player_profile(gamertag, xuid) → str` — crée `data/players/{gamertag}/stats.duckdb`,
     met à jour `db_profiles.json`
   - `store_refresh_token_in_db(db_path, refresh_token)` — persiste le token dans `sync_meta`
     (clé `oauth_refresh_token`) du `stats.duckdb` du joueur
   - `load_refresh_token_from_db(db_path) → str | None` — lit depuis `sync_meta`

2. **Modifier `streamlit_app.py` (section query_params, ~lignes 431–450)** — Avant le
   `st.query_params.clear()` existant, détecter `code` et `state` pour handler le callback OAuth :
   - Lire `qp.get("code")` et `qp.get("state")`
   - Vérifier le `state` CSRF contre `st.session_state["_oauth_state"]`
   - Si valide → déclencher `handle_xbox_callback(code)` (async via `ThreadPoolExecutor`,
     cohérent avec le pattern de `src/ui/profile_api_tokens.py`)
   - Stocker le résultat dans `st.session_state["_pending_xbox_login"]`

3. **Créer `src/ui/pages/login.py`** — Page Streamlit dédiée "Connexion" avec :
   - Bouton "Se connecter avec Xbox 🎮" → génère `state` aléatoire dans `st.session_state`,
     ouvre l'URL OAuth via `st.link_button`
   - Spinner d'attente si `_pending_xbox_login` est en cours
   - Confirmation / erreur après callback

4. **Modifier `src/ui/profile_api_tokens.py` → `ensure_spnkr_tokens()`** — Ajouter un chemin
   alternatif : si un `db_path` de session est disponible, lire le `refresh_token` depuis
   `sync_meta` plutôt que depuis `.env.local`. Tokens par-session plutôt que globaux.

5. **Modifier la sidebar dans `streamlit_app.py` (~ligne 453)** — Indicateur de session
   ("Connecté en tant que JGtm") + bouton "Changer de compte". Si aucun profil configuré,
   afficher un bandeau "Connecter un compte Xbox".

6. **Mettre à jour `src/data/sync/engine.py`** — S'assurer que `sync_meta` accepte la clé
   `oauth_refresh_token` sans la supprimer lors des syncs.

7. **Enregistrer la page `login` dans la navigation** (`streamlit_app.py` ou
   `src/app/routing.py`).

---

## Vérification

- Test unitaire : `tests/test_xbox_login.py` — mock HTTP pour `exchange_code_for_refresh_token`
- Test manuel :
  1. Ajouter `http://localhost:8501` dans Azure Portal
  2. `streamlit run streamlit_app.py`
  3. Page Connexion → bouton → auth → vérifier profil dans `db_profiles.json` et
     `oauth_refresh_token` dans `sync_meta`

---

## Décisions

- **`st.query_params` comme callback receiver** : natif Streamlit, pas de serveur HTTP séparé
- **Tokens stockés dans `sync_meta`** : isolation par joueur, compatible multi-user
- **`ThreadPoolExecutor` pour l'async** : cohérent avec `profile_api_tokens.py`
- **`SPNKR_AZURE_REDIRECT_URI=http://localhost:8501`** : variable déjà supportée, juste à changer sa valeur
- **NAS** : URL configurable via `SPNKR_AZURE_REDIRECT_URI`

## Limites connues

- Le `code` OAuth expire en ~5 min — rare en pratique locale
- Pour le NAS, la Redirect URI Azure devra être mise à jour manuellement à chaque changement d'IP/port
