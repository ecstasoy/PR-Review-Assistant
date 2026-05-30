// Unified diff patch 解析器：把 GitHub PR API 的 patch 字符串拆成 hunks + lines
// 输入示例：
//   @@ -38,17 +38,24 @@ func (s *shard) get(...) {
//   	return it.value, true
//   }
//   -// every cleanupInterval. Holds a read lock while scanning.
//   +// every cleanupInterval. Takes the write lock.

export type DiffLineType = "context" | "add" | "del";

export interface DiffLine {
  type: DiffLineType;
  old: number | null; // del / context 有；add 时 null
  new: number | null; // add / context 有；del 时 null
  text: string;       // 行内容，不含前导 +/-/空格
}

export interface DiffHunk {
  header: string;     // 原 @@ ... @@ 行，渲染时直接显示
  oldStart: number;
  oldCount: number;
  newStart: number;
  newCount: number;
  lines: DiffLine[];
}

// HUNK_HEADER_RE 匹配 @@ -<old>,<oldCount> +<new>,<newCount> @@
// count 可省略（=1 时 GitHub 省略）
const HUNK_HEADER_RE = /^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@/;

export function parsePatch(patch: string): DiffHunk[] {
  if (!patch) return [];
  const hunks: DiffHunk[] = [];
  const lines = patch.split("\n");
  let current: DiffHunk | null = null;
  let oldLine = 0;
  let newLine = 0;

  for (const line of lines) {
    const m = HUNK_HEADER_RE.exec(line);
    if (m) {
      current = {
        header: line,
        oldStart: Number(m[1]),
        oldCount: m[2] ? Number(m[2]) : 1,
        newStart: Number(m[3]),
        newCount: m[4] ? Number(m[4]) : 1,
        lines: [],
      };
      oldLine = current.oldStart;
      newLine = current.newStart;
      hunks.push(current);
      continue;
    }
    if (!current) continue; // 文件头 / 元信息（+++ / --- 等）跳过

    const prefix = line[0];
    if (prefix === "+") {
      current.lines.push({ type: "add", old: null, new: newLine, text: line.slice(1) });
      newLine++;
    } else if (prefix === "-") {
      current.lines.push({ type: "del", old: oldLine, new: null, text: line.slice(1) });
      oldLine++;
    } else if (prefix === "\\") {
      // "\ No newline at end of file" 标记，不算行
    } else {
      // context（多以 " " 起头；空行容忍直接收）
      const text = prefix === " " ? line.slice(1) : line;
      current.lines.push({ type: "context", old: oldLine, new: newLine, text });
      oldLine++;
      newLine++;
    }
  }
  return hunks;
}
