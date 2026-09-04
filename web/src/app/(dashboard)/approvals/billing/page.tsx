import { redirect } from "next/navigation";

export default function BillingApprovalsRedirect() {
  redirect("/approvals/billing-runs");
}
