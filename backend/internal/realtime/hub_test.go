package realtime

import "testing"

type fakeSender struct {
	received [][]byte
	failing  bool
}

func (f *fakeSender) Send(data []byte) error {
	if f.failing {
		return errSendFailed
	}
	f.received = append(f.received, data)
	return nil
}

var errSendFailed = &sendError{}

type sendError struct{}

func (*sendError) Error() string { return "send failed" }

func TestBroadcastLocal_OnlyReachesSameProjectClients(t *testing.T) {
	hub := NewHub()
	clientA := &fakeSender{}
	clientB := &fakeSender{}
	hub.Register("project-1", clientA)
	hub.Register("project-2", clientB)

	hub.BroadcastLocal("project-1", []byte("hello"))

	if len(clientA.received) != 1 || string(clientA.received[0]) != "hello" {
		t.Errorf("expected clientA to receive the message, got %v", clientA.received)
	}
	if len(clientB.received) != 0 {
		t.Errorf("expected clientB (different project) to receive nothing, got %v", clientB.received)
	}
}

func TestBroadcastLocal_ReachesAllClientsInProject(t *testing.T) {
	hub := NewHub()
	a, b := &fakeSender{}, &fakeSender{}
	hub.Register("project-1", a)
	hub.Register("project-1", b)

	hub.BroadcastLocal("project-1", []byte("update"))

	if len(a.received) != 1 || len(b.received) != 1 {
		t.Errorf("expected both clients to receive the broadcast, got a=%v b=%v", a.received, b.received)
	}
}

func TestUnregister_StopsFurtherDelivery(t *testing.T) {
	hub := NewHub()
	client := &fakeSender{}
	hub.Register("project-1", client)
	hub.Unregister("project-1", client)

	hub.BroadcastLocal("project-1", []byte("late message"))

	if len(client.received) != 0 {
		t.Errorf("expected no delivery after unregister, got %v", client.received)
	}
	if count := hub.ConnectionCount("project-1"); count != 0 {
		t.Errorf("expected connection count 0 after last unregister, got %d", count)
	}
}

func TestBroadcastLocal_FailingSenderDoesNotBlockOthers(t *testing.T) {
	hub := NewHub()
	dead := &fakeSender{failing: true}
	alive := &fakeSender{}
	hub.Register("project-1", dead)
	hub.Register("project-1", alive)

	hub.BroadcastLocal("project-1", []byte("update"))

	if len(alive.received) != 1 {
		t.Errorf("expected the alive client to still receive the broadcast, got %v", alive.received)
	}
}
