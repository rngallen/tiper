import { redirect } from "next/navigation";

export default function NotificationsIndexPage() {
  redirect("/settings/notifications/mail");
}
