import React, { useState, useRef, useEffect, useMemo } from "react";
import { cn } from "../lib/utils";
import { UI } from "../constants";
import { Chip } from "./Chip";

export function DropdownAddToList<T extends { id: string; name: string }>({
    label,
    placeholder,
    options,
    selectedIds,
    onAdd,
    onRemove,
    renderSelected,
}: {
    label: string;
    placeholder: string;
    options: T[];
    selectedIds: string[];
    onAdd: (id: string) => void;
    onRemove: (id: string) => void;
    renderSelected?: (id: string) => React.ReactNode;
}) {
    const [open, setOpen] = useState(false);
    const [q, setQ] = useState("");
    const wrapRef = useRef<HTMLDivElement | null>(null);

    useEffect(() => {
        const onDoc = (e: MouseEvent) => {
            if (!wrapRef.current) return;
            if (!wrapRef.current.contains(e.target as Node)) setOpen(false);
        };
        document.addEventListener("mousedown", onDoc);
        return () => document.removeEventListener("mousedown", onDoc);
    }, []);

    const selectedSet = useMemo(() => new Set(selectedIds), [selectedIds]);
    const filtered = useMemo(() => {
        const s = q.trim().toLowerCase();
        const base = s ? options.filter((o) => o.name.toLowerCase().includes(s)) : options;
        return base;
    }, [options, q]);

    return (
        <div className="space-y-2" ref={wrapRef}>
            <div className="flex items-center justify-between gap-3">
                <div className={UI.label}>{label}</div>
                <button type="button" onClick={() => setOpen((v) => !v)} className={cn(UI.button, "px-3 py-1.5 text-[11px]", UI.btnYellow)}>
                    {open ? "Close" : "Add"}
                </button>
            </div>

            <div className="flex flex-wrap gap-2">
                {selectedIds.length === 0 ? (
                    <span className="text-xs font-bold text-zinc-400">None selected.</span>
                ) : (
                    selectedIds.map((id) => (
                        <span key={id}>
                            {renderSelected ? (
                                <span onClick={() => onRemove(id)} className="cursor-pointer inline-block">
                                    {renderSelected(id)}
                                </span>
                            ) : (
                                <Chip label={options.find((o) => o.id === id)?.name ?? id} onRemove={() => onRemove(id)} />
                            )}
                        </span>
                    ))
                )}
            </div>

            <div className="relative">
                <button type="button" onClick={() => setOpen(!open)} className={cn(UI.input, "text-left", "flex items-center justify-between")}>
                    <span className="text-zinc-500 font-bold">{placeholder}</span>
                    <span className="text-zinc-400">▾</span>
                </button>

                {open ? (
                    <div className="absolute z-20 mt-2 w-full overflow-hidden rounded-2xl border-4 border-zinc-200 bg-white shadow-[6px_6px_0px_rgba(0,0,0,0.18)]">
                        <div className="p-2">
                            <input value={q} onChange={(e) => setQ(e.target.value)} placeholder="Search…" className={UI.input} />
                        </div>
                        <div className="max-h-56 overflow-auto p-1">
                            {filtered.length === 0 ? (
                                <div className="px-3 py-3 text-sm font-bold text-zinc-400">No matches.</div>
                            ) : (
                                filtered.map((o) => {
                                    const selected = selectedSet.has(o.id);
                                    return (
                                        <button
                                            key={o.id}
                                            type="button"
                                            onClick={() => {
                                                if (selected) {
                                                    onRemove(o.id);
                                                } else {
                                                    onAdd(o.id);
                                                }
                                                // setQ(""); // Keep search query? Or clear it?
                                                // decision: keep it to allow multi-select easily
                                            }}
                                            className={cn(
                                                "w-full rounded-xl px-3 py-2 text-left text-sm font-bold",
                                                selected ? "text-zinc-300 bg-zinc-50 hover:text-red-400 hover:bg-red-50" : "text-zinc-800 hover:bg-yellow-100",
                                                "active:translate-x-[1px] active:translate-y-[1px]"
                                            )}
                                            title={selected ? "Click to remove" : "Click to add"}
                                        >
                                            {o.name} {selected ? "(Added)" : ""}
                                        </button>
                                    );
                                })
                            )}
                        </div>
                    </div>
                ) : null}
            </div>
        </div>
    );
}
