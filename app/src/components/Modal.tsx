import React, { useEffect } from "react";
import { cn } from "../lib/utils";
import { UI } from "../constants";
import { Patterns } from "./Patterns";

export function Modal({ open, title, children, onClose }: { open: boolean; title: string; children: React.ReactNode; onClose: () => void }) {
    useEffect(() => {
        if (!open) return;
        const onKey = (e: KeyboardEvent) => {
            if (e.key === "Escape") onClose();
        };
        window.addEventListener("keydown", onKey);
        return () => window.removeEventListener("keydown", onKey);
    }, [open, onClose]);

    if (!open) return null;
    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
            <div className="absolute inset-0 bg-black/30" onClick={onClose} />
            <div className={cn("relative w-full max-w-lg overflow-hidden", UI.card)}>
                <Patterns.Polka color="rgba(0,0,0,0.06)" />
                <div className="pointer-events-none absolute top-[-16px] left-[-16px] h-24 w-24 rounded-full border-4 border-white bg-blue-400 shadow-lg" />
                <div className="pointer-events-none absolute bottom-[-10px] right-[-10px] h-20 w-20 rotate-12 border-4 border-white bg-red-400 shadow-lg" />

                <div className="relative z-10 p-5">
                    <div className="flex items-start justify-between gap-3">
                        <div>
                            <div className="inline-block rounded-xl border-4 border-red-400 bg-white px-4 py-2 rotate-[-2deg] shadow-[4px_4px_0px_rgba(248,113,113,1)]">
                                <div className="text-base font-black uppercase tracking-tight text-red-500">{title}</div>
                            </div>
                        </div>
                        <button type="button" onClick={onClose} className={cn(UI.button, "px-3 py-1.5 text-xs", UI.btnYellow)}>
                            Close
                        </button>
                    </div>

                    <div className="mt-4">{children}</div>
                </div>
            </div>
        </div>
    );
}
