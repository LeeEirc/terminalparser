<script setup lang="ts">
import { Terminal } from '@xterm/xterm';
import { useWindowSize, useDebounceFn } from '@vueuse/core';
const { height,width } = useWindowSize();
import { AttachAddon } from '@xterm/addon-attach';
import {onMounted, ref, watch, nextTick, onUnmounted} from "vue";
const PORT = document.location.port ? `:${document.location.port}` : '';
const SCHEME = document.location.protocol === 'https:' ? 'wss' : 'ws';

const BASE_WS_URL = `${SCHEME}://${document.location.hostname}${PORT}`;
const terminalRef = ref()
const wsRef = ref()
onMounted(()=>{
  const containerElement = document.getElementById("terminal");
  const wsURL = `${BASE_WS_URL}/ws/ssh/`;
  const ws = new WebSocket(wsURL);
  wsRef.value= ws
  const terminal = new Terminal(
    {
      cols:120,
      rows:80,
      fontSize: 10,
    }
  );
  const attachAddon = new AttachAddon(ws);
  terminal.loadAddon(attachAddon);
  terminal.open(containerElement!);
  terminalRef.value=terminal;
})

const sendSize = useDebounceFn(() => {
  const windowSize = {high:terminalRef.value.rows, width: terminalRef.value.cols};
  const blob = new Blob([JSON.stringify(windowSize)], {type : 'application/json'});
  console.log(windowSize)
  wsRef.value.send(blob);
})


</script>


<template>
<div id="terminal" class="w-full">
</div>
</template>

<style scoped>
#terminal {
  width: 1000px;
  height: 800px;
}
</style>
