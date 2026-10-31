package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Liao-Jun-Tao/couple-daily/internal/config"
	"github.com/Liao-Jun-Tao/couple-daily/internal/dbmigrate"
	"github.com/Liao-Jun-Tao/couple-daily/migrations"
	"github.com/jackc/pgx/v5"
)

func TestP0AuthInviteAndSpace(t *testing.T) {
	cfg := integrationConfig(t)
	ctx := context.Background()
	if err := dbmigrate.Apply(ctx, cfg.DatabaseURL, migrations.Files); err != nil {
		t.Fatal(err)
	}
	if err := dbmigrate.Apply(ctx, cfg.DatabaseURL, migrations.Files); err != nil {
		t.Fatalf("migrate twice: %v", err)
	}

	application, err := New(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(application.Close)
	if _, err := application.DB.Pool.Exec(ctx, `
		TRUNCATE TABLE
			device_tokens, anniversaries, restaurant_favorites, restaurant_checkins,
			comments, likes, post_media, posts, media_assets, invite_codes,
			space_members, couple_spaces, refresh_tokens, users
		RESTART IDENTITY CASCADE
	`); err != nil {
		t.Fatal(err)
	}

	h := application.Handler
	if code, body := do(t, h, http.MethodPost, "/v1/auth/register", "", map[string]any{
		"email": "a@example.com", "password": "short", "display_name": "Alice",
	}); code != http.StatusBadRequest {
		t.Fatalf("short password %d %v", code, body)
	}

	alice := register(t, h, "a@example.com", "Alice")
	if code, body := do(t, h, http.MethodPost, "/v1/auth/register", "", map[string]any{
		"email": "a@example.com", "password": "password123", "display_name": "Alice",
	}); code != http.StatusConflict || body["code"] != "EMAIL_TAKEN" {
		t.Fatalf("duplicate %d %v", code, body)
	}
	if code, body := do(t, h, http.MethodPost, "/v1/auth/login", "", map[string]any{
		"email": "a@example.com", "password": "wrong-password",
	}); code != http.StatusUnauthorized || body["code"] != "INVALID_CREDENTIALS" {
		t.Fatalf("bad password %d %v", code, body)
	}
	loginCode, loginBody := do(t, h, http.MethodPost, "/v1/auth/login", "", map[string]any{
		"email": "A@Example.com", "password": "password123",
	})
	if loginCode != http.StatusOK || loginBody["access_token"] == "" {
		t.Fatalf("login %d %v", loginCode, loginBody)
	}
	alice.token = loginBody["access_token"].(string)

	meCode, me := do(t, h, http.MethodGet, "/v1/auth/me", alice.token, nil)
	if meCode != http.StatusOK || me["pairing"].(map[string]any)["status"] != "unpaired" {
		t.Fatalf("me unpaired %d %v", meCode, me)
	}

	invCode, inv := do(t, h, http.MethodPost, "/v1/pairing/invites", alice.token, nil)
	if invCode != http.StatusCreated {
		t.Fatalf("invite %d %v", invCode, inv)
	}
	code := inv["invite_code"].(string)
	if len(code) != 6 || strings.ToUpper(code) != code {
		t.Fatalf("code %q", code)
	}
	space := inv["space"].(map[string]any)
	if space["member_count"].(float64) != 1 {
		t.Fatalf("member_count %v", space["member_count"])
	}
	spaceID := space["id"].(string)
	expires, err := time.Parse(time.RFC3339, inv["expires_at"].(string))
	if err != nil {
		t.Fatal(err)
	}
	delta := time.Until(expires)
	if delta < 23*time.Hour || delta > 25*time.Hour {
		t.Fatalf("ttl %s", delta)
	}

	if code, body := do(t, h, http.MethodPost, "/v1/pairing/invites/accept", alice.token, map[string]any{"invite_code": code}); code != http.StatusConflict || body["code"] != "ALREADY_PAIRED" {
		t.Fatalf("self accept %d %v", code, body)
	}

	bob := register(t, h, "b@example.com", "Bob")
	acceptCode, accepted := do(t, h, http.MethodPost, "/v1/pairing/invites/accept", bob.token, map[string]any{"invite_code": strings.ToLower(code)})
	if acceptCode != http.StatusOK || accepted["id"] != spaceID || accepted["member_count"].(float64) != 2 {
		t.Fatalf("accept %d %v", acceptCode, accepted)
	}
	members := accepted["members"].([]any)
	if len(members) != 2 {
		t.Fatalf("members %v", members)
	}

	carol := register(t, h, "c@example.com", "Carol")
	if code, body := do(t, h, http.MethodPost, "/v1/pairing/invites/accept", carol.token, map[string]any{"invite_code": code}); code != http.StatusConflict || body["code"] != "SPACE_FULL" {
		t.Fatalf("third %d %v", code, body)
	}
	if code, body := do(t, h, http.MethodPost, "/v1/pairing/invites", alice.token, nil); code != http.StatusConflict || body["code"] != "ALREADY_PAIRED" {
		t.Fatalf("reinvite %d %v", code, body)
	}
	if code, body := do(t, h, http.MethodGet, "/v1/posts", carol.token, nil); code != http.StatusForbidden || body["code"] != "NOT_PAIRED" {
		t.Fatalf("unpaired posts %d %v", code, body)
	}

	meCode, me = do(t, h, http.MethodGet, "/v1/auth/me", bob.token, nil)
	pairing := me["pairing"].(map[string]any)
	if meCode != http.StatusOK || pairing["status"] != "paired" || pairing["space_id"] != spaceID {
		t.Fatalf("bob me %d %v", meCode, me)
	}
	spaceCode, gotSpace := do(t, h, http.MethodGet, "/v1/pairing/space", alice.token, nil)
	if spaceCode != http.StatusOK || gotSpace["id"] != spaceID {
		t.Fatalf("space %d %v", spaceCode, gotSpace)
	}

	dave := register(t, h, "d@example.com", "Dave")
	_, daveInv := do(t, h, http.MethodPost, "/v1/pairing/invites", dave.token, nil)
	daveCode := daveInv["invite_code"].(string)
	if _, err := application.DB.Pool.Exec(ctx, `UPDATE invite_codes SET expires_at = now() - interval '1 minute' WHERE code = $1`, daveCode); err != nil {
		t.Fatal(err)
	}
	eve := register(t, h, "e@example.com", "Eve")
	if code, body := do(t, h, http.MethodPost, "/v1/pairing/invites/accept", eve.token, map[string]any{"invite_code": daveCode}); code != http.StatusConflict || body["code"] != "INVITE_EXPIRED" {
		t.Fatalf("expired %d %v", code, body)
	}

	frank := register(t, h, "f@example.com", "Frank")
	_, frankInv := do(t, h, http.MethodPost, "/v1/pairing/invites", frank.token, nil)
	frankCode := frankInv["invite_code"].(string)
	if _, err := application.DB.Pool.Exec(ctx, `UPDATE invite_codes SET status = 'accepted', accepted_at = now() WHERE code = $1`, frankCode); err != nil {
		t.Fatal(err)
	}
	gina := register(t, h, "g@example.com", "Gina")
	if code, body := do(t, h, http.MethodPost, "/v1/pairing/invites/accept", gina.token, map[string]any{"invite_code": frankCode}); code != http.StatusConflict || body["code"] != "INVITE_USED" {
		t.Fatalf("used %d %v", code, body)
	}
	if code, body := do(t, h, http.MethodPost, "/v1/pairing/invites/accept", gina.token, map[string]any{"invite_code": "NOPE"}); code != http.StatusBadRequest {
		t.Fatalf("bad code %d %v", code, body)
	}
	if code, body := do(t, h, http.MethodPost, "/v1/pairing/invites/accept", gina.token, map[string]any{"invite_code": "ZZZZZZ"}); code != http.StatusNotFound || body["code"] != "INVITE_NOT_FOUND" {
		t.Fatalf("missing code %d %v", code, body)
	}

	refresh := alice.refresh
	refCode, refBody := do(t, h, http.MethodPost, "/v1/auth/refresh", "", map[string]any{"refresh_token": refresh})
	if refCode != http.StatusOK {
		t.Fatalf("refresh %d %v", refCode, refBody)
	}
	alice.token = refBody["access_token"].(string)
	if code, _ := do(t, h, http.MethodGet, "/v1/pairing/space", alice.token, nil); code != http.StatusOK {
		t.Fatalf("refreshed access %d", code)
	}
	if code, body := do(t, h, http.MethodPost, "/v1/auth/refresh", "", map[string]any{"refresh_token": refresh}); code != http.StatusUnauthorized {
		t.Fatalf("reused refresh %d %v", code, body)
	}

	if code, body := do(t, h, http.MethodPost, "/v1/posts", alice.token, map[string]any{"body": "   "}); code != http.StatusBadRequest || body["code"] != "EMPTY_POST" {
		t.Fatalf("empty post %d %v", code, body)
	}
	if code, body := do(t, h, http.MethodPost, "/v1/media/uploads", alice.token, map[string]any{"content_type": "image/gif", "byte_size": 10}); code != http.StatusBadRequest {
		t.Fatalf("gif %d %v", code, body)
	}
	if code, body := do(t, h, http.MethodPost, "/v1/media/uploads", alice.token, map[string]any{"content_type": "image/jpeg", "byte_size": 10485761}); code != http.StatusBadRequest {
		t.Fatalf("too big %d %v", code, body)
	}
	upCode, up := do(t, h, http.MethodPost, "/v1/media/uploads", alice.token, map[string]any{"content_type": "image/jpeg", "byte_size": 12345})
	if upCode != http.StatusCreated || up["upload_url"] == "" || up["media_id"] == "" {
		t.Fatalf("upload %d %v", upCode, up)
	}
	postCode, post := do(t, h, http.MethodPost, "/v1/posts", alice.token, map[string]any{
		"body":      "今天天气真好",
		"media_ids": []string{up["media_id"].(string)},
		"location":  map[string]any{"lat": 31.24, "lng": 121.49, "name": "外滩"},
	})
	if postCode != http.StatusCreated {
		t.Fatalf("post %d %v", postCode, post)
	}
	if post["location"].(map[string]any)["visibility"] != "exact" {
		t.Fatalf("visibility %v", post["location"])
	}
	if len(post["media"].([]any)) != 1 {
		t.Fatalf("media %v", post["media"])
	}
	postID := post["id"].(string)

	_, bobPost := do(t, h, http.MethodPost, "/v1/posts", bob.token, map[string]any{"body": "第二条，应排在前面"})
	listCode, list := do(t, h, http.MethodGet, "/v1/posts?limit=20", alice.token, nil)
	if listCode != http.StatusOK {
		t.Fatalf("list %d %v", listCode, list)
	}
	items := list["items"].([]any)
	if len(items) < 2 || items[0].(map[string]any)["body"] != "第二条，应排在前面" {
		t.Fatalf("order %v", items)
	}
	_, bobList := do(t, h, http.MethodGet, "/v1/posts?limit=20", bob.token, nil)
	if bobList["items"].([]any)[0].(map[string]any)["id"] != bobPost["id"] {
		t.Fatalf("bob timeline %v", bobList)
	}

	likeCode, like := do(t, h, http.MethodPut, "/v1/posts/"+postID+"/likes", alice.token, nil)
	if likeCode != http.StatusOK || like["liked_by_me"] != true || like["like_count"].(float64) != 1 {
		t.Fatalf("like %d %v", likeCode, like)
	}
	cmtCode, cmt := do(t, h, http.MethodPost, "/v1/posts/"+postID+"/comments", bob.token, map[string]any{"body": "拍得真好看"})
	if cmtCode != http.StatusCreated || cmt["author"].(map[string]any)["display_name"] != "Bob" {
		t.Fatalf("comment %d %v", cmtCode, cmt)
	}
	if code, body := do(t, h, http.MethodDelete, "/v1/comments/"+cmt["id"].(string), alice.token, nil); code != http.StatusForbidden {
		t.Fatalf("comment delete %d %v", code, body)
	}
	if code, _ := do(t, h, http.MethodDelete, "/v1/comments/"+cmt["id"].(string), bob.token, nil); code != http.StatusNoContent {
		t.Fatalf("author comment delete %d", code)
	}
	if code, body := do(t, h, http.MethodDelete, "/v1/posts/"+postID, bob.token, nil); code != http.StatusForbidden {
		t.Fatalf("post delete %d %v", code, body)
	}

	ckCode, ck := do(t, h, http.MethodPost, "/v1/checkins", alice.token, map[string]any{"name": "一乐拉面"})
	if ckCode != http.StatusCreated || ck["name"] != "一乐拉面" || ck["place_id"] != nil {
		t.Fatalf("checkin %d %v", ckCode, ck)
	}
	if code, body := do(t, h, http.MethodGet, "/v1/checkins", bob.token, nil); code != http.StatusOK || len(body["items"].([]any)) != 1 {
		t.Fatalf("checkin list %d %v", code, body)
	}

	annCode, ann := do(t, h, http.MethodPost, "/v1/anniversaries", alice.token, map[string]any{
		"title": "在一起", "event_date": "2024-05-20",
	})
	if annCode != http.StatusCreated || ann["remind_local_time"] != "09:00" || ann["timezone"] != "Asia/Shanghai" || ann["remind"] != true {
		t.Fatalf("anniversary %d %v", annCode, ann)
	}
	annID := ann["id"].(string)
	if code, body := do(t, h, http.MethodPatch, "/v1/anniversaries/"+annID, bob.token, map[string]any{
		"title": "被改掉", "remind": false, "remind_days_before": 1, "remind_local_time": "10:00", "timezone": "Asia/Tokyo",
	}); code != http.StatusForbidden || body["code"] != "FORBIDDEN" {
		t.Fatalf("patch anniversary non-creator %d %v", code, body)
	}
	if code, body := do(t, h, http.MethodPatch, "/v1/anniversaries/"+annID, bob.token, map[string]any{"remind_local_time": ""}); code != http.StatusForbidden || body["code"] != "FORBIDDEN" {
		t.Fatalf("patch anniversary invalid field non-creator %d %v", code, body)
	}
	listCode, listBody := do(t, h, http.MethodGet, "/v1/anniversaries", bob.token, nil)
	if listCode != http.StatusOK {
		t.Fatalf("list after forbidden patch %d %v", listCode, listBody)
	}
	var seen map[string]any
	for _, raw := range listBody["items"].([]any) {
		item := raw.(map[string]any)
		if item["id"] == annID {
			seen = item
			break
		}
	}
	if seen == nil || seen["title"] != "在一起" || seen["remind"] != true || seen["remind_days_before"].(float64) != 0 || seen["remind_local_time"] != "09:00" || seen["timezone"] != "Asia/Shanghai" {
		t.Fatalf("forbidden patch mutated anniversary %v", seen)
	}
	if code, body := do(t, h, http.MethodPatch, "/v1/anniversaries/"+annID, alice.token, map[string]any{"remind_days_before": 1}); code != http.StatusOK || body["remind_days_before"].(float64) != 1 {
		t.Fatalf("patch anniversary creator %d %v", code, body)
	}
	if code, body := do(t, h, http.MethodDelete, "/v1/anniversaries/"+annID, bob.token, nil); code != http.StatusForbidden || body["code"] != "FORBIDDEN" {
		t.Fatalf("ann delete %d %v", code, body)
	}
	if code, body := do(t, h, http.MethodDelete, "/v1/anniversaries/"+annID, alice.token, nil); code != http.StatusNoContent {
		t.Fatalf("ann creator delete %d %v", code, body)
	}

	devCode, dev := do(t, h, http.MethodPost, "/v1/devices", alice.token, map[string]any{"platform": "android", "token": "TEST_FCM_TOKEN_A"})
	if devCode != http.StatusOK || dev["platform"] != "android" || dev["token"] != "TEST_FCM_TOKEN_A" {
		t.Fatalf("fcm device %d %v", devCode, dev)
	}
	iosCode, iosDev := do(t, h, http.MethodPost, "/v1/devices", bob.token, map[string]any{"platform": "ios", "token": "TEST_APNS_TOKEN_B"})
	if iosCode != http.StatusOK || iosDev["platform"] != "ios" {
		t.Fatalf("apns device %d %v", iosCode, iosDev)
	}
	if code, _ := do(t, h, http.MethodDelete, "/v1/devices", alice.token, map[string]any{"token": "TEST_FCM_TOKEN_A"}); code != http.StatusNoContent {
		t.Fatalf("unregister %d", code)
	}

	favCode, fav := do(t, h, http.MethodPost, "/v1/favorites", bob.token, map[string]any{"name": "海底捞"})
	if favCode != http.StatusCreated {
		t.Fatalf("favorite %d %v", favCode, fav)
	}
	if code, body := do(t, h, http.MethodDelete, "/v1/favorites/"+fav["id"].(string), alice.token, nil); code != http.StatusForbidden {
		t.Fatalf("favorite auth %d %v", code, body)
	}
	if code, _ := do(t, h, http.MethodDelete, "/v1/favorites/"+fav["id"].(string), bob.token, nil); code != http.StatusNoContent {
		t.Fatalf("favorite delete %d", code)
	}
}

type session struct {
	token   string
	refresh string
}

func register(t *testing.T, h http.Handler, email, name string) session {
	t.Helper()
	code, body := do(t, h, http.MethodPost, "/v1/auth/register", "", map[string]any{
		"email": email, "password": "password123", "display_name": name,
	})
	if code != http.StatusCreated {
		t.Fatalf("register %s: %d %v", email, code, body)
	}
	return session{token: body["access_token"].(string), refresh: body["refresh_token"].(string)}
}

func do(t *testing.T, h http.Handler, method, path, token string, body any) (int, map[string]any) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Body.Len() == 0 {
		return rr.Code, nil
	}
	var decoded map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("%s %s status %d body %s", method, path, rr.Code, rr.Body.String())
	}
	return rr.Code, decoded
}

func integrationConfig(t *testing.T) config.Config {
	t.Helper()
	cfg := config.Load()
	cfg.AppEnv = "local"
	cfg.JWTSecret = "test-secret"
	cfg.JWTAccessTTL = time.Hour
	cfg.JWTRefreshTTL = 24 * time.Hour
	if !strings.Contains(cfg.DatabaseURL, "localhost") && !strings.Contains(cfg.DatabaseURL, "127.0.0.1") {
		t.Skip("integration test only runs against localhost")
	}
	if !strings.Contains(cfg.DatabaseURL, "/couple_daily") {
		t.Skip("refusing database that is not couple_daily")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	_ = conn.Close(ctx)
	return cfg
}
