<script setup lang="ts">
import { RouterView } from 'vue-router'
import NavBar from './components/NavBar.vue'
</script>

<template>
  <div class="app-layout">
    <NavBar />
    <main class="app-content">
      <RouterView />
    </main>
  </div>
</template>

<style scoped>
.app-layout {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background: var(--color-bg);
}

.app-content {
  flex: 1;
  overflow: hidden;
}
</style>
