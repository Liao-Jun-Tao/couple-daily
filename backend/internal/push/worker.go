package push

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Liao-Jun-Tao/couple-daily/internal/store"
	"github.com/redis/go-redis/v9"
)

// Dedupe makes sure one anniversary fire is sent once.
type Dedupe interface {
	Once(ctx context.Context, key string, ttl time.Duration) (bool, error)
}

// MemoryDedupe is the single-process fallback when Redis is down.
type MemoryDedupe struct {
	mu    sync.Mutex
	until map[string]time.Time
}

func (d *MemoryDedupe) Once(_ context.Context, key string, ttl time.Duration) (bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.until == nil {
		d.until = map[string]time.Time{}
	}
	now := time.Now()
	if exp, ok := d.until[key]; ok && exp.After(now) {
		return false, nil
	}
	d.until[key] = now.Add(ttl)
	return true, nil
}

// RedisDedupe shares reminder idempotency across API processes.
type RedisDedupe struct {
	Client *redis.Client
}

func (d RedisDedupe) Once(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return d.Client.SetNX(ctx, "reminder:"+key, "1", ttl).Result()
}

// RunReminders scans due anniversaries once a minute until ctx is cancelled.
func RunReminders(ctx context.Context, db *store.DB, notifier *Notifier, dedupe Dedupe) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			func() {
				defer func() {
					if rec := recover(); rec != nil {
						log.Printf("push reminder panic: %v", rec)
					}
				}()
				if err := TickReminders(ctx, db, notifier, dedupe, time.Now()); err != nil {
					log.Printf("push reminder: %v", err)
				}
			}()
		}
	}
}

// TickReminders sends notifications that are due at now.
func TickReminders(ctx context.Context, db *store.DB, notifier *Notifier, dedupe Dedupe, now time.Time) error {
	if db == nil || notifier == nil || dedupe == nil {
		return nil
	}
	rows, err := db.ListReminders(ctx)
	if err != nil {
		return err
	}
	for _, row := range rows {
		fireAt, ok := ShouldFire(row.EventDate, row.Recurrence, row.LocalTime, row.Timezone, row.DaysBefore, now)
		if !ok {
			continue
		}
		key := row.ID + ":" + fireAt.UTC().Format(time.RFC3339)
		first, err := dedupe.Once(ctx, key, 48*time.Hour)
		if err != nil {
			log.Printf("push dedupe: %v", err)
			continue
		}
		if !first {
			continue
		}
		tokens, err := db.DevicesForSpace(ctx, row.SpaceID)
		if err != nil {
			log.Printf("push devices: %v", err)
			continue
		}
		msg := Message{Title: row.Title, Body: reminderBody(row.Title, row.DaysBefore)}
		for _, token := range tokens {
			if err := notifier.Send(ctx, token.Platform, token.Token, msg); err != nil {
				log.Printf("push send platform=%s: %v", token.Platform, err)
			}
		}
	}
	return nil
}

func reminderBody(title string, daysBefore int) string {
	if daysBefore > 0 {
		return fmt.Sprintf("「%s」还有 %d 天", title, daysBefore)
	}
	return fmt.Sprintf("今天是「%s」", title)
}
