package vertexai

import (
	"context"
	"fmt"
	"strings"
	"time"

	cx "cloud.google.com/go/dialogflow/cx/apiv3"
	"cloud.google.com/go/dialogflow/cx/apiv3/cxpb"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/structpb"
)

// VertexAIClient wraps the Google Cloud Dialogflow CX Client (for Playbooks/Agents)
type VertexAIClient struct {
	sessionClient *cx.SessionsClient
	projectID     string
	location      string
	agentID       string // This is the "Agent ID" from the URL
}

// SendMessageRequest describes the data needed to interact with the Agent.
type SendMessageRequest struct {
	SessionID        string
	Query            string
	History          []HistoryEntry // Kept for compatibility, but CX handles history server-side via SessionID
	UserContext      *UserContext
	ContextDocuments []string // Not directly used in CX DetectIntent, relies on Agent Data Store settings
	ActiveDocument   string   // Not used in standard CX
	Metadata         map[string]string
	MaxHistory       int    // Handled server-side by CX
	ServingConfig    string // Not used in standard CX
}

// HistoryEntry captures a single message.
// Note: In Dialogflow CX, history is automatic based on SessionID.
type HistoryEntry struct {
	Role      HistoryRole
	Text      string
	CreatedAt time.Time
}

// HistoryRole defines author.
type HistoryRole string

const (
	HistoryRoleUser HistoryRole = "user"
	HistoryRoleAI   HistoryRole = "ai"
)

// UserContext provides labels/params for the Agent.
type UserContext struct {
	UserID      string
	DisplayName string
	Metadata    map[string]string
}

// AIResponse captures the unified response from the agent
type AIResponse struct {
	Text               string
	Confidence         float32
	Sources            []string
	Intent             string
	RequiresEscalation bool
}

// NewVertexAIClient creates a new client for Dialogflow CX (Agent/Playbook)
func NewVertexAIClient(ctx context.Context, projectID, location, agentID, credentialsPath string) (*VertexAIClient, error) {
	opts := []option.ClientOption{}
	if credentialsPath != "" {
		opts = append(opts, option.WithCredentialsFile(credentialsPath))
	}

	// For regional endpoints (anything other than 'global'), you must specify the API endpoint.
	if location != "global" {
		endpoint := fmt.Sprintf("%s-dialogflow.googleapis.com:443", location)
		opts = append(opts, option.WithEndpoint(endpoint))
	}

	// Initialize the Sessions client
	client, err := cx.NewSessionsClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create dialogflow cx client: %w", err)
	}

	return &VertexAIClient{
		sessionClient: client,
		projectID:     projectID,
		location:      location,
		agentID:       agentID,
	}, nil
}

// Close closes the underlying client connection
func (c *VertexAIClient) Close() error {
	return c.sessionClient.Close()
}

// SendMessage sends a user query to the Playbook Agent.
func (c *VertexAIClient) SendMessage(ctx context.Context, req *SendMessageRequest) (*AIResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	if strings.TrimSpace(req.SessionID) == "" {
		return nil, fmt.Errorf("session id is required")
	}
	if strings.TrimSpace(req.Query) == "" {
		return nil, fmt.Errorf("query text is required")
	}

	// 1. Construct the Session Path
	// Format: projects/{project}/locations/{location}/agents/{agent}/sessions/{session}
	sessionPath := fmt.Sprintf("projects/%s/locations/%s/agents/%s/sessions/%s",
		c.projectID, c.location, c.agentID, req.SessionID)

	// 2. Build Session Parameters (User Context)
	// This lets your Playbook use variables like $session.params.user_name
	sessionParams := make(map[string]interface{})
	if req.UserContext != nil {
		sessionParams["user_id"] = req.UserContext.UserID
		sessionParams["user_name"] = req.UserContext.DisplayName
		for k, v := range req.UserContext.Metadata {
			sessionParams[k] = v
		}
	}
	// Add extra metadata to params
	for k, v := range req.Metadata {
		sessionParams[k] = v
	}

	paramsStruct, err := structpb.NewStruct(sessionParams)
	if err != nil {
		return nil, fmt.Errorf("failed to encode session params: %w", err)
	}

	// 3. Create the Request
	cxReq := &cxpb.DetectIntentRequest{
		Session: sessionPath,
		QueryInput: &cxpb.QueryInput{
			Input: &cxpb.QueryInput_Text{
				Text: &cxpb.TextInput{
					Text: strings.TrimSpace(req.Query),
				},
			},
			LanguageCode: "en", // Default to English, or make this configurable
		},
		QueryParams: &cxpb.QueryParameters{
			Parameters: paramsStruct,
		},
	}

	// 4. Call the API
	resp, err := c.sessionClient.DetectIntent(ctx, cxReq)
	if err != nil {
		return nil, fmt.Errorf("agent interaction failed: %w", err)
	}

	// 5. Extract Results
	result := &AIResponse{
		Text:       extractResponseText(resp),
		Confidence: extractConfidence(resp),
		Sources:    extractSources(resp),
		Intent:     extractIntent(resp),
	}

	// 6. Escalation Logic (The "Magic Word" Check)
	// We check if the AI output contains our specific signal code
	if strings.Contains(result.Text, "###ESCALATE_TO_HUMAN###") {
		result.RequiresEscalation = true
		// Clean up the text so the user doesn't see the ugly code
		result.Text = strings.ReplaceAll(result.Text, "###ESCALATE_TO_HUMAN###", "")
		if strings.TrimSpace(result.Text) == "" {
			result.Text = "Connecting you to a human agent..."
		}
	}

	if result.Text == "" {
		result.Text = "I apologize, but I couldn't process that request."
	}

	return result, nil
}

// extractResponseText concatenates the agent's text responses
func extractResponseText(resp *cxpb.DetectIntentResponse) string {
	if resp == nil || resp.QueryResult == nil {
		return ""
	}
	var sb strings.Builder
	for _, msg := range resp.QueryResult.ResponseMessages {
		if text := msg.GetText(); text != nil {
			for _, line := range text.Text {
				sb.WriteString(line)
				sb.WriteString(" ")
			}
		}
	}
	return strings.TrimSpace(sb.String())
}

// extractConfidence gets the match confidence (0.0 to 1.0)
func extractConfidence(resp *cxpb.DetectIntentResponse) float32 {
	if resp == nil || resp.QueryResult == nil || resp.QueryResult.Match == nil {
		return 0
	}
	return float32(resp.QueryResult.Match.Confidence)
}

// extractIntent gets the matched intent display name
func extractIntent(resp *cxpb.DetectIntentResponse) string {
	if resp == nil || resp.QueryResult == nil || resp.QueryResult.Match == nil || resp.QueryResult.Match.Intent == nil {
		return ""
	}
	return resp.QueryResult.Match.Intent.DisplayName
}

// extractSources attempts to find citations (if Playbook is using a Data Store)
// Note: Playbook citations structure is complex; this is a basic extraction.
func extractSources(_ *cxpb.DetectIntentResponse) []string {
	// Dialogflow CX doesn't always return structured "sources" in the same simple way
	// as the Search API. Often citations are embedded in the text or metadata.
	// We return empty here unless you specifically configure response metadata parsing.
	return []string{}
}
