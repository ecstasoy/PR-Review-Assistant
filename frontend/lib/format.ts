// formatAuthorRole 把 GitHub author_association 枚举翻成中文显示
// 后端透传原始值（OWNER / CONTRIBUTOR / 等），UI 层做语言化
const ROLE_LABELS: Record<string, string> = {
  OWNER: "维护者",
  MEMBER: "成员",
  COLLABORATOR: "协作者",
  CONTRIBUTOR: "贡献者",
  FIRST_TIMER: "首次贡献",
  FIRST_TIME_CONTRIBUTOR: "首次贡献",
  MANNEQUIN: "Mannequin",
  NONE: "",
};

export function formatAuthorRole(role: string | undefined): string {
  if (!role) return "";
  return ROLE_LABELS[role.toUpperCase()] ?? role;
}
