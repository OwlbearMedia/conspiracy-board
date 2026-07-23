<script setup lang="ts">
import { onMounted, ref } from 'vue'

const apiStatus = ref<'checking' | 'ready' | 'down'>('checking')

onMounted(async () => {
  try {
    const res = await fetch('/api/v1/readyz')
    apiStatus.value = res.ok ? 'ready' : 'down'
  } catch {
    apiStatus.value = 'down'
  }
})
</script>

<template>
  <main>
    <h1>🧵 Conspiracy Board</h1>
    <p>
      API status:
      <strong :class="apiStatus">{{ apiStatus }}</strong>
    </p>
  </main>
</template>

<style scoped>
main {
  font-family: system-ui, sans-serif;
  max-width: 40rem;
  margin: 4rem auto;
}
.ready { color: green; }
.down { color: crimson; }
.checking { color: gray; }
</style>
