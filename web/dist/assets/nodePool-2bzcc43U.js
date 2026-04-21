import{E as h,p as c,q as v,D as d,aK as j,aL as F,d as x,j as u,v as S,af as O,x as _,z as H,bs as q,A as K,n as V,C as U,I as W,a$ as Y,G}from"./index-B5eGNbQJ.js";const Q=h([c("list",`
 --n-merged-border-color: var(--n-border-color);
 --n-merged-color: var(--n-color);
 --n-merged-color-hover: var(--n-color-hover);
 margin: 0;
 font-size: var(--n-font-size);
 transition:
 background-color .3s var(--n-bezier),
 color .3s var(--n-bezier),
 border-color .3s var(--n-bezier);
 padding: 0;
 list-style-type: none;
 color: var(--n-text-color);
 background-color: var(--n-merged-color);
 `,[v("show-divider",[c("list-item",[h("&:not(:last-child)",[d("divider",`
 background-color: var(--n-merged-border-color);
 `)])])]),v("clickable",[c("list-item",`
 cursor: pointer;
 `)]),v("bordered",`
 border: 1px solid var(--n-merged-border-color);
 border-radius: var(--n-border-radius);
 `),v("hoverable",[c("list-item",`
 border-radius: var(--n-border-radius);
 `,[h("&:hover",`
 background-color: var(--n-merged-color-hover);
 `,[d("divider",`
 background-color: transparent;
 `)])])]),v("bordered, hoverable",[c("list-item",`
 padding: 12px 20px;
 `),d("header, footer",`
 padding: 12px 20px;
 `)]),d("header, footer",`
 padding: 12px 0;
 box-sizing: border-box;
 transition: border-color .3s var(--n-bezier);
 `,[h("&:not(:last-child)",`
 border-bottom: 1px solid var(--n-merged-border-color);
 `)]),c("list-item",`
 position: relative;
 padding: 12px 0; 
 box-sizing: border-box;
 display: flex;
 flex-wrap: nowrap;
 align-items: center;
 transition:
 background-color .3s var(--n-bezier),
 border-color .3s var(--n-bezier);
 `,[d("prefix",`
 margin-right: 20px;
 flex: 0;
 `),d("suffix",`
 margin-left: 20px;
 flex: 0;
 `),d("main",`
 flex: 1;
 `),d("divider",`
 height: 1px;
 position: absolute;
 bottom: 0;
 left: 0;
 right: 0;
 background-color: transparent;
 transition: background-color .3s var(--n-bezier);
 pointer-events: none;
 `)])]),j(c("list",`
 --n-merged-color-hover: var(--n-color-hover-modal);
 --n-merged-color: var(--n-color-modal);
 --n-merged-border-color: var(--n-border-color-modal);
 `)),F(c("list",`
 --n-merged-color-hover: var(--n-color-hover-popover);
 --n-merged-color: var(--n-color-popover);
 --n-merged-border-color: var(--n-border-color-popover);
 `))]),J=Object.assign(Object.assign({},_.props),{size:{type:String,default:"medium"},bordered:Boolean,clickable:Boolean,hoverable:Boolean,showDivider:{type:Boolean,default:!0}}),w=V("n-list"),ie=x({name:"List",props:J,slots:Object,setup(e){const{mergedClsPrefixRef:o,inlineThemeDisabled:r,mergedRtlRef:n}=S(e),t=O("List",n,o),p=_("List","-list",Q,q,e,o);U(w,{showDividerRef:W(e,"showDivider"),mergedClsPrefixRef:o});const g=K(()=>{const{common:{cubicBezierEaseInOut:C},self:{fontSize:P,textColor:M,color:I,colorModal:R,colorPopover:N,borderColor:z,borderColorModal:$,borderColorPopover:T,borderRadius:B,colorHover:E,colorHoverModal:A,colorHoverPopover:L}}=p.value;return{"--n-font-size":P,"--n-bezier":C,"--n-text-color":M,"--n-color":I,"--n-border-radius":B,"--n-border-color":z,"--n-border-color-modal":$,"--n-border-color-popover":T,"--n-color-modal":R,"--n-color-popover":N,"--n-color-hover":E,"--n-color-hover-modal":A,"--n-color-hover-popover":L}}),a=r?H("list",void 0,g,e):void 0;return{mergedClsPrefix:o,rtlEnabled:t,cssVars:r?void 0:g,themeClass:a==null?void 0:a.themeClass,onRender:a==null?void 0:a.onRender}},render(){var e;const{$slots:o,mergedClsPrefix:r,onRender:n}=this;return n==null||n(),u("ul",{class:[`${r}-list`,this.rtlEnabled&&`${r}-list--rtl`,this.bordered&&`${r}-list--bordered`,this.showDivider&&`${r}-list--show-divider`,this.hoverable&&`${r}-list--hoverable`,this.clickable&&`${r}-list--clickable`,this.themeClass],style:this.cssVars},o.header?u("div",{class:`${r}-list__header`},o.header()):null,(e=o.default)===null||e===void 0?void 0:e.call(o),o.footer?u("div",{class:`${r}-list__footer`},o.footer()):null)}}),se=x({name:"ListItem",slots:Object,setup(){const e=G(w,null);return e||Y("list-item","`n-list-item` must be placed in `n-list`."),{showDivider:e.showDividerRef,mergedClsPrefix:e.mergedClsPrefixRef}},render(){const{$slots:e,mergedClsPrefix:o}=this;return u("li",{class:`${o}-list-item`},e.prefix?u("div",{class:`${o}-list-item__prefix`},e.prefix()):null,e.default?u("div",{class:`${o}-list-item__main`},e):null,e.suffix?u("div",{class:`${o}-list-item__suffix`},e.suffix()):null,this.showDivider&&u("div",{class:`${o}-list-item__divider`}))}}),X={openai:["domain:openai.com","domain:api.openai.com","domain:auth.openai.com","domain:chatgpt.com","domain:chat.openai.com","domain:oaistatic.com","domain:oaiusercontent.com"],chatgpt:["domain:chatgpt.com","domain:chat.openai.com","domain:oaistatic.com","domain:oaiusercontent.com"],claude:["domain:claude.ai","domain:anthropic.com"],gemini:["domain:gemini.google.com","domain:ai.google.dev","domain:aistudio.google.com","full:generativelanguage.googleapis.com"],github:["full:api.github.com","domain:github.com","domain:githubusercontent.com","domain:githubassets.com","domain:github.io"],github_copilot:["full:github.com","full:api.github.com","full:copilot.github.com"],openrouter:["domain:openrouter.ai"],cursor:["domain:cursor.com"],qwen:["domain:qwen.ai","full:dashscope.aliyuncs.com"],perplexity:["domain:perplexity.ai"],deepseek:["domain:deepseek.com"]};function b(e){switch(e){case"trusted":return 0;case"unknown":return 1;case"suspicious":return 2;default:return 3}}function k(e){switch(e){case"residential_likely":return 0;case"unknown":return 1;case"isp_likely":return 2;case"datacenter_likely":return 3;default:return 4}}function y(e){switch(e){case"available":return 0;case"unknown":return 1;case"error":return 2;default:return 3}}function Z(e){return[...e].sort((o,r)=>{const n=b(o.cleanliness)-b(r.cleanliness);if(n!==0)return n;const t=k(o.networkType)-k(r.networkType);if(t!==0)return t;const p=y(o.exitIpStatus)-y(r.exitIpStatus);if(p!==0)return p;const g=l(o.totalPings?s(o):null,r.totalPings?s(r):null,"asc");if(g!==0)return g;const a=l(o.avgDelayMs>0?o.avgDelayMs:null,r.avgDelayMs>0?r.avgDelayMs:null,"asc");return a!==0?a:f(r)-f(o)})}function le(e){var o;return((o=Z(e)[0])==null?void 0:o.id)||""}function s(e){return e.totalPings?e.failedPings/e.totalPings:Number.POSITIVE_INFINITY}function D(e){return e.avgDelayMs>0?e.avgDelayMs:Number.POSITIVE_INFINITY}function f(e){const o=e.lastCheckedAt||e.statusUpdatedAt||e.addedAt,r=o?new Date(o).getTime():0;return Number.isFinite(r)?r:0}function i(e){const o=e.statusUpdatedAt||e.lastEventAt||e.addedAt,r=o?new Date(o).getTime():0;return Number.isFinite(r)?r:0}function m(e,o){const r=s(e)-s(o);if(r!==0)return r;const n=D(e)-D(o);return n!==0?n:e.consecutiveFails!==o.consecutiveFails?e.consecutiveFails-o.consecutiveFails:e.totalPings!==o.totalPings?o.totalPings-e.totalPings:f(o)-f(e)}function ee(e,o){const r=b(e.cleanliness)-b(o.cleanliness);return r!==0?r:m(e,o)}function l(e,o,r){return e==null&&o==null?0:e==null?1:o==null?-1:r==="asc"?e-o:o-e}function ae(e,o){return[...e].sort((r,n)=>{switch(o){case"cleanliness_desc":return ee(r,n);case"last_checked_desc":return f(n)-f(r);case"last_checked_asc":return f(r)-f(n);case"fail_rate_asc":{const t=l(r.totalPings?s(r):null,n.totalPings?s(n):null,"asc");return t!==0?t:m(r,n)}case"fail_rate_desc":{const t=l(r.totalPings?s(r):null,n.totalPings?s(n):null,"desc");return t!==0?t:m(r,n)}case"avg_delay_asc":{const t=l(r.avgDelayMs>0?r.avgDelayMs:null,n.avgDelayMs>0?n.avgDelayMs:null,"asc");return t!==0?t:m(r,n)}case"avg_delay_desc":{const t=l(r.avgDelayMs>0?r.avgDelayMs:null,n.avgDelayMs>0?n.avgDelayMs:null,"desc");return t!==0?t:m(r,n)}default:return m(r,n)}})}function ce(e,o){return[...e].sort((r,n)=>{switch(o){case"removed_asc":return i(r)-i(n);case"fail_rate_asc":{const t=l(r.totalPings?s(r):null,n.totalPings?s(n):null,"asc");return t!==0?t:i(n)-i(r)}case"fail_rate_desc":{const t=l(r.totalPings?s(r):null,n.totalPings?s(n):null,"desc");return t!==0?t:i(n)-i(r)}case"avg_delay_asc":{const t=l(r.avgDelayMs>0?r.avgDelayMs:null,n.avgDelayMs>0?n.avgDelayMs:null,"asc");return t!==0?t:i(n)-i(r)}case"avg_delay_desc":{const t=l(r.avgDelayMs>0?r.avgDelayMs:null,n.avgDelayMs>0?n.avgDelayMs:null,"desc");return t!==0?t:i(n)-i(r)}default:return i(n)-i(r)}})}function de(e){return e.reduce((o,r)=>{switch(r.cleanliness){case"trusted":o.trustedCount+=1;break;case"suspicious":o.suspiciousCount+=1;break;default:o.unknownCleanCount+=1;break}switch(r.networkType){case"residential_likely":o.residentialCount+=1;break;case"isp_likely":o.ispLikeCount+=1;break;case"datacenter_likely":o.datacenterCount+=1;break;default:o.unknownNetworkCount+=1;break}return o},{trustedCount:0,suspiciousCount:0,unknownCleanCount:0,residentialCount:0,ispLikeCount:0,datacenterCount:0,unknownNetworkCount:0})}function ue(e){const o=[e.cleanlinessDetail,e.networkTypeDetail,e.intelligenceError,e.exitIpError];for(const r of o)if(typeof r=="string"&&r.trim())return r.trim();return""}function fe(e){return Array.from(new Set(e.split(/[\n,]/).map(o=>o.trim()).filter(Boolean)))}function re(e){return e.preset==="custom"?oe(e.domains):X[e.preset]||[]}function me(e){return(re(e)[0]||"").replace(/^(full:|domain:)/,"")}function oe(e){return Array.from(new Set(e.map(o=>ne(o)).filter(Boolean)))}function ne(e){const o=e.trim();if(!o)return"";if(o.startsWith("*.")){const r=o.slice(2).replace(/^\.+|\.+$/g,"");return r?`domain:${r}`:""}if(o.startsWith(".")){const r=o.slice(1).replace(/^\.+|\.+$/g,"");return r?`domain:${r}`:""}return o}export{ie as N,se as a,le as b,re as c,Z as d,me as e,ue as f,ae as g,ce as h,fe as n,de as s};
