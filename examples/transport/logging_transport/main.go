// Implement a custom client.Client transport. The SDK's generated services
// depend only on the six-verb interface, so wrapping or replacing the
// transport is the extension point for real executors (an OMA-DM session,
// the MDM WMI bridge) and for cross-cutting concerns like logging.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/deploymenttheory/go-sdk-windowscsp/windowscsp"
	"github.com/deploymenttheory/go-sdk-windowscsp/windowscsp/client"
	"github.com/deploymenttheory/go-sdk-windowscsp/windowscsp/clienttest"
	"github.com/deploymenttheory/go-sdk-windowscsp/windowscsp/policy/camera"
)

// loggingTransport wraps any client.Client and logs every operation.
type loggingTransport struct {
	next client.Client
}

func (t *loggingTransport) Get(ctx context.Context, uri string) (client.Value, error) {
	log.Printf("GET     %s", uri)
	return t.next.Get(ctx, uri)
}

func (t *loggingTransport) List(ctx context.Context, uri string) ([]string, error) {
	log.Printf("LIST    %s", uri)
	return t.next.List(ctx, uri)
}

func (t *loggingTransport) Add(ctx context.Context, uri string, v client.Value) error {
	log.Printf("ADD     %s = %q (%s)", uri, v.Data, v.Format)
	return t.next.Add(ctx, uri, v)
}

func (t *loggingTransport) Replace(ctx context.Context, uri string, v client.Value) error {
	log.Printf("REPLACE %s = %q (%s)", uri, v.Data, v.Format)
	return t.next.Replace(ctx, uri, v)
}

func (t *loggingTransport) Delete(ctx context.Context, uri string) error {
	log.Printf("DELETE  %s", uri)
	return t.next.Delete(ctx, uri)
}

func (t *loggingTransport) Exec(ctx context.Context, uri string, v client.Value) error {
	log.Printf("EXEC    %s", uri)
	return t.next.Exec(ctx, uri, v)
}

func main() {
	c := windowscsp.NewClient(&loggingTransport{next: clienttest.New()})
	ctx := context.Background()

	if err := c.Policy.Camera.UpdateAllowCamera(ctx, camera.AllowCameraAllowed); err != nil {
		log.Fatalf("update: %v", err)
	}
	v, err := c.Policy.Camera.GetAllowCamera(ctx)
	if err != nil {
		log.Fatalf("get: %v", err)
	}
	fmt.Printf("AllowCamera = %d\n", v)
}
