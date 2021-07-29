package nakama

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/SherClockHolmes/webpush-go"
)

const (
	webPushNoticationSendTimeout = time.Second * 30
	webPushNoticationContact     = "contact@nakama.social"
)

func (svc *Service) AddWebPushSubscription(ctx context.Context, sub json.RawMessage) error {
	uid, ok := ctx.Value(KeyAuthUserID).(string)
	if !ok {
		return ErrUnauthenticated
	}

	query := "INSERT INTO user_web_push_subscriptions (user_id, sub) VALUES ($1, $2)"
	_, err := svc.DB.ExecContext(ctx, query, uid, sub)
	if isUniqueViolation(err) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("could not sql insert user web push subscription: %w", err)
	}

	return nil
}

func (svc *Service) webPushSubscriptions(ctx context.Context, userID string) ([]json.RawMessage, error) {
	query := "SELECT sub FROM user_web_push_subscriptions WHERE user_id = $1 ORDER BY created_at DESC"
	rows, err := svc.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("could not sql query select user web push susbcriptions: %w", err)
	}

	defer rows.Close()

	var subs []json.RawMessage
	for rows.Next() {
		var sub json.RawMessage
		err := rows.Scan(&sub)
		if err != nil {
			return nil, fmt.Errorf("could not sql scan user web push subscription: %w", err)
		}

		subs = append(subs, sub)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("could not sql query iterate over user web push subscriptions: %w", err)
	}

	return subs, nil
}

func (svc *Service) sendWebPushNotifications(n Notification) {
	ctx := context.Background()
	subs, err := svc.webPushSubscriptions(ctx, n.UserID)
	if err != nil {
		_ = svc.Logger.Log("err", err)
		return
	}

	if len(subs) == 0 {
		return
	}

	message, err := json.Marshal(n)
	if err != nil {
		_ = svc.Logger.Log("err", fmt.Errorf("could not json marshal web push notification message: %w", err))
		return
	}

	var topic string
	if n.PostID != nil {
		topic = "post." + *n.PostID
	}

	var wg sync.WaitGroup

	for _, sub := range subs {
		wg.Add(1)
		sub := sub
		go func() {
			defer wg.Done()

			err := svc.sendWebPushNotification(sub, message, topic)
			if err != nil {
				_ = svc.Logger.Log("err", err)
			}
		}()
	}

	wg.Wait()
}

func (svc *Service) sendWebPushNotification(rawSub json.RawMessage, message []byte, topic string) error {
	sub := &webpush.Subscription{}
	err := json.Unmarshal(rawSub, sub)
	if err != nil {
		return fmt.Errorf("could not json unmarshal web push subscription: %w", err)
	}

	resp, err := webpush.SendNotification(message, sub, &webpush.Options{
		Subscriber:      webPushNoticationContact,
		Topic:           topic,
		VAPIDPrivateKey: svc.VAPIDPrivateKey,
		VAPIDPublicKey:  svc.VAPIDPublicKey,
		TTL:             int(webPushNoticationSendTimeout.Seconds()),
	})
	if err != nil {
		return fmt.Errorf("could not send web push notification: %w", err)
	}

	_ = resp.Body.Close()

	return nil
}
