<script setup lang="ts">
withDefaults(defineProps<{ color?: string; compact?: boolean }>(), { color: '#776bec', compact: false })
</script>
<template>
  <svg class="vehicle-art" :class="{ compact }" viewBox="0 0 560 240" fill="none" role="img" aria-label="AURORA 智能电动车侧视图">
    <defs>
      <linearGradient id="body" x1="100" y1="45" x2="415" y2="202"><stop stop-color="#F6F7FC"/><stop offset=".45" :stop-color="color"/><stop offset="1" stop-color="#34374B"/></linearGradient>
      <linearGradient id="glass" x1="242" y1="55" x2="304" y2="131"><stop stop-color="#64728F"/><stop offset="1" stop-color="#171D30"/></linearGradient>
      <linearGradient id="shine" x1="0" y1="0" x2="1" y2="1"><stop stop-color="white" stop-opacity=".8"/><stop offset="1" stop-color="white" stop-opacity="0"/></linearGradient>
      <radialGradient id="wheel"><stop offset=".45" stop-color="#272C3A"/><stop offset=".48" stop-color="#9097A7"/><stop offset=".72" stop-color="#303543"/><stop offset=".76" stop-color="#10131C"/><stop offset="1" stop-color="#090B11"/></radialGradient>
      <filter id="shadow" x="-30%" y="-100%" width="160%" height="300%"><feGaussianBlur stdDeviation="11"/></filter>
    </defs>
    <ellipse cx="286" cy="202" rx="210" ry="14" fill="#333955" opacity=".18" filter="url(#shadow)"/>
    <path d="M75 161c5-26 20-40 53-48l72-15 53-50c10-9 24-14 39-14h65c20 0 35 6 49 20l51 54 39 14c15 6 24 17 27 34l3 22c1 8-6 14-14 14h-30c-4-34-24-52-52-52s-49 19-53 52H210c-4-34-24-52-52-52s-49 19-53 52H77c-12 0-19-12-15-23l13-4Z" fill="url(#body)"/>
    <path d="m210 97 52-48c8-7 18-10 30-10h28v62l-110-4Z" fill="url(#glass)"/>
    <path d="M331 39h27c15 0 27 5 38 16l43 47-108-1V39Z" fill="url(#glass)"/>
    <path d="m270 46 39-1-77 54-16-1 54-52Z" fill="url(#shine)" opacity=".45"/>
    <path d="M321 39v65M207 103l234 2" stroke="#D8DCE8" stroke-opacity=".6"/>
    <path d="M84 150c82-13 281-27 403-5" stroke="white" stroke-opacity=".38" stroke-width="2"/>
    <path d="M219 116h70M333 116h84" stroke="#303648" stroke-opacity=".35"/>
    <path d="M271 116h22M365 116h22" stroke="#F0F1F6" stroke-width="3" stroke-linecap="round"/>
    <path d="m454 119 31 11-6 15-33-5 8-21Z" fill="#D7FAFF"/>
    <path d="m72 145 26-8 6 17-35 4 3-13Z" fill="#FF7793"/>
    <path d="M73 176h35M438 176h46" stroke="#C9CEDA" stroke-width="4" stroke-linecap="round"/>
    <path d="M301 144h60" stroke="#E4E6ED" stroke-opacity=".6" stroke-width="2"/>
    <g><circle cx="158" cy="181" r="43" fill="url(#wheel)"/><circle cx="158" cy="181" r="24" fill="#242938" stroke="#9199AA" stroke-width="2"/><path d="m158 158 5 17 17 6-17 5-5 17-5-17-17-5 17-6 5-17Z" fill="#AEB5C4"/><circle cx="158" cy="181" r="5" fill="#202431"/></g>
    <g><circle cx="392" cy="181" r="43" fill="url(#wheel)"/><circle cx="392" cy="181" r="24" fill="#242938" stroke="#9199AA" stroke-width="2"/><path d="m392 158 5 17 17 6-17 5-5 17-5-17-17-5 17-6 5-17Z" fill="#AEB5C4"/><circle cx="392" cy="181" r="5" fill="#202431"/></g>
    <path d="M245 162h65" stroke="#A8F7EF" stroke-width="2" stroke-linecap="round" opacity=".85"/>
    <text x="272" y="154" fill="white" fill-opacity=".8" font-size="8" font-family="Manrope" letter-spacing="3">AURORA</text>
  </svg>
</template>
<style scoped>.vehicle-art{width:100%;height:auto;display:block;filter:drop-shadow(0 16px 20px rgba(40,45,80,.08));animation:float 7s ease-in-out infinite}.vehicle-art.compact{animation:none}</style>
