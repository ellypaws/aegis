import { cn } from "../lib/utils";

export function SkeletonCard() {
    return (
        <div
            className={cn(
                "relative overflow-hidden rounded-3xl border-4 bg-white",
                "h-60 flex flex-col w-full",
                "border-zinc-200",
                "shadow-[6px_6px_0px_rgba(0,0,0,0.16)]"
            )}
        >
            {/* Image area skeleton */}
            <div className="h-48 w-full border-b-4 border-zinc-100 bg-zinc-100 shrink-0 relative overflow-hidden">
                <div className="skeleton-shimmer absolute inset-0" />
            </div>
            {/* Title area skeleton */}
            <div className="p-3 min-w-0 flex-1 flex flex-col justify-center">
                <div className="h-3.5 w-3/5 rounded-full bg-zinc-200 relative overflow-hidden">
                    <div className="skeleton-shimmer absolute inset-0" />
                </div>
            </div>
        </div>
    );
}
