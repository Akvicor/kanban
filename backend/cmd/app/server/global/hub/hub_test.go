package hub

import (
	"sync"
	"testing"
)

type fakeClient struct {
	mu        sync.Mutex
	delivered []Event
	closed    int
}

func (c *fakeClient) Deliver(events []Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.delivered = append(c.delivered, events...)
}

func (c *fakeClient) Close(code int, _ string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = code
}

func TestPublishOnlyReachesSameUser(t *testing.T) {
	h := New()
	alicePhone, aliceLaptop, bob := &fakeClient{}, &fakeClient{}, &fakeClient{}
	h.Register(1, 10, alicePhone)
	unregister := h.Register(1, 11, aliceLaptop)
	h.Register(2, 20, bob)

	h.Publish(1, []Event{{Revision: 1}})
	if len(alicePhone.delivered) != 1 || len(aliceLaptop.delivered) != 1 || len(bob.delivered) != 0 {
		t.Fatalf("推送范围不正确: %d %d %d", len(alicePhone.delivered), len(aliceLaptop.delivered), len(bob.delivered))
	}

	unregister()
	h.Publish(1, []Event{{Revision: 2}})
	if len(aliceLaptop.delivered) != 1 {
		t.Fatal("注销后的连接仍收到推送")
	}
}

func TestDisconnect(t *testing.T) {
	h := New()
	phone, laptop, bob := &fakeClient{}, &fakeClient{}, &fakeClient{}
	h.Register(1, 10, phone)
	h.Register(1, 11, laptop)
	h.Register(2, 20, bob)

	h.DisconnectDevice(1, 10)
	if phone.closed != CloseRevoked || laptop.closed != 0 {
		t.Fatalf("断开设备: phone=%d laptop=%d", phone.closed, laptop.closed)
	}
	h.DisconnectUser(1)
	if laptop.closed != CloseRevoked || bob.closed != 0 {
		t.Fatalf("断开用户: laptop=%d bob=%d", laptop.closed, bob.closed)
	}
}
