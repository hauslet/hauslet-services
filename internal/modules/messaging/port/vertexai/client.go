package vertexai

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	discoveryengine "cloud.google.com/go/discoveryengine/apiv1"
	discoveryenginepb "cloud.google.com/go/discoveryengine/apiv1/discoveryenginepb"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// VertexAIClient wraps the Google Cloud Discovery Engine Client
type VertexAIClient struct {
	client      *discoveryengine.ConversationalSearchClient
	projectID   string
	location    string
	dataStoreID string // This is the "Agent ID" or "Engine ID"
}

// SendMessageRequest describes the data needed to interact with Vertex AI conversations.
type SendMessageRequest struct {
	SessionID        string
	Query            string
	History          []HistoryEntry
	UserContext      *UserContext
	ContextDocuments []string
	ActiveDocument   string
	Metadata         map[string]string
	MaxHistory       int
	ServingConfig    string
}

// HistoryEntry captures a single message that should be sent as part of the conversation context.
type HistoryEntry struct {
	Role      HistoryRole
	Text      string
	CreatedAt time.Time
}

// HistoryRole defines whether the entry was authored by a user or the bot.
type HistoryRole string

const (
	HistoryRoleUser HistoryRole = "user"
	HistoryRoleAI   HistoryRole = "ai"
)

// UserContext provides labels that Vertex AI can use to personalize or filter responses.
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

// NewVertexAIClient creates a new client
func NewVertexAIClient(ctx context.Context, projectID, location, dataStoreID, credentialsPath string) (*VertexAIClient, error) {
	opts := []option.ClientOption{}
	if credentialsPath != "" {
		opts = append(opts, option.WithCredentialsFile(credentialsPath))
	}

	// Initialize the conversational search client
	client, err := discoveryengine.NewConversationalSearchClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery engine client: %w", err)
	}

	return &VertexAIClient{
		client:      client,
		projectID:   projectID,
		location:    location,
		dataStoreID: dataStoreID,
	}, nil
}

// Close closes the underlying client connection
func (c *VertexAIClient) Close() error {
	return c.client.Close()
}

// SendMessage sends a user query to the agent within a specific session.
// sessionID should be unique per user-conversation context.
func (c *VertexAIClient) SendMessage(ctx context.Context, req *SendMessageRequest) (*AIResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("vertex ai send request is required")
	}
	if strings.TrimSpace(req.SessionID) == "" {
		return nil, fmt.Errorf("session id is required")
	}
	if strings.TrimSpace(req.Query) == "" {
		return nil, fmt.Errorf("query text is required")
	}

	maxHistory := req.MaxHistory
	if maxHistory <= 0 {
		maxHistory = 10
	}

	history := req.History
	if len(history) > maxHistory {
		history = history[len(history)-maxHistory:]
	}

	name := fmt.Sprintf("projects/%s/locations/%s/collections/default_collection/engines/%s/sessions/%s",
		c.projectID, c.location, c.dataStoreID, req.SessionID)

	conversation := &discoveryenginepb.Conversation{
		UserPseudoId: req.UserContextValue(),
		Messages:     buildConversationMessages(history),
	}

	conversationContext := &discoveryenginepb.ConversationContext{
		ContextDocuments: append([]string{}, req.ContextDocuments...),
		ActiveDocument:   req.ActiveDocument,
	}

	request := &discoveryenginepb.ConverseConversationRequest{
		Name: name,
		Query: &discoveryenginepb.TextInput{
			Input:   strings.TrimSpace(req.Query),
			Context: conversationContext,
		},
		Conversation: conversation,
		UserLabels:   buildUserLabels(req.UserContext, req.Metadata),
	}
	if req.ServingConfig != "" {
		request.ServingConfig = req.ServingConfig
	}

	resp, err := c.client.ConverseConversation(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("vertex ai conversation failed: %w", err)
	}

	result := &AIResponse{
		Text:               extractSummaryText(resp),
		Confidence:         extractConfidence(resp),
		Sources:            extractSources(resp),
		RequiresEscalation: hasEscalationHints(resp),
	}

	if req.UserContext != nil {
		result.Intent = req.UserContext.Metadata["intent"]
	}

	if result.Text == "" {
		result.Text = "I apologize, but I couldn't find relevant information to answer your question."
	}

	return result, nil
}

func (r *SendMessageRequest) UserContextValue() string {
	if r == nil || r.UserContext == nil {
		return ""
	}
	return r.UserContext.UserID
}

func buildConversationMessages(entries []HistoryEntry) []*discoveryenginepb.ConversationMessage {
	out := make([]*discoveryenginepb.ConversationMessage, 0, len(entries))
	for _, entry := range entries {
		if strings.TrimSpace(entry.Text) == "" {
			continue
		}
		msg := &discoveryenginepb.ConversationMessage{}
		if !entry.CreatedAt.IsZero() {
			msg.CreateTime = timestamppb.New(entry.CreatedAt)
		}

		if entry.Role == HistoryRoleAI {
			msg.Message = &discoveryenginepb.ConversationMessage_Reply{
				Reply: &discoveryenginepb.Reply{
					Summary: &discoveryenginepb.SearchResponse_Summary{
						SummaryText: entry.Text,
					},
				},
			}
		} else {
			msg.Message = &discoveryenginepb.ConversationMessage_UserInput{
				UserInput: &discoveryenginepb.TextInput{
					Input: entry.Text,
				},
			}
		}

		out = append(out, msg)
	}
	return out
}

func buildUserLabels(ctx *UserContext, extra map[string]string) map[string]string {
	labels := make(map[string]string)
	if ctx != nil {
		if ctx.UserID != "" {
			labels["hauslet_user_id"] = ctx.UserID
		}
		if ctx.DisplayName != "" {
			labels["hauslet_user_name"] = ctx.DisplayName
		}
		for k, v := range ctx.Metadata {
			if k == "" || v == "" {
				continue
			}
			labels[sanitizeLabelKey("hauslet_"+k)] = v
		}
	}
	for k, v := range extra {
		if k == "" || v == "" {
			continue
		}
		labels[sanitizeLabelKey("hauslet_"+k)] = v
	}
	if len(labels) == 0 {
		return nil
	}
	return labels
}

func sanitizeLabelKey(key string) string {
	key = strings.ToLower(strings.TrimSpace(key))
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_', r == '-':
			return r
		default:
			return '_'
		}
	}, key)
}

func extractSummaryText(resp *discoveryenginepb.ConverseConversationResponse) string {
	if resp == nil {
		return ""
	}
	if resp.Reply != nil && resp.Reply.Summary != nil && resp.Reply.Summary.SummaryText != "" {
		return resp.Reply.Summary.SummaryText
	}
	if resp.Reply != nil && resp.Reply.Summary != nil && resp.Reply.Summary.SummaryWithMetadata != nil {
		if resp.Reply.Summary.SummaryWithMetadata.Summary != "" {
			return resp.Reply.Summary.SummaryWithMetadata.Summary
		}
	}
	if resp.Conversation != nil {
		for _, msg := range resp.Conversation.Messages {
			if reply := msg.GetReply(); reply != nil && reply.Summary != nil && reply.Summary.SummaryText != "" {
				return reply.Summary.SummaryText
			}
		}
	}
	if len(resp.SearchResults) > 0 {
		if doc := resp.SearchResults[0].GetDocument(); doc != nil && doc.GetDerivedStructData() != nil {
			if title, ok := doc.GetDerivedStructData().Fields["title"]; ok {
				text := title.GetStringValue()
				if text != "" {
					return text
				}
			}
		}
	}
	return ""
}

func extractSources(resp *discoveryenginepb.ConverseConversationResponse) []string {
	var sources []string
	if resp == nil {
		return sources
	}
	for _, result := range resp.SearchResults {
		if result.Document != nil {
			if result.Document.DerivedStructData != nil {
				if title, ok := result.Document.DerivedStructData.Fields["title"]; ok {
					sources = append(sources, title.GetStringValue())
					continue
				}
			}
			if result.Document.Name != "" {
				sources = append(sources, result.Document.Name)
			}
		}
	}
	return sources
}

func extractConfidence(resp *discoveryenginepb.ConverseConversationResponse) float32 {
	if resp == nil {
		return 0
	}
	var best float32
	for _, result := range resp.SearchResults {
		for key, list := range result.ModelScores {
			if len(list.GetValues()) == 0 {
				continue
			}
			key = strings.ToLower(key)
			if !strings.Contains(key, "confidence") && !strings.Contains(key, "score") {
				continue
			}
			for _, raw := range list.GetValues() {
				best = float32(math.Max(float64(best), raw))
			}
		}
		if signals := result.GetRankSignals(); signals != nil {
			best = float32(math.Max(float64(best), float64(signals.GetRelevanceScore())))
		}
	}
	if best == 0 {
		return 1.0
	}
	if best > 1 {
		return 1
	}
	return best
}

func hasEscalationHints(resp *discoveryenginepb.ConverseConversationResponse) bool {
	if resp == nil || resp.Reply == nil || resp.Reply.Summary == nil {
		return false
	}
	return len(resp.Reply.Summary.SummarySkippedReasons) > 0
}
