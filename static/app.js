const messageInput = document.getElementById('message-input');
const sendButton = document.getElementById('send-button');
const responseArea = document.getElementById('response-area');
const recordButton = document.getElementById('record-button');
const languageSelector = document.getElementById('language-selector');

let mediaRecorder = null;
let audioChunks = [];
let isRecording = false;

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
        const previousResponseId = getCookie('previousResponseId');
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
            setCookie('previousResponseId', data.responseId);
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

        mediaRecorder.addEventListener('dataavailable', event => {
            audioChunks.push(event.data);
        });

        mediaRecorder.addEventListener('stop', async () => {
            const audioBlob = new Blob(audioChunks, { type: mimeType });
            await sendAudioMessage(audioBlob, mimeType);

            // Stop all tracks to release the microphone
            stream.getTracks().forEach(track => track.stop());
        });

        mediaRecorder.start();
        isRecording = true;

        recordButton.classList.add('recording');
        recordButton.querySelector('.record-text').textContent = 'Recording';

    console.log('stopRecording called, isRecording:', isRecording, 'mediaRecorder:', mediaRecorder);

    if (!isRecording || !mediaRecorder) {
        console.log('Not recording or no mediaRecorder, returning');
        return;
    }
    } catch (error) {
    console.log('Stopping recording...');
        console.error('Error accessing microphone:', error);
        responseArea.innerHTML = '<p class="error">Error: Could not access microphone. Please grant permission.</p>';
    }
}

function stopRecording() {
    if (!isRecording || !mediaRecorder) return;

    mediaRecorder.stop();
    isRecording = false;

    recordButton.classList.remove('recording');
    recordButton.querySelector('.record-text').textContent = 'Hold to Record';

    console.log('Recording stopped');
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
        const previousResponseId = getCookie('previousResponseId');
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
            setCookie('previousResponseId', data.responseId);
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

// Event listeners for text input
sendButton.addEventListener('click', sendMessage);
messageInput.addEventListener('keypress', (e) => {
    if (e.key === 'Enter') {
        sendMessage();
    }
});

// Voice recording event listeners (press and hold)
recordButton.addEventListener('mousedown', startRecording);
recordButton.addEventListener('mouseup', stopRecording);
recordButton.addEventListener('mouseleave', () => {
    if (isRecording) stopRecording();
});

// Touch events for mobile
recordButton.addEventListener('touchstart', (e) => {
    e.preventDefault();
    startRecording();
});
recordButton.addEventListener('touchend', (e) => {
    e.preventDefault();
    stopRecording();
});

// Focus input on load
messageInput.focus();

