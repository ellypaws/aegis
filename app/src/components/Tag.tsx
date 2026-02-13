import React from "react";
import { cn } from "../lib/utils";

export function Tag({ label, active, onClick }: { label: string; active?: boolean; onClick: () => void }) {
    return (
        <button
            type="button"
            onClick={onClick}
            className={cn(
                "rounded-full border-4 px-3 py-1 text-xs font-black uppercase tracking-wide shadow-[3px_3px_0px_rgba(0,0,0,0.12)]",
                active ? "border-zinc-900 bg-yellow-200 text-zinc-900" : "border-zinc-200 bg-white text-zinc-700 hover:bg-yellow-100",
                "active:translate-x-[1px] active:translate-y-[1px]"
            )}
            title="Filter by tag"
        >
            #{label}
        </button>
    );
}
