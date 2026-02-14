package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var defaultCouncilRoles = []string{
	"ux_director",
	"frontend_lead",
	"backend_lead",
	"db_lead",
	"asset_manager",
	"security_manager",
}

func forceCouncilConsensus(t *testing.T, srv *MCPServer, sessionID string) {
	t.Helper()
	out, err := srv.toolClarifyIntent([]byte(`{"session_id":"` + sessionID + `","answers":{"goal":"로그인 실패율을 낮춰 재발 장애를 막는다","scope":"internal/server","constraints":"로컬 환경만, 파괴적 변경 금지","success_criteria":["테스트 통과","빌드 성공"],"proposal_decision":"accept","proposal_feedback":"A안 진행"}}`))
	if err != nil {
		t.Fatalf("clarify before council failed: %v", err)
	}
	result := out.(map[string]any)
	if result["status"] != "clarified" {
		t.Fatalf("expected clarified before council, got %v", result["status"])
	}
	if result["next_step"] != "council_start_briefing" {
		t.Fatalf("expected council_start_briefing next_step, got %v", result["next_step"])
	}

	if _, err := srv.toolCouncilStartBriefing([]byte(`{"session_id":"` + sessionID + `"}`)); err != nil {
		t.Fatalf("council_start_briefing failed: %v", err)
	}
	for _, role := range defaultCouncilRoles {
		payload := fmt.Sprintf(`{"session_id":"%s","role":"%s","priority":"core","contribution":"%s contribution","quick_decisions":"none"}`, sessionID, role, role)
		if _, err := srv.toolCouncilSubmitBrief([]byte(payload)); err != nil {
			t.Fatalf("council_submit_brief failed (%s): %v", role, err)
		}
	}
	summaryOut, err := srv.toolCouncilSummarizeBriefs([]byte(`{"session_id":"` + sessionID + `"}`))
	if err != nil {
		t.Fatalf("council_summarize_briefs failed: %v", err)
	}
	summary := summaryOut.(map[string]any)
	topics, ok := summary["topics"].([]councilTopic)
	if !ok || len(topics) == 0 {
		t.Fatalf("expected topics from council summary, got %T / %v", summary["topics"], summary["topics"])
	}

	for _, topic := range topics {
		reqOut, err := srv.toolCouncilRequestFloor([]byte(fmt.Sprintf(`{"session_id":"%s","topic_id":%d,"role":"ux_director","reason":"kickoff"}`, sessionID, topic.ID)))
		if err != nil {
			t.Fatalf("council_request_floor failed: %v", err)
		}
		reqID, ok := reqOut.(map[string]any)["request_id"].(int64)
		if !ok {
			t.Fatalf("expected int64 request_id, got %T", reqOut.(map[string]any)["request_id"])
		}
		if _, err := srv.toolCouncilGrantFloor([]byte(fmt.Sprintf(`{"session_id":"%s","request_id":%d}`, sessionID, reqID))); err != nil {
			t.Fatalf("council_grant_floor failed: %v", err)
		}
		if _, err := srv.toolCouncilPublishStatement([]byte(fmt.Sprintf(`{"session_id":"%s","request_id":%d,"content":"statement"}`, sessionID, reqID))); err != nil {
			t.Fatalf("council_publish_statement failed: %v", err)
		}
		for _, role := range defaultCouncilRoles {
			if role == "ux_director" {
				continue
			}
			if _, err := srv.toolCouncilRespondTopic([]byte(fmt.Sprintf(`{"session_id":"%s","topic_id":%d,"role":"%s","decision":"pass","content":"pass"}`, sessionID, topic.ID, role))); err != nil {
				t.Fatalf("council_respond_topic failed: %v", err)
			}
		}
		if _, err := srv.toolCouncilCloseTopic([]byte(fmt.Sprintf(`{"session_id":"%s","topic_id":%d}`, sessionID, topic.ID))); err != nil {
			t.Fatalf("council_close_topic failed: %v", err)
		}
	}
	if _, err := srv.toolCouncilFinalizeConsensus([]byte(`{"session_id":"` + sessionID + `"}`)); err != nil {
		t.Fatalf("council_finalize_consensus failed: %v", err)
	}
}

func TestParseIntentFallbackGoal(t *testing.T) {
	intent := parseIntent("로그인 에러 처리 추가")
	if intent.Goal != "로그인 에러 처리 추가" {
		t.Fatalf("unexpected goal: %q", intent.Goal)
	}
	if len(intent.SuccessCriteria) == 0 {
		t.Fatal("expected default success criteria")
	}
}

func TestStartInterviewWithoutIntent(t *testing.T) {
	srv := NewMCPServer(Config{StatePath: filepath.Join(t.TempDir(), "state.json")})
	out, err := srv.toolStartInterview([]byte(`{"session_id":"iv-1"}`))
	if err != nil {
		t.Fatalf("start_interview failed: %v", err)
	}
	result := out.(map[string]any)
	if result["step"] != StepReceived {
		t.Fatalf("expected received step, got %v", result["step"])
	}
	if result["next_step"] != "ingest_intent" {
		t.Fatalf("expected next_step ingest_intent, got %v", result["next_step"])
	}
	qs, ok := result["interview_questions"].([]string)
	if !ok || len(qs) == 0 {
		t.Fatalf("expected interview questions, got %T / %v", result["interview_questions"], result["interview_questions"])
	}
}

func TestStartInterviewWithIntent(t *testing.T) {
	srv := NewMCPServer(Config{StatePath: filepath.Join(t.TempDir(), "state.json")})
	out, err := srv.toolStartInterview([]byte(`{"session_id":"iv-2","raw_intent":"목표: API 안정화\n성공기준: 테스트 통과"}`))
	if err != nil {
		t.Fatalf("start_interview failed: %v", err)
	}
	result := out.(map[string]any)
	if result["step"] != StepIntentCaptured {
		t.Fatalf("expected intent_captured step, got %v", result["step"])
	}
	if result["next_step"] != "council_start_briefing" {
		t.Fatalf("expected next_step council_start_briefing, got %v", result["next_step"])
	}
}

func TestConsultantLanguageDetection(t *testing.T) {
	srv := NewMCPServer(Config{StatePath: filepath.Join(t.TempDir(), "state.json")})
	outKo, err := srv.toolIngestIntent([]byte(`{"session_id":"lang-ko-1","raw_intent":"목표: 로그인 안정화"}`))
	if err != nil {
		t.Fatalf("ingest ko failed: %v", err)
	}
	ko := outKo.(map[string]any)
	if ko["consultant_lang"] != "ko" {
		t.Fatalf("expected consultant_lang ko, got %v", ko["consultant_lang"])
	}

	outEn, err := srv.toolIngestIntent([]byte(`{"session_id":"lang-en-1","raw_intent":"goal: stabilize login API"}`))
	if err != nil {
		t.Fatalf("ingest en failed: %v", err)
	}
	en := outEn.(map[string]any)
	if en["consultant_lang"] != "en" {
		t.Fatalf("expected consultant_lang en, got %v", en["consultant_lang"])
	}
}

func TestStartInterviewAppliesUserProfile(t *testing.T) {
	srv := NewMCPServer(Config{StatePath: filepath.Join(t.TempDir(), "state.json")})
	out, err := srv.toolStartInterview([]byte(`{"session_id":"iv-profile-1","raw_intent":"목표: API 안정화\n성공기준: 테스트 통과","user_profile":{"overall":"advanced","response_need":"high","technical_depth":"technical","domain_knowledge":{"backend":"advanced"}}}`))
	if err != nil {
		t.Fatalf("start_interview failed: %v", err)
	}
	result := out.(map[string]any)
	profile, ok := result["user_profile"].(UserKnowledgeProfile)
	if !ok {
		t.Fatalf("expected user_profile type UserKnowledgeProfile, got %T", result["user_profile"])
	}
	if profile.Overall != "advanced" {
		t.Fatalf("expected advanced overall profile, got %q", profile.Overall)
	}
	if profile.DomainKnowledge["backend"] != "advanced" {
		t.Fatalf("expected backend knowledge advanced, got %#v", profile.DomainKnowledge)
	}
	if profile.Confidence < 0.7 {
		t.Fatalf("expected high confidence for explicit profile, got %f", profile.Confidence)
	}
	if len(profile.Evidence) == 0 {
		t.Fatal("expected profile evidence entries")
	}
}

func TestParseIntentStructured(t *testing.T) {
	input := "목표: 인증 API 테스트 안정화\n범위: auth/login\n제약: 외부 라이브러리 변경 금지"
	intent := parseIntent(input)
	if intent.Goal != "인증 API 테스트 안정화" {
		t.Fatalf("goal parse failed: %q", intent.Goal)
	}
	if len(intent.Scope) != 1 || intent.Scope[0] != "auth/login" {
		t.Fatalf("scope parse failed: %#v", intent.Scope)
	}
	if len(intent.Constraints) != 1 || intent.Constraints[0] != "외부 라이브러리 변경 금지" {
		t.Fatalf("constraints parse failed: %#v", intent.Constraints)
	}
}

func TestParseIntentSuccessCriteria(t *testing.T) {
	intent := parseIntent("목표: API 응답 시간 개선\n성공기준: P95 < 250ms\n성공기준: 테스트 통과")
	if !intent.ExplicitCriteria {
		t.Fatal("expected explicit criteria flag true")
	}
	if len(intent.SuccessCriteria) != 2 {
		t.Fatalf("expected 2 criteria, got %d", len(intent.SuccessCriteria))
	}
	if intent.SuccessCriteria[0] != "P95 < 250ms" {
		t.Fatalf("unexpected first criteria: %q", intent.SuccessCriteria[0])
	}
}

func TestTransitionRules(t *testing.T) {
	if !IsAllowedTransition(StepPlanGenerated, StepPlanApproved) {
		t.Fatal("expected plan_generated -> plan_approved")
	}
	if IsAllowedTransition(StepPlanGenerated, StepActionExecuted) {
		t.Fatal("should not allow plan_generated -> action_executed")
	}
}

func TestIngestIntentToolAndPersist(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")

	srv := NewMCPServer(Config{StatePath: statePath})
	defer os.RemoveAll(dir)

	raw := []byte(`{"raw_intent":"목표: 회귀 테스트 추가\n범위: auth","session_id":""}`)
	out, err := srv.toolIngestIntent(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.(map[string]any)
	sidRaw, ok := got["session_id"]
	if !ok {
		t.Fatal("missing session_id")
	}
	sid := sidRaw.(string)
	s := srv.getOrCreateSession(sid)

	if s.Step != StepIntentCaptured {
		t.Fatalf("unexpected step: %s", s.Step)
	}
	if s.Intent.Raw == "" {
		t.Fatal("intent not captured")
	}

	if err := srv.persistSessions(); err != nil {
		t.Fatalf("failed to persist state: %v", err)
	}

	persisted, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("state should be persisted: %v", err)
	}
	if len(persisted) == 0 {
		t.Fatal("empty persisted state")
	}
}

func TestIngestIntentResetsWorkflowState(t *testing.T) {
	srv := NewMCPServer(Config{StatePath: filepath.Join(t.TempDir(), "state.json")})
	sid := "reset-1"
	session := srv.getOrCreateSession(sid)
	session.SetStep(StepSummarized)
	session.PlanApproved = true
	session.RequirementTags = []string{"old"}
	session.ActionResults = []CommandResult{{Command: "echo old"}}
	session.VerifyResults = []CommandResult{{Command: "echo old"}}
	session.LastError = "old error"

	if _, err := srv.toolIngestIntent([]byte(`{"session_id":"reset-1","raw_intent":"목표: reset"}`)); err != nil {
		t.Fatalf("ingest failed: %v", err)
	}

	if session.Step != StepIntentCaptured {
		t.Fatalf("expected intent_captured, got %s", session.Step)
	}
	if session.PlanApproved {
		t.Fatal("plan approval should be reset")
	}
	if len(session.ActionResults) != 0 || len(session.VerifyResults) != 0 {
		t.Fatalf("expected results reset, got action=%d verify=%d", len(session.ActionResults), len(session.VerifyResults))
	}
	if session.LastError != "" {
		t.Fatalf("last error should be reset, got %q", session.LastError)
	}
	if len(session.StepHistory) != 2 || session.StepHistory[0] != StepReceived || session.StepHistory[1] != StepIntentCaptured {
		t.Fatalf("unexpected step history after reset: %v", session.StepHistory)
	}
}

func TestClarifyIntentNeedsMoreInfoLoop(t *testing.T) {
	srv := NewMCPServer(Config{StatePath: filepath.Join(t.TempDir(), "state.json")})
	if _, err := srv.toolIngestIntent([]byte(`{"session_id":"clarify-loop-1","raw_intent":"목표: 안정성 개선"}`)); err != nil {
		t.Fatalf("ingest failed: %v", err)
	}

	out, err := srv.toolClarifyIntent([]byte(`{"session_id":"clarify-loop-1","answers":{"scope":"internal/server"}}`))
	if err != nil {
		t.Fatalf("clarify failed: %v", err)
	}

	result := out.(map[string]any)
	if result["status"] != "needs_more_info" {
		t.Fatalf("expected needs_more_info, got %v", result["status"])
	}
	if result["next_step"] != "clarify_intent" {
		t.Fatalf("expected next_step clarify_intent, got %v", result["next_step"])
	}
	pending, ok := result["pending_review"].([]string)
	if !ok || len(pending) == 0 {
		t.Fatalf("expected pending follow-up questions, got %T / %v", result["pending_review"], result["pending_review"])
	}
}

func TestClarifyIntentAppliesAnswersAndMovesForward(t *testing.T) {
	srv := NewMCPServer(Config{StatePath: filepath.Join(t.TempDir(), "state.json")})
	if _, err := srv.toolIngestIntent([]byte(`{"session_id":"clarify-loop-2","raw_intent":"목표: 안정성 개선"}`)); err != nil {
		t.Fatalf("ingest failed: %v", err)
	}

	out, err := srv.toolClarifyIntent([]byte(`{"session_id":"clarify-loop-2","answers":{"goal":"로그인 실패율을 줄여 장애 재발을 막기","scope":"internal/server","constraints":"외부망 금지","success_criteria":["테스트 통과","빌드 성공"],"requirement_tags":["stability","server"]}}`))
	if err != nil {
		t.Fatalf("clarify failed: %v", err)
	}

	result := out.(map[string]any)
	if result["status"] != "clarified" {
		t.Fatalf("expected clarified, got %v", result["status"])
	}
	if result["next_step"] != "council_start_briefing" {
		t.Fatalf("expected next_step council_start_briefing, got %v", result["next_step"])
	}

	session := srv.getOrCreateSession("clarify-loop-2")
	if len(session.Intent.Scope) == 0 || session.Intent.Scope[0] != "internal/server" {
		t.Fatalf("scope not reflected: %#v", session.Intent.Scope)
	}
	if len(session.Intent.Constraints) == 0 || session.Intent.Constraints[0] != "외부망 금지" {
		t.Fatalf("constraints not reflected: %#v", session.Intent.Constraints)
	}
	if !session.Intent.ExplicitCriteria {
		t.Fatal("expected explicit criteria true after clarify")
	}
}

func TestClarifyIntentLowResponseNeedAutoDecides(t *testing.T) {
	srv := NewMCPServer(Config{StatePath: filepath.Join(t.TempDir(), "state.json")})
	if _, err := srv.toolIngestIntent([]byte(`{"session_id":"clarify-auto-1","raw_intent":"목표: 로그인 실패율을 줄여 재발을 막는다","user_profile":{"overall":"beginner","response_need":"low","technical_depth":"abstract"}}`)); err != nil {
		t.Fatalf("ingest failed: %v", err)
	}
	out, err := srv.toolClarifyIntent([]byte(`{"session_id":"clarify-auto-1","answers":{"goal":"로그인 실패율을 줄여 재발을 막는다"}}`))
	if err != nil {
		t.Fatalf("clarify failed: %v", err)
	}
	result := out.(map[string]any)
	if result["status"] != "clarified" {
		t.Fatalf("expected clarified status, got %v", result["status"])
	}
	auto, ok := result["auto_decidable"].([]string)
	if !ok {
		t.Fatalf("expected auto_decidable []string, got %T", result["auto_decidable"])
	}
	joined := strings.Join(auto, ",")
	for _, topic := range []string{"scope", "constraints", "success_criteria"} {
		if !strings.Contains(joined, topic) {
			t.Fatalf("expected auto_decidable to include %s, got %v", topic, auto)
		}
	}
}

func TestClarifyIntentLowConfidenceKeepsConservativeAutoDecide(t *testing.T) {
	srv := NewMCPServer(Config{StatePath: filepath.Join(t.TempDir(), "state.json")})
	if _, err := srv.toolIngestIntent([]byte(`{"session_id":"clarify-auto-conservative-1","raw_intent":"목표: 로그인 실패율을 줄여 재발을 막는다","user_profile":{"response_need":"low"}}`)); err != nil {
		t.Fatalf("ingest failed: %v", err)
	}
	out, err := srv.toolClarifyIntent([]byte(`{"session_id":"clarify-auto-conservative-1","answers":{"goal":"로그인 실패율을 줄여 재발을 막는다"}}`))
	if err != nil {
		t.Fatalf("clarify failed: %v", err)
	}
	result := out.(map[string]any)
	if result["status"] != "needs_more_info" {
		t.Fatalf("expected needs_more_info in conservative mode, got %v", result["status"])
	}
	auto, ok := result["auto_decidable"].([]string)
	if !ok {
		t.Fatalf("expected auto_decidable []string, got %T", result["auto_decidable"])
	}
	for _, topic := range auto {
		if topic == "scope" || topic == "constraints" || topic == "success_criteria" {
			t.Fatalf("expected conservative mode to avoid broad auto decide, got %v", auto)
		}
	}
}

func TestIngestIntentInferredProfileHasLowConfidence(t *testing.T) {
	srv := NewMCPServer(Config{StatePath: filepath.Join(t.TempDir(), "state.json")})
	out, err := srv.toolIngestIntent([]byte(`{"session_id":"infer-confidence-1","raw_intent":"목표: 개선"}`))
	if err != nil {
		t.Fatalf("ingest failed: %v", err)
	}
	result := out.(map[string]any)
	profile, ok := result["user_profile"].(UserKnowledgeProfile)
	if !ok {
		t.Fatalf("expected user_profile type UserKnowledgeProfile, got %T", result["user_profile"])
	}
	if profile.Confidence >= 0.55 {
		t.Fatalf("expected low confidence for inferred profile, got %f", profile.Confidence)
	}
	if len(profile.Evidence) == 0 {
		t.Fatal("expected inferred evidence log")
	}
}

func TestCouncilBriefPromptIncludesAutonomyPolicy(t *testing.T) {
	srv := NewMCPServer(Config{StatePath: filepath.Join(t.TempDir(), "state.json")})
	if _, err := srv.toolIngestIntent([]byte(`{"session_id":"council-policy-1","raw_intent":"목표: 인증 API 안정화","user_profile":{"overall":"beginner","domain_knowledge":{"backend":"beginner"}}}`)); err != nil {
		t.Fatalf("ingest failed: %v", err)
	}
	out, err := srv.toolCouncilStartBriefing([]byte(`{"session_id":"council-policy-1"}`))
	if err != nil {
		t.Fatalf("council_start_briefing failed: %v", err)
	}
	result := out.(map[string]any)
	prompts, ok := result["brief_prompts"].([]map[string]any)
	if !ok {
		t.Fatalf("expected []map[string]any brief_prompts, got %T", result["brief_prompts"])
	}
	foundBackend := false
	for _, p := range prompts {
		role, _ := p["role"].(string)
		if role != "backend_lead" {
			continue
		}
		foundBackend = true
		if p["autonomy_level"] != "high" {
			t.Fatalf("expected backend autonomy_level high, got %v", p["autonomy_level"])
		}
		if p["consult_depth"] != "abstract" {
			t.Fatalf("expected backend consult_depth abstract, got %v", p["consult_depth"])
		}
	}
	if !foundBackend {
		t.Fatal("backend_lead prompt not found")
	}
}

func TestCouncilLowConfidenceUsesConservativePolicy(t *testing.T) {
	srv := NewMCPServer(Config{StatePath: filepath.Join(t.TempDir(), "state.json")})
	if _, err := srv.toolIngestIntent([]byte(`{"session_id":"council-policy-low-1","raw_intent":"목표: 개선"}`)); err != nil {
		t.Fatalf("ingest failed: %v", err)
	}
	out, err := srv.toolCouncilStartBriefing([]byte(`{"session_id":"council-policy-low-1"}`))
	if err != nil {
		t.Fatalf("council_start_briefing failed: %v", err)
	}
	result := out.(map[string]any)
	prompts, ok := result["brief_prompts"].([]map[string]any)
	if !ok {
		t.Fatalf("expected []map[string]any brief_prompts, got %T", result["brief_prompts"])
	}
	found := false
	for _, p := range prompts {
		if p["autonomy_level"] != "balanced" {
			continue
		}
		note, _ := p["policy_note"].(string)
		if strings.Contains(strings.ToLower(note), "low confidence") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected conservative low-confidence policy note, got %v", prompts)
	}
}

func TestRunActionDryRun(t *testing.T) {
	srv := NewMCPServer(Config{StatePath: filepath.Join(t.TempDir(), "state.json")})
	sid := "test-session"
	sess := srv.getOrCreateSession(sid)
	sess.Step = StepPlanApproved

	out, err := srv.toolRunAction([]byte(`{"session_id":"test-session","commands":["echo ok"],"dry_run":true,"timeout_sec":1}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := out.(map[string]any)
	step, ok := result["step"].(WorkStep)
	if !ok {
		t.Fatalf("unexpected step type: %T", result["step"])
	}
	if step != StepActionExecuted {
		t.Fatalf("unexpected step after dry run: %s", step)
	}
}

func TestGenerateMockupRequiresPlanGenerated(t *testing.T) {
	srv := NewMCPServer(Config{StatePath: filepath.Join(t.TempDir(), "state.json")})
	s := srv.getOrCreateSession("mockup-1")
	s.Step = StepIntentCaptured

	if _, err := srv.toolGenerateMockup([]byte(`{"session_id":"mockup-1"}`)); err == nil {
		t.Fatal("expected error when generate_mockup is called before plan_generated")
	}
}

func TestSetAndGetAgentRoutingPolicy(t *testing.T) {
	srv := NewMCPServer(Config{StatePath: filepath.Join(t.TempDir(), "state.json")})
	if _, err := srv.toolSetAgentRoutingPolicy([]byte(`{"session_id":"route-1","client_interview_model":"gpt-5.2","worker_model":"gpt-5.3-codex-spark"}`)); err != nil {
		t.Fatalf("set routing policy failed: %v", err)
	}

	out, err := srv.toolGetAgentRoutingPolicy([]byte(`{"session_id":"route-1"}`))
	if err != nil {
		t.Fatalf("get routing policy failed: %v", err)
	}
	result := out.(map[string]any)
	policy, ok := result["routing_policy"].(AgentRoutingPolicy)
	if !ok {
		t.Fatalf("unexpected routing policy type: %T", result["routing_policy"])
	}
	if policy.ClientInterviewModel != "gpt-5.2" {
		t.Fatalf("unexpected interview model: %s", policy.ClientInterviewModel)
	}
	if policy.WorkerModel != "gpt-5.3-codex-spark" {
		t.Fatalf("unexpected worker model: %s", policy.WorkerModel)
	}
}

func TestGeneratePlanRequiresCouncilConsensus(t *testing.T) {
	srv := NewMCPServer(Config{StatePath: filepath.Join(t.TempDir(), "state.json")})
	sid := "council-gate-1"
	if _, err := srv.toolIngestIntent([]byte(`{"session_id":"council-gate-1","raw_intent":"목표: 안정화\n범위: internal/server\n제약: 로컬\n성공기준: 테스트 통과"}`)); err != nil {
		t.Fatalf("ingest failed: %v", err)
	}
	if _, err := srv.toolGeneratePlan([]byte(`{"session_id":"council-gate-1"}`)); err == nil {
		t.Fatal("expected error when generate_plan is called before council consensus")
	}

	forceCouncilConsensus(t, srv, sid)
	if _, err := srv.toolGeneratePlan([]byte(`{"session_id":"council-gate-1"}`)); err != nil {
		t.Fatalf("generate_plan should work after consensus: %v", err)
	}
}

func TestCouncilCloseTopicRequiresAllPass(t *testing.T) {
	srv := NewMCPServer(Config{StatePath: filepath.Join(t.TempDir(), "state.json")})
	sid := "council-topic-gate-1"
	if _, err := srv.toolIngestIntent([]byte(`{"session_id":"council-topic-gate-1","raw_intent":"목표: 안정화\n범위: internal/server\n제약: 로컬\n성공기준: 테스트 통과"}`)); err != nil {
		t.Fatalf("ingest failed: %v", err)
	}
	srv.getOrCreateSession(sid).ProposalAccepted = true
	if _, err := srv.toolCouncilStartBriefing([]byte(`{"session_id":"council-topic-gate-1"}`)); err != nil {
		t.Fatalf("council_start_briefing failed: %v", err)
	}
	for _, role := range defaultCouncilRoles {
		payload := fmt.Sprintf(`{"session_id":"%s","role":"%s","priority":"core","contribution":"%s contribution","quick_decisions":"none"}`, sid, role, role)
		if _, err := srv.toolCouncilSubmitBrief([]byte(payload)); err != nil {
			t.Fatalf("submit brief failed (%s): %v", role, err)
		}
	}
	summaryOut, err := srv.toolCouncilSummarizeBriefs([]byte(`{"session_id":"council-topic-gate-1"}`))
	if err != nil {
		t.Fatalf("council_summarize_briefs failed: %v", err)
	}
	topics := summaryOut.(map[string]any)["topics"].([]councilTopic)
	if len(topics) == 0 {
		t.Fatal("expected at least one topic")
	}
	topicID := topics[0].ID

	reqOut, err := srv.toolCouncilRequestFloor([]byte(fmt.Sprintf(`{"session_id":"%s","topic_id":%d,"role":"ux_director","reason":"kickoff"}`, sid, topicID)))
	if err != nil {
		t.Fatalf("request floor failed: %v", err)
	}
	requestID := reqOut.(map[string]any)["request_id"].(int64)
	if _, err := srv.toolCouncilGrantFloor([]byte(fmt.Sprintf(`{"session_id":"%s","request_id":%d}`, sid, requestID))); err != nil {
		t.Fatalf("grant floor failed: %v", err)
	}
	if _, err := srv.toolCouncilPublishStatement([]byte(fmt.Sprintf(`{"session_id":"%s","request_id":%d,"content":"statement"}`, sid, requestID))); err != nil {
		t.Fatalf("publish statement failed: %v", err)
	}
	if _, err := srv.toolCouncilRespondTopic([]byte(fmt.Sprintf(`{"session_id":"%s","topic_id":%d,"role":"frontend_lead","decision":"pass","content":"pass"}`, sid, topicID))); err != nil {
		t.Fatalf("respond topic failed: %v", err)
	}
	if _, err := srv.toolCouncilCloseTopic([]byte(fmt.Sprintf(`{"session_id":"%s","topic_id":%d}`, sid, topicID))); err == nil {
		t.Fatal("expected close topic to fail before all roles pass")
	}
}

func TestWorkflowHappyPath(t *testing.T) {
	srv := NewMCPServer(Config{StatePath: filepath.Join(t.TempDir(), "state.json")})
	sid := "workflow-1"

	ingestInput := []byte(`{"session_id":"workflow-1","raw_intent":"목표: 로그인 안정성 개선\n범위: internal/auth\n제약: 외부 라이브러리 변경 금지\n성공기준: 테스트 통과\n성공기준: 빌드 성공"}`)
	if _, err := srv.toolIngestIntent(ingestInput); err != nil {
		t.Fatalf("ingest failed: %v", err)
	}
	session := srv.getOrCreateSession(sid)
	if session.Step != StepIntentCaptured {
		t.Fatalf("expected intent_captured, got %s", session.Step)
	}

	clarifyInput := []byte(`{"session_id":"workflow-1","answers":{"requirement_tags":["auth","test"],"success_criteria":["테스트 통과","빌드 성공"]}}`)
	if _, err := srv.toolClarifyIntent(clarifyInput); err != nil {
		t.Fatalf("clarify failed: %v", err)
	}
	forceCouncilConsensus(t, srv, sid)

	if _, err := srv.toolGeneratePlan([]byte(`{"session_id":"workflow-1"}`)); err != nil {
		t.Fatalf("generate plan failed: %v", err)
	}
	if session.Step != StepPlanGenerated {
		t.Fatalf("expected plan_generated, got %s", session.Step)
	}
	if _, err := srv.toolGenerateMockup([]byte(`{"session_id":"workflow-1"}`)); err != nil {
		t.Fatalf("generate mockup failed: %v", err)
	}
	if session.Step != StepMockupReady {
		t.Fatalf("expected mockup_ready, got %s", session.Step)
	}

	approveInput := []byte(`{"session_id":"workflow-1","approved":true,"requirement_tags":["auth","test"],"success_criteria":["테스트 통과","빌드 성공"]}`)
	approveOut, err := srv.toolApprovePlan(approveInput)
	if err != nil {
		t.Fatalf("approve failed: %v", err)
	}
	approve := approveOut.(map[string]any)
	if approve["approved"] != true || approve["step"] != StepPlanApproved {
		t.Fatalf("unexpected approve result: %#v", approve)
	}

	runActionOut, err := srv.toolRunAction([]byte(`{"session_id":"workflow-1","commands":["echo action"],"dry_run":true,"timeout_sec":1}`))
	if err != nil {
		t.Fatalf("run_action failed: %v", err)
	}
	if runActionOut.(map[string]any)["step"] != StepActionExecuted {
		t.Fatalf("run_action should move to action_executed")
	}

	if _, err := srv.toolVerifyResult([]byte(`{"session_id":"workflow-1","commands":["echo ok"],"timeout_sec":1}`)); err != nil {
		t.Fatalf("verify_result failed: %v", err)
	}
	if session.Step != StepVerifyRun {
		t.Fatalf("expected verify_run, got %s", session.Step)
	}
	if _, err := srv.toolRecordUserFeedback([]byte(`{"session_id":"workflow-1","approved":true,"feedback":"looks good"}`)); err != nil {
		t.Fatalf("record_user_feedback failed: %v", err)
	}
	if session.Step != StepSummarized {
		t.Fatalf("expected summarized after user approval, got %s", session.Step)
	}

	summaryOut, err := srv.toolSummarize([]byte(`{"session_id":"workflow-1"}`))
	if err != nil {
		t.Fatalf("summarize failed: %v", err)
	}
	summary := summaryOut.(map[string]any)
	if summary["step"] != StepSummarized {
		t.Fatalf("unexpected summary step: %v", summary["step"])
	}
	if summary["next"] != "done" {
		t.Fatalf("unexpected next action after summarize: %v", summary["next"])
	}
	if summary["action_count"] != 1 {
		t.Fatalf("expected action_count=1, got %v", summary["action_count"])
	}
	if history, ok := summary["step_history"].([]WorkStep); ok {
		if len(history) < 5 {
			t.Fatalf("expected rich step history, got %v", history)
		}
	}
}

func TestSummarizeMovesVerifyRunToSummarized(t *testing.T) {
	srv := NewMCPServer(Config{StatePath: filepath.Join(t.TempDir(), "state.json")})
	sid := "sum-1"
	session := srv.getOrCreateSession(sid)
	session.Step = StepVerifyRun
	session.UserApproved = true
	session.Intent.Goal = "goal"

	out, err := srv.toolSummarize([]byte(`{"session_id":"sum-1"}`))
	if err != nil {
		t.Fatalf("summarize failed: %v", err)
	}

	result := out.(map[string]any)
	if result["step"] != StepSummarized {
		t.Fatalf("expected summarized, got %v", result["step"])
	}
	if session.Step != StepSummarized {
		t.Fatalf("session step not updated: %s", session.Step)
	}
}

func TestConsultProposalPersistedInCouncilDB(t *testing.T) {
	srv := NewMCPServer(Config{StatePath: filepath.Join(t.TempDir(), "state.json")})
	sid := "proposal-db-1"
	if _, err := srv.toolIngestIntent([]byte(`{"session_id":"proposal-db-1","raw_intent":"목표: 안정화\n범위: internal/server\n제약: 로컬\n성공기준: 테스트 통과"}`)); err != nil {
		t.Fatalf("ingest failed: %v", err)
	}
	forceCouncilConsensus(t, srv, sid)
	srv.getOrCreateSession(sid).ProposalAccepted = false

	if _, err := srv.toolClarifyIntent([]byte(`{"session_id":"proposal-db-1","answers":{"goal":"로그인 실패율을 낮춰 재발 장애를 막는다","scope":"internal/server","constraints":"로컬 환경만","success_criteria":["테스트 통과"]}}`)); err != nil {
		t.Fatalf("clarify(create proposal) failed: %v", err)
	}

	statusOut, err := srv.toolCouncilGetStatus([]byte(`{"session_id":"proposal-db-1","message_limit":20}`))
	if err != nil {
		t.Fatalf("council_get_status failed: %v", err)
	}
	status := statusOut.(map[string]any)
	proposals, ok := status["proposals"].([]ConsultProposal)
	if !ok || len(proposals) == 0 {
		t.Fatalf("expected persisted proposals, got %T / %v", status["proposals"], status["proposals"])
	}

	if _, err := srv.toolClarifyIntent([]byte(`{"session_id":"proposal-db-1","answers":{"proposal_feedback":"이대로 진행해"}}`)); err != nil {
		t.Fatalf("clarify(accept proposal) failed: %v", err)
	}

	statusOut, err = srv.toolCouncilGetStatus([]byte(`{"session_id":"proposal-db-1","message_limit":20}`))
	if err != nil {
		t.Fatalf("council_get_status after accept failed: %v", err)
	}
	status = statusOut.(map[string]any)
	proposals, ok = status["proposals"].([]ConsultProposal)
	if !ok || len(proposals) == 0 {
		t.Fatalf("expected proposals after accept, got %T / %v", status["proposals"], status["proposals"])
	}
	last := proposals[len(proposals)-1]
	if last.UserDecision != "accept" {
		t.Fatalf("expected proposal user_decision=accept, got %q", last.UserDecision)
	}
}

func TestClarifyIntentProposalOneQuestionLoop(t *testing.T) {
	srv := NewMCPServer(Config{StatePath: filepath.Join(t.TempDir(), "state.json")})
	sid := "proposal-loop-1"
	if _, err := srv.toolIngestIntent([]byte(`{"session_id":"proposal-loop-1","raw_intent":"목표: 안정화\n범위: internal/server\n제약: 로컬\n성공기준: 테스트 통과"}`)); err != nil {
		t.Fatalf("ingest failed: %v", err)
	}
	forceCouncilConsensus(t, srv, sid)
	srv.getOrCreateSession(sid).ProposalAccepted = false

	out, err := srv.toolClarifyIntent([]byte(`{"session_id":"proposal-loop-1","answers":{"goal":"로그인 실패율을 낮춰 재발 장애를 막는다","scope":"internal/server","constraints":"로컬 환경만","success_criteria":["테스트 통과"]}}`))
	if err != nil {
		t.Fatalf("clarify(create proposal) failed: %v", err)
	}
	result := out.(map[string]any)
	questions, ok := result["follow_up_questions"].([]string)
	if !ok || len(questions) != 1 {
		t.Fatalf("expected exactly 1 follow-up question, got %T / %v", result["follow_up_questions"], result["follow_up_questions"])
	}
	proposal, ok := result["current_proposal"].(*ConsultProposal)
	if !ok || proposal == nil {
		t.Fatalf("expected current_proposal pointer, got %T / %v", result["current_proposal"], result["current_proposal"])
	}
	if len(proposal.Options) != 0 {
		t.Fatalf("expected no fixed options in proposal, got %v", proposal.Options)
	}
}

func TestClarifyIntentAcceptsNaturalProposalFeedback(t *testing.T) {
	srv := NewMCPServer(Config{StatePath: filepath.Join(t.TempDir(), "state.json")})
	sid := "proposal-natural-accept"
	if _, err := srv.toolIngestIntent([]byte(`{"session_id":"proposal-natural-accept","raw_intent":"목표: 안정화\n범위: internal/server\n제약: 로컬\n성공기준: 테스트 통과"}`)); err != nil {
		t.Fatalf("ingest failed: %v", err)
	}
	forceCouncilConsensus(t, srv, sid)
	srv.getOrCreateSession(sid).ProposalAccepted = false

	if _, err := srv.toolClarifyIntent([]byte(`{"session_id":"proposal-natural-accept","answers":{"goal":"로그인 실패율을 낮춰 재발 장애를 막는다","scope":"internal/server","constraints":"로컬 환경만","success_criteria":["테스트 통과"]}}`)); err != nil {
		t.Fatalf("clarify(create proposal) failed: %v", err)
	}
	out, err := srv.toolClarifyIntent([]byte(`{"session_id":"proposal-natural-accept","answers":{"proposal_feedback":"좋아, 이대로 진행해"}}`))
	if err != nil {
		t.Fatalf("clarify(natural accept) failed: %v", err)
	}
	result := out.(map[string]any)
	if result["proposal_accepted"] != true {
		t.Fatalf("expected proposal_accepted true, got %v", result["proposal_accepted"])
	}
	if result["next_step"] != "generate_plan" {
		t.Fatalf("expected next_step generate_plan, got %v", result["next_step"])
	}
}

func TestClarifyIntentConflictFeedbackTriggersCouncilRebrief(t *testing.T) {
	srv := NewMCPServer(Config{StatePath: filepath.Join(t.TempDir(), "state.json")})
	sid := "proposal-conflict-1"
	if _, err := srv.toolIngestIntent([]byte(`{"session_id":"proposal-conflict-1","raw_intent":"목표: 안정화\n범위: internal/server\n제약: 로컬\n성공기준: 테스트 통과"}`)); err != nil {
		t.Fatalf("ingest failed: %v", err)
	}
	forceCouncilConsensus(t, srv, sid)
	srv.getOrCreateSession(sid).ProposalAccepted = false

	if _, err := srv.toolClarifyIntent([]byte(`{"session_id":"proposal-conflict-1","answers":{"goal":"로그인 실패율을 낮춰 재발 장애를 막는다","scope":"internal/server","constraints":"로컬 환경만","success_criteria":["테스트 통과"]}}`)); err != nil {
		t.Fatalf("clarify(create proposal) failed: %v", err)
	}
	out, err := srv.toolClarifyIntent([]byte(`{"session_id":"proposal-conflict-1","answers":{"proposal_feedback":"모바일 몰입감을 유지하면서도 관리자 화면은 즉시 편집 가능해야 해서 요구가 상충돼"}}`))
	if err != nil {
		t.Fatalf("clarify(conflict feedback) failed: %v", err)
	}
	result := out.(map[string]any)
	if result["next_step"] != "council_start_briefing" {
		t.Fatalf("expected next_step council_start_briefing, got %v", result["next_step"])
	}
	if result["question_topic"] != "council_rebrief" {
		t.Fatalf("expected question_topic council_rebrief, got %v", result["question_topic"])
	}
}

func TestJSONRPCWorkflowProcess(t *testing.T) {
	srv := NewMCPServer(Config{StatePath: filepath.Join(t.TempDir(), "state.json")})
	sid := "rpc-1"

	callTool := func(id int, name, args string) map[string]any {
		req := jsonRPCRequest{
			JSONRPC: "2.0",
			ID:      id,
			Method:  "tools/call",
			Params:  json.RawMessage(`{"name":"` + name + `","arguments":` + args + `}`),
		}
		resp := srv.handle(req)
		if resp.Error != nil {
			t.Fatalf("%s failed via jsonrpc: %v", name, resp.Error)
		}
		payload, ok := resp.Result.(map[string]any)
		if !ok {
			t.Fatalf("unexpected result type for %s: %T", name, resp.Result)
		}
		content, ok := payload["structuredContent"].(map[string]any)
		if !ok {
			t.Fatalf("missing structuredContent for %s: %#v", name, payload)
		}
		return content
	}

	callTool(1, "ingest_intent", `{"session_id":"rpc-1","raw_intent":"목표: 안정화\n범위: internal/server\n성공기준: 테스트 통과"}`)
	callTool(2, "council_start_briefing", `{"session_id":"rpc-1"}`)
	for _, role := range defaultCouncilRoles {
		callTool(3, "council_submit_brief", fmt.Sprintf(`{"session_id":"rpc-1","role":"%s","priority":"core","contribution":"%s","quick_decisions":"none"}`, role, role))
	}
	briefSummary := callTool(4, "council_summarize_briefs", `{"session_id":"rpc-1"}`)
	topics, ok := briefSummary["topics"].([]councilTopic)
	if !ok || len(topics) == 0 {
		t.Fatalf("expected topics from council_summarize_briefs")
	}
	for _, topic := range topics {
		req := callTool(5, "council_request_floor", fmt.Sprintf(`{"session_id":"rpc-1","topic_id":%d,"role":"ux_director","reason":"kickoff"}`, topic.ID))
		reqID, ok := req["request_id"].(int64)
		if !ok {
			t.Fatalf("expected request_id int64, got %T", req["request_id"])
		}
		callTool(6, "council_grant_floor", fmt.Sprintf(`{"session_id":"rpc-1","request_id":%d}`, reqID))
		callTool(7, "council_publish_statement", fmt.Sprintf(`{"session_id":"rpc-1","request_id":%d,"content":"statement"}`, reqID))
		for _, role := range defaultCouncilRoles {
			if role == "ux_director" {
				continue
			}
			callTool(8, "council_respond_topic", fmt.Sprintf(`{"session_id":"rpc-1","topic_id":%d,"role":"%s","decision":"pass","content":"pass"}`, topic.ID, role))
		}
		callTool(9, "council_close_topic", fmt.Sprintf(`{"session_id":"rpc-1","topic_id":%d}`, topic.ID))
	}
	callTool(10, "council_finalize_consensus", `{"session_id":"rpc-1"}`)
	callTool(11, "clarify_intent", `{"session_id":"rpc-1","answers":{"goal":"로그인 실패율을 낮춰 재발 장애를 막는다","scope":"internal/server","constraints":"로컬 환경만, 파괴적 변경 금지","requirement_tags":["server"],"success_criteria":["테스트 통과"]}}`)
	callTool(12, "clarify_intent", `{"session_id":"rpc-1","answers":{"proposal_feedback":"좋아, 이대로 진행"}}`)
	callTool(13, "generate_plan", `{"session_id":"rpc-1"}`)
	callTool(14, "generate_mockup", `{"session_id":"rpc-1"}`)
	callTool(15, "approve_plan", `{"session_id":"rpc-1","approved":true,"requirement_tags":["server"],"success_criteria":["테스트 통과"]}`)
	callTool(16, "run_action", `{"session_id":"rpc-1","commands":["echo run"],"dry_run":true}`)
	callTool(17, "verify_result", `{"session_id":"rpc-1","commands":["echo ok"]}`)
	callTool(18, "record_user_feedback", `{"session_id":"rpc-1","approved":true,"feedback":"approved"}`)
	summary := callTool(19, "summarize", `{"session_id":"rpc-1"}`)
	status := callTool(20, "get_session_status", `{"session_id":"rpc-1"}`)

	if summary["step"] != StepSummarized {
		t.Fatalf("expected summarized via jsonrpc, got %v", summary["step"])
	}
	if summary["session_id"] != sid {
		t.Fatalf("unexpected session id: %v", summary["session_id"])
	}
	if status["step"] != StepSummarized {
		t.Fatalf("expected summarized status, got %v", status["step"])
	}
}

func TestVerifyResultFailureReentersPersistentLoop(t *testing.T) {
	srv := NewMCPServer(Config{StatePath: filepath.Join(t.TempDir(), "state.json")})
	sid := "verify-fail-1"
	session := srv.getOrCreateSession(sid)
	session.Step = StepActionExecuted

	out, err := srv.toolVerifyResult([]byte(`{"session_id":"verify-fail-1","commands":["go tool not-a-real-command"]}`))
	if err != nil {
		t.Fatalf("verify_result returned unexpected error: %v", err)
	}
	result := out.(map[string]any)
	if result["step"] != StepIntentCaptured {
		t.Fatalf("expected step intent_captured, got %v", result["step"])
	}
	if result["next_step"] != "generate_plan" {
		t.Fatalf("expected next_step generate_plan, got %v", result["next_step"])
	}
	if session.FixLoopCount != 1 {
		t.Fatalf("expected fix loop count 1, got %d", session.FixLoopCount)
	}
}

func TestVisualReviewGateWithRendererMCP(t *testing.T) {
	srv := NewMCPServer(Config{StatePath: filepath.Join(t.TempDir(), "state.json")})
	sid := "visual-gate-1"
	session := srv.getOrCreateSession(sid)
	session.Step = StepActionExecuted
	session.Intent.Raw = "목표: 웹 전시 UI 몰입감 개선"
	session.Intent.Goal = "웹 전시 UI 몰입감 개선"

	out, err := srv.toolVerifyResult([]byte(`{"session_id":"visual-gate-1","commands":["echo ok"],"available_mcps":["playwright"],"available_mcp_tools":["playwright.screenshot"]}`))
	if err != nil {
		t.Fatalf("verify_result failed: %v", err)
	}
	result := out.(map[string]any)
	if result["next_step"] != "visual_review" {
		t.Fatalf("expected next_step visual_review, got %v", result["next_step"])
	}
	vr, ok := result["visual_review"].(VisualReviewState)
	if !ok {
		t.Fatalf("expected VisualReviewState, got %T", result["visual_review"])
	}
	if !vr.Required {
		t.Fatalf("expected visual review required, got %#v", vr)
	}

	if _, err := srv.toolRecordUserFeedback([]byte(`{"session_id":"visual-gate-1","approved":true}`)); err == nil {
		t.Fatal("expected record_user_feedback to fail before visual_review completion")
	}

	reviewOut, err := srv.toolVisualReview([]byte(`{"session_id":"visual-gate-1","artifacts":["screens/home.png"],"findings":["초기 진입 애니메이션 정상 동작"],"reviewer_notes":"모바일 첫 페인트 정상","ux_director_summary":"작품 진입 흐름이 자연스럽고 몰입 유지됨","ux_decision":"pass"}`))
	if err != nil {
		t.Fatalf("visual_review failed: %v", err)
	}
	review := reviewOut.(map[string]any)
	if review["status"] != "completed" {
		t.Fatalf("expected visual review completed, got %v", review["status"])
	}
	if review["next_step"] != "record_user_feedback" {
		t.Fatalf("expected next_step record_user_feedback, got %v", review["next_step"])
	}

	if _, err := srv.toolRecordUserFeedback([]byte(`{"session_id":"visual-gate-1","approved":true,"feedback":"visual ok"}`)); err != nil {
		t.Fatalf("record_user_feedback failed after visual review: %v", err)
	}
	if session.Step != StepSummarized {
		t.Fatalf("expected summarized after approval, got %s", session.Step)
	}
}

func TestVisualReviewSkippedWithoutRendererMCP(t *testing.T) {
	srv := NewMCPServer(Config{StatePath: filepath.Join(t.TempDir(), "state.json")})
	sid := "visual-skip-1"
	session := srv.getOrCreateSession(sid)
	session.Step = StepActionExecuted
	session.Intent.Raw = "목표: 웹 UI 접근성 개선"
	session.Intent.Goal = "웹 UI 접근성 개선"

	out, err := srv.toolVerifyResult([]byte(`{"session_id":"visual-skip-1","commands":["echo ok"]}`))
	if err != nil {
		t.Fatalf("verify_result failed: %v", err)
	}
	result := out.(map[string]any)
	if result["next_step"] != "record_user_feedback" {
		t.Fatalf("expected next_step record_user_feedback, got %v", result["next_step"])
	}
	vr, ok := result["visual_review"].(VisualReviewState)
	if !ok {
		t.Fatalf("expected VisualReviewState, got %T", result["visual_review"])
	}
	if vr.Required {
		t.Fatalf("expected visual review not required without renderer mcp, got %#v", vr)
	}
	if vr.Status != "skipped" {
		t.Fatalf("expected visual review skipped, got %s", vr.Status)
	}
}

func TestRecordUserFeedbackRejectLoopsBack(t *testing.T) {
	srv := NewMCPServer(Config{StatePath: filepath.Join(t.TempDir(), "state.json")})
	sid := "feedback-1"
	session := srv.getOrCreateSession(sid)
	session.Step = StepVerifyRun
	session.MaxFixLoops = 3

	out, err := srv.toolRecordUserFeedback([]byte(`{"session_id":"feedback-1","approved":false,"feedback":"모바일 레이아웃 수정","required_fixes":["mobile layout"]}`))
	if err != nil {
		t.Fatalf("record_user_feedback failed: %v", err)
	}
	result := out.(map[string]any)
	if result["step"] != StepIntentCaptured {
		t.Fatalf("expected step intent_captured, got %v", result["step"])
	}
	if result["next_step"] != "generate_plan" {
		t.Fatalf("expected generate_plan next_step, got %v", result["next_step"])
	}
	if session.UserApproved {
		t.Fatal("user approval should remain false")
	}
	if session.FixLoopCount != 1 {
		t.Fatalf("expected fix loop count 1, got %d", session.FixLoopCount)
	}
	if len(session.PendingReview) == 0 {
		t.Fatal("expected pending review notes")
	}
}

func TestRecordUserFeedbackRequiresVerifyOrSummarized(t *testing.T) {
	srv := NewMCPServer(Config{StatePath: filepath.Join(t.TempDir(), "state.json")})
	sid := "feedback-invalid-step"
	session := srv.getOrCreateSession(sid)
	session.Step = StepPlanApproved

	if _, err := srv.toolRecordUserFeedback([]byte(`{"session_id":"feedback-invalid-step","approved":true}`)); err == nil {
		t.Fatal("expected error when feedback is called outside verify/summarized step")
	}
}

func TestGetSessionStatus(t *testing.T) {
	srv := NewMCPServer(Config{StatePath: filepath.Join(t.TempDir(), "state.json")})
	sid := "status-1"
	session := srv.getOrCreateSession(sid)
	session.Intent.Goal = "status goal"
	session.SetStep(StepIntentCaptured)

	out, err := srv.toolGetSessionStatus([]byte(`{"session_id":"status-1"}`))
	if err != nil {
		t.Fatalf("status failed: %v", err)
	}

	status := out.(map[string]any)
	if status["step"] != StepIntentCaptured {
		t.Fatalf("unexpected step: %v", status["step"])
	}
	if status["next"] != "council_start_briefing" {
		t.Fatalf("unexpected next: %v", status["next"])
	}
}

func TestSessionPersistenceAndReload(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")

	sid := "reload-test"

	srv1 := NewMCPServer(Config{StatePath: statePath})
	s1 := srv1.getOrCreateSession(sid)
	s1.Intent.Goal = "goal-1"
	s1.Step = StepPlanGenerated
	s1.UpdatedAt = time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	if err := srv1.persistSessions(); err != nil {
		t.Fatalf("persist failed: %v", err)
	}

	srv2 := NewMCPServer(Config{StatePath: statePath})
	s2 := srv2.getOrCreateSession(sid)
	if s2.Intent.Goal != "goal-1" {
		t.Fatalf("goal not restored: %q", s2.Intent.Goal)
	}
	if s2.Step != StepPlanGenerated {
		t.Fatalf("step not restored: %s", s2.Step)
	}
	if !s2.CreatedAt.IsZero() {
		t.Logf("loaded session created_at: %v", s2.CreatedAt)
	}
}

func TestApprovePlanRequiresTagsAndCriteria(t *testing.T) {
	srv := NewMCPServer(Config{StatePath: filepath.Join(t.TempDir(), "state.json")})
	sid := "ap-1"
	session := srv.getOrCreateSession(sid)
	session.Step = StepMockupReady
	session.Plan = &Plan{Title: "plan", Steps: []string{"step1"}, Assumptions: []string{}}
	session.Intent = Intent{
		Goal:             "goal",
		SuccessCriteria:  []string{"테스트 통과"},
		ExplicitCriteria: true,
	}

	out, err := srv.toolApprovePlan([]byte(`{"session_id":"ap-1","approved":true,"requirement_tags":[],"success_criteria":["기타 기준"]}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result := out.(map[string]any)
	if result["approved"] != false {
		t.Fatal("expected approval to be blocked")
	}

	session.Step = StepMockupReady
	session.PlanApproved = false
	session.LastError = ""
	session.Plan = &Plan{Title: "plan", Steps: []string{"step1"}, Assumptions: []string{}}

	out, err = srv.toolApprovePlan([]byte(`{"session_id":"ap-1","approved":true,"requirement_tags":["auth","perf"],"success_criteria":["테스트 통과"]}`))
	if err != nil {
		t.Fatalf("unexpected error on valid approval: %v", err)
	}
	result = out.(map[string]any)
	if result["approved"] != true {
		t.Fatalf("expected approval true, got %#v", result["approved"])
	}
	if result["step"] != StepPlanApproved {
		t.Fatalf("unexpected step: %v", result["step"])
	}
}

func TestToolsListResponseContainsCoreTools(t *testing.T) {
	raw := toolListResponse()
	rawTools, ok := raw["tools"]
	if !ok {
		t.Fatal("tools missing from list response")
	}
	tools, ok := rawTools.([]toolSchema)
	if !ok {
		t.Fatalf("unexpected tools type: %T", rawTools)
	}
	if len(tools) < 30 {
		t.Fatalf("expected at least 30 tools, got %d", len(tools))
	}

	expected := map[string]bool{
		"start_interview":               false,
		"ingest_intent":                 false,
		"clarify_intent":                false,
		"generate_plan":                 false,
		"generate_mockup":               false,
		"approve_plan":                  false,
		"reconcile_session_state":       false,
		"set_agent_routing_policy":      false,
		"get_agent_routing_policy":      false,
		"council_start_briefing":        false,
		"council_submit_brief":          false,
		"council_summarize_briefs":      false,
		"council_request_floor":         false,
		"council_grant_floor":           false,
		"council_publish_statement":     false,
		"council_respond_topic":         false,
		"council_close_topic":           false,
		"council_finalize_consensus":    false,
		"council_get_status":            false,
		"validate_workflow_transition":  false,
		"run_action":                    false,
		"verify_result":                 false,
		"visual_review":                 false,
		"summarize":                     false,
		"record_user_feedback":          false,
		"continue_persistent_execution": false,
		"get_session_status":            false,
		"git_get_state":                 false,
		"git_diff_symbols":              false,
		"git_commit_with_context":       false,
		"git_resolve_conflict":          false,
		"git_bisect_start":              false,
		"git_recover_state":             false,
	}

	for _, tool := range tools {
		if _, ok := expected[tool.Name]; ok {
			expected[tool.Name] = true
		}
		if tool.Name == "" || tool.Description == "" {
			t.Fatalf("tool entry invalid: %+v", tool)
		}
	}
	for name, seen := range expected {
		if !seen {
			t.Fatalf("missing expected tool: %s", name)
		}
	}
}

func TestHandleInitializeAndUnknownMethod(t *testing.T) {
	srv := NewMCPServer(Config{StatePath: filepath.Join(t.TempDir(), "state.json")})

	initReq, _ := json.Marshal(jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
	})
	var initJSONReq jsonRPCRequest
	if err := json.Unmarshal(initReq, &initJSONReq); err != nil {
		t.Fatalf("marshal/unmarshal init request failed: %v", err)
	}
	initResp := srv.handle(initJSONReq)
	if initResp.Error != nil {
		t.Fatalf("initialize should not return error: %v", initResp.Error)
	}
	gotResult, ok := initResp.Result.(map[string]any)
	if !ok {
		t.Fatalf("unexpected initialize result type: %T", initResp.Result)
	}
	if gotResult["protocolVersion"] != "2024-11-05" {
		t.Fatalf("unexpected protocolVersion: %v", gotResult["protocolVersion"])
	}

	unknownReq := jsonRPCRequest{JSONRPC: "2.0", ID: 2, Method: "unknown"}
	unknownResp := srv.handle(unknownReq)
	if unknownResp.Error == nil {
		t.Fatal("expected error for unknown method")
	}
}

func TestIsAllowedCommand(t *testing.T) {
	allow := []string{"go", "git", "npm", "make", "echo"}

	if !isAllowedCommand("go test ./...", allow) {
		t.Fatal("expected go command allowed")
	}
	if !isAllowedCommand("FOO=1 go test ./...", allow) {
		t.Fatal("expected env-prefix command allowed")
	}
	if !isAllowedCommand("./node_modules/.bin/npm run test", allow) {
		t.Fatal("expected npm path allowed by basename")
	}
	if isAllowedCommand("go test ./...; echo hacked", allow) {
		t.Fatal("expected chained command blocked")
	}
	if isAllowedCommand("go test ./... && echo hacked", allow) {
		t.Fatal("expected command with && blocked")
	}
	if isAllowedCommand("rm -rf /", allow) {
		t.Fatal("expected rm command blocked")
	}
}

func TestReadRequestContentLengthMode(t *testing.T) {
	body := `{"jsonrpc":"2.0","id":1,"method":"initialize"}`
	raw := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)
	reader := bufio.NewReader(strings.NewReader(raw))

	req, mode, err := readRequest(reader, wireModeAuto)
	if err != nil {
		t.Fatalf("readRequest failed: %v", err)
	}
	if mode != wireModeContentLength {
		t.Fatalf("expected content-length mode, got %v", mode)
	}
	if req.Method != "initialize" {
		t.Fatalf("unexpected method: %s", req.Method)
	}
}
