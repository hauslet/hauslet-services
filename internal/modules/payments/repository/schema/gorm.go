package schema

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// Payment represents the database model for payments
type Payment struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`

	// Reference identifiers
	Reference    string       `gorm:"type:varchar(255);uniqueIndex;not null"`
	ProviderRef  string       `gorm:"type:varchar(255);index"`
	BookingID    *uuid.UUID   `gorm:"type:uuid;index"`
	BusinessID   *uuid.UUID   `gorm:"type:uuid;index"`
	ResourceType ResourceType `gorm:"type:varchar(50);not null;default:'general'"`
	ResourceID   *uuid.UUID   `gorm:"type:uuid;index"`

	// Payer information
	PayerID    uuid.UUID `gorm:"type:uuid;not null;index"`
	PayerEmail string    `gorm:"type:varchar(255);not null"`
	PayerName  string    `gorm:"type:varchar(255);not null"`

	// Payment details
	Amount        int64             `gorm:"not null"`
	Currency      string            `gorm:"type:varchar(10);not null"`
	Market        Market            `gorm:"type:varchar(10);not null;default:'OTHER'"`
	Status        PaymentStatus     `gorm:"type:varchar(20);not null;default:'pending';index"`
	PaymentMethod PaymentMethodType `gorm:"type:varchar(20)"`

	// Provider details
	Provider          string  `gorm:"type:varchar(50);not null;default:'paystack'"`
	AuthorizationCode *string `gorm:"type:varchar(255)"`
	RedirectURL       *string `gorm:"type:text"`
	RequiresAction    bool    `gorm:"default:false"`

	// Metadata
	Description string         `gorm:"type:text"`
	Metadata    pq.StringArray `gorm:"type:text[]"` // Stored as key=value pairs

	// Refund tracking
	RefundedAmount int64 `gorm:"default:0"`
	RefundedAt     *time.Time

	// Audit fields
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// TableName specifies the table name for Payment
func (Payment) TableName() string {
	return "payments"
}

// Transaction represents the database model for transactions
type Transaction struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`

	// Related entities
	PaymentID  *uuid.UUID `gorm:"type:uuid;index"`
	BookingID  *uuid.UUID `gorm:"type:uuid;index"`
	BusinessID *uuid.UUID `gorm:"type:uuid;index"`

	// Transaction details
	Type         TransactionType   `gorm:"type:varchar(20);not null;index"`
	Reference    string            `gorm:"type:varchar(255);uniqueIndex;not null"`
	Amount       int64             `gorm:"not null"`
	Currency     string            `gorm:"type:varchar(10);not null"`
	Status       TransactionStatus `gorm:"type:varchar(20);not null;default:'pending';index"`
	Provider     string            `gorm:"type:varchar(50);not null"`
	ProviderTxID *string           `gorm:"type:varchar(255);index"`

	// Context
	Description string         `gorm:"type:text"`
	Metadata    pq.StringArray `gorm:"type:text[]"`

	// Error tracking
	ErrorMessage *string `gorm:"type:text"`
	ErrorCode    *string `gorm:"type:varchar(50)"`

	// Audit fields
	ProcessedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// TableName specifies the table name for Transaction
func (Transaction) TableName() string {
	return "transactions"
}

// PaymentMethod represents the database model for saved payment methods
type PaymentMethod struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`

	// Owner
	UserID uuid.UUID `gorm:"type:uuid;not null;index"`

	// Payment method details
	Type              PaymentMethodType `gorm:"type:varchar(20);not null"`
	Provider          string            `gorm:"type:varchar(50);not null;default:'paystack'"`
	AuthorizationCode string            `gorm:"type:varchar(255);not null;uniqueIndex"`
	Currency          string            `gorm:"type:varchar(10);not null"`

	// Card details (masked)
	Last4Digits *string `gorm:"type:varchar(4)"`
	CardType    *string `gorm:"type:varchar(50)"`
	Brand       *string `gorm:"type:varchar(50)"`
	ExpiryMonth *int
	ExpiryYear  *int
	BankName    *string `gorm:"type:varchar(255)"`

	// Status
	IsDefault bool `gorm:"default:false"`
	IsActive  bool `gorm:"default:true;index"`

	// Metadata
	CustomerCode *string `gorm:"type:varchar(255)"`

	// Audit fields
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// TableName specifies the table name for PaymentMethod
func (PaymentMethod) TableName() string {
	return "payment_methods"
}

// PayoutDetail represents the database model for payout details
type PayoutDetail struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`

	// Owner (user or business)
	UserID     *uuid.UUID `gorm:"type:uuid;index"`
	BusinessID *uuid.UUID `gorm:"type:uuid;index"`

	// Bank account details
	BankCode      string `gorm:"type:varchar(50);not null"`
	BankName      string `gorm:"type:varchar(255);not null"`
	AccountNumber string `gorm:"type:varchar(50);not null"`
	AccountName   string `gorm:"type:varchar(255);not null"`
	Currency      string `gorm:"type:varchar(10);not null"`
	Market        Market `gorm:"type:varchar(10);not null"`

	// Provider details
	Provider      string `gorm:"type:varchar(50);not null;default:'paystack'"`
	RecipientCode string `gorm:"type:varchar(255);not null;uniqueIndex"`
	IsVerified    bool   `gorm:"default:false"`
	VerifiedAt    *time.Time

	// Status
	IsDefault bool `gorm:"default:false"`
	IsActive  bool `gorm:"default:true;index"`

	// Metadata
	Metadata pq.StringArray `gorm:"type:text[]"`

	// Audit fields
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// TableName specifies the table name for PayoutDetail
func (PayoutDetail) TableName() string {
	return "payout_details"
}
