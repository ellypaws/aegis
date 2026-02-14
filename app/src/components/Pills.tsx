import { cn } from "../lib/utils";
import { UI } from "../constants";

export function RolePill({ name, color }: { name: string; color?: number }) {
    // Color helper
    const colorStyle = color ? { backgroundColor: `#${color.toString(16).padStart(6, '0')}` } : {};

    return (
        <span className={cn(UI.pill, "border-blue-200")} title="Role">
            <span className={cn("h-2.5 w-2.5 rounded-full border-2 border-white", !color && "bg-zinc-500")} style={colorStyle} />
            <span className="truncate">{name}</span>
        </span>
    );
}

export function ChannelPill({ name }: { name: string }) {
    return (
        <span className={cn(UI.pill, "border-green-200")} title="Channel">
            <span className="text-zinc-400">#</span>
            <span className="truncate">{name}</span>
        </span>
    );
}
