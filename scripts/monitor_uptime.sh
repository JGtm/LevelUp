#!/usr/bin/env bash
# monitor_uptime.sh — Surveillance de disponibilité du site LevelUp (Tailscale Funnel)
#
# Même logique que monitor_uptime.py :
#   • Ping HTTP → état ONLINE / OFFLINE
#   • Notification Discord uniquement sur changement d'état
#   • Anti-flapping : UPTIME_OFFLINE_THRESHOLD checks consécutifs avant notif OFFLINE
#   • URL Tailscale : TAILSCALE_FUNNEL_URL (env) > tailscale funnel status --json
#   • Webhook : DISCORD_WEBHOOK_URL (.env.local) > discord_webhook_url (app_settings.json)
#
# Dépendances : curl, jq
# Usage : bash scripts/monitor_uptime.sh
#         (ou via cron / Planificateur de tâches)

set -euo pipefail

# ---------------------------------------------------------------------------
# Chemins
# ---------------------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

ENV_FILE="$ROOT_DIR/.env.local"
APP_SETTINGS="$ROOT_DIR/app_settings.json"
STATE_FILE="$ROOT_DIR/data/cache/uptime_state.json"
LOG_FILE="$ROOT_DIR/data/cache/uptime_monitor.log"

# ---------------------------------------------------------------------------
# Constantes
# ---------------------------------------------------------------------------
HTTP_TIMEOUT=10
DISCORD_RETRY_DELAY=2
DISCORD_MAX_RETRIES=2
DEFAULT_OFFLINE_THRESHOLD=3
USER_AGENT="LevelUp-UptimeMonitor/1.0"

COLOR_GREEN="5763719"   # 0x57F287
COLOR_RED="15548997"    # 0xED4245

# ---------------------------------------------------------------------------
# Logging
# ---------------------------------------------------------------------------
mkdir -p "$(dirname "$LOG_FILE")"

log() {
    local level="$1"
    shift
    local msg="$(date '+%Y-%m-%d %H:%M:%S') [$level] $*"
    echo "$msg" | tee -a "$LOG_FILE"
}

log_info()  { log "INFO"  "$@"; }
log_warn()  { log "WARN"  "$@"; }
log_error() { log "ERROR" "$@"; }
log_debug() { log "DEBUG" "$@"; }

# ---------------------------------------------------------------------------
# Chargement .env.local
# ---------------------------------------------------------------------------
load_env_local() {
    if [[ -f "$ENV_FILE" ]]; then
        # Charge uniquement les lignes KEY=VALUE (ignore commentaires et lignes vides)
        while IFS='=' read -r key value; do
            [[ "$key" =~ ^[[:space:]]*# ]] && continue
            [[ -z "$key" ]] && continue
            key="${key// /}"
            value="${value%\"}"
            value="${value#\"}"
            value="${value%\'}"
            value="${value#\'}"
            export "$key=$value"
        done < <(grep -E '^[A-Za-z_][A-Za-z0-9_]*=' "$ENV_FILE" || true)
    fi
}

# ---------------------------------------------------------------------------
# Lecture app_settings.json via jq
# ---------------------------------------------------------------------------
app_setting() {
    local key="$1"
    local default="${2:-}"
    if [[ -f "$APP_SETTINGS" ]] && command -v jq &>/dev/null; then
        local val
        val="$(jq -r --arg k "$key" '.[$k] // empty' "$APP_SETTINGS" 2>/dev/null || true)"
        echo "${val:-$default}"
    else
        echo "$default"
    fi
}

# ---------------------------------------------------------------------------
# Langue et traductions
# ---------------------------------------------------------------------------
get_lang() {
    local lang="${DISCORD_LANG:-$(app_setting discord_lang fr)}"
    echo "${lang:-fr}"
}

t() {
    local key="$1"
    local lang="$2"
    case "${lang}_${key}" in
        fr_online_title)  echo "✅ LevelUp — Site en ligne" ;;
        fr_online_body)   echo "Le dashboard est **accessible**." ;;
        fr_online_link)   echo "🔗 **Lien Tailscale Funnel :**" ;;
        fr_online_since)  echo "🕐 En ligne depuis :" ;;
        fr_offline_title) echo "❌ LevelUp — Site hors ligne" ;;
        fr_offline_body)  echo "Le dashboard est **inaccessible**." ;;
        fr_offline_since) echo "🕐 Hors ligne depuis :" ;;
        fr_footer)        echo "LevelUp Uptime Monitor" ;;
        en_online_title)  echo "✅ LevelUp — Dashboard online" ;;
        en_online_body)   echo "The dashboard is **reachable**." ;;
        en_online_link)   echo "🔗 **Tailscale Funnel link:**" ;;
        en_online_since)  echo "🕐 Online since:" ;;
        en_offline_title) echo "❌ LevelUp — Dashboard offline" ;;
        en_offline_body)  echo "The dashboard is **unreachable**." ;;
        en_offline_since) echo "🕐 Offline since:" ;;
        en_footer)        echo "LevelUp Uptime Monitor" ;;
        # Fallback fr
        *_online_title)   echo "✅ LevelUp — Site en ligne" ;;
        *_offline_title)  echo "❌ LevelUp — Site hors ligne" ;;
        *)                echo "[$key]" ;;
    esac
}

# ---------------------------------------------------------------------------
# Webhook Discord
# ---------------------------------------------------------------------------
get_webhook_url() {
    local enabled
    enabled="$(app_setting discord_notifications_enabled false)"
    if [[ "$enabled" != "true" && "$enabled" != "1" ]]; then
        log_debug "[Discord] Notifications désactivées (discord_notifications_enabled=false)."
        echo ""
        return
    fi

    local url="${DISCORD_WEBHOOK_URL:-}"
    if [[ -z "$url" ]]; then
        url="$(app_setting discord_webhook_url "")"
    fi

    if [[ "$url" == https://discord.com/api/webhooks/* ]]; then
        echo "$url"
    else
        log_warn "[Discord] Notifications activées mais aucun webhook valide trouvé."
        echo ""
    fi
}

# ---------------------------------------------------------------------------
# Résolution URL Tailscale
# ---------------------------------------------------------------------------
resolve_tailscale_url() {
    local override="${TAILSCALE_FUNNEL_URL:-}"
    if [[ -n "$override" ]]; then
        log_debug "[Tailscale] URL configurée manuellement : $override"
        echo "$override"
        return
    fi

    if command -v tailscale &>/dev/null && command -v jq &>/dev/null; then
        local json
        json="$(tailscale funnel status --json 2>/dev/null || true)"
        if [[ -n "$json" ]]; then
            local url
            url="$(echo "$json" | jq -r '
                .Self.DNSName // empty |
                if . then "https://" + ltrimstr("https://") + (if endswith(".") then .[:-1] else . end) else empty end
            ' 2>/dev/null || true)"
            # Fallback : chercher directement dans AllowFunnel ou DNSName
            if [[ -z "$url" ]]; then
                url="$(echo "$json" | jq -r '
                    (.Self.DNSName // "") |
                    if . != "" then "https://" + (if endswith(".") then .[:-1] else . end) else "" end
                ' 2>/dev/null || true)"
            fi
            if [[ -n "$url" && "$url" != "https://" ]]; then
                log_debug "[Tailscale] URL détectée via CLI : $url"
                echo "$url"
                return
            fi
        fi
    fi

    echo ""
}

# ---------------------------------------------------------------------------
# État persistant
# ---------------------------------------------------------------------------
load_state() {
    if [[ -f "$STATE_FILE" ]] && command -v jq &>/dev/null; then
        jq -r '.' "$STATE_FILE" 2>/dev/null || echo '{"status":"unknown"}'
    else
        echo '{"status":"unknown"}'
    fi
}

get_state_field() {
    local state_json="$1"
    local field="$2"
    local default="${3:-}"
    if command -v jq &>/dev/null; then
        local val
        val="$(echo "$state_json" | jq -r --arg f "$field" '.[$f] // empty' 2>/dev/null || true)"
        echo "${val:-$default}"
    else
        echo "$default"
    fi
}

save_state() {
    local status="$1"
    local since="$2"
    local url="${3:-}"
    local failures="${4:-0}"
    local prev_url="${5:-}"

    mkdir -p "$(dirname "$STATE_FILE")"

    local final_url="${url:-$prev_url}"
    local payload
    if command -v jq &>/dev/null; then
        payload="$(jq -n \
            --arg status "$status" \
            --arg since "$since" \
            --arg url "$final_url" \
            --argjson failures "$failures" \
            '{status: $status, since: $since, consecutive_failures: $failures} |
             if $url != "" then . + {url: $url} else . end')"
    else
        payload="{\"status\":\"$status\",\"since\":\"$since\",\"consecutive_failures\":$failures,\"url\":\"$final_url\"}"
    fi

    echo "$payload" > "$STATE_FILE"
}

# ---------------------------------------------------------------------------
# Vérification HTTP
# ---------------------------------------------------------------------------
check_site() {
    local url="$1"
    local http_code
    http_code="$(curl -s -o /dev/null -w "%{http_code}" \
        --max-time "$HTTP_TIMEOUT" \
        --user-agent "$USER_AGENT" \
        "$url" 2>/dev/null || echo "000")"

    if [[ "$http_code" -ge 100 && "$http_code" -lt 500 ]]; then
        return 0  # site vivant
    else
        return 1  # site mort
    fi
}

# ---------------------------------------------------------------------------
# Notification Discord
# ---------------------------------------------------------------------------
send_discord() {
    local webhook_url="$1"
    local payload="$2"

    local attempt=1
    while [[ $attempt -le $DISCORD_MAX_RETRIES ]]; do
        local http_code
        http_code="$(curl -s -o /dev/null -w "%{http_code}" \
            -X POST \
            -H "Content-Type: application/json" \
            -d "$payload" \
            "$webhook_url" 2>/dev/null || echo "000")"

        if [[ "$http_code" -ge 200 && "$http_code" -lt 300 ]]; then
            return 0
        fi

        log_debug "[Discord] Tentative $attempt/$DISCORD_MAX_RETRIES échouée (HTTP $http_code)."
        if [[ $attempt -lt $DISCORD_MAX_RETRIES ]]; then
            sleep "$DISCORD_RETRY_DELAY"
        fi
        ((attempt++))
    done

    log_warn "[Discord] Échec d'envoi après $DISCORD_MAX_RETRIES tentatives."
    return 1
}

notify_online() {
    local webhook_url="$1"
    local site_url="$2"
    local since="$3"
    local lang="$4"

    local ts
    ts="$(date '+%d/%m/%Y à %H:%M:%S')"

    local title body_link body_since footer
    title="$(t online_title "$lang")"
    body_link="$(t online_link "$lang")"
    body_since="$(t online_since "$lang")"
    footer="$(t footer "$lang")"

    local description
    description="$(t online_body "$lang")\n\n${body_link} ${site_url}\n${body_since} ${ts}"

    local payload
    payload="$(jq -n \
        --arg title "$title" \
        --arg desc "$description" \
        --argjson color "$COLOR_GREEN" \
        --arg footer "$footer" \
        --arg ts "$since" \
        '{embeds:[{title:$title,description:$desc,color:$color,footer:{text:$footer},timestamp:$ts}]}')"

    if send_discord "$webhook_url" "$payload"; then
        log_info "[Discord] Notification ONLINE envoyée."
    fi
}

notify_offline() {
    local webhook_url="$1"
    local since="$2"
    local lang="$3"

    local ts
    ts="$(date '+%d/%m/%Y à %H:%M:%S')"

    local title body_since footer
    title="$(t offline_title "$lang")"
    body_since="$(t offline_since "$lang")"
    footer="$(t footer "$lang")"

    local description
    description="$(t offline_body "$lang")\n\n${body_since} ${ts}"

    local payload
    payload="$(jq -n \
        --arg title "$title" \
        --arg desc "$description" \
        --argjson color "$COLOR_RED" \
        --arg footer "$footer" \
        --arg ts "$since" \
        '{embeds:[{title:$title,description:$desc,color:$color,footer:{text:$footer},timestamp:$ts}]}')"

    if send_discord "$webhook_url" "$payload"; then
        log_info "[Discord] Notification OFFLINE envoyée."
    fi
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
main() {
    load_env_local

    local offline_threshold="${UPTIME_OFFLINE_THRESHOLD:-$DEFAULT_OFFLINE_THRESHOLD}"
    local lang
    lang="$(get_lang)"

    # Charger l'état précédent
    local state_json
    state_json="$(load_state)"
    local prev_status prev_url prev_failures
    prev_status="$(get_state_field "$state_json" status unknown)"
    prev_url="$(get_state_field "$state_json" url "")"
    prev_failures="$(get_state_field "$state_json" consecutive_failures 0)"

    # 1. Résolution URL Tailscale
    local tailscale_url
    tailscale_url="$(resolve_tailscale_url)"

    if [[ -z "$tailscale_url" ]]; then
        if [[ -n "$prev_url" ]]; then
            log_warn "[Tailscale] Funnel non détecté via CLI — utilisation de l'URL en cache : $prev_url"
            tailscale_url="$prev_url"
        else
            log_error "[Tailscale] Aucune URL disponible. Démarrez le funnel : tailscale funnel --bg 8501"
            exit 0
        fi
    fi

    # 2. Webhook Discord (optionnel)
    local webhook_url
    webhook_url="$(get_webhook_url)"
    if [[ -z "$webhook_url" ]]; then
        log_warn "[Discord] Aucun webhook configuré — monitoring actif sans notifications."
    fi

    # 3. Ping HTTP
    local now
    now="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
    local current_status="offline"
    if check_site "$tailscale_url"; then
        current_status="online"
    fi

    log_info "URL : $tailscale_url | état précédent : $prev_status | état actuel : $current_status"

    # 4. Logique de transition avec anti-flapping
    if [[ "$current_status" == "online" ]]; then
        if [[ "$prev_status" != "online" ]]; then
            # Transition → ONLINE (immédiate)
            save_state "online" "$now" "$tailscale_url" 0 "$prev_url"
            if [[ -n "$webhook_url" ]]; then
                notify_online "$webhook_url" "$tailscale_url" "$now" "$lang"
            fi
        else
            # Toujours online : MAJ URL si changée
            if [[ "$tailscale_url" != "$prev_url" ]]; then
                save_state "online" "$now" "$tailscale_url" 0 "$prev_url"
            else
                log_debug "Pas de changement d'état (online)."
            fi
        fi
    else
        # Site DOWN
        local failures=$prev_failures
        if [[ "$prev_status" != "offline" ]]; then
            ((failures = prev_failures + 1))
        fi

        if [[ "$prev_status" == "offline" ]]; then
            log_debug "Pas de changement d'état (offline, $failures échecs)."
        elif [[ $failures -ge $offline_threshold ]]; then
            log_info "[Debounce] $failures/$offline_threshold checks en échec → notification OFFLINE."
            save_state "offline" "$now" "$tailscale_url" "$failures" "$prev_url"
            if [[ -n "$webhook_url" ]]; then
                notify_offline "$webhook_url" "$now" "$lang"
            fi
        else
            log_info "[Debounce] Check en échec $failures/$offline_threshold — pas encore de notification."
            save_state "$prev_status" "$now" "$tailscale_url" "$failures" "$prev_url"
        fi
    fi
}

main "$@"
