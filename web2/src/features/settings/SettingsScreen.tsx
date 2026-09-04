import { useState } from "react";
import { useSession } from "@/features/auth/session";
import { http } from "@/shared/api/http";
import { errorMessage } from "@/shared/api/errors";
import { Button } from "@/shared/ui/button";
import { Input } from "@/shared/ui/input";
import { Label } from "@/shared/ui/label";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card";
import { fullName } from "@/shared/lib/format";

export function SettingsScreen() {
  const { user, changePassword } = useSession();
  const [oldPassword, setOld] = useState("");
  const [newPassword, setNew] = useState("");
  const [confirmPassword, setConfirm] = useState("");
  const [msg, setMsg] = useState("");
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const [theme, setTheme] = useState(() => (typeof document !== "undefined" && document.documentElement.classList.contains("dark") ? "dark" : "light"));

  async function onPassword(e: React.FormEvent) {
    e.preventDefault();
    setErr(""); setMsg(""); setBusy(true);
    try {
      await changePassword(oldPassword, newPassword, confirmPassword);
      setMsg("Password updated."); setOld(""); setNew(""); setConfirm("");
    } catch (e) { setErr(errorMessage(e)); }
    finally { setBusy(false); }
  }

  function toggleTheme(next: "light" | "dark") {
    setTheme(next);
    document.documentElement.classList.toggle("dark", next === "dark");
    void http.patch("/auth/profile/apperance-settings", { theme: next }).catch(() => undefined);
  }

  return (
    <div className="grid gap-4 lg:grid-cols-2">
      <Card>
        <CardHeader><CardTitle>Session</CardTitle></CardHeader>
        <CardContent className="space-y-2 text-sm">
          <p className="font-medium">{fullName(user) || user?.email}</p>
          <p className="text-muted-foreground">{user?.email}</p>
          <p className="text-xs text-muted-foreground">PASETO cookies · CSRF double-submit · idle refresh via /auth/refresh</p>
        </CardContent>
      </Card>
      <Card>
        <CardHeader><CardTitle>Appearance</CardTitle></CardHeader>
        <CardContent className="flex gap-2">
          <Button size="sm" variant={theme === "light" ? "default" : "outline"} onClick={() => toggleTheme("light")}>Light</Button>
          <Button size="sm" variant={theme === "dark" ? "default" : "outline"} onClick={() => toggleTheme("dark")}>Dark</Button>
        </CardContent>
      </Card>
      <Card className="lg:col-span-2">
        <CardHeader><CardTitle>Change password</CardTitle></CardHeader>
        <CardContent>
          <form className="grid max-w-md gap-3" onSubmit={onPassword}>
            <label className="space-y-1 text-sm"><Label>Current password</Label><Input type="password" value={oldPassword} onChange={(e) => setOld(e.target.value)} required /></label>
            <label className="space-y-1 text-sm"><Label>New password</Label><Input type="password" value={newPassword} onChange={(e) => setNew(e.target.value)} required /></label>
            <label className="space-y-1 text-sm"><Label>Confirm</Label><Input type="password" value={confirmPassword} onChange={(e) => setConfirm(e.target.value)} required /></label>
            {err ? <p className="text-sm text-destructive">{err}</p> : null}
            {msg ? <p className="text-sm text-emerald-600">{msg}</p> : null}
            <Button type="submit" className="w-fit" loading={busy}>Update password</Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
