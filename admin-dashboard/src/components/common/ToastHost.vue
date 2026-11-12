<script setup lang="ts">
import { CheckCircle2, Info, XCircle } from 'lucide-vue-next'
import { useUiStore } from '../../stores/ui'
const ui = useUiStore()
</script>
<template><div class="toast-host" aria-live="polite"><TransitionGroup name="toast"><div v-for="toast in ui.toasts" :key="toast.id" class="toast" :class="`toast-${toast.tone}`"><CheckCircle2 v-if="toast.tone==='success'"/><XCircle v-else-if="toast.tone==='error'"/><Info v-else/><div><strong>{{ toast.title }}</strong><p v-if="toast.message">{{ toast.message }}</p></div></div></TransitionGroup></div></template>
<style scoped>
.toast-host{position:fixed;right:24px;bottom:24px;z-index:200;display:flex;flex-direction:column;gap:10px;width:min(360px,calc(100vw - 32px))}.toast{display:flex;gap:12px;padding:15px 17px;background:var(--surface-solid);border:1px solid var(--line);border-radius:13px;box-shadow:var(--shadow-md);color:var(--primary)}.toast svg{width:19px;flex:none}.toast strong{display:block;color:var(--text);font-size:12px}.toast p{font-size:11px;color:var(--text-2);margin-top:4px;line-height:1.5}.toast-success{color:var(--green)}.toast-error{color:var(--red)}.toast-enter-active,.toast-leave-active{transition:all .25s}.toast-enter-from,.toast-leave-to{opacity:0;transform:translateX(15px)}
</style>
