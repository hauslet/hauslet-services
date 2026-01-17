package service

import (
	"context"
	"fmt"
	"path"
	"strings"
	"sync"
	"time"

	"hauslet/config"
	"hauslet/internal/modules/messaging/domain"
	"hauslet/internal/modules/messaging/notification"
	"hauslet/internal/modules/messaging/repository"
	"hauslet/internal/platform/events"
	"hauslet/internal/platform/storage"

	"log/slog"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type messagingServiceImpl struct {
	db              *gorm.DB // Added for transactions
	convRepo        repository.ConversationRepository
	messageRepo     repository.MessageRepository
	participantRepo repository.ParticipantRepository
	leadHooks       domain.LeadHooks
	bookingHooks    domain.BookingHooks
	profileHooks    domain.ProfileHooks
	aiSupport       AISupportService
	eventPublisher  *events.Publisher
	eventSubscriber *events.Subscriber // For subscribing to lead events
	storage         *storage.R2Storage
	notificationSvc *notification.NotificationService
	platformConfig  *config.PlatformYAMLConfig
	log             *slog.Logger
	presence        *presenceTracker
}

// NewMessagingService wires the messaging dependencies into a concrete implementation.
func NewMessagingService(
	db *gorm.DB,
	convRepo repository.ConversationRepository,
	messageRepo repository.MessageRepository,
	participantRepo repository.ParticipantRepository,
	aiSupport AISupportService,
	leadHooks domain.LeadHooks,
	bookingHooks domain.BookingHooks,
	profileHooks domain.ProfileHooks,
	eventPublisher *events.Publisher,
	eventSubscriber *events.Subscriber,
	storage *storage.R2Storage,
	notificationSvc *notification.NotificationService,
	platformConfig *config.PlatformYAMLConfig,
	log *slog.Logger,
) MessagingService {
	return &messagingServiceImpl{
		db:              db,
		convRepo:        convRepo,
		messageRepo:     messageRepo,
		participantRepo: participantRepo,
		leadHooks:       leadHooks,
		bookingHooks:    bookingHooks,
		profileHooks:    profileHooks,
		aiSupport:       aiSupport,
		eventPublisher:  eventPublisher,
		eventSubscriber: eventSubscriber,
		storage:         storage,
		notificationSvc: notificationSvc,
		platformConfig:  platformConfig,
		log:             log,
		presence:        newPresenceTracker(),
	}
}

func (s *messagingServiceImpl) GenerateUploadURL(ctx context.Context, input AttachmentUploadRequest) (*AttachmentUploadResult, error) {
	if s.storage == nil {
		return nil, fmt.Errorf("messaging storage not configured")
	}
	if input.ConversationID == uuid.Nil {
		return nil, fmt.Errorf("conversation_id is required")
	}
	if input.AttachmentID == uuid.Nil {
		return nil, fmt.Errorf("attachment_id is required")
	}

	filename := strings.TrimSpace(path.Base(input.Filename))
	if filename == "" || filename == "." {
		return nil, fmt.Errorf("filename is required")
	}

	contentType := strings.TrimSpace(input.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	expires := input.ExpiresIn
	if expires <= 0 {
		expires = 15 * time.Minute
	}

	key := fmt.Sprintf("messaging/%s/attachments/%s/%s", input.ConversationID.String(), input.AttachmentID.String(), filename)
	url, err := s.storage.GenerateSignedUploadURL(ctx, key, contentType, expires)
	if err != nil {
		return nil, fmt.Errorf("failed to generate attachment upload url: %w", err)
	}

	return &AttachmentUploadResult{
		URL: url,
		Key: key,
	}, nil
}

func (s *messagingServiceImpl) MarkParticipantActive(conversationID, userID uuid.UUID) {
	if s.presence == nil || conversationID == uuid.Nil || userID == uuid.Nil {
		return
	}
	s.presence.markPresent(conversationID, userID)
}

func (s *messagingServiceImpl) MarkParticipantInactive(conversationID, userID uuid.UUID) {
	if s.presence == nil || conversationID == uuid.Nil || userID == uuid.Nil {
		return
	}
	s.presence.markAbsent(conversationID, userID)
}

func (s *messagingServiceImpl) IsParticipantPresent(conversationID, userID uuid.UUID) bool {
	if s.presence == nil || conversationID == uuid.Nil || userID == uuid.Nil {
		return false
	}
	return s.presence.isPresent(conversationID, userID)
}

type presenceTracker struct {
	mu       sync.RWMutex
	entries  map[uuid.UUID]map[uuid.UUID]time.Time
	entryTTL time.Duration
}

const participantPresenceTTL = 2 * time.Minute

func newPresenceTracker() *presenceTracker {
	return &presenceTracker{
		entries:  make(map[uuid.UUID]map[uuid.UUID]time.Time),
		entryTTL: participantPresenceTTL,
	}
}

func (p *presenceTracker) markPresent(conversationID, userID uuid.UUID) {
	now := time.Now()

	p.mu.Lock()
	defer p.mu.Unlock()

	users, ok := p.entries[conversationID]
	if !ok {
		users = make(map[uuid.UUID]time.Time)
		p.entries[conversationID] = users
	}
	users[userID] = now
}

func (p *presenceTracker) markAbsent(conversationID, userID uuid.UUID) {
	p.mu.Lock()
	defer p.mu.Unlock()

	users, ok := p.entries[conversationID]
	if !ok {
		return
	}
	delete(users, userID)
	if len(users) == 0 {
		delete(p.entries, conversationID)
	}
}

func (p *presenceTracker) isPresent(conversationID, userID uuid.UUID) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	users, ok := p.entries[conversationID]
	if !ok {
		return false
	}

	lastSeen, ok := users[userID]
	if !ok {
		return false
	}

	if time.Since(lastSeen) > p.entryTTL {
		delete(users, userID)
		if len(users) == 0 {
			delete(p.entries, conversationID)
		}
		return false
	}

	return true
}
