import React, { useState, useEffect } from "react";
import { cn, clamp } from "../lib/utils";
import { MOCK_USERS } from "../data/mock";
import { DiscordUser } from "../types";
import { Modal } from "./Modal";
import { RolePill } from "./Pills";

export function LoginModal({
    open,
    onClose,
    onLogin,
}: {
    open: boolean;
    onClose: () => void;
    onLogin: (u: DiscordUser) => void;
}) {
    const [loadingId, setLoadingId] = useState<string | null>(null);
    const [progress, setProgress] = useState(0);

    useEffect(() => {
        if (!open) {
            setLoadingId(null);
            setProgress(0);
        }
    }, [open]);

    useEffect(() => {
        if (!loadingId) return;
        let raf = 0;
        let start = performance.now();
        const dur = 1100 + Math.random() * 700;
        const tick = (t: number) => {
            const p = clamp((t - start) / dur, 0, 1);
            setProgress(p);
            if (p < 1) raf = requestAnimationFrame(tick);
        };
        raf = requestAnimationFrame(tick);
        return () => cancelAnimationFrame(raf);
    }, [loadingId]);

    return (
        <Modal open={open} title="Discord Login" onClose={() => (loadingId ? null : onClose())}>
            <div className="text-sm font-bold text-zinc-700">Pick a simulated account / role set. We’ll pretend to wait for Discord.</div>

            <div className="mt-4 grid gap-3">
                {MOCK_USERS.map((u) => {
                    const busy = loadingId === u.id;
                    const disabled = !!loadingId && !busy;
                    return (
                        <button
                            key={u.id}
                            type="button"
                            disabled={disabled}
                            onClick={() => {
                                setLoadingId(u.id);
                                const ms = 1100 + Math.random() * 700;
                                window.setTimeout(() => {
                                    onLogin(u);
                                    setLoadingId(null);
                                    onClose();
                                }, ms);
                            }}
                            className={cn(
                                "w-full rounded-3xl border-4 bg-white p-4 text-left",
                                busy ? "border-zinc-900" : "border-zinc-200",
                                disabled ? "opacity-60" : "hover:bg-yellow-50",
                                "shadow-[6px_6px_0px_rgba(0,0,0,0.16)]",
                                "active:translate-x-[1px] active:translate-y-[1px]"
                            )}
                        >
                            <div className="flex items-center justify-between gap-3">
                                <div className="min-w-0">
                                    <div className="truncate text-base font-black uppercase tracking-wide text-zinc-900">{u.username}</div>
                                    <div className="mt-2 flex flex-wrap gap-2">
                                        {u.roles.map((r) => (
                                            <RolePill key={r.id} roleId={r.id} />
                                        ))}
                                    </div>
                                </div>
                                <div className={cn("rounded-2xl border-4 px-3 py-2 text-xs font-black uppercase", u.isAuthor ? "border-red-300 bg-red-200 text-red-900" : "border-blue-300 bg-blue-200 text-blue-900")}>
                                    {u.isAuthor ? "Author" : "Viewer"}
                                </div>
                            </div>

                            {busy ? (
                                <div className="mt-4">
                                    <div className="text-xs font-black uppercase tracking-wide text-zinc-500">Waiting for Discord…</div>
                                    <div className="mt-2 h-4 overflow-hidden rounded-full border-4 border-zinc-200 bg-white shadow-[3px_3px_0px_rgba(0,0,0,0.10)]">
                                        <div className="h-full bg-green-300" style={{ width: `${Math.round(progress * 100)}%` }} />
                                    </div>
                                </div>
                            ) : null}
                        </button>
                    );
                })}
            </div>

            <div className="mt-4 text-xs font-bold text-zinc-400">Real app: open Discord OAuth popup + wait for callback + exchange code.</div>
        </Modal>
    );
}
