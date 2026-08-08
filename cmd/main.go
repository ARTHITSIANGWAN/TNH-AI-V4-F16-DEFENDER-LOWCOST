package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type AIDispatch struct {
	Sender    string                 `json:"sender"`
	Receiver  string                 `json:"receiver"`
	Action    string                 `json:"action"`
	Payload   map[string]interface{} `json:"payload"`
	Timestamp string                 `json:"timestamp"`
}

func sendTaskToPythonAgent(ctx context.Context, targetURL string, action string, payload map[string]interface{}) error {
	msg := AIDispatch{
		Sender:    "ThitNuea-Core-Go",
		Receiver:  "Kaewta-AI-Python",
		Action:    action,
		Payload:   payload,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("agent returned status: %d", resp.StatusCode)
	}

	slog.Info("Command executed successfully", "action", action, "status", resp.Status)
	return nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	slog.Info("🐅 [ThitNuea Core]: Starting Sovereign Engine on Port 2026...")

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/dispatch", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "Accepted",
			"engine": "ThitNuea-Core-Go",
		})
	})

	server := &http.Server{
		Addr:         ":2026",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server error", "error", err)
		}
	}()

	slog.Info("🌐 Server Listening on http://127.0.0.1:2026")

	// ทดสอบส่งคำสั่งไปยัง Python Agent (Port 5000)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	taskData := map[string]interface{}{
		"project_name": "F-16 DEFENDER V.2",
		"status":       "Testing Zero-Garbage Matrix",
	}

	_ = sendTaskToPythonAgent(ctx, "http://127.0.0.1:5000/process", "ANALYZE_PROBABILITY", taskData)

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	slog.Info("🛑 Shutting down server gracefully...")
}

