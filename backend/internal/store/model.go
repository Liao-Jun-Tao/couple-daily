package store

import "time"

// User is a login account. PasswordHash is empty unless loaded for login.
type User struct {
	ID           string
	Email        string
	PasswordHash string
	DisplayName  string
	CreatedAt    time.Time
}

// Membership is the caller's couple space, if any.
type Membership struct {
	SpaceID     string
	MemberCount int
	CreatedAt   time.Time
}

// Member is one person in a space.
type Member struct {
	UserID      string
	DisplayName string
	JoinedAt    time.Time
}

// Space is a couple space and its members.
type Space struct {
	ID          string
	MemberCount int
	CreatedAt   time.Time
	Members     []Member
}

// Geo is an optional post location.
type Geo struct {
	Lat        float64
	Lng        float64
	Name       *string
	PlaceID    *string
	Visibility string
}

// PostMedia is an image attached to a post.
type PostMedia struct {
	ID          string
	ContentType string
	StorageKey  string
	SortOrder   int
}

// Post is a timeline item.
type Post struct {
	ID           string
	SpaceID      string
	Body         string
	Author       User
	Media        []PostMedia
	Location     *Geo
	LikeCount    int
	LikedByMe    bool
	CommentCount int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Comment is a post comment.
type Comment struct {
	ID        string
	PostID    string
	Author    User
	Body      string
	CreatedAt time.Time
}

// Checkin is a restaurant check-in.
type Checkin struct {
	ID          string
	SpaceID     string
	Creator     User
	Name        string
	Address     *string
	Lat         *float64
	Lng         *float64
	PlaceID     *string
	CheckedInAt time.Time
	CreatedAt   time.Time
}

// Favorite is a saved restaurant.
type Favorite struct {
	ID        string
	SpaceID   string
	Creator   User
	Name      string
	Address   *string
	Lat       *float64
	Lng       *float64
	PlaceID   *string
	Note      *string
	CreatedAt time.Time
}

// Anniversary is a shared date.
type Anniversary struct {
	ID               string
	SpaceID          string
	Creator          User
	CreatorID        string
	Title            string
	EventDate        string
	Recurrence       string
	Remind           bool
	RemindDaysBefore int
	RemindLocalTime  string
	Timezone         string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// Device is a registered push token.
type Device struct {
	ID        string
	Platform  string
	Token     string
	UpdatedAt time.Time
}

// ReminderRow is an anniversary the worker may notify.
type ReminderRow struct {
	ID         string
	SpaceID    string
	Title      string
	EventDate  string
	Recurrence string
	DaysBefore int
	LocalTime  string
	Timezone   string
}

// DeviceRef is a token addressed by the reminder worker.
type DeviceRef struct {
	Platform string
	Token    string
}
