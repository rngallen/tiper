import { redirect } from "next/navigation";

export default async function VcfFeeRedirect({ params }: { params: Promise<{ id: string }> }) {
  redirect(`/finance/setups/vsf/${(await params).id}`);
}
