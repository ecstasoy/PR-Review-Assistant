// 语法高亮：用 highlight.js 覆盖 190+ 语言；按文件后缀给 lang 提示，
// hljs 返回带 hljs-* class 的 HTML 字符串，CSS 把 class 映射到 globals.css 的 --tok-* token 色。
// 注意：per-line 高亮不识别跨行结构（多行 string / 块注释），diff 场景可接受。
//
// hljs.value 已 HTML-escape，dangerouslySetInnerHTML 用法安全。

import hljs from "highlight.js";

// langFromPath 文件后缀 → hljs 语言名映射；未列出的返 ""（不高亮，回退裸文本 escape）
export function langFromPath(path: string): string {
  const lower = path.toLowerCase();
  // 特例：无后缀文件名
  if (lower.endsWith("/dockerfile") || lower === "dockerfile") return "dockerfile";
  if (lower.endsWith("/makefile") || lower === "makefile") return "makefile";
  if (lower.endsWith("/cmakelists.txt") || lower === "cmakelists.txt") return "cmake";

  const ext = lower.split(".").pop() ?? "";
  switch (ext) {
    case "go":
      return "go";
    case "ts":
    case "tsx":
    case "mts":
    case "cts":
      return "typescript";
    case "js":
    case "jsx":
    case "mjs":
    case "cjs":
      return "javascript";
    case "py":
    case "pyi":
    case "pyw":
      return "python";
    case "rs":
      return "rust";
    case "java":
      return "java";
    case "kt":
    case "kts":
      return "kotlin";
    case "swift":
      return "swift";
    case "rb":
    case "rake":
      return "ruby";
    case "php":
    case "phtml":
      return "php";
    case "cs":
      return "csharp";
    case "scala":
    case "sc":
      return "scala";
    case "c":
    case "h":
      return "c";
    case "cpp":
    case "cc":
    case "cxx":
    case "hpp":
    case "hh":
    case "hxx":
      return "cpp";
    case "m":
    case "mm":
      return "objectivec";
    case "sh":
    case "bash":
    case "zsh":
    case "ksh":
      return "bash";
    case "fish":
      return "shell";
    case "yml":
    case "yaml":
      return "yaml";
    case "toml":
      return "toml";
    case "ini":
    case "cfg":
    case "conf":
      return "ini";
    case "json":
    case "jsonc":
      return "json";
    case "sql":
      return "sql";
    case "html":
    case "htm":
      return "xml";
    case "xml":
    case "svg":
    case "xsd":
    case "wsdl":
      return "xml";
    case "css":
      return "css";
    case "scss":
    case "sass":
      return "scss";
    case "less":
      return "less";
    case "dart":
      return "dart";
    case "lua":
      return "lua";
    case "r":
      return "r";
    case "ex":
    case "exs":
      return "elixir";
    case "erl":
    case "hrl":
      return "erlang";
    case "hs":
    case "lhs":
      return "haskell";
    case "clj":
    case "cljs":
    case "cljc":
    case "edn":
      return "clojure";
    case "ml":
    case "mli":
      return "ocaml";
    case "fs":
    case "fsi":
    case "fsx":
      return "fsharp";
    case "elm":
      return "elm";
    case "nim":
    case "nims":
      return "nim";
    case "zig":
      return "zig";
    case "vim":
      return "vim";
    case "tf":
    case "tfvars":
    case "hcl":
      return "hcl";
    case "graphql":
    case "gql":
      return "graphql";
    case "proto":
      return "protobuf";
    case "groovy":
    case "gradle":
      return "groovy";
    case "perl":
    case "pl":
    case "pm":
      return "perl";
    case "ps1":
    case "psm1":
    case "psd1":
      return "powershell";
    case "md":
    case "markdown":
    case "mdx":
      return "markdown";
    case "diff":
    case "patch":
      return "diff";
    case "tex":
    case "ltx":
      return "latex";
    case "vue":
      return "xml";
    default:
      return "";
  }
}

// highlightHTML 返回 hljs 高亮后的 HTML 字符串。
// langHint 未命中或为空时返回 HTML-escape 的裸文本（避免每行调用 highlightAuto 的开销）。
export function highlightHTML(code: string, langHint?: string): string {
  if (!langHint || !hljs.getLanguage(langHint)) {
    return escapeHTML(code);
  }
  try {
    return hljs.highlight(code, { language: langHint, ignoreIllegals: true }).value;
  } catch {
    return escapeHTML(code);
  }
}

function escapeHTML(s: string): string {
  return s.replace(/[&<>"']/g, (c) =>
    c === "&"
      ? "&amp;"
      : c === "<"
        ? "&lt;"
        : c === ">"
          ? "&gt;"
          : c === '"'
            ? "&quot;"
            : "&#39;",
  );
}
