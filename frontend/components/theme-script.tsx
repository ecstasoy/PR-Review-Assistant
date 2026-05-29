// ThemeScript 是个 SSR-only 组件：在 <head> 内插同步内联脚本，
// 让浏览器**在 React 水合前**就按 localStorage / prefers-color-scheme 设置 data-theme，
// 避免"先白后黑"的 FOUC。
//
// 注意：脚本内不能引外部模块；只能用全局 API（localStorage / matchMedia）。

import { THEME_STORAGE_KEY } from "@/lib/theme";

export function ThemeScript() {
  const code = `(function(){try{var s=localStorage.getItem(${JSON.stringify(THEME_STORAGE_KEY)});var t=s==="dark"||s==="light"?s:(matchMedia("(prefers-color-scheme: dark)").matches?"dark":"light");document.documentElement.setAttribute("data-theme",t);}catch(e){document.documentElement.setAttribute("data-theme","light");}})();`;
  return <script dangerouslySetInnerHTML={{ __html: code }} />;
}
