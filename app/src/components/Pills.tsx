import React from "react";
import { cn } from "../lib/utils";
import { UI } from "../constants";
import { MOCK_GUILD } from "../data/mock";

export function RolePill({ roleId }: { roleId: string }) {
    const r = MOCK_GUILD.roles.find((x) => x.id === roleId);
    return (
        <span className={cn(UI.pill, "border-blue-200")} title="Role">
            <span className={cn("h-2.5 w-2.5 rounded-full border-2 border-white", r?.color ?? "bg-zinc-500")} />
            <span className="truncate">{r?.name ?? roleId}</span>
        </span>
    );
}

export function ChannelPill({ channelId }: { channelId: string }) {
    const c = MOCK_GUILD.channels.find((x) => x.id === channelId);
    return (
        <span className={cn(UI.pill, "border-green-200")} title="Channel">
            <span className="text-zinc-400">#</span>
            <span className="truncate">{c?.name ?? channelId}</span>
        </span>
    );
}
