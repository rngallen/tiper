"use client";

import { useCallback, useMemo, useRef, useState } from "react";
import { Trash2 } from "lucide-react";
import { AsyncSearchableSelect } from "@/components/SearchableSelect";
import { Button, ErrorBanner } from "@/components/ui";
import { api, formatApiError } from "@/lib/api";

export type PoolUser = {
  id: string;
  email: string;
  firstName?: string;
  lastName?: string;
};

function displayName(u: PoolUser) {
  const name = [u.firstName, u.lastName].filter(Boolean).join(" ").trim();
  return name || u.email;
}

/**
 * Membership editor for workflow initiator (soft-reject) pools.
 * Search-to-add sits above the assigned ledger.
 */
export function UserPoolPicker({
  members,
  onChange,
  disabled,
  helpText,
  title = "Assigned",
}: {
  members: PoolUser[];
  onChange: (next: PoolUser[]) => void;
  disabled?: boolean;
  helpText?: string;
  title?: string;
}) {
  const [addValue, setAddValue] = useState("");
  const [searchError, setSearchError] = useState("");
  const cacheRef = useRef<Map<string, PoolUser>>(new Map());

  const memberIds = useMemo(() => new Set(members.map((m) => m.id)), [members]);

  const loadOptions = useCallback(
    async (query: string) => {
      setSearchError("");
      try {
        const rows = await api.get<PoolUser[]>("/auth/users/search", {
          search: query || undefined,
          pageSize: 20,
          excludeSelf: true,
        });
        for (const u of rows || []) {
          cacheRef.current.set(u.id, u);
        }
        return (rows || [])
          .filter((u) => !memberIds.has(u.id))
          .map((u) => ({
            value: u.id,
            label: `${displayName(u)} · ${u.email}`,
            keywords: `${u.firstName || ""} ${u.lastName || ""} ${u.email}`,
          }));
      } catch (err) {
        setSearchError(formatApiError(err, "Failed to search users"));
        return [];
      }
    },
    [memberIds],
  );

  function addUser(uid: string) {
    if (!uid || memberIds.has(uid) || disabled) return;
    const hit = cacheRef.current.get(uid);
    if (!hit) return;
    onChange([...members, hit]);
    setAddValue("");
  }

  function removeUser(id: string) {
    if (disabled) return;
    onChange(members.filter((m) => m.id !== id));
  }

  return (
    <div className="space-y-2">
      {helpText ? <p className="text-xs text-slate-500">{helpText}</p> : null}
      {searchError ? <ErrorBanner message={searchError} /> : null}

      <section className="rounded-lg border border-slate-200 bg-white">
        <div className="flex flex-wrap items-end justify-between gap-3 border-b border-slate-100 px-4 py-3">
          <h4 className="text-xs font-bold uppercase tracking-wide text-brand-800">
            {title} ({members.length})
          </h4>
          <div className="w-full min-w-[14rem] sm:w-80">
            <AsyncSearchableSelect
              value={addValue}
              selectedLabel=""
              disabled={disabled}
              placeholder="Search name or email to add…"
              emptyLabel="No matching users"
              loadOptions={loadOptions}
              onChange={(value) => {
                if (value) addUser(value);
              }}
            />
          </div>
        </div>
        {members.length === 0 ? (
          <p className="px-4 py-5 text-sm text-slate-500">
            No users assigned. Search above to add.
          </p>
        ) : (
          <table className="pp-ledger min-w-full text-left text-sm">
            <thead className="pp-table-head">
              <tr>
                <th>Name</th>
                <th>Email</th>
                <th className="w-16"> </th>
              </tr>
            </thead>
            <tbody>
              {members.map((u) => (
                <tr key={u.id}>
                  <td className="font-semibold text-slate-900">
                    {displayName(u)}
                  </td>
                  <td className="text-slate-700">{u.email}</td>
                  <td>
                    <Button
                      type="button"
                      variant="ghost"
                      className="!px-2 !py-1 text-red-700"
                      disabled={disabled}
                      onClick={() => removeUser(u.id)}
                      title="Remove from pool"
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>
    </div>
  );
}
