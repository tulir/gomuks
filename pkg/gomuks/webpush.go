package gomuks

import (
	"encoding/json"
	"net/http"

	"github.com/SherClockHolmes/webpush-go"
)

func (gmx *Gomuks) handleNewSubscription(w http.ResponseWriter, r *http.Request) {
	if gmx.Config.Push.Vapid == nil || gmx.Config.Push.Vapid.PublicKey == "" || gmx.Config.Push.Vapid.PrivateKey == "" {
		http.Error(w, "Push notifications are not enabled", http.StatusNotImplemented)
	}
	subscription := webpush.Subscription{}
	err := json.NewDecoder(r.Body).Decode(&subscription)
	if err != nil {
		http.Error(w, "Invalid subscription data", http.StatusBadRequest)
		return
	}
	if subscription.Endpoint == "" || subscription.Keys.P256dh == "" || subscription.Keys.Auth == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}
	gmx.Config.Push.WebPushSubscriptions = append(gmx.Config.Push.WebPushSubscriptions, subscription)
	if err := gmx.SaveConfig(); err != nil {
		http.Error(w, "Failed to save subscription", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (gmx *Gomuks) GetPubKey(w http.ResponseWriter, _ *http.Request) {
	if gmx.Config.Push.Vapid == nil || gmx.Config.Push.Vapid.PublicKey == "" {
		http.Error(w, "Push notifications are not enabled", http.StatusNotImplemented)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(gmx.Config.Push.Vapid.PublicKey))
	w.WriteHeader(http.StatusOK)
}

func (gmx *Gomuks) SendTestWebPushNotification(w http.ResponseWriter, _ *http.Request) {
	if gmx.Config.Push.Vapid == nil || gmx.Config.Push.Vapid.PublicKey == "" || gmx.Config.Push.Vapid.PrivateKey == "" {
		gmx.Log.Warn().Msg("Push notifications are not enabled")
		return
	}
	gmx.sendWebpushNotification(
		&PushNewMessage{Text: "Test webpush notification"})
	w.WriteHeader(http.StatusOK)
}

func (gmx *Gomuks) sendWebpushNotification(msg *PushNewMessage) {
	if gmx.Config.Push.Vapid == nil || gmx.Config.Push.Vapid.PublicKey == "" || gmx.Config.Push.Vapid.PrivateKey == "" {
		return
	}
	for _, subscription := range gmx.Config.Push.WebPushSubscriptions {
		resp, err := webpush.SendNotification(
			[]byte(msg.Text), &subscription, &webpush.Options{
				HTTPClient:      pushClient,
				VAPIDPublicKey:  gmx.Config.Push.Vapid.PublicKey,
				VAPIDPrivateKey: gmx.Config.Push.Vapid.PrivateKey,
			})
		if err != nil {
			gmx.Log.Err(err).Msg("Failed to send push notification")
			continue
		}
		resp.Body.Close()
	}
}
