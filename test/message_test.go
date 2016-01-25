package test

import (
	"sync"
	"testing"
	"time"

	"golang.org/x/net/websocket"

	"github.com/neptulon/neptulon"
	"github.com/neptulon/neptulon/middleware"
	"github.com/neptulon/randstr"
)

type echoMsg struct {
	Message string `json:"message"`
}

var (
	msg1 = "Lorem ipsum dolor sit amet, consectetur adipiscing elit."
	msg2 = "In sit amet lectus felis, at pellentesque turpis."
	msg3 = "Nunc urna enim, cursus varius aliquet ac, imperdiet eget tellus."
	msg4 = randstr.Get(45 * 1000)       // 0.45 MB
	msg5 = randstr.Get(5 * 1000 * 1000) // 5.0 MB
)

func TestMessages(t *testing.T) {
	// todo: verify all message echoes from small to big
}

func TestBidirectional(t *testing.T) {
	// todo: test simultaneous read/writes
}

func TestEcho(t *testing.T) {
	sh := NewServerHelper(t).Start()
	defer sh.CloseWait()

	rout := middleware.NewRouter()
	sh.Middleware(rout.Middleware)
	rout.Request("echo", middleware.Echo)

	ch := sh.GetConnHelper().Connect()
	defer ch.CloseWait()

	m := "Hello!"
	ch.SendRequest("echo", echoMsg{Message: m}, func(ctx *neptulon.ResCtx) error {
		var msg echoMsg
		if err := ctx.Result(&msg); err != nil {
			t.Fatal(err)
		}
		if msg.Message != m {
			t.Fatalf("expected: %v got: %v", m, msg.Message)
		}
		return nil
	})
}

func TestEchoWithoutTestHelpers(t *testing.T) {
	s := neptulon.NewServer("127.0.0.1:3001")
	go s.Start()
	defer s.Close()
	time.Sleep(time.Millisecond)

	var wg sync.WaitGroup
	s.Middleware(func(ctx *neptulon.ReqCtx) error {
		defer wg.Done()
		t.Log("Request received:", ctx.Method)
		ctx.Res = "response-wow!"
		return ctx.Next()
	})

	wg.Add(1)

	origin := "http://127.0.0.1"
	url := "ws://127.0.0.1:3001"
	ws, err := websocket.Dial(url, "", origin)
	if err != nil {
		t.Fatal(err)
	}
	if err := websocket.JSON.Send(ws, map[string]string{"id": "123", "method": "test"}); err != nil {
		t.Fatal(err)
	}
	var res interface{}
	if err := websocket.JSON.Receive(ws, &res); err != nil {
		t.Fatal(err)
	}
	t.Log("Got response:", res)

	wg.Wait()
}

func TestTLS(t *testing.T) {
	// sh := NewServerHelper(t).UseTLS().Start()
	// defer sh.Close()
	//
	// rout := middleware.NewRouter()
	// sh.Middleware(rout.Middleware)
	// rout.Request("echo", middleware.Echo)
	//
	// ch := sh.GetConnHelper().UseTLS().Connect()
	// defer ch.Close()
	//
	// ch.SendRequest("echo", echoMsg{Message: "Hello!"}, func(ctx *neptulon.ResCtx) error {
	// 	var msg echoMsg
	// 	if err := ctx.Result(&msg); err != nil {
	// 		t.Fatal(err)
	// 	}
	// 	if msg.Message != "Hello!" {
	// 		t.Fatalf("expected: %v got: %v", "Hello!", msg.Message)
	// 	}
	// 	return nil
	// })
}
