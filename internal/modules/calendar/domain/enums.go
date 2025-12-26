package domain

// EventType represents the type of calendar event
type EventType string

const (
	EventTypeBooking     EventType = "booking"
	EventTypeShowing     EventType = "showing"
	EventTypeMaintenance EventType = "maintenance"
	EventTypeBlock       EventType = "block"
	EventTypeOpenHouse   EventType = "open_house"
)

// EventStatus represents the current status of an event
type EventStatus string

const (
	EventStatusPending    EventStatus = "pending"     // Awaiting confirmation
	EventStatusConfirmed  EventStatus = "confirmed"   // Confirmed
	EventStatusInProgress EventStatus = "in_progress" // Currently happening
	EventStatusCompleted  EventStatus = "completed"   // Finished
	EventStatusCancelled  EventStatus = "cancelled"   // Cancelled by user/owner
	EventStatusNoShow     EventStatus = "no_show"     // Guest didn't show up
)

// MaintenanceType represents the category of maintenance
type MaintenanceType string

const (
	MaintenanceRoutine    MaintenanceType = "routine"    // Regular scheduled maintenance
	MaintenanceEmergency  MaintenanceType = "emergency"  // Urgent repairs
	MaintenanceRepair     MaintenanceType = "repair"     // Specific repairs
	MaintenanceRenovation MaintenanceType = "renovation" // Major work
)
