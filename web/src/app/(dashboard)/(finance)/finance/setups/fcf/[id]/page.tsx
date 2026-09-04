import { redirect } from "next/navigation";

export default async function FcfFeeRedirect({ params }: { params: Promise<{ id: string }> }) {
  redirect(`/finance/setups/fsf/${(await params).id}`);
}
