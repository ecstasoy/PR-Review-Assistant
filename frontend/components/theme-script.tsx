// ThemeScript 是个 SSR-only 组件：在 <head> 内插同步内联脚本，
// 让浏览器**在 React 水合前**就按 localStorage / prefers-color-scheme 设置 data-theme，
// 避免"先白后黑"的 FOUC。
//
// 注意：脚本内不能引外部模块；只能用全局 API（localStorage / matchMedia）。

export function ThemeScript() {
  const code = `(function(){var t;try{var s=localStorage.getItem("pr-review-theme");if(s==="dark"||s==="light")t=s;}catch(e){}if(!t){try{t=matchMedia("(prefers-color-scheme: dark)").matches?"dark":"light";}catch(e){t="light";}}document.documentElement.setAttribute("data-theme",t);})();`;
  return <script dangerouslySetInnerHTML={{ __html: code }} />;
}
