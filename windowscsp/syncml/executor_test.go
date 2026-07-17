package syncml

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/deploymenttheory/go-sdk-windowscsp/windowscsp/client"
)

// cannedSender asserts the outgoing request and returns a fixed response.
func cannedSender(t *testing.T, wantInRequest []string, response string) SenderFunc {
	t.Helper()
	return func(_ context.Context, request string) (string, error) {
		for _, want := range wantInRequest {
			if !strings.Contains(request, want) {
				t.Errorf("request missing %q:\n%s", want, request)
			}
		}
		return response, nil
	}
}

func TestExecutorGet(t *testing.T) {
	resp := `<SyncBody>
	  <Status><CmdID>1</CmdID><MsgRef>1</MsgRef><CmdRef>0</CmdRef><Cmd>SyncHdr</Cmd><Data>200</Data></Status>
	  <Status><CmdID>2</CmdID><CmdRef>1</CmdRef><Cmd>Get</Cmd><Data>200</Data></Status>
	  <Results><CmdID>3</CmdID><CmdRef>1</CmdRef>
	    <Item>
	      <Source><LocURI>./Device/Vendor/MSFT/Policy/Config/Camera/AllowCamera</LocURI></Source>
	      <Meta><Format xmlns="syncml:metinf">int</Format></Meta>
	      <Data>1</Data>
	    </Item>
	  </Results>
	</SyncBody>`
	e := NewExecutor(cannedSender(t, []string{"<Get>", "<LocURI>./Device/Vendor/MSFT/Policy/Config/Camera/AllowCamera</LocURI>"}, resp))

	v, err := e.Get(context.Background(), "./Device/Vendor/MSFT/Policy/Config/Camera/AllowCamera")
	if err != nil {
		t.Fatal(err)
	}
	if v.Format != client.FormatInt {
		t.Errorf("format = %q", v.Format)
	}
	if n, err := v.Int(); err != nil || n != 1 {
		t.Errorf("value = %v %v", n, err)
	}
}

func TestExecutorGetFullEnvelope(t *testing.T) {
	resp := `<SyncML xmlns="SYNCML:SYNCML1.2"><SyncHdr></SyncHdr><SyncBody>
	  <Status><CmdRef>1</CmdRef><Cmd>Get</Cmd><Data>200</Data></Status>
	  <Results><CmdRef>1</CmdRef><Item><Data>hello</Data></Item></Results>
	</SyncBody></SyncML>`
	e := NewExecutor(cannedSender(t, nil, resp))

	v, err := e.Get(context.Background(), "./x")
	if err != nil {
		t.Fatal(err)
	}
	// Missing Meta/Format defaults to chr.
	if v.Format != client.FormatChr || v.Str() != "hello" {
		t.Errorf("value = %+v", v)
	}
}

func TestExecutorList(t *testing.T) {
	resp := `<SyncBody>
	  <Status><CmdRef>1</CmdRef><Cmd>Get</Cmd><Data>200</Data></Status>
	  <Results><CmdRef>1</CmdRef><Item><Meta><Format>node</Format></Meta><Data>Corp/Personal</Data></Item></Results>
	</SyncBody>`
	e := NewExecutor(cannedSender(t, nil, resp))

	names, err := e.List(context.Background(), "./Device/Vendor/MSFT/VPNv2")
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 || names[0] != "Corp" || names[1] != "Personal" {
		t.Fatalf("names = %v", names)
	}
}

func TestExecutorWriteAndStatusError(t *testing.T) {
	okResp := `<SyncBody><Status><CmdRef>1</CmdRef><Cmd>Replace</Cmd><Data>200</Data></Status></SyncBody>`
	e := NewExecutor(cannedSender(t, []string{"<Replace>", "<Data>0</Data>"}, okResp))
	if err := e.Replace(context.Background(), "./x", client.Int(0)); err != nil {
		t.Fatal(err)
	}

	notFound := `<SyncBody><Status><CmdRef>1</CmdRef><Cmd>Delete</Cmd><Data>404</Data></Status></SyncBody>`
	e = NewExecutor(cannedSender(t, nil, notFound))
	err := e.Delete(context.Background(), "./gone")
	var se *client.StatusError
	if !errors.As(err, &se) || se.Code != 404 {
		t.Fatalf("err = %v", err)
	}
	if !errors.Is(err, client.ErrNotFound) {
		t.Fatalf("404 should unwrap to ErrNotFound, got %v", err)
	}
}

func TestExecutorMalformedResponses(t *testing.T) {
	e := NewExecutor(cannedSender(t, nil, ""))
	if _, err := e.Get(context.Background(), "./x"); err == nil {
		t.Error("empty response should fail")
	}

	e = NewExecutor(cannedSender(t, nil, `<SyncBody></SyncBody>`))
	if err := e.Exec(context.Background(), "./x", client.Null()); err == nil {
		t.Error("response without Status should fail")
	}

	// Status OK but no Results for a Get.
	e = NewExecutor(cannedSender(t, nil, `<SyncBody><Status><CmdRef>1</CmdRef><Cmd>Get</Cmd><Data>200</Data></Status></SyncBody>`))
	if _, err := e.Get(context.Background(), "./x"); err == nil {
		t.Error("Get without Results should fail")
	}
}

func TestExecutorSenderError(t *testing.T) {
	e := NewExecutor(func(context.Context, string) (string, error) {
		return "", errors.New("pipe broken")
	})
	if err := e.Replace(context.Background(), "./x", client.Int(1)); err == nil || !strings.Contains(err.Error(), "pipe broken") {
		t.Fatalf("err = %v", err)
	}
}
