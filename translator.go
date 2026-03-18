package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/mozillazg/go-pinyin"
)

// TranslateRequest represents the request body for translation
type TranslateRequest struct {
	Text string `json:"text"`
	From string `json:"from"` // source language (default: "en")
	To   string `json:"to"`   // target language (default: "zh")
}

// TranslateResponse represents the translation API response
type TranslateResponse struct {
	Translation  string `json:"translation"`
	Original     string `json:"original"`
	Pinyin       string `json:"pinyin"`
	AudioURL     string `json:"audioUrl"`
}

// translateHandler handles translation requests
func translateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TranslateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Text == "" {
		http.Error(w, "Text field is required", http.StatusBadRequest)
		return
	}

	// Set defaults
	if req.From == "" {
		req.From = "en"
	}
	if req.To == "" {
		req.To = "zh-CN"
	}

	// Translate using MyMemory API (free, no API key required)
	translatedText, err := translateText(req.Text, req.From, req.To)
	if err != nil {
		http.Error(w, fmt.Sprintf("Translation failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Get pinyin pronunciation
	pinyin, _ := getPinyin(translatedText)

	// Generate Google TTS URL for pronunciation
	audioURL := generateTTSURL(translatedText, req.To)

	response := TranslateResponse{
		Translation: translatedText,
		Original:    req.Text,
		Pinyin:      pinyin,
		AudioURL:    audioURL,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// translateText uses MyMemory translation API
func translateText(text, from, to string) (string, error) {
	baseURL := "https://api.mymemory.translated.net/get"
	params := url.Values{}
	params.Add("q", text)
	params.Add("langpair", fmt.Sprintf("%s|%s", from, to))

	resp, err := http.Get(baseURL + "?" + params.Encode())
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result struct {
		ResponseData struct {
			TranslatedText string `json:"translatedText"`
		} `json:"responseData"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	return result.ResponseData.TranslatedText, nil
}

// getPinyin converts Chinese characters to Pinyin
func getPinyin(text string) (string, error) {
	// Use go-pinyin library to convert Chinese to pinyin with tone marks
	args := pinyin.NewArgs()
	args.Style = pinyin.Tone // Use tone marks (nǐ hǎo)

	result := pinyin.Pinyin(text, args)

	// Join all pinyin syllables with spaces
	var pinyinWords []string
	for _, syllables := range result {
		if len(syllables) > 0 {
			pinyinWords = append(pinyinWords, syllables[0])
		}
	}

	return strings.Join(pinyinWords, " "), nil
}

// pronounceHandler serves audio pronunciation
func pronounceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	text := r.URL.Query().Get("text")
	lang := r.URL.Query().Get("lang")

	if text == "" {
		http.Error(w, "Text parameter is required", http.StatusBadRequest)
		return
	}

	if lang == "" {
		lang = "zh-CN"
	}

	// Fetch audio from Google TTS
	audioData, err := fetchTTSAudio(text, lang)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate audio: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Content-Disposition", "inline; filename=\"pronunciation.mp3\"")
	w.Write(audioData)
}

// generateTTSURL creates a URL to our own pronounce endpoint
func generateTTSURL(text, lang string) string {
	params := url.Values{}
	params.Add("text", text)
	params.Add("lang", lang)
	return "/pronounce?" + params.Encode()
}

// fetchTTSAudio downloads audio from Google TTS
func fetchTTSAudio(text, lang string) ([]byte, error) {
	baseURL := "https://translate.google.com/translate_tts"
	params := url.Values{}
	params.Add("ie", "UTF-8")
	params.Add("tl", lang)
	params.Add("client", "tw-ob")
	params.Add("q", text)
	ttsURL := baseURL + "?" + params.Encode()

	client := &http.Client{}
	req, err := http.NewRequest("GET", ttsURL, nil)
	if err != nil {
		return nil, err
	}

	// Add User-Agent to avoid blocking
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TTS API returned status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// demoHandler serves a simple demo page
func demoHandler(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Chinese Translator</title>
    <style>
        body { font-family: , sans-serif; max-width: 600px; margin: 50px auto; padding: 20px; background: #3209ba; border-radius: 10px; }
        textarea { width: 100%; height: 100px; margin: 10px 0; padding: 10px; }
        button { background: #4CAF50; color: white; padding: 10px 20px; border: none; cursor: pointer; }
        button:hover { background: #45a049; }
        .result { margin-top: 20px; padding: 15px; background: #f0f0f0; border-radius: 5px; }
    </style>
</head>
<body>
    <h1>Chinese Translator</h1>
    <textarea id="input" placeholder="Enter English text..."></textarea>
    <button id="translateBtn">Translate & Pronounce</button>
    <div id="result" class="result" style="display:none;">
        <h3>Translation:</h3>
        <p id="translation" style="font-size: 34px; font-weight: bold;"></p>
        <p id="pinyin" style="color: #666; font-style: italic;"></p>
        <button id="playBtn">🔊 Play Pronunciation</button>
    </div>
    <audio id="audio" style="display:none;"></audio>

    <script>
        let audioUrl = '';

        async function translate() {
            const text = document.getElementById('input').value;
            if (!text) return;

            const response = await fetch('/translate', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ text: text })
            });

            const data = await response.json();
            document.getElementById('translation').textContent = data.translation;
            document.getElementById('pinyin').textContent = data.pinyin ? 'Pronunciation: ' + data.pinyin : '';
            document.getElementById('result').style.display = 'block';
            audioUrl = data.audioUrl;
        }

        function playAudio() {
            const audio = document.getElementById('audio');
            audio.src = audioUrl;
            audio.play();
        }

        // Add event listeners after DOM loads
        document.getElementById('translateBtn').addEventListener('click', translate);
        document.getElementById('playBtn').addEventListener('click', playAudio);
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, html)
}
