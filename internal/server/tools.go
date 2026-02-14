package server

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

type toolSchema struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

func toolListResponse() map[string]any {
	return map[string]any{
		"tools": []toolSchema{
			newTool(
				"start_interview",
				"Workflow entrypoint. Initialize session and generate interview questions",
				map[string]any{
					"type": "object",
					"properties": map[string]any{
						"session_id": map[string]any{"type": "string"},
						"raw_intent": map[string]any{"type": "string"},
						"user_profile": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"overall":         map[string]any{"type": "string", "description": "unknown|beginner|intermediate|advanced"},
								"response_need":   map[string]any{"type": "string", "description": "low|balanced|high"},
								"technical_depth": map[string]any{"type": "string", "description": "abstract|balanced|technical"},
								"domain_knowledge": map[string]any{
									"type":                 "object",
									"additionalProperties": map[string]any{"type": "string"},
								},
							},
						},
						"available_mcps": map[string]any{
							"type":        "array",
							"items":       map[string]any{"type": "string"},
							"description": "List of available MCP server names in current runtime",
						},
						"available_mcp_tools": map[string]any{
							"type":        "array",
							"items":       map[string]any{"type": "string"},
							"description": "List of available MCP tool names in current runtime",
						},
					},
				},
			),
			newTool(
				"ingest_intent",
				"Collect raw user requirement text and normalize into base intent structure",
				map[string]any{
					"type": "object",
					"properties": map[string]any{
						"raw_intent": map[string]any{"type": "string", "description": "Raw user request text"},
						"session_id": map[string]any{"type": "string"},
						"user_profile": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"overall":         map[string]any{"type": "string"},
								"response_need":   map[string]any{"type": "string"},
								"technical_depth": map[string]any{"type": "string"},
								"domain_knowledge": map[string]any{
									"type":                 "object",
									"additionalProperties": map[string]any{"type": "string"},
								},
							},
						},
						"available_mcps": map[string]any{
							"type":        "array",
							"items":       map[string]any{"type": "string"},
							"description": "List of available MCP server names in current runtime",
						},
						"available_mcp_tools": map[string]any{
							"type":        "array",
							"items":       map[string]any{"type": "string"},
							"description": "List of available MCP tool names in current runtime",
						},
					},
					"required": []string{"raw_intent"},
				},
			),
			newTool(
				"clarify_intent",
				"Fill missing intent fields and run consultant outline loop (one focused question per turn)",
				map[string]any{
					"type": "object",
					"properties": map[string]any{
						"session_id": map[string]any{"type": "string"},
						"answers": map[string]any{
							"type":        "object",
							"description": "Examples: goal/scope/constraints/success_criteria + proposal_feedback + knowledge_level/response_need/technical_depth/domain_knowledge",
						},
					},
					"required": []string{"session_id"},
				},
			),
			newTool(
				"generate_plan",
				"Generate executable plan from clarified requirements",
				map[string]any{
					"type": "object",
					"properties": map[string]any{
						"session_id": map[string]any{"type": "string"},
					},
					"required": []string{"session_id"},
				},
			),
			newTool(
				"generate_mockup",
				"Generate quick mockup (text prototype) from current intent/plan",
				map[string]any{
					"type": "object",
					"properties": map[string]any{
						"session_id": map[string]any{"type": "string"},
					},
					"required": []string{"session_id"},
				},
			),
			newTool(
				"approve_plan",
				"Approve or reject generated plan",
				map[string]any{
					"type": "object",
					"properties": map[string]any{
						"session_id": map[string]any{"type": "string"},
						"approved":   map[string]any{"type": "boolean"},
						"notes":      map[string]any{"type": "string"},
						"requirement_tags": map[string]any{
							"type":        "array",
							"items":       map[string]any{"type": "string"},
							"description": "Requirement tags (e.g., auth, tests, performance)",
						},
						"success_criteria": map[string]any{
							"type":        "array",
							"items":       map[string]any{"type": "string"},
							"description": "Requirement success criteria aligned with intent success_criteria",
						},
					},
					"required": []string{"session_id", "approved"},
				},
			),
			newTool(
				"reconcile_session_state",
				"Reconcile persisted session state with current repo state (git + footprint)",
				map[string]any{
					"type": "object",
					"properties": map[string]any{
						"session_id": map[string]any{"type": "string"},
						"mode": map[string]any{
							"type": "string",
							"enum": []string{"check", "keep_context", "restart_context"},
						},
					},
					"required": []string{"session_id"},
				},
			),
			newTool(
				"set_agent_routing_policy",
				"Set role-based model routing policy",
				map[string]any{
					"type": "object",
					"properties": map[string]any{
						"session_id":             map[string]any{"type": "string"},
						"client_interview_model": map[string]any{"type": "string"},
						"orchestrator_model":     map[string]any{"type": "string"},
						"reviewer_model":         map[string]any{"type": "string"},
						"worker_model":           map[string]any{"type": "string"},
					},
					"required": []string{"session_id"},
				},
			),
			newTool(
				"get_agent_routing_policy",
				"Get current role-based model routing policy",
				map[string]any{
					"type": "object",
					"properties": map[string]any{
						"session_id": map[string]any{"type": "string"},
					},
					"required": []string{"session_id"},
				},
			),
			newTool(
				"council_start_briefing",
				"Start manager council parallel briefing round",
				map[string]any{
					"type": "object",
					"properties": map[string]any{
						"session_id": map[string]any{"type": "string"},
					},
					"required": []string{"session_id"},
				},
			),
			newTool(
				"council_submit_brief",
				"Submit briefing for each manager role (parallel execution result input)",
				map[string]any{
					"type": "object",
					"properties": map[string]any{
						"session_id":      map[string]any{"type": "string"},
						"role":            map[string]any{"type": "string"},
						"priority":        map[string]any{"type": "string"},
						"contribution":    map[string]any{"type": "string"},
						"quick_decisions": map[string]any{"type": "string"},
						"topic_proposals": map[string]any{
							"type":  "array",
							"items": map[string]any{"type": "string"},
						},
					},
					"required": []string{"session_id", "role", "priority", "contribution"},
				},
			),
			newTool(
				"council_summarize_briefs",
				"Moderator summarizes briefings and builds agenda topics",
				map[string]any{
					"type": "object",
					"properties": map[string]any{
						"session_id": map[string]any{"type": "string"},
					},
					"required": []string{"session_id"},
				},
			),
			newTool(
				"council_request_floor",
				"Manager requests speaking floor (topic_id required)",
				map[string]any{
					"type": "object",
					"properties": map[string]any{
						"session_id": map[string]any{"type": "string"},
						"topic_id":   map[string]any{"type": "number"},
						"role":       map[string]any{"type": "string"},
						"reason":     map[string]any{"type": "string"},
					},
					"required": []string{"session_id", "topic_id", "role"},
				},
			),
			newTool(
				"council_grant_floor",
				"Moderator grants speaking floor",
				map[string]any{
					"type": "object",
					"properties": map[string]any{
						"session_id": map[string]any{"type": "string"},
						"request_id": map[string]any{"type": "number"},
					},
					"required": []string{"session_id", "request_id"},
				},
			),
			newTool(
				"council_publish_statement",
				"Publish floor-granted manager statement and propagate to other roles",
				map[string]any{
					"type": "object",
					"properties": map[string]any{
						"session_id": map[string]any{"type": "string"},
						"request_id": map[string]any{"type": "number"},
						"content":    map[string]any{"type": "string"},
					},
					"required": []string{"session_id", "request_id", "content"},
				},
			),
			newTool(
				"council_respond_topic",
				"Record each manager's pass/raise response per topic",
				map[string]any{
					"type": "object",
					"properties": map[string]any{
						"session_id": map[string]any{"type": "string"},
						"topic_id":   map[string]any{"type": "number"},
						"role":       map[string]any{"type": "string"},
						"decision":   map[string]any{"type": "string", "enum": []string{"pass", "raise"}},
						"content":    map[string]any{"type": "string"},
					},
					"required": []string{"session_id", "topic_id", "role", "decision"},
				},
			),
			newTool(
				"council_close_topic",
				"Close topic when all manager roles pass",
				map[string]any{
					"type": "object",
					"properties": map[string]any{
						"session_id": map[string]any{"type": "string"},
						"topic_id":   map[string]any{"type": "number"},
					},
					"required": []string{"session_id", "topic_id"},
				},
			),
			newTool(
				"council_finalize_consensus",
				"Finalize consensus after all topics are closed",
				map[string]any{
					"type": "object",
					"properties": map[string]any{
						"session_id": map[string]any{"type": "string"},
					},
					"required": []string{"session_id"},
				},
			),
			newTool(
				"council_get_status",
				"Get council discussion status/topics/messages",
				map[string]any{
					"type": "object",
					"properties": map[string]any{
						"session_id":    map[string]any{"type": "string"},
						"message_limit": map[string]any{"type": "number"},
					},
					"required": []string{"session_id"},
				},
			),
			newTool(
				"validate_workflow_transition",
				"Validate whether workflow transition is allowed",
				map[string]any{
					"type": "object",
					"properties": map[string]any{
						"session_id":   map[string]any{"type": "string"},
						"current_step": map[string]any{"type": "string"},
						"next_step":    map[string]any{"type": "string"},
					},
					"required": []string{"session_id", "current_step", "next_step"},
				},
			),
			newTool(
				"run_action",
				"Run allowlisted commands from approved plan",
				map[string]any{
					"type": "object",
					"properties": map[string]any{
						"session_id": map[string]any{"type": "string"},
						"commands": map[string]any{
							"type":  "array",
							"items": map[string]any{"type": "string"},
						},
						"dry_run":     map[string]any{"type": "boolean"},
						"timeout_sec": map[string]any{"type": "number", "default": 30},
					},
					"required": []string{"session_id", "commands"},
				},
			),
			newTool(
				"verify_result",
				"Run verification commands",
				map[string]any{
					"type": "object",
					"properties": map[string]any{
						"session_id": map[string]any{"type": "string"},
						"commands": map[string]any{
							"type":  "array",
							"items": map[string]any{"type": "string"},
						},
						"timeout_sec": map[string]any{"type": "number", "default": 120},
						"available_mcps": map[string]any{
							"type":        "array",
							"items":       map[string]any{"type": "string"},
							"description": "MCP servers that can render visuals (e.g., playwright)",
						},
						"available_mcp_tools": map[string]any{
							"type":        "array",
							"items":       map[string]any{"type": "string"},
							"description": "MCP tools that can render visuals (e.g., playwright.screenshot)",
						},
					},
					"required": []string{"session_id"},
				},
			),
			newTool(
				"visual_review",
				"Run Visual Reviewer checks and record UX Director meeting outcome",
				map[string]any{
					"type": "object",
					"properties": map[string]any{
						"session_id": map[string]any{"type": "string"},
						"available_mcps": map[string]any{
							"type":  "array",
							"items": map[string]any{"type": "string"},
						},
						"available_mcp_tools": map[string]any{
							"type":  "array",
							"items": map[string]any{"type": "string"},
						},
						"artifacts": map[string]any{
							"type":        "array",
							"items":       map[string]any{"type": "string"},
							"description": "Rendered artifact paths/URLs (screenshots, recordings, etc.)",
						},
						"findings": map[string]any{
							"type":        "array",
							"items":       map[string]any{"type": "string"},
							"description": "Visual Reviewer findings",
						},
						"reviewer_notes": map[string]any{
							"type":        "string",
							"description": "Visual quality/behavior verification notes",
						},
						"ux_director_summary": map[string]any{
							"type":        "string",
							"description": "UX Director meeting summary based on built artifact",
						},
						"ux_decision": map[string]any{
							"type": "string",
							"enum": []string{"pass", "raise"},
						},
						"skip_reason": map[string]any{
							"type":        "string",
							"description": "Reason for skipping visual review (e.g., no render target)",
						},
					},
					"required": []string{"session_id"},
				},
			),
			newTool(
				"summarize",
				"Summarize session intent-plan-execution-verification",
				map[string]any{
					"type": "object",
					"properties": map[string]any{
						"session_id": map[string]any{"type": "string"},
					},
					"required": []string{"session_id"},
				},
			),
			newTool(
				"record_user_feedback",
				"Record user approval/feedback and decide next loop",
				map[string]any{
					"type": "object",
					"properties": map[string]any{
						"session_id": map[string]any{"type": "string"},
						"approved":   map[string]any{"type": "boolean"},
						"feedback":   map[string]any{"type": "string"},
						"required_fixes": map[string]any{
							"type":  "array",
							"items": map[string]any{"type": "string"},
						},
					},
					"required": []string{"session_id", "approved"},
				},
			),
			newTool(
				"continue_persistent_execution",
				"Resume persistent execution loop after failure/feedback",
				map[string]any{
					"type": "object",
					"properties": map[string]any{
						"session_id": map[string]any{"type": "string"},
					},
					"required": []string{"session_id"},
				},
			),
			newTool(
				"get_session_status",
				"Get session status, step history, and next action",
				map[string]any{
					"type": "object",
					"properties": map[string]any{
						"session_id": map[string]any{"type": "string"},
					},
					"required": []string{"session_id"},
				},
			),
			newTool(
				"git_get_state",
				"Get git state (branch, HEAD, change summary)",
				map[string]any{
					"type": "object",
					"properties": map[string]any{
						"path": map[string]any{"type": "string", "default": "."},
					},
				},
			),
			newTool(
				"git_diff_symbols",
				"Map changed files between base/target to estimated symbols",
				map[string]any{
					"type": "object",
					"properties": map[string]any{
						"base":              map[string]any{"type": "string"},
						"target":            map[string]any{"type": "string", "default": "HEAD"},
						"include_untracked": map[string]any{"type": "boolean", "default": false},
					},
					"required": []string{"base"},
				},
			),
			newTool(
				"git_commit_with_context",
				"Create commit with requirement tags in commit message",
				map[string]any{
					"type": "object",
					"properties": map[string]any{
						"goal_id":          map[string]any{"type": "string"},
						"goal_summary":     map[string]any{"type": "string"},
						"requirement_tags": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
						"agent_id":         map[string]any{"type": "string"},
						"risk_level": map[string]any{
							"type": "string",
							"enum": []string{"low", "medium", "high"},
						},
					},
					"required": []string{"goal_summary"},
				},
			),
			newTool(
				"git_resolve_conflict",
				"Guide conflict-resolution modes",
				map[string]any{
					"type": "object",
					"properties": map[string]any{
						"files": map[string]any{
							"type":  "array",
							"items": map[string]any{"type": "string"},
						},
						"strategy": map[string]any{
							"type": "string",
							"enum": []string{"abort", "manual_review", "ours", "theirs", "skip"},
						},
						"notes": map[string]any{"type": "string"},
					},
					"required": []string{"strategy", "files"},
				},
			),
			newTool(
				"git_bisect_start",
				"Start bisect for regression range",
				map[string]any{
					"type": "object",
					"properties": map[string]any{
						"good_commit":  map[string]any{"type": "string"},
						"bad_commit":   map[string]any{"type": "string"},
						"test_command": map[string]any{"type": "string"},
					},
					"required": []string{"good_commit", "bad_commit"},
				},
			),
			newTool(
				"git_recover_state",
				"Recover local state (worktree/branch based)",
				map[string]any{
					"type": "object",
					"properties": map[string]any{
						"mode": map[string]any{
							"type": "string",
							"enum": []string{"checkout_safe_point", "undo_uncommitted", "restore_branch"},
						},
						"safe_point": map[string]any{"type": "string"},
						"branch":     map[string]any{"type": "string"},
					},
					"required": []string{"mode"},
				},
			),
		},
	}
}

func newTool(name, description string, schema map[string]any) toolSchema {
	return toolSchema{
		Name:        name,
		Description: description,
		InputSchema: schema,
	}
}

func defaultInterviewQuestions(rawIntent string, session *SessionState) []string {
	if strings.TrimSpace(rawIntent) == "" {
		ensureUserProfileDefaults(session)
		if session.UserProfile.Overall == "advanced" || session.UserProfile.TechnicalDepth == "technical" {
			return []string{
				consultantText(session,
					"Describe the problem you want to solve in one sentence. If possible, include the affected module/service.",
					"지금 해결하려는 문제를 한 문장으로 말해줘. 가능하면 영향 범위(모듈/서비스)도 같이 알려줘.",
				),
			}
		}
		return []string{
			consultantText(session,
				"Describe the problem you want to solve in one sentence. It is okay if you do not know implementation details yet.",
				"지금 해결하고 싶은 문제를 한 문장으로 말해줘. 아직 구현 방식이나 기술 선택은 몰라도 괜찮아.",
			),
		}
	}

	intent := parseIntent(rawIntent)
	intent.Raw = rawIntent
	tempSession := &SessionState{
		Intent:         intent,
		TopicDecisions: map[string]string{},
		UserProfile:    session.UserProfile,
	}
	decision := buildClarifyDecision(tempSession)
	questions := []string{}
	if decision.Question != "" {
		questions = append(questions, decision.Question)
	}
	if len(questions) == 0 {
		return []string{
			consultantText(session,
				"Can I generate the plan now without more questions? If not, share one must-keep constraint.",
				"추가 질문 없이 바로 계획을 생성해도 될까? 안 된다면 꼭 지켜야 할 제약 1가지만 알려줘.",
			),
		}
	}
	return questions
}

func detectIntentDomain(raw string) string {
	lower := strings.ToLower(raw)
	switch {
	case containsAny(lower, "game", "unity", "unreal", "플레이", "레벨", "캐릭터"):
		return "game"
	case containsAny(lower, "web", "website", "page", "frontend", "ui", "ux", "html", "css", "react", "next"):
		return "frontend"
	case containsAny(lower, "api", "backend", "server", "db", "database", "microservice", "백엔드", "서버"):
		return "backend"
	case containsAny(lower, "maint", "maintenance", "bug", "fix", "refactor", "legacy", "유지보수", "리팩토링", "버그", "장애"):
		return "maintenance"
	case containsAny(lower, "cli", "automation", "script", "infra", "devops", "ci", "cd", "도구"):
		return "automation"
	default:
		return "general"
	}
}

func containsAny(text string, keywords ...string) bool {
	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			return true
		}
	}
	return false
}

func containsHangul(text string) bool {
	for _, r := range text {
		if r >= 0xAC00 && r <= 0xD7A3 {
			return true
		}
	}
	return false
}

func detectConsultantLanguage(text string) string {
	if containsHangul(text) {
		return "ko"
	}
	return "en"
}

func ensureConsultantLanguageDefaults(session *SessionState) {
	if strings.TrimSpace(session.ConsultantLang) == "" {
		session.ConsultantLang = "en"
	}
}

func updateConsultantLanguage(session *SessionState, inputs ...string) {
	ensureConsultantLanguageDefaults(session)
	for _, input := range inputs {
		v := strings.TrimSpace(input)
		if v == "" {
			continue
		}
		detected := detectConsultantLanguage(v)
		if session.ConsultantLang == "ko" && detected == "en" {
			return
		}
		session.ConsultantLang = detected
		return
	}
}

func consultantText(session *SessionState, en, ko string) string {
	ensureConsultantLanguageDefaults(session)
	if strings.ToLower(strings.TrimSpace(session.ConsultantLang)) == "ko" {
		return ko
	}
	return en
}

func answersToText(answers map[string]any) string {
	if len(answers) == 0 {
		return ""
	}
	lines := make([]string, 0, len(answers))
	for k, v := range answers {
		lines = append(lines, fmt.Sprintf("%s: %v", k, v))
	}
	return strings.Join(lines, "\n")
}

func ensureVisualReviewDefaults(session *SessionState) {
	if session.AvailableMCPs == nil {
		session.AvailableMCPs = []string{}
	}
	if session.AvailableMCPTools == nil {
		session.AvailableMCPTools = []string{}
	}
	if session.VisualReview.RendererMatches == nil {
		session.VisualReview.RendererMatches = []string{}
	}
	if session.VisualReview.Artifacts == nil {
		session.VisualReview.Artifacts = []string{}
	}
	if session.VisualReview.Findings == nil {
		session.VisualReview.Findings = []string{}
	}
	if strings.TrimSpace(session.VisualReview.Status) == "" {
		session.VisualReview.Status = "not_required"
	}
}

func mergeMCPInventory(session *SessionState, mcps []string, tools []string) {
	ensureVisualReviewDefaults(session)
	session.AvailableMCPs = mergeUniqueStrings(session.AvailableMCPs, normalizeStringList(mcps)...)
	session.AvailableMCPTools = mergeUniqueStrings(session.AvailableMCPTools, normalizeStringList(tools)...)
}

func normalizeStringList(values []string) []string {
	out := make([]string, 0, len(values))
	for _, item := range values {
		v := strings.TrimSpace(item)
		if v == "" {
			continue
		}
		out = append(out, v)
	}
	return out
}

func detectRenderingProvider(session *SessionState) (bool, string, []string) {
	ensureVisualReviewDefaults(session)
	renderKeywords := []string{
		"playwright", "browser", "screenshot", "snapshot", "viewport", "render", "renderer", "headless",
		"canvas", "webview", "puppeteer", "blender",
	}
	matches := []string{}
	source := ""
	candidates := append([]string{}, session.AvailableMCPs...)
	candidates = append(candidates, session.AvailableMCPTools...)
	for _, entry := range candidates {
		lower := strings.ToLower(strings.TrimSpace(entry))
		for _, keyword := range renderKeywords {
			if strings.Contains(lower, keyword) {
				matches = append(matches, entry)
				if source == "" {
					source = entry
				}
				break
			}
		}
	}
	matches = mergeUniqueStrings([]string{}, matches...)
	if len(matches) == 0 {
		return false, "", []string{}
	}
	return true, source, matches
}

func intentNeedsVisualReview(session *SessionState) bool {
	raw := strings.ToLower(strings.TrimSpace(session.Intent.Raw + "\n" + session.Intent.Goal + "\n" + strings.Join(session.Intent.Scope, " ")))
	domain := detectIntentDomain(raw)
	if domain == "frontend" || domain == "game" {
		return true
	}
	return containsAny(raw,
		"ui", "ux", "screen", "view", "page", "layout", "interaction", "animation", "responsive",
		"화면", "페이지", "랜딩", "반응형", "인터랙션", "시각", "렌더링", "웹", "모바일",
	)
}

func evaluateVisualReviewState(session *SessionState) {
	ensureVisualReviewDefaults(session)
	rendererAvailable, rendererSource, rendererMatches := detectRenderingProvider(session)
	session.VisualReview.RendererAvailable = rendererAvailable
	session.VisualReview.RendererSource = rendererSource
	session.VisualReview.RendererMatches = rendererMatches

	needed := intentNeedsVisualReview(session)
	session.VisualReview.Required = needed && rendererAvailable

	switch {
	case session.VisualReview.Required:
		if session.VisualReview.Status != "completed" && session.VisualReview.Status != "skipped" {
			session.VisualReview.Status = "pending"
		}
	case needed && !rendererAvailable:
		session.VisualReview.Status = "skipped"
		if strings.TrimSpace(session.VisualReview.ReviewerNotes) == "" {
			session.VisualReview.ReviewerNotes = "No rendering-capable MCP detected; Visual Reviewer step skipped"
		}
	default:
		session.VisualReview.Status = "not_required"
	}
}

func visualReviewPending(session *SessionState) bool {
	ensureVisualReviewDefaults(session)
	evaluateVisualReviewState(session)
	return session.VisualReview.Required && session.VisualReview.Status != "completed" && session.VisualReview.Status != "skipped"
}

type userProfileInput struct {
	Overall         string            `json:"overall"`
	DomainKnowledge map[string]string `json:"domain_knowledge"`
	ResponseNeed    string            `json:"response_need"`
	TechnicalDepth  string            `json:"technical_depth"`
}

func normalizeKnowledgeLevel(raw string) string {
	v := strings.ToLower(strings.TrimSpace(raw))
	switch v {
	case "beginner", "novice", "non-technical", "초보", "입문", "비전문가":
		return "beginner"
	case "advanced", "expert", "senior", "전문가", "고급":
		return "advanced"
	case "intermediate", "mid", "중급":
		return "intermediate"
	case "unknown", "":
		return "unknown"
	default:
		return "unknown"
	}
}

func normalizeResponseNeed(raw string) string {
	v := strings.ToLower(strings.TrimSpace(raw))
	switch v {
	case "low", "minimal", "자율", "적게":
		return "low"
	case "high", "detailed", "많이":
		return "high"
	case "balanced", "medium", "보통", "":
		return "balanced"
	default:
		return "balanced"
	}
}

func normalizeTechnicalDepth(raw string) string {
	v := strings.ToLower(strings.TrimSpace(raw))
	switch v {
	case "abstract", "high-level", "추상", "비기술":
		return "abstract"
	case "technical", "deep", "구체", "기술":
		return "technical"
	case "balanced", "normal", "중간", "":
		return "balanced"
	default:
		return "balanced"
	}
}

func normalizeKnowledgeDomain(raw string) string {
	v := strings.ToLower(strings.TrimSpace(raw))
	switch v {
	case "front", "frontend", "ui", "ux":
		return "frontend"
	case "back", "backend", "api", "server":
		return "backend"
	case "db", "database", "data":
		return "db"
	case "security", "auth", "permission":
		return "security"
	case "asset", "assets", "content":
		return "asset"
	case "infra", "devops", "ops", "ci", "cd":
		return "infra"
	case "gameplay", "game":
		return "game"
	default:
		return v
	}
}

func ensureUserProfileDefaults(session *SessionState) {
	if session.UserProfile.Overall == "" {
		session.UserProfile.Overall = "unknown"
	}
	if session.UserProfile.ResponseNeed == "" {
		session.UserProfile.ResponseNeed = "balanced"
	}
	if session.UserProfile.TechnicalDepth == "" {
		session.UserProfile.TechnicalDepth = "balanced"
	}
	if session.UserProfile.DomainKnowledge == nil {
		session.UserProfile.DomainKnowledge = map[string]string{}
	}
	if session.UserProfile.Evidence == nil {
		session.UserProfile.Evidence = []string{}
	}
	if session.UserProfile.Confidence < 0 {
		session.UserProfile.Confidence = 0
	}
	if session.UserProfile.Confidence > 1 {
		session.UserProfile.Confidence = 1
	}
	if session.UserProfile.Confidence == 0 {
		session.UserProfile.Confidence = 0.2
	}
}

func appendProfileEvidence(session *SessionState, evidence string) {
	ensureUserProfileDefaults(session)
	e := strings.TrimSpace(evidence)
	if e == "" {
		return
	}
	if len(session.UserProfile.Evidence) == 0 || session.UserProfile.Evidence[len(session.UserProfile.Evidence)-1] != e {
		session.UserProfile.Evidence = append(session.UserProfile.Evidence, e)
	}
	const maxEvidence = 20
	if len(session.UserProfile.Evidence) > maxEvidence {
		session.UserProfile.Evidence = session.UserProfile.Evidence[len(session.UserProfile.Evidence)-maxEvidence:]
	}
}

func raiseProfileConfidence(session *SessionState, delta float64) {
	ensureUserProfileDefaults(session)
	session.UserProfile.Confidence = math.Max(0, math.Min(1, session.UserProfile.Confidence+delta))
}

func isLowConfidenceProfile(session *SessionState) bool {
	ensureUserProfileDefaults(session)
	return session.UserProfile.Confidence < 0.55
}

func inferKnowledgeLevelFromText(raw string) string {
	v := strings.ToLower(strings.TrimSpace(raw))
	if v == "" {
		return "unknown"
	}
	if containsAny(v, "개발 잘 모르", "비개발", "초보", "입문", "non-technical", "not technical") {
		return "beginner"
	}
	if containsAny(v, "아키텍처", "tradeoff", "latency", "schema", "테스트 전략", "refactor", "성능 목표", "slo", "design doc") {
		return "advanced"
	}
	return "intermediate"
}

func mergeUserProfile(session *SessionState, incoming userProfileInput, source string) {
	ensureUserProfileDefaults(session)
	if lvl := normalizeKnowledgeLevel(incoming.Overall); lvl != "unknown" {
		session.UserProfile.Overall = lvl
		raiseProfileConfidence(session, 0.35)
		appendProfileEvidence(session, fmt.Sprintf("%s: overall=%s", source, lvl))
	}
	if rn := normalizeResponseNeed(incoming.ResponseNeed); rn != "balanced" || strings.TrimSpace(incoming.ResponseNeed) != "" {
		session.UserProfile.ResponseNeed = rn
		if strings.TrimSpace(incoming.ResponseNeed) != "" {
			raiseProfileConfidence(session, 0.15)
			appendProfileEvidence(session, fmt.Sprintf("%s: response_need=%s", source, rn))
		}
	}
	if td := normalizeTechnicalDepth(incoming.TechnicalDepth); td != "balanced" || strings.TrimSpace(incoming.TechnicalDepth) != "" {
		session.UserProfile.TechnicalDepth = td
		if strings.TrimSpace(incoming.TechnicalDepth) != "" {
			raiseProfileConfidence(session, 0.15)
			appendProfileEvidence(session, fmt.Sprintf("%s: technical_depth=%s", source, td))
		}
	}
	domainCount := 0
	for k, v := range incoming.DomainKnowledge {
		domain := normalizeKnowledgeDomain(k)
		if domain == "" {
			continue
		}
		level := normalizeKnowledgeLevel(v)
		if level == "unknown" {
			continue
		}
		session.UserProfile.DomainKnowledge[domain] = level
		domainCount++
		appendProfileEvidence(session, fmt.Sprintf("%s: domain_knowledge[%s]=%s", source, domain, level))
	}
	if domainCount > 0 {
		raiseProfileConfidence(session, math.Min(0.35, 0.15*float64(domainCount)))
	}
}

func parseUserProfileAny(v any) userProfileInput {
	out := userProfileInput{}
	switch t := v.(type) {
	case map[string]any:
		if raw, ok := t["overall"]; ok {
			out.Overall = strings.TrimSpace(fmt.Sprint(raw))
		}
		if raw, ok := t["response_need"]; ok {
			out.ResponseNeed = strings.TrimSpace(fmt.Sprint(raw))
		}
		if raw, ok := t["technical_depth"]; ok {
			out.TechnicalDepth = strings.TrimSpace(fmt.Sprint(raw))
		}
		if raw, ok := t["domain_knowledge"]; ok {
			out.DomainKnowledge = anyToStringMap(raw)
		}
	case map[string]string:
		out.DomainKnowledge = map[string]string{}
		for k, val := range t {
			switch strings.ToLower(strings.TrimSpace(k)) {
			case "overall":
				out.Overall = strings.TrimSpace(val)
			case "response_need":
				out.ResponseNeed = strings.TrimSpace(val)
			case "technical_depth":
				out.TechnicalDepth = strings.TrimSpace(val)
			default:
				out.DomainKnowledge[k] = strings.TrimSpace(val)
			}
		}
	}
	if out.DomainKnowledge == nil {
		out.DomainKnowledge = map[string]string{}
	}
	return out
}

func inferAndSetUserProfile(session *SessionState, rawIntent string) {
	ensureUserProfileDefaults(session)
	if session.UserProfile.Overall == "unknown" {
		inferred := inferKnowledgeLevelFromText(rawIntent)
		session.UserProfile.Overall = inferred
		appendProfileEvidence(session, fmt.Sprintf("inferred_from_intent: overall=%s", inferred))
		raiseProfileConfidence(session, 0.12)
	}
	domain := detectIntentDomain(rawIntent)
	if domain != "" && domain != "general" {
		if _, ok := session.UserProfile.DomainKnowledge[domain]; !ok {
			session.UserProfile.DomainKnowledge[domain] = session.UserProfile.Overall
			appendProfileEvidence(session, fmt.Sprintf("inferred_from_intent: domain_knowledge[%s]=%s", domain, session.UserProfile.Overall))
			raiseProfileConfidence(session, 0.08)
		}
	}
	if session.UserProfile.TechnicalDepth == "balanced" {
		switch session.UserProfile.Overall {
		case "beginner":
			session.UserProfile.TechnicalDepth = "abstract"
		case "advanced":
			session.UserProfile.TechnicalDepth = "technical"
		}
	}
	switch session.UserProfile.Overall {
	case "beginner":
		if session.UserProfile.ResponseNeed == "balanced" {
			session.UserProfile.ResponseNeed = "low"
		}
	case "advanced":
		if session.UserProfile.ResponseNeed == "balanced" {
			session.UserProfile.ResponseNeed = "high"
		}
	}
}

func userKnowledgeForDomain(session *SessionState, domain string) string {
	ensureUserProfileDefaults(session)
	key := normalizeKnowledgeDomain(domain)
	level := session.UserProfile.Overall
	if key != "" {
		if level, ok := session.UserProfile.DomainKnowledge[key]; ok && level != "" {
			if isLowConfidenceProfile(session) && (level == "beginner" || level == "advanced") {
				return "intermediate"
			}
			return level
		}
	}
	if isLowConfidenceProfile(session) && (level == "beginner" || level == "advanced") {
		return "intermediate"
	}
	return level
}

type clarifyDecision struct {
	Status          string
	NextStep        string
	Question        string
	QuestionTopic   string
	QuestionReason  string
	MustConfirm     []string
	AutoDecidable   []string
	AutoAssumptions []string
}

func (s *MCPServer) toolStartInterview(raw json.RawMessage) (any, error) {
	var args struct {
		SessionID         string           `json:"session_id"`
		RawIntent         string           `json:"raw_intent"`
		UserProfile       userProfileInput `json:"user_profile"`
		AvailableMCPs     []string         `json:"available_mcps"`
		AvailableMCPTools []string         `json:"available_mcp_tools"`
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &args); err != nil {
			return nil, err
		}
	}

	session := s.getOrCreateSession(args.SessionID)
	updateConsultantLanguage(session, args.RawIntent)
	mergeMCPInventory(session, args.AvailableMCPs, args.AvailableMCPTools)
	mergeUserProfile(session, args.UserProfile, "start_interview.input")
	hasExistingContext := args.SessionID != "" && (session.Intent.Raw != "" || session.Plan != nil || len(session.StepHistory) > 1 || session.Step != StepReceived)
	if hasExistingContext && strings.TrimSpace(args.RawIntent) == "" {
		current := repoFootprint(s.cfg.WorkDir)
		if session.BaselineFootprint.Head == "" {
			session.BaselineFootprint = current
		}
		session.LastFootprint = current
		driftLevel, driftReason := classifyFootprintDrift(session.BaselineFootprint, current)
		session.ReconcileNeeded = driftLevel == "high"
		if session.ReconcileNeeded {
			session.PendingReview = []string{
				"Code state changed significantly. Choose `keep_context` to continue or `restart_context` to start fresh.",
			}
		}
		session.UpdatedAt = time.Now().UTC()
		return map[string]any{
			"session_id":          session.SessionID,
			"step":                session.Step,
			"entrypoint":          "resume_interview",
			"resume":              true,
			"next_step":           nextAction(session),
			"pending_review":      session.PendingReview,
			"user_profile":        session.UserProfile,
			"available_mcps":      session.AvailableMCPs,
			"available_mcp_tools": session.AvailableMCPTools,
			"visual_review":       session.VisualReview,
			"drift_level":         driftLevel,
			"drift_reason":        driftReason,
			"baseline_footprint":  session.BaselineFootprint,
			"current_footprint":   current,
		}, nil
	}

	resetWorkflowState(session)
	if s.council != nil {
		if err := s.council.resetConsultProposals(session.SessionID); err != nil {
			return nil, err
		}
	}

	questions := []string{}
	nextStep := "ingest_intent"

	if strings.TrimSpace(args.RawIntent) != "" {
		session.Intent = parseIntent(args.RawIntent)
		session.Intent.Raw = args.RawIntent
		inferAndSetUserProfile(session, args.RawIntent)
		session.SetStep(StepIntentCaptured)
		session.PendingReview = nil
		if strings.TrimSpace(session.Intent.Goal) == "" {
			session.PendingReview = []string{
				consultantText(session, "Tell me the goal of this task in one sentence.", "이번 작업의 목적을 한 문장으로 알려줘."),
			}
			questions = append([]string{}, session.PendingReview...)
			nextStep = "clarify_intent"
		} else {
			nextStep = "council_start_briefing"
		}
	} else {
		questions = defaultInterviewQuestions(args.RawIntent, session)
		session.PendingReview = append([]string{}, questions...)
	}

	current := repoFootprint(s.cfg.WorkDir)
	session.BaselineFootprint = current
	session.LastFootprint = current
	session.ReconcileNeeded = false
	session.UpdatedAt = time.Now().UTC()
	questionTopic := firstPendingTopic(session)
	if questionTopic == "" && len(session.PendingReview) > 0 && !session.ProposalAccepted {
		questionTopic = "proposal_alignment"
	}

	return map[string]any{
		"session_id":          session.SessionID,
		"step":                session.Step,
		"interview_questions": questions,
		"pending_review":      session.PendingReview,
		"question_topic":      questionTopic,
		"must_confirm_topics": mustConfirmTopics(session),
		"auto_decidable":      autoDecidableTopics(session),
		"proposal_history":    session.ProposalHistory,
		"proposal_accepted":   session.ProposalAccepted,
		"user_profile":        session.UserProfile,
		"consultant_lang":     session.ConsultantLang,
		"available_mcps":      session.AvailableMCPs,
		"available_mcp_tools": session.AvailableMCPTools,
		"next_step":           nextStep,
		"entrypoint":          "start_interview",
	}, nil
}

func (s *MCPServer) toolIngestIntent(raw json.RawMessage) (any, error) {
	var args struct {
		SessionID         string           `json:"session_id"`
		RawIntent         string           `json:"raw_intent"`
		UserProfile       userProfileInput `json:"user_profile"`
		AvailableMCPs     []string         `json:"available_mcps"`
		AvailableMCPTools []string         `json:"available_mcp_tools"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	session := s.getOrCreateSession(args.SessionID)
	updateConsultantLanguage(session, args.RawIntent)
	mergeMCPInventory(session, args.AvailableMCPs, args.AvailableMCPTools)
	mergeUserProfile(session, args.UserProfile, "ingest_intent.input")
	resetWorkflowState(session)
	if s.council != nil {
		if err := s.council.resetConsultProposals(session.SessionID); err != nil {
			return nil, err
		}
	}
	session.Intent = parseIntent(args.RawIntent)
	session.Intent.Raw = args.RawIntent
	inferAndSetUserProfile(session, args.RawIntent)
	session.SetStep(StepIntentCaptured)
	current := repoFootprint(s.cfg.WorkDir)
	session.BaselineFootprint = current
	session.LastFootprint = current
	session.ReconcileNeeded = false
	session.UpdatedAt = time.Now().UTC()
	decision := buildClarifyDecision(session)
	session.PendingReview = nil
	nextStep := "council_start_briefing"
	questionTopic := ""
	if strings.TrimSpace(session.Intent.Goal) == "" {
		session.PendingReview = []string{
			consultantText(session, "Tell me the goal of this task in one sentence.", "이번 작업의 목적을 한 문장으로 알려줘."),
		}
		nextStep = "clarify_intent"
		questionTopic = "goal"
	}

	return map[string]any{
		"session_id":          session.SessionID,
		"step":                session.Step,
		"intent":              session.Intent,
		"pending_review":      session.PendingReview,
		"next_step":           nextStep,
		"question_topic":      questionTopic,
		"question_reason":     decision.QuestionReason,
		"must_confirm_topics": decision.MustConfirm,
		"auto_decidable":      decision.AutoDecidable,
		"auto_assumptions":    decision.AutoAssumptions,
		"proposal_history":    session.ProposalHistory,
		"proposal_accepted":   session.ProposalAccepted,
		"user_profile":        session.UserProfile,
		"consultant_lang":     session.ConsultantLang,
		"available_mcps":      session.AvailableMCPs,
		"available_mcp_tools": session.AvailableMCPTools,
	}, nil
}

func resetWorkflowState(session *SessionState) {
	session.Step = StepReceived
	session.StepHistory = []WorkStep{StepReceived}
	session.Plan = nil
	session.Mockup = nil
	session.ProposalHistory = nil
	session.ProposalAccepted = false
	session.RequirementTags = nil
	session.ApprovedCriteria = nil
	session.CouncilConsensus = false
	session.CouncilPhase = ""
	session.PlanApproved = false
	session.UserApproved = false
	session.UserFeedback = nil
	session.FixLoopCount = 0
	if session.MaxFixLoops <= 0 {
		session.MaxFixLoops = 5
	}
	session.ActionResults = nil
	session.VerifyResults = nil
	session.ClarifyNotes = nil
	session.PendingReview = nil
	session.TopicDecisions = map[string]string{}
	session.VisualReview = VisualReviewState{
		Status: "not_required",
	}
	session.LastError = ""
}

func parseIntent(raw string) Intent {
	lines := strings.Split(raw, "\n")
	intent := Intent{Goal: strings.TrimSpace(raw), SuccessCriteria: []string{"Manual verification passes", "Tests pass"}}
	explicitCriteria := []string{}

	for _, line := range lines {
		rawLine := strings.TrimSpace(line)
		if rawLine == "" {
			continue
		}
		lower := strings.ToLower(rawLine)

		if strings.HasPrefix(rawLine, "목표:") {
			intent.Goal = strings.TrimSpace(strings.TrimPrefix(rawLine, "목표:"))
		} else if strings.HasPrefix(lower, "goal:") {
			intent.Goal = strings.TrimSpace(rawLine[len("goal:"):])
		}

		if strings.HasPrefix(rawLine, "범위:") {
			scope := strings.TrimSpace(strings.TrimPrefix(rawLine, "범위:"))
			if scope != "" {
				intent.Scope = append(intent.Scope, scope)
			}
		} else if strings.HasPrefix(lower, "scope:") {
			scope := strings.TrimSpace(rawLine[len("scope:"):])
			if scope != "" {
				intent.Scope = append(intent.Scope, scope)
			}
		}

		if strings.HasPrefix(rawLine, "제약:") {
			c := strings.TrimSpace(strings.TrimPrefix(rawLine, "제약:"))
			if c != "" {
				intent.Constraints = append(intent.Constraints, c)
			}
		} else if strings.HasPrefix(lower, "constraints:") {
			c := strings.TrimSpace(rawLine[len("constraints:"):])
			if c != "" {
				intent.Constraints = append(intent.Constraints, c)
			}
		}

		if strings.HasPrefix(rawLine, "성공기준:") {
			criterion := strings.TrimSpace(strings.TrimPrefix(rawLine, "성공기준:"))
			if criterion != "" {
				explicitCriteria = append(explicitCriteria, criterion)
			}
		} else if strings.HasPrefix(lower, "success_criteria:") {
			criterion := strings.TrimSpace(rawLine[len("success_criteria:"):])
			if criterion != "" {
				explicitCriteria = append(explicitCriteria, criterion)
			}
		}
	}
	if len(explicitCriteria) > 0 {
		intent.SuccessCriteria = explicitCriteria
		intent.ExplicitCriteria = true
	}
	if intent.Goal == "" {
		intent.Goal = strings.TrimSpace(raw)
	}
	if strings.TrimSpace(intent.Goal) != "" {
		intent.Assumptions = append(intent.Assumptions, "Request is interpreted as executable within local environment scope")
	}
	return intent
}

func suggestClarifyQuestions(intent Intent) []string {
	session := &SessionState{Intent: intent, TopicDecisions: map[string]string{}}
	decision := buildClarifyDecision(session)
	if decision.Question == "" {
		return nil
	}
	return []string{decision.Question}
}

func (s *MCPServer) toolClarifyIntent(raw json.RawMessage) (any, error) {
	var args struct {
		SessionID string         `json:"session_id"`
		Answers   map[string]any `json:"answers"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	session := s.getOrCreateSession(args.SessionID)
	if session.Step != StepIntentCaptured {
		return nil, fmt.Errorf("clarify intent requires intent_captured state")
	}
	updateConsultantLanguage(session, answersToText(args.Answers))

	for k, v := range args.Answers {
		session.ClarifyNotes = append(session.ClarifyNotes, fmt.Sprintf("%s: %v", k, v))
	}
	proposalDecision, proposalFeedback := extractProposalSignals(args.Answers)
	if proposalDecision == "accept" {
		session.ProposalAccepted = true
	} else if proposalDecision == "refine" || proposalDecision == "alternative" {
		session.ProposalAccepted = false
	}
	if len(session.ProposalHistory) > 0 && (proposalDecision != "" || strings.TrimSpace(proposalFeedback) != "") {
		last := session.ProposalHistory[len(session.ProposalHistory)-1]
		if proposalDecision != "" {
			last.UserDecision = proposalDecision
		}
		if strings.TrimSpace(proposalFeedback) != "" {
			last.UserFeedback = strings.TrimSpace(proposalFeedback)
		}
		session.ProposalHistory[len(session.ProposalHistory)-1] = last
		if s.council != nil {
			if err := s.council.upsertConsultProposal(session.SessionID, last); err != nil {
				return nil, err
			}
		}
	}

	applyClarifyAnswers(session, args.Answers)
	rebriefNeeded := session.CouncilConsensus && shouldRebriefCouncil(proposalDecision, proposalFeedback, args.Answers)
	if rebriefNeeded {
		session.CouncilConsensus = false
		session.CouncilPhase = "needs_rebrief"
	}

	decision := buildClarifyDecision(session)
	status := decision.Status
	nextStep := decision.NextStep
	questionTopic := decision.QuestionTopic
	questionReason := decision.QuestionReason
	session.PendingReview = nil
	if decision.Question != "" {
		session.PendingReview = []string{decision.Question}
		nextStep = "clarify_intent"
	} else if !session.CouncilConsensus {
		nextStep = "council_start_briefing"
		if rebriefNeeded {
			status = "needs_more_info"
			questionTopic = "council_rebrief"
			questionReason = "Possible conflicts among gathered requirements; manager council re-briefing is required"
			session.PendingReview = []string{
				consultantText(
					session,
					"I collected more requirements. I will run a manager council re-brief to resolve conflicts before continuing.",
					"요구를 더 모아두었고, 이제 팀장 council에서 충돌 해소안/우회안을 먼저 정리할게.",
				),
			}
		}
	} else if !session.ProposalAccepted {
		shouldCreateProposal := len(session.ProposalHistory) == 0 ||
			proposalDecision == "refine" ||
			proposalDecision == "alternative" ||
			strings.TrimSpace(proposalFeedback) != ""
		if shouldCreateProposal {
			nextProposal := s.createConsultProposal(session, proposalFeedback, proposalDecision)
			session.ProposalHistory = append(session.ProposalHistory, nextProposal)
			if s.council != nil {
				if err := s.council.upsertConsultProposal(session.SessionID, nextProposal); err != nil {
					return nil, err
				}
			}
		}
		status = "needs_more_info"
		nextStep = "clarify_intent"
		questionTopic = "proposal_alignment"
		questionReason = "Do not finalize outline at once; refine with one focused follow-up at a time"
		var activeProposal *ConsultProposal
		if len(session.ProposalHistory) > 0 {
			last := session.ProposalHistory[len(session.ProposalHistory)-1]
			activeProposal = &last
		}
		session.PendingReview = []string{proposalFollowupQuestion(session, activeProposal)}
	}
	session.Intent.Assumptions = mergeUniqueStrings(session.Intent.Assumptions, decision.AutoAssumptions...)
	session.UpdatedAt = time.Now().UTC()

	var currentProposal *ConsultProposal
	if len(session.ProposalHistory) > 0 {
		last := session.ProposalHistory[len(session.ProposalHistory)-1]
		currentProposal = &last
	}

	return map[string]any{
		"session_id":          session.SessionID,
		"step":                session.Step,
		"status":              status,
		"next_step":           nextStep,
		"notes_count":         len(session.ClarifyNotes),
		"pending_review":      session.PendingReview,
		"follow_up_questions": session.PendingReview,
		"question_topic":      questionTopic,
		"question_reason":     questionReason,
		"must_confirm_topics": decision.MustConfirm,
		"auto_decidable":      decision.AutoDecidable,
		"auto_assumptions":    decision.AutoAssumptions,
		"proposal_decision":   proposalDecision,
		"proposal_accepted":   session.ProposalAccepted,
		"current_proposal":    currentProposal,
		"proposal_history":    session.ProposalHistory,
		"consultant_message":  consultProposalMessage(currentProposal),
		"user_profile":        session.UserProfile,
		"consultant_lang":     session.ConsultantLang,
		"intent":              session.Intent,
	}, nil
}

func (s *MCPServer) toolGeneratePlan(raw json.RawMessage) (any, error) {
	var args struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	session := s.getOrCreateSession(args.SessionID)
	if session.Step != StepIntentCaptured {
		return nil, fmt.Errorf("generate plan requires intent_captured state")
	}
	if !session.CouncilConsensus {
		return nil, fmt.Errorf("generate plan requires council consensus; call council_start_briefing and council_finalize_consensus first")
	}
	if decision := buildClarifyDecision(session); decision.Question != "" {
		return nil, fmt.Errorf("generate plan requires clarified intent; continue clarify_intent first")
	}
	if !session.ProposalAccepted {
		return nil, fmt.Errorf("generate plan requires proposal alignment; continue clarify_intent first")
	}
	plan := &Plan{
		Title:       fmt.Sprintf("Plan for %s", session.Intent.Goal),
		Steps:       []string{"Reconfirm requirement consistency", "Break down into executable units", "Prioritize work items", "Define verification checkpoints", "Finalize summary and approval checkpoints"},
		Assumptions: append(session.Intent.Assumptions, session.ClarifyNotes...),
		Risks:       []string{"Plan drift when requirements are not explicit", "Side effects from dependency changes"},
	}
	session.Plan = plan
	session.SetStep(StepPlanGenerated)
	session.UpdatedAt = time.Now().UTC()
	return map[string]any{
		"session_id": session.SessionID,
		"step":       session.Step,
		"plan":       plan,
		"next_step":  "generate_mockup",
	}, nil
}

func (s *MCPServer) toolGenerateMockup(raw json.RawMessage) (any, error) {
	var args struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	session := s.getOrCreateSession(args.SessionID)
	if session.Step != StepPlanGenerated {
		return nil, fmt.Errorf("generate_mockup requires plan_generated state")
	}

	version := 1
	if session.Mockup != nil {
		version = session.Mockup.Version + 1
	}
	mockup := &MockupArtifact{
		Version: version,
		Summary: fmt.Sprintf("First mockup for %s", strings.TrimSpace(session.Intent.Goal)),
		KeyFlows: []string{
			"Intent input -> interview refinement -> planning",
			"Mockup review -> approve/revise loop",
			"Post-approval execution -> verification -> final user approval",
		},
		OpenQuestions: append([]string{}, session.PendingReview...),
		Assumptions:   append([]string{}, session.Intent.Assumptions...),
		CreatedAt:     time.Now().UTC(),
	}

	session.Mockup = mockup
	session.SetStep(StepMockupReady)
	session.UpdatedAt = time.Now().UTC()
	return map[string]any{
		"session_id": session.SessionID,
		"step":       session.Step,
		"mockup":     mockup,
		"next_step":  "approve_plan",
	}, nil
}

func (s *MCPServer) toolApprovePlan(raw json.RawMessage) (any, error) {
	var args struct {
		SessionID       string   `json:"session_id"`
		Approved        bool     `json:"approved"`
		Notes           string   `json:"notes"`
		RequirementTags []string `json:"requirement_tags"`
		SuccessCriteria []string `json:"success_criteria"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	session := s.getOrCreateSession(args.SessionID)
	if session.Step != StepMockupReady {
		return nil, fmt.Errorf("approve plan requires mockup_ready state")
	}
	if args.Approved {
		session.RequirementTags = mergeUniqueStrings(session.RequirementTags, args.RequirementTags...)
		session.ApprovedCriteria = mergeUniqueStrings(session.ApprovedCriteria, args.SuccessCriteria...)

		missingTags, missingCriteria := validateApproveInputs(session.Intent, session.RequirementTags, session.ApprovedCriteria)
		if len(session.RequirementTags) == 0 {
			missingTags = append(missingTags, "Requirement tags are empty")
		}
		if len(missingCriteria) > 0 {
			missingCriteria = append(missingCriteria, "Provided success criteria do not match intent criteria")
		}
		if len(missingTags) > 0 || len(missingCriteria) > 0 {
			session.LastError = "Approval requirements not satisfied"
			session.SetStep(StepFailed)
			if len(missingTags) > 0 {
				session.LastError += ": " + strings.Join(missingTags, ", ")
			}
			if len(missingCriteria) > 0 {
				if strings.Contains(session.LastError, ": ") {
					session.LastError += ", "
				} else {
					session.LastError += ": "
				}
				session.LastError += strings.Join(missingCriteria, ", ")
			}
			session.UpdatedAt = time.Now().UTC()
			return map[string]any{
				"session_id":       session.SessionID,
				"step":             session.Step,
				"approved":         false,
				"blocking_reasons": append(missingTags, missingCriteria...),
				"required_actions": []string{"Provide at least one requirement tag", "Re-enter requirement success criteria"},
				"session_requirements": map[string]any{
					"intent_success_criteria": session.Intent.SuccessCriteria,
					"approved_criteria":       session.ApprovedCriteria,
					"requirement_tags":        session.RequirementTags,
				},
				"notes": args.Notes,
			}, nil
		}

		session.PlanApproved = true
		session.VisualReview = VisualReviewState{
			Status: "not_required",
		}
		session.SetStep(StepPlanApproved)
	} else {
		session.PlanApproved = false
		session.FixLoopCount++
		if args.Notes != "" {
			session.PendingReview = append(session.PendingReview, "mockup feedback: "+args.Notes)
		}
		session.SetStep(StepIntentCaptured)
		session.LastError = "Re-planning required after mockup feedback"
	}
	session.UpdatedAt = time.Now().UTC()
	nextStep := "run_action"
	if !session.PlanApproved {
		nextStep = "generate_plan"
	}
	return map[string]any{
		"session_id": session.SessionID,
		"step":       session.Step,
		"approved":   session.PlanApproved,
		"notes":      args.Notes,
		"next_step":  nextStep,
	}, nil
}

func validateApproveInputs(intent Intent, tags []string, criteria []string) ([]string, []string) {
	if len(intent.SuccessCriteria) == 0 || !intent.ExplicitCriteria {
		// If only defaults are present, strict criteria matching is skipped.
		// Strict checks require explicit user-provided success criteria.
		return []string{}, []string{}
	}

	normalizedIntent := uniqueNormalizedValues(intent.SuccessCriteria)
	normalizedCriteria := uniqueNormalizedValues(criteria)

	missing := []string{}
	for _, c := range normalizedIntent {
		if c == "" {
			continue
		}
		found := false
		for _, provided := range normalizedCriteria {
			if strings.Contains(provided, c) || strings.Contains(c, provided) {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, c)
		}
	}

	var tagIssues []string
	for _, t := range uniqueNormalizedValues(tags) {
		if t == "" {
			tagIssues = append(tagIssues, "Empty requirement tag")
			break
		}
	}

	if len(tags) == 0 {
		tagIssues = append(tagIssues, "No requirement tags provided")
	}

	return tagIssues, missing
}

func mergeUniqueStrings(dst []string, src ...string) []string {
	seen := map[string]struct{}{}
	for _, v := range dst {
		seen[normalizeToken(v)] = struct{}{}
	}
	out := append([]string{}, dst...)
	for _, v := range src {
		nv := normalizeToken(v)
		if nv == "" {
			continue
		}
		if _, ok := seen[nv]; ok {
			continue
		}
		out = append(out, v)
		seen[nv] = struct{}{}
	}
	return out
}

func anyToStrings(v any) []string {
	switch t := v.(type) {
	case []any:
		var out []string
		for _, item := range t {
			out = append(out, fmt.Sprint(item))
		}
		return out
	case string:
		return []string{t}
	case nil:
		return nil
	default:
		return []string{fmt.Sprint(v)}
	}
}

func splitValueList(values []string) []string {
	out := []string{}
	for _, item := range values {
		for _, part := range strings.FieldsFunc(item, func(r rune) bool {
			return r == '\n' || r == ',' || r == ';'
		}) {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				out = append(out, trimmed)
			}
		}
	}
	return out
}

func anyToStringMap(v any) map[string]string {
	out := map[string]string{}
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			out[k] = strings.TrimSpace(fmt.Sprint(val))
		}
	case map[string]string:
		for k, val := range t {
			out[k] = strings.TrimSpace(val)
		}
	case string:
		parts := strings.FieldsFunc(t, func(r rune) bool { return r == ',' || r == ';' || r == '\n' })
		for _, part := range parts {
			chunks := strings.SplitN(strings.TrimSpace(part), ":", 2)
			if len(chunks) != 2 {
				continue
			}
			out[strings.TrimSpace(chunks[0])] = strings.TrimSpace(chunks[1])
		}
	}
	return out
}

func normalizeTopicKey(topic string) string {
	switch strings.ToLower(strings.TrimSpace(topic)) {
	case "goal", "목표", "problem", "problem_statement":
		return "goal"
	case "scope", "범위", "scope_items", "target_area":
		return "scope"
	case "constraints", "constraint", "제약", "constraints_list", "guardrails":
		return "constraints"
	case "success_criteria", "success_criteria_checked", "성공기준", "done_criteria":
		return "success_criteria"
	case "knowledge_level", "overall_knowledge", "user_knowledge":
		return "knowledge_level"
	case "response_need", "response_level", "question_depth":
		return "response_need"
	case "technical_depth", "detail_level", "explanation_depth":
		return "technical_depth"
	case "domain_knowledge", "knowledge_by_domain", "domain_skill":
		return "domain_knowledge"
	case "user_profile", "profile", "user_context":
		return "user_profile"
	default:
		return strings.ToLower(strings.TrimSpace(topic))
	}
}

func isNoPreference(value string) bool {
	v := strings.ToLower(strings.TrimSpace(value))
	if v == "" {
		return true
	}
	exact := map[string]struct{}{
		"none":           {},
		"n/a":            {},
		"na":             {},
		"auto":           {},
		"autodecide":     {},
		"let you decide": {},
		"무관":             {},
		"없음":             {},
		"dont care":      {},
		"doesn't matter": {},
	}
	if _, ok := exact[v]; ok {
		return true
	}
	block := []string{
		"상관없", "알아서",
	}
	for _, token := range block {
		if strings.Contains(v, token) {
			return true
		}
	}
	return false
}

func hasUserAnswer(session *SessionState, topic string) bool {
	normalized := normalizeTopicKey(topic)
	for _, note := range session.ClarifyNotes {
		prefix := strings.ToLower(strings.TrimSpace(strings.SplitN(note, ":", 2)[0]))
		if normalizeTopicKey(prefix) == normalized {
			return true
		}
	}
	return false
}

func isHighRiskIntent(intent Intent) bool {
	text := strings.ToLower(strings.Join(append(append([]string{intent.Raw, intent.Goal}, intent.Scope...), intent.Constraints...), " "))
	riskKeywords := []string{
		"prod", "production", "보안", "security", "auth", "인증", "결제", "payment", "database", "db", "schema",
		"migration", "마이그레이션", "배포", "삭제", "drop", "권한", "permission",
	}
	return containsAny(text, riskKeywords...)
}

func isGoalAmbiguous(goal string) bool {
	g := strings.ToLower(strings.TrimSpace(goal))
	if g == "" {
		return true
	}
	if len([]rune(g)) < 8 {
		return true
	}
	vagueTokens := []string{
		"개선", "정리", "최적화", "좋게", "잘", "알아서", "대충", "업그레이드",
	}
	for _, token := range vagueTokens {
		if strings.Contains(g, token) {
			return true
		}
	}
	return false
}

func domainSpecificScopeQuestion(domain string, session *SessionState) string {
	knowledge := userKnowledgeForDomain(session, domain)
	switch domain {
	case "frontend":
		if knowledge == "advanced" {
			return consultantText(session, "Name one screen path and component to update first (e.g., /login, AuthForm).", "우선 수정할 화면 경로와 컴포넌트 단위 한 곳만 지정해줘. (예: /login, AuthForm)")
		}
		return consultantText(session, "Pick one screen/flow to touch first (e.g., login screen, checkout flow).", "우선 손댈 화면/플로우 한 곳만 지정해줘. (예: 로그인 화면, 결제 플로우)")
	case "backend":
		if knowledge == "advanced" {
			return consultantText(session, "Name one API endpoint/service module to update first (e.g., POST /auth/login, auth service).", "우선 수정할 API 엔드포인트/서비스 모듈 한 곳만 지정해줘. (예: POST /auth/login, auth service)")
		}
		return consultantText(session, "Pick one API/module to update first (e.g., /auth/login, payment service).", "우선 수정할 API/모듈 한 곳만 지정해줘. (예: /auth/login, payment service)")
	case "game":
		if knowledge == "advanced" {
			return consultantText(session, "Name one core loop and related system to complete first (e.g., combat loop + hit detection).", "우선 완성할 핵심 루프와 연관 시스템 한 곳만 지정해줘. (예: 전투 루프 + 히트 판정)")
		}
		return consultantText(session, "Pick one game loop segment to complete first (e.g., movement, combat, inventory).", "우선 완성할 게임 루프 구간 1개만 지정해줘. (예: 이동, 전투, 인벤토리)")
	case "maintenance":
		if knowledge == "advanced" {
			return consultantText(session, "Name one failing entry path and responsible module (e.g., auth/session.go, token refresh path).", "문제가 재현되는 진입 경로와 책임 모듈 한 곳만 지정해줘. (예: auth/session.go, token refresh path)")
		}
		return consultantText(session, "Pick one file/module/function where the issue reproduces (e.g., auth/session.go).", "문제가 재현되는 파일/모듈/기능 한 곳만 지정해줘. (예: auth/session.go)")
	default:
		if knowledge == "advanced" {
			return consultantText(session, "Define one system boundary to work on first (e.g., auth boundary, payment boundary).", "우선 손댈 범위를 시스템 경계 기준으로 한 곳만 지정해줘. (예: 인증 경계, 결제 경계)")
		}
		return consultantText(session, "Pick one high-level area to work on first (e.g., auth, payment, inventory, deploy pipeline).", "우선 손댈 범위를 큰 단위로 한 곳만 지정해줘. (예: 인증, 결제, 인벤토리, 배포 파이프라인)")
	}
}

func domainSpecificCriteriaQuestion(domain string, session *SessionState) string {
	knowledge := userKnowledgeForDomain(session, domain)
	switch domain {
	case "frontend":
		if knowledge == "advanced" {
			return consultantText(session, "Give 1-2 completion criteria using UX metrics/breakpoints (e.g., CLS < 0.1, layout intact at 375px).", "완료 판정 기준을 UX 지표/브레이크포인트 기준으로 1~2개 알려줘. (예: CLS < 0.1, 375px 레이아웃 무결성)")
		}
		return consultantText(session, "Give 1-2 completion criteria in user-behavior terms (e.g., no UI break on 375px mobile).", "완료 판정 기준을 사용자 동작 기준으로 1~2개 알려줘. (예: 모바일 375px에서 UI 깨짐 없음)")
	case "backend":
		if knowledge == "advanced" {
			return consultantText(session, "Give 1-2 completion criteria using measurable metrics (tests/error rate/latency), e.g., integration tests pass, p95 < 300ms.", "완료 판정 기준을 테스트/에러율/지연시간 같은 수치로 1~2개 알려줘. (예: 통합 테스트 통과, p95 300ms 이하)")
		}
		return consultantText(session, "Give 1-2 completion criteria in test/metric terms (e.g., integration tests pass, p95 < 300ms).", "완료 판정 기준을 테스트/수치 기준으로 1~2개 알려줘. (예: 통합 테스트 통과, p95 300ms 이하)")
	case "game":
		if knowledge == "advanced" {
			return consultantText(session, "Give 1-2 completion criteria using gameplay metrics (loop stability/frame/input responsiveness).", "완료 판정 기준을 루프 안정성/프레임/입력 반응 같은 플레이 지표로 1~2개 알려줘.")
		}
		return consultantText(session, "Give 1-2 completion criteria in play-outcome terms (e.g., tutorial completable within 3 minutes).", "완료 판정 기준을 플레이 결과 기준으로 1~2개 알려줘. (예: 튜토리얼 3분 내 완료 가능)")
	default:
		if knowledge == "advanced" {
			return consultantText(session, "Give 1-2 verifiable checks that define done.", "완료됐다고 볼 수 있는 검증 가능한 체크 항목 1~2개를 알려줘.")
		}
		return consultantText(session, "Give 1-2 checks that define done.", "완료됐다고 볼 수 있는 체크 항목 1~2개를 알려줘.")
	}
}

func normalizeProposalDecision(raw string) string {
	v := strings.ToLower(strings.TrimSpace(raw))
	if v == "" {
		return ""
	}
	if containsAny(v, "accept", "approved", "ok", "go as-is", "as-is", "ship it", "좋", "진행", "승인", "채택") {
		return "accept"
	}
	if containsAny(v, "refine", "revise", "수정", "보완", "다듬") {
		return "refine"
	}
	if containsAny(v, "alternative", "another", "other", "대안", "다른") {
		return "alternative"
	}
	return ""
}

func inferProposalDecisionFromFeedback(raw string) string {
	v := strings.ToLower(strings.TrimSpace(raw))
	if v == "" {
		return ""
	}
	if containsAny(v, "alternative", "another", "other", "대안", "다른 방향", "다른 안", "새 안") {
		return "alternative"
	}
	if containsAny(v, "refine", "revise", "수정", "보완", "다듬", "바꿔", "변경", "추가", "제외", "말고", "대신", "조정", "하지만", "근데") {
		return "refine"
	}
	if containsAny(v, "accept", "approved", "ok", "go as-is", "as-is", "ship it", "오케이", "이대로", "그대로", "진행", "승인", "좋", "괜찮", "문제없", "맞아") {
		return "accept"
	}
	return ""
}

func extractProposalSignals(answers map[string]any) (string, string) {
	decision := ""
	feedback := ""
	for key, value := range answers {
		norm := normalizeTopicKey(key)
		switch norm {
		case "proposal_decision", "decision":
			values := anyToStrings(value)
			if len(values) > 0 {
				decision = normalizeProposalDecision(values[0])
			}
		case "proposal_feedback", "feedback", "message", "response", "comment", "opinion":
			values := anyToStrings(value)
			if len(values) > 0 {
				feedback = strings.TrimSpace(values[0])
			}
		}
	}
	if decision == "" && feedback != "" {
		decision = inferProposalDecisionFromFeedback(feedback)
		if decision == "" {
			decision = "refine"
		}
	}
	return decision, feedback
}

func shouldRebriefCouncil(proposalDecision, proposalFeedback string, answers map[string]any) bool {
	if proposalDecision == "accept" {
		return false
	}
	if proposalDecision == "alternative" {
		return true
	}
	feedback := strings.ToLower(strings.TrimSpace(proposalFeedback))
	if containsAny(feedback, "conflict", "contradict", "incompatible", "tradeoff", "trade-off", "both", "at the same time", "priority clash", "모순", "상충", "동시에", "둘 다", "트레이드오프", "우선순위 충돌", "하지만", "근데") {
		return true
	}
	for key, value := range answers {
		normalized := strings.ToLower(strings.TrimSpace(key))
		if normalized != "conflict" && normalized != "conflicts" && normalized != "tradeoff" && normalized != "trade_off" {
			continue
		}
		for _, item := range anyToStrings(value) {
			if strings.TrimSpace(item) != "" && !isNoPreference(item) {
				return true
			}
		}
	}
	return false
}

func proposalStrategyByDomain(session *SessionState, domain string) string {
	switch domain {
	case "frontend":
		return consultantText(session, "We will lock one core screen/flow first, then stabilize responsive behavior and key interactions.", "핵심 화면/흐름 1개를 먼저 고정하고, 반응형과 핵심 상호작용을 우선 잠그는 방식으로 갈게.")
	case "backend":
		return consultantText(session, "We will stabilize one high-impact API/module first and lock regression tests around it.", "장애 영향이 큰 API/모듈 1개를 먼저 안정화하고 회귀 테스트를 잠그는 방식으로 갈게.")
	case "maintenance":
		return consultantText(session, "We will fix one reproducible failure first and add regression checks to prevent recurrence.", "재현 가능한 실패 1개를 먼저 고치고, 같은 유형 재발을 막는 회귀 검증을 같이 넣을게.")
	case "game":
		return consultantText(session, "We will complete one player-facing core loop first and tune input/feedback feel early.", "플레이어가 즉시 체감하는 핵심 루프 1개를 먼저 완성하고 입력/피드백 감각을 먼저 맞출게.")
	default:
		return consultantText(session, "We will minimize first-pass scope, ship a working result quickly, then expand in the next loop.", "1차 범위를 최소화해서 빠르게 동작 결과를 만들고, 다음 루프에서 확장하는 방식으로 갈게.")
	}
}

func proposalNarrativeByKnowledge(session *SessionState, domain string) string {
	knowledge := userKnowledgeForDomain(session, domain)
	switch knowledge {
	case "beginner":
		return consultantText(session, "I will keep explanations outcome-focused, and team leads will decide most implementation details autonomously.", "설명은 기능 변화와 사용자 체감 결과 중심으로 유지하고, 구현 세부는 팀장 에이전트가 자율 결정할게.")
	case "advanced":
		return consultantText(session, "When needed, I will include structural choices and tradeoffs with concrete rationale.", "필요하면 구조적 연결 방식과 예상 트레이드오프를 근거와 함께 바로 이어서 설명할게.")
	default:
		return consultantText(session, "I will confirm core outcomes first and open details only when needed.", "핵심 결과를 먼저 확인하고, 필요한 만큼만 세부를 열어가는 방식으로 진행할게.")
	}
}

func (s *MCPServer) createConsultProposal(session *SessionState, feedback string, decision string) ConsultProposal {
	domain := detectIntentDomain(session.Intent.Raw + "\n" + session.Intent.Goal)
	version := len(session.ProposalHistory) + 1
	goal := strings.TrimSpace(session.Intent.Goal)
	if goal == "" {
		goal = consultantText(session, "current request", "현재 요청")
	}
	scope := consultantText(session, "one core path", "핵심 경로 1개")
	if len(session.Intent.Scope) > 0 {
		scope = strings.TrimSpace(session.Intent.Scope[0])
	}
	criterion := consultantText(session, "default done criteria (build/tests pass)", "기본 완료 기준(빌드/테스트 통과)")
	if len(session.Intent.SuccessCriteria) > 0 {
		criterion = strings.TrimSpace(session.Intent.SuccessCriteria[0])
	}
	strategy := proposalStrategyByDomain(session, domain)
	narrative := proposalNarrativeByKnowledge(session, domain)
	summary := consultantText(
		session,
		fmt.Sprintf("Outline v%d: For goal `%s`, I will keep first-pass scope to `%s` and prioritize `%s`. %s %s", version, goal, scope, criterion, strategy, narrative),
		fmt.Sprintf("윤곽 v%d: `%s` 목표 기준으로 1차 범위를 `%s`에 고정하고 `%s`를 우선 만족시키는 방식으로 진행할게. %s %s", version, goal, scope, criterion, strategy, narrative),
	)
	if strings.TrimSpace(feedback) != "" {
		summary = consultantText(
			session,
			fmt.Sprintf("%s (updated from latest feedback: %s)", summary, strings.TrimSpace(feedback)),
			fmt.Sprintf("%s (직전 피드백 반영: %s)", summary, strings.TrimSpace(feedback)),
		)
	}
	return ConsultProposal{
		Version:      version,
		Domain:       domain,
		Summary:      summary,
		Options:      []string{},
		Recommended:  "draft",
		UserDecision: decision,
		UserFeedback: strings.TrimSpace(feedback),
		CreatedAt:    time.Now().UTC(),
	}
}

func proposalFollowupQuestion(session *SessionState, proposal *ConsultProposal) string {
	domain := "general"
	if proposal != nil && strings.TrimSpace(proposal.Domain) != "" {
		domain = proposal.Domain
	}
	knowledge := userKnowledgeForDomain(session, domain)
	if knowledge == "advanced" {
		if proposal == nil || proposal.Version <= 1 {
			return consultantText(session, "I drafted a first outline from this conversation. If you have one additional requirement, share the most important one from a structure/risk perspective. If nothing else, reply `go as-is` and I will prepare the mockup.", "지금 대화 기반으로 1차 윤곽을 잡았어. 추가로 생각난 요구사항이 있으면 구조/리스크 관점에서 가장 중요한 것 1가지만 말해줘. 더 없으면 `이대로`라고 답해줘. 그러면 이 윤곽 기준으로 목업 준비를 시작할게.")
		}
		return consultantText(session, "I updated the outline with your feedback. If you have one more requirement, share the most important technical point. If not, reply `go as-is` and I will move to mockup preparation.", "피드백 반영해서 윤곽을 업데이트했어. 추가로 생각난 요구사항이 있으면 기술적으로 중요한 것 1가지만 더 말해줘. 더 없으면 `이대로`라고 답해줘. 바로 목업 준비로 넘어갈게.")
	}
	if proposal == nil || proposal.Version <= 1 {
		return consultantText(session, "I drafted a first outline from what we discussed. If you have one additional requirement, share the most important one. If nothing else, reply `go as-is` and I will prepare the mockup.", "지금까지 대화로 1차 윤곽을 잡았어. 추가로 생각난 요구사항이 있으면 가장 중요한 것 1가지만 말해줘. 더 없으면 `이대로`라고 답해줘. 그러면 이 윤곽 기준으로 목업 준비를 시작할게.")
	}
	return consultantText(session, "I updated the outline with your feedback. If you have one more requirement, share it. If not, reply `go as-is` and I will move to mockup preparation.", "피드백 반영해서 윤곽을 업데이트했어. 추가로 생각난 요구사항이 있으면 1가지만 더 말해줘. 더 없으면 `이대로`라고 답해줘. 바로 목업 준비로 넘어갈게.")
}

func consultProposalMessage(proposal *ConsultProposal) string {
	if proposal == nil {
		return ""
	}
	return proposal.Summary
}

func buildClarifyDecision(session *SessionState) clarifyDecision {
	decision := clarifyDecision{
		Status:        "clarified",
		NextStep:      "generate_plan",
		MustConfirm:   []string{},
		AutoDecidable: []string{},
	}

	intent := session.Intent
	domain := detectIntentDomain(intent.Raw + "\n" + intent.Goal)
	knowledge := userKnowledgeForDomain(session, domain)
	ensureUserProfileDefaults(session)
	responseNeed := session.UserProfile.ResponseNeed
	lowConfidence := isLowConfidenceProfile(session)
	highRisk := isHighRiskIntent(intent)

	goalMissing := strings.TrimSpace(intent.Goal) == ""
	goalAmbiguous := !goalMissing && isGoalAmbiguous(intent.Goal)
	scopeMissing := len(intent.Scope) == 0
	constraintsMissing := len(intent.Constraints) == 0
	criteriaMissing := !intent.ExplicitCriteria

	scopeAutoAllowed := session.TopicDecisions["scope"] == "auto"
	constraintsAutoAllowed := session.TopicDecisions["constraints"] == "auto"
	criteriaAutoAllowed := session.TopicDecisions["success_criteria"] == "auto"
	if responseNeed == "low" && !lowConfidence {
		scopeAutoAllowed = scopeAutoAllowed || scopeMissing
		if !highRisk {
			constraintsAutoAllowed = constraintsAutoAllowed || constraintsMissing
			criteriaAutoAllowed = criteriaAutoAllowed || criteriaMissing
		}
	}

	if goalMissing || goalAmbiguous {
		decision.MustConfirm = append(decision.MustConfirm, "goal")
	}

	if scopeMissing && !scopeAutoAllowed && !goalMissing {
		decision.MustConfirm = append(decision.MustConfirm, "scope")
	}

	if constraintsMissing && !constraintsAutoAllowed {
		decision.MustConfirm = append(decision.MustConfirm, "constraints")
	}

	if criteriaMissing && !criteriaAutoAllowed && (highRisk || goalAmbiguous || hasUserAnswer(session, "success_criteria")) {
		decision.MustConfirm = append(decision.MustConfirm, "success_criteria")
	}

	if scopeMissing && scopeAutoAllowed {
		decision.AutoDecidable = append(decision.AutoDecidable, "scope")
		decision.AutoAssumptions = append(decision.AutoAssumptions, "Scope unspecified: first implementation is limited to one core path")
		if responseNeed == "low" {
			decision.AutoAssumptions = append(decision.AutoAssumptions, "Lowering response burden: team leads decompose detailed scope internally and share summary")
		}
	}
	if constraintsMissing && constraintsAutoAllowed {
		decision.AutoDecidable = append(decision.AutoDecidable, "constraints")
		decision.AutoAssumptions = append(decision.AutoAssumptions, "Constraints unspecified: destructive changes/external permission operations are blocked by default")
	}
	if criteriaMissing && criteriaAutoAllowed {
		decision.AutoDecidable = append(decision.AutoDecidable, "success_criteria")
		decision.AutoAssumptions = append(decision.AutoAssumptions, "Done criteria unspecified: use build/tests pass as baseline")
	}
	if lowConfidence {
		decision.AutoAssumptions = append(decision.AutoAssumptions, "Low confidence in expertise inference: keep auto-decision scope conservative")
	}

	if len(decision.MustConfirm) == 0 {
		return decision
	}

	decision.Status = "needs_more_info"
	decision.NextStep = "clarify_intent"
	topic := decision.MustConfirm[0]
	decision.QuestionTopic = topic

	switch topic {
	case "goal":
		if goalMissing {
			decision.Question = consultantText(session, "In one sentence, what should be different when this task is done?", "이번 작업이 끝났을 때 무엇이 달라져야 하는지 한 문장으로 알려줘.")
			decision.QuestionReason = "Goal is missing, so design baseline cannot be fixed"
		} else {
			if knowledge == "advanced" {
				decision.Question = consultantText(
					session,
					fmt.Sprintf("For `%s`, narrow to one risk/failure you want reduced first. If possible, include a cause hypothesis.", strings.TrimSpace(intent.Goal)),
					fmt.Sprintf("`%s`에서 가장 먼저 줄이고 싶은 리스크/실패를 한 가지로 좁혀줘. 가능하면 원인 가설도 같이 알려줘.", strings.TrimSpace(intent.Goal)),
				)
			} else {
				decision.Question = consultantText(
					session,
					fmt.Sprintf("For `%s`, tell me one concrete pain/failure you want solved first.", strings.TrimSpace(intent.Goal)),
					fmt.Sprintf("`%s`에서 가장 먼저 해결하고 싶은 불편/실패 한 가지를 구체적으로 알려줘.", strings.TrimSpace(intent.Goal)),
				)
			}
			decision.QuestionReason = "Goal is still abstract, so prioritization is ambiguous"
		}
	case "scope":
		decision.Question = domainSpecificScopeQuestion(domain, session)
		decision.QuestionReason = "Without scope, agents may over-expand interpretation"
	case "constraints":
		if highRisk {
			if knowledge == "advanced" {
				decision.Question = consultantText(session, "Tell me 1-2 boundaries that must never be changed in this task (data/permissions/deploy).", "이 작업에서 절대 변경하면 안 되는 경계(데이터/권한/배포) 1~2개를 알려줘.")
			} else {
				decision.Question = consultantText(session, "Tell me 1-2 boundaries we must not touch (e.g., prod DB schema, payment/permission logic).", "이 작업에서 절대 건드리면 안 되는 경계 1~2개를 알려줘. (예: 운영 DB 스키마, 결제/권한 로직)")
			}
			decision.QuestionReason = "High-risk areas are included; safety boundaries must be explicit"
		} else {
			if knowledge == "advanced" {
				decision.Question = consultantText(session, "Share 1-2 hard constraints that must not be violated. If none, tell me if auto-decisions are allowed.", "절대 위반하면 안 되는 조건 1~2개를 알려줘. 없다면 자동결정 허용 여부만 알려줘.")
			} else {
				decision.Question = consultantText(session, "Share 1-2 hard constraints that must not be violated. If none, reply `constraints: auto`.", "절대 위반하면 안 되는 조건 1~2개를 알려줘. 상관없으면 `constraints: 알아서`라고 답해줘.")
			}
			decision.QuestionReason = "Auto-decision boundary must be confirmed"
		}
	case "success_criteria":
		decision.Question = domainSpecificCriteriaQuestion(domain, session)
		decision.QuestionReason = "Done criteria are missing, so outcome satisfaction cannot be evaluated"
	}

	return decision
}

func firstPendingTopic(session *SessionState) string {
	decision := buildClarifyDecision(session)
	return decision.QuestionTopic
}

func mustConfirmTopics(session *SessionState) []string {
	decision := buildClarifyDecision(session)
	return decision.MustConfirm
}

func autoDecidableTopics(session *SessionState) []string {
	decision := buildClarifyDecision(session)
	return decision.AutoDecidable
}

func applyClarifyAnswers(session *SessionState, answers map[string]any) {
	ensureUserProfileDefaults(session)
	for key, value := range answers {
		switch normalizeTopicKey(key) {
		case "goal":
			values := anyToStrings(value)
			if len(values) == 0 {
				continue
			}
			goal := strings.TrimSpace(values[0])
			if isNoPreference(goal) {
				session.TopicDecisions["goal"] = "required"
				continue
			}
			if !isNoPreference(goal) {
				session.Intent.Goal = goal
				session.TopicDecisions["goal"] = "confirmed"
			}
		case "scope":
			values := splitValueList(anyToStrings(value))
			filtered := []string{}
			autoOnly := true
			for _, scope := range values {
				if !isNoPreference(scope) {
					filtered = append(filtered, scope)
					autoOnly = false
				}
			}
			if autoOnly && len(values) > 0 {
				session.TopicDecisions["scope"] = "auto"
			}
			session.Intent.Scope = mergeUniqueStrings(session.Intent.Scope, filtered...)
			if len(filtered) > 0 {
				session.TopicDecisions["scope"] = "confirmed"
			}
		case "constraints":
			values := splitValueList(anyToStrings(value))
			filtered := []string{}
			autoOnly := true
			for _, c := range values {
				if !isNoPreference(c) {
					filtered = append(filtered, c)
					autoOnly = false
				}
			}
			if autoOnly && len(values) > 0 {
				session.TopicDecisions["constraints"] = "auto"
			}
			session.Intent.Constraints = mergeUniqueStrings(session.Intent.Constraints, filtered...)
			if len(filtered) > 0 {
				session.TopicDecisions["constraints"] = "confirmed"
			}
		case "success_criteria":
			values := splitValueList(anyToStrings(value))
			filtered := []string{}
			autoOnly := true
			for _, criterion := range values {
				if !isNoPreference(criterion) {
					filtered = append(filtered, criterion)
					autoOnly = false
				}
			}
			if autoOnly && len(values) > 0 {
				session.TopicDecisions["success_criteria"] = "auto"
			}
			if len(filtered) > 0 {
				session.Intent.SuccessCriteria = mergeUniqueStrings(session.Intent.SuccessCriteria, filtered...)
				session.Intent.ExplicitCriteria = true
				session.TopicDecisions["success_criteria"] = "confirmed"
			}
			session.ApprovedCriteria = mergeUniqueStrings(session.ApprovedCriteria, filtered...)
		case "requirement_tags", "tags":
			values := splitValueList(anyToStrings(value))
			session.RequirementTags = mergeUniqueStrings(session.RequirementTags, values...)
		case "auto_decide_topics":
			topics := splitValueList(anyToStrings(value))
			for _, topic := range topics {
				normalized := normalizeTopicKey(topic)
				if normalized == "" {
					continue
				}
				session.TopicDecisions[normalized] = "auto"
			}
		case "must_confirm_topics":
			topics := splitValueList(anyToStrings(value))
			for _, topic := range topics {
				normalized := normalizeTopicKey(topic)
				if normalized == "" {
					continue
				}
				session.TopicDecisions[normalized] = "required"
			}
		case "knowledge_level":
			values := anyToStrings(value)
			if len(values) > 0 {
				level := normalizeKnowledgeLevel(values[0])
				if level != "unknown" {
					session.UserProfile.Overall = level
					raiseProfileConfidence(session, 0.2)
					appendProfileEvidence(session, fmt.Sprintf("clarify_intent.answer: overall=%s", level))
				}
			}
		case "response_need":
			values := anyToStrings(value)
			if len(values) > 0 {
				rn := normalizeResponseNeed(values[0])
				session.UserProfile.ResponseNeed = rn
				raiseProfileConfidence(session, 0.1)
				appendProfileEvidence(session, fmt.Sprintf("clarify_intent.answer: response_need=%s", rn))
			}
		case "technical_depth":
			values := anyToStrings(value)
			if len(values) > 0 {
				td := normalizeTechnicalDepth(values[0])
				session.UserProfile.TechnicalDepth = td
				raiseProfileConfidence(session, 0.1)
				appendProfileEvidence(session, fmt.Sprintf("clarify_intent.answer: technical_depth=%s", td))
			}
		case "domain_knowledge":
			entries := anyToStringMap(value)
			count := 0
			for domain, levelRaw := range entries {
				level := normalizeKnowledgeLevel(levelRaw)
				if level == "unknown" {
					continue
				}
				normDomain := normalizeKnowledgeDomain(domain)
				session.UserProfile.DomainKnowledge[normDomain] = level
				appendProfileEvidence(session, fmt.Sprintf("clarify_intent.answer: domain_knowledge[%s]=%s", normDomain, level))
				count++
			}
			if count > 0 {
				raiseProfileConfidence(session, math.Min(0.3, 0.12*float64(count)))
			}
		case "user_profile":
			mergeUserProfile(session, parseUserProfileAny(value), "clarify_intent.user_profile")
		}
	}
}

func uniqueNormalizedValues(values []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, v := range values {
		nv := normalizeToken(v)
		if nv == "" {
			continue
		}
		if _, ok := seen[nv]; ok {
			continue
		}
		out = append(out, nv)
		seen[nv] = struct{}{}
	}
	return out
}

func normalizeToken(v string) string {
	return strings.ToLower(strings.TrimSpace(removePunctuation(v)))
}

func removePunctuation(input string) string {
	var b strings.Builder
	for _, r := range input {
		if unicode.IsPunct(r) && r != '_' && r != '/' && r != '-' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func (s *MCPServer) toolValidateTransition(raw json.RawMessage) (any, error) {
	var args struct {
		SessionID   string `json:"session_id"`
		CurrentStep string `json:"current_step"`
		NextStep    string `json:"next_step"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	session := s.getOrCreateSession(args.SessionID)
	current := session.Step
	if args.CurrentStep != "" && string(current) != args.CurrentStep {
		return map[string]any{"allowed": false, "blocking_reasons": []string{"session state mismatch"}, "next_step": string(current)}, nil
	}
	next := WorkStep(args.NextStep)
	allowed := IsAllowedTransition(current, next)
	reason := []string{}
	if !allowed {
		reason = append(reason, fmt.Sprintf("%s -> %s is not allowed", current, next))
	}
	return map[string]any{
		"allowed":                allowed,
		"blocking_reasons":       reason,
		"required_checks":        []string{"intent consistency", "approval status", "permission boundaries"},
		"suggested_next_actions": []string{"satisfy stage constraints via clarify or approve"},
		"next_step":              string(next),
		"confidence":             0.86,
	}, nil
}

func (s *MCPServer) toolRunAction(raw json.RawMessage) (any, error) {
	var args struct {
		SessionID string   `json:"session_id"`
		Commands  []string `json:"commands"`
		DryRun    bool     `json:"dry_run"`
		Timeout   int      `json:"timeout_sec"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	session := s.getOrCreateSession(args.SessionID)
	if session.Step != StepPlanApproved {
		return nil, fmt.Errorf("run_action requires plan_approved state")
	}
	if len(args.Commands) == 0 {
		return nil, fmt.Errorf("commands is required")
	}

	timeout := time.Duration(args.Timeout)
	if timeout <= 0 {
		timeout = 30
	}

	for _, cmd := range args.Commands {
		if !isAllowedCommand(cmd, s.cfg.AllowedCommands) {
			return nil, fmt.Errorf("command not allowed: %s", cmd)
		}

		start := time.Now()
		if args.DryRun {
			session.ActionResults = append(session.ActionResults, CommandResult{Command: cmd, ExitCode: 0, Stdout: "DRY RUN", DurationMS: int64(time.Since(start).Milliseconds())})
			continue
		}

		stdout, stderr, code, err := runCommandWithTimeout(context.Background(), cmd, s.cfg.WorkDir, timeout)
		res := CommandResult{Command: cmd, ExitCode: code, Stdout: stdout, Stderr: stderr, DurationMS: int64(time.Since(start).Milliseconds())}
		if err != nil {
			res.Error = err.Error()
		}
		session.ActionResults = append(session.ActionResults, res)
		if code != 0 {
			session.SetStep(StepFailed)
			session.LastError = res.Error
			session.UpdatedAt = time.Now().UTC()
			return map[string]any{"session_id": session.SessionID, "step": session.Step, "results": session.ActionResults, "error": res.Error}, nil
		}
	}
	session.SetStep(StepActionExecuted)
	session.UpdatedAt = time.Now().UTC()
	return map[string]any{"session_id": session.SessionID, "step": session.Step, "results": session.ActionResults}, nil
}

func (s *MCPServer) toolVerifyResult(raw json.RawMessage) (any, error) {
	var args struct {
		SessionID         string   `json:"session_id"`
		Commands          []string `json:"commands"`
		Timeout           int      `json:"timeout_sec"`
		AvailableMCPs     []string `json:"available_mcps"`
		AvailableMCPTools []string `json:"available_mcp_tools"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	session := s.getOrCreateSession(args.SessionID)
	mergeMCPInventory(session, args.AvailableMCPs, args.AvailableMCPTools)
	if session.Step != StepActionExecuted {
		return nil, fmt.Errorf("verify_result requires action_executed state")
	}
	cmds := args.Commands
	if len(cmds) == 0 {
		cmds = []string{"go test ./..."}
	}
	timeout := time.Duration(args.Timeout)
	if timeout <= 0 {
		timeout = 120
	}

	for _, cmd := range cmds {
		if !isAllowedCommand(cmd, s.cfg.AllowedCommands) {
			return nil, fmt.Errorf("command not allowed: %s", cmd)
		}

		start := time.Now()
		stdout, stderr, code, err := runCommandWithTimeout(context.Background(), cmd, s.cfg.WorkDir, timeout)
		res := CommandResult{Command: cmd, ExitCode: code, Stdout: stdout, Stderr: stderr, DurationMS: int64(time.Since(start).Milliseconds())}
		if err != nil {
			res.Error = err.Error()
		}
		session.VerifyResults = append(session.VerifyResults, res)
		if code != 0 {
			session.FixLoopCount++
			session.UserApproved = false
			session.LastError = res.Error
			session.PendingReview = append(session.PendingReview, fmt.Sprintf("Verification failed (%s): %s", cmd, res.Error))
			if session.FixLoopCount >= session.MaxFixLoops {
				session.SetStep(StepFailed)
				session.LastError = fmt.Sprintf("Verification failed %d times; manual intervention required", session.FixLoopCount)
				session.UpdatedAt = time.Now().UTC()
				return map[string]any{
					"session_id":     session.SessionID,
					"step":           session.Step,
					"results":        session.VerifyResults,
					"error":          session.LastError,
					"persistent_max": session.MaxFixLoops,
					"required_next":  []string{"reconfirm requirements", "after manual intervention, run continue_persistent_execution"},
				}, nil
			}
			session.SetStep(StepIntentCaptured)
			session.UpdatedAt = time.Now().UTC()
			return map[string]any{
				"session_id":      session.SessionID,
				"step":            session.Step,
				"results":         session.VerifyResults,
				"error":           res.Error,
				"persistent_mode": "continue",
				"next_step":       "generate_plan",
				"fix_loop_count":  session.FixLoopCount,
			}, nil
		}
	}
	session.SetStep(StepVerifyRun)
	session.UserApproved = false
	evaluateVisualReviewState(session)
	if visualReviewPending(session) {
		session.PendingReview = mergeUniqueStrings(session.PendingReview,
			"Visual Reviewer step required: review implementation quality against rendered artifacts.",
			"Add UX Director meeting summary based on built artifact.",
		)
	}
	session.UpdatedAt = time.Now().UTC()
	return map[string]any{
		"session_id":    session.SessionID,
		"step":          session.Step,
		"results":       session.VerifyResults,
		"visual_review": session.VisualReview,
		"next_step":     nextAction(session),
	}, nil
}

func (s *MCPServer) toolVisualReview(raw json.RawMessage) (any, error) {
	var args struct {
		SessionID         string   `json:"session_id"`
		AvailableMCPs     []string `json:"available_mcps"`
		AvailableMCPTools []string `json:"available_mcp_tools"`
		Artifacts         []string `json:"artifacts"`
		Findings          []string `json:"findings"`
		ReviewerNotes     string   `json:"reviewer_notes"`
		UXDirectorSummary string   `json:"ux_director_summary"`
		UXDecision        string   `json:"ux_decision"`
		SkipReason        string   `json:"skip_reason"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	session := s.getOrCreateSession(args.SessionID)
	if session.Step != StepVerifyRun {
		return nil, fmt.Errorf("visual_review requires verify_run state")
	}

	mergeMCPInventory(session, args.AvailableMCPs, args.AvailableMCPTools)
	evaluateVisualReviewState(session)

	if strings.TrimSpace(args.SkipReason) != "" {
		session.VisualReview.Status = "skipped"
		session.VisualReview.ReviewerNotes = strings.TrimSpace(args.SkipReason)
		session.VisualReview.Required = false
		session.VisualReview.UpdatedAt = time.Now().UTC()
		session.UpdatedAt = time.Now().UTC()
		return map[string]any{
			"session_id":    session.SessionID,
			"step":          session.Step,
			"visual_review": session.VisualReview,
			"next_step":     "record_user_feedback",
		}, nil
	}

	if !session.VisualReview.Required {
		session.VisualReview.UpdatedAt = time.Now().UTC()
		session.UpdatedAt = time.Now().UTC()
		return map[string]any{
			"session_id":    session.SessionID,
			"step":          session.Step,
			"visual_review": session.VisualReview,
			"next_step":     "record_user_feedback",
		}, nil
	}

	artifacts := normalizeStringList(args.Artifacts)
	findings := normalizeStringList(args.Findings)
	reviewerNotes := strings.TrimSpace(args.ReviewerNotes)
	uxSummary := strings.TrimSpace(args.UXDirectorSummary)
	uxDecision := strings.ToLower(strings.TrimSpace(args.UXDecision))
	if uxDecision == "" {
		uxDecision = "pass"
	}
	if uxDecision != "pass" && uxDecision != "raise" {
		return nil, fmt.Errorf("ux_decision must be pass or raise")
	}

	if len(artifacts) == 0 && len(findings) == 0 && reviewerNotes == "" {
		return map[string]any{
			"session_id": session.SessionID,
			"step":       session.Step,
			"status":     "needs_visual_evidence",
			"required": []string{
				"Provide rendered artifacts (screenshot/recording) or Visual Reviewer findings.",
			},
			"visual_review": session.VisualReview,
			"next_step":     "visual_review",
		}, nil
	}
	if uxSummary == "" {
		return map[string]any{
			"session_id": session.SessionID,
			"step":       session.Step,
			"status":     "needs_ux_director_meeting",
			"required": []string{
				"Provide UX Director meeting summary based on the built artifact.",
			},
			"visual_review": session.VisualReview,
			"next_step":     "visual_review",
		}, nil
	}

	session.VisualReview.Status = "completed"
	session.VisualReview.Artifacts = mergeUniqueStrings(session.VisualReview.Artifacts, artifacts...)
	session.VisualReview.Findings = mergeUniqueStrings(session.VisualReview.Findings, findings...)
	if reviewerNotes != "" {
		session.VisualReview.ReviewerNotes = reviewerNotes
	}
	session.VisualReview.UXDirectorSummary = uxSummary
	session.VisualReview.UXDecision = uxDecision
	session.VisualReview.UpdatedAt = time.Now().UTC()
	if uxDecision == "raise" {
		session.PendingReview = mergeUniqueStrings(session.PendingReview, "UX Director raised concerns: UX improvements required on built artifact")
	}
	session.UpdatedAt = time.Now().UTC()
	return map[string]any{
		"session_id":    session.SessionID,
		"step":          session.Step,
		"status":        "completed",
		"visual_review": session.VisualReview,
		"next_step":     "record_user_feedback",
	}, nil
}

func (s *MCPServer) toolSummarize(raw json.RawMessage) (any, error) {
	var args struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	session := s.getOrCreateSession(args.SessionID)
	evaluateVisualReviewState(session)
	if session.Step == StepVerifyRun && session.UserApproved && !visualReviewPending(session) {
		session.SetStep(StepSummarized)
	}
	gate := "awaiting_user_ok"
	if session.UserApproved {
		gate = "approved"
	}
	summary := fmt.Sprintf("session=%s step=%s goal=%s", session.SessionID, session.Step, session.Intent.Goal)
	if session.Plan != nil {
		summary += ", plan=" + session.Plan.Title
	}
	if session.LastError != "" {
		summary += ", error=" + session.LastError
	}
	session.UpdatedAt = time.Now().UTC()
	return map[string]any{
		"session_id":        session.SessionID,
		"step":              session.Step,
		"step_history":      session.StepHistory,
		"summary":           summary,
		"next":              nextAction(session),
		"user_gate":         gate,
		"user_approved":     session.UserApproved,
		"intent":            session.Intent,
		"plan":              session.Plan,
		"mockup":            session.Mockup,
		"proposal_accepted": session.ProposalAccepted,
		"proposal_history":  session.ProposalHistory,
		"routing_policy":    session.RoutingPolicy,
		"council_consensus": session.CouncilConsensus,
		"council_phase":     session.CouncilPhase,
		"action_count":      len(session.ActionResults),
		"verify_count":      len(session.VerifyResults),
		"visual_review":     session.VisualReview,
		"fix_loop_count":    session.FixLoopCount,
		"max_fix_loops":     session.MaxFixLoops,
		"consultant_lang":   session.ConsultantLang,
	}, nil
}

func (s *MCPServer) toolRecordUserFeedback(raw json.RawMessage) (any, error) {
	var args struct {
		SessionID     string   `json:"session_id"`
		Approved      bool     `json:"approved"`
		Feedback      string   `json:"feedback"`
		RequiredFixes []string `json:"required_fixes"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	session := s.getOrCreateSession(args.SessionID)
	if session.Step != StepVerifyRun && session.Step != StepSummarized && session.Step != StepMockupReady {
		return nil, fmt.Errorf("record_user_feedback requires mockup_ready or verify_run or summarized state")
	}
	if session.Step == StepVerifyRun && visualReviewPending(session) {
		return nil, fmt.Errorf("record_user_feedback requires visual_review completion first")
	}
	if args.Feedback != "" {
		session.UserFeedback = append(session.UserFeedback, args.Feedback)
	}
	if session.Step == StepMockupReady {
		if args.Approved {
			session.UpdatedAt = time.Now().UTC()
			return map[string]any{
				"session_id":    session.SessionID,
				"step":          session.Step,
				"user_approved": true,
				"next_step":     "approve_plan",
			}, nil
		}
		session.FixLoopCount++
		if args.Feedback != "" {
			session.PendingReview = append(session.PendingReview, "mockup feedback: "+args.Feedback)
		}
		session.SetStep(StepIntentCaptured)
		session.UpdatedAt = time.Now().UTC()
		return map[string]any{
			"session_id":     session.SessionID,
			"step":           session.Step,
			"user_approved":  false,
			"fix_loop_count": session.FixLoopCount,
			"next_step":      "generate_plan",
			"pending_review": session.PendingReview,
		}, nil
	}

	session.UserApproved = args.Approved

	if args.Approved {
		if session.Step == StepVerifyRun {
			session.SetStep(StepSummarized)
		}
		session.UpdatedAt = time.Now().UTC()
		return map[string]any{
			"session_id":    session.SessionID,
			"step":          session.Step,
			"user_approved": true,
			"next_step":     "done",
		}, nil
	}

	session.FixLoopCount++
	session.PendingReview = mergeUniqueStrings(session.PendingReview, args.RequiredFixes...)
	if args.Feedback != "" {
		session.PendingReview = append(session.PendingReview, "User feedback requires updates: "+args.Feedback)
	}
	if session.FixLoopCount >= session.MaxFixLoops {
		session.SetStep(StepFailed)
		session.LastError = fmt.Sprintf("User-feedback loop reached %d cycles; manual decision required", session.FixLoopCount)
		session.UpdatedAt = time.Now().UTC()
		return map[string]any{
			"session_id":     session.SessionID,
			"step":           session.Step,
			"user_approved":  false,
			"fix_loop_count": session.FixLoopCount,
			"max_fix_loops":  session.MaxFixLoops,
			"next_step":      "manual_review",
			"last_error":     session.LastError,
		}, nil
	}

	session.SetStep(StepIntentCaptured)
	session.UpdatedAt = time.Now().UTC()
	return map[string]any{
		"session_id":     session.SessionID,
		"step":           session.Step,
		"user_approved":  false,
		"fix_loop_count": session.FixLoopCount,
		"next_step":      "generate_plan",
		"pending_review": session.PendingReview,
	}, nil
}

func (s *MCPServer) toolContinuePersistentExecution(raw json.RawMessage) (any, error) {
	var args struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	session := s.getOrCreateSession(args.SessionID)
	if session.UserApproved && session.Step == StepSummarized {
		return map[string]any{
			"session_id":    session.SessionID,
			"step":          session.Step,
			"next_step":     "done",
			"user_approved": session.UserApproved,
		}, nil
	}
	if session.FixLoopCount >= session.MaxFixLoops {
		session.SetStep(StepFailed)
		session.LastError = fmt.Sprintf("Retry limit exceeded (%d)", session.MaxFixLoops)
		session.UpdatedAt = time.Now().UTC()
		return map[string]any{
			"session_id":     session.SessionID,
			"step":           session.Step,
			"next_step":      "manual_review",
			"fix_loop_count": session.FixLoopCount,
			"max_fix_loops":  session.MaxFixLoops,
			"last_error":     session.LastError,
		}, nil
	}
	if session.Step == StepFailed {
		session.SetStep(StepIntentCaptured)
	}
	session.UpdatedAt = time.Now().UTC()
	return map[string]any{
		"session_id":     session.SessionID,
		"step":           session.Step,
		"fix_loop_count": session.FixLoopCount,
		"max_fix_loops":  session.MaxFixLoops,
		"next_step":      nextAction(session),
	}, nil
}

func (s *MCPServer) toolGetSessionStatus(raw json.RawMessage) (any, error) {
	var args struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	session := s.getOrCreateSession(args.SessionID)
	evaluateVisualReviewState(session)
	return map[string]any{
		"session_id":          session.SessionID,
		"step":                session.Step,
		"step_history":        session.StepHistory,
		"next":                nextAction(session),
		"proposal_accepted":   session.ProposalAccepted,
		"proposal_history":    session.ProposalHistory,
		"council_consensus":   session.CouncilConsensus,
		"council_phase":       session.CouncilPhase,
		"plan_approved":       session.PlanApproved,
		"mockup":              session.Mockup,
		"user_approved":       session.UserApproved,
		"requirement_tags":    session.RequirementTags,
		"approved_criteria":   session.ApprovedCriteria,
		"user_feedback":       session.UserFeedback,
		"pending_review":      session.PendingReview,
		"action_count":        len(session.ActionResults),
		"verify_count":        len(session.VerifyResults),
		"fix_loop_count":      session.FixLoopCount,
		"max_fix_loops":       session.MaxFixLoops,
		"reconcile_needed":    session.ReconcileNeeded,
		"baseline_footprint":  session.BaselineFootprint,
		"last_footprint":      session.LastFootprint,
		"routing_policy":      session.RoutingPolicy,
		"user_profile":        session.UserProfile,
		"consultant_lang":     session.ConsultantLang,
		"available_mcps":      session.AvailableMCPs,
		"available_mcp_tools": session.AvailableMCPTools,
		"visual_review":       session.VisualReview,
		"last_error":          session.LastError,
		"updated_at":          session.UpdatedAt,
	}, nil
}

func repoFootprint(workdir string) RepoFootprint {
	out := RepoFootprint{CapturedAt: time.Now().UTC()}
	branch, _, berr := gitCommand(workdir, "rev-parse", "--abbrev-ref", "HEAD")
	head, _, herr := gitCommand(workdir, "rev-parse", "HEAD")
	status, _, _ := gitCommand(workdir, "status", "--porcelain")
	if berr == nil {
		out.Branch = strings.TrimSpace(branch)
	}
	if herr == nil {
		out.Head = strings.TrimSpace(head)
	}
	trimmed := strings.TrimSpace(status)
	if trimmed != "" {
		out.Dirty = true
		out.ChangedFiles = len(strings.Split(trimmed, "\n"))
	}
	sum := sha256.Sum256([]byte(trimmed))
	out.StatusDigest = fmt.Sprintf("%x", sum[:8])
	return out
}

func classifyFootprintDrift(base RepoFootprint, current RepoFootprint) (string, string) {
	if base.Head == "" {
		return "unknown", "baseline_missing"
	}
	if base.Head == current.Head && base.StatusDigest == current.StatusDigest {
		return "low", "unchanged"
	}
	if base.Head != current.Head && current.ChangedFiles > 20 {
		return "high", "head_changed_many_files"
	}
	if base.Branch != "" && current.Branch != "" && base.Branch != current.Branch {
		return "high", "branch_changed"
	}
	if base.Head != current.Head || base.StatusDigest != current.StatusDigest {
		return "medium", "code_changed"
	}
	return "low", "unchanged"
}

func (s *MCPServer) toolReconcileSessionState(raw json.RawMessage) (any, error) {
	var args struct {
		SessionID string `json:"session_id"`
		Mode      string `json:"mode"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	session := s.getOrCreateSession(args.SessionID)
	mode := strings.TrimSpace(args.Mode)
	if mode == "" {
		mode = "check"
	}

	current := repoFootprint(s.cfg.WorkDir)
	session.LastFootprint = current
	if session.BaselineFootprint.Head == "" {
		session.BaselineFootprint = current
	}
	driftLevel, reason := classifyFootprintDrift(session.BaselineFootprint, current)

	switch mode {
	case "keep_context":
		session.ReconcileNeeded = false
		session.BaselineFootprint = current
		session.PendingReview = nil
	case "restart_context":
		previousID := session.SessionID
		resetWorkflowState(session)
		session.SessionID = previousID
		session.BaselineFootprint = current
		session.LastFootprint = current
		session.ReconcileNeeded = false
	case "check":
		session.ReconcileNeeded = driftLevel == "high"
		if session.ReconcileNeeded {
			session.PendingReview = []string{
				"Large code-state drift detected. Choose `keep_context` to continue or `restart_context` to start fresh.",
			}
		}
	default:
		return nil, fmt.Errorf("invalid mode: %s", mode)
	}

	session.UpdatedAt = time.Now().UTC()
	return map[string]any{
		"session_id":         session.SessionID,
		"mode":               mode,
		"drift_level":        driftLevel,
		"drift_reason":       reason,
		"baseline_footprint": session.BaselineFootprint,
		"current_footprint":  current,
		"reconcile_needed":   session.ReconcileNeeded,
		"options":            []string{"keep_context", "restart_context"},
		"pending_review":     session.PendingReview,
		"next_step":          nextAction(session),
	}, nil
}

func (s *MCPServer) toolSetAgentRoutingPolicy(raw json.RawMessage) (any, error) {
	var args struct {
		SessionID            string `json:"session_id"`
		ClientInterviewModel string `json:"client_interview_model"`
		OrchestratorModel    string `json:"orchestrator_model"`
		ReviewerModel        string `json:"reviewer_model"`
		WorkerModel          string `json:"worker_model"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	session := s.getOrCreateSession(args.SessionID)
	if strings.TrimSpace(args.ClientInterviewModel) != "" {
		session.RoutingPolicy.ClientInterviewModel = strings.TrimSpace(args.ClientInterviewModel)
	}
	if strings.TrimSpace(args.OrchestratorModel) != "" {
		session.RoutingPolicy.OrchestratorModel = strings.TrimSpace(args.OrchestratorModel)
	}
	if strings.TrimSpace(args.ReviewerModel) != "" {
		session.RoutingPolicy.ReviewerModel = strings.TrimSpace(args.ReviewerModel)
	}
	if strings.TrimSpace(args.WorkerModel) != "" {
		session.RoutingPolicy.WorkerModel = strings.TrimSpace(args.WorkerModel)
	}
	session.UpdatedAt = time.Now().UTC()
	return map[string]any{
		"session_id":     session.SessionID,
		"routing_policy": session.RoutingPolicy,
	}, nil
}

func (s *MCPServer) toolGetAgentRoutingPolicy(raw json.RawMessage) (any, error) {
	var args struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	session := s.getOrCreateSession(args.SessionID)
	return map[string]any{
		"session_id":     session.SessionID,
		"routing_policy": session.RoutingPolicy,
	}, nil
}

func councilRoleDomain(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "ux_director", "frontend_lead":
		return "frontend"
	case "backend_lead":
		return "backend"
	case "db_lead":
		return "db"
	case "security_manager":
		return "security"
	case "asset_manager":
		return "asset"
	default:
		return "general"
	}
}

func councilAutonomyPolicy(session *SessionState, role string) (string, string, string) {
	domain := councilRoleDomain(role)
	if isLowConfidenceProfile(session) {
		return "balanced", "balanced", "Low confidence in expertise inference: avoid aggressive autonomy, expose assumptions/risks, and surface checkpoints first"
	}
	knowledge := userKnowledgeForDomain(session, domain)
	switch knowledge {
	case "beginner":
		return "high", "abstract", "In low-familiarity domains, team decides implementation choices autonomously and reports impact/constraints at abstract level"
	case "advanced":
		return "balanced", "technical", "In high-familiarity domains, include concrete architecture/risk/tradeoff rationale for detailed verification"
	default:
		return "balanced", "balanced", "Prioritize core outcomes first, then expand technical details only when needed"
	}
}

func (s *MCPServer) toolCouncilStartBriefing(raw json.RawMessage) (any, error) {
	if s.council == nil {
		return nil, fmt.Errorf("council store is not available")
	}
	var args struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	session := s.getOrCreateSession(args.SessionID)
	if session.Step != StepIntentCaptured {
		return nil, fmt.Errorf("council_start_briefing requires intent_captured state")
	}
	if strings.TrimSpace(session.Intent.Goal) == "" {
		return nil, fmt.Errorf("council_start_briefing requires goal; continue clarify_intent first")
	}

	roles, topics, err := s.council.startBriefing(session.SessionID, session.RoutingPolicy, session.Intent)
	if err != nil {
		return nil, err
	}
	session.CouncilPhase = "briefing"
	session.CouncilConsensus = false
	session.UpdatedAt = time.Now().UTC()

	briefPrompts := []map[string]any{}
	for _, role := range roles {
		autonomy, consultDepth, policyNote := councilAutonomyPolicy(session, role.Role)
		briefPrompts = append(briefPrompts, map[string]any{
			"role":           role.Role,
			"model":          role.Model,
			"autonomy_level": autonomy,
			"consult_depth":  consultDepth,
			"policy_note":    policyNote,
			"prompt": fmt.Sprintf(
				"Project goal: %s\nRole: %s\nAutonomy level: %s\nConsult depth: %s\nPolicy: %s\nBriefing items: (1) top priority (2) role contribution (3) quick decisions to lock (4) additional agenda proposals",
				strings.TrimSpace(session.Intent.Goal), role.Role, autonomy, consultDepth, policyNote,
			),
		})
	}

	return map[string]any{
		"session_id":    session.SessionID,
		"phase":         session.CouncilPhase,
		"next_step":     "council_submit_brief",
		"roles":         roles,
		"topics":        topics,
		"brief_prompts": briefPrompts,
		"user_profile":  session.UserProfile,
	}, nil
}

func (s *MCPServer) toolCouncilSubmitBrief(raw json.RawMessage) (any, error) {
	if s.council == nil {
		return nil, fmt.Errorf("council store is not available")
	}
	var args struct {
		SessionID      string   `json:"session_id"`
		Role           string   `json:"role"`
		Priority       string   `json:"priority"`
		Contribution   string   `json:"contribution"`
		QuickDecisions string   `json:"quick_decisions"`
		TopicProposals []string `json:"topic_proposals"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	session := s.getOrCreateSession(args.SessionID)
	if strings.TrimSpace(args.Role) == "" {
		return nil, fmt.Errorf("role is required")
	}
	if err := s.council.submitBrief(session.SessionID, strings.TrimSpace(args.Role), args.Priority, args.Contribution, args.QuickDecisions, args.TopicProposals); err != nil {
		return nil, err
	}
	submitted, total, err := s.council.countBriefSubmitted(session.SessionID)
	if err != nil {
		return nil, err
	}
	if submitted == total && total > 0 {
		session.CouncilPhase = "brief_collected"
	}
	session.UpdatedAt = time.Now().UTC()
	return map[string]any{
		"session_id":      session.SessionID,
		"submitted_role":  strings.TrimSpace(args.Role),
		"brief_submitted": submitted,
		"brief_total":     total,
		"phase":           session.CouncilPhase,
		"next_step":       "council_summarize_briefs",
	}, nil
}

func (s *MCPServer) toolCouncilSummarizeBriefs(raw json.RawMessage) (any, error) {
	if s.council == nil {
		return nil, fmt.Errorf("council store is not available")
	}
	var args struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	session := s.getOrCreateSession(args.SessionID)
	summary, topics, err := s.council.summarizeBriefs(session.SessionID)
	if err != nil {
		return nil, err
	}
	session.CouncilPhase = "agenda_ready"
	session.UpdatedAt = time.Now().UTC()
	return map[string]any{
		"session_id": session.SessionID,
		"phase":      session.CouncilPhase,
		"summary":    summary,
		"topics":     topics,
		"next_step":  "council_request_floor",
	}, nil
}

func (s *MCPServer) toolCouncilRequestFloor(raw json.RawMessage) (any, error) {
	if s.council == nil {
		return nil, fmt.Errorf("council store is not available")
	}
	var args struct {
		SessionID string `json:"session_id"`
		TopicID   int64  `json:"topic_id"`
		Role      string `json:"role"`
		Reason    string `json:"reason"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	if args.TopicID <= 0 {
		return nil, fmt.Errorf("topic_id is required")
	}
	requestID, err := s.council.requestFloor(args.SessionID, strings.TrimSpace(args.Role), args.TopicID, args.Reason)
	if err != nil {
		return nil, err
	}
	session := s.getOrCreateSession(args.SessionID)
	session.CouncilPhase = "discussion"
	session.UpdatedAt = time.Now().UTC()
	return map[string]any{
		"session_id": session.SessionID,
		"topic_id":   args.TopicID,
		"request_id": requestID,
		"next_step":  "council_grant_floor",
	}, nil
}

func (s *MCPServer) toolCouncilGrantFloor(raw json.RawMessage) (any, error) {
	if s.council == nil {
		return nil, fmt.Errorf("council store is not available")
	}
	var args struct {
		SessionID string `json:"session_id"`
		RequestID int64  `json:"request_id"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	if args.RequestID <= 0 {
		return nil, fmt.Errorf("request_id is required")
	}
	topicID, role, err := s.council.grantFloor(args.SessionID, args.RequestID)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"session_id": args.SessionID,
		"request_id": args.RequestID,
		"topic_id":   topicID,
		"granted_to": role,
		"next_step":  "council_publish_statement",
	}, nil
}

func (s *MCPServer) toolCouncilPublishStatement(raw json.RawMessage) (any, error) {
	if s.council == nil {
		return nil, fmt.Errorf("council store is not available")
	}
	var args struct {
		SessionID string `json:"session_id"`
		RequestID int64  `json:"request_id"`
		Content   string `json:"content"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	if strings.TrimSpace(args.Content) == "" {
		return nil, fmt.Errorf("content is required")
	}
	topicID, waitingRoles, err := s.council.publishStatement(args.SessionID, args.RequestID, args.Content)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"session_id":    args.SessionID,
		"topic_id":      topicID,
		"waiting_roles": waitingRoles,
		"next_step":     "council_respond_topic",
	}, nil
}

func (s *MCPServer) toolCouncilRespondTopic(raw json.RawMessage) (any, error) {
	if s.council == nil {
		return nil, fmt.Errorf("council store is not available")
	}
	var args struct {
		SessionID string `json:"session_id"`
		TopicID   int64  `json:"topic_id"`
		Role      string `json:"role"`
		Decision  string `json:"decision"`
		Content   string `json:"content"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	closable, pendingRoles, err := s.council.respondTopic(args.SessionID, args.TopicID, strings.TrimSpace(args.Role), args.Decision, args.Content)
	if err != nil {
		return nil, err
	}
	nextStep := "council_request_floor"
	if closable {
		nextStep = "council_close_topic"
	}
	return map[string]any{
		"session_id":    args.SessionID,
		"topic_id":      args.TopicID,
		"closable":      closable,
		"pending_roles": pendingRoles,
		"next_step":     nextStep,
	}, nil
}

func (s *MCPServer) toolCouncilCloseTopic(raw json.RawMessage) (any, error) {
	if s.council == nil {
		return nil, fmt.Errorf("council store is not available")
	}
	var args struct {
		SessionID string `json:"session_id"`
		TopicID   int64  `json:"topic_id"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	openCount, err := s.council.closeTopic(args.SessionID, args.TopicID)
	if err != nil {
		return nil, err
	}
	nextStep := "council_request_floor"
	if openCount == 0 {
		nextStep = "council_finalize_consensus"
	}
	return map[string]any{
		"session_id":  args.SessionID,
		"topic_id":    args.TopicID,
		"open_topics": openCount,
		"next_step":   nextStep,
	}, nil
}

func (s *MCPServer) toolCouncilFinalizeConsensus(raw json.RawMessage) (any, error) {
	if s.council == nil {
		return nil, fmt.Errorf("council store is not available")
	}
	var args struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	session := s.getOrCreateSession(args.SessionID)
	if err := s.council.finalizeConsensus(session.SessionID); err != nil {
		return nil, err
	}
	session.CouncilConsensus = true
	session.CouncilPhase = "finalized"
	session.UpdatedAt = time.Now().UTC()
	nextStep := "clarify_intent"
	if decision := buildClarifyDecision(session); decision.Question == "" && session.ProposalAccepted {
		nextStep = "generate_plan"
	}
	return map[string]any{
		"session_id":        session.SessionID,
		"council_consensus": session.CouncilConsensus,
		"phase":             session.CouncilPhase,
		"next_step":         nextStep,
	}, nil
}

func (s *MCPServer) toolCouncilGetStatus(raw json.RawMessage) (any, error) {
	if s.council == nil {
		return nil, fmt.Errorf("council store is not available")
	}
	var args struct {
		SessionID    string `json:"session_id"`
		MessageLimit int    `json:"message_limit"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	status, phase, summary, err := s.council.getSessionMeta(args.SessionID)
	if err != nil {
		return nil, err
	}
	roles, err := s.council.loadRoles(args.SessionID)
	if err != nil {
		return nil, err
	}
	topics, err := s.council.loadTopics(args.SessionID)
	if err != nil {
		return nil, err
	}
	messages, err := s.council.loadMessages(args.SessionID, args.MessageLimit)
	if err != nil {
		return nil, err
	}
	proposals, err := s.council.loadConsultProposals(args.SessionID)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"session_id": args.SessionID,
		"status":     status,
		"phase":      phase,
		"summary":    summary,
		"roles":      roles,
		"topics":     topics,
		"messages":   messages,
		"proposals":  proposals,
	}, nil
}

func nextAction(session *SessionState) string {
	if session.ReconcileNeeded {
		return "reconcile_session_state"
	}
	switch session.Step {
	case StepReceived:
		return "ingest_intent"
	case StepIntentCaptured:
		if strings.TrimSpace(session.Intent.Goal) == "" {
			return "clarify_intent"
		}
		if !session.CouncilConsensus {
			return "council_start_briefing"
		}
		if decision := buildClarifyDecision(session); decision.Question != "" {
			return "clarify_intent"
		}
		if !session.ProposalAccepted {
			return "clarify_intent"
		}
		return "generate_plan"
	case StepPlanGenerated:
		return "generate_mockup"
	case StepMockupReady:
		return "approve_plan"
	case StepPlanApproved:
		return "run_action"
	case StepActionExecuted:
		return "verify_result"
	case StepVerifyRun:
		if visualReviewPending(session) {
			return "visual_review"
		}
		if session.UserApproved {
			return "summarize"
		}
		return "record_user_feedback"
	case StepSummarized:
		return "done"
	case StepFailed:
		if session.FixLoopCount < session.MaxFixLoops {
			return "continue_persistent_execution"
		}
		return "manual_review"
	default:
		return "re-meeting (requirement check)"
	}
}

func isAllowedCommand(command string, allowList []string) bool {
	raw := strings.TrimSpace(command)
	if raw == "" {
		return false
	}
	if strings.Contains(raw, "&&") || strings.Contains(raw, "||") || strings.Contains(raw, ";") || strings.Contains(raw, "|") {
		return false
	}
	fields := strings.Fields(raw)
	for len(fields) > 0 {
		candidate := fields[0]
		candidate = strings.TrimSpace(candidate)
		// allow env var prefixes like VAR=value command
		if strings.Contains(candidate, "=") && !strings.Contains(candidate, "/") && !strings.HasPrefix(candidate, "./") {
			fields = fields[1:]
			continue
		}
		base := filepath.Base(candidate)
		for _, allow := range allowList {
			if candidate == allow || base == allow {
				return true
			}
		}
		return false
	}
	return false
}

func runCommandWithTimeout(ctx context.Context, command string, dir string, timeoutSeconds time.Duration) (string, string, int, error) {
	ctx2, cancel := context.WithTimeout(ctx, timeoutSeconds*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx2, "sh", "-c", command)
	if dir != "" {
		cmd.Dir = dir
	}
	var outb bytes.Buffer
	var errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	err := cmd.Run()
	if exitErr, ok := err.(*exec.ExitError); ok {
		return outb.String(), errb.String(), exitErr.ExitCode(), err
	}
	if err != nil {
		return outb.String(), errb.String(), -1, err
	}
	return outb.String(), errb.String(), 0, nil
}

func (s *MCPServer) toolGitGetState(raw json.RawMessage) (any, error) {
	var args struct {
		Path string `json:"path"`
	}
	_ = json.Unmarshal(raw, &args)
	path := args.Path
	if path == "" {
		path = "."
	}
	abs, _ := filepath.Abs(path)
	head, _, _ := gitCommand(abs, "rev-parse", "--abbrev-ref", "HEAD")
	commit, _, _ := gitCommand(abs, "rev-parse", "HEAD")
	status, errOut, err := gitCommand(abs, "status", "--short")
	if err != nil {
		return nil, fmt.Errorf("git_get_state failed: %s", strings.TrimSpace(errOut))
	}
	return map[string]any{
		"path":   abs,
		"branch": strings.TrimSpace(head),
		"head":   strings.TrimSpace(commit),
		"status": strings.TrimSpace(status),
	}, nil
}

func (s *MCPServer) toolGitDiffSymbols(raw json.RawMessage) (any, error) {
	var args struct {
		Base             string `json:"base"`
		Target           string `json:"target"`
		IncludeUntracked bool   `json:"include_untracked"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	if args.Base == "" {
		return nil, fmt.Errorf("base is required")
	}
	target := args.Target
	if target == "" {
		target = "HEAD"
	}

	diffFiles, errOut, err := gitCommand(".", "diff", "--name-only", args.Base, target)
	if err != nil {
		return nil, fmt.Errorf("git_diff_symbols failed: %s", strings.TrimSpace(errOut))
	}
	files := []string{}
	scanner := bufio.NewScanner(strings.NewReader(strings.TrimSpace(diffFiles)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			files = append(files, line)
		}
	}

	changedSymbols := []map[string]string{}
	for _, f := range files {
		changedSymbols = append(changedSymbols, map[string]string{"symbol": filepath.Base(f), "file": f})
	}

	return map[string]any{
		"base":              args.Base,
		"target":            target,
		"changed_symbols":   changedSymbols,
		"deleted_symbols":   []string{},
		"renamed_symbols":   []string{},
		"tests_affected":    []string{},
		"confidence":        0.72,
		"include_untracked": args.IncludeUntracked,
	}, nil
}

func (s *MCPServer) toolGitCommitWithContext(raw json.RawMessage) (any, error) {
	var args struct {
		GoalID          string   `json:"goal_id"`
		GoalSummary     string   `json:"goal_summary"`
		RequirementTags []string `json:"requirement_tags"`
		AgentID         string   `json:"agent_id"`
		RiskLevel       string   `json:"risk_level"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	if args.GoalSummary == "" {
		return nil, fmt.Errorf("goal_summary is required")
	}
	msg := fmt.Sprintf("feat(%s): %s", strings.TrimSpace(args.RiskLevel), args.GoalSummary)
	if args.GoalID != "" {
		msg = fmt.Sprintf("%s [goal:%s]", msg, args.GoalID)
	}
	if len(args.RequirementTags) > 0 {
		msg = fmt.Sprintf("%s tags=%v", msg, args.RequirementTags)
	}
	if strings.TrimSpace(args.AgentID) != "" {
		msg = fmt.Sprintf("%s by=%s", msg, args.AgentID)
	}

	if _, _, err := gitCommand(".", "add", "-A"); err != nil {
		return nil, fmt.Errorf("git add failed")
	}
	if out, errOut, err := gitCommand(".", "commit", "-m", msg); err != nil {
		if strings.Contains(errOut, "nothing to commit") {
			return nil, fmt.Errorf("nothing to commit")
		}
		return nil, fmt.Errorf("git commit failed: %s", strings.TrimSpace(errOut))
	} else {
		return map[string]any{"commit_message": msg, "commit_output": out}, nil
	}
}

func (s *MCPServer) toolGitResolveConflict(raw json.RawMessage) (any, error) {
	var args struct {
		Files    []string `json:"files"`
		Strategy string   `json:"strategy"`
		Notes    string   `json:"notes"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	if len(args.Files) == 0 {
		return nil, fmt.Errorf("files is required")
	}
	if args.Strategy == "" {
		return nil, fmt.Errorf("strategy is required")
	}

	switch args.Strategy {
	case "abort":
		if out, errOut, err := gitCommand(".", "merge", "--abort"); err != nil {
			return nil, fmt.Errorf("git merge --abort failed: %s", strings.TrimSpace(errOut))
		} else {
			return map[string]any{"resolved": true, "strategy": args.Strategy, "output": out}, nil
		}
	case "ours", "theirs":
		for _, file := range args.Files {
			if _, errOut, err := gitCommand(".", "checkout", "--"+args.Strategy, "--", file); err != nil {
				return nil, fmt.Errorf("checkout %s failed for %s: %s", args.Strategy, file, strings.TrimSpace(errOut))
			}
		}
		return map[string]any{"resolved": true, "strategy": args.Strategy, "notes": args.Notes}, nil
	case "manual_review", "skip":
		return map[string]any{"resolved": false, "strategy": args.Strategy, "notes": args.Notes}, nil
	default:
		return nil, fmt.Errorf("unknown strategy")
	}
}

func (s *MCPServer) toolGitBisectStart(raw json.RawMessage) (any, error) {
	var args struct {
		Good string `json:"good_commit"`
		Bad  string `json:"bad_commit"`
		Test string `json:"test_command"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	if args.Good == "" || args.Bad == "" {
		return nil, fmt.Errorf("good_commit and bad_commit required")
	}
	if _, errOut, err := gitCommand(".", "bisect", "start"); err != nil {
		return nil, fmt.Errorf("bisect start failed: %s", strings.TrimSpace(errOut))
	}
	if _, errOut, err := gitCommand(".", "bisect", "good", args.Good); err != nil {
		return nil, fmt.Errorf("bisect good failed: %s", strings.TrimSpace(errOut))
	}
	if _, errOut, err := gitCommand(".", "bisect", "bad", args.Bad); err != nil {
		return nil, fmt.Errorf("bisect bad failed: %s", strings.TrimSpace(errOut))
	}
	if args.Test != "" {
		return map[string]any{"status": "started", "note": "test command provided. run bisect manually", "test_command": args.Test}, nil
	}
	return map[string]any{"status": "started"}, nil
}

func (s *MCPServer) toolGitRecoverState(raw json.RawMessage) (any, error) {
	var args struct {
		Mode      string `json:"mode"`
		SafePoint string `json:"safe_point"`
		Branch    string `json:"branch"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}

	switch args.Mode {
	case "checkout_safe_point":
		if args.SafePoint == "" {
			return nil, fmt.Errorf("safe_point required")
		}
		if out, errOut, err := gitCommand(".", "checkout", args.SafePoint); err != nil {
			return nil, fmt.Errorf("checkout safe point failed: %s", strings.TrimSpace(errOut))
		} else {
			return map[string]any{"restored": true, "mode": args.Mode, "output": out}, nil
		}
	case "undo_uncommitted":
		if _, errOut, err := gitCommand(".", "restore", "--staged", "."); err != nil {
			return nil, fmt.Errorf("undo staged failed: %s", strings.TrimSpace(errOut))
		}
		if out, errOut, err := gitCommand(".", "restore", "."); err != nil {
			return nil, fmt.Errorf("undo workingtree failed: %s", strings.TrimSpace(errOut))
		} else {
			return map[string]any{"restored": true, "mode": args.Mode, "output": out}, nil
		}
	case "restore_branch":
		if args.Branch == "" {
			return nil, fmt.Errorf("branch required")
		}
		if out, errOut, err := gitCommand(".", "checkout", args.Branch); err != nil {
			return nil, fmt.Errorf("restore branch failed: %s", strings.TrimSpace(errOut))
		} else {
			return map[string]any{"restored": true, "mode": args.Mode, "branch": args.Branch, "output": out}, nil
		}
	default:
		return nil, fmt.Errorf("unknown mode")
	}
}

func gitCommand(dir string, args ...string) (string, string, error) {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	err := cmd.Run()
	return strings.TrimSpace(outb.String()), strings.TrimSpace(errb.String()), err
}
