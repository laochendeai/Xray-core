package webpanel

import "testing"

func TestAuthManagerSuccessfulLoginsDoNotConsumeRateLimit(t *testing.T) {
	t.Parallel()

	auth := NewAuthManager("devadmin", "secret", "jwt-secret")

	for i := 0; i < 6; i++ {
		if _, err := auth.Login("devadmin", "secret", "127.0.0.1"); err != nil {
			t.Fatalf("successful login %d returned error: %v", i+1, err)
		}
	}
}

func TestAuthManagerRateLimitsRepeatedFailedLogins(t *testing.T) {
	t.Parallel()

	auth := NewAuthManager("devadmin", "secret", "jwt-secret")

	for i := 0; i < 5; i++ {
		if _, err := auth.Login("devadmin", "wrong", "127.0.0.1"); err == nil {
			t.Fatalf("failed login %d unexpectedly succeeded", i+1)
		}
	}

	if _, err := auth.Login("devadmin", "wrong", "127.0.0.1"); err == nil || err.Error() != "too many login attempts, please try again later" {
		t.Fatalf("expected rate limit error on 6th failed login, got %v", err)
	}
}
