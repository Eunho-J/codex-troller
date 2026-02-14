package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type councilStore struct {
	db *sql.DB
}

type councilRoleState struct {
	Role           string `json:"role"`
	Model          string `json:"model"`
	BriefSubmitted bool   `json:"brief_submitted"`
	Priority       string `json:"priority"`
	Contribution   string `json:"contribution"`
	QuickDecisions string `json:"quick_decisions"`
}

type councilTopic struct {
	ID        int64  `json:"id"`
	TopicKey  string `json:"topic_key"`
	Title     string `json:"title"`
	Detail    string `json:"detail"`
	Status    string `json:"status"`
	CreatedBy string `json:"created_by"`
}

type councilMessage struct {
	ID        int64     `json:"id"`
	TopicID   int64     `json:"topic_id"`
	Role      string    `json:"role"`
	Action    string    `json:"action"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

var councilRoleOrder = []string{
	"ux_director",
	"frontend_lead",
	"backend_lead",
	"db_lead",
	"asset_manager",
	"security_manager",
}

func isCouncilRole(role string) bool {
	for _, item := range councilRoleOrder {
		if item == role {
			return true
		}
	}
	return false
}

func newCouncilStore(path string) (*councilStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	store := &councilStore{db: db}
	if err := store.initSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (c *councilStore) initSchema() error {
	stmts := []string{
		`PRAGMA journal_mode=WAL;`,
		`CREATE TABLE IF NOT EXISTS council_sessions (
			session_id TEXT PRIMARY KEY,
			status TEXT NOT NULL,
			phase TEXT NOT NULL,
			summary TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS council_roles (
			session_id TEXT NOT NULL,
			role TEXT NOT NULL,
			model TEXT NOT NULL,
			brief TEXT NOT NULL DEFAULT '',
			priority TEXT NOT NULL DEFAULT '',
			contribution TEXT NOT NULL DEFAULT '',
			quick_decisions TEXT NOT NULL DEFAULT '',
			brief_submitted INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY(session_id, role)
		);`,
		`CREATE TABLE IF NOT EXISTS council_topics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL,
			topic_key TEXT NOT NULL,
			title TEXT NOT NULL,
			detail TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'open',
			created_by TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_council_topic_unique ON council_topics(session_id, topic_key);`,
		`CREATE TABLE IF NOT EXISTS council_floor_requests (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL,
			topic_id INTEGER NOT NULL,
			role TEXT NOT NULL,
			status TEXT NOT NULL,
			reason TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS council_topic_votes (
			session_id TEXT NOT NULL,
			topic_id INTEGER NOT NULL,
			role TEXT NOT NULL,
			decision TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			PRIMARY KEY(session_id, topic_id, role)
		);`,
		`CREATE TABLE IF NOT EXISTS council_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL,
			topic_id INTEGER NOT NULL DEFAULT 0,
			role TEXT NOT NULL,
			action TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS council_proposals (
			session_id TEXT NOT NULL,
			version INTEGER NOT NULL,
			domain TEXT NOT NULL,
			summary TEXT NOT NULL,
			options_json TEXT NOT NULL DEFAULT '[]',
			recommended TEXT NOT NULL DEFAULT '',
			user_decision TEXT NOT NULL DEFAULT '',
			user_feedback TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			PRIMARY KEY(session_id, version)
		);`,
	}
	for _, stmt := range stmts {
		if _, err := c.db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func topicKey(input string) string {
	trimmed := strings.TrimSpace(strings.ToLower(input))
	trimmed = strings.ReplaceAll(trimmed, " ", "_")
	trimmed = strings.ReplaceAll(trimmed, "/", "_")
	trimmed = strings.ReplaceAll(trimmed, "-", "_")
	if trimmed == "" {
		return "topic"
	}
	return trimmed
}

func parseTimeOrZero(input string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, input)
	if err != nil {
		return time.Time{}
	}
	return t
}

func modelForCouncilRole(role string, routing AgentRoutingPolicy) string {
	if role == "ux_director" {
		return routing.OrchestratorModel
	}
	return routing.OrchestratorModel
}

func (c *councilStore) startBriefing(sessionID string, routing AgentRoutingPolicy, intent Intent) ([]councilRoleState, []councilTopic, error) {
	tx, err := c.db.Begin()
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	now := nowRFC3339()
	if _, err = tx.Exec(
		`INSERT INTO council_sessions(session_id,status,phase,summary,created_at,updated_at)
		 VALUES(?,?,?,?,?,?)
		 ON CONFLICT(session_id) DO UPDATE SET status=excluded.status, phase=excluded.phase, updated_at=excluded.updated_at`,
		sessionID, "active", "briefing", "", now, now,
	); err != nil {
		return nil, nil, err
	}

	for _, role := range councilRoleOrder {
		if _, err = tx.Exec(
			`INSERT INTO council_roles(session_id,role,model)
			 VALUES(?,?,?)
			 ON CONFLICT(session_id,role) DO UPDATE SET model=excluded.model`,
			sessionID, role, modelForCouncilRole(role, routing),
		); err != nil {
			return nil, nil, err
		}
	}

	defaultTopics := []struct {
		Key    string
		Title  string
		Detail string
	}{
		{"ux_definition", "User Experience Definition", fmt.Sprintf("Goal: %s", strings.TrimSpace(intent.Goal))},
		{"architecture_split", "Implementation Responsibility Split", "Lock responsibility boundaries across frontend/backend/db/assets/security"},
		{"asset_strategy", "Asset Management Strategy", "Define git + local footprint based resume/trace strategy"},
		{"security_boundary", "Security/Permission Boundary", "Lock no-auto-execution zones and approval-required zones"},
	}
	for _, topic := range defaultTopics {
		if _, err = tx.Exec(
			`INSERT INTO council_topics(session_id,topic_key,title,detail,status,created_by,created_at,updated_at)
			 VALUES(?,?,?,?,?,?,?,?)
			 ON CONFLICT(session_id,topic_key) DO UPDATE SET detail=excluded.detail, updated_at=excluded.updated_at`,
			sessionID, topic.Key, topic.Title, topic.Detail, "open", "moderator", now, now,
		); err != nil {
			return nil, nil, err
		}
	}

	if _, err = tx.Exec(
		`INSERT INTO council_messages(session_id,topic_id,role,action,content,created_at) VALUES(?,?,?,?,?,?)`,
		sessionID, 0, "moderator", "kickoff", "Parallel briefing round started", now,
	); err != nil {
		return nil, nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, nil, err
	}
	roles, err := c.loadRoles(sessionID)
	if err != nil {
		return nil, nil, err
	}
	topics, err := c.loadTopics(sessionID)
	if err != nil {
		return nil, nil, err
	}
	return roles, topics, nil
}

func (c *councilStore) ensureSessionRow(sessionID string) error {
	now := nowRFC3339()
	_, err := c.db.Exec(
		`INSERT INTO council_sessions(session_id,status,phase,summary,created_at,updated_at)
		 VALUES(?,?,?,?,?,?)
		 ON CONFLICT(session_id) DO UPDATE SET updated_at=excluded.updated_at`,
		sessionID, "active", "consulting", "", now, now,
	)
	return err
}

func (c *councilStore) resetConsultProposals(sessionID string) error {
	if err := c.ensureSessionRow(sessionID); err != nil {
		return err
	}
	now := nowRFC3339()
	tx, err := c.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.Exec(`DELETE FROM council_proposals WHERE session_id=?`, sessionID); err != nil {
		return err
	}
	if _, err = tx.Exec(`UPDATE council_sessions SET updated_at=? WHERE session_id=?`, now, sessionID); err != nil {
		return err
	}
	return tx.Commit()
}

func (c *councilStore) upsertConsultProposal(sessionID string, proposal ConsultProposal) error {
	if err := c.ensureSessionRow(sessionID); err != nil {
		return err
	}
	optionsJSON, err := json.Marshal(proposal.Options)
	if err != nil {
		return err
	}
	now := nowRFC3339()
	created := proposal.CreatedAt.UTC().Format(time.RFC3339Nano)
	if proposal.CreatedAt.IsZero() {
		created = now
	}
	tx, err := c.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.Exec(
		`INSERT INTO council_proposals(session_id,version,domain,summary,options_json,recommended,user_decision,user_feedback,created_at,updated_at)
		 VALUES(?,?,?,?,?,?,?,?,?,?)
		 ON CONFLICT(session_id,version) DO UPDATE SET
		   domain=excluded.domain,
		   summary=excluded.summary,
		   options_json=excluded.options_json,
		   recommended=excluded.recommended,
		   user_decision=excluded.user_decision,
		   user_feedback=excluded.user_feedback,
		   updated_at=excluded.updated_at`,
		sessionID, proposal.Version, proposal.Domain, proposal.Summary, string(optionsJSON), proposal.Recommended, proposal.UserDecision, proposal.UserFeedback, created, now,
	); err != nil {
		return err
	}

	action := "consult_proposal_upsert"
	content := fmt.Sprintf("v%d decision=%s summary=%s", proposal.Version, strings.TrimSpace(proposal.UserDecision), strings.TrimSpace(proposal.Summary))
	if _, err = tx.Exec(
		`INSERT INTO council_messages(session_id,topic_id,role,action,content,created_at) VALUES(?,?,?,?,?,?)`,
		sessionID, 0, "consultant", action, content, now,
	); err != nil {
		return err
	}

	if _, err = tx.Exec(`UPDATE council_sessions SET updated_at=? WHERE session_id=?`, now, sessionID); err != nil {
		return err
	}
	return tx.Commit()
}

func (c *councilStore) submitBrief(sessionID, role, priority, contribution, quickDecisions string, topicProposals []string) error {
	role = strings.TrimSpace(role)
	if !isCouncilRole(role) {
		return fmt.Errorf("unknown role: %s", role)
	}
	now := nowRFC3339()
	tx, err := c.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.Exec(
		`UPDATE council_roles
		 SET priority=?, contribution=?, quick_decisions=?, brief=?, brief_submitted=1
		 WHERE session_id=? AND role=?`,
		priority, contribution, quickDecisions,
		fmt.Sprintf("priority=%s\ncontribution=%s\nquick_decisions=%s", priority, contribution, quickDecisions),
		sessionID, role,
	); err != nil {
		return err
	}
	if _, err = tx.Exec(
		`INSERT INTO council_messages(session_id,topic_id,role,action,content,created_at) VALUES(?,?,?,?,?,?)`,
		sessionID, 0, role, "brief_submitted", contribution, now,
	); err != nil {
		return err
	}

	for _, proposal := range topicProposals {
		title := strings.TrimSpace(proposal)
		if title == "" {
			continue
		}
		key := topicKey(title)
		if _, err = tx.Exec(
			`INSERT INTO council_topics(session_id,topic_key,title,detail,status,created_by,created_at,updated_at)
			 VALUES(?,?,?,?,?,?,?,?)
			 ON CONFLICT(session_id,topic_key) DO NOTHING`,
			sessionID, key, title, "Topic proposed from manager briefing", "open", role, now, now,
		); err != nil {
			return err
		}
	}

	if _, err = tx.Exec(`UPDATE council_sessions SET updated_at=? WHERE session_id=?`, now, sessionID); err != nil {
		return err
	}
	return tx.Commit()
}

func (c *councilStore) countBriefSubmitted(sessionID string) (int, int, error) {
	var submitted int
	var total int
	if err := c.db.QueryRow(`SELECT COUNT(*) FROM council_roles WHERE session_id=?`, sessionID).Scan(&total); err != nil {
		return 0, 0, err
	}
	if err := c.db.QueryRow(`SELECT COUNT(*) FROM council_roles WHERE session_id=? AND brief_submitted=1`, sessionID).Scan(&submitted); err != nil {
		return 0, 0, err
	}
	return submitted, total, nil
}

func (c *councilStore) summarizeBriefs(sessionID string) (string, []councilTopic, error) {
	submitted, total, err := c.countBriefSubmitted(sessionID)
	if err != nil {
		return "", nil, err
	}
	if total == 0 || submitted < total {
		return "", nil, fmt.Errorf("not all briefs submitted (%d/%d)", submitted, total)
	}

	rows, err := c.db.Query(
		`SELECT role, priority, contribution, quick_decisions FROM council_roles WHERE session_id=? ORDER BY role`, sessionID,
	)
	if err != nil {
		return "", nil, err
	}
	defer rows.Close()

	parts := []string{}
	for rows.Next() {
		var role, priority, contribution, quick string
		if err := rows.Scan(&role, &priority, &contribution, &quick); err != nil {
			return "", nil, err
		}
		parts = append(parts, fmt.Sprintf("[%s] priority=%s / contribution=%s / quick=%s", role, priority, contribution, quick))
	}
	if err := rows.Err(); err != nil {
		return "", nil, err
	}
	summary := strings.Join(parts, "\n")

	now := nowRFC3339()
	if _, err := c.db.Exec(
		`UPDATE council_sessions SET phase='agenda_ready', summary=?, updated_at=? WHERE session_id=?`,
		summary, now, sessionID,
	); err != nil {
		return "", nil, err
	}
	if _, err := c.db.Exec(
		`INSERT INTO council_messages(session_id,topic_id,role,action,content,created_at) VALUES(?,?,?,?,?,?)`,
		sessionID, 0, "moderator", "brief_summary", summary, now,
	); err != nil {
		return "", nil, err
	}
	topics, err := c.loadTopics(sessionID)
	if err != nil {
		return "", nil, err
	}
	return summary, topics, nil
}

func (c *councilStore) requestFloor(sessionID, role string, topicID int64, reason string) (int64, error) {
	role = strings.TrimSpace(role)
	if !isCouncilRole(role) {
		return 0, fmt.Errorf("unknown role: %s", role)
	}
	var topicStatus string
	if err := c.db.QueryRow(
		`SELECT status FROM council_topics WHERE session_id=? AND id=?`,
		sessionID, topicID,
	).Scan(&topicStatus); err != nil {
		return 0, err
	}
	if topicStatus != "open" {
		return 0, fmt.Errorf("topic %d is not open", topicID)
	}

	now := nowRFC3339()
	res, err := c.db.Exec(
		`INSERT INTO council_floor_requests(session_id,topic_id,role,status,reason,created_at,updated_at)
		 VALUES(?,?,?,?,?,?,?)`,
		sessionID, topicID, role, "requested", reason, now, now,
	)
	if err != nil {
		return 0, err
	}
	requestID, _ := res.LastInsertId()
	if _, err := c.db.Exec(
		`INSERT INTO council_messages(session_id,topic_id,role,action,content,created_at) VALUES(?,?,?,?,?,?)`,
		sessionID, topicID, role, "floor_requested", reason, now,
	); err != nil {
		return 0, err
	}
	return requestID, nil
}

func (c *councilStore) grantFloor(sessionID string, requestID int64) (int64, string, error) {
	now := nowRFC3339()
	var topicID int64
	var role string
	var requestStatus string
	if err := c.db.QueryRow(
		`SELECT topic_id, role, status FROM council_floor_requests WHERE id=? AND session_id=?`,
		requestID, sessionID,
	).Scan(&topicID, &role, &requestStatus); err != nil {
		return 0, "", err
	}
	if requestStatus != "requested" {
		return 0, "", fmt.Errorf("floor request %d is not in requested state", requestID)
	}
	if _, err := c.db.Exec(
		`UPDATE council_floor_requests SET status='granted', updated_at=? WHERE id=? AND session_id=?`,
		now, requestID, sessionID,
	); err != nil {
		return 0, "", err
	}
	if _, err := c.db.Exec(
		`INSERT INTO council_messages(session_id,topic_id,role,action,content,created_at) VALUES(?,?,?,?,?,?)`,
		sessionID, topicID, "moderator", "floor_granted", fmt.Sprintf("Floor granted to %s", role), now,
	); err != nil {
		return 0, "", err
	}
	return topicID, role, nil
}

func (c *councilStore) publishStatement(sessionID string, requestID int64, content string) (int64, []string, error) {
	now := nowRFC3339()
	var topicID int64
	var role string
	var status string
	if err := c.db.QueryRow(
		`SELECT topic_id, role, status FROM council_floor_requests WHERE id=? AND session_id=?`,
		requestID, sessionID,
	).Scan(&topicID, &role, &status); err != nil {
		return 0, nil, err
	}
	if status != "granted" {
		return 0, nil, fmt.Errorf("floor request %d is not granted", requestID)
	}

	tx, err := c.db.Begin()
	if err != nil {
		return 0, nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.Exec(
		`UPDATE council_floor_requests SET status='done', updated_at=? WHERE id=? AND session_id=?`,
		now, requestID, sessionID,
	); err != nil {
		return 0, nil, err
	}
	if _, err = tx.Exec(
		`INSERT INTO council_messages(session_id,topic_id,role,action,content,created_at) VALUES(?,?,?,?,?,?)`,
		sessionID, topicID, role, "statement", content, now,
	); err != nil {
		return 0, nil, err
	}

	waitingRoles := []string{}
	for _, r := range councilRoleOrder {
		decision := "pending"
		if r == role {
			decision = "pass"
		} else {
			waitingRoles = append(waitingRoles, r)
		}
		if _, err = tx.Exec(
			`INSERT INTO council_topic_votes(session_id,topic_id,role,decision,updated_at)
			 VALUES(?,?,?,?,?)
			 ON CONFLICT(session_id,topic_id,role) DO UPDATE SET decision=excluded.decision, updated_at=excluded.updated_at`,
			sessionID, topicID, r, decision, now,
		); err != nil {
			return 0, nil, err
		}
	}
	if err = tx.Commit(); err != nil {
		return 0, nil, err
	}
	return topicID, waitingRoles, nil
}

func (c *councilStore) respondTopic(sessionID string, topicID int64, role, decision, content string) (bool, []string, error) {
	normalized := strings.ToLower(strings.TrimSpace(decision))
	if normalized != "pass" && normalized != "raise" {
		return false, nil, fmt.Errorf("invalid decision: %s", decision)
	}
	role = strings.TrimSpace(role)
	if !isCouncilRole(role) {
		return false, nil, fmt.Errorf("unknown role: %s", role)
	}
	var topicStatus string
	if err := c.db.QueryRow(
		`SELECT status FROM council_topics WHERE session_id=? AND id=?`,
		sessionID, topicID,
	).Scan(&topicStatus); err != nil {
		return false, nil, err
	}
	if topicStatus != "open" {
		return false, nil, fmt.Errorf("topic %d is not open", topicID)
	}
	now := nowRFC3339()
	if _, err := c.db.Exec(
		`INSERT INTO council_topic_votes(session_id,topic_id,role,decision,updated_at)
		 VALUES(?,?,?,?,?)
		 ON CONFLICT(session_id,topic_id,role) DO UPDATE SET decision=excluded.decision, updated_at=excluded.updated_at`,
		sessionID, topicID, role, normalized, now,
	); err != nil {
		return false, nil, err
	}
	action := "topic_pass"
	if normalized == "raise" {
		action = "topic_raise"
	}
	if _, err := c.db.Exec(
		`INSERT INTO council_messages(session_id,topic_id,role,action,content,created_at) VALUES(?,?,?,?,?,?)`,
		sessionID, topicID, role, action, content, now,
	); err != nil {
		return false, nil, err
	}

	rows, err := c.db.Query(
		`SELECT role, decision FROM council_topic_votes WHERE session_id=? AND topic_id=?`,
		sessionID, topicID,
	)
	if err != nil {
		return false, nil, err
	}
	defer rows.Close()

	pending := map[string]struct{}{}
	for _, r := range councilRoleOrder {
		pending[r] = struct{}{}
	}
	raiseCount := 0
	for rows.Next() {
		var r, d string
		if err := rows.Scan(&r, &d); err != nil {
			return false, nil, err
		}
		delete(pending, r)
		if d == "pending" {
			pending[r] = struct{}{}
		}
		if d == "raise" {
			raiseCount++
		}
	}
	if err := rows.Err(); err != nil {
		return false, nil, err
	}
	pendingRoles := []string{}
	for _, r := range councilRoleOrder {
		if _, ok := pending[r]; ok {
			pendingRoles = append(pendingRoles, r)
		}
	}
	closable := len(pendingRoles) == 0 && raiseCount == 0
	return closable, pendingRoles, nil
}

func (c *councilStore) closeTopic(sessionID string, topicID int64) (int, error) {
	var topicStatus string
	if err := c.db.QueryRow(
		`SELECT status FROM council_topics WHERE session_id=? AND id=?`,
		sessionID, topicID,
	).Scan(&topicStatus); err != nil {
		return 0, err
	}
	if topicStatus != "open" {
		return 0, fmt.Errorf("topic %d is not open", topicID)
	}

	rows, err := c.db.Query(
		`SELECT role, decision FROM council_topic_votes WHERE session_id=? AND topic_id=?`,
		sessionID, topicID,
	)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	seen := map[string]bool{}
	for rows.Next() {
		var role, decision string
		if err := rows.Scan(&role, &decision); err != nil {
			return 0, err
		}
		if decision != "pass" {
			return 0, fmt.Errorf("topic %d cannot close: role %s decision=%s", topicID, role, decision)
		}
		seen[role] = true
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(seen) < len(councilRoleOrder) {
		return 0, fmt.Errorf("topic %d cannot close: not all roles responded", topicID)
	}

	now := nowRFC3339()
	if _, err = c.db.Exec(
		`UPDATE council_topics SET status='closed', updated_at=? WHERE session_id=? AND id=?`,
		now, sessionID, topicID,
	); err != nil {
		return 0, err
	}
	if _, err := c.db.Exec(
		`INSERT INTO council_messages(session_id,topic_id,role,action,content,created_at) VALUES(?,?,?,?,?,?)`,
		sessionID, topicID, "moderator", "topic_closed", "Topic closed", now,
	); err != nil {
		return 0, err
	}
	var openCount int
	if err := c.db.QueryRow(`SELECT COUNT(*) FROM council_topics WHERE session_id=? AND status='open'`, sessionID).Scan(&openCount); err != nil {
		return 0, err
	}
	return openCount, nil
}

func (c *councilStore) finalizeConsensus(sessionID string) error {
	submitted, total, err := c.countBriefSubmitted(sessionID)
	if err != nil {
		return err
	}
	if submitted < total {
		return fmt.Errorf("cannot finalize before all briefs submitted (%d/%d)", submitted, total)
	}
	var openCount int
	if err := c.db.QueryRow(`SELECT COUNT(*) FROM council_topics WHERE session_id=? AND status='open'`, sessionID).Scan(&openCount); err != nil {
		return err
	}
	if openCount > 0 {
		return fmt.Errorf("cannot finalize with %d open topics", openCount)
	}
	now := nowRFC3339()
	if _, err := c.db.Exec(
		`UPDATE council_sessions SET status='consensus_reached', phase='finalized', updated_at=? WHERE session_id=?`,
		now, sessionID,
	); err != nil {
		return err
	}
	if _, err := c.db.Exec(
		`INSERT INTO council_messages(session_id,topic_id,role,action,content,created_at) VALUES(?,?,?,?,?,?)`,
		sessionID, 0, "moderator", "consensus_finalized", "Consensus finalized for all topics", now,
	); err != nil {
		return err
	}
	return nil
}

func (c *councilStore) loadRoles(sessionID string) ([]councilRoleState, error) {
	rows, err := c.db.Query(
		`SELECT role, model, brief_submitted, priority, contribution, quick_decisions
		 FROM council_roles WHERE session_id=? ORDER BY role`,
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []councilRoleState{}
	for rows.Next() {
		var item councilRoleState
		var submitted int
		if err := rows.Scan(&item.Role, &item.Model, &submitted, &item.Priority, &item.Contribution, &item.QuickDecisions); err != nil {
			return nil, err
		}
		item.BriefSubmitted = submitted == 1
		out = append(out, item)
	}
	return out, rows.Err()
}

func (c *councilStore) loadTopics(sessionID string) ([]councilTopic, error) {
	rows, err := c.db.Query(
		`SELECT id, topic_key, title, detail, status, created_by
		 FROM council_topics WHERE session_id=? ORDER BY id`,
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []councilTopic{}
	for rows.Next() {
		var item councilTopic
		if err := rows.Scan(&item.ID, &item.TopicKey, &item.Title, &item.Detail, &item.Status, &item.CreatedBy); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (c *councilStore) loadMessages(sessionID string, limit int) ([]councilMessage, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := c.db.Query(
		`SELECT id, topic_id, role, action, content, created_at
		 FROM council_messages WHERE session_id=? ORDER BY id DESC LIMIT ?`,
		sessionID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	reversed := []councilMessage{}
	for rows.Next() {
		var item councilMessage
		var created string
		if err := rows.Scan(&item.ID, &item.TopicID, &item.Role, &item.Action, &item.Content, &created); err != nil {
			return nil, err
		}
		item.CreatedAt = parseTimeOrZero(created)
		reversed = append(reversed, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]councilMessage, 0, len(reversed))
	for i := len(reversed) - 1; i >= 0; i-- {
		out = append(out, reversed[i])
	}
	return out, nil
}

func (c *councilStore) loadConsultProposals(sessionID string) ([]ConsultProposal, error) {
	rows, err := c.db.Query(
		`SELECT version, domain, summary, options_json, recommended, user_decision, user_feedback, created_at
		 FROM council_proposals WHERE session_id=? ORDER BY version ASC`,
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []ConsultProposal{}
	for rows.Next() {
		var item ConsultProposal
		var optionsJSON string
		var created string
		if err := rows.Scan(&item.Version, &item.Domain, &item.Summary, &optionsJSON, &item.Recommended, &item.UserDecision, &item.UserFeedback, &created); err != nil {
			return nil, err
		}
		if strings.TrimSpace(optionsJSON) != "" {
			var options []string
			if err := json.Unmarshal([]byte(optionsJSON), &options); err != nil {
				return nil, err
			}
			item.Options = options
		}
		item.CreatedAt = parseTimeOrZero(created)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (c *councilStore) getSessionMeta(sessionID string) (string, string, string, error) {
	var status, phase, summary string
	err := c.db.QueryRow(
		`SELECT status, phase, summary FROM council_sessions WHERE session_id=?`,
		sessionID,
	).Scan(&status, &phase, &summary)
	if err != nil {
		return "", "", "", err
	}
	return status, phase, summary, nil
}
