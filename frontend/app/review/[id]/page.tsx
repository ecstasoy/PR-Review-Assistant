import { SummaryCard } from "@/components/SummaryCard";
import { RiskList } from "@/components/RiskList";
import { DiffViewerWithAnnotations } from "@/components/DiffViewerWithAnnotations";

interface PageProps {
  params: Promise<{ id: string }>;
}

export default async function ReviewDetailPage({ params }: PageProps) {
  const { id } = await params;
  return (
    <section className="space-y-6">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight">
          评审结果 <span className="font-mono text-base text-zinc-500">#{id}</span>
        </h1>
      </header>
      <SummaryCard reviewId={id} />
      {/* 详情页尚未接通 store；后续 PR 调 getReview(id) 拿真实数据 */}
      <RiskList risks={[]} />
      <DiffViewerWithAnnotations reviewId={id} />
    </section>
  );
}
