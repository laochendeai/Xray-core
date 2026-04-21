import{E as L,p as N,a6 as G,q as O,d as D,j as b,a7 as H,a8 as K,v as M,x as W,z as q,A as S,T as F,a9 as U,r as w,aa as X,ab as Y,ac as Z,u as J,a2 as Q,Z as x,w as s,e,a4 as ee,o as _,b as z,t as i,f as n,h as m,_ as T,B as P,N as j,c as R,ad as A,P as E,a1 as te}from"./index-BDH_x4ve.js";import{N as B,r as se,a as ae,b as ne,d as ie,c as re}from"./readiness-yo2Y9J2b.js";import{N as oe}from"./Empty-D9ezntLB.js";import{u as le,N as $}from"./Tag-slxu22dm.js";import{N as C}from"./Space-CkTfJyf1.js";import{N as I}from"./Alert-jwyGbJpm.js";import{N as ce,a as V}from"./Grid-B51PNBJJ.js";import{_ as de}from"./_plugin-vue_export-helper-DlAUqK2U.js";import"./use-locale-V36vuiO_.js";import"./next-frame-once-C5Ksf8W7.js";const ue=L([L("@keyframes spin-rotate",`
 from {
 transform: rotate(0);
 }
 to {
 transform: rotate(360deg);
 }
 `),N("spin-container",`
 position: relative;
 `,[N("spin-body",`
 position: absolute;
 top: 50%;
 left: 50%;
 transform: translateX(-50%) translateY(-50%);
 `,[G()])]),N("spin-body",`
 display: inline-flex;
 align-items: center;
 justify-content: center;
 flex-direction: column;
 `),N("spin",`
 display: inline-flex;
 height: var(--n-size);
 width: var(--n-size);
 font-size: var(--n-size);
 color: var(--n-color);
 `,[O("rotate",`
 animation: spin-rotate 2s linear infinite;
 `)]),N("spin-description",`
 display: inline-block;
 font-size: var(--n-font-size);
 color: var(--n-text-color);
 transition: color .3s var(--n-bezier);
 margin-top: 8px;
 `),N("spin-content",`
 opacity: 1;
 transition: opacity .3s var(--n-bezier);
 pointer-events: all;
 `,[O("spinning",`
 user-select: none;
 -webkit-user-select: none;
 pointer-events: none;
 opacity: var(--n-opacity-spinning);
 `)])]),pe={small:20,medium:18,large:16},fe=Object.assign(Object.assign(Object.assign({},W.props),{contentClass:String,contentStyle:[Object,String],description:String,size:{type:[String,Number],default:"medium"},show:{type:Boolean,default:!0},rotate:{type:Boolean,default:!0},spinning:{type:Boolean,validator:()=>!0,default:void 0},delay:Number}),X),me=D({name:"Spin",props:fe,slots:Object,setup(l){const{mergedClsPrefixRef:y,inlineThemeDisabled:t}=M(l),a=W("Spin","-spin",ue,U,l,y),p=S(()=>{const{size:r}=l,{common:{cubicBezierEaseInOut:h},self:k}=a.value,{opacitySpinning:o,color:g,textColor:c}=k,d=typeof r=="number"?Y(r):k[Z("size",r)];return{"--n-bezier":h,"--n-opacity-spinning":o,"--n-size":d,"--n-color":g,"--n-text-color":c}}),u=t?q("spin",S(()=>{const{size:r}=l;return typeof r=="number"?String(r):r[0]}),p,l):void 0,f=le(l,["spinning","show"]),v=w(!1);return F(r=>{let h;if(f.value){const{delay:k}=l;if(k){h=window.setTimeout(()=>{v.value=!0},k),r(()=>{clearTimeout(h)});return}}v.value=f.value}),{mergedClsPrefix:y,active:v,mergedStrokeWidth:S(()=>{const{strokeWidth:r}=l;if(r!==void 0)return r;const{size:h}=l;return pe[typeof h=="number"?"medium":h]}),cssVars:t?void 0:p,themeClass:u==null?void 0:u.themeClass,onRender:u==null?void 0:u.onRender}},render(){var l,y;const{$slots:t,mergedClsPrefix:a,description:p}=this,u=t.icon&&this.rotate,f=(p||t.description)&&b("div",{class:`${a}-spin-description`},p||((l=t.description)===null||l===void 0?void 0:l.call(t))),v=t.icon?b("div",{class:[`${a}-spin-body`,this.themeClass]},b("div",{class:[`${a}-spin`,u&&`${a}-spin--rotate`],style:t.default?"":this.cssVars},t.icon()),f):b("div",{class:[`${a}-spin-body`,this.themeClass]},b(H,{clsPrefix:a,style:t.default?"":this.cssVars,stroke:this.stroke,"stroke-width":this.mergedStrokeWidth,radius:this.radius,scale:this.scale,class:`${a}-spin`}),f);return(y=this.onRender)===null||y===void 0||y.call(this),t.default?b("div",{class:[`${a}-spin-container`,this.themeClass],style:this.cssVars},b("div",{class:[`${a}-spin-content`,this.active&&`${a}-spin-content--spinning`,this.contentClass],style:this.contentStyle},t),b(K,{name:"fade-in-transition"},{default:()=>this.active?v:null})):v}}),ve={class:"page-title"},he={class:"page-subtitle"},ye={class:"page-subtitle"},ge=D({__name:"Readiness",setup(l){const y=te(),{t}=J(),a=w(null),p=w(!1),u=w(""),f=S(()=>ie(t,a.value)),v=S(()=>{var o;return(((o=a.value)==null?void 0:o.checks)??[]).map(g=>({check:g,description:re(t,g)}))});async function r(){p.value=!0,u.value="";try{a.value=await ee.get()}catch(o){u.value=(o==null?void 0:o.error)||t("common.error")}finally{p.value=!1}}function h(o){y.push(o)}function k(o){if(!o)return"-";const g=new Date(o);return Number.isNaN(g.getTime())?o:g.toLocaleString()}return Q(()=>{r()}),(o,g)=>(_(),x(e(C),{vertical:"",size:16},{default:s(()=>[z("div",null,[z("h2",ve,i(e(t)("readiness.title")),1),z("div",he,i(e(t)("readiness.subtitle")),1)]),n(e(j),{size:"small"},{header:s(()=>[n(e(C),{align:"center",size:12},{default:s(()=>[z("span",null,i(e(t)("readiness.summaryTitle")),1),n(e($),{type:f.value.type},{default:s(()=>[m(i(f.value.badgeLabel),1)]),_:1},8,["type"])]),_:1})]),"header-extra":s(()=>[n(e(P),{size:"small",onClick:r,loading:p.value},{default:s(()=>[m(i(e(t)("common.refresh")),1)]),_:1},8,["loading"])]),default:s(()=>[n(e(C),{vertical:"",size:12},{default:s(()=>{var c;return[n(e(I),{type:f.value.type,title:f.value.title},{default:s(()=>[m(i(f.value.description),1)]),_:1},8,["type","title"]),u.value?(_(),x(e(I),{key:0,type:"error"},{default:s(()=>[m(i(u.value),1)]),_:1})):T("",!0),n(e(ce),{cols:3,"x-gap":12,responsive:"screen","item-responsive":""},{default:s(()=>[n(e(V),{span:"3 m:1"},{default:s(()=>[n(e(B),{label:e(t)("readiness.cards.blocking")},{default:s(()=>{var d;return[m(i(((d=a.value)==null?void 0:d.blockingCount)??0),1)]}),_:1},8,["label"])]),_:1}),n(e(V),{span:"3 m:1"},{default:s(()=>[n(e(B),{label:e(t)("readiness.cards.warning")},{default:s(()=>{var d;return[m(i(((d=a.value)==null?void 0:d.warningCount)??0),1)]}),_:1},8,["label"])]),_:1}),n(e(V),{span:"3 m:1"},{default:s(()=>[n(e(B),{label:e(t)("readiness.cards.checks")},{default:s(()=>{var d;return[m(i(((d=a.value)==null?void 0:d.checks.length)??0),1)]}),_:1},8,["label"])]),_:1})]),_:1}),z("div",ye,i(e(t)("readiness.lastUpdated"))+": "+i(k((c=a.value)==null?void 0:c.updatedAt)),1)]}),_:1})]),_:1}),n(e(me),{show:p.value},{default:s(()=>[n(e(C),{vertical:"",size:12},{default:s(()=>[(_(!0),R(E,null,A(v.value,c=>(_(),x(e(j),{key:c.check.key,size:"small",embedded:""},{header:s(()=>[n(e(C),{align:"center",size:8},{default:s(()=>[n(e($),{type:e(ae)(c.check.severity)},{default:s(()=>[m(i(e(ne)(e(t),c.check.severity)),1)]),_:2},1032,["type"]),z("strong",null,i(c.description.title),1)]),_:2},1024)]),"header-extra":s(()=>[n(e(C),{align:"center",size:8},{default:s(()=>[n(e($),{size:"small",bordered:!1},{default:s(()=>[m(i(e(se)(e(t),c.check.area)),1)]),_:2},1024),c.check.actionRoute?(_(),x(e(P),{key:0,text:"",type:"primary",onClick:d=>h(c.check.actionRoute)},{default:s(()=>[m(i(e(t)("readiness.goToArea")),1)]),_:1},8,["onClick"])):T("",!0)]),_:2},1024)]),default:s(()=>[n(e(C),{vertical:"",size:8},{default:s(()=>[z("div",null,i(c.description.summary),1),(_(!0),R(E,null,A(c.description.details,d=>(_(),R("div",{key:d,class:"check-detail"},i(d),1))),128))]),_:2},1024)]),_:2},1024))),128)),!v.value.length&&!p.value?(_(),x(e(oe),{key:0,description:e(t)("readiness.noChecks")},null,8,["description"])):T("",!0)]),_:1})]),_:1},8,["show"])]),_:1}))}}),Re=de(ge,[["__scopeId","data-v-8a411173"]]);export{Re as default};
