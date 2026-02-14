package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"log/slog"
)

type Config struct {
	WorkDir          string
	AllowedCommands  []string
	Logger           *slog.Logger
	StatePath        string
	DiscussionDBPath string
	DefaultProfile   string
}

type MCPServer struct {
	cfg                Config
	mu                 sync.Mutex
	sessions           map[string]*SessionState
	council            *councilStore
	logger             *slog.Logger
	defaultUserProfile userProfileInput
	hasDefaultProfile  bool
	autostartMode      string
	autostartSessionID string
}

func NewMCPServer(cfg Config) *MCPServer {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	if cfg.AllowedCommands == nil || len(cfg.AllowedCommands) == 0 {
		cfg.AllowedCommands = []string{"go", "git", "npm", "make", "echo"}
	}
	srv := &MCPServer{
		cfg:      cfg,
		sessions: map[string]*SessionState{},
		logger:   logger,
		// Session-scoped skill mode: reset to off when MCP process restarts.
		autostartMode: "off",
	}
	if srv.cfg.WorkDir == "" {
		srv.cfg.WorkDir = "."
	}
	if srv.cfg.StatePath == "" {
		srv.cfg.StatePath = filepath.Join(".codex-mcp", "state", "sessions.json")
	}
	if srv.cfg.DiscussionDBPath == "" {
		srv.cfg.DiscussionDBPath = filepath.Join(filepath.Dir(srv.cfg.StatePath), "council.db")
	}
	if srv.cfg.DefaultProfile == "" {
		srv.cfg.DefaultProfile = filepath.Join(filepath.Dir(srv.cfg.StatePath), "default_user_profile.json")
	}
	store, err := newCouncilStore(srv.cfg.DiscussionDBPath)
	if err != nil {
		srv.logger.Error("failed to initialize council store", "error", err)
	} else {
		srv.council = store
	}
	if profile, ok, err := loadDefaultUserProfile(srv.cfg.DefaultProfile); err != nil {
		srv.logger.Warn("failed to load default user profile", "path", srv.cfg.DefaultProfile, "error", err)
	} else if ok {
		srv.defaultUserProfile = profile
		srv.hasDefaultProfile = true
	}
	_ = srv.loadSessions()
	return srv
}

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type jsonRPCResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id"`
	Result  any       `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type toolCallRequest struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

func (s *MCPServer) Run(ctx context.Context) error {
	reader := bufio.NewReader(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)
	defer writer.Flush()
	mode := wireModeAuto

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			req, nextMode, err := readRequest(reader, mode)
			if err != nil {
				if err == io.EOF {
					return nil
				}
				s.send(writer, jsonRPCResponse{JSONRPC: "2.0", ID: nil, Error: &rpcError{Code: -32700, Message: "parse error"}}, mode)
				continue
			}
			mode = nextMode

			resp := s.handle(req)
			if req.ID == nil {
				// JSON-RPC notification: do not reply.
				continue
			}
			s.send(writer, resp, mode)
		}
	}
}

func (s *MCPServer) send(w *bufio.Writer, resp jsonRPCResponse, mode wireMode) {
	bytes, err := json.Marshal(resp)
	if err != nil {
		s.logger.Error("response marshal failed", "error", err)
		return
	}
	if mode == wireModeContentLength {
		_, _ = fmt.Fprintf(w, "Content-Length: %d\r\n\r\n", len(bytes))
		_, _ = w.Write(bytes)
	} else {
		_, _ = w.Write(append(bytes, '\n'))
	}
	_ = w.Flush()
}

type wireMode int

const (
	wireModeAuto wireMode = iota
	wireModeLine
	wireModeContentLength
)

func readRequest(reader *bufio.Reader, mode wireMode) (jsonRPCRequest, wireMode, error) {
	switch mode {
	case wireModeLine:
		req, err := readLineRequest(reader)
		return req, wireModeLine, err
	case wireModeContentLength:
		req, err := readContentLengthRequest(reader)
		return req, wireModeContentLength, err
	default:
		next, err := detectWireMode(reader)
		if err != nil {
			return jsonRPCRequest{}, wireModeAuto, err
		}
		if next == wireModeContentLength {
			req, err := readContentLengthRequest(reader)
			return req, next, err
		}
		req, err := readLineRequest(reader)
		return req, next, err
	}
}

func detectWireMode(reader *bufio.Reader) (wireMode, error) {
	for {
		b, err := reader.Peek(1)
		if err != nil {
			return wireModeAuto, err
		}
		if len(b) == 0 {
			return wireModeAuto, io.EOF
		}
		switch b[0] {
		case ' ', '\t', '\r', '\n':
			_, _ = reader.ReadByte()
			continue
		case 'C', 'c':
			return wireModeContentLength, nil
		default:
			return wireModeLine, nil
		}
	}
}

func readLineRequest(reader *bufio.Reader) (jsonRPCRequest, error) {
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF && strings.TrimSpace(line) != "" {
				var req jsonRPCRequest
				if uerr := json.Unmarshal([]byte(strings.TrimSpace(line)), &req); uerr != nil {
					return jsonRPCRequest{}, uerr
				}
				return req, nil
			}
			return jsonRPCRequest{}, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var req jsonRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			return jsonRPCRequest{}, err
		}
		return req, nil
	}
}

func readContentLengthRequest(reader *bufio.Reader) (jsonRPCRequest, error) {
	contentLength := -1
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return jsonRPCRequest{}, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(parts[0]))
		val := strings.TrimSpace(parts[1])
		if key == "content-length" {
			n, err := strconv.Atoi(val)
			if err != nil {
				return jsonRPCRequest{}, err
			}
			contentLength = n
		}
	}
	if contentLength < 0 {
		return jsonRPCRequest{}, fmt.Errorf("missing content-length header")
	}
	body := make([]byte, contentLength)
	if _, err := io.ReadFull(reader, body); err != nil {
		return jsonRPCRequest{}, err
	}
	var req jsonRPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return jsonRPCRequest{}, err
	}
	return req, nil
}

func (s *MCPServer) handle(req jsonRPCRequest) jsonRPCResponse {
	if req.JSONRPC != "" && req.JSONRPC != "2.0" {
		return jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32600, Message: "invalid jsonrpc version"}}
	}

	switch req.Method {
	case "initialize":
		return jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    "codex-mcp-local",
				"version": "0.1.0",
			},
		}}
	case "tools/list":
		return jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: toolListResponse()}
	case "tools/call":
		var callReq toolCallRequest
		if err := json.Unmarshal(req.Params, &callReq); err != nil {
			return jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32602, Message: "invalid params"}}
		}
		result, err := s.handleTool(callReq)
		if err != nil {
			return jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32000, Message: err.Error()}}
		}
		_ = s.persistSessions()
		return jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{
			"content": []any{
				map[string]any{
					"type": "text",
					"text": mustJSON(result),
				},
			},
			"structuredContent": result,
		}}
	default:
		return jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32601, Message: fmt.Sprintf("method not found: %s", req.Method)}}
	}
}

func (s *MCPServer) getOrCreateSession(id string) *SessionState {
	s.mu.Lock()
	defer s.mu.Unlock()

	if id == "" {
		session := NewSession()
		s.sessions[session.SessionID] = session
		return session
	}

	if session, ok := s.sessions[id]; ok {
		if len(session.StepHistory) == 0 {
			session.StepHistory = []WorkStep{session.Step}
		}
		if session.MaxFixLoops <= 0 {
			session.MaxFixLoops = 5
		}
		if session.TopicDecisions == nil {
			session.TopicDecisions = map[string]string{}
		}
		ensureRoutingPolicyDefaults(session)
		ensureCouncilManagerDefaults(session)
		ensureConsultantLanguageDefaults(session)
		ensureVisualReviewDefaults(session)
		return session
	}

	session := NewSession()
	session.SessionID = id
	s.sessions[id] = session
	return session
}

func (s *MCPServer) handleTool(call toolCallRequest) (any, error) {
	switch call.Name {
	case "start_interview":
		return s.toolStartInterview(call.Arguments)
	case "ingest_intent":
		return s.toolIngestIntent(call.Arguments)
	case "clarify_intent":
		return s.toolClarifyIntent(call.Arguments)
	case "generate_plan":
		return s.toolGeneratePlan(call.Arguments)
	case "generate_mockup":
		return s.toolGenerateMockup(call.Arguments)
	case "approve_plan":
		return s.toolApprovePlan(call.Arguments)
	case "validate_workflow_transition":
		return s.toolValidateTransition(call.Arguments)
	case "run_action":
		return s.toolRunAction(call.Arguments)
	case "verify_result":
		return s.toolVerifyResult(call.Arguments)
	case "visual_review":
		return s.toolVisualReview(call.Arguments)
	case "summarize":
		return s.toolSummarize(call.Arguments)
	case "reconcile_session_state":
		return s.toolReconcileSessionState(call.Arguments)
	case "set_agent_routing_policy":
		return s.toolSetAgentRoutingPolicy(call.Arguments)
	case "get_agent_routing_policy":
		return s.toolGetAgentRoutingPolicy(call.Arguments)
	case "council_configure_team":
		return s.toolCouncilConfigureTeam(call.Arguments)
	case "council_start_briefing":
		return s.toolCouncilStartBriefing(call.Arguments)
	case "council_submit_brief":
		return s.toolCouncilSubmitBrief(call.Arguments)
	case "council_summarize_briefs":
		return s.toolCouncilSummarizeBriefs(call.Arguments)
	case "council_request_floor":
		return s.toolCouncilRequestFloor(call.Arguments)
	case "council_grant_floor":
		return s.toolCouncilGrantFloor(call.Arguments)
	case "council_publish_statement":
		return s.toolCouncilPublishStatement(call.Arguments)
	case "council_respond_topic":
		return s.toolCouncilRespondTopic(call.Arguments)
	case "council_close_topic":
		return s.toolCouncilCloseTopic(call.Arguments)
	case "council_finalize_consensus":
		return s.toolCouncilFinalizeConsensus(call.Arguments)
	case "council_get_status":
		return s.toolCouncilGetStatus(call.Arguments)
	case "record_user_feedback":
		return s.toolRecordUserFeedback(call.Arguments)
	case "continue_persistent_execution":
		return s.toolContinuePersistentExecution(call.Arguments)
	case "get_session_status":
		return s.toolGetSessionStatus(call.Arguments)
	case "autostart_set_mode":
		return s.toolAutostartSetMode(call.Arguments)
	case "autostart_get_mode":
		return s.toolAutostartGetMode(call.Arguments)
	case "git_get_state":
		return s.toolGitGetState(call.Arguments)
	case "git_diff_symbols":
		return s.toolGitDiffSymbols(call.Arguments)
	case "git_commit_with_context":
		return s.toolGitCommitWithContext(call.Arguments)
	case "git_resolve_conflict":
		return s.toolGitResolveConflict(call.Arguments)
	case "git_bisect_start":
		return s.toolGitBisectStart(call.Arguments)
	case "git_recover_state":
		return s.toolGitRecoverState(call.Arguments)
	default:
		return nil, fmt.Errorf("unknown tool: %s", call.Name)
	}
}

func mustJSON(v any) string {
	raw, _ := json.Marshal(v)
	return string(raw)
}

func loadDefaultUserProfile(path string) (userProfileInput, bool, error) {
	if strings.TrimSpace(path) == "" {
		return userProfileInput{}, false, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return userProfileInput{}, false, nil
		}
		return userProfileInput{}, false, err
	}
	var profile userProfileInput
	if err := json.Unmarshal(raw, &profile); err != nil {
		return userProfileInput{}, false, err
	}
	return profile, true, nil
}

func (s *MCPServer) applyDefaultUserProfile(session *SessionState, source string) bool {
	if !s.hasDefaultProfile {
		return false
	}
	mergeUserProfile(session, s.defaultUserProfile, source)
	return true
}

func (s *MCPServer) loadSessions() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	raw, err := os.ReadFile(s.cfg.StatePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	sessions := map[string]*SessionState{}
	if err := json.Unmarshal(raw, &sessions); err != nil {
		return err
	}

	s.sessions = sessions
	for _, session := range s.sessions {
		if len(session.StepHistory) == 0 {
			session.StepHistory = []WorkStep{session.Step}
		}
		if session.MaxFixLoops <= 0 {
			session.MaxFixLoops = 5
		}
		if session.TopicDecisions == nil {
			session.TopicDecisions = map[string]string{}
		}
		ensureRoutingPolicyDefaults(session)
		ensureCouncilManagerDefaults(session)
		ensureConsultantLanguageDefaults(session)
		ensureVisualReviewDefaults(session)
	}
	return nil
}

func ensureRoutingPolicyDefaults(session *SessionState) {
	if session.RoutingPolicy.ClientInterviewModel == "" {
		session.RoutingPolicy.ClientInterviewModel = "gpt-5.2"
	}
	if session.RoutingPolicy.OrchestratorModel == "" {
		session.RoutingPolicy.OrchestratorModel = "gpt-5.3-codex"
	}
	if session.RoutingPolicy.ReviewerModel == "" {
		session.RoutingPolicy.ReviewerModel = "gpt-5.3-codex"
	}
	if session.RoutingPolicy.WorkerModel == "" {
		session.RoutingPolicy.WorkerModel = "gpt-5.3-codex-spark"
	}
}

func (s *MCPServer) persistSessions() error {
	if s.cfg.StatePath == "" {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(s.cfg.StatePath), 0o755); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	raw, err := json.MarshalIndent(s.sessions, "", "  ")
	if err != nil {
		return err
	}

	tmp := s.cfg.StatePath + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.cfg.StatePath)
}
