package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/line/line-bot-sdk-go/v7/linebot"
)

// --- 💎 โครงสร้าง Gemini 3 Flash ---
type GeminiRequest struct {
	Contents []struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"contents"`
}

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

type Mission struct {
	Platform   string
	ReplyToken string
	Text       string
	UserID     string
	Timestamp  time.Time
}

type ThitNueaHub struct {
	bot       *linebot.Client
	db        *firestore.Client
	missionCh chan Mission
	secret    string
	wg        sync.WaitGroup
}

type DiscordPayload struct {
	Content  string `json:"content"`
	Username string `json:"username,omitempty"`
	Avatar   string `json:"avatar_url,omitempty"`
}

// --- 🚀 ยิงรายงานเข้าห้องบัญชาการ Discord ---
func sendToDiscord(message string, agentName string) {
	webhookURL := os.Getenv("DISCORD_WEBHOOK_URL")
	if webhookURL == "" {
		return
	}
	payload := DiscordPayload{
		Content:  message,
		Username: agentName,
		Avatar:   "https://cdn-icons-png.flaticon.com/512/4712/4712109.png",
	}
	jsonData, _ := json.Marshal(payload)
	_, _ = http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
}

// --- ⚡ หัวใจ Gemini 3: แกะเจ้าตาก ---
func askGemini(prompt string) string {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return "⚠️ กุญแจหาย! กรุณาเช็ก Secret Manager"
	}
	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=" + apiKey

	payload := GeminiRequest{}
	payload.Contents = append(payload.Contents, struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	}{})
	payload.Contents[0].Parts = append(payload.Contents[0].Parts, struct {
		Text string `json:"text"`
	}{Text: "จงแกะเจ้าตากจากข้อความนี้: " + prompt})

	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "❌ เชื่อมต่อ Gemini พลาด"
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var geminiResp GeminiResponse
	_ = json.Unmarshal(body, &geminiResp)

	if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
		return geminiResp.Candidates[0].Content.Parts[0].Text
	}
	return "💎 แก้วตา: กำลังประมวลผลแรงเกินไป หรือ API Key มีปัญหาค่ะ"
}

func main() {
	log.Println("🐅 [ทิศเหนือ ฮับ V4]: IGNITE - Full Power Online...")
	
	// 🔒 บังคับสับสวิตช์ล็อกเลนเข้าพอร์ตเดี่ยว 2026 ของมหาจักรวรรดิเพื่อความกริบ
	port := "2026"
	ctx := context.Background()

	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	dbClient, _ := firestore.NewClient(ctx, projectID)

	hub := &ThitNueaHub{
		db:        dbClient,
		missionCh: make(chan Mission, 10000), // 🔥 คิว 10,000 รองรับงานหนัก
		secret:    os.Getenv("LINE_CHANNEL_SECRET"),
	}

	lineToken := os.Getenv("LINE_CHANNEL_ACCESS_TOKEN")
	hub.bot, _ = linebot.New(hub.secret, lineToken)

	sendToDiscord("🚀 **[SYSTEM REIGNITE]** F-16 V4 F-16 พร้อมถลุงงบระบบปิดแล้วเจ้านาย!", "🐅 ทิศเหนือ ฮับ (Core V4)")

	// ปล่อยคนงาน ไอ้จอร์จ 15 คน (Zero-Garbage Worker Pool)
	for i := 1; i <= 15; i++ {
		hub.wg.Add(1)
		go hub.GeorgeWorker(ctx, i)
	}

	// ล็อกเส้นทางเชื่อมต่อระบบ (Endpoints)
	http.HandleFunc("/webhook/line", hub.PhraiThongLine)
	http.HandleFunc("/api/surgery", hub.NamIngSurgeryHandler)
	http.HandleFunc("/api/v4/status", hub.V4LiveStatusHandler) // ท่อสเตตัสพ่นไฟออกหน้าบ้าน V3
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, "<h1>✅ F-16 DEFENDER V4: Active & Heavy Loaded over Port 2026</h1>")
	})

	fmt.Printf("👑 THITNUEA EMPIRE V4 | 🛩️ F-16 ONLINE | Sovereign Port: %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func (h *ThitNueaHub) PhraiThongLine(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	hash := hmac.New(sha256.New, []byte(h.secret))
	hash.Write(body)
	sig := r.Header.Get("X-Line-Signature")
	if base64.StdEncoding.EncodeToString(hash.Sum(nil)) != sig {
		http.Error(w, "Unauthorized", 401)
		return
	}
	r.Body = io.NopCloser(strings.NewReader(string(body)))
	events, _ := h.bot.ParseRequest(r)
	for _, event := range events {
		if event.Type == linebot.EventTypeMessage {
			if msg, ok := event.Message
			
