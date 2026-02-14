# codex-troller Repository Agent Rules

1. For any user task request, run `codex-troller-mode-router` first.
2. `codex-troller-mode-router` must check `autostart_get_mode` before routing.
3. If mode is `on`, follow `codex-troller-autostart` workflow end-to-end.
4. If mode is `off`, use normal Codex behavior unless user explicitly enables codex-troller mode.
5. Respect strict execution separation:
   - consultant/manager roles do planning/review only,
   - worker roles execute `run_action`.
6. Before each workflow tool call after `start_interview`, check `get_session_status.next` and follow it exactly.
