package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	NotificationCreated = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "notification_created_total",
			Help: "Total notifications created",
		},
		[]string{"channel", "priority"},
	)

	NotificationDelivered = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "notification_delivered_total",
			Help: "Total notifications successfully delivered",
		},
		[]string{"channel"},
	)

	NotificationFailed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "notification_failed_total",
			Help: "Total notifications that failed delivery",
		},
		[]string{"channel", "reason"},
	)

	NotificationRetried = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "notification_retried_total",
			Help: "Total notification retry attempts",
		},
		[]string{"channel"},
	)

	NotificationRateLimited = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "notification_rate_limited_total",
			Help: "Total notifications rate limited",
		},
		[]string{"channel"},
	)

	DeliveryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "notification_delivery_duration_seconds",
			Help:    "Time taken to deliver a notification",
			Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"channel"},
	)

	QueueDepth = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "notification_queue_depth",
			Help: "Current number of items in queue",
		},
		[]string{"queue"},
	)
)

func init() {
	prometheus.MustRegister(
		NotificationCreated,
		NotificationDelivered,
		NotificationFailed,
		NotificationRetried,
		NotificationRateLimited,
		DeliveryDuration,
		QueueDepth,
	)
}
