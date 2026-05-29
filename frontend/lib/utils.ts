import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

// cn 合并 className 并消化 tailwind 冲突；shadcn 项目惯用模式
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}
