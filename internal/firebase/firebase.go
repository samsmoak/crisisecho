package firebase

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

var (
	client      *auth.Client
	initialized bool
)

// Initialize sets up the Firebase Admin SDK. It is safe to call from main;
// if credentials are missing, Firebase features are gracefully disabled.
func Initialize() error {
	ctx := context.Background()

	appEnv := os.Getenv("APP_ENV")
	isProduction := appEnv == "production" || appEnv == "prod" ||
		os.Getenv("K_SERVICE") != "" || os.Getenv("K_REVISION") != ""

	log.Printf("firebase: initializing — env=%s production=%v", appEnv, isProduction)

	var opt option.ClientOption

	if isProduction {
		credsJSON := os.Getenv("FIREBASE_ADMIN_CREDENTIALS")
		if credsJSON == "" {
			log.Println("firebase: FIREBASE_ADMIN_CREDENTIALS not set — features disabled")
			initialized = false
			return nil
		}
		if !isValidJSON(credsJSON) {
			log.Println("firebase: FIREBASE_ADMIN_CREDENTIALS contains invalid JSON — features disabled")
			initialized = false
			return nil
		}
		opt = option.WithCredentialsJSON([]byte(credsJSON))
	} else {
		credsPath := getDevCredentialsPath()
		if _, err := os.Stat(credsPath); err != nil {
			log.Printf("firebase: credentials file not found at %s — features disabled", credsPath)
			initialized = false
			return nil
		}
		log.Printf("firebase: using credentials file %s", credsPath)
		opt = option.WithCredentialsFile(credsPath)
	}

	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		log.Printf("firebase: failed to create app: %v — features disabled", err)
		initialized = false
		return nil
	}

	client, err = app.Auth(ctx)
	if err != nil {
		log.Printf("firebase: failed to create auth client: %v — features disabled", err)
		initialized = false
		return nil
	}

	initialized = true
	log.Println("firebase: initialized successfully")
	return nil
}

// IsAvailable reports whether the Firebase Auth client is ready.
func IsAvailable() bool {
	return initialized && client != nil
}

// VerifyIDToken validates a Firebase ID token and returns the decoded token.
func VerifyIDToken(ctx context.Context, idToken string) (*auth.Token, error) {
	if !IsAvailable() {
		return nil, fmt.Errorf("firebase is not available — check credentials configuration")
	}
	return client.VerifyIDToken(ctx, idToken)
}

func isValidJSON(s string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(s), &js) == nil
}

func getDevCredentialsPath() string {
	if path := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); path != "" {
		return path
	}
	if path := os.Getenv("FIREBASE_CREDENTIALS_PATH"); path != "" {
		return path
	}
	return "./config/firebase-admin.json"
}
