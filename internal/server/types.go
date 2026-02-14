package server

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

type WorkStep string

const (
	StepReceived       WorkStep = "received"
	StepIntentCaptured WorkStep = "intent_captured"
	StepPlanGenerated  WorkStep = "plan_generated"
	StepMockupReady    WorkStep = "mockup_ready"
	StepPlanApproved   WorkStep = "plan_approved"
	StepActionExecuted WorkStep = "action_executed"
	StepVerifyRun      WorkStep = "verify_run"
	StepSummarized     WorkStep = "summarized"
	StepFailed         WorkStep = "failed"
)

type Intent struct {
	Raw              string   `json:"raw"`
	Goal             string   `json:"goal"`
	Scope            []string `json:"scope"`
	Constraints      []string `json:"constraints"`
	SuccessCriteria  []string `json:"success_criteria"`
	ExplicitCriteria bool     `json:"explicit_criteria"`
	Assumptions      []string `json:"assumptions"`
}

type Plan struct {
	Title       string   `json:"title"`
	Steps       []string `json:"steps"`
	Assumptions []string `json:"assumptions"`
	Risks       []string `json:"risks"`
}

type MockupArtifact struct {
	Version       int       `json:"version"`
	Summary       string    `json:"summary"`
	KeyFlows      []string  `json:"key_flows"`
	OpenQuestions []string  `json:"open_questions"`
	Assumptions   []string  `json:"assumptions"`
	CreatedAt     time.Time `json:"created_at"`
}

type ConsultProposal struct {
	Version      int       `json:"version"`
	Domain       string    `json:"domain"`
	Summary      string    `json:"summary"`
	Options      []string  `json:"options"`
	Recommended  string    `json:"recommended"`
	UserDecision string    `json:"user_decision"`
	UserFeedback string    `json:"user_feedback"`
	CreatedAt    time.Time `json:"created_at"`
}

type RepoFootprint struct {
	Head         string    `json:"head"`
	Branch       string    `json:"branch"`
	Dirty        bool      `json:"dirty"`
	ChangedFiles int       `json:"changed_files"`
	StatusDigest string    `json:"status_digest"`
	CapturedAt   time.Time `json:"captured_at"`
}

type AgentRoutingPolicy struct {
	ClientInterviewModel string `json:"client_interview_model"`
	OrchestratorModel    string `json:"orchestrator_model"`
	ReviewerModel        string `json:"reviewer_model"`
	WorkerModel          string `json:"worker_model"`
}

type CouncilManager struct {
	Role   string `json:"role"`
	Domain string `json:"domain"`
	Model  string `json:"model"`
}

type UserKnowledgeProfile struct {
	Overall         string            `json:"overall"`
	DomainKnowledge map[string]string `json:"domain_knowledge"`
	ResponseNeed    string            `json:"response_need"`
	TechnicalDepth  string            `json:"technical_depth"`
	Confidence      float64           `json:"confidence"`
	Evidence        []string          `json:"evidence"`
}

type VisualReviewState struct {
	Required          bool      `json:"required"`
	RendererAvailable bool      `json:"renderer_available"`
	RendererSource    string    `json:"renderer_source"`
	RendererMatches   []string  `json:"renderer_matches"`
	Status            string    `json:"status"`
	Artifacts         []string  `json:"artifacts"`
	Findings          []string  `json:"findings"`
	ReviewerNotes     string    `json:"reviewer_notes"`
	UXDirectorSummary string    `json:"ux_director_summary"`
	UXDecision        string    `json:"ux_decision"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type CommandResult struct {
	Command    string `json:"command"`
	ExitCode   int    `json:"exit_code"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	DurationMS int64  `json:"duration_ms"`
	Error      string `json:"error,omitempty"`
}

type SessionState struct {
	SessionID         string               `json:"session_id"`
	Step              WorkStep             `json:"step"`
	StepHistory       []WorkStep           `json:"step_history"`
	Intent            Intent               `json:"intent"`
	Plan              *Plan                `json:"plan"`
	Mockup            *MockupArtifact      `json:"mockup"`
	ProposalHistory   []ConsultProposal    `json:"proposal_history"`
	ProposalAccepted  bool                 `json:"proposal_accepted"`
	CouncilConsensus  bool                 `json:"council_consensus"`
	CouncilPhase      string               `json:"council_phase"`
	RequirementTags   []string             `json:"requirement_tags"`
	ApprovedCriteria  []string             `json:"approved_criteria"`
	PlanApproved      bool                 `json:"plan_approved"`
	UserApproved      bool                 `json:"user_approved"`
	UserFeedback      []string             `json:"user_feedback"`
	FixLoopCount      int                  `json:"fix_loop_count"`
	MaxFixLoops       int                  `json:"max_fix_loops"`
	ActionResults     []CommandResult      `json:"action_results"`
	VerifyResults     []CommandResult      `json:"verify_results"`
	ClarifyNotes      []string             `json:"clarify_notes"`
	PendingReview     []string             `json:"pending_review"`
	TopicDecisions    map[string]string    `json:"topic_decisions"`
	BaselineFootprint RepoFootprint        `json:"baseline_footprint"`
	LastFootprint     RepoFootprint        `json:"last_footprint"`
	ReconcileNeeded   bool                 `json:"reconcile_needed"`
	RoutingPolicy     AgentRoutingPolicy   `json:"routing_policy"`
	CouncilManagers   []CouncilManager     `json:"council_managers"`
	UserProfile       UserKnowledgeProfile `json:"user_profile"`
	ConsultantLang    string               `json:"consultant_lang"`
	AvailableMCPs     []string             `json:"available_mcps"`
	AvailableMCPTools []string             `json:"available_mcp_tools"`
	VisualReview      VisualReviewState    `json:"visual_review"`
	LastError         string               `json:"last_error"`
	AllowedDomains    []string             `json:"allowed_domains"`
	CreatedAt         time.Time            `json:"created_at"`
	UpdatedAt         time.Time            `json:"updated_at"`
}

func NewSession() *SessionState {
	id := randomID()
	now := time.Now().UTC()
	return &SessionState{
		SessionID:      id,
		Step:           StepReceived,
		StepHistory:    []WorkStep{StepReceived},
		MaxFixLoops:    5,
		TopicDecisions: map[string]string{},
		RoutingPolicy: AgentRoutingPolicy{
			ClientInterviewModel: "gpt-5.2",
			OrchestratorModel:    "gpt-5.3-codex",
			ReviewerModel:        "gpt-5.3-codex",
			WorkerModel:          "gpt-5.3-codex-spark",
		},
		CouncilManagers: []CouncilManager{
			{Role: "ux_director", Domain: "frontend"},
			{Role: "frontend_lead", Domain: "frontend"},
			{Role: "backend_lead", Domain: "backend"},
			{Role: "db_lead", Domain: "db"},
			{Role: "asset_manager", Domain: "asset"},
			{Role: "security_manager", Domain: "security"},
		},
		UserProfile: UserKnowledgeProfile{
			Overall:         "unknown",
			DomainKnowledge: map[string]string{},
			ResponseNeed:    "balanced",
			TechnicalDepth:  "balanced",
			Confidence:      0.2,
			Evidence:        []string{},
		},
		ConsultantLang: "en",
		VisualReview: VisualReviewState{
			Status: "not_required",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (s *SessionState) SetStep(next WorkStep) {
	if len(s.StepHistory) == 0 {
		s.StepHistory = []WorkStep{s.Step}
	}
	if s.Step != next {
		s.Step = next
		s.StepHistory = append(s.StepHistory, next)
		return
	}
	s.Step = next
}

func randomID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("mcp-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

var TransitionRules = map[WorkStep][]WorkStep{
	StepReceived:       {StepIntentCaptured, StepFailed},
	StepIntentCaptured: {StepPlanGenerated, StepFailed},
	StepPlanGenerated:  {StepMockupReady, StepPlanApproved, StepFailed},
	StepMockupReady:    {StepPlanApproved, StepIntentCaptured, StepFailed},
	StepPlanApproved:   {StepActionExecuted, StepFailed},
	StepActionExecuted: {StepVerifyRun, StepFailed},
	StepVerifyRun:      {StepSummarized, StepIntentCaptured, StepPlanGenerated, StepFailed},
	StepSummarized:     {StepSummarized},
	StepFailed:         {StepIntentCaptured, StepPlanGenerated, StepFailed, StepReceived},
}

func IsAllowedTransition(current, next WorkStep) bool {
	allowed, ok := TransitionRules[current]
	if !ok {
		return false
	}
	for _, v := range allowed {
		if v == next {
			return true
		}
	}
	return false
}
