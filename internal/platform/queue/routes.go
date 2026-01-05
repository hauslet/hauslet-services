package queue

import "time"

type QueueRoute struct {
	Name    string
	Path    string
	Timeout time.Duration
}

var defaultQueueRoutes = map[string]QueueRoute{
	"email": {
		Path:    "/tasks/email",
		Timeout: 30 * time.Second,
	},
	"media_thumbnail": {
		Path:    "/tasks/media/thumbnail",
		Timeout: 180 * time.Second,
	},
	"media_cleanup": {
		Path:    "/tasks/media/cleanup",
		Timeout: 60 * time.Second,
	},
	"ai_moderation": {
		Path:    "/tasks/ai/moderation",
		Timeout: 300 * time.Second,
	},
	"booking_expiry": {
		Path:    "/tasks/booking/expiry",
		Timeout: 60 * time.Second,
	},
	"booking_completion": {
		Path:    "/tasks/booking/completion",
		Timeout: 60 * time.Second,
	},
	"booking_checkin_out": {
		Path:    "/tasks/booking/checkin-out",
		Timeout: 60 * time.Second,
	},
	"booking_refund": {
		Path:    "/tasks/booking/refund",
		Timeout: 60 * time.Second,
	},
	"payment_webhook": {
		Path:    "/tasks/payment/webhook",
		Timeout: 60 * time.Second,
	},
	"payout_process": {
		Path:    "/tasks/finance/payout/process",
		Timeout: 120 * time.Second,
	},
	"payout_retry": {
		Path:    "/tasks/finance/payout/retry",
		Timeout: 120 * time.Second,
	},
	"finance_reconciliation": {
		Path:    "/tasks/finance/reconciliation",
		Timeout: 300 * time.Second,
	},
	"review_stats": {
		Path:    "/tasks/review/stats",
		Timeout: 120 * time.Second,
	},
	"review_standoff": {
		Path:    "/tasks/review/standoff/publish",
		Timeout: 120 * time.Second,
	},
	"review_reminders": {
		Path:    "/tasks/review/reminders",
		Timeout: 120 * time.Second,
	},
	"promotion_expiry": {
		Path:    "/tasks/promotion/expiry",
		Timeout: 120 * time.Second,
	},
	"subscription_billing": {
		Path:    "/tasks/promotion/billing",
		Timeout: 120 * time.Second,
	},
	"calendar_showing_reminders": {
		Path:    "/tasks/calendar/showing/reminders",
		Timeout: 120 * time.Second,
	},
	"calendar_open_house_reminders": {
		Path:    "/tasks/calendar/open-house/reminders",
		Timeout: 120 * time.Second,
	},
	"verification_submission": {
		Path:    "/tasks/verification/submission",
		Timeout: 180 * time.Second,
	},
	"verification_sms": {
		Path:    "/tasks/verification/sms",
		Timeout: 30 * time.Second,
	},
	"verification_reconciliation": {
		Path:    "/tasks/verification/reconciliation",
		Timeout: 300 * time.Second,
	},
	"interactions_batch": {
		Path:    "/tasks/interactions/batch",
		Timeout: 60 * time.Second,
	},
}

func BuildQueueRoutes(queueNames map[string]string) map[string]QueueRoute {
	routes := make(map[string]QueueRoute)
	for key, queueName := range queueNames {
		if queueName == "" {
			continue
		}
		route, ok := defaultQueueRoutes[key]
		if !ok {
			continue
		}
		route.Name = queueName
		routes[queueName] = route
	}
	return routes
}

func QueueRouteList(queueNames map[string]string) []QueueRoute {
	routes := BuildQueueRoutes(queueNames)
	list := make([]QueueRoute, 0, len(routes))
	for _, route := range routes {
		list = append(list, route)
	}
	return list
}
