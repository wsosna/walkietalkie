const messageInput = document.getElementById('message-input');
const sendButton = document.getElementById('send-button');
const responseArea = document.getElementById('response-area');
const recordButton = document.getElementById('record-button');
const languageSelector = document.getElementById('language-selector');
const recordingOverlay = document.getElementById('recording-overlay');
const waveform = document.getElementById('waveform');

let mediaRecorder = null;
let audioChunks = [];
let isRecording = false;
let isCanceled = false;
let recordStartX = 0;
let recordStartY = 0;
const CANCEL_THRESHOLD = 100; // pixels to drag left before canceling

// Cookie helper functions
function setCookie(name, value, days = 7) {
    const expires = new Date();
    expires.setTime(expires.getTime() + days * 24 * 60 * 60 * 1000);
    document.cookie = `${name}=${value};expires=${expires.toUTCString()};path=/`;
}

function getCookie(name) {
    const nameEQ = name + "=";
    const cookies = document.cookie.split(';');
    for (let i = 0; i < cookies.length; i++) {
        let cookie = cookies[i].trim();
        if (cookie.indexOf(nameEQ) === 0) {
            return cookie.substring(nameEQ.length);
        }
    }
    return null;
}

function deleteCookie(name) {
    document.cookie = `${name}=;expires=Thu, 01 Jan 1970 00:00:00 UTC;path=/`;
}

// Get cookie name for current language
function previousResponseIdByCurrLanguage() {
    return 'previousResponseId-' + languageSelector.value;
}

// Play audio response from base64 data
function playAudioResponse(base64Audio) {
    try {
        // Decode base64 to binary
        const binaryString = atob(base64Audio);
        const bytes = new Uint8Array(binaryString.length);
        for (let i = 0; i < binaryString.length; i++) {
            bytes[i] = binaryString.charCodeAt(i);
        }

        // Create blob from binary data
        const audioBlob = new Blob([bytes], { type: 'audio/mpeg' });
        const audioUrl = URL.createObjectURL(audioBlob);

        // Create and play audio element
        const audio = new Audio(audioUrl);

        // Clean up the URL after playing
        audio.onended = () => {
            URL.revokeObjectURL(audioUrl);
        };

        // Handle playback errors
        audio.onerror = (error) => {
            console.error('Error playing audio:', error);
        };

        // Play the audio
        audio.play().catch(error => {
            console.error('Failed to play audio:', error);
            // Note: Autoplay might be blocked by browser policies
            // In such cases, user needs to interact with the page first
        });
    } catch (error) {
        console.error('Error decoding audio:', error);
    }
}

// Send text message
async function sendMessage() {
    const message = messageInput.value.trim();
    if (!message) return;

    // Disable input while processing
    sendButton.disabled = true;
    messageInput.disabled = true;

    // Show loading state
    responseArea.innerHTML = '<p class="loading">Thinking...</p>';

    try {
        const requestBody = {
            message: message,
            language: languageSelector.value
        };

        // Get previousResponseId from cookie
        const previousResponseId = getCookie(previousResponseIdByCurrLanguage());
        if (previousResponseId) {
            requestBody.previousResponseId = previousResponseId;
        }

        const response = await fetch('/chat', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(requestBody),
        });

        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        const data = await response.json();

        // Store the response ID in cookie for next request
        if (data.responseId) {
            setCookie(previousResponseIdByCurrLanguage(), data.responseId);
        }

        // Display the response
        responseArea.innerHTML = `<p>${data.reply}</p>`;

        // Play audio response if available
        if (data.audioData) {
            playAudioResponse(data.audioData);
        }

        // Clear input
        messageInput.value = '';
    } catch (error) {
        responseArea.innerHTML = `<p class="error">Error: ${error.message}</p>`;
        console.error('Error:', error);
    } finally {
        // Re-enable input
        sendButton.disabled = false;
        messageInput.disabled = false;
        messageInput.focus();
    }
}

// Voice recording functions
async function startRecording() {
    console.log('startRecording called, isRecording:', isRecording);

    if (isRecording) {
        console.log('Already recording, returning');
        return;
    }

    console.log('Requesting microphone access...');

    try {
        const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
        console.log('Microphone access granted');

        // Choose the best supported audio format
        const mimeTypes = [
            'audio/webm;codecs=opus',
            'audio/webm',
            'audio/ogg;codecs=opus',
            'audio/mp4'
        ];

        const mimeType = mimeTypes.find(type => MediaRecorder.isTypeSupported(type)) || 'audio/webm';

        mediaRecorder = new MediaRecorder(stream, { mimeType });
        audioChunks = [];
        isCanceled = false; // Reset cancel state

        mediaRecorder.addEventListener('dataavailable', event => {
            audioChunks.push(event.data);
        });

        mediaRecorder.addEventListener('stop', async () => {
            // Hide overlay and reset button state
            recordingOverlay.classList.add('hidden');
            recordButton.classList.remove('recording');
            recordButton.classList.remove('canceling');
            waveform.classList.add('hidden');

            // Only send if not canceled
            if (!isCanceled) {
                const audioBlob = new Blob(audioChunks, { type: mimeType });
                await sendAudioMessage(audioBlob, mimeType);
            } else {
                // Show canceled message briefly
                responseArea.innerHTML = '<p style="color: #666; font-style: italic;">Recording canceled</p>';
                setTimeout(() => {
                    if (responseArea.innerHTML.includes('Recording canceled')) {
                        responseArea.innerHTML = '<p class="placeholder">Your response will appear here...</p>';
                    }
                }, 1500);
            }

            // Stop all tracks to release the microphone
            stream.getTracks().forEach(track => track.stop());
        });

        mediaRecorder.start();
        isRecording = true;

        // Show overlay and enlarge original button
        recordingOverlay.classList.remove('hidden');
        recordButton.classList.add('recording');
        waveform.classList.remove('hidden');

    } catch (error) {
        console.error('Error accessing microphone:', error);
        responseArea.innerHTML = '<p class="error">Error: Could not access microphone. Please grant permission.</p>';
    }
}

function stopRecording() {
    if (!isRecording || !mediaRecorder) return;

    console.log('Recording stopped', 'canceled:', isCanceled);

    mediaRecorder.stop();
    isRecording = false;
}

async function sendAudioMessage(audioBlob, mimeType) {
    // Disable buttons while processing
    recordButton.disabled = true;
    sendButton.disabled = true;
    messageInput.disabled = true;

    // Show loading state
    responseArea.innerHTML = '<p class="loading">Transcribing and processing...</p>';

    try {
        const formData = new FormData();

        // Determine file extension from mimeType
        let extension = '.webm';
        if (mimeType.includes('ogg')) extension = '.ogg';
        else if (mimeType.includes('mp4')) extension = '.mp4';

        formData.append('audio', audioBlob, `recording${extension}`);

        // Get previousResponseId from cookie
        const previousResponseId = getCookie(previousResponseIdByCurrLanguage());
        if (previousResponseId) {
            formData.append('previousResponseId', previousResponseId);
        }

        // Add language
        formData.append('language', languageSelector.value);

        console.log('Sending audio file:', `recording${extension}`, 'Size:', audioBlob.size, 'bytes');

        const response = await fetch('/chat', {
            method: 'POST',
            body: formData,
        });

        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        const data = await response.json();

        // Store the response ID in cookie for next request
        if (data.responseId) {
            setCookie(previousResponseIdByCurrLanguage(), data.responseId);
        }

        // Display the response
        responseArea.innerHTML = `<p>${data.reply}</p>`;

        // Play audio response if available
        if (data.audioData) {
            playAudioResponse(data.audioData);
        }
    } catch (error) {
        responseArea.innerHTML = `<p class="error">Error: ${error.message}</p>`;
        console.error('Error:', error);
    } finally {
        // Re-enable buttons
        recordButton.disabled = false;
        sendButton.disabled = false;
        messageInput.disabled = false;
        messageInput.focus();
    }
}

// Track drag position for cancel gesture
function handleDragMove(e) {
    if (!isRecording) return;

    const currentX = e.clientX || e.touches?.[0]?.clientX || 0;
    const currentY = e.clientY || e.touches?.[0]?.clientY || 0;

    const deltaX = recordStartX - currentX;
    const deltaY = Math.abs(recordStartY - currentY);

    // Cancel if dragged left beyond threshold (and not too much vertical movement)
    if (deltaX > CANCEL_THRESHOLD && deltaY < 50) {
        recordButton.classList.add('canceling');
        isCanceled = true;
    } else {
        recordButton.classList.remove('canceling');
        isCanceled = false;
    }
}

// Event listeners for text input
sendButton.addEventListener('click', sendMessage);
messageInput.addEventListener('keypress', (e) => {
    if (e.key === 'Enter') {
        sendMessage();
    }
});

// Voice recording event listeners (press and hold)
recordButton.addEventListener('mousedown', (e) => {
    recordStartX = e.clientX;
    recordStartY = e.clientY;
    startRecording();
});

// Track mouse movement for cancel gesture
document.addEventListener('mousemove', handleDragMove);

// Global mouseup to stop recording anywhere
document.addEventListener('mouseup', () => {
    if (isRecording) {
        stopRecording();
    }
});

// Touch events for mobile
recordButton.addEventListener('touchstart', (e) => {
    e.preventDefault();
    recordStartX = e.touches[0].clientX;
    recordStartY = e.touches[0].clientY;
    startRecording();
});

// Track touch movement for cancel gesture
document.addEventListener('touchmove', handleDragMove);

// Global touchend to stop recording anywhere
document.addEventListener('touchend', (e) => {
    if (isRecording) {
        e.preventDefault();
        stopRecording();
    }
});

// Focus input on load
messageInput.focus();

