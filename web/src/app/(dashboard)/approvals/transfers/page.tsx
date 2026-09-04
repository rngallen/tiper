import { redirect } from "next/navigation";

export default function TransferApprovalsRedirect() {
  redirect("/approvals/itt");
}
