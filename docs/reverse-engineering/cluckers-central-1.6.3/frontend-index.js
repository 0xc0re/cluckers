(function(){const a=document.createElement("link").relList;if(a&&a.supports&&a.supports("modulepreload"))return;for(const o of document.querySelectorAll('link[rel="modulepreload"]'))r(o);new MutationObserver(o=>{for(const i of o)if(i.type==="childList")for(const f of i.addedNodes)f.tagName==="LINK"&&f.rel==="modulepreload"&&r(f)}).observe(document,{childList:!0,subtree:!0});function n(o){const i={};return o.integrity&&(i.integrity=o.integrity),o.referrerPolicy&&(i.referrerPolicy=o.referrerPolicy),o.crossOrigin==="use-credentials"?i.credentials="include":o.crossOrigin==="anonymous"?i.credentials="omit":i.credentials="same-origin",i}function r(o){if(o.ep)return;o.ep=!0;const i=n(o);fetch(o.href,i)}})();function Xt(e,a=!1){return window.__TAURI_INTERNALS__.transformCallback(e,a)}async function l(e,a={},n){return window.__TAURI_INTERNALS__.invoke(e,a,n)}var xt;(function(e){e.WINDOW_RESIZED="tauri://resize",e.WINDOW_MOVED="tauri://move",e.WINDOW_CLOSE_REQUESTED="tauri://close-requested",e.WINDOW_DESTROYED="tauri://destroyed",e.WINDOW_FOCUS="tauri://focus",e.WINDOW_BLUR="tauri://blur",e.WINDOW_SCALE_FACTOR_CHANGED="tauri://scale-change",e.WINDOW_THEME_CHANGED="tauri://theme-changed",e.WINDOW_CREATED="tauri://window-created",e.WEBVIEW_CREATED="tauri://webview-created",e.DRAG_ENTER="tauri://drag-enter",e.DRAG_OVER="tauri://drag-over",e.DRAG_DROP="tauri://drag-drop",e.DRAG_LEAVE="tauri://drag-leave"})(xt||(xt={}));async function Zt(e,a){window.__TAURI_EVENT_PLUGIN_INTERNALS__.unregisterListener(e,a),await l("plugin:event|unlisten",{event:e,eventId:a})}async function Et(e,a,n){var r;const o=(r=void 0)!==null&&r!==void 0?r:{kind:"Any"};return l("plugin:event|listen",{event:e,target:o,handler:Xt(a)}).then(i=>async()=>Zt(e,i))}async function te(e,a={},n){return window.__TAURI_INTERNALS__.invoke(e,a,n)}async function ee(e={}){return typeof e=="object"&&Object.freeze(e),await te("plugin:dialog|open",{options:e})}async function ae(e,a={},n){return window.__TAURI_INTERNALS__.invoke(e,a,n)}async function ne(){await ae("plugin:process|restart")}function w(e,a,n,r){if(typeof a=="function"?e!==a||!0:!a.has(e))throw new TypeError("Cannot read private member from an object whose class did not declare it");return n==="m"?r:n==="a"?r.call(e):r?r.value:a.get(e)}function V(e,a,n,r,o){if(typeof a=="function"?e!==a||!0:!a.has(e))throw new TypeError("Cannot write private member to an object whose class did not declare it");return a.set(e,n),n}var P,_,U,F;const Ct="__TAURI_TO_IPC_KEY__";function se(e,a=!1){return window.__TAURI_INTERNALS__.transformCallback(e,a)}class Nt{constructor(){this.__TAURI_CHANNEL_MARKER__=!0,P.set(this,()=>{}),_.set(this,0),U.set(this,[]),this.id=se(({message:a,id:n})=>{if(n==w(this,_,"f"))for(w(this,P,"f").call(this,a),V(this,_,w(this,_,"f")+1);w(this,_,"f")in w(this,U,"f");){const r=w(this,U,"f")[w(this,_,"f")];w(this,P,"f").call(this,r),delete w(this,U,"f")[w(this,_,"f")],V(this,_,w(this,_,"f")+1)}else w(this,U,"f")[n]=a})}set onmessage(a){V(this,P,a)}get onmessage(){return w(this,P,"f")}[(P=new WeakMap,_=new WeakMap,U=new WeakMap,Ct)](){return`__CHANNEL__:${this.id}`}toJSON(){return this[Ct]()}}async function $(e,a={},n){return window.__TAURI_INTERNALS__.invoke(e,a,n)}class At{get rid(){return w(this,F,"f")}constructor(a){F.set(this,void 0),V(this,F,a)}async close(){return $("plugin:resources|close",{rid:this.rid})}}F=new WeakMap;class re extends At{constructor(a){super(a.rid),this.available=!0,this.currentVersion=a.currentVersion,this.version=a.version,this.date=a.date,this.body=a.body,this.rawJson=a.rawJson}async download(a,n){const r=new Nt;a&&(r.onmessage=a);const o=await $("plugin:updater|download",{onEvent:r,rid:this.rid,...n});this.downloadedBytes=new At(o)}async install(){if(!this.downloadedBytes)throw new Error("Update.install called before Update.download");await $("plugin:updater|install",{updateRid:this.rid,bytesRid:this.downloadedBytes.rid}),this.downloadedBytes=void 0}async downloadAndInstall(a,n){const r=new Nt;a&&(r.onmessage=a),await $("plugin:updater|download_and_install",{onEvent:r,rid:this.rid,...n})}async close(){var a;await((a=this.downloadedBytes)==null?void 0:a.close()),await super.close()}}async function $t(e){const a=await $("plugin:updater|check",{...e});return a?new re(a):null}const Dt=[{code:"INT",label:"English"},{code:"CHN",label:"简体中文"},{code:"CHT",label:"繁體中文"},{code:"DEU",label:"Deutsch"},{code:"ESL",label:"Español (Latinoamérica)"},{code:"ESN",label:"Español (España)"},{code:"FRA",label:"Français"},{code:"JPN",label:"日本語"},{code:"KOR",label:"한국어"},{code:"POL",label:"Polski"},{code:"POR",label:"Português (Brasil)"},{code:"RUS",label:"Русский"},{code:"TUR",label:"Türkçe"}],Z="INT",oe=!0,ie=12e4,t={view:"auth",busy:!1,username:"",password:"",passwordResetCode:"",linkCode:"",loginToken:"",installPath:"",latestVersion:"",baseUrl:"",filesStatus:"",verifiedOk:!1,progressPct:0,gatewayStatus:{text:"Server: Checking…",tone:"info"},authStatus:{text:"",tone:"neutral"},linkStatus:{text:"DM the code to the bot before it expires. We'll auto-check every 3 seconds.",tone:"neutral"},pinStatus:{text:"",tone:"neutral"},verifyStatus:{text:"—",tone:"neutral"},downloadStatus:{text:"—",tone:"neutral"},launchStatus:{text:"",tone:"neutral"},launcherUpdate:{available:!1,version:"",busy:!1,status:{text:"",tone:"neutral"}},supporterTierName:"none",supporterTierLevel:0,supporterBotSlotsTotal:0,supporterBotSlotsUsed:0,supporterPerksPreview:"",supporterAnnouncementText:"",supporterAnnouncementSeconds:0,supporterBotNames:[],devCommandLineEnabled:!1,devCommandLineArgs:"",devCommandLineStatus:{text:"",tone:"neutral"},devBypassVerification:!1,devBypassFileChecks:!1,gameLanguage:Z,gameLanguageStatus:{text:"",tone:"neutral"}},u=e=>document.getElementById(e),Ot=(e,a)=>{const n=u(e);if(!n)return;n.textContent=a;const r=n;r.dataset.title==="auto"&&(a?r.setAttribute("title",a):r.removeAttribute("title"))},T=e=>{var a;return(((a=u(e))==null?void 0:a.value)??"").trim()},ue=(e,a,n)=>Math.max(a,Math.min(n,e)),le=80;let J=null;function O(e,a){const n=a.text?"":"data-empty";return`<div id="${e}" class="status status-inline" data-tone="${a.tone}" ${n}>${v(a.text)}</div>`}function tt(){if(!t.launcherUpdate.available)return"";const e=t.launcherUpdate.busy?"Updating…":`Update required (v${v(t.launcherUpdate.version)})`;return`<button id="btn-update" class="ghost" type="button" ${t.launcherUpdate.busy?"disabled":""}>${e}</button>`}async function Tt(){try{if(t.launcherUpdate.busy)return;const e=await $t();if(J=e,!e||t.launcherUpdate.available&&t.launcherUpdate.version===e.version)return;t.launcherUpdate.available=!0,t.launcherUpdate.version=e.version,t.launcherUpdate.status={text:`Update available: v${e.version}`,tone:"info"},h(),await y(),oe&&!t.launcherUpdate.busy&&Bt()}catch{}}async function Bt(){if(t.launcherUpdate.busy)return;c(!0),t.launcherUpdate.busy=!0,t.launcherUpdate.status={text:"Preparing update…",tone:"info"},h();let e=null,a=0;const n=r=>{var o;if(r.event==="Started")e=typeof((o=r.data)==null?void 0:o.contentLength)=="number"?r.data.contentLength:null,a=0,t.launcherUpdate.status={text:"Downloading update…",tone:"info"};else if(r.event==="Progress"){a+=r.data.chunkLength??0;const i=e?Math.min(100,Math.round(a/e*100)):null;t.launcherUpdate.status={text:i!=null?`Downloading update… (${i}%)`:"Downloading update…",tone:"info"}}else r.event==="Finished"&&(t.launcherUpdate.status={text:"Installing update…",tone:"info"});s("update-status",t.launcherUpdate.status)};try{const r=J??await $t();if(J=r,!r){t.launcherUpdate.available=!1,t.launcherUpdate.version="",t.launcherUpdate.status={text:"No update available.",tone:"success"},h(),await y();return}await r.downloadAndInstall(n),await ne()}catch(r){t.launcherUpdate.status={text:`Update failed: ${String(r)}`,tone:"danger"},h()}finally{t.launcherUpdate.busy=!1,c(!1),h(),await y()}}const de=`
  <span class="btn-icon kofi-icon" aria-hidden="true">
    <svg viewBox="0 0 24 24" focusable="false">
      <path d="M4 5h12a1 1 0 0 1 1 1v3a5 5 0 0 1-5 5H7a3 3 0 0 1-3-3V6a1 1 0 0 1 1-1z" fill="currentColor" opacity="0.2" />
      <path d="M4 5h12a1 1 0 0 1 1 1v3a5 5 0 0 1-5 5H7a3 3 0 0 1-3-3V6a1 1 0 0 1 1-1z" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linejoin="round" />
      <path d="M17 7h1a3 3 0 0 1 0 6h-2" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" />
      <g transform="translate(10 10.8) scale(0.88) translate(-10 -10.8) translate(-0.32 -0.32)">
        <path d="M10 12.6c-1.6-1-2.8-2-2.8-3.2 0-1 .8-1.9 1.9-1.9 .8 0 1.4.4 1.7 1.1 .3-.7 1-1.1 1.7-1.1 1.1 0 1.9.9 1.9 1.9 0 1.2-1.2 2.2-2.8 3.2l-.9.6-.9-.6z" fill="currentColor" />
      </g>
    </svg>
  </span>
`,ce=`
  <span class="btn-icon" aria-hidden="true">
    <svg viewBox="0 0 24 24" focusable="false">
      <path d="M20.317 4.37a19.8 19.8 0 0 0-4.885-1.515.08.08 0 0 0-.079.037 19.8 19.8 0 0 0-.618 1.269 18.8 18.8 0 0 0-5.487 0 19.8 19.8 0 0 0-.626-1.269.08.08 0 0 0-.079-.037 19.8 19.8 0 0 0-4.885 1.515.07.07 0 0 0-.032.028C.534 9.046-.319 13.58.099 18.058a.08.08 0 0 0 .031.056 19.8 19.8 0 0 0 5.993 3.054.08.08 0 0 0 .084-.028 14.3 14.3 0 0 0 1.227-1.994.08.08 0 0 0-.042-.106 13.1 13.1 0 0 1-1.872-.892.08.08 0 0 1-.008-.128c.126-.094.252-.192.372-.291a.07.07 0 0 1 .078-.011c3.928 1.793 8.18 1.793 12.061 0a.07.07 0 0 1 .079.01c.12.099.246.198.372.292a.08.08 0 0 1-.007.128 12.3 12.3 0 0 1-1.873.891.08.08 0 0 0-.041.107 14.6 14.6 0 0 0 1.227 1.993.08.08 0 0 0 .084.029 19.9 19.9 0 0 0 6.002-3.054.08.08 0 0 0 .031-.055c.5-5.177-.838-9.674-3.548-13.66a.06.06 0 0 0-.032-.029z" fill="currentColor" />
      <circle cx="8.9" cy="12" r="1.55" fill="#ffffff" />
      <circle cx="15.1" cy="12" r="1.55" fill="#ffffff" />
    </svg>
  </span>
`;function z(e){return`
    <header class="global-header">
      <div class="global-header-inner">
        <div class="brand-row">
          <img class="brand-logo" src="/finallogo.png" alt="Cluckers Central" />
          <div>
            <div class="brand-title">Cluckers Central</div>
            <div class="brand-sub">Realm Royale Private Servers • v<span id="app-version">?</span></div>
          </div>
        </div>
        <div class="global-header-actions">
          <div class="header-stack">
            <div id="gateway-status" class="header-action header-status" data-tone="${t.gatewayStatus.tone}">${v(t.gatewayStatus.text)}</div>
            ${e||'<div class="header-spacer"></div>'}
          </div>
          <div class="header-stack">
            <button id="btn-kofi" class="header-action" type="button">${de}Support</button>
            <button id="btn-community" class="header-action" type="button">${ce}Discord</button>
          </div>
          ${O("update-status",t.launcherUpdate.status)}
          ${tt()}
        </div>
      </div>
    </header>
  `}function s(e,a){const n=u(e);if(!n)return;const r=n;n.textContent=a.text,r.dataset.tone=a.tone,r.toggleAttribute("data-empty",!a.text),a.text?r.setAttribute("title",a.text):r.removeAttribute("title")}function R(e){t.progressPct=ue(e,0,100);const a=u("progress-bar");a&&a.style.setProperty("--progress-scale",`${t.progressPct/100}`);const n=document.querySelector(".console-progress");n&&n.toggleAttribute("data-active",q()),Ot("progress-pct",`${Math.round(t.progressPct)}%`);const r=u("progress-pct");r&&r.toggleAttribute("data-empty",!q())}function pe(e,a){if(!e)return"";if(e.length<=a)return e;const n=Math.max(0,a-3);return`${e.slice(0,n).trimEnd()}...`}function Mt(e){return e?pe(e,le):"Choose install folder…"}function Vt(e){const a=u("install-path-text");if(!a)return;const n=(e==null?void 0:e.trim())??"",r=Mt(n);a.textContent=r,n?a.setAttribute("title",n):a.removeAttribute("title")}function v(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#39;")}function x(e){return e&&typeof e=="object"?e:{}}function E(e){return e.SUCCESS===1}function d(e,a){const n=e[a];return typeof n=="string"?n:""}function b(e,a){const n=e[a];if(typeof n=="number"&&Number.isFinite(n))return n;if(typeof n=="string"&&n.trim()){const r=Number(n);if(Number.isFinite(r))return r}return 0}function Ft(e){return e.trim().toLowerCase().replace(/\s+/g,"_")}function fe(e){switch(Ft(e)){case"cluck_overlord":case"overlord":return 4;case"cluck_commander":case"commander":case"supporter":return 3;case"cluck_soldier":case"soldier":return 2;case"cluckling":return 1;default:return 0}}function Gt(e){return e>=4?2:e>=3?1:0}function ve(e){const a=Ft(e);return!a||a==="none"?"None":a.split("_").filter(n=>!!n).map(n=>n.charAt(0).toUpperCase()+n.slice(1)).join(" ")}function Wt(e,a){const n=b(e,"CUSTOM_VALUE_1");return n>0?n:fe(d(e,"CUSTOM_MESSAGE")||a)}function et(e){if(!e)return[];try{const a=JSON.parse(e);return Array.isArray(a)?a.filter(n=>typeof n=="string"):[]}catch{return[]}}function Ht(e){t.supporterTierName=d(e,"CUSTOM_MESSAGE")||"none",t.supporterTierLevel=Wt(e,t.supporterTierName),t.supporterBotSlotsTotal=Math.max(b(e,"CUSTOM_VALUE_2"),Gt(t.supporterTierLevel)),t.supporterBotSlotsUsed=b(e,"CUSTOM_VALUE_3"),t.supporterPerksPreview=d(e,"PORTAL_INFO_1"),t.supporterAnnouncementText=d(e,"TEXT_VALUE"),t.supporterAnnouncementSeconds=b(e,"CUSTOM_VALUE_4")}function at(e){t.supporterTierName=d(e,"CUSTOM_MESSAGE")||t.supporterTierName||"none",t.supporterTierLevel=Wt(e,t.supporterTierName||"none"),t.supporterBotSlotsTotal=Math.max(b(e,"CUSTOM_VALUE_2"),Gt(t.supporterTierLevel)),t.supporterBotSlotsUsed=b(e,"CUSTOM_VALUE_3")}async function Kt(){if(t.loginToken){if(t.supporterTierLevel<3){t.supporterBotNames=[];return}try{const e=x(await l("launcher_supporter_bot_names_list",{accessToken:t.loginToken}));if(!E(e))return;at(e),t.supporterBotNames=et(d(e,"PORTAL_INFO_1"))}catch{}}}function c(e){t.busy=e;const a=u("app");a&&a.toggleAttribute("data-busy",e)}function he(){try{t.username=localStorage.getItem("cc.username")??""}catch{}}function B(){try{localStorage.setItem("cc.username",t.username)}catch{}}function ge(e){switch(e){case"upToDate":return"success";case"notInstalled":case"updateRequired":return"warning";case"corruptOrModified":return"danger";default:return"neutral"}}function K(e){const a=(e??"").trim().toUpperCase();return Dt.some(n=>n.code===a)?a:Z}function Se(e){const a=Dt.find(n=>n.code===K(e));return a?a.label:"English"}function qt(){return!!t.loginToken&&!!t.installPath&&(t.devBypassVerification||t.devBypassFileChecks||t.verifiedOk)}function q(){return t.progressPct>0&&(t.verifyStatus.tone==="info"||t.downloadStatus.tone==="info")}function L(e){return e.text==="—"?{...e,text:""}:e}function N(){Vt(t.installPath),s("verify-status",L(t.verifyStatus)),s("download-status",L(t.downloadStatus)),s("launch-status",L(t.launchStatus)),s("dev-cli-status",t.devCommandLineStatus),s("game-language-status",t.gameLanguageStatus),R(t.progressPct);const e=u("btn-launch"),a=document.querySelector(".console-progress"),n=qt();e&&(e.disabled=!n),a&&a.toggleAttribute("data-active",q())}function zt(){const e=document.querySelector(".supporter-card .summary-meta");e&&(e.textContent=`${t.supporterBotSlotsUsed}/${t.supporterBotSlotsTotal} bot slots`);for(let a=1;a<=Math.max(1,t.supporterBotSlotsTotal);a++){const n=u(`supporter-bot-name-${a}`);n&&(n.value=t.supporterBotNames[a-1]??"")}N()}async function S(){try{const e=await l("get_app_version");Ot("app-version",e)}catch{}try{const e=await l("get_install_path");t.installPath=e,Vt(e)}catch{}}async function Y(){try{const e=x(await l("launcher_health"));if(!E(e)){t.gatewayStatus={text:"Server: Offline",tone:"danger"},s("gateway-status",t.gatewayStatus),t.view!=="auth"&&Q("Server offline. Please try again later.");return}const a=b(e,"READY_FLAG")===1;t.gatewayStatus=a?{text:"Server: Online",tone:"success"}:{text:"Server: Degraded",tone:"warning"},s("gateway-status",t.gatewayStatus)}catch{t.gatewayStatus={text:"Server: Offline",tone:"danger"},s("gateway-status",t.gatewayStatus),t.view!=="auth"&&Q("Server offline. Please try again later.")}}function Q(e){t.view==="auth"&&t.authStatus.text===e||(nt(),t.loginToken="",t.password="",t.linkCode="",t.passwordResetCode="",t.latestVersion="",t.baseUrl="",t.filesStatus="",t.verifiedOk=!1,t.progressPct=0,t.verifyStatus={text:"—",tone:"neutral"},t.downloadStatus={text:"—",tone:"neutral"},t.launchStatus={text:"",tone:"neutral"},t.linkStatus={text:"DM the code to the bot before it expires. We'll auto-check every 3 seconds.",tone:"neutral"},t.pinStatus={text:"",tone:"neutral"},t.view="auth",t.authStatus={text:e,tone:"danger"},h(),S(),y())}let m=null,G=null,W=800,I=null;const ye=3e3;let D=null,H=!1,jt=0;function me(e){t.linkStatus.text===e.text&&t.linkStatus.tone===e.tone||(t.linkStatus=e,s("link-status",t.linkStatus))}function nt(){D!=null&&(window.clearInterval(D),D=null),H=!1,jt=0}async function Pt(){if(!document.hidden&&t.view==="link"&&!(!t.username||!t.password)&&!t.busy&&!H&&!(Date.now()<jt)){H=!0;try{await Qt(!1)}catch{me({text:"Server unavailable. Retrying…",tone:"danger"})}finally{H=!1}}}function we(){D==null&&(document.hidden||t.view==="link"&&(!t.username||!t.password||(D=window.setInterval(()=>{Pt()},ye),Pt())))}function Jt(){if(document.hidden||t.view!=="link"||!t.username||!t.password){nt();return}we()}function be(){if(I!=null&&(window.clearTimeout(I),I=null),m){try{m.onopen=null,m.onmessage=null,m.onerror=null,m.onclose=null,m.close()}catch{}m=null}}function Ut(){if(!G||I!=null)return;const e=W;W=Math.min(6e4,Math.round(W*1.8)),I=window.setTimeout(()=>{I=null,Yt()},e)}async function Yt(){if(!G)try{const e=await l("get_api_config");if(!(e!=null&&e.ws_base_url))return;G=String(e.ws_base_url)}catch{return}be();try{m=new WebSocket(G)}catch{Ut();return}m.onopen=()=>{W=800,t.gatewayStatus={text:"Server: Connected",tone:"info"},s("gateway-status",t.gatewayStatus)},m.onmessage=e=>{const a=typeof e.data=="string"?e.data:"";if(a)try{const n=JSON.parse(a);if((typeof(n==null?void 0:n.type)=="string"?n.type:"")==="gateway_status"){const o=(typeof n.ready=="number"?n.ready:0)===1,i=typeof n.online=="boolean"?n.online:!0;t.gatewayStatus=i?o?{text:"Server: Online",tone:"success"}:{text:"Server: Degraded",tone:"warning"}:{text:"Server: Offline",tone:"danger"},s("gateway-status",t.gatewayStatus),!i&&t.view!=="auth"&&Q("Server offline. Please try again later.")}}catch{}},m.onerror=()=>{},m.onclose=()=>{m=null,t.gatewayStatus={text:"Server: Checking…",tone:"info"},s("gateway-status",t.gatewayStatus),Y(),Ut()}}function ke(){return`
    <div class="shell auth-shell">
      ${z()}

      <main class="stage auth-stage">
        <section class="auth-single">
          <section class="panel auth-card auth-card-main">
            <div class="panel-head">
              <h2 class="panel-title">Sign in</h2>
            </div>

            <div class="form">
              <div class="auth-grid">
                <label class="field">
                  <span>Username</span>
                  <input id="user_name" autocomplete="username" spellcheck="false" value="${v(t.username)}" />
                </label>

                <label class="field">
                  <span>Password</span>
                  <input id="password" type="password" autocomplete="current-password" value="${v(t.password)}" />
                </label>
              </div>

              <div class="actions">
                <button id="btn-login" class="primary" type="button">Sign in</button>
                <button id="btn-register" type="button">Create account</button>
                <button id="btn-forgot-password" class="text-action" type="button">Forgot password</button>
              </div>

              ${t.passwordResetCode?`
                <div class="copy-card">
                  <div class="copy-label">Password reset code</div>
                  <div class="copy-row">
                    <div id="password-reset-code" class="code mono">${v(t.passwordResetCode)}</div>
                    <button id="btn-copy-reset-code" type="button">Copy</button>
                  </div>
                  <div class="subtle">DM this code to the Discord bot, then reply in DM with your new password.</div>
                  <div class="actions">
                    <button id="btn-open-dm-auth" type="button">Open Discord DM</button>
                  </div>
                </div>
              `:""}

              <div class="status" id="auth-status" data-tone="${t.authStatus.tone}">
                ${v(t.authStatus.text)}
              </div>
            </div>
          </section>
        </section>
      </main>
    </div>
  `}function _e(){return`
    <div class="shell">
      <header class="topbar">
        <div class="brand">
          <div class="brand-row">
            <img class="brand-logo" src="/finallogo.png" alt="Cluckers Central" />
            <div>
              <div class="brand-title">Cluckers Central</div>
              <div class="brand-sub">Discord linking • v<span id="app-version">?</span></div>
            </div>
          </div>
        </div>
        <div class="top-actions">
          ${O("gateway-status",t.gatewayStatus)}
          ${O("update-status",t.launcherUpdate.status)}
          ${tt()}
          <button id="btn-back" type="button">Back</button>
        </div>
      </header>

      <main class="stage">
        <section class="panel flow-panel">
          <div class="panel-head">
            <div>
              <div class="eyebrow">Discord link</div>
              <h1>Link your Discord</h1>
              <div class="subtle">You must link your Discord account and be in the server to continue. Link codes expire automatically.</div>
            </div>
          </div>

          <div class="copy-card">
            <div class="copy-label">Your link code</div>
            <div class="copy-row">
              <div id="link-code" class="code mono">${t.linkCode?v(t.linkCode):"—"}</div>
              <button id="btn-copy-code" type="button">Copy</button>
            </div>
          </div>

          <div class="actions">
            <button id="btn-open-dm" class="primary" type="button">Open Discord DM</button>
            <button id="btn-join-discord" type="button">Join Discord server</button>
            <button id="btn-new-code" type="button">New code</button>
          </div>

          <div class="status" id="link-status" data-tone="${t.linkStatus.tone}">
            ${v(t.linkStatus.text)}
          </div>
        </section>
      </main>

      ${z()}
    </div>
  `}function Le(){return`
    <div class="shell">
      <header class="topbar">
        <div class="brand">
          <div class="brand-row">
            <img class="brand-logo" src="/finallogo.png" alt="Cluckers Central" />
            <div>
              <div class="brand-title">Cluckers Central</div>
              <div class="brand-sub">Developer access • v<span id="app-version">?</span></div>
            </div>
          </div>
        </div>
        <div class="top-actions">
          ${O("gateway-status",t.gatewayStatus)}
          ${O("update-status",t.launcherUpdate.status)}
          ${tt()}
          <button id="btn-pin-back" type="button">Back</button>
        </div>
      </header>

      <main class="stage">
        <section class="panel narrow flow-panel">
          <div class="panel-head">
            <div>
              <div class="eyebrow">Developer gate</div>
              <h1>Enter the developer PIN</h1>
              <div class="subtle">This server is in dev-only mode. Enter the PIN to access the launcher.</div>
            </div>
          </div>

          <div class="form">
            <label class="field">
              <span>PIN</span>
              <input id="dev_pin" type="password" autocomplete="one-time-code" />
            </label>

            <div class="actions">
              <button id="btn-pin-submit" class="primary" type="button">Continue</button>
            </div>

            <div class="status" id="pin-status" data-tone="${t.pinStatus.tone}">
              ${v(t.pinStatus.text)}
            </div>
          </div>
        </section>
      </main>

      ${z()}
    </div>
  `}function xe(){const e=qt(),a=t.supporterTierLevel>=3,n=t.supporterTierLevel>0,r=q(),o=L(t.verifyStatus),i=L(t.downloadStatus),f=L(t.launchStatus),g=a?Array.from({length:Math.max(1,t.supporterBotSlotsTotal)},(A,C)=>{const k=C+1,M=t.supporterBotNames[C]??"";return`
          <div class="supporter-slot">
            <div class="supporter-slot-head">Slot ${k}</div>
            <div class="supporter-slot-actions">
              <input id="supporter-bot-name-${k}" class="supporter-bot-input" value="${v(M)}" maxlength="32" />
              <button id="btn-bot-save-${k}" type="button">Save</button>
              <button id="btn-bot-delete-${k}" type="button">Clear</button>
            </div>
          </div>
        `}).join(""):"";return`
    <div class="shell post-login-shell">
      ${z('<button id="btn-logout" type="button">Logout</button>')}

      <main class="stage post-login-stage">
        <div class="main-stack post-login-stack">
          <section class="launcher-dashboard" aria-label="Game launcher">
            <div class="launcher-core">
              <section class="launcher-console" aria-label="Install and file management">
                <div class="setup-row install-workflow">
                  <div class="path-summary">
                    <span>Install folder</span>
                    <div id="install-path-text" class="path-text mono" data-title="auto">
                      ${v(Mt(t.installPath))}
                    </div>
                  </div>

                  <div class="module-actions">
                    <button id="btn-browse-install" class="path-action" type="button">Browse</button>
                    <button id="btn-verify" type="button">Verify</button>
                    <button id="btn-download" type="button" ${t.devBypassFileChecks?"disabled":""}>Repair</button>
                  </div>

                </div>

                <div class="progress console-progress" ${r?"data-active":""}>
                  <div id="progress-bar" class="progress-inner" style="--progress-scale: ${t.progressPct/100}"></div>
                </div>

                <div class="console-status-row">
                  <div class="launcher-state">
                    <div id="verify-status" class="meter-value mono" data-tone="${o.tone}" ${o.text?"":"data-empty"}>
                      ${v(o.text)}
                    </div>
                    <div id="download-status" class="meter-value mono" data-tone="${i.tone}" ${i.text?"":"data-empty"}>
                      ${v(i.text)}
                    </div>
                  </div>

                  <div class="launch-status-compact">
                    <strong id="progress-pct" class="mono" ${r?"":"data-empty"}>${Math.round(t.progressPct)}%</strong>
                    <div class="status" id="launch-status" data-tone="${f.tone}" ${f.text?"":"data-empty"}>
                      ${v(f.text)}
                    </div>
                  </div>

                  <div class="launch-inline">
                    <button id="btn-launch" class="primary mega launch-primary" type="button" ${e?"":"disabled"}>
                      <span class="btn-label">Launch</span>
                      <span class="launch-glyph" aria-hidden="true">&gt;</span>
                    </button>
                  </div>
                </div>
              </section>
            </div>
          </section>

          ${n?`
            <details class="utility-panel supporter-card collapsed-panel">
              <summary>
                <span class="supporter-summary-main">
                  <span class="utility-title">Supporter</span>
                  <span class="supporter-role">${v(ve(t.supporterTierName))}</span>
                </span>
                <span class="summary-meta mono">${t.supporterBotSlotsUsed}/${t.supporterBotSlotsTotal} bot slots</span>
              </summary>
              <div class="collapsed-body">
                ${t.supporterPerksPreview?`<div class="supporter-preview"><strong>Unlocked:</strong> ${v(t.supporterPerksPreview)}</div>`:""}
                ${a?`<div class="supporter-slots">${g}</div>`:""}
              </div>
            </details>
          `:""}

          ${t.devCommandLineEnabled?`
            <details class="utility-panel dev-cli-card collapsed-panel">
              <summary>
                <span>
                  <span class="utility-title">Launch arguments</span>
                </span>
                <span class="summary-meta">optional</span>
              </summary>
              <div class="collapsed-body">
                <div class="subtle">Appended to the default args. Auth/bootstrap args are blocked.</div>
                <div class="dev-cli-row">
                  <input
                    id="dev-command-line-args"
                    class="dev-cli-input mono"
                    value="${v(t.devCommandLineArgs)}"
                    placeholder="-log -windowed"
                    autocomplete="off"
                    spellcheck="false"
                  />
                  <button id="btn-dev-cli-save" type="button">Save</button>
                  <button id="btn-dev-cli-clear" type="button">Clear</button>
                </div>
                <div class="status" id="dev-cli-status" data-tone="${t.devCommandLineStatus.tone}">
                  ${v(t.devCommandLineStatus.text)}
                </div>
              </div>
            </details>
          `:""}
        </div>
      </main>
    </div>
  `}function h(){const e=u("app");e&&(t.view==="auth"?e.innerHTML=ke():t.view==="link"?e.innerHTML=_e():t.view==="pin"?e.innerHTML=Le():e.innerHTML=xe(),s("gateway-status",t.gatewayStatus),s("auth-status",t.authStatus),s("link-status",t.linkStatus),s("pin-status",t.pinStatus),s("verify-status",L(t.verifyStatus)),s("download-status",L(t.downloadStatus)),s("launch-status",L(t.launchStatus)),s("dev-cli-status",t.devCommandLineStatus),s("game-language-status",t.gameLanguageStatus),s("update-status",t.launcherUpdate.status),R(t.progressPct),t.view==="main"&&N(),Jt(),c(t.busy))}function Ee(){document.querySelectorAll("details.collapsed-panel").forEach(e=>{e.dataset.collapseBound!=="1"&&(e.dataset.collapseBound="1")})}async function st(e,a,n){if(!e||!a){t.authStatus={text:"Enter username and password.",tone:"warning"},s("auth-status",t.authStatus);return}t.username=e,t.password=a,t.passwordResetCode="",B(),c(!0),t.authStatus={text:"Signing in…",tone:"info"},s("auth-status",t.authStatus);let r=!1;try{const o={userName:e,password:a};n&&(o.pin=n);const i=x(await l("launcher_login_or_link",o));if(!E(i)){const k=d(i,"STRING_VALUE")||"Login failed";n||t.view==="pin"?(t.pinStatus={text:k,tone:"danger"},s("pin-status",t.pinStatus)):(t.authStatus={text:k,tone:"danger"},s("auth-status",t.authStatus));return}const f=b(i,"LINKED_FLAG")===1,g=(d(i,"STRING_VALUE")||"").trim();if(f&&(g==="PIN_REQUIRED"||g==="PIN_INVALID")){t.loginToken="",t.pinStatus={text:g==="PIN_INVALID"?"Invalid PIN. Try again.":"Enter the developer PIN to continue.",tone:g==="PIN_INVALID"?"danger":"info"},t.view="pin",h(),await S(),await y();return}const A=d(i,"ACCESS_TOKEN");if(!f){t.linkCode=A,t.supporterTierName="none",t.supporterTierLevel=0,t.supporterBotSlotsTotal=0,t.supporterBotSlotsUsed=0,t.supporterPerksPreview="",t.supporterAnnouncementText="",t.supporterAnnouncementSeconds=0,t.supporterBotNames=[],t.pinStatus={text:"",tone:"neutral"},t.linkStatus={text:"DM the code to the bot before it expires. We'll auto-check every 3 seconds.",tone:"neutral"},t.view="link",t.verifiedOk=!1,h(),await S(),await y();return}t.loginToken=A;const C=d(i,"USER_NAME");C&&(t.username=C,B()),Ht(i),t.supporterBotNames=[],t.pinStatus={text:"",tone:"neutral"},t.view="main",t.verifiedOk=!1,t.filesStatus="",t.latestVersion="",t.baseUrl="",t.progressPct=0,t.verifyStatus={text:"Preparing verification…",tone:"info"},t.downloadStatus={text:"—",tone:"neutral"},t.launchStatus={text:"",tone:"neutral"},await rt(),await ot(),await it(),h(),await Kt(),await S(),await y(),r=!0}catch(o){const f=(typeof o=="string"?o:o instanceof Error?o.message:String(o)).match(/HTTP\s+(\d{3})[^:]*:\s*(.+)$/);let g;f?g=f[1]==="401"?"Invalid username or password.":f[2].trim():g="Login failed. The game server is most likely unreachable.",n||t.view==="pin"?(t.pinStatus={text:g,tone:"danger"},s("pin-status",t.pinStatus)):(t.authStatus={text:g,tone:"danger"},s("auth-status",t.authStatus))}finally{c(!1),r&&j(!0)}}async function It(){await st(T("user_name"),T("password"))}async function Rt(){const e=T("dev_pin");if(!t.username||!t.password){t.pinStatus={text:"Go back and sign in first.",tone:"warning"},s("pin-status",t.pinStatus);return}if(!e){t.pinStatus={text:"Enter the developer PIN.",tone:"warning"},s("pin-status",t.pinStatus);return}t.pinStatus={text:"Checking PIN…",tone:"info"},s("pin-status",t.pinStatus),await st(t.username,t.password,e)}async function Ce(){const e=T("user_name"),a=T("password");if(!e||!a){t.authStatus={text:"Enter username and password.",tone:"warning"},s("auth-status",t.authStatus);return}t.username=e,t.password=a,t.passwordResetCode="",B(),c(!0),t.authStatus={text:"Creating account…",tone:"info"},s("auth-status",t.authStatus);try{const n=`${e}@example.com`,r=x(await l("launcher_register",{userName:e,password:a,email:n}));if(!E(r)){t.authStatus={text:d(r,"STRING_VALUE")||"Registration failed",tone:"danger"},s("auth-status",t.authStatus);return}const o=b(r,"LINKED_FLAG")===1,i=d(r,"ACCESS_TOKEN");if(!o&&i){t.linkCode=i,t.supporterTierName="none",t.supporterTierLevel=0,t.supporterBotSlotsTotal=0,t.supporterBotSlotsUsed=0,t.supporterPerksPreview="",t.supporterAnnouncementText="",t.supporterAnnouncementSeconds=0,t.supporterBotNames=[],t.pinStatus={text:"",tone:"neutral"},t.linkStatus={text:"DM the code to the bot before it expires. We'll auto-check every 3 seconds.",tone:"neutral"},t.view="link",t.verifiedOk=!1,h(),await S(),await y();return}t.authStatus={text:"Account created. Signing in…",tone:"success"},s("auth-status",t.authStatus)}catch(n){t.authStatus={text:`Registration failed: ${String(n)}`,tone:"danger"},s("auth-status",t.authStatus);return}finally{c(!1)}await st(e,a)}async function Ne(){const e=T("user_name");if(!e){t.authStatus={text:"Enter username first.",tone:"warning"},s("auth-status",t.authStatus);return}t.username=e,B(),c(!0),t.authStatus={text:"Generating password reset code…",tone:"info"},s("auth-status",t.authStatus);try{const a=x(await l("launcher_request_password_reset",{userName:e}));if(!E(a)){t.authStatus={text:d(a,"STRING_VALUE")||"Password reset request failed.",tone:"danger"},s("auth-status",t.authStatus);return}const n=d(a,"ACCESS_TOKEN");if(!n){t.authStatus={text:"Password reset request failed: missing code.",tone:"danger"},s("auth-status",t.authStatus);return}t.passwordResetCode=n;const r=d(a,"STRING_VALUE")||"DM this code to the Discord bot, then send your new password.";t.authStatus={text:r,tone:"success"},h(),await S(),await y()}catch(a){t.authStatus={text:`Password reset request failed: ${String(a)}`,tone:"danger"},s("auth-status",t.authStatus)}finally{c(!1)}}async function Qt(e=!1){if(!t.username||!t.password){t.linkStatus={text:"Go back and sign in again first.",tone:"warning"},s("link-status",t.linkStatus);return}c(!0),t.linkStatus={text:e?"Linked. Signing in…":"Checking…",tone:"info"},s("link-status",t.linkStatus);let a=!1;try{const n=x(await l("launcher_login_or_link",{userName:t.username,password:t.password}));if(!E(n)){t.linkStatus={text:d(n,"STRING_VALUE")||"Login failed",tone:"danger"},s("link-status",t.linkStatus);return}if(!(b(n,"LINKED_FLAG")===1)){const f=d(n,"ACCESS_TOKEN");f&&(t.linkCode=f);const g=(d(n,"STRING_VALUE")||"").trim();t.linkStatus=g?{text:g,tone:"warning"}:{text:"Not linked yet. DM the active code and wait for auto-check.",tone:"warning"},s("link-status",t.linkStatus);return}const o=(d(n,"STRING_VALUE")||"").trim();if(o==="PIN_REQUIRED"||o==="PIN_INVALID"){t.loginToken="",t.pinStatus={text:o==="PIN_INVALID"?"Invalid PIN. Try again.":"Enter the developer PIN to continue.",tone:o==="PIN_INVALID"?"danger":"info"},t.view="pin",h(),await S(),await y();return}t.loginToken=d(n,"ACCESS_TOKEN");const i=d(n,"USER_NAME");i&&(t.username=i,B()),Ht(n),t.supporterBotNames=[],t.pinStatus={text:"",tone:"neutral"},t.view="main",t.verifiedOk=!1,t.filesStatus="",t.latestVersion="",t.baseUrl="",t.progressPct=0,t.verifyStatus={text:"Preparing verification…",tone:"info"},t.downloadStatus={text:"—",tone:"neutral"},t.launchStatus={text:"",tone:"neutral"},await rt(),await ot(),await it(),h(),await Kt(),await S(),await y(),a=!0}catch(n){t.linkStatus={text:`Status check failed: ${String(n)}`,tone:"danger"},s("link-status",t.linkStatus)}finally{c(!1),a&&j(!0)}}async function Ae(){if(!t.username||!t.password){t.linkStatus={text:"Go back and sign in again first.",tone:"warning"},s("link-status",t.linkStatus);return}c(!0),t.linkStatus={text:"Generating a new code…",tone:"info"},s("link-status",t.linkStatus);let e=!1;try{const a=x(await l("launcher_login_or_link",{userName:t.username,password:t.password}));if(!E(a)){t.linkStatus={text:d(a,"STRING_VALUE")||"Failed to generate code",tone:"danger"},s("link-status",t.linkStatus);return}if(b(a,"LINKED_FLAG")===1){t.linkStatus={text:"Already linked. Signing in…",tone:"success"},s("link-status",t.linkStatus),e=!0,nt(),Qt(!0);return}t.linkCode=d(a,"ACCESS_TOKEN"),h(),await S(),await y(),t.linkStatus={text:"New code generated. DM it to the bot. We'll auto-check every 3 seconds.",tone:"success"},s("link-status",t.linkStatus)}catch(a){t.linkStatus={text:`Failed to generate code: ${String(a)}`,tone:"danger"},s("link-status",t.linkStatus)}finally{e||c(!1)}}async function j(e=!1){if(!t.busy){c(!0),t.verifiedOk=!1,t.verifyStatus={text:e?"Auto verifying…":"Verifying…",tone:"info"},t.downloadStatus={text:"—",tone:"neutral"},t.launchStatus={text:"",tone:"neutral"},s("verify-status",t.verifyStatus),s("download-status",t.downloadStatus),s("launch-status",t.launchStatus),R(0);try{const a=await l("check_for_game_updates");t.latestVersion=a.latestVersion,t.baseUrl=a.baseUrl,t.filesStatus=a.status,t.verifiedOk=a.status==="upToDate",t.verifyStatus={text:t.verifiedOk?"Verified ✓":"Not ready",tone:t.verifiedOk?"success":ge(a.status)},t.verifiedOk?t.launchStatus={text:"",tone:"neutral"}:t.devBypassVerification||t.devBypassFileChecks?t.launchStatus=a.status==="notInstalled"?{text:"Install folder doesn't contain RealmEAC.exe. Pick the folder that contains Realm-Royale/ (or the Realm-Royale folder itself).",tone:"warning"}:{text:"Verify reported a non-ready state (DEV BYPASS enabled). Try launching anyway or pick a different folder.",tone:"warning"}:t.launchStatus={text:"Files not ready. Run Repair if needed.",tone:"warning"},N(),await S()}catch(a){t.verifyStatus={text:`Verify failed: ${String(a)}`,tone:"danger"},s("verify-status",t.verifyStatus)}finally{c(!1),N()}}}async function Te(){if(t.busy)return;if(t.devBypassFileChecks){t.downloadStatus={text:"DEV BYPASS: download/repair disabled.",tone:"warning"},s("download-status",t.downloadStatus);return}const e=(t.installPath??"").trim();if(!e){t.downloadStatus={text:"Choose an install folder first.",tone:"warning"},s("download-status",t.downloadStatus);return}c(!0),t.verifiedOk=!1,t.downloadStatus={text:"Starting…",tone:"info"},t.launchStatus={text:"",tone:"neutral"},s("download-status",t.downloadStatus),s("launch-status",t.launchStatus),R(0);try{await l("start_download",{version:"latest",installPath:e,baseUrl:t.baseUrl||""}),t.downloadStatus={text:"Download finished. Run Verify to enable Launch.",tone:"success"},t.launchStatus={text:"Run Verify to enable Launch.",tone:"warning"},N(),await S()}catch(a){t.downloadStatus={text:`Download failed: ${String(a)}`,tone:"danger"},t.launchStatus={text:"Download/repair failed.",tone:"danger"},s("download-status",t.downloadStatus),s("launch-status",t.launchStatus)}finally{c(!1),N()}}async function Pe(){if(t.busy)return;const e=(t.installPath??"").trim();if(!t.loginToken){t.launchStatus={text:"Login required.",tone:"warning"},s("launch-status",t.launchStatus);return}if(!t.devBypassVerification&&!t.devBypassFileChecks&&!t.verifiedOk){t.launchStatus={text:"Verify files before launching.",tone:"warning"},s("launch-status",t.launchStatus);return}if(!e){t.launchStatus={text:"Choose an install folder first.",tone:"warning"},s("launch-status",t.launchStatus);return}c(!0),t.launchStatus={text:"Launching…",tone:"info"},s("launch-status",t.launchStatus);try{const a=t.latestVersion||"local",n=t.devCommandLineEnabled?t.devCommandLineArgs:"";await l("launch_game",{version:a,installPath:e,username:t.username,accessToken:t.loginToken,extraArgs:n}),t.launchStatus={text:"Game launched.",tone:"success"},s("launch-status",t.launchStatus)}catch(a){t.launchStatus={text:`Launch failed: ${String(a)}`,tone:"danger"},s("launch-status",t.launchStatus)}finally{c(!1)}}async function rt(){try{const e=await l("get_dev_command_line_settings");t.devCommandLineEnabled=!!(e!=null&&e.enabled),t.devCommandLineArgs=(e==null?void 0:e.args)??""}catch{t.devCommandLineEnabled=!1,t.devCommandLineArgs=""}}async function ot(){try{const e=await l("get_game_language");t.gameLanguage=K(String(e??""))}catch{t.gameLanguage=Z}}async function Ue(e){const a=K(t.gameLanguage),n=K(e);t.gameLanguage=n;try{await l("save_game_language",{language:n}),t.gameLanguageStatus={text:"Saved. Applies the next time you launch.",tone:"success"}}catch(o){t.gameLanguage=a,t.gameLanguageStatus={text:`Save failed: ${String(o)}`,tone:"danger"}}const r=u("game-language");r&&(r.value!==t.gameLanguage&&(r.value=t.gameLanguage),r.title=Se(t.gameLanguage)),s("game-language-status",t.gameLanguageStatus)}async function it(){try{const e=await l("get_dev_bypass_settings");t.devBypassVerification=!!(e!=null&&e.bypass_verification),t.devBypassFileChecks=!!(e!=null&&e.bypass_file_checks)}catch{t.devBypassVerification=!1,t.devBypassFileChecks=!1}}async function X(){var a;if(!t.devCommandLineEnabled)return;const e=(((a=u("dev-command-line-args"))==null?void 0:a.value)??"").toString();t.devCommandLineArgs=e;try{await l("save_dev_command_line_args",{args:e}),t.devCommandLineStatus={text:"Saved.",tone:"success"}}catch(n){t.devCommandLineStatus={text:`Save failed: ${String(n)}`,tone:"danger"}}s("dev-cli-status",t.devCommandLineStatus)}async function Ie(){if(!t.devCommandLineEnabled)return;t.devCommandLineArgs="";const e=u("dev-command-line-args");e&&(e.value=""),await X()}async function Re(){if(!t.busy)try{const e=await ee({directory:!0,multiple:!1,title:"Select install folder",defaultPath:t.installPath||void 0}),a=typeof e=="string"?e.trim():"";if(!a)return;c(!0);let n=!1;try{t.installPath=a,t.verifiedOk=!1,t.filesStatus="",t.progressPct=0,t.verifyStatus={text:"Install folder updated. Saving…",tone:"info"},t.launchStatus={text:"",tone:"neutral"},N(),await l("save_install_path",{path:a}),n=!0,await S()}finally{c(!1),N(),n&&j(!0)}}catch(e){t.verifyStatus={text:`Folder select failed: ${String(e)}`,tone:"danger"},s("verify-status",t.verifyStatus)}}async function $e(e){if(t.busy)return;const a=T(`supporter-bot-name-${e}`);if(t.loginToken){if(!a){t.launchStatus={text:"Enter a bot name before saving.",tone:"warning"},s("launch-status",t.launchStatus);return}try{c(!0);const n=x(await l("launcher_supporter_bot_name_upsert",{accessToken:t.loginToken,slotIndex:e,botName:a}));if(!E(n)){t.launchStatus={text:d(n,"STRING_VALUE")||"Failed to save bot name.",tone:"danger"},s("launch-status",t.launchStatus);return}at(n),t.supporterBotNames=et(d(n,"PORTAL_INFO_1")),t.launchStatus={text:"Supporter bot name saved.",tone:"success"},zt()}catch(n){t.launchStatus={text:`Failed to save bot name: ${String(n)}`,tone:"danger"},s("launch-status",t.launchStatus)}finally{c(!1)}}}async function De(e){if(!t.busy&&t.loginToken)try{c(!0);const a=x(await l("launcher_supporter_bot_name_delete",{accessToken:t.loginToken,slotIndex:e}));if(!E(a)){t.launchStatus={text:d(a,"STRING_VALUE")||"Failed to clear bot name.",tone:"danger"},s("launch-status",t.launchStatus);return}at(a),t.supporterBotNames=et(d(a,"PORTAL_INFO_1")),t.launchStatus={text:"Supporter bot name cleared.",tone:"success"},zt()}catch(a){t.launchStatus={text:`Failed to clear bot name: ${String(a)}`,tone:"danger"},s("launch-status",t.launchStatus)}finally{c(!1)}}async function y(){var e,a,n,r,o,i,f,g,A,C,k,M,ut,lt,dt,ct,pt,ft,vt,ht,gt,St,yt,mt,wt,bt,kt,_t,Lt;Ee(),(e=u("btn-update"))==null||e.addEventListener("click",()=>void Bt()),(a=u("btn-kofi"))==null||a.addEventListener("click",async()=>{try{await l("open_kofi")}catch{}}),(n=u("btn-community"))==null||n.addEventListener("click",async()=>{try{await l("open_discord_server")}catch{}}),(r=u("btn-login"))==null||r.addEventListener("click",It),(o=u("btn-register"))==null||o.addEventListener("click",Ce),(i=u("btn-forgot-password"))==null||i.addEventListener("click",Ne),(f=u("btn-pin-submit"))==null||f.addEventListener("click",Rt),(g=u("btn-copy-reset-code"))==null||g.addEventListener("click",async()=>{if(t.passwordResetCode)try{await navigator.clipboard.writeText(t.passwordResetCode),t.authStatus={text:"Password reset code copied. DM it to the bot.",tone:"success"},s("auth-status",t.authStatus)}catch{t.authStatus={text:"Copy failed. Copy the code manually.",tone:"warning"},s("auth-status",t.authStatus)}}),(A=u("btn-open-dm-auth"))==null||A.addEventListener("click",async()=>{try{await l("open_discord_dm")}catch(p){t.authStatus={text:`Failed to open Discord: ${String(p)}`,tone:"danger"},s("auth-status",t.authStatus)}}),(C=u("password"))==null||C.addEventListener("keydown",async p=>{p.key==="Enter"&&(p.preventDefault(),await It())}),(k=u("dev_pin"))==null||k.addEventListener("keydown",async p=>{p.key==="Enter"&&(p.preventDefault(),await Rt())}),(M=u("btn-back"))==null||M.addEventListener("click",()=>{t.view="auth",t.linkCode="",t.passwordResetCode="",t.linkStatus={text:"DM the code to the bot before it expires. We'll auto-check every 3 seconds.",tone:"neutral"},t.authStatus={text:"",tone:"neutral"},t.pinStatus={text:"",tone:"neutral"},h(),S(),y()}),(ut=u("btn-pin-back"))==null||ut.addEventListener("click",()=>{t.view="auth",t.pinStatus={text:"",tone:"neutral"},h(),S(),y()}),(lt=u("btn-copy-code"))==null||lt.addEventListener("click",async()=>{if(t.linkCode)try{await navigator.clipboard.writeText(t.linkCode),t.linkStatus={text:"Copied. DM the code before it expires. We'll auto-check every 3 seconds.",tone:"success"},s("link-status",t.linkStatus)}catch{t.linkStatus={text:"Copy failed. Copy manually.",tone:"warning"},s("link-status",t.linkStatus)}}),(dt=u("btn-open-dm"))==null||dt.addEventListener("click",async()=>{try{await l("open_discord_dm")}catch(p){t.linkStatus={text:`Failed to open Discord: ${String(p)}`,tone:"danger"},s("link-status",t.linkStatus)}}),(ct=u("btn-join-discord"))==null||ct.addEventListener("click",async()=>{try{await l("open_discord_server")}catch(p){t.linkStatus={text:`Failed to open Discord server: ${String(p)}`,tone:"danger"},s("link-status",t.linkStatus)}}),(pt=u("btn-new-code"))==null||pt.addEventListener("click",Ae),(ft=u("btn-logout"))==null||ft.addEventListener("click",()=>{t.loginToken="",t.password="",t.linkCode="",t.passwordResetCode="",t.latestVersion="",t.baseUrl="",t.filesStatus="",t.verifiedOk=!1,t.progressPct=0,t.verifyStatus={text:"—",tone:"neutral"},t.downloadStatus={text:"—",tone:"neutral"},t.launchStatus={text:"",tone:"neutral"},t.authStatus={text:"",tone:"neutral"},t.linkStatus={text:"DM the code to the bot before it expires. We'll auto-check every 3 seconds.",tone:"neutral"},t.pinStatus={text:"",tone:"neutral"},t.supporterTierName="none",t.supporterTierLevel=0,t.supporterBotSlotsTotal=0,t.supporterBotSlotsUsed=0,t.supporterPerksPreview="",t.supporterAnnouncementText="",t.supporterAnnouncementSeconds=0,t.supporterBotNames=[],t.view="auth",h(),S(),y()}),(vt=u("btn-browse-install"))==null||vt.addEventListener("click",Re),(ht=u("btn-verify"))==null||ht.addEventListener("click",()=>void j(!1)),(gt=u("btn-download"))==null||gt.addEventListener("click",Te),(St=u("btn-launch"))==null||St.addEventListener("click",Pe),(yt=u("dev-command-line-args"))==null||yt.addEventListener("input",p=>{t.devCommandLineArgs=p.target.value}),(mt=u("dev-command-line-args"))==null||mt.addEventListener("keydown",p=>{p.key==="Enter"&&X()}),(wt=u("game-language"))==null||wt.addEventListener("change",p=>{Ue(p.target.value)}),(bt=u("btn-dev-cli-save"))==null||bt.addEventListener("click",()=>void X()),(kt=u("btn-dev-cli-clear"))==null||kt.addEventListener("click",()=>void Ie());for(let p=1;p<=Math.max(2,t.supporterBotSlotsTotal);p++)(_t=u(`btn-bot-save-${p}`))==null||_t.addEventListener("click",()=>void $e(p)),(Lt=u(`btn-bot-delete-${p}`))==null||Lt.addEventListener("click",()=>void De(p))}async function Oe(){await Et("download-progress",e=>{var o,i,f;const a=((o=e.payload)==null?void 0:o.percentage)??null,n=((i=e.payload)==null?void 0:i.status_text)??"",r=((f=e.payload)==null?void 0:f.speed_mbs)??null;t.downloadStatus={text:`${n}${r!=null?` (${r.toFixed(1)} MB/s)`:""}`.trim()||"—",tone:"info"},s("download-status",t.downloadStatus),a!=null&&R(a)}),await Et("verification-progress",e=>{var r,o;const a=((r=e.payload)==null?void 0:r.progress)??0,n=((o=e.payload)==null?void 0:o.status_text)??"";t.verifyStatus={text:n||"Verifying…",tone:"info"},s("verify-status",t.verifyStatus),R(a*100)})}window.addEventListener("DOMContentLoaded",async()=>{he(),h(),document.addEventListener("visibilitychange",()=>{Jt()}),await S(),await rt(),await ot(),await it(),Tt(),window.setInterval(()=>{Tt()},ie),Yt(),await Y(),window.setInterval(()=>{(!m||m.readyState!==WebSocket.OPEN)&&Y()},6e4),await Oe(),await y()});
