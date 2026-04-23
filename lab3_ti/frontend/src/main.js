import './style.css';
import './app.css';
import { EncryptFile, DecryptFile } from '../wailsjs/go/main/App';

document.querySelector('#app').innerHTML = `
<div class="container">
    <h1>🔐 Шифратор Рабина</h1>

    <div class="card">
        <div class="card-title">Параметры ключа</div>
        <div class="params-grid">
            <div class="param-group">
                <label for="paramP">p <span class="hint">(простое, p ≡ 3 mod 4)</span></label>
                <input type="number" id="paramP" class="param-input"
                    placeholder="Например: 523" min="3" />
            </div>
            <div class="param-group">
                <label for="paramQ">q <span class="hint">(простое, q ≡ 3 mod 4)</span></label>
                <input type="number" id="paramQ" class="param-input"
                    placeholder="Например: 3511" min="3" />
            </div>
            <div class="param-group">
                <label for="paramB">b <span class="hint">(0 &lt; b &lt; n)</span></label>
                <input type="number" id="paramB" class="param-input"
                    placeholder="Например: 1234" min="1" />
            </div>
        </div>
        <div class="params-info" id="paramsInfo"></div>
    </div>

    <div class="card">
        <div class="card-title">Файл</div>
        <div class="file-row">
            <button class="btn btn-secondary" id="selectFileBtn">📁 Выбрать файл</button>
            <div id="selectedFile" class="selected-file">Файл не выбран</div>
        </div>
        <input type="file" id="fileInput" style="display:none" />
        <div class="file-hint" id="fileHint"></div>
    </div>

    <div class="actions-row">
        <button class="btn btn-encrypt" id="encryptBtn">🔒 Зашифровать</button>
        <button class="btn btn-decrypt" id="decryptBtn">🔓 Расшифровать</button>
    </div>

    <div class="output-grid">
        <div class="card output-card">
            <div class="card-title" id="label-original">📄 Байты исходного файла</div>
            <div class="bytes-display" id="originalOutput">—</div>
        </div>
        <div class="card output-card">
            <div class="card-title" id="label-result">🔐 Зашифрованные значения</div>
            <div class="bytes-display" id="resultOutput">—</div>
        </div>
    </div>

    <div class="status-bar" id="statusBar"></div>
</div>
`;

const paramP         = document.getElementById('paramP');
const paramQ         = document.getElementById('paramQ');
const paramB         = document.getElementById('paramB');
const paramsInfo     = document.getElementById('paramsInfo');
const fileInput      = document.getElementById('fileInput');
const selectFileBtn  = document.getElementById('selectFileBtn');
const selectedFileEl = document.getElementById('selectedFile');
const fileHint       = document.getElementById('fileHint');
const encryptBtn     = document.getElementById('encryptBtn');
const decryptBtn     = document.getElementById('decryptBtn');
const originalOutput = document.getElementById('originalOutput');
const resultOutput   = document.getElementById('resultOutput');
const statusBar      = document.getElementById('statusBar');
const labelOriginal  = document.getElementById('label-original');
const labelResult    = document.getElementById('label-result');

function updateInfo() {
    const p = parseInt(paramP.value);
    const q = parseInt(paramQ.value);
    const b = parseInt(paramB.value);

    if (isNaN(p) || isNaN(q)) {
        paramsInfo.textContent = '';
        return;
    }

    const n    = p * q;
    const pOk  = Number.isInteger(p) && p % 4 === 3;
    const qOk  = Number.isInteger(q) && q % 4 === 3;
    const nOk  = n > 256;
    const bOk  = !isNaN(b) && b > 0 && b < n;

    paramsInfo.innerHTML =
        `n = p × q = <b>${n}</b> &nbsp;|&nbsp; ` +
        `p mod 4 = ${p % 4} ${pOk ? '✅' : '❌'} &nbsp;|&nbsp; ` +
        `q mod 4 = ${q % 4} ${qOk ? '✅' : '❌'} &nbsp;|&nbsp; ` +
        `n > 256: ${nOk ? '✅' : '❌'}` +
        (!isNaN(b) ? ` &nbsp;|&nbsp; b < n: ${bOk ? '✅' : '❌'}` : '');

    paramsInfo.style.color = (pOk && qOk && nOk && bOk) ? '#4caf50' : '#ff7043';
}

paramP.addEventListener('input', updateInfo);
paramQ.addEventListener('input', updateInfo);
paramB.addEventListener('input', updateInfo);

selectFileBtn.addEventListener('click', () => fileInput.click());

fileInput.addEventListener('change', () => {
    const f = fileInput.files[0];
    if (f) {
        const isEncrypted = f.name.includes('_encrypted');
        selectedFileEl.textContent = `${f.name}  (${formatSize(f.size)})`;
        selectedFileEl.className = 'selected-file has-file';
        fileHint.textContent = isEncrypted
            ? '💡 Файл зашифрован — используйте «Расшифровать»'
            : '💡 Обычный файл — используйте «Зашифровать»';
        fileHint.style.color = isEncrypted ? '#8957e5' : '#1f6feb';
    } else {
        selectedFileEl.textContent = 'Файл не выбран';
        selectedFileEl.className = 'selected-file';
        fileHint.textContent = '';
    }
});

function formatSize(bytes) {
    if (bytes < 1024) return bytes + ' Б';
    if (bytes < 1048576) return (bytes / 1024).toFixed(1) + ' КБ';
    return (bytes / 1048576).toFixed(2) + ' МБ';
}

function readFileBytes(file) {
    return new Promise((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = (e) => {
            const uint8 = new Uint8Array(e.target.result);
            const arr = new Array(uint8.length);
            for (let i = 0; i < uint8.length; i++) {
                arr[i] = uint8[i];
            }
            resolve(arr);
        };
        reader.onerror = reject;
        reader.readAsArrayBuffer(file);
    });
}

function getParams() {
    const p = parseInt(paramP.value);
    const q = parseInt(paramQ.value);
    const b = parseInt(paramB.value);
    if (isNaN(p) || isNaN(q) || isNaN(b)) {
        throw new Error('Введите все три параметра p, q и b');
    }
    return { p, q, b };
}

function setStatus(msg, type = 'info') {
    statusBar.textContent = msg;
    statusBar.className = 'status-bar status-' + type;
}

function setBusy(btn, busy, normal, loading) {
    btn.disabled = busy;
    btn.textContent = busy ? loading : normal;
}
encryptBtn.addEventListener('click', async () => {
    let params;
    try { params = getParams(); }
    catch (e) { setStatus('❌ ' + e.message, 'error'); return; }

    const file = fileInput.files[0];
    if (!file) { setStatus('❌ Выберите файл', 'error'); return; }

    try {
        setBusy(encryptBtn, true, '🔒 Зашифровать', '⏳ Шифрование...');
        setStatus('Шифрование...', 'info');
        originalOutput.textContent = '—';
        resultOutput.textContent   = '—';

        const bytes  = await readFileBytes(file);
        const result = await EncryptFile(
            params.p, params.q, params.b,
            bytes,
            file.name
        );

        labelOriginal.textContent = '📄 Байты исходного файла (десятичные)';
        labelResult.textContent   = '🔐 Зашифрованные значения (десятичные)';
        originalOutput.textContent = result.original_bytes;
        resultOutput.textContent   = result.encrypted_bytes;

        setStatus(
            `✅ Зашифровано → ${result.saved_as}  (n=${result.n_value}, ${result.bytes_per_value} байт/число)`,
            'success'
        );
    } catch (e) {
        setStatus('❌ ' + e, 'error');
        console.error(e);
    } finally {
        setBusy(encryptBtn, false, '🔒 Зашифровать', '');
    }
});

decryptBtn.addEventListener('click', async () => {
    let params;
    try { params = getParams(); }
    catch (e) { setStatus('❌ ' + e.message, 'error'); return; }

    const file = fileInput.files[0];
    if (!file) { setStatus('❌ Выберите зашифрованный файл', 'error'); return; }

    try {
        setBusy(decryptBtn, true, '🔓 Расшифровать', '⏳ Расшифрование...');
        setStatus('Расшифрование...', 'info');
        originalOutput.textContent = '—';
        resultOutput.textContent   = '—';

        const bytes  = await readFileBytes(file);
        const result = await DecryptFile(
            params.p, params.q, params.b,
            bytes,
            file.name
        );

        labelOriginal.textContent = '🔐 Зашифрованные значения (десятичные)';
        labelResult.textContent   = '📄 Расшифрованные байты (десятичные)';
        originalOutput.textContent = result.original_bytes;
        resultOutput.textContent   = result.encrypted_bytes;

        setStatus(`✅ Расшифровано → ${result.saved_as}`, 'success');
    } catch (e) {
        setStatus('❌ ' + e, 'error');
        console.error(e);
    } finally {
        setBusy(decryptBtn, false, '🔓 Расшифровать', '');
    }
});