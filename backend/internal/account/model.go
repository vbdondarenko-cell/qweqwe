package account

import "time"

type User struct {
	ID                string    `json:"id"`
	Email             string    `json:"email"`
	Username          string    `json:"username"`
	DisplayName       string    `json:"displayName"`
	AvatarURL         *string   `json:"avatarUrl,omitempty"`
	ProfileVisibility string    `json:"profileVisibility"`
	Language          string    `json:"language"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type Registration struct {
	Email       string
	Username    string
	DisplayName string
	Password    string
	Language    string
	DeviceLabel string
}

type Login struct {
	Identifier  string
	Password    string
	DeviceLabel string
}

type Session struct {
	ID          string
	UserID      string
	TokenHash   []byte
	CreatedAt   time.Time
	ExpiresAt   time.Time
	DeviceLabel string
}

type UserWithPassword struct {
	User
	PasswordHash string
}

type ProfilePatch struct {
	DisplayName       *string
	AvatarURL         *string
	ProfileVisibility *string
	Language          *string
}

type AuthResult struct {
	User      User      `json:"user"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
}
