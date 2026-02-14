#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN_PATH="$ROOT_DIR/.codex-mcp/bin/codex-mcp"
LAUNCHER_PATH="$ROOT_DIR/.codex-mcp/bin/codex-troller-launch"
SKILL_SRC="$ROOT_DIR/skills/codex-troller-autostart"
STATE_DIR_NAME=".codex-troller"

NON_INTERACTIVE="${AGENT_INSTALL_NON_INTERACTIVE:-0}"
INSTALL_SCOPE="${INSTALL_SCOPE:-}"
PLAYWRIGHT_CONSENT="${INSTALL_PLAYWRIGHT_MCP:-}"
TERMS_CONSENT="${INSTALL_TERMS_AGREED:-}"
TERMS_VERSION="2026-02-14"

PROFILE_OVERALL="${INSTALL_PROFILE_OVERALL:-}"
PROFILE_RESPONSE_NEED="${INSTALL_PROFILE_RESPONSE_NEED:-}"
PROFILE_TECHNICAL_DEPTH="${INSTALL_PROFILE_TECHNICAL_DEPTH:-}"
PROFILE_DOMAIN_HINTS="${INSTALL_PROFILE_DOMAIN_HINTS:-}"

lower() {
  printf '%s' "$1" | tr '[:upper:]' '[:lower:]'
}

normalize_yes_no() {
  case "$(lower "$(echo "$1" | xargs)")" in
    y|yes|agree|agreed) echo "yes" ;;
    n|no|disagree) echo "no" ;;
    *) echo "" ;;
  esac
}

normalize_install_scope() {
  case "$(lower "$(echo "$1" | xargs)")" in
    global|g|1) echo "global" ;;
    local|l|2) echo "local" ;;
    *) echo "" ;;
  esac
}

normalize_level() {
  case "$(lower "$(echo "$1" | xargs)")" in
    beginner|novice|1) echo "beginner" ;;
    intermediate|mid|2) echo "intermediate" ;;
    advanced|expert|3) echo "advanced" ;;
    *) echo "" ;;
  esac
}

normalize_response_need() {
  case "$(lower "$(echo "$1" | xargs)")" in
    low|light|1) echo "low" ;;
    balanced|normal|2) echo "balanced" ;;
    high|detailed|3) echo "high" ;;
    *) echo "" ;;
  esac
}

normalize_technical_depth() {
  case "$(lower "$(echo "$1" | xargs)")" in
    abstract|high-level|1) echo "abstract" ;;
    balanced|normal|2) echo "balanced" ;;
    technical|deep|3) echo "technical" ;;
    *) echo "" ;;
  esac
}

normalize_domain_key() {
  case "$(lower "$(echo "$1" | xargs)")" in
    frontend|front|ui|ux) echo "frontend" ;;
    backend|back|api|server) echo "backend" ;;
    db|database|data) echo "db" ;;
    security|auth|permission) echo "security" ;;
    asset|assets|content|media) echo "asset" ;;
    ai|ml|llm|vllm|deep-learning|deeplearning) echo "ai_ml" ;;
    infra|devops|ops|ci|cd|platform) echo "infra" ;;
    game|gameplay) echo "game" ;;
    *) echo "" ;;
  esac
}

prompt_yes_no() {
  local message="$1"
  local default_value="$2"
  local reply=""
  while true; do
    read -r -p "$message " reply || true
    if [[ -z "$(echo "$reply" | xargs)" ]]; then
      reply="$default_value"
    fi
    local normalized
    normalized="$(normalize_yes_no "$reply")"
    if [[ "$normalized" == "yes" || "$normalized" == "no" ]]; then
      echo "$normalized"
      return 0
    fi
    echo "Please answer yes or no."
  done
}

prompt_scope() {
  local reply=""
  while true; do
    read -r -p "Install scope? [global/local] (default: global): " reply || true
    if [[ -z "$(echo "$reply" | xargs)" ]]; then
      reply="global"
    fi
    local normalized
    normalized="$(normalize_install_scope "$reply")"
    if [[ "$normalized" == "global" || "$normalized" == "local" ]]; then
      echo "$normalized"
      return 0
    fi
    echo "Please answer global or local."
  done
}

strip_mcp_section() {
  local config_path="$1"
  local section="$2"
  local tmp_file
  tmp_file="$(mktemp)"
  awk -v sec="[""$section""]" '
  BEGIN { skip = 0 }
  {
    if ($0 == sec) {
      skip = 1
      next
    }
    if (skip == 1 && $0 ~ /^\[/) {
      skip = 0
    }
    if (skip == 0) {
      print $0
    }
  }
  ' "$config_path" >"$tmp_file"
  mv "$tmp_file" "$config_path"
}

echo "[agent-install] start"
echo "[agent-install] root: $ROOT_DIR"

if [[ "$NON_INTERACTIVE" != "1" ]]; then
  cat <<'EOF'
[agent-install] LLM installer consent gate
Please review and confirm all items below:
1) This repository software is not sufficiently validated.
2) You accept full responsibility for any issues/damages caused by use.
3) You acknowledge the project license is GNU GPL v3.0.
EOF
  TERMS_CONSENT="$(prompt_yes_no "Do you confirm all 3 items above? [yes/no] (default: no):" "no")"
else
  TERMS_CONSENT="$(normalize_yes_no "$TERMS_CONSENT")"
fi

if [[ "$TERMS_CONSENT" != "yes" ]]; then
  echo "[agent-install] aborted: terms were not accepted" >&2
  exit 1
fi

if [[ "$NON_INTERACTIVE" != "1" ]]; then
  INSTALL_SCOPE="$(prompt_scope)"
else
  INSTALL_SCOPE="$(normalize_install_scope "$INSTALL_SCOPE")"
  if [[ "$INSTALL_SCOPE" == "" ]]; then
    INSTALL_SCOPE="global"
  fi
fi

CODEX_HOME_DEFAULT="$HOME/.codex"
if [[ "$INSTALL_SCOPE" == "local" ]]; then
  CODEX_HOME_DEFAULT="$ROOT_DIR/.codex"
fi

CODEX_HOME_PATH="${CODEX_HOME_PATH:-$CODEX_HOME_DEFAULT}"
CONFIG_PATH="${CODEX_CONFIG_PATH:-$CODEX_HOME_PATH/config.toml}"
SKILL_DST="$CODEX_HOME_PATH/skills/codex-troller-autostart"
STATE_DIR="$CODEX_HOME_PATH/$STATE_DIR_NAME"
LEGACY_STATE_DIR="$CODEX_HOME_PATH/codex-troller"
DEFAULT_PROFILE_PATH="$STATE_DIR/default_user_profile.json"
LEGACY_PROFILE_PATH="$LEGACY_STATE_DIR/default_user_profile.json"
PROFILE_PATH="${CODEX_TROLLER_PROFILE_PATH:-$DEFAULT_PROFILE_PATH}"

# Normalize legacy profile path to hidden-state default.
if [[ "$PROFILE_PATH" == "$LEGACY_PROFILE_PATH" ]]; then
  PROFILE_PATH="$DEFAULT_PROFILE_PATH"
fi

echo "[agent-install] scope: $INSTALL_SCOPE"

if [[ "$NON_INTERACTIVE" != "1" ]]; then
  PLAYWRIGHT_CONSENT="$(prompt_yes_no "Install Playwright MCP integration now? [yes/no] (default: no):" "no")"
else
  PLAYWRIGHT_CONSENT="$(normalize_yes_no "$PLAYWRIGHT_CONSENT")"
  if [[ "$PLAYWRIGHT_CONSENT" == "" ]]; then
    PLAYWRIGHT_CONSENT="no"
  fi
fi

make -C "$ROOT_DIR" setup >/dev/null

if [[ "$PLAYWRIGHT_CONSENT" == "yes" ]]; then
  if ! command -v npx >/dev/null 2>&1; then
    echo "[agent-install] failed: Playwright MCP requires Node.js+npx, but npx is not available." >&2
    exit 1
  fi

  echo "[agent-install] installing Playwright runtime dependencies..."
  if [[ "$(uname -s)" == "Linux" ]]; then
    # Linux needs browser binaries + OS dependencies.
    if ! npx -y playwright@latest install --with-deps chromium firefox webkit; then
      echo "[agent-install] warning: failed to install Linux OS dependencies (sudo may be required)." >&2
      echo "[agent-install] retrying browser-only Playwright install..." >&2
      if ! npx -y playwright@latest install chromium firefox webkit; then
        echo "[agent-install] failed: could not install Playwright browser binaries." >&2
        exit 1
      fi
    fi
  else
    if ! npx -y playwright@latest install chromium firefox webkit; then
      echo "[agent-install] failed: could not install Playwright browser binaries." >&2
      exit 1
    fi
  fi
fi

# Migrate legacy non-hidden state directory if it exists.
if [[ -d "$LEGACY_STATE_DIR" && "$LEGACY_STATE_DIR" != "$STATE_DIR" ]]; then
  mkdir -p "$STATE_DIR"
  shopt -s dotglob nullglob
  for entry in "$LEGACY_STATE_DIR"/*; do
    base="$(basename "$entry")"
    if [[ ! -e "$STATE_DIR/$base" ]]; then
      mv "$entry" "$STATE_DIR/$base"
    fi
  done
  shopt -u dotglob nullglob
  rmdir "$LEGACY_STATE_DIR" 2>/dev/null || true
fi

mkdir -p "$(dirname "$CONFIG_PATH")"
touch "$CONFIG_PATH"
mkdir -p "$(dirname "$PROFILE_PATH")"

if [[ "$NON_INTERACTIVE" != "1" ]]; then
  local_overall_raw=""
  local_response_raw=""
  local_technical_raw=""
  local_domain_raw=""
  while true; do
    read -r -p "Your software/build expertise level? [beginner/intermediate/advanced] (default: intermediate): " local_overall_raw || true
    if [[ -z "$(echo "$local_overall_raw" | xargs)" ]]; then
      local_overall_raw="intermediate"
    fi
    PROFILE_OVERALL="$(normalize_level "$local_overall_raw")"
    [[ -n "$PROFILE_OVERALL" ]] && break
    echo "Please choose beginner, intermediate, or advanced."
  done
  while true; do
    read -r -p "How much detail should the consultant ask from you? [low/balanced/high] (default: balanced): " local_response_raw || true
    if [[ -z "$(echo "$local_response_raw" | xargs)" ]]; then
      local_response_raw="balanced"
    fi
    PROFILE_RESPONSE_NEED="$(normalize_response_need "$local_response_raw")"
    [[ -n "$PROFILE_RESPONSE_NEED" ]] && break
    echo "Please choose low, balanced, or high."
  done
  while true; do
    read -r -p "Preferred explanation depth? [abstract/balanced/technical] (default: balanced): " local_technical_raw || true
    if [[ -z "$(echo "$local_technical_raw" | xargs)" ]]; then
      local_technical_raw="balanced"
    fi
    PROFILE_TECHNICAL_DEPTH="$(normalize_technical_depth "$local_technical_raw")"
    [[ -n "$PROFILE_TECHNICAL_DEPTH" ]] && break
    echo "Please choose abstract, balanced, or technical."
  done
  read -r -p "Optional domains you know well (comma-separated, e.g. backend,frontend,security): " local_domain_raw || true
  PROFILE_DOMAIN_HINTS="$local_domain_raw"
else
  PROFILE_OVERALL="$(normalize_level "$PROFILE_OVERALL")"
  PROFILE_RESPONSE_NEED="$(normalize_response_need "$PROFILE_RESPONSE_NEED")"
  PROFILE_TECHNICAL_DEPTH="$(normalize_technical_depth "$PROFILE_TECHNICAL_DEPTH")"
  PROFILE_DOMAIN_HINTS="${PROFILE_DOMAIN_HINTS:-}"
fi

if [[ "$PROFILE_OVERALL" == "" ]]; then
  PROFILE_OVERALL="intermediate"
fi
if [[ "$PROFILE_RESPONSE_NEED" == "" ]]; then
  PROFILE_RESPONSE_NEED="balanced"
fi
if [[ "$PROFILE_TECHNICAL_DEPTH" == "" ]]; then
  PROFILE_TECHNICAL_DEPTH="balanced"
fi

declare -A DOMAIN_MAP=()
IFS=',' read -r -a DOMAIN_ITEMS <<<"$PROFILE_DOMAIN_HINTS"
for item in "${DOMAIN_ITEMS[@]}"; do
  token="$(echo "$item" | xargs)"
  [[ -z "$token" ]] && continue
  raw_key="$token"
  raw_level="$PROFILE_OVERALL"
  if [[ "$token" == *"="* ]]; then
    raw_key="${token%%=*}"
    raw_level="${token#*=}"
  fi
  domain_key="$(normalize_domain_key "$raw_key")"
  domain_level="$(normalize_level "$raw_level")"
  if [[ -z "$domain_key" ]]; then
    continue
  fi
  if [[ -z "$domain_level" ]]; then
    domain_level="$PROFILE_OVERALL"
  fi
  DOMAIN_MAP["$domain_key"]="$domain_level"
done

{
  echo "{"
  echo "  \"overall\": \"${PROFILE_OVERALL}\","
  echo "  \"response_need\": \"${PROFILE_RESPONSE_NEED}\","
  echo "  \"technical_depth\": \"${PROFILE_TECHNICAL_DEPTH}\","
  echo "  \"domain_knowledge\": {"
  idx=0
  total="${#DOMAIN_MAP[@]}"
  for key in "${!DOMAIN_MAP[@]}"; do
    idx=$((idx + 1))
    comma=","
    if [[ "$idx" -ge "$total" ]]; then
      comma=""
    fi
    echo "    \"${key}\": \"${DOMAIN_MAP[$key]}\"${comma}"
  done
  echo "  }"
  echo "}"
} >"$PROFILE_PATH"

mkdir -p "$(dirname "$LAUNCHER_PATH")"
cat >"$LAUNCHER_PATH" <<EOF
#!/usr/bin/env bash
set -euo pipefail
export CODEX_TROLLER_DEFAULT_PROFILE_PATH="$PROFILE_PATH"
exec "$BIN_PATH"
EOF
chmod +x "$LAUNCHER_PATH"

strip_mcp_section "$CONFIG_PATH" "mcp_servers.codex-troller"
strip_mcp_section "$CONFIG_PATH" "mcp_servers.playwright"

{
  echo
  echo "[mcp_servers.codex-troller]"
  echo "command = \"$LAUNCHER_PATH\""
} >>"$CONFIG_PATH"

if [[ "$PLAYWRIGHT_CONSENT" == "yes" ]]; then
  {
    echo
    echo "[mcp_servers.playwright]"
    echo 'command = "npx"'
    echo 'args = ["-y", "@playwright/mcp@latest"]'
  } >>"$CONFIG_PATH"
fi

if ! grep -q '^\[mcp_servers\.codex-troller\]$' "$CONFIG_PATH"; then
  echo "[agent-install] failed to write MCP section" >&2
  exit 1
fi

if ! grep -q "^command = \"$LAUNCHER_PATH\"$" "$CONFIG_PATH"; then
  echo "[agent-install] failed to write command path" >&2
  exit 1
fi

if [[ ! -f "$SKILL_SRC/SKILL.md" ]]; then
  echo "[agent-install] missing skill source: $SKILL_SRC/SKILL.md" >&2
  exit 1
fi

mkdir -p "$(dirname "$SKILL_DST")"
rm -rf "$SKILL_DST"
cp -R "$SKILL_SRC" "$SKILL_DST"

CONSENT_LOG="$STATE_DIR/install-consent.log"
mkdir -p "$(dirname "$CONSENT_LOG")"
{
  echo "timestamp: $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
  echo "terms_version: $TERMS_VERSION"
  echo "terms_assertions:"
  echo "  - unverified_software=true"
  echo "  - user_assumes_responsibility=true"
  echo "  - license_gnu_gpl_v3_ack=true"
  echo "scope: $INSTALL_SCOPE"
  echo "terms_accepted: $TERMS_CONSENT"
  echo "playwright_mcp: $PLAYWRIGHT_CONSENT"
  echo "profile_path: $PROFILE_PATH"
} >"$CONSENT_LOG"

echo "[agent-install] done"
echo "[agent-install] registered mcp_servers.codex-troller"
if [[ "$PLAYWRIGHT_CONSENT" == "yes" ]]; then
  echo "[agent-install] registered mcp_servers.playwright"
fi
echo "[agent-install] installed skill: codex-troller-autostart"
if [[ "$INSTALL_SCOPE" == "local" ]]; then
  echo "[agent-install] next: restart Codex in this folder."
else
  echo "[agent-install] next: restart Codex. It will be available from any folder."
fi
