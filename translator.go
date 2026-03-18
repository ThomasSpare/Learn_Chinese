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
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Chinese Translator</title>
    <style>
        * { box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Arial, sans-serif;
            max-width: 600px;
            margin: 0 auto;
            padding: 20px;
            background: #f5f5f5;
        }
        .container {
            background: white;
            padding: 20px;
            border-radius: 12px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        h1 {
            margin: 0 0 20px 0;
            font-size: 24px;
            color: #333;
            text-align: center;
        }
        textarea {
            width: 100%;
            min-height: 120px;
            margin: 10px 0;
            padding: 15px;
            font-size: 16px;
            border: 2px solid #ddd;
            border-radius: 8px;
            resize: vertical;
        }
        textarea:focus {
            outline: none;
            border-color: #4CAF50;
        }
        button {
            width: 100%;
            background: #4CAF50;
            color: white;
            padding: 15px 20px;
            border: none;
            border-radius: 8px;
            font-size: 16px;
            font-weight: 600;
            cursor: pointer;
            transition: background 0.3s;
        }
        button:hover { background: #45a049; }
        button:active { transform: scale(0.98); }
        .result {
            margin-top: 20px;
            padding: 20px;
            background: #f9f9f9;
            border-radius: 8px;
            border-left: 4px solid #4CAF50;
        }
        .result h3 {
            margin: 0 0 10px 0;
            font-size: 14px;
            color: #666;
            text-transform: uppercase;
            letter-spacing: 1px;
        }
        #translation {
            font-size: 32px;
            font-weight: bold;
            color: #333;
            margin: 10px 0;
            word-wrap: break-word;
        }
        #pinyin {
            color: #666;
            font-style: italic;
            font-size: 18px;
            margin: 10px 0;
        }
        #playBtn {
            margin-top: 15px;
            background: #2196F3;
        }
        #playBtn:hover {
            background: #1976D2;
        }
        canvas {
            width: 100%;
            height: 80px;
            margin: 15px 0;
            background: #f0f0f0;
            border-radius: 8px;
        }
        .waveform-fallback {
            width: 100%;
            height: 80px;
            margin: 15px 0;
            background: #f0f0f0;
            border-radius: 8px;
            display: none;
            align-items: center;
            justify-content: center;
            position: relative;
            overflow: hidden;
        }
        .waveform-fallback.active {
            display: flex;
        }
        .wave-line {
            width: 90%;
            height: 4px;
            background: #4CAF50;
            position: relative;
            border-radius: 2px;
            animation: vibrate 0.1s ease-in-out infinite;
            box-shadow: 0 0 15px rgba(76, 175, 80, 0.8);
        }
        .wave-line::before,
        .wave-line::after {
            content: '';
            position: absolute;
            width: 100%;
            height: 4px;
            background: #4CAF50;
            opacity: 0.5;
            border-radius: 2px;
        }
        .wave-line::before {
            top: -8px;
            animation: vibrate 0.15s ease-in-out infinite;
        }
        .wave-line::after {
            top: 8px;
            animation: vibrate 0.12s ease-in-out infinite reverse;
        }
        @keyframes vibrate {
            0% { transform: translateY(0) scaleY(1); }
            25% { transform: translateY(-3px) scaleY(1.2); }
            50% { transform: translateY(0) scaleY(0.8); }
            75% { transform: translateY(3px) scaleY(1.2); }
            100% { transform: translateY(0) scaleY(1); }
        }
        .word-by-word {
            margin-top: 15px;
            padding: 15px;
            background: #e8f5e9;
            border-radius: 8px;
        }
        .word-display {
            font-size: 48px;
            font-weight: bold;
            text-align: center;
            margin: 20px 0;
            min-height: 80px;
            color: #2196F3;
        }
        .word-pinyin {
            font-size: 24px;
            text-align: center;
            color: #666;
            font-style: italic;
            margin-bottom: 20px;
        }
        #nextWordBtn {
            background: #FF9800;
            margin-top: 10px;
        }
        #nextWordBtn:hover {
            background: #F57C00;
        }
        #toggleWordMode {
            background: #9C27B0;
            margin-top: 10px;
        }
        #toggleWordMode:hover {
            background: #7B1FA2;
        }
        @media (max-width: 640px) {
            body { padding: 10px; }
            .container { padding: 15px; }
            h1 { font-size: 20px; }
            #translation { font-size: 28px; }
            #pinyin { font-size: 16px; }
            .word-display { font-size: 36px; }
            .word-pinyin { font-size: 20px; }
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🇨🇳 Chinese Translator</h1>
        <textarea id="input" placeholder="Enter English text..."></textarea>
        <button id="translateBtn">Translate & Pronounce</button>
        <div id="result" class="result" style="display:none;">
            <h3>Translation:</h3>
            <p id="translation"></p>
            <p id="pinyin"></p>
            <canvas id="waveform"></canvas>
            <div id="waveformFallback" class="waveform-fallback">
                <div class="wave-line"></div>
            </div>
            <div id="debugMsg" style="font-size: 12px; color: #999; margin: 10px 0;"></div>
            <button id="playBtn">🔊 Play Pronunciation</button>
            <button id="toggleWordMode">📖 Word-by-Word Mode</button>
            <div id="wordByWord" class="word-by-word" style="display:none;">
                <div class="word-display" id="currentWord"></div>
                <div class="word-pinyin" id="currentPinyin"></div>
                <button id="nextWordBtn">Next Word →</button>
            </div>
        </div>
    </div>
    <audio id="audio" style="display:none;"></audio>
    <audio id="wordAudio" style="display:none;"></audio>

    <script>
        let audioUrl = '';
        let translationData = null;
        let audioContext = null;
        let analyser = null;
        let animationId = null;
        let audioSource = null;
        let isAudioConnected = false;

        // Word-by-word mode
        let chineseChars = [];
        let pinyinWords = [];
        let currentWordIndex = 0;

        // Unlock audio context on first user interaction (critical for mobile)
        async function unlockAudio() {
            if (!audioContext) {
                audioContext = new (window.AudioContext || window.webkitAudioContext)();
                analyser = audioContext.createAnalyser();
                analyser.fftSize = 256;
            }
            if (audioContext.state === 'suspended') {
                await audioContext.resume();
            }
        }

        // Listen for first touch/click to unlock audio
        document.addEventListener('touchstart', unlockAudio, { once: true, passive: true });
        document.addEventListener('click', unlockAudio, { once: true, passive: true });

        async function translate() {
            const text = document.getElementById('input').value;
            if (!text) return;

            const response = await fetch('/translate', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ text: text })
            });

            const data = await response.json();
            translationData = data;
            document.getElementById('translation').textContent = data.translation;
            document.getElementById('pinyin').textContent = data.pinyin ? 'Pronunciation: ' + data.pinyin : '';
            document.getElementById('result').style.display = 'block';
            audioUrl = data.audioUrl;

            // Prepare word-by-word data
            chineseChars = data.translation.split('');
            pinyinWords = data.pinyin ? data.pinyin.split(' ') : [];
            currentWordIndex = 0;

            // Reset word mode
            document.getElementById('wordByWord').style.display = 'none';
        }


        function drawWaveform() {
            const canvas = document.getElementById('waveform');
            if (!canvas) {
                console.error('Canvas not found');
                return;
            }

            const ctx = canvas.getContext('2d');
            if (!ctx) {
                console.error('Canvas context is null');
                return;
            }

            // Set canvas size properly
            const rect = canvas.getBoundingClientRect();
            if (rect.width === 0) {
                console.error('Canvas has zero width');
                return;
            }
            canvas.width = rect.width;
            canvas.height = 80;

            if (!analyser) {
                console.error('Analyser is null');
                return;
            }

            const bufferLength = analyser.frequencyBinCount;
            const dataArray = new Uint8Array(bufferLength);

            console.log('Starting waveform draw loop');

            function draw() {
                if (!analyser || !ctx || !canvas) return;

                animationId = requestAnimationFrame(draw);
                analyser.getByteFrequencyData(dataArray);

                // Clear canvas
                ctx.fillStyle = '#f0f0f0';
                ctx.fillRect(0, 0, canvas.width, canvas.height);

                // Draw bars
                const barWidth = (canvas.width / bufferLength) * 2.5;
                let x = 0;
                let hasData = false;

                for (let i = 0; i < bufferLength; i++) {
                    const barHeight = Math.max((dataArray[i] / 255) * canvas.height * 0.8, 5); // Minimum 5px height

                    if (dataArray[i] > 0) hasData = true;

                    // Create gradient for each bar
                    const gradient = ctx.createLinearGradient(0, canvas.height - barHeight, 0, canvas.height);
                    gradient.addColorStop(0, '#4CAF50');
                    gradient.addColorStop(1, '#81C784');

                    ctx.fillStyle = gradient;
                    ctx.fillRect(x, canvas.height - barHeight, barWidth - 1, barHeight);
                    x += barWidth;
                }

                // Debug: show if we're getting data
                if (!hasData) {
                    ctx.fillStyle = 'red';
                    ctx.font = '12px Arial';
                    ctx.fillText('No audio data', 10, 20);
                }
            }

            draw();
        }

        async function playAudio() {
            const audio = document.getElementById('audio');
            const canvas = document.getElementById('waveform');
            const fallback = document.getElementById('waveformFallback');
            const debug = document.getElementById('debugMsg');

            debug.textContent = 'Initializing audio...';

            // CRITICAL: Unlock/resume AudioContext INSIDE the gesture handler (double-resume pattern)
            await unlockAudio();
            debug.textContent = 'AudioContext unlocked';

            // Resume again right before play (iOS quirk)
            if (audioContext && audioContext.state === 'suspended') {
                await audioContext.resume();
                debug.textContent = 'AudioContext resumed (double-resume)';
            }

            // Stop previous animation
            if (animationId) {
                cancelAnimationFrame(animationId);
            }

            // Set audio source
            audio.src = audioUrl;

            // Try to connect Web Audio API on first play only
            if (!isAudioConnected && audioContext) {
                try {
                    // Load audio first for iOS
                    audio.load();
                    // Wait for iOS to register the audio element
                    await new Promise(resolve => setTimeout(resolve, 100));

                    audioSource = audioContext.createMediaElementSource(audio);
                    audioSource.connect(analyser);
                    analyser.connect(audioContext.destination);
                    isAudioConnected = true;
                    fallback.style.display = 'none'; // Hide CSS fallback
                    debug.textContent = '✓ Web Audio API connected!';
                    console.log('Media source created and connected');
                } catch (e) {
                    debug.textContent = '⚠️ Using CSS fallback: ' + e.message;
                    console.error('Web Audio connection failed:', e);
                    // Ensure fallback shows
                    fallback.style.display = 'flex';
                    fallback.classList.add('active');
                }
            }

            try {
                await audio.play();
                console.log('Audio playing');

                // Draw a test pattern first to verify canvas works
                const testCanvas = document.getElementById('waveform');
                const testCtx = testCanvas.getContext('2d');
                if (testCtx) {
                    testCanvas.width = testCanvas.offsetWidth || 300;
                    testCanvas.height = 80;
                    testCtx.fillStyle = '#4CAF50';
                    testCtx.fillRect(0, 0, testCanvas.width, testCanvas.height);
                    testCtx.fillStyle = 'white';
                    testCtx.font = '20px Arial';
                    testCtx.fillText('CANVAS TEST', 10, 40);
                    console.log('Canvas test pattern drawn');
                }

                // Start real-time waveform if Web Audio is connected
                if (analyser && isAudioConnected) {
                    setTimeout(() => drawWaveform(), 100); // Small delay
                    console.log('Drawing waveform');
                } else {
                    // Use CSS fallback
                    fallback.style.display = 'flex';
                    fallback.classList.add('active');
                    console.log('Using CSS fallback');
                }
            } catch (e) {
                console.error('Audio play failed:', e);
            }

            // Clean up when audio ends
            audio.onended = () => {
                console.log('Audio ended');
                if (animationId) {
                    cancelAnimationFrame(animationId);
                }
                if (canvas) {
                    const ctx = canvas.getContext('2d');
                    ctx.fillStyle = '#f0f0f0';
                    ctx.fillRect(0, 0, canvas.width, canvas.height);
                }
                fallback.classList.remove('active');
            };
        }

        function toggleWordMode() {
            const wordByWord = document.getElementById('wordByWord');
            if (wordByWord.style.display === 'none') {
                wordByWord.style.display = 'block';
                currentWordIndex = 0;
                showCurrentWord();
            } else {
                wordByWord.style.display = 'none';
            }
        }

        function showCurrentWord() {
            if (currentWordIndex >= chineseChars.length) {
                document.getElementById('currentWord').textContent = '✓ Complete!';
                document.getElementById('currentPinyin').textContent = '';
                return;
            }

            const char = chineseChars[currentWordIndex];
            const pinyin = pinyinWords[currentWordIndex] || '';

            document.getElementById('currentWord').textContent = char;
            document.getElementById('currentPinyin').textContent = pinyin;

            // Auto-play pronunciation for this character
            playCharacterAudio(char);
        }

        async function playCharacterAudio(char) {
            const audio = document.getElementById('wordAudio');
            const charUrl = '/pronounce?text=' + encodeURIComponent(char) + '&lang=zh-CN';
            audio.src = charUrl;

            // Ensure we wait for the audio to be ready
            try {
                await audio.play();
            } catch (e) {
                console.log('Audio play failed:', e);
            }
        }

        function nextWord() {
            currentWordIndex++;
            showCurrentWord();
        }

        // Add event listeners after DOM loads
        document.getElementById('translateBtn').addEventListener('click', translate);
        document.getElementById('playBtn').addEventListener('click', playAudio);
        document.getElementById('toggleWordMode').addEventListener('click', toggleWordMode);
        document.getElementById('nextWordBtn').addEventListener('click', nextWord);
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, html)
}
