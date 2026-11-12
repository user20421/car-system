<script setup lang="ts">
defineProps<{ eyebrow?: string; title: string; subtitle?: string }>()
</script>
<template>
  <header class="page-header">
    <div><div v-if="eyebrow" class="eyebrow">{{ eyebrow }}</div><h1 class="page-title">{{ title }}</h1><p v-if="subtitle" class="page-subtitle">{{ subtitle }}</p></div>
    <div class="page-actions"><slot name="actions" /></div>
  </header>
</template>
