package validator

import (
	"testing"

	"github.com/beyzacanbay/notification-service/internal/dto"
	"github.com/beyzacanbay/notification-service/internal/model"
)

func TestValidate_SMSSuccess(t *testing.T) {
	req := &dto.CreateNotificationRequest{
		Channel:   model.ChannelSMS,
		Recipient: "+905551234567",
		Content:   "Hello",
	}
	errs := ValidateCreateRequest(req)
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

func TestValidate_SMSInvalidPhone(t *testing.T) {
	req := &dto.CreateNotificationRequest{
		Channel:   model.ChannelSMS,
		Recipient: "invalid",
		Content:   "Hello",
	}
	errs := ValidateCreateRequest(req)
	if len(errs) == 0 {
		t.Fatal("expected validation error for invalid phone")
	}
}

func TestValidate_SMSContentTooLong(t *testing.T) {
	long := make([]byte, 1601)
	for i := range long {
		long[i] = 'a'
	}
	req := &dto.CreateNotificationRequest{
		Channel:   model.ChannelSMS,
		Recipient: "+905551234567",
		Content:   string(long),
	}
	errs := ValidateCreateRequest(req)
	found := false
	for _, e := range errs {
		if e.Field == "content" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected content validation error for SMS")
	}
}

func TestValidate_EmailInvalid(t *testing.T) {
	req := &dto.CreateNotificationRequest{
		Channel:   model.ChannelEmail,
		Recipient: "not-an-email",
		Content:   "Hello",
	}
	errs := ValidateCreateRequest(req)
	if len(errs) == 0 {
		t.Fatal("expected validation error for invalid email")
	}
}

func TestValidate_PushContentTooLong(t *testing.T) {
	long := make([]byte, 4097)
	for i := range long {
		long[i] = 'a'
	}
	req := &dto.CreateNotificationRequest{
		Channel:   model.ChannelPush,
		Recipient: "device-token-abc123",
		Content:   string(long),
	}
	errs := ValidateCreateRequest(req)
	found := false
	for _, e := range errs {
		if e.Field == "content" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected content validation error for push")
	}
}

func TestValidate_MissingFields(t *testing.T) {
	req := &dto.CreateNotificationRequest{}
	errs := ValidateCreateRequest(req)
	if len(errs) < 3 {
		t.Fatalf("expected at least 3 errors (channel, recipient, content), got %d", len(errs))
	}
}
