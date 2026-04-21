import{A as S,l as so,cD as to,bu as r,p as io,q as u,D as m,J as I,E as z,d as ho,ae as N,j as x,bh as go,v as bo,x as V,af as Co,z as vo,r as uo,H as fo,ac as h,bm as po,aB as U,C as ko,I as mo,n as xo}from"./index-Bs7Z848W.js";function $o(l,t){return S(()=>{for(const e of t)if(l[e]!==void 0)return l[e];return l[t[t.length-1]]})}function Po(l){const{textColor2:t,primaryColorHover:e,primaryColorPressed:f,primaryColor:a,infoColor:d,successColor:n,warningColor:s,errorColor:i,baseColor:p,borderColor:k,opacityDisabled:b,tagColor:B,closeIconColor:P,closeIconColorHover:v,closeIconColorPressed:o,borderRadiusSmall:c,fontSizeMini:C,fontSizeTiny:g,fontSizeSmall:H,fontSizeMedium:$,heightMini:R,heightTiny:M,heightSmall:T,heightMedium:E,closeColorHover:_,closeColorPressed:j,buttonColor2Hover:O,buttonColor2Pressed:W,fontWeightStrong:w}=l;return Object.assign(Object.assign({},to),{closeBorderRadius:c,heightTiny:R,heightSmall:M,heightMedium:T,heightLarge:E,borderRadius:c,opacityDisabled:b,fontSizeTiny:C,fontSizeSmall:g,fontSizeMedium:H,fontSizeLarge:$,fontWeightStrong:w,textColorCheckable:t,textColorHoverCheckable:t,textColorPressedCheckable:t,textColorChecked:p,colorCheckable:"#0000",colorHoverCheckable:O,colorPressedCheckable:W,colorChecked:a,colorCheckedHover:e,colorCheckedPressed:f,border:`1px solid ${k}`,textColor:t,color:B,colorBordered:"rgb(250, 250, 252)",closeIconColor:P,closeIconColorHover:v,closeIconColorPressed:o,closeColorHover:_,closeColorPressed:j,borderPrimary:`1px solid ${r(a,{alpha:.3})}`,textColorPrimary:a,colorPrimary:r(a,{alpha:.12}),colorBorderedPrimary:r(a,{alpha:.1}),closeIconColorPrimary:a,closeIconColorHoverPrimary:a,closeIconColorPressedPrimary:a,closeColorHoverPrimary:r(a,{alpha:.12}),closeColorPressedPrimary:r(a,{alpha:.18}),borderInfo:`1px solid ${r(d,{alpha:.3})}`,textColorInfo:d,colorInfo:r(d,{alpha:.12}),colorBorderedInfo:r(d,{alpha:.1}),closeIconColorInfo:d,closeIconColorHoverInfo:d,closeIconColorPressedInfo:d,closeColorHoverInfo:r(d,{alpha:.12}),closeColorPressedInfo:r(d,{alpha:.18}),borderSuccess:`1px solid ${r(n,{alpha:.3})}`,textColorSuccess:n,colorSuccess:r(n,{alpha:.12}),colorBorderedSuccess:r(n,{alpha:.1}),closeIconColorSuccess:n,closeIconColorHoverSuccess:n,closeIconColorPressedSuccess:n,closeColorHoverSuccess:r(n,{alpha:.12}),closeColorPressedSuccess:r(n,{alpha:.18}),borderWarning:`1px solid ${r(s,{alpha:.35})}`,textColorWarning:s,colorWarning:r(s,{alpha:.15}),colorBorderedWarning:r(s,{alpha:.12}),closeIconColorWarning:s,closeIconColorHoverWarning:s,closeIconColorPressedWarning:s,closeColorHoverWarning:r(s,{alpha:.12}),closeColorPressedWarning:r(s,{alpha:.18}),borderError:`1px solid ${r(i,{alpha:.23})}`,textColorError:i,colorError:r(i,{alpha:.1}),colorBorderedError:r(i,{alpha:.08}),closeIconColorError:i,closeIconColorHoverError:i,closeIconColorPressedError:i,closeColorHoverError:r(i,{alpha:.12}),closeColorPressedError:r(i,{alpha:.18})})}const yo={common:so,self:Po},Io={color:Object,type:{type:String,default:"default"},round:Boolean,size:String,closable:Boolean,disabled:{type:Boolean,default:void 0}},zo=io("tag",`
 --n-close-margin: var(--n-close-margin-top) var(--n-close-margin-right) var(--n-close-margin-bottom) var(--n-close-margin-left);
 white-space: nowrap;
 position: relative;
 box-sizing: border-box;
 cursor: default;
 display: inline-flex;
 align-items: center;
 flex-wrap: nowrap;
 padding: var(--n-padding);
 border-radius: var(--n-border-radius);
 color: var(--n-text-color);
 background-color: var(--n-color);
 transition: 
 border-color .3s var(--n-bezier),
 background-color .3s var(--n-bezier),
 color .3s var(--n-bezier),
 box-shadow .3s var(--n-bezier),
 opacity .3s var(--n-bezier);
 line-height: 1;
 height: var(--n-height);
 font-size: var(--n-font-size);
`,[u("strong",`
 font-weight: var(--n-font-weight-strong);
 `),m("border",`
 pointer-events: none;
 position: absolute;
 left: 0;
 right: 0;
 top: 0;
 bottom: 0;
 border-radius: inherit;
 border: var(--n-border);
 transition: border-color .3s var(--n-bezier);
 `),m("icon",`
 display: flex;
 margin: 0 4px 0 0;
 color: var(--n-text-color);
 transition: color .3s var(--n-bezier);
 font-size: var(--n-avatar-size-override);
 `),m("avatar",`
 display: flex;
 margin: 0 6px 0 0;
 `),m("close",`
 margin: var(--n-close-margin);
 transition:
 background-color .3s var(--n-bezier),
 color .3s var(--n-bezier);
 `),u("round",`
 padding: 0 calc(var(--n-height) / 3);
 border-radius: calc(var(--n-height) / 2);
 `,[m("icon",`
 margin: 0 4px 0 calc((var(--n-height) - 8px) / -2);
 `),m("avatar",`
 margin: 0 6px 0 calc((var(--n-height) - 8px) / -2);
 `),u("closable",`
 padding: 0 calc(var(--n-height) / 4) 0 calc(var(--n-height) / 3);
 `)]),u("icon, avatar",[u("round",`
 padding: 0 calc(var(--n-height) / 3) 0 calc(var(--n-height) / 2);
 `)]),u("disabled",`
 cursor: not-allowed !important;
 opacity: var(--n-opacity-disabled);
 `),u("checkable",`
 cursor: pointer;
 box-shadow: none;
 color: var(--n-text-color-checkable);
 background-color: var(--n-color-checkable);
 `,[I("disabled",[z("&:hover","background-color: var(--n-color-hover-checkable);",[I("checked","color: var(--n-text-color-hover-checkable);")]),z("&:active","background-color: var(--n-color-pressed-checkable);",[I("checked","color: var(--n-text-color-pressed-checkable);")])]),u("checked",`
 color: var(--n-text-color-checked);
 background-color: var(--n-color-checked);
 `,[I("disabled",[z("&:hover","background-color: var(--n-color-checked-hover);"),z("&:active","background-color: var(--n-color-checked-pressed);")])])])]),So=Object.assign(Object.assign(Object.assign({},V.props),Io),{bordered:{type:Boolean,default:void 0},checked:Boolean,checkable:Boolean,strong:Boolean,triggerClickOnClose:Boolean,onClose:[Array,Function],onMouseenter:Function,onMouseleave:Function,"onUpdate:checked":Function,onUpdateChecked:Function,internalCloseFocusable:{type:Boolean,default:!0},internalCloseIsButtonTag:{type:Boolean,default:!0},onCheckedChange:Function}),Bo=xo("n-tag"),Ro=ho({name:"Tag",props:So,slots:Object,setup(l){const t=uo(null),{mergedBorderedRef:e,mergedClsPrefixRef:f,inlineThemeDisabled:a,mergedRtlRef:d,mergedComponentPropsRef:n}=bo(l),s=S(()=>{var o,c;return l.size||((c=(o=n==null?void 0:n.value)===null||o===void 0?void 0:o.Tag)===null||c===void 0?void 0:c.size)||"medium"}),i=V("Tag","-tag",zo,yo,l,f);ko(Bo,{roundRef:mo(l,"round")});function p(){if(!l.disabled&&l.checkable){const{checked:o,onCheckedChange:c,onUpdateChecked:C,"onUpdate:checked":g}=l;C&&C(!o),g&&g(!o),c&&c(!o)}}function k(o){if(l.triggerClickOnClose||o.stopPropagation(),!l.disabled){const{onClose:c}=l;c&&fo(c,o)}}const b={setTextContent(o){const{value:c}=t;c&&(c.textContent=o)}},B=Co("Tag",d,f),P=S(()=>{const{type:o,color:{color:c,textColor:C}={}}=l,g=s.value,{common:{cubicBezierEaseInOut:H},self:{padding:$,closeMargin:R,borderRadius:M,opacityDisabled:T,textColorCheckable:E,textColorHoverCheckable:_,textColorPressedCheckable:j,textColorChecked:O,colorCheckable:W,colorHoverCheckable:w,colorPressedCheckable:K,colorChecked:L,colorCheckedHover:A,colorCheckedPressed:q,closeBorderRadius:J,fontWeightStrong:G,[h("colorBordered",o)]:Q,[h("closeSize",g)]:X,[h("closeIconSize",g)]:Y,[h("fontSize",g)]:Z,[h("height",g)]:F,[h("color",o)]:oo,[h("textColor",o)]:eo,[h("border",o)]:ro,[h("closeIconColor",o)]:D,[h("closeIconColorHover",o)]:lo,[h("closeIconColorPressed",o)]:co,[h("closeColorHover",o)]:ao,[h("closeColorPressed",o)]:no}}=i.value,y=po(R);return{"--n-font-weight-strong":G,"--n-avatar-size-override":`calc(${F} - 8px)`,"--n-bezier":H,"--n-border-radius":M,"--n-border":ro,"--n-close-icon-size":Y,"--n-close-color-pressed":no,"--n-close-color-hover":ao,"--n-close-border-radius":J,"--n-close-icon-color":D,"--n-close-icon-color-hover":lo,"--n-close-icon-color-pressed":co,"--n-close-icon-color-disabled":D,"--n-close-margin-top":y.top,"--n-close-margin-right":y.right,"--n-close-margin-bottom":y.bottom,"--n-close-margin-left":y.left,"--n-close-size":X,"--n-color":c||(e.value?Q:oo),"--n-color-checkable":W,"--n-color-checked":L,"--n-color-checked-hover":A,"--n-color-checked-pressed":q,"--n-color-hover-checkable":w,"--n-color-pressed-checkable":K,"--n-font-size":Z,"--n-height":F,"--n-opacity-disabled":T,"--n-padding":$,"--n-text-color":C||eo,"--n-text-color-checkable":E,"--n-text-color-checked":O,"--n-text-color-hover-checkable":_,"--n-text-color-pressed-checkable":j}}),v=a?vo("tag",S(()=>{let o="";const{type:c,color:{color:C,textColor:g}={}}=l;return o+=c[0],o+=s.value[0],C&&(o+=`a${U(C)}`),g&&(o+=`b${U(g)}`),e.value&&(o+="c"),o}),P,l):void 0;return Object.assign(Object.assign({},b),{rtlEnabled:B,mergedClsPrefix:f,contentRef:t,mergedBordered:e,handleClick:p,handleCloseClick:k,cssVars:a?void 0:P,themeClass:v==null?void 0:v.themeClass,onRender:v==null?void 0:v.onRender})},render(){var l,t;const{mergedClsPrefix:e,rtlEnabled:f,closable:a,color:{borderColor:d}={},round:n,onRender:s,$slots:i}=this;s==null||s();const p=N(i.avatar,b=>b&&x("div",{class:`${e}-tag__avatar`},b)),k=N(i.icon,b=>b&&x("div",{class:`${e}-tag__icon`},b));return x("div",{class:[`${e}-tag`,this.themeClass,{[`${e}-tag--rtl`]:f,[`${e}-tag--strong`]:this.strong,[`${e}-tag--disabled`]:this.disabled,[`${e}-tag--checkable`]:this.checkable,[`${e}-tag--checked`]:this.checkable&&this.checked,[`${e}-tag--round`]:n,[`${e}-tag--avatar`]:p,[`${e}-tag--icon`]:k,[`${e}-tag--closable`]:a}],style:this.cssVars,onClick:this.handleClick,onMouseenter:this.onMouseenter,onMouseleave:this.onMouseleave},k||p,x("span",{class:`${e}-tag__content`,ref:"contentRef"},(t=(l=this.$slots).default)===null||t===void 0?void 0:t.call(l)),!this.checkable&&a?x(go,{clsPrefix:e,class:`${e}-tag__close`,disabled:this.disabled,onClick:this.handleCloseClick,focusable:this.internalCloseFocusable,round:n,isButtonTag:this.internalCloseIsButtonTag,absolute:!0}):null,!this.checkable&&this.mergedBordered?x("div",{class:`${e}-tag__border`,style:{borderColor:d}}):null)}});export{Ro as N,$o as u};
