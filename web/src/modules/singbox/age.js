import { appState } from '../../state.js';
import { localize, tAlert } from '../../i18n-runtime.js';
import { alert, copyText, showToast } from '../../utils.js';
import { fetchSingBoxOverview } from './nodes.js';

export async function openAgeKeyHelper() {
    try {
        const res = await fetch('/api/settings');
        if (res.ok) {
            const data = await res.json();
            const toggleEl = document.getElementById('age-encrypt-enabled');
            const pubInput = document.getElementById('age-public-key-input');
            if (toggleEl) toggleEl.checked = !!data.age_encrypt_enabled;
            if (pubInput) pubInput.value = data.age_public_key || '';
            if (data.age_public_key && data.age_public_key.startsWith('age1pq')) {
                switchAgeType('mlkem768-x25519');
            } else {
                switchAgeType('x25519');
            }
        }
    } catch (e) {
        console.warn('获取 age 配置失败:', e);
    }
    document.getElementById('age-key-helper-modal').classList.add('open');
}

export function closeAgeKeyHelper() {
    document.getElementById('age-key-helper-modal').classList.remove('open');
}

export function switchAgeType(type) {
    appState.currentAgeType = type;
    const xBtn = document.getElementById('age-type-x25519');
    const pqBtn = document.getElementById('age-type-pq');
    if (type === 'x25519') {
        if (xBtn) xBtn.classList.add('active');
        if (pqBtn) pqBtn.classList.remove('active');
    } else {
        if (pqBtn) pqBtn.classList.add('active');
        if (xBtn) xBtn.classList.remove('active');
    }
}

export async function generateAgeKey() {
    try {
        const res = await fetch(`/api/singbox/subscription/age/generate?type=${encodeURIComponent(appState.currentAgeType)}`, {
            method: 'POST'
        });
        const data = await res.json();
        if (!res.ok || !data.ok) {
            tAlert('生成密钥失败: ' + (data.error || '未知错误'), 'Key generation failed: ' + (data.error || 'Unknown error'));
            return;
        }
        document.getElementById('age-secret-key-input').value = data.secret_key;
        document.getElementById('age-public-key-input').value = data.public_key;
        showToast(localize(`已成功生成全新 ${data.type} 密钥对！解密私钥请妥善保存。`, `Generated a new ${data.type} key pair. Keep the private key safe.`));
    } catch (err) {
        alert(localize('网络请求失败: ', 'Network request failed: ') + err);
    }
}

export function copyAgeSecretKey() {
    const sec = document.getElementById('age-secret-key-input').value.trim();
    if (!sec) {
        tAlert('请先输入或生成 age 解密私钥', 'Please enter or generate an age secret key first');
        return;
    }
    copyText(sec, localize('age 解密私钥', 'age private key'));
}

export async function deriveAgePublicKey() {
    const sec = document.getElementById('age-secret-key-input').value.trim();
    if (!sec) {
        tAlert('请先在私钥框中粘贴或生成 age 解密私钥', 'Please paste or generate an age secret key in the textarea first');
        return;
    }
    try {
        const res = await fetch('/api/singbox/subscription/age/derive', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ secret_key: sec })
        });
        const data = await res.json();
        if (!res.ok || !data.ok) {
            tAlert('从私钥推导公钥失败: ' + (data.error || '未知错误'), 'Deriving public key failed: ' + (data.error || 'Unknown error'));
            return;
        }
        document.getElementById('age-public-key-input').value = data.public_key;
        if (data.type === 'MLKEM768-X25519') {
            switchAgeType('mlkem768-x25519');
        } else {
            switchAgeType('x25519');
        }
        showToast(localize(`已成功推导出 ${data.type} 加密公钥！`, `Derived the ${data.type} public key.`));
    } catch (err) {
        alert(localize('网络请求失败: ', 'Network request failed: ') + err);
    }
}

export function copyAgePublicKey() {
    const pub = document.getElementById('age-public-key-input').value.trim();
    if (!pub) {
        tAlert('请先输入或生成 age 加密公钥', 'Please enter or generate an age public key first');
        return;
    }
    copyText(pub, localize('age 加密公钥', 'age public key'));
}

export async function applyAgePublicKey() {
    const pub = document.getElementById('age-public-key-input').value.trim();
    if (!pub) {
        tAlert('请先生成或输入 age 加密公钥', 'Please generate or enter an age public key first');
        return;
    }
    const toggleEl = document.getElementById('age-encrypt-enabled');
    if (toggleEl) toggleEl.checked = true;
    try {
        const res = await fetch('/api/settings', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                age_encrypt_enabled: true,
                age_public_key: pub
            })
        });
        if (!res.ok) {
            const err = await res.json();
            alert(localize('保存失败: ', 'Save failed: ') + (err.error || localize('未知错误', 'Unknown error')));
            return;
        }
        const cfgPub = document.getElementById('cfg-age-pubkey');
        if (cfgPub) cfgPub.value = pub;
        showToast(localize('已一键填入公钥并开启 age 订阅端到端加密！', 'Public key applied and age subscription encryption enabled.'));
        await fetchSingBoxOverview();
    } catch (err) {
        alert(localize('网络请求失败: ', 'Network request failed: ') + err);
    }
}

export async function onAgeToggleChanged() {
    const toggleEl = document.getElementById('age-encrypt-enabled');
    const isEnabled = toggleEl.checked;
    const pub = document.getElementById('age-public-key-input').value.trim();

    if (isEnabled && !pub) {
        tAlert('请先在下方输入或一键生成 age 加密公钥，再开启加密功能！', 'Please enter or generate an age public key before enabling encryption!');
        toggleEl.checked = false;
        return;
    }

    try {
        const res = await fetch('/api/settings', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                age_encrypt_enabled: isEnabled,
                age_public_key: pub
            })
        });
        if (!res.ok) {
            const err = await res.json();
            alert(localize('保存开关失败: ', 'Failed to save setting: ') + (err.error || localize('未知错误', 'Unknown error')));
            toggleEl.checked = !isEnabled;
            return;
        }
        showToast(localize(`age 订阅端到端加密已${isEnabled ? '开启' : '关闭'}`, `age subscription encryption ${isEnabled ? 'enabled' : 'disabled'}`));
        await fetchSingBoxOverview();
    } catch (err) {
        alert(localize('网络请求失败: ', 'Network request failed: ') + err);
        toggleEl.checked = !isEnabled;
    }
}
