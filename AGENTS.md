# codex-troller Repository Agent Rules

1. For any actionable user request, call `autostart_get_mode` before deciding execution style.
2. If mode is `on`, continue codex-troller workflow and follow `get_session_status.next` strictly.
3. If mode is `off`, use normal Codex behavior unless the user explicitly asks to enable/start codex-troller.
4. On explicit enable/start request, call `autostart_set_mode` with `mode="on"` and begin/resume workflow.
5. On explicit disable/off request, call `autostart_set_mode` with `mode="off"` and return to default behavior.
6. Consultant/manager roles must never implement directly; only worker roles can execute `run_action`.
