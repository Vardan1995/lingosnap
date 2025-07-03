// Gemini Translator with Enhanced GUI (Fyne v2)

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/atotto/clipboard"
	"github.com/go-vgo/robotgo"
	hook "github.com/robotn/gohook"
	"google.golang.org/genai"
)

type Prompt struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

type Config struct {
	APIKey         string   `json:"api_key"`
	Model          string   `json:"model"`
	Hotkey         string   `json:"hotkey"`
	Prompts        []Prompt `json:"prompts"`
	SelectedPrompt string   `json:"selected_prompt"`
}

const defaultPromptTitle = "Default"
const defaultPromptText = `Translate this text to English and fix any grammar or spelling errors.
If the text is already in English, just correct any errors.
If it's in Armenian (including transliterated Armenian), translate to English.
Return only the corrected/translated text without any additional comments or explanations.`

var hotkeyOptions = []string{"rshift", "ctrl+alt+x", "alt+z", "ctrl+alt+space"}

type TranslatorApp struct {
	app            fyne.App
	window         fyne.Window
	config         Config
	hotkeyStopChan chan struct{}
	hotkeyMutex    sync.Mutex
	selectedIndex  int
	promptList     *fyne.Container
}

func main() {
	a := app.NewWithID("com.gemini.translator")
	w := a.NewWindow("Gemini Translator Settings")
	tApp := &TranslatorApp{
		app:            a,
		window:         w,
		hotkeyStopChan: make(chan struct{}),
	}
	tApp.loadConfig()
	w.SetContent(tApp.buildUI())
	w.Resize(fyne.NewSize(800, 600))
	go tApp.runHotkeyListener()
	w.ShowAndRun()
}

func (t *TranslatorApp) buildUI() fyne.CanvasObject {
	apiEntry := widget.NewPasswordEntry()
	apiEntry.SetText(t.config.APIKey)
	apiEntry.OnChanged = func(s string) {
		t.config.APIKey = s
		t.saveConfig()
	}

	modelSelect := widget.NewSelect([]string{"gemini-2.5-flash", "gemini-2.5-pro"}, func(s string) {
		t.config.Model = s
		t.saveConfig()
	})
	modelSelect.SetSelected(t.config.Model)

	hotkeySelect := widget.NewSelect(hotkeyOptions, func(s string) {
		t.config.Hotkey = s
		t.saveConfig()
		t.restartHotkeyListener()
	})
	hotkeySelect.SetSelected(t.config.Hotkey)

	t.promptList = container.NewVBox()
	t.refreshPromptList()

	addBtn := widget.NewButton("Add Prompt", func() {
		t.showPromptEditor(Prompt{"", ""}, -1)
	})

	form := container.NewVBox(
		widget.NewLabel("Gemini API Key"), apiEntry,
		widget.NewLabel("AI Model"), modelSelect,
		widget.NewLabel("Hotkey"), hotkeySelect,
		widget.NewLabelWithStyle("Prompt List", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		t.promptList,
		container.NewCenter(addBtn),
	)

	return container.NewVScroll(form)
}

func (t *TranslatorApp) refreshPromptList() {
	t.promptList.Objects = nil

	// Default prompt (non-deletable)
	defaultItem := container.NewHBox(
		widget.NewLabel(defaultPromptTitle),
		layout.NewSpacer(),
		widget.NewButton("View", func() {
			dialog.NewInformation("Prompt", defaultPromptText, t.window).Show()
		}),
	)
	t.promptList.Add(defaultItem)

	for i, p := range t.config.Prompts {
		index := i // capture index
		item := container.NewHBox(
			widget.NewLabel(p.Title),
			layout.NewSpacer(),
			widget.NewButton("Edit", func() {
				t.showPromptEditor(p, index)
			}),
			widget.NewButton("Delete", func() {
				t.config.Prompts = append(t.config.Prompts[:index], t.config.Prompts[index+1:]...)
				t.saveConfig()
				t.refreshPromptList()
			}),
		)
		t.promptList.Add(item)
	}

	t.window.Content().Refresh()
}

func (t *TranslatorApp) showPromptEditor(p Prompt, index int) {
	title := widget.NewEntry()
	title.SetText(p.Title)
	body := widget.NewMultiLineEntry()
	body.SetText(p.Text)

	d := dialog.NewForm("Edit Prompt", "Save", "Cancel",
		[]*widget.FormItem{
			widget.NewFormItem("Title", title),
			widget.NewFormItem("Prompt Text", body),
		},
		func(ok bool) {
			if !ok {
				return
			}
			newPrompt := Prompt{Title: title.Text, Text: body.Text}
			if newPrompt.Title == "" || newPrompt.Text == "" {
				dialog.NewInformation("Error", "Title and Prompt Text are required.", t.window).Show()
				return
			}
			if index >= 0 {
				t.config.Prompts[index] = newPrompt
			} else {
				t.config.Prompts = append(t.config.Prompts, newPrompt)
			}
			t.saveConfig()
			t.refreshPromptList()
		}, t.window)
	d.Resize(fyne.NewSize(600, 400))
	d.Show()
}

func (t *TranslatorApp) configPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}
	return filepath.Join(dir, "gemini-translator-settings.json")
}

func (t *TranslatorApp) saveConfig() {
	data, _ := json.MarshalIndent(t.config, "", "  ")
	_ = os.WriteFile(t.configPath(), data, 0644)
}

func (t *TranslatorApp) loadConfig() {
	f, err := os.ReadFile(t.configPath())
	if err != nil || len(f) == 0 {
		t.config = Config{
			APIKey:         os.Getenv("GEMINI_API_KEY"),
			Model:          "gemini-2.5-flash",
			Hotkey:         "rshift",
			Prompts:        []Prompt{},
			SelectedPrompt: defaultPromptText,
		}
		t.saveConfig()
		return
	}
	_ = json.Unmarshal(f, &t.config)
}

func (t *TranslatorApp) restartHotkeyListener() {
	t.hotkeyMutex.Lock()
	defer t.hotkeyMutex.Unlock()
	if t.hotkeyStopChan != nil {
		close(t.hotkeyStopChan)
	}
	t.hotkeyStopChan = make(chan struct{})
	go t.runHotkeyListener()
}

func (t *TranslatorApp) runHotkeyListener() {
	t.hotkeyMutex.Lock()
	hotkey := t.config.Hotkey
	stop := t.hotkeyStopChan
	t.hotkeyMutex.Unlock()

	keys := strings.Split(hotkey, "+")
	hook.Register(hook.KeyUp, keys, func(e hook.Event) {
		go t.processSelectedText()
	})
	s := hook.Start()
	select {
	case <-stop:
		hook.StopEvent()
	case <-hook.Process(s):
	}
}

func (t *TranslatorApp) processSelectedText() {
	prev, _ := clipboard.ReadAll()
	copyToClipboard()
	time.Sleep(100 * time.Millisecond)
	text, _ := clipboard.ReadAll()
	if strings.TrimSpace(text) == "" {
		restoreClipboard(prev)
		return
	}

	promptText := defaultPromptText
	if t.selectedIndex > 0 && t.selectedIndex-1 < len(t.config.Prompts) {
		promptText = t.config.Prompts[t.selectedIndex-1].Text
	}

	txt, err := translateWithGemini(t.config.APIKey, t.config.Model, promptText, text)
	if err == nil {
		clipboard.WriteAll(txt)
		time.Sleep(100 * time.Millisecond)
		pasteFromClipboard()
		time.Sleep(100 * time.Millisecond)
		restoreClipboard(prev)
	}
}

func translateWithGemini(apiKey, model, prompt, text string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: apiKey})
	if err != nil {
		return "", err
	}

	resp, err := client.Models.GenerateContent(ctx, model, genai.Text(fmt.Sprintf("%s\n\n%s", prompt, text)), nil)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resp.Text()), nil
}

func copyToClipboard() {
	if runtime.GOOS == "darwin" {
		robotgo.KeyTap("c", "cmd")
	} else {
		robotgo.KeyTap("c", "ctrl")
	}
}

func pasteFromClipboard() {
	if runtime.GOOS == "darwin" {
		robotgo.KeyTap("v", "cmd")
	} else {
		robotgo.KeyTap("v", "ctrl")
	}
}

func restoreClipboard(s string) {
	if s != "" {
		clipboard.WriteAll(s)
	}
}
