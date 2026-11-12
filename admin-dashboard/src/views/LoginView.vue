<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowRight, Check, Eye, EyeOff, LockKeyhole, ShieldCheck, UserRound, Wifi } from 'lucide-vue-next'
import AppLogo from '../components/common/AppLogo.vue'
import VehicleArt from '../components/vehicle/VehicleArt.vue'
import { useAuthStore } from '../stores/auth'
import { useUiStore } from '../stores/ui'
import { useMockApi } from '../api'
import { asApiError } from '../api/errors'
const auth = useAuthStore(), ui = useUiStore(), route = useRoute(), router = useRouter()
const username = ref(useMockApi ? 'aurora.admin' : ''), password = ref(useMockApi ? 'demo12345' : ''), showPassword = ref(false), remember = ref(true), error = ref('')
async function submit() {
  error.value = ''
  if (!username.value || password.value.length < 8) { error.value = '请输入账号与至少 8 位密码'; return }
  try { await auth.login(username.value, password.value); await router.push(String(route.query.redirect || '/overview')) }
  catch (cause) { const problem = asApiError(cause); error.value = `${problem.message}${problem.requestId !== 'unavailable' ? ` · 请求 ${problem.requestId}` : ''}` }
}
</script>
<template>
  <main class="login-page">
    <section class="login-visual">
      <div class="visual-glow one"/><div class="visual-glow two"/>
      <header><AppLogo light /><span>运营指挥中心 <b>CN / 01</b></span></header>
      <div class="visual-copy"><div class="eyebrow">INTELLIGENCE IN MOTION</div><h1>让每一次连接<br>驱动<span>更远的可能。</span></h1><p>从车辆心跳到城市脉动，在一个安全、实时的运营空间中，洞悉整个车队。</p></div>
      <div class="login-car"><VehicleArt /><div class="car-data data-one"><span class="dot"/><div><small>FLEET ONLINE</small><strong>99.2%</strong></div></div><div class="car-data data-two"><Wifi /><div><small>TELEMETRY</small><strong>实时连接</strong></div></div></div>
      <div class="visual-footer"><div><strong>24 / 7</strong><span>全天候车辆洞察</span></div><i/><div><strong>E2E SECURE</strong><span>全链路加密连接</span></div><i/><div><strong>ONE PLATFORM</strong><span>多角色权限协同</span></div></div>
    </section>
    <section class="login-form-side">
      <button class="theme-login btn btn-ghost btn-icon" @click="ui.toggleTheme"><span v-if="ui.theme==='light'">◐</span><span v-else>☼</span></button>
      <div class="login-form-wrap">
        <div class="mobile-logo"><AppLogo /></div><div class="eyebrow">WELCOME BACK</div><h2>登录 AURORA</h2><p class="login-intro">进入您的车联网运营工作空间</p>
        <div v-if="useMockApi" class="demo-note"><span>DEMO</span> 演示工作空间已启用，登录凭证已预填。数据不代表真实车辆。</div>
        <form @submit.prevent="submit">
          <div class="field"><label for="username">账号</label><div class="login-input"><UserRound /><input id="username" v-model.trim="username" autocomplete="username" placeholder="用户名或企业邮箱" :disabled="auth.loading"></div></div>
          <div class="field"><div class="row-between"><label for="password">密码</label><button type="button" class="forgot" @click="error='请联系企业管理员重置密码。'">忘记密码？</button></div><div class="login-input"><LockKeyhole /><input id="password" v-model="password" :type="showPassword?'text':'password'" autocomplete="current-password" placeholder="输入至少 8 位密码" :disabled="auth.loading"><button type="button" aria-label="显示密码" @click="showPassword=!showPassword"><EyeOff v-if="showPassword"/><Eye v-else /></button></div></div>
          <label class="remember"><input v-model="remember" type="checkbox"><span><Check /></span>在此设备上保持偏好</label>
          <Transition name="fade"><div v-if="error" class="login-error" role="alert">{{ error }}</div></Transition>
          <button class="login-submit" type="submit" :disabled="auth.loading"><span v-if="auth.loading" class="spinner"/><template v-else>安全登录<ArrowRight /></template></button>
        </form>
        <div class="security-note"><ShieldCheck /><p>访问令牌仅保存在内存中。登录与会话刷新通过加密通道完成，所有操作均受服务端权限校验。</p></div>
        <p class="login-help">无法访问工作空间？<a href="mailto:ops@aurora.auto">联系运营支持</a></p>
      </div>
      <footer>© 2026 AURORA MOBILITY <span>隐私政策 · 安全白皮书</span></footer>
    </section>
  </main>
</template>
<style scoped>
.login-page{min-height:100vh;display:grid;grid-template-columns:minmax(480px,1.2fr) minmax(440px,.8fr);background:var(--surface-solid)}.login-visual{background:#111427;color:#fff;position:relative;overflow:hidden;padding:36px 46px;display:flex;flex-direction:column;isolation:isolate}.login-visual::before{content:'';position:absolute;inset:0;background-image:linear-gradient(rgba(255,255,255,.025) 1px,transparent 1px),linear-gradient(90deg,rgba(255,255,255,.025) 1px,transparent 1px);background-size:48px 48px;mask-image:linear-gradient(to bottom,black,transparent 80%);z-index:-1}.visual-glow{position:absolute;border-radius:50%;filter:blur(2px);z-index:-1}.visual-glow.one{width:600px;height:600px;left:-150px;top:10%;background:radial-gradient(circle,rgba(103,85,229,.2),transparent 68%)}.visual-glow.two{width:500px;height:500px;right:-100px;bottom:-150px;background:radial-gradient(circle,rgba(23,190,181,.13),transparent 65%)}.login-visual header{display:flex;justify-content:space-between;align-items:center}.login-visual header>span{font-size:9px;color:#727a95;display:flex;gap:13px;align-items:center}.login-visual header b{font-size:8px;color:#a1a8be;border:1px solid #2b3046;border-radius:5px;padding:4px 6px}.visual-copy{margin-top:10vh;max-width:520px}.visual-copy .eyebrow{color:#7edbd2}.visual-copy h1{font-size:clamp(34px,3.4vw,53px);line-height:1.22;font-weight:570;letter-spacing:-.055em}.visual-copy h1 span{background:linear-gradient(90deg,#a397ff,#61d8d0);background-clip:text;color:transparent}.visual-copy p{color:#888fa7;font-size:12px;line-height:1.9;max-width:430px;margin-top:20px}.login-car{width:min(570px,90%);margin:2vh auto 0;position:relative}.car-data{position:absolute;background:rgba(28,32,54,.72);border:1px solid rgba(255,255,255,.1);backdrop-filter:blur(12px);border-radius:11px;padding:10px 13px;display:flex;align-items:center;gap:10px;box-shadow:0 12px 30px rgba(0,0,0,.2)}.car-data>div{display:flex;flex-direction:column;gap:3px}.car-data small{font-size:6px;letter-spacing:.12em;color:#7b839e}.car-data strong{font-size:12px}.car-data>svg{width:17px;color:#59d4c8}.car-data>.dot{color:#50d3b1;box-shadow:0 0 0 4px rgba(80,211,177,.1)}.data-one{left:-10px;top:36%}.data-two{right:-5px;bottom:25%}.visual-footer{display:flex;align-items:center;justify-content:space-between;margin-top:auto;padding-top:20px;border-top:1px solid rgba(255,255,255,.07)}.visual-footer>div{display:flex;flex-direction:column;gap:5px}.visual-footer strong{font-size:10px;letter-spacing:.08em}.visual-footer span{font-size:8px;color:#6d748e}.visual-footer i{width:1px;height:25px;background:#292d41}.login-form-side{position:relative;display:flex;flex-direction:column;align-items:center;justify-content:center;padding:48px;background:var(--surface-solid)}.theme-login{position:absolute;right:25px;top:25px}.login-form-wrap{width:min(370px,100%)}.mobile-logo{display:none}.login-form-wrap h2{font-size:29px;font-weight:710}.login-intro{font-size:12px;color:var(--muted);margin-top:9px}.demo-note{font-size:9px;line-height:1.6;padding:10px 12px;background:var(--primary-soft);color:var(--primary);border-radius:9px;margin-top:22px}.demo-note span{font-size:7px;border:1px solid currentColor;border-radius:4px;padding:2px 4px;margin-right:5px}.login-form-wrap form{display:flex;flex-direction:column;gap:20px;margin-top:31px}.login-input{height:45px;border:1px solid var(--line-strong);border-radius:10px;display:flex;align-items:center;padding:0 13px;gap:10px;transition:border .2s,box-shadow .2s}.login-input:focus-within{border-color:var(--primary);box-shadow:0 0 0 3px color-mix(in srgb,var(--primary) 10%,transparent)}.login-input>svg{width:16px;color:var(--muted)}.login-input input{flex:1;min-width:0;border:0;background:transparent;outline:0;color:var(--text);font-size:12px}.login-input button{border:0;background:transparent;color:var(--muted);display:grid;place-items:center;cursor:pointer}.login-input button svg{width:15px}.forgot{border:0;background:transparent;color:var(--primary);font-size:10px;cursor:pointer}.remember{display:flex;align-items:center;gap:7px;font-size:10px;color:var(--text-2);cursor:pointer}.remember input{display:none}.remember>span{width:15px;height:15px;border:1px solid var(--line-strong);border-radius:4px;display:grid;place-items:center}.remember svg{width:10px;opacity:0}.remember input:checked+span{background:var(--primary);border-color:var(--primary);color:white}.remember input:checked+span svg{opacity:1}.login-submit{height:45px;border:0;border-radius:10px;background:linear-gradient(120deg,#6759e8,#8375f2);color:#fff;font-size:12px;font-weight:650;display:flex;align-items:center;justify-content:center;gap:10px;cursor:pointer;box-shadow:0 8px 20px rgba(103,89,232,.2);transition:all .2s}.login-submit:hover:not(:disabled){transform:translateY(-1px);box-shadow:0 10px 25px rgba(103,89,232,.3)}.login-submit:disabled{opacity:.65}.login-submit svg{width:16px}.login-submit .spinner{border-color:rgba(255,255,255,.25);border-top-color:#fff}.login-error{padding:10px 12px;border-radius:8px;background:var(--red-soft);color:var(--red);font-size:10px;line-height:1.5;margin-top:-8px}.security-note{display:flex;gap:10px;padding:14px;margin-top:26px;background:var(--bg-soft);border-radius:10px}.security-note svg{width:17px;flex:none;color:var(--green)}.security-note p{font-size:9px;color:var(--muted);line-height:1.65}.login-help{text-align:center;font-size:10px;color:var(--muted);margin-top:23px}.login-help a{color:var(--primary);margin-left:4px}.login-form-side>footer{position:absolute;bottom:24px;left:35px;right:35px;display:flex;justify-content:space-between;font-size:7px;color:var(--muted);letter-spacing:.06em}
@media(max-width:900px){.login-page{grid-template-columns:1fr}.login-visual{display:none}.mobile-logo{display:block;margin-bottom:45px}.login-form-side{min-height:100vh;padding:35px}.login-form-side>footer{position:static;margin-top:60px;width:100%}}
</style>
