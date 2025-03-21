package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/getlantern/systray"
	"github.com/go-vgo/robotgo"
	"github.com/joho/godotenv"
	hook "github.com/robotn/gohook"
)

// API configuration
const (
	modelName         = "gemini-2.0-flash"
	translateEndpoint = "https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent"
)

// API request/response structures
type (
	request struct {
		Contents []content `json:"contents"`
	}

	content struct {
		Parts []part `json:"parts"`
	}

	part struct {
		Text string `json:"text"`
	}

	response struct {
		Candidates []struct {
			Content content `json:"content"`
		} `json:"candidates"`
	}
)

func main() {
	// Load API key from .env or environment
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY environment variable not set")
	}

	log.Println("Starting Text Translator")
	log.Println("Usage: Select text, then press and release Right Shift to translate")

	// Set up systray
	go systray.Run(onReady, onExit)

	// Set up keyboard hook
	hook.Register(hook.KeyUp, []string{"rshift"}, func(e hook.Event) {
		log.Println("Right Shift released - translating selected text")
		go func() {
			if err := processSelectedText(apiKey); err != nil {
				log.Printf("Translation error: %v", err)
			}
		}()
	})

	// Start event listener (blocks until terminated)
	s := hook.Start()
	<-hook.Process(s)
}

func onReady() {
	systray.SetTitle("Text Translator")
	systray.SetTooltip("Select text + Right Shift to translate")

	mQuit := systray.AddMenuItem("Quit", "Exit the application")
	go func() {
		<-mQuit.ClickedCh
		systray.Quit()
	}()
}

func onExit() {
	// Cleanup resources if needed
}

func processSelectedText(apiKey string) error {
	// Copy selected text to clipboard
	robotgo.KeyTap("c", "ctrl")
	time.Sleep(200 * time.Millisecond)

	// Get text from clipboard
	text, err := clipboard.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read clipboard: %w", err)
	}

	if text == "" {
		return fmt.Errorf("no text selected")
	}

	log.Printf("Selected text: %s", truncate(text, 50))

	// Translate the text
	translated, err := translateText(context.Background(), apiKey, text)
	if err != nil {
		return err
	}

	// Write translation to clipboard
	if err := clipboard.WriteAll(translated); err != nil {
		return fmt.Errorf("failed to write to clipboard: %w", err)
	}

	// Paste the translated text
	time.Sleep(100 * time.Millisecond)
	robotgo.KeyTap("v", "ctrl")

	log.Printf("Translated and pasted: %s", truncate(translated, 50))
	return nil
}

func translateText(ctx context.Context, apiKey, text string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Prepare request
	reqBody := request{
		Contents: []content{{
			Parts: []part{{
				Text: fmt.Sprintf("Translate the following text to standard English, correcting any grammar or spelling errors. If the text is in Armenian (including Latin transliteration), translate it to English. Reply only with the translation, no comments: %s", text),
			}},
		}},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create and send HTTP request
	url := fmt.Sprintf(translateEndpoint, modelName) + "?key=" + apiKey
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read and parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, truncate(string(body), 100))
	}

	var result response
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no translation result returned")
	}

	return strings.TrimSpace(result.Candidates[0].Content.Parts[0].Text), nil
}

// Helper function to truncate strings for logging
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
