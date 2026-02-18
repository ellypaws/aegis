import React, { useState, useRef, useEffect, useMemo } from "react";
import { createPortal } from "react-dom";
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
    const buttonRef = useRef<HTMLButtonElement | null>(null);
    const [menuStyle, setMenuStyle] = useState<React.CSSProperties>({});

    const menuRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        const onDoc = (e: MouseEvent) => {
            if (!wrapRef.current) return;
            if (
                wrapRef.current.contains(e.target as Node) ||
                menuRef.current?.contains(e.target as Node)
            ) {
                return;
            }
            setOpen(false);
        };
        document.addEventListener("mousedown", onDoc);
        return () => document.removeEventListener("mousedown", onDoc);
    }, []);

    // Update menu position when opening or resizing
    useEffect(() => {
        if (!open || !buttonRef.current) return;

        const updatePosition = () => {
            if (!buttonRef.current) return;
            const rect = buttonRef.current.getBoundingClientRect();
            setMenuStyle({
                position: "fixed",
                top: rect.bottom + 8,
                left: rect.left,
                width: rect.width,
                zIndex: 10,
            });
        };

        updatePosition();
        window.addEventListener("resize", updatePosition);
        window.addEventListener("scroll", updatePosition, true);

        return () => {
            window.removeEventListener("resize", updatePosition);
            window.removeEventListener("scroll", updatePosition, true);
        };
    }, [open]);

    const selectedSet = useMemo(() => new Set(selectedIds), [selectedIds]);
    const filtered = useMemo(() => {
        const s = q.trim().toLowerCase();
        const base = s ? options.filter((o) => o.name.toLowerCase().includes(s)) : options;
        return base;
    }, [options, q]);

    const dropdownMenu = open ? (
        <div
            ref={menuRef}
            style={menuStyle}
            className="overflow-hidden rounded-2xl border-4 border-zinc-200 bg-white shadow-[6px_6px_0px_rgba(0,0,0,0.18)]"
        >
            <div className="p-2">
                <input
                    value={q}
                    onChange={(e) => setQ(e.target.value)}
                    placeholder="Search…"
                    className={UI.input}
                    onClick={(e) => e.stopPropagation()}
                />
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
                                        setQ(""); // Clear search on select
                                        onAdd(o.id);
                                    }
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
    ) : null;

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
                <button
                    ref={buttonRef}
                    type="button"
                    onClick={() => setOpen(!open)}
                    className={cn(UI.input, "text-left", "flex items-center justify-between w-full")}
                >
                    <span className="text-zinc-500 font-bold">{placeholder}</span>
                    <span className="text-zinc-400">▾</span>
                </button>
            </div>

            {dropdownMenu ? createPortal(dropdownMenu, document.body) : null}
        </div>
    );
}
