export function LockedOverlay({ label }: { label: string }) {
    return (
        <div className="absolute inset-0 flex items-center justify-center">
            <div className="rounded-2xl border-4 border-zinc-200 bg-white/90 px-4 py-3 text-center backdrop-blur shadow-[4px_4px_0px_rgba(0,0,0,0.16)]">
                <div className="text-sm font-black uppercase text-zinc-700">🔒 Locked</div>
                <div className="mt-1 text-xs font-bold text-zinc-500">{label}</div>
            </div>
        </div>
    );
}
