<script setup lang="ts">
import { X } from 'lucide-vue-next'
defineProps<{ open: boolean; title: string; eyebrow?: string; busy?: boolean }>()
const emit = defineEmits<{ close: [] }>()
</script>
<template>
  <Teleport to="body"><Transition name="modal"><div v-if="open" class="modal-backdrop" @mousedown.self="!busy && emit('close')">
    <section class="modal" role="dialog" aria-modal="true" :aria-label="title">
      <header class="modal-header"><div><div v-if="eyebrow" class="eyebrow">{{ eyebrow }}</div><h2 class="modal-title">{{ title }}</h2></div><button class="btn btn-ghost btn-icon" :disabled="busy" aria-label="关闭" @click="emit('close')"><X /></button></header>
      <div class="modal-body"><slot /></div><footer class="modal-footer"><slot name="footer" /></footer>
    </section>
  </div></Transition></Teleport>
</template>
