package validator

import (
	"fmt"
	"net/mail"
	"regexp"

	"github.com/beyzacanbay/notification-service/internal/dto"
	"github.com/beyzacanbay/notification-service/internal/model"
)

var phoneRegex = regexp.MustCompile(`^\+[1-9]\d{6,14}$`)

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	return fmt.Sprintf("%d validation error(s): %s", len(e), e[0].Error())
}

func ValidateCreateRequest(req *dto.CreateNotificationRequest) ValidationErrors {
	var errs ValidationErrors

	// Channel
	if req.Channel == "" {
		errs = append(errs, ValidationError{Field: "channel", Message: "is required"})
	} else if !req.Channel.IsValid() {
		errs = append(errs, ValidationError{Field: "channel", Message: "must be one of: sms, email, push"})
	}

	// Recipient — channel-specific format
	if req.Recipient == "" {
		errs = append(errs, ValidationError{Field: "recipient", Message: "is required"})
	} else {
		switch req.Channel {
		case model.ChannelSMS:
			if !phoneRegex.MatchString(req.Recipient) {
				errs = append(errs, ValidationError{Field: "recipient", Message: "must be a valid E.164 phone number (e.g., +905551234567)"})
			}
		case model.ChannelEmail:
			if _, err := mail.ParseAddress(req.Recipient); err != nil {
				errs = append(errs, ValidationError{Field: "recipient", Message: "must be a valid email address"})
			}
		case model.ChannelPush:
			if len(req.Recipient) < 10 {
				errs = append(errs, ValidationError{Field: "recipient", Message: "device token is too short"})
			}
		}
	}

	// Content — required
	if req.Content == "" {
		errs = append(errs, ValidationError{Field: "content", Message: "is required"})
	}

	// Content — channel-specific character limits
	switch req.Channel {
	case model.ChannelSMS:
		if len(req.Content) > 1600 {
			errs = append(errs, ValidationError{Field: "content", Message: "SMS content must not exceed 1600 characters"})
		}
	case model.ChannelEmail:
		if len(req.Content) > 102400 {
			errs = append(errs, ValidationError{Field: "content", Message: "email content must not exceed 100KB"})
		}
	case model.ChannelPush:
		if len(req.Content) > 4096 {
			errs = append(errs, ValidationError{Field: "content", Message: "push content must not exceed 4096 characters"})
		}
	}

	// Priority
	if req.Priority != nil && !req.Priority.IsValid() {
		errs = append(errs, ValidationError{Field: "priority", Message: "must be 0 (high), 1 (normal), or 2 (low)"})
	}

	return errs
}
