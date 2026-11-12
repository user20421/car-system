<script setup lang="ts">
import ToastHost from './components/common/ToastHost.vue'
</script>

<template>
  <RouterView v-slot="{ Component }">
    <Transition name="page" mode="out-in"><component :is="Component" /></Transition>
  </RouterView>
  <ToastHost />
</template>
