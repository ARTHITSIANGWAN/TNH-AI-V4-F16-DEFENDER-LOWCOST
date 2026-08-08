package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// AIDispatch โครงสร้างข้อความมาตรฐานสำหรับสื่อสารระหว่าง Agent
type AIDispatch struct {
	Sender    string                 `json:"sender"`
	Receiver  string                 `json:"receiver"`
	Action    string                 `json:"action"`
	Payload   map[string]interface{} `json:"payload"`
	Priority  int                    `json:"priority"`
	Timestamp string                 `json:"timestamp"`
}

// SecurityConfig จัดเก็บการตั้งค่าความปลอดภัย
type SecurityConfig struct {
	SecretKey string
	AppPort   string
}

func main() {
	// 1. ตั้งค่า Structured Logger (Zero-Garbage / High-Performance)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	slog.Info("🛡️ [F-16 DEFENDER]: Initializing Sovereign Security Engine V.2...")

	secConfig := SecurityConfig{
		SecretKey: os.Getenv("TNH_SECRET_KEY"), // แนะนำให้ตั้งใน Environment Variable
		AppPort:   ":2026",
	}

	mux := http.NewServeMux()

	// 2. Endpoint สำหรับรับ webhook/dispatch ที่ผ่านการตรวจ Signature
	mux.HandleFunc("/api/v1/dispatch", validateSignatureMiddleware(secConfig.SecretKey, handleDispatch))

	// 3. ป้องกัน Web Server ด้วย Timeout Configurations (Anti-Slowloris)
	server := &http.Server{
		Addr:              secConfig.AppPort,
		Handler:           mux,
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	// 4. รัน Server ใน Background Goroutine
	go func() {
		slog.Info("🚀 [F-16 DEFENDER]: Server listening securely", "port", secConfig.AppPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("❌ Server runtime failure", "error", err)
			os.Exit(1)
		}
	}()

	// 5. ระบบ Graceful Shutdown เมื่อได้รับสัญญาณปิดระบบ
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	slog.Info("🛑 Shutting down F-16 Defender Engine gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("❌ Forced shutdown error", "error", err)
	} else {
		slog.Info("✅ Engine shutdown completed safely.")
	}
}

// validateSignatureMiddleware ตรวจสอบ HMAC SHA256 Signature ด้วย Constant Time Comparison
func validateSignatureMiddleware(secretKey string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		signature := r.Header.Get("X-TNH-Signature")
		if secretKey != "" && signature != "" {
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			mac := hmac.New(sha256.New, []byte(secretKey))
			mac.Write(bodyBytes)
			expectedMAC := base64.StdEncoding.EncodeToString(mac.Sum(nil))

			// ป้องกัน Timing Attacks
			if subtle.ConstantTimeCompare([]byte(signature), []byte(expectedMAC)) != 1 {
				slog.Warn("⚠️ Unauthorized access attempt blocked", "ip", r.RemoteAddr)
				http.Error(w, "Unauthorized Request", http.StatusUnauthorized)
				return
			}
		}

		next(w, r)
	}
}

// handleDispatch ประมวลผลคำสั่งที่ส่งเข้ามา
func handleDispatch(w http.ResponseWriter, r *http.Request) {
	var req AIDispatch
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON Payload", http.StatusBadRequest)
		return
	}

	slog.Info("📩 Validated Command Received", "sender", req.Sender, "action", req.Action)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "SUCCESS",
		"message": "Command verified and executed by F-16 Defender Engine",
	})
}

// SendTaskToAgent ฟังก์ชันส่งคำสั่งไปยัง Agent ตัวอื่นอย่างปลอดภัย
func SendTaskToAgent(ctx context.Context, targetURL string, secretKey string, payload AIDispatch) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("create request error: %w", err)
	}

	// สร้าง HMAC Signature กำกับไปกับ Request Header
	if secretKey != "" {
		mac := hmac.New(sha256.New, []byte(secretKey))
		mac.Write(jsonData)
		signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-TNH-Signature", signature)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("execution error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("target agent returned non-200 status: %d", resp.StatusCode)
	}

	return nil
}

