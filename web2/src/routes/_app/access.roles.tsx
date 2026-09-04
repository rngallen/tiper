import { createFileRoute } from "@tanstack/react-router";
import { RegisterPage } from "@/app/pages";

export const Route = createFileRoute("/_app/access/roles")({
  component: () => (
    <RegisterPage
      title="Roles"
      columns={["Role", "Permissions", "Users"]}
      rows={[["Terminal admin", "users.manage, inventory.manage", "2"]]}
    />
  ),
});
