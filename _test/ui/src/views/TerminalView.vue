<script setup lang="ts">
import { Terminal } from '@xterm/xterm';
import {FitAddon} from '@xterm/addon-fit'
import { useWindowSize } from '@vueuse/core';
const { height,width } = useWindowSize();
import { AttachAddon } from '@xterm/addon-attach';
import {onMounted, ref, watch, nextTick, onUnmounted} from "vue";
const PORT = document.location.port ? `:${document.location.port}` : '';
const SCHEME = document.location.protocol === 'https:' ? 'wss' : 'ws';

const BASE_WS_URL = `${SCHEME}://${document.location.hostname}${PORT}`;
const terminalRef = ref()
const fitAddon = new FitAddon();
const wsRef = ref()
onMounted(()=>{
  const containerElement = document.getElementById("terminal");
  const wsURL = `${BASE_WS_URL}/ws/ssh/`;
  const ws = new WebSocket(wsURL);
  wsRef.value= ws
  const terminal = new Terminal();

  const attachAddon = new AttachAddon(ws);
  terminal.loadAddon(attachAddon);
  terminal.loadAddon(fitAddon);
  terminal.open(containerElement!);
  fitAddon.fit();
  terminalRef.value=terminal;
})

const sendSize = () => {
  const windowSize = {high: terminalRef.value.rows, width: terminalRef.value.cols};
  const blob = new Blob([JSON.stringify(windowSize)], {type : 'application/json'});
  wsRef.value.send(blob);
}


watch([width, height], ([_newWidth, _newHeight]: [number, number]) => {
  if (!terminalRef.value || !fitAddon) return;

  nextTick(() => {
    fitAddon.fit();
    sendSize()
  });
});


</script>


<template>
<div id="terminal" class="h-full w-full" style="height: calc(100vh)">
</div>
</template>

<style scoped>
@import "@xterm/xterm/css/xterm.css";

</style>
