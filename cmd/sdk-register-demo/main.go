package main

import (
    "crypto/tls"
    "fmt"
    "log"
    "net/http"
    "time"

    "github.com/yourorg/lcc-sdk/pkg/client"
    "github.com/yourorg/lcc-sdk/pkg/config"
)

func newInsecureHTTPClient(timeout time.Duration) *http.Client {
    tr := &http.Transport{
        TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
    }
    return &http.Client{
        Timeout:   timeout,
        Transport: tr,
    }
}

func testProductRegister(lccURL, productID string) error {
    cfg := &config.SDKConfig{
        LCCURL:         lccURL,
        ProductID:      productID,
        ProductVersion: "1.0.0",
        Timeout:        10 * time.Second,
        CacheTTL:       5 * time.Second,
    }

    c, err := client.NewClient(cfg)
    if err != nil {
        return fmt.Errorf("new client failed: %w", err)
    }
    defer c.Close()

    // Use HTTPS with self-signed cert
    c.SetHTTPClient(newInsecureHTTPClient(cfg.Timeout))

    if err := c.Register(); err != nil {
        return fmt.Errorf("register failed: %w", err)
    }

    log.Printf("✅ register success for product %s, instance=%s", productID, c.GetInstanceID())
    return nil
}

func main() {
    lccURL := "https://localhost:8088"
    products := []string{
        "demo-analytics-basic",
        "demo-analytics-pro",
        "demo-analytics-ent",
    }

    for _, pid := range products {
        log.Printf("==== testing register for %s ===", pid)
        if err := testProductRegister(lccURL, pid); err != nil {
            log.Printf("❌ register failed for %s: %v", pid, err)
        }
    }
}
