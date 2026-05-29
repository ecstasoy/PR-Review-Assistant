import { cva, type VariantProps } from "class-variance-authority";

import { cn } from "@/lib/utils";

// Spinner 圆环旋转；color 用 currentColor 让 className 控（如 text-muted）
const spinnerVariants = cva(
  "inline-block animate-spin rounded-full border-2 border-current border-t-transparent",
  {
    variants: {
      size: {
        xs: "h-3 w-3",
        sm: "h-4 w-4",
        md: "h-5 w-5",
        lg: "h-6 w-6",
      },
    },
    defaultVariants: { size: "sm" },
  },
);

export interface SpinnerProps extends VariantProps<typeof spinnerVariants> {
  className?: string;
  label?: string; // a11y：屏幕阅读器看到的文本
}

export function Spinner({ size, className, label = "加载中" }: SpinnerProps) {
  return (
    <span
      role="status"
      aria-label={label}
      className={cn(spinnerVariants({ size }), className)}
    />
  );
}
