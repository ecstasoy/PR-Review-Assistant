// 轻量语法高亮：按 design 原型 Diff.jsx highlightCode 思路重做，但扩多语言。
// token 颜色靠 globals.css 的 --tok-kw / --tok-str / --tok-num / --tok-com（@theme 已暴露成 text-tok-* utility）。
// 不依赖 Prism/Shiki —— v1 信息密度优先，符号/操作符不高亮；评审场景行级阅读够用。

export type TokenKind = "kw" | "str" | "num" | "com" | "text";

export interface Token {
  text: string;
  kind: TokenKind;
}

// 按 lang 维护关键字集；空集合表示该语言不识别关键字（仍会做 string/number/comment 高亮）
const KEYWORDS: Record<string, Set<string>> = {
  go: new Set(
    "func return for if else range break continue var const type struct interface map chan go defer package import switch case default nil true false int int64 int32 int16 int8 uint uint64 uint32 uint16 uint8 uintptr string bool byte rune any float64 float32 error atomic make new len cap append copy delete close panic recover".split(
      " ",
    ),
  ),
  ts: new Set(
    "function return for if else const let var while do switch case break continue type interface class extends implements abstract import export default async await new this super throw try catch finally null undefined true false void never any unknown public private protected readonly static enum as in of instanceof typeof keyof".split(
      " ",
    ),
  ),
  js: new Set(
    "function return for if else const let var while do switch case break continue class extends import export default async await new this super throw try catch finally null undefined true false void in of instanceof typeof".split(
      " ",
    ),
  ),
  py: new Set(
    "def return for if elif else while break continue class import from as lambda True False None async await with yield raise try except finally pass not and or in is global nonlocal".split(
      " ",
    ),
  ),
  rs: new Set(
    "fn return for if else while loop match break continue let mut const struct enum impl trait pub use mod where async await dyn move ref self Self super crate as in true false unsafe extern type box".split(
      " ",
    ),
  ),
};

// langFromPath 简单后缀 → lang 映射；未知返 ""（仍做 string/number/comment 高亮）
export function langFromPath(path: string): string {
  const ext = path.split(".").pop()?.toLowerCase() ?? "";
  switch (ext) {
    case "go":
      return "go";
    case "ts":
    case "tsx":
      return "ts";
    case "js":
    case "jsx":
    case "mjs":
    case "cjs":
      return "js";
    case "py":
      return "py";
    case "rs":
      return "rs";
    case "md":
    case "markdown":
      return "md";
    default:
      return "";
  }
}

// highlightCode 返回 token 数组；DiffRow 把每个 token 映射到 span class
export function highlightCode(text: string, lang: string): Token[] {
  if (lang === "md") return [{ text, kind: "text" }];

  const out: Token[] = [];

  // 1) 先切出整行注释（// ...）；要避开字符串内的 //（朴素：检查前面双引号配对奇偶）
  let code = text;
  let comment = "";
  const cIdx = text.indexOf("//");
  if (cIdx >= 0) {
    const before = text.slice(0, cIdx);
    const dqCount = (before.match(/"/g) ?? []).length;
    if (dqCount % 2 === 0) {
      code = text.slice(0, cIdx);
      comment = text.slice(cIdx);
    }
  }

  // 2) Python / Rust 整行 # 注释（rs 不用 #，但容忍）
  if (!comment && lang === "py") {
    const hIdx = text.indexOf("#");
    if (hIdx >= 0) {
      const before = text.slice(0, hIdx);
      const dqCount = (before.match(/["']/g) ?? []).length;
      if (dqCount % 2 === 0) {
        code = text.slice(0, hIdx);
        comment = text.slice(hIdx);
      }
    }
  }

  // 3) 分词：string("..." 或 `...` 或 '...') / 标识符（含关键字判断）/ 数字 / 其它
  const re =
    /("(?:[^"\\]|\\.)*"|`[^`]*`|'(?:[^'\\]|\\.)*')|(\b[A-Za-z_]\w*\b)|(\b\d[\d._]*\b)|([^\s"`'\w]+)|(\s+)/g;
  let m: RegExpExecArray | null;
  const kwSet = KEYWORDS[lang];

  while ((m = re.exec(code)) !== null) {
    if (m[1]) {
      out.push({ text: m[1], kind: "str" });
    } else if (m[2]) {
      if (kwSet?.has(m[2])) {
        out.push({ text: m[2], kind: "kw" });
      } else {
        out.push({ text: m[2], kind: "text" });
      }
    } else if (m[3]) {
      out.push({ text: m[3], kind: "num" });
    } else {
      out.push({ text: m[0], kind: "text" });
    }
  }
  if (comment) {
    out.push({ text: comment, kind: "com" });
  }
  return out;
}
