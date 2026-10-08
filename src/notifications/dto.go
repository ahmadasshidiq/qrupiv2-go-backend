package notifications

import "clasenna-go-backend/libs/models"

type AnnouncementDTO struct {
	Audience models.AnnouncementAudience `json:"audience"`
	TargetID *string                     `json:"target_id"`
	Title    string                      `json:"title"`
	Message  string                      `json:"message"`
}

type DeviceDTO struct {
	Token    string `json:"token" binding:"required"`
	Platform string `json:"platform"`
}

type EventDataDTO struct {
	RecipientIDs []string       `json:"recipient_ids"`
	Title        string         `json:"title"`
	Message      string         `json:"message"`
	EntityID     string         `json:"entity_id"`
	Deeplink     string         `json:"deeplink"`
	WebURL       string         `json:"web_url"`
	MobileRoute  string         `json:"mobile_route"`
	Data         map[string]any `json:"data"`
}
