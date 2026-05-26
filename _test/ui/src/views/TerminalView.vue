<script setup lang="ts">
import { Terminal } from '@xterm/xterm';
import type { IDisposable } from '@xterm/xterm';
import { onMounted, onUnmounted, reactive, shallowRef } from "vue";
const PORT = document.location.port ? `:${document.location.port}` : '';
const SCHEME = document.location.protocol === 'https:' ? 'wss' : 'ws';

const BASE_WS_URL = `${SCHEME}://${document.location.hostname}${PORT}`;
type ConnectResponse = {
  type: 'connected' | 'error';
  uuid?: string;
  message?: string;
}

const form = reactive({
  host: '',
  port: 22,
  username: '',
  password: '',
});
const terminalRef = shallowRef<Terminal>();
const wsRef = shallowRef<WebSocket>();
const terminalInputRef = shallowRef<IDisposable>();
const uuid = shallowRef('');
const errorMessage = shallowRef('');
const connected = shallowRef(false);
const connecting = shallowRef(false);

onMounted(() => {
  const containerElement = document.getElementById("terminal");
  const terminal = new Terminal(
    {
      cols: 120,
      rows: 80,
      fontSize: 10,
    }
  );
  terminal.open(containerElement!);
  terminalRef.value = terminal;
});

onUnmounted(() => {
  terminalInputRef.value?.dispose();
  wsRef.value?.close();
  terminalRef.value?.dispose();
});

function connectSSH() {
  if (connecting.value || connected.value) {
    return;
  }
  errorMessage.value = '';
  uuid.value = '';
  connecting.value = true;

  const wsURL = `${BASE_WS_URL}/ws/ssh/`;
  const ws = new WebSocket(wsURL);
  ws.binaryType = 'arraybuffer';
  wsRef.value = ws;

  ws.addEventListener('open', () => {
    ws.send(JSON.stringify({
      type: 'connect',
      config: {
        host: form.host,
        port: Number(form.port) || 22,
        username: form.username,
        password: form.password,
      },
    }));
  });

  ws.addEventListener('message', (event) => {
    if (event.data instanceof ArrayBuffer) {
      terminalRef.value?.write(new Uint8Array(event.data));
      return;
    }
    if (event.data instanceof Blob) {
      event.data.arrayBuffer().then((buffer) => {
        terminalRef.value?.write(new Uint8Array(buffer));
      });
      return;
    }
    if (typeof event.data !== 'string') {
      return;
    }
    try {
      const message = JSON.parse(event.data) as ConnectResponse;
      if (message.type === 'connected' && message.uuid) {
        uuid.value = message.uuid;
        connected.value = true;
        connecting.value = false;
        terminalInputRef.value?.dispose();
        terminalInputRef.value = terminalRef.value?.onData((data) => {
          if (ws.readyState === WebSocket.OPEN) {
            ws.send(data);
          }
        });
        return;
      }
      if (message.type === 'error') {
        errorMessage.value = message.message || '连接失败';
        connecting.value = false;
        connected.value = false;
        ws.close();
      }
    } catch {
      terminalRef.value?.write(event.data);
    }
  });

  ws.addEventListener('close', () => {
    terminalInputRef.value?.dispose();
    terminalInputRef.value = undefined;
    connecting.value = false;
    connected.value = false;
  });

  ws.addEventListener('error', () => {
    errorMessage.value = 'WebSocket 连接失败';
    connecting.value = false;
    connected.value = false;
  });
}


</script>


<template>
  <div class="terminal-page">
    <form class="connect-form" @submit.prevent="connectSSH">
      <input v-model.trim="form.host" class="connect-input" placeholder="Host" :disabled="connecting || connected">
      <input v-model.number="form.port" class="connect-input connect-port" type="number" min="1" max="65535" placeholder="Port" :disabled="connecting || connected">
      <input v-model.trim="form.username" class="connect-input" placeholder="Username" :disabled="connecting || connected">
      <input v-model="form.password" class="connect-input" type="password" placeholder="Password" :disabled="connecting || connected">
      <button class="connect-button" type="submit" :disabled="connecting || connected">
        {{ connecting ? 'Connecting' : connected ? 'Connected' : 'Connect' }}
      </button>
    </form>
    <div class="status-row">
      <span v-if="uuid">UUID: {{ uuid }}</span>
      <a v-if="uuid" :href="`/api/ssh/${uuid}/result`" target="_blank">View Result</a>
      <span v-if="errorMessage" class="error-message">{{ errorMessage }}</span>
    </div>
    <div id="terminal" class="terminal-host"></div>
  </div>
</template>

<style scoped>
.terminal-page {
  width: 1000px;
}

.connect-form {
  display: grid;
  grid-template-columns: 1fr 96px 1fr 1fr 120px;
  gap: 8px;
  margin-bottom: 8px;
}

.connect-input {
  min-width: 0;
  height: 32px;
  padding: 4px 8px;
  border: 1px solid #c8cdd4;
  border-radius: 4px;
  font-size: 13px;
}

.connect-port {
  text-align: right;
}

.connect-button {
  height: 32px;
  border: 1px solid #2d6cdf;
  border-radius: 4px;
  background: #2d6cdf;
  color: #fff;
  font-size: 13px;
  cursor: pointer;
}

.connect-button:disabled {
  cursor: default;
  opacity: 0.7;
}

.status-row {
  min-height: 20px;
  margin-bottom: 8px;
  color: #384252;
  font-size: 12px;
}

.error-message {
  color: #b42318;
}

.terminal-host {
  width: 1000px;
  height: 800px;
}
</style>
