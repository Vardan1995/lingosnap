package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/go-vgo/robotgo"
	"github.com/joho/godotenv"
	hook "github.com/robotn/gohook"
	"google.golang.org/genai"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}
	
	if os.Getenv("GEMINI_API_KEY") == "" {
		log.Fatal("GEMINI_API_KEY environment variable is required")
	}

	log.Println("✅ Text Translator is running...")
	log.Println("   Usage: Select text, then press and release Right Shift")
	log.Println("   The text will be automatically translated and pasted")
	if runtime.GOOS == "darwin" {
		log.Println("   Note: On macOS, you may need to grant accessibility permissions")
	}
	log.Println("   Press Ctrl+C to exit")

	// Register Right Shift key release event
	hook.Register(hook.KeyUp, []string{"rshift"}, func(e hook.Event) {
		log.Println("▶ Right Shift detected - processing selected text...")
		go processSelectedText()
	})

	s := hook.Start()
	<-hook.Process(s)
}

func processSelectedText() {
	// Copy selected text to clipboard
	copyToClipboard()
	time.Sleep(200 * time.Millisecond)

	originalText, err := clipboard.ReadAll()
	if err != nil {
		log.Printf("❌ Failed to read clipboard: %v", err)
		return
	}

	if strings.TrimSpace(originalText) == "" {
		log.Println("⚠️  No text selected")
		return
	}

	log.Printf("   Original: %s", truncateText(originalText, 50))

	correctedText, err := translateWithGemini(originalText)
	if err != nil {
		log.Printf("❌ Translation failed: %v", err)
		return
	}

	if err := clipboard.WriteAll(correctedText); err != nil {
		log.Printf("❌ Failed to write to clipboard: %v", err)
		return
	}

	time.Sleep(100 * time.Millisecond)
	pasteFromClipboard()

	log.Printf("   Corrected: %s", truncateText(correctedText, 50))
	log.Println("✅ Text translated and pasted successfully")
	clipboard.WriteAll("")
}

func translateWithGemini(text string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Create genai client - gets API key from GEMINI_API_KEY env var
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create client: %w", err)
	}

	prompt := fmt.Sprintf(`Translate this text to English and fix any grammar or spelling errors. 
If the text is already in English, just correct any errors. 
If it's in Armenian (including transliterated Armenian), translate to English.
Return only the corrected/translated text without any additional comments or explanations:

%s`, text)

	result, err := client.Models.GenerateContent(
		ctx,
		"gemini-2.0-flash",
		genai.Text(prompt),
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("generation failed: %w", err)
	}

	return strings.TrimSpace(result.Text()), nil
}

// copyToClipboard handles OS-specific copy shortcuts
func copyToClipboard() {
	if runtime.GOOS == "darwin" {
		robotgo.KeyTap("c", "cmd") // macOS uses Cmd+C
	} else {
		robotgo.KeyTap("c", "ctrl") // Windows/Linux use Ctrl+C
	}
}

// pasteFromClipboard handles OS-specific paste shortcuts  
func pasteFromClipboard() {
	if runtime.GOOS == "darwin" {
		robotgo.KeyTap("v", "cmd") // macOS uses Cmd+V
	} else {
		robotgo.KeyTap("v", "ctrl") // Windows/Linux use Ctrl+V
	}
}

func truncateText(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
