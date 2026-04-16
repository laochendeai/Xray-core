import{E as p,p as e,J as V,q as z,D as M,aJ as F,aK as G,d as E,aL as H,j as n,aM as J,v as K,x as L,z as q,A as O,aN as W,ac as T}from"./index-CxPJ5SeK.js";import{g as Q}from"./Space-Cl9Y7VVz.js";import{u as U}from"./use-compitable-DUgLe3MC.js";function B(t,h="default",d=[]){const{children:s}=t;if(s!==null&&typeof s=="object"&&!Array.isArray(s)){const i=s[h];if(typeof i=="function")return i()}return d}const X=p([e("descriptions",{fontSize:"var(--n-font-size)"},[e("descriptions-separator",`
 display: inline-block;
 margin: 0 8px 0 2px;
 `),e("descriptions-table-wrapper",[e("descriptions-table",[e("descriptions-table-row",[e("descriptions-table-header",{padding:"var(--n-th-padding)"}),e("descriptions-table-content",{padding:"var(--n-td-padding)"})])])]),V("bordered",[e("descriptions-table-wrapper",[e("descriptions-table",[e("descriptions-table-row",[p("&:last-child",[e("descriptions-table-content",{paddingBottom:0})])])])])]),z("left-label-placement",[e("descriptions-table-content",[p("> *",{verticalAlign:"top"})])]),z("left-label-align",[p("th",{textAlign:"left"})]),z("center-label-align",[p("th",{textAlign:"center"})]),z("right-label-align",[p("th",{textAlign:"right"})]),z("bordered",[e("descriptions-table-wrapper",`
 border-radius: var(--n-border-radius);
 overflow: hidden;
 background: var(--n-merged-td-color);
 border: 1px solid var(--n-merged-border-color);
 `,[e("descriptions-table",[e("descriptions-table-row",[p("&:not(:last-child)",[e("descriptions-table-content",{borderBottom:"1px solid var(--n-merged-border-color)"}),e("descriptions-table-header",{borderBottom:"1px solid var(--n-merged-border-color)"})]),e("descriptions-table-header",`
 font-weight: 400;
 background-clip: padding-box;
 background-color: var(--n-merged-th-color);
 `,[p("&:not(:last-child)",{borderRight:"1px solid var(--n-merged-border-color)"})]),e("descriptions-table-content",[p("&:not(:last-child)",{borderRight:"1px solid var(--n-merged-border-color)"})])])])])]),e("descriptions-header",`
 font-weight: var(--n-th-font-weight);
 font-size: 18px;
 transition: color .3s var(--n-bezier);
 line-height: var(--n-line-height);
 margin-bottom: 16px;
 color: var(--n-title-text-color);
 `),e("descriptions-table-wrapper",`
 transition:
 background-color .3s var(--n-bezier),
 border-color .3s var(--n-bezier);
 `,[e("descriptions-table",`
 width: 100%;
 border-collapse: separate;
 border-spacing: 0;
 box-sizing: border-box;
 `,[e("descriptions-table-row",`
 box-sizing: border-box;
 transition: border-color .3s var(--n-bezier);
 `,[e("descriptions-table-header",`
 font-weight: var(--n-th-font-weight);
 line-height: var(--n-line-height);
 display: table-cell;
 box-sizing: border-box;
 color: var(--n-th-text-color);
 transition:
 color .3s var(--n-bezier),
 background-color .3s var(--n-bezier),
 border-color .3s var(--n-bezier);
 `),e("descriptions-table-content",`
 vertical-align: top;
 line-height: var(--n-line-height);
 display: table-cell;
 box-sizing: border-box;
 color: var(--n-td-text-color);
 transition:
 color .3s var(--n-bezier),
 background-color .3s var(--n-bezier),
 border-color .3s var(--n-bezier);
 `,[M("content",`
 transition: color .3s var(--n-bezier);
 display: inline-block;
 color: var(--n-td-text-color);
 `)]),M("label",`
 font-weight: var(--n-th-font-weight);
 transition: color .3s var(--n-bezier);
 display: inline-block;
 margin-right: 14px;
 color: var(--n-th-text-color);
 `)])])])]),e("descriptions-table-wrapper",`
 --n-merged-th-color: var(--n-th-color);
 --n-merged-td-color: var(--n-td-color);
 --n-merged-border-color: var(--n-border-color);
 `),F(e("descriptions-table-wrapper",`
 --n-merged-th-color: var(--n-th-color-modal);
 --n-merged-td-color: var(--n-td-color-modal);
 --n-merged-border-color: var(--n-border-color-modal);
 `)),G(e("descriptions-table-wrapper",`
 --n-merged-th-color: var(--n-th-color-popover);
 --n-merged-td-color: var(--n-td-color-popover);
 --n-merged-border-color: var(--n-border-color-popover);
 `))]),N="DESCRIPTION_ITEM_FLAG";function Y(t){return typeof t=="object"&&t&&!Array.isArray(t)?t.type&&t.type[N]:!1}const Z=Object.assign(Object.assign({},L.props),{title:String,column:{type:Number,default:3},columns:Number,labelPlacement:{type:String,default:"top"},labelAlign:{type:String,default:"left"},separator:{type:String,default:":"},size:String,bordered:Boolean,labelClass:String,labelStyle:[Object,String],contentClass:String,contentStyle:[Object,String]}),ne=E({name:"Descriptions",props:Z,slots:Object,setup(t){const{mergedClsPrefixRef:h,inlineThemeDisabled:d,mergedComponentPropsRef:s}=K(t),i=O(()=>{var l,a;return t.size||((a=(l=s==null?void 0:s.value)===null||l===void 0?void 0:l.Descriptions)===null||a===void 0?void 0:a.size)||"medium"}),g=L("Descriptions","-descriptions",X,W,t,h),$=O(()=>{const{bordered:l}=t,a=i.value,{common:{cubicBezierEaseInOut:A},self:{titleTextColor:r,thColor:P,thColorModal:v,thColorPopover:R,thTextColor:I,thFontWeight:_,tdTextColor:j,tdColor:o,tdColorModal:f,tdColorPopover:D,borderColor:c,borderColorModal:m,borderColorPopover:y,borderRadius:w,lineHeight:u,[T("fontSize",a)]:S,[T(l?"thPaddingBordered":"thPadding",a)]:x,[T(l?"tdPaddingBordered":"tdPadding",a)]:C}}=g.value;return{"--n-title-text-color":r,"--n-th-padding":x,"--n-td-padding":C,"--n-font-size":S,"--n-bezier":A,"--n-th-font-weight":_,"--n-line-height":u,"--n-th-text-color":I,"--n-td-text-color":j,"--n-th-color":P,"--n-th-color-modal":v,"--n-th-color-popover":R,"--n-td-color":o,"--n-td-color-modal":f,"--n-td-color-popover":D,"--n-border-radius":w,"--n-border-color":c,"--n-border-color-modal":m,"--n-border-color-popover":y}}),b=d?q("descriptions",O(()=>{let l="";const{bordered:a}=t;return a&&(l+="a"),l+=i.value[0],l}),$,t):void 0;return{mergedClsPrefix:h,cssVars:d?void 0:$,themeClass:b==null?void 0:b.themeClass,onRender:b==null?void 0:b.onRender,compitableColumn:U(t,["columns","column"]),inlineThemeDisabled:d,mergedSize:i}},render(){const t=this.$slots.default,h=t?H(t()):[];h.length;const{contentClass:d,labelClass:s,compitableColumn:i,labelPlacement:g,labelAlign:$,mergedSize:b,bordered:l,title:a,cssVars:A,mergedClsPrefix:r,separator:P,onRender:v}=this;v==null||v();const R=h.filter(o=>Y(o)),I={span:0,row:[],secondRow:[],rows:[]},j=R.reduce((o,f,D)=>{const c=f.props||{},m=R.length-1===D,y=["label"in c?c.label:B(f,"label")],w=[B(f)],u=c.span||1,S=o.span;o.span+=u;const x=c.labelStyle||c["label-style"]||this.labelStyle,C=c.contentStyle||c["content-style"]||this.contentStyle;if(g==="left")l?o.row.push(n("th",{class:[`${r}-descriptions-table-header`,s],colspan:1,style:x},y),n("td",{class:[`${r}-descriptions-table-content`,d],colspan:m?(i-S)*2+1:u*2-1,style:C},w)):o.row.push(n("td",{class:`${r}-descriptions-table-content`,colspan:m?(i-S)*2:u*2},n("span",{class:[`${r}-descriptions-table-content__label`,s],style:x},[...y,P&&n("span",{class:`${r}-descriptions-separator`},P)]),n("span",{class:[`${r}-descriptions-table-content__content`,d],style:C},w)));else{const k=m?(i-S)*2:u*2;o.row.push(n("th",{class:[`${r}-descriptions-table-header`,s],colspan:k,style:x},y)),o.secondRow.push(n("td",{class:[`${r}-descriptions-table-content`,d],colspan:k,style:C},w))}return(o.span>=i||m)&&(o.span=0,o.row.length&&(o.rows.push(o.row),o.row=[]),g!=="left"&&o.secondRow.length&&(o.rows.push(o.secondRow),o.secondRow=[])),o},I).rows.map(o=>n("tr",{class:`${r}-descriptions-table-row`},o));return n("div",{style:A,class:[`${r}-descriptions`,this.themeClass,`${r}-descriptions--${g}-label-placement`,`${r}-descriptions--${$}-label-align`,`${r}-descriptions--${b}-size`,l&&`${r}-descriptions--bordered`]},a||this.$slots.header?n("div",{class:`${r}-descriptions-header`},a||Q(this,"header")):null,n("div",{class:`${r}-descriptions-table-wrapper`},n("table",{class:`${r}-descriptions-table`},n("tbody",null,g==="top"&&n("tr",{class:`${r}-descriptions-table-row`,style:{visibility:"collapse"}},J(i*2,n("td",null))),j))))}}),ee={label:String,span:{type:Number,default:1},labelClass:String,labelStyle:[Object,String],contentClass:String,contentStyle:[Object,String]},le=E({name:"DescriptionsItem",[N]:!0,props:ee,slots:Object,render(){return null}});export{ne as N,le as a};
