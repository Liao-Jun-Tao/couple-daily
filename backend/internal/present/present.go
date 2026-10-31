// Package present builds the JSON shapes from the OpenAPI contract.
package present

import (
	"time"

	"github.com/Liao-Jun-Tao/couple-daily/internal/store"
)

// FormatTime renders a UTC RFC3339 timestamp.
func FormatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// User is the public account object.
type User struct {
	ID          string  `json:"id"`
	Email       string  `json:"email"`
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
	CreatedAt   string  `json:"created_at"`
}

// AuthToken is the register, login, and refresh body.
type AuthToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	User         User   `json:"user"`
}

// Pairing is the caller's couple-space status.
type Pairing struct {
	Status      string  `json:"status"`
	SpaceID     *string `json:"space_id"`
	MemberCount *int    `json:"member_count"`
}

// Me is GET /auth/me.
type Me struct {
	User    User    `json:"user"`
	Pairing Pairing `json:"pairing"`
}

// Member is one person in a space.
type Member struct {
	UserID      string  `json:"user_id"`
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
	JoinedAt    string  `json:"joined_at"`
}

// Space is a couple space.
type Space struct {
	ID          string   `json:"id"`
	MemberCount int      `json:"member_count"`
	Members     []Member `json:"members"`
	CreatedAt   string   `json:"created_at"`
}

// Invite is the create-invite response.
type Invite struct {
	InviteCode string `json:"invite_code"`
	ExpiresAt  string `json:"expires_at"`
	Space      Space  `json:"space"`
}

// MediaUpload is a presigned upload intent.
type MediaUpload struct {
	MediaID       string            `json:"media_id"`
	UploadURL     string            `json:"upload_url"`
	UploadMethod  string            `json:"upload_method"`
	UploadHeaders map[string]string `json:"upload_headers"`
	ExpiresAt     string            `json:"expires_at"`
}

// Location is a stored post location.
type Location struct {
	Lat        float64 `json:"lat"`
	Lng        float64 `json:"lng"`
	Name       *string `json:"name"`
	PlaceID    *string `json:"place_id"`
	Visibility string  `json:"visibility"`
}

// MediaRef is an image on a post.
type MediaRef struct {
	ID          string `json:"id"`
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	SortOrder   int    `json:"sort_order"`
}

// Post is a timeline item.
type Post struct {
	ID           string     `json:"id"`
	SpaceID      string     `json:"space_id"`
	Author       User       `json:"author"`
	Body         string     `json:"body"`
	Media        []MediaRef `json:"media"`
	Location     *Location  `json:"location"`
	LikeCount    int        `json:"like_count"`
	LikedByMe    bool       `json:"liked_by_me"`
	CommentCount int        `json:"comment_count"`
	CreatedAt    string     `json:"created_at"`
	UpdatedAt    string     `json:"updated_at"`
}

// LikeStatus is the current like state.
type LikeStatus struct {
	PostID    string `json:"post_id"`
	LikedByMe bool   `json:"liked_by_me"`
	LikeCount int    `json:"like_count"`
}

// Comment is a post comment.
type Comment struct {
	ID        string `json:"id"`
	PostID    string `json:"post_id"`
	Author    User   `json:"author"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

// Checkin is a restaurant visit.
type Checkin struct {
	ID          string   `json:"id"`
	SpaceID     string   `json:"space_id"`
	Creator     User     `json:"creator"`
	Name        string   `json:"name"`
	Address     *string  `json:"address"`
	Lat         *float64 `json:"lat"`
	Lng         *float64 `json:"lng"`
	PlaceID     *string  `json:"place_id"`
	CheckedInAt string   `json:"checked_in_at"`
	CreatedAt   string   `json:"created_at"`
}

// Favorite is a saved restaurant.
type Favorite struct {
	ID        string   `json:"id"`
	SpaceID   string   `json:"space_id"`
	Creator   User     `json:"creator"`
	Name      string   `json:"name"`
	Address   *string  `json:"address"`
	Lat       *float64 `json:"lat"`
	Lng       *float64 `json:"lng"`
	PlaceID   *string  `json:"place_id"`
	Note      *string  `json:"note"`
	CreatedAt string   `json:"created_at"`
}

// Anniversary is a shared date and its reminder.
type Anniversary struct {
	ID               string `json:"id"`
	SpaceID          string `json:"space_id"`
	Creator          User   `json:"creator"`
	Title            string `json:"title"`
	EventDate        string `json:"event_date"`
	Recurrence       string `json:"recurrence"`
	Remind           bool   `json:"remind"`
	RemindDaysBefore int    `json:"remind_days_before"`
	RemindLocalTime  string `json:"remind_local_time"`
	Timezone         string `json:"timezone"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

// Device is a registered push token.
type Device struct {
	ID        string `json:"id"`
	Platform  string `json:"platform"`
	Token     string `json:"token"`
	UpdatedAt string `json:"updated_at"`
}

// List is a cursor page.
type List[T any] struct {
	Items      []T     `json:"items"`
	NextCursor *string `json:"next_cursor"`
}

// NewList returns a page, using an empty array instead of null.
func NewList[T any](items []T, next *string) List[T] {
	if items == nil {
		items = []T{}
	}
	return List[T]{Items: items, NextCursor: next}
}

func userFrom(u store.User) User {
	return User{
		ID:          u.ID,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		CreatedAt:   FormatTime(u.CreatedAt),
	}
}

// UserFrom maps a stored account.
func UserFrom(u store.User) User { return userFrom(u) }

// SpaceFrom maps a stored space.
func SpaceFrom(s store.Space) Space {
	members := make([]Member, 0, len(s.Members))
	for _, m := range s.Members {
		members = append(members, Member{
			UserID:      m.UserID,
			DisplayName: m.DisplayName,
			JoinedAt:    FormatTime(m.JoinedAt),
		})
	}
	return Space{
		ID:          s.ID,
		MemberCount: s.MemberCount,
		Members:     members,
		CreatedAt:   FormatTime(s.CreatedAt),
	}
}

// PostFrom maps a post, signing each media URL.
func PostFrom(p store.Post, urlOf func(string) (string, error)) (Post, error) {
	media := make([]MediaRef, 0, len(p.Media))
	for _, item := range p.Media {
		url, err := urlOf(item.StorageKey)
		if err != nil {
			return Post{}, err
		}
		media = append(media, MediaRef{
			ID:          item.ID,
			URL:         url,
			ContentType: item.ContentType,
			SortOrder:   item.SortOrder,
		})
	}
	out := Post{
		ID:           p.ID,
		SpaceID:      p.SpaceID,
		Author:       userFrom(p.Author),
		Body:         p.Body,
		Media:        media,
		LikeCount:    p.LikeCount,
		LikedByMe:    p.LikedByMe,
		CommentCount: p.CommentCount,
		CreatedAt:    FormatTime(p.CreatedAt),
		UpdatedAt:    FormatTime(p.UpdatedAt),
	}
	if p.Location != nil {
		out.Location = &Location{
			Lat:        p.Location.Lat,
			Lng:        p.Location.Lng,
			Name:       p.Location.Name,
			PlaceID:    p.Location.PlaceID,
			Visibility: p.Location.Visibility,
		}
	}
	return out, nil
}

// PostsFrom maps a page of posts.
func PostsFrom(items []store.Post, urlOf func(string) (string, error)) ([]Post, error) {
	out := make([]Post, 0, len(items))
	for _, item := range items {
		p, err := PostFrom(item, urlOf)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

// CommentFrom maps a comment.
func CommentFrom(c store.Comment) Comment {
	return Comment{
		ID:        c.ID,
		PostID:    c.PostID,
		Author:    userFrom(c.Author),
		Body:      c.Body,
		CreatedAt: FormatTime(c.CreatedAt),
	}
}

// CheckinFrom maps a check-in.
func CheckinFrom(c store.Checkin) Checkin {
	return Checkin{
		ID:          c.ID,
		SpaceID:     c.SpaceID,
		Creator:     userFrom(c.Creator),
		Name:        c.Name,
		Address:     c.Address,
		Lat:         c.Lat,
		Lng:         c.Lng,
		PlaceID:     c.PlaceID,
		CheckedInAt: FormatTime(c.CheckedInAt),
		CreatedAt:   FormatTime(c.CreatedAt),
	}
}

// FavoriteFrom maps a favorite.
func FavoriteFrom(f store.Favorite) Favorite {
	return Favorite{
		ID:        f.ID,
		SpaceID:   f.SpaceID,
		Creator:   userFrom(f.Creator),
		Name:      f.Name,
		Address:   f.Address,
		Lat:       f.Lat,
		Lng:       f.Lng,
		PlaceID:   f.PlaceID,
		Note:      f.Note,
		CreatedAt: FormatTime(f.CreatedAt),
	}
}

// AnniversaryFrom maps an anniversary.
func AnniversaryFrom(a store.Anniversary) Anniversary {
	return Anniversary{
		ID:               a.ID,
		SpaceID:          a.SpaceID,
		Creator:          userFrom(a.Creator),
		Title:            a.Title,
		EventDate:        a.EventDate,
		Recurrence:       a.Recurrence,
		Remind:           a.Remind,
		RemindDaysBefore: a.RemindDaysBefore,
		RemindLocalTime:  a.RemindLocalTime,
		Timezone:         a.Timezone,
		CreatedAt:        FormatTime(a.CreatedAt),
		UpdatedAt:        FormatTime(a.UpdatedAt),
	}
}
