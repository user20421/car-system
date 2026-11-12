package realtime

import (
	"testing"
	"time"
)

func TestTicketSingleUseAndExpiry(t *testing.T) {
	now := time.Unix(1000, 0)
	tickets := NewTickets(time.Minute)
	tickets.now = func() time.Time { return now }
	token, expires, err := tickets.Issue("user-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(token) < 32 || expires != now.Add(time.Minute).UnixMilli() {
		t.Fatal("invalid ticket")
	}
	uid, err := tickets.RedeemUserID(token)
	if err != nil || uid != "user-1" {
		t.Fatalf("redeem: %s %v", uid, err)
	}
	if _, err := tickets.RedeemUserID(token); err == nil {
		t.Fatal("ticket was reusable")
	}
	token2, _, _ := tickets.Issue("user-2")
	now = now.Add(61 * time.Second)
	if _, err := tickets.RedeemUserID(token2); err == nil {
		t.Fatal("expired ticket accepted")
	}
}
