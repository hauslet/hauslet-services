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
