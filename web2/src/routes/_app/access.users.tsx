import { createFileRoute } from "@tanstack/react-router";
import { RegisterPage } from "@/app/pages";

export const Route = createFileRoute("/_app/access/users")({
  component: () => (
    <RegisterPage
      title="Users"
      columns={["Name", "Email", "Role", "Status"]}
      rows={[["Luqman Jr", "ngallen4@gmail.com", "Terminal admin", "Active"]]}
    />
  ),
});
