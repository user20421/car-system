<script setup lang="ts">
defineProps<{ compact?: boolean; light?: boolean }>()
</script>
<template>
  <div class="brand" :class="{ compact, light }">
    <svg class="brand-mark" viewBox="0 0 44 44" fill="none" aria-hidden="true">
      <defs><linearGradient id="logo-a" x1="7" y1="5" x2="39" y2="40"><stop stop-color="#9184FF"/><stop offset="1" stop-color="#34D8D0"/></linearGradient></defs>
      <rect x="1" y="1" width="42" height="42" rx="13" fill="url(#logo-a)" fill-opacity=".13" stroke="url(#logo-a)" stroke-opacity=".45"/>
      <path d="M9.5 31.5 19.4 11h5.8l10 20.5h-6.6l-2.1-4.9h-9.3l-2.1 4.9H9.5Z" fill="url(#logo-a)"/>
      <path d="m19.3 22.2 2.5-5.8 2.6 5.8h-5.1Z" fill="currentColor"/>
      <path d="M10 35h24" stroke="url(#logo-a)" stroke-width="1.5" stroke-linecap="round"/>
    </svg>
    <div v-if="!compact" class="brand-type"><strong>AURORA</strong><span>MOBILITY INTELLIGENCE</span></div>
  </div>
</template>
<style scoped>
.brand{display:flex;align-items:center;gap:11px;color:var(--text)}.brand.light{color:#fff}.brand-mark{width:38px;height:38px;flex:none}.brand-type{display:flex;flex-direction:column;gap:2px}.brand-type strong{font-size:17px;line-height:1;letter-spacing:.15em;font-weight:760}.brand-type span{font-size:6px;letter-spacing:.19em;color:var(--muted);font-weight:700}.light .brand-type span{color:#777f9f}.compact .brand-mark{width:36px;height:36px}
</style>
