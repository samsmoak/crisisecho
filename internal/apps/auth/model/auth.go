package model

// AuthProvider enumerates supported authentication methods.
type AuthProvider string

const (
	ProviderGoogle AuthProvider = "google"
	ProviderApple  AuthProvider = "apple"
	ProviderPhone  AuthProvider = "phone"
)

// GoogleAuthRequest is the body for POST /api/auth/google.
type GoogleAuthRequest struct {
	IDToken string `json:"id_token"`
}

// AppleAuthRequest is the body for POST /api/auth/apple (placeholder).
type AppleAuthRequest struct {
	IDToken string `json:"id_token"`
}

// PhoneOTPRequest is the body for POST /api/auth/phone/send-otp (placeholder).
type PhoneOTPRequest struct {
	Phone string `json:"phone"`
}

// PhoneVerifyRequest is the body for POST /api/auth/phone/verify (placeholder).
type PhoneVerifyRequest struct {
	Phone   string `json:"phone"`
	OTPCode string `json:"otp_code"`
}

// AuthResponse is the standard response returned after successful authentication.
type AuthResponse struct {
	Token string      `json:"token"`
	User  interface{} `json:"user"`
}
