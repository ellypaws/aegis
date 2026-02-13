import React from "react";
import { cn } from "../lib/utils";

export function Chip({ label, onRemove }: { label: string; onRemove?: () => void }) {
    return (
        <button
            type="button"
            onClick={onRemove}
            className={cn(
                "inline-flex items-center gap-2 rounded-full border-4 px-3 py-1 text-xs font-black uppercase tracking-wide",
                "border-zinc-200 bg-white text-zinc-800 shadow-[3px_3px_0px_rgba(0,0,0,0.12)]",
                onRemove && "hover:bg-zinc-50 active:translate-x-[1px] active:translate-y-[1px]"
            )}
            title={onRemove ? "Click to remove" : undefined}
        >
            <span className="truncate max-w-[18ch]">{label}</span>
            {onRemove ? <span className="text-zinc-400">×</span> : null}
        </button>
    );
}
