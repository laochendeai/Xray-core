import{d as O,j as d,k as Ze,s as Je,l as Qe,m as Ie,n as Q,p as u,q as w,S as Ne,v as ne,x as q,y as Ae,z as le,A as C,r as $,C as X,D as c,E as S,F as ke,G,H as M,I as re,J as Z,K as eo,L as W,M as pe,O as ue,P as oo,Q as ae,R as to,V as ro,T as Se,U as no,W as lo,X as io,Y as ao,a as so,u as co,Z as uo,w as V,e as _,o as we,f as F,b as E,c as vo,t as J,_ as ho,B as se,h as mo,$ as po,a0 as fo,a1 as go}from"./index-CelY_MPY.js";import{C as bo,N as xo,a as _e,V as Co,c as ce,b as yo}from"./Dropdown-BZvwEOoi.js";import{f as de,u as ve}from"./Suffix-Dxo0NrBY.js";import{u as zo}from"./Tag-BWAC16kI.js";import{_ as Io}from"./_plugin-vue_export-helper-DlAUqK2U.js";import"./next-frame-once-C5Ksf8W7.js";import"./use-locale-CPa8dLjL.js";const So=O({name:"ChevronDownFilled",render(){return d("svg",{viewBox:"0 0 16 16",fill:"none",xmlns:"http://www.w3.org/2000/svg"},d("path",{d:"M3.20041 5.73966C3.48226 5.43613 3.95681 5.41856 4.26034 5.70041L8 9.22652L11.7397 5.70041C12.0432 5.41856 12.5177 5.43613 12.7996 5.73966C13.0815 6.0432 13.0639 6.51775 12.7603 6.7996L8.51034 10.7996C8.22258 11.0668 7.77743 11.0668 7.48967 10.7996L3.23966 6.7996C2.93613 6.51775 2.91856 6.0432 3.20041 5.73966Z",fill:"currentColor"}))}});function wo(e){const{baseColor:t,textColor2:r,bodyColor:i,cardColor:a,dividerColor:l,actionColor:v,scrollbarColor:m,scrollbarColorHover:s,invertedColor:x}=e;return{textColor:r,textColorInverted:"#FFF",color:i,colorEmbedded:v,headerColor:a,headerColorInverted:x,footerColor:v,footerColorInverted:x,headerBorderColor:l,headerBorderColorInverted:x,footerBorderColor:l,footerBorderColorInverted:x,siderBorderColor:l,siderBorderColorInverted:x,siderColor:a,siderColorInverted:x,siderToggleButtonBorder:`1px solid ${l}`,siderToggleButtonColor:t,siderToggleButtonIconColor:r,siderToggleButtonIconColorInverted:r,siderToggleBarColor:Ie(i,m),siderToggleBarColorHover:Ie(i,s),__invertScrollbar:"true"}}const fe=Ze({name:"Layout",common:Qe,peers:{Scrollbar:Je},self:wo}),He=Q("n-layout-sider"),ge={type:String,default:"static"},Ro=u("layout",`
 color: var(--n-text-color);
 background-color: var(--n-color);
 box-sizing: border-box;
 position: relative;
 z-index: auto;
 flex: auto;
 overflow: hidden;
 transition:
 box-shadow .3s var(--n-bezier),
 background-color .3s var(--n-bezier),
 color .3s var(--n-bezier);
`,[u("layout-scroll-container",`
 overflow-x: hidden;
 box-sizing: border-box;
 height: 100%;
 `),w("absolute-positioned",`
 position: absolute;
 left: 0;
 right: 0;
 top: 0;
 bottom: 0;
 `)]),Po={embedded:Boolean,position:ge,nativeScrollbar:{type:Boolean,default:!0},scrollbarProps:Object,onScroll:Function,contentClass:String,contentStyle:{type:[String,Object],default:""},hasSider:Boolean,siderPlacement:{type:String,default:"left"}},Be=Q("n-layout");function Oe(e){return O({name:e?"LayoutContent":"Layout",props:Object.assign(Object.assign({},q.props),Po),setup(t){const r=$(null),i=$(null),{mergedClsPrefixRef:a,inlineThemeDisabled:l}=ne(t),v=q("Layout","-layout",Ro,fe,t,a);function m(f,g){if(t.nativeScrollbar){const{value:T}=r;T&&(g===void 0?T.scrollTo(f):T.scrollTo(f,g))}else{const{value:T}=i;T&&T.scrollTo(f,g)}}X(Be,t);let s=0,x=0;const H=f=>{var g;const T=f.target;s=T.scrollLeft,x=T.scrollTop,(g=t.onScroll)===null||g===void 0||g.call(t,f)};Ae(()=>{if(t.nativeScrollbar){const f=r.value;f&&(f.scrollTop=x,f.scrollLeft=s)}});const N={display:"flex",flexWrap:"nowrap",width:"100%",flexDirection:"row"},p={scrollTo:m},A=C(()=>{const{common:{cubicBezierEaseInOut:f},self:g}=v.value;return{"--n-bezier":f,"--n-color":t.embedded?g.colorEmbedded:g.color,"--n-text-color":g.textColor}}),P=l?le("layout",C(()=>t.embedded?"e":""),A,t):void 0;return Object.assign({mergedClsPrefix:a,scrollableElRef:r,scrollbarInstRef:i,hasSiderStyle:N,mergedTheme:v,handleNativeElScroll:H,cssVars:l?void 0:A,themeClass:P==null?void 0:P.themeClass,onRender:P==null?void 0:P.onRender},p)},render(){var t;const{mergedClsPrefix:r,hasSider:i}=this;(t=this.onRender)===null||t===void 0||t.call(this);const a=i?this.hasSiderStyle:void 0,l=[this.themeClass,e&&`${r}-layout-content`,`${r}-layout`,`${r}-layout--${this.position}-positioned`];return d("div",{class:l,style:this.cssVars},this.nativeScrollbar?d("div",{ref:"scrollableElRef",class:[`${r}-layout-scroll-container`,this.contentClass],style:[this.contentStyle,a],onScroll:this.handleNativeElScroll},this.$slots):d(Ne,Object.assign({},this.scrollbarProps,{onScroll:this.onScroll,ref:"scrollbarInstRef",theme:this.mergedTheme.peers.Scrollbar,themeOverrides:this.mergedTheme.peerOverrides.Scrollbar,contentClass:this.contentClass,contentStyle:[this.contentStyle,a]}),this.$slots))}})}const Re=Oe(!1),To=Oe(!0),No=u("layout-header",`
 transition:
 color .3s var(--n-bezier),
 background-color .3s var(--n-bezier),
 box-shadow .3s var(--n-bezier),
 border-color .3s var(--n-bezier);
 box-sizing: border-box;
 width: 100%;
 background-color: var(--n-color);
 color: var(--n-text-color);
`,[w("absolute-positioned",`
 position: absolute;
 left: 0;
 right: 0;
 top: 0;
 `),w("bordered",`
 border-bottom: solid 1px var(--n-border-color);
 `)]),Ao={position:ge,inverted:Boolean,bordered:{type:Boolean,default:!1}},ko=O({name:"LayoutHeader",props:Object.assign(Object.assign({},q.props),Ao),setup(e){const{mergedClsPrefixRef:t,inlineThemeDisabled:r}=ne(e),i=q("Layout","-layout-header",No,fe,e,t),a=C(()=>{const{common:{cubicBezierEaseInOut:v},self:m}=i.value,s={"--n-bezier":v};return e.inverted?(s["--n-color"]=m.headerColorInverted,s["--n-text-color"]=m.textColorInverted,s["--n-border-color"]=m.headerBorderColorInverted):(s["--n-color"]=m.headerColor,s["--n-text-color"]=m.textColor,s["--n-border-color"]=m.headerBorderColor),s}),l=r?le("layout-header",C(()=>e.inverted?"a":"b"),a,e):void 0;return{mergedClsPrefix:t,cssVars:r?void 0:a,themeClass:l==null?void 0:l.themeClass,onRender:l==null?void 0:l.onRender}},render(){var e;const{mergedClsPrefix:t}=this;return(e=this.onRender)===null||e===void 0||e.call(this),d("div",{class:[`${t}-layout-header`,this.themeClass,this.position&&`${t}-layout-header--${this.position}-positioned`,this.bordered&&`${t}-layout-header--bordered`],style:this.cssVars},this.$slots)}}),_o=u("layout-sider",`
 flex-shrink: 0;
 box-sizing: border-box;
 position: relative;
 z-index: 1;
 color: var(--n-text-color);
 transition:
 color .3s var(--n-bezier),
 border-color .3s var(--n-bezier),
 min-width .3s var(--n-bezier),
 max-width .3s var(--n-bezier),
 transform .3s var(--n-bezier),
 background-color .3s var(--n-bezier);
 background-color: var(--n-color);
 display: flex;
 justify-content: flex-end;
`,[w("bordered",[c("border",`
 content: "";
 position: absolute;
 top: 0;
 bottom: 0;
 width: 1px;
 background-color: var(--n-border-color);
 transition: background-color .3s var(--n-bezier);
 `)]),c("left-placement",[w("bordered",[c("border",`
 right: 0;
 `)])]),w("right-placement",`
 justify-content: flex-start;
 `,[w("bordered",[c("border",`
 left: 0;
 `)]),w("collapsed",[u("layout-toggle-button",[u("base-icon",`
 transform: rotate(180deg);
 `)]),u("layout-toggle-bar",[S("&:hover",[c("top",{transform:"rotate(-12deg) scale(1.15) translateY(-2px)"}),c("bottom",{transform:"rotate(12deg) scale(1.15) translateY(2px)"})])])]),u("layout-toggle-button",`
 left: 0;
 transform: translateX(-50%) translateY(-50%);
 `,[u("base-icon",`
 transform: rotate(0);
 `)]),u("layout-toggle-bar",`
 left: -28px;
 transform: rotate(180deg);
 `,[S("&:hover",[c("top",{transform:"rotate(12deg) scale(1.15) translateY(-2px)"}),c("bottom",{transform:"rotate(-12deg) scale(1.15) translateY(2px)"})])])]),w("collapsed",[u("layout-toggle-bar",[S("&:hover",[c("top",{transform:"rotate(-12deg) scale(1.15) translateY(-2px)"}),c("bottom",{transform:"rotate(12deg) scale(1.15) translateY(2px)"})])]),u("layout-toggle-button",[u("base-icon",`
 transform: rotate(0);
 `)])]),u("layout-toggle-button",`
 transition:
 color .3s var(--n-bezier),
 right .3s var(--n-bezier),
 left .3s var(--n-bezier),
 border-color .3s var(--n-bezier),
 background-color .3s var(--n-bezier);
 cursor: pointer;
 width: 24px;
 height: 24px;
 position: absolute;
 top: 50%;
 right: 0;
 border-radius: 50%;
 display: flex;
 align-items: center;
 justify-content: center;
 font-size: 18px;
 color: var(--n-toggle-button-icon-color);
 border: var(--n-toggle-button-border);
 background-color: var(--n-toggle-button-color);
 box-shadow: 0 2px 4px 0px rgba(0, 0, 0, .06);
 transform: translateX(50%) translateY(-50%);
 z-index: 1;
 `,[u("base-icon",`
 transition: transform .3s var(--n-bezier);
 transform: rotate(180deg);
 `)]),u("layout-toggle-bar",`
 cursor: pointer;
 height: 72px;
 width: 32px;
 position: absolute;
 top: calc(50% - 36px);
 right: -28px;
 `,[c("top, bottom",`
 position: absolute;
 width: 4px;
 border-radius: 2px;
 height: 38px;
 left: 14px;
 transition: 
 background-color .3s var(--n-bezier),
 transform .3s var(--n-bezier);
 `),c("bottom",`
 position: absolute;
 top: 34px;
 `),S("&:hover",[c("top",{transform:"rotate(12deg) scale(1.15) translateY(-2px)"}),c("bottom",{transform:"rotate(-12deg) scale(1.15) translateY(2px)"})]),c("top, bottom",{backgroundColor:"var(--n-toggle-bar-color)"}),S("&:hover",[c("top, bottom",{backgroundColor:"var(--n-toggle-bar-color-hover)"})])]),c("border",`
 position: absolute;
 top: 0;
 right: 0;
 bottom: 0;
 width: 1px;
 transition: background-color .3s var(--n-bezier);
 `),u("layout-sider-scroll-container",`
 flex-grow: 1;
 flex-shrink: 0;
 box-sizing: border-box;
 height: 100%;
 opacity: 0;
 transition: opacity .3s var(--n-bezier);
 max-width: 100%;
 `),w("show-content",[u("layout-sider-scroll-container",{opacity:1})]),w("absolute-positioned",`
 position: absolute;
 left: 0;
 top: 0;
 bottom: 0;
 `)]),Ho=O({props:{clsPrefix:{type:String,required:!0},onClick:Function},render(){const{clsPrefix:e}=this;return d("div",{onClick:this.onClick,class:`${e}-layout-toggle-bar`},d("div",{class:`${e}-layout-toggle-bar__top`}),d("div",{class:`${e}-layout-toggle-bar__bottom`}))}}),Bo=O({name:"LayoutToggleButton",props:{clsPrefix:{type:String,required:!0},onClick:Function},render(){const{clsPrefix:e}=this;return d("div",{class:`${e}-layout-toggle-button`,onClick:this.onClick},d(ke,{clsPrefix:e},{default:()=>d(bo,null)}))}}),Oo={position:ge,bordered:Boolean,collapsedWidth:{type:Number,default:48},width:{type:[Number,String],default:272},contentClass:String,contentStyle:{type:[String,Object],default:""},collapseMode:{type:String,default:"transform"},collapsed:{type:Boolean,default:void 0},defaultCollapsed:Boolean,showCollapsedContent:{type:Boolean,default:!0},showTrigger:{type:[Boolean,String],default:!1},nativeScrollbar:{type:Boolean,default:!0},inverted:Boolean,scrollbarProps:Object,triggerClass:String,triggerStyle:[String,Object],collapsedTriggerClass:String,collapsedTriggerStyle:[String,Object],"onUpdate:collapsed":[Function,Array],onUpdateCollapsed:[Function,Array],onAfterEnter:Function,onAfterLeave:Function,onExpand:[Function,Array],onCollapse:[Function,Array],onScroll:Function},Eo=O({name:"LayoutSider",props:Object.assign(Object.assign({},q.props),Oo),setup(e){const t=G(Be),r=$(null),i=$(null),a=$(e.defaultCollapsed),l=ve(re(e,"collapsed"),a),v=C(()=>de(l.value?e.collapsedWidth:e.width)),m=C(()=>e.collapseMode!=="transform"?{}:{minWidth:de(e.width)}),s=C(()=>t?t.siderPlacement:"left");function x(k,z){if(e.nativeScrollbar){const{value:I}=r;I&&(z===void 0?I.scrollTo(k):I.scrollTo(k,z))}else{const{value:I}=i;I&&I.scrollTo(k,z)}}function H(){const{"onUpdate:collapsed":k,onUpdateCollapsed:z,onExpand:I,onCollapse:U}=e,{value:K}=l;z&&M(z,!K),k&&M(k,!K),a.value=!K,K?I&&M(I):U&&M(U)}let N=0,p=0;const A=k=>{var z;const I=k.target;N=I.scrollLeft,p=I.scrollTop,(z=e.onScroll)===null||z===void 0||z.call(e,k)};Ae(()=>{if(e.nativeScrollbar){const k=r.value;k&&(k.scrollTop=p,k.scrollLeft=N)}}),X(He,{collapsedRef:l,collapseModeRef:re(e,"collapseMode")});const{mergedClsPrefixRef:P,inlineThemeDisabled:f}=ne(e),g=q("Layout","-layout-sider",_o,fe,e,P);function T(k){var z,I;k.propertyName==="max-width"&&(l.value?(z=e.onAfterLeave)===null||z===void 0||z.call(e):(I=e.onAfterEnter)===null||I===void 0||I.call(e))}const j={scrollTo:x},D=C(()=>{const{common:{cubicBezierEaseInOut:k},self:z}=g.value,{siderToggleButtonColor:I,siderToggleButtonBorder:U,siderToggleBarColor:K,siderToggleBarColorHover:ie}=z,B={"--n-bezier":k,"--n-toggle-button-color":I,"--n-toggle-button-border":U,"--n-toggle-bar-color":K,"--n-toggle-bar-color-hover":ie};return e.inverted?(B["--n-color"]=z.siderColorInverted,B["--n-text-color"]=z.textColorInverted,B["--n-border-color"]=z.siderBorderColorInverted,B["--n-toggle-button-icon-color"]=z.siderToggleButtonIconColorInverted,B.__invertScrollbar=z.__invertScrollbar):(B["--n-color"]=z.siderColor,B["--n-text-color"]=z.textColor,B["--n-border-color"]=z.siderBorderColor,B["--n-toggle-button-icon-color"]=z.siderToggleButtonIconColor),B}),L=f?le("layout-sider",C(()=>e.inverted?"a":"b"),D,e):void 0;return Object.assign({scrollableElRef:r,scrollbarInstRef:i,mergedClsPrefix:P,mergedTheme:g,styleMaxWidth:v,mergedCollapsed:l,scrollContainerStyle:m,siderPlacement:s,handleNativeElScroll:A,handleTransitionend:T,handleTriggerClick:H,inlineThemeDisabled:f,cssVars:D,themeClass:L==null?void 0:L.themeClass,onRender:L==null?void 0:L.onRender},j)},render(){var e;const{mergedClsPrefix:t,mergedCollapsed:r,showTrigger:i}=this;return(e=this.onRender)===null||e===void 0||e.call(this),d("aside",{class:[`${t}-layout-sider`,this.themeClass,`${t}-layout-sider--${this.position}-positioned`,`${t}-layout-sider--${this.siderPlacement}-placement`,this.bordered&&`${t}-layout-sider--bordered`,r&&`${t}-layout-sider--collapsed`,(!r||this.showCollapsedContent)&&`${t}-layout-sider--show-content`],onTransitionend:this.handleTransitionend,style:[this.inlineThemeDisabled?void 0:this.cssVars,{maxWidth:this.styleMaxWidth,width:de(this.width)}]},this.nativeScrollbar?d("div",{class:[`${t}-layout-sider-scroll-container`,this.contentClass],onScroll:this.handleNativeElScroll,style:[this.scrollContainerStyle,{overflow:"auto"},this.contentStyle],ref:"scrollableElRef"},this.$slots):d(Ne,Object.assign({},this.scrollbarProps,{onScroll:this.onScroll,ref:"scrollbarInstRef",style:this.scrollContainerStyle,contentStyle:this.contentStyle,contentClass:this.contentClass,theme:this.mergedTheme.peers.Scrollbar,themeOverrides:this.mergedTheme.peerOverrides.Scrollbar,builtinThemeOverrides:this.inverted&&this.cssVars.__invertScrollbar==="true"?{colorHover:"rgba(255, 255, 255, .4)",color:"rgba(255, 255, 255, .3)"}:void 0}),this.$slots),i?i==="bar"?d(Ho,{clsPrefix:t,class:r?this.collapsedTriggerClass:this.triggerClass,style:r?this.collapsedTriggerStyle:this.triggerStyle,onClick:this.handleTriggerClick}):d(Bo,{clsPrefix:t,class:r?this.collapsedTriggerClass:this.triggerClass,style:r?this.collapsedTriggerStyle:this.triggerStyle,onClick:this.handleTriggerClick}):null,this.bordered?d("div",{class:`${t}-layout-sider__border`}):null)}}),ee=Q("n-menu"),Ee=Q("n-submenu"),be=Q("n-menu-item-group"),Pe=[S("&::before","background-color: var(--n-item-color-hover);"),c("arrow",`
 color: var(--n-arrow-color-hover);
 `),c("icon",`
 color: var(--n-item-icon-color-hover);
 `),u("menu-item-content-header",`
 color: var(--n-item-text-color-hover);
 `,[S("a",`
 color: var(--n-item-text-color-hover);
 `),c("extra",`
 color: var(--n-item-text-color-hover);
 `)])],Te=[c("icon",`
 color: var(--n-item-icon-color-hover-horizontal);
 `),u("menu-item-content-header",`
 color: var(--n-item-text-color-hover-horizontal);
 `,[S("a",`
 color: var(--n-item-text-color-hover-horizontal);
 `),c("extra",`
 color: var(--n-item-text-color-hover-horizontal);
 `)])],$o=S([u("menu",`
 background-color: var(--n-color);
 color: var(--n-item-text-color);
 overflow: hidden;
 transition: background-color .3s var(--n-bezier);
 box-sizing: border-box;
 font-size: var(--n-font-size);
 padding-bottom: 6px;
 `,[w("horizontal",`
 max-width: 100%;
 width: 100%;
 display: flex;
 overflow: hidden;
 padding-bottom: 0;
 `,[u("submenu","margin: 0;"),u("menu-item","margin: 0;"),u("menu-item-content",`
 padding: 0 20px;
 border-bottom: 2px solid #0000;
 `,[S("&::before","display: none;"),w("selected","border-bottom: 2px solid var(--n-border-color-horizontal)")]),u("menu-item-content",[w("selected",[c("icon","color: var(--n-item-icon-color-active-horizontal);"),u("menu-item-content-header",`
 color: var(--n-item-text-color-active-horizontal);
 `,[S("a","color: var(--n-item-text-color-active-horizontal);"),c("extra","color: var(--n-item-text-color-active-horizontal);")])]),w("child-active",`
 border-bottom: 2px solid var(--n-border-color-horizontal);
 `,[u("menu-item-content-header",`
 color: var(--n-item-text-color-child-active-horizontal);
 `,[S("a",`
 color: var(--n-item-text-color-child-active-horizontal);
 `),c("extra",`
 color: var(--n-item-text-color-child-active-horizontal);
 `)]),c("icon",`
 color: var(--n-item-icon-color-child-active-horizontal);
 `)]),Z("disabled",[Z("selected, child-active",[S("&:focus-within",Te)]),w("selected",[Y(null,[c("icon","color: var(--n-item-icon-color-active-hover-horizontal);"),u("menu-item-content-header",`
 color: var(--n-item-text-color-active-hover-horizontal);
 `,[S("a","color: var(--n-item-text-color-active-hover-horizontal);"),c("extra","color: var(--n-item-text-color-active-hover-horizontal);")])])]),w("child-active",[Y(null,[c("icon","color: var(--n-item-icon-color-child-active-hover-horizontal);"),u("menu-item-content-header",`
 color: var(--n-item-text-color-child-active-hover-horizontal);
 `,[S("a","color: var(--n-item-text-color-child-active-hover-horizontal);"),c("extra","color: var(--n-item-text-color-child-active-hover-horizontal);")])])]),Y("border-bottom: 2px solid var(--n-border-color-horizontal);",Te)]),u("menu-item-content-header",[S("a","color: var(--n-item-text-color-horizontal);")])])]),Z("responsive",[u("menu-item-content-header",`
 overflow: hidden;
 text-overflow: ellipsis;
 `)]),w("collapsed",[u("menu-item-content",[w("selected",[S("&::before",`
 background-color: var(--n-item-color-active-collapsed) !important;
 `)]),u("menu-item-content-header","opacity: 0;"),c("arrow","opacity: 0;"),c("icon","color: var(--n-item-icon-color-collapsed);")])]),u("menu-item",`
 height: var(--n-item-height);
 margin-top: 6px;
 position: relative;
 `),u("menu-item-content",`
 box-sizing: border-box;
 line-height: 1.75;
 height: 100%;
 display: grid;
 grid-template-areas: "icon content arrow";
 grid-template-columns: auto 1fr auto;
 align-items: center;
 cursor: pointer;
 position: relative;
 padding-right: 18px;
 transition:
 background-color .3s var(--n-bezier),
 padding-left .3s var(--n-bezier),
 border-color .3s var(--n-bezier);
 `,[S("> *","z-index: 1;"),S("&::before",`
 z-index: auto;
 content: "";
 background-color: #0000;
 position: absolute;
 left: 8px;
 right: 8px;
 top: 0;
 bottom: 0;
 pointer-events: none;
 border-radius: var(--n-border-radius);
 transition: background-color .3s var(--n-bezier);
 `),w("disabled",`
 opacity: .45;
 cursor: not-allowed;
 `),w("collapsed",[c("arrow","transform: rotate(0);")]),w("selected",[S("&::before","background-color: var(--n-item-color-active);"),c("arrow","color: var(--n-arrow-color-active);"),c("icon","color: var(--n-item-icon-color-active);"),u("menu-item-content-header",`
 color: var(--n-item-text-color-active);
 `,[S("a","color: var(--n-item-text-color-active);"),c("extra","color: var(--n-item-text-color-active);")])]),w("child-active",[u("menu-item-content-header",`
 color: var(--n-item-text-color-child-active);
 `,[S("a",`
 color: var(--n-item-text-color-child-active);
 `),c("extra",`
 color: var(--n-item-text-color-child-active);
 `)]),c("arrow",`
 color: var(--n-arrow-color-child-active);
 `),c("icon",`
 color: var(--n-item-icon-color-child-active);
 `)]),Z("disabled",[Z("selected, child-active",[S("&:focus-within",Pe)]),w("selected",[Y(null,[c("arrow","color: var(--n-arrow-color-active-hover);"),c("icon","color: var(--n-item-icon-color-active-hover);"),u("menu-item-content-header",`
 color: var(--n-item-text-color-active-hover);
 `,[S("a","color: var(--n-item-text-color-active-hover);"),c("extra","color: var(--n-item-text-color-active-hover);")])])]),w("child-active",[Y(null,[c("arrow","color: var(--n-arrow-color-child-active-hover);"),c("icon","color: var(--n-item-icon-color-child-active-hover);"),u("menu-item-content-header",`
 color: var(--n-item-text-color-child-active-hover);
 `,[S("a","color: var(--n-item-text-color-child-active-hover);"),c("extra","color: var(--n-item-text-color-child-active-hover);")])])]),w("selected",[Y(null,[S("&::before","background-color: var(--n-item-color-active-hover);")])]),Y(null,Pe)]),c("icon",`
 grid-area: icon;
 color: var(--n-item-icon-color);
 transition:
 color .3s var(--n-bezier),
 font-size .3s var(--n-bezier),
 margin-right .3s var(--n-bezier);
 box-sizing: content-box;
 display: inline-flex;
 align-items: center;
 justify-content: center;
 `),c("arrow",`
 grid-area: arrow;
 font-size: 16px;
 color: var(--n-arrow-color);
 transform: rotate(180deg);
 opacity: 1;
 transition:
 color .3s var(--n-bezier),
 transform 0.2s var(--n-bezier),
 opacity 0.2s var(--n-bezier);
 `),u("menu-item-content-header",`
 grid-area: content;
 transition:
 color .3s var(--n-bezier),
 opacity .3s var(--n-bezier);
 opacity: 1;
 white-space: nowrap;
 color: var(--n-item-text-color);
 `,[S("a",`
 outline: none;
 text-decoration: none;
 transition: color .3s var(--n-bezier);
 color: var(--n-item-text-color);
 `,[S("&::before",`
 content: "";
 position: absolute;
 left: 0;
 right: 0;
 top: 0;
 bottom: 0;
 `)]),c("extra",`
 font-size: .93em;
 color: var(--n-group-text-color);
 transition: color .3s var(--n-bezier);
 `)])]),u("submenu",`
 cursor: pointer;
 position: relative;
 margin-top: 6px;
 `,[u("menu-item-content",`
 height: var(--n-item-height);
 `),u("submenu-children",`
 overflow: hidden;
 padding: 0;
 `,[eo({duration:".2s"})])]),u("menu-item-group",[u("menu-item-group-title",`
 margin-top: 6px;
 color: var(--n-group-text-color);
 cursor: default;
 font-size: .93em;
 height: 36px;
 display: flex;
 align-items: center;
 transition:
 padding-left .3s var(--n-bezier),
 color .3s var(--n-bezier);
 `)])]),u("menu-tooltip",[S("a",`
 color: inherit;
 text-decoration: none;
 `)]),u("menu-divider",`
 transition: background-color .3s var(--n-bezier);
 background-color: var(--n-divider-color);
 height: 1px;
 margin: 6px 18px;
 `)]);function Y(e,t){return[w("hover",e,t),S("&:hover",e,t)]}const $e=O({name:"MenuOptionContent",props:{collapsed:Boolean,disabled:Boolean,title:[String,Function],icon:Function,extra:[String,Function],showArrow:Boolean,childActive:Boolean,hover:Boolean,paddingLeft:Number,selected:Boolean,maxIconSize:{type:Number,required:!0},activeIconSize:{type:Number,required:!0},iconMarginRight:{type:Number,required:!0},clsPrefix:{type:String,required:!0},onClick:Function,tmNode:{type:Object,required:!0},isEllipsisPlaceholder:Boolean},setup(e){const{props:t}=G(ee);return{menuProps:t,style:C(()=>{const{paddingLeft:r}=e;return{paddingLeft:r&&`${r}px`}}),iconStyle:C(()=>{const{maxIconSize:r,activeIconSize:i,iconMarginRight:a}=e;return{width:`${r}px`,height:`${r}px`,fontSize:`${i}px`,marginRight:`${a}px`}})}},render(){const{clsPrefix:e,tmNode:t,menuProps:{renderIcon:r,renderLabel:i,renderExtra:a,expandIcon:l}}=this,v=r?r(t.rawNode):W(this.icon);return d("div",{onClick:m=>{var s;(s=this.onClick)===null||s===void 0||s.call(this,m)},role:"none",class:[`${e}-menu-item-content`,{[`${e}-menu-item-content--selected`]:this.selected,[`${e}-menu-item-content--collapsed`]:this.collapsed,[`${e}-menu-item-content--child-active`]:this.childActive,[`${e}-menu-item-content--disabled`]:this.disabled,[`${e}-menu-item-content--hover`]:this.hover}],style:this.style},v&&d("div",{class:`${e}-menu-item-content__icon`,style:this.iconStyle,role:"none"},[v]),d("div",{class:`${e}-menu-item-content-header`,role:"none"},this.isEllipsisPlaceholder?this.title:i?i(t.rawNode):W(this.title),this.extra||a?d("span",{class:`${e}-menu-item-content-header__extra`}," ",a?a(t.rawNode):W(this.extra)):null),this.showArrow?d(ke,{ariaHidden:!0,class:`${e}-menu-item-content__arrow`,clsPrefix:e},{default:()=>l?l(t.rawNode):d(So,null)}):null)}}),te=8;function xe(e){const t=G(ee),{props:r,mergedCollapsedRef:i}=t,a=G(Ee,null),l=G(be,null),v=C(()=>r.mode==="horizontal"),m=C(()=>v.value?r.dropdownPlacement:"tmNodes"in e?"right-start":"right"),s=C(()=>{var p;return Math.max((p=r.collapsedIconSize)!==null&&p!==void 0?p:r.iconSize,r.iconSize)}),x=C(()=>{var p;return!v.value&&e.root&&i.value&&(p=r.collapsedIconSize)!==null&&p!==void 0?p:r.iconSize}),H=C(()=>{if(v.value)return;const{collapsedWidth:p,indent:A,rootIndent:P}=r,{root:f,isGroup:g}=e,T=P===void 0?A:P;return f?i.value?p/2-s.value/2:T:l&&typeof l.paddingLeftRef.value=="number"?A/2+l.paddingLeftRef.value:a&&typeof a.paddingLeftRef.value=="number"?(g?A/2:A)+a.paddingLeftRef.value:0}),N=C(()=>{const{collapsedWidth:p,indent:A,rootIndent:P}=r,{value:f}=s,{root:g}=e;return v.value||!g||!i.value?te:(P===void 0?A:P)+f+te-(p+f)/2});return{dropdownPlacement:m,activeIconSize:x,maxIconSize:s,paddingLeft:H,iconMarginRight:N,NMenu:t,NSubmenu:a,NMenuOptionGroup:l}}const Ce={internalKey:{type:[String,Number],required:!0},root:Boolean,isGroup:Boolean,level:{type:Number,required:!0},title:[String,Function],extra:[String,Function]},Lo=O({name:"MenuDivider",setup(){const e=G(ee),{mergedClsPrefixRef:t,isHorizontalRef:r}=e;return()=>r.value?null:d("div",{class:`${t.value}-menu-divider`})}}),Le=Object.assign(Object.assign({},Ce),{tmNode:{type:Object,required:!0},disabled:Boolean,icon:Function,onClick:Function}),Fo=pe(Le),Mo=O({name:"MenuOption",props:Le,setup(e){const t=xe(e),{NSubmenu:r,NMenu:i,NMenuOptionGroup:a}=t,{props:l,mergedClsPrefixRef:v,mergedCollapsedRef:m}=i,s=r?r.mergedDisabledRef:a?a.mergedDisabledRef:{value:!1},x=C(()=>s.value||e.disabled);function H(p){const{onClick:A}=e;A&&A(p)}function N(p){x.value||(i.doSelect(e.internalKey,e.tmNode.rawNode),H(p))}return{mergedClsPrefix:v,dropdownPlacement:t.dropdownPlacement,paddingLeft:t.paddingLeft,iconMarginRight:t.iconMarginRight,maxIconSize:t.maxIconSize,activeIconSize:t.activeIconSize,mergedTheme:i.mergedThemeRef,menuProps:l,dropdownEnabled:ue(()=>e.root&&m.value&&l.mode!=="horizontal"&&!x.value),selected:ue(()=>i.mergedValueRef.value===e.internalKey),mergedDisabled:x,handleClick:N}},render(){const{mergedClsPrefix:e,mergedTheme:t,tmNode:r,menuProps:{renderLabel:i,nodeProps:a}}=this,l=a==null?void 0:a(r.rawNode);return d("div",Object.assign({},l,{role:"menuitem",class:[`${e}-menu-item`,l==null?void 0:l.class]}),d(xo,{theme:t.peers.Tooltip,themeOverrides:t.peerOverrides.Tooltip,trigger:"hover",placement:this.dropdownPlacement,disabled:!this.dropdownEnabled||this.title===void 0,internalExtraClass:["menu-tooltip"]},{default:()=>i?i(r.rawNode):W(this.title),trigger:()=>d($e,{tmNode:r,clsPrefix:e,paddingLeft:this.paddingLeft,iconMarginRight:this.iconMarginRight,maxIconSize:this.maxIconSize,activeIconSize:this.activeIconSize,selected:this.selected,title:this.title,extra:this.extra,disabled:this.mergedDisabled,icon:this.icon,onClick:this.handleClick})}))}}),Fe=Object.assign(Object.assign({},Ce),{tmNode:{type:Object,required:!0},tmNodes:{type:Array,required:!0}}),jo=pe(Fe),Ko=O({name:"MenuOptionGroup",props:Fe,setup(e){const t=xe(e),{NSubmenu:r}=t,i=C(()=>r!=null&&r.mergedDisabledRef.value?!0:e.tmNode.disabled);X(be,{paddingLeftRef:t.paddingLeft,mergedDisabledRef:i});const{mergedClsPrefixRef:a,props:l}=G(ee);return function(){const{value:v}=a,m=t.paddingLeft.value,{nodeProps:s}=l,x=s==null?void 0:s(e.tmNode.rawNode);return d("div",{class:`${v}-menu-item-group`,role:"group"},d("div",Object.assign({},x,{class:[`${v}-menu-item-group-title`,x==null?void 0:x.class],style:[(x==null?void 0:x.style)||"",m!==void 0?`padding-left: ${m}px;`:""]}),W(e.title),e.extra?d(oo,null," ",W(e.extra)):null),d("div",null,e.tmNodes.map(H=>ye(H,l))))}}});function he(e){return e.type==="divider"||e.type==="render"}function Vo(e){return e.type==="divider"}function ye(e,t){const{rawNode:r}=e,{show:i}=r;if(i===!1)return null;if(he(r))return Vo(r)?d(Lo,Object.assign({key:e.key},r.props)):null;const{labelField:a}=t,{key:l,level:v,isGroup:m}=e,s=Object.assign(Object.assign({},r),{title:r.title||r[a],extra:r.titleExtra||r.extra,key:l,internalKey:l,level:v,root:v===0,isGroup:m});return e.children?e.isGroup?d(Ko,ae(s,jo,{tmNode:e,tmNodes:e.children,key:l})):d(me,ae(s,Do,{key:l,rawNodes:r[t.childrenField],tmNodes:e.children,tmNode:e})):d(Mo,ae(s,Fo,{key:l,tmNode:e}))}const Me=Object.assign(Object.assign({},Ce),{rawNodes:{type:Array,default:()=>[]},tmNodes:{type:Array,default:()=>[]},tmNode:{type:Object,required:!0},disabled:Boolean,icon:Function,onClick:Function,domId:String,virtualChildActive:{type:Boolean,default:void 0},isEllipsisPlaceholder:Boolean}),Do=pe(Me),me=O({name:"Submenu",props:Me,setup(e){const t=xe(e),{NMenu:r,NSubmenu:i}=t,{props:a,mergedCollapsedRef:l,mergedThemeRef:v}=r,m=C(()=>{const{disabled:p}=e;return i!=null&&i.mergedDisabledRef.value||a.disabled?!0:p}),s=$(!1);X(Ee,{paddingLeftRef:t.paddingLeft,mergedDisabledRef:m}),X(be,null);function x(){const{onClick:p}=e;p&&p()}function H(){m.value||(l.value||r.toggleExpand(e.internalKey),x())}function N(p){s.value=p}return{menuProps:a,mergedTheme:v,doSelect:r.doSelect,inverted:r.invertedRef,isHorizontal:r.isHorizontalRef,mergedClsPrefix:r.mergedClsPrefixRef,maxIconSize:t.maxIconSize,activeIconSize:t.activeIconSize,iconMarginRight:t.iconMarginRight,dropdownPlacement:t.dropdownPlacement,dropdownShow:s,paddingLeft:t.paddingLeft,mergedDisabled:m,mergedValue:r.mergedValueRef,childActive:ue(()=>{var p;return(p=e.virtualChildActive)!==null&&p!==void 0?p:r.activePathRef.value.includes(e.internalKey)}),collapsed:C(()=>a.mode==="horizontal"?!1:l.value?!0:!r.mergedExpandedKeysRef.value.includes(e.internalKey)),dropdownEnabled:C(()=>!m.value&&(a.mode==="horizontal"||l.value)),handlePopoverShowChange:N,handleClick:H}},render(){var e;const{mergedClsPrefix:t,menuProps:{renderIcon:r,renderLabel:i}}=this,a=()=>{const{isHorizontal:v,paddingLeft:m,collapsed:s,mergedDisabled:x,maxIconSize:H,activeIconSize:N,title:p,childActive:A,icon:P,handleClick:f,menuProps:{nodeProps:g},dropdownShow:T,iconMarginRight:j,tmNode:D,mergedClsPrefix:L,isEllipsisPlaceholder:k,extra:z}=this,I=g==null?void 0:g(D.rawNode);return d("div",Object.assign({},I,{class:[`${L}-menu-item`,I==null?void 0:I.class],role:"menuitem"}),d($e,{tmNode:D,paddingLeft:m,collapsed:s,disabled:x,iconMarginRight:j,maxIconSize:H,activeIconSize:N,title:p,extra:z,showArrow:!v,childActive:A,clsPrefix:L,icon:P,hover:T,onClick:f,isEllipsisPlaceholder:k}))},l=()=>d(to,null,{default:()=>{const{tmNodes:v,collapsed:m}=this;return m?null:d("div",{class:`${t}-submenu-children`,role:"menu"},v.map(s=>ye(s,this.menuProps)))}});return this.root?d(_e,Object.assign({size:"large",trigger:"hover"},(e=this.menuProps)===null||e===void 0?void 0:e.dropdownProps,{themeOverrides:this.mergedTheme.peerOverrides.Dropdown,theme:this.mergedTheme.peers.Dropdown,builtinThemeOverrides:{fontSizeLarge:"14px",optionIconSizeLarge:"18px"},value:this.mergedValue,disabled:!this.dropdownEnabled,placement:this.dropdownPlacement,keyField:this.menuProps.keyField,labelField:this.menuProps.labelField,childrenField:this.menuProps.childrenField,onUpdateShow:this.handlePopoverShowChange,options:this.rawNodes,onSelect:this.doSelect,inverted:this.inverted,renderIcon:r,renderLabel:i}),{default:()=>d("div",{class:`${t}-submenu`,role:"menu","aria-expanded":!this.collapsed,id:this.domId},a(),this.isHorizontal?null:l())}):d("div",{class:`${t}-submenu`,role:"menu","aria-expanded":!this.collapsed,id:this.domId},a(),l())}}),Uo=Object.assign(Object.assign({},q.props),{options:{type:Array,default:()=>[]},collapsed:{type:Boolean,default:void 0},collapsedWidth:{type:Number,default:48},iconSize:{type:Number,default:20},collapsedIconSize:{type:Number,default:24},rootIndent:Number,indent:{type:Number,default:32},labelField:{type:String,default:"label"},keyField:{type:String,default:"key"},childrenField:{type:String,default:"children"},disabledField:{type:String,default:"disabled"},defaultExpandAll:Boolean,defaultExpandedKeys:Array,expandedKeys:Array,value:[String,Number],defaultValue:{type:[String,Number],default:null},mode:{type:String,default:"vertical"},watchProps:{type:Array,default:void 0},disabled:Boolean,show:{type:Boolean,default:!0},inverted:Boolean,"onUpdate:expandedKeys":[Function,Array],onUpdateExpandedKeys:[Function,Array],onUpdateValue:[Function,Array],"onUpdate:value":[Function,Array],expandIcon:Function,renderIcon:Function,renderLabel:Function,renderExtra:Function,dropdownProps:Object,accordion:Boolean,nodeProps:Function,dropdownPlacement:{type:String,default:"bottom"},responsive:Boolean,items:Array,onOpenNamesChange:[Function,Array],onSelect:[Function,Array],onExpandedNamesChange:[Function,Array],expandedNames:Array,defaultExpandedNames:Array}),Go=O({name:"Menu",inheritAttrs:!1,props:Uo,setup(e){const{mergedClsPrefixRef:t,inlineThemeDisabled:r}=ne(e),i=q("Menu","-menu",$o,io,e,t),a=G(He,null),l=C(()=>{var h;const{collapsed:y}=e;if(y!==void 0)return y;if(a){const{collapseModeRef:o,collapsedRef:b}=a;if(o.value==="width")return(h=b.value)!==null&&h!==void 0?h:!1}return!1}),v=C(()=>{const{keyField:h,childrenField:y,disabledField:o}=e;return ce(e.items||e.options,{getIgnored(b){return he(b)},getChildren(b){return b[y]},getDisabled(b){return b[o]},getKey(b){var R;return(R=b[h])!==null&&R!==void 0?R:b.name}})}),m=C(()=>new Set(v.value.treeNodes.map(h=>h.key))),{watchProps:s}=e,x=$(null);s!=null&&s.includes("defaultValue")?Se(()=>{x.value=e.defaultValue}):x.value=e.defaultValue;const H=re(e,"value"),N=ve(H,x),p=$([]),A=()=>{p.value=e.defaultExpandAll?v.value.getNonLeafKeys():e.defaultExpandedNames||e.defaultExpandedKeys||v.value.getPath(N.value,{includeSelf:!1}).keyPath};s!=null&&s.includes("defaultExpandedKeys")?Se(A):A();const P=zo(e,["expandedNames","expandedKeys"]),f=ve(P,p),g=C(()=>v.value.treeNodes),T=C(()=>v.value.getPath(N.value).keyPath);X(ee,{props:e,mergedCollapsedRef:l,mergedThemeRef:i,mergedValueRef:N,mergedExpandedKeysRef:f,activePathRef:T,mergedClsPrefixRef:t,isHorizontalRef:C(()=>e.mode==="horizontal"),invertedRef:re(e,"inverted"),doSelect:j,toggleExpand:L});function j(h,y){const{"onUpdate:value":o,onUpdateValue:b,onSelect:R}=e;b&&M(b,h,y),o&&M(o,h,y),R&&M(R,h,y),x.value=h}function D(h){const{"onUpdate:expandedKeys":y,onUpdateExpandedKeys:o,onExpandedNamesChange:b,onOpenNamesChange:R}=e;y&&M(y,h),o&&M(o,h),b&&M(b,h),R&&M(R,h),p.value=h}function L(h){const y=Array.from(f.value),o=y.findIndex(b=>b===h);if(~o)y.splice(o,1);else{if(e.accordion&&m.value.has(h)){const b=y.findIndex(R=>m.value.has(R));b>-1&&y.splice(b,1)}y.push(h)}D(y)}const k=h=>{const y=v.value.getPath(h??N.value,{includeSelf:!1}).keyPath;if(!y.length)return;const o=Array.from(f.value),b=new Set([...o,...y]);e.accordion&&m.value.forEach(R=>{b.has(R)&&!y.includes(R)&&b.delete(R)}),D(Array.from(b))},z=C(()=>{const{inverted:h}=e,{common:{cubicBezierEaseInOut:y},self:o}=i.value,{borderRadius:b,borderColorHorizontal:R,fontSize:Ye,itemHeight:We,dividerColor:Xe}=o,n={"--n-divider-color":Xe,"--n-bezier":y,"--n-font-size":Ye,"--n-border-color-horizontal":R,"--n-border-radius":b,"--n-item-height":We};return h?(n["--n-group-text-color"]=o.groupTextColorInverted,n["--n-color"]=o.colorInverted,n["--n-item-text-color"]=o.itemTextColorInverted,n["--n-item-text-color-hover"]=o.itemTextColorHoverInverted,n["--n-item-text-color-active"]=o.itemTextColorActiveInverted,n["--n-item-text-color-child-active"]=o.itemTextColorChildActiveInverted,n["--n-item-text-color-child-active-hover"]=o.itemTextColorChildActiveInverted,n["--n-item-text-color-active-hover"]=o.itemTextColorActiveHoverInverted,n["--n-item-icon-color"]=o.itemIconColorInverted,n["--n-item-icon-color-hover"]=o.itemIconColorHoverInverted,n["--n-item-icon-color-active"]=o.itemIconColorActiveInverted,n["--n-item-icon-color-active-hover"]=o.itemIconColorActiveHoverInverted,n["--n-item-icon-color-child-active"]=o.itemIconColorChildActiveInverted,n["--n-item-icon-color-child-active-hover"]=o.itemIconColorChildActiveHoverInverted,n["--n-item-icon-color-collapsed"]=o.itemIconColorCollapsedInverted,n["--n-item-text-color-horizontal"]=o.itemTextColorHorizontalInverted,n["--n-item-text-color-hover-horizontal"]=o.itemTextColorHoverHorizontalInverted,n["--n-item-text-color-active-horizontal"]=o.itemTextColorActiveHorizontalInverted,n["--n-item-text-color-child-active-horizontal"]=o.itemTextColorChildActiveHorizontalInverted,n["--n-item-text-color-child-active-hover-horizontal"]=o.itemTextColorChildActiveHoverHorizontalInverted,n["--n-item-text-color-active-hover-horizontal"]=o.itemTextColorActiveHoverHorizontalInverted,n["--n-item-icon-color-horizontal"]=o.itemIconColorHorizontalInverted,n["--n-item-icon-color-hover-horizontal"]=o.itemIconColorHoverHorizontalInverted,n["--n-item-icon-color-active-horizontal"]=o.itemIconColorActiveHorizontalInverted,n["--n-item-icon-color-active-hover-horizontal"]=o.itemIconColorActiveHoverHorizontalInverted,n["--n-item-icon-color-child-active-horizontal"]=o.itemIconColorChildActiveHorizontalInverted,n["--n-item-icon-color-child-active-hover-horizontal"]=o.itemIconColorChildActiveHoverHorizontalInverted,n["--n-arrow-color"]=o.arrowColorInverted,n["--n-arrow-color-hover"]=o.arrowColorHoverInverted,n["--n-arrow-color-active"]=o.arrowColorActiveInverted,n["--n-arrow-color-active-hover"]=o.arrowColorActiveHoverInverted,n["--n-arrow-color-child-active"]=o.arrowColorChildActiveInverted,n["--n-arrow-color-child-active-hover"]=o.arrowColorChildActiveHoverInverted,n["--n-item-color-hover"]=o.itemColorHoverInverted,n["--n-item-color-active"]=o.itemColorActiveInverted,n["--n-item-color-active-hover"]=o.itemColorActiveHoverInverted,n["--n-item-color-active-collapsed"]=o.itemColorActiveCollapsedInverted):(n["--n-group-text-color"]=o.groupTextColor,n["--n-color"]=o.color,n["--n-item-text-color"]=o.itemTextColor,n["--n-item-text-color-hover"]=o.itemTextColorHover,n["--n-item-text-color-active"]=o.itemTextColorActive,n["--n-item-text-color-child-active"]=o.itemTextColorChildActive,n["--n-item-text-color-child-active-hover"]=o.itemTextColorChildActiveHover,n["--n-item-text-color-active-hover"]=o.itemTextColorActiveHover,n["--n-item-icon-color"]=o.itemIconColor,n["--n-item-icon-color-hover"]=o.itemIconColorHover,n["--n-item-icon-color-active"]=o.itemIconColorActive,n["--n-item-icon-color-active-hover"]=o.itemIconColorActiveHover,n["--n-item-icon-color-child-active"]=o.itemIconColorChildActive,n["--n-item-icon-color-child-active-hover"]=o.itemIconColorChildActiveHover,n["--n-item-icon-color-collapsed"]=o.itemIconColorCollapsed,n["--n-item-text-color-horizontal"]=o.itemTextColorHorizontal,n["--n-item-text-color-hover-horizontal"]=o.itemTextColorHoverHorizontal,n["--n-item-text-color-active-horizontal"]=o.itemTextColorActiveHorizontal,n["--n-item-text-color-child-active-horizontal"]=o.itemTextColorChildActiveHorizontal,n["--n-item-text-color-child-active-hover-horizontal"]=o.itemTextColorChildActiveHoverHorizontal,n["--n-item-text-color-active-hover-horizontal"]=o.itemTextColorActiveHoverHorizontal,n["--n-item-icon-color-horizontal"]=o.itemIconColorHorizontal,n["--n-item-icon-color-hover-horizontal"]=o.itemIconColorHoverHorizontal,n["--n-item-icon-color-active-horizontal"]=o.itemIconColorActiveHorizontal,n["--n-item-icon-color-active-hover-horizontal"]=o.itemIconColorActiveHoverHorizontal,n["--n-item-icon-color-child-active-horizontal"]=o.itemIconColorChildActiveHorizontal,n["--n-item-icon-color-child-active-hover-horizontal"]=o.itemIconColorChildActiveHoverHorizontal,n["--n-arrow-color"]=o.arrowColor,n["--n-arrow-color-hover"]=o.arrowColorHover,n["--n-arrow-color-active"]=o.arrowColorActive,n["--n-arrow-color-active-hover"]=o.arrowColorActiveHover,n["--n-arrow-color-child-active"]=o.arrowColorChildActive,n["--n-arrow-color-child-active-hover"]=o.arrowColorChildActiveHover,n["--n-item-color-hover"]=o.itemColorHover,n["--n-item-color-active"]=o.itemColorActive,n["--n-item-color-active-hover"]=o.itemColorActiveHover,n["--n-item-color-active-collapsed"]=o.itemColorActiveCollapsed),n}),I=r?le("menu",C(()=>e.inverted?"a":"b"),z,e):void 0,U=no(),K=$(null),ie=$(null);let B=!0;const ze=()=>{var h;B?B=!1:(h=K.value)===null||h===void 0||h.sync({showAllItemsBeforeCalculate:!0})};function je(){return document.getElementById(U)}const oe=$(-1);function Ke(h){oe.value=e.options.length-h}function Ve(h){h||(oe.value=-1)}const De=C(()=>{const h=oe.value;return{children:h===-1?[]:e.options.slice(h)}}),Ue=C(()=>{const{childrenField:h,disabledField:y,keyField:o}=e;return ce([De.value],{getIgnored(b){return he(b)},getChildren(b){return b[h]},getDisabled(b){return b[y]},getKey(b){var R;return(R=b[o])!==null&&R!==void 0?R:b.name}})}),Ge=C(()=>ce([{}]).treeNodes[0]);function qe(){var h;if(oe.value===-1)return d(me,{root:!0,level:0,key:"__ellpisisGroupPlaceholder__",internalKey:"__ellpisisGroupPlaceholder__",title:"···",tmNode:Ge.value,domId:U,isEllipsisPlaceholder:!0});const y=Ue.value.treeNodes[0],o=T.value,b=!!(!((h=y.children)===null||h===void 0)&&h.some(R=>o.includes(R.key)));return d(me,{level:0,root:!0,key:"__ellpisisGroup__",internalKey:"__ellpisisGroup__",title:"···",virtualChildActive:b,tmNode:y,domId:U,rawNodes:y.rawNode.children||[],tmNodes:y.children||[],isEllipsisPlaceholder:!0})}return{mergedClsPrefix:t,controlledExpandedKeys:P,uncontrolledExpanededKeys:p,mergedExpandedKeys:f,uncontrolledValue:x,mergedValue:N,activePath:T,tmNodes:g,mergedTheme:i,mergedCollapsed:l,cssVars:r?void 0:z,themeClass:I==null?void 0:I.themeClass,overflowRef:K,counterRef:ie,updateCounter:()=>{},onResize:ze,onUpdateOverflow:Ve,onUpdateCount:Ke,renderCounter:qe,getCounter:je,onRender:I==null?void 0:I.onRender,showOption:k,deriveResponsiveState:ze}},render(){const{mergedClsPrefix:e,mode:t,themeClass:r,onRender:i}=this;i==null||i();const a=()=>this.tmNodes.map(s=>ye(s,this.$props)),v=t==="horizontal"&&this.responsive,m=()=>d("div",lo(this.$attrs,{role:t==="horizontal"?"menubar":"menu",class:[`${e}-menu`,r,`${e}-menu--${t}`,v&&`${e}-menu--responsive`,this.mergedCollapsed&&`${e}-menu--collapsed`],style:this.cssVars}),v?d(Co,{ref:"overflowRef",onUpdateOverflow:this.onUpdateOverflow,getCounter:this.getCounter,onUpdateCount:this.onUpdateCount,updateCounter:this.updateCounter,style:{width:"100%",display:"flex",overflow:"hidden"}},{default:a,counter:this.renderCounter}):a());return v?d(ro,{onResize:this.onResize},{default:m}):m()}}),qo={class:"panel-brand"},Yo={key:0,class:"panel-brand-copy"},Wo={class:"panel-brand-subtitle"},Xo={class:"panel-topbar-left"},Zo={class:"panel-route"},Jo={class:"panel-route-title"},Qo={class:"panel-route-path"},et={class:"panel-topbar-actions"},ot={class:"panel-content-wrap"},tt=O({__name:"AppShell",setup(e){const t=go(),r=po(),i=ao(),a=so(),{t:l}=co(),v=$(i.locale),m=C(()=>r.path.split("/")[1]||"dashboard"),s=C(()=>[{label:l("nav.dashboard"),key:"dashboard"},{label:l("nav.readiness"),key:"readiness"},{label:l("nav.inbounds"),key:"inbounds"},{label:l("nav.outbounds"),key:"outbounds"},{label:l("nav.users"),key:"users"},{label:l("nav.subscriptions"),key:"subscriptions"},{label:l("nav.nodePool"),key:"node-pool"},{label:l("nav.routing"),key:"routing"},{label:l("nav.dns"),key:"dns"},{label:l("nav.monitor"),key:"monitor"},{label:l("nav.settings"),key:"settings"},{label:l("nav.config"),key:"config"},{label:l("nav.support"),key:"support"}]),x=[{label:"中文",value:"zh-CN"},{label:"English",value:"en"}],H=[{label:l("auth.logout"),key:"logout"}],N=C(()=>{const f=s.value.find(g=>g.key===m.value);return typeof(f==null?void 0:f.label)=="string"?f.label:"Xray Panel"});function p(f){t.push("/"+f)}function A(f){i.setLocale(f),window.location.reload()}function P(f){f==="logout"&&a.logout()}return(f,g)=>{const T=fo("router-view");return we(),uo(_(Re),{"has-sider":"",class:"panel-shell"},{default:V(()=>[F(_(Eo),{bordered:"",collapsed:_(i).sidebarCollapsed,"collapse-mode":"width","collapsed-width":64,width:220,"show-trigger":"",onCollapse:g[0]||(g[0]=j=>_(i).sidebarCollapsed=!0),onExpand:g[1]||(g[1]=j=>_(i).sidebarCollapsed=!1),"native-scrollbar":!1,class:"panel-sider"},{default:V(()=>[E("div",qo,[g[4]||(g[4]=E("span",{class:"panel-brand-mark"},"X",-1)),_(i).sidebarCollapsed?ho("",!0):(we(),vo("div",Yo,[g[3]||(g[3]=E("span",{class:"panel-brand-title"},"Xray Panel",-1)),E("span",Wo,J(N.value),1)]))]),F(_(Go),{collapsed:_(i).sidebarCollapsed,"collapsed-width":64,"collapsed-icon-size":22,options:s.value,value:m.value,"onUpdate:value":p,class:"panel-menu"},null,8,["collapsed","options","value"])]),_:1},8,["collapsed"]),F(_(Re),{class:"panel-main"},{default:V(()=>[F(_(ko),{bordered:"",class:"panel-topbar"},{default:V(()=>[E("div",Xo,[F(_(se),{quaternary:"",circle:"",size:"small",onClick:_(i).toggleSidebar,class:"mobile-menu"},{icon:V(()=>[...g[5]||(g[5]=[E("span",{class:"panel-menu-icon"},"☰",-1)])]),_:1},8,["onClick"]),E("div",Zo,[E("span",Jo,J(N.value),1),E("span",Qo,J(_(r).path),1)])]),E("div",et,[F(_(yo),{value:v.value,"onUpdate:value":[g[2]||(g[2]=j=>v.value=j),A],options:x,size:"small",class:"panel-locale-select"},null,8,["value"]),F(_(se),{quaternary:"",circle:"",size:"small",onClick:_(i).toggleTheme},{icon:V(()=>[E("span",null,J(_(i).isDark?"☀":"☾"),1)]),_:1},8,["onClick"]),F(_(_e),{options:H,onSelect:P},{default:V(()=>[F(_(se),{quaternary:"",size:"small",class:"panel-user-button"},{default:V(()=>[mo(J(_(a).username),1)]),_:1})]),_:1})])]),_:1}),F(_(To),{class:"panel-content","native-scrollbar":!1},{default:V(()=>[E("div",ot,[F(T)])]),_:1})]),_:1})]),_:1})}}}),dt=Io(tt,[["__scopeId","data-v-a1c74caa"]]);export{dt as default};
