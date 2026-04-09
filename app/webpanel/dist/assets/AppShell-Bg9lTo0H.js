import{d as O,i as d,j as Je,s as Qe,k as eo,l as Ie,m as J,n as u,p as w,S as Ae,q as re,v as U,x as ke,y as ne,z as x,r as E,A as Y,C as c,D as I,E as _e,F as D,G as F,H as te,I as X,J as oo,K as q,L as pe,M as ue,O as to,P as ie,Q as ro,V as no,R as Se,T as lo,U as io,W as ao,X as so,a as co,u as uo,Y as vo,w as j,e as _,o as ae,b as L,Z,c as we,B as se,t as Re,g as ho,_ as mo,$ as po,a0 as go}from"./index-7x_dIBDU.js";import{C as fo,N as bo,a as He,V as xo,c as ce,b as Co}from"./Dropdown-Vox1zOhl.js";import{f as de,u as ve}from"./Suffix-DkltMwHY.js";import{u as yo}from"./Tag-BCH1rFsL.js";import"./use-locale-vsCeAob5.js";const zo=O({name:"ChevronDownFilled",render(){return d("svg",{viewBox:"0 0 16 16",fill:"none",xmlns:"http://www.w3.org/2000/svg"},d("path",{d:"M3.20041 5.73966C3.48226 5.43613 3.95681 5.41856 4.26034 5.70041L8 9.22652L11.7397 5.70041C12.0432 5.41856 12.5177 5.43613 12.7996 5.73966C13.0815 6.0432 13.0639 6.51775 12.7603 6.7996L8.51034 10.7996C8.22258 11.0668 7.77743 11.0668 7.48967 10.7996L3.23966 6.7996C2.93613 6.51775 2.91856 6.0432 3.20041 5.73966Z",fill:"currentColor"}))}});function Io(e){const{baseColor:t,textColor2:r,bodyColor:i,cardColor:a,dividerColor:l,actionColor:v,scrollbarColor:m,scrollbarColorHover:s,invertedColor:f}=e;return{textColor:r,textColorInverted:"#FFF",color:i,colorEmbedded:v,headerColor:a,headerColorInverted:f,footerColor:v,footerColorInverted:f,headerBorderColor:l,headerBorderColorInverted:f,footerBorderColor:l,footerBorderColorInverted:f,siderBorderColor:l,siderBorderColorInverted:f,siderColor:a,siderColorInverted:f,siderToggleButtonBorder:`1px solid ${l}`,siderToggleButtonColor:t,siderToggleButtonIconColor:r,siderToggleButtonIconColorInverted:r,siderToggleBarColor:Ie(i,m),siderToggleBarColorHover:Ie(i,s),__invertScrollbar:"true"}}const ge=Je({name:"Layout",common:eo,peers:{Scrollbar:Qe},self:Io}),Be=J("n-layout-sider"),fe={type:String,default:"static"},So=u("layout",`
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
 `)]),wo={embedded:Boolean,position:fe,nativeScrollbar:{type:Boolean,default:!0},scrollbarProps:Object,onScroll:Function,contentClass:String,contentStyle:{type:[String,Object],default:""},hasSider:Boolean,siderPlacement:{type:String,default:"left"}},Oe=J("n-layout");function Ee(e){return O({name:e?"LayoutContent":"Layout",props:Object.assign(Object.assign({},U.props),wo),setup(t){const r=E(null),i=E(null),{mergedClsPrefixRef:a,inlineThemeDisabled:l}=re(t),v=U("Layout","-layout",So,ge,t,a);function m(b,R){if(t.nativeScrollbar){const{value:P}=r;P&&(R===void 0?P.scrollTo(b):P.scrollTo(b,R))}else{const{value:P}=i;P&&P.scrollTo(b,R)}}Y(Oe,t);let s=0,f=0;const H=b=>{var R;const P=b.target;s=P.scrollLeft,f=P.scrollTop,(R=t.onScroll)===null||R===void 0||R.call(t,b)};ke(()=>{if(t.nativeScrollbar){const b=r.value;b&&(b.scrollTop=f,b.scrollLeft=s)}});const k={display:"flex",flexWrap:"nowrap",width:"100%",flexDirection:"row"},p={scrollTo:m},N=x(()=>{const{common:{cubicBezierEaseInOut:b},self:R}=v.value;return{"--n-bezier":b,"--n-color":t.embedded?R.colorEmbedded:R.color,"--n-text-color":R.textColor}}),S=l?ne("layout",x(()=>t.embedded?"e":""),N,t):void 0;return Object.assign({mergedClsPrefix:a,scrollableElRef:r,scrollbarInstRef:i,hasSiderStyle:k,mergedTheme:v,handleNativeElScroll:H,cssVars:l?void 0:N,themeClass:S==null?void 0:S.themeClass,onRender:S==null?void 0:S.onRender},p)},render(){var t;const{mergedClsPrefix:r,hasSider:i}=this;(t=this.onRender)===null||t===void 0||t.call(this);const a=i?this.hasSiderStyle:void 0,l=[this.themeClass,e&&`${r}-layout-content`,`${r}-layout`,`${r}-layout--${this.position}-positioned`];return d("div",{class:l,style:this.cssVars},this.nativeScrollbar?d("div",{ref:"scrollableElRef",class:[`${r}-layout-scroll-container`,this.contentClass],style:[this.contentStyle,a],onScroll:this.handleNativeElScroll},this.$slots):d(Ae,Object.assign({},this.scrollbarProps,{onScroll:this.onScroll,ref:"scrollbarInstRef",theme:this.mergedTheme.peers.Scrollbar,themeOverrides:this.mergedTheme.peerOverrides.Scrollbar,contentClass:this.contentClass,contentStyle:[this.contentStyle,a]}),this.$slots))}})}const Pe=Ee(!1),Ro=Ee(!0),Po=u("layout-header",`
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
 `)]),To={position:fe,inverted:Boolean,bordered:{type:Boolean,default:!1}},No=O({name:"LayoutHeader",props:Object.assign(Object.assign({},U.props),To),setup(e){const{mergedClsPrefixRef:t,inlineThemeDisabled:r}=re(e),i=U("Layout","-layout-header",Po,ge,e,t),a=x(()=>{const{common:{cubicBezierEaseInOut:v},self:m}=i.value,s={"--n-bezier":v};return e.inverted?(s["--n-color"]=m.headerColorInverted,s["--n-text-color"]=m.textColorInverted,s["--n-border-color"]=m.headerBorderColorInverted):(s["--n-color"]=m.headerColor,s["--n-text-color"]=m.textColor,s["--n-border-color"]=m.headerBorderColor),s}),l=r?ne("layout-header",x(()=>e.inverted?"a":"b"),a,e):void 0;return{mergedClsPrefix:t,cssVars:r?void 0:a,themeClass:l==null?void 0:l.themeClass,onRender:l==null?void 0:l.onRender}},render(){var e;const{mergedClsPrefix:t}=this;return(e=this.onRender)===null||e===void 0||e.call(this),d("div",{class:[`${t}-layout-header`,this.themeClass,this.position&&`${t}-layout-header--${this.position}-positioned`,this.bordered&&`${t}-layout-header--bordered`],style:this.cssVars},this.$slots)}}),Ao=u("layout-sider",`
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
 `)]),u("layout-toggle-bar",[I("&:hover",[c("top",{transform:"rotate(-12deg) scale(1.15) translateY(-2px)"}),c("bottom",{transform:"rotate(12deg) scale(1.15) translateY(2px)"})])])]),u("layout-toggle-button",`
 left: 0;
 transform: translateX(-50%) translateY(-50%);
 `,[u("base-icon",`
 transform: rotate(0);
 `)]),u("layout-toggle-bar",`
 left: -28px;
 transform: rotate(180deg);
 `,[I("&:hover",[c("top",{transform:"rotate(12deg) scale(1.15) translateY(-2px)"}),c("bottom",{transform:"rotate(-12deg) scale(1.15) translateY(2px)"})])])]),w("collapsed",[u("layout-toggle-bar",[I("&:hover",[c("top",{transform:"rotate(-12deg) scale(1.15) translateY(-2px)"}),c("bottom",{transform:"rotate(12deg) scale(1.15) translateY(2px)"})])]),u("layout-toggle-button",[u("base-icon",`
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
 `),I("&:hover",[c("top",{transform:"rotate(12deg) scale(1.15) translateY(-2px)"}),c("bottom",{transform:"rotate(-12deg) scale(1.15) translateY(2px)"})]),c("top, bottom",{backgroundColor:"var(--n-toggle-bar-color)"}),I("&:hover",[c("top, bottom",{backgroundColor:"var(--n-toggle-bar-color-hover)"})])]),c("border",`
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
 `)]),ko=O({props:{clsPrefix:{type:String,required:!0},onClick:Function},render(){const{clsPrefix:e}=this;return d("div",{onClick:this.onClick,class:`${e}-layout-toggle-bar`},d("div",{class:`${e}-layout-toggle-bar__top`}),d("div",{class:`${e}-layout-toggle-bar__bottom`}))}}),_o=O({name:"LayoutToggleButton",props:{clsPrefix:{type:String,required:!0},onClick:Function},render(){const{clsPrefix:e}=this;return d("div",{class:`${e}-layout-toggle-button`,onClick:this.onClick},d(_e,{clsPrefix:e},{default:()=>d(fo,null)}))}}),Ho={position:fe,bordered:Boolean,collapsedWidth:{type:Number,default:48},width:{type:[Number,String],default:272},contentClass:String,contentStyle:{type:[String,Object],default:""},collapseMode:{type:String,default:"transform"},collapsed:{type:Boolean,default:void 0},defaultCollapsed:Boolean,showCollapsedContent:{type:Boolean,default:!0},showTrigger:{type:[Boolean,String],default:!1},nativeScrollbar:{type:Boolean,default:!0},inverted:Boolean,scrollbarProps:Object,triggerClass:String,triggerStyle:[String,Object],collapsedTriggerClass:String,collapsedTriggerStyle:[String,Object],"onUpdate:collapsed":[Function,Array],onUpdateCollapsed:[Function,Array],onAfterEnter:Function,onAfterLeave:Function,onExpand:[Function,Array],onCollapse:[Function,Array],onScroll:Function},Bo=O({name:"LayoutSider",props:Object.assign(Object.assign({},U.props),Ho),setup(e){const t=D(Oe),r=E(null),i=E(null),a=E(e.defaultCollapsed),l=ve(te(e,"collapsed"),a),v=x(()=>de(l.value?e.collapsedWidth:e.width)),m=x(()=>e.collapseMode!=="transform"?{}:{minWidth:de(e.width)}),s=x(()=>t?t.siderPlacement:"left");function f(A,y){if(e.nativeScrollbar){const{value:z}=r;z&&(y===void 0?z.scrollTo(A):z.scrollTo(A,y))}else{const{value:z}=i;z&&z.scrollTo(A,y)}}function H(){const{"onUpdate:collapsed":A,onUpdateCollapsed:y,onExpand:z,onCollapse:V}=e,{value:M}=l;y&&F(y,!M),A&&F(A,!M),a.value=!M,M?z&&F(z):V&&F(V)}let k=0,p=0;const N=A=>{var y;const z=A.target;k=z.scrollLeft,p=z.scrollTop,(y=e.onScroll)===null||y===void 0||y.call(e,A)};ke(()=>{if(e.nativeScrollbar){const A=r.value;A&&(A.scrollTop=p,A.scrollLeft=k)}}),Y(Be,{collapsedRef:l,collapseModeRef:te(e,"collapseMode")});const{mergedClsPrefixRef:S,inlineThemeDisabled:b}=re(e),R=U("Layout","-layout-sider",Ao,ge,e,S);function P(A){var y,z;A.propertyName==="max-width"&&(l.value?(y=e.onAfterLeave)===null||y===void 0||y.call(e):(z=e.onAfterEnter)===null||z===void 0||z.call(e))}const W={scrollTo:f},K=x(()=>{const{common:{cubicBezierEaseInOut:A},self:y}=R.value,{siderToggleButtonColor:z,siderToggleButtonBorder:V,siderToggleBarColor:M,siderToggleBarColorHover:le}=y,B={"--n-bezier":A,"--n-toggle-button-color":z,"--n-toggle-button-border":V,"--n-toggle-bar-color":M,"--n-toggle-bar-color-hover":le};return e.inverted?(B["--n-color"]=y.siderColorInverted,B["--n-text-color"]=y.textColorInverted,B["--n-border-color"]=y.siderBorderColorInverted,B["--n-toggle-button-icon-color"]=y.siderToggleButtonIconColorInverted,B.__invertScrollbar=y.__invertScrollbar):(B["--n-color"]=y.siderColor,B["--n-text-color"]=y.textColor,B["--n-border-color"]=y.siderBorderColor,B["--n-toggle-button-icon-color"]=y.siderToggleButtonIconColor),B}),$=b?ne("layout-sider",x(()=>e.inverted?"a":"b"),K,e):void 0;return Object.assign({scrollableElRef:r,scrollbarInstRef:i,mergedClsPrefix:S,mergedTheme:R,styleMaxWidth:v,mergedCollapsed:l,scrollContainerStyle:m,siderPlacement:s,handleNativeElScroll:N,handleTransitionend:P,handleTriggerClick:H,inlineThemeDisabled:b,cssVars:K,themeClass:$==null?void 0:$.themeClass,onRender:$==null?void 0:$.onRender},W)},render(){var e;const{mergedClsPrefix:t,mergedCollapsed:r,showTrigger:i}=this;return(e=this.onRender)===null||e===void 0||e.call(this),d("aside",{class:[`${t}-layout-sider`,this.themeClass,`${t}-layout-sider--${this.position}-positioned`,`${t}-layout-sider--${this.siderPlacement}-placement`,this.bordered&&`${t}-layout-sider--bordered`,r&&`${t}-layout-sider--collapsed`,(!r||this.showCollapsedContent)&&`${t}-layout-sider--show-content`],onTransitionend:this.handleTransitionend,style:[this.inlineThemeDisabled?void 0:this.cssVars,{maxWidth:this.styleMaxWidth,width:de(this.width)}]},this.nativeScrollbar?d("div",{class:[`${t}-layout-sider-scroll-container`,this.contentClass],onScroll:this.handleNativeElScroll,style:[this.scrollContainerStyle,{overflow:"auto"},this.contentStyle],ref:"scrollableElRef"},this.$slots):d(Ae,Object.assign({},this.scrollbarProps,{onScroll:this.onScroll,ref:"scrollbarInstRef",style:this.scrollContainerStyle,contentStyle:this.contentStyle,contentClass:this.contentClass,theme:this.mergedTheme.peers.Scrollbar,themeOverrides:this.mergedTheme.peerOverrides.Scrollbar,builtinThemeOverrides:this.inverted&&this.cssVars.__invertScrollbar==="true"?{colorHover:"rgba(255, 255, 255, .4)",color:"rgba(255, 255, 255, .3)"}:void 0}),this.$slots),i?i==="bar"?d(ko,{clsPrefix:t,class:r?this.collapsedTriggerClass:this.triggerClass,style:r?this.collapsedTriggerStyle:this.triggerStyle,onClick:this.handleTriggerClick}):d(_o,{clsPrefix:t,class:r?this.collapsedTriggerClass:this.triggerClass,style:r?this.collapsedTriggerStyle:this.triggerStyle,onClick:this.handleTriggerClick}):null,this.bordered?d("div",{class:`${t}-layout-sider__border`}):null)}}),Q=J("n-menu"),$e=J("n-submenu"),be=J("n-menu-item-group"),Te=[I("&::before","background-color: var(--n-item-color-hover);"),c("arrow",`
 color: var(--n-arrow-color-hover);
 `),c("icon",`
 color: var(--n-item-icon-color-hover);
 `),u("menu-item-content-header",`
 color: var(--n-item-text-color-hover);
 `,[I("a",`
 color: var(--n-item-text-color-hover);
 `),c("extra",`
 color: var(--n-item-text-color-hover);
 `)])],Ne=[c("icon",`
 color: var(--n-item-icon-color-hover-horizontal);
 `),u("menu-item-content-header",`
 color: var(--n-item-text-color-hover-horizontal);
 `,[I("a",`
 color: var(--n-item-text-color-hover-horizontal);
 `),c("extra",`
 color: var(--n-item-text-color-hover-horizontal);
 `)])],Oo=I([u("menu",`
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
 `,[I("&::before","display: none;"),w("selected","border-bottom: 2px solid var(--n-border-color-horizontal)")]),u("menu-item-content",[w("selected",[c("icon","color: var(--n-item-icon-color-active-horizontal);"),u("menu-item-content-header",`
 color: var(--n-item-text-color-active-horizontal);
 `,[I("a","color: var(--n-item-text-color-active-horizontal);"),c("extra","color: var(--n-item-text-color-active-horizontal);")])]),w("child-active",`
 border-bottom: 2px solid var(--n-border-color-horizontal);
 `,[u("menu-item-content-header",`
 color: var(--n-item-text-color-child-active-horizontal);
 `,[I("a",`
 color: var(--n-item-text-color-child-active-horizontal);
 `),c("extra",`
 color: var(--n-item-text-color-child-active-horizontal);
 `)]),c("icon",`
 color: var(--n-item-icon-color-child-active-horizontal);
 `)]),X("disabled",[X("selected, child-active",[I("&:focus-within",Ne)]),w("selected",[G(null,[c("icon","color: var(--n-item-icon-color-active-hover-horizontal);"),u("menu-item-content-header",`
 color: var(--n-item-text-color-active-hover-horizontal);
 `,[I("a","color: var(--n-item-text-color-active-hover-horizontal);"),c("extra","color: var(--n-item-text-color-active-hover-horizontal);")])])]),w("child-active",[G(null,[c("icon","color: var(--n-item-icon-color-child-active-hover-horizontal);"),u("menu-item-content-header",`
 color: var(--n-item-text-color-child-active-hover-horizontal);
 `,[I("a","color: var(--n-item-text-color-child-active-hover-horizontal);"),c("extra","color: var(--n-item-text-color-child-active-hover-horizontal);")])])]),G("border-bottom: 2px solid var(--n-border-color-horizontal);",Ne)]),u("menu-item-content-header",[I("a","color: var(--n-item-text-color-horizontal);")])])]),X("responsive",[u("menu-item-content-header",`
 overflow: hidden;
 text-overflow: ellipsis;
 `)]),w("collapsed",[u("menu-item-content",[w("selected",[I("&::before",`
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
 `,[I("> *","z-index: 1;"),I("&::before",`
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
 `),w("collapsed",[c("arrow","transform: rotate(0);")]),w("selected",[I("&::before","background-color: var(--n-item-color-active);"),c("arrow","color: var(--n-arrow-color-active);"),c("icon","color: var(--n-item-icon-color-active);"),u("menu-item-content-header",`
 color: var(--n-item-text-color-active);
 `,[I("a","color: var(--n-item-text-color-active);"),c("extra","color: var(--n-item-text-color-active);")])]),w("child-active",[u("menu-item-content-header",`
 color: var(--n-item-text-color-child-active);
 `,[I("a",`
 color: var(--n-item-text-color-child-active);
 `),c("extra",`
 color: var(--n-item-text-color-child-active);
 `)]),c("arrow",`
 color: var(--n-arrow-color-child-active);
 `),c("icon",`
 color: var(--n-item-icon-color-child-active);
 `)]),X("disabled",[X("selected, child-active",[I("&:focus-within",Te)]),w("selected",[G(null,[c("arrow","color: var(--n-arrow-color-active-hover);"),c("icon","color: var(--n-item-icon-color-active-hover);"),u("menu-item-content-header",`
 color: var(--n-item-text-color-active-hover);
 `,[I("a","color: var(--n-item-text-color-active-hover);"),c("extra","color: var(--n-item-text-color-active-hover);")])])]),w("child-active",[G(null,[c("arrow","color: var(--n-arrow-color-child-active-hover);"),c("icon","color: var(--n-item-icon-color-child-active-hover);"),u("menu-item-content-header",`
 color: var(--n-item-text-color-child-active-hover);
 `,[I("a","color: var(--n-item-text-color-child-active-hover);"),c("extra","color: var(--n-item-text-color-child-active-hover);")])])]),w("selected",[G(null,[I("&::before","background-color: var(--n-item-color-active-hover);")])]),G(null,Te)]),c("icon",`
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
 `,[I("a",`
 outline: none;
 text-decoration: none;
 transition: color .3s var(--n-bezier);
 color: var(--n-item-text-color);
 `,[I("&::before",`
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
 `,[oo({duration:".2s"})])]),u("menu-item-group",[u("menu-item-group-title",`
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
 `)])]),u("menu-tooltip",[I("a",`
 color: inherit;
 text-decoration: none;
 `)]),u("menu-divider",`
 transition: background-color .3s var(--n-bezier);
 background-color: var(--n-divider-color);
 height: 1px;
 margin: 6px 18px;
 `)]);function G(e,t){return[w("hover",e,t),I("&:hover",e,t)]}const Le=O({name:"MenuOptionContent",props:{collapsed:Boolean,disabled:Boolean,title:[String,Function],icon:Function,extra:[String,Function],showArrow:Boolean,childActive:Boolean,hover:Boolean,paddingLeft:Number,selected:Boolean,maxIconSize:{type:Number,required:!0},activeIconSize:{type:Number,required:!0},iconMarginRight:{type:Number,required:!0},clsPrefix:{type:String,required:!0},onClick:Function,tmNode:{type:Object,required:!0},isEllipsisPlaceholder:Boolean},setup(e){const{props:t}=D(Q);return{menuProps:t,style:x(()=>{const{paddingLeft:r}=e;return{paddingLeft:r&&`${r}px`}}),iconStyle:x(()=>{const{maxIconSize:r,activeIconSize:i,iconMarginRight:a}=e;return{width:`${r}px`,height:`${r}px`,fontSize:`${i}px`,marginRight:`${a}px`}})}},render(){const{clsPrefix:e,tmNode:t,menuProps:{renderIcon:r,renderLabel:i,renderExtra:a,expandIcon:l}}=this,v=r?r(t.rawNode):q(this.icon);return d("div",{onClick:m=>{var s;(s=this.onClick)===null||s===void 0||s.call(this,m)},role:"none",class:[`${e}-menu-item-content`,{[`${e}-menu-item-content--selected`]:this.selected,[`${e}-menu-item-content--collapsed`]:this.collapsed,[`${e}-menu-item-content--child-active`]:this.childActive,[`${e}-menu-item-content--disabled`]:this.disabled,[`${e}-menu-item-content--hover`]:this.hover}],style:this.style},v&&d("div",{class:`${e}-menu-item-content__icon`,style:this.iconStyle,role:"none"},[v]),d("div",{class:`${e}-menu-item-content-header`,role:"none"},this.isEllipsisPlaceholder?this.title:i?i(t.rawNode):q(this.title),this.extra||a?d("span",{class:`${e}-menu-item-content-header__extra`}," ",a?a(t.rawNode):q(this.extra)):null),this.showArrow?d(_e,{ariaHidden:!0,class:`${e}-menu-item-content__arrow`,clsPrefix:e},{default:()=>l?l(t.rawNode):d(zo,null)}):null)}}),oe=8;function xe(e){const t=D(Q),{props:r,mergedCollapsedRef:i}=t,a=D($e,null),l=D(be,null),v=x(()=>r.mode==="horizontal"),m=x(()=>v.value?r.dropdownPlacement:"tmNodes"in e?"right-start":"right"),s=x(()=>{var p;return Math.max((p=r.collapsedIconSize)!==null&&p!==void 0?p:r.iconSize,r.iconSize)}),f=x(()=>{var p;return!v.value&&e.root&&i.value&&(p=r.collapsedIconSize)!==null&&p!==void 0?p:r.iconSize}),H=x(()=>{if(v.value)return;const{collapsedWidth:p,indent:N,rootIndent:S}=r,{root:b,isGroup:R}=e,P=S===void 0?N:S;return b?i.value?p/2-s.value/2:P:l&&typeof l.paddingLeftRef.value=="number"?N/2+l.paddingLeftRef.value:a&&typeof a.paddingLeftRef.value=="number"?(R?N/2:N)+a.paddingLeftRef.value:0}),k=x(()=>{const{collapsedWidth:p,indent:N,rootIndent:S}=r,{value:b}=s,{root:R}=e;return v.value||!R||!i.value?oe:(S===void 0?N:S)+b+oe-(p+b)/2});return{dropdownPlacement:m,activeIconSize:f,maxIconSize:s,paddingLeft:H,iconMarginRight:k,NMenu:t,NSubmenu:a,NMenuOptionGroup:l}}const Ce={internalKey:{type:[String,Number],required:!0},root:Boolean,isGroup:Boolean,level:{type:Number,required:!0},title:[String,Function],extra:[String,Function]},Eo=O({name:"MenuDivider",setup(){const e=D(Q),{mergedClsPrefixRef:t,isHorizontalRef:r}=e;return()=>r.value?null:d("div",{class:`${t.value}-menu-divider`})}}),Fe=Object.assign(Object.assign({},Ce),{tmNode:{type:Object,required:!0},disabled:Boolean,icon:Function,onClick:Function}),$o=pe(Fe),Lo=O({name:"MenuOption",props:Fe,setup(e){const t=xe(e),{NSubmenu:r,NMenu:i,NMenuOptionGroup:a}=t,{props:l,mergedClsPrefixRef:v,mergedCollapsedRef:m}=i,s=r?r.mergedDisabledRef:a?a.mergedDisabledRef:{value:!1},f=x(()=>s.value||e.disabled);function H(p){const{onClick:N}=e;N&&N(p)}function k(p){f.value||(i.doSelect(e.internalKey,e.tmNode.rawNode),H(p))}return{mergedClsPrefix:v,dropdownPlacement:t.dropdownPlacement,paddingLeft:t.paddingLeft,iconMarginRight:t.iconMarginRight,maxIconSize:t.maxIconSize,activeIconSize:t.activeIconSize,mergedTheme:i.mergedThemeRef,menuProps:l,dropdownEnabled:ue(()=>e.root&&m.value&&l.mode!=="horizontal"&&!f.value),selected:ue(()=>i.mergedValueRef.value===e.internalKey),mergedDisabled:f,handleClick:k}},render(){const{mergedClsPrefix:e,mergedTheme:t,tmNode:r,menuProps:{renderLabel:i,nodeProps:a}}=this,l=a==null?void 0:a(r.rawNode);return d("div",Object.assign({},l,{role:"menuitem",class:[`${e}-menu-item`,l==null?void 0:l.class]}),d(bo,{theme:t.peers.Tooltip,themeOverrides:t.peerOverrides.Tooltip,trigger:"hover",placement:this.dropdownPlacement,disabled:!this.dropdownEnabled||this.title===void 0,internalExtraClass:["menu-tooltip"]},{default:()=>i?i(r.rawNode):q(this.title),trigger:()=>d(Le,{tmNode:r,clsPrefix:e,paddingLeft:this.paddingLeft,iconMarginRight:this.iconMarginRight,maxIconSize:this.maxIconSize,activeIconSize:this.activeIconSize,selected:this.selected,title:this.title,extra:this.extra,disabled:this.mergedDisabled,icon:this.icon,onClick:this.handleClick})}))}}),Me=Object.assign(Object.assign({},Ce),{tmNode:{type:Object,required:!0},tmNodes:{type:Array,required:!0}}),Fo=pe(Me),Mo=O({name:"MenuOptionGroup",props:Me,setup(e){const t=xe(e),{NSubmenu:r}=t,i=x(()=>r!=null&&r.mergedDisabledRef.value?!0:e.tmNode.disabled);Y(be,{paddingLeftRef:t.paddingLeft,mergedDisabledRef:i});const{mergedClsPrefixRef:a,props:l}=D(Q);return function(){const{value:v}=a,m=t.paddingLeft.value,{nodeProps:s}=l,f=s==null?void 0:s(e.tmNode.rawNode);return d("div",{class:`${v}-menu-item-group`,role:"group"},d("div",Object.assign({},f,{class:[`${v}-menu-item-group-title`,f==null?void 0:f.class],style:[(f==null?void 0:f.style)||"",m!==void 0?`padding-left: ${m}px;`:""]}),q(e.title),e.extra?d(to,null," ",q(e.extra)):null),d("div",null,e.tmNodes.map(H=>ye(H,l))))}}});function he(e){return e.type==="divider"||e.type==="render"}function jo(e){return e.type==="divider"}function ye(e,t){const{rawNode:r}=e,{show:i}=r;if(i===!1)return null;if(he(r))return jo(r)?d(Eo,Object.assign({key:e.key},r.props)):null;const{labelField:a}=t,{key:l,level:v,isGroup:m}=e,s=Object.assign(Object.assign({},r),{title:r.title||r[a],extra:r.titleExtra||r.extra,key:l,internalKey:l,level:v,root:v===0,isGroup:m});return e.children?e.isGroup?d(Mo,ie(s,Fo,{tmNode:e,tmNodes:e.children,key:l})):d(me,ie(s,Ko,{key:l,rawNodes:r[t.childrenField],tmNodes:e.children,tmNode:e})):d(Lo,ie(s,$o,{key:l,tmNode:e}))}const je=Object.assign(Object.assign({},Ce),{rawNodes:{type:Array,default:()=>[]},tmNodes:{type:Array,default:()=>[]},tmNode:{type:Object,required:!0},disabled:Boolean,icon:Function,onClick:Function,domId:String,virtualChildActive:{type:Boolean,default:void 0},isEllipsisPlaceholder:Boolean}),Ko=pe(je),me=O({name:"Submenu",props:je,setup(e){const t=xe(e),{NMenu:r,NSubmenu:i}=t,{props:a,mergedCollapsedRef:l,mergedThemeRef:v}=r,m=x(()=>{const{disabled:p}=e;return i!=null&&i.mergedDisabledRef.value||a.disabled?!0:p}),s=E(!1);Y($e,{paddingLeftRef:t.paddingLeft,mergedDisabledRef:m}),Y(be,null);function f(){const{onClick:p}=e;p&&p()}function H(){m.value||(l.value||r.toggleExpand(e.internalKey),f())}function k(p){s.value=p}return{menuProps:a,mergedTheme:v,doSelect:r.doSelect,inverted:r.invertedRef,isHorizontal:r.isHorizontalRef,mergedClsPrefix:r.mergedClsPrefixRef,maxIconSize:t.maxIconSize,activeIconSize:t.activeIconSize,iconMarginRight:t.iconMarginRight,dropdownPlacement:t.dropdownPlacement,dropdownShow:s,paddingLeft:t.paddingLeft,mergedDisabled:m,mergedValue:r.mergedValueRef,childActive:ue(()=>{var p;return(p=e.virtualChildActive)!==null&&p!==void 0?p:r.activePathRef.value.includes(e.internalKey)}),collapsed:x(()=>a.mode==="horizontal"?!1:l.value?!0:!r.mergedExpandedKeysRef.value.includes(e.internalKey)),dropdownEnabled:x(()=>!m.value&&(a.mode==="horizontal"||l.value)),handlePopoverShowChange:k,handleClick:H}},render(){var e;const{mergedClsPrefix:t,menuProps:{renderIcon:r,renderLabel:i}}=this,a=()=>{const{isHorizontal:v,paddingLeft:m,collapsed:s,mergedDisabled:f,maxIconSize:H,activeIconSize:k,title:p,childActive:N,icon:S,handleClick:b,menuProps:{nodeProps:R},dropdownShow:P,iconMarginRight:W,tmNode:K,mergedClsPrefix:$,isEllipsisPlaceholder:A,extra:y}=this,z=R==null?void 0:R(K.rawNode);return d("div",Object.assign({},z,{class:[`${$}-menu-item`,z==null?void 0:z.class],role:"menuitem"}),d(Le,{tmNode:K,paddingLeft:m,collapsed:s,disabled:f,iconMarginRight:W,maxIconSize:H,activeIconSize:k,title:p,extra:y,showArrow:!v,childActive:N,clsPrefix:$,icon:S,hover:P,onClick:b,isEllipsisPlaceholder:A}))},l=()=>d(ro,null,{default:()=>{const{tmNodes:v,collapsed:m}=this;return m?null:d("div",{class:`${t}-submenu-children`,role:"menu"},v.map(s=>ye(s,this.menuProps)))}});return this.root?d(He,Object.assign({size:"large",trigger:"hover"},(e=this.menuProps)===null||e===void 0?void 0:e.dropdownProps,{themeOverrides:this.mergedTheme.peerOverrides.Dropdown,theme:this.mergedTheme.peers.Dropdown,builtinThemeOverrides:{fontSizeLarge:"14px",optionIconSizeLarge:"18px"},value:this.mergedValue,disabled:!this.dropdownEnabled,placement:this.dropdownPlacement,keyField:this.menuProps.keyField,labelField:this.menuProps.labelField,childrenField:this.menuProps.childrenField,onUpdateShow:this.handlePopoverShowChange,options:this.rawNodes,onSelect:this.doSelect,inverted:this.inverted,renderIcon:r,renderLabel:i}),{default:()=>d("div",{class:`${t}-submenu`,role:"menu","aria-expanded":!this.collapsed,id:this.domId},a(),this.isHorizontal?null:l())}):d("div",{class:`${t}-submenu`,role:"menu","aria-expanded":!this.collapsed,id:this.domId},a(),l())}}),Vo=Object.assign(Object.assign({},U.props),{options:{type:Array,default:()=>[]},collapsed:{type:Boolean,default:void 0},collapsedWidth:{type:Number,default:48},iconSize:{type:Number,default:20},collapsedIconSize:{type:Number,default:24},rootIndent:Number,indent:{type:Number,default:32},labelField:{type:String,default:"label"},keyField:{type:String,default:"key"},childrenField:{type:String,default:"children"},disabledField:{type:String,default:"disabled"},defaultExpandAll:Boolean,defaultExpandedKeys:Array,expandedKeys:Array,value:[String,Number],defaultValue:{type:[String,Number],default:null},mode:{type:String,default:"vertical"},watchProps:{type:Array,default:void 0},disabled:Boolean,show:{type:Boolean,default:!0},inverted:Boolean,"onUpdate:expandedKeys":[Function,Array],onUpdateExpandedKeys:[Function,Array],onUpdateValue:[Function,Array],"onUpdate:value":[Function,Array],expandIcon:Function,renderIcon:Function,renderLabel:Function,renderExtra:Function,dropdownProps:Object,accordion:Boolean,nodeProps:Function,dropdownPlacement:{type:String,default:"bottom"},responsive:Boolean,items:Array,onOpenNamesChange:[Function,Array],onSelect:[Function,Array],onExpandedNamesChange:[Function,Array],expandedNames:Array,defaultExpandedNames:Array}),Do=O({name:"Menu",inheritAttrs:!1,props:Vo,setup(e){const{mergedClsPrefixRef:t,inlineThemeDisabled:r}=re(e),i=U("Menu","-menu",Oo,ao,e,t),a=D(Be,null),l=x(()=>{var h;const{collapsed:C}=e;if(C!==void 0)return C;if(a){const{collapseModeRef:o,collapsedRef:g}=a;if(o.value==="width")return(h=g.value)!==null&&h!==void 0?h:!1}return!1}),v=x(()=>{const{keyField:h,childrenField:C,disabledField:o}=e;return ce(e.items||e.options,{getIgnored(g){return he(g)},getChildren(g){return g[C]},getDisabled(g){return g[o]},getKey(g){var T;return(T=g[h])!==null&&T!==void 0?T:g.name}})}),m=x(()=>new Set(v.value.treeNodes.map(h=>h.key))),{watchProps:s}=e,f=E(null);s!=null&&s.includes("defaultValue")?Se(()=>{f.value=e.defaultValue}):f.value=e.defaultValue;const H=te(e,"value"),k=ve(H,f),p=E([]),N=()=>{p.value=e.defaultExpandAll?v.value.getNonLeafKeys():e.defaultExpandedNames||e.defaultExpandedKeys||v.value.getPath(k.value,{includeSelf:!1}).keyPath};s!=null&&s.includes("defaultExpandedKeys")?Se(N):N();const S=yo(e,["expandedNames","expandedKeys"]),b=ve(S,p),R=x(()=>v.value.treeNodes),P=x(()=>v.value.getPath(k.value).keyPath);Y(Q,{props:e,mergedCollapsedRef:l,mergedThemeRef:i,mergedValueRef:k,mergedExpandedKeysRef:b,activePathRef:P,mergedClsPrefixRef:t,isHorizontalRef:x(()=>e.mode==="horizontal"),invertedRef:te(e,"inverted"),doSelect:W,toggleExpand:$});function W(h,C){const{"onUpdate:value":o,onUpdateValue:g,onSelect:T}=e;g&&F(g,h,C),o&&F(o,h,C),T&&F(T,h,C),f.value=h}function K(h){const{"onUpdate:expandedKeys":C,onUpdateExpandedKeys:o,onExpandedNamesChange:g,onOpenNamesChange:T}=e;C&&F(C,h),o&&F(o,h),g&&F(g,h),T&&F(T,h),p.value=h}function $(h){const C=Array.from(b.value),o=C.findIndex(g=>g===h);if(~o)C.splice(o,1);else{if(e.accordion&&m.value.has(h)){const g=C.findIndex(T=>m.value.has(T));g>-1&&C.splice(g,1)}C.push(h)}K(C)}const A=h=>{const C=v.value.getPath(h??k.value,{includeSelf:!1}).keyPath;if(!C.length)return;const o=Array.from(b.value),g=new Set([...o,...C]);e.accordion&&m.value.forEach(T=>{g.has(T)&&!C.includes(T)&&g.delete(T)}),K(Array.from(g))},y=x(()=>{const{inverted:h}=e,{common:{cubicBezierEaseInOut:C},self:o}=i.value,{borderRadius:g,borderColorHorizontal:T,fontSize:We,itemHeight:Xe,dividerColor:Ze}=o,n={"--n-divider-color":Ze,"--n-bezier":C,"--n-font-size":We,"--n-border-color-horizontal":T,"--n-border-radius":g,"--n-item-height":Xe};return h?(n["--n-group-text-color"]=o.groupTextColorInverted,n["--n-color"]=o.colorInverted,n["--n-item-text-color"]=o.itemTextColorInverted,n["--n-item-text-color-hover"]=o.itemTextColorHoverInverted,n["--n-item-text-color-active"]=o.itemTextColorActiveInverted,n["--n-item-text-color-child-active"]=o.itemTextColorChildActiveInverted,n["--n-item-text-color-child-active-hover"]=o.itemTextColorChildActiveInverted,n["--n-item-text-color-active-hover"]=o.itemTextColorActiveHoverInverted,n["--n-item-icon-color"]=o.itemIconColorInverted,n["--n-item-icon-color-hover"]=o.itemIconColorHoverInverted,n["--n-item-icon-color-active"]=o.itemIconColorActiveInverted,n["--n-item-icon-color-active-hover"]=o.itemIconColorActiveHoverInverted,n["--n-item-icon-color-child-active"]=o.itemIconColorChildActiveInverted,n["--n-item-icon-color-child-active-hover"]=o.itemIconColorChildActiveHoverInverted,n["--n-item-icon-color-collapsed"]=o.itemIconColorCollapsedInverted,n["--n-item-text-color-horizontal"]=o.itemTextColorHorizontalInverted,n["--n-item-text-color-hover-horizontal"]=o.itemTextColorHoverHorizontalInverted,n["--n-item-text-color-active-horizontal"]=o.itemTextColorActiveHorizontalInverted,n["--n-item-text-color-child-active-horizontal"]=o.itemTextColorChildActiveHorizontalInverted,n["--n-item-text-color-child-active-hover-horizontal"]=o.itemTextColorChildActiveHoverHorizontalInverted,n["--n-item-text-color-active-hover-horizontal"]=o.itemTextColorActiveHoverHorizontalInverted,n["--n-item-icon-color-horizontal"]=o.itemIconColorHorizontalInverted,n["--n-item-icon-color-hover-horizontal"]=o.itemIconColorHoverHorizontalInverted,n["--n-item-icon-color-active-horizontal"]=o.itemIconColorActiveHorizontalInverted,n["--n-item-icon-color-active-hover-horizontal"]=o.itemIconColorActiveHoverHorizontalInverted,n["--n-item-icon-color-child-active-horizontal"]=o.itemIconColorChildActiveHorizontalInverted,n["--n-item-icon-color-child-active-hover-horizontal"]=o.itemIconColorChildActiveHoverHorizontalInverted,n["--n-arrow-color"]=o.arrowColorInverted,n["--n-arrow-color-hover"]=o.arrowColorHoverInverted,n["--n-arrow-color-active"]=o.arrowColorActiveInverted,n["--n-arrow-color-active-hover"]=o.arrowColorActiveHoverInverted,n["--n-arrow-color-child-active"]=o.arrowColorChildActiveInverted,n["--n-arrow-color-child-active-hover"]=o.arrowColorChildActiveHoverInverted,n["--n-item-color-hover"]=o.itemColorHoverInverted,n["--n-item-color-active"]=o.itemColorActiveInverted,n["--n-item-color-active-hover"]=o.itemColorActiveHoverInverted,n["--n-item-color-active-collapsed"]=o.itemColorActiveCollapsedInverted):(n["--n-group-text-color"]=o.groupTextColor,n["--n-color"]=o.color,n["--n-item-text-color"]=o.itemTextColor,n["--n-item-text-color-hover"]=o.itemTextColorHover,n["--n-item-text-color-active"]=o.itemTextColorActive,n["--n-item-text-color-child-active"]=o.itemTextColorChildActive,n["--n-item-text-color-child-active-hover"]=o.itemTextColorChildActiveHover,n["--n-item-text-color-active-hover"]=o.itemTextColorActiveHover,n["--n-item-icon-color"]=o.itemIconColor,n["--n-item-icon-color-hover"]=o.itemIconColorHover,n["--n-item-icon-color-active"]=o.itemIconColorActive,n["--n-item-icon-color-active-hover"]=o.itemIconColorActiveHover,n["--n-item-icon-color-child-active"]=o.itemIconColorChildActive,n["--n-item-icon-color-child-active-hover"]=o.itemIconColorChildActiveHover,n["--n-item-icon-color-collapsed"]=o.itemIconColorCollapsed,n["--n-item-text-color-horizontal"]=o.itemTextColorHorizontal,n["--n-item-text-color-hover-horizontal"]=o.itemTextColorHoverHorizontal,n["--n-item-text-color-active-horizontal"]=o.itemTextColorActiveHorizontal,n["--n-item-text-color-child-active-horizontal"]=o.itemTextColorChildActiveHorizontal,n["--n-item-text-color-child-active-hover-horizontal"]=o.itemTextColorChildActiveHoverHorizontal,n["--n-item-text-color-active-hover-horizontal"]=o.itemTextColorActiveHoverHorizontal,n["--n-item-icon-color-horizontal"]=o.itemIconColorHorizontal,n["--n-item-icon-color-hover-horizontal"]=o.itemIconColorHoverHorizontal,n["--n-item-icon-color-active-horizontal"]=o.itemIconColorActiveHorizontal,n["--n-item-icon-color-active-hover-horizontal"]=o.itemIconColorActiveHoverHorizontal,n["--n-item-icon-color-child-active-horizontal"]=o.itemIconColorChildActiveHorizontal,n["--n-item-icon-color-child-active-hover-horizontal"]=o.itemIconColorChildActiveHoverHorizontal,n["--n-arrow-color"]=o.arrowColor,n["--n-arrow-color-hover"]=o.arrowColorHover,n["--n-arrow-color-active"]=o.arrowColorActive,n["--n-arrow-color-active-hover"]=o.arrowColorActiveHover,n["--n-arrow-color-child-active"]=o.arrowColorChildActive,n["--n-arrow-color-child-active-hover"]=o.arrowColorChildActiveHover,n["--n-item-color-hover"]=o.itemColorHover,n["--n-item-color-active"]=o.itemColorActive,n["--n-item-color-active-hover"]=o.itemColorActiveHover,n["--n-item-color-active-collapsed"]=o.itemColorActiveCollapsed),n}),z=r?ne("menu",x(()=>e.inverted?"a":"b"),y,e):void 0,V=lo(),M=E(null),le=E(null);let B=!0;const ze=()=>{var h;B?B=!1:(h=M.value)===null||h===void 0||h.sync({showAllItemsBeforeCalculate:!0})};function Ke(){return document.getElementById(V)}const ee=E(-1);function Ve(h){ee.value=e.options.length-h}function De(h){h||(ee.value=-1)}const Ue=x(()=>{const h=ee.value;return{children:h===-1?[]:e.options.slice(h)}}),Ge=x(()=>{const{childrenField:h,disabledField:C,keyField:o}=e;return ce([Ue.value],{getIgnored(g){return he(g)},getChildren(g){return g[h]},getDisabled(g){return g[C]},getKey(g){var T;return(T=g[o])!==null&&T!==void 0?T:g.name}})}),qe=x(()=>ce([{}]).treeNodes[0]);function Ye(){var h;if(ee.value===-1)return d(me,{root:!0,level:0,key:"__ellpisisGroupPlaceholder__",internalKey:"__ellpisisGroupPlaceholder__",title:"···",tmNode:qe.value,domId:V,isEllipsisPlaceholder:!0});const C=Ge.value.treeNodes[0],o=P.value,g=!!(!((h=C.children)===null||h===void 0)&&h.some(T=>o.includes(T.key)));return d(me,{level:0,root:!0,key:"__ellpisisGroup__",internalKey:"__ellpisisGroup__",title:"···",virtualChildActive:g,tmNode:C,domId:V,rawNodes:C.rawNode.children||[],tmNodes:C.children||[],isEllipsisPlaceholder:!0})}return{mergedClsPrefix:t,controlledExpandedKeys:S,uncontrolledExpanededKeys:p,mergedExpandedKeys:b,uncontrolledValue:f,mergedValue:k,activePath:P,tmNodes:R,mergedTheme:i,mergedCollapsed:l,cssVars:r?void 0:y,themeClass:z==null?void 0:z.themeClass,overflowRef:M,counterRef:le,updateCounter:()=>{},onResize:ze,onUpdateOverflow:De,onUpdateCount:Ve,renderCounter:Ye,getCounter:Ke,onRender:z==null?void 0:z.onRender,showOption:A,deriveResponsiveState:ze}},render(){const{mergedClsPrefix:e,mode:t,themeClass:r,onRender:i}=this;i==null||i();const a=()=>this.tmNodes.map(s=>ye(s,this.$props)),v=t==="horizontal"&&this.responsive,m=()=>d("div",io(this.$attrs,{role:t==="horizontal"?"menubar":"menu",class:[`${e}-menu`,r,`${e}-menu--${t}`,v&&`${e}-menu--responsive`,this.mergedCollapsed&&`${e}-menu--collapsed`],style:this.cssVars}),v?d(xo,{ref:"overflowRef",onUpdateOverflow:this.onUpdateOverflow,getCounter:this.getCounter,onUpdateCount:this.onUpdateCount,updateCounter:this.updateCounter,style:{width:"100%",display:"flex",overflow:"hidden"}},{default:a,counter:this.renderCounter}):a());return v?d(no,{onResize:this.onResize},{default:m}):m()}}),Uo={style:{padding:"16px","text-align":"center","font-weight":"bold","font-size":"18px"}},Go={key:0},qo={key:1},Yo={style:{display:"flex","align-items":"center",gap:"12px"}},Wo={style:{display:"flex","align-items":"center",gap:"12px"}},ot=O({__name:"AppShell",setup(e){const t=po(),r=go(),i=so(),a=co(),{t:l}=uo(),v=E(i.locale),m=x(()=>r.path.split("/")[1]||"dashboard"),s=x(()=>[{label:l("nav.dashboard"),key:"dashboard"},{label:l("nav.readiness"),key:"readiness"},{label:l("nav.inbounds"),key:"inbounds"},{label:l("nav.outbounds"),key:"outbounds"},{label:l("nav.users"),key:"users"},{label:l("nav.subscriptions"),key:"subscriptions"},{label:l("nav.nodePool"),key:"node-pool"},{label:l("nav.routing"),key:"routing"},{label:l("nav.dns"),key:"dns"},{label:l("nav.monitor"),key:"monitor"},{label:l("nav.settings"),key:"settings"},{label:l("nav.config"),key:"config"}]),f=[{label:"中文",value:"zh-CN"},{label:"English",value:"en"}],H=[{label:l("auth.logout"),key:"logout"}];function k(S){t.push("/"+S)}function p(S){i.setLocale(S),window.location.reload()}function N(S){S==="logout"&&a.logout()}return(S,b)=>{const R=mo("router-view");return ae(),vo(_(Pe),{"has-sider":"",style:{height:"100vh"}},{default:j(()=>[L(_(Bo),{bordered:"",collapsed:_(i).sidebarCollapsed,"collapse-mode":"width","collapsed-width":64,width:220,"show-trigger":"",onCollapse:b[0]||(b[0]=P=>_(i).sidebarCollapsed=!0),onExpand:b[1]||(b[1]=P=>_(i).sidebarCollapsed=!1),"native-scrollbar":!1,style:{height:"100vh"}},{default:j(()=>[Z("div",Uo,[_(i).sidebarCollapsed?(ae(),we("span",qo,"X")):(ae(),we("span",Go,"Xray Panel"))]),L(_(Do),{collapsed:_(i).sidebarCollapsed,"collapsed-width":64,"collapsed-icon-size":22,options:s.value,value:m.value,"onUpdate:value":k},null,8,["collapsed","options","value"])]),_:1},8,["collapsed"]),L(_(Pe),null,{default:j(()=>[L(_(No),{bordered:"",style:{height:"56px",padding:"0 24px",display:"flex","align-items":"center","justify-content":"space-between"}},{default:j(()=>[Z("div",Yo,[L(_(se),{quaternary:"",circle:"",size:"small",onClick:_(i).toggleSidebar,class:"mobile-menu"},{icon:j(()=>[...b[3]||(b[3]=[Z("span",{style:{"font-size":"18px"}},"☰",-1)])]),_:1},8,["onClick"])]),Z("div",Wo,[L(_(Co),{value:v.value,"onUpdate:value":[b[2]||(b[2]=P=>v.value=P),p],options:f,size:"small",style:{width:"100px"}},null,8,["value"]),L(_(se),{quaternary:"",circle:"",size:"small",onClick:_(i).toggleTheme},{icon:j(()=>[Z("span",null,Re(_(i).isDark?"☀":"☾"),1)]),_:1},8,["onClick"]),L(_(He),{options:H,onSelect:N},{default:j(()=>[L(_(se),{quaternary:"",size:"small"},{default:j(()=>[ho(Re(_(a).username),1)]),_:1})]),_:1})])]),_:1}),L(_(Ro),{"content-style":"padding: 24px","native-scrollbar":!1},{default:j(()=>[L(R)]),_:1})]),_:1})]),_:1})}}});export{ot as default};
