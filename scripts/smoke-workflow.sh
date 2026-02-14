#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN="$ROOT_DIR/.codex-mcp/bin/codex-mcp"
SESSION_ID="${1:-smoke-$(date +%s%N)}"
SMOKE_GOAL="${SMOKE_GOAL:-smoke workflow}"
SMOKE_SCOPE="${SMOKE_SCOPE:-internal/server}"
SMOKE_CONSTRAINT="${SMOKE_CONSTRAINT:-로컬만 사용}"
SMOKE_CRITERIA="${SMOKE_CRITERIA:-테스트 통과}"
SMOKE_TAGS_CSV="${SMOKE_TAGS_CSV:-smoke,workflow}"
SMOKE_AVAILABLE_MCPS="${SMOKE_AVAILABLE_MCPS:-}"
SMOKE_AVAILABLE_MCP_TOOLS="${SMOKE_AVAILABLE_MCP_TOOLS:-}"

if [[ ! -x "$BIN" ]]; then
  make -C "$ROOT_DIR" build >/dev/null
fi

ID=1
LAST_STRUCTURED=""
OUTPUT_LOG=""

append_output() {
  local line="$1"
  OUTPUT_LOG+="$line"$'\n'
}

rpc_raw() {
  local req="$1"
  printf '%s\n' "$req" | "$BIN"
}

rpc_call() {
  local method="$1"
  local params_json
  if [[ $# -ge 2 ]]; then
    params_json="$2"
  else
    params_json='{}'
  fi
  local req
  local resp

  req="$(jq -cn --argjson id "$ID" --arg method "$method" --argjson params "$params_json" '{jsonrpc:"2.0",id:$id,method:$method,params:$params}')"
  ID=$((ID + 1))
  resp="$(rpc_raw "$req")"
  append_output "$resp"

  if echo "$resp" | jq -e '.error != null' >/dev/null; then
    echo "$OUTPUT_LOG"
    echo "smoke failed: rpc error detected for method $method" >&2
    exit 1
  fi

  LAST_STRUCTURED="$(echo "$resp" | jq -c '.result.structuredContent // {}')"
}

tool_call() {
  local name="$1"
  local args_json="$2"
  local params
  params="$(jq -cn --arg name "$name" --argjson args "$args_json" '{name:$name,arguments:$args}')"
  rpc_call "tools/call" "$params"
}

csv_to_json_array() {
  local csv="$1"
  local out='[]'
  if [[ -n "$(echo "$csv" | tr -d '[:space:]')" ]]; then
    out="$(jq -cn --arg csv "$csv" '$csv | split(",") | map(gsub("^\\s+|\\s+$";"")) | map(select(length > 0))')"
  fi
  if [[ "$out" == "[]" ]]; then
    out='["smoke"]'
  fi
  printf '%s' "$out"
}

csv_to_optional_json_array() {
  local csv="$1"
  if [[ -z "$(echo "$csv" | tr -d '[:space:]')" ]]; then
    printf '[]'
    return 0
  fi
  jq -cn --arg csv "$csv" '$csv | split(",") | map(gsub("^\\s+|\\s+$";"")) | map(select(length > 0))'
}

TAGS_JSON="$(csv_to_json_array "$SMOKE_TAGS_CSV")"
AVAILABLE_MCPS_JSON="$(csv_to_optional_json_array "$SMOKE_AVAILABLE_MCPS")"
AVAILABLE_MCP_TOOLS_JSON="$(csv_to_optional_json_array "$SMOKE_AVAILABLE_MCP_TOOLS")"
RAW_INTENT="$(printf '목표: %s\n범위: %s\n제약: %s\n성공기준: %s' "$SMOKE_GOAL" "$SMOKE_SCOPE" "$SMOKE_CONSTRAINT" "$SMOKE_CRITERIA")"

rpc_call "initialize" "{}"
tool_call "ingest_intent" "$(jq -cn --arg sid "$SESSION_ID" --arg raw "$RAW_INTENT" '{session_id:$sid,raw_intent:$raw}')"
tool_call "council_start_briefing" "$(jq -cn --arg sid "$SESSION_ID" '{session_id:$sid}')"
mapfile -t COUNCIL_ROLES < <(echo "$LAST_STRUCTURED" | jq -r '.roles[]?.role')
if [[ ${#COUNCIL_ROLES[@]} -eq 0 ]]; then
  echo "$OUTPUT_LOG"
  echo "smoke failed: no council roles returned from council_start_briefing" >&2
  exit 1
fi
FLOOR_ROLE="${COUNCIL_ROLES[0]}"
for role in "${COUNCIL_ROLES[@]}"; do
  if [[ "$role" == "ux_director" ]]; then
    FLOOR_ROLE="ux_director"
    break
  fi
done

for role in "${COUNCIL_ROLES[@]}"; do
  tool_call "council_submit_brief" "$(jq -cn --arg sid "$SESSION_ID" --arg role "$role" '{session_id:$sid,role:$role,priority:"core",contribution:($role + " contribution"),quick_decisions:"none"}')"
done

tool_call "council_summarize_briefs" "$(jq -cn --arg sid "$SESSION_ID" '{session_id:$sid}')"
mapfile -t TOPIC_IDS < <(echo "$LAST_STRUCTURED" | jq -r '.topics[]?.id')
if [[ ${#TOPIC_IDS[@]} -eq 0 ]]; then
  echo "$OUTPUT_LOG"
  echo "smoke failed: no council topics found" >&2
  exit 1
fi

for topic_id in "${TOPIC_IDS[@]}"; do
  tool_call "council_request_floor" "$(jq -cn --arg sid "$SESSION_ID" --argjson topic "$topic_id" --arg role "$FLOOR_ROLE" '{session_id:$sid,topic_id:$topic,role:$role,reason:"smoke kickoff"}')"
  request_id="$(echo "$LAST_STRUCTURED" | jq -r '.request_id')"
  if [[ -z "$request_id" || "$request_id" == "null" ]]; then
    echo "$OUTPUT_LOG"
    echo "smoke failed: missing request_id for topic $topic_id" >&2
    exit 1
  fi
  tool_call "council_grant_floor" "$(jq -cn --arg sid "$SESSION_ID" --argjson request "$request_id" '{session_id:$sid,request_id:$request}')"
  tool_call "council_publish_statement" "$(jq -cn --arg sid "$SESSION_ID" --argjson request "$request_id" '{session_id:$sid,request_id:$request,content:"smoke statement"}')"
  for role in "${COUNCIL_ROLES[@]}"; do
    if [[ "$role" == "$FLOOR_ROLE" ]]; then
      continue
    fi
    tool_call "council_respond_topic" "$(jq -cn --arg sid "$SESSION_ID" --argjson topic "$topic_id" --arg role "$role" '{session_id:$sid,topic_id:$topic,role:$role,decision:"pass",content:"pass"}')"
  done
  tool_call "council_close_topic" "$(jq -cn --arg sid "$SESSION_ID" --argjson topic "$topic_id" '{session_id:$sid,topic_id:$topic}')"
done

tool_call "council_finalize_consensus" "$(jq -cn --arg sid "$SESSION_ID" '{session_id:$sid}')"
tool_call "clarify_intent" "$(jq -cn --arg sid "$SESSION_ID" --argjson tags "$TAGS_JSON" --arg criteria "$SMOKE_CRITERIA" '{session_id:$sid,answers:{goal:"Reduce login failure rate and prevent recurrence",scope:"internal/server",constraints:"local environment only, no destructive changes",requirement_tags:$tags,success_criteria:[$criteria]}}')"
tool_call "clarify_intent" "$(jq -cn --arg sid "$SESSION_ID" '{session_id:$sid,answers:{proposal_feedback:"go as-is"}}')"
tool_call "generate_plan" "$(jq -cn --arg sid "$SESSION_ID" '{session_id:$sid}')"
tool_call "generate_mockup" "$(jq -cn --arg sid "$SESSION_ID" '{session_id:$sid}')"
tool_call "approve_plan" "$(jq -cn --arg sid "$SESSION_ID" --argjson tags "$TAGS_JSON" --arg criteria "$SMOKE_CRITERIA" '{session_id:$sid,approved:true,requirement_tags:$tags,success_criteria:[$criteria]}')"
tool_call "run_action" "$(jq -cn --arg sid "$SESSION_ID" '{session_id:$sid,commands:["echo smoke"],dry_run:true}')"
tool_call "verify_result" "$(jq -cn --arg sid "$SESSION_ID" --argjson mcps "$AVAILABLE_MCPS_JSON" --argjson tools "$AVAILABLE_MCP_TOOLS_JSON" '{session_id:$sid,commands:["echo ok"],available_mcps:$mcps,available_mcp_tools:$tools}')"
VERIFY_NEXT="$(echo "$LAST_STRUCTURED" | jq -r '.next_step // ""')"
if [[ "$VERIFY_NEXT" == "visual_review" ]]; then
  tool_call "visual_review" "$(jq -cn --arg sid "$SESSION_ID" '{session_id:$sid,artifacts:["smoke://render/home"],findings:["smoke visual check pass"],reviewer_notes:"smoke visual reviewer",ux_director_summary:"smoke UX director meeting completed",ux_decision:"pass"}')"
fi
tool_call "record_user_feedback" "$(jq -cn --arg sid "$SESSION_ID" '{session_id:$sid,approved:true,feedback:"smoke approved"}')"
tool_call "summarize" "$(jq -cn --arg sid "$SESSION_ID" '{session_id:$sid}')"
tool_call "get_session_status" "$(jq -cn --arg sid "$SESSION_ID" '{session_id:$sid}')"

echo "$OUTPUT_LOG"
if ! echo "$LAST_STRUCTURED" | jq -e '.step == "summarized"' >/dev/null; then
  echo "smoke failed: summarized step not reached" >&2
  exit 1
fi

echo "smoke success: workflow reached summarized"
