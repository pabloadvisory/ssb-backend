package fcm

import "testing"

func TestDecodeFailureUsesStructuredFCMErrorCode(t *testing.T) {
	t.Parallel()
	failure := decodeFailure([]byte(`{
		"error": {
			"status": "NOT_FOUND",
			"message": "Requested entity was not found.",
			"details": [{
				"@type": "type.googleapis.com/google.firebase.fcm.v1.FcmError",
				"errorCode": "UNREGISTERED"
			}]
		}
	}`))
	if failure.Error.Status != "NOT_FOUND" || len(failure.Error.Details) != 1 || failure.Error.Details[0].ErrorCode != "UNREGISTERED" {
		t.Fatalf("unexpected decoded failure: %+v", failure)
	}
	if !failure.invalidEndpoint() {
		t.Fatal("structured UNREGISTERED detail should invalidate the endpoint")
	}
}

func TestDecodeFailureDoesNotScanMessageText(t *testing.T) {
	t.Parallel()
	failure := decodeFailure([]byte(`{"error":{"status":"INVALID_ARGUMENT","message":"mentions UNREGISTERED but has no detail"}}`))
	if len(failure.Error.Details) != 0 {
		t.Fatalf("expected no structured error detail: %+v", failure)
	}
	if failure.invalidEndpoint() {
		t.Fatal("message text alone must not invalidate the endpoint")
	}
}
